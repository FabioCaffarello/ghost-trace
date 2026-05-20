// Command list-hypotheses materializes the projection over EVERY
// Category III BehavioralCluster hypothesis in the substrate per
// Charter §2.5 BC3 + decision-log §0052. Wraps
// internal/projection.ListHypotheses with command-line filtering,
// paging, and structured JSON output.
//
// Discharges the §0051 named carry-forward "Multi-hypothesis
// aggregate queries". This is the second binary in the read-only
// classification (alongside `hypothesis-state`, §0051) and the
// 11th operational binary overall.
//
// Scope per §0052:
//   - State filter (-state forming|promoted|demoted|dissolved|
//     merged_into|split_into). Empty = no filter.
//   - Paging via -limit + -offset. Cursor-based paging deferred.
//   - Single linear walk over the substrate via
//     projection.ProjectAll regardless of formation count.
//
// Output: structured JSON ARRAY to stdout (each element is the same
// shape as `hypothesis-state`'s single-projection output) + a brief
// human summary to stderr.
//
// Exit codes: 0 success (including empty results); 2 tool/config
// error (e.g. invalid -state value).
package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/projection"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

const exitToolError = 2

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "list-hypotheses: %v\n", err)
		os.Exit(exitToolError)
	}
}

func run() error {
	dbPath := flag.String("db", "./ghost-trace.db", "SQLite primary-event-log path")
	blobDir := flag.String("blobs", "./blobs", "content-addressed blob-store directory")
	stateFilter := flag.String("state", "", "filter by projected state: forming|promoted|demoted|dissolved|merged_into|split_into (empty = all)")
	afterNs := flag.Int64("after-ns", 0, "inclusive lower bound (Unix nanoseconds) on the latest event_time of each projection; 0 disables the lower bound")
	beforeNs := flag.Int64("before-ns", 0, "inclusive upper bound (Unix nanoseconds) on the latest event_time of each projection; 0 disables the upper bound")
	limit := flag.Int("limit", 0, "cap the number of projections returned (0 = unbounded)")
	offset := flag.Int("offset", 0, "skip the first N projections (0 = start at the first)")
	flag.Parse()

	if *stateFilter != "" && !validState(*stateFilter) {
		return fmt.Errorf("--state %q is not one of: forming, promoted, demoted, dissolved, merged_into, split_into", *stateFilter)
	}
	if *limit < 0 {
		return fmt.Errorf("--limit must be non-negative; got %d", *limit)
	}
	if *offset < 0 {
		return fmt.Errorf("--offset must be non-negative; got %d", *offset)
	}
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

	results, err := projection.ListHypotheses(ctx, sub, projection.ListOptions{
		StateFilter:  projection.State(*stateFilter),
		TimeAfterNs:  *afterNs,
		TimeBeforeNs: *beforeNs,
		Limit:        *limit,
		Offset:       *offset,
	})
	if err != nil {
		return err
	}

	out := make([]entry, 0, len(results))
	for _, p := range results {
		out = append(out, buildEntry(p))
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}

	fmt.Fprintf(os.Stderr,
		"list-hypotheses: returned=%d state_filter=%q after_ns=%d before_ns=%d limit=%d offset=%d\n",
		len(results), *stateFilter, *afterNs, *beforeNs, *limit, *offset)
	return nil
}

func validState(s string) bool {
	switch projection.State(s) {
	case projection.StateForming, projection.StatePromoted, projection.StateDemoted,
		projection.StateDissolved, projection.StateMergedInto, projection.StateSplitInto:
		return true
	}
	return false
}

type entry struct {
	FormationHash    string         `json:"formation_event_hash"`
	State            string         `json:"state"`
	LatestPromotion  *promotionView `json:"latest_promotion,omitempty"`
	LatestDemotion   *demotionView  `json:"latest_demotion,omitempty"`
	Dissolution      *eventView     `json:"dissolution,omitempty"`
	MergedInto       *mergeView     `json:"merged_into,omitempty"`
	SplitInto        *splitView     `json:"split_into,omitempty"`
	LifecycleEntries int            `json:"lifecycle_event_count"`
}

type promotionView struct {
	PromotedAt     int64  `json:"promoted_at"`
	CadenceSeconds int64  `json:"cadence_seconds"`
	Reason         string `json:"reason,omitempty"`
}

type demotionView struct {
	DemotedAt int64  `json:"demoted_at"`
	Reason    string `json:"reason,omitempty"`
}

type eventView struct {
	At     int64  `json:"at"`
	Reason string `json:"reason,omitempty"`
}

type mergeView struct {
	MergedAt              int64    `json:"merged_at"`
	ProducedFormationHash string   `json:"produced_formation_event_hash"`
	AntecedentHashes      []string `json:"antecedent_formation_event_hashes"`
	Reason                string   `json:"reason,omitempty"`
}

type splitView struct {
	SplitAt         int64    `json:"split_at"`
	SuccessorHashes []string `json:"successor_formation_event_hashes"`
	Reason          string   `json:"reason,omitempty"`
}

func buildEntry(p projection.HypothesisProjection) entry {
	e := entry{
		FormationHash:    hex.EncodeToString(p.FormationHash[:]),
		State:            string(p.State),
		LifecycleEntries: len(p.LifecycleHistory),
	}
	if p.LatestPromotion != nil {
		e.LatestPromotion = &promotionView{
			PromotedAt:     p.LatestPromotion.PromotedAt,
			CadenceSeconds: p.LatestPromotion.CadenceSeconds,
			Reason:         p.LatestPromotion.Reason,
		}
	}
	if p.LatestDemotion != nil {
		e.LatestDemotion = &demotionView{
			DemotedAt: p.LatestDemotion.DemotedAt,
			Reason:    p.LatestDemotion.Reason,
		}
	}
	if p.Dissolution != nil {
		e.Dissolution = &eventView{
			At:     p.Dissolution.DissolvedAt,
			Reason: p.Dissolution.Reason,
		}
	}
	if p.MergedInto != nil {
		ants := make([]string, len(p.MergedInto.AntecedentFormationEventHashes))
		for i, h := range p.MergedInto.AntecedentFormationEventHashes {
			ants[i] = hex.EncodeToString(h)
		}
		e.MergedInto = &mergeView{
			MergedAt:              p.MergedInto.MergedAt,
			ProducedFormationHash: hex.EncodeToString(p.MergedInto.ProducedFormationEventHash),
			AntecedentHashes:      ants,
			Reason:                p.MergedInto.Reason,
		}
	}
	if p.SplitInto != nil {
		succs := make([]string, len(p.SplitInto.SuccessorFormationEventHashes))
		for i, h := range p.SplitInto.SuccessorFormationEventHashes {
			succs[i] = hex.EncodeToString(h)
		}
		e.SplitInto = &splitView{
			SplitAt:         p.SplitInto.SplitAt,
			SuccessorHashes: succs,
			Reason:          p.SplitInto.Reason,
		}
	}
	return e
}
