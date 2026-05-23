package hypothesis

import (
	"bytes"
	"fmt"
	"sort"

	commonv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/common/v1"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
)

// CoOccurrenceWindowV1Signature is the stable pattern signature
// for the first canonical CoordinationRing formation pattern.
// Inference: pairs of actors whose declared_at timestamps fall
// within a co-occurrence window on the same session_descriptor
// form an interaction edge. The set of edges meeting a minimum
// support count constitutes a CoordinationRing.
//
// Minimum-viable canonical example for CoordinationRing formation
// — analogous to session-descriptor-shared-v1 (§0045 BC),
// uniform-cadence-v1 (§0056 AG), and temporal-descriptor-cohort-v1
// (§0063 CH).
const CoOccurrenceWindowV1Signature = "co-occurrence-window-v1"

// CoOccurrenceWindowV1 is the first canonical CoordinationRing
// formation pattern.
type CoOccurrenceWindowV1 struct {
	// MinEdgeSupport is the minimum number of co-occurrence
	// observations between the same actor pair for the edge to
	// qualify. Default 3.
	MinEdgeSupport int64

	// MaxWindowSeconds is the maximum elapsed seconds between two
	// declared sessions sharing a descriptor for them to count as
	// a single co-occurrence. Default 600 (10 minutes).
	MaxWindowSeconds int64
}

// Signature implements CoordinationRingFormationPattern.
func (p CoOccurrenceWindowV1) Signature() string {
	return CoOccurrenceWindowV1Signature
}

// Parameters implements CoordinationRingFormationPattern.
// Canonical form: keys sorted alphabetically.
func (p CoOccurrenceWindowV1) Parameters() string {
	return fmt.Sprintf("max_window_seconds=%d;min_edge_support=%d",
		p.MaxWindowSeconds, p.MinEdgeSupport)
}

// edgeKey is the canonical undirected actor pair, lex-ordered.
type edgeKey struct {
	a, b string
}

// canonicalEdge returns the lex-ordered (a, b) pair.
func canonicalEdge(x, y string) (edgeKey, bool) {
	if x == y {
		return edgeKey{}, false
	}
	if x < y {
		return edgeKey{a: x, b: y}, true
	}
	return edgeKey{a: y, b: x}, true
}

// ringEdge accumulates the support count + the contributing
// observation hashes for an actor pair.
type ringEdge struct {
	supports    int64
	sourceHashes map[[32]byte]struct{}
}

// Form implements CoordinationRingFormationPattern. Groups
// DeclaredSessions by session_descriptor; within each group, scans
// all unordered actor pairs whose declared_at timestamps fall
// within MaxWindowSeconds; emits ONE CoordinationRingFormation per
// connected interaction component whose edges all meet
// MinEdgeSupport.
//
// formation_at is derived deterministically as the max declared_at
// across the ring's contributing observations. The caller-supplied
// formationAt argument is IGNORED to preserve hypothesis-identity
// stability per §0045+§0063.
func (p CoOccurrenceWindowV1) Form(fctx CoordinationRingFormationContext, _ int64) []*eventsv1.CoordinationRingFormation {
	type member struct {
		hash       [32]byte
		actorRef   string
		declaredAt int64
	}

	groups := map[string][]member{}
	for _, src := range fctx.DeclaredSessions() {
		descriptor := string(src.Session.GetSessionDescriptor())
		actor := src.Session.GetActorRef()
		if actor == "" {
			continue
		}
		groups[descriptor] = append(groups[descriptor], member{
			hash:       src.Hash,
			actorRef:   actor,
			declaredAt: src.Session.GetDeclaredAt(),
		})
	}

	descriptorKeys := make([]string, 0, len(groups))
	for k := range groups {
		descriptorKeys = append(descriptorKeys, k)
	}
	sort.Strings(descriptorKeys)

	windowNs := p.MaxWindowSeconds * int64(1e9)
	edges := map[edgeKey]*ringEdge{}
	maxDeclaredAt := map[edgeKey]int64{}

	for _, key := range descriptorKeys {
		members := groups[key]
		sort.Slice(members, func(i, j int) bool {
			return members[i].declaredAt < members[j].declaredAt
		})
		for i := 0; i < len(members); i++ {
			for j := i + 1; j < len(members); j++ {
				if members[j].declaredAt-members[i].declaredAt > windowNs {
					break
				}
				ek, ok := canonicalEdge(members[i].actorRef, members[j].actorRef)
				if !ok {
					continue
				}
				e, exists := edges[ek]
				if !exists {
					e = &ringEdge{sourceHashes: map[[32]byte]struct{}{}}
					edges[ek] = e
				}
				e.supports++
				e.sourceHashes[members[i].hash] = struct{}{}
				e.sourceHashes[members[j].hash] = struct{}{}
				if members[j].declaredAt > maxDeclaredAt[ek] {
					maxDeclaredAt[ek] = members[j].declaredAt
				}
			}
		}
	}

	// Filter edges by support threshold, then connected components.
	type edgeRecord struct {
		key          edgeKey
		sourceHashes map[[32]byte]struct{}
		maxAt        int64
	}
	var qualifying []edgeRecord
	for ek, e := range edges {
		if e.supports < p.MinEdgeSupport {
			continue
		}
		qualifying = append(qualifying, edgeRecord{
			key:          ek,
			sourceHashes: e.sourceHashes,
			maxAt:        maxDeclaredAt[ek],
		})
	}
	sort.Slice(qualifying, func(i, j int) bool {
		if qualifying[i].key.a != qualifying[j].key.a {
			return qualifying[i].key.a < qualifying[j].key.a
		}
		return qualifying[i].key.b < qualifying[j].key.b
	})

	// Union-find over qualifying edges -> connected components.
	parent := map[string]string{}
	var find func(x string) string
	find = func(x string) string {
		if parent[x] == "" {
			parent[x] = x
			return x
		}
		if parent[x] == x {
			return x
		}
		root := find(parent[x])
		parent[x] = root
		return root
	}
	union := func(x, y string) {
		rx, ry := find(x), find(y)
		if rx != ry {
			parent[rx] = ry
		}
	}
	for _, e := range qualifying {
		union(e.key.a, e.key.b)
	}

	type component struct {
		edges        []edgeKey
		sourceHashes map[[32]byte]struct{}
		maxAt        int64
	}
	components := map[string]*component{}
	for _, e := range qualifying {
		root := find(e.key.a)
		c, ok := components[root]
		if !ok {
			c = &component{sourceHashes: map[[32]byte]struct{}{}}
			components[root] = c
		}
		c.edges = append(c.edges, e.key)
		for h := range e.sourceHashes {
			c.sourceHashes[h] = struct{}{}
		}
		if e.maxAt > c.maxAt {
			c.maxAt = e.maxAt
		}
	}

	rootKeys := make([]string, 0, len(components))
	for k := range components {
		rootKeys = append(rootKeys, k)
	}
	sort.Strings(rootKeys)

	var formations []*eventsv1.CoordinationRingFormation
	for _, root := range rootKeys {
		c := components[root]
		formations = append(formations, buildCoordinationRingFormation(c.edges, c.sourceHashes, c.maxAt))
	}
	return formations
}

// buildCoordinationRingFormation constructs a
// CoordinationRingFormation from a connected component. Sorts
// edges ascending by (actor_a, actor_b) and source_event_hashes
// ascending by byte-lex.
func buildCoordinationRingFormation(edges []edgeKey, sourceHashes map[[32]byte]struct{}, maxAt int64) *eventsv1.CoordinationRingFormation {
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].a != edges[j].a {
			return edges[i].a < edges[j].a
		}
		return edges[i].b < edges[j].b
	})
	interactions := make([]*eventsv1.CoordinationRingInteraction, 0, len(edges))
	for _, ek := range edges {
		interactions = append(interactions, &eventsv1.CoordinationRingInteraction{
			ActorA: ek.a,
			ActorB: ek.b,
		})
	}

	hashes := make([][32]byte, 0, len(sourceHashes))
	for h := range sourceHashes {
		hashes = append(hashes, h)
	}
	sort.Slice(hashes, func(i, j int) bool {
		return bytes.Compare(hashes[i][:], hashes[j][:]) < 0
	})
	hashBytes := make([][]byte, 0, len(hashes))
	for _, h := range hashes {
		cp := make([]byte, 32)
		copy(cp, h[:])
		hashBytes = append(hashBytes, cp)
	}

	return &eventsv1.CoordinationRingFormation{
		Interactions:      interactions,
		FormationAt:       maxAt,
		Confidence:        confidenceFromClusterSize(len(edges) + 1),
		// EvidentialIndependence per §0140 paired-dimension marshalling-
		// boundary enforcement: this formation reads only from Cat I
		// observations (DeclaredSessions + NetworkEvents in the
		// co-occurrence window) and does not read from any promoted
		// hypothesis. Per §0133 Q3-α formula, all Cat I roots in the
		// subject_ref_* chain are NOT reachable via any
		// influenced_by edge from a promoted hypothesis → numerator =
		// denominator = total Cat I roots → α = 1. The inception-
		// phase value is structurally fixed at 1/1 (full independence)
		// for any formation path that does not consume hypothesis
		// records; once such a path exists, α must be computed per
		// the formula.
		EvidentialIndependence: &commonv1.EvidentialIndependence{Numerator: 1, Denominator: 1},
		SourceEventHashes:      hashBytes,
	}
}

// Ensure CoOccurrenceWindowV1 satisfies
// CoordinationRingFormationPattern at compile time.
var _ CoordinationRingFormationPattern = CoOccurrenceWindowV1{}
