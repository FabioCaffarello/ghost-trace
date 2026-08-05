package decision

import (
	"context"
	"fmt"

	"github.com/FabioCaffarello/ghost-trace/libs/id"
	"github.com/FabioCaffarello/ghost-trace/libs/policy"
)

// Input is the §3 POST /v1/decisions request, transport-free.
// SubjectID and Action arrive only from the authenticated application
// server — authenticating that is the transport's job.
type Input struct {
	// TenantID is who the caller proved to be, resolved from the secret
	// it presented — never a field a request may set.
	TenantID string

	SessionToken string
	Action       string
	SubjectID    string
}

// Output is the judgement plus everything the caller needs to act on it
// and to join it later (§3).
type Output struct {
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
// needs an answer, and a missing session is a cold start the confidence
// dimension already models (§7). State is per-session, scoring is
// per-action (§8.6).
func (s *Service) Decide(ctx context.Context, in Input) (Output, error) {
	if in.TenantID == "" {
		return Output{}, ErrTenantRequired
	}
	if in.Action == "" {
		return Output{}, ErrActionRequired
	}

	// Scoped to the tenant the caller PROVED, so a token belonging to
	// someone else reads as a session that does not exist rather than
	// as a session about which this caller may be told anything.
	sess, found, err := s.sessions.Lookup(ctx, in.TenantID, in.SessionToken)
	if err != nil {
		return Output{}, fmt.Errorf("decide: %w", err)
	}

	// Attribution is the caller's tenant either way. A found session
	// carries the same id by construction — the lookup refused anything
	// else — and an unknown token is a cold start belonging to whoever
	// asked about it.
	tenantID := in.TenantID

	j := policy.Judge(sess.State, in.Action)
	outcome, err := policy.Apply(j, s.cfg.Mode)
	if err != nil {
		return Output{}, fmt.Errorf("decide: policy apply: %w", err)
	}

	evalID, err := id.New("ev_")
	if err != nil {
		return Output{}, fmt.Errorf("decide: mint evaluation id: %w", err)
	}

	out := Output{
		EvaluationID:   evalID,
		Decision:       outcome.Decision,
		ShadowDecision: outcome.Shadow,
		Score:          j.Score(),
		Confidence:     j.Confidence(),
		Reasons:        j.Reasons(),
		EvidenceEvents: sess.State.Pointer.Points + sess.State.Keystroke.Keys,
		Mode:           s.cfg.Mode,
	}
	if found {
		out.EvidenceMs = sess.LastEventMs
	}

	rec := buildEvaluation(tenantID, sess.ID, in, out, sess.State, j, s.now().UnixNano())
	s.archiveBestEffort(ctx, rec, rec.DecidedAt, "evaluation", "evaluation_id", evalID)

	return out, nil
}
