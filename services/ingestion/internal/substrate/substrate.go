// Package substrate implements the primary event log: SQLite for the
// index, a content-addressed blob store on the local filesystem for the
// payloads.
//
// Reads are concurrent without restriction; writes serialize through a
// single Append entry point, which is what makes the append-only
// guarantee cheap to state and cheap to check — there is exactly one
// place where bytes become durable.
package substrate

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	// Pure Go, deliberately: it is what lets CGO_ENABLED=0 produce a
	// fully static binary and the runtime image be distroless.
	_ "modernc.org/sqlite"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
)

// eventsSchemaDDL — the events table. WITHOUT ROWID suppresses the implicit SQLite ROWID
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

// canonicalPragmas — applied at Open. journal_mode=WAL for concurrent-reader
// + single-writer semantics; synchronous=FULL for §2.1 durability
// guarantee under power loss.
var canonicalPragmas = []string{
	"PRAGMA journal_mode=WAL",
	"PRAGMA synchronous=FULL",
}

// ErrHashMismatch indicates a §2.1 immutability violation: the recomputed
// content hash does not match the stored hash. Per
// AP4 (§2.1-inheritance restatement) + AP5 (operational discipline).
var ErrHashMismatch = errors.New("substrate: hash-mismatch — §2.1 violation")

// ErrBlobCollision indicates a write attempt whose hash matches an
// existing blob but whose payload bytes differ. Per
// AP6 — apparent-duplicate-write byte-equality verification.
var ErrBlobCollision = errors.New("substrate: blob byte-equality violation on apparent-duplicate write")

// Substrate is the inception-phase primary event log + blob-store.
//
// Concurrent reads are safe. Writes serialize through the
// application-layer writeMu — a single Append entry point, so the
// append-only guarantee has one enforcement site rather than several.
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
// prefix shard
func (s *Substrate) blobPath(hash [32]byte) (shardDir, finalPath string) {
	hex := canonical.HashHex(hash)
	shardDir = filepath.Join(s.blobDir, hex[:2])
	finalPath = filepath.Join(shardDir, hex[2:])
	return shardDir, finalPath
}

// writeBlob writes payload to the blob-store under hash. Idempotent on
// identical content; ErrBlobCollision on byte-inequality with existing
// blob (
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
// blob-read path" and AP5.
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

// WalkBlobs iterates over every file in the blob-store that matches the
// blob-path convention (<2-char-prefix>/<62-char-suffix>, where the
// concatenated 64 chars decode to a 32-byte hash). For each match, fn
// is called with the decoded hash + the absolute filesystem path.
// Files that do not match the convention (temp files, accidentally-
// placed artifacts) are silently skipped.
//
// Written for orphan-blob detection: a blob with no event referencing
// it. Read-only walk; no writeMu.
//
// Order is filesystem-iteration order (operating-system dependent;
// typically sorted by shard then filename on POSIX). Callers MUST NOT
// rely on a specific traversal order.
func (s *Substrate) WalkBlobs(ctx context.Context, fn func(hash [32]byte, path string) error) error {
	return filepath.WalkDir(s.blobDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		// Honor context cancellation between entries.
		if err := ctx.Err(); err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		// Skip leftover tmp files (write-temp-then-rename leaves none
		// in steady state, but a crashed write may orphan one).
		if strings.HasPrefix(d.Name(), "tmp-blob-") {
			return nil
		}
		rel, err := filepath.Rel(s.blobDir, path)
		if err != nil {
			return fmt.Errorf("substrate.WalkBlobs: rel %s: %w", path, err)
		}
		cleaned := filepath.ToSlash(rel)
		// Expected form: "<2-hex>/<62-hex>".
		parts := strings.SplitN(cleaned, "/", 2)
		if len(parts) != 2 || len(parts[0]) != 2 || len(parts[1]) != 62 {
			return nil // not a blob-store file; skip silently
		}
		hexStr := parts[0] + parts[1]
		var hash [32]byte
		if _, err := hex.Decode(hash[:], []byte(hexStr)); err != nil {
			return nil // unparseable hex; skip
		}
		return fn(hash, path)
	})
}

// WalkEvents iterates over every events-table row in commit order
// (committed_at ascending). For each row, fn is called with the row's
// content. If fn returns a non-nil error, iteration stops and the
// error is propagated to the caller; the cursor is then closed.
//
// Written for integrity verification and restoration. The walk is
// read-only and does NOT acquire writeMu: WAL mode permits concurrent
// readers, so a verification pass cannot stall ingestion.
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
// paired IngestionEvent.
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
