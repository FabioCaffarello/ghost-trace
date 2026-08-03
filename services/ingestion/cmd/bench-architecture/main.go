// Command bench-architecture measures maintained state against
// compute-on-call across a concurrency × duration grid.
//
// This is M5's sixth number and the project's central claim. The claim
// is NOT that decisions are fast — docs/architecture.md §8.1 records
// that per-request arithmetic is cheap either way, and M1 measured a
// p99 of 6.4ms against an 80ms budget. The claim is that:
//
//	maintained state  costs O(sessions)
//	compute-on-call   costs O(sessions × duration)
//
// so the two diverge on the DIAGONAL, and the divergence is in memory
// and GC pressure rather than in single-request CPU.
//
// The prediction is written down before the run (see the plan): no
// difference at low N and short sessions; growing separation along the
// diagonal; compute-on-call failing outright at the top right. If that
// does not appear, the architecture is wrong and the write-up says so.
//
// HTTP is deliberately excluded. Both architectures would sit behind the
// same handler, so including it would add noise to every cell without
// changing any comparison. What is measured is state management, which
// is the variable under test.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"sort"
	"time"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/feature"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/policy"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/session"
)

type cell struct {
	Architecture string  `json:"architecture"`
	Sessions     int     `json:"sessions"`
	DurationS    int     `json:"duration_s"`
	Batches      int     `json:"batches_per_session"`
	HeapMB       float64 `json:"heap_mb"`
	BytesPerSess int64   `json:"bytes_per_session"`
	P50Ms        float64 `json:"decide_p50_ms"`
	P99Ms        float64 `json:"decide_p99_ms"`
	MaxMs        float64 `json:"decide_max_ms"`
}

func main() {
	var (
		concurrency = flag.String("sessions", "100,1000,10000", "comma-separated session counts")
		durations   = flag.String("durations", "10,300,1800", "comma-separated session durations in seconds")
		sample      = flag.Int("sample", 200, "decisions sampled per cell for latency")
		out         = flag.String("out", "", "write JSON results to this path")
	)
	flag.Parse()

	sessionCounts := parseInts(*concurrency)
	durationList := parseInts(*durations)

	fmt.Printf("%-14s %9s %9s %10s %12s %9s %9s %9s\n",
		"architecture", "sessions", "duration", "heap MB", "bytes/sess", "p50 ms", "p99 ms", "max ms")
	fmt.Println(repeat('-', 90))

	var results []cell
	for _, dur := range durationList {
		for _, n := range sessionCounts {
			for _, arch := range []string{"maintained", "on-call"} {
				c := runCell(arch, n, dur, *sample)
				results = append(results, c)
				fmt.Printf("%-14s %9d %8ds %10.1f %12d %9.3f %9.3f %9.3f\n",
					c.Architecture, c.Sessions, c.DurationS, c.HeapMB,
					c.BytesPerSess, c.P50Ms, c.P99Ms, c.MaxMs)
			}
		}
		fmt.Println()
	}

	if *out != "" {
		f, err := os.Create(*out)
		if err != nil {
			fmt.Fprintf(os.Stderr, "write results: %v\n", err)
			os.Exit(1)
		}
		defer func() { _ = f.Close() }()
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(results); err != nil {
			fmt.Fprintf(os.Stderr, "encode results: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("wrote %s\n", *out)
	}
}

// batchesFor converts a session duration into a batch count under the
// M1 collection policy: 20Hz pointer until the decision, 5Hz after
// (§8.5), batched every 2s.
func batchesFor(durationS int) int {
	b := durationS / 2
	if b < 1 {
		b = 1
	}
	return b
}

// pointsPerBatch models the collect policy: the first batches carry
// 20Hz sampling, later ones 5Hz once the decision has passed.
func pointsPerBatch(batchIndex int) int {
	if batchIndex < 5 {
		return 40
	}
	return 10
}

func runCell(arch string, sessions, durationS, sample int) cell {
	batches := batchesFor(durationS)
	rng := rand.New(rand.NewSource(int64(sessions*1000 + durationS)))

	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	var latencies []float64

	// The store must outlive the measurement below. Without this it
	// becomes unreachable at the end of the switch and the GC collects
	// it before ReadMemStats runs — which reports every cell at the same
	// baseline heap and makes the whole comparison vanish.
	var keepalive any

	switch arch {
	case "maintained":
		store := session.NewStore(time.Hour, time.Now)
		tokens := make([]string, 0, sessions)
		for i := 0; i < sessions; i++ {
			tok, _, err := store.Create("t_bench", "/login", session.Client{})
			if err != nil {
				panic(err)
			}
			tokens = append(tokens, tok)
		}
		for b := 0; b < batches; b++ {
			pts := makePoints(pointsPerBatch(b), rng)
			startMs := uint32(1200 + b*2000)
			for _, tok := range tokens {
				_ = store.With(tok, func(st *session.State) {
					st.Pointer.Add(startMs, pts)
					st.Keystroke.AddKey(startMs+10, "down", "alpha", "f_1")
					st.Keystroke.AddKey(startMs+50, "up", "alpha", "f_1")
					st.LastEventMs = startMs
				})
			}
		}
		keepalive = store
		latencies = sampleLatency(sample, len(tokens), func(i int) {
			_ = store.With(tokens[i], func(st *session.State) {
				s := policy.State{
					Pointer:     st.Pointer.State(),
					Keystroke:   st.Keystroke.State(),
					Interaction: st.Interaction.State(),
				}
				_ = policy.Judge(s, "login")
			})
		})

	case "on-call":
		store := session.NewOnCallStore(time.Hour, time.Now)
		tokens := make([]string, 0, sessions)
		for i := 0; i < sessions; i++ {
			tok, _, err := store.Create("t_bench", "/login", session.Client{})
			if err != nil {
				panic(err)
			}
			tokens = append(tokens, tok)
		}
		for b := 0; b < batches; b++ {
			pts := makePoints(pointsPerBatch(b), rng)
			startMs := uint32(1200 + b*2000)
			for _, tok := range tokens {
				_ = store.With(tok, func(st *session.OnCallState) {
					st.AppendPointer(startMs, pts)
					st.AppendKey(startMs+10, "down", "alpha", "f_1")
					st.AppendKey(startMs+50, "up", "alpha", "f_1")
					st.LastEventMs = startMs
				})
			}
		}
		keepalive = store
		latencies = sampleLatency(sample, len(tokens), func(i int) {
			_ = store.With(tokens[i], func(st *session.OnCallState) {
				p, k, in := st.Compute()
				_ = policy.Judge(policy.State{Pointer: p, Keystroke: k, Interaction: in}, "login")
			})
		})
	}

	runtime.GC()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	runtime.KeepAlive(keepalive)

	heap := float64(after.HeapAlloc) / (1 << 20)
	var perSession int64
	if sessions > 0 && after.HeapAlloc > before.HeapAlloc {
		perSession = int64(after.HeapAlloc-before.HeapAlloc) / int64(sessions)
	}

	sort.Float64s(latencies)
	return cell{
		Architecture: arch,
		Sessions:     sessions,
		DurationS:    durationS,
		Batches:      batches,
		HeapMB:       heap,
		BytesPerSess: perSession,
		P50Ms:        pct(latencies, 0.50),
		P99Ms:        pct(latencies, 0.99),
		MaxMs:        pct(latencies, 1.0),
	}
}

func sampleLatency(sample, n int, decide func(int)) []float64 {
	if n == 0 {
		return nil
	}
	if sample > n {
		sample = n
	}
	out := make([]float64, 0, sample)
	step := n / sample
	if step < 1 {
		step = 1
	}
	for i := 0; i < n && len(out) < sample; i += step {
		start := time.Now()
		decide(i)
		out = append(out, float64(time.Since(start).Nanoseconds())/1e6)
	}
	return out
}

func makePoints(n int, rng *rand.Rand) []feature.Point {
	pts := make([]feature.Point, 0, n)
	for i := 0; i < n; i++ {
		dt := uint32(50)
		if i == 0 {
			dt = 0
		}
		pts = append(pts, feature.Point{
			X:    int32(100 + i*8 + rng.Intn(3)),
			Y:    int32(120 + i*3 + rng.Intn(3)),
			DtMs: dt,
		})
	}
	return pts
}

func pct(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	i := int(float64(len(sorted)-1) * p)
	return sorted[i]
}

func parseInts(s string) []int {
	var out []int
	cur := 0
	has := false
	for _, r := range s {
		if r >= '0' && r <= '9' {
			cur = cur*10 + int(r-'0')
			has = true
			continue
		}
		if has {
			out = append(out, cur)
		}
		cur, has = 0, false
	}
	if has {
		out = append(out, cur)
	}
	return out
}

func repeat(r rune, n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = r
	}
	return string(b)
}
