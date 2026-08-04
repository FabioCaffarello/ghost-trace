package decision

import (
	"errors"

	"github.com/FabioCaffarello/ghost-trace/libs/archive"
)

// Typed errors are the use cases' entire failure vocabulary; the
// handlers below own the single table that maps them to status codes.
// Nothing may invent a per-site translation — that was how the same
// error came to mean different things at different endpoints.
var (
	// ErrArchiveUnavailable is re-exported so a host can match on it
	// without importing libs/archive. It is the SAME error value, not a
	// parallel one: errors.Is must hold across the boundary or outcomes
	// would refuse with a 500 in one service and a 503 in the other.
	ErrArchiveUnavailable = archive.ErrUnavailable

	// Validation failures, surfaced as 400s.
	ErrActionRequired       = errors.New("decision: action is required")
	ErrEvaluationIDRequired = errors.New("decision: evaluation_id is required")
	ErrUnknownOutcome       = errors.New("decision: unknown outcome")
)
