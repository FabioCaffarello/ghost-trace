package hypothesis

import (
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// formAndCollect populates a substrate with two actors sharing a
// session_descriptor, runs FormAll under SessionDescriptorSharedV1
// with min-cluster-size=2, and returns the substrate + the single
// formation event hash that resulted.
func formAndCollect(t *testing.T) (*substrate.Substrate, [32]byte) {
	t.Helper()
	sub, _ := populateSubstrate(t, []struct {
		ActorRef   string
		Descriptor []byte
	}{
		{"actor-a", []byte("alpha")},
		{"actor-b", []byte("alpha")},
	})

	rep, err := FormAll(context.Background(), sub, SessionDescriptorSharedV1{MinClusterSize: 2}, func() time.Time { return time.Unix(0, 1) })
	if err != nil {
		t.Fatalf("FormAll: %v", err)
	}
	if rep.NewlyFormed != 1 {
		t.Fatalf("expected 1 formation; got %d", rep.NewlyFormed)
	}

	formations := walkFormations(t, sub)
	if len(formations) != 1 {
		t.Fatalf("expected 1 formation row; got %d", len(formations))
	}
	// Recover the formation event hash by re-walking with EventRow
	// access. Simpler: re-marshal + hash.
	// Actually the substrate row has the hash; let's walk that way.
	var formationHash [32]byte
	if err := sub.WalkEvents(context.Background(), func(row substrate.EventRow) error {
		if row.MessageType == "ghosttrace.events.v1.BehavioralClusterFormation" {
			formationHash = row.EventHash
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	return sub, formationHash
}

func walkPromotions(t *testing.T, sub *substrate.Substrate) []*eventsv1.BehavioralClusterPromotion {
	t.Helper()
	ctx := context.Background()
	var out []*eventsv1.BehavioralClusterPromotion
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != "ghosttrace.events.v1.BehavioralClusterPromotion" {
			return nil
		}
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return err
		}
		ev := &eventsv1.BehavioralClusterPromotion{}
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

func TestPromoteHappyPath(t *testing.T) {
	sub, formationHash := formAndCollect(t)

	rep, err := Promote(context.Background(), sub, PromoteOptions{
		FormationEventHash: formationHash,
		PromotedAt:         1716120000000000000,
		CadenceSeconds:     3600,
		Reason:             "operational pilot",
	}, nil)
	if err != nil {
		t.Fatalf("Promote: %v", err)
	}
	if rep.AlreadyPromoted {
		t.Errorf("unexpectedly AlreadyPromoted on first invocation")
	}
	if rep.PromotionEventHashHex == "" {
		t.Errorf("missing PromotionEventHashHex")
	}

	promotions := walkPromotions(t, sub)
	if len(promotions) != 1 {
		t.Fatalf("substrate carries %d promotions, want 1", len(promotions))
	}
	got := promotions[0]
	if got.CadenceSeconds != 3600 {
		t.Errorf("cadence_seconds: got %d, want 3600", got.CadenceSeconds)
	}
	if got.PromotedAt != 1716120000000000000 {
		t.Errorf("promoted_at: got %d, want 1716120000000000000", got.PromotedAt)
	}
	if got.Reason != "operational pilot" {
		t.Errorf("reason: got %q, want %q", got.Reason, "operational pilot")
	}
	// formation_event_hash must reference the formation event's hash.
	for i, b := range got.FormationEventHash {
		if b != formationHash[i] {
			t.Errorf("formation_event_hash[%d]: got %x, want %x (hash mismatch at byte %d)", i, b, formationHash[i], i)
			break
		}
	}
}

func TestPromoteIdempotent(t *testing.T) {
	sub, formationHash := formAndCollect(t)
	opts := PromoteOptions{
		FormationEventHash: formationHash,
		PromotedAt:         1716120000000000000,
		CadenceSeconds:     3600,
		Reason:             "operational pilot",
	}

	ctx := context.Background()
	rep1, err := Promote(ctx, sub, opts, nil)
	if err != nil {
		t.Fatalf("first Promote: %v", err)
	}
	if rep1.AlreadyPromoted {
		t.Errorf("first invocation should not report AlreadyPromoted")
	}
	rep2, err := Promote(ctx, sub, opts, nil)
	if err != nil {
		t.Fatalf("second Promote: %v", err)
	}
	if !rep2.AlreadyPromoted {
		t.Errorf("second invocation should report AlreadyPromoted (content-hash collision)")
	}
	if rep1.PromotionEventHashHex != rep2.PromotionEventHashHex {
		t.Errorf("idempotency violated: hashes differ %q != %q", rep1.PromotionEventHashHex, rep2.PromotionEventHashHex)
	}
	if got := len(walkPromotions(t, sub)); got != 1 {
		t.Errorf("substrate holds %d promotions after re-run; want 1", got)
	}
}

func TestPromoteVersioningProducesNewRecord(t *testing.T) {
	// Re-promoting with a DIFFERENT cadence_seconds produces a NEW
	// promotion event alongside the prior one (§2.5 immutability —
	// every lifecycle event is a discrete substrate row).
	sub, formationHash := formAndCollect(t)
	ctx := context.Background()
	if _, err := Promote(ctx, sub, PromoteOptions{
		FormationEventHash: formationHash,
		PromotedAt:         1716120000000000000,
		CadenceSeconds:     3600,
	}, nil); err != nil {
		t.Fatalf("first Promote: %v", err)
	}
	rep, err := Promote(ctx, sub, PromoteOptions{
		FormationEventHash: formationHash,
		PromotedAt:         1716120000000000000,
		CadenceSeconds:     86400,
	}, nil)
	if err != nil {
		t.Fatalf("second Promote: %v", err)
	}
	if rep.AlreadyPromoted {
		t.Errorf("changing cadence_seconds should produce new record; got AlreadyPromoted")
	}
	if got := len(walkPromotions(t, sub)); got != 2 {
		t.Errorf("after two cadence parameterizations, substrate holds %d promotions; want 2", got)
	}
}

func TestPromoteRejectsZeroCadence(t *testing.T) {
	sub, formationHash := formAndCollect(t)
	_, err := Promote(context.Background(), sub, PromoteOptions{
		FormationEventHash: formationHash,
		CadenceSeconds:     0,
	}, nil)
	if err == nil {
		t.Fatal("Promote with cadence=0 should error")
	}
}

func TestPromoteUnknownTarget(t *testing.T) {
	sub, _ := formAndCollect(t)
	var bogus [32]byte
	for i := range bogus {
		bogus[i] = byte(i)
	}
	_, err := Promote(context.Background(), sub, PromoteOptions{
		FormationEventHash: bogus,
		CadenceSeconds:     3600,
	}, nil)
	if !errors.Is(err, ErrTargetNotFound) {
		t.Errorf("expected ErrTargetNotFound; got %v", err)
	}
}

func TestPromoteWrongTypeTarget(t *testing.T) {
	// Pointing the formation hash at a DeclaredSession row (not a
	// formation event) should return ErrTargetWrongType — preserves
	// §2.5 lifecycle integrity (promotion references only formations).
	sub, _ := populateSubstrate(t, []struct {
		ActorRef   string
		Descriptor []byte
	}{
		{"actor-a", []byte("alpha")},
	})
	// Find the DeclaredSession row's hash.
	var declHash [32]byte
	if err := sub.WalkEvents(context.Background(), func(row substrate.EventRow) error {
		if row.MessageType == "ghosttrace.events.v1.DeclaredSession" {
			declHash = row.EventHash
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}

	_, err := Promote(context.Background(), sub, PromoteOptions{
		FormationEventHash: declHash,
		CadenceSeconds:     3600,
	}, nil)
	if !errors.Is(err, ErrTargetWrongType) {
		t.Errorf("expected ErrTargetWrongType; got %v", err)
	}
}

func TestPromoteDefaultPromotedAt(t *testing.T) {
	// With PromotedAt=0, the function uses now().UnixNano(); the
	// resulting promotion event has a wall-clock-time PromotedAt.
	sub, formationHash := formAndCollect(t)
	fixedNow := func() time.Time { return time.Unix(0, 1234567890) }

	_, err := Promote(context.Background(), sub, PromoteOptions{
		FormationEventHash: formationHash,
		CadenceSeconds:     3600,
	}, fixedNow)
	if err != nil {
		t.Fatalf("Promote: %v", err)
	}
	promotions := walkPromotions(t, sub)
	if len(promotions) != 1 {
		t.Fatalf("expected 1 promotion; got %d", len(promotions))
	}
	if promotions[0].PromotedAt != 1234567890 {
		t.Errorf("promoted_at: got %d, want 1234567890 (from injected now)", promotions[0].PromotedAt)
	}
}

// walkCLIIngestionEvents returns every IngestionEvent record with
// channel="cli" — the CLI-channel actor-attribution pairs landed via
// §0097, ignoring the channel="test" IngestionEvents produced by the
// populateSubstrate ingest path (which pairs every Cat I observation
// with an IngestionEvent per §0038).
func walkCLIIngestionEvents(t *testing.T, sub *substrate.Substrate) []*eventsv1.IngestionEvent {
	t.Helper()
	ctx := context.Background()
	var out []*eventsv1.IngestionEvent
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != "ghosttrace.events.v1.IngestionEvent" {
			return nil
		}
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return err
		}
		ev := &eventsv1.IngestionEvent{}
		if err := proto.Unmarshal(payload, ev); err != nil {
			return err
		}
		if ev.Channel != "cli" {
			return nil
		}
		out = append(out, ev)
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	return out
}

func TestPromoteWithActorCommitsIngestionEventPair(t *testing.T) {
	// With Actor non-empty, Promote uses AppendPair to commit the
	// promotion AND an IngestionEvent atomically. The IngestionEvent
	// records channel="cli" + client_common_name=Actor per §0097.
	sub, formationHash := formAndCollect(t)

	rep, err := Promote(context.Background(), sub, PromoteOptions{
		FormationEventHash: formationHash,
		PromotedAt:         1716120000000000000,
		CadenceSeconds:     3600,
		Reason:             "operator-attributed promotion pilot",
		Actor:              "operator-alice",
	}, func() time.Time { return time.Unix(0, 1716120000000000000) })
	if err != nil {
		t.Fatalf("Promote: %v", err)
	}
	if rep.PromotionEventHashHex == "" {
		t.Errorf("missing PromotionEventHashHex")
	}
	if rep.IngestionEventHashHex == "" {
		t.Errorf("expected non-empty IngestionEventHashHex (Actor was supplied)")
	}

	promotions := walkPromotions(t, sub)
	if len(promotions) != 1 {
		t.Fatalf("substrate carries %d promotions, want 1", len(promotions))
	}

	ingestions := walkCLIIngestionEvents(t, sub)
	if len(ingestions) != 1 {
		t.Fatalf("substrate carries %d IngestionEvents, want 1", len(ingestions))
	}
	ing := ingestions[0]
	if ing.Channel != "cli" {
		t.Errorf("channel: got %q, want %q", ing.Channel, "cli")
	}
	if ing.ClientCommonName != "operator-alice" {
		t.Errorf("client_common_name: got %q, want %q", ing.ClientCommonName, "operator-alice")
	}
	if ing.ClientCertSha256 != "" {
		t.Errorf("client_cert_sha256: got %q, want empty (CLI channel, no mTLS)", ing.ClientCertSha256)
	}
	if len(ing.ClientSubjectAltNames) != 0 {
		t.Errorf("client_subject_alt_names: got %v, want empty (CLI channel, no mTLS)", ing.ClientSubjectAltNames)
	}

	// IngestionEvent.primary_event_hash must reference the promotion
	// event's content-hash (the pair-by-reference shape per §0038).
	if len(ing.PrimaryEventHash) != 32 {
		t.Fatalf("primary_event_hash length: got %d, want 32", len(ing.PrimaryEventHash))
	}
	// Recover the promotion hash from the report's hex.
	var promHash [32]byte
	for i := 0; i < 32; i++ {
		var nibble [2]byte
		copy(nibble[:], rep.PromotionEventHashHex[2*i:2*i+2])
		v, err := hexNibblesToByte(nibble[0], nibble[1])
		if err != nil {
			t.Fatalf("decode promotion hash: %v", err)
		}
		promHash[i] = v
	}
	for i := 0; i < 32; i++ {
		if ing.PrimaryEventHash[i] != promHash[i] {
			t.Errorf("primary_event_hash[%d]: got %x, want %x (must reference promotion event hash)", i, ing.PrimaryEventHash[i], promHash[i])
			break
		}
	}
}

func TestPromoteWithoutActorPreservesSingleAppendPath(t *testing.T) {
	// Without Actor, Promote uses the §0046 single-Append path; no
	// IngestionEvent is committed. Backward compatibility check.
	sub, formationHash := formAndCollect(t)

	rep, err := Promote(context.Background(), sub, PromoteOptions{
		FormationEventHash: formationHash,
		PromotedAt:         1716120000000000000,
		CadenceSeconds:     3600,
		Reason:             "backward-compat path",
	}, func() time.Time { return time.Unix(0, 1716120000000000000) })
	if err != nil {
		t.Fatalf("Promote: %v", err)
	}
	if rep.IngestionEventHashHex != "" {
		t.Errorf("IngestionEventHashHex: got %q, want empty (no Actor supplied)", rep.IngestionEventHashHex)
	}
	if got := walkCLIIngestionEvents(t, sub); len(got) != 0 {
		t.Errorf("substrate carries %d IngestionEvents, want 0", len(got))
	}
}

func TestPromoteWithActorIdempotent(t *testing.T) {
	// Re-running with identical (formation, cadence, reason,
	// promoted_at, actor) is idempotent — the second invocation
	// finds the same promotion hash + the same IngestionEvent hash,
	// no new rows committed.
	sub, formationHash := formAndCollect(t)
	opts := PromoteOptions{
		FormationEventHash: formationHash,
		PromotedAt:         1716120000000000000,
		CadenceSeconds:     3600,
		Reason:             "idempotent actor pilot",
		Actor:              "operator-bob",
	}
	fixedNow := func() time.Time { return time.Unix(0, 1716120000000000000) }

	rep1, err := Promote(context.Background(), sub, opts, fixedNow)
	if err != nil {
		t.Fatalf("Promote 1: %v", err)
	}
	rep2, err := Promote(context.Background(), sub, opts, fixedNow)
	if err != nil {
		t.Fatalf("Promote 2: %v", err)
	}
	if rep1.PromotionEventHashHex != rep2.PromotionEventHashHex {
		t.Errorf("promotion hash differs across invocations: %q vs %q", rep1.PromotionEventHashHex, rep2.PromotionEventHashHex)
	}
	if rep1.IngestionEventHashHex != rep2.IngestionEventHashHex {
		t.Errorf("ingestion hash differs across invocations: %q vs %q", rep1.IngestionEventHashHex, rep2.IngestionEventHashHex)
	}
	if !rep2.AlreadyPromoted {
		t.Errorf("expected AlreadyPromoted=true on second invocation")
	}
	if got := walkPromotions(t, sub); len(got) != 1 {
		t.Errorf("substrate carries %d promotions, want 1", len(got))
	}
	if got := walkCLIIngestionEvents(t, sub); len(got) != 1 {
		t.Errorf("substrate carries %d IngestionEvents, want 1", len(got))
	}
}

// hexNibblesToByte parses two hex characters into a byte. Local helper
// to avoid pulling encoding/hex into the test for one operation.
func hexNibblesToByte(hi, lo byte) (byte, error) {
	h, err := nibble(hi)
	if err != nil {
		return 0, err
	}
	l, err := nibble(lo)
	if err != nil {
		return 0, err
	}
	return h<<4 | l, nil
}

func nibble(b byte) (byte, error) {
	switch {
	case b >= '0' && b <= '9':
		return b - '0', nil
	case b >= 'a' && b <= 'f':
		return b - 'a' + 10, nil
	case b >= 'A' && b <= 'F':
		return b - 'A' + 10, nil
	default:
		return 0, errors.New("invalid hex nibble")
	}
}
