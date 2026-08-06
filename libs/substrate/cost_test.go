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

func openAt(t *testing.T, dir string, pragmas ...string) *Substrate {
	t.Helper()
	dbPath := filepath.Join(dir, "events.db")
	blobDir := filepath.Join(dir, "blobs")
	if err := os.MkdirAll(blobDir, 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	applied := canonicalPragmas
	if len(pragmas) > 0 {
		applied = pragmas
	}
	for _, p := range applied {
		if _, err := db.ExecContext(ctx, p); err != nil {
			t.Fatalf("%s: %v", p, err)
		}
	}
	for _, ddl := range []string{eventsSchemaDDL, positionSchemaDDL} {
		if _, err := db.ExecContext(ctx, ddl); err != nil {
			t.Fatal(err)
		}
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

func TestWhereACommitSpendsItsTime(t *testing.T) {
	// The decomposition. Each variant removes exactly one thing, so the
	// difference between two rows is that thing's cost and not a guess
	// about it.
	if testing.Short() {
		t.Skip("timing measurement")
	}
	ctx := context.Background()
	bodies, hashes := payloads(200_000)

	// 1. What the archive actually does today.
	full := openAt(t, t.TempDir())
	whole, err := rate(window, func(i int) error {
		return full.AppendCanonicalAt(ctx, bodies[i], hashes[i], 1, "t", 2, uint64(i+1))
	})
	if err != nil {
		t.Fatal(err)
	}

	// 2. The blob half alone: mkdir, stat, temp file, write, FSYNC,
	//    rename. No SQLite at all.
	blobOnly := openAt(t, t.TempDir())
	blob, err := rate(window, func(i int) error {
		return blobOnly.writeBlob(hashes[i], bodies[i])
	})
	if err != nil {
		t.Fatal(err)
	}

	// 3. The SQL half alone: the same transaction, no blob.
	sqlOnly := openAt(t, t.TempDir())
	sqlRate, err := rate(window, func(i int) error {
		return sqlOnly.txOnly(ctx, hashes[i], uint64(i+1))
	})
	if err != nil {
		t.Fatal(err)
	}

	// 4. The blob half with the fsync removed. This is the measurement
	//    that separates "writing a file is slow" from "fsyncing it is".
	noSync := openAt(t, t.TempDir())
	blobNoSync, err := rate(window, func(i int) error {
		return noSync.writeBlobUnsynced(hashes[i], bodies[i])
	})
	if err != nil {
		t.Fatal(err)
	}

	// 5. The SQL half at synchronous=NORMAL, which is the pragma the
	//    hypothesis pointed at.
	normal := openAt(t, t.TempDir(),
		"PRAGMA journal_mode=WAL", "PRAGMA synchronous=NORMAL")
	sqlNormal, err := rate(window, func(i int) error {
		return normal.txOnly(ctx, hashes[i], uint64(i+1))
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("full AppendCanonicalAt      %9.0f /s", whole)
	t.Logf("blob only (with fsync)      %9.0f /s", blob)
	t.Logf("blob only (no fsync)        %9.0f /s", blobNoSync)
	t.Logf("sql only (synchronous=FULL) %9.0f /s", sqlRate)
	t.Logf("sql only (synchronous=NORMAL)%8.0f /s", sqlNormal)

	// Serial costs add as reciprocals: the halves run one after the
	// other inside one writeMu, so 1/full should be about 1/blob +
	// 1/sql. Checking it keeps the decomposition honest — if the parts
	// do not account for the whole, something else is being paid for.
	predicted := 1 / (1/blob + 1/sqlRate)
	t.Logf("blob and sql together predict %7.0f /s against %.0f measured "+
		"(%.0f%% accounted)", predicted, whole, 100*whole/predicted)

	// Wide on purpose: shared CI hardware makes a tight band a flake
	// rather than a gate. This still fails if a third cost appears, or
	// if the two halves stop being serial.
	if whole > predicted*2.5 || whole < predicted*0.4 {
		t.Errorf("the two halves predict %.0f/s but the whole runs at %.0f/s; the "+
			"decomposition does not account for the commit and the attribution "+
			"below cannot be trusted", predicted, whole)
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
	if testing.Short() {
		t.Skip("timing measurement")
	}
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
	if _, err := tx.ExecContext(ctx, positionUpsert, seq, seq, 1, 0); err != nil {
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
		firstSeq, last, int64(len(hashes)), 0); err != nil {
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
