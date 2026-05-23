package derivation

import (
	"fmt"
	"time"

	commonv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/common/v1"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
)

// PaddedV1Version is the stable identifier of the padded-v1 operational
// definition: the minimal Cat II derivation per decision-log §0043
// (first Cat II construct landing). Boundary derivation:
// operational_start_at = source.declared_at;
// operational_end_at   = source.declared_at + pad_seconds.
// Identity-tier (actor_ref) inherits from source per entity-model.md
// line 36.
const PaddedV1Version = "padded-v1"

// PaddedV1 is the first operational definition. It produces a divergent
// operational boundary (operational_end_at != declared_at) with a
// single configurable parameter (pad_seconds). Deterministic by
// construction; idempotent on re-run.
//
// PadSeconds is the boundary-padding parameter. Encoded into
// definition_parameters as "pad_seconds=<int>"; the encoding is part
// of identity, so changing PadSeconds produces a new OperationalSession
// for the same source per the entity-model.md line 45 versioning rule
// ("Re-derivation under a new definition produces a new construct,
// never a mutation of the existing one.").
type PaddedV1 struct {
	PadSeconds int64
}

// Version implements OperationalDefinition.
func (p PaddedV1) Version() string { return PaddedV1Version }

// Parameters implements OperationalDefinition. Canonical form:
// lowercase key=value pairs separated by semicolons, sorted by key.
// PaddedV1 has a single parameter; the format scales to multi-parameter
// definitions via the same sort-key-then-format-key=value discipline.
func (p PaddedV1) Parameters() string {
	return fmt.Sprintf("pad_seconds=%d", p.PadSeconds)
}

// Derive implements OperationalDefinition. operational_start_at copies
// the source's declared_at; operational_end_at adds PadSeconds in
// nanoseconds. PaddedV1 ignores DerivationContext — it derives from
// the source DeclaredSession alone.
func (p PaddedV1) Derive(source *eventsv1.DeclaredSession, _ [32]byte, _ DerivationContext) *eventsv1.OperationalSession {
	padNanos := p.PadSeconds * int64(time.Second)
	return &eventsv1.OperationalSession{
		ActorRef:           source.GetActorRef(),
		OperationalStartAt: source.GetDeclaredAt(),
		OperationalEndAt:   source.GetDeclaredAt() + padNanos,
		// EvidentialIndependence per §0140 — α = 1/1. PaddedV1 reads
		// only the single Cat I DeclaredSession source; no
		// promoted-hypothesis influence is consumed per §0133 Q3-α.
		EvidentialIndependence: &commonv1.EvidentialIndependence{Numerator: 1, Denominator: 1},
	}
}

// Ensure PaddedV1 satisfies OperationalDefinition at compile time.
var _ OperationalDefinition = PaddedV1{}
