package substrate

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/FabioCaffarello/ghost-trace/libs/canonical"
)

func bodyOf(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte('a' + i%26)
	}
	return b
}

func TestASmallPayloadWritesNoFileAtAll(t *testing.T) {
	// The whole point. PR-4.4 measured the blob write — mkdir, stat,
	// temp file, FSYNC, rename — as the larger of the two fsyncs a
	// commit pays. For a payload that fits, this removes it rather than
	// making it cheaper.
	ctx := context.Background()
	s := newTestSubstrate(t)

	payload := bodyOf(161) // the size every real record measured at
	hash := canonical.Hash(payload)
	if err := s.AppendCanonicalAt(ctx, payload, hash, 1, "t", 2, 1); err != nil {
		t.Fatalf("append: %v", err)
	}

	var files int
	_ = filepath.WalkDir(s.BlobDir(), func(_ string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			files++
		}
		return nil
	})
	if files != 0 {
		t.Errorf("%d file(s) written for an inlined payload; the fsync this change "+
			"exists to remove is still being paid", files)
	}

	got, err := s.ReadBlob(ctx, hash)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Error("the inlined payload did not come back byte for byte")
	}
}

func TestAPayloadOverTheThresholdStillGetsAFile(t *testing.T) {
	// The fallback has to actually work, or the threshold is a cliff
	// that loses records instead of storing them differently.
	ctx := context.Background()
	s := newTestSubstrate(t)

	payload := bodyOf(InlineThreshold + 1)
	hash := canonical.Hash(payload)
	if err := s.AppendCanonicalAt(ctx, payload, hash, 1, "t", 2, 1); err != nil {
		t.Fatalf("append: %v", err)
	}

	_, finalPath := s.blobPath(hash)
	if _, err := os.Stat(finalPath); err != nil {
		t.Fatalf("no file for a payload above the threshold: %v", err)
	}

	var inline []byte
	if err := s.db.QueryRow(`SELECT payload FROM events WHERE event_hash = ?`,
		hash[:]).Scan(&inline); err != nil {
		t.Fatalf("read row: %v", err)
	}
	if inline != nil {
		t.Errorf("a payload above the threshold was ALSO stored in the row; the " +
			"bytes are now in two places and can disagree")
	}

	got, err := s.ReadBlob(ctx, hash)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Error("the file-backed payload did not come back byte for byte")
	}
}

func TestExactlyAtTheThresholdIsInlined(t *testing.T) {
	// Off-by-one at a storage boundary is the kind of thing that works
	// for a year and then does not.
	ctx := context.Background()
	s := newTestSubstrate(t)

	payload := bodyOf(InlineThreshold)
	hash := canonical.Hash(payload)
	if err := s.AppendCanonicalAt(ctx, payload, hash, 1, "t", 2, 1); err != nil {
		t.Fatalf("append: %v", err)
	}
	if _, finalPath := s.blobPath(hash); fileExists(finalPath) {
		t.Error("a payload exactly at the threshold was written to a file; the " +
			"boundary is documented as inclusive")
	}
	if _, err := s.ReadBlob(ctx, hash); err != nil {
		t.Fatalf("read back: %v", err)
	}
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func TestRecordsWrittenBeforeInliningStayReadable(t *testing.T) {
	// THE compatibility claim, and the one worth a test rather than a
	// sentence. An archive is append-only and long-lived: the records
	// already on disk when this change ships were written as files, and
	// a reader that only looked at the row would report them missing —
	// which for a content-addressed archive is indistinguishable from
	// having lost them.
	ctx := context.Background()
	dir := t.TempDir()
	s := newTestSubstrateAt(t, dir)

	payload := bodyOf(161)
	hash := canonical.Hash(payload)

	// Written the OLD way: file on disk, row with no inline payload.
	if err := s.writeBlob(hash, payload); err != nil {
		t.Fatalf("write blob: %v", err)
	}
	hex := canonical.HashHex(hash)
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO events (event_hash, event_time, message_type, payload_ref, committed_at)
		 VALUES (?, ?, ?, ?, ?)`,
		hash[:], 1, "t", hex[:2]+"/"+hex[2:], 2); err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}

	got, err := s.ReadBlob(ctx, hash)
	if err != nil {
		t.Fatalf("a record written before inlining is unreadable: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Error("a legacy record came back with different bytes")
	}
}

func TestOpeningAnOlderArchiveAddsTheColumnOnce(t *testing.T) {
	// The migration runs on every Open, so it has to be idempotent —
	// and it has to work on a database created before the column
	// existed, which is every archive currently deployed.
	ctx := context.Background()
	dir := t.TempDir()

	// A database with the pre-inline schema, and one record in it.
	pre := newTestSubstrateAt(t, dir)
	if _, err := pre.db.ExecContext(ctx, `ALTER TABLE events DROP COLUMN payload`); err != nil {
		t.Fatalf("simulate the older schema: %v", err)
	}
	payload := bodyOf(161)
	hash := canonical.Hash(payload)
	if err := pre.writeBlob(hash, payload); err != nil {
		t.Fatal(err)
	}
	hex := canonical.HashHex(hash)
	if _, err := pre.db.ExecContext(ctx,
		`INSERT INTO events (event_hash, event_time, message_type, payload_ref, committed_at)
		 VALUES (?, ?, ?, ?, ?)`, hash[:], 1, "t", hex[:2]+"/"+hex[2:], 2); err != nil {
		t.Fatal(err)
	}
	if err := pre.Close(); err != nil {
		t.Fatal(err)
	}

	// Opened twice by the new binary: the column appears, and appearing
	// again is not an error.
	for i := range 2 {
		s, err := Open(ctx, filepath.Join(dir, "events.db"), filepath.Join(dir, "blobs"))
		if err != nil {
			t.Fatalf("open %d: %v", i+1, err)
		}
		got, err := s.ReadBlob(ctx, hash)
		if err != nil {
			t.Fatalf("open %d: the pre-existing record is unreadable: %v", i+1, err)
		}
		if !bytes.Equal(got, payload) {
			t.Fatalf("open %d: the pre-existing record came back different", i+1)
		}
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestInliningDoesNotDisturbTheDurablePosition(t *testing.T) {
	// The position and the payload are written in one transaction. A
	// change to what that transaction carries must not change what it
	// accounts for.
	ctx := context.Background()
	s := newTestSubstrate(t)

	for i := range 5 {
		payload := bodyOf(100 + i)
		if err := s.AppendCanonicalAt(ctx, payload, canonical.Hash(payload),
			1, "t", 2, uint64(i+1)); err != nil {
			t.Fatalf("append %d: %v", i, err)
		}
	}
	big := bodyOf(InlineThreshold + 7)
	if err := s.AppendCanonicalAt(ctx, big, canonical.Hash(big), 1, "t", 2, 6); err != nil {
		t.Fatalf("append big: %v", err)
	}

	p, ok, err := s.Position(ctx)
	if err != nil || !ok {
		t.Fatalf("position: %v (ok=%v)", err, ok)
	}
	if p.Committed != 6 || p.HighestSeq != 6 || p.Unaccounted() != 0 {
		t.Errorf("position = %+v; want 6 commits through sequence 6 with nothing "+
			"unaccounted, whichever way the payload was stored", p)
	}
}

func newTestSubstrateAt(t *testing.T, dir string) *Substrate {
	t.Helper()
	s, err := Open(context.Background(),
		filepath.Join(dir, "events.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}
