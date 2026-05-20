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

// formThreeBehavioralClusters populates a substrate with three
// distinct shared-descriptor actor pairs and runs FormAll, producing
// three distinct BehavioralClusterFormation events. Returns the
// substrate plus the three formation hashes (alpha, beta, gamma) for
// merge-test wiring.
func formThreeBehavioralClusters(t *testing.T) (*substrate.Substrate, [32]byte, [32]byte, [32]byte) {
	t.Helper()
	sub, _ := populateSubstrate(t, []struct {
		ActorRef   string
		Descriptor []byte
	}{
		{"actor-a", []byte("alpha")},
		{"actor-b", []byte("alpha")},
		{"actor-c", []byte("beta")},
		{"actor-d", []byte("beta")},
		{"actor-e", []byte("gamma")},
		{"actor-f", []byte("gamma")},
	})

	ctx := context.Background()
	rep, err := FormAll(ctx, sub, SessionDescriptorSharedV1{MinClusterSize: 2}, func() time.Time { return time.Unix(0, 1) })
	if err != nil {
		t.Fatalf("FormAll: %v", err)
	}
	if rep.NewlyFormed != 3 {
		t.Fatalf("expected 3 formations; got %d", rep.NewlyFormed)
	}

	type formationEntry struct {
		hash [32]byte
		ev   *eventsv1.BehavioralClusterFormation
	}
	var entries []formationEntry
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != behavioralClusterFormationMessageType {
			return nil
		}
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return err
		}
		ev := &eventsv1.BehavioralClusterFormation{}
		if err := proto.Unmarshal(payload, ev); err != nil {
			return err
		}
		entries = append(entries, formationEntry{hash: row.EventHash, ev: ev})
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 formation rows; got %d", len(entries))
	}

	// Identify which formation corresponds to which descriptor by
	// the actor_refs membership.
	var alpha, beta, gamma [32]byte
	for _, e := range entries {
		members := e.ev.ActorRefs
		switch {
		case contains(members, "actor-a") && contains(members, "actor-b"):
			alpha = e.hash
		case contains(members, "actor-c") && contains(members, "actor-d"):
			beta = e.hash
		case contains(members, "actor-e") && contains(members, "actor-f"):
			gamma = e.hash
		default:
			t.Fatalf("unexpected formation members: %v", members)
		}
	}
	return sub, alpha, beta, gamma
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

func walkMerges(t *testing.T, sub *substrate.Substrate) []*eventsv1.BehavioralClusterMerge {
	t.Helper()
	ctx := context.Background()
	var out []*eventsv1.BehavioralClusterMerge
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != "ghosttrace.events.v1.BehavioralClusterMerge" {
			return nil
		}
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return err
		}
		ev := &eventsv1.BehavioralClusterMerge{}
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

func TestMergeHappyPath(t *testing.T) {
	sub, alpha, beta, gamma := formThreeBehavioralClusters(t)

	rep, err := Merge(context.Background(), sub, MergeOptions{
		AntecedentAFormationHash: alpha,
		AntecedentBFormationHash: beta,
		ProducedFormationHash:    gamma,
		MergedAt:                 1716120000000000000,
		Reason:                   "alpha and beta recognized as same phenomenon",
	}, nil)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if rep.AlreadyMerged {
		t.Errorf("unexpected AlreadyMerged on first invocation")
	}

	merges := walkMerges(t, sub)
	if len(merges) != 1 {
		t.Fatalf("substrate carries %d merges, want 1", len(merges))
	}
	got := merges[0]
	if got.Reason != "alpha and beta recognized as same phenomenon" {
		t.Errorf("reason: got %q", got.Reason)
	}
	if got.MergedAt != 1716120000000000000 {
		t.Errorf("merged_at: got %d, want 1716120000000000000", got.MergedAt)
	}
	if len(got.AntecedentFormationEventHashes) != 2 {
		t.Fatalf("antecedent_formation_event_hashes: got %d, want 2", len(got.AntecedentFormationEventHashes))
	}
	// Sorted ascending — verify byte-lex order.
	if bytes.Compare(got.AntecedentFormationEventHashes[0], got.AntecedentFormationEventHashes[1]) >= 0 {
		t.Errorf("antecedents not sorted ascending: %x vs %x", got.AntecedentFormationEventHashes[0], got.AntecedentFormationEventHashes[1])
	}
	if !bytes.Equal(got.ProducedFormationEventHash, gamma[:]) {
		t.Errorf("produced_formation_event_hash: got %x, want %x", got.ProducedFormationEventHash, gamma)
	}
}

func TestMergeArgumentOrderInvariance(t *testing.T) {
	// Merge is symmetric — calling Merge(A,B) and Merge(B,A) with the
	// same MergedAt + Reason MUST produce a single substrate row
	// (content-hash collision via ascending-sort normalization).
	sub, alpha, beta, gamma := formThreeBehavioralClusters(t)
	ctx := context.Background()

	rep1, err := Merge(ctx, sub, MergeOptions{
		AntecedentAFormationHash: alpha,
		AntecedentBFormationHash: beta,
		ProducedFormationHash:    gamma,
		MergedAt:                 1716120000000000000,
		Reason:                   "symmetric",
	}, nil)
	if err != nil {
		t.Fatalf("first Merge: %v", err)
	}

	rep2, err := Merge(ctx, sub, MergeOptions{
		AntecedentAFormationHash: beta, // swapped
		AntecedentBFormationHash: alpha,
		ProducedFormationHash:    gamma,
		MergedAt:                 1716120000000000000,
		Reason:                   "symmetric",
	}, nil)
	if err != nil {
		t.Fatalf("second Merge (swapped args): %v", err)
	}

	if rep1.MergeEventHashHex != rep2.MergeEventHashHex {
		t.Errorf("argument-order invariance violated: %q != %q", rep1.MergeEventHashHex, rep2.MergeEventHashHex)
	}
	if !rep2.AlreadyMerged {
		t.Errorf("swapped-args second invocation should report AlreadyMerged (content-hash collision)")
	}
	if got := len(walkMerges(t, sub)); got != 1 {
		t.Errorf("substrate holds %d merges; want 1 (symmetry collapses)", got)
	}
}

func TestMergeIdenticalAntecedents(t *testing.T) {
	sub, alpha, _, gamma := formThreeBehavioralClusters(t)
	_, err := Merge(context.Background(), sub, MergeOptions{
		AntecedentAFormationHash: alpha,
		AntecedentBFormationHash: alpha, // same as A
		ProducedFormationHash:    gamma,
	}, nil)
	if !errors.Is(err, ErrMergeAntecedentsIdentical) {
		t.Errorf("expected ErrMergeAntecedentsIdentical; got %v", err)
	}
}

func TestMergeIdempotent(t *testing.T) {
	sub, alpha, beta, gamma := formThreeBehavioralClusters(t)
	opts := MergeOptions{
		AntecedentAFormationHash: alpha,
		AntecedentBFormationHash: beta,
		ProducedFormationHash:    gamma,
		MergedAt:                 1716120000000000000,
		Reason:                   "idempotent",
	}
	ctx := context.Background()
	rep1, err := Merge(ctx, sub, opts, nil)
	if err != nil {
		t.Fatalf("first Merge: %v", err)
	}
	if rep1.AlreadyMerged {
		t.Errorf("first invocation should not report AlreadyMerged")
	}
	rep2, err := Merge(ctx, sub, opts, nil)
	if err != nil {
		t.Fatalf("second Merge: %v", err)
	}
	if !rep2.AlreadyMerged {
		t.Errorf("second invocation should report AlreadyMerged")
	}
	if rep1.MergeEventHashHex != rep2.MergeEventHashHex {
		t.Errorf("idempotency violated: %q != %q", rep1.MergeEventHashHex, rep2.MergeEventHashHex)
	}
	if got := len(walkMerges(t, sub)); got != 1 {
		t.Errorf("substrate holds %d merges after re-run; want 1", got)
	}
}

func TestMergeVersioningProducesNewRecord(t *testing.T) {
	sub, alpha, beta, gamma := formThreeBehavioralClusters(t)
	ctx := context.Background()
	if _, err := Merge(ctx, sub, MergeOptions{
		AntecedentAFormationHash: alpha,
		AntecedentBFormationHash: beta,
		ProducedFormationHash:    gamma,
		MergedAt:                 1716120000000000000,
		Reason:                   "first",
	}, nil); err != nil {
		t.Fatalf("first Merge: %v", err)
	}
	rep, err := Merge(ctx, sub, MergeOptions{
		AntecedentAFormationHash: alpha,
		AntecedentBFormationHash: beta,
		ProducedFormationHash:    gamma,
		MergedAt:                 1716120000000000000,
		Reason:                   "second", // different reason
	}, nil)
	if err != nil {
		t.Fatalf("second Merge: %v", err)
	}
	if rep.AlreadyMerged {
		t.Errorf("changing reason should produce new record; got AlreadyMerged")
	}
	if got := len(walkMerges(t, sub)); got != 2 {
		t.Errorf("substrate holds %d merges after distinct reasons; want 2", got)
	}
}

func TestMergeUnknownAntecedent(t *testing.T) {
	sub, alpha, _, gamma := formThreeBehavioralClusters(t)
	var bogus [32]byte
	for i := range bogus {
		bogus[i] = byte(0xff - i)
	}
	_, err := Merge(context.Background(), sub, MergeOptions{
		AntecedentAFormationHash: alpha,
		AntecedentBFormationHash: bogus,
		ProducedFormationHash:    gamma,
	}, nil)
	if !errors.Is(err, ErrTargetNotFound) {
		t.Errorf("expected ErrTargetNotFound for bogus antecedent; got %v", err)
	}
}

func TestMergeUnknownProduced(t *testing.T) {
	sub, alpha, beta, _ := formThreeBehavioralClusters(t)
	var bogus [32]byte
	for i := range bogus {
		bogus[i] = byte(0xaa)
	}
	_, err := Merge(context.Background(), sub, MergeOptions{
		AntecedentAFormationHash: alpha,
		AntecedentBFormationHash: beta,
		ProducedFormationHash:    bogus,
	}, nil)
	if !errors.Is(err, ErrTargetNotFound) {
		t.Errorf("expected ErrTargetNotFound for bogus produced; got %v", err)
	}
}

func TestMergeWrongTypeTarget(t *testing.T) {
	// Use a PROMOTION hash as an antecedent. Promotion is not a
	// formation; Merge MUST reject — preserves §2.5-lifecycle-integrity.
	sub, promotionHash := formAndPromote(t, 1000*int64(time.Second), 60)

	// Form a second BehavioralCluster on the same substrate to act
	// as the produced formation.
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

	_, err := Merge(ctx, sub, MergeOptions{
		AntecedentAFormationHash: formationHash,
		AntecedentBFormationHash: promotionHash, // promotion hash, wrong type
		ProducedFormationHash:    formationHash,
	}, nil)
	if !errors.Is(err, ErrTargetWrongType) {
		t.Errorf("expected ErrTargetWrongType; got %v", err)
	}
}

func TestMergeDefaultMergedAt(t *testing.T) {
	sub, alpha, beta, gamma := formThreeBehavioralClusters(t)
	fixedNow := func() time.Time { return time.Unix(0, 9999999999) }
	_, err := Merge(context.Background(), sub, MergeOptions{
		AntecedentAFormationHash: alpha,
		AntecedentBFormationHash: beta,
		ProducedFormationHash:    gamma,
	}, fixedNow)
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	merges := walkMerges(t, sub)
	if len(merges) != 1 {
		t.Fatalf("expected 1 merge; got %d", len(merges))
	}
	if merges[0].MergedAt != 9999999999 {
		t.Errorf("merged_at: got %d, want 9999999999 (from injected now)", merges[0].MergedAt)
	}
}

func TestMergeFullLifecycleInSubstrate(t *testing.T) {
	// After form-three → promote(α) → demote(α) → dissolve(α) → merge(α,β→γ):
	// substrate carries 6 DeclaredSession + 6 IngestionEvent + 3 Formation
	// + 1 Promotion + 1 Demotion + 1 Dissolution + 1 Merge = 19 rows
	// across 7 distinct message types.
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

	// Merge α + β → γ. Note: dissolving alpha first does NOT block
	// the merge — the substrate does not enforce cross-operation
	// ordering at this layer per §0048 carry-forwards.
	if _, err := Merge(ctx, sub, MergeOptions{
		AntecedentAFormationHash: alpha,
		AntecedentBFormationHash: beta,
		ProducedFormationHash:    gamma,
		MergedAt:                 1300 * int64(time.Second),
		Reason:                   "lifecycle test",
	}, nil); err != nil {
		t.Fatalf("Merge: %v", err)
	}

	n, err := sub.Count(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 19 {
		t.Errorf("substrate Count: got %d, want 19", n)
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
	}
	for mt, want := range wantCounts {
		if got := typeCounts[mt]; got != want {
			t.Errorf("%s count: got %d, want %d", mt, got, want)
		}
	}
}
