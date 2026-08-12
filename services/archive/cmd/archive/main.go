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
	"fmt"
	"log/slog"
	"net"
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

	"github.com/FabioCaffarello/ghost-trace/services/archive/internal/adapters/archivemeter"
	"github.com/FabioCaffarello/ghost-trace/services/archive/internal/consumer"
)

func main() {
	var (
		natsURL = flag.String("nats", envOr("GT_NATS_URL", "nats://127.0.0.1:4222"),
			"NATS URL carrying the event stream")
		dataDir = flag.String("data", envOr("GT_DATA", "./.archive-data"),
			"directory for the SQLite index and blob store")
		statsEvery = flag.Duration("stats-every", 10*time.Second,
			"how often to ask the broker how far behind this consumer is; the "+
				"reading's age is exposed as archive_stream_observed_timestamp_seconds")
		addr = flag.String("addr", envOr("GT_ADDR", "127.0.0.1:8081"),
			"address for /healthz and /metrics")
		health = flag.Bool("healthcheck", false,
			"probe /healthz on -addr and exit 0/1; the container health check execs "+
				"the binary because distroless ships no shell or curl")
	)
	flag.Parse()

	// The last service to get this flag, and its absence propagated
	// further than a missing flag should. Distroless has no shell, so a
	// compose healthcheck has to exec the binary — with no flag to exec,
	// the archive had no healthcheck; with no healthcheck, CI's topology
	// job had nothing to wait on and left it out of the wait list. The
	// gate reported green with the archive dead.
	if *health {
		if err := probeHealth(*addr); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

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

	// Binds rather than declares, so compose DOES have to order the
	// collector ahead of the archive — which is the trade PR-5.3 makes
	// deliberately. "Whichever side starts first declares it" meant the
	// consumer could choose the retention it is measured against, and
	// once the stream has a byte cap that is the backlog bound itself.
	if err := eventstream.OpenStream(ctx, js); err != nil {
		log.Error("open event stream", "err", err)
		os.Exit(1)
	}

	reg := metrics.New()
	meter := archivemeter.New(reg, time.Now)
	cons := consumer.New(sub, meter, time.Now, log)

	// One registry per process: the HTTP series and every domain
	// counter are exposed together, because two registries behind one
	// endpoint would need two encoders and would drop whichever the
	// handler forgot.
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

	// The broker's view of how far behind this consumer is, polled on a
	// timer. Local counters cannot see it: they go up while a thousand
	// records queue behind them, and the lag stays invisible until
	// records age out of the stream.
	//
	// The durable position is read on the same tick, and the two are
	// reported together on purpose. Age-out is the subtraction between
	// them — what the stream still holds, minus where this archive got to
	// — and reading them a poll apart would compare a fresh number with a
	// stale one at exactly the moment they diverge.
	onStats := func(st eventstream.Stats, err error) {
		meter.Observe(st, err)

		// One snapshot, not two reads. Separate queries let records
		// commit in between, and the published pair then says the
		// archive holds more rows than it performed commits.
		pos, rows, ok, perr := sub.PositionAndCount(ctx)
		meter.ObservePosition(pos, rows, ok, perr)
		if perr != nil {
			log.Error("read durable position", "err", perr)
			return
		}
		if err == nil {
			meter.ObserveSkipped(st.FirstSeq, pos, ok)
		}
	}

	if err := eventstream.Consume(ctx, js, cons.HandleBatch,
		eventstream.WithStats(*statsEvery, onStats),
		eventstream.WithUndecodable(cons.Undecodable)); err != nil {
		log.Error("consume", "err", err)
	}

	committed, rejected := cons.Counts()
	fields := []any{"committed", committed, "rejected", rejected}
	if pos, ok, err := sub.Position(context.Background()); err == nil && ok {
		// The durable figures, not the process ones. This line is the
		// last thing a stopping archive says, and it is the one an
		// operator reads after an incident.
		fields = append(fields,
			"position_first", pos.FirstSeq, "position_highest", pos.HighestSeq,
			"position_committed", pos.Committed, "position_rejected", pos.Rejected,
			"unaccounted", pos.Unaccounted())
	}
	log.Info("archive stopping", fields...)

	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdown)
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

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
