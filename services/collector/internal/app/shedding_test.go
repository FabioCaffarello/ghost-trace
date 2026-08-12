package app_test

// The admission decision, tested without a broker.
//
// This is the first place the collector acts on something it learned
// about ANOTHER service, so the tests are mostly about restraint: what
// it does when it knows nothing, and what it refuses to infer.

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/FabioCaffarello/ghost-trace/services/collector/internal/app"
	"github.com/FabioCaffarello/ghost-trace/services/collector/internal/session"
	"google.golang.org/protobuf/proto"
)

type countingArchive struct {
	appended int
	err      error
}

func (a *countingArchive) Append(context.Context, proto.Message, int64) error {
	a.appended++
	return a.err
}

type fixedPressure float64

func (p fixedPressure) Level() float64 { return float64(p) }

type recordingLoss struct{ dropped map[string]int }

func (l *recordingLoss) Written(string) {}
func (l *recordingLoss) Dropped(_, reason string) {
	if l.dropped == nil {
		l.dropped = map[string]int{}
	}
	l.dropped[reason]++
}

func newShedApp(t *testing.T, arch app.EventArchive, p app.ArchivePressure,
	loss app.LossMeter) *app.App {
	t.Helper()
	return app.New(app.Config{}, session.NewStore(time.Minute, time.Now), arch,
		time.Now, slog.New(slog.NewTextHandler(io.Discard, nil))).
		WithLossMeter(loss).
		WithArchivePressure(p)
}

func TestAnUnknownPressureShedsNothing(t *testing.T) {
	// The collector must behave exactly as it did before this signal
	// existed when it has no signal — no broker, watcher not bound yet,
	// or the reading gone stale.
	arch := &countingArchive{}
	loss := &recordingLoss{}
	a := newShedApp(t, arch, app.NoPressure{}, loss)

	if _, err := a.StartSession(context.Background(), app.StartSessionInput{
		TenantID: "t", PagePath: "/",
	}); err != nil {
		t.Fatal(err)
	}
	if arch.appended == 0 {
		t.Error("nothing was offered to the archive while pressure was unknown; " +
			"an absent signal must not read as a full queue")
	}
	if n := loss.dropped[app.ReasonShed]; n != 0 {
		t.Errorf("shed %d records with no reading at all", n)
	}
}

func TestPressureBelowTheThresholdStillArchives(t *testing.T) {
	// Behind is not the same as out of runway. Every record handed over
	// below the threshold is one the archive will still get to.
	arch := &countingArchive{}
	loss := &recordingLoss{}
	a := newShedApp(t, arch, fixedPressure(app.SheddingThreshold-0.01), loss)

	if _, err := a.StartSession(context.Background(), app.StartSessionInput{
		TenantID: "t", PagePath: "/",
	}); err != nil {
		t.Fatal(err)
	}
	if arch.appended == 0 {
		t.Error("shed just below the threshold")
	}
}

func TestAtTheThresholdTheRecordIsShedAndCounted(t *testing.T) {
	// The whole point: it is a drop either way, and this one is
	// attributed to a decision instead of arriving as an age-out
	// nobody chose.
	arch := &countingArchive{}
	loss := &recordingLoss{}
	a := newShedApp(t, arch, fixedPressure(app.SheddingThreshold), loss)

	if _, err := a.StartSession(context.Background(), app.StartSessionInput{
		TenantID: "t", PagePath: "/",
	}); err != nil {
		t.Fatalf("shedding must never fail the request: %v", err)
	}
	if arch.appended != 0 {
		t.Errorf("offered %d records to an archive that is out of runway",
			arch.appended)
	}
	if n := loss.dropped[app.ReasonShed]; n != 1 {
		t.Errorf("shed count = %d, want 1 — a silent shed is the loss this "+
			"project spent a phase making countable", n)
	}
	if n := loss.dropped[app.ReasonError] + loss.dropped[app.ReasonDeadline]; n != 0 {
		t.Errorf("a deliberate shed was attributed to failure (%d)", n)
	}
}

func TestSheddingNeverFailsTheRequest(t *testing.T) {
	// §5 fail-open. Shedding is about what the archive is offered, not
	// about whether the caller is served.
	a := newShedApp(t, &countingArchive{err: errors.New("unused")},
		fixedPressure(1.0), &recordingLoss{})
	if _, err := a.StartSession(context.Background(), app.StartSessionInput{
		TenantID: "t", PagePath: "/",
	}); err != nil {
		t.Errorf("a fully backed-up archive failed a session start: %v", err)
	}
}
