package app

import "errors"

// Typed errors are the use cases' entire failure vocabulary; the
// transport adapter owns the single table that maps them to status
// codes. Handlers must not invent per-site translations — that was how
// the same error came to mean different things at different endpoints.
var (
	// ErrSessionNotFound: unknown or expired session token. Telemetry
	// treats it as expected loss; decisions treat it as cold start.
	ErrSessionNotFound = errors.New("app: session not found")

	// ErrArchiveUnavailable: the run has no durable archive configured.
	// Best-effort archival ignores it; outcomes refuse on it, because a
	// label the caller believes recorded but is not would silently
	// poison calibration.
	ErrArchiveUnavailable = errors.New("app: event archive unavailable")

	// Validation failures, surfaced as 400s by the transport.
	ErrActionRequired       = errors.New("app: action is required")
	ErrEvaluationIDRequired = errors.New("app: evaluation_id is required")
	ErrUnknownOutcome       = errors.New("app: unknown outcome")
)
