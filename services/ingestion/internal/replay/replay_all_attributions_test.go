package replay

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

func TestReplayAllDerivedActorAttributions_HappyPath(t *testing.T) {
	sub, daaHashes := substrateWithDerivedAttributions(t)
	ctx := context.Background()

	report, err := ReplayAllDerivedActorAttributions(ctx, sub)
	if err != nil {
		t.Fatalf("ReplayAllDerivedActorAttributions: %v", err)
	}
	if report.Total != len(daaHashes) {
		t.Errorf("Total: got %d want %d", report.Total, len(daaHashes))
	}
	if report.Matched != len(daaHashes) {
		t.Errorf("Matched: got %d want %d (all should match)", report.Matched, len(daaHashes))
	}
	if report.Drifted != 0 {
		t.Errorf("Drifted: got %d want 0", report.Drifted)
	}
	if report.Errored != 0 {
		t.Errorf("Errored: got %d want 0", report.Errored)
	}
	if len(report.Drift) != 0 {
		t.Errorf("Drift slice: got %d entries want 0", len(report.Drift))
	}
	if len(report.Errors) != 0 {
		t.Errorf("Errors slice: got %d entries want 0", len(report.Errors))
	}
}

func TestReplayAllDerivedActorAttributions_EmptySubstrate(t *testing.T) {
	// Empty substrate (no DerivedActorAttribution rows) → all-zero
	// report; no error.
	ctx := context.Background()
	freshSub, _ := substrateWithoutAttributions(t)
	report, err := ReplayAllDerivedActorAttributions(ctx, freshSub)
	if err != nil {
		t.Fatalf("ReplayAllDerivedActorAttributions: %v", err)
	}
	if report.Total != 0 {
		t.Errorf("Total: got %d want 0 (empty substrate)", report.Total)
	}
	if report.Matched != 0 || report.Drifted != 0 || report.Errored != 0 {
		t.Errorf("non-zero counts on empty substrate: matched=%d drifted=%d errored=%d",
			report.Matched, report.Drifted, report.Errored)
	}
}

// TestReplayAllDerivedActorAttributions_StableReportShape verifies the
// batch report shape is identical to ReplayAllOperationalSessions's
// BatchReplayReport (per §0173 stable-structure discipline + §0163
// stable-wire-contract). Both batch replay functions emit the same
// report shape; operators can apply the same parsing logic.
func TestReplayAllDerivedActorAttributions_StableReportShape(t *testing.T) {
	sub, _ := substrateWithDerivedAttributions(t)
	ctx := context.Background()

	report, err := ReplayAllDerivedActorAttributions(ctx, sub)
	if err != nil {
		t.Fatalf("ReplayAllDerivedActorAttributions: %v", err)
	}

	// The BatchReplayReport type is shared with ReplayAllOperational
	// Sessions; verify via type-assertion-by-compile that the same
	// struct is returned.
	var _ BatchReplayReport = report
}

// substrateWithoutAttributions opens a fresh substrate with no rows.
// Helper for empty-substrate test cases.
func substrateWithoutAttributions(t *testing.T) (*substrate.Substrate, [][32]byte) {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "fresh.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })
	return sub, nil
}

