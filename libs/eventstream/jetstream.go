package eventstream

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"google.golang.org/protobuf/proto"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/libs/genproto/events/v1"
)

// Connect dials NATS and returns a JetStream handle.
//
// The options are chosen for a service that must survive the broker
// being restarted underneath it: reconnect forever rather than give up,
// because a collector that stops publishing after a broker blip is a
// collector losing records for reasons nobody will notice until the
// archive is short.
func Connect(url, name string) (*nats.Conn, jetstream.JetStream, error) {
	nc, err := nats.Connect(url,
		nats.Name(name),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(time.Second),
		nats.ReconnectBufSize(8*1024*1024),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("eventstream: connect %s: %w", url, err)
	}
	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, nil, fmt.Errorf("eventstream: jetstream: %w", err)
	}
	return nc, js, nil
}

// EnsureStream declares the stream idempotently.
//
// Both the producer and the consumer call it, deliberately: whichever
// starts first creates it, and neither has to be ordered after the
// other in a compose file. Retention is limits-based — records are
// dropped by age or size, never by having been acknowledged, because an
// archive that has not caught up must still be able to.
func EnsureStream(ctx context.Context, js jetstream.JetStream) error {
	_, err := js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:        Stream,
		Description: "Ghost Trace Category I records in flight to the archive.",
		Subjects:    []string{AllSubjects()},
		Retention:   jetstream.LimitsPolicy,
		Storage:     jetstream.FileStorage,
		MaxAge:      7 * 24 * time.Hour,
		Discard:     jetstream.DiscardOld,
		// Duplicate suppression by Nats-Msg-Id, which Publish sets to
		// the record's content hash. A retried publish of the same
		// record is collapsed by the broker rather than delivered
		// twice — the archive is idempotent anyway, so this is about
		// not paying for the redelivery.
		Duplicates: 5 * time.Minute,
	})
	if err != nil {
		return fmt.Errorf("eventstream: ensure stream %s: %w", Stream, err)
	}
	return nil
}

// Publisher writes records onto the stream.
type Publisher struct {
	js jetstream.JetStream
}

// NewPublisher wraps a JetStream handle.
func NewPublisher(js jetstream.JetStream) *Publisher { return &Publisher{js: js} }

// Publish sends one record to the subject its own contents name.
//
// The message id is the record's content hash, so a publish retried
// after an ambiguous failure is deduplicated by the broker instead of
// arriving twice.
func (p *Publisher) Publish(ctx context.Context, rec *eventsv1.ArchiveRecord) error {
	subject, err := SubjectFor(rec)
	if err != nil {
		return err
	}
	body, err := proto.Marshal(rec)
	if err != nil {
		return fmt.Errorf("eventstream: marshal record: %w", err)
	}
	msg := &nats.Msg{
		Subject: subject,
		Data:    body,
		Header:  nats.Header{jetstream.MsgIDHeader: []string{rec.GetEventHash()}},
	}
	if _, err := p.js.PublishMsg(ctx, msg); err != nil {
		return fmt.Errorf("eventstream: publish %s: %w", subject, err)
	}
	return nil
}

// ConsumerName is the durable this stream's archive binds to. Durable
// so that an archive restarting resumes where it stopped rather than
// replaying seven days of records or, worse, skipping to the end.
const ConsumerName = "archive"

// Consume delivers every record to fn, acknowledging only after fn
// returns nil.
//
// Explicit ack after a successful commit is the whole delivery
// contract: a record whose commit failed is redelivered, and one that
// was committed twice is collapsed by the substrate's content
// addressing. At-least-once plus idempotency, which is the only pair
// that survives a crash between the two.
// Stats is what the BROKER knows about a consumer's progress, as
// opposed to what the consumer knows about itself.
//
// The difference matters. A consumer can count what it committed
// without ever learning that a thousand records are queued behind it —
// the local counters go up, everything looks healthy, and the lag is
// invisible until records age out of the stream.
type Stats struct {
	// Pending: accepted by the stream, not yet acknowledged by us.
	Pending uint64

	// AckPending: delivered and awaiting an ack right now.
	AckPending int

	// Redelivered: messages this consumer has been sent more than once.
	// A non-zero value is not a fault — at-least-once delivery means
	// redelivery is the mechanism, not the exception — but a growing
	// one says commits are failing.
	Redelivered int

	// OldestAge is how old the oldest message still in the stream is.
	//
	// Not the oldest UNACKED one, which would need a fetch per poll;
	// this bounds it from above, because the oldest unacked cannot be
	// older than the oldest present. An upper bound is the right
	// direction for a safety signal: it warns early rather than late.
	//
	// Read together with Pending and MaxAge, it answers the question
	// age-out actually poses — how close is the backlog to being
	// discarded — while there is still time to act on the answer.
	OldestAge time.Duration

	// MaxAge is the stream's retention window, exposed so the ratio
	// that matters can be computed rather than hardcoded into whatever
	// dashboard happens to be reading.
	MaxAge time.Duration
}

// StatsFunc receives each reading, or the error that prevented one.
//
// The error is passed rather than swallowed because a reading that
// could not be taken is not a reading of zero. A caller that treats a
// failed poll as "no lag" reports perfect health at exactly the moment
// the broker stopped answering.
type StatsFunc func(Stats, error)

type consumeOptions struct {
	statsEvery time.Duration
	stats      StatsFunc
}

// ConsumeOption configures Consume.
type ConsumeOption func(*consumeOptions)

// WithStats polls the broker for this consumer's progress and reports
// each reading to fn.
func WithStats(every time.Duration, fn StatsFunc) ConsumeOption {
	return func(o *consumeOptions) {
		o.statsEvery, o.stats = every, fn
	}
}

func Consume(ctx context.Context, js jetstream.JetStream,
	fn func(context.Context, *eventsv1.ArchiveRecord) error,
	opts ...ConsumeOption) error {

	var o consumeOptions
	for _, apply := range opts {
		apply(&o)
	}

	cons, err := js.CreateOrUpdateConsumer(ctx, Stream, jetstream.ConsumerConfig{
		Durable:       ConsumerName,
		Description:   "The archive service.",
		AckPolicy:     jetstream.AckExplicitPolicy,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckWait:       30 * time.Second,
		MaxDeliver:    -1,
		FilterSubject: AllSubjects(),
	})
	if err != nil {
		return fmt.Errorf("eventstream: consumer: %w", err)
	}

	sub, err := cons.Consume(func(msg jetstream.Msg) {
		var rec eventsv1.ArchiveRecord
		if err := proto.Unmarshal(msg.Data(), &rec); err != nil {
			// Undecodable: redelivering forever would wedge the
			// consumer on one bad message, and this cannot become
			// decodable later. Terminate drops it from redelivery and
			// leaves it in the stream for a human.
			_ = msg.Term()
			return
		}
		if err := fn(ctx, &rec); err != nil {
			// Nak rather than drop: a commit that failed is a record
			// the archive does not have, and the whole point is that it
			// ends up having it.
			_ = msg.Nak()
			return
		}
		_ = msg.Ack()
	})
	if err != nil {
		return fmt.Errorf("eventstream: consume: %w", err)
	}
	defer sub.Stop()

	if o.stats != nil && o.statsEvery > 0 {
		go pollStats(ctx, js, cons, o.statsEvery, o.stats)
	}

	<-ctx.Done()
	if errors.Is(ctx.Err(), context.Canceled) {
		return nil
	}
	return ctx.Err()
}

// pollStats reports the broker's view of this consumer until ctx ends.
//
// The first reading is taken immediately rather than after one
// interval, so a process that starts behind says so at once instead of
// looking healthy for its first tick.
func pollStats(ctx context.Context, js jetstream.JetStream, cons jetstream.Consumer,
	every time.Duration, report StatsFunc) {

	read := func() {
		info, err := cons.Info(ctx)
		if err != nil {
			report(Stats{}, err)
			return
		}

		// The stream's own state, for the retention window and the age
		// of what is still in it. A failure here fails the whole
		// reading rather than filling half of it: a Stats with a real
		// Pending and a zero OldestAge would be read as a healthy
		// backlog, which is the opposite of what a missing answer
		// means.
		st, err := js.Stream(ctx, Stream)
		if err != nil {
			report(Stats{}, err)
			return
		}
		si, err := st.Info(ctx)
		if err != nil {
			report(Stats{}, err)
			return
		}

		var oldest time.Duration
		if !si.State.FirstTime.IsZero() && si.State.Msgs > 0 {
			oldest = time.Since(si.State.FirstTime)
		}

		// TWO WAYS OF COUNTING AGE-OUT AFTER THE FACT WERE BUILT HERE
		// AND BOTH WERE DELETED, because both were measured and neither
		// fires when it matters.
		//
		// Comparing the stream's first sequence to the consumer's ack
		// floor should reveal records that left unacknowledged, and
		// would survive a restart. It cannot: the broker ADVANCES the
		// ack floor when messages are removed from under a consumer,
		// since there is nothing left to acknowledge. Purging four
		// unconsumed records left first_seq=15 and ack_floor=14.
		//
		// Counting jumps in delivered sequence should reveal the same
		// thing from this side. It does not either: a gap requires
		// records to leave the stream without being delivered, and this
		// consumer is dispatched concurrently and keeps up — in a test
		// built to force it, the messages were delivered before the
		// purge and no jump existed. The case it would catch is a
		// consumer running far enough behind that discard overtakes it,
		// which needs load this repository has not built.
		//
		// Publishing a counter that reads zero through real loss would
		// be worse than publishing none, so neither ships. What ships
		// is the early warning below; see the archive's meter.
		report(Stats{
			Pending:     info.NumPending,
			AckPending:  info.NumAckPending,
			Redelivered: info.NumRedelivered,
			OldestAge:   oldest,
			MaxAge:      si.Config.MaxAge,
		}, nil)
	}

	read()
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			read()
		}
	}
}
