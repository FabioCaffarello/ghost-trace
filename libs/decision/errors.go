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

	// ErrTenantRequired: the use case was called with no tenant.
	//
	// Unreachable through HTTP — the handlers only ever pass a tenant
	// they resolved from a presented secret, and the registry refuses
	// an empty id. It exists for callers of the LIBRARY, because
	// without it a forgotten tenant scopes every lookup to "" and every
	// session reads as a cold start: a silent wrong answer rather than
	// a loud one. That is exactly how it was found — the shadow test
	// dropped the field and compared a cold start against a real
	// judgement.
	ErrTenantRequired = errors.New("decision: tenant is required")

	// Validation failures, surfaced as 400s.
	ErrActionRequired       = errors.New("decision: action is required")
	ErrEvaluationIDRequired = errors.New("decision: evaluation_id is required")
	ErrUnknownOutcome       = errors.New("decision: unknown outcome")
)
