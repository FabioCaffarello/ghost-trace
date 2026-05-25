// End-to-end integration test exercising the full F3 → formation
// → promotion → demotion lifecycle arc for CampaignHypothesis per
// §0183. Mirrors §0160 + §0178 patterns on the CampaignHypothesis
// subtype side using PromoteCampaignHypothesis / DemoteCampaignHypothesis
// subtype-suffixed functions.
package main

import (
	"bytes"
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

func TestDemoteCampaignHypothesis_FromF3CandidateLifecycle(t *testing.T) {
	sub, in := newDemoteE2ESubstrate(t)
	ctx := context.Background()

	const bucketStart = int64(1716120000_000_000_000)
	appendNetworkObs(t, in, "actor-1", "10.0.0.1:443", bucketStart)
	appendNetworkObs(t, in, "actor-2", "10.0.0.1:443", bucketStart+10_000_000_000)
	appendNetworkObs(t, in, "actor-3", "10.0.0.1:443", bucketStart+20_000_000_000)

	observations, err := collectNetworkObservations(ctx, sub)
	if err != nil {
		t.Fatalf("collectNetworkObservations: %v", err)
	}
	sig := &signatures.TemporalEndpointCohortV1{}
	result, err := sig.EvaluateNetwork(ctx, observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("candidates count: got %d want 1", len(result.Candidates))
	}

	// Step 3: Operator-elected CampaignHypothesisFormation commit via
	// §0183 foundation helper (drops ActorRefs per §0182 event-centric
	// ontology).
	formationAt := int64(1716120100000000000)
	formationHash := commitCampaignHypothesisFormationFromCandidate(t, sub, result.Candidates[0], formationAt)

	// Step 4: PromoteCampaignHypothesis (subtype-suffixed per §0178
	// MO1 — only BehavioralCluster uses generic-named functions).
	promotedAt := int64(1716120200000000000)
	layerBParams := &commonv1.LayerBParameters{
		TB:                    &commonv1.EvidentialIndependence{Numerator: 1, Denominator: 2},
		KC:                    &commonv1.EvidentialIndependence{Numerator: 1, Denominator: 2},
		NWindow:               100,
		NADurationNanoseconds: 86400000000000,
	}
	now := func() time.Time { return time.Unix(0, promotedAt) }
	promoteReport, err := hypothesis.PromoteCampaignHypothesis(ctx, sub, hypothesis.CampaignHypothesisPromoteOptions{
		FormationEventHash: formationHash,
		PromotedAt:         promotedAt,
		CadenceSeconds:     3600,
		Reason:             "e2e test promotion (CampaignHypothesis)",
		LayerBParameters:   layerBParams,
	}, now)
	if err != nil {
		t.Fatalf("PromoteCampaignHypothesis: %v", err)
	}
	var promotionHash [32]byte
	if _, err := hexDecodeInto(promoteReport.PromotionEventHashHex, promotionHash[:]); err != nil {
		t.Fatalf("decode promotion hash hex: %v", err)
	}

	// Step 5: DemoteCampaignHypothesis within cadence (Layer A
	// advisory test).
	demotedAt := int64(1716120300000000000)
	nowDemote := func() time.Time { return time.Unix(0, demotedAt) }
	demoteReport, err := hypothesis.DemoteCampaignHypothesis(ctx, sub, hypothesis.CampaignHypothesisDemoteOptions{
		PromotionEventHash: promotionHash,
		DemotedAt:          demotedAt,
		Reason:             "e2e test demotion (CampaignHypothesis)",
	}, nowDemote)
	if err != nil {
		t.Fatalf("DemoteCampaignHypothesis: %v", err)
	}

	// Step 6: Verify all 3 lifecycle events in substrate.
	formationRow, err := sub.LookupRow(ctx, formationHash)
	if err != nil {
		t.Fatalf("LookupRow formation: %v", err)
	}
	if formationRow.MessageType != "ghosttrace.events.v1.CampaignHypothesisFormation" {
		t.Errorf("formation MessageType: got %q want CampaignHypothesisFormation", formationRow.MessageType)
	}
	promotionRow, err := sub.LookupRow(ctx, promotionHash)
	if err != nil {
		t.Fatalf("LookupRow promotion: %v", err)
	}
	if promotionRow.MessageType != "ghosttrace.events.v1.CampaignHypothesisPromotion" {
		t.Errorf("promotion MessageType: got %q want CampaignHypothesisPromotion", promotionRow.MessageType)
	}
	var demotionHash [32]byte
	if _, err := hexDecodeInto(demoteReport.DemotionEventHashHex, demotionHash[:]); err != nil {
		t.Fatalf("decode demotion hash hex: %v", err)
	}
	demotionRow, err := sub.LookupRow(ctx, demotionHash)
	if err != nil {
		t.Fatalf("LookupRow demotion: %v", err)
	}
	if demotionRow.MessageType != "ghosttrace.events.v1.CampaignHypothesisDemotion" {
		t.Errorf("demotion MessageType: got %q want CampaignHypothesisDemotion", demotionRow.MessageType)
	}

	// Step 7: Verify demotion proto references promotion event hash.
	demotionPayload, err := sub.ReadBlob(ctx, demotionHash)
	if err != nil {
		t.Fatalf("ReadBlob demotion: %v", err)
	}
	gotDemotion := &eventsv1.CampaignHypothesisDemotion{}
	if err := proto.Unmarshal(demotionPayload, gotDemotion); err != nil {
		t.Fatalf("Unmarshal demotion: %v", err)
	}
	if !bytes.Equal(gotDemotion.PromotionEventHash, promotionHash[:]) {
		t.Error("demotion PromotionEventHash does not match promotion hash")
	}

	// Step 8: Verify Layer A cadence + Layer B verdict captured.
	if demoteReport.CadenceSatisfied {
		t.Errorf("CadenceSatisfied: got true want false (demoted within cadence)")
	}
	if demoteReport.CadenceElapsedSeconds != 100 {
		t.Errorf("CadenceElapsedSeconds: got %d want 100", demoteReport.CadenceElapsedSeconds)
	}
	t.Logf("demote E2E (CampaignHypothesis): cadence=%v elapsed=%ds LayerB.Fired=%v",
		demoteReport.CadenceSatisfied, demoteReport.CadenceElapsedSeconds, demoteReport.LayerB.Fired)
}

// hexDecodeInto + helpers — local copies mirroring §0160 / §0178.
func hexDecodeInto(hexStr string, dst []byte) (int, error) {
	if len(hexStr) != 2*len(dst) {
		return 0, &hexLenErr{}
	}
	for i := 0; i < len(dst); i++ {
		hi, lo := hexNibble(hexStr[2*i]), hexNibble(hexStr[2*i+1])
		dst[i] = byte(hi<<4 | lo)
	}
	return len(dst), nil
}

type hexLenErr struct{}

func (e *hexLenErr) Error() string { return "hex length mismatch" }

func hexNibble(c byte) int {
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
