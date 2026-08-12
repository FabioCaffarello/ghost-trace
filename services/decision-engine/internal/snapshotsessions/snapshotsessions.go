// Package snapshotsessions reads decision state out of the snapshot
// store the collector publishes to.
//
// This is the decision engine's half of the A/B. The collector answers
// a decision from the session it is maintaining in memory; this answers
// the same question from the snapshot that session was published as,
// and the two are meant to agree. libs/snapshot owns the mapping and
// measured its loss (ADR-0004); this owns only the fetch.
package snapshotsessions

import (
	"context"
	"errors"
	"time"

	"github.com/FabioCaffarello/ghost-trace/libs/decision"
	"github.com/FabioCaffarello/ghost-trace/libs/eventstream"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/libs/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/libs/snapshot"
)

// Store is the subset of the snapshot bucket this needs.
type Store interface {
	Get(ctx context.Context, tenant, token string) (*eventsv1.SessionSnapshot, error)
}

// LookupTimeout bounds how long a decision waits for its evidence.
//
// Found by `make kill-test` on the first run of the latency assertion
// PR-5.1 added, and it is the same defect as that PR's in a second
// place: with the broker stopped, this read waits for the client's own
// ack timeout — about five seconds — and the engine answers 500 five
// seconds late. A caller kept waiting five seconds for an error is
// worse off than one kept waiting for a wrong answer, and this is the
// path with an 80ms budget and someone at a risk moment.
//
// 250ms is deliberately LARGER than that budget. A bound tight enough
// to fire inside it would convert "slow" into "broken" on an ordinary
// bad day; this one only fires when the store is not answering at all.
// The KV read costs single-digit milliseconds when it works, so this
// is roughly eighty times the normal cost and a twentieth of the stall
// it replaces.
const LookupTimeout = 250 * time.Millisecond

// Sessions answers lookups from the snapshot bucket.
type Sessions struct {
	store Store
}

// New reads snapshots out of store.
//
// The tenant is no longer configuration. The bucket key has always been
// (tenant, token) and the caller only ever supplied the token; the
// missing half used to come from a flag, which meant one engine could
// serve exactly one customer. It now comes from the secret the caller
// authenticated with, so a token is only ever looked up under the
// tenant that proved it may ask.
func New(store Store) *Sessions {
	return &Sessions{store: store}
}

// Lookup fetches the snapshot and maps it to decision state.
//
// The two failure modes are kept apart, which is the whole reason this
// adapter is worth reading. A session with no snapshot is a cold start
// and scores as one. A store that did not answer is NOT: reporting it
// as a missing session would turn a broker outage into a stream of
// brand-new sessions, every one of them judged innocent for lack of
// evidence — a detector that fails open silently at exactly the moment
// its evidence supply breaks.
func (s *Sessions) Lookup(ctx context.Context, tenantID, token string) (decision.Session, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, LookupTimeout)
	defer cancel()

	// Isolation is the KEY here rather than a comparison: another
	// tenant's token simply addresses a key that does not exist, and
	// reads as a cold start.
	snap, err := s.store.Get(ctx, tenantID, token)
	if err != nil {
		if errors.Is(err, eventstream.ErrNoSnapshot) {
			return decision.Session{}, false, nil
		}
		// A deadline is an ERROR and not a miss, which is the same
		// distinction the rest of this function turns on. Timing out is
		// the store failing to answer; treating it as "no snapshot"
		// would score every session innocent for lack of evidence at
		// exactly the moment the evidence supply broke.
		return decision.Session{}, false, err
	}
	return decision.Session{
		ID:          snap.GetSessionId(),
		TenantID:    snap.GetTenantId(),
		State:       snapshot.ToState(snap.GetFeatures()),
		LastEventMs: snap.GetLastEventMs(),
	}, true, nil
}
