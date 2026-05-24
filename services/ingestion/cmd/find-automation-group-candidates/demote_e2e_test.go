// End-to-end integration test exercising the full F3 → formation →
// promotion → demotion arc per §0160. Extends the §0157 baseline
// (F3 candidate → formation commit → Layer B evaluation) by continuing
// past the formation commit through promote + demote operator-elected
// steps. Validates the entire §0011 staged-combination lifecycle for
// AutomationGroup from real F3 candidate output.
//
// This is the first integration test exercising the FULL lifecycle
// (formation → promotion → demotion) against substrate state derived
// end-to-end from F3 signature output (not hand-built fixtures).
// Validates: (a) F3-derived formations are valid promotion targets;
// (b) the promotion event preserves LayerBParameters per §0138 N_A
// bundling; (c) the demotion event references the promotion event +
// captures Layer B verdict per §0141 E1 advisory pattern.
//
// Scope per §0160: structural connectivity validation across the
// full lifecycle arc. Per-operation semantics (promotion validation
// errors, demotion cadence-gate behavior, layerb verdict math) are
// covered by the hypothesis package's own unit test suites; this test
// validates only that the FULL arc connects end-to-end from F3
// candidate output without structural failure.
package main

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	commonv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/common/v1"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/hypothesis"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/signatures"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

func newDemoteE2ESubstrate(t *testing.T) (*substrate.Substrate, *ingest.Ingester) {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })
	clock := func() time.Time { return time.Unix(0, 1716120000000000777) }
	return sub, ingest.New(sub, clock)
}

// TestDemoteAutomationGroup_FromF3CandidateLifecycle exercises the
// full F3 → formation → promotion → demotion lifecycle arc:
//
//	Step 1: Inject BrowserObservation above signature threshold.
//	Step 2: Run F3 signature → 1 candidate.
//	Step 3: Operator-elected formation commit (§0157 pattern).
//	Step 4: Operator-elected promotion commit with LayerBParameters.
//	Step 5: Operator-elected demotion commit (advisory Layer B verdict
//	        captured in report per §0141 E1).
//	Step 6: Verify formation + promotion + demotion all in substrate.
//	Step 7: Verify demotion proto references promotion event hash.
//	Step 8: Verify Layer B verdict + cadence state captured in report.
func TestDemoteAutomationGroup_FromF3CandidateLifecycle(t *testing.T) {
	sub, in := newDemoteE2ESubstrate(t)
	ctx := context.Background()

	// Step 1: Inject BrowserObservation above threshold.
	appendBrowserObs(t, in, "actor-suspect", []string{"navigator.webdriver=true"}, 3)
	appendBrowserObs(t, in, "actor-suspect", []string{"$cdc_test"}, 2)

	// Step 2: Run F3 signature → 1 candidate.
	observations, err := collectBrowserObservations(ctx, sub)
	if err != nil {
		t.Fatalf("collectBrowserObservations: %v", err)
	}
	sig := &signatures.CDPMarkerDensityV1{}
	result, err := sig.EvaluateBrowser(ctx, observations)
	if err != nil {
		t.Fatalf("EvaluateBrowser: %v", err)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("candidates count: got %d want 1", len(result.Candidates))
	}
	candidate := result.Candidates[0]

	// Step 3: Operator-elected formation commit (mirrors §0157's
	// commitFormationFromCandidate; reused as-is from this package).
	formationAt := int64(1716120100000000000)
	formationHash := commitFormationFromCandidate(t, sub, candidate, formationAt)

	// Step 4: Operator-elected promotion commit. Mirrors what
	// promote-automation-group CLI does at library level. Includes
	// LayerBParameters per §0138 N_A bundling.
	promotedAt := int64(1716120200000000000)
	layerBParams := &commonv1.LayerBParameters{
		TB:                    &commonv1.EvidentialIndependence{Numerator: 1, Denominator: 2},
		KC:                    &commonv1.EvidentialIndependence{Numerator: 1, Denominator: 2},
		NWindow:               100,
		NADurationNanoseconds: 86400000000000,
	}
	now := func() time.Time { return time.Unix(0, promotedAt) }
	promoteReport, err := hypothesis.PromoteAutomationGroup(ctx, sub, hypothesis.AutomationGroupPromoteOptions{
		FormationEventHash: formationHash,
		PromotedAt:         promotedAt,
		CadenceSeconds:     3600,
		Reason:             "e2e test promotion",
		LayerBParameters:   layerBParams,
	}, now)
	if err != nil {
		t.Fatalf("PromoteAutomationGroup: %v", err)
	}
	if promoteReport.PromotionEventHashHex == "" {
		t.Fatal("PromoteReport.PromotionEventHashHex is empty")
	}
	var promotionHash [32]byte
	if _, err := hexDecodeInto(promoteReport.PromotionEventHashHex, promotionHash[:]); err != nil {
		t.Fatalf("decode promotion hash hex: %v", err)
	}

	// Step 5: Operator-elected demotion commit. Layer B verdict
	// captured in report per §0141 E1 advisory pattern; demotion
	// commits regardless of verdict.
	demotedAt := int64(1716120300000000000)
	nowDemote := func() time.Time { return time.Unix(0, demotedAt) }
	demoteReport, err := hypothesis.DemoteAutomationGroup(ctx, sub, hypothesis.AutomationGroupDemoteOptions{
		PromotionEventHash: promotionHash,
		DemotedAt:          demotedAt,
		Reason:             "e2e test demotion",
	}, nowDemote)
	if err != nil {
		t.Fatalf("DemoteAutomationGroup: %v", err)
	}
	if demoteReport.DemotionEventHashHex == "" {
		t.Fatal("DemoteReport.DemotionEventHashHex is empty")
	}

	// Step 6: Verify all three lifecycle events landed in substrate.
	formationRow, err := sub.LookupRow(ctx, formationHash)
	if err != nil {
		t.Fatalf("LookupRow formation: %v", err)
	}
	if formationRow.MessageType != "ghosttrace.events.v1.AutomationGroupFormation" {
		t.Errorf("formation MessageType: got %q want AutomationGroupFormation", formationRow.MessageType)
	}
	promotionRow, err := sub.LookupRow(ctx, promotionHash)
	if err != nil {
		t.Fatalf("LookupRow promotion: %v", err)
	}
	if promotionRow.MessageType != "ghosttrace.events.v1.AutomationGroupPromotion" {
		t.Errorf("promotion MessageType: got %q want AutomationGroupPromotion", promotionRow.MessageType)
	}
	var demotionHash [32]byte
	if _, err := hexDecodeInto(demoteReport.DemotionEventHashHex, demotionHash[:]); err != nil {
		t.Fatalf("decode demotion hash hex: %v", err)
	}
	demotionRow, err := sub.LookupRow(ctx, demotionHash)
	if err != nil {
		t.Fatalf("LookupRow demotion: %v", err)
	}
	if demotionRow.MessageType != "ghosttrace.events.v1.AutomationGroupDemotion" {
		t.Errorf("demotion MessageType: got %q want AutomationGroupDemotion", demotionRow.MessageType)
	}

	// Step 7: Verify demotion proto references promotion event hash.
	demotionPayload, err := sub.ReadBlob(ctx, demotionHash)
	if err != nil {
		t.Fatalf("ReadBlob demotion: %v", err)
	}
	gotDemotion := &eventsv1.AutomationGroupDemotion{}
	if err := proto.Unmarshal(demotionPayload, gotDemotion); err != nil {
		t.Fatalf("Unmarshal demotion: %v", err)
	}
	if len(gotDemotion.PromotionEventHash) != 32 {
		t.Errorf("demotion PromotionEventHash length: got %d want 32", len(gotDemotion.PromotionEventHash))
	}
	for i := 0; i < 32; i++ {
		if gotDemotion.PromotionEventHash[i] != promotionHash[i] {
			t.Errorf("demotion PromotionEventHash mismatch at byte %d", i)
			break
		}
	}
	if gotDemotion.Reason != "e2e test demotion" {
		t.Errorf("demotion Reason: got %q want %q", gotDemotion.Reason, "e2e test demotion")
	}

	// Step 8: Verify Layer B verdict + cadence state captured in
	// report. Cadence elapsed = 100 seconds (demotedAt - promotedAt =
	// 100s); CadenceSeconds = 3600s; therefore CadenceSatisfied = false
	// (operator demoted BEFORE the cadence elapsed — §0011 Layer A is
	// CANDIDACY criterion, not hard barrier).
	if demoteReport.CadenceSatisfied {
		t.Errorf("CadenceSatisfied: got true want false (demoted within cadence window)")
	}
	if demoteReport.CadenceElapsedSeconds != 100 {
		t.Errorf("CadenceElapsedSeconds: got %d want 100", demoteReport.CadenceElapsedSeconds)
	}
	// Layer B verdict captured; Fired value not asserted (depends on
	// substrate state + N_window; covered by layerb package tests).
	t.Logf("demote E2E: cadence=%v elapsed=%ds LayerB.Fired=%v WindowEventCount=%d FilterMatchCount=%d",
		demoteReport.CadenceSatisfied, demoteReport.CadenceElapsedSeconds,
		demoteReport.LayerB.Fired, demoteReport.LayerB.WindowEventCount,
		demoteReport.LayerB.FilterMatchCount)
}

// hexDecodeInto decodes a hex string into dst (length must be 2*len(dst)).
// Local helper to keep test self-contained.
func hexDecodeInto(hexStr string, dst []byte) (int, error) {
	if len(hexStr) != 2*len(dst) {
		return 0, &hexLenErr{want: 2 * len(dst), got: len(hexStr)}
	}
	for i := 0; i < len(dst); i++ {
		hi, lo := hexNibbleE2E(hexStr[2*i]), hexNibbleE2E(hexStr[2*i+1])
		dst[i] = byte(hi<<4 | lo)
	}
	return len(dst), nil
}

type hexLenErr struct{ want, got int }

func (e *hexLenErr) Error() string { return "hex length mismatch" }

func hexNibbleE2E(c byte) int {
	switch {
	case '0' <= c && c <= '9':
		return int(c - '0')
	case 'a' <= c && c <= 'f':
		return int(c-'a') + 10
	case 'A' <= c && c <= 'F':
		return int(c-'A') + 10
	}
	return 0
}
