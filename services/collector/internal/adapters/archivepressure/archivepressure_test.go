package archivepressure_test

import (
	"errors"
	"testing"
	"time"

	"github.com/FabioCaffarello/ghost-trace/libs/eventstream"
	"github.com/FabioCaffarello/ghost-trace/services/collector/internal/adapters/archivepressure"
	"github.com/FabioCaffarello/ghost-trace/services/collector/internal/app"
)

func TestNoReadingIsNotCalm(t *testing.T) {
	// The property the whole type is arranged around. Zero would mean
	// "the archive is completely caught up", and guessing that wrong is
	// the expensive direction: the collector would keep handing records
	// to a queue about to discard them.
	p := archivepressure.New(nil, time.Now)
	if lvl := p.Level(); lvl >= 0 {
		t.Errorf("Level() = %v before any reading, want negative", lvl)
	}
	if p.Level() >= app.SheddingThreshold {
		t.Error("an unknown level must not trigger shedding either; with no " +
			"signal the collector behaves as it did before this existed")
	}
}

func TestAFailedPollDoesNotMoveTheLevel(t *testing.T) {
	// An unreachable broker is not evidence about the backlog in either
	// direction. It must not lower the level (which would stop shedding
	// during an outage) and must not raise it (which would start
	// shedding because monitoring broke).
	p := archivepressure.New(nil, time.Now)
	p.Observe(eventstream.Stats{OldestAge: 9 * time.Hour, MaxAge: 10 * time.Hour}, nil)
	before := p.Level()

	p.Observe(eventstream.Stats{}, errors.New("broker unreachable"))
	if got := p.Level(); got != before {
		t.Errorf("a failed poll moved the level from %v to %v", before, got)
	}
}

func TestTheLevelIsTheFractionOfTheRetentionWindow(t *testing.T) {
	p := archivepressure.New(nil, time.Now)
	p.Observe(eventstream.Stats{OldestAge: 42 * time.Hour, MaxAge: 168 * time.Hour}, nil)
	if got := p.Level(); got < 0.24 || got > 0.26 {
		t.Errorf("Level() = %v for a quarter-full window, want ~0.25", got)
	}
}

func TestAStaleReadingStopsBeingUsed(t *testing.T) {
	// Acting on the past is how a collector keeps shedding after the
	// archive recovered — or stops shedding because the last thing it
	// heard was calm.
	now := time.Now()
	clock := func() time.Time { return now }
	p := archivepressure.New(nil, clock)

	p.Observe(eventstream.Stats{OldestAge: 9 * time.Hour, MaxAge: 10 * time.Hour}, nil)
	if p.Level() < app.SheddingThreshold {
		t.Fatalf("a fresh 0.9 reading should shed, got %v", p.Level())
	}

	now = now.Add(archivepressure.Stale + time.Second)
	if lvl := p.Level(); lvl >= 0 {
		t.Errorf("Level() = %v for a reading %v old, want unknown",
			lvl, archivepressure.Stale)
	}
}

func TestAStreamWithNoRetentionWindowYieldsNoReading(t *testing.T) {
	// Dividing by a zero MaxAge would be +Inf, which compares greater
	// than the threshold and would shed everything, forever.
	p := archivepressure.New(nil, time.Now)
	p.Observe(eventstream.Stats{OldestAge: time.Hour, MaxAge: 0}, nil)
	if lvl := p.Level(); lvl >= 0 {
		t.Errorf("Level() = %v with no retention window, want unknown", lvl)
	}
}
