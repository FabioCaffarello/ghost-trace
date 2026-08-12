package eventstream

// The stream's shape, tested without a broker.
//
// White-box because streamConfig() is the single declaration both the
// owner and the binder read, and the point of these tests is that it
// stays single and stays finite.

import (
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

func TestTheStreamIsBoundedBySizeAndNotOnlyByAge(t *testing.T) {
	// Age alone is not a bound on anything an operator controls. At the
	// measured surplus — the archive commits ~4 133 records/s against a
	// collector that bends near 16 000 — a seven-day window fills the
	// disk in hours, and when it does NATS dies and the collector dies
	// with it. A byte cap converts that into an age-out the accounting
	// already describes. See ADR-0012.
	cfg := streamConfig()

	if cfg.MaxBytes <= 0 {
		t.Error("the stream has no MaxBytes. Sustained overload then writes to " +
			"disk until the disk ends, which takes the broker and the collector " +
			"with it — a strictly worse outcome than the bounded, counted loss " +
			"DiscardOld produces")
	}
	if cfg.MaxAge <= 0 {
		t.Error("the stream has no MaxAge")
	}
	if cfg.Discard != jetstream.DiscardOld {
		t.Errorf("Discard = %v, want DiscardOld — discarding NEW would refuse the "+
			"publish, which is a fail-closed answer to a problem this project "+
			"fails open on (contract §5, ADR-0012)", cfg.Discard)
	}
	// Limits-based, never interest-based: an archive that has not caught
	// up must still be able to read what it has not acknowledged.
	if cfg.Retention != jetstream.LimitsPolicy {
		t.Errorf("Retention = %v, want LimitsPolicy", cfg.Retention)
	}
}

func TestTheDeclarationIsStableAcrossCalls(t *testing.T) {
	// The owner declares from streamConfig() and every binder compares
	// against it. If it ever became a function of anything local — an
	// env var, a flag, a clock — the two would disagree for reasons
	// neither could see, which is the failure OpenStream exists to
	// catch and would then be causing.
	a, b := streamConfig(), streamConfig()
	if a.MaxAge != b.MaxAge || a.MaxBytes != b.MaxBytes || a.Name != b.Name {
		t.Fatal("streamConfig() is not deterministic; the owner and its readers " +
			"would be comparing different things")
	}
	if a.MaxAge != StreamMaxAge || a.MaxBytes != StreamMaxBytes {
		t.Error("streamConfig() does not use the exported constants, so a reader " +
			"built against them would refuse a stream the owner just created")
	}
}

func TestTheByteCapHoldsAPlausibleBacklog(t *testing.T) {
	// Not a tuned number and not claimed to be — but a cap small enough
	// to age out records the archive could have worked off would be
	// worse than none, so the order of magnitude is worth pinning.
	//
	// A real record measured 60 to 161 bytes. At the archive's measured
	// 4 133 records/s, the cap should hold well over an hour of full-rate
	// backlog.
	const worstCaseRecordBytes = 256
	const archiveRate = 4133

	seconds := float64(StreamMaxBytes) / (worstCaseRecordBytes * archiveRate)
	if seconds < time.Hour.Seconds() {
		t.Errorf("the byte cap holds %.0fs of backlog at the archive's measured "+
			"rate; under an hour is too tight to survive an outage the archive "+
			"could otherwise recover from", seconds)
	}
}
