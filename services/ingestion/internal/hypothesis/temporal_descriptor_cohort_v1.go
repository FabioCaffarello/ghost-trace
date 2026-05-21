package hypothesis

import (
	"bytes"
	"fmt"
	"sort"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
)

// TemporalDescriptorCohortV1Signature is the stable pattern
// signature for the first canonical CampaignHypothesis formation
// pattern. Inference: events with byte-identical session_descriptor
// arriving within a maximum inter-event gap form a "campaign" —
// a temporally-coherent cohort sharing a thematic descriptor.
//
// Minimum-viable canonical example for CampaignHypothesis
// formation — analogous to session-descriptor-shared-v1 (§0045 BC)
// and uniform-cadence-v1 (§0056 AG).
const TemporalDescriptorCohortV1Signature = "temporal-descriptor-cohort-v1"

// TemporalDescriptorCohortV1 is the first canonical
// CampaignHypothesis formation pattern.
type TemporalDescriptorCohortV1 struct {
	// MinCampaignSize is the minimum number of events in a cohort
	// for it to qualify as a campaign. Default 3.
	MinCampaignSize int64

	// MaxIntraEventGapSeconds is the maximum elapsed seconds
	// between consecutive events (sorted by declared_at) for them
	// to remain in the same cohort. Default 300 (5 minutes).
	MaxIntraEventGapSeconds int64
}

// Signature implements CampaignHypothesisFormationPattern.
func (p TemporalDescriptorCohortV1) Signature() string {
	return TemporalDescriptorCohortV1Signature
}

// Parameters implements CampaignHypothesisFormationPattern.
// Canonical form: keys sorted alphabetically.
func (p TemporalDescriptorCohortV1) Parameters() string {
	return fmt.Sprintf("max_intra_event_gap_seconds=%d;min_campaign_size=%d",
		p.MaxIntraEventGapSeconds, p.MinCampaignSize)
}

// campaignMember is the per-event tuple used by the cohort scanner.
type campaignMember struct {
	hash       [32]byte
	declaredAt int64
}

// Form implements CampaignHypothesisFormationPattern. Groups
// DeclaredSessions by session_descriptor; within each group, scans
// chronologically and emits one CampaignHypothesisFormation per
// cohort meeting the size + gap criteria.
//
// formation_at is derived deterministically as the max declared_at
// across the cohort. The caller-supplied formationAt argument is
// IGNORED to preserve hypothesis-identity stability per §0045.
func (p TemporalDescriptorCohortV1) Form(fctx CampaignHypothesisFormationContext, _ int64) []*eventsv1.CampaignHypothesisFormation {
	groups := map[string][]campaignMember{}
	for _, src := range fctx.DeclaredSessions() {
		descriptor := src.Session.GetSessionDescriptor()
		key := string(descriptor)
		groups[key] = append(groups[key], campaignMember{
			hash:       src.Hash,
			declaredAt: src.Session.GetDeclaredAt(),
		})
	}

	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	gapNs := p.MaxIntraEventGapSeconds * int64(1e9)
	var formations []*eventsv1.CampaignHypothesisFormation
	for _, key := range keys {
		members := groups[key]
		sort.Slice(members, func(i, j int) bool {
			return members[i].declaredAt < members[j].declaredAt
		})

		var cohort []campaignMember
		flush := func() {
			if int64(len(cohort)) < p.MinCampaignSize {
				cohort = nil
				return
			}
			formations = append(formations, buildCampaignFormation(cohort))
			cohort = nil
		}
		for _, m := range members {
			if len(cohort) == 0 {
				cohort = append(cohort, m)
				continue
			}
			last := cohort[len(cohort)-1]
			if m.declaredAt-last.declaredAt > gapNs {
				flush()
			}
			cohort = append(cohort, m)
		}
		flush()
	}

	return formations
}

// buildCampaignFormation constructs a CampaignHypothesisFormation
// from a sorted-by-declaredAt cohort. Sorts source_event_hashes
// ascending for content-hash stability.
func buildCampaignFormation(cohort []campaignMember) *eventsv1.CampaignHypothesisFormation {
	maxDeclaredAt := int64(0)
	for _, m := range cohort {
		if m.declaredAt > maxDeclaredAt {
			maxDeclaredAt = m.declaredAt
		}
	}
	hashes := make([][32]byte, 0, len(cohort))
	seen := map[[32]byte]struct{}{}
	for _, m := range cohort {
		if _, dup := seen[m.hash]; dup {
			continue
		}
		seen[m.hash] = struct{}{}
		hashes = append(hashes, m.hash)
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
	return &eventsv1.CampaignHypothesisFormation{
		FormationAt:       maxDeclaredAt,
		Confidence:        confidenceFromClusterSize(len(hashes)),
		SourceEventHashes: hashBytes,
	}
}

// Ensure TemporalDescriptorCohortV1 satisfies
// CampaignHypothesisFormationPattern at compile time.
var _ CampaignHypothesisFormationPattern = TemporalDescriptorCohortV1{}
