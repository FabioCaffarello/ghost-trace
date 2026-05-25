// End-to-end integration test exercising the full F3 → formation
// → promotion → demotion lifecycle arc for CoordinationRing per
// §0186. Mirrors §0160 + §0178 + §0183 patterns on the
// CoordinationRing subtype side using PromoteCoordinationRing /
// DemoteCoordinationRing subtype-suffixed functions per §0178 MO1.
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
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/observationcollector"
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

func TestDemoteCoordinationRing_FromF3CandidateLifecycle(t *testing.T) {
	sub, in := newDemoteE2ESubstrate(t)
	ctx := context.Background()

	const bucketStart = int64(1716120000_000_000_000)
	appendNetworkObs(t, in, "actor-1", "10.0.0.1:443", bucketStart)
	appendNetworkObs(t, in, "actor-2", "10.0.0.1:443", bucketStart+10_000_000_000)
	appendNetworkObs(t, in, "actor-3", "10.0.0.1:443", bucketStart+20_000_000_000)

	observations, err := observationcollector.CollectNetwork(ctx, sub)
	if err != nil {
		t.Fatalf("observationcollector.CollectNetwork: %v", err)
	}
	sig := &signatures.EndpointCoVisitV1{}
	result, err := sig.EvaluateNetwork(ctx, observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("candidates count: got %d want 1", len(result.Candidates))
	}

	formationAt := int64(1716120100000000000)
	formationHash := commitCoordinationRingFormationFromCandidate(t, sub, result.Candidates[0], formationAt)

	promotedAt := int64(1716120200000000000)
	layerBParams := &commonv1.LayerBParameters{
		TB:                    &commonv1.EvidentialIndependence{Numerator: 1, Denominator: 2},
		KC:                    &commonv1.EvidentialIndependence{Numerator: 1, Denominator: 2},
		NWindow:               100,
		NADurationNanoseconds: 86400000000000,
	}
	now := func() time.Time { return time.Unix(0, promotedAt) }
	promoteReport, err := hypothesis.PromoteCoordinationRing(ctx, sub, hypothesis.CoordinationRingPromoteOptions{
		FormationEventHash: formationHash,
		PromotedAt:         promotedAt,
		CadenceSeconds:     3600,
		Reason:             "e2e test promotion (CoordinationRing)",
		LayerBParameters:   layerBParams,
	}, now)
	if err != nil {
		t.Fatalf("PromoteCoordinationRing: %v", err)
	}
	var promotionHash [32]byte
	if _, err := hexDecodeInto(promoteReport.PromotionEventHashHex, promotionHash[:]); err != nil {
		t.Fatalf("decode promotion hash hex: %v", err)
	}

	demotedAt := int64(1716120300000000000)
	nowDemote := func() time.Time { return time.Unix(0, demotedAt) }
	demoteReport, err := hypothesis.DemoteCoordinationRing(ctx, sub, hypothesis.CoordinationRingDemoteOptions{
		PromotionEventHash: promotionHash,
		DemotedAt:          demotedAt,
		Reason:             "e2e test demotion (CoordinationRing)",
	}, nowDemote)
	if err != nil {
		t.Fatalf("DemoteCoordinationRing: %v", err)
	}

	formationRow, err := sub.LookupRow(ctx, formationHash)
	if err != nil {
		t.Fatalf("LookupRow formation: %v", err)
	}
	if formationRow.MessageType != "ghosttrace.events.v1.CoordinationRingFormation" {
		t.Errorf("formation MessageType: got %q want CoordinationRingFormation", formationRow.MessageType)
	}
	promotionRow, err := sub.LookupRow(ctx, promotionHash)
	if err != nil {
		t.Fatalf("LookupRow promotion: %v", err)
	}
	if promotionRow.MessageType != "ghosttrace.events.v1.CoordinationRingPromotion" {
		t.Errorf("promotion MessageType: got %q want CoordinationRingPromotion", promotionRow.MessageType)
	}
	var demotionHash [32]byte
	if _, err := hexDecodeInto(demoteReport.DemotionEventHashHex, demotionHash[:]); err != nil {
		t.Fatalf("decode demotion hash hex: %v", err)
	}
	demotionRow, err := sub.LookupRow(ctx, demotionHash)
	if err != nil {
		t.Fatalf("LookupRow demotion: %v", err)
	}
	if demotionRow.MessageType != "ghosttrace.events.v1.CoordinationRingDemotion" {
		t.Errorf("demotion MessageType: got %q want CoordinationRingDemotion", demotionRow.MessageType)
	}

	demotionPayload, err := sub.ReadBlob(ctx, demotionHash)
	if err != nil {
		t.Fatalf("ReadBlob demotion: %v", err)
	}
	gotDemotion := &eventsv1.CoordinationRingDemotion{}
	if err := proto.Unmarshal(demotionPayload, gotDemotion); err != nil {
		t.Fatalf("Unmarshal demotion: %v", err)
	}
	if !bytes.Equal(gotDemotion.PromotionEventHash, promotionHash[:]) {
		t.Error("demotion PromotionEventHash does not match promotion hash")
	}

	if demoteReport.CadenceSatisfied {
		t.Errorf("CadenceSatisfied: got true want false (demoted within cadence)")
	}
	if demoteReport.CadenceElapsedSeconds != 100 {
		t.Errorf("CadenceElapsedSeconds: got %d want 100", demoteReport.CadenceElapsedSeconds)
	}
	t.Logf("demote E2E (CoordinationRing): cadence=%v elapsed=%ds LayerB.Fired=%v",
		demoteReport.CadenceSatisfied, demoteReport.CadenceElapsedSeconds, demoteReport.LayerB.Fired)
}

// hexDecodeInto + helpers — local copies mirroring §0160 / §0178 / §0183.
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
