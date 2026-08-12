// Package session holds per-session feature state in process.
//
// This is the maintained-state side of the architecture argument: each
// session carries a fixed-size accumulator that is O(1) in session
// duration, so a decision is a read of current state plus a policy
// evaluation rather than a fetch-and-compute over the raw event
// history.
//
// M1 keeps this in a map in one process. That is not the interesting
// version — surviving a deploy, spanning replicas, and holding tens of
// thousands of concurrent sessions is M4's problem, and M5 measures
// whether any of it was necessary. Keeping it deliberately naive here
// means the measurement compares architectures rather than
// optimisations.
package session

import (
	"errors"
	"sync"
	"time"

	"github.com/FabioCaffarello/ghost-trace/libs/feature"
	"github.com/FabioCaffarello/ghost-trace/libs/id"
)

// ErrNotFound is returned for an unknown or expired token.
var ErrNotFound = errors.New("session: token not found or expired")

// Client holds the normalization properties reported at session start.
//
// These change how behaviour should be interpreted; they are not used
// to tell one browser from another (contract §0).
type Client struct {
	PointerType   string `json:"pointer"`
	Touch         bool   `json:"touch"`
	Viewport      [2]int `json:"viewport"`
	TZOffsetMin   int    `json:"tz_offset"`
	ReducedMotion bool   `json:"reduced_motion"`
}

// State is one session's accumulated evidence.
type State struct {
	ID        string
	TenantID  string
	PagePath  string
	Client    Client
	StartedAt time.Time

	// Feature accumulators, all O(1) in session duration.
	Pointer     feature.Pointer
	Keystroke   feature.Keystroke
	Interaction feature.Interaction

	// HighestSeq tracks the largest batch counter seen. Batches arrive
	// out of order, so this is a high-water mark, not a cursor.
	HighestSeq uint32

	// BatchesSeen counts accepted batches. With HighestSeq it exposes
	// gaps, which are meaningful (contract §2).
	BatchesSeen uint32

	// LastEventMs is the largest client-claimed session-relative
	// timestamp seen, used as the session's observed duration.
	LastEventMs uint32
}

// ObserveBatch records the arrival of one accepted telemetry envelope.
//
// Batches arrive out of order and get retried (contract §2), so both
// counters are high-water marks, never cursors: HighestSeq minus
// BatchesSeen exposes gaps, and gaps are meaningful. This is the
// domain invariant — callers must not reimplement it inline.
func (st *State) ObserveBatch(seq, sentAtMs uint32) {
	st.BatchesSeen++
	if seq > st.HighestSeq {
		st.HighestSeq = seq
	}
	if sentAtMs > st.LastEventMs {
		st.LastEventMs = sentAtMs
	}
}

// ObserveEventTime advances the observed session duration to t if it
// is the latest client-claimed timestamp seen.
func (st *State) ObserveEventTime(t uint32) {
	if t > st.LastEventMs {
		st.LastEventMs = t
	}
}

// entry pairs a session with its expiry so the two cannot fall out of
// step — one map, one delete, no parallel bookkeeping.
type entry struct {
	state     *State
	expiresAt time.Time
}

// Store maps tokens to session state.
//
// One mutex guards the whole map. At M1 volume that is correct and
// boring; sharding it before there is a measurement showing contention
// would be optimising against a guess.
type Store struct {
	mu      sync.Mutex
	byToken map[string]*entry
	ttl     time.Duration
	now     func() time.Time
}

// NewStore constructs a Store with the given token lifetime.
func NewStore(ttl time.Duration, now func() time.Time) *Store {
	if now == nil {
		now = time.Now
	}
	return &Store{
		byToken: make(map[string]*entry),
		ttl:     ttl,
		now:     now,
	}
}

// Identity is the part of a session that is fixed at Create and never
// changes: what the record is about, rather than what it has observed.
//
// It exists so Create has something to return that is NOT the state the
// store holds. Every field here is a value, and nothing in the store
// writes to any of them after Create, so a copy stays true — which is
// not something a copy of State could claim, since two of the three
// accumulators carry maps.
type Identity struct {
	ID        string
	TenantID  string
	PagePath  string
	StartedAt time.Time
}

// Create issues a new session and returns its token and identity.
//
// The token and the session id are distinct values. The token is a
// bearer credential the browser holds; the id is what the archive
// records. Reusing one as the other would put a live credential into
// storage that outlives it by 7 days.
//
// It returns an Identity and not the *State, because With's contract is
// that the pointer never leaves the store — "access is serialized
// rather than handing out the pointer". Create used to hand out exactly
// that pointer, to the one session most likely to be written next. No
// caller had raced it yet, only because the token had not reached the
// browser; the fix is structural rather than a rule to remember.
func (s *Store) Create(tenantID, pagePath string, c Client) (token string, ident Identity, err error) {
	token, err = NewID("st_")
	if err != nil {
		return "", Identity{}, err
	}
	id, err := NewID("s_")
	if err != nil {
		return "", Identity{}, err
	}

	st := &State{
		ID:        id,
		TenantID:  tenantID,
		PagePath:  pagePath,
		Client:    c,
		StartedAt: s.now(),
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.byToken[token] = &entry{state: st, expiresAt: s.now().Add(s.ttl)}
	return token, st.identity(), nil
}

// identity copies the fixed fields out of a live State.
func (st *State) identity() Identity {
	return Identity{
		ID:        st.ID,
		TenantID:  st.TenantID,
		PagePath:  st.PagePath,
		StartedAt: st.StartedAt,
	}
}

// With runs fn against the session behind token, holding the store lock.
//
// Access is serialized rather than handing out the pointer because the
// feature accumulator is not safe for concurrent mutation and telemetry
// batches for one session can land on several connections at once.
func (s *Store) With(token string, fn func(*State)) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	e, ok := s.byToken[token]
	if !ok {
		return ErrNotFound
	}
	if s.now().After(e.expiresAt) {
		delete(s.byToken, token)
		return ErrNotFound
	}
	fn(e.state)
	return nil
}

// Len reports the number of live sessions.
func (s *Store) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.byToken)
}

// Sweep drops expired sessions and returns how many were removed.
//
// Without this the map is a leak: sessions are created on every page
// load and nothing else deletes them.
func (s *Store) Sweep() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	n := 0
	for tok, e := range s.byToken {
		if now.After(e.expiresAt) {
			delete(s.byToken, tok)
			n++
		}
	}
	return n
}

// NewID mints a prefixed identifier. The implementation is libs/id,
// which the decision engine also mints from: session tokens, session ids
// and evaluation ids are one identifier space, and it stopped being
// possible to keep that true inside one package when a second service
// started minting.
func NewID(prefix string) (string, error) { return id.New(prefix) }
