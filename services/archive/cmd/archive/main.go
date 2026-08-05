// Command archive stores Category I records arriving off the event
// stream.
//
// It owns exactly one thing: turning a stream of canonical bytes into a
// durable, content-addressed archive. It serves no API — /healthz and
// /metrics only — because everything it knows is reachable through the
// records themselves, and giving it a query surface before anything
// queries it would be inventing a contract nobody has asked for.
package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/FabioCaffarello/ghost-trace/libs/eventstream"
	"github.com/FabioCaffarello/ghost-trace/libs/metrics"
	"github.com/FabioCaffarello/ghost-trace/libs/middleware"
	"github.com/FabioCaffarello/ghost-trace/libs/substrate"

	"github.com/FabioCaffarello/ghost-trace/services/archive/internal/consumer"
)

func main() {
	var (
		natsURL = flag.String("nats", envOr("GT_NATS_URL", "nats://127.0.0.1:4222"),
			"NATS URL carrying the event stream")
		dataDir = flag.String("data", envOr("GT_DATA", "./.archive-data"),
			"directory for the SQLite index and blob store")
		addr = flag.String("addr", envOr("GT_ADDR", "127.0.0.1:8081"),
			"address for /healthz and /metrics")
	)
	flag.Parse()

	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	sub, err := substrate.Open(ctx,
		filepath.Join(*dataDir, "events.db"), filepath.Join(*dataDir, "blobs"))
	if err != nil {
		log.Error("open substrate", "err", err, "dir", *dataDir)
		os.Exit(1)
	}
	defer func() { _ = sub.Close() }()

	nc, js, err := eventstream.Connect(*natsURL, "ghost-trace-archive")
	if err != nil {
		log.Error("connect event stream", "err", err, "url", *natsURL)
		os.Exit(1)
	}
	defer nc.Close()

	// Declared by whichever side starts first, so compose does not have
	// to order the collector and the archive against each other.
	if err := eventstream.EnsureStream(ctx, js); err != nil {
		log.Error("ensure stream", "err", err)
		os.Exit(1)
	}

	cons := consumer.New(sub, time.Now, log)

	// One registry per process: the HTTP series and every domain
	// counter are exposed together, because two registries behind one
	// endpoint would need two encoders and would drop whichever the
	// handler forgot.
	reg := metrics.New()
	httpMetrics := middleware.NewMetrics(reg)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
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
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("serve", "err", err)
		}
	}()

	log.Info("archive consuming", "nats", *natsURL, "data", *dataDir, "addr", *addr)

	if err := eventstream.Consume(ctx, js, cons.Handle); err != nil {
		log.Error("consume", "err", err)
	}

	committed, rejected := cons.Counts()
	log.Info("archive stopping", "committed", committed, "rejected", rejected)

	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdown)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
