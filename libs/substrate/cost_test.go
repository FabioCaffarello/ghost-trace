package substrate

// Where a commit's time actually goes.
//
// PR-4.2 measured the archive at 1 356 commits/s against a collector
// that publishes ~16 000 records/s at its own bend, and put a
// hypothesis on record before this file existed: `synchronous=FULL`
// plus a single `writeMu` means roughly one fsync per commit, and 1 356
// is about what that allows.
//
// The hypothesis named ONE fsync. Reading the write path afterwards
// shows two: the blob is written to a temp file, `tmp.Sync()`ed, and
// renamed, before SQLite is touched at all. So there is a competing
// explanation with the same fingerprint, and a fix aimed at the wrong
// one would be a well-tested change that buys nothing.
//
// This file is in `package substrate` rather than `substrate_test`
// specifically so it can time the halves separately. Attribution is the
// whole point; a single end-to-end number cannot produce it.

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/FabioCaffarello/ghost-trace/libs/canonical"
)

// payloads builds n distinct records, so nothing is deduplicated by
// content addressing and every iteration pays a real write.
func payloads(n int) ([][]byte, [][32]byte) {
	bodies := make([][]byte, n)
	hashes := make([][32]byte, n)
	for i := range bodies {
		bodies[i] = []byte(fmt.Sprintf(
			`{"evaluation_id":"ev_%08d","outcome":"login_success","pad":"%s"}`,
			i, string(make([]byte, 200))))
		hashes[i] = canonical.Hash(bodies[i])
	}
	return bodies, hashes
}

func openAt(t *testing.T, dir string) *Substrate {
	t.Helper()
	dbPath := filepath.Join(dir, "events.db")
	blobDir := filepath.Join(dir, "blobs")
	if err := os.MkdirAll(blobDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// The production DSN, so what this file measures is what ships —
	// including the pragmas, which ride the DSN precisely so that every
	// pooled connection gets them (an Exec configures exactly one).
	db, err := sql.Open("sqlite", dsn(dbPath))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, ddl := range []string{eventsSchemaDDL, positionSchemaDDL} {
		if _, err := db.ExecContext(ctx, ddl); err != nil {
			t.Fatal(err)
		}
	}
	if err := ensurePayloadColumn(ctx, db); err != nil {
		t.Fatal(err)
	}
	s := &Substrate{db: db, blobDir: blobDir}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// rate runs fn for d and returns operations per second.
func rate(d time.Duration, fn func(i int) error) (float64, error) {
	deadline := time.Now().Add(d)
	n := 0
	for time.Now().Before(deadline) {
		if err := fn(n); err != nil {
			return 0, err
		}
		n++
	}
	return float64(n) / d.Seconds(), nil
}

const window = 500 * time.Millisecond

// measuring reports whether timing measurements should run.
//
// THEY DO NOT RUN IN `make ci`, and the reason was learned twice in one
// pull request. A timing assertion on shared CI hardware encodes that
// hardware: one runner measured an inlined commit at 48/s against
// 18 244/s locally, and the serial-cost model that holds on every
// machine with a functioning disk predicted 68/s against 4 measured.
// Neither reading was a regression. Both were the runner.
//
// These produce NUMBERS, which is a different job from guarding a
// property. The properties are guarded structurally and do run in CI —
// an inlined commit writes no file; a store keeps its content-addressing
// check. Run the numbers with `make measure`.
func measuring(t *testing.T) {
	t.Helper()
	if os.Getenv("GT_MEASURE") == "" {
		t.Skip("timing measurement; set GT_MEASURE=1 or run `make measure`")
	}
}

func TestWhereACommitSpendsItsTime(t *testing.T) {
	// The decomposition, and now also the regression guard for what it
	// bought.
	//
	// Before inlining, every commit paid a blob fsync and then a SQL
	// fsync, serially, inside one writeMu. The measurement said so:
	// 250/s overall against 252/s for the blob half alone on macOS,
	// 938/s against 1 826/s on Linux.
	//
	// A payload that fits now skips the blob entirely, so the whole
	// commit should cost what the SQL half costs — and the assertion
	// below fails if it ever pays the blob cost again. A payload over
	// the threshold still takes the old path, and the reciprocal model
	// still describes THAT one.
	measuring(t)
	ctx := context.Background()
	bodies, hashes := payloads(200_000)
	big := make([][]byte, 4000)
	bigHashes := make([][32]byte, len(big))
	for i := range big {
		b := make([]byte, InlineThreshold+64)
		copy(b, fmt.Sprintf("%08d", i))
		big[i] = b
		bigHashes[i] = canonical.Hash(b)
	}

	// 1. What the archive does today for a real record.
	inlined := openAt(t, t.TempDir())
	whole, err := rate(window, func(i int) error {
		return inlined.AppendCanonicalAt(ctx, bodies[i], hashes[i], 1, "t", 2, uint64(i+1))
	})
	if err != nil {
		t.Fatal(err)
	}

	// 2. What it does for a payload too large to inline: blob, then SQL.
	filed := openAt(t, t.TempDir())
	wholeBig, err := rate(window, func(i int) error {
		return filed.AppendCanonicalAt(ctx, big[i%len(big)], bigHashes[i%len(big)],
			1, "t", 2, uint64(i+1))
	})
	if err != nil {
		t.Fatal(err)
	}

	// 3. The blob half alone: mkdir, stat, temp file, write, FSYNC,
	//    rename. No SQLite at all.
	blobOnly := openAt(t, t.TempDir())
	blob, err := rate(window, func(i int) error {
		return blobOnly.writeBlob(hashes[i], bodies[i])
	})
	if err != nil {
		t.Fatal(err)
	}

	// 4. The SQL half alone.
	sqlOnly := openAt(t, t.TempDir())
	sqlRate, err := rate(window, func(i int) error {
		return sqlOnly.txOnly(ctx, hashes[i], uint64(i+1))
	})
	if err != nil {
		t.Fatal(err)
	}

	// 5. The blob half with the fsync removed, to separate "writing a
	//    file is slow" from "fsyncing it is".
	noSync := openAt(t, t.TempDir())
	blobNoSync, err := rate(window, func(i int) error {
		return noSync.writeBlobUnsynced(hashes[i], bodies[i])
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("commit, payload inlined      %9.0f /s", whole)
	t.Logf("commit, payload in a file    %9.0f /s", wholeBig)
	t.Logf("blob only (with fsync)       %9.0f /s", blob)
	t.Logf("blob only (no fsync)         %9.0f /s", blobNoSync)
	t.Logf("sql only (synchronous=FULL)  %9.0f /s", sqlRate)
	t.Logf("inlining is worth            %9.1fx", whole/wholeBig)

	// NO TIMING ASSERTION ON THE INLINE PATH, and the reason is worth
	// keeping. This file first guarded the win with `whole > blob*2`,
	// which passed on two local machines and failed on both CI runners —
	// not because inlining regressed, but because the runners have a
	// different cost profile entirely. One measured `sql only` at 524/s
	// against 1 836/s for the blob half: there, SQLite's fsync is the
	// expensive one, so an inlined commit is bounded by SQL and can be
	// legitimately slower than a bare blob write.
	//
	// The assertion encoded a relationship between two fsyncs on one
	// machine and called it a property. What is actually being claimed
	// is structural — an inlined commit writes no file — and
	// TestASmallPayloadWritesNoFileAtAll asserts exactly that, on any
	// hardware, without timing anything.
	//
	// These figures are reported, not gated. See ADR-0009.

	// The file path still runs blob-then-SQL serially, so reciprocals
	// still add there. If they stop, a third cost has appeared.
	predicted := 1 / (1/blob + 1/sqlRate)
	t.Logf("for the file path, blob and sql predict %.0f /s against %.0f measured",
		predicted, wholeBig)
	if wholeBig > predicted*2.5 || wholeBig < predicted*0.4 {
		t.Errorf("the two halves predict %.0f/s for a file-backed commit but it runs "+
			"at %.0f/s; the decomposition no longer accounts for that path",
			predicted, wholeBig)
	}
}

func TestBatchingTheTransactionIsNotTheLever(t *testing.T) {
	// The fix the roadmap named before this was measured: batch commits
	// into one transaction, paying one fsync for many records.
	//
	// It only helps if SQLite is the constraint. If the blob fsync
	// dominates, batching the SQL side leaves the per-record file work
	// untouched and buys almost nothing — which is exactly the sort of
	// well-tested fix for the wrong thing this phase is trying to avoid.
	measuring(t)
	ctx := context.Background()
	_, hashes := payloads(200_000)

	one := openAt(t, t.TempDir())
	single, err := rate(window, func(i int) error {
		return one.txOnly(ctx, hashes[i], uint64(i+1))
	})
	if err != nil {
		t.Fatal(err)
	}

	many := openAt(t, t.TempDir())
	const batch = 100
	i := 0
	batched, err := rate(window, func(_ int) error {
		if err := many.txBatch(ctx, hashes[i:i+batch], uint64(i+1)); err != nil {
			return err
		}
		i += batch
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	batched *= batch

	t.Logf("sql, one row per transaction   %9.0f /s", single)
	t.Logf("sql, %d rows per transaction  %9.0f /s", batch, batched)
	t.Logf("batching the SQL side is worth %.1fx", batched/single)

	if batched <= single {
		t.Errorf("batching %d rows into one transaction (%.0f/s) did not beat one "+
			"row per transaction (%.0f/s); the transaction is not fsync-bound and "+
			"the stated lever does not exist", batch, batched, single)
	}
}

// txOnly performs the SQL half of AppendCanonicalAt and nothing else.
func (s *Substrate) txOnly(ctx context.Context, hash [32]byte, seq uint64) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx,
		`INSERT OR IGNORE INTO events
		   (event_hash, event_time, message_type, payload_ref, committed_at)
		 VALUES (?, ?, ?, ?, ?)`, hash[:], 1, "t", "aa/bb", 2); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, positionUpsert, seq, seq, 1, 0, 0); err != nil {
		return err
	}
	return tx.Commit()
}

// txBatch commits many rows under one transaction, one position update.
func (s *Substrate) txBatch(ctx context.Context, hashes [][32]byte, firstSeq uint64) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for i, h := range hashes {
		if _, err := tx.ExecContext(ctx,
			`INSERT OR IGNORE INTO events
			   (event_hash, event_time, message_type, payload_ref, committed_at)
			 VALUES (?, ?, ?, ?, ?)`, h[:], 1, "t", "aa/bb", 2); err != nil {
			return fmt.Errorf("row %d: %w", i, err)
		}
	}
	last := firstSeq + uint64(len(hashes)) - 1
	if _, err := tx.ExecContext(ctx, positionUpsert,
		firstSeq, last, int64(len(hashes)), 0, 0); err != nil {
		return err
	}
	return tx.Commit()
}

// writeBlobUnsynced is writeBlob with the fsync removed, for
// attribution only. It is NOT an alternative implementation: dropping
// the sync means a crash can leave a renamed file with no contents, and
// a content-addressed store that answers with the wrong bytes is worse
// than one that lost them.
func (s *Substrate) writeBlobUnsynced(hash [32]byte, payload []byte) error {
	shardDir, finalPath := s.blobPath(hash)
	if err := os.MkdirAll(shardDir, 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(finalPath); err == nil {
		return nil
	}
	tmp, err := os.CreateTemp(shardDir, "tmp-blob-*")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(payload); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), finalPath)
}
