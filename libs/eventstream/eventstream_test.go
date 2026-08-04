package eventstream

import (
	"errors"
	"strings"
	"testing"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/libs/genproto/events/v1"
	"google.golang.org/protobuf/proto"
)

// Every Category I record the services append must have a subject.
// Without this, adding a record type and forgetting the map entry
// produces something that publishes nowhere — indistinguishable from a
// record nobody produced.
func TestEveryArchivedMessageHasASubject(t *testing.T) {
	archived := []proto.Message{
		&eventsv1.SessionStart{},
		&eventsv1.TelemetryBatch{},
		&eventsv1.Evaluation{},
		&eventsv1.Outcome{},
	}
	for _, m := range archived {
		name := string(m.ProtoReflect().Descriptor().FullName())
		if _, err := KindFor(name); err != nil {
			t.Errorf("%s is appended to the archive but has no subject: %v", name, err)
		}
	}
	if got, want := len(kindByMessageType), len(archived); got != want {
		t.Errorf("subject map has %d entries for %d archived types — a kind with no "+
			"producer publishes a subject nobody fills", got, want)
	}
}

func TestUnknownMessageTypeIsAnError(t *testing.T) {
	_, err := KindFor("ghosttrace.events.v1.NotARecord")
	if !errors.Is(err, ErrUnknownMessageType) {
		t.Fatalf("want ErrUnknownMessageType, got %v", err)
	}
}

func TestSubjectDerivesFromTheRecord(t *testing.T) {
	rec := &eventsv1.ArchiveRecord{
		MessageType: "ghosttrace.events.v1.Evaluation",
		Tenant:      "t_demo",
	}
	got, err := SubjectFor(rec)
	if err != nil {
		t.Fatal(err)
	}
	if want := "gt.events.v1.evaluation.t_demo"; got != want {
		t.Errorf("subject = %q, want %q", got, want)
	}
}

// A tenant carrying a NATS separator or wildcard would publish onto a
// different subject, or onto all of them.
func TestTenantCannotBreakOutOfItsSubject(t *testing.T) {
	for _, tenant := range []string{"a.b", "a*b", "a>b", "a b"} {
		s := Subject(KindOutcome, tenant)
		rest := strings.TrimPrefix(s, SubjectPrefix+"."+string(KindOutcome)+".")
		if strings.ContainsAny(rest, ".*> ") {
			t.Errorf("tenant %q produced subject %q — it can address other subjects", tenant, s)
		}
	}
	if got := Subject(KindOutcome, ""); !strings.HasSuffix(got, ".unknown") {
		t.Errorf("empty tenant produced %q; want a placeholder token, not an empty one", got)
	}
}

func TestFiltersCoverEveryKind(t *testing.T) {
	if len(Kinds()) != len(kindByMessageType) {
		t.Fatalf("Kinds() lists %d, the map has %d", len(Kinds()), len(kindByMessageType))
	}
	for _, k := range Kinds() {
		if !strings.HasPrefix(SubjectFilter(k), SubjectPrefix) {
			t.Errorf("filter for %s is outside the stream prefix", k)
		}
	}
}
