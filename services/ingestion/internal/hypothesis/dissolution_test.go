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

func walkDissolutions(t *testing.T, sub *substrate.Substrate) []*eventsv1.BehavioralClusterDissolution {
	t.Helper()
	ctx := context.Background()
	var out []*eventsv1.BehavioralClusterDissolution
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != "ghosttrace.events.v1.BehavioralClusterDissolution" {
			return nil
		}
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return err
		}
		ev := &eventsv1.BehavioralClusterDissolution{}
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

func TestDissolveDirectFromFormation(t *testing.T) {
	// Dissolution may occur directly against a formation that was
	// never promoted — dissolution recognizes non-existence of the
	// underlying phenomenon, not withdrawal of operational use.
	sub, formationHash := formAndCollect(t)
	rep, err := Dissolve(context.Background(), sub, DissolveOptions{
		FormationEventHash: formationHash,
		DissolvedAt:        1716120000000000000,
		Reason:             "phenomenon never existed",
	}, nil)
	if err != nil {
		t.Fatalf("Dissolve: %v", err)
	}
	if rep.AlreadyDissolved {
		t.Errorf("unexpected AlreadyDissolved on first invocation")
	}

	dissolutions := walkDissolutions(t, sub)
	if len(dissolutions) != 1 {
		t.Fatalf("substrate carries %d dissolutions, want 1", len(dissolutions))
	}
	got := dissolutions[0]
	if got.Reason != "phenomenon never existed" {
		t.Errorf("reason: got %q, want %q", got.Reason, "phenomenon never existed")
	}
	if got.DissolvedAt != 1716120000000000000 {
		t.Errorf("dissolved_at: got %d, want 1716120000000000000", got.DissolvedAt)
	}
	for i, b := range got.FormationEventHash {
		if b != formationHash[i] {
			t.Errorf("formation_event_hash mismatch at byte %d: got %x, want %x", i, b, formationHash[i])
			break
		}
	}
}

func TestDissolveAfterPromoteDemote(t *testing.T) {
	// Dissolution may occur AFTER the optional promote/demote arc.
	// Substrate ends with all four lifecycle events present, plus
	// the underlying observations.
	sub, promotionHash := formAndPromote(t, 1000*int64(time.Second), 60)
	ctx := context.Background()

	// Demote first.
	if _, err := Demote(ctx, sub, DemoteOptions{
		PromotionEventHash: promotionHash,
		DemotedAt:          1100 * int64(time.Second),
		Reason:             "cycle close",
	}, nil); err != nil {
		t.Fatalf("Demote: %v", err)
	}

	// Recover the formation hash for dissolution.
	var formationHash [32]byte
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType == behavioralClusterFormationMessageType {
			formationHash = row.EventHash
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}

	rep, err := Dissolve(ctx, sub, DissolveOptions{
		FormationEventHash: formationHash,
		DissolvedAt:        1200 * int64(time.Second),
		Reason:             "phenomenon recognized non-existent",
	}, nil)
	if err != nil {
		t.Fatalf("Dissolve: %v", err)
	}
	if rep.AlreadyDissolved {
		t.Errorf("unexpected AlreadyDissolved on first invocation")
	}
	if len(walkDissolutions(t, sub)) != 1 {
		t.Fatalf("substrate carries %d dissolutions, want 1", len(walkDissolutions(t, sub)))
	}
}

func TestDissolveIdempotent(t *testing.T) {
	sub, formationHash := formAndCollect(t)
	opts := DissolveOptions{
		FormationEventHash: formationHash,
		DissolvedAt:        1200 * int64(time.Second),
		Reason:             "terminal",
	}

	ctx := context.Background()
	rep1, err := Dissolve(ctx, sub, opts, nil)
	if err != nil {
		t.Fatalf("first Dissolve: %v", err)
	}
	if rep1.AlreadyDissolved {
		t.Errorf("first invocation should not report AlreadyDissolved")
	}
	rep2, err := Dissolve(ctx, sub, opts, nil)
	if err != nil {
		t.Fatalf("second Dissolve: %v", err)
	}
	if !rep2.AlreadyDissolved {
		t.Errorf("second invocation should report AlreadyDissolved (content-hash collision)")
	}
	if rep1.DissolutionEventHashHex != rep2.DissolutionEventHashHex {
		t.Errorf("idempotency violated: hashes differ %q != %q", rep1.DissolutionEventHashHex, rep2.DissolutionEventHashHex)
	}
	if got := len(walkDissolutions(t, sub)); got != 1 {
		t.Errorf("substrate holds %d dissolutions after re-run; want 1", got)
	}
}

func TestDissolveVersioningProducesNewRecord(t *testing.T) {
	// Re-dissolving with a DIFFERENT dissolved_at OR different reason
	// produces a NEW dissolution event alongside the prior (§2.5
	// immutability — operation history records every parameter).
	sub, formationHash := formAndCollect(t)
	ctx := context.Background()
	if _, err := Dissolve(ctx, sub, DissolveOptions{
		FormationEventHash: formationHash,
		DissolvedAt:        1100 * int64(time.Second),
		Reason:             "first",
	}, nil); err != nil {
		t.Fatalf("first Dissolve: %v", err)
	}
	rep, err := Dissolve(ctx, sub, DissolveOptions{
		FormationEventHash: formationHash,
		DissolvedAt:        1100 * int64(time.Second),
		Reason:             "second",
	}, nil)
	if err != nil {
		t.Fatalf("second Dissolve: %v", err)
	}
	if rep.AlreadyDissolved {
		t.Errorf("changing reason should produce new record; got AlreadyDissolved")
	}
	if got := len(walkDissolutions(t, sub)); got != 2 {
		t.Errorf("after two dissolutions with distinct reasons, substrate holds %d dissolutions; want 2", got)
	}
}

func TestDissolveUnknownTarget(t *testing.T) {
	sub, _ := formAndCollect(t)
	var bogus [32]byte
	for i := range bogus {
		bogus[i] = byte(0xff - i)
	}
	_, err := Dissolve(context.Background(), sub, DissolveOptions{
		FormationEventHash: bogus,
		DissolvedAt:        1100 * int64(time.Second),
	}, nil)
	if !errors.Is(err, ErrTargetNotFound) {
		t.Errorf("expected ErrTargetNotFound; got %v", err)
	}
}

func TestDissolveWrongTypeTarget(t *testing.T) {
	// Pointing the formation hash at a PROMOTION row (not a formation)
	// should return ErrTargetWrongType — preserves §2.5-lifecycle-
	// integrity (dissolution references only formation events).
	sub, promotionHash := formAndPromote(t, 1000*int64(time.Second), 60)
	_, err := Dissolve(context.Background(), sub, DissolveOptions{
		FormationEventHash: promotionHash, // promotion hash; wrong type
		DissolvedAt:        1100 * int64(time.Second),
	}, nil)
	if !errors.Is(err, ErrTargetWrongType) {
		t.Errorf("expected ErrTargetWrongType; got %v", err)
	}
}

func TestDissolveDefaultDissolvedAt(t *testing.T) {
	sub, formationHash := formAndCollect(t)
	fixedNow := func() time.Time { return time.Unix(0, 9999999999) }
	_, err := Dissolve(context.Background(), sub, DissolveOptions{
		FormationEventHash: formationHash,
	}, fixedNow)
	if err != nil {
		t.Fatalf("Dissolve: %v", err)
	}
	dissolutions := walkDissolutions(t, sub)
	if len(dissolutions) != 1 {
		t.Fatalf("expected 1 dissolution; got %d", len(dissolutions))
	}
	if dissolutions[0].DissolvedAt != 9999999999 {
		t.Errorf("dissolved_at: got %d, want 9999999999 (from injected now)", dissolutions[0].DissolvedAt)
	}
}

func TestDissolveFullLifecycleInSubstrate(t *testing.T) {
	// After form → promote → demote → dissolve, the substrate
	// carries:
	// 2 DeclaredSession + 2 IngestionEvent + 1 Formation + 1 Promotion
	// + 1 Demotion + 1 Dissolution = 8 rows.
	sub, promotionHash := formAndPromote(t, 1000*int64(time.Second), 60)
	ctx := context.Background()
	if _, err := Demote(ctx, sub, DemoteOptions{
		PromotionEventHash: promotionHash,
		DemotedAt:          1100 * int64(time.Second),
		Reason:             "lifecycle test",
	}, nil); err != nil {
		t.Fatalf("Demote: %v", err)
	}

	var formationHash [32]byte
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType == behavioralClusterFormationMessageType {
			formationHash = row.EventHash
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	if _, err := Dissolve(ctx, sub, DissolveOptions{
		FormationEventHash: formationHash,
		DissolvedAt:        1200 * int64(time.Second),
		Reason:             "terminal",
	}, nil); err != nil {
		t.Fatalf("Dissolve: %v", err)
	}

	n, err := sub.Count(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 8 {
		t.Errorf("substrate Count: got %d, want 8 (2 DeclaredSession + 2 IngestionEvent + 1 Formation + 1 Promotion + 1 Demotion + 1 Dissolution)", n)
	}

	typeCounts := map[string]int{}
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		typeCounts[row.MessageType]++
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	wantCounts := map[string]int{
		"ghosttrace.events.v1.DeclaredSession":              2,
		"ghosttrace.events.v1.IngestionEvent":               2,
		"ghosttrace.events.v1.BehavioralClusterFormation":   1,
		"ghosttrace.events.v1.BehavioralClusterPromotion":   1,
		"ghosttrace.events.v1.BehavioralClusterDemotion":    1,
		"ghosttrace.events.v1.BehavioralClusterDissolution": 1,
	}
	for mt, want := range wantCounts {
		if got := typeCounts[mt]; got != want {
			t.Errorf("%s count: got %d, want %d", mt, got, want)
		}
	}
}
