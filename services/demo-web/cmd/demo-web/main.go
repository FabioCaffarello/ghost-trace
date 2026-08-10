// Command demo-web serves the demo page, the SDK, and a stand-in host
// application.
//
// It is the only service here that is not Ghost Trace. It plays the
// CUSTOMER: it embeds the SDK in a page, and when a login is submitted
// it calls /v1/decisions server-to-server with the secret key, exactly
// as a real integrator would. Reaching into the policy package
// in-process would demonstrate neither the trust boundary (§1: subject
// and action are asserted by the application, never by the browser) nor
// the failure behaviour (§5: client timeout, then fail open).
//
// Running on its own origin is the point rather than a side effect. A
// customer's site and the collector are never the same host, and while
// this page was served by the collector the browser endpoints never had
// to answer a cross-origin request.
//
// It holds the secret key; the page never sees it.
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
	"syscall"
	"time"

	"github.com/FabioCaffarello/ghost-trace/libs/metrics"
	"github.com/FabioCaffarello/ghost-trace/libs/middleware"
	"github.com/FabioCaffarello/ghost-trace/libs/tenant"
	"github.com/FabioCaffarello/ghost-trace/services/demo-web/internal/web"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "demo-web: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		addr    = flag.String("addr", "127.0.0.1:8083", "listen address")
		apiBase = flag.String("api", os.Getenv("GT_API_BASE"),
			"collector origin the page's SDK will call (e.g. http://127.0.0.1:8080); "+
				"it must list this service's origin in its -cors-origin")
		engineBase = flag.String("engine", "",
			"origin serving /v1/decisions; defaults to -api")
		siteKey    = flag.String("site-key", "pk_demo", "public site key, embedded in the page")
		secretKey  = flag.String("secret-key", "sk_demo", "secret key for the server-to-server decision call")
		captureLog = flag.String("capture-log", "",
			"JSONL file of labelled human sessions for the M2 study; empty disables capture")
		decisionTO = flag.Duration("decision-timeout", 250*time.Millisecond,
			"client-side decision budget (contract §5: ~3x the p99 target)")
		health = flag.Bool("healthcheck", false,
			"probe /healthz on -addr and exit 0/1; the container health check execs the binary because distroless ships no shell or curl")
	)
	flag.Parse()

	if *health {
		return probeHealth(*addr)
	}

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if *apiBase == "" {
		return fmt.Errorf("-api is required: the page has to tell the SDK which " +
			"origin to send sessions and telemetry to, and this service is not it")
	}
	engine := *engineBase
	if engine == "" {
		engine = *apiBase
	}

	// The shipped credentials authenticate nobody: they are flag
	// defaults, they are in compose.yml, and the README prints them.
	// This service holds no registry — it is a customer, not a tenant —
	// so it compares against the same constants the registry does.
	// A warning, not a refusal: the demo is meant to run on them.
	if *secretKey == tenant.DemoSecretKey || *siteKey == tenant.DemoSiteKey {
		log.Warn("running on the credentials this repository ships; the decision "+
			"call this service makes is one anyone who has read the source can make",
			"fix", "-site-key/-secret-key")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// The decision engine is reached over HTTP with the secret key —
	// the dogfooding that makes this a demonstration rather than an
	// illustration.
	decisions := web.NewHTTPDecisionClient(engine, *secretKey, *decisionTO)

	var capture web.CaptureSink = web.NoCapture{}
	if *captureLog != "" {
		capture = web.NewFileCaptureSink(*captureLog)
		log.Info("human capture enabled", "log", *captureLog)
	}

	handler, err := web.New(web.Config{SiteKey: *siteKey, APIBase: *apiBase},
		decisions, capture, log)
	if err != nil {
		return fmt.Errorf("demo handler: %w", err)
	}

	mux := http.NewServeMux()
	handler.Register(mux)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	// One registry per process: the HTTP series and every domain
	// counter are exposed together, because two registries behind one
	// endpoint would need two encoders and would drop whichever the
	// handler forgot.
	reg := metrics.New()
	httpMetrics := middleware.NewMetrics(reg)
	mux.Handle("GET /metrics", reg.Handler())
	srv := &http.Server{
		Addr: *addr,
		Handler: middleware.Chain(mux,
			middleware.RequestID(),
			middleware.Recovery(log),
			middleware.Logging(log),
			httpMetrics.Collect(),
		),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", *addr, "api", *apiBase, "engine", engine)
		log.Info("demo page", "url", "http://"+*addr+"/")
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
