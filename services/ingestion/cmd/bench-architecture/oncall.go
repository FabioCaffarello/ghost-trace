package main

import (
	"sync"
	"time"

	"github.com/FabioCaffarello/ghost-trace/libs/feature"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/session"
)

// OnCallStore is the alternative architecture: retain raw events and
// compute features when the decision arrives.
//
// It exists to be measured against session.Store, not to be used —
// which is why it lives in the benchmark binary rather than in
// internal/session, where it used to link into the production build. The whole
// project rests on the claim that maintained state is necessary, and a
// claim with no measured alternative is an assertion. Contract §8.1
// predicts where they diverge:
//
//	maintained state  O(sessions)              — a fixed accumulator
//	compute-on-call   O(sessions × duration)   — a growing event buffer
//
// If that divergence does not appear on the concurrency × duration
// diagonal, the architecture is wrong and the write-up says so.
//
// The comparison is deliberately fair. This is not a strawman: it uses
// the same feature extractors, the same policy, and the same locking
// discipline. The only difference is WHEN the arithmetic happens, which
// is the variable under test.
type OnCallStore struct {
	mu      sync.Mutex
	byToken map[string]*OnCallState
	expires map[string]time.Time
	ttl     time.Duration
	now     func() time.Time
}

// OnCallState retains the raw event stream for one session.
type OnCallState struct {
	ID        string
	TenantID  string
	PagePath  string
	Client    session.Client
	StartedAt time.Time

	HighestSeq  uint32
	BatchesSeen uint32
	LastEventMs uint32

	// The retained history. This is the entire difference between the
	// two architectures: it cannot be bounded without deciding in
	// advance which features will be wanted, which is exactly what
	// compute-on-call declines to do.
	polylines []polyline
	keys      []keyEvent
	scrolls   []string
	focuses   []string
	forms     []formEvent
}

type polyline struct {
	startMs uint32
	pts     []feature.Point
}

type keyEvent struct {
	tMs                  uint32
	phase, class, target string
}

type formEvent struct {
	action, target string
}

// NewOnCallStore constructs the comparison store.
func NewOnCallStore(ttl time.Duration, now func() time.Time) *OnCallStore {
	if now == nil {
		now = time.Now
	}
	return &OnCallStore{
		byToken: make(map[string]*OnCallState),
		expires: make(map[string]time.Time),
		ttl:     ttl,
		now:     now,
	}
}

// Create issues a session.
func (s *OnCallStore) Create(tenantID, pagePath string, c session.Client) (string, *OnCallState, error) {
	token, err := session.NewID("st_")
	if err != nil {
		return "", nil, err
	}
	id, err := session.NewID("s_")
	if err != nil {
		return "", nil, err
	}
	st := &OnCallState{
		ID: id, TenantID: tenantID, PagePath: pagePath,
		Client: c, StartedAt: s.now(),
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byToken[token] = st
	s.expires[token] = s.now().Add(s.ttl)
	return token, st, nil
}

// With runs fn against the retained state.
func (s *OnCallStore) With(token string, fn func(*OnCallState)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, ok := s.byToken[token]
	if !ok {
		return session.ErrNotFound
	}
	if exp, ok := s.expires[token]; ok && s.now().After(exp) {
		delete(s.byToken, token)
		delete(s.expires, token)
		return session.ErrNotFound
	}
	fn(st)
	return nil
}

// AppendPointer retains a polyline.
func (st *OnCallState) AppendPointer(startMs uint32, pts []feature.Point) {
	cp := make([]feature.Point, len(pts))
	copy(cp, pts)
	st.polylines = append(st.polylines, polyline{startMs: startMs, pts: cp})
}

// AppendKey retains a key event.
func (st *OnCallState) AppendKey(tMs uint32, phase, class, target string) {
	st.keys = append(st.keys, keyEvent{tMs, phase, class, target})
}

// AppendScroll retains a scroll mode.
func (st *OnCallState) AppendScroll(mode string) { st.scrolls = append(st.scrolls, mode) }

// AppendFocus retains a focus target.
func (st *OnCallState) AppendFocus(target string) { st.focuses = append(st.focuses, target) }

// AppendForm retains a form action.
func (st *OnCallState) AppendForm(action, target string) {
	st.forms = append(st.forms, formEvent{action, target})
}

// Compute replays the whole retained history through the feature
// extractors. This is the work maintained state does incrementally and
// this architecture defers to the decision path — the p99 the contract
// budgets for.
func (st *OnCallState) Compute() (feature.PointerState, feature.KeystrokeState, feature.InteractionState) {
	var p feature.Pointer
	for _, pl := range st.polylines {
		p.Add(pl.startMs, pl.pts)
	}
	var k feature.Keystroke
	for _, ke := range st.keys {
		k.AddKey(ke.tMs, ke.phase, ke.class, ke.target)
	}
	var in feature.Interaction
	for _, m := range st.scrolls {
		in.AddScroll(m)
	}
	for _, f := range st.focuses {
		in.AddFocus(f)
	}
	for _, f := range st.forms {
		in.AddForm(f.action, f.target)
	}
	return p.State(), k.State(), in.State()
}

// Len reports live sessions.
func (s *OnCallStore) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.byToken)
}
