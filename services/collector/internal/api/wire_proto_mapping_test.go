// What the browser sends, and what reaches the archive, are two
// different shapes — and nothing checked that they still correspond.
//
// `contract/architecture.md` claimed until PR-6.3 that "the JSON
// accepted at the HTTP boundary mirrors [the proto] field-for-field".
// It does not, and never did: the JSON carries one `events` array
// discriminated by `type`, the proto carries six typed lists; the JSON
// nests `page` and `client`, the proto is flat; and three proto fields
// have no JSON counterpart at all, on purpose.
//
// The guard table in the wire-contract-change skill listed five guards
// and none of them saw this seam. `vocabulary_test.go` compares the SDK
// against the contract; OpenAPI conformance compares the spec against
// responses; the goldens freeze bytes a client receives. Between the
// request struct and the archived record there was nothing — which is
// the exact shape of finding M22, where a rename does not fail because
// the server zero-fills and the suite stays green.
//
// So the correspondence is written down here as three tables, and a new
// or renamed field on either side fails until somebody says which of
// the three it belongs in.
package api

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/libs/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/libs/wire"
	"google.golang.org/protobuf/proto"
)

// carried: a JSON field whose value travels into the archive, and the
// proto field that carries it. The names differ often enough that a
// table is the only honest way to state it.
var carried = map[string]string{
	// POST /v1/telemetry
	"seq":            "seq",
	"sent_at_ms":     "sent_at_ms",
	"page.path":      "page_path",
	"page.viewport":  "viewport_w+viewport_h", // one JSON array, two proto fields
	"events[].t":     "t_ms",                  // every event family
	"events[].src":   "pointer_events[].src",
	"events[].pts":   "pointer_events[].pts",
	"events[].phase": "key_events[].phase",
	"events[].class": "key_events[].key_class", // renamed on the way in
	"events[].target": "key_events[].target, focus_events[].target, " +
		"form_events[].target",
	"events[].dy":     "scroll_events[].dy",
	"events[].mode":   "scroll_events[].mode",
	"events[].state":  "focus_events[].state, visibility_events[].state",
	"events[].action": "form_events[].action",

	// POST /v1/sessions
	"sessions:page.path":             "page_path",
	"sessions:client.pointer":        "pointer_type",
	"sessions:client.touch":          "touch",
	"sessions:client.viewport":       "viewport_w+viewport_h",
	"sessions:client.tz_offset":      "tz_offset_min",
	"sessions:client.reduced_motion": "reduced_motion",
}

// jsonOnly: accepted at the boundary, and deliberately not archived as
// itself. Each entry is a decision, not an omission.
var jsonOnly = map[string]string{
	"session_token": "exchanged for session_id, never stored — the token is a " +
		"bearer credential the browser holds and the id outlives it by 7 days " +
		"in the archive (session.Store.Create)",
	"events[].type": "structural: it selects WHICH typed list the event lands " +
		"in, so it is consumed by the dispatch rather than carried",
	"sessions:site_key": "resolved to tenant_id server-side; it identifies a " +
		"tenant and does not authenticate one (§1)",
}

// protoOnly: archived, and never supplied by the browser. A browser that
// could set any of these could lie about it.
var protoOnly = map[string]string{
	"tenant_id":  "resolved from site_key by the server; a browser never states its own tenant (§1)",
	"session_id": "issued by the store; the browser holds a token instead",
	"received_at": "the server's clock. The client's claim about time is " +
		"sent_at_ms and stays separate from it",
	"started_at": "the server's clock, same reason",
}

func TestEveryWireFieldIsAccountedForInTheArchive(t *testing.T) {
	jsonFields := map[string]bool{}
	collect := func(rt reflect.Type, prefix string) {
		var walk func(reflect.Type, string)
		walk = func(rt reflect.Type, at string) {
			for i := 0; i < rt.NumField(); i++ {
				f := rt.Field(i)
				tag := strings.Split(f.Tag.Get("json"), ",")[0]
				if tag == "" || tag == "-" {
					continue
				}
				ft := f.Type
				if ft.Kind() == reflect.Slice && ft.Elem().Kind() == reflect.Struct {
					walk(ft.Elem(), at+tag+"[].")
					continue
				}
				if ft.Kind() == reflect.Struct {
					walk(ft, at+tag+".")
					continue
				}
				jsonFields[at+tag] = true
			}
		}
		walk(rt, prefix)
	}
	// Prefixed per message rather than merged: both bodies carry
	// `page.path`, into the same proto field name but from different
	// requests, and a single key would let a rename on one of them be
	// covered by the other's entry.
	collect(reflect.TypeOf(wire.TelemetryBatch{}), "")
	collect(reflect.TypeOf(wire.SessionsRequest{}), "sessions:")

	for f := range jsonFields {
		if carried[f] == "" && jsonOnly[f] == "" {
			t.Errorf("JSON field %q is accepted at the boundary and appears in "+
				"neither table — say where it goes, or say that it goes nowhere",
				f)
		}
	}

	protoFields := map[string]bool{}
	for _, m := range []proto.Message{
		&eventsv1.TelemetryBatch{}, &eventsv1.PointerEvent{}, &eventsv1.KeyEvent{},
		&eventsv1.ScrollEvent{}, &eventsv1.FocusEvent{}, &eventsv1.VisibilityEvent{},
		&eventsv1.FormEvent{}, &eventsv1.SessionStart{},
	} {
		d := m.ProtoReflect().Descriptor()
		for i := 0; i < d.Fields().Len(); i++ {
			protoFields[string(d.Fields().Get(i).Name())] = true
		}
	}

	// The right-hand side of `carried` names proto fields in several
	// notations — "viewport_w+viewport_h", "key_events[].key_class",
	// comma-separated alternatives. Flatten to bare field names.
	mapped := map[string]bool{}
	for _, target := range carried {
		for _, part := range strings.FieldsFunc(target, func(r rune) bool {
			return r == '+' || r == ',' || r == ' '
		}) {
			if i := strings.LastIndex(part, "."); i >= 0 {
				part = part[i+1:]
			}
			mapped[part] = true
		}
	}

	var unexplained []string
	for f := range protoFields {
		if strings.HasSuffix(f, "_events") {
			continue // the six lists are containers, their fields are checked
		}
		if !mapped[f] && protoOnly[f] == "" {
			unexplained = append(unexplained, f)
		}
	}
	sort.Strings(unexplained)
	for _, f := range unexplained {
		t.Errorf("proto field %q is archived and no JSON field feeds it — if "+
			"the server sets it, say so in protoOnly; if a browser should, it "+
			"is missing from the wire", f)
	}
}

// TestTheTablesDescribeFieldsThatExist is the other direction: a table
// entry for a field nobody has means the table is a fossil, and a fossil
// table passes the test above for the wrong reason.
func TestTheTablesDescribeFieldsThatExist(t *testing.T) {
	live := map[string]bool{}
	for _, m := range []proto.Message{
		&eventsv1.TelemetryBatch{}, &eventsv1.PointerEvent{}, &eventsv1.KeyEvent{},
		&eventsv1.ScrollEvent{}, &eventsv1.FocusEvent{}, &eventsv1.VisibilityEvent{},
		&eventsv1.FormEvent{}, &eventsv1.SessionStart{},
	} {
		d := m.ProtoReflect().Descriptor()
		for i := 0; i < d.Fields().Len(); i++ {
			live[string(d.Fields().Get(i).Name())] = true
		}
	}
	for f := range protoOnly {
		if !live[f] {
			t.Errorf("protoOnly names %q, which no message declares — the "+
				"table outlived the field", f)
		}
	}
}
