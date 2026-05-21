// Command summarize-hypotheses returns aggregate counters + latency
// aggregates over the projection of every Category III hypothesis in
// the substrate, per Charter §2.5 BC3 + decision-log §0053 +
// decision-log §0078 (combined cross-subtype counters) + decision-log
// §0079 (per-subtype + combined latency aggregates).
//
// Output: structured JSON to stdout with a top-level `combined`
// section and four per-subtype sections (`behavioral_cluster`,
// `automation_group`, `campaign_hypothesis`, `coordination_ring`).
// Each section carries `total`, `by_state` (every State key present;
// predictable wire shape per §0053), and `latencies` (per-dimension
// LatencyAggregate per §0079).
//
// Per the §0053 equivalence invariant: for every State value, the
// count in each section's `by_state` equals the length of
// `list-hypotheses -state <state> -subtype <subtype>` (or no subtype
// filter for the combined section).
//
// Per §0079 the combined latency aggregate is computed from the
// UNION of per-subtype samples — not approximated from the per-subtype
// aggregates — so combined percentiles are exact.
//
// Exit codes: 0 success (including empty substrate); 2 tool/config
// error.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/projection"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

const exitToolError = 2

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "summarize-hypotheses: %v\n", err)
		os.Exit(exitToolError)
	}
}

func run() error {
	dbPath := flag.String("db", "./ghost-trace.db", "SQLite primary-event-log path")
	blobDir := flag.String("blobs", "./blobs", "content-addressed blob-store directory")
	afterNs := flag.Int64("after-ns", 0, "inclusive lower bound (Unix nanoseconds) on the latest event_time of each projection; 0 disables the lower bound")
	beforeNs := flag.Int64("before-ns", 0, "inclusive upper bound (Unix nanoseconds) on the latest event_time of each projection; 0 disables the upper bound")
	flag.Parse()

	if *afterNs < 0 {
		return fmt.Errorf("--after-ns must be non-negative; got %d", *afterNs)
	}
	if *beforeNs < 0 {
		return fmt.Errorf("--before-ns must be non-negative; got %d", *beforeNs)
	}
	if *afterNs != 0 && *beforeNs != 0 && *afterNs > *beforeNs {
		return fmt.Errorf("--after-ns (%d) must not exceed --before-ns (%d)", *afterNs, *beforeNs)
	}

	ctx := context.Background()
	sub, err := substrate.Open(ctx, *dbPath, *blobDir)
	if err != nil {
		return fmt.Errorf("open substrate: %w", err)
	}
	defer func() { _ = sub.Close() }()

	bcProjs, err := projection.ListHypotheses(ctx, sub, projection.ListOptions{
		TimeAfterNs:  *afterNs,
		TimeBeforeNs: *beforeNs,
	})
	if err != nil {
		return err
	}
	agProjs, err := projection.ListAutomationGroups(ctx, sub, projection.AutomationGroupListOptions{
		TimeAfterNs:  *afterNs,
		TimeBeforeNs: *beforeNs,
	})
	if err != nil {
		return err
	}
	chProjs, err := projection.ListCampaignHypotheses(ctx, sub, projection.CampaignHypothesisListOptions{
		TimeAfterNs:  *afterNs,
		TimeBeforeNs: *beforeNs,
	})
	if err != nil {
		return err
	}
	crProjs, err := projection.ListCoordinationRings(ctx, sub, projection.CoordinationRingListOptions{
		TimeAfterNs:  *afterNs,
		TimeBeforeNs: *beforeNs,
	})
	if err != nil {
		return err
	}

	bcAgg := bcAggregate(bcProjs)
	agAgg := agAggregate(agProjs)
	chAgg := chAggregate(chProjs)
	crAgg := crAggregate(crProjs)

	combinedCounts := combineCounts(
		bcAgg.counts, agAgg.counts, chAgg.counts, crAgg.counts,
	)
	combinedSamples := combineSamples(
		bcAgg.samples, agAgg.samples, chAgg.samples, crAgg.samples,
	)

	output := perSubtype{
		Combined: aggregateSection{
			StateCounts: combinedCounts,
			Latencies:   aggregateAllLatencies(combinedSamples),
		},
		BehavioralCluster:  bcAgg.section(),
		AutomationGroup:    agAgg.section(),
		CampaignHypothesis: chAgg.section(),
		CoordinationRing:   crAgg.section(),
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(output); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}

	fmt.Fprintf(os.Stderr,
		"summarize-hypotheses: combined_total=%d bc_total=%d ag_total=%d ch_total=%d cr_total=%d after_ns=%d before_ns=%d\n",
		combinedCounts.Total, bcAgg.counts.Total, agAgg.counts.Total,
		chAgg.counts.Total, crAgg.counts.Total,
		*afterNs, *beforeNs)
	return nil
}

// aggregateSection is the wire shape of every per-subtype + combined
// JSON section. StateCounts is embedded so its `total` + `by_state`
// fields appear at the parent level; Latencies nests under
// `latencies`.
type aggregateSection struct {
	projection.StateCounts
	Latencies latencyAggregates `json:"latencies"`
}

// latencyAggregates is the per-dimension latency aggregation for one
// section. The three dimensions mirror the per-projection latency
// fields landed at §0055 across all four subtypes.
type latencyAggregates struct {
	FormationToFirstPromotionNs       latencyAggregate `json:"formation_to_first_promotion_ns"`
	LatestPromotionToLatestDemotionNs latencyAggregate `json:"latest_promotion_to_latest_demotion_ns"`
	FormationToDissolutionNs          latencyAggregate `json:"formation_to_dissolution_ns"`
}

// latencyAggregate is the per-dimension percentile summary over the
// non-nil samples in a projection slice. SampleCount is the number of
// non-nil latencies contributed; Min/P50/P90/Max are nil when
// SampleCount is zero.
//
// Percentile method: nearest-rank. For a sorted ascending sample
// slice of length N, P_k is the value at index ceil(k * N / 100) - 1.
// P50 of [10,20,30,40,50] is 30; P90 of the same is 50. Method is
// stable + simple + does not interpolate between samples.
type latencyAggregate struct {
	SampleCount int    `json:"sample_count"`
	MinNs       *int64 `json:"min_ns,omitempty"`
	P50Ns       *int64 `json:"p50_ns,omitempty"`
	P90Ns       *int64 `json:"p90_ns,omitempty"`
	MaxNs       *int64 `json:"max_ns,omitempty"`
}

// latencySamples carries the raw per-formation latency samples
// extracted from a projection slice. The combined latency aggregate
// per §0079 is computed from the UNION of per-subtype samples (exact
// percentiles), not from approximation over per-subtype aggregates.
type latencySamples struct {
	FormationToFirstPromotion       []int64
	LatestPromotionToLatestDemotion []int64
	FormationToDissolution          []int64
}

// subtypeAggregate carries both the state counts and the raw latency
// samples for one subtype, plus a section() helper that emits the
// JSON wire shape.
type subtypeAggregate struct {
	counts  projection.StateCounts
	samples latencySamples
}

func (a subtypeAggregate) section() aggregateSection {
	return aggregateSection{
		StateCounts: a.counts,
		Latencies:   aggregateAllLatencies(a.samples),
	}
}

func aggregateAllLatencies(s latencySamples) latencyAggregates {
	return latencyAggregates{
		FormationToFirstPromotionNs:       aggregateLatencies(s.FormationToFirstPromotion),
		LatestPromotionToLatestDemotionNs: aggregateLatencies(s.LatestPromotionToLatestDemotion),
		FormationToDissolutionNs:          aggregateLatencies(s.FormationToDissolution),
	}
}

func bcAggregate(projs []projection.HypothesisProjection) subtypeAggregate {
	a := subtypeAggregate{counts: emptyCounts()}
	for _, p := range projs {
		a.counts.Total++
		a.counts.ByState[p.State]++
		if p.FormationToFirstPromotionLatencyNs != nil {
			a.samples.FormationToFirstPromotion = append(a.samples.FormationToFirstPromotion, *p.FormationToFirstPromotionLatencyNs)
		}
		if p.LatestPromotionToLatestDemotionLatencyNs != nil {
			a.samples.LatestPromotionToLatestDemotion = append(a.samples.LatestPromotionToLatestDemotion, *p.LatestPromotionToLatestDemotionLatencyNs)
		}
		if p.FormationToDissolutionLatencyNs != nil {
			a.samples.FormationToDissolution = append(a.samples.FormationToDissolution, *p.FormationToDissolutionLatencyNs)
		}
	}
	return a
}

func agAggregate(projs []projection.AutomationGroupProjection) subtypeAggregate {
	a := subtypeAggregate{counts: emptyCounts()}
	for _, p := range projs {
		a.counts.Total++
		a.counts.ByState[p.State]++
		if p.FormationToFirstPromotionLatencyNs != nil {
			a.samples.FormationToFirstPromotion = append(a.samples.FormationToFirstPromotion, *p.FormationToFirstPromotionLatencyNs)
		}
		if p.LatestPromotionToLatestDemotionLatencyNs != nil {
			a.samples.LatestPromotionToLatestDemotion = append(a.samples.LatestPromotionToLatestDemotion, *p.LatestPromotionToLatestDemotionLatencyNs)
		}
		if p.FormationToDissolutionLatencyNs != nil {
			a.samples.FormationToDissolution = append(a.samples.FormationToDissolution, *p.FormationToDissolutionLatencyNs)
		}
	}
	return a
}

func chAggregate(projs []projection.CampaignHypothesisProjection) subtypeAggregate {
	a := subtypeAggregate{counts: emptyCounts()}
	for _, p := range projs {
		a.counts.Total++
		a.counts.ByState[p.State]++
		if p.FormationToFirstPromotionLatencyNs != nil {
			a.samples.FormationToFirstPromotion = append(a.samples.FormationToFirstPromotion, *p.FormationToFirstPromotionLatencyNs)
		}
		if p.LatestPromotionToLatestDemotionLatencyNs != nil {
			a.samples.LatestPromotionToLatestDemotion = append(a.samples.LatestPromotionToLatestDemotion, *p.LatestPromotionToLatestDemotionLatencyNs)
		}
		if p.FormationToDissolutionLatencyNs != nil {
			a.samples.FormationToDissolution = append(a.samples.FormationToDissolution, *p.FormationToDissolutionLatencyNs)
		}
	}
	return a
}

func crAggregate(projs []projection.CoordinationRingProjection) subtypeAggregate {
	a := subtypeAggregate{counts: emptyCounts()}
	for _, p := range projs {
		a.counts.Total++
		a.counts.ByState[p.State]++
		if p.FormationToFirstPromotionLatencyNs != nil {
			a.samples.FormationToFirstPromotion = append(a.samples.FormationToFirstPromotion, *p.FormationToFirstPromotionLatencyNs)
		}
		if p.LatestPromotionToLatestDemotionLatencyNs != nil {
			a.samples.LatestPromotionToLatestDemotion = append(a.samples.LatestPromotionToLatestDemotion, *p.LatestPromotionToLatestDemotionLatencyNs)
		}
		if p.FormationToDissolutionLatencyNs != nil {
			a.samples.FormationToDissolution = append(a.samples.FormationToDissolution, *p.FormationToDissolutionLatencyNs)
		}
	}
	return a
}

func emptyCounts() projection.StateCounts {
	return projection.StateCounts{
		ByState: map[projection.State]int{
			projection.StateForming:    0,
			projection.StatePromoted:   0,
			projection.StateDemoted:    0,
			projection.StateDissolved:  0,
			projection.StateMergedInto: 0,
			projection.StateSplitInto:  0,
		},
	}
}

// aggregateLatencies returns a nearest-rank percentile summary over
// the sample slice. Returns sample_count=0 + nil percentile fields
// when samples is empty.
func aggregateLatencies(samples []int64) latencyAggregate {
	if len(samples) == 0 {
		return latencyAggregate{}
	}
	cp := append([]int64(nil), samples...)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })

	min := cp[0]
	max := cp[len(cp)-1]
	p50 := percentileNearestRank(cp, 50)
	p90 := percentileNearestRank(cp, 90)

	return latencyAggregate{
		SampleCount: len(cp),
		MinNs:       &min,
		P50Ns:       &p50,
		P90Ns:       &p90,
		MaxNs:       &max,
	}
}

// percentileNearestRank returns the nearest-rank percentile over a
// pre-sorted ascending slice. Pre-condition: len(sorted) >= 1.
func percentileNearestRank(sorted []int64, k int) int64 {
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

// combineCounts returns a per-state-aligned sum across all four
// per-subtype StateCounts. Per §0078: combined.total equals the sum
// of per-subtype totals; combined.by_state[s] equals the sum of each
// subtype's by_state[s] for every State.
func combineCounts(parts ...projection.StateCounts) projection.StateCounts {
	combined := emptyCounts()
	for _, p := range parts {
		combined.Total += p.Total
		for state, count := range p.ByState {
			combined.ByState[state] += count
		}
	}
	return combined
}

// combineSamples returns the union of per-dimension samples across
// per-subtype latencySamples. The combined latency aggregate is
// computed from this union (NOT approximated from per-subtype
// aggregates), so combined percentiles are exact per §0079.
func combineSamples(parts ...latencySamples) latencySamples {
	combined := latencySamples{}
	for _, p := range parts {
		combined.FormationToFirstPromotion = append(combined.FormationToFirstPromotion, p.FormationToFirstPromotion...)
		combined.LatestPromotionToLatestDemotion = append(combined.LatestPromotionToLatestDemotion, p.LatestPromotionToLatestDemotion...)
		combined.FormationToDissolution = append(combined.FormationToDissolution, p.FormationToDissolution...)
	}
	return combined
}

type perSubtype struct {
	Combined           aggregateSection `json:"combined"`
	BehavioralCluster  aggregateSection `json:"behavioral_cluster"`
	AutomationGroup    aggregateSection `json:"automation_group"`
	CampaignHypothesis aggregateSection `json:"campaign_hypothesis"`
	CoordinationRing   aggregateSection `json:"coordination_ring"`
}
