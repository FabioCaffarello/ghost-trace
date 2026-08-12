// Package archivepressure turns the broker's view of the archive's
// progress into the one number the collector acts on.
//
// The collector cannot compute this itself. It knows what it published
// and nothing about whether any of it was stored — local counters climb
// while a backlog builds, and the first news of trouble is records
// ageing out. The archive publishes the readings that answer it; this
// takes the same readings from the broker directly, so the ingest path
// never depends on another service answering HTTP.
package archivepressure

import (
	"sync"
	"time"

	"github.com/FabioCaffarello/ghost-trace/libs/eventstream"
	"github.com/FabioCaffarello/ghost-trace/libs/metrics"
)

// Unknown is the level reported when there is no usable reading.
//
// Negative rather than zero, and that distinction is the whole reason
// this type is careful: a poll that failed, a stream with no retention
// window, or a watcher that has not run yet are all states in which
// nothing is known — and zero would mean "the archive is completely
// caught up", which is the most dangerous thing to guess wrong.
const Unknown = -1.0

// Stale is how long a reading stays usable.
//
// Longer than any sane poll interval and far shorter than the retention
// window it describes. A reading nobody refreshed is a reading of the
// past, and acting on the past is how a collector keeps shedding after
// the archive recovered — or worse, stops shedding because the last
// thing it heard was calm.
const Stale = 90 * time.Second

// Pressure is the archive's backlog as a fraction of the stream's
// retention window, refreshed by a watcher and read by the ingest path.
type Pressure struct {
	now func() time.Time

	mu     sync.RWMutex
	level  float64
	readAt time.Time
}

// New returns a Pressure reporting Unknown until its first reading.
func New(reg *metrics.Registry, now func() time.Time) *Pressure {
	if now == nil {
		now = time.Now
	}
	p := &Pressure{now: now, level: Unknown}

	// Published so the decision is auditable from outside: when a run
	// shows shed records, this is what the collector believed at the
	// time. Read at scrape rather than mirrored, so it cannot drift
	// from what Level() returns.
	if reg != nil {
		reg.GaugeFunc("archive_pressure",
			"How close the archive is to losing its backlog, as a fraction of the "+
				"stream's retention window, as the COLLECTOR sees it. Negative means "+
				"no usable reading — which is not zero.",
			p.Level)
	}
	return p
}

// Observe records one reading, or the fact that one could not be taken.
//
// A failed poll does not lower the level and does not raise it: it
// simply stops refreshing, and the reading goes stale on its own. That
// is deliberate — an unreachable broker is not evidence about the
// archive's backlog in either direction.
func (p *Pressure) Observe(st eventstream.Stats, err error) {
	if err != nil || st.MaxAge <= 0 {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.level = float64(st.OldestAge) / float64(st.MaxAge)
	p.readAt = p.now()
}

// Level reports the most recent reading, or Unknown if there is none or
// it has gone stale.
func (p *Pressure) Level() float64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.readAt.IsZero() || p.now().Sub(p.readAt) > Stale {
		return Unknown
	}
	return p.level
}
