// Command decision-engine serves /v1/decisions and /v1/outcomes from
// session snapshots rather than from sessions it observed.
//
// It holds no session state. The collector owns sessions and publishes
// each one's feature state to a snapshot bucket (ADR-0004); this reads
// that bucket, judges, and publishes the durable evaluation onto the
// event stream for the archive service. That is the entire split: the
// same endpoints, the same judgement, a different answer to "where does
// the state come from".
//
// Both endpoints are libs/decision's, mounted here exactly as the
// collector mounts them. Nothing about the contract is decided in this
// file — if it were, the two services could differ in ways no test
// would catch (ADR-0005).
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
	"syscall"
	"time"

	"github.com/FabioCaffarello/ghost-trace/libs/decision"
	"github.com/FabioCaffarello/ghost-trace/libs/eventstream"
	"github.com/FabioCaffarello/ghost-trace/libs/metrics"
	"github.com/FabioCaffarello/ghost-trace/libs/middleware"
	"github.com/FabioCaffarello/ghost-trace/libs/policy"
	"github.com/FabioCaffarello/ghost-trace/libs/tenant"
	"github.com/FabioCaffarello/ghost-trace/services/decision-engine/internal/snapshotsessions"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "decision-engine: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		addr    = flag.String("addr", "127.0.0.1:8082", "listen address")
		natsURL = flag.String("nats", os.Getenv("GT_NATS_URL"),
			"NATS URL carrying the session snapshots and the event stream")
		mode        = flag.String("mode", policy.ModeMonitor, "monitor | enforce")
		policyFile  = flag.String("policy", "", "calibration JSON; empty uses the embedded default")
		tenantsFile = flag.String("tenants", os.Getenv("GT_TENANTS"),
			"JSON registry of tenants (id, site_key, secret_key); empty uses the "+
				"single-tenant flags below")
		tenantID  = flag.String("tenant", "t_demo", "tenant id, when -tenants is not given")
		secretKey = flag.String("secret-key", "sk_demo",
			"secret key for server-to-server calls, when -tenants is not given")
		siteKey = flag.String("site-key", "pk_demo",
			"public site key, when -tenants is not given. This service never resolves a "+
				"tenant BY site_key — only the collector does — but a tenant HAS one, and "+
				"the single-tenant fallback builds a whole tenant rather than a partial one")
		ttl = flag.Duration("session-ttl", 30*time.Minute,
			"snapshot bucket TTL. Must match the collector's -session-ttl: the "+
				"collector owns the bucket and this service refuses to start on a "+
				"mismatch rather than rewriting it")
		health = flag.Bool("healthcheck", false,
			"probe /healthz on -addr and exit 0/1; the container health check execs the binary because distroless ships no shell or curl")
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
	//
	// It must ALSO match the collector's. Two services scoring the same
	// session under different calibrations would disagree for a reason
	// the shadow comparison cannot distinguish from a real defect, which
	// is why policy.Ref travels on every evaluation record.
	if *policyFile != "" {
		if err := policy.LoadCalibration(*policyFile); err != nil {
			return err
		}
	}

	// Unlike the collector, this service cannot run without a broker.
	// The collector degrades to a local substrate when the stream is
	// gone (ADR-0003); this one has no session state of its own and no
	// durable store of its own, so a missing broker is a
	// misconfiguration and not a degraded mode. Failing at startup is
	// the honest response — the alternative is answering every decision
	// as a cold start and looking healthy while doing it.
	if *natsURL == "" {
		return fmt.Errorf("-nats is required: this service reads session snapshots " +
			"and publishes evaluations, and has no local store to fall back to")
	}

	registry, err := tenants(*tenantsFile, *tenantID, *siteKey, *secretKey)
	if err != nil {
		return err
	}
	log.Info("serving tenants", "ids", registry.IDs(),
		"source", map[bool]string{true: *tenantsFile, false: "flags"}[*tenantsFile != ""])

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	nc, js, err := eventstream.Connect(*natsURL, "ghost-trace-decision-engine")
	if err != nil {
		return fmt.Errorf("connect event stream: %w", err)
	}
	defer nc.Close()

	if err := eventstream.EnsureStream(ctx, js); err != nil {
		return fmt.Errorf("ensure event stream: %w", err)
	}
	// Bind to the bucket the collector owns, and refuse a TTL that is
	// not the one this process was told to expect. Starting before the
	// collector has created it is an error rather than a silent
	// creation — a reader that created the bucket would be declaring a
	// lifetime it has no business declaring.
	snapshots, err := eventstream.OpenSessions(ctx, js, *ttl)
	if err != nil {
		return fmt.Errorf("open session snapshots: %w", err)
	}

	// The stream IS the durable store here, so publish failures are
	// returned rather than counted: an outcome answered 202 but never
	// stored would silently poison calibration.
	archive := eventstream.NewArchive(eventstream.NewPublisher(js), *tenantID)

	decisions := decision.New(decision.Config{
		Mode:    *mode,
		Tenants: registry,
	}, snapshotsessions.New(snapshots), archive, time.Now, log)

	mux := http.NewServeMux()
	decisions.Mount(mux)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	// Observability chain, outermost first: request-id so every layer
	// can correlate, recovery so a panic is logged with its id, logging
	// so the 500 a panic produces still gets its line, metrics innermost
	// so it measures the handler rather than the logging.
	// One registry per process: the HTTP series and every domain
	// counter are exposed together, because two registries behind one
	// endpoint would need two encoders and would drop whichever the
	// handler forgot.
	reg := metrics.New()
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
			"tenants", registry.IDs(), "snapshots", eventstream.SessionBucket)
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
