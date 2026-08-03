package session

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/feature"
)

func TestCreateIssuesDistinctTokenAndID(t *testing.T) {
	// The token is a bearer credential the browser holds; the id is what
	// the archive records and outlives it by 7 days. Reusing one as the
	// other would put a live credential into storage.
	s := NewStore(time.Minute, time.Now)
	tok, st, err := s.Create("t_demo", "/login", Client{})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if tok == st.ID {
		t.Error("token and session id are the same value")
	}
	if tok == "" || st.ID == "" {
		t.Fatal("empty token or id")
	}
}

func TestTokensAreUnique(t *testing.T) {
	s := NewStore(time.Minute, time.Now)
	seen := make(map[string]bool)
	for i := 0; i < 500; i++ {
		tok, st, err := s.Create("t", "/", Client{})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if seen[tok] || seen[st.ID] {
			t.Fatal("duplicate token or id issued")
		}
		seen[tok], seen[st.ID] = true, true
	}
}

func TestWithUnknownTokenReturnsNotFound(t *testing.T) {
	s := NewStore(time.Minute, time.Now)
	err := s.With("st_nope", func(*State) { t.Error("callback ran for unknown token") })
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestExpiredTokenIsRejectedAndDropped(t *testing.T) {
	now := time.Now()
	clock := func() time.Time { return now }
	s := NewStore(time.Minute, clock)

	tok, _, err := s.Create("t", "/", Client{})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	now = now.Add(2 * time.Minute)
	if err := s.With(tok, func(*State) {}); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound after expiry", err)
	}
	if s.Len() != 0 {
		t.Errorf("Len = %d, want 0; the expired entry should be dropped on access", s.Len())
	}
}

func TestSweepRemovesOnlyExpired(t *testing.T) {
	// Without a sweep the map grows for the life of the process:
	// sessions are created on every page load and nothing else deletes
	// them.
	now := time.Now()
	clock := func() time.Time { return now }
	s := NewStore(time.Minute, clock)

	old1, _, _ := s.Create("t", "/", Client{})
	old2, _, _ := s.Create("t", "/", Client{})

	now = now.Add(90 * time.Second)
	fresh, _, _ := s.Create("t", "/", Client{})

	if n := s.Sweep(); n != 2 {
		t.Errorf("Sweep removed %d, want 2", n)
	}
	if err := s.With(fresh, func(*State) {}); err != nil {
		t.Errorf("fresh session was swept: %v", err)
	}
	for _, tok := range []string{old1, old2} {
		if err := s.With(tok, func(*State) {}); !errors.Is(err, ErrNotFound) {
			t.Errorf("expired session survived the sweep")
		}
	}
}

func TestWithSerializesConcurrentMutation(t *testing.T) {
	// Telemetry batches for one session can land on several connections
	// at once, and the feature accumulator is not safe for concurrent
	// mutation. Run with -race.
	s := NewStore(time.Minute, time.Now)
	tok, _, _ := s.Create("t", "/", Client{})

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = s.With(tok, func(st *State) {
				st.BatchesSeen++
				st.Pointer.Add(0, []feature.Point{{X: 1, Y: 1}})
			})
		}()
	}
	wg.Wait()

	_ = s.With(tok, func(st *State) {
		if st.BatchesSeen != 50 {
			t.Errorf("BatchesSeen = %d, want 50", st.BatchesSeen)
		}
	})
}

func TestHighestSeqIsAHighWaterMark(t *testing.T) {
	// Batches are accepted out of order (contract §2), so seq tracking
	// must not assume arrival order.
	s := NewStore(time.Minute, time.Now)
	tok, _, _ := s.Create("t", "/", Client{})

	for _, seq := range []uint32{0, 3, 1, 7, 2} {
		_ = s.With(tok, func(st *State) {
			st.ObserveBatch(seq, 0)
		})
	}

	_ = s.With(tok, func(st *State) {
		if st.HighestSeq != 7 {
			t.Errorf("HighestSeq = %d, want 7", st.HighestSeq)
		}
		// 5 batches with a high-water mark of 7 means gaps, and gaps
		// are meaningful.
		if st.BatchesSeen != 5 {
			t.Errorf("BatchesSeen = %d, want 5", st.BatchesSeen)
		}
	})
}

func TestObserveBatchAndEventTime(t *testing.T) {
	var st State

	// Envelope high-water marks: out-of-order sent_at must not regress.
	st.ObserveBatch(2, 5000)
	st.ObserveBatch(1, 3000)
	if st.BatchesSeen != 2 || st.HighestSeq != 2 || st.LastEventMs != 5000 {
		t.Errorf("after batches: %+v", st)
	}

	// Event times advance duration monotonically.
	st.ObserveEventTime(4000) // older than 5000: no regress
	st.ObserveEventTime(6200)
	if st.LastEventMs != 6200 {
		t.Errorf("LastEventMs = %d, want 6200", st.LastEventMs)
	}
}

func TestNewIDShape(t *testing.T) {
	a, err := NewID("ev_")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := NewID("ev_")
	if a == b {
		t.Error("two ids collided")
	}
	if len(a) != 3+24 { // prefix + base64url(18 bytes)
		t.Errorf("id length = %d: %q", len(a), a)
	}
}
