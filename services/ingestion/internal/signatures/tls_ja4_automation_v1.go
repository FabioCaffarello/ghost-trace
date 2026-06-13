// tls_ja4_automation_v1 is an F3 signature per decision-log §0221: a
// Camada A canonical-aberta signature recognizing AutomationGroup
// formation candidates by matching an actor's observed TLS ClientHello
// fingerprint (JA4 and/or JA3) against an operator-supplied set of
// fingerprints associated with automated (non-human) client stacks.
// First TLS-modality NetworkSignature; mirrors §0152's
// cdp_marker_density_v1 single-actor shape on the NetworkObservation
// tls_ja4 sub-modality side per §0144 discriminated-union framing.
//
// Detection axis: a fingerprint match is a single-observation indicator
// — automation tooling (curl, Go net/http, python-requests, headless
// stacks) emits a TLS ClientHello whose cipher / extension / curve
// ordering differs from mainstream browsers, yielding a stable JA3/JA4
// digest. Unlike the multi-actor clustering of tcp_fingerprint_-
// clustering_v1, this signature is single-actor: a known-automation
// fingerprint on one actor forms a single-actor AutomationGroup
// candidate ("this client is probably automated"), per the user-facing
// inference shape of the §0221 vertical slice.
//
// Constitutional posture on the known-set (§3 N1 truth-vs-structure):
// the signature provides the MATCH MECHANISM; the judgment of WHICH
// fingerprints indicate automation is OPERATOR POLICY, supplied via
// KnownJA4 / KnownJA3. The default (empty) set emits zero candidates —
// the signature does not bake an unverifiable "fingerprint X is a bot"
// claim into open code. Operators curate the reference set out-of-band
// (the same way an allow/deny list is operator-owned). The committed
// hypothesis still carries the §2.6 paired dimensions + §2.3 provenance
// back to the matched Cat I observation, so the inference remains fully
// auditable regardless of how the reference set was curated.
//
// MVP threshold: an actor with >= 1 matching observation triggers a
// single-actor AutomationGroup candidate (default Threshold 1), per
// §0023 + §0141 conservative-default discipline.
package signatures

import (
	"context"
	"fmt"
	"sort"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
)

// TLSJa4AutomationV1 implements the tls_ja4_automation_v1 signature.
// Stateless; instances are reusable across invocations. The KnownJA4 /
// KnownJA3 reference sets are operator-supplied (see the constitutional
// note in the package-doc above); an empty set means "no fingerprint is
// treated as automation" and the signature emits zero candidates.
type TLSJa4AutomationV1 struct {
	// KnownJA4 is the set of JA4 fingerprint digests treated as
	// automation indicators. Keyed by the exact `ja4` string.
	KnownJA4 map[string]struct{}
	// KnownJA3 is the set of JA3 fingerprint digests treated as
	// automation indicators. Keyed by the exact `ja3` string.
	KnownJA3 map[string]struct{}
	// Threshold is the minimum count of matching observations per actor
	// for a candidate to be emitted. Default 1 (a single match is the
	// signal for this single-observation signature). 0 = use default.
	Threshold uint32
}

// Name identifies the signature for instrumentation + versioning.
func (s *TLSJa4AutomationV1) Name() string { return "tls_ja4_automation_v1" }

// Subtype identifies the Cat III hypothesis subtype this signature
// produces candidates for: AutomationGroup.
func (s *TLSJa4AutomationV1) Subtype() HypothesisSubtype {
	return HypothesisSubtypeAutomationGroup
}

// isNetworkSignature is the NetworkSignature marker.
func (s *TLSJa4AutomationV1) isNetworkSignature() {}

// effectiveThreshold returns Threshold or the default (1) when unset.
func (s *TLSJa4AutomationV1) effectiveThreshold() uint32 {
	if s.Threshold == 0 {
		return 1
	}
	return s.Threshold
}

// EffectiveThreshold exposes the applied threshold (Threshold or the
// default 1) for orchestrator-facing reporting.
func (s *TLSJa4AutomationV1) EffectiveThreshold() uint32 { return s.effectiveThreshold() }

// matches reports whether the tls_ja4 payload's JA4 or JA3 digest is in
// the operator-supplied reference sets.
func (s *TLSJa4AutomationV1) matches(tls *eventsv1.NetworkTlsJa4) bool {
	if tls == nil {
		return false
	}
	if ja4 := tls.GetJa4(); ja4 != "" && s.KnownJA4 != nil {
		if _, ok := s.KnownJA4[ja4]; ok {
			return true
		}
	}
	if ja3 := tls.GetJa3(); ja3 != "" && s.KnownJA3 != nil {
		if _, ok := s.KnownJA3[ja3]; ok {
			return true
		}
	}
	return false
}

// EvaluateNetwork evaluates the signature against a slice of
// NetworkObservation records. Per-actor aggregation over tls_ja4
// observations whose fingerprint matches the operator-supplied known
// set; emits a single-actor FormationCandidate per actor whose match
// count meets the threshold.
//
// Returns candidates in actor_ref alphabetical order (deterministic
// fixtures / reproducible orchestrator behavior) + EvaluationStats per
// §0143 Sub-benchmark 1 instrumentation.
//
// Skip semantics (mirrors tcp_fingerprint_clustering_v1):
//   - nil observation: skipped (no counter).
//   - empty actor_ref (and no Cat II attribution): ObservationsSkippedNoActor++.
//   - non-tls_ja4 modality: ObservationsSkippedWrongModality++.
//   - tls_ja4 present but fingerprint NOT in the known set: counted as
//     scanned + actor aggregated, but contributes no match (the actor
//     only becomes a candidate if its match count >= threshold).
//
// Per §0168 Decision A.1: when obs.ActorRef is empty AND attribution !=
// nil AND attribution.For(obs_hash) resolves, the derived actor is used
// and the Cat II derivation hash is threaded into SourceHashes alongside
// the Cat I observation hash (preserving the §2.3 provenance chain).
func (s *TLSJa4AutomationV1) EvaluateNetwork(ctx context.Context, observations []*eventsv1.NetworkObservation, attribution AttributionLookup) (*EvaluationResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	threshold := s.effectiveThreshold()
	stats := EvaluationStats{
		ObservationsScanned: uint32(len(observations)),
		PerCollector:        make(map[string]uint32),
	}

	type aggregate struct {
		matchCount   uint32
		sourceHashes [][]byte
	}
	perActor := make(map[string]*aggregate)
	distinctActors := make(map[string]struct{})

	for _, obs := range observations {
		if obs == nil {
			continue
		}
		if obs.CollectorRef != "" {
			stats.PerCollector[obs.CollectorRef]++
		}

		_, obsHash, err := canonical.MarshalAndHash(obs)
		if err != nil {
			return nil, fmt.Errorf("tls_ja4_automation_v1: hash observation: %w", err)
		}

		// Resolve effective actor: declared (Cat I) OR derived (Cat II
		// via AttributionLookup) per §0168.
		effectiveActor := obs.ActorRef
		var attributionHash [32]byte
		var attributionPresent bool
		if effectiveActor == "" && attribution != nil {
			if derived, attHash, ok := attribution.For(obsHash); ok {
				effectiveActor = derived
				attributionHash = attHash
				attributionPresent = true
			}
		}
		if effectiveActor == "" {
			stats.ObservationsSkippedNoActor++
			continue
		}

		tls := obs.GetTlsJa4()
		if tls == nil {
			stats.ObservationsSkippedWrongModality++
			continue
		}
		distinctActors[effectiveActor] = struct{}{}
		if !s.matches(tls) {
			// In-modality but not a known-automation fingerprint: the
			// actor is aggregated (so ActorsAggregated reflects the TLS
			// population) but contributes no match toward the threshold.
			continue
		}

		agg, ok := perActor[effectiveActor]
		if !ok {
			agg = &aggregate{}
			perActor[effectiveActor] = agg
		}
		agg.matchCount++

		obsHashCopy := make([]byte, 32)
		copy(obsHashCopy, obsHash[:])
		agg.sourceHashes = append(agg.sourceHashes, obsHashCopy)

		if attributionPresent {
			attHashCopy := make([]byte, 32)
			copy(attHashCopy, attributionHash[:])
			agg.sourceHashes = append(agg.sourceHashes, attHashCopy)
		}
	}
	stats.ActorsAggregated = uint32(len(distinctActors))

	actors := make([]string, 0, len(perActor))
	for actor := range perActor {
		actors = append(actors, actor)
	}
	sort.Strings(actors)

	out := make([]*FormationCandidate, 0)
	for _, actor := range actors {
		agg := perActor[actor]
		if agg.matchCount < threshold {
			continue
		}
		stats.ActorsAboveThreshold++
		sort.Slice(agg.sourceHashes, func(i, j int) bool {
			return bytesLess(agg.sourceHashes[i], agg.sourceHashes[j])
		})
		out = append(out, &FormationCandidate{
			SignatureName:     s.Name(),
			HypothesisSubtype: s.Subtype(),
			ActorRefs:         []string{actor},
			SourceHashes:      agg.sourceHashes,
			EvidenceCount:     uint32(len(agg.sourceHashes)),
			// ConfidenceHint reuses the cdp_marker_density_v1 scale:
			// linear from 0.5 at threshold to 0.9 at 5×threshold, capped
			// at 0.9 (§3 N1 — never claims certainty). Input is the
			// per-actor match count.
			ConfidenceHint: confidenceFromCount(agg.matchCount, threshold),
		})
	}
	stats.CandidatesEmitted = uint32(len(out))
	return &EvaluationResult{Candidates: out, Stats: stats}, nil
}
