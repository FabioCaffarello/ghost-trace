package replay

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/derivation"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// BatchReplayOutcome categorizes one record's replay result.
type BatchReplayOutcome string

const (
	// OutcomeMatch: re-derivation produced byte-identical content-hash.
	OutcomeMatch BatchReplayOutcome = "match"

	// OutcomeDrift: re-derivation completed but produced a different
	// content-hash. Indicates derivation-implementation drift or
	// substrate-record inconsistency. The originally-committed record
	// remains authoritative per §2.1.
	OutcomeDrift BatchReplayOutcome = "drift"

	// OutcomeError: replay could not complete due to a precondition
	// failure (unknown definition version, missing source, malformed
	// parameters, etc.). Distinct from drift — drift means replay ran;
	// error means it couldn't.
	OutcomeError BatchReplayOutcome = "error"
)

// BatchReplayEntry is the per-target outcome carried in BatchReplayReport.
// Records that errored carry Reason; records that matched or drifted
// carry their original + recomputed hashes.
type BatchReplayEntry struct {
	TargetHashHex     string             `json:"target_event_hash"`
	Outcome           BatchReplayOutcome `json:"outcome"`
	RecomputedHashHex string             `json:"recomputed_event_hash,omitempty"`
	Reason            string             `json:"reason,omitempty"`
}

// BatchReplayReport is the per-ReplayAllOperationalSessions outcome.
// Total = Matched + Drifted + Errored. Entries records every target
// replayed, in substrate-walk order. Drift + Error lists carry the
// subset of entries whose outcome was non-match (convenience for
// operators investigating).
type BatchReplayReport struct {
	Total   int                `json:"total"`
	Matched int                `json:"matched"`
	Drifted int                `json:"drifted"`
	Errored int                `json:"errored"`
	Drift   []BatchReplayEntry `json:"drift,omitempty"`
	Errors  []BatchReplayEntry `json:"errors,omitempty"`
}

// ReplayAllOperationalSessions walks every OperationalSession in the
// substrate, re-derives each from its declared source under the same
// operational definition, and reports aggregate match/drift/error
// counts.
//
// Per §0085, the DerivationContext is collected ONCE (one substrate
// walk to index NetworkEvents by actor_ref) and reused across all
// per-target replays. Cost: substrate walks = 2 (DerivationContext +
// the iteration walk) + 1 lookup-per-OperationalSession (vs N+1 walks
// if ReplayOperationalSession were called naively in a loop).
//
// The returned BatchReplayReport is informational; the substrate is
// not modified. A non-empty Drift slice indicates derivation-
// implementation drift since the original commits, which is a §2.1-
// observability concern even though §2.1 itself is not violated
// (the substrate records remain immutable; the derivation
// implementation has changed).
func ReplayAllOperationalSessions(ctx context.Context, sub *substrate.Substrate) (BatchReplayReport, error) {
	// Pre-collect DerivationContext once for efficiency.
	dctx, err := derivation.CollectDerivationContext(ctx, sub)
	if err != nil {
		return BatchReplayReport{}, fmt.Errorf("replay.ReplayAllOperationalSessions: collect derivation context: %w", err)
	}

	report := BatchReplayReport{}

	walkErr := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != operationalSessionMessageType {
			return nil
		}
		report.Total++

		entry := replayOneFromRow(ctx, sub, dctx, row)
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
		return report, fmt.Errorf("replay.ReplayAllOperationalSessions: walk: %w", walkErr)
	}

	return report, nil
}

// replayOneFromRow performs the per-OperationalSession replay given
// the row + pre-collected DerivationContext. Returns a
// BatchReplayEntry with the appropriate outcome. Mirrors
// ReplayOperationalSession's logic but reuses the supplied dctx
// rather than re-collecting it per record.
func replayOneFromRow(ctx context.Context, sub *substrate.Substrate, dctx derivation.DerivationContext, row substrate.EventRow) BatchReplayEntry {
	targetHex := canonical.HashHex(row.EventHash)
	entry := BatchReplayEntry{TargetHashHex: targetHex}

	payload, err := sub.ReadBlob(ctx, row.EventHash)
	if err != nil {
		entry.Outcome = OutcomeError
		entry.Reason = fmt.Sprintf("read target blob: %v", err)
		return entry
	}
	original := &eventsv1.OperationalSession{}
	if err := proto.Unmarshal(payload, original); err != nil {
		entry.Outcome = OutcomeError
		entry.Reason = fmt.Sprintf("unmarshal target: %v", err)
		return entry
	}

	def, err := ResolveOperationalDefinition(original.DefinitionVersion, original.DefinitionParameters)
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
	if sourceRow.MessageType != declaredSessionMessageType {
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
	source := &eventsv1.DeclaredSession{}
	if err := proto.Unmarshal(sourcePayload, source); err != nil {
		entry.Outcome = OutcomeError
		entry.Reason = fmt.Sprintf("unmarshal source: %v", err)
		return entry
	}

	rederived := def.Derive(source, sourceHash, dctx)
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
