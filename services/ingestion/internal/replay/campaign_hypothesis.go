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

const campaignHypothesisFormationMessageType = "ghosttrace.events.v1.CampaignHypothesisFormation"

// CampaignHypothesisFormationReport is the per-ReplayCampaignHypothesisFormation
// outcome. Mirrors §0086 + §0087.
type CampaignHypothesisFormationReport struct {
	TargetHashHex                string
	Match                        bool
	RecomputedHashHex            string
	PatternSignature             string
	PatternParameters            string
	ReconstructedFormationCount  int
	ContributingObservationCount int
	MaxCommittedAtNs             int64
}

// ReplayCampaignHypothesisFormation performs Phase 3 reconstructive
// replay of a CampaignHypothesisFormation per decision-log §0088.
// Same shape as ReplayBehavioralClusterFormation + ReplayAutomationGroupFormation;
// the FormationContext interface differs but is satisfied by the same
// underlying walker.
func ReplayCampaignHypothesisFormation(ctx context.Context, sub *substrate.Substrate, targetHash [32]byte) (CampaignHypothesisFormationReport, error) {
	row, err := sub.LookupRow(ctx, targetHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CampaignHypothesisFormationReport{}, fmt.Errorf("%w: %x", ErrTargetNotFound, targetHash)
		}
		return CampaignHypothesisFormationReport{}, fmt.Errorf("replay.ReplayCampaignHypothesisFormation: lookup target: %w", err)
	}
	if row.MessageType != campaignHypothesisFormationMessageType {
		return CampaignHypothesisFormationReport{}, fmt.Errorf("%w: %x is %q (expected %s)",
			ErrTargetWrongType, targetHash, row.MessageType, campaignHypothesisFormationMessageType)
	}

	payload, err := sub.ReadBlob(ctx, targetHash)
	if err != nil {
		return CampaignHypothesisFormationReport{}, fmt.Errorf("replay.ReplayCampaignHypothesisFormation: read target blob: %w", err)
	}
	original := &eventsv1.CampaignHypothesisFormation{}
	if err := proto.Unmarshal(payload, original); err != nil {
		return CampaignHypothesisFormationReport{}, fmt.Errorf("replay.ReplayCampaignHypothesisFormation: unmarshal target: %w", err)
	}

	pattern, err := ResolveCHFormationPattern(original.PatternSignature, original.PatternParameters)
	if err != nil {
		return CampaignHypothesisFormationReport{}, err
	}
	if pattern.Parameters() != original.PatternParameters {
		return CampaignHypothesisFormationReport{}, fmt.Errorf("%w: pattern %q produced %q, original carried %q",
			ErrPatternParameterMismatch, pattern.Signature(),
			pattern.Parameters(), original.PatternParameters)
	}

	bcFctx, err := hypothesis.CollectFormationContextAt(ctx, sub, row.CommittedAt)
	if err != nil {
		return CampaignHypothesisFormationReport{}, fmt.Errorf("replay.ReplayCampaignHypothesisFormation: collect formation context: %w", err)
	}
	chFctx, ok := bcFctx.(hypothesis.CampaignHypothesisFormationContext)
	if !ok {
		return CampaignHypothesisFormationReport{}, fmt.Errorf("replay.ReplayCampaignHypothesisFormation: internal type assertion failed (FormationContext must satisfy CampaignHypothesisFormationContext)")
	}

	formations := pattern.Form(chFctx, original.FormationAt)

	report := CampaignHypothesisFormationReport{
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
			return report, fmt.Errorf("replay.ReplayCampaignHypothesisFormation: marshal candidate: %w", err)
		}
		if bytes.Equal(targetHash[:], recomputedHash[:]) {
			report.Match = true
			report.RecomputedHashHex = canonical.HashHex(recomputedHash)
			return report, nil
		}
	}

	return report, nil
}

// ResolveCHFormationPattern maps a CampaignHypothesisFormation's
// (pattern_signature, pattern_parameters) tuple back to a concrete
// CampaignHypothesisFormationPattern.
//
// Currently supports temporal-descriptor-cohort-v1
// ("max_intra_event_gap_seconds=N;min_campaign_size=N").
func ResolveCHFormationPattern(signature, parameters string) (hypothesis.CampaignHypothesisFormationPattern, error) {
	switch signature {
	case hypothesis.TemporalDescriptorCohortV1Signature:
		maxGap, err := parseIntParam(parameters, "max_intra_event_gap_seconds")
		if err != nil {
			return nil, fmt.Errorf("parse %s parameters %q: %w",
				hypothesis.TemporalDescriptorCohortV1Signature, parameters, err)
		}
		minSize, err := parseIntParam(parameters, "min_campaign_size")
		if err != nil {
			return nil, fmt.Errorf("parse %s parameters %q: %w",
				hypothesis.TemporalDescriptorCohortV1Signature, parameters, err)
		}
		return hypothesis.TemporalDescriptorCohortV1{
			MinCampaignSize:         minSize,
			MaxIntraEventGapSeconds: maxGap,
		}, nil

	default:
		return nil, fmt.Errorf("%w: %q (known: %s)",
			ErrPatternUnknown, signature,
			hypothesis.TemporalDescriptorCohortV1Signature)
	}
}
