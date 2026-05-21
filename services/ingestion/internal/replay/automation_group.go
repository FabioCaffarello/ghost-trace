package replay

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"google.golang.org/protobuf/proto"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/hypothesis"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

const automationGroupFormationMessageType = "ghosttrace.events.v1.AutomationGroupFormation"

// AutomationGroupFormationReport is the per-ReplayAutomationGroupFormation
// outcome. Mirrors BehavioralClusterFormationReport's shape.
type AutomationGroupFormationReport struct {
	TargetHashHex                string
	Match                        bool
	RecomputedHashHex            string
	PatternSignature             string
	PatternParameters            string
	ReconstructedFormationCount  int
	ContributingObservationCount int
	MaxCommittedAtNs             int64
}

// ReplayAutomationGroupFormation performs Phase 3 reconstructive
// replay of an AutomationGroupFormation per decision-log §0087.
// Same shape as ReplayBehavioralClusterFormation (§0086); the
// FormationContext interface differs by typed-subtype-landings
// discipline (§0056) but the same underlying walker satisfies both.
func ReplayAutomationGroupFormation(ctx context.Context, sub *substrate.Substrate, targetHash [32]byte) (AutomationGroupFormationReport, error) {
	row, err := sub.LookupRow(ctx, targetHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AutomationGroupFormationReport{}, fmt.Errorf("%w: %x", ErrTargetNotFound, targetHash)
		}
		return AutomationGroupFormationReport{}, fmt.Errorf("replay.ReplayAutomationGroupFormation: lookup target: %w", err)
	}
	if row.MessageType != automationGroupFormationMessageType {
		return AutomationGroupFormationReport{}, fmt.Errorf("%w: %x is %q (expected %s)",
			ErrTargetWrongType, targetHash, row.MessageType, automationGroupFormationMessageType)
	}

	payload, err := sub.ReadBlob(ctx, targetHash)
	if err != nil {
		return AutomationGroupFormationReport{}, fmt.Errorf("replay.ReplayAutomationGroupFormation: read target blob: %w", err)
	}
	original := &eventsv1.AutomationGroupFormation{}
	if err := proto.Unmarshal(payload, original); err != nil {
		return AutomationGroupFormationReport{}, fmt.Errorf("replay.ReplayAutomationGroupFormation: unmarshal target: %w", err)
	}

	pattern, err := ResolveAGFormationPattern(original.PatternSignature, original.PatternParameters)
	if err != nil {
		return AutomationGroupFormationReport{}, err
	}
	if pattern.Parameters() != original.PatternParameters {
		return AutomationGroupFormationReport{}, fmt.Errorf("%w: pattern %q produced %q, original carried %q",
			ErrPatternParameterMismatch, pattern.Signature(),
			pattern.Parameters(), original.PatternParameters)
	}

	// Collect the FormationContext bounded by the original
	// formation's commit time. The BC helper's returned context
	// satisfies AG's AutomationGroupFormationContext too — both
	// interfaces have only DeclaredSessions() per the
	// typed-subtype-landings discipline.
	bcFctx, err := hypothesis.CollectFormationContextAt(ctx, sub, row.CommittedAt)
	if err != nil {
		return AutomationGroupFormationReport{}, fmt.Errorf("replay.ReplayAutomationGroupFormation: collect formation context: %w", err)
	}
	agFctx, ok := bcFctx.(hypothesis.AutomationGroupFormationContext)
	if !ok {
		return AutomationGroupFormationReport{}, fmt.Errorf("replay.ReplayAutomationGroupFormation: internal type assertion failed (the FormationContext returned by CollectFormationContextAt must also satisfy AutomationGroupFormationContext)")
	}

	formations := pattern.Form(agFctx, original.FormationAt)

	report := AutomationGroupFormationReport{
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
			return report, fmt.Errorf("replay.ReplayAutomationGroupFormation: marshal candidate: %w", err)
		}
		if bytes.Equal(targetHash[:], recomputedHash[:]) {
			report.Match = true
			report.RecomputedHashHex = canonical.HashHex(recomputedHash)
			return report, nil
		}
	}

	return report, nil
}

// ResolveAGFormationPattern maps an AutomationGroupFormation's
// (pattern_signature, pattern_parameters) tuple back to a concrete
// AutomationGroupFormationPattern implementation.
//
// Currently supports uniform-cadence-v1 ("max_cov_threshold=F;
// min_observation_count=N"). Other AG formation patterns register
// here as they land.
func ResolveAGFormationPattern(signature, parameters string) (hypothesis.AutomationGroupFormationPattern, error) {
	switch signature {
	case hypothesis.UniformCadenceV1Signature:
		maxCoV, err := parseFloatParam(parameters, "max_cov_threshold")
		if err != nil {
			return nil, fmt.Errorf("parse %s parameters %q: %w",
				hypothesis.UniformCadenceV1Signature, parameters, err)
		}
		minObsCount, err := parseIntParam(parameters, "min_observation_count")
		if err != nil {
			return nil, fmt.Errorf("parse %s parameters %q: %w",
				hypothesis.UniformCadenceV1Signature, parameters, err)
		}
		return hypothesis.UniformCadenceV1{
			MinObservationCount: minObsCount,
			MaxCoVThreshold:     maxCoV,
		}, nil

	default:
		return nil, fmt.Errorf("%w: %q (known: %s)",
			ErrPatternUnknown, signature,
			hypothesis.UniformCadenceV1Signature)
	}
}

// parseFloatParam extracts the value of a single-key canonical
// parameter string of the form "key=N.NNN". Parallel to parseIntParam
// in replay.go.
func parseFloatParam(parameters, key string) (float64, error) {
	prefix := key + "="
	for _, segment := range strings.Split(parameters, ";") {
		if strings.HasPrefix(segment, prefix) {
			v, err := strconv.ParseFloat(segment[len(prefix):], 64)
			if err != nil {
				return 0, fmt.Errorf("parse %s value: %w", key, err)
			}
			return v, nil
		}
	}
	return 0, fmt.Errorf("parameter %s not found in %q", key, parameters)
}
