// Package loadgen issues requests on a SCHEDULE rather than in a loop.
//
// THE ENTIRE REASON THIS EXISTS. The obvious way to write a load
// generator is a loop: send a request, wait for the response, send the
// next. That generator has a property nobody wants and almost nobody
// notices — when the server slows down, the generator slows down with
// it, so the server stops receiving the load that would have revealed
// the problem, and the slow period contributes FEWER samples to the
// percentiles rather than more.
//
// The effect has a name, coordinated omission, and its signature is a
// latency report that looks better the worse the server behaves. A
// server that stalls for one second while a closed-loop generator waits
// records exactly one slow request. The hundred requests that would have
// arrived during the stall were never sent, so they never appear, and a
// p99 computed over what remains describes a system that was not under
// test.
//
// For a phase whose entire purpose is to replace an idle-system floor
// with a number that survives contention, using a generator that hides
// contention would not merely fail — it would produce a confident,
// publishable, wrong answer. That is the specific failure this
// repository exists to avoid.
//
// WHAT THIS DOES INSTEAD. Arrivals are scheduled in advance at a fixed
// rate. A request's latency is measured from the moment it was DUE, not
// from the moment a worker got round to it. If the driver itself falls
// behind, that shows up as latency rather than disappearing — which is
// correct, because a user whose request waits in a queue experienced the
// wait regardless of which queue it was.
//
// Both readings are reported, because the DIFFERENCE between them is
// the measurement:
//
//	Service   — end - actual start.   What a closed-loop driver reports.
//	Response  — end - intended start. What the load actually experienced.
//
// When the two agree, the system kept up. When Response is far above
// Service, requests were queuing, and a report showing only Service
// would have hidden exactly that.
package loadgen

import (
	"context"
	"math"
	"sort"
	"sync"
	"time"
)

// Result is one attempted request.
type Result struct {
	// Intended is when the schedule said this request was due.
	Intended time.Time

	// Started is when a worker actually began it. Later than Intended
	// whenever every worker was busy, which is the condition worth
	// seeing rather than smoothing away.
	Started time.Time

	// Ended is when the response completed, or the attempt failed.
	Ended time.Time

	// Status is the HTTP status, or 0 when nothing came back.
	Status int

	// Err is why nothing came back. A failed request is NOT dropped from
	// the report: an error under load is a result, and excluding errors
	// is how a saturated system comes to look fast.
	Err error
}

// Service is what a closed-loop driver would call the latency.
func (r Result) Service() time.Duration { return r.Ended.Sub(r.Started) }

// Response is the latency the scheduled load actually experienced.
func (r Result) Response() time.Duration { return r.Ended.Sub(r.Intended) }

// Deficit is how long this request waited for the driver rather than
// for the server. Non-zero means the generator itself was the queue.
func (r Result) Deficit() time.Duration { return r.Started.Sub(r.Intended) }

// Config describes a run.
type Config struct {
	// Rate is requests per second, as a schedule rather than a target.
	Rate float64

	// Duration is how long to keep issuing.
	Duration time.Duration

	// Workers bounds concurrent in-flight requests.
	//
	// A bound is necessary — an unbounded generator answers a slow
	// server by allocating goroutines until it dies, and measures its
	// own death. But the bound is also a lie waiting to happen, because
	// once every worker is busy the driver stops being open-loop. That
	// is why Deficit is recorded and why Report refuses to be read
	// without it: a run whose deficit is large did not measure the
	// server, it measured this number.
	Workers int
}

// Do performs one request. It returns the status, or an error.
type Do func(ctx context.Context) (int, error)

// Run issues requests on schedule and returns every attempt.
//
// The schedule is computed from the start time and never from the
// previous request's completion, which is the whole difference between
// this and a loop.
func Run(ctx context.Context, cfg Config, do Do) []Result {
	if cfg.Rate <= 0 || cfg.Duration <= 0 {
		return nil
	}
	workers := cfg.Workers
	if workers <= 0 {
		workers = 1
	}

	interval := time.Duration(float64(time.Second) / cfg.Rate)
	if interval <= 0 {
		interval = time.Nanosecond
	}
	total := int(cfg.Duration / interval)
	if total <= 0 {
		return nil
	}

	start := time.Now()
	results := make([]Result, total)

	// A counting semaphore rather than a worker pool with a queue. The
	// difference matters: a queue would let the driver accept work it
	// cannot start and report the wait as if it were service time. Here
	// a request waits explicitly, holding its intended time, and the
	// wait is recorded.
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup

	for i := range total {
		intended := start.Add(time.Duration(i) * interval)

		if wait := time.Until(intended); wait > 0 {
			select {
			case <-time.After(wait):
			case <-ctx.Done():
				return results[:i]
			}
		}

		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			return results[:i]
		}

		wg.Add(1)
		go func(i int, intended time.Time) {
			defer wg.Done()
			defer func() { <-sem }()

			began := time.Now()
			status, err := do(ctx)
			results[i] = Result{
				Intended: intended,
				Started:  began,
				Ended:    time.Now(),
				Status:   status,
				Err:      err,
			}
		}(i, intended)
	}

	wg.Wait()
	return results
}

// Report is a run reduced to numbers.
type Report struct {
	Requests int `json:"requests"`
	OK       int `json:"ok"`
	Failed   int `json:"failed"`

	// ServiceP50/P99 are what a closed-loop driver would publish.
	ServiceP50 float64 `json:"service_p50_ms"`
	ServiceP99 float64 `json:"service_p99_ms"`

	// ResponseP50/P99 include the time a request spent waiting to be
	// issued. This is the honest reading.
	ResponseP50 float64 `json:"response_p50_ms"`
	ResponseP99 float64 `json:"response_p99_ms"`

	// DeficitP99 is how far behind its own schedule the driver fell.
	//
	// Read this FIRST. A large deficit means the generator saturated
	// before the server did, so ResponseP99 describes the driver's
	// worker bound and not the system under test. Omitted, a reader has
	// no way to tell that from a genuinely overloaded server.
	DeficitP99 float64 `json:"deficit_p99_ms"`

	// Omission is ResponseP99 minus ServiceP99: what a closed-loop
	// driver would not have seen.
	Omission float64 `json:"coordinated_omission_ms"`

	// Achieved is requests actually issued per second, against Intended.
	Achieved float64 `json:"achieved_rps"`
	Intended float64 `json:"intended_rps"`
}

// Summarise reduces results to a Report.
//
// Failed requests are counted and their latencies INCLUDED. An error
// that took four seconds to arrive cost four seconds; excluding it
// because it was not a 200 is how a collapsing system produces its best
// ever latency report.
func Summarise(results []Result, cfg Config) Report {
	rep := Report{Requests: len(results), Intended: cfg.Rate}
	if len(results) == 0 {
		return rep
	}

	service := make([]float64, 0, len(results))
	response := make([]float64, 0, len(results))
	deficit := make([]float64, 0, len(results))

	var first, last time.Time
	for _, r := range results {
		if r.Intended.IsZero() {
			// Never issued: the run ended before its turn. Not a
			// failure, and counting it as one would punish a short run.
			rep.Requests--
			continue
		}
		if r.Err != nil || r.Status == 0 || r.Status >= 500 {
			rep.Failed++
		} else {
			rep.OK++
		}
		service = append(service, ms(r.Service()))
		response = append(response, ms(r.Response()))
		deficit = append(deficit, ms(r.Deficit()))

		if first.IsZero() || r.Started.Before(first) {
			first = r.Started
		}
		if r.Ended.After(last) {
			last = r.Ended
		}
	}
	if len(service) == 0 {
		return rep
	}

	rep.ServiceP50, rep.ServiceP99 = pct(service, 50), pct(service, 99)
	rep.ResponseP50, rep.ResponseP99 = pct(response, 50), pct(response, 99)
	rep.DeficitP99 = pct(deficit, 99)
	rep.Omission = rep.ResponseP99 - rep.ServiceP99

	if elapsed := last.Sub(first).Seconds(); elapsed > 0 {
		rep.Achieved = float64(len(service)) / elapsed
	}
	return rep
}

func ms(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }

// pct is the nearest-rank percentile.
//
// Nearest-rank rather than interpolated on purpose: an interpolated p99
// invents a value between two measurements, and every number this
// repository publishes is supposed to be one that was observed.
func pct(xs []float64, p float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	rank := int(math.Ceil(p / 100 * float64(len(s))))
	if rank < 1 {
		rank = 1
	}
	if rank > len(s) {
		rank = len(s)
	}
	return s[rank-1]
}
