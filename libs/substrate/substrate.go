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

	"github.com/FabioCaffarello/ghost-trace/libs/canonical"
)

// eventsSchemaDDL — the events table. WITHOUT ROWID suppresses the implicit SQLite ROWID
// per AP "Using SQLite ROWID as event identity instead of event_hash".
const eventsSchemaDDL = `
CREATE TABLE IF NOT EXISTS events (
    event_hash    BLOB    PRIMARY KEY,
    event_time    INTEGER NOT NULL,
    message_type  TEXT    NOT NULL,
    payload_ref   TEXT    NOT NULL,
    committed_at  INTEGER NOT NULL,
    payload       BLOB
) WITHOUT ROWID;
`

// positionSchemaDDL — where the archive is in the stream, durably.
//
// WHY THIS IS IN THE SUBSTRATE AND NOT A COUNTER. Process-local counters
// reset on restart, so "committed" answers what one process did rather
// than what the archive holds; restarting mid-backlog was observed to
// take a committed counter from 5 to 10 while fifteen records had in
// fact arrived. And the broker cannot be asked either: it advances a
// consumer's ack floor when records are removed from under it, so the
// bookkeeping that would reveal loss is destroyed by the loss.
//
// A durable position is the only remaining place to stand. It lives in
// the same database as the events and is written in the same
// transaction, so "the archive committed record N" and "the archive
// reached sequence N" cannot disagree across a crash.
//
// One row, enforced by the CHECK. There is one stream and one archive.
const positionSchemaDDL = `
CREATE TABLE IF NOT EXISTS stream_position (
    id           INTEGER PRIMARY KEY CHECK (id = 1),
    first_seq    INTEGER NOT NULL,
    highest_seq  INTEGER NOT NULL,
    committed    INTEGER NOT NULL,
    rejected     INTEGER NOT NULL,
    duplicates   INTEGER NOT NULL DEFAULT 0
);
`

// InlineThreshold is the payload size at or below which the bytes live
// in the events row instead of in a file.
//
// WHY THIS EXISTS. PR-4.4 decomposed a commit and found two fsyncs, not
// one: the blob is written to a temp file, fsynced and renamed BEFORE
// SQLite is touched. On Linux the blob half runs at 1 826/s against
// 2 988/s for the SQL half, so it is the larger of the two costs.
//
// Inlining removes it OUTRIGHT for a payload that fits, and — this is
// the part that made it the first move rather than one option among
// three — it weakens nothing. SQLite's own synchronous=FULL then covers
// the payload as well as the index, in the same transaction that was
// already being paid for. The alternatives on the table (synchronous=
// NORMAL, dropping the blob sync) all bought speed by giving up a
// durability promise; this one does not.
//
// THE THRESHOLD IS NOT TUNED, and deliberately so. Every payload
// measured in a real run was between 60 and 161 bytes — 100% under
// 256 — while request bodies are capped at 1 MiB, so anything between
// those two numbers is equally correct for today's traffic and the
// choice cannot be made by measurement. 16 KiB is far above what the
// system produces and far below what would make a row unwieldy, and the
// file path still exists for anything larger.
const InlineThreshold = 16 << 10

// The canonical pragmas ride in the DSN, not in an Exec at Open.
//
// They used to be applied with db.ExecContext on the *sql.DB, and that
// is a trap this file walked into: the pool hands Exec ONE connection,
// so `synchronous=FULL` landed on exactly one of N — probed at 8
// pooled connections, the split was {FULL:1, NORMAL:7}. Nothing failed,
// because modernc.org/sqlite's compiled default is already FULL, which
// means the §2.1 durability guarantee was being provided by the
// driver's default and not by this code; a driver upgrade or swap
// would have dropped it silently. journal_mode=WAL never had the
// problem — it persists in the database file — which is exactly why
// the bug was invisible.
//
// The `_pragma` DSN form is executed by the driver on EVERY new
// connection, and TestPragmasApplyToTheWholePool holds it there.
const canonicalDSNPragmas = "?_pragma=journal_mode(WAL)&_pragma=synchronous(FULL)"

func dsn(dbPath string) string {
	return "file:" + dbPath + canonicalDSNPragmas
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

	db, err := sql.Open("sqlite", dsn(dbPath))
	if err != nil {
		return nil, fmt.Errorf("substrate.Open: sql.Open: %w", err)
	}

	// sql.Open validates nothing; force one connection now so a bad
	// path or an unparseable DSN fails here rather than on first use.
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("substrate.Open: ping: %w", err)
	}

	if _, err := db.ExecContext(ctx, eventsSchemaDDL); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("substrate.Open: create events table: %w", err)
	}

	if _, err := db.ExecContext(ctx, positionSchemaDDL); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("substrate.Open: create stream_position table: %w", err)
	}

	// The payload column, added to databases that predate it. SQLite has
	// no ADD COLUMN IF NOT EXISTS, so this asks first. An archive opened
	// by an older binary and then a newer one must keep every record it
	// already holds readable, which is what the file fallback in
	// ReadBlob is for.
	if err := ensurePayloadColumn(ctx, db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("substrate.Open: %w", err)
	}
	if err := ensureDuplicatesColumn(ctx, db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("substrate.Open: %w", err)
	}

	return &Substrate{db: db, blobDir: blobDir}, nil
}

// ensurePayloadColumn adds events.payload to a database that predates
// it.
//
// A database created by this version already has the column from the
// DDL above; this exists only for archives written before inlining. The
// column is declared in BOTH places on purpose — a fresh database
// should be correct from its schema alone, without depending on a
// migration having run, and a migration that is the only definition is
// a schema nobody can read in one place.
func ensurePayloadColumn(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, `PRAGMA table_info(events)`)
	if err != nil {
		return fmt.Errorf("read events schema: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dflt, &pk); err != nil {
			return fmt.Errorf("scan events schema: %w", err)
		}
		if name == "payload" {
			return rows.Err()
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read events schema: %w", err)
	}

	if _, err := db.ExecContext(ctx, `ALTER TABLE events ADD COLUMN payload BLOB`); err != nil {
		return fmt.Errorf("add payload column: %w", err)
	}
	return nil
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

	inline := len(payload) <= InlineThreshold
	if !inline {
		if err := s.writeBlob(row.EventHash, payload); err != nil {
			return fmt.Errorf("substrate.Append: blob write: %w", err)
		}
	}

	if _, err := s.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO events
		   (event_hash, event_time, message_type, payload_ref, committed_at, payload)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		row.EventHash[:], row.EventTime, row.MessageType, row.PayloadRef, row.CommittedAt,
		inlinedOrNil(payload, inline),
	); err != nil {
		return fmt.Errorf("substrate.Append: insert: %w", err)
	}
	return nil
}

// inlinedOrNil returns the payload when it belongs in the row, and nil
// otherwise. A NULL payload column is what tells a reader to go and
// find the file, so it is the discriminator rather than payload_ref —
// which every row written before this change already carries.
func inlinedOrNil(payload []byte, inline bool) any {
	if inline {
		return payload
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
func (s *Substrate) ReadBlob(ctx context.Context, hash [32]byte) ([]byte, error) {
	// The row first, because that is where a payload written by this
	// version lives. A miss falls through to the file, which is what
	// keeps every record written before the inline change readable —
	// including on an archive that an older binary populated and a newer
	// one opened.
	var inline []byte
	err := s.db.QueryRowContext(ctx,
		`SELECT payload FROM events WHERE event_hash = ?`, hash[:]).Scan(&inline)
	switch {
	case err == nil && inline != nil:
		recomputed := canonical.Hash(inline)
		if subtle.ConstantTimeCompare(hash[:], recomputed[:]) != 1 {
			// The same check the file path makes, for the same reason:
			// a content-addressed store that answers with bytes not
			// describing the name it was asked for is worse than one
			// that answers nothing.
			return nil, fmt.Errorf("substrate.ReadBlob inline %s: %w",
				canonical.HashHex(hash), ErrHashMismatch)
		}
		return inline, nil
	case err != nil && !errors.Is(err, sql.ErrNoRows):
		return nil, fmt.Errorf("substrate.ReadBlob: read row: %w", err)
	}

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

// AppendCanonical commits a record from bytes that are ALREADY
// canonical, deriving the row from them.
//
// It exists so the two producers cannot disagree about how a record
// becomes a row. The collector canonicalizes a live message and commits
// it; the archive service receives canonical bytes over the wire and
// commits those. If each built its own EventRow, the blob-sharding
// scheme would be written twice and would eventually be written
// differently — and a content-addressed store whose two writers shard
// differently is a store with two halves.
//
// The hash is VERIFIED against the payload rather than trusted. For the
// archive service that is the whole safety argument: it never
// re-marshals a message (protobuf's deterministic marshalling makes no
// cross-build promise, so re-marshalling could legitimately produce
// different bytes and a different identity), so verifying is the only
// check it has that the bytes survived the trip.
func (s *Substrate) AppendCanonical(ctx context.Context, payload []byte, eventHash [32]byte,
	eventTime int64, messageType string, committedAt int64) error {

	if got := canonical.Hash(payload); got != eventHash {
		return fmt.Errorf("%w: payload hashes to %s, record claims %s",
			ErrHashMismatch, canonical.HashHex(got), canonical.HashHex(eventHash))
	}

	hexed := canonical.HashHex(eventHash)
	return s.Append(ctx, EventRow{
		EventHash:   eventHash,
		EventTime:   eventTime,
		MessageType: messageType,
		PayloadRef:  hexed[:2] + "/" + hexed[2:],
		CommittedAt: committedAt,
	}, payload)
}

// ---------------------------------------------------------------
// The durable position
// ---------------------------------------------------------------

// ErrNoSequence rejects a position update that carries no sequence.
//
// JetStream numbers stream sequences from 1, so a zero is a caller that
// did not plumb one through rather than a record that genuinely sits at
// the beginning. Recording it would silently drag first_seq to zero and
// make every audit afterwards report a span far larger than the traffic,
// which reads as catastrophic loss. Refusing is the cheaper failure.
var ErrNoSequence = errors.New("substrate: stream sequence is zero")

// ensureDuplicatesColumn adds stream_position.duplicates to a database
// that predates it. Same reasoning as ensurePayloadColumn.
func ensureDuplicatesColumn(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, `PRAGMA table_info(stream_position)`)
	if err != nil {
		return fmt.Errorf("read stream_position schema: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dflt, &pk); err != nil {
			return fmt.Errorf("scan stream_position schema: %w", err)
		}
		if name == "duplicates" {
			return rows.Err()
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read stream_position schema: %w", err)
	}
	if _, err := db.ExecContext(ctx,
		`ALTER TABLE stream_position ADD COLUMN duplicates INTEGER NOT NULL DEFAULT 0`); err != nil {
		return fmt.Errorf("add duplicates column: %w", err)
	}
	return nil
}

// Position is what the archive durably knows about its own progress.
//
// The four numbers exist to make one subtraction possible:
//
//	span        = HighestSeq - FirstSeq + 1
//	accounted   = Committed + Rejected
//	unaccounted = span - accounted
//
// Unaccounted is the number this whole phase was opened to produce:
// records that entered the stream within the range this archive has
// walked, and are not in it, and were not refused. Records still in
// flight count as unaccounted, which is correct — they are genuinely not
// in the archive yet — so the figure is read after traffic has drained,
// not during.
//
// Committed counts COMMIT OPERATIONS, not rows. A record delivered twice
// commits twice and dedups to one row, so Committed minus the row count
// is the duplicate volume rather than a discrepancy. Counting rows here
// instead would have made every deduplicated redelivery look like a
// record that entered the stream and vanished.
type Position struct {
	FirstSeq   uint64
	HighestSeq uint64
	Committed  int64
	Rejected   int64

	// Duplicates are deliveries whose record was ALREADY in the
	// substrate — at-least-once delivery working, not loss.
	//
	// They are counted apart from Committed because folding them in
	// makes the subtraction below go negative. ADR-0008 had Committed
	// counting commit OPERATIONS, which double-counts a redelivered
	// sequence and left `unaccounted` reading -70 under a broker
	// disruption. A loss figure that can be negative is not a loss
	// figure. See ADR-0010.
	Duplicates int64
}

// Span is the number of stream sequences this archive has walked past.
func (p Position) Span() int64 {
	if p.HighestSeq == 0 {
		return 0
	}
	return int64(p.HighestSeq-p.FirstSeq) + 1
}

// Unaccounted is the span minus everything the archive can explain.
//
// DUPLICATES ARE NOT SUBTRACTED, and getting that wrong in either
// direction produces a negative number:
//
//   - ADR-0008 had Committed counting commit OPERATIONS, so a
//     redelivered sequence was counted twice and Committed could exceed
//     the span. The Phase 4 gate observed -70 during a broker
//     disruption.
//   - Subtracting Duplicates as well over-corrects: a redelivery adds
//     no sequence to the span, because the sequence it carries was
//     already accounted for by the original commit. Twenty redeliveries
//     of one record produced -19 in a test written for the first fix.
//
// Committed now counts DISTINCT records, so each sequence is explained
// exactly once and Duplicates is reported beside the subtraction rather
// than inside it.
func (p Position) Unaccounted() int64 {
	return p.Span() - p.Committed - p.Rejected
}

// positionUpsert folds one observation into the single row. MIN and MAX
// keep first_seq and highest_seq monotonic in their own directions, so
// out-of-order delivery — which JetStream permits — cannot walk either
// backwards.
const positionUpsert = `
INSERT INTO stream_position (id, first_seq, highest_seq, committed, rejected, duplicates)
VALUES (1, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    first_seq   = MIN(first_seq,   excluded.first_seq),
    highest_seq = MAX(highest_seq, excluded.highest_seq),
    committed   = committed  + excluded.committed,
    rejected    = rejected   + excluded.rejected,
    duplicates  = duplicates + excluded.duplicates`

// Position reads the durable position. The bool is false when nothing
// has ever been recorded, which is a fresh archive rather than an
// archive at zero — the distinction this repository does not collapse.
func (s *Substrate) Position(ctx context.Context) (Position, bool, error) {
	var p Position
	err := s.db.QueryRowContext(ctx,
		`SELECT first_seq, highest_seq, committed, rejected, duplicates
		   FROM stream_position WHERE id = 1`).
		Scan(&p.FirstSeq, &p.HighestSeq, &p.Committed, &p.Rejected, &p.Duplicates)
	if errors.Is(err, sql.ErrNoRows) {
		return Position{}, false, nil
	}
	if err != nil {
		return Position{}, false, fmt.Errorf("substrate.Position: %w", err)
	}
	return p, true, nil
}

// PositionAndCount reads the durable position and the row count as ONE
// consistent pair.
//
// Reading them with two separate queries looks equivalent and is not.
// Under load, records commit between the two reads, so the row count
// includes commits the position read missed and the published pair says
// the archive holds more rows than it performed commits — which reads as
// rows appearing from nowhere. It was the Phase 4 gate that surfaced
// this: 169 104 commits against 169 106 rows, a skew of two records at
// 2 000 sessions/s.
//
// The numbers were never wrong; the SNAPSHOT was. A single read
// transaction makes the pair describe one instant, which is what a
// scraper assumes it is getting.
func (s *Substrate) PositionAndCount(ctx context.Context) (Position, int64, bool, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return Position{}, 0, false, fmt.Errorf("substrate.PositionAndCount: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var p Position
	err = tx.QueryRowContext(ctx,
		`SELECT first_seq, highest_seq, committed, rejected, duplicates
		   FROM stream_position WHERE id = 1`).
		Scan(&p.FirstSeq, &p.HighestSeq, &p.Committed, &p.Rejected, &p.Duplicates)
	if errors.Is(err, sql.ErrNoRows) {
		return Position{}, 0, false, nil
	}
	if err != nil {
		return Position{}, 0, false, fmt.Errorf("substrate.PositionAndCount: position: %w", err)
	}

	var rows int64
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM events`).Scan(&rows); err != nil {
		return Position{}, 0, false, fmt.Errorf("substrate.PositionAndCount: count: %w", err)
	}
	return p, rows, true, nil
}

// AppendCanonicalAt commits a record and advances the position in ONE
// transaction.
//
// That is the point of the method existing beside AppendCanonical. If
// the two were separate writes, a crash between them would leave an
// archive holding a record it does not believe it holds, or believing it
// holds one it does not — and the audit built on top would report a gap
// that is an artefact of when the power failed.
//
// The blob is written first and outside the transaction. A blob with no
// row is already possible and already harmless: it is content-addressed,
// so it is either garbage-collected or rewritten identically. A row with
// no blob would not be.
// BatchRecord is one record in a batched commit.
type BatchRecord struct {
	Payload     []byte
	EventHash   [32]byte
	EventTime   int64
	MessageType string
	Seq         uint64
}

// AppendCanonicalBatch commits many records in ONE transaction.
//
// synchronous=FULL costs one fsync per transaction, not per row, so a
// batch of N amortises that fsync across N records. Everything else is
// unchanged and deliberately so: each payload is still hash-verified
// before it is written, each oversized payload still gets its own
// fsynced blob first, and the durable position still advances inside
// the same transaction as the rows — so "the archive holds record N"
// and "the archive reached sequence N" still cannot disagree across a
// crash.
//
// REJECTED SEQUENCES TRAVEL WITH THE BATCH, and that is not a
// convenience. A caller that recorded rejections separately and then
// failed to commit the rest would have advanced the position for the
// rejects; redelivery would record them a second time, inflating
// `rejected` past the sequences actually walked and driving
// `unaccounted` negative — the ADR-0010 failure, reintroduced by the
// back door. One transaction covers what was kept and what was
// refused, so a rollback un-decides both.
//
// A batch is ALL OR NOTHING. One bad record fails the whole
// transaction, and the caller re-drives the batch one record at a time
// to find it; that keeps the fast path free of per-record error
// handling and makes the slow path exact.
func (s *Substrate) AppendCanonicalBatch(ctx context.Context, recs []BatchRecord,
	rejected []uint64, committedAt int64) error {

	if len(recs) == 0 && len(rejected) == 0 {
		return nil
	}

	// Verify before touching anything durable. A hash mismatch found
	// halfway through would otherwise leave blobs on disk for records
	// the transaction is about to roll back.
	for _, r := range recs {
		if r.Seq == 0 {
			return ErrNoSequence
		}
		if got := canonical.Hash(r.Payload); got != r.EventHash {
			return fmt.Errorf("%w: payload hashes to %s, record claims %s",
				ErrHashMismatch, canonical.HashHex(got), canonical.HashHex(r.EventHash))
		}
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	// Oversized payloads keep their own file, written and fsynced
	// before the rows that name them. These are NOT amortised — a blob
	// is a blob — which is the honest limit of this optimisation and
	// the reason PR-4.3's inlining had to come first.
	for _, r := range recs {
		if len(r.Payload) > InlineThreshold {
			if err := s.writeBlob(r.EventHash, r.Payload); err != nil {
				return fmt.Errorf("substrate.AppendCanonicalBatch: blob write: %w", err)
			}
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("substrate.AppendCanonicalBatch: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Prepared once and reused: N inserts in one transaction is the
	// point, and re-parsing the statement N times would give some of it
	// back.
	stmt, err := tx.PrepareContext(ctx,
		`INSERT OR IGNORE INTO events
		   (event_hash, event_time, message_type, payload_ref, committed_at, payload)
		 VALUES (?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("substrate.AppendCanonicalBatch: prepare: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	var committed, duplicates int64
	first, highest := uint64(0), uint64(0)
	span := func(seq uint64) {
		if first == 0 || seq < first {
			first = seq
		}
		if seq > highest {
			highest = seq
		}
	}
	for _, seq := range rejected {
		if seq == 0 {
			return ErrNoSequence
		}
		span(seq)
	}

	for _, r := range recs {
		hexed := canonical.HashHex(r.EventHash)
		inline := len(r.Payload) <= InlineThreshold
		res, err := stmt.ExecContext(ctx,
			r.EventHash[:], r.EventTime, r.MessageType, hexed[:2]+"/"+hexed[2:],
			committedAt, inlinedOrNil(r.Payload, inline))
		if err != nil {
			return fmt.Errorf("substrate.AppendCanonicalBatch: insert: %w", err)
		}
		// Same rule as the single-record path (ADR-0010): a
		// primary-key conflict is a redelivery, not a commit, and
		// counting it as one drives unaccounted negative.
		affected, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("substrate.AppendCanonicalBatch: rows affected: %w", err)
		}
		if affected == 0 {
			duplicates++
		} else {
			committed++
		}
		span(r.Seq)
	}

	if _, err := tx.ExecContext(ctx, positionUpsert,
		first, highest, committed, int64(len(rejected)), duplicates); err != nil {
		return fmt.Errorf("substrate.AppendCanonicalBatch: position: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("substrate.AppendCanonicalBatch: commit: %w", err)
	}
	return nil
}

func (s *Substrate) AppendCanonicalAt(ctx context.Context, payload []byte, eventHash [32]byte,
	eventTime int64, messageType string, committedAt int64, seq uint64) error {

	if seq == 0 {
		return ErrNoSequence
	}
	if got := canonical.Hash(payload); got != eventHash {
		return fmt.Errorf("%w: payload hashes to %s, record claims %s",
			ErrHashMismatch, canonical.HashHex(got), canonical.HashHex(eventHash))
	}

	hexed := canonical.HashHex(eventHash)
	payloadRef := hexed[:2] + "/" + hexed[2:]

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	// A payload that fits goes into the transaction below, which is
	// already paying for an fsync. One that does not still gets its own
	// file, written and fsynced before the row that names it — a row
	// pointing at a file that is not there yet would be a record the
	// archive believes it holds and cannot produce.
	inline := len(payload) <= InlineThreshold
	if !inline {
		if err := s.writeBlob(eventHash, payload); err != nil {
			return fmt.Errorf("substrate.AppendCanonicalAt: blob write: %w", err)
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("substrate.AppendCanonicalAt: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx,
		`INSERT OR IGNORE INTO events
		   (event_hash, event_time, message_type, payload_ref, committed_at, payload)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		eventHash[:], eventTime, messageType, payloadRef, committedAt,
		inlinedOrNil(payload, inline),
	)
	if err != nil {
		return fmt.Errorf("substrate.AppendCanonicalAt: insert: %w", err)
	}

	// WAS THIS RECORD ALREADY HERE? INSERT OR IGNORE reports zero rows
	// affected on a primary-key conflict, which is exactly a delivery of
	// a record the substrate already holds. Counting it as a commit
	// inflates `committed` past the number of sequences walked and drives
	// `unaccounted` negative — which the Phase 4 gate observed at -70
	// while the broker was being disrupted. See ADR-0010.
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("substrate.AppendCanonicalAt: rows affected: %w", err)
	}
	committed, duplicates := int64(1), int64(0)
	if affected == 0 {
		committed, duplicates = 0, 1
	}
	if _, err := tx.ExecContext(ctx, positionUpsert,
		seq, seq, committed, 0, duplicates); err != nil {
		return fmt.Errorf("substrate.AppendCanonicalAt: position: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("substrate.AppendCanonicalAt: commit: %w", err)
	}
	return nil
}

// RecordRejected advances the position for a record the archive refused.
//
// A refusal is an ACCOUNTED outcome, not a loss. Without this the
// sequence would still be walked past — highest_seq comes from whatever
// arrives last — and the record would surface as unaccounted, blaming
// the stream for a decision the archive made deliberately and logged.
func (s *Substrate) RecordRejected(ctx context.Context, seq uint64) error {
	if seq == 0 {
		return ErrNoSequence
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	if _, err := s.db.ExecContext(ctx, positionUpsert, seq, seq, 0, 1, 0); err != nil {
		return fmt.Errorf("substrate.RecordRejected: %w", err)
	}
	return nil
}
