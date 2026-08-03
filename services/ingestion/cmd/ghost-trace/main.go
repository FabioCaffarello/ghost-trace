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
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/api"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/policy"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/session"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/web"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "ghost-trace: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		addr       = flag.String("addr", "127.0.0.1:8080", "listen address")
		dataDir    = flag.String("data", "", "substrate directory; empty disables the raw event archive")
		mode       = flag.String("mode", policy.ModeMonitor, "monitor | enforce")
		tenantID   = flag.String("tenant", "t_demo", "tenant id")
		siteKey    = flag.String("site-key", "pk_demo", "public site key, embedded in the page")
		secretKey  = flag.String("secret-key", "sk_demo", "secret key for server-to-server decision calls")
		pointerHz  = flag.Int("pointer-hz", 20, "collect policy: pointer sample rate")
		batchMs    = flag.Int("batch-ms", 2000, "collect policy: telemetry batch interval")
		ttl        = flag.Duration("session-ttl", 30*time.Minute, "session token lifetime")
		captureLog = flag.String("capture-log", "", "JSONL file of labelled human sessions for the M2 study; empty disables capture")
	)
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if *mode != policy.ModeMonitor && *mode != policy.ModeEnforce {
		return fmt.Errorf("unknown -mode %q (want %q or %q)", *mode, policy.ModeMonitor, policy.ModeEnforce)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Raw event archive. Optional: the decision path does not read from
	// it, so a slice can run entirely in memory. It exists so M2 can
	// re-score recorded sessions when a threshold moves.
	var archive *ingest.Ingester
	if *dataDir != "" {
		sub, err := substrate.Open(ctx, *dataDir+"/events.db", *dataDir+"/blobs")
		if err != nil {
			return fmt.Errorf("open substrate: %w", err)
		}
		defer func() { _ = sub.Close() }()
		archive = ingest.New(sub, time.Now)
		log.Info("raw event archive enabled", "dir", *dataDir)
	} else {
		log.Warn("raw event archive disabled; sessions will not be replayable (-data to enable)")
	}

	sessions := session.NewStore(*ttl, time.Now)

	apiSrv := api.New(api.Config{
		TenantID:  *tenantID,
		SiteKey:   *siteKey,
		SecretKey: *secretKey,
		Mode:      *mode,
		CollectPolicy: api.CollectPolicy{
			PointerHz: *pointerHz,
			BatchMs:   *batchMs,
			Types:     []string{"pointer", "key", "scroll", "focus", "visibility", "form"},
		},
	}, sessions, archive, time.Now, log)

	mux := apiSrv.Routes()

	demo, err := web.New(web.Config{
		APIBase:         "http://" + *addr,
		SiteKey:         *siteKey,
		SecretKey:       *secretKey,
		DecisionTimeout: 250 * time.Millisecond,
		CaptureLog:      *captureLog,
	}, log)
	if err != nil {
		return fmt.Errorf("demo handler: %w", err)
	}
	demo.Register(mux)
	if *captureLog != "" {
		log.Info("human capture enabled", "log", *captureLog)
	}

	// Expired sessions are never otherwise removed, so without this the
	// store grows for the life of the process.
	go sweepLoop(ctx, sessions, log)

	srv := &http.Server{
		Addr:              *addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", *addr, "mode", *mode, "policy", policy.Ref)
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
