package layerb

import (
	"context"
	"path/filepath"
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	commonv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/common/v1"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// halfHalf builds the §0138 inception-phase parameters: t_b = k_c = 1/2,
// n_window = N.
func halfHalf(n uint64) *commonv1.LayerBParameters {
	return &commonv1.LayerBParameters{
		TB:                    &commonv1.EvidentialIndependence{Numerator: 1, Denominator: 2},
		KC:                    &commonv1.EvidentialIndependence{Numerator: 1, Denominator: 2},
		NWindow:               n,
		NADurationNanoseconds: 86400000000000, // 1 day
	}
}

func openSub(t *testing.T) *substrate.Substrate {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })
	return sub
}

// appendFormation commits a BehavioralClusterFormation event to substrate
// with the given closure_hashes + direct_influenced_by + evidential_
// independence. Returns the formation event's content-hash.
func appendFormation(t *testing.T, sub *substrate.Substrate, formationAt int64, closure [][32]byte, direct [][32]byte, ei *commonv1.EvidentialIndependence) [32]byte {
	t.Helper()
	ctx := context.Background()

	// Convert [32]byte arrays to []byte slices, sorted ascending +
	// deduplicated per the hash-list discipline per §0139.
	closureBytes := sortDedupHashes(closure)
	directBytes := sortDedupHashes(direct)

	ev := &eventsv1.BehavioralClusterFormation{
		PatternSignature:       "test-pattern",
		PatternParameters:      "test=1",
		ActorRefs:              []string{"actor-a", "actor-b"},
		FormationAt:            formationAt,
		Confidence:             0.5,
		EvidentialIndependence: ei,
		ClosureHashes:          closureBytes,
		DirectInfluencedBy:     directBytes,
	}

	payload, hash, err := canonical.MarshalAndHash(ev)
	if err != nil {
		t.Fatalf("MarshalAndHash: %v", err)
	}
	hex := canonical.HashHex(hash)
	row := substrate.EventRow{
		EventHash:   hash,
		EventTime:   formationAt,
		MessageType: string(ev.ProtoReflect().Descriptor().FullName()),
		PayloadRef:  hex[:2] + "/" + hex[2:],
		CommittedAt: formationAt,
	}
	if err := sub.Append(ctx, row, payload); err != nil {
		t.Fatalf("substrate.Append: %v", err)
	}
	return hash
}

func sortDedupHashes(in [][32]byte) [][]byte {
	if len(in) == 0 {
		return nil
	}
	// Simple insertion-sort + dedup; tests construct small inputs.
	out := make([][]byte, 0, len(in))
	for i, h := range in {
		out = append(out, append([]byte(nil), h[:]...))
		_ = i
	}
	// Sort ascending.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && bytesLess(out[j], out[j-1]); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	// Dedup.
	dedup := out[:0]
	var prev []byte
	for _, h := range out {
		if prev != nil && bytesEq(prev, h) {
			continue
		}
		dedup = append(dedup, h)
		prev = h
	}
	return dedup
}

func bytesLess(a, b []byte) bool {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return len(a) < len(b)
}

func bytesEq(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestEvaluate_NilSubstrate(t *testing.T) {
	_, err := Evaluate(context.Background(), nil, EvaluateOptions{Params: halfHalf(1000)})
	if err == nil {
		t.Fatal("Evaluate(nil sub) returned nil error; want non-nil")
	}
}

func TestEvaluate_NilParams(t *testing.T) {
	sub := openSub(t)
	_, err := Evaluate(context.Background(), sub, EvaluateOptions{FormationEventHash: [32]byte{1, 2, 3}, Params: nil})
	if err == nil {
		t.Fatal("Evaluate(nil Params) returned nil error; want non-nil")
	}
}

func TestEvaluate_ZeroWindow(t *testing.T) {
	sub := openSub(t)
	params := halfHalf(0)
	_, err := Evaluate(context.Background(), sub, EvaluateOptions{FormationEventHash: [32]byte{1, 2, 3}, Params: params})
	if err == nil {
		t.Fatal("Evaluate(n_window=0) returned nil error; want non-nil")
	}
}

func TestEvaluate_ZeroDenominator(t *testing.T) {
	sub := openSub(t)
	params := &commonv1.LayerBParameters{
		TB:      &commonv1.EvidentialIndependence{Numerator: 1, Denominator: 0},
		KC:      &commonv1.EvidentialIndependence{Numerator: 1, Denominator: 2},
		NWindow: 1000,
	}
	_, err := Evaluate(context.Background(), sub, EvaluateOptions{FormationEventHash: [32]byte{1, 2, 3}, Params: params})
	if err == nil {
		t.Fatal("Evaluate(t_b.denominator=0) returned nil error; want non-nil")
	}
}

func TestEvaluate_EmptySubstrate(t *testing.T) {
	sub := openSub(t)
	verdict, err := Evaluate(context.Background(), sub, EvaluateOptions{FormationEventHash: [32]byte{1, 2, 3}, Params: halfHalf(1000)})
	if err != nil {
		t.Fatalf("Evaluate(empty): %v", err)
	}
	if verdict.WindowEventCount != 0 {
		t.Errorf("WindowEventCount: got %d, want 0", verdict.WindowEventCount)
	}
	if verdict.FilterMatchCount != 0 {
		t.Errorf("FilterMatchCount: got %d, want 0", verdict.FilterMatchCount)
	}
	if !verdict.FreshnessUndefined {
		t.Error("FreshnessUndefined: got false, want true (no matching events)")
	}
	if verdict.FreshnessFired {
		t.Error("FreshnessFired: got true, want false (undefined freshness cannot fire)")
	}
	if verdict.SaturationFired {
		t.Error("SaturationFired: got true, want false (0/1000 = 0, not > 0.5)")
	}
	if verdict.Fired {
		t.Error("Fired: got true, want false")
	}
}

func TestEvaluate_SubstrateWithNoMatchingFormations(t *testing.T) {
	sub := openSub(t)
	// Append a formation whose closure_hashes does NOT include our target.
	otherHash := [32]byte{99, 99, 99}
	_ = appendFormation(t, sub, 1000, [][32]byte{otherHash}, nil, &commonv1.EvidentialIndependence{Numerator: 0, Denominator: 1})

	target := [32]byte{1, 2, 3}
	verdict, err := Evaluate(context.Background(), sub, EvaluateOptions{FormationEventHash: target, Params: halfHalf(1000)})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if verdict.WindowEventCount != 1 {
		t.Errorf("WindowEventCount: got %d, want 1", verdict.WindowEventCount)
	}
	if verdict.FilterMatchCount != 0 {
		t.Errorf("FilterMatchCount: got %d, want 0", verdict.FilterMatchCount)
	}
	if !verdict.FreshnessUndefined {
		t.Error("FreshnessUndefined: got false, want true")
	}
	if verdict.Fired {
		t.Error("Fired: got true, want false")
	}
}

func TestEvaluate_FreshnessFires_LowEIBelowThreshold(t *testing.T) {
	sub := openSub(t)
	target := [32]byte{42, 42, 42}
	// Append two formations whose closure_hashes include target, each
	// with EI = 1/4 (= 0.25). avg = 1/4 < T_B = 1/2 → freshness fires.
	low := &commonv1.EvidentialIndependence{Numerator: 1, Denominator: 4}
	_ = appendFormation(t, sub, 1000, [][32]byte{target}, nil, low)
	_ = appendFormation(t, sub, 2000, [][32]byte{target}, nil, low)

	verdict, err := Evaluate(context.Background(), sub, EvaluateOptions{FormationEventHash: target, Params: halfHalf(1000)})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if verdict.FilterMatchCount != 2 {
		t.Errorf("FilterMatchCount: got %d, want 2", verdict.FilterMatchCount)
	}
	if verdict.FreshnessUndefined {
		t.Error("FreshnessUndefined: got true, want false")
	}
	if !verdict.FreshnessFired {
		t.Errorf("FreshnessFired: got false, want true (avg=1/4 < T_B=1/2). Verdict: freshness=%d/%d", verdict.FreshnessBNumerator, verdict.FreshnessBDenominator)
	}
	if !verdict.Fired {
		t.Error("Fired (L-BC-OR): got false, want true")
	}
}

func TestEvaluate_FreshnessDoesNotFire_HighEIAboveThreshold(t *testing.T) {
	sub := openSub(t)
	target := [32]byte{77, 77, 77}
	// Two formations with EI = 3/4 = 0.75 > T_B = 0.5 → freshness does
	// not fire.
	high := &commonv1.EvidentialIndependence{Numerator: 3, Denominator: 4}
	_ = appendFormation(t, sub, 1000, [][32]byte{target}, nil, high)
	_ = appendFormation(t, sub, 2000, [][32]byte{target}, nil, high)

	verdict, err := Evaluate(context.Background(), sub, EvaluateOptions{FormationEventHash: target, Params: halfHalf(1000)})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if verdict.FreshnessFired {
		t.Errorf("FreshnessFired: got true, want false (avg=3/4 > T_B=1/2). Verdict: freshness=%d/%d", verdict.FreshnessBNumerator, verdict.FreshnessBDenominator)
	}
}

func TestEvaluate_LCExclusion_DirectInfluencedByExcludes(t *testing.T) {
	sub := openSub(t)
	target := [32]byte{88, 88, 88}
	// Three formations:
	//   - F1: closure contains target, direct does NOT → contributes to saturation_C
	//   - F2: closure contains target, direct ALSO contains target → EXCLUDED from saturation_C numerator (L-C exclusion)
	//   - F3: closure contains target, direct does NOT → contributes to saturation_C
	mid := &commonv1.EvidentialIndependence{Numerator: 1, Denominator: 2}
	_ = appendFormation(t, sub, 1000, [][32]byte{target}, nil, mid)
	_ = appendFormation(t, sub, 2000, [][32]byte{target}, [][32]byte{target}, mid)
	_ = appendFormation(t, sub, 3000, [][32]byte{target}, nil, mid)

	// Use n_window=2 so that saturation_C = 2/2 = 1.0 > 0.5 (saturation fires).
	// Actually wait — n_window is the GLOBAL count, not the filtered count.
	// With 3 formations in substrate and N=2, the window is the last 2:
	//   F2 (closure ∈, direct ∈) → excluded from numerator by L-C
	//   F3 (closure ∈, direct ∉) → counted
	// saturation_C = 1/2 = 0.5 — exactly the threshold; > comparison is strict so does NOT fire.
	params := halfHalf(2)
	verdict, err := Evaluate(context.Background(), sub, EvaluateOptions{FormationEventHash: target, Params: params})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if verdict.WindowEventCount != 2 {
		t.Errorf("WindowEventCount: got %d, want 2", verdict.WindowEventCount)
	}
	if verdict.FilterMatchCount != 2 {
		t.Errorf("FilterMatchCount: got %d, want 2 (last 2 events: F2 + F3)", verdict.FilterMatchCount)
	}
	// saturation_C numerator = 1 (only F3 counted; F2 excluded by L-C).
	if verdict.SaturationCNumerator != 1 || verdict.SaturationCDenominator != 2 {
		t.Errorf("saturation_C: got %d/%d, want 1/2 (F3 counted; F2 L-C-excluded)", verdict.SaturationCNumerator, verdict.SaturationCDenominator)
	}
	if verdict.SaturationFired {
		t.Error("SaturationFired: got true, want false (0.5 is NOT > 0.5; strict inequality)")
	}
}

func TestEvaluate_SaturationFires_HighFilterMatchRatio(t *testing.T) {
	sub := openSub(t)
	target := [32]byte{55, 55, 55}
	// 3 formations all contain target in closure_hashes, none in direct.
	// N=3 → saturation_C = 3/3 = 1.0 > 0.5 → fires.
	mid := &commonv1.EvidentialIndependence{Numerator: 3, Denominator: 4} // high EI so freshness doesn't fire
	_ = appendFormation(t, sub, 1000, [][32]byte{target}, nil, mid)
	_ = appendFormation(t, sub, 2000, [][32]byte{target}, nil, mid)
	_ = appendFormation(t, sub, 3000, [][32]byte{target}, nil, mid)

	verdict, err := Evaluate(context.Background(), sub, EvaluateOptions{FormationEventHash: target, Params: halfHalf(3)})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if !verdict.SaturationFired {
		t.Errorf("SaturationFired: got false, want true (3/3=1 > 0.5). Verdict: saturation=%d/%d", verdict.SaturationCNumerator, verdict.SaturationCDenominator)
	}
	if verdict.FreshnessFired {
		t.Errorf("FreshnessFired: got true, want false (avg=3/4 > 0.5)")
	}
	if !verdict.Fired {
		t.Error("Fired (L-BC-OR): got false, want true (saturation fires)")
	}
}

func TestEvaluate_NonClosureEventsFilteredNaturally(t *testing.T) {
	// Append a DeclaredSession (Cat I, no closure_hashes) — should be
	// counted in WindowEventCount but contribute nothing to filter or
	// saturation.
	sub := openSub(t)
	ctx := context.Background()
	target := [32]byte{1, 2, 3}

	ds := &eventsv1.DeclaredSession{
		DeclaredAt:        1000,
		ActorRef:          "actor",
		SessionDescriptor: []byte("descriptor"),
	}
	payload, hash, err := canonical.MarshalAndHash(ds)
	if err != nil {
		t.Fatalf("MarshalAndHash: %v", err)
	}
	hex := canonical.HashHex(hash)
	row := substrate.EventRow{
		EventHash:   hash,
		EventTime:   1000,
		MessageType: string(ds.ProtoReflect().Descriptor().FullName()),
		PayloadRef:  hex[:2] + "/" + hex[2:],
		CommittedAt: 1000,
	}
	if err := sub.Append(ctx, row, payload); err != nil {
		t.Fatalf("substrate.Append: %v", err)
	}
	_ = proto.Marshal // keep import

	verdict, err := Evaluate(ctx, sub, EvaluateOptions{FormationEventHash: target, Params: halfHalf(1000)})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if verdict.WindowEventCount != 1 {
		t.Errorf("WindowEventCount: got %d, want 1 (the DeclaredSession)", verdict.WindowEventCount)
	}
	if verdict.FilterMatchCount != 0 {
		t.Errorf("FilterMatchCount: got %d, want 0 (DeclaredSession has no closure_hashes)", verdict.FilterMatchCount)
	}
	if !verdict.FreshnessUndefined {
		t.Error("FreshnessUndefined: got false, want true")
	}
	if verdict.Fired {
		t.Error("Fired: got true, want false")
	}
}
