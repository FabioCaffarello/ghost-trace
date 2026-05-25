// ReplayAllDerivedActorAttributions per decision-log §0173. Mirrors
// ReplayAllOperationalSessions on the attribution Cat II side per
// §0171 → §0172 → §0173 chain (library replay → CLI → batch).
//
// Together with ReplayAllOperationalSessions, both Cat II types
// committed today (per §0043 OperationalSession + §0168
// DerivedActorAttribution) now have parallel batch-replay surfaces.

package replay

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// ReplayAllDerivedActorAttributions walks every DerivedActorAttribution
// in the substrate, re-derives each from its declared source Cat I
// NetworkObservation under the same attribution definition recorded on
// the original, and reports aggregate match/drift/error counts.
//
// Unlike ReplayAllOperationalSessions, this function does NOT
// pre-collect a DerivationContext: attribution definitions registered
// today (network_5tuple_actor_v1 per §0168) consume input from the
// source Cat I observation ALONE. Future attribution definitions
// requiring cross-observation join would need an AttributionContext
// analog at that time.
//
// Cost: substrate walks = 1 (the iteration walk) + 1 lookup per
// DerivedActorAttribution (vs N+1 walks if ReplayDerivedActorAttribution
// were called naively in a loop). Cheaper than ReplayAllOperationalSessions
// because no DerivationContext pre-collection walk is needed.
//
// The returned BatchReplayReport reuses the existing shape from §0085
// per stable-structure discipline — both batch replay functions emit
// identical report shapes; only their target Cat II type differs.
// Operators reading the JSON output can apply the same parsing logic
// against either CLI.
func ReplayAllDerivedActorAttributions(ctx context.Context, sub *substrate.Substrate) (BatchReplayReport, error) {
	report := BatchReplayReport{}

	walkErr := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != derivedActorAttributionMessageType {
			return nil
		}
		report.Total++

		entry := replayOneAttributionFromRow(ctx, sub, row)
		switch entry.Outcome {
		case OutcomeMatch:
			report.Matched++
		case OutcomeDrift:
			report.Drifted++
			report.Drift = append(report.Drift, entry)
		case OutcomeError:
			report.Errored++
			report.Errors = append(report.Errors, entry)
		}
		return nil
	})
	if walkErr != nil {
		return report, fmt.Errorf("replay.ReplayAllDerivedActorAttributions: walk: %w", walkErr)
	}

	return report, nil
}

// replayOneAttributionFromRow performs the per-DerivedActorAttribution
// replay given the row. Returns a BatchReplayEntry with the appropriate
// outcome. Mirrors replayOneFromRow on the attribution side.
func replayOneAttributionFromRow(ctx context.Context, sub *substrate.Substrate, row substrate.EventRow) BatchReplayEntry {
	targetHex := canonical.HashHex(row.EventHash)
	entry := BatchReplayEntry{TargetHashHex: targetHex}

	payload, err := sub.ReadBlob(ctx, row.EventHash)
	if err != nil {
		entry.Outcome = OutcomeError
		entry.Reason = fmt.Sprintf("read target blob: %v", err)
		return entry
	}
	original := &eventsv1.DerivedActorAttribution{}
	if err := proto.Unmarshal(payload, original); err != nil {
		entry.Outcome = OutcomeError
		entry.Reason = fmt.Sprintf("unmarshal target: %v", err)
		return entry
	}

	def, err := ResolveAttributionDefinition(original.DefinitionVersion, original.DefinitionParameters)
	if err != nil {
		entry.Outcome = OutcomeError
		entry.Reason = err.Error()
		return entry
	}
	if def.Parameters() != original.DefinitionParameters {
		entry.Outcome = OutcomeError
		entry.Reason = fmt.Sprintf("%v: definition %q produced %q, original carried %q",
			ErrDefinitionParameterMismatch, def.Version(),
			def.Parameters(), original.DefinitionParameters)
		return entry
	}

	var sourceHash [32]byte
	if len(original.SourceEventHash) != 32 {
		entry.Outcome = OutcomeError
		entry.Reason = fmt.Sprintf("source_event_hash must be 32 bytes; got %d", len(original.SourceEventHash))
		return entry
	}
	copy(sourceHash[:], original.SourceEventHash)

	sourceRow, err := sub.LookupRow(ctx, sourceHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			entry.Outcome = OutcomeError
			entry.Reason = fmt.Sprintf("%v: %x", ErrSourceNotFound, sourceHash)
			return entry
		}
		entry.Outcome = OutcomeError
		entry.Reason = fmt.Sprintf("lookup source: %v", err)
		return entry
	}
	if sourceRow.MessageType != networkObservationMessageType {
		entry.Outcome = OutcomeError
		entry.Reason = fmt.Sprintf("%v: source %x is %q", ErrSourceWrongType, sourceHash, sourceRow.MessageType)
		return entry
	}

	sourcePayload, err := sub.ReadBlob(ctx, sourceHash)
	if err != nil {
		entry.Outcome = OutcomeError
		entry.Reason = fmt.Sprintf("read source blob: %v", err)
		return entry
	}
	source := &eventsv1.NetworkObservation{}
	if err := proto.Unmarshal(sourcePayload, source); err != nil {
		entry.Outcome = OutcomeError
		entry.Reason = fmt.Sprintf("unmarshal source: %v", err)
		return entry
	}

	rederived, ok := def.Derive(source)
	if !ok || rederived == nil {
		entry.Outcome = OutcomeError
		entry.Reason = fmt.Sprintf("definition %q declined to derive from source %x (Derive returned ok=false)",
			def.Version(), sourceHash)
		return entry
	}
	rederived.DefinitionVersion = def.Version()
	rederived.DefinitionParameters = def.Parameters()
	rederived.SourceEventHash = sourceHash[:]

	_, recomputedHash, err := canonical.MarshalAndHash(rederived)
	if err != nil {
		entry.Outcome = OutcomeError
		entry.Reason = fmt.Sprintf("marshal rederived: %v", err)
		return entry
	}
	entry.RecomputedHashHex = canonical.HashHex(recomputedHash)
	if bytes.Equal(row.EventHash[:], recomputedHash[:]) {
		entry.Outcome = OutcomeMatch
	} else {
		entry.Outcome = OutcomeDrift
	}
	return entry
}
