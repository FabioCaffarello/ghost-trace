package decision

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/FabioCaffarello/ghost-trace/libs/archive"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/libs/genproto/events/v1"
)

// ValidOutcomes is the §3 enumeration. An unknown value is rejected
// rather than stored: a typo'd label is worse than a missing one,
// because it silently degrades the calibration everything else depends
// on.
var ValidOutcomes = map[string]bool{
	"login_success":    true,
	"login_failure":    true,
	"challenge_passed": true,
	"challenge_failed": true,
	"fraud_confirmed":  true,
	"user_appealed":    true,
	"abandoned":        true,
}

// OutcomeInput is the §3 POST /v1/outcomes request, transport-free.
// A zero ObservedAt means "now".
type OutcomeInput struct {
	// TenantID is who the caller proved to be. A label filed against
	// another tenant's evaluation would poison THAT tenant's
	// calibration, which is the quietest cross-tenant damage available.
	TenantID string

	EvaluationID string
	Outcome      string
	ObservedAt   time.Time
}

// RecordOutcome durably records a label against an evaluation.
//
// Unlike every other archive write, this one is not best-effort:
// returning success for a label that was not stored would silently
// poison the calibration loop, so ErrUnavailable and write failures
// surface to the caller.
func (s *Service) RecordOutcome(ctx context.Context, in OutcomeInput) error {
	if in.TenantID == "" {
		return ErrTenantRequired
	}
	if in.EvaluationID == "" {
		return ErrEvaluationIDRequired
	}
	if !ValidOutcomes[in.Outcome] {
		return ErrUnknownOutcome
	}

	observedAt := in.ObservedAt
	if observedAt.IsZero() {
		observedAt = s.now()
	}

	rec := &eventsv1.Outcome{
		TenantId:     in.TenantID,
		EvaluationId: in.EvaluationID,
		Outcome:      in.Outcome,
		ObservedAt:   observedAt.UnixNano(),
		RecordedAt:   s.now().UnixNano(),
	}

	if err := s.archive.Append(ctx, rec, rec.RecordedAt); err != nil {
		if errors.Is(err, archive.ErrUnavailable) {
			return ErrArchiveUnavailable
		}
		return fmt.Errorf("record outcome: %w", err)
	}
	return nil
}
