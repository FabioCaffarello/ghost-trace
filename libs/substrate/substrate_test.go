package substrate

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/FabioCaffarello/ghost-trace/libs/canonical"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/libs/genproto/events/v1"
)

func newTestSubstrate(t *testing.T) *Substrate {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	s, err := Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func newTestPayload(t *testing.T) ([]byte, [32]byte) {
	t.Helper()
	msg := &eventsv1.SessionStart{
		TenantId:  "t_test",
		SessionId: "s_test",
		StartedAt: 1716120000000000000,
		PagePath:  "/login",
	}
	payload, hash, err := canonical.MarshalAndHash(msg)
	if err != nil {
		t.Fatalf("MarshalAndHash: %v", err)
	}
	return payload, hash
}

func TestOpenCloseRoundtrip(t *testing.T) {
	s := newTestSubstrate(t)
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	// Idempotent close should not error.
	if err := s.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestAppendAndLookup(t *testing.T) {
	ctx := context.Background()
	s := newTestSubstrate(t)
	payload, hash := newTestPayload(t)

	hex := canonical.HashHex(hash)
	row := EventRow{
		EventHash:   hash,
		EventTime:   1716120000000000000,
		MessageType: "ghosttrace.events.v1.SessionStart",
		PayloadRef:  hex[:2] + "/" + hex[2:],
		CommittedAt: 1716120000000000001,
	}

	if err := s.Append(ctx, row, payload); err != nil {
		t.Fatalf("Append: %v", err)
	}

	got, err := s.LookupRow(ctx, hash)
	if err != nil {
		t.Fatalf("LookupRow: %v", err)
	}
	if got.EventHash != hash {
		t.Errorf("EventHash mismatch")
	}
	if got.EventTime != row.EventTime {
		t.Errorf("EventTime: got %d, want %d", got.EventTime, row.EventTime)
	}
	if got.MessageType != row.MessageType {
		t.Errorf("MessageType: got %q, want %q", got.MessageType, row.MessageType)
	}
}

func TestAppendIdempotent(t *testing.T) {
	ctx := context.Background()
	s := newTestSubstrate(t)
	payload, hash := newTestPayload(t)

	hex := canonical.HashHex(hash)
	row := EventRow{
		EventHash:   hash,
		EventTime:   1,
		MessageType: "ghosttrace.events.v1.SessionStart",
		PayloadRef:  hex[:2] + "/" + hex[2:],
		CommittedAt: 2,
	}

	for i := 0; i < 3; i++ {
		if err := s.Append(ctx, row, payload); err != nil {
			t.Fatalf("Append iteration %d: %v", i, err)
		}
	}

	n, err := s.Count(ctx)
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if n != 1 {
		t.Errorf("Count after 3 idempotent appends: got %d, want 1", n)
	}
}

func TestAppendHashMismatchRejected(t *testing.T) {
	ctx := context.Background()
	s := newTestSubstrate(t)
	payload, hash := newTestPayload(t)

	var wrong [32]byte
	copy(wrong[:], hash[:])
	wrong[0] ^= 0xff // flip a bit so hash no longer matches payload

	row := EventRow{EventHash: wrong, EventTime: 1, MessageType: "x", PayloadRef: "x/y", CommittedAt: 1}
	err := s.Append(ctx, row, payload)
	if err == nil {
		t.Fatal("expected ErrHashMismatch, got nil")
	}
	if !errors.Is(err, ErrHashMismatch) {
		t.Fatalf("expected ErrHashMismatch, got: %v", err)
	}
}

func TestReadBlobAfterAppend(t *testing.T) {
	ctx := context.Background()
	s := newTestSubstrate(t)
	payload, hash := newTestPayload(t)

	hex := canonical.HashHex(hash)
	row := EventRow{EventHash: hash, EventTime: 1, MessageType: "x", PayloadRef: hex[:2] + "/" + hex[2:], CommittedAt: 1}
	if err := s.Append(ctx, row, payload); err != nil {
		t.Fatalf("Append: %v", err)
	}

	got, err := s.ReadBlob(ctx, hash)
	if err != nil {
		t.Fatalf("ReadBlob: %v", err)
	}
	if string(got) != string(payload) {
		t.Errorf("blob bytes differ from payload")
	}
}

func TestReadBlobHashMismatchOnCorruption(t *testing.T) {
	// Both storage paths, because there are now two. A payload that fits
	// lives in the row and a larger one lives in a file, and the
	// content-addressing guarantee has to hold identically for each —
	// otherwise inlining quietly bought speed by dropping the check that
	// makes the store trustworthy.
	for _, tc := range []struct {
		name    string
		size    int
		corrupt func(t *testing.T, s *Substrate, hash [32]byte)
	}{
		{
			name: "inline",
			size: 64,
			corrupt: func(t *testing.T, s *Substrate, hash [32]byte) {
				t.Helper()
				if _, err := s.db.Exec(
					`UPDATE events SET payload = ? WHERE event_hash = ?`,
					[]byte("corrupted-content"), hash[:]); err != nil {
					t.Fatalf("corrupt inline payload: %v", err)
				}
			},
		},
		{
			name: "file",
			size: InlineThreshold + 1,
			corrupt: func(t *testing.T, s *Substrate, hash [32]byte) {
				t.Helper()
				_, finalPath := s.blobPath(hash)
				if err := os.WriteFile(finalPath, []byte("corrupted-content"), 0o644); err != nil {
					t.Fatalf("corrupt blob: %v", err)
				}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			s := newTestSubstrate(t)

			payload := make([]byte, tc.size)
			for i := range payload {
				payload[i] = byte('a' + i%26)
			}
			hash := canonical.Hash(payload)

			hex := canonical.HashHex(hash)
			row := EventRow{EventHash: hash, EventTime: 1, MessageType: "x",
				PayloadRef: hex[:2] + "/" + hex[2:], CommittedAt: 1}
			if err := s.Append(ctx, row, payload); err != nil {
				t.Fatalf("Append: %v", err)
			}

			got, err := s.ReadBlob(ctx, hash)
			if err != nil {
				t.Fatalf("ReadBlob before corruption: %v", err)
			}
			if !bytes.Equal(got, payload) {
				t.Fatal("ReadBlob returned different bytes than were written")
			}

			tc.corrupt(t, s, hash)

			if _, err := s.ReadBlob(ctx, hash); !errors.Is(err, ErrHashMismatch) {
				t.Fatalf("after corrupting the %s payload: %v, want ErrHashMismatch",
					tc.name, err)
			}
		})
	}
}

func TestLookupRowAbsent(t *testing.T) {
	ctx := context.Background()
	s := newTestSubstrate(t)

	var absent [32]byte
	for i := range absent {
		absent[i] = 0xab
	}
	_, err := s.LookupRow(ctx, absent)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows, got: %v", err)
	}
}

func TestWalkEventsOrderedByCommittedAt(t *testing.T) {
	ctx := context.Background()
	s := newTestSubstrate(t)

	// Insert three events with deliberately-non-sequential CommittedAt
	// to verify the walk orders them ascending.
	type seed struct {
		payload []byte
		commit  int64
	}
	seeds := []seed{
		{payload: []byte("alpha"), commit: 300},
		{payload: []byte("beta"), commit: 100},
		{payload: []byte("gamma"), commit: 200},
	}
	for _, s0 := range seeds {
		hash := canonical.Hash(s0.payload)
		hex := canonical.HashHex(hash)
		row := EventRow{EventHash: hash, EventTime: s0.commit, MessageType: "x", PayloadRef: hex[:2] + "/" + hex[2:], CommittedAt: s0.commit}
		if err := s.Append(ctx, row, s0.payload); err != nil {
			t.Fatalf("Append %s: %v", string(s0.payload), err)
		}
	}

	var commits []int64
	if err := s.WalkEvents(ctx, func(row EventRow) error {
		commits = append(commits, row.CommittedAt)
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}

	want := []int64{100, 200, 300}
	if len(commits) != len(want) {
		t.Fatalf("got %d rows, want %d", len(commits), len(want))
	}
	for i, c := range commits {
		if c != want[i] {
			t.Errorf("row %d: got commit %d, want %d", i, c, want[i])
		}
	}
}

func TestWalkEventsStopsOnError(t *testing.T) {
	ctx := context.Background()
	s := newTestSubstrate(t)
	for i := 0; i < 3; i++ {
		payload := []byte{byte('a' + i)}
		hash := canonical.Hash(payload)
		hex := canonical.HashHex(hash)
		row := EventRow{EventHash: hash, EventTime: int64(i), MessageType: "x", PayloadRef: hex[:2] + "/" + hex[2:], CommittedAt: int64(i)}
		if err := s.Append(ctx, row, payload); err != nil {
			t.Fatal(err)
		}
	}

	sentinel := errors.New("stop here")
	var seen int
	err := s.WalkEvents(ctx, func(row EventRow) error {
		seen++
		if seen == 2 {
			return sentinel
		}
		return nil
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got: %v", err)
	}
	if seen != 2 {
		t.Errorf("walked %d rows after stop, want 2", seen)
	}
}
