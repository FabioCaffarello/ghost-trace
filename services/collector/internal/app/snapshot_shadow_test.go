// The shadow: a session judged through the snapshot store decides
// exactly what the collector decides in process.
//
// PR-2.3a proved the MAPPING preserves decisions, with no
// infrastructure in the way. This proves the same thing through the
// real path a decision engine will take — the collector writes on
// telemetry, a reader fetches from a live KV bucket, and the two
// judgements are compared.
//
// It needs a broker. Without GT_NATS_URL it SKIPS rather than passes,
// because a shadow test that quietly does nothing is the vacuous green
// this repository keeps finding.
package app_test

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/FabioCaffarello/ghost-trace/libs/decision"
	"github.com/FabioCaffarello/ghost-trace/libs/eventstream"
	"github.com/FabioCaffarello/ghost-trace/libs/policy"
	"github.com/FabioCaffarello/ghost-trace/libs/snapshot"
	"github.com/FabioCaffarello/ghost-trace/services/collector/internal/adapters/livesessions"
	"github.com/FabioCaffarello/ghost-trace/services/collector/internal/app"
	"github.com/FabioCaffarello/ghost-trace/services/collector/internal/session"
)

func quiet() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// linearBatch is the naive-automation shape: a straight constant
// velocity drag, which scores high and therefore sits where a
// difference would show.
func linearBatch(token string, seq uint32, n int) app.TelemetryEnvelope {
	pts := make([][3]int32, 0, n)
	for i := 0; i < n; i++ {
		dt := int32(16)
		if i == 0 {
			dt = 0
		}
		pts = append(pts, [3]int32{int32(100 + i*10), 120, dt})
	}
	start := uint32(1200 + seq*5000)
	return app.TelemetryEnvelope{
		SessionToken: token,
		Seq:          seq,
		SentAtMs:     start + uint32(n*16),
		PagePath:     "/login",
		Events: []app.TelemetryEvent{
			{Type: "pointer", T: start, Src: "mouse", Pts: pts},
			{Type: "key", T: start + 40, Phase: "down", KeyClass: "alpha", Target: "f_1"},
			{Type: "key", T: start + 70, Phase: "up", KeyClass: "alpha", Target: "f_1"},
			{Type: "form", T: start + 90, Action: "injected", Target: "f_1"},
		},
	}
}

func TestDecisionThroughTheSnapshotStoreMatchesInProcess(t *testing.T) {
	url := os.Getenv("GT_NATS_URL")
	if url == "" {
		t.Skip("GT_NATS_URL not set — start a broker to run the shadow test " +
			"(docker run -p 4222:4222 nats:alpine -js)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	nc, js, err := eventstream.Connect(url, "shadow-test")
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer nc.Close()

	store, err := eventstream.OpenSessions(ctx, js, 30*time.Minute)
	if err != nil {
		t.Fatalf("open sessions bucket: %v", err)
	}

	log := quiet()
	sessions := session.NewStore(30*time.Minute, time.Now)
	application := app.New(app.Config{TenantID: "t_shadow"},
		sessions, app.NullArchive{}, time.Now, log).
		WithSnapshots(store)

	// The collector's own decision path, mounted over the live store —
	// the same package the decision engine will mount over snapshots.
	decisions := decision.New(decision.Config{
		TenantID: "t_shadow", Mode: policy.ModeMonitor,
	}, livesessions.New(sessions), app.NullArchive{}, time.Now, log)

	out, err := application.StartSession(ctx, app.StartSessionInput{PagePath: "/login"})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	for seq := uint32(1); seq <= 3; seq++ {
		if err := application.IngestTelemetry(ctx, linearBatch(out.Token, seq, 40)); err != nil {
			t.Fatalf("ingest %d: %v", seq, err)
		}
	}

	// What the collector decides, holding the state in memory.
	inProcess, err := decisions.Decide(ctx, decision.Input{
		SessionToken: out.Token, Action: "login", SubjectID: "u_shadow",
	})
	if err != nil {
		t.Fatalf("decide in process: %v", err)
	}

	// What a decision engine would decide, holding only the snapshot —
	// fetched by the TOKEN, which is all a decision request carries.
	snap, err := store.Get(ctx, "t_shadow", out.Token)
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}

	j := policy.Judge(snapshot.ToState(snap.GetFeatures()), "login")
	shadow, err := policy.Apply(j, policy.ModeMonitor)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	if shadow.Decision != inProcess.Decision {
		t.Errorf("decision through the snapshot store is %q, in process it is %q",
			shadow.Decision, inProcess.Decision)
	}
	if shadow.Shadow != inProcess.ShadowDecision {
		t.Errorf("shadow decision differs: %q vs %q", shadow.Shadow, inProcess.ShadowDecision)
	}
	if got, want := j.Score(), inProcess.Score; !closeEnough(got, want) {
		t.Errorf("score through the store is %v, in process %v", got, want)
	}
	if got, want := j.Confidence(), inProcess.Confidence; !closeEnough(got, want) {
		t.Errorf("confidence through the store is %v, in process %v", got, want)
	}

	// The snapshot must describe the session that was actually
	// observed, not an empty one — otherwise the comparison above could
	// pass by both sides seeing nothing.
	if snap.GetLastEventMs() == 0 {
		t.Error("snapshot records no elapsed session time; it may be describing an empty session")
	}
	if snap.GetFeatures().GetPointerPoints() == 0 {
		t.Error("snapshot carries no pointer points; the batches did not reach it")
	}
}

// float32 on the wire, float64 in the domain — ADR-0004 measured the
// loss at under 1e-6 relative.
func closeEnough(a, b float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	m := a
	if b > m {
		m = b
	}
	if m == 0 {
		return d == 0
	}
	return d/m < 1e-6
}
