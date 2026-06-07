package hypothesis

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	commonv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/common/v1"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/signatures"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// agSubstrate constructs a fresh substrate and ingests one
// DeclaredSession per (actor, declared_at) tuple. Returns the
// substrate.
func agSubstrate(t *testing.T, items []struct {
	ActorRef   string
	DeclaredAt int64
}) *substrate.Substrate {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	in := ingest.New(sub, func() time.Time { return time.Unix(0, 1716120000000000777) })
	for i, it := range items {
		msg := &eventsv1.DeclaredSession{
			DeclaredAt:        it.DeclaredAt,
			ActorRef:          it.ActorRef,
			SessionDescriptor: []byte("ag-test-" + it.ActorRef),
		}
		if _, err := in.Append(ctx, msg, msg.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("Append %d: %v", i, err)
		}
	}
	return sub
}

func walkAutomationGroupFormations(t *testing.T, sub *substrate.Substrate) []*eventsv1.AutomationGroupFormation {
	t.Helper()
	ctx := context.Background()
	var out []*eventsv1.AutomationGroupFormation
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != "ghosttrace.events.v1.AutomationGroupFormation" {
			return nil
		}
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return err
		}
		ev := &eventsv1.AutomationGroupFormation{}
		if err := proto.Unmarshal(payload, ev); err != nil {
			return err
		}
		out = append(out, ev)
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	return out
}

func TestUniformCadenceV1HappyPath(t *testing.T) {
	// Actor "bot-a" has perfectly uniform 1000ns cadence over 5
	// observations → CoV = 0 < 0.15 → matches automation signature.
	// Actor "human-b" has highly variable cadence → CoV > 0.15 →
	// excluded.
	sub := agSubstrate(t, []struct {
		ActorRef   string
		DeclaredAt int64
	}{
		{"bot-a", 1000}, {"bot-a", 2000}, {"bot-a", 3000}, {"bot-a", 4000}, {"bot-a", 5000},
		{"human-b", 1000}, {"human-b", 1100}, {"human-b", 5000}, {"human-b", 5050}, {"human-b", 9000},
	})

	rep, err := FormAutomationGroupAll(context.Background(), sub,
		UniformCadenceV1{MinObservationCount: 5, MaxCoVThreshold: 0.15},
		func() time.Time { return time.Unix(0, 1) })
	if err != nil {
		t.Fatalf("FormAutomationGroupAll: %v", err)
	}
	if rep.NewlyFormed != 1 {
		t.Errorf("NewlyFormed: got %d, want 1 (only bot-a)", rep.NewlyFormed)
	}

	forms := walkAutomationGroupFormations(t, sub)
	if len(forms) != 1 {
		t.Fatalf("substrate carries %d formations, want 1", len(forms))
	}
	got := forms[0]
	if got.PatternSignature != UniformCadenceV1Signature {
		t.Errorf("pattern_signature: got %q, want %q", got.PatternSignature, UniformCadenceV1Signature)
	}
	if got.PatternParameters != "max_cov_threshold=0.15;min_observation_count=5" {
		t.Errorf("pattern_parameters: got %q", got.PatternParameters)
	}
	if len(got.ActorRefs) != 1 || got.ActorRefs[0] != "bot-a" {
		t.Errorf("actor_refs: got %v, want [bot-a]", got.ActorRefs)
	}
	if got.FormationAt != 5000 {
		t.Errorf("formation_at: got %d, want 5000 (max declared_at for bot-a)", got.FormationAt)
	}
	// CoV = 0 → confidence = 1.0.
	if got.Confidence != 1.0 {
		t.Errorf("confidence: got %v, want 1.0 (perfect uniformity)", got.Confidence)
	}
}

func TestUniformCadenceV1RespectsMinObservationCount(t *testing.T) {
	// Actor with only 4 uniformly-spaced observations and
	// MinObservationCount=5 → excluded even though CoV would be 0.
	sub := agSubstrate(t, []struct {
		ActorRef   string
		DeclaredAt int64
	}{
		{"bot-a", 1000}, {"bot-a", 2000}, {"bot-a", 3000}, {"bot-a", 4000},
	})

	rep, err := FormAutomationGroupAll(context.Background(), sub,
		UniformCadenceV1{MinObservationCount: 5, MaxCoVThreshold: 0.15},
		nil)
	if err != nil {
		t.Fatalf("FormAutomationGroupAll: %v", err)
	}
	if rep.NewlyFormed != 0 {
		t.Errorf("NewlyFormed: got %d, want 0 (4 < min 5)", rep.NewlyFormed)
	}
}

func TestUniformCadenceV1ExcludesEmptyActorRef(t *testing.T) {
	// Unattributed (empty actor_ref) sessions are excluded; even if
	// their cadence is uniform, no automation-signature attribution
	// is possible without an actor identity.
	sub := agSubstrate(t, []struct {
		ActorRef   string
		DeclaredAt int64
	}{
		{"", 1000}, {"", 2000}, {"", 3000}, {"", 4000}, {"", 5000},
	})

	rep, err := FormAutomationGroupAll(context.Background(), sub,
		UniformCadenceV1{MinObservationCount: 5, MaxCoVThreshold: 0.15},
		nil)
	if err != nil {
		t.Fatalf("FormAutomationGroupAll: %v", err)
	}
	if rep.NewlyFormed != 0 {
		t.Errorf("NewlyFormed: got %d, want 0 (unattributed sessions excluded)", rep.NewlyFormed)
	}
}

func TestUniformCadenceV1Idempotent(t *testing.T) {
	sub := agSubstrate(t, []struct {
		ActorRef   string
		DeclaredAt int64
	}{
		{"bot-a", 1000}, {"bot-a", 2000}, {"bot-a", 3000}, {"bot-a", 4000}, {"bot-a", 5000},
	})

	ctx := context.Background()
	pattern := UniformCadenceV1{MinObservationCount: 5, MaxCoVThreshold: 0.15}
	rep1, _ := FormAutomationGroupAll(ctx, sub, pattern, nil)
	if rep1.NewlyFormed != 1 {
		t.Fatalf("first run NewlyFormed: got %d, want 1", rep1.NewlyFormed)
	}
	rep2, _ := FormAutomationGroupAll(ctx, sub, pattern, nil)
	if rep2.NewlyFormed != 0 {
		t.Errorf("second run NewlyFormed: got %d, want 0 (idempotent)", rep2.NewlyFormed)
	}
	if rep2.AlreadyFormed != 1 {
		t.Errorf("second run AlreadyFormed: got %d, want 1", rep2.AlreadyFormed)
	}
}

func TestUniformCadenceV1VersioningProducesNewRecord(t *testing.T) {
	// Different MaxCoVThreshold produces a different
	// pattern_parameters string → different content-hash → new
	// substrate row alongside the prior.
	sub := agSubstrate(t, []struct {
		ActorRef   string
		DeclaredAt int64
	}{
		{"bot-a", 1000}, {"bot-a", 2000}, {"bot-a", 3000}, {"bot-a", 4000}, {"bot-a", 5000},
	})

	ctx := context.Background()
	if _, err := FormAutomationGroupAll(ctx, sub,
		UniformCadenceV1{MinObservationCount: 5, MaxCoVThreshold: 0.15}, nil); err != nil {
		t.Fatalf("first FormAutomationGroupAll: %v", err)
	}
	rep, err := FormAutomationGroupAll(ctx, sub,
		UniformCadenceV1{MinObservationCount: 5, MaxCoVThreshold: 0.25}, nil)
	if err != nil {
		t.Fatalf("second FormAutomationGroupAll: %v", err)
	}
	if rep.NewlyFormed != 1 {
		t.Errorf("changing threshold did not produce new record: NewlyFormed=%d, want 1", rep.NewlyFormed)
	}
	if got := len(walkAutomationGroupFormations(t, sub)); got != 2 {
		t.Errorf("substrate should hold 2 formations after two parameterizations; got %d", got)
	}
}

func TestUniformCadenceV1ThresholdBoundary(t *testing.T) {
	// Edge: an actor with CoV exactly equal to MaxCoVThreshold is
	// INCLUDED (the pattern uses <= per the algorithm comment).
	// Construct deltas with known CoV: deltas = [900, 1100], mean
	// = 1000, std-dev = 100, CoV = 0.1.
	sub := agSubstrate(t, []struct {
		ActorRef   string
		DeclaredAt int64
	}{
		{"actor-edge", 1000}, {"actor-edge", 1900}, {"actor-edge", 3000},
	})

	rep, err := FormAutomationGroupAll(context.Background(), sub,
		UniformCadenceV1{MinObservationCount: 3, MaxCoVThreshold: 0.1}, nil)
	if err != nil {
		t.Fatalf("FormAutomationGroupAll: %v", err)
	}
	if rep.NewlyFormed != 1 {
		t.Errorf("CoV == threshold: NewlyFormed=%d, want 1 (boundary INCLUSIVE)", rep.NewlyFormed)
	}
}

func TestUniformCadenceV1MultipleActors(t *testing.T) {
	// Two distinct uniform-cadence actors → two separate
	// single-actor AutomationGroupFormation events.
	sub := agSubstrate(t, []struct {
		ActorRef   string
		DeclaredAt int64
	}{
		{"bot-a", 1000}, {"bot-a", 2000}, {"bot-a", 3000}, {"bot-a", 4000}, {"bot-a", 5000},
		{"bot-b", 10000}, {"bot-b", 20000}, {"bot-b", 30000}, {"bot-b", 40000}, {"bot-b", 50000},
	})

	rep, err := FormAutomationGroupAll(context.Background(), sub,
		UniformCadenceV1{MinObservationCount: 5, MaxCoVThreshold: 0.15}, nil)
	if err != nil {
		t.Fatalf("FormAutomationGroupAll: %v", err)
	}
	if rep.NewlyFormed != 2 {
		t.Errorf("NewlyFormed: got %d, want 2 (one per actor)", rep.NewlyFormed)
	}
	forms := walkAutomationGroupFormations(t, sub)
	if len(forms) != 2 {
		t.Fatalf("substrate carries %d formations, want 2", len(forms))
	}
	// Each formation has exactly 1 actor_ref (single-actor groups
	// per the inception §0056 pattern; multi-actor grouping deferred).
	for _, f := range forms {
		if len(f.ActorRefs) != 1 {
			t.Errorf("formation has %d actors, want 1 (single-actor groups)", len(f.ActorRefs))
		}
	}
}

func TestUniformCadenceV1ZeroMeanSkipped(t *testing.T) {
	// Pathological: three observations at IDENTICAL declared_at.
	// All inter-event deltas are 0 → mean = 0 → CoV undefined.
	// The pattern skips this case rather than emit a degenerate
	// formation.
	sub := agSubstrate(t, []struct {
		ActorRef   string
		DeclaredAt int64
	}{
		{"bot-zero", 1000}, {"bot-zero", 1000}, {"bot-zero", 1000}, {"bot-zero", 1000}, {"bot-zero", 1000},
	})

	rep, err := FormAutomationGroupAll(context.Background(), sub,
		UniformCadenceV1{MinObservationCount: 5, MaxCoVThreshold: 0.15}, nil)
	if err != nil {
		t.Fatalf("FormAutomationGroupAll: %v", err)
	}
	if rep.NewlyFormed != 0 {
		t.Errorf("zero-mean case: NewlyFormed=%d, want 0", rep.NewlyFormed)
	}
}

func TestFormAutomationGroupAllEmptySubstrate(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	rep, err := FormAutomationGroupAll(ctx, sub,
		UniformCadenceV1{MinObservationCount: 3, MaxCoVThreshold: 0.15}, nil)
	if err != nil {
		t.Fatalf("FormAutomationGroupAll on empty: %v", err)
	}
	if rep.Examined != 0 || rep.NewlyFormed != 0 || rep.AlreadyFormed != 0 {
		t.Errorf("empty substrate Report: got %+v, want zero", rep)
	}
}

func TestFormAutomationGroupAllNilPatternRejected(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	_, err = FormAutomationGroupAll(ctx, sub, nil, nil)
	if err == nil {
		t.Error("nil pattern should return error")
	}
}

func TestUniformCadenceV1CoexistsWithBehavioralCluster(t *testing.T) {
	// Substrate carrying both subtypes — verify they're stored as
	// distinct message_types per the typed-subtype-landings
	// commitment of §0045 + §0056.
	sub := agSubstrate(t, []struct {
		ActorRef   string
		DeclaredAt int64
	}{
		{"bot-a", 1000}, {"bot-a", 2000}, {"bot-a", 3000}, {"bot-a", 4000}, {"bot-a", 5000},
	})
	ctx := context.Background()

	// Run AutomationGroup formation.
	if _, err := FormAutomationGroupAll(ctx, sub,
		UniformCadenceV1{MinObservationCount: 5, MaxCoVThreshold: 0.15}, nil); err != nil {
		t.Fatalf("FormAutomationGroupAll: %v", err)
	}

	// Add observations to support a BehavioralCluster formation
	// alongside (two actors sharing a session descriptor).
	in := ingest.New(sub, func() time.Time { return time.Unix(0, 6000) })
	for _, actor := range []string{"sharing-a", "sharing-b"} {
		msg := &eventsv1.DeclaredSession{
			DeclaredAt:        6000,
			ActorRef:          actor,
			SessionDescriptor: []byte("shared-desc"),
		}
		if _, err := in.Append(ctx, msg, msg.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("Append %s: %v", actor, err)
		}
	}
	if _, err := FormAll(ctx, sub,
		SessionDescriptorSharedV1{MinClusterSize: 2}, func() time.Time { return time.Unix(0, 7000) }); err != nil {
		t.Fatalf("FormAll: %v", err)
	}

	// Substrate should carry both message types.
	typeCounts := map[string]int{}
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		typeCounts[row.MessageType]++
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	if typeCounts["ghosttrace.events.v1.AutomationGroupFormation"] != 1 {
		t.Errorf("AutomationGroupFormation count: got %d, want 1",
			typeCounts["ghosttrace.events.v1.AutomationGroupFormation"])
	}
	if typeCounts["ghosttrace.events.v1.BehavioralClusterFormation"] != 1 {
		t.Errorf("BehavioralClusterFormation count: got %d, want 1",
			typeCounts["ghosttrace.events.v1.BehavioralClusterFormation"])
	}
}

// TestAutomationGroupFormationFromCandidate_HappyPath witnesses §0213:
// the bridge from F3 signature candidate to committed
// AutomationGroupFormation event. Mirrors the §0157 test helper
// pattern lifted to package-public API.
func TestAutomationGroupFormationFromCandidate_HappyPath(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	candidate := &signatures.FormationCandidate{
		SignatureName:     "tcp_flow_features_clustering_v1",
		HypothesisSubtype: signatures.HypothesisSubtypeAutomationGroup,
		ActorRefs:         []string{"a:1/tcp", "b:2/tcp", "c:3/tcp"},
		SourceHashes:      [][]byte{bytes32("aaa"), bytes32("bbb"), bytes32("ccc")},
		EvidenceCount:     3,
		ConfidenceHint:    0.75,
	}

	formationAt := int64(1716120100000000000)
	hash, alreadyPresent, err := AutomationGroupFormationFromCandidate(
		ctx, sub, candidate,
		AutomationGroupFormationFromCandidateOptions{FormationAt: formationAt},
		func() time.Time { return time.Unix(0, formationAt) },
	)
	if err != nil {
		t.Fatalf("AutomationGroupFormationFromCandidate: %v", err)
	}
	if alreadyPresent {
		t.Errorf("alreadyPresent: got true want false on first commit")
	}
	if hash == ([32]byte{}) {
		t.Errorf("hash: got zero value")
	}

	formations := walkAutomationGroupFormations(t, sub)
	if len(formations) != 1 {
		t.Fatalf("formations count: got %d want 1", len(formations))
	}
	got := formations[0]
	if got.PatternSignature != "tcp_flow_features_clustering_v1" {
		t.Errorf("PatternSignature: got %q want tcp_flow_features_clustering_v1", got.PatternSignature)
	}
	if len(got.ActorRefs) != 3 {
		t.Errorf("ActorRefs count: got %d want 3", len(got.ActorRefs))
	}
	if got.FormationAt != formationAt {
		t.Errorf("FormationAt: got %d want %d", got.FormationAt, formationAt)
	}
	// §0213 cravoes Confidence=0.0 default (departs from §0157 helper which
	// used ConfidenceHint); registers the §0214 Layer B gating empirical
	// prediction explicitly in the entry.
	if got.Confidence != 0.0 {
		t.Errorf("Confidence default: got %f want 0.0 per §0213 inception discipline", got.Confidence)
	}
	// §0213 cravoes EI={1,1} default per §0157 helper precedent; required
	// at marshalling boundary per §0140 paired-dimension check.
	if got.EvidentialIndependence == nil {
		t.Fatal("EvidentialIndependence: got nil; required at marshalling boundary per §0140")
	}
	if got.EvidentialIndependence.Numerator != 1 || got.EvidentialIndependence.Denominator != 1 {
		t.Errorf("EvidentialIndependence: got %d/%d want 1/1 per §0213 default",
			got.EvidentialIndependence.Numerator, got.EvidentialIndependence.Denominator)
	}
	if len(got.SourceEventHashes) != 3 {
		t.Errorf("SourceEventHashes count: got %d want 3 (verbatim from candidate.SourceHashes)", len(got.SourceEventHashes))
	}
}

// TestAutomationGroupFormationFromCandidate_Idempotent witnesses
// content-addressed immutability per §0027 AP6: re-committing the
// SAME candidate + SAME options produces SAME content-hash; second
// commit returns alreadyPresent=true.
func TestAutomationGroupFormationFromCandidate_Idempotent(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	candidate := &signatures.FormationCandidate{
		SignatureName:     "tcp_flow_features_clustering_v1",
		HypothesisSubtype: signatures.HypothesisSubtypeAutomationGroup,
		ActorRefs:         []string{"x:1/tcp"},
		SourceHashes:      [][]byte{bytes32("xxx")},
	}
	formationAt := int64(1716120100000000000)
	opts := AutomationGroupFormationFromCandidateOptions{FormationAt: formationAt}
	clock := func() time.Time { return time.Unix(0, formationAt) }

	hash1, alreadyPresent1, err := AutomationGroupFormationFromCandidate(ctx, sub, candidate, opts, clock)
	if err != nil {
		t.Fatalf("first commit: %v", err)
	}
	if alreadyPresent1 {
		t.Errorf("first commit alreadyPresent: got true want false")
	}

	hash2, alreadyPresent2, err := AutomationGroupFormationFromCandidate(ctx, sub, candidate, opts, clock)
	if err != nil {
		t.Fatalf("second commit: %v", err)
	}
	if !alreadyPresent2 {
		t.Errorf("second commit alreadyPresent: got false want true (idempotency violated)")
	}
	if hash1 != hash2 {
		t.Errorf("hash mismatch: first=%x second=%x (content-hash should be identical)", hash1, hash2)
	}
}

// TestAutomationGroupFormationFromCandidate_RejectsCrossSubtype witnesses
// the cross-subtype guard: a candidate carrying HypothesisSubtype !=
// AutomationGroup MUST error. Future bridge functions for BC / CH / CR
// will mirror this guard.
func TestAutomationGroupFormationFromCandidate_RejectsCrossSubtype(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	candidate := &signatures.FormationCandidate{
		SignatureName:     "keystroke_timing_clustering_v1",
		HypothesisSubtype: signatures.HypothesisSubtypeBehavioralCluster,
		ActorRefs:         []string{"a:1"},
		SourceHashes:      [][]byte{bytes32("aaa")},
	}
	_, _, err = AutomationGroupFormationFromCandidate(
		ctx, sub, candidate,
		AutomationGroupFormationFromCandidateOptions{},
		nil,
	)
	if err == nil {
		t.Fatal("expected error for cross-subtype candidate; got nil")
	}
}

// TestAutomationGroupFormationFromCandidate_OperatorOverrides witnesses
// the operator-override path for Confidence + EvidentialIndependence
// defaults. §0213 named operator-overridable for §0214 Layer B
// experimentation tier-3 if needed.
func TestAutomationGroupFormationFromCandidate_OperatorOverrides(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	candidate := &signatures.FormationCandidate{
		SignatureName:     "tcp_flow_features_clustering_v1",
		HypothesisSubtype: signatures.HypothesisSubtypeAutomationGroup,
		ActorRefs:         []string{"a:1/tcp"},
		SourceHashes:      [][]byte{bytes32("aaa")},
	}
	formationAt := int64(1716120100000000000)
	opts := AutomationGroupFormationFromCandidateOptions{
		FormationAt:            formationAt,
		PatternParameters:      "min-cluster=3;feature=window-size",
		Confidence:             0.85,
		EvidentialIndependence: &commonv1.EvidentialIndependence{Numerator: 3, Denominator: 4},
	}
	_, _, err = AutomationGroupFormationFromCandidate(
		ctx, sub, candidate, opts,
		func() time.Time { return time.Unix(0, formationAt) },
	)
	if err != nil {
		t.Fatalf("AutomationGroupFormationFromCandidate: %v", err)
	}

	formations := walkAutomationGroupFormations(t, sub)
	if len(formations) != 1 {
		t.Fatalf("formations count: got %d want 1", len(formations))
	}
	got := formations[0]
	if got.PatternParameters != "min-cluster=3;feature=window-size" {
		t.Errorf("PatternParameters: got %q want override", got.PatternParameters)
	}
	if got.Confidence != 0.85 {
		t.Errorf("Confidence override: got %f want 0.85", got.Confidence)
	}
	if got.EvidentialIndependence.Numerator != 3 || got.EvidentialIndependence.Denominator != 4 {
		t.Errorf("EI override: got %d/%d want 3/4",
			got.EvidentialIndependence.Numerator, got.EvidentialIndependence.Denominator)
	}
}

// bytes32 produces a 32-byte test hash from a short label (right-padded).
// Mirrors the §0157 helper pattern for deterministic test hashes.
func bytes32(label string) []byte {
	out := make([]byte, 32)
	copy(out, label)
	return out
}
