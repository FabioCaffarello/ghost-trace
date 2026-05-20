package hypothesis

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

func walkSplits(t *testing.T, sub *substrate.Substrate) []*eventsv1.BehavioralClusterSplit {
	t.Helper()
	ctx := context.Background()
	var out []*eventsv1.BehavioralClusterSplit
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != "ghosttrace.events.v1.BehavioralClusterSplit" {
			return nil
		}
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return err
		}
		ev := &eventsv1.BehavioralClusterSplit{}
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

func TestSplitHappyPath(t *testing.T) {
	// Use the three-formation substrate from merge tests: alpha as
	// antecedent, beta + gamma as successors. Substrate-side this is
	// indistinguishable from a real split; the operational claim is
	// in the split event itself.
	sub, alpha, beta, gamma := formThreeBehavioralClusters(t)

	rep, err := Split(context.Background(), sub, SplitOptions{
		AntecedentFormationHash:  alpha,
		SuccessorFormationHashes: [][32]byte{beta, gamma},
		SplitAt:                  1716120000000000000,
		Reason:                   "alpha recognized as containing two distinct phenomena",
	}, nil)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}
	if rep.AlreadySplit {
		t.Errorf("unexpected AlreadySplit on first invocation")
	}

	splits := walkSplits(t, sub)
	if len(splits) != 1 {
		t.Fatalf("substrate carries %d splits, want 1", len(splits))
	}
	got := splits[0]
	if got.Reason != "alpha recognized as containing two distinct phenomena" {
		t.Errorf("reason: got %q", got.Reason)
	}
	if got.SplitAt != 1716120000000000000 {
		t.Errorf("split_at: got %d, want 1716120000000000000", got.SplitAt)
	}
	if !bytes.Equal(got.AntecedentFormationEventHash, alpha[:]) {
		t.Errorf("antecedent_formation_event_hash mismatch")
	}
	if len(got.SuccessorFormationEventHashes) != 2 {
		t.Fatalf("successor_formation_event_hashes: got %d, want 2", len(got.SuccessorFormationEventHashes))
	}
	// Sorted ascending.
	if bytes.Compare(got.SuccessorFormationEventHashes[0], got.SuccessorFormationEventHashes[1]) >= 0 {
		t.Errorf("successors not sorted ascending: %x vs %x", got.SuccessorFormationEventHashes[0], got.SuccessorFormationEventHashes[1])
	}
}

func TestSplitSuccessorOrderInvariance(t *testing.T) {
	// Split([beta, gamma]) and Split([gamma, beta]) MUST produce a
	// single substrate row (content-hash collision via ascending-
	// sort normalization — successors form a SET).
	sub, alpha, beta, gamma := formThreeBehavioralClusters(t)
	ctx := context.Background()

	rep1, err := Split(ctx, sub, SplitOptions{
		AntecedentFormationHash:  alpha,
		SuccessorFormationHashes: [][32]byte{beta, gamma},
		SplitAt:                  1716120000000000000,
		Reason:                   "order-invariance",
	}, nil)
	if err != nil {
		t.Fatalf("first Split: %v", err)
	}

	rep2, err := Split(ctx, sub, SplitOptions{
		AntecedentFormationHash:  alpha,
		SuccessorFormationHashes: [][32]byte{gamma, beta}, // swapped
		SplitAt:                  1716120000000000000,
		Reason:                   "order-invariance",
	}, nil)
	if err != nil {
		t.Fatalf("second Split (swapped successors): %v", err)
	}

	if rep1.SplitEventHashHex != rep2.SplitEventHashHex {
		t.Errorf("successor-order invariance violated: %q != %q", rep1.SplitEventHashHex, rep2.SplitEventHashHex)
	}
	if !rep2.AlreadySplit {
		t.Errorf("swapped-successors invocation should report AlreadySplit (content-hash collision)")
	}
	if got := len(walkSplits(t, sub)); got != 1 {
		t.Errorf("substrate holds %d splits; want 1 (set-equality collapses)", got)
	}
}

func TestSplitInsufficientSuccessors(t *testing.T) {
	sub, alpha, beta, _ := formThreeBehavioralClusters(t)
	_, err := Split(context.Background(), sub, SplitOptions{
		AntecedentFormationHash:  alpha,
		SuccessorFormationHashes: [][32]byte{beta}, // single successor
	}, nil)
	if !errors.Is(err, ErrSplitInsufficientSuccessors) {
		t.Errorf("expected ErrSplitInsufficientSuccessors; got %v", err)
	}
}

func TestSplitDuplicateSuccessors(t *testing.T) {
	sub, alpha, beta, _ := formThreeBehavioralClusters(t)
	_, err := Split(context.Background(), sub, SplitOptions{
		AntecedentFormationHash:  alpha,
		SuccessorFormationHashes: [][32]byte{beta, beta}, // duplicate
	}, nil)
	if !errors.Is(err, ErrSplitSuccessorsNotDistinct) {
		t.Errorf("expected ErrSplitSuccessorsNotDistinct for duplicate; got %v", err)
	}
}

func TestSplitAntecedentEqualsSuccessor(t *testing.T) {
	sub, alpha, beta, _ := formThreeBehavioralClusters(t)
	_, err := Split(context.Background(), sub, SplitOptions{
		AntecedentFormationHash:  alpha,
		SuccessorFormationHashes: [][32]byte{alpha, beta}, // antecedent reappears
	}, nil)
	if !errors.Is(err, ErrSplitSuccessorsNotDistinct) {
		t.Errorf("expected ErrSplitSuccessorsNotDistinct for antecedent-overlap; got %v", err)
	}
}

func TestSplitIdempotent(t *testing.T) {
	sub, alpha, beta, gamma := formThreeBehavioralClusters(t)
	opts := SplitOptions{
		AntecedentFormationHash:  alpha,
		SuccessorFormationHashes: [][32]byte{beta, gamma},
		SplitAt:                  1716120000000000000,
		Reason:                   "idempotent",
	}
	ctx := context.Background()
	rep1, err := Split(ctx, sub, opts, nil)
	if err != nil {
		t.Fatalf("first Split: %v", err)
	}
	if rep1.AlreadySplit {
		t.Errorf("first invocation should not report AlreadySplit")
	}
	rep2, err := Split(ctx, sub, opts, nil)
	if err != nil {
		t.Fatalf("second Split: %v", err)
	}
	if !rep2.AlreadySplit {
		t.Errorf("second invocation should report AlreadySplit")
	}
	if rep1.SplitEventHashHex != rep2.SplitEventHashHex {
		t.Errorf("idempotency violated: %q != %q", rep1.SplitEventHashHex, rep2.SplitEventHashHex)
	}
	if got := len(walkSplits(t, sub)); got != 1 {
		t.Errorf("substrate holds %d splits after re-run; want 1", got)
	}
}

func TestSplitVersioningProducesNewRecord(t *testing.T) {
	sub, alpha, beta, gamma := formThreeBehavioralClusters(t)
	ctx := context.Background()
	if _, err := Split(ctx, sub, SplitOptions{
		AntecedentFormationHash:  alpha,
		SuccessorFormationHashes: [][32]byte{beta, gamma},
		SplitAt:                  1716120000000000000,
		Reason:                   "first",
	}, nil); err != nil {
		t.Fatalf("first Split: %v", err)
	}
	rep, err := Split(ctx, sub, SplitOptions{
		AntecedentFormationHash:  alpha,
		SuccessorFormationHashes: [][32]byte{beta, gamma},
		SplitAt:                  1716120000000000000,
		Reason:                   "second", // different reason
	}, nil)
	if err != nil {
		t.Fatalf("second Split: %v", err)
	}
	if rep.AlreadySplit {
		t.Errorf("changing reason should produce new record; got AlreadySplit")
	}
	if got := len(walkSplits(t, sub)); got != 2 {
		t.Errorf("substrate holds %d splits after distinct reasons; want 2", got)
	}
}

func TestSplitUnknownAntecedent(t *testing.T) {
	sub, _, beta, gamma := formThreeBehavioralClusters(t)
	var bogus [32]byte
	for i := range bogus {
		bogus[i] = byte(0xff - i)
	}
	_, err := Split(context.Background(), sub, SplitOptions{
		AntecedentFormationHash:  bogus,
		SuccessorFormationHashes: [][32]byte{beta, gamma},
	}, nil)
	if !errors.Is(err, ErrTargetNotFound) {
		t.Errorf("expected ErrTargetNotFound for bogus antecedent; got %v", err)
	}
}

func TestSplitUnknownSuccessor(t *testing.T) {
	sub, alpha, beta, _ := formThreeBehavioralClusters(t)
	var bogus [32]byte
	for i := range bogus {
		bogus[i] = byte(0xaa)
	}
	_, err := Split(context.Background(), sub, SplitOptions{
		AntecedentFormationHash:  alpha,
		SuccessorFormationHashes: [][32]byte{beta, bogus},
	}, nil)
	if !errors.Is(err, ErrTargetNotFound) {
		t.Errorf("expected ErrTargetNotFound for bogus successor; got %v", err)
	}
}

func TestSplitWrongTypeTarget(t *testing.T) {
	// Use a PROMOTION hash as a successor. Promotion is not a
	// formation; Split MUST reject — preserves §2.5-lifecycle-integrity.
	sub, promotionHash := formAndPromote(t, 1000*int64(time.Second), 60)

	// Recover the formation hash to use as the antecedent.
	var formationHash [32]byte
	ctx := context.Background()
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType == behavioralClusterFormationMessageType {
			formationHash = row.EventHash
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	// Use a synthetic non-equal "second successor" to clear the
	// distinctness check; the promotion-hash successor should still
	// fail wrong-type validation. Any byte-distinct hash works.
	var otherSuccessor [32]byte
	for i := range otherSuccessor {
		otherSuccessor[i] = byte(0x5a)
	}

	_, err := Split(ctx, sub, SplitOptions{
		AntecedentFormationHash:  formationHash,
		SuccessorFormationHashes: [][32]byte{promotionHash, otherSuccessor}, // promotion is wrong type
	}, nil)
	if !errors.Is(err, ErrTargetWrongType) {
		t.Errorf("expected ErrTargetWrongType for promotion-hash successor; got %v", err)
	}
}

func TestSplitDefaultSplitAt(t *testing.T) {
	sub, alpha, beta, gamma := formThreeBehavioralClusters(t)
	fixedNow := func() time.Time { return time.Unix(0, 9999999999) }
	_, err := Split(context.Background(), sub, SplitOptions{
		AntecedentFormationHash:  alpha,
		SuccessorFormationHashes: [][32]byte{beta, gamma},
	}, fixedNow)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}
	splits := walkSplits(t, sub)
	if len(splits) != 1 {
		t.Fatalf("expected 1 split; got %d", len(splits))
	}
	if splits[0].SplitAt != 9999999999 {
		t.Errorf("split_at: got %d, want 9999999999 (from injected now)", splits[0].SplitAt)
	}
}

func TestSplitAllSixLifecycleOpsInSubstrate(t *testing.T) {
	// After form-three → promote(α) → demote(α) → dissolve(α) →
	// merge(α,β→γ) → split(γ→[α,β]): substrate carries all SIX
	// lifecycle operation types alongside 6 DeclaredSession +
	// 6 IngestionEvent + 3 Formation = 6+6+3+1+1+1+1+1 = 20 rows.
	//
	// Operational note: splitting γ into successors that include the
	// previously-merged α + β is semantically odd (it "un-merges"),
	// but the substrate accepts the discrete event — Charter §2.5 +
	// §0048/§0049 carry-forwards explicitly defer cross-operation
	// ordering to projection-layer design. This test exercises the
	// structural surface, not operational coherence.
	sub, alpha, beta, gamma := formThreeBehavioralClusters(t)
	ctx := context.Background()

	// Promote α.
	promRep, err := Promote(ctx, sub, PromoteOptions{
		FormationEventHash: alpha,
		PromotedAt:         1000 * int64(time.Second),
		CadenceSeconds:     60,
		Reason:             "lifecycle test",
	}, nil)
	if err != nil {
		t.Fatalf("Promote: %v", err)
	}
	var promotionHash [32]byte
	if raw, err := hexDecode(promRep.PromotionEventHashHex); err == nil {
		copy(promotionHash[:], raw)
	} else {
		t.Fatalf("decode promotion hash: %v", err)
	}

	// Demote.
	if _, err := Demote(ctx, sub, DemoteOptions{
		PromotionEventHash: promotionHash,
		DemotedAt:          1100 * int64(time.Second),
		Reason:             "lifecycle test",
	}, nil); err != nil {
		t.Fatalf("Demote: %v", err)
	}

	// Dissolve α.
	if _, err := Dissolve(ctx, sub, DissolveOptions{
		FormationEventHash: alpha,
		DissolvedAt:        1200 * int64(time.Second),
		Reason:             "lifecycle test",
	}, nil); err != nil {
		t.Fatalf("Dissolve: %v", err)
	}

	// Merge α + β → γ.
	if _, err := Merge(ctx, sub, MergeOptions{
		AntecedentAFormationHash: alpha,
		AntecedentBFormationHash: beta,
		ProducedFormationHash:    gamma,
		MergedAt:                 1300 * int64(time.Second),
		Reason:                   "lifecycle test",
	}, nil); err != nil {
		t.Fatalf("Merge: %v", err)
	}

	// Split γ → [α, β]. Distinct from antecedent (γ); successors
	// distinct from each other.
	if _, err := Split(ctx, sub, SplitOptions{
		AntecedentFormationHash:  gamma,
		SuccessorFormationHashes: [][32]byte{alpha, beta},
		SplitAt:                  1400 * int64(time.Second),
		Reason:                   "lifecycle test",
	}, nil); err != nil {
		t.Fatalf("Split: %v", err)
	}

	n, err := sub.Count(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 20 {
		t.Errorf("substrate Count: got %d, want 20", n)
	}

	typeCounts := map[string]int{}
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		typeCounts[row.MessageType]++
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	wantCounts := map[string]int{
		"ghosttrace.events.v1.DeclaredSession":              6,
		"ghosttrace.events.v1.IngestionEvent":               6,
		"ghosttrace.events.v1.BehavioralClusterFormation":   3,
		"ghosttrace.events.v1.BehavioralClusterPromotion":   1,
		"ghosttrace.events.v1.BehavioralClusterDemotion":    1,
		"ghosttrace.events.v1.BehavioralClusterDissolution": 1,
		"ghosttrace.events.v1.BehavioralClusterMerge":       1,
		"ghosttrace.events.v1.BehavioralClusterSplit":       1,
	}
	for mt, want := range wantCounts {
		if got := typeCounts[mt]; got != want {
			t.Errorf("%s count: got %d, want %d", mt, got, want)
		}
	}
}
