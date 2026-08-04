package app

import (
	"errors"

	"github.com/FabioCaffarello/ghost-trace/libs/archive"
)

// Typed errors are the use cases' entire failure vocabulary; the
// transport adapter owns the single table that maps them to status
// codes. Handlers must not invent per-site translations — that was how
// the same error came to mean different things at different endpoints.
var (
	// ErrSessionNotFound: unknown or expired session token. Telemetry
	// treats it as expected loss.
	ErrSessionNotFound = errors.New("app: session not found")

	// ErrArchiveUnavailable is libs/archive's sentinel, re-exported so
	// this package's callers need not import it. It is the SAME value —
	// errors.Is has to hold across the boundary.
	ErrArchiveUnavailable = archive.ErrUnavailable
)
