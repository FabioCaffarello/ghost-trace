package loadgen_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/FabioCaffarello/ghost-trace/tools/loadgen"
)

// closedLoop is the generator this package exists to not be: send,
// wait, send again. It is here so the claim about coordinated omission
// is demonstrated against a real alternative rather than asserted.
func closedLoop(ctx context.Context, d time.Duration, do loadgen.Do) []loadgen.Result {
	deadline := time.Now().Add(d)
	var out []loadgen.Result
	for time.Now().Before(deadline) {
		began := time.Now()
		status, err := do(ctx)
		now := time.Now()
		// Intended == Started is precisely the lie: this generator has
		// no schedule, so a request is by definition never late.
		out = append(out, loadgen.Result{
			Intended: began, Started: began, Ended: now, Status: status, Err: err,
		})
	}
	return out
}

// stallingServer holds a GLOBAL lock every `every` requests, so a stall
// blocks the whole server rather than one request.
//
// The distinction is the whole demonstration. A server where one request
// in twenty is individually slow does not produce coordinated omission
// at all — with enough workers every request is independent and nothing
// queues. Coordinated omission needs a stall that stops EVERYTHING: a
// global lock, a stop-the-world pause, a failover. That is also the
// shape of the thing this phase is hunting, since `session.Store` is one
// mutex on the path of every request.
func stallingServer(t *testing.T, every int, stall time.Duration) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	var n atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		if n.Add(1)%int64(every) == 0 {
			time.Sleep(stall)
		}
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func get(url string) loadgen.Do {
	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: &http.Transport{MaxIdleConnsPerHost: 256, MaxConnsPerHost: 0},
	}
	return func(ctx context.Context) (int, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return 0, err
		}
		resp, err := client.Do(req)
		if err != nil {
			return 0, err
		}
		defer func() { _ = resp.Body.Close() }()
		return resp.StatusCode, nil
	}
}

func slowerThan(rs []loadgen.Result, d time.Duration) float64 {
	if len(rs) == 0 {
		return 0
	}
	var n int
	for _, r := range rs {
		if r.Response() >= d {
			n++
		}
	}
	return float64(n) / float64(len(rs))
}

func TestTheClosedLoopGeneratorHidesWhatTheOpenLoopOneSees(t *testing.T) {
	// THE test this package exists for, and the statistic that shows it
	// is the MEDIAN rather than the tail.
	//
	// A global 200ms stall every 20 requests. A closed-loop generator
	// holds one request at a time, so it simply waits it out: it issues
	// far fewer requests, and only the handful that were in flight are
	// recorded as slow. Its median stays near zero and its tail looks
	// like a well-behaved server with occasional hiccups.
	//
	// Under a schedule, every request DUE during a stall is affected,
	// which is most of them. The median moves.
	//
	// Workers is set high enough that the driver is not the bottleneck —
	// otherwise this would demonstrate the driver's own worker bound and
	// call it a finding. DeficitP99 is asserted to stay small for
	// exactly that reason.
	srv := stallingServer(t, 20, 200*time.Millisecond)
	do := get(srv.URL)

	cfg := loadgen.Config{Rate: 200, Duration: 2 * time.Second, Workers: 256}
	openResults := loadgen.Run(context.Background(), cfg, do)
	open := loadgen.Summarise(openResults, cfg)

	closedResults := closedLoop(context.Background(), 2*time.Second, do)
	closed := loadgen.Summarise(closedResults, cfg)

	t.Logf("open-loop:   n=%d p50=%.1fms p99=%.1fms deficit_p99=%.1fms slow=%.0f%%",
		open.Requests, open.ResponseP50, open.ResponseP99, open.DeficitP99,
		100*slowerThan(openResults, 100*time.Millisecond))
	t.Logf("closed-loop: n=%d p50=%.1fms p99=%.1fms slow=%.0f%%",
		closed.Requests, closed.ResponseP50, closed.ResponseP99,
		100*slowerThan(closedResults, 100*time.Millisecond))

	if open.DeficitP99 > 100 {
		t.Fatalf("deficit p99 = %.1fms: the driver saturated before the server did, "+
			"so this run measured the worker bound rather than the stall",
			open.DeficitP99)
	}

	openSlow := slowerThan(openResults, 100*time.Millisecond)
	closedSlow := slowerThan(closedResults, 100*time.Millisecond)
	if openSlow <= closedSlow {
		t.Errorf("%.0f%% of open-loop requests were slow against %.0f%% closed-loop; "+
			"the schedule is not keeping requests flowing during a stall",
			100*openSlow, 100*closedSlow)
	}
	if open.ResponseP50 <= closed.ResponseP50 {
		t.Errorf("open-loop median %.1fms did not exceed closed-loop median %.1fms — "+
			"the closed-loop generator waited the stall out and reported a healthy "+
			"median, which is exactly what this driver exists to not do",
			open.ResponseP50, closed.ResponseP50)
	}
}

func TestServiceTimeAloneWouldHaveLookedFine(t *testing.T) {
	// The subtler half. Even WITHIN the open-loop run, reading only
	// service time reproduces the closed-loop lie: each individual
	// request was fast, and the queue in front of it is invisible.
	srv := stallingServer(t, 25, 250*time.Millisecond)

	cfg := loadgen.Config{Rate: 200, Duration: 2 * time.Second, Workers: 2}
	rep := loadgen.Summarise(loadgen.Run(context.Background(), cfg, get(srv.URL)), cfg)

	t.Logf("service p50=%.1f p99=%.1f · response p50=%.1f p99=%.1f · deficit p99=%.1f",
		rep.ServiceP50, rep.ServiceP99, rep.ResponseP50, rep.ResponseP99, rep.DeficitP99)

	if rep.ServiceP50 >= rep.ResponseP99 {
		t.Skip("the machine is too loaded for this comparison to mean anything")
	}
	if rep.ResponseP99 <= rep.ServiceP99 {
		t.Errorf("response p99 (%.1fms) did not exceed service p99 (%.1fms) — "+
			"queueing was absorbed rather than reported", rep.ResponseP99, rep.ServiceP99)
	}
}

func TestAServerThatKeepsUpShowsNoOmission(t *testing.T) {
	// The instrument must not manufacture the effect it is looking for.
	// Against a server that never stalls, response and service should
	// agree closely, and the deficit should be small.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := loadgen.Config{Rate: 100, Duration: time.Second, Workers: 4}
	rep := loadgen.Summarise(loadgen.Run(context.Background(), cfg, get(srv.URL)), cfg)

	t.Logf("service p99=%.2fms response p99=%.2fms deficit p99=%.2fms",
		rep.ServiceP99, rep.ResponseP99, rep.DeficitP99)

	if rep.Omission > 25 {
		t.Errorf("omission = %.1fms against a server that never stalled; the "+
			"driver is inventing latency it should be measuring", rep.Omission)
	}
	if rep.OK == 0 {
		t.Error("no request succeeded against a trivial server")
	}
}

func TestAFailedRequestKeepsItsLatency(t *testing.T) {
	// Excluding errors is how a collapsing system produces its best ever
	// latency report. A 500 that took a long time cost that long.
	slow := func(ctx context.Context) (int, error) {
		time.Sleep(40 * time.Millisecond)
		return 500, nil
	}
	cfg := loadgen.Config{Rate: 50, Duration: 400 * time.Millisecond, Workers: 8}
	rep := loadgen.Summarise(loadgen.Run(context.Background(), cfg, slow), cfg)

	if rep.Failed == 0 {
		t.Fatal("a run of 500s reported no failures")
	}
	if rep.OK != 0 {
		t.Errorf("ok = %d; a 500 is not a success", rep.OK)
	}
	if rep.ServiceP99 < 30 {
		t.Errorf("service p99 = %.1fms over 40ms responses; failed requests were "+
			"dropped from the distribution", rep.ServiceP99)
	}
}

func TestATransportErrorIsAResultRatherThanASilence(t *testing.T) {
	boom := func(ctx context.Context) (int, error) {
		time.Sleep(10 * time.Millisecond)
		return 0, errors.New("connection refused")
	}
	cfg := loadgen.Config{Rate: 50, Duration: 300 * time.Millisecond, Workers: 4}
	rep := loadgen.Summarise(loadgen.Run(context.Background(), cfg, boom), cfg)

	if rep.Failed == 0 || rep.OK != 0 {
		t.Errorf("report = %d ok / %d failed; every attempt errored", rep.OK, rep.Failed)
	}
	if rep.Requests != rep.Failed {
		t.Errorf("requests=%d failed=%d; an attempt that produced no response is "+
			"still an attempt", rep.Requests, rep.Failed)
	}
}

func TestTheScheduleIsNotDerivedFromCompletions(t *testing.T) {
	// The structural property behind every assertion above. Intended
	// times must be evenly spaced regardless of how long responses take,
	// because a schedule computed from the previous completion IS a
	// loop wearing a schedule's name.
	var calls atomic.Int64
	do := func(ctx context.Context) (int, error) {
		// Deliberately erratic.
		if calls.Add(1)%3 == 0 {
			time.Sleep(30 * time.Millisecond)
		}
		return 200, nil
	}
	cfg := loadgen.Config{Rate: 100, Duration: 500 * time.Millisecond, Workers: 8}
	results := loadgen.Run(context.Background(), cfg, do)

	if len(results) < 10 {
		t.Fatalf("only %d results", len(results))
	}
	want := 10 * time.Millisecond // 100/s
	for i := 1; i < len(results); i++ {
		if results[i].Intended.IsZero() || results[i-1].Intended.IsZero() {
			continue
		}
		gap := results[i].Intended.Sub(results[i-1].Intended)
		if gap != want {
			t.Fatalf("intended gap between %d and %d is %v, want exactly %v — the "+
				"schedule is being influenced by completions", i-1, i, gap, want)
		}
	}
}

func TestPercentilesAreObservedValuesRatherThanInterpolated(t *testing.T) {
	// Every number this repository publishes is supposed to be one that
	// was measured. An interpolated p99 is a value between two
	// observations that nothing ever exhibited.
	results := make([]loadgen.Result, 100)
	base := time.Now()
	for i := range results {
		results[i] = loadgen.Result{
			Intended: base,
			Started:  base,
			Ended:    base.Add(time.Duration(i+1) * time.Millisecond),
			Status:   200,
		}
	}
	rep := loadgen.Summarise(results, loadgen.Config{Rate: 1})

	if rep.ServiceP99 != 99 {
		t.Errorf("p99 = %v, want exactly 99 (the 99th of 100 observations)", rep.ServiceP99)
	}
	if rep.ServiceP50 != 50 {
		t.Errorf("p50 = %v, want exactly 50", rep.ServiceP50)
	}
}
