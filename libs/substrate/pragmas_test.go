package substrate

// White-box like inline_test.go: the pool being probed is s.db, which
// only this package can see.

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPragmasApplyToTheWholePool holds N pooled connections open at
// once and asserts the canonical pragmas on every one of them.
//
// This is the guard for a bug that never failed anything: pragmas
// applied with ExecContext on the *sql.DB configure the ONE connection
// the pool hands that Exec — probed at 8 connections the split was
// {FULL:1, NORMAL:7} — and the §2.1 durability guarantee held only
// because the driver's compiled default happens to be FULL already.
// journal_mode hides the same mistake forever, because WAL persists in
// the database file; synchronous is per-connection and does not.
func TestPragmasApplyToTheWholePool(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := Open(ctx, filepath.Join(dir, "events.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = s.Close() }()

	const conns = 8
	s.db.SetMaxOpenConns(conns)

	held := make([]*sql.Conn, 0, conns)
	defer func() {
		for _, c := range held {
			_ = c.Close()
		}
	}()

	full := 0
	for i := 0; i < conns; i++ {
		// Holding every prior connection forces db.Conn to dial a NEW
		// one; releasing them between iterations would probe the same
		// connection eight times and prove nothing.
		c, err := s.db.Conn(ctx)
		if err != nil {
			t.Fatalf("conn %d: %v", i, err)
		}
		held = append(held, c)

		var sync int
		if err := c.QueryRowContext(ctx, "PRAGMA synchronous").Scan(&sync); err != nil {
			t.Fatalf("conn %d: PRAGMA synchronous: %v", i, err)
		}
		if sync == 2 { // 2 == FULL
			full++
		}

		var journal string
		if err := c.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&journal); err != nil {
			t.Fatalf("conn %d: PRAGMA journal_mode: %v", i, err)
		}
		if !strings.EqualFold(journal, "wal") {
			t.Errorf("conn %d: journal_mode = %q, want wal", i, journal)
		}
	}

	if full != conns {
		t.Fatalf("synchronous=FULL on %d of %d pooled connections. The pragma "+
			"must ride the DSN so the driver applies it to every connection; "+
			"an Exec on the pool configures exactly one", full, conns)
	}

	// The outcome assertion above has a blind spot the original bug
	// lived in: the driver's compiled DEFAULT is FULL, so every
	// connection reads FULL even when nothing applied anything. Two
	// more assertions close it. First, the mechanism itself is pinned —
	// the pragmas must be in the DSN, because that is the only form the
	// driver applies per-connection.
	for _, want := range []string{"_pragma=synchronous(FULL)", "_pragma=journal_mode(WAL)"} {
		if !strings.Contains(dsn("x"), want) {
			t.Fatalf("the canonical DSN no longer carries %s: %q — pragmas "+
				"applied any other way reach one pooled connection, and the "+
				"driver default masks the loss", want, dsn("x"))
		}
	}

	// Second, the probe is calibrated: a pool whose DSN says NORMAL
	// must read NORMAL on every connection. If this fails, the probe
	// above was reading defaults, not pragmas, and proves nothing.
	caldir := t.TempDir()
	cal, err := sql.Open("sqlite",
		"file:"+filepath.Join(caldir, "cal.db")+"?_pragma=synchronous(NORMAL)")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cal.Close() }()
	var sync int
	if err := cal.QueryRowContext(ctx, "PRAGMA synchronous").Scan(&sync); err != nil {
		t.Fatal(err)
	}
	if sync != 1 { // 1 == NORMAL
		t.Fatalf("calibration pool reads synchronous=%d, want 1 (NORMAL) — the "+
			"probe cannot distinguish a DSN pragma from the driver default, so "+
			"the assertions above are vacuous", sync)
	}
}

// TestOpenOnAPathWithASpace exists because this repository itself lives
// under a directory with a space in its name, and a file: DSN is the
// kind of place where that stops being true quietly.
func TestOpenOnAPathWithASpace(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "with space")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	s, err := Open(context.Background(), filepath.Join(dir, "events.db"),
		filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("Open on a path with a space: %v", err)
	}
	_ = s.Close()
}
