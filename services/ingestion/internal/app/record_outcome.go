package app

import (
	"context"
	"errors"
	"fmt"
	"time"

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

// RecordOutcomeInput is the §3 POST /v1/outcomes request,
// transport-free. A zero ObservedAt means "now".
type RecordOutcomeInput struct {
	EvaluationID string
	Outcome      string
	ObservedAt   time.Time
}

// RecordOutcome durably records a label against an evaluation.
//
// Unlike every other archive write, this one is not best-effort:
// returning success for a label that was not stored would silently
// poison the calibration loop, so ErrArchiveUnavailable and write
// failures surface to the caller.
func (a *App) RecordOutcome(ctx context.Context, in RecordOutcomeInput) error {
	if in.EvaluationID == "" {
		return ErrEvaluationIDRequired
	}
	if !ValidOutcomes[in.Outcome] {
		return ErrUnknownOutcome
	}

	observedAt := in.ObservedAt
	if observedAt.IsZero() {
		observedAt = a.now()
	}

	rec := &eventsv1.Outcome{
		TenantId:     a.cfg.TenantID,
		EvaluationId: in.EvaluationID,
		Outcome:      in.Outcome,
		ObservedAt:   observedAt.UnixNano(),
		RecordedAt:   a.now().UnixNano(),
	}

	if err := a.archive.Append(ctx, rec, rec.RecordedAt); err != nil {
		if errors.Is(err, ErrArchiveUnavailable) {
			return ErrArchiveUnavailable
		}
		return fmt.Errorf("record outcome: %w", err)
	}
	return nil
}
