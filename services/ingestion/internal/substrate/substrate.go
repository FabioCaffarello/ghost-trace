// Package substrate implements the inception-phase Tier 1 primary event
// log per decision-log §0027 (SQLite + content-addressed blob-store on
// local filesystem). Reads concurrent-without-restriction; writes
// serialize through a single Append entry point per
// docs/architecture/concurrency-pattern.md §Substrate-Writer Serialization.
package substrate

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "modernc.org/sqlite" // pure-Go driver per decision-log §0027 Proposal item 4.

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
)

// eventsSchemaDDL — events-table definition per decision-log §0027
// Proposal item 1. WITHOUT ROWID suppresses the implicit SQLite ROWID
// per AP "Using SQLite ROWID as event identity instead of event_hash".
const eventsSchemaDDL = `
CREATE TABLE IF NOT EXISTS events (
    event_hash    BLOB    PRIMARY KEY,
    event_time    INTEGER NOT NULL,
    message_type  TEXT    NOT NULL,
    payload_ref   TEXT    NOT NULL,
    committed_at  INTEGER NOT NULL
) WITHOUT ROWID;
`

// canonicalPragmas — applied at Open per decision-log §0027 Proposal
// item 1 + §0029 modification 2. journal_mode=WAL for concurrent-reader
// + single-writer semantics; synchronous=FULL for §2.1 durability
// guarantee under power loss.
var canonicalPragmas = []string{
	"PRAGMA journal_mode=WAL",
	"PRAGMA synchronous=FULL",
}

// ErrHashMismatch indicates a §2.1 immutability violation: the recomputed
// content hash does not match the stored hash. Per decision-log §0027
// AP4 (§2.1-inheritance restatement) + AP5 (operational discipline).
var ErrHashMismatch = errors.New("substrate: hash-mismatch — §2.1 violation")

// ErrBlobCollision indicates a write attempt whose hash matches an
// existing blob but whose payload bytes differ. Per decision-log §0027
// AP6 — apparent-duplicate-write byte-equality verification.
var ErrBlobCollision = errors.New("substrate: blob byte-equality violation on apparent-duplicate write")

// Substrate is the inception-phase primary event log + blob-store.
//
// Concurrent reads are safe. Writes serialize through the application-
// layer writeMu per docs/architecture/concurrency-pattern.md §Substrate-
// Writer Serialization (single Append entry point).
type Substrate struct {
	db      *sql.DB
	blobDir string

	writeMu sync.Mutex
}

// Open initializes a Substrate at dbPath + blobDir. Applies canonical
// PRAGMA configuration; creates the events table if missing.
func Open(ctx context.Context, dbPath, blobDir string) (*Substrate, error) {
	if err := os.MkdirAll(blobDir, 0o755); err != nil {
		return nil, fmt.Errorf("substrate.Open: mkdir blob dir: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("substrate.Open: sql.Open: %w", err)
	}

	for _, pragma := range canonicalPragmas {
		if _, err := db.ExecContext(ctx, pragma); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("substrate.Open: %s: %w", pragma, err)
		}
	}

	if _, err := db.ExecContext(ctx, eventsSchemaDDL); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("substrate.Open: create events table: %w", err)
	}

	return &Substrate{db: db, blobDir: blobDir}, nil
}

// BlobDir returns the configured blob-store directory. Exposed for
// operational tooling (e.g. the verify CLI per §0039) and tests that
// need to inspect on-disk state. Service code SHOULD NOT manipulate
// the blob-store directly; use ReadBlob / Append / AppendPair.
func (s *Substrate) BlobDir() string { return s.blobDir }

// Close releases the underlying database connection. Idempotent.
func (s *Substrate) Close() error {
	if s.db == nil {
		return nil
	}
	err := s.db.Close()
	s.db = nil
	return err
}

// EventRow is the typed row content of the events table.
type EventRow struct {
	EventHash   [32]byte
	EventTime   int64
	MessageType string
	PayloadRef  string
	CommittedAt int64
}

// Append commits a single event atomically. Performs:
//  1. Verify the caller-supplied row.EventHash matches Hash(payload).
//  2. Write payload to the blob-store at content-hash-derived path
//     (write-once; idempotent on identical content; rejects on
//     byte-inequality with existing blob).
//  3. INSERT OR IGNORE into events table (idempotent on PRIMARY KEY
//     conflict).
//
// Serializes via writeMu per concurrency-pattern §Substrate-Writer
// Serialization.
func (s *Substrate) Append(ctx context.Context, row EventRow, payload []byte) error {
	computed := canonical.Hash(payload)
	if subtle.ConstantTimeCompare(row.EventHash[:], computed[:]) != 1 {
		return fmt.Errorf("substrate.Append: %w (row.EventHash != Hash(payload))", ErrHashMismatch)
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	if err := s.writeBlob(row.EventHash, payload); err != nil {
		return fmt.Errorf("substrate.Append: blob write: %w", err)
	}

	if _, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO events
		   (event_hash, event_time, message_type, payload_ref, committed_at)
		 VALUES (?, ?, ?, ?, ?)`,
		row.EventHash[:], row.EventTime, row.MessageType, row.PayloadRef, row.CommittedAt,
	); err != nil {
		return fmt.Errorf("substrate.Append: insert: %w", err)
	}
	return nil
}

// blobPath returns the on-disk path for a content-hash. Two-character
// prefix shard per decision-log §0027 Proposal item 2.
func (s *Substrate) blobPath(hash [32]byte) (shardDir, finalPath string) {
	hex := canonical.HashHex(hash)
	shardDir = filepath.Join(s.blobDir, hex[:2])
	finalPath = filepath.Join(shardDir, hex[2:])
	return shardDir, finalPath
}

// writeBlob writes payload to the blob-store under hash. Idempotent on
// identical content; ErrBlobCollision on byte-inequality with existing
// blob (per decision-log §0027 AP6).
//
// POSIX-only: uses write-temp-then-rename atomicity. Decision-log §0027
// Open Questions surfaces the POSIX-only inception-phase constraint;
// reversal condition R-store-4 captures the Windows-substrate trigger.
func (s *Substrate) writeBlob(hash [32]byte, payload []byte) error {
	shardDir, finalPath := s.blobPath(hash)

	if err := os.MkdirAll(shardDir, 0o755); err != nil {
		return fmt.Errorf("mkdir shard %s: %w", shardDir, err)
	}

	if existing, err := os.ReadFile(finalPath); err == nil {
		if subtle.ConstantTimeCompare(existing, payload) != 1 {
			return fmt.Errorf("at %s: %w", finalPath, ErrBlobCollision)
		}
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat blob %s: %w", finalPath, err)
	}

	tmp, err := os.CreateTemp(shardDir, "tmp-blob-*")
	if err != nil {
		return fmt.Errorf("create temp blob: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpPath) }

	if _, err := tmp.Write(payload); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("write temp blob: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("sync temp blob: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temp blob: %w", err)
	}
	if err := os.Rename(tmpPath, finalPath); err != nil {
		cleanup()
		return fmt.Errorf("rename temp blob: %w", err)
	}
	return nil
}

// ReadBlob reads the blob for hash. Recomputes the content hash on read
// and returns ErrHashMismatch on §2.1 violation per the canonical-
// serialization-contract anti-pattern "hash-verification omitted from
// blob-read path" and decision-log §0027 AP5.
func (s *Substrate) ReadBlob(_ context.Context, hash [32]byte) ([]byte, error) {
	_, finalPath := s.blobPath(hash)

	payload, err := os.ReadFile(finalPath)
	if err != nil {
		return nil, fmt.Errorf("substrate.ReadBlob: read %s: %w", finalPath, err)
	}

	recomputed := canonical.Hash(payload)
	if subtle.ConstantTimeCompare(hash[:], recomputed[:]) != 1 {
		return nil, fmt.Errorf("substrate.ReadBlob at %s: %w", finalPath, ErrHashMismatch)
	}
	return payload, nil
}

// LookupRow reads the events-table row for hash. Returns sql.ErrNoRows
// if absent.
func (s *Substrate) LookupRow(ctx context.Context, hash [32]byte) (EventRow, error) {
	var row EventRow
	var hashBytes []byte
	err := s.db.QueryRowContext(ctx,
		`SELECT event_hash, event_time, message_type, payload_ref, committed_at
		   FROM events
		  WHERE event_hash = ?`,
		hash[:],
	).Scan(&hashBytes, &row.EventTime, &row.MessageType, &row.PayloadRef, &row.CommittedAt)
	if err != nil {
		return EventRow{}, err
	}
	copy(row.EventHash[:], hashBytes)
	return row, nil
}

// Count returns the number of committed events. Useful for tests +
// operational diagnostics.
func (s *Substrate) Count(ctx context.Context) (int64, error) {
	var n int64
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM events`).Scan(&n)
	return n, err
}

// WalkEvents iterates over every events-table row in commit order
// (committed_at ascending). For each row, fn is called with the row's
// content. If fn returns a non-nil error, iteration stops and the
// error is propagated to the caller; the cursor is then closed.
//
// Used by the verify CLI per docs/charter/decision-log.md §0039 +
// §0033 §Restoration Procedure. The walk is read-only; the writeMu
// is NOT acquired (WAL mode permits concurrent readers per §0027).
func (s *Substrate) WalkEvents(ctx context.Context, fn func(EventRow) error) error {
	rows, err := s.db.QueryContext(ctx,
		`SELECT event_hash, event_time, message_type, payload_ref, committed_at
		   FROM events
		  ORDER BY committed_at ASC, event_hash ASC`)
	if err != nil {
		return fmt.Errorf("substrate.WalkEvents: query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var row EventRow
		var hashBytes []byte
		if err := rows.Scan(&hashBytes, &row.EventTime, &row.MessageType, &row.PayloadRef, &row.CommittedAt); err != nil {
			return fmt.Errorf("substrate.WalkEvents: scan: %w", err)
		}
		copy(row.EventHash[:], hashBytes)
		if err := fn(row); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("substrate.WalkEvents: rows iteration: %w", err)
	}
	return nil
}

// AppendPair commits two events atomically: a primary observation and
// an enrichment record paired by reference. Used by the ingestion path
// to commit a primary observation (e.g. DeclaredSession) alongside its
// paired IngestionEvent per docs/charter/decision-log.md §0038.
//
// Atomicity discipline:
//  1. Both hashes are verified against their payloads (hash mismatch
//     rejects the call without writing anything).
//  2. Both blobs are written to the blob-store (idempotent on
//     content-hash; safe outside the SQL transaction — orphan blobs
//     are harmless per §0027 Proposal item 5 + §0033 §Restoration).
//  3. Both events-table rows are inserted inside a single SQL
//     transaction. Either both rows commit or neither (SQLite WAL +
//     synchronous=FULL provides the durability guarantee).
//
// Either both events become visible in subsequent reads or neither.
// The pairing is by reference (enrichment carries a hash to the
// primary); recovery from orphan blobs is operator concern per §0033.
//
// Serializes via writeMu per concurrency-pattern §Substrate-Writer
// Serialization (same single-writer semantics as Append).
func (s *Substrate) AppendPair(ctx context.Context,
	primaryRow EventRow, primaryPayload []byte,
	enrichmentRow EventRow, enrichmentPayload []byte,
) error {
	if want := canonical.Hash(primaryPayload); subtle.ConstantTimeCompare(primaryRow.EventHash[:], want[:]) != 1 {
		return fmt.Errorf("substrate.AppendPair: %w (primary row.EventHash != Hash(primaryPayload))", ErrHashMismatch)
	}
	if want := canonical.Hash(enrichmentPayload); subtle.ConstantTimeCompare(enrichmentRow.EventHash[:], want[:]) != 1 {
		return fmt.Errorf("substrate.AppendPair: %w (enrichment row.EventHash != Hash(enrichmentPayload))", ErrHashMismatch)
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	if err := s.writeBlob(primaryRow.EventHash, primaryPayload); err != nil {
		return fmt.Errorf("substrate.AppendPair: primary blob write: %w", err)
	}
	if err := s.writeBlob(enrichmentRow.EventHash, enrichmentPayload); err != nil {
		return fmt.Errorf("substrate.AppendPair: enrichment blob write: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("substrate.AppendPair: begin tx: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT OR IGNORE INTO events
		   (event_hash, event_time, message_type, payload_ref, committed_at)
		 VALUES (?, ?, ?, ?, ?)`,
		primaryRow.EventHash[:], primaryRow.EventTime, primaryRow.MessageType, primaryRow.PayloadRef, primaryRow.CommittedAt,
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("substrate.AppendPair: insert primary: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT OR IGNORE INTO events
		   (event_hash, event_time, message_type, payload_ref, committed_at)
		 VALUES (?, ?, ?, ?, ?)`,
		enrichmentRow.EventHash[:], enrichmentRow.EventTime, enrichmentRow.MessageType, enrichmentRow.PayloadRef, enrichmentRow.CommittedAt,
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("substrate.AppendPair: insert enrichment: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("substrate.AppendPair: commit: %w", err)
	}
	return nil
}
