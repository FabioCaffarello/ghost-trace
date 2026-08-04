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
func Consume(ctx context.Context, js jetstream.JetStream,
	fn func(context.Context, *eventsv1.ArchiveRecord) error) error {

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

	<-ctx.Done()
	if errors.Is(ctx.Err(), context.Canceled) {
		return nil
	}
	return ctx.Err()
}
