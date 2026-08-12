package eventstream_test

// Who a record says it belongs to.
//
// The envelope's tenant used to be a constructor argument, taken from
// the process's -tenant flag. Every payload already carried the tenant
// the REQUEST proved, so with one tenant configured the two always
// agreed — and with `-tenants <file>` they agreed only for whichever
// customer matched the flag. Everyone else's records were archived, and
// subject-routed, as that one.

import (
	"errors"
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/FabioCaffarello/ghost-trace/libs/eventstream"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/libs/genproto/events/v1"
)

func TestTheRecordSaysWhatThePayloadSays(t *testing.T) {
	// One process, two customers — the case a per-tenant registry
	// creates and a per-process flag cannot express.
	for _, tenant := range []string{"t_acme", "t_globex"} {
		rec, err := eventstream.Record(&eventsv1.SessionStart{
			TenantId: tenant, SessionId: "s_1",
		}, 42)
		if err != nil {
			t.Fatalf("%s: %v", tenant, err)
		}
		if rec.GetTenant() != tenant {
			t.Errorf("record says %q for a payload from %q — the archive would "+
				"attribute this customer's session to another one",
				rec.GetTenant(), tenant)
		}
		subject, err := eventstream.SubjectFor(rec)
		if err != nil {
			t.Fatal(err)
		}
		if want := eventstream.Subject(eventstream.KindSession, tenant); subject != want {
			t.Errorf("subject = %q, want %q — a consumer filtering to one tenant "+
				"with a wildcard would get somebody else's records", subject, want)
		}
	}
}

func TestASubjectCannotDisagreeWithItsPayload(t *testing.T) {
	// The structural claim, and the reason this is derived rather than
	// supplied: there is no argument left to pass the wrong value to.
	rec, err := eventstream.Record(&eventsv1.Evaluation{
		TenantId: "t_acme", EvaluationId: "ev_1",
	}, 1)
	if err != nil {
		t.Fatal(err)
	}
	subject, err := eventstream.SubjectFor(rec)
	if err != nil {
		t.Fatal(err)
	}
	if got := eventstream.Subject(eventstream.KindEvaluation, rec.GetTenant()); got != subject {
		t.Errorf("SubjectFor(%q) = %q but the record's own tenant gives %q",
			rec.GetTenant(), subject, got)
	}
}

func TestAPayloadWithNoTenantIsRefused(t *testing.T) {
	// An empty tenant routes to `gt.events.v1.<kind>.` and is
	// attributed to nobody. The archive's whole job is saying who a
	// record belongs to, so publishing one that cannot answer is worse
	// than refusing it.
	_, err := eventstream.Record(&eventsv1.SessionStart{SessionId: "s_1"}, 1)
	if !errors.Is(err, eventstream.ErrNoTenant) {
		t.Errorf("err = %v, want ErrNoTenant", err)
	}
}

func TestEveryArchivedTypeCarriesATenant(t *testing.T) {
	// The reflection reads a field named tenant_id, so a payload type
	// that stopped declaring one would fail at publish rather than at
	// review. Checking all of them here means that failure arrives in
	// CI instead of in production.
	for _, msg := range []interface {
		ProtoReflect() protoreflect.Message
	}{
		&eventsv1.SessionStart{TenantId: "t"},
		&eventsv1.TelemetryBatch{TenantId: "t"},
		&eventsv1.Evaluation{TenantId: "t"},
		&eventsv1.Outcome{TenantId: "t"},
	} {
		if _, err := eventstream.Record(msg, 1); err != nil {
			t.Errorf("%T: %v", msg, err)
		}
	}
}
