// Command ghost-trace runs the collection and decision endpoints, and
// an event archive, in one process.
//
// It was the M1 vertical slice, and this comment used to argue against
// splitting it before a measurement justified one. The measurement
// arrived and the split happened (docs/the-split.md): the demo host is
// its own origin now, and this binary no longer serves a page.
//
// It is kept deliberately and permanently. It is the path `make
// numbers` takes — the canonical six-numbers run needs no broker and no
// Docker — and it is the cheap rollback if the composed topology has to
// be abandoned. Both shapes are built from one codebase, and
// `make shadow-http` is what keeps them answering alike.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/FabioCaffarello/ghost-trace/libs/decision"
	"github.com/FabioCaffarello/ghost-trace/libs/eventstream"
	"github.com/FabioCaffarello/ghost-trace/libs/metrics"
	"github.com/FabioCaffarello/ghost-trace/libs/middleware"
	"github.com/FabioCaffarello/ghost-trace/libs/policy"
	"github.com/FabioCaffarello/ghost-trace/libs/substrate"
	"github.com/FabioCaffarello/ghost-trace/libs/tenant"
	"github.com/FabioCaffarello/ghost-trace/libs/wire"
	"github.com/FabioCaffarello/ghost-trace/services/collector/internal/adapters/livesessions"
	"github.com/FabioCaffarello/ghost-trace/services/collector/internal/adapters/lossmeter"
	"github.com/FabioCaffarello/ghost-trace/services/collector/internal/adapters/substratearchive"
	"github.com/FabioCaffarello/ghost-trace/services/collector/internal/api"
	"github.com/FabioCaffarello/ghost-trace/services/collector/internal/app"
	"github.com/FabioCaffarello/ghost-trace/services/collector/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/collector/internal/session"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "ghost-trace: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		natsURL = flag.String("nats", os.Getenv("GT_NATS_URL"),
			"mirror archived records onto this NATS event stream (empty disables it)")
		addr        = flag.String("addr", "127.0.0.1:8080", "listen address")
		dataDir     = flag.String("data", "", "substrate directory; empty disables the raw event archive")
		mode        = flag.String("mode", policy.ModeMonitor, "monitor | enforce")
		policyFile  = flag.String("policy", "", "calibration JSON; empty uses the embedded default")
		tenantsFile = flag.String("tenants", os.Getenv("GT_TENANTS"),
			"JSON registry of tenants (id, site_key, secret_key); empty uses the "+
				"single-tenant flags below")
		tenantID    = flag.String("tenant", "t_demo", "tenant id, when -tenants is not given")
		siteKey     = flag.String("site-key", "pk_demo", "public site key, embedded in the page")
		secretKey   = flag.String("secret-key", "sk_demo", "secret key for server-to-server decision calls")
		pointerHz   = flag.Int("pointer-hz", 20, "collect policy: pointer sample rate")
		batchMs     = flag.Int("batch-ms", 2000, "collect policy: telemetry batch interval")
		ttl         = flag.Duration("session-ttl", 30*time.Minute, "session token lifetime")
		corsOrigins = flag.String("cors-origin", os.Getenv("GT_CORS_ORIGINS"),
			"comma-separated page origins allowed to call /v1/sessions and /v1/telemetry "+
				"cross-origin; empty disables CORS (same-origin deployments)")
		health = flag.Bool("healthcheck", false, "probe /healthz on -addr and exit 0/1; the container health check execs the binary because distroless ships no shell or curl")
	)
	flag.Parse()

	if *health {
		return probeHealth(*addr)
	}

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if *mode != policy.ModeMonitor && *mode != policy.ModeEnforce {
		return fmt.Errorf("unknown -mode %q (want %q or %q)", *mode, policy.ModeMonitor, policy.ModeEnforce)
	}

	// Calibration must be settled before anything serves: every
	// evaluation records policy.Ref, and a ref that changed mid-process
	// would make the archive lie about which numbers judged a session.
	if *policyFile != "" {
		if err := policy.LoadCalibration(*policyFile); err != nil {
			return err
		}
	}

	registry, err := tenants(*tenantsFile, *tenantID, *siteKey, *secretKey)
	if err != nil {
		return err
	}
	log.Info("serving tenants", "ids", registry.IDs(),
		"source", map[bool]string{true: *tenantsFile, false: "flags"}[*tenantsFile != ""])

	// The shipped credentials authenticate nobody: they are flag
	// defaults, they are in compose.yml and .env.example, and the
	// README prints them. Starting on one was silent, which is the
	// wrong default for the setting that decides whether the
	// secret-key endpoints are protected at all. A warning, not a
	// refusal — `make numbers`, compose and every gate run on these
	// keys deliberately.
	if demo := registry.DemoTenants(); len(demo) > 0 {
		log.Warn("running on the credentials this repository ships; anyone who has "+
			"read the source can call the secret-key endpoints",
			"tenants", demo, "fix", "-tenants <file>, or -site-key/-secret-key")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// THE ARCHIVE IS ONE STORE, NOT TWO (ADR-0006).
	//
	// -nats and -data are alternatives now, and giving both is an error
	// rather than a precedence rule nobody would remember. Until PR-2.5b
	// the collector wrote locally AND mirrored onto the stream, with the
	// local write authoritative; parity across a full experiment run
	// (3419 records, nothing missing either way) retired that.
	//
	//   -nats   the stream is the archive; the archive service stores it
	//   -data   a local substrate, for the all-in-one development binary
	//           and for the six-numbers run, which needs no broker
	//
	// Either way the archive is optional to the DECISION path: nothing
	// reads from it to judge a session. It exists so a recorded session
	// can be re-scored when a threshold moves.
	var eventArchive app.EventArchive = app.NullArchive{}
	var sessionStore app.SessionSnapshots

	switch {
	case *natsURL != "" && *dataDir != "":
		return fmt.Errorf("-nats and -data are alternatives: the stream is the archive " +
			"when a broker is configured, and writing both was the transitional " +
			"dual-write that ADR-0006 retired")

	case *natsURL != "":
		nc, js, err := eventstream.Connect(*natsURL, "ghost-trace-collector")
		if err != nil {
			return fmt.Errorf("connect event stream: %w", err)
		}
		defer nc.Close()
		if err := eventstream.EnsureStream(ctx, js); err != nil {
			return fmt.Errorf("ensure event stream: %w", err)
		}
		eventArchive = eventstream.NewArchive(eventstream.NewPublisher(js), *tenantID)
		log.Info("event stream is the archive", "nats", *natsURL)

		// Session snapshots: what the decision engine reads instead of
		// holding the session itself. Best effort — a bucket that is
		// unreachable makes another process's view stale, which is that
		// process's problem to detect, not a reason to reject telemetry
		// here.
		sessionStore, err = eventstream.EnsureSessions(ctx, js, *ttl)
		if err != nil {
			return fmt.Errorf("open session snapshots: %w", err)
		}
		log.Info("session snapshots enabled", "bucket", eventstream.SessionBucket, "ttl", *ttl)

	case *dataDir != "":
		sub, err := substrate.Open(ctx, *dataDir+"/events.db", *dataDir+"/blobs")
		if err != nil {
			return fmt.Errorf("open substrate: %w", err)
		}
		defer func() { _ = sub.Close() }()
		eventArchive = substratearchive.New(ingest.New(sub, time.Now))
		log.Info("local substrate is the archive", "dir", *dataDir)

	default:
		log.Warn("no archive configured; sessions will not be replayable " +
			"(-data for a local substrate, -nats for the event stream)")
	}

	sessions := session.NewStore(*ttl, time.Now)

	// One registry per process: the HTTP series and every domain counter
	// are exposed together, because two registries behind one endpoint
	// would need two encoders and would drop whichever the handler
	// forgot. Built here rather than with the rest of the observability
	// chain because the application counts into it too.
	reg := metrics.New()

	application := app.New(app.Config{}, sessions, eventArchive, time.Now, log).
		WithLossMeter(lossmeter.New(reg))
	if sessionStore != nil {
		application = application.WithSnapshots(sessionStore)
	}

	// The two server-to-server endpoints, served from the state this
	// process is maintaining. The decision engine mounts the same
	// package over the snapshots that state is published as; the only
	// difference between the two services is the port below.
	decisions := decision.New(decision.Config{
		Mode:    *mode,
		Tenants: registry,
	}, livesessions.New(sessions), eventArchive, time.Now, log)

	apiSrv := api.New(api.Config{
		Tenants:        registry,
		AllowedOrigins: splitOrigins(*corsOrigins),
		CollectPolicy: wire.CollectPolicy{
			PointerHz: *pointerHz,
			BatchMs:   *batchMs,
			Types:     []string{"pointer", "key", "scroll", "focus", "visibility", "form"},
		},
		SessionTTL: *ttl,
	}, application, log)

	mux := apiSrv.Routes()
	decisions.Mount(mux)

	// Expired sessions are never otherwise removed, so without this the
	// store grows for the life of the process.
	go sweepLoop(ctx, sessions, log)

	// Observability chain, outermost first: request-id so every layer
	// can correlate, recovery so a panic is logged with its id, logging
	// so the 500 a panic produces still gets its line, metrics
	// innermost so it measures the handler rather than the logging.
	// Published rather than logged so the two services can be COMPARED
	// from outside. Two registries that disagree about who exists each
	// behave correctly alone and wrongly together, and no request fails
	// on the way; `make shadow-http` reads this from both.
	reg.Info("tenant_registry_info",
		"The tenant set this process serves. The fingerprint covers ids and site "+
			"keys, never secrets.",
		map[string]string{"fingerprint": registry.Fingerprint(),
			"tenants": strconv.Itoa(registry.Len())})

	httpMetrics := middleware.NewMetrics(reg)
	mux.Handle("GET /metrics", reg.Handler())
	handler := middleware.Chain(mux,
		middleware.RequestID(),
		middleware.Recovery(log),
		middleware.Logging(log),
		httpMetrics.Collect(),
	)

	srv := &http.Server{
		Addr:              *addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", *addr, "mode", *mode, "policy", policy.Ref,
			"cors_origins", splitOrigins(*corsOrigins))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

// tenants builds the registry a service serves.
//
// A file is the multi-tenant path; without one the three single-tenant
// flags become a registry of exactly one, which keeps the all-in-one
// development binary, the compose defaults and the six-numbers run
// working without a new file to remember. One tenant is a registry with
// one row, not a different mode.
func tenants(path, id, siteKey, secretKey string) (*tenant.Registry, error) {
	if path != "" {
		return tenant.Load(path)
	}
	return tenant.New(tenant.Tenant{ID: id, SiteKey: siteKey, SecretKey: secretKey})
}

// splitOrigins parses the comma-separated allowlist. Empty entries are
// dropped rather than becoming an origin of "", which would match a
// request that sent no Origin header at all.
func splitOrigins(csv string) []string {
	var out []string
	for _, o := range strings.Split(csv, ",") {
		if o = strings.TrimSpace(o); o != "" {
			out = append(out, o)
		}
	}
	return out
}

// probeHealth GETs /healthz on the listen address. A wildcard listen
// host is probed via loopback — inside the container the server binds
// 0.0.0.0 but the probe runs in the same network namespace.
func probeHealth(addr string) error {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("healthcheck: bad -addr %q: %w", addr, err)
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://" + net.JoinHostPort(host, port) + "/healthz")
	if err != nil {
		return fmt.Errorf("healthcheck: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("healthcheck: /healthz returned %d", resp.StatusCode)
	}
	return nil
}

func sweepLoop(ctx context.Context, sessions *session.Store, log *slog.Logger) {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if n := sessions.Sweep(); n > 0 {
				log.Info("swept expired sessions", "removed", n, "live", sessions.Len())
			}
		}
	}
}
