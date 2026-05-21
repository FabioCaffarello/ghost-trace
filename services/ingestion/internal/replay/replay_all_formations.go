package replay

import (
	"context"
	"fmt"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// ReplayAllBehavioralClusterFormations walks every
// BehavioralClusterFormation in the substrate, runs Phase 3
// reconstructive replay against each, and aggregates outcomes into a
// BatchReplayReport. Mirrors §0085's ReplayAllOperationalSessions but
// for Phase 3 BC.
//
// Per-formation cost: one CollectFormationContextAt walk (each
// formation has its own committed_at bound). The naive call-per-
// target-CLI-in-a-loop approach would do the same number of walks;
// no batch optimization is possible without the substrate-scan +
// client-side-filter refactor named in §0090's carry-forward.
func ReplayAllBehavioralClusterFormations(ctx context.Context, sub *substrate.Substrate) (BatchReplayReport, error) {
	return replayAllFormations(ctx, sub, behavioralClusterFormationMessageType, func(ctx context.Context, sub *substrate.Substrate, hash [32]byte) (BatchReplayEntry, error) {
		report, err := ReplayBehavioralClusterFormation(ctx, sub, hash)
		return bcReportToEntry(report), err
	})
}

// ReplayAllAutomationGroupFormations is the AG analog.
func ReplayAllAutomationGroupFormations(ctx context.Context, sub *substrate.Substrate) (BatchReplayReport, error) {
	return replayAllFormations(ctx, sub, automationGroupFormationMessageType, func(ctx context.Context, sub *substrate.Substrate, hash [32]byte) (BatchReplayEntry, error) {
		report, err := ReplayAutomationGroupFormation(ctx, sub, hash)
		return agReportToEntry(report), err
	})
}

// ReplayAllCampaignHypothesisFormations is the CH analog.
func ReplayAllCampaignHypothesisFormations(ctx context.Context, sub *substrate.Substrate) (BatchReplayReport, error) {
	return replayAllFormations(ctx, sub, campaignHypothesisFormationMessageType, func(ctx context.Context, sub *substrate.Substrate, hash [32]byte) (BatchReplayEntry, error) {
		report, err := ReplayCampaignHypothesisFormation(ctx, sub, hash)
		return chReportToEntry(report), err
	})
}

// ReplayAllCoordinationRingFormations is the CR analog.
func ReplayAllCoordinationRingFormations(ctx context.Context, sub *substrate.Substrate) (BatchReplayReport, error) {
	return replayAllFormations(ctx, sub, coordinationRingFormationMessageType, func(ctx context.Context, sub *substrate.Substrate, hash [32]byte) (BatchReplayEntry, error) {
		report, err := ReplayCoordinationRingFormation(ctx, sub, hash)
		return crReportToEntry(report), err
	})
}

// replayAllFormations is the shared skeleton for Phase 3 batch replay
// across all four Cat III subtypes. Walks the substrate filtered to
// messageType, invokes replayOne for each matching row, accumulates
// outcomes into BatchReplayReport.
//
// The replayOne callback returns a BatchReplayEntry directly so the
// per-subtype function can convert its typed Report into the shared
// entry shape without leaking the subtype-specific Report type into
// the batch skeleton.
func replayAllFormations(
	ctx context.Context,
	sub *substrate.Substrate,
	messageType string,
	replayOne func(ctx context.Context, sub *substrate.Substrate, hash [32]byte) (BatchReplayEntry, error),
) (BatchReplayReport, error) {
	report := BatchReplayReport{}

	walkErr := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != messageType {
			return nil
		}
		report.Total++

		entry, err := replayOne(ctx, sub, row.EventHash)
		if err != nil {
			// Per-target replay errored out (substrate-precondition
			// failure: unknown pattern, missing source, etc.). Record
			// as OutcomeError with the error message.
			entry = BatchReplayEntry{
				TargetHashHex: canonical.HashHex(row.EventHash),
				Outcome:       OutcomeError,
				Reason:        err.Error(),
			}
		}

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
		return report, fmt.Errorf("replay.replayAllFormations[%s]: walk: %w", messageType, walkErr)
	}
	return report, nil
}

// Per-subtype Report → BatchReplayEntry converters. Each carries the
// match/drift outcome; recomputed hash + drift diagnostic populate
// when Match=false.

func bcReportToEntry(r BehavioralClusterFormationReport) BatchReplayEntry {
	e := BatchReplayEntry{TargetHashHex: r.TargetHashHex}
	if r.Match {
		e.Outcome = OutcomeMatch
		e.RecomputedHashHex = r.RecomputedHashHex
	} else {
		e.Outcome = OutcomeDrift
		// Phase 3 drift: the reconstructed formation set produced N
		// candidates, none of which matched the target hash. The
		// diagnostic surfaces the reconstruction shape so operators
		// can investigate.
		e.Reason = fmt.Sprintf("reconstructed %d candidate formations from %d contributing observations; none matched target hash",
			r.ReconstructedFormationCount, r.ContributingObservationCount)
	}
	return e
}

func agReportToEntry(r AutomationGroupFormationReport) BatchReplayEntry {
	e := BatchReplayEntry{TargetHashHex: r.TargetHashHex}
	if r.Match {
		e.Outcome = OutcomeMatch
		e.RecomputedHashHex = r.RecomputedHashHex
	} else {
		e.Outcome = OutcomeDrift
		e.Reason = fmt.Sprintf("reconstructed %d candidate formations from %d contributing observations; none matched target hash",
			r.ReconstructedFormationCount, r.ContributingObservationCount)
	}
	return e
}

func chReportToEntry(r CampaignHypothesisFormationReport) BatchReplayEntry {
	e := BatchReplayEntry{TargetHashHex: r.TargetHashHex}
	if r.Match {
		e.Outcome = OutcomeMatch
		e.RecomputedHashHex = r.RecomputedHashHex
	} else {
		e.Outcome = OutcomeDrift
		e.Reason = fmt.Sprintf("reconstructed %d candidate formations from %d contributing observations; none matched target hash",
			r.ReconstructedFormationCount, r.ContributingObservationCount)
	}
	return e
}

func crReportToEntry(r CoordinationRingFormationReport) BatchReplayEntry {
	e := BatchReplayEntry{TargetHashHex: r.TargetHashHex}
	if r.Match {
		e.Outcome = OutcomeMatch
		e.RecomputedHashHex = r.RecomputedHashHex
	} else {
		e.Outcome = OutcomeDrift
		e.Reason = fmt.Sprintf("reconstructed %d candidate formations from %d contributing observations; none matched target hash",
			r.ReconstructedFormationCount, r.ContributingObservationCount)
	}
	return e
}
