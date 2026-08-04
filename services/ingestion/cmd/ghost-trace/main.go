// Command ghost-trace runs the M1 vertical slice: the API, the raw
// event archive, and the demo host application, in one process.
//
// One binary is the right shape here. M1 is a slice, not a topology, and
// splitting it into services before there is a measurement showing why
// would repeat v1's mistake of building the decomposition first.
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
	"strings"
	"syscall"
	"time"

	"github.com/FabioCaffarello/ghost-trace/libs/decision"
	"github.com/FabioCaffarello/ghost-trace/libs/eventstream"
	"github.com/FabioCaffarello/ghost-trace/libs/middleware"
	"github.com/FabioCaffarello/ghost-trace/libs/policy"
	"github.com/FabioCaffarello/ghost-trace/libs/substrate"
	"github.com/FabioCaffarello/ghost-trace/libs/wire"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/adapters/livesessions"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/adapters/streamarchive"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/adapters/substratearchive"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/api"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/app"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/session"
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
		tenantID    = flag.String("tenant", "t_demo", "tenant id")
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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Raw event archive. Optional: the decision path does not read from
	// it, so a slice can run entirely in memory. It exists so M2 can
	// re-score recorded sessions when a threshold moves.
	var archive app.EventArchive = app.NullArchive{}
	var sessionStore app.SessionSnapshots
	if *dataDir != "" {
		sub, err := substrate.Open(ctx, *dataDir+"/events.db", *dataDir+"/blobs")
		if err != nil {
			return fmt.Errorf("open substrate: %w", err)
		}
		defer func() { _ = sub.Close() }()
		archive = substratearchive.New(ingest.New(sub, time.Now))
		log.Info("raw event archive enabled", "dir", *dataDir)

		// Transitional dual-write. With -nats set, every record still
		// goes to the local substrate AND is mirrored onto the event
		// stream for the archive service. The local write stays
		// authoritative: a broker outage is logged and counted, never
		// returned, because adding a broker must not make this service
		// less reliable than it was without one. PR-2.5 removes the
		// local write once parity is demonstrated.
		if *natsURL != "" {
			nc, js, err := eventstream.Connect(*natsURL, "ghost-trace-collector")
			if err != nil {
				return fmt.Errorf("connect event stream: %w", err)
			}
			defer nc.Close()
			if err := eventstream.EnsureStream(ctx, js); err != nil {
				return fmt.Errorf("ensure event stream: %w", err)
			}
			mirror := streamarchive.New(archive, eventstream.NewPublisher(js), *tenantID, log)
			archive = mirror
			defer func() {
				appended, published, dropped := mirror.Counts()
				log.Info("event stream mirror", "appended", appended,
					"published", published, "dropped", dropped)
			}()
			log.Info("event stream mirror enabled", "nats", *natsURL)

			// Session snapshots: what a decision engine reads instead of
			// holding the session itself. Best effort — a bucket that is
			// unreachable makes another process's view stale, which is
			// that process's problem to detect, not a reason to reject
			// telemetry here.
			sessionStore, err = eventstream.OpenSessions(ctx, js, *ttl)
			if err != nil {
				return fmt.Errorf("open session snapshots: %w", err)
			}
			log.Info("session snapshots enabled", "bucket", eventstream.SessionBucket, "ttl", *ttl)
		}
	} else {
		log.Warn("raw event archive disabled; sessions will not be replayable (-data to enable)")
	}

	sessions := session.NewStore(*ttl, time.Now)

	application := app.New(app.Config{
		TenantID: *tenantID,
	}, sessions, archive, time.Now, log)
	if sessionStore != nil {
		application = application.WithSnapshots(sessionStore)
	}

	// The two server-to-server endpoints, served from the state this
	// process is maintaining. The decision engine mounts the same
	// package over the snapshots that state is published as; the only
	// difference between the two services is the port below.
	decisions := decision.New(decision.Config{
		TenantID:  *tenantID,
		Mode:      *mode,
		SecretKey: *secretKey,
	}, livesessions.New(sessions), archive, time.Now, log)

	apiSrv := api.New(api.Config{
		SiteKey:        *siteKey,
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
	metrics := middleware.NewMetrics()
	mux.Handle("GET /metrics", metrics.Handler())
	handler := middleware.Chain(mux,
		middleware.RequestID(),
		middleware.Recovery(log),
		middleware.Logging(log),
		metrics.Collect(),
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
