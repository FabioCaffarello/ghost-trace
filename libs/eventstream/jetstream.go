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

// StreamMaxAge and StreamMaxBytes bound the stream. See ADR-0012.
//
// AGE alone was the bound until PR-5.3, and age alone is not a bound on
// anything an operator controls. The archive commits about 4 133
// records/s against a collector that bends near 16 000, so a sustained
// overload writes the difference to disk for seven days — and at that
// surplus the volume fills in hours. When it does, NATS dies, and the
// collector dies with it, because compose makes it depend on a healthy
// broker. The system's worst failure was a disk-full, not a data loss.
//
// A byte cap converts that into the loss the design already accounts
// for: DiscardOld drops the oldest unread records, `stream_skipped`
// counts them, and the alert on oldest-age fires long before either.
// Bounded, counted, and survivable is strictly better than unbounded
// and fatal.
//
// 4 GiB is DERIVED, not chosen: it is the smallest round cap that holds
// more than an hour of backlog at the archive's measured drain rate of
// 4 133 records/s, taking a conservative 256 bytes per record (real
// ones measured 60 to 161). That works out at about 68 minutes — long
// enough that an archive restart, a redeploy, or an outage somebody is
// actively fixing does not cost records.
//
// The requirement is the decision; the number follows from it and from
// two measurements, and a deployment with a different disk or a
// different drain rate should recompute rather than copy it. The first
// attempt at this constant was 2 GiB, picked by taste, and the test
// below rejected it at 34 minutes.
const (
	StreamMaxAge   = 7 * 24 * time.Hour
	StreamMaxBytes = 4 << 30
)

// streamConfig is the declaration, in one place, so the owner and the
// binder cannot disagree about what they are describing.
func streamConfig() jetstream.StreamConfig {
	return jetstream.StreamConfig{
		Name:        Stream,
		Description: "Ghost Trace Category I records in flight to the archive.",
		Subjects:    []string{AllSubjects()},
		Retention:   jetstream.LimitsPolicy,
		Storage:     jetstream.FileStorage,
		MaxAge:      StreamMaxAge,
		MaxBytes:    StreamMaxBytes,
		Discard:     jetstream.DiscardOld,
		// Duplicate suppression by Nats-Msg-Id, which Publish sets to
		// the record's content hash. A retried publish of the same
		// record is collapsed by the broker rather than delivered
		// twice — the archive is idempotent anyway, so this is about
		// not paying for the redelivery.
		Duplicates: 5 * time.Minute,
	}
}

// ErrStreamLimitsMismatch is returned when the stream a reader binds to
// is not bounded the way that reader expects.
var ErrStreamLimitsMismatch = errors.New("eventstream: stream limits differ from this process's")

// EnsureStream declares the stream. THE PRODUCER CALLS THIS.
//
// Ownership follows the writer, the same rule the snapshot bucket
// already follows (EnsureSessions / OpenSessions). Until PR-5.3 all
// three services called CreateOrUpdateStream, so whichever started LAST
// silently rewrote the limits — including, once a byte cap exists, how
// much backlog the archive is allowed to fall behind by. A retention
// policy that depends on container start order is not a policy.
//
// Retention is limits-based: records are dropped by age or size, never
// by having been acknowledged, because an archive that has not caught
// up must still be able to.
func EnsureStream(ctx context.Context, js jetstream.JetStream) error {
	if _, err := js.CreateOrUpdateStream(ctx, streamConfig()); err != nil {
		return fmt.Errorf("eventstream: ensure stream %s: %w", Stream, err)
	}
	return nil
}

// OpenStream binds to the existing stream and refuses limits that are
// not the ones this process expects. READERS CALL THIS.
//
// Refusing rather than rewriting is the whole point, and it is the
// lesson the KV bucket taught first: a consumer that quietly accepted a
// different MaxBytes would be reasoning about a backlog bound it does
// not know, and `archive_stream_max_age_seconds` — which the alert
// rules divide by — would describe a limit nobody chose.
func OpenStream(ctx context.Context, js jetstream.JetStream) error {
	s, err := js.Stream(ctx, Stream)
	if err != nil {
		return fmt.Errorf("eventstream: open stream %s: %w (the collector creates "+
			"it; a reader starting first will see this until it has)", Stream, err)
	}
	info, err := s.Info(ctx)
	if err != nil {
		return fmt.Errorf("eventstream: info %s: %w", Stream, err)
	}
	want := streamConfig()
	if info.Config.MaxAge != want.MaxAge || info.Config.MaxBytes != want.MaxBytes {
		return fmt.Errorf("%w: stream says max_age=%s max_bytes=%d, this process "+
			"expects max_age=%s max_bytes=%d — the collector owns these and every "+
			"reader must be built from the same constants",
			ErrStreamLimitsMismatch,
			info.Config.MaxAge, info.Config.MaxBytes, want.MaxAge, want.MaxBytes)
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

// How much the consumer pulls at once, and how long it waits for a
// full batch before taking what it has.
//
// 128 is where the measured curve flattens: one record per transaction
// runs at 17 628/s on the development machine, a batch of 8 at 2.3x, of
// 128 at 3.2x and of 512 at only 3.5x. Past that the return is small
// and the redelivery window keeps growing, which is the cost side of
// the same trade.
//
// The wait bounds latency when the stream is nearly idle: a lone record
// arriving at 3am is committed within 250ms rather than waiting for 127
// companions that are not coming.
const (
	BatchSize = 128
	BatchWait = 250 * time.Millisecond

	// NakBackoff delays redelivery of a failed batch.
	//
	// A bare Nak asks for the message back immediately, and the errors
	// this consumer naks on are a full disk or a locked database —
	// conditions that do not clear in microseconds. Retrying at full
	// speed turns a failing resource into a hot loop against a failing
	// resource, which is how a recoverable problem becomes an
	// unrecoverable one.
	NakBackoff = 2 * time.Second
)

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

	// FirstSeq is the lowest sequence still IN the stream.
	//
	// It is the number that makes age-out countable after the fact, but
	// only against a position the broker does not control. Compared to a
	// consumer's ack floor it reveals nothing — the broker advances that
	// floor when records are removed, so the evidence is destroyed by the
	// event it would evidence. Compared to a position the archive wrote
	// down itself, the subtraction is sound: sequences between what the
	// archive has walked past and what the stream still holds left
	// without ever being delivered.
	FirstSeq uint64

	// LastSeq is the highest sequence the stream has accepted, so a
	// consumer's own position can be read as a distance from the head
	// rather than as a bare number.
	LastSeq uint64
}

// StatsFunc receives each reading, or the error that prevented one.
//
// The error is passed rather than swallowed because a reading that
// could not be taken is not a reading of zero. A caller that treats a
// failed poll as "no lag" reports perfect health at exactly the moment
// the broker stopped answering.
type StatsFunc func(Stats, error)

// Delivery is what the BROKER says about one message, as opposed to
// what the message says about itself.
//
// Sequence is the stream sequence: a monotonic number the broker assigns
// at publish, stable across redelivery of the same message. It is the
// only identifier that lets a consumer say what it has walked PAST as
// well as what it has taken in — and therefore the only basis on which
// records that entered the stream and never arrived can be counted at
// all. A consumer that sees only payloads can count what it received and
// never learn what it did not.
type Delivery struct {
	Sequence    uint64
	Redelivered bool
}

// Handler commits one delivered record.
type Handler func(context.Context, *eventsv1.ArchiveRecord, Delivery) error

// Item is one decoded delivery.
type Item struct {
	Record   *eventsv1.ArchiveRecord
	Delivery Delivery
}

// BatchHandler commits a whole fetch at once. Returning an error naks
// every message in it; returning nil acks every message in it.
//
// THE UNIT OF ACKNOWLEDGEMENT IS NOW THE BATCH, and that is the whole
// change. At-least-once still holds — a batch whose commit failed is
// redelivered in full — and the substrate's content addressing still
// collapses the duplicates that produces. What grows is the WINDOW: a
// crash mid-batch redelivers up to BatchSize records instead of one.
// That is affordable precisely because idempotency was never optional
// here, and it buys one fsync per batch instead of one per record.
type BatchHandler func(context.Context, []Item) error

type consumeOptions struct {
	statsEvery  time.Duration
	stats       StatsFunc
	undecodable func(Delivery)
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

// WithUndecodable reports messages this library drops without ever
// calling the handler.
//
// Without it those messages are an accounting hole: the sequence is
// consumed, the handler never hears about it, and a reconciliation built
// on sequences reports the record as unexplained loss. It is not
// unexplained — this library terminated it on purpose — and the
// difference between "the transport lost a record" and "the archive
// refused a corrupt one" is the difference the whole audit exists to
// draw.
func WithUndecodable(fn func(Delivery)) ConsumeOption {
	return func(o *consumeOptions) { o.undecodable = fn }
}

func Consume(ctx context.Context, js jetstream.JetStream, fn BatchHandler,
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
		// The pull loop below asks for BatchSize at a time; without a
		// ceiling on outstanding acks the broker would happily hand out
		// far more than one batch and the window this design bounds
		// would be unbounded again.
		MaxAckPending: 4 * BatchSize,
	})
	if err != nil {
		return fmt.Errorf("eventstream: consumer: %w", err)
	}

	if o.stats != nil && o.statsEvery > 0 {
		go pollStats(ctx, js, cons, o.statsEvery, o.stats)
	}

	// A PULL LOOP RATHER THAN A PUSH CALLBACK.
	//
	// The callback form delivers one message at a time, which is
	// exactly the shape that forced one transaction — and one fsync —
	// per record. Fetch hands over a slice, so the whole slice can be
	// one transaction and one ack decision.
	for {
		if err := ctx.Err(); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}

		msgs, err := cons.Fetch(BatchSize, jetstream.FetchMaxWait(BatchWait))
		if err != nil {
			if errors.Is(err, context.Canceled) || ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("eventstream: fetch: %w", err)
		}

		batch := make([]Item, 0, BatchSize)
		held := make([]jetstream.Msg, 0, BatchSize)

		for msg := range msgs.Messages() {
			// The broker's view of this delivery, read before anything
			// can fail. A message whose metadata cannot be read is one
			// whose sequence is unknown, and a zero sequence is refused
			// downstream rather than silently recorded as the beginning
			// of the stream.
			var d Delivery
			if md, err := msg.Metadata(); err == nil {
				d = Delivery{Sequence: md.Sequence.Stream, Redelivered: md.NumDelivered > 1}
			}

			var rec eventsv1.ArchiveRecord
			if err := proto.Unmarshal(msg.Data(), &rec); err != nil {
				// Undecodable: redelivering forever would wedge the
				// consumer on one bad message, and this cannot become
				// decodable later. Terminate drops it from redelivery
				// and leaves it in the stream for a human.
				//
				// Reported first, and NOT added to the batch: the
				// sequence is consumed either way, and a consumed
				// sequence nobody accounts for is indistinguishable
				// from a record the transport lost.
				if o.undecodable != nil {
					o.undecodable(d)
				}
				_ = msg.Term()
				continue
			}
			batch = append(batch, Item{Record: &rec, Delivery: d})
			held = append(held, msg)
		}
		if err := msgs.Error(); err != nil && ctx.Err() == nil {
			return fmt.Errorf("eventstream: fetch batch: %w", err)
		}
		if len(batch) == 0 {
			continue
		}

		if err := fn(ctx, batch); err != nil {
			// The batch did not commit, so none of it is acknowledged.
			// Delayed rather than immediate: the failures this naks on
			// — a full disk, a locked database — do not clear in
			// microseconds, and retrying at full speed is a hot loop
			// against whatever is already failing.
			for _, msg := range held {
				_ = msg.NakWithDelay(NakBackoff)
			}
			continue
		}
		for _, msg := range held {
			_ = msg.Ack()
		}
	}
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
		// fires when it matters. A third now works, and it works because
		// it stopped asking the broker where the consumer had got to —
		// see FirstSeq above and the archive's startup check.
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
		// be worse than publishing none, so neither ships. What ships is
		// the early warning, plus FirstSeq for the archive to subtract
		// its own durable position from.
		report(Stats{
			Pending:     info.NumPending,
			AckPending:  info.NumAckPending,
			Redelivered: info.NumRedelivered,
			OldestAge:   oldest,
			MaxAge:      si.Config.MaxAge,
			FirstSeq:    si.State.FirstSeq,
			LastSeq:     si.State.LastSeq,
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
