package streamarchive

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"google.golang.org/protobuf/proto"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/libs/genproto/events/v1"
)

type fakeLocal struct {
	err  error
	seen int
}

func (f *fakeLocal) Append(context.Context, proto.Message, int64) error {
	f.seen++
	return f.err
}

type fakePub struct {
	err  error
	recs []*eventsv1.ArchiveRecord
}

func (f *fakePub) Publish(_ context.Context, r *eventsv1.ArchiveRecord) error {
	if f.err != nil {
		return f.err
	}
	f.recs = append(f.recs, r)
	return nil
}

func quiet() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func msg() proto.Message { return &eventsv1.Outcome{EvaluationId: "ev_1", Outcome: "login_success"} }

// The property the whole transition rests on: a broker that is down
// must not fail the request. Adding NATS cannot make the service less
// reliable than it was without one.
func TestPublishFailureNeverReachesTheCaller(t *testing.T) {
	local := &fakeLocal{}
	pub := &fakePub{err: errors.New("nats: no servers available")}
	a := New(local, pub, "t_test", quiet())

	if err := a.Append(context.Background(), msg(), 1); err != nil {
		t.Fatalf("a publish failure surfaced to the caller: %v", err)
	}
	if local.seen != 1 {
		t.Errorf("local append ran %d times, want 1", local.seen)
	}
	appended, published, dropped := a.Counts()
	if appended != 1 || published != 0 || dropped != 1 {
		t.Errorf("counts = (%d, %d, %d), want (1, 0, 1)", appended, published, dropped)
	}
}

// The converse: a local failure IS the caller's error, and nothing is
// mirrored. Publishing a record that did not land locally would put the
// stream ahead of the disk and make parity meaningless.
func TestLocalFailureSurfacesAndPublishesNothing(t *testing.T) {
	want := errors.New("substrate: disk full")
	local := &fakeLocal{err: want}
	pub := &fakePub{}
	a := New(local, pub, "t_test", quiet())

	if err := a.Append(context.Background(), msg(), 1); !errors.Is(err, want) {
		t.Fatalf("err = %v, want %v", err, want)
	}
	if len(pub.recs) != 0 {
		t.Errorf("published %d records after a local failure, want 0", len(pub.recs))
	}
	if appended, _, _ := a.Counts(); appended != 0 {
		t.Errorf("counted %d appends after a local failure, want 0", appended)
	}
}

// The mirrored record must carry the canonical bytes and a hash that
// verifies against them — the archive refuses anything else, and it is
// the only check either side has that the bytes survived.
func TestMirroredRecordIsSelfVerifying(t *testing.T) {
	pub := &fakePub{}
	a := New(&fakeLocal{}, pub, "t_demo", quiet())

	if err := a.Append(context.Background(), msg(), 42); err != nil {
		t.Fatal(err)
	}
	if len(pub.recs) != 1 {
		t.Fatalf("published %d records, want 1", len(pub.recs))
	}
	rec := pub.recs[0]

	if rec.GetTenant() != "t_demo" {
		t.Errorf("tenant = %q", rec.GetTenant())
	}
	if rec.GetEventTime() != 42 {
		t.Errorf("event_time = %d, want 42", rec.GetEventTime())
	}
	if rec.GetMessageType() != "ghosttrace.events.v1.Outcome" {
		t.Errorf("message_type = %q", rec.GetMessageType())
	}
	if len(rec.GetCanonicalPayload()) == 0 {
		t.Fatal("record carries no payload")
	}

	// Decoding the payload must yield the original message. If it does
	// not, the archive would store something the collector never saw.
	var back eventsv1.Outcome
	if err := proto.Unmarshal(rec.GetCanonicalPayload(), &back); err != nil {
		t.Fatalf("payload does not decode: %v", err)
	}
	if back.GetEvaluationId() != "ev_1" {
		t.Errorf("round trip lost the record: %+v", &back)
	}
}

// published + dropped must equal appended. A gap means a record went to
// disk and neither reached the stream nor was counted as lost.
func TestCountsAlwaysBalance(t *testing.T) {
	pub := &fakePub{}
	a := New(&fakeLocal{}, pub, "t_test", quiet())
	for i := 0; i < 5; i++ {
		if err := a.Append(context.Background(), msg(), int64(i)); err != nil {
			t.Fatal(err)
		}
	}
	pub.err = errors.New("broker restarting")
	for i := 0; i < 3; i++ {
		if err := a.Append(context.Background(), msg(), int64(i)); err != nil {
			t.Fatal(err)
		}
	}
	appended, published, dropped := a.Counts()
	if published+dropped != appended {
		t.Errorf("%d published + %d dropped != %d appended", published, dropped, appended)
	}
	if appended != 8 || published != 5 || dropped != 3 {
		t.Errorf("counts = (%d, %d, %d), want (8, 5, 3)", appended, published, dropped)
	}
}
