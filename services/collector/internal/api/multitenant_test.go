package api

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/FabioCaffarello/ghost-trace/libs/decision"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/libs/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/libs/policy"
	"github.com/FabioCaffarello/ghost-trace/libs/substrate"
	"github.com/FabioCaffarello/ghost-trace/libs/tenant"
	"github.com/FabioCaffarello/ghost-trace/services/collector/internal/adapters/livesessions"
	"github.com/FabioCaffarello/ghost-trace/services/collector/internal/adapters/substratearchive"
	"github.com/FabioCaffarello/ghost-trace/services/collector/internal/app"
	"github.com/FabioCaffarello/ghost-trace/services/collector/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/collector/internal/session"
)

// Two customers on one collector, over real HTTP.
//
// Until this milestone a tenant was process configuration, so "two
// tenants" meant two deployments and the questions below could not be
// asked. They are the ones that matter now that sessions share a store,
// snapshots share a bucket and records share a stream.

const (
	tenantA, siteA, secretA = "t_alpha", "pk_alpha", "sk_alpha"
	tenantB, siteB, secretB = "t_beta", "pk_beta", "sk_beta"
)

type twoTenants struct {
	srv *httptest.Server
	sub *substrate.Substrate
}

func startTwoTenants(t *testing.T) *twoTenants {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	registry, err := tenant.New(
		tenant.Tenant{ID: tenantA, SiteKey: siteA, SecretKey: secretA},
		tenant.Tenant{ID: tenantB, SiteKey: siteB, SecretKey: secretB},
	)
	if err != nil {
		t.Fatalf("registry: %v", err)
	}

	dir := t.TempDir()
	sub, err := substrate.Open(context.Background(), dir+"/db.sqlite", dir+"/blobs")
	if err != nil {
		t.Fatalf("substrate: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	// ONE store, ONE archive, ONE process — which is the point. Two
	// deployments would prove nothing about isolation.
	store := session.NewStore(30*time.Minute, time.Now)
	arch := substratearchive.New(ingest.New(sub, time.Now))
	a := app.New(app.Config{}, store, arch, time.Now, log)
	s := New(Config{Tenants: registry}, a, log)

	mux := s.Routes()
	decision.New(decision.Config{Mode: policy.ModeMonitor, Tenants: registry},
		livesessions.New(store), arch, time.Now, log).Mount(mux)

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return &twoTenants{srv: srv, sub: sub}
}

func (tt *twoTenants) session(t *testing.T, siteKey string) string {
	t.Helper()
	status, body := post(t, tt.srv, "/v1/sessions", "", map[string]any{
		"site_key": siteKey,
		"page":     map[string]any{"path": "/login"},
		"client":   map[string]any{"pointer": "fine"},
	})
	if status != http.StatusOK {
		t.Fatalf("sessions for %s = %d", siteKey, status)
	}
	token, _ := body["session_token"].(string)
	if token == "" {
		t.Fatalf("no token for %s", siteKey)
	}
	return token
}

func (tt *twoTenants) feed(t *testing.T, token string) {
	t.Helper()
	status, _ := post(t, tt.srv, "/v1/telemetry", token, map[string]any{
		"session_token": token, "seq": 1, "sent_at_ms": 2000,
		"page": map[string]any{"path": "/login"},
		"events": []map[string]any{
			{"type": "key", "t": 1000, "phase": "down", "class": "alpha", "target": "f"},
			{"type": "key", "t": 1080, "phase": "up", "class": "alpha", "target": "f"},
			{"type": "form", "t": 1200, "action": "injected", "target": "f"},
		},
	})
	if status != http.StatusAccepted {
		t.Fatalf("telemetry = %d", status)
	}
}

func TestEachSiteKeyOpensASessionForItsOwnTenant(t *testing.T) {
	tt := startTwoTenants(t)
	a, b := tt.session(t, siteA), tt.session(t, siteB)
	if a == b {
		t.Fatal("two tenants were issued the same token")
	}

	// Attribution is what every downstream join depends on, so count
	// the durable records rather than trusting the responses.
	starts := 0
	if err := tt.sub.WalkEvents(context.Background(), func(r substrate.EventRow) error {
		if r.MessageType == "ghosttrace.events.v1.SessionStart" {
			starts++
		}
		return nil
	}); err != nil {
		t.Fatalf("walk: %v", err)
	}
	if starts != 2 {
		t.Errorf("archived %d session starts, want one per tenant", starts)
	}
}

func TestOneTenantCannotAskAboutAnothersSession(t *testing.T) {
	// THE isolation property. A token is 144 bits of randomness so
	// nobody guesses one — but tokens are handed to browsers, land in
	// logs, and can be read out of a page. Presenting one with a
	// different tenant's secret must not answer.
	tt := startTwoTenants(t)
	token := tt.session(t, siteA)
	tt.feed(t, token)

	status, mine := post(t, tt.srv, "/v1/decisions", secretA, map[string]any{
		"session_token": token, "action": "login",
	})
	if status != http.StatusOK {
		t.Fatalf("alpha asking about its own session = %d", status)
	}
	ev, _ := mine["evidence"].(map[string]any)
	events, _ := ev["events"].(float64)
	if events == 0 {
		t.Fatal("alpha sees no evidence for its own session; the test proves nothing")
	}

	status, theirs := post(t, tt.srv, "/v1/decisions", secretB, map[string]any{
		"session_token": token, "action": "login",
	})
	// Answered, not refused: a 404 or 403 would confirm the token
	// exists. Beta gets what it would get for any token it does not
	// own — a cold start.
	if status != http.StatusOK {
		t.Fatalf("beta asking about alpha's session = %d, want a cold start", status)
	}
	ev, _ = theirs["evidence"].(map[string]any)
	if got, _ := ev["events"].(float64); got != 0 {
		t.Errorf("beta saw %v events of alpha's session; evidence leaked across tenants", got)
	}
	if got, _ := ev["duration_ms"].(float64); got != 0 {
		t.Errorf("beta saw %v ms of alpha's session; evidence leaked across tenants", got)
	}
}

func TestAnEvaluationIsAttributedToTheCallerWhoAskedForIt(t *testing.T) {
	tt := startTwoTenants(t)
	token := tt.session(t, siteA)
	tt.feed(t, token)

	status, _ := post(t, tt.srv, "/v1/decisions", secretB, map[string]any{
		"session_token": token, "action": "login",
	})
	if status != http.StatusOK {
		t.Fatalf("status = %d", status)
	}

	// Beta asked, so beta owns the evaluation — even though the token
	// it named belongs to alpha. Attributing it to alpha would put a
	// record beta caused into alpha's archive, where alpha would later
	// calibrate against it.
	//
	// Read the ATTRIBUTION out of the payload, not the row: counting
	// records would pass whatever tenant they carried, and the tenant
	// is the whole claim.
	var owners []string
	ctx := context.Background()
	if err := tt.sub.WalkEvents(ctx, func(r substrate.EventRow) error {
		if r.MessageType != "ghosttrace.events.v1.Evaluation" {
			return nil
		}
		blob, err := tt.sub.ReadBlob(ctx, r.EventHash)
		if err != nil {
			return err
		}
		var rec eventsv1.Evaluation
		if err := proto.Unmarshal(blob, &rec); err != nil {
			return err
		}
		owners = append(owners, rec.GetTenantId())
		return nil
	}); err != nil {
		t.Fatalf("walk: %v", err)
	}

	if len(owners) == 0 {
		t.Fatal("no evaluation was archived; there was nothing to attribute")
	}
	for _, owner := range owners {
		if owner != tenantB {
			t.Errorf("an evaluation beta asked for is attributed to %q, want %q",
				owner, tenantB)
		}
	}
}

func TestAWrongSiteKeyOpensNothing(t *testing.T) {
	tt := startTwoTenants(t)
	status, _ := post(t, tt.srv, "/v1/sessions", "", map[string]any{
		"site_key": "pk_not_a_tenant",
		"page":     map[string]any{"path": "/login"},
	})
	if status != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", status)
	}
}

func TestAWrongSecretResolvesToNoTenantAtAll(t *testing.T) {
	tt := startTwoTenants(t)
	token := tt.session(t, siteA)
	status, _ := post(t, tt.srv, "/v1/decisions", "sk_not_a_tenant", map[string]any{
		"session_token": token, "action": "login",
	})
	if status != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", status)
	}
}
