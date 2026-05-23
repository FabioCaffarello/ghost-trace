package derivation

import (
	"fmt"
	"time"

	commonv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/common/v1"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
)

// InactivityWindowV1Version is the stable identifier of the
// inactivity-window-v1 operational definition. Boundary derivation:
// the session ends after InactivitySeconds of no NetworkEvents for
// the same actor_ref following the source DeclaredSession's
// declared_at. The canonical example from entity-model.md line 39
// ("events from one actor within a 30-minute inactivity window")
// maps directly to InactivityWindowV1 with InactivitySeconds=1800.
const InactivityWindowV1Version = "inactivity-window-v1"

// InactivityWindowV1 is the second operational definition. It
// consumes a DeclaredSession + every NetworkEvent in the substrate
// for the same actor_ref following declared_at, and derives an
// operational_end_at by extending the boundary past the last
// contiguous network event.
//
// Boundary algorithm (deterministic):
//
//  1. lastObserved := source.declared_at.
//  2. For each NetworkEvent with matching actor_ref + observed_at ≥
//     declared_at, in observed_at-ascending order:
//     - if observed_at - lastObserved ≤ InactivitySeconds (in
//       nanoseconds): lastObserved = observed_at.
//     - else: stop (the gap exceeded the inactivity window; the
//       operational session ended at lastObserved + window).
//  3. operational_end_at = lastObserved + InactivitySeconds*1e9.
//
// Determinism: the same (source, substrate's NetworkEvent set for
// this actor) + the same InactivitySeconds produces an identical
// OperationalSession. NetworkEvents arriving AFTER a derivation
// produce a DIFFERENT (newly-content-hashed) OperationalSession on
// re-derivation; the prior record is preserved per the
// entity-model.md line 45 versioning rule.
//
// Identity-tier (actor_ref) inherits from source per
// entity-model.md line 36.
type InactivityWindowV1 struct {
	InactivitySeconds int64
}

// Version implements OperationalDefinition.
func (i InactivityWindowV1) Version() string { return InactivityWindowV1Version }

// Parameters implements OperationalDefinition. Canonical form:
// "inactivity_seconds=<int>".
func (i InactivityWindowV1) Parameters() string {
	return fmt.Sprintf("inactivity_seconds=%d", i.InactivitySeconds)
}

// Derive implements OperationalDefinition. Consults dctx for the set
// of NetworkEvents associated with source.actor_ref; walks them in
// observed_at-ascending order; extends lastObserved while consecutive
// events stay within the inactivity window. The derived
// operational_end_at is lastObserved + InactivitySeconds in nanoseconds.
//
// When dctx is nil OR has no NetworkEvents for source.actor_ref, the
// derivation falls back to lastObserved = declared_at — operationally
// equivalent to a padded-style boundary, but under a DIFFERENT
// definition_version + parameters so the resulting OperationalSession
// is identity-distinct from any PaddedV1 derivation.
func (i InactivityWindowV1) Derive(source *eventsv1.DeclaredSession, _ [32]byte, dctx DerivationContext) *eventsv1.OperationalSession {
	windowNanos := i.InactivitySeconds * int64(time.Second)
	declaredAt := source.GetDeclaredAt()
	lastObserved := declaredAt

	if dctx != nil {
		events := dctx.NetworkEventsForActor(source.GetActorRef())
		for _, ne := range events {
			obs := ne.GetObservedAt()
			if obs < declaredAt {
				// Pre-session network events do not contribute to the
				// operational boundary; the operational session began
				// at declared_at per step (1) of the algorithm.
				continue
			}
			if obs-lastObserved > windowNanos {
				// Gap exceeds the inactivity window; the session ended.
				break
			}
			lastObserved = obs
		}
	}

	return &eventsv1.OperationalSession{
		ActorRef:           source.GetActorRef(),
		OperationalStartAt: declaredAt,
		OperationalEndAt:   lastObserved + windowNanos,
		// EvidentialIndependence per §0140 — α = 1/1. The derivation
		// reads only Cat I inputs (the DeclaredSession + zero or more
		// NetworkEvents for the same actor_ref); no promoted-hypothesis
		// influence is consumed per §0133 Q3-α formula.
		EvidentialIndependence: &commonv1.EvidentialIndependence{Numerator: 1, Denominator: 1},
	}
}

// Ensure InactivityWindowV1 satisfies OperationalDefinition at compile
// time.
var _ OperationalDefinition = InactivityWindowV1{}
