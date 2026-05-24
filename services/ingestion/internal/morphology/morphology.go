// Package morphology measures chain morphology of Cat III hypothesis
// formation events per §0143 Sub-benchmark 1 instrumentation
// requirement: "chain morphology (chain_depth_max + chain_breadth_at_root)".
//
// Per §0154 Methodological observation 3: signature-layer
// instrumentation (§0154) is partial; full Sub-benchmark 1
// measurement requires substrate-layer chain morphology
// (post-formation-commit). This package operationalizes the
// substrate-layer half — walks formation events, computes per-
// hypothesis morphology metrics, surfaces aggregates for operator
// + Sub-benchmark 1 diagnostic use.
//
// Definitions per §0143:
//   - chain_depth_max: longest `influenced_by` path from this
//     formation to a Cat I root (formation with no
//     direct_influenced_by predecessors). A formation with no
//     predecessors has depth_max = 0.
//   - chain_breadth_at_root: count of direct_influenced_by entries
//     on the formation (number of immediate predecessor hypotheses).
//
// Both metrics derive from substrate-committed fields per §0136
// canonical-serialization-contract + §0139 hash-list discipline +
// §0140 paired-dimension enforcement. The substrate's β-graph
// storage per §0134-τ + §0136 makes direct_influenced_by + closure_hashes
// both readable per formation event; this package consumes both.
package morphology

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// formationSubtypes enumerates Cat III formation message_type strings
// per §0010 Q2-A.2 4-subtype hierarchy. Each value is matched against
// EventRow.MessageType during substrate walk.
var formationSubtypes = map[string]bool{
	"ghosttrace.events.v1.BehavioralClusterFormation":  true,
	"ghosttrace.events.v1.AutomationGroupFormation":    true,
	"ghosttrace.events.v1.CampaignHypothesisFormation": true,
	"ghosttrace.events.v1.CoordinationRingFormation":   true,
}

// ChainMorphology carries per-hypothesis morphology metrics derived
// from a single substrate-committed formation event. Per §0143
// instrumentation-by-morfologia axis: ChainDepthMax +
// ChainBreadthAtRoot are the load-bearing metrics for Sub-benchmark 1
// "chains-fracas vs chains-fortes" diagnostic distinction.
type ChainMorphology struct {
	// HypothesisHash is the BLAKE3 content-hash of the formation
	// event itself. Identifies the hypothesis at substrate level.
	HypothesisHash [32]byte
	// SubtypeName is the fully-qualified proto message_type string
	// (e.g., "ghosttrace.events.v1.AutomationGroupFormation").
	SubtypeName string
	// ActorRefs are the actor_refs extracted from the formation
	// event. May be empty (e.g., CampaignHypothesisFormation models
	// event sets rather than actor sets per ontology).
	ActorRefs []string
	// ChainDepthMax is the longest path from this formation to a
	// Cat I root, measured along direct_influenced_by edges via
	// recursive walk + memoization. Formations with no direct
	// predecessors have ChainDepthMax = 0.
	ChainDepthMax uint32
	// ChainBreadthAtRoot is the count of direct_influenced_by
	// entries on this formation event. Mirrors the §0143 definition.
	ChainBreadthAtRoot uint32
	// ClosureCount is the count of closure_hashes (total transitive
	// ancestors per §0134-τ). Differs from ChainDepthMax: closure
	// is set-count (deduplicated); depth is path-length.
	ClosureCount uint32
	// SourceEventCount is the count of source_event_hashes (Cat I
	// observation roots that contributed to formation).
	SourceEventCount uint32
	// FormationAt is the formation_at timestamp (Unix nanoseconds)
	// from the formation event, when present in the proto.
	FormationAt int64
}

// AggregateStats summarizes per-substrate-walk metrics across all
// formation events measured. Per §0143 Sub-benchmark 1: aggregates
// enable operator-readable "chains-fracas vs chains-fortes" diagnosis
// at a glance, without forcing inspection of per-hypothesis records.
type AggregateStats struct {
	// TotalFormations is the count of formation events walked.
	TotalFormations uint32
	// PerSubtype is the count per subtype message_type string.
	PerSubtype map[string]uint32
	// DepthHistogram is the count of formations per ChainDepthMax
	// value. Empty when TotalFormations == 0. Useful for visualizing
	// chain-depth distribution at a glance.
	DepthHistogram map[uint32]uint32
	// BreadthHistogram is the count of formations per
	// ChainBreadthAtRoot value.
	BreadthHistogram map[uint32]uint32
	// ChainsFracasCount counts formations meeting the §0143
	// chains-fracas predicate: ChainDepthMax <= 2 OR
	// ChainBreadthAtRoot <= 3. This is the §0143-named "F3
	// insufficient" diagnostic indicator at substrate layer.
	ChainsFracasCount uint32
	// ChainsFortesCount counts formations meeting the §0143
	// chains-fortes predicate: ChainDepthMax > 2 AND
	// ChainBreadthAtRoot > 3. This is the §0143-named "constitutional
	// thesis confirmed" diagnostic indicator at substrate layer.
	ChainsFortesCount uint32
}

// Measurement is the full output: per-hypothesis morphology slice +
// aggregate stats.
type Measurement struct {
	Hypotheses []*ChainMorphology
	Stats      AggregateStats
}

// formationEntry is the in-memory representation of a formation event
// used during Measure's three-pass walk. Lifted to package level so
// parseFormationEntry can return it.
type formationEntry struct {
	hash               [32]byte
	messageType        string
	actorRefs          []string
	directInfluencedBy [][]byte
	closureCount       uint32
	sourceEventCount   uint32
	formationAt        int64
}

// Measure walks the substrate's formation events, computes per-
// hypothesis chain morphology, and returns the measurement + aggregate
// stats. O(N + E) where N = number of formation events and E = total
// edges in the influenced_by graph (bounded by sum of direct_influenced_by
// list sizes). Memoization caches per-hash depth computations to avoid
// re-walking shared ancestor subtrees.
//
// Returns Measurement with Hypotheses sorted by HypothesisHash
// ascending (deterministic test fixtures + reproducible operator
// output).
func Measure(ctx context.Context, sub *substrate.Substrate) (*Measurement, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// Pass 1: collect all formation events into an in-memory map
	// keyed by content-hash. Each entry carries the proto fields
	// needed for morphology computation.
	formations := make(map[[32]byte]*formationEntry)

	err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if !formationSubtypes[row.MessageType] {
			return nil
		}
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return fmt.Errorf("ReadBlob %x: %w", row.EventHash[:8], err)
		}
		entry, err := parseFormationEntry(row, payload)
		if err != nil {
			return fmt.Errorf("parseFormation %x: %w", row.EventHash[:8], err)
		}
		formations[row.EventHash] = entry
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Pass 2: per-formation, compute depth via memoized DFS over
	// direct_influenced_by edges. Cycles are structurally impossible
	// per §2.1 substrate immutability + §0021 substrate-time
	// generation (an influenced_by edge can only reference a record
	// committed at or before the influenced record's commit time;
	// content-hash references therefore form a DAG).
	depthCache := make(map[[32]byte]uint32)
	var computeDepth func(hash [32]byte) uint32
	computeDepth = func(hash [32]byte) uint32 {
		if d, ok := depthCache[hash]; ok {
			return d
		}
		entry, ok := formations[hash]
		if !ok {
			// Hash references a record not in the formation set
			// (e.g., a Cat I observation or a hash committed
			// outside the walk's scope). Treat as terminal.
			depthCache[hash] = 0
			return 0
		}
		var maxChildDepth uint32
		for _, child := range entry.directInfluencedBy {
			if len(child) != 32 {
				continue // malformed; skip
			}
			var childHash [32]byte
			copy(childHash[:], child)
			d := computeDepth(childHash)
			if d > maxChildDepth {
				maxChildDepth = d
			}
		}
		if len(entry.directInfluencedBy) == 0 {
			depthCache[hash] = 0
			return 0
		}
		result := maxChildDepth + 1
		depthCache[hash] = result
		return result
	}

	// Pass 3: emit ChainMorphology per formation; aggregate stats.
	hashes := make([][32]byte, 0, len(formations))
	for h := range formations {
		hashes = append(hashes, h)
	}
	sortHashesAscending(hashes)

	hypotheses := make([]*ChainMorphology, 0, len(hashes))
	stats := AggregateStats{
		PerSubtype:       make(map[string]uint32),
		DepthHistogram:   make(map[uint32]uint32),
		BreadthHistogram: make(map[uint32]uint32),
	}

	for _, h := range hashes {
		entry := formations[h]
		depthMax := computeDepth(h)
		breadth := uint32(len(entry.directInfluencedBy))

		hypotheses = append(hypotheses, &ChainMorphology{
			HypothesisHash:     h,
			SubtypeName:        entry.messageType,
			ActorRefs:          entry.actorRefs,
			ChainDepthMax:      depthMax,
			ChainBreadthAtRoot: breadth,
			ClosureCount:       entry.closureCount,
			SourceEventCount:   entry.sourceEventCount,
			FormationAt:        entry.formationAt,
		})

		stats.TotalFormations++
		stats.PerSubtype[entry.messageType]++
		stats.DepthHistogram[depthMax]++
		stats.BreadthHistogram[breadth]++

		// §0143 chains-fracas predicate: depth_max <= 2 OR
		// breadth_at_root <= 3 indicates F3 insufficient.
		if depthMax <= 2 || breadth <= 3 {
			stats.ChainsFracasCount++
		} else {
			stats.ChainsFortesCount++
		}
	}

	return &Measurement{
		Hypotheses: hypotheses,
		Stats:      stats,
	}, nil
}

// parseFormationEntry unmarshals a formation event payload into the
// internal formationEntry shape, normalizing across the 4 subtype
// proto message types. Each subtype carries the same morphology-
// relevant fields (actor_refs / source_event_hashes / direct_influenced_by
// / closure_hashes / formation_at) per §0136 canonical-serialization-
// contract.
func parseFormationEntry(row substrate.EventRow, payload []byte) (*formationEntry, error) {
	entry := &formationEntry{
		hash:        row.EventHash,
		messageType: row.MessageType,
	}
	switch row.MessageType {
	case "ghosttrace.events.v1.BehavioralClusterFormation":
		m := &eventsv1.BehavioralClusterFormation{}
		if err := proto.Unmarshal(payload, m); err != nil {
			return nil, err
		}
		entry.actorRefs = m.ActorRefs
		entry.directInfluencedBy = m.DirectInfluencedBy
		entry.closureCount = uint32(len(m.ClosureHashes))
		entry.sourceEventCount = uint32(len(m.SourceEventHashes))
		entry.formationAt = m.FormationAt
	case "ghosttrace.events.v1.AutomationGroupFormation":
		m := &eventsv1.AutomationGroupFormation{}
		if err := proto.Unmarshal(payload, m); err != nil {
			return nil, err
		}
		entry.actorRefs = m.ActorRefs
		entry.directInfluencedBy = m.DirectInfluencedBy
		entry.closureCount = uint32(len(m.ClosureHashes))
		entry.sourceEventCount = uint32(len(m.SourceEventHashes))
		entry.formationAt = m.FormationAt
	case "ghosttrace.events.v1.CampaignHypothesisFormation":
		m := &eventsv1.CampaignHypothesisFormation{}
		if err := proto.Unmarshal(payload, m); err != nil {
			return nil, err
		}
		entry.directInfluencedBy = m.DirectInfluencedBy
		entry.closureCount = uint32(len(m.ClosureHashes))
		entry.sourceEventCount = uint32(len(m.SourceEventHashes))
		entry.formationAt = m.FormationAt
		// CampaignHypothesis models event sets, not actor sets;
		// actor_refs left empty per ontology.
	case "ghosttrace.events.v1.CoordinationRingFormation":
		m := &eventsv1.CoordinationRingFormation{}
		if err := proto.Unmarshal(payload, m); err != nil {
			return nil, err
		}
		// CoordinationRingFormation models actor-interaction pairs
		// rather than a flat actor_refs list. Derive the unique
		// actor set from the interactions for morphology consumers
		// that want per-actor breakdown.
		actorSet := make(map[string]struct{})
		for _, ix := range m.Interactions {
			if ix == nil {
				continue
			}
			if ix.ActorA != "" {
				actorSet[ix.ActorA] = struct{}{}
			}
			if ix.ActorB != "" {
				actorSet[ix.ActorB] = struct{}{}
			}
		}
		entry.actorRefs = make([]string, 0, len(actorSet))
		for a := range actorSet {
			entry.actorRefs = append(entry.actorRefs, a)
		}
		entry.directInfluencedBy = m.DirectInfluencedBy
		entry.closureCount = uint32(len(m.ClosureHashes))
		entry.sourceEventCount = uint32(len(m.SourceEventHashes))
		entry.formationAt = m.FormationAt
	default:
		return nil, fmt.Errorf("unknown formation message type: %s", row.MessageType)
	}
	return entry, nil
}

// sortHashesAscending sorts a slice of 32-byte hashes ascending
// lexicographically per §0139 hash-list element-shape discipline.
// Used for deterministic Measurement.Hypotheses ordering.
func sortHashesAscending(hashes [][32]byte) {
	for i := 0; i < len(hashes); i++ {
		for j := i + 1; j < len(hashes); j++ {
			if bytesLess(hashes[i][:], hashes[j][:]) {
				continue
			}
			hashes[i], hashes[j] = hashes[j], hashes[i]
		}
	}
}

func bytesLess(a, b []byte) bool {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return len(a) < len(b)
}
