package layerb

import (
	"google.golang.org/protobuf/proto"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
)

// concreteMessageRegistry maps substrate message_type strings to
// factory functions producing fresh concrete proto.Message instances.
//
// Per §0136 + §0140 paired-dimension subject enumeration, only these
// message types carry the closure_hashes + direct_influenced_by +
// evidential_independence fields subject to Layer B's filter. All
// other substrate event types are filtered out of Layer B's
// computation by construction (extractLayerBFields returns ok=false).
//
// The registry is intentionally explicit rather than reflection-based
// over genproto package contents — adding a new closure-carrying
// message type requires a deliberate registration here, which
// surfaces the schemas-evolution event at code-review time per the
// canonical-serialization-contract §Schemas-Evolution Events boundary.
//
// Coverage parity with the canonical-package paired-dimension subject
// set (services/ingestion/internal/canonical/canonical.go
// pairedDimensionSubjectSet per §0140) is asserted by the package's
// coverage_test.go — a future formation proto added without
// registration here surfaces at the same commit.
var concreteMessageRegistry = map[string]func() proto.Message{
	"ghosttrace.events.v1.BehavioralClusterFormation":  func() proto.Message { return &eventsv1.BehavioralClusterFormation{} },
	"ghosttrace.events.v1.AutomationGroupFormation":    func() proto.Message { return &eventsv1.AutomationGroupFormation{} },
	"ghosttrace.events.v1.CampaignHypothesisFormation": func() proto.Message { return &eventsv1.CampaignHypothesisFormation{} },
	"ghosttrace.events.v1.CoordinationRingFormation":   func() proto.Message { return &eventsv1.CoordinationRingFormation{} },
	"ghosttrace.events.v1.OperationalSession":          func() proto.Message { return &eventsv1.OperationalSession{} },
}
