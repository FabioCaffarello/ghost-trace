package session_test

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/FabioCaffarello/ghost-trace/libs/feature"
	"github.com/FabioCaffarello/ghost-trace/services/collector/internal/session"
)

// The measurement `session.Store` asked for in its own comment:
//
//	One mutex guards the whole map. At M1 volume that is correct and
//	boring; sharding it before there is a measurement showing contention
//	would be optimising against a guess.
//
// This is that measurement, and it is deliberately a TEST rather than
// only a benchmark. A benchmark reports a number nobody reads until
// somebody goes looking; an assertion fails when the property changes.
//
// What is being measured is not how fast the store is. It is whether
// throughput responds to CORES AT ALL — because `Store.With` holds one
// process-global mutex across its whole callback, and in the real
// telemetry path that callback performs the full per-event feature
// update for every event in a batch. If that is the bound, adding cores
// buys nothing and every capacity plan drawn from a single-session
// latency figure is wrong in a direction nobody is watching.

// telemetryBatch is what ingest_telemetry does inside the lock: one
// batch observation and then per-event feature updates. Kept in step
// with the real call sites by shape rather than by import, because the
// point is to measure the WORK DONE UNDER THE LOCK.
func telemetryBatch(st *session.State, events int) {
	st.ObserveBatch(1, 900)
	pts := []feature.Point{{X: 300, Y: 200, DtMs: 10}, {X: 316, Y: 214, DtMs: 14}}
	for i := range events {
		t := uint32(100 + i*17)
		st.ObserveEventTime(t)
		if i%2 == 0 {
			st.Pointer.Add(t, pts)
			continue
		}
		st.Keystroke.AddKey(t, "down", "alpha", "f")
	}
	_ = st.Pointer.State()
	_ = st.Keystroke.State()
	_ = st.Interaction.State()
}

// throughput drives the store from `workers` goroutines for `d` and
// returns completed batches per second.
func throughput(store *session.Store, tokens []string, workers, events int, d time.Duration) float64 {
	var wg sync.WaitGroup
	var mu sync.Mutex
	total := 0
	deadline := time.Now().Add(d)

	for w := range workers {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			n := 0
			i := w
			for time.Now().Before(deadline) {
				// Each worker walks its own slice of the token space, so
				// this is not several goroutines fighting over one map
				// ENTRY — it is many sessions, which is the real shape.
				_ = store.With(tokens[i%len(tokens)], func(st *session.State) {
					telemetryBatch(st, events)
				})
				i += workers
				n++
			}
			mu.Lock()
			total += n
			mu.Unlock()
		}(w)
	}
	wg.Wait()
	return float64(total) / d.Seconds()
}

func newStore(t *testing.T, sessions int) (*session.Store, []string) {
	t.Helper()
	store := session.NewStore(30*time.Minute, time.Now)
	tokens := make([]string, sessions)
	for i := range tokens {
		tok, _, err := store.Create("t_demo", "/login", session.Client{PointerType: "fine"})
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		tokens[i] = tok
	}
	return store, tokens
}

func TestTelemetryIngestDoesNotScaleWithCores(t *testing.T) {
	// THE prediction, on record in contract/roadmap.md before it was
	// taken: telemetry throughput is bounded by the work done under one
	// global mutex, so it should barely improve as workers are added.
	//
	// The assertion is deliberately generous. It does not claim a
	// precise scaling factor — that would be a machine-dependent number
	// dressed as a property. It claims the SHAPE: eight workers do not
	// get anywhere near eight times the work of one.
	if testing.Short() {
		t.Skip("timing measurement")
	}
	if runtime.NumCPU() < 4 {
		t.Skipf("needs at least 4 cores, this machine has %d", runtime.NumCPU())
	}

	store, tokens := newStore(t, 512)
	const events = 8
	const per = 400 * time.Millisecond

	// Warm: the first pass pays for map growth and CPU frequency ramp,
	// and charging that to the one-worker reading would manufacture the
	// very scaling curve this test is looking for.
	throughput(store, tokens, 1, events, 200*time.Millisecond)

	one := throughput(store, tokens, 1, events, per)
	eight := throughput(store, tokens, 8, events, per)
	speedup := eight / one

	t.Logf("1 worker: %8.0f batches/s", one)
	t.Logf("8 workers:%8.0f batches/s  (speedup %.2fx on %d cores)",
		eight, speedup, runtime.NumCPU())

	if speedup > 3 {
		t.Errorf("eight workers achieved %.2fx the throughput of one. The global "+
			"mutex in session.Store is no longer the bound, which means either the "+
			"store was changed or this test stopped measuring what it claims. "+
			"Re-read the hypothesis in contract/roadmap.md §4.2 before deleting "+
			"this assertion", speedup)
	}
}

func TestTheLockIsHeldForTheFeatureWorkAndNotJustTheMapLookup(t *testing.T) {
	// WHY it does not scale, which is a different claim from THAT it
	// does not. A mutex protecting a map lookup would be held for
	// nanoseconds and would contend only at implausible rates. This one
	// is held for the entire per-event feature update.
	//
	// Measured by comparing a callback that does the real work against
	// one that does nothing at all: the ratio is how much of the
	// critical section is the store's own business.
	if testing.Short() {
		t.Skip("timing measurement")
	}
	store, tokens := newStore(t, 512)
	const per = 300 * time.Millisecond

	throughput(store, tokens, 1, 8, 150*time.Millisecond)

	empty := throughputWith(store, tokens, per, func(*session.State) {})
	real8 := throughputWith(store, tokens, per, func(st *session.State) {
		telemetryBatch(st, 8)
	})

	t.Logf("lookup only:        %9.0f ops/s", empty)
	t.Logf("lookup + 8 events:  %9.0f ops/s  (%.0fx slower)", real8, empty/real8)

	// The bound is loose on purpose. This runs on shared CI hardware
	// where a tight timing assertion is a flake generator rather than a
	// gate; 2.5x still fails if the feature work leaves the critical
	// section, which is the only change worth failing for.
	if empty/real8 < 2.5 {
		t.Errorf("the real callback is only %.1fx slower than an empty one, so the "+
			"critical section is dominated by the map lookup rather than by feature "+
			"work. That contradicts the stated diagnosis and the fix implied by it",
			empty/real8)
	}
}

func throughputWith(store *session.Store, tokens []string, d time.Duration, fn func(*session.State)) float64 {
	deadline := time.Now().Add(d)
	n, i := 0, 0
	for time.Now().Before(deadline) {
		_ = store.With(tokens[i%len(tokens)], fn)
		i++
		n++
	}
	return float64(n) / d.Seconds()
}

func TestBatchSizeMovesThroughputProportionally(t *testing.T) {
	// The consequence that matters operationally. If the lock is held
	// for per-event work, then a client sending bigger batches does not
	// merely do more work — it holds the whole collector for longer.
	// Batch size becomes a lever any single client can pull on every
	// other client's latency.
	if testing.Short() {
		t.Skip("timing measurement")
	}
	store, tokens := newStore(t, 512)
	const per = 300 * time.Millisecond
	throughput(store, tokens, 1, 8, 150*time.Millisecond)

	small := throughput(store, tokens, 4, 4, per)
	large := throughput(store, tokens, 4, 64, per)

	t.Logf("4 events/batch:  %8.0f batches/s", small)
	t.Logf("64 events/batch: %8.0f batches/s", large)

	if large >= small {
		t.Errorf("a 16x larger batch did not reduce batch throughput (%.0f vs %.0f); "+
			"per-event work is not dominating and the diagnosis needs revisiting",
			large, small)
	}
}

// BenchmarkStoreWith exists so `go test -bench` produces the curve
// rather than only the assertion. Run it as:
//
//	go test ./internal/session -bench StoreWith -cpu 1,2,4,8 -benchtime 300ms
func BenchmarkStoreWith(b *testing.B) {
	for _, events := range []int{0, 8, 32} {
		b.Run(fmt.Sprintf("events=%d", events), func(b *testing.B) {
			store := session.NewStore(30*time.Minute, time.Now)
			tokens := make([]string, 512)
			for i := range tokens {
				tok, _, err := store.Create("t_demo", "/login",
					session.Client{PointerType: "fine"})
				if err != nil {
					b.Fatal(err)
				}
				tokens[i] = tok
			}

			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					_ = store.With(tokens[i%len(tokens)], func(st *session.State) {
						if events > 0 {
							telemetryBatch(st, events)
						}
					})
					i++
				}
			})
		})
	}
}
