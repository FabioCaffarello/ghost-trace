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
	"crypto/rand"
	"encoding/base64"
	"errors"
	"sync"
	"time"

	"github.com/FabioCaffarello/ghost-trace/libs/feature"
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

// Create issues a new session and returns its token.
//
// The token and the session id are distinct values. The token is a
// bearer credential the browser holds; the id is what the archive
// records. Reusing one as the other would put a live credential into
// storage that outlives it by 7 days.
func (s *Store) Create(tenantID, pagePath string, c Client) (token string, st *State, err error) {
	token, err = NewID("st_")
	if err != nil {
		return "", nil, err
	}
	id, err := NewID("s_")
	if err != nil {
		return "", nil, err
	}

	st = &State{
		ID:        id,
		TenantID:  tenantID,
		PagePath:  pagePath,
		Client:    c,
		StartedAt: s.now(),
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.byToken[token] = &entry{state: st, expiresAt: s.now().Add(s.ttl)}
	return token, st, nil
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

// NewID returns a prefixed, URL-safe, 144-bit random identifier. It is
// the single implementation for every identifier the system mints —
// session tokens, session ids, evaluation ids — so entropy and format
// decisions live in one place.
func NewID(prefix string) (string, error) {
	b := make([]byte, 18)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return prefix + base64.RawURLEncoding.EncodeToString(b), nil
}
