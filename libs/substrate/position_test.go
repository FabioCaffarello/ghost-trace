package substrate_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/FabioCaffarello/ghost-trace/libs/canonical"
	"github.com/FabioCaffarello/ghost-trace/libs/substrate"
)

func open(t *testing.T, dir string) *substrate.Substrate {
	t.Helper()
	s, err := substrate.Open(context.Background(),
		filepath.Join(dir, "events.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func commit(t *testing.T, s *substrate.Substrate, body string, seq uint64) {
	t.Helper()
	payload := []byte(body)
	if err := s.AppendCanonicalAt(context.Background(), payload, canonical.Hash(payload),
		1, "ghosttrace.events.v1.Outcome", 2, seq); err != nil {
		t.Fatalf("append at %d: %v", seq, err)
	}
}

func position(t *testing.T, s *substrate.Substrate) substrate.Position {
	t.Helper()
	p, ok, err := s.Position(context.Background())
	if err != nil {
		t.Fatalf("position: %v", err)
	}
	if !ok {
		t.Fatal("position was never recorded")
	}
	return p
}

func TestAFreshArchiveHasNoPositionRatherThanAZeroOne(t *testing.T) {
	// The repository's own rule, in the one place it is most tempting to
	// break: an archive that has never consumed anything must not report
	// "committed 0 of span 0", because that is indistinguishable from an
	// archive whose bookkeeping was never wired.
	s := open(t, t.TempDir())

	p, ok, err := s.Position(context.Background())
	if err != nil {
		t.Fatalf("position: %v", err)
	}
	if ok {
		t.Errorf("a fresh archive reports a position %+v; absence is not zero", p)
	}
}

func TestThePositionSurvivesReopening(t *testing.T) {
	// THE property 3.4 proved was missing. Process-local counters reset,
	// so an archive that restarts mid-backlog forgets what it committed
	// and every reconciliation built on those counters is a
	// reconciliation of one process's lifetime.
	dir := t.TempDir()

	s := open(t, dir)
	commit(t, s, "one", 10)
	commit(t, s, "two", 11)
	commit(t, s, "three", 12)
	before := position(t, s)
	if err := s.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	reopened := open(t, dir)
	after := position(t, reopened)

	if after != before {
		t.Fatalf("position after reopening = %+v, want %+v — a restart must not "+
			"lose the archive's place", after, before)
	}
	if after.Committed != 3 || after.FirstSeq != 10 || after.HighestSeq != 12 {
		t.Errorf("position = %+v, want first=10 highest=12 committed=3", after)
	}
	if after.Unaccounted() != 0 {
		t.Errorf("unaccounted = %d over a complete run, want 0", after.Unaccounted())
	}
}

func TestAGapInTheStreamShowsAsUnaccounted(t *testing.T) {
	// Sequences 5 and 6 never arrive. The span still walks past them,
	// because 7 did, and nothing accounts for them.
	s := open(t, t.TempDir())
	commit(t, s, "a", 4)
	commit(t, s, "d", 7)

	p := position(t, s)
	if got, want := p.Span(), int64(4); got != want {
		t.Errorf("span = %d, want %d", got, want)
	}
	if got, want := p.Unaccounted(), int64(2); got != want {
		t.Errorf("unaccounted = %d, want %d — two sequences entered the stream "+
			"inside the walked range and are not in the archive", got, want)
	}
}

func TestARefusalIsAccountedRatherThanLost(t *testing.T) {
	// A record the archive deliberately refused is not a record the
	// stream lost. Without RecordRejected the sequence would still be
	// walked past and the refusal would read as unexplained loss —
	// blaming the transport for a decision this service made and logged.
	s := open(t, t.TempDir())
	commit(t, s, "good", 1)
	if err := s.RecordRejected(context.Background(), 2); err != nil {
		t.Fatalf("record rejected: %v", err)
	}
	commit(t, s, "also good", 3)

	p := position(t, s)
	if p.Rejected != 1 {
		t.Errorf("rejected = %d, want 1", p.Rejected)
	}
	if p.Unaccounted() != 0 {
		t.Errorf("unaccounted = %d, want 0 — a refusal explains its sequence", p.Unaccounted())
	}
}

func TestOutOfOrderDeliveryCannotWalkTheBoundsBackwards(t *testing.T) {
	// JetStream permits concurrent handling, so sequences do not
	// necessarily arrive in order. first_seq must stay the lowest ever
	// seen and highest_seq the highest, or the span — and therefore the
	// loss figure — depends on delivery order.
	s := open(t, t.TempDir())
	commit(t, s, "middle", 20)
	commit(t, s, "late", 30)
	commit(t, s, "early", 10)

	p := position(t, s)
	if p.FirstSeq != 10 || p.HighestSeq != 30 {
		t.Errorf("bounds = [%d, %d], want [10, 30]", p.FirstSeq, p.HighestSeq)
	}
	if p.Committed != 3 {
		t.Errorf("committed = %d, want 3", p.Committed)
	}
}

func TestARedeliveredRecordCountsTwiceAndStoresOnce(t *testing.T) {
	// Committed counts commit OPERATIONS. The same record arriving twice
	// carries the same stream sequence, dedups to one row, and must
	// account for its sequence exactly once — counting rows instead would
	// have made every redelivery look like a vanished record.
	s := open(t, t.TempDir())
	commit(t, s, "same", 1)
	commit(t, s, "same", 1)
	commit(t, s, "other", 2)

	p := position(t, s)
	if p.Span() != 2 {
		t.Errorf("span = %d, want 2", p.Span())
	}
	if p.Committed != 3 {
		t.Errorf("committed = %d, want 3 commit operations", p.Committed)
	}

	rows, err := s.Count(context.Background())
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if rows != 2 {
		t.Errorf("rows = %d, want 2 — content addressing collapses the duplicate", rows)
	}
	// The difference is the duplicate volume, and it is informative
	// rather than a discrepancy.
	if p.Committed-rows != 1 {
		t.Errorf("committed - rows = %d, want 1 duplicate", p.Committed-rows)
	}
}

func TestASequenceOfZeroIsRefused(t *testing.T) {
	// A zero is a caller that did not plumb the sequence through.
	// Accepting it would drag first_seq to zero and make every later
	// audit report a span the size of the whole stream — which reads as
	// catastrophic loss that never happened.
	s := open(t, t.TempDir())
	payload := []byte("x")

	err := s.AppendCanonicalAt(context.Background(), payload, canonical.Hash(payload),
		1, "t", 2, 0)
	if !errors.Is(err, substrate.ErrNoSequence) {
		t.Errorf("AppendCanonicalAt with seq 0: %v, want ErrNoSequence", err)
	}
	if err := s.RecordRejected(context.Background(), 0); !errors.Is(err, substrate.ErrNoSequence) {
		t.Errorf("RecordRejected with seq 0: %v, want ErrNoSequence", err)
	}
	if _, ok, _ := s.Position(context.Background()); ok {
		t.Error("a refused write recorded a position anyway")
	}
}

func TestAFailedCommitAdvancesNeitherTheRowsNorThePosition(t *testing.T) {
	// The reason the two writes share a transaction. A record whose hash
	// does not describe it is refused before anything is written, and the
	// position must not move — an archive that counted a commit it did
	// not perform reports less loss than there is.
	s := open(t, t.TempDir())
	commit(t, s, "real", 1)

	wrong := canonical.Hash([]byte("something else"))
	err := s.AppendCanonicalAt(context.Background(), []byte("payload"), wrong,
		1, "t", 2, 2)
	if !errors.Is(err, substrate.ErrHashMismatch) {
		t.Fatalf("err = %v, want ErrHashMismatch", err)
	}

	p := position(t, s)
	if p.Committed != 1 || p.HighestSeq != 1 {
		t.Errorf("position = %+v after a refused write, want committed=1 highest=1", p)
	}
}
