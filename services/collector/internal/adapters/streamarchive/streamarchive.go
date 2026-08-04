// Package streamarchive is the transitional dual-write: every record
// still goes to the local substrate, and a copy is published onto the
// event stream for the archive service to store.
//
// THE LOCAL SUBSTRATE REMAINS AUTHORITATIVE. Publication failures are
// counted and logged, never returned. That asymmetry is the point of
// the transition: adding a broker must not make the service less
// reliable than it was without one, and until the archive service has
// been proven to hold everything the substrate holds, a NATS outage
// taking down /v1/outcomes would be a regression bought with an
// architecture diagram.
//
// PR-2.5 removes the local write once parity is demonstrated. Until
// then the counters below are the evidence for that decision: if
// published and dropped do not add up to appended, the stream is
// lossier than the disk and the cutover is not yet earned.
package streamarchive

import (
	"context"
	"log/slog"
	"sync/atomic"

	"google.golang.org/protobuf/proto"

	"github.com/FabioCaffarello/ghost-trace/libs/eventstream"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/libs/genproto/events/v1"
)

// Publisher is the subset of eventstream.Publisher this needs, named
// here so tests can substitute a failing one without a broker.
type Publisher interface {
	Publish(ctx context.Context, rec *eventsv1.ArchiveRecord) error
}

// Archive is an app.EventArchive that writes through to a local
// archive and mirrors onto the stream.
type Archive struct {
	local  LocalArchive
	pub    Publisher
	tenant string
	log    *slog.Logger

	appended  atomic.Int64
	published atomic.Int64
	dropped   atomic.Int64
}

// LocalArchive is the authoritative side: the existing substrate-backed
// adapter, unchanged.
type LocalArchive interface {
	Append(ctx context.Context, msg proto.Message, eventTime int64) error
}

// New wraps local, mirroring every successful append onto pub.
func New(local LocalArchive, pub Publisher, tenant string, log *slog.Logger) *Archive {
	if log == nil {
		log = slog.Default()
	}
	return &Archive{local: local, pub: pub, tenant: tenant, log: log}
}

// Append writes locally, then mirrors.
//
// The order matters: the local write is what the caller's error
// reflects, so a record is never acknowledged to a user before it is
// durable somewhere this process controls. Mirroring a record that
// failed to land locally would put the stream ahead of the disk and
// make the parity check meaningless in the wrong direction.
func (a *Archive) Append(ctx context.Context, msg proto.Message, eventTime int64) error {
	if err := a.local.Append(ctx, msg, eventTime); err != nil {
		return err
	}
	a.appended.Add(1)

	// The envelope is built by eventstream.Record, which the decision
	// engine's stream-only archive also uses. Only the POLICY differs
	// between the two — that one reports a publish failure and this one
	// counts it — and a second copy of the envelope construction would
	// have made the two records differ in more than policy.
	rec, err := eventstream.Record(msg, eventTime, a.tenant)
	if err != nil {
		a.drop(ctx, "canonicalize", err, "")
		return nil
	}
	if err := a.pub.Publish(ctx, rec); err != nil {
		a.drop(ctx, "publish", err, rec.EventHash)
		return nil
	}
	a.published.Add(1)
	return nil
}

func (a *Archive) drop(ctx context.Context, stage string, err error, hash string) {
	a.dropped.Add(1)
	a.log.WarnContext(ctx, "event stream mirror dropped a record",
		"stage", stage, "err", err, "event_hash", hash,
		"dropped_total", a.dropped.Load())
}

// Counts reports the mirror's own arithmetic. published + dropped
// should equal appended; a gap means a record went to disk and neither
// reached the stream nor was counted as lost, which is the one outcome
// this adapter must not produce quietly.
func (a *Archive) Counts() (appended, published, dropped int64) {
	return a.appended.Load(), a.published.Load(), a.dropped.Load()
}
