package projection

import (
	"sort"
)

// LatencyAggregate is the per-dimension percentile summary over the
// non-nil latency samples in a projection slice. SampleCount is the
// number of samples contributed; Min/P50/P90/Max are nil when
// SampleCount is zero. Mirrors the wire shape used by
// cmd/summarize-hypotheses and internal/httpapi/hypotheses_summary.
//
// Percentile method: nearest-rank. For a sorted ascending sample
// slice of length N, P_k is the value at index `ceil(k * N / 100) - 1`.
// Stable + simple + no interpolation. P50 of [10,20,30,40,50] is 30;
// P90 of the same is 50.
type LatencyAggregate struct {
	SampleCount int    `json:"sample_count"`
	MinNs       *int64 `json:"min_ns,omitempty"`
	P50Ns       *int64 `json:"p50_ns,omitempty"`
	P90Ns       *int64 `json:"p90_ns,omitempty"`
	MaxNs       *int64 `json:"max_ns,omitempty"`
}

// LatencyAggregates is the per-dimension latency aggregation for one
// section. Three dimensions mirror the per-projection latency fields
// landed at §0055 across all four Cat III subtypes.
type LatencyAggregates struct {
	FormationToFirstPromotionNs       LatencyAggregate `json:"formation_to_first_promotion_ns"`
	LatestPromotionToLatestDemotionNs LatencyAggregate `json:"latest_promotion_to_latest_demotion_ns"`
	FormationToDissolutionNs          LatencyAggregate `json:"formation_to_dissolution_ns"`
}

// LatencySamples carries the raw per-formation latency samples
// extracted from a projection slice. The combined latency aggregate
// per §0079 is computed from the UNION of per-subtype samples
// (exact percentiles), not from approximation over per-subtype
// aggregates.
type LatencySamples struct {
	FormationToFirstPromotion       []int64
	LatestPromotionToLatestDemotion []int64
	FormationToDissolution          []int64
}

// SubtypeAggregate carries both the state counts and the raw latency
// samples for one subtype, plus a Section() helper that emits the
// JSON wire shape.
type SubtypeAggregate struct {
	Counts  StateCounts
	Samples LatencySamples
}

// AggregateSection is the wire shape of every per-subtype + combined
// JSON section emitted by cmd/summarize-hypotheses and
// internal/httpapi/hypotheses_summary. Embeds StateCounts so its
// `total` + `by_state` fields appear at the parent level; Latencies
// nests under `latencies`.
type AggregateSection struct {
	StateCounts
	Latencies LatencyAggregates `json:"latencies"`
}

// Section returns the wire-shape AggregateSection for this subtype.
func (a SubtypeAggregate) Section() AggregateSection {
	return AggregateSection{
		StateCounts: a.Counts,
		Latencies:   AggregateAllLatencies(a.Samples),
	}
}

// AggregateAllLatencies returns the per-dimension LatencyAggregates
// over the supplied raw samples.
func AggregateAllLatencies(s LatencySamples) LatencyAggregates {
	return LatencyAggregates{
		FormationToFirstPromotionNs:       AggregateLatencies(s.FormationToFirstPromotion),
		LatestPromotionToLatestDemotionNs: AggregateLatencies(s.LatestPromotionToLatestDemotion),
		FormationToDissolutionNs:          AggregateLatencies(s.FormationToDissolution),
	}
}

// AggregateBC builds a SubtypeAggregate from a BehavioralCluster
// projection slice (as returned by ListHypotheses). Extracts counts
// + raw latency samples in a single pass.
func AggregateBC(projs []HypothesisProjection) SubtypeAggregate {
	a := SubtypeAggregate{Counts: emptyStateCounts()}
	for _, p := range projs {
		a.Counts.Total++
		a.Counts.ByState[p.State]++
		if p.FormationToFirstPromotionLatencyNs != nil {
			a.Samples.FormationToFirstPromotion = append(a.Samples.FormationToFirstPromotion, *p.FormationToFirstPromotionLatencyNs)
		}
		if p.LatestPromotionToLatestDemotionLatencyNs != nil {
			a.Samples.LatestPromotionToLatestDemotion = append(a.Samples.LatestPromotionToLatestDemotion, *p.LatestPromotionToLatestDemotionLatencyNs)
		}
		if p.FormationToDissolutionLatencyNs != nil {
			a.Samples.FormationToDissolution = append(a.Samples.FormationToDissolution, *p.FormationToDissolutionLatencyNs)
		}
	}
	return a
}

// AggregateAG builds a SubtypeAggregate from an AutomationGroup
// projection slice.
func AggregateAG(projs []AutomationGroupProjection) SubtypeAggregate {
	a := SubtypeAggregate{Counts: emptyStateCounts()}
	for _, p := range projs {
		a.Counts.Total++
		a.Counts.ByState[p.State]++
		if p.FormationToFirstPromotionLatencyNs != nil {
			a.Samples.FormationToFirstPromotion = append(a.Samples.FormationToFirstPromotion, *p.FormationToFirstPromotionLatencyNs)
		}
		if p.LatestPromotionToLatestDemotionLatencyNs != nil {
			a.Samples.LatestPromotionToLatestDemotion = append(a.Samples.LatestPromotionToLatestDemotion, *p.LatestPromotionToLatestDemotionLatencyNs)
		}
		if p.FormationToDissolutionLatencyNs != nil {
			a.Samples.FormationToDissolution = append(a.Samples.FormationToDissolution, *p.FormationToDissolutionLatencyNs)
		}
	}
	return a
}

// AggregateCH builds a SubtypeAggregate from a CampaignHypothesis
// projection slice.
func AggregateCH(projs []CampaignHypothesisProjection) SubtypeAggregate {
	a := SubtypeAggregate{Counts: emptyStateCounts()}
	for _, p := range projs {
		a.Counts.Total++
		a.Counts.ByState[p.State]++
		if p.FormationToFirstPromotionLatencyNs != nil {
			a.Samples.FormationToFirstPromotion = append(a.Samples.FormationToFirstPromotion, *p.FormationToFirstPromotionLatencyNs)
		}
		if p.LatestPromotionToLatestDemotionLatencyNs != nil {
			a.Samples.LatestPromotionToLatestDemotion = append(a.Samples.LatestPromotionToLatestDemotion, *p.LatestPromotionToLatestDemotionLatencyNs)
		}
		if p.FormationToDissolutionLatencyNs != nil {
			a.Samples.FormationToDissolution = append(a.Samples.FormationToDissolution, *p.FormationToDissolutionLatencyNs)
		}
	}
	return a
}

// AggregateCR builds a SubtypeAggregate from a CoordinationRing
// projection slice.
func AggregateCR(projs []CoordinationRingProjection) SubtypeAggregate {
	a := SubtypeAggregate{Counts: emptyStateCounts()}
	for _, p := range projs {
		a.Counts.Total++
		a.Counts.ByState[p.State]++
		if p.FormationToFirstPromotionLatencyNs != nil {
			a.Samples.FormationToFirstPromotion = append(a.Samples.FormationToFirstPromotion, *p.FormationToFirstPromotionLatencyNs)
		}
		if p.LatestPromotionToLatestDemotionLatencyNs != nil {
			a.Samples.LatestPromotionToLatestDemotion = append(a.Samples.LatestPromotionToLatestDemotion, *p.LatestPromotionToLatestDemotionLatencyNs)
		}
		if p.FormationToDissolutionLatencyNs != nil {
			a.Samples.FormationToDissolution = append(a.Samples.FormationToDissolution, *p.FormationToDissolutionLatencyNs)
		}
	}
	return a
}

// CombineCounts returns a per-state-aligned sum across StateCounts.
// Per §0078: combined.total equals the sum of subtype totals;
// combined.by_state[s] equals the sum of each subtype's by_state[s]
// for every state. Initializes every State enum key.
func CombineCounts(parts ...StateCounts) StateCounts {
	combined := emptyStateCounts()
	for _, p := range parts {
		combined.Total += p.Total
		for state, count := range p.ByState {
			combined.ByState[state] += count
		}
	}
	return combined
}

// CombineLatencySamples returns the union of per-dimension samples
// across LatencySamples. The combined latency aggregate is computed
// from this union (NOT approximated from per-subtype aggregates) per
// §0079 — combined percentiles are exact.
func CombineLatencySamples(parts ...LatencySamples) LatencySamples {
	combined := LatencySamples{}
	for _, p := range parts {
		combined.FormationToFirstPromotion = append(combined.FormationToFirstPromotion, p.FormationToFirstPromotion...)
		combined.LatestPromotionToLatestDemotion = append(combined.LatestPromotionToLatestDemotion, p.LatestPromotionToLatestDemotion...)
		combined.FormationToDissolution = append(combined.FormationToDissolution, p.FormationToDissolution...)
	}
	return combined
}

// AggregateLatencies returns a nearest-rank percentile summary over
// the sample slice. Returns SampleCount=0 + nil percentile fields
// when samples is empty. The caller's sample slice is NOT mutated.
func AggregateLatencies(samples []int64) LatencyAggregate {
	if len(samples) == 0 {
		return LatencyAggregate{}
	}
	cp := append([]int64(nil), samples...)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })

	min := cp[0]
	max := cp[len(cp)-1]
	p50 := PercentileNearestRank(cp, 50)
	p90 := PercentileNearestRank(cp, 90)

	return LatencyAggregate{
		SampleCount: len(cp),
		MinNs:       &min,
		P50Ns:       &p50,
		P90Ns:       &p90,
		MaxNs:       &max,
	}
}

// PercentileNearestRank returns the nearest-rank percentile over a
// pre-sorted ascending slice. Pre-condition: len(sorted) >= 1.
// Method: P_k = sorted[ceil(k * N / 100) - 1], clamped to [0, N-1].
func PercentileNearestRank(sorted []int64, k int) int64 {
	n := len(sorted)
	idx := (k*n + 99) / 100 // ceil(k*n/100)
	idx--
	if idx < 0 {
		idx = 0
	}
	if idx >= n {
		idx = n - 1
	}
	return sorted[idx]
}

// emptyStateCounts returns a StateCounts with every State enum key
// initialized to zero. Used by CombineCounts + the four Aggregate*
// helpers to preserve the §0053 + §0078 predictable-wire-shape
// commitment (every State key present in by_state, even at zero).
func emptyStateCounts() StateCounts {
	return StateCounts{
		ByState: map[State]int{
			StateForming:    0,
			StatePromoted:   0,
			StateDemoted:    0,
			StateDissolved:  0,
			StateMergedInto: 0,
			StateSplitInto:  0,
		},
	}
}
