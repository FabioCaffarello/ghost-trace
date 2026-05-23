package layerb

import (
	"sort"
	"testing"
)

// expectedRegistryMessageTypes is the proto-full-name enumeration of
// paired-dimension subject types per §0136 + §0140. Coverage parity
// with the canonical-package pairedDimensionSubjectSet
// (services/ingestion/internal/canonical/coverage_test.go) is asserted
// by this test — a future formation proto added without registration
// here surfaces at the same commit as the canonical-package coverage
// test surfacing the registration gap there.
//
// Per §0141 sub-decision A1 + §0136 paired-dimension subject set: only
// these five message types carry closure_hashes (+ direct_influenced_by
// + evidential_independence) and are therefore the only message types
// subject to Layer B's filter. All other substrate event types are
// filtered out of Layer B's computation by construction
// (extractLayerBFields returns ok=false).
var expectedRegistryMessageTypes = []string{
	"ghosttrace.events.v1.AutomationGroupFormation",
	"ghosttrace.events.v1.BehavioralClusterFormation",
	"ghosttrace.events.v1.CampaignHypothesisFormation",
	"ghosttrace.events.v1.CoordinationRingFormation",
	"ghosttrace.events.v1.OperationalSession",
}

// TestRegistryCoverage_MatchesPairedDimensionSubjectSet asserts that
// the layerb package's concreteMessageRegistry contains exactly the
// expected paired-dimension subject set. A future formation proto
// added to schemas/events/v1/ + registered in the canonical-package
// pairedDimensionSubjectSet but NOT registered in concreteMessageRegistry
// here will silently be excluded from Layer B's filter — defeating the
// §2.6 + §2.4 enforcement at the Layer B layer.
//
// This test is the drift-protection mechanism per §0141 Methodological
// observation 2 (canonical-serialization-contract re-reading should
// catch structural-preclusion at Phase 3 — applied here at the
// implementation layer as test-time drift protection).
func TestRegistryCoverage_MatchesPairedDimensionSubjectSet(t *testing.T) {
	got := make([]string, 0, len(concreteMessageRegistry))
	for k := range concreteMessageRegistry {
		got = append(got, k)
	}
	sort.Strings(got)

	want := make([]string, len(expectedRegistryMessageTypes))
	copy(want, expectedRegistryMessageTypes)
	sort.Strings(want)

	if len(got) != len(want) {
		t.Fatalf("registry size: got %d, want %d\n  got:  %v\n  want: %v", len(got), len(want), got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("registry[%d]: got %q, want %q", i, got[i], want[i])
		}
	}
}

// TestRegistryFactories_ProduceNonNil verifies each registry entry
// returns a non-nil proto.Message instance per call. Guards against a
// factory regressing to "return nil" via copy-paste error.
func TestRegistryFactories_ProduceNonNil(t *testing.T) {
	for messageType, factory := range concreteMessageRegistry {
		msg := factory()
		if msg == nil {
			t.Errorf("registry[%q] factory returned nil; want non-nil proto.Message", messageType)
			continue
		}
		fullName := string(msg.ProtoReflect().Descriptor().FullName())
		if fullName != messageType {
			t.Errorf("registry[%q] factory produced message of type %q; expected key/value parity", messageType, fullName)
		}
	}
}

// TestConcreteMessageFor_UnknownType returns nil for unregistered
// message types. The contract per §0136 + §0140 implies the unregistered
// set is "everything except the paired-dimension subject set"; nil
// signals Layer B's filter to exclude the event from computation.
func TestConcreteMessageFor_UnknownType(t *testing.T) {
	msg := concreteMessageFor("ghosttrace.events.v1.DeclaredSession")
	if msg != nil {
		t.Errorf("concreteMessageFor(DeclaredSession): got non-nil %T; want nil (DeclaredSession is Cat I, not subject to paired-dimension)", msg)
	}
	msg = concreteMessageFor("ghosttrace.events.v1.IngestionEvent")
	if msg != nil {
		t.Errorf("concreteMessageFor(IngestionEvent): got non-nil %T; want nil (IngestionEvent is audit, not subject to Layer B)", msg)
	}
	msg = concreteMessageFor("nonexistent.message.Type")
	if msg != nil {
		t.Errorf("concreteMessageFor(nonexistent): got non-nil %T; want nil", msg)
	}
}
