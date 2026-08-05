// Package livesessions adapts the collector's in-memory session store
// to the port libs/decision reads state through.
//
// This is the collector's half of the A/B: it answers a decision from
// the state it is maintaining right now. The decision engine's half
// answers the same question from the snapshot that state was published
// as, and the two are meant to agree. Keeping the adapter this thin is
// what makes that comparison about the STATE and not about two
// different ways of asking for it.
package livesessions

import (
	"context"
	"errors"

	"github.com/FabioCaffarello/ghost-trace/libs/decision"
	"github.com/FabioCaffarello/ghost-trace/libs/policy"
	"github.com/FabioCaffarello/ghost-trace/services/collector/internal/session"
)

// Store is the subset of the session store this needs.
type Store interface {
	With(token string, fn func(*session.State)) error
}

// Sessions reads decision state out of a live session store.
type Sessions struct{ store Store }

// New adapts store.
func New(store Store) *Sessions { return &Sessions{store: store} }

// Lookup copies the state a decision needs out from under the store
// lock.
//
// Everything is copied BY VALUE inside the callback. The *session.State
// pointer must not escape: concurrent telemetry batches keep mutating
// it under the same lock, and a read after With returns would race with
// them.
func (s *Sessions) Lookup(_ context.Context, tenantID, token string) (decision.Session, bool, error) {
	var out decision.Session
	var mine bool
	if err := s.store.With(token, func(sess *session.State) {
		// THE ISOLATION CHECK. Tokens are 144 bits of randomness, so
		// nobody GUESSES another tenant's token — but a token can be
		// handed over, logged, or copied out of a page, and before this
		// existed presenting one with a different tenant's secret
		// returned a real decision about a session the caller had no
		// claim to. Both halves looked valid on their own, which is why
		// nothing caught it.
		if sess.TenantID != tenantID {
			return
		}
		mine = true
		out = decision.Session{
			ID:       sess.ID,
			TenantID: sess.TenantID,
			State: policy.State{
				Pointer:     sess.Pointer.State(),
				Keystroke:   sess.Keystroke.State(),
				Interaction: sess.Interaction.State(),
			},
			LastEventMs: sess.LastEventMs,
		}
	}); err != nil {
		if errors.Is(err, session.ErrNotFound) {
			// Not an error: an unknown token is a cold start, which the
			// confidence dimension already models (§7).
			return decision.Session{}, false, nil
		}
		return decision.Session{}, false, err
	}
	// Someone else's session is reported as no session at all, never as
	// an error: the answer a caller gets must not tell them whether a
	// token they do not own exists.
	if !mine {
		return decision.Session{}, false, nil
	}
	return out, true, nil
}
