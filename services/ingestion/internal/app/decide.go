package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/FabioCaffarello/ghost-trace/libs/policy"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/session"
)

// DecideInput is the §3 POST /v1/decisions request, transport-free.
// SubjectID and Action arrive only from the authenticated application
// server — authenticating that is the transport's job.
type DecideInput struct {
	SessionToken string
	Action       string
	SubjectID    string
}

// DecideOutput is the judgement plus everything the caller needs to
// act on it and to join it later (§3).
type DecideOutput struct {
	EvaluationID   string
	Decision       string
	ShadowDecision string
	Score          float64
	Confidence     float64
	Reasons        []policy.Reason
	EvidenceEvents uint32
	EvidenceMs     uint32
	Mode           string
}

// Decide scores the session for one action and archives the durable
// evaluation record.
//
// An unknown token is not an error: the caller is at a risk moment and
// needs an answer, and a missing session is a cold start the
// confidence dimension already models (§7). State is per-session,
// scoring is per-action (§8.6).
func (a *App) Decide(ctx context.Context, in DecideInput) (DecideOutput, error) {
	if in.Action == "" {
		return DecideOutput{}, ErrActionRequired
	}

	// Everything needed after the callback is copied by value while the
	// store lock is held. The *session.State pointer must not escape:
	// concurrent telemetry batches keep mutating the state under the
	// lock, and a read after With returns would race with them.
	var (
		st         policy.State
		durationMs uint32
		sessionID  string
		found      = true
	)
	tenantID := a.cfg.TenantID
	if err := a.sessions.With(in.SessionToken, func(sess *session.State) {
		st = policy.State{
			Pointer:     sess.Pointer.State(),
			Keystroke:   sess.Keystroke.State(),
			Interaction: sess.Interaction.State(),
		}
		durationMs = sess.LastEventMs
		sessionID = sess.ID
		tenantID = sess.TenantID
	}); err != nil {
		if !errors.Is(err, session.ErrNotFound) {
			return DecideOutput{}, fmt.Errorf("decide: %w", err)
		}
		found = false
	}

	j := policy.Judge(st, in.Action)
	outcome, err := policy.Apply(j, a.cfg.Mode)
	if err != nil {
		return DecideOutput{}, fmt.Errorf("decide: policy apply: %w", err)
	}

	evalID, err := session.NewID("ev_")
	if err != nil {
		return DecideOutput{}, fmt.Errorf("decide: mint evaluation id: %w", err)
	}

	out := DecideOutput{
		EvaluationID:   evalID,
		Decision:       outcome.Decision,
		ShadowDecision: outcome.Shadow,
		Score:          j.Score(),
		Confidence:     j.Confidence(),
		Reasons:        j.Reasons(),
		EvidenceEvents: st.Pointer.Points + st.Keystroke.Keys,
		Mode:           a.cfg.Mode,
	}
	if found {
		out.EvidenceMs = durationMs
	}

	rec := buildEvaluation(tenantID, sessionID, in, out, st, j, a.now().UnixNano())
	a.archiveBestEffort(ctx, rec, rec.DecidedAt, "evaluation", "evaluation_id", evalID)

	return out, nil
}
