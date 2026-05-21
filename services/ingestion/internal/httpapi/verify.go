package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/verify"
)

// handleVerify serves GET /v1/verify per decision-log §0093.
// Substrate-integrity audit; mirrors cmd/verify (per §0039 + §0040)
// over HTTP. Walks every events-table row, recomputes each blob's
// BLAKE3 hash via substrate.ReadBlob, surfaces hash-mismatch +
// missing-blob failures. With check_orphans=true, also walks the
// blob-store directory and reports orphan blobs (informational per
// §0033 + §0040; not a §2.1 violation).
//
// Wire shape matches cmd/verify's stdout JSON exactly so operators
// get the same response regardless of channel (CLI or HTTP). The
// CLI's exit-code semantic maps to HTTP as follows:
//   - CLI exit 0 (pass) → HTTP 200 with passed=true
//   - CLI exit 1 (substrate-integrity violation) → HTTP 200 with
//     passed=false. Operators inspect the `passed` boolean (and the
//     hash_mismatch_hashes / missing_blob_hashes arrays) to detect
//     violations; the substrate-integrity failure verdict is a body
//     field, not an HTTP error code. This matches §0091's drift
//     semantic (replay drift returns 200 with match=false).
//   - CLI exit 2 (tool/config error, e.g. substrate not openable) →
//     HTTP 503 (substrate not configured) or 500 (walk error).
//
// Query parameters:
//   - check_orphans (optional, default false) — "true"/"1" enables
//     the orphan-blob second pass.
//
// Response codes:
//   - 200 — JSON report (passed boolean carries the verdict).
//   - 400 — invalid check_orphans value.
//   - 405 — non-GET method.
//   - 500 — internal error during the walk (rare; database I/O).
//   - 503 — substrate not configured on this handler (the
//     WithSubstrate option was not supplied at construction).
//
// Operational cost: this endpoint walks every substrate row + reads
// every blob. For large substrates the call is slow. Operators
// SHOULD run it on a schedule (cron / systemd timer) against a
// dedicated read-only auth token rather than from interactive
// dashboards. Concurrent invocations are safe under SQLite WAL but
// each multiplies the read load.
func (h *Handler) handleVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.sub == nil {
		writeIngestError(w, http.StatusServiceUnavailable,
			"verify endpoint not configured (handler constructed without WithSubstrate)")
		return
	}

	checkOrphans, ok := parseBoolQuery(w, r, "check_orphans", false)
	if !ok {
		return
	}

	report, err := verify.Verify(r.Context(), h.sub, verify.Options{CheckOrphans: checkOrphans})
	if err != nil {
		writeIngestError(w, http.StatusInternalServerError, fmt.Sprintf("verify: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(verifyPayload{
		Verified:           report.VerifiedCount,
		HashMismatch:       report.HashMismatchCount,
		MissingBlob:        report.MissingBlobCount,
		OrphanBlob:         report.OrphanBlobCount,
		Passed:             !report.Failed(),
		HashMismatchHashes: report.HashMismatchHashes,
		MissingBlobHashes:  report.MissingBlobHashes,
		OrphanBlobPaths:    report.OrphanBlobPaths,
	})
}

// verifyPayload is the wire shape of the GET /v1/verify JSON response.
// Mirrors cmd/verify's reportPayload exactly so the CLI and HTTP
// channels emit identical bytes for the same substrate.
type verifyPayload struct {
	Verified           int64    `json:"verified"`
	HashMismatch       int64    `json:"hash_mismatch"`
	MissingBlob        int64    `json:"missing_blob"`
	OrphanBlob         int64    `json:"orphan_blob"`
	Passed             bool     `json:"passed"`
	HashMismatchHashes []string `json:"hash_mismatch_hashes,omitempty"`
	MissingBlobHashes  []string `json:"missing_blob_hashes,omitempty"`
	OrphanBlobPaths    []string `json:"orphan_blob_paths,omitempty"`
}

// parseBoolQuery extracts an optional boolean query parameter. Accepts
// strconv.ParseBool's vocabulary ("1", "t", "T", "TRUE", "true",
// "True", "0", "f", "F", "FALSE", "false", "False"). Missing → returns
// defaultValue, true. Invalid → writes 400 + returns false. Used for
// HTTP-flag parity with CLI boolean flags.
func parseBoolQuery(w http.ResponseWriter, r *http.Request, name string, defaultValue bool) (bool, bool) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return defaultValue, true
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		writeIngestError(w, http.StatusBadRequest,
			fmt.Sprintf("query parameter %s=%q is not a valid boolean (accepts true/false/1/0)", name, raw))
		return false, false
	}
	return v, true
}
