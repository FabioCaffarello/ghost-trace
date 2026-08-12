// These test the package standing alone, not through a host.
//
// Until now these two endpoints were only ever exercised through the
// collector's HTTP suite. That suite still runs and still holds the
// golden bytes, but a package that can only be tested by mounting it
// somewhere is a package whose second host inherits no coverage — and
// the second host is the entire reason this package exists.
package decision_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/FabioCaffarello/ghost-trace/libs/archive"
	"github.com/FabioCaffarello/ghost-trace/libs/decision"
	"github.com/FabioCaffarello/ghost-trace/libs/feature"
	"github.com/FabioCaffarello/ghost-trace/libs/policy"
	"github.com/FabioCaffarello/ghost-trace/libs/tenant"
)

func testTenants(t *testing.T) *tenant.Registry {
	t.Helper()
	r, err := tenant.New(tenant.Tenant{ID: "t_test", SiteKey: "pk_test", SecretKey: "sk_test"})
	if err != nil {
		t.Fatalf("tenant registry: %v", err)
	}
	return r
}

func quiet() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// fakeSessions answers from a fixed table. A token that is absent is a
// miss, which is the cold-start path and not an error.
type fakeSessions map[string]decision.Session

func (f fakeSessions) Lookup(_ context.Context, tenantID, token string) (decision.Session, bool, error) {
	s, ok := f[token]
	// Someone else's session is no session at all, never an error: the
	// answer must not tell a caller whether a token they do not own
	// exists.
	if ok && s.TenantID != tenantID {
		return decision.Session{}, false, nil
	}
	return s, ok, nil
}

// brokenSessions is the other kind of miss: the lookup itself failed.
type brokenSessions struct{ err error }

func (b brokenSessions) Lookup(context.Context, string, string) (decision.Session, bool, error) {
	return decision.Session{}, false, b.err
}

func newService(t *testing.T, sessions decision.Sessions, store archive.Store) *decision.Service {
	t.Helper()
	return decision.New(decision.Config{
		Mode: policy.ModeMonitor, Tenants: testTenants(t),
	}, sessions, store, time.Now, quiet())
}

func TestUnknownTokenIsAColdStartAndNotAnError(t *testing.T) {
	// §7: the caller is at a risk moment and needs an answer. A missing
	// session is a cold start the confidence dimension models, so this
	// must return a decision rather than fail.
	svc := newService(t, fakeSessions{}, archive.Null{})

	out, err := svc.Decide(context.Background(), decision.Input{
		TenantID: "t_test", SessionToken: "st_nope", Action: "login",
	})
	if err != nil {
		t.Fatalf("unknown token returned an error: %v", err)
	}
	if out.Decision == "" {
		t.Error("no decision returned for an unknown token")
	}
	if out.EvidenceMs != 0 {
		t.Errorf("evidence duration = %d for a session that was never seen, want 0", out.EvidenceMs)
	}
	if out.EvaluationID == "" {
		t.Error("no evaluation id minted")
	}
}

func TestBrokenLookupIsAnErrorAndNotAColdStart(t *testing.T) {
	// The distinction that matters: a store that is broken must not be
	// reported as a session that does not exist, because a cold start
	// scores as innocent.
	boom := errors.New("store is on fire")
	svc := newService(t, brokenSessions{err: boom}, archive.Null{})

	if _, err := svc.Decide(context.Background(), decision.Input{
		TenantID: "t_test", SessionToken: "st_x", Action: "login",
	}); !errors.Is(err, boom) {
		t.Errorf("a failed lookup produced %v, want the store's error", err)
	}
}

func TestAMissingTenantIsAnErrorAndNotAColdStart(t *testing.T) {
	// A caller of the library that forgets the tenant would otherwise
	// scope every lookup to "" and get a cold start for every session —
	// a silent wrong answer. The HTTP path cannot produce this, since
	// the handlers only pass a tenant they resolved from a presented
	// secret; the guard is for everything else, and it exists because
	// the shadow test dropped the field and spent a CI run comparing a
	// cold start against a real judgement.
	svc := newService(t, fakeSessions{}, archive.Null{})

	if _, err := svc.Decide(context.Background(), decision.Input{
		SessionToken: "st_x", Action: "login",
	}); !errors.Is(err, decision.ErrTenantRequired) {
		t.Errorf("Decide err = %v, want ErrTenantRequired", err)
	}
	if err := svc.RecordOutcome(context.Background(), decision.OutcomeInput{
		EvaluationID: "ev_1", Outcome: "login_success",
	}); !errors.Is(err, decision.ErrTenantRequired) {
		t.Errorf("RecordOutcome err = %v, want ErrTenantRequired", err)
	}
}

func TestDecideRequiresAnAction(t *testing.T) {
	svc := newService(t, fakeSessions{}, archive.Null{})
	if _, err := svc.Decide(context.Background(), decision.Input{
		TenantID: "t_test", SessionToken: "st_x",
	}); !errors.Is(err, decision.ErrActionRequired) {
		t.Errorf("err = %v, want ErrActionRequired", err)
	}
}

func TestEvidenceCountsPointerAndKeyEvents(t *testing.T) {
	svc := newService(t, fakeSessions{"st_live": {
		ID: "s_1", TenantID: "t_test", LastEventMs: 4200,
		State: policy.State{
			Pointer:   feature.PointerState{Points: 30},
			Keystroke: feature.KeystrokeState{Keys: 12},
		},
	}}, archive.Null{})

	out, err := svc.Decide(context.Background(), decision.Input{
		TenantID: "t_test", SessionToken: "st_live", Action: "login",
	})
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	if out.EvidenceEvents != 42 {
		t.Errorf("evidence events = %d, want 42 (30 pointer + 12 key)", out.EvidenceEvents)
	}
	if out.EvidenceMs != 4200 {
		t.Errorf("evidence duration = %d, want 4200", out.EvidenceMs)
	}
}

func TestASessionBelongingToAnotherTenantIsNotVisible(t *testing.T) {
	// The isolation property. Tokens are 144 bits of randomness, so
	// nobody guesses one — but a token can be handed over, logged, or
	// copied out of a page, and presenting it with a DIFFERENT tenant's
	// secret used to return a real decision about a session the caller
	// had no claim to. Both halves authenticated on their own, which is
	// why nothing caught it.
	svc := newService(t, fakeSessions{"st_theirs": {
		ID: "s_theirs", TenantID: "t_somebody_else", LastEventMs: 9999,
		State: policy.State{
			Pointer:   feature.PointerState{Points: 30},
			Keystroke: feature.KeystrokeState{Keys: 12},
		},
	}}, archive.Null{})

	out, err := svc.Decide(context.Background(), decision.Input{
		TenantID: "t_test", SessionToken: "st_theirs", Action: "login",
	})
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	// Answered as a cold start, not refused: refusing would confirm the
	// token exists. The caller learns nothing about a session that is
	// not theirs.
	if out.EvidenceEvents != 0 || out.EvidenceMs != 0 {
		t.Errorf("another tenant's session leaked: events=%d duration=%d",
			out.EvidenceEvents, out.EvidenceMs)
	}
}

func TestOutcomeRejectsUnknownLabels(t *testing.T) {
	svc := newService(t, fakeSessions{}, archive.Null{})
	if err := svc.RecordOutcome(context.Background(), decision.OutcomeInput{
		TenantID: "t_test", EvaluationID: "ev_1", Outcome: "login_sucess", // sic
	}); !errors.Is(err, decision.ErrUnknownOutcome) {
		t.Errorf("err = %v, want ErrUnknownOutcome — a typo'd label is worse "+
			"than a missing one, because it degrades calibration silently", err)
	}
}

func TestOutcomeRefusesWhenThereIsNowhereDurableToPutIt(t *testing.T) {
	// The sentinel has to survive crossing the module boundary. If
	// libs/decision declared its own "archive unavailable" instead of
	// re-exporting libs/archive's, errors.Is would be false here and the
	// handler below would answer 500 instead of 503 — a caller would be
	// told to retry a request that will never succeed, or worse, believe
	// a label was recorded.
	svc := newService(t, fakeSessions{}, archive.Null{})

	err := svc.RecordOutcome(context.Background(), decision.OutcomeInput{
		TenantID: "t_test", EvaluationID: "ev_1", Outcome: "login_success",
	})
	if !errors.Is(err, decision.ErrArchiveUnavailable) {
		t.Fatalf("err = %v, want ErrArchiveUnavailable", err)
	}
	if !errors.Is(err, archive.ErrUnavailable) {
		t.Error("the re-exported sentinel is not libs/archive's; errors.Is does " +
			"not hold across the module boundary")
	}
}

// ---------------------------------------------------------------
// the mounted surface
// ---------------------------------------------------------------

func mounted(t *testing.T, store archive.Store) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	newService(t, fakeSessions{"st_live": {ID: "s_1", TenantID: "t_live"}}, store).Mount(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func post(t *testing.T, srv *httptest.Server, path, bearer, body string) int {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, srv.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode
}

func TestBothEndpointsRefuseWithoutTheSecretKey(t *testing.T) {
	// These are the only endpoints that accept subject_id and action,
	// which is why neither is ever read from a browser request (§1).
	srv := mounted(t, archive.Null{})

	for _, tc := range []struct {
		path, bearer, body string
		want               int
	}{
		{"/v1/decisions", "", `{"session_token":"st_live","action":"login"}`, http.StatusUnauthorized},
		{"/v1/decisions", "st_live", `{"session_token":"st_live","action":"login"}`, http.StatusUnauthorized},
		{"/v1/outcomes", "", `{"evaluation_id":"ev_1","outcome":"login_success"}`, http.StatusUnauthorized},
		{"/v1/decisions", "sk_test", `{"session_token":"st_live","action":"login"}`, http.StatusOK},
	} {
		if got := post(t, srv, tc.path, tc.bearer, tc.body); got != tc.want {
			t.Errorf("POST %s with bearer %q = %d, want %d", tc.path, tc.bearer, got, tc.want)
		}
	}
}

func TestOutcomesAnswers503WhenTheArchiveIsUnavailable(t *testing.T) {
	srv := mounted(t, archive.Null{})
	if got := post(t, srv, "/v1/outcomes", "sk_test",
		`{"evaluation_id":"ev_1","outcome":"login_success"}`); got != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", got)
	}
}

func TestObservedAtMustBeRFC3339(t *testing.T) {
	// Rejected rather than replaced with the server clock: observed_at
	// is the application's claim and recorded_at the server's
	// observation, and the gap between them is itself a signal.
	srv := mounted(t, archive.Null{})
	if got := post(t, srv, "/v1/outcomes", "sk_test",
		`{"evaluation_id":"ev_1","outcome":"login_success","observed_at":"yesterday"}`); got != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", got)
	}
}

// ---------------------------------------------------------------
// the best-effort bound on the decision path
// ---------------------------------------------------------------

// okStore accepts everything, so the written counter has something to
// count.
type okStore struct{}

func (okStore) Append(context.Context, proto.Message, int64) error { return nil }

// brokerAckTimeout is roughly what a JetStream publish waits for an ack
// from a server that is gone — the five seconds the unbounded path
// actually took.
//
// The stalled store gives up after it rather than blocking forever, so
// deleting the bound under test makes this test FAIL on elapsed time
// instead of hanging. A red proof that hangs is a broken test, not a
// demonstration; the first version of this file hung, which is how the
// distinction got learned.
const brokerAckTimeout = 5 * time.Second

// stalledStore is a broker that has stopped answering: the publish
// blocks until whichever comes first — the caller's deadline, or the
// ack timeout the client would eventually hit on its own.
type stalledStore struct{}

func (stalledStore) Append(ctx context.Context, _ proto.Message, _ int64) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(brokerAckTimeout):
		return errors.New("no ack from the broker")
	}
}

// countingMeter records what the service claims it wrote and lost.
type countingMeter struct {
	written map[string]int
	dropped map[string]int
}

func newCountingMeter() *countingMeter {
	return &countingMeter{written: map[string]int{}, dropped: map[string]int{}}
}

func (m *countingMeter) Written(kind string) { m.written[kind]++ }
func (m *countingMeter) Dropped(kind, reason string) {
	m.dropped[kind+"/"+reason]++
}

func TestADecisionIsNotHeldOpenByAStalledArchive(t *testing.T) {
	// The defect this closes was a LATENCY defect, not a result defect:
	// the verdict was always fail-open, and the request that carried it
	// took as long as the broker's own ack timeout — about five seconds
	// — on the one path with a caller at a risk moment and an 80ms
	// budget. The collector bounded this at 250ms; this package kept
	// passing the raw request context straight through.
	meter := newCountingMeter()
	svc := newService(t, fakeSessions{"st_live": {ID: "s_1", TenantID: "t_test"}},
		stalledStore{}).WithLossMeter(meter)

	start := time.Now()
	out, err := svc.Decide(context.Background(), decision.Input{
		TenantID: "t_test", SessionToken: "st_live", Action: "login",
	})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("a stalled archive must not fail the decision: %v", err)
	}
	if out.Decision == "" {
		t.Fatal("a stalled archive must not empty the decision")
	}

	// Generous: the assertion is that SOME bound applies, not that the
	// machine is fast. Unbounded is five seconds.
	if limit := 2 * decision.BestEffortTimeout; elapsed > limit {
		t.Errorf("the decision took %v with the archive stalled; the best-effort "+
			"bound is %v, so anything near the broker's own ack timeout means the "+
			"request context is reaching the store unbounded",
			elapsed, decision.BestEffortTimeout)
	}

	// And the record is gone, so it has to be counted. A drop that only
	// reaches a log is the silent loss Phase 3 exists to remove.
	if got := meter.dropped[decision.KindEvaluation+"/"+decision.ReasonDeadline]; got != 1 {
		t.Errorf("dropped{evaluation,deadline} = %d, want 1 (counts: %v)",
			got, meter.dropped)
	}
	if got := meter.written[decision.KindEvaluation]; got != 0 {
		t.Errorf("written = %d, want 0 — nothing reached the store", got)
	}
}

func TestAnArchivedEvaluationIsCounted(t *testing.T) {
	meter := newCountingMeter()
	svc := newService(t, fakeSessions{"st_live": {ID: "s_1", TenantID: "t_test"}},
		okStore{}).WithLossMeter(meter)

	if _, err := svc.Decide(context.Background(), decision.Input{
		TenantID: "t_test", SessionToken: "st_live", Action: "login",
	}); err != nil {
		t.Fatal(err)
	}
	if got := meter.written[decision.KindEvaluation]; got != 1 {
		t.Errorf("written[evaluation] = %d, want 1", got)
	}
	if len(meter.dropped) != 0 {
		t.Errorf("nothing was lost, but dropped = %v", meter.dropped)
	}
}

func TestNoArchiveIsNeitherWrittenNorDropped(t *testing.T) {
	// A host with no store configured has not lost anything. Counting
	// it as a drop would make every no-archive run look like an outage.
	meter := newCountingMeter()
	svc := newService(t, fakeSessions{"st_live": {ID: "s_1", TenantID: "t_test"}},
		archive.Null{}).WithLossMeter(meter)

	if _, err := svc.Decide(context.Background(), decision.Input{
		TenantID: "t_test", SessionToken: "st_live", Action: "login",
	}); err != nil {
		t.Fatal(err)
	}
	if len(meter.written) != 0 || len(meter.dropped) != 0 {
		t.Errorf("no archive configured, but written=%v dropped=%v",
			meter.written, meter.dropped)
	}
}
