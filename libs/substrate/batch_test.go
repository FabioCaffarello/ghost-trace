package substrate

// What batching a commit is actually worth, measured rather than
// assumed.
//
// The roadmap records "batching the transaction ceilings at ~1.5x end
// to end" — but that was measured in PR-4.4, BEFORE inlining removed
// the blob fsync for payloads that fit. On that path a commit paid two
// fsyncs and batching amortised only one of them. Post-inlining the
// commit pays one, and one fsync per transaction amortised across N
// records is a different proposition. The old figure describes code
// that no longer runs.

import (
	"context"
	"fmt"
	"testing"
)

func TestWhatBatchingACommitIsWorth(t *testing.T) {
	measuring(t)
	ctx := context.Background()
	bodies, hashes := payloads(200_000)

	one := openAt(t, t.TempDir())
	single, err := rate(window, func(i int) error {
		return one.AppendCanonicalAt(ctx, bodies[i], hashes[i], 1, "t", 2, uint64(i+1))
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("one record per transaction   %9.0f /s", single)

	for _, n := range []int{8, 32, 128, 512} {
		s := openAt(t, t.TempDir())
		seq := uint64(0)
		batched, err := rate(window, func(i int) error {
			recs := make([]BatchRecord, n)
			for j := range recs {
				k := (i*n + j) % len(bodies)
				seq++
				recs[j] = BatchRecord{
					Payload: bodies[k], EventHash: hashes[k],
					EventTime: 1, MessageType: "t", Seq: seq,
				}
			}
			return s.AppendCanonicalBatch(ctx, recs, nil, 2)
		})
		if err != nil {
			t.Fatal(err)
		}
		// rate() counts CALLS; each call commits n records.
		perRecord := batched * float64(n)
		t.Logf("batch of %-4d                %9.0f /s  (%.1fx)", n, perRecord, perRecord/single)
	}
}

func TestABatchCommitsEveryRecordAndCountsDuplicatesOnce(t *testing.T) {
	// The property, on any hardware: a batch is not a shortcut past the
	// accounting. Every record lands, the position spans the whole
	// batch, and a redelivered record inside a batch is a duplicate
	// rather than a commit — the ADR-0010 rule that keeps unaccounted
	// from going negative.
	ctx := context.Background()
	s := openAt(t, t.TempDir())
	bodies, hashes := payloads(10)

	recs := make([]BatchRecord, 6)
	for i := range recs {
		recs[i] = BatchRecord{
			Payload: bodies[i], EventHash: hashes[i],
			EventTime: 1, MessageType: "t", Seq: uint64(i + 10),
		}
	}
	if err := s.AppendCanonicalBatch(ctx, recs, nil, 2); err != nil {
		t.Fatal(err)
	}

	pos, ok, err := s.Position(ctx)
	if err != nil || !ok {
		t.Fatalf("position: %v ok=%v", err, ok)
	}
	if pos.FirstSeq != 10 || pos.HighestSeq != 15 {
		t.Errorf("position spans %d..%d, want 10..15", pos.FirstSeq, pos.HighestSeq)
	}
	if pos.Committed != 6 {
		t.Errorf("committed = %d, want 6", pos.Committed)
	}
	if u := pos.Unaccounted(); u != 0 {
		t.Errorf("unaccounted = %d after a clean batch, want 0", u)
	}

	// The same batch again: every record is a duplicate, none a commit.
	if err := s.AppendCanonicalBatch(ctx, recs, nil, 2); err != nil {
		t.Fatal(err)
	}
	pos, _, _ = s.Position(ctx)
	if pos.Committed != 6 {
		t.Errorf("committed = %d after redelivering the same batch, want 6 — "+
			"counting a duplicate as a commit is what drove unaccounted to -70 "+
			"in Phase 4 (ADR-0010)", pos.Committed)
	}
	if pos.Duplicates != 6 {
		t.Errorf("duplicates = %d, want 6", pos.Duplicates)
	}
	if u := pos.Unaccounted(); u != 0 {
		t.Errorf("unaccounted = %d after a redelivered batch, want 0", u)
	}
}

func TestABadRecordFailsTheWholeBatchAndWritesNothing(t *testing.T) {
	// All or nothing, and nothing means nothing: a hash mismatch found
	// during verification must not leave the earlier records committed
	// or their blobs on disk.
	ctx := context.Background()
	s := openAt(t, t.TempDir())
	bodies, hashes := payloads(4)

	recs := []BatchRecord{
		{Payload: bodies[0], EventHash: hashes[0], EventTime: 1, MessageType: "t", Seq: 1},
		{Payload: bodies[1], EventHash: hashes[2], EventTime: 1, MessageType: "t", Seq: 2},
	}
	err := s.AppendCanonicalBatch(ctx, recs, nil, 2)
	if err == nil {
		t.Fatal("a payload that does not match its hash was accepted")
	}
	if _, ok, _ := s.Position(ctx); ok {
		t.Error("the batch failed and still advanced the durable position; a " +
			"rolled-back batch that moved the position would read as records " +
			"walked past and lost")
	}
	if n, err := s.Count(ctx); err != nil || n != 0 {
		t.Errorf("rows = %d after a failed batch, want 0 (err %v)", n, err)
	}
}

var _ = fmt.Sprintf

func TestARejectedSequenceIsAccountedInsideTheBatch(t *testing.T) {
	// The reason rejects travel with the batch rather than beside it.
	// Recorded separately, a reject would survive a rolled-back commit,
	// and redelivery would record it twice — `rejected` climbing past
	// the sequences actually walked, `unaccounted` going negative. That
	// is ADR-0010's failure arriving through a different door.
	ctx := context.Background()
	s := openAt(t, t.TempDir())
	bodies, hashes := payloads(4)

	recs := []BatchRecord{
		{Payload: bodies[0], EventHash: hashes[0], EventTime: 1, MessageType: "t", Seq: 1},
		{Payload: bodies[1], EventHash: hashes[1], EventTime: 1, MessageType: "t", Seq: 2},
	}
	if err := s.AppendCanonicalBatch(ctx, recs, []uint64{3, 4}, 2); err != nil {
		t.Fatal(err)
	}
	pos, ok, err := s.Position(ctx)
	if err != nil || !ok {
		t.Fatalf("position: %v ok=%v", err, ok)
	}
	if pos.FirstSeq != 1 || pos.HighestSeq != 4 {
		t.Errorf("span %d..%d, want 1..4 — the rejects are part of what this "+
			"batch walked", pos.FirstSeq, pos.HighestSeq)
	}
	if pos.Committed != 2 || pos.Rejected != 2 {
		t.Errorf("committed=%d rejected=%d, want 2 and 2", pos.Committed, pos.Rejected)
	}
	if u := pos.Unaccounted(); u != 0 {
		t.Errorf("unaccounted = %d; every sequence in the span is either held or "+
			"refused on purpose", u)
	}
}

func TestABatchOfNothingButRejectsStillAdvancesThePosition(t *testing.T) {
	// A run of malformed records is still a run of sequences walked. If
	// the position did not move, they would read as transport loss.
	ctx := context.Background()
	s := openAt(t, t.TempDir())
	if err := s.AppendCanonicalBatch(ctx, nil, []uint64{7, 8, 9}, 2); err != nil {
		t.Fatal(err)
	}
	pos, ok, _ := s.Position(ctx)
	if !ok || pos.FirstSeq != 7 || pos.HighestSeq != 9 || pos.Rejected != 3 {
		t.Errorf("position = %+v, want span 7..9 with 3 rejected", pos)
	}
	if u := pos.Unaccounted(); u != 0 {
		t.Errorf("unaccounted = %d, want 0", u)
	}
}
