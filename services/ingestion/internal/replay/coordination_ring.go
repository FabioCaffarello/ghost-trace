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
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/hypothesis"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

const coordinationRingFormationMessageType = "ghosttrace.events.v1.CoordinationRingFormation"

// CoordinationRingFormationReport is the per-ReplayCoordinationRingFormation
// outcome. Mirrors §0086+§0087+§0088.
type CoordinationRingFormationReport struct {
	TargetHashHex                string
	Match                        bool
	RecomputedHashHex            string
	PatternSignature             string
	PatternParameters            string
	ReconstructedFormationCount  int
	ContributingObservationCount int
	MaxCommittedAtNs             int64
}

// ReplayCoordinationRingFormation performs Phase 3 reconstructive
// replay of a CoordinationRingFormation per decision-log §0089.
// Closes the four-subtype Phase 3 arc opened at §0086.
func ReplayCoordinationRingFormation(ctx context.Context, sub *substrate.Substrate, targetHash [32]byte) (CoordinationRingFormationReport, error) {
	row, err := sub.LookupRow(ctx, targetHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CoordinationRingFormationReport{}, fmt.Errorf("%w: %x", ErrTargetNotFound, targetHash)
		}
		return CoordinationRingFormationReport{}, fmt.Errorf("replay.ReplayCoordinationRingFormation: lookup target: %w", err)
	}
	if row.MessageType != coordinationRingFormationMessageType {
		return CoordinationRingFormationReport{}, fmt.Errorf("%w: %x is %q (expected %s)",
			ErrTargetWrongType, targetHash, row.MessageType, coordinationRingFormationMessageType)
	}

	payload, err := sub.ReadBlob(ctx, targetHash)
	if err != nil {
		return CoordinationRingFormationReport{}, fmt.Errorf("replay.ReplayCoordinationRingFormation: read target blob: %w", err)
	}
	original := &eventsv1.CoordinationRingFormation{}
	if err := proto.Unmarshal(payload, original); err != nil {
		return CoordinationRingFormationReport{}, fmt.Errorf("replay.ReplayCoordinationRingFormation: unmarshal target: %w", err)
	}

	pattern, err := ResolveCRFormationPattern(original.PatternSignature, original.PatternParameters)
	if err != nil {
		return CoordinationRingFormationReport{}, err
	}
	if pattern.Parameters() != original.PatternParameters {
		return CoordinationRingFormationReport{}, fmt.Errorf("%w: pattern %q produced %q, original carried %q",
			ErrPatternParameterMismatch, pattern.Signature(),
			pattern.Parameters(), original.PatternParameters)
	}

	bcFctx, err := hypothesis.CollectFormationContextAt(ctx, sub, row.CommittedAt)
	if err != nil {
		return CoordinationRingFormationReport{}, fmt.Errorf("replay.ReplayCoordinationRingFormation: collect formation context: %w", err)
	}
	crFctx, ok := bcFctx.(hypothesis.CoordinationRingFormationContext)
	if !ok {
		return CoordinationRingFormationReport{}, fmt.Errorf("replay.ReplayCoordinationRingFormation: internal type assertion failed (FormationContext must satisfy CoordinationRingFormationContext)")
	}

	formations := pattern.Form(crFctx, original.FormationAt)

	report := CoordinationRingFormationReport{
		TargetHashHex:                canonical.HashHex(targetHash),
		PatternSignature:             original.PatternSignature,
		PatternParameters:            original.PatternParameters,
		ReconstructedFormationCount:  len(formations),
		ContributingObservationCount: len(bcFctx.DeclaredSessions()),
		MaxCommittedAtNs:             row.CommittedAt,
	}

	for _, ev := range formations {
		ev.PatternSignature = pattern.Signature()
		ev.PatternParameters = pattern.Parameters()
		_, recomputedHash, err := canonical.MarshalAndHash(ev)
		if err != nil {
			return report, fmt.Errorf("replay.ReplayCoordinationRingFormation: marshal candidate: %w", err)
		}
		if bytes.Equal(targetHash[:], recomputedHash[:]) {
			report.Match = true
			report.RecomputedHashHex = canonical.HashHex(recomputedHash)
			return report, nil
		}
	}

	return report, nil
}

// ResolveCRFormationPattern maps a CoordinationRingFormation's
// (pattern_signature, pattern_parameters) tuple back to a concrete
// CoordinationRingFormationPattern.
//
// Currently supports co-occurrence-window-v1
// ("max_window_seconds=N;min_edge_support=N").
func ResolveCRFormationPattern(signature, parameters string) (hypothesis.CoordinationRingFormationPattern, error) {
	switch signature {
	case hypothesis.CoOccurrenceWindowV1Signature:
		maxWindow, err := parseIntParam(parameters, "max_window_seconds")
		if err != nil {
			return nil, fmt.Errorf("parse %s parameters %q: %w",
				hypothesis.CoOccurrenceWindowV1Signature, parameters, err)
		}
		minEdgeSupport, err := parseIntParam(parameters, "min_edge_support")
		if err != nil {
			return nil, fmt.Errorf("parse %s parameters %q: %w",
				hypothesis.CoOccurrenceWindowV1Signature, parameters, err)
		}
		return hypothesis.CoOccurrenceWindowV1{
			MinEdgeSupport:   minEdgeSupport,
			MaxWindowSeconds: maxWindow,
		}, nil

	default:
		return nil, fmt.Errorf("%w: %q (known: %s)",
			ErrPatternUnknown, signature,
			hypothesis.CoOccurrenceWindowV1Signature)
	}
}
