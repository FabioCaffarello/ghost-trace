package hypothesis

import (
	"bytes"
	"fmt"
	"sort"

	commonv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/common/v1"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
)

// SessionDescriptorSharedV1Signature is the stable pattern signature
// for the first canonical formation pattern. Inference: actors whose
// DeclaredSessions carry byte-identical session_descriptor payloads
// share an underlying operational pattern (per entity-model.md
// line 65 "behavioral patterns suggest operation by a common
// underlying entity").
//
// This is the minimum-viable canonical example for Category III
// formation — analogous to padded-v1 (§0043) being the minimum-viable
// canonical example for Category II derivation. Sophisticated
// patterns (multi-Cat-I-input, fuzzy similarity, temporal windowing)
// extend the same FormationPattern pathway mechanically.
const SessionDescriptorSharedV1Signature = "session-descriptor-shared-v1"

// SessionDescriptorSharedV1 is the first canonical formation pattern.
// Algorithm:
//
//  1. Group every DeclaredSession in the substrate by byte-equal
//     session_descriptor payload.
//  2. For each group, collect the distinct actor_refs (empty
//     actor_refs are excluded — an actor must be identifiable to
//     participate in a behavioral cluster).
//  3. Emit one BehavioralClusterFormation per group whose distinct-
//     actor count is >= MinClusterSize.
//
// Determinism: the grouping + sorting is byte-deterministic — same
// substrate state + same MinClusterSize produces content-hash-
// identical formation events.
//
// Confidence (placeholder pending Charter §2.6 redaction): computed
// as 1.0 - 1.0/cluster_size, so 2-actor clusters have confidence 0.5,
// 3-actor clusters 0.67, ..., asymptotically approaching 1.0 for
// large clusters. The numeric form is structural-placeholder; §2.6
// will refine the semantics.
type SessionDescriptorSharedV1 struct {
	MinClusterSize int64
}

// Signature implements FormationPattern.
func (s SessionDescriptorSharedV1) Signature() string { return SessionDescriptorSharedV1Signature }

// Parameters implements FormationPattern. Canonical form:
// "min_cluster_size=<int>".
func (s SessionDescriptorSharedV1) Parameters() string {
	return fmt.Sprintf("min_cluster_size=%d", s.MinClusterSize)
}

// Form implements FormationPattern. Groups DeclaredSessions by
// session_descriptor; emits one BehavioralClusterFormation per group
// with >= MinClusterSize distinct actor_refs.
//
// formation_at is derived deterministically as the max declared_at
// across the cluster's contributing DeclaredSessions — the "inference
// window closes at the last contributing observation" semantics. The
// caller-supplied formationAt argument is IGNORED on purpose: the
// formation event's content-hash is the hypothesis's identity per
// Charter §2.3, and re-running with the same substrate state MUST
// produce the same hypothesis identity (the formation_at field must
// be content-deterministic).
func (s SessionDescriptorSharedV1) Form(fctx FormationContext, _ int64) []*eventsv1.BehavioralClusterFormation {
	type member struct {
		actorRef   string
		hash       [32]byte
		declaredAt int64
	}

	// Group by session_descriptor (key is the bytes-as-string for map
	// lookup; preserves byte-identity discipline).
	groups := map[string][]member{}
	for _, src := range fctx.DeclaredSessions() {
		actorRef := src.Session.GetActorRef()
		if actorRef == "" {
			continue // unattributed sessions cannot participate
		}
		descriptor := src.Session.GetSessionDescriptor()
		key := string(descriptor)
		groups[key] = append(groups[key], member{
			actorRef:   actorRef,
			hash:       src.Hash,
			declaredAt: src.Session.GetDeclaredAt(),
		})
	}

	// Iterate groups in deterministic order (sorted by descriptor key
	// ascending) so the output slice is deterministic.
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var formations []*eventsv1.BehavioralClusterFormation
	for _, key := range keys {
		members := groups[key]
		// Deduplicate actor_refs; collect contributing source hashes
		// + the max declared_at across the group.
		actorSet := map[string]struct{}{}
		hashSet := map[[32]byte]struct{}{}
		var maxDeclaredAt int64
		for _, m := range members {
			actorSet[m.actorRef] = struct{}{}
			hashSet[m.hash] = struct{}{}
			if m.declaredAt > maxDeclaredAt {
				maxDeclaredAt = m.declaredAt
			}
		}
		if int64(len(actorSet)) < s.MinClusterSize {
			continue
		}

		actors := make([]string, 0, len(actorSet))
		for a := range actorSet {
			actors = append(actors, a)
		}
		sort.Strings(actors)

		hashes := make([][32]byte, 0, len(hashSet))
		for h := range hashSet {
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

		formations = append(formations, &eventsv1.BehavioralClusterFormation{
			ActorRefs:   actors,
			FormationAt: maxDeclaredAt,
			Confidence:  confidenceFromClusterSize(len(actors)),
			// EvidentialIndependence per §0140 paired-dimension
			// marshalling-boundary enforcement: α = 1/1 (full
			// independence) — this pattern reads only from Cat I
			// DeclaredSessions and does not consume promoted
			// hypothesis records, so all Cat I roots in the chain
			// are NOT reachable via any promoted-hypothesis
			// influenced_by edge per §0133 Q3-α formula. Once a
			// formation path consumes hypothesis records, α must
			// be computed per the formula.
			EvidentialIndependence: &commonv1.EvidentialIndependence{Numerator: 1, Denominator: 1},
			SourceEventHashes:      hashBytes,
		})
	}

	return formations
}

// confidenceFromClusterSize is the placeholder confidence formula
// pending Charter §2.6 redaction: 1.0 - 1.0/size. Returns 0.0 for
// degenerate sizes (caller filters via MinClusterSize anyway).
func confidenceFromClusterSize(size int) float32 {
	if size <= 0 {
		return 0
	}
	return float32(1.0 - 1.0/float64(size))
}

// Ensure SessionDescriptorSharedV1 satisfies FormationPattern at
// compile time.
var _ FormationPattern = SessionDescriptorSharedV1{}
