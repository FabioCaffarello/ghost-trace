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

const behavioralClusterFormationMessageType = "ghosttrace.events.v1.BehavioralClusterFormation"

// Sentinels specific to BC formation replay.
var (
	// ErrPatternUnknown: the BC formation's pattern_signature does
	// not resolve to a registered FormationPattern.
	ErrPatternUnknown = errors.New("replay: pattern_signature not registered")

	// ErrPatternParameterMismatch: the resolved pattern's Parameters()
	// does not match the BC formation's pattern_parameters byte-for-
	// byte (mirrors §0084's ErrDefinitionParameterMismatch).
	ErrPatternParameterMismatch = errors.New("replay: pattern parameters drifted")
)

// BehavioralClusterFormationReport is the per-ReplayBehavioralClusterFormation
// outcome. Match is true iff one of the reconstructed-formations has a
// content-hash equal to the input formation's.
type BehavioralClusterFormationReport struct {
	// TargetHashHex is the hex content-hash of the BC formation being
	// replayed.
	TargetHashHex string

	// Match is true iff the reconstructed formation set contains a
	// formation whose canonical content-hash equals TargetHashHex.
	Match bool

	// RecomputedHashHex is the content-hash of the matching
	// reconstructed formation (only populated when Match=true).
	RecomputedHashHex string

	// PatternSignature is the pattern identifier read from the
	// original formation.
	PatternSignature string

	// PatternParameters is the canonical-parameter string read from
	// the original formation.
	PatternParameters string

	// ReconstructedFormationCount is the number of BC formations the
	// pattern produced when re-run against the time-filtered substrate.
	// Useful diagnostic: large counts when the original substrate had
	// many descriptor groups; zero when no group met the
	// min-cluster-size threshold.
	ReconstructedFormationCount int

	// ContributingObservationCount is the number of DeclaredSessions
	// present in the substrate at the original formation's commit
	// time (i.e. the size of the FormationContext the pattern saw at
	// replay).
	ContributingObservationCount int

	// MaxCommittedAtNs is the substrate-time bound used to filter
	// DeclaredSessions for the reconstruction (= original formation
	// row's committed_at). All DeclaredSessions with committed_at ≤
	// this value were visible to the replayed pattern.
	MaxCommittedAtNs int64
}

// ReplayBehavioralClusterFormation performs Phase 3 reconstructive
// replay of a BehavioralCluster formation per decision-log §0086 +
// docs/architecture/replay-model.md L25-28. Walks the substrate
// filtered to events with committed_at ≤ the original formation's
// committed_at, re-runs the formation pattern against this filtered
// view, and searches for a reconstructed formation whose canonical
// content-hash matches the original.
//
// Phase 3 vs Phase 1: Phase 1 verifies deterministic re-derivation
// of a single Cat II record from a single Cat I source. Phase 3
// reconstructs a Cat III hypothesis from the substrate-at-commit-
// time view of all Cat I observations. The BC formation pattern is
// deterministic given its FormationContext, so in practice Match
// SHOULD be true; per replay-model.md L27 a divergence is
// "acknowledged to potentially yield a different result" but for
// our concrete patterns a divergence would indicate either (a)
// pattern-implementation drift since the original commit, or (b)
// substrate-time vs event-time inconsistency in how the original
// pattern selected its contributing observations.
//
// Errors:
//   - ErrTargetNotFound: formation hash not in substrate.
//   - ErrTargetWrongType: hash resolves to non-BehavioralClusterFormation.
//   - ErrPatternUnknown: pattern_signature unresolvable.
//   - ErrPatternParameterMismatch: pattern.Parameters() != stored.
func ReplayBehavioralClusterFormation(ctx context.Context, sub *substrate.Substrate, targetHash [32]byte) (BehavioralClusterFormationReport, error) {
	row, err := sub.LookupRow(ctx, targetHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return BehavioralClusterFormationReport{}, fmt.Errorf("%w: %x", ErrTargetNotFound, targetHash)
		}
		return BehavioralClusterFormationReport{}, fmt.Errorf("replay.ReplayBehavioralClusterFormation: lookup target: %w", err)
	}
	if row.MessageType != behavioralClusterFormationMessageType {
		return BehavioralClusterFormationReport{}, fmt.Errorf("%w: %x is %q (expected %s)",
			ErrTargetWrongType, targetHash, row.MessageType, behavioralClusterFormationMessageType)
	}

	payload, err := sub.ReadBlob(ctx, targetHash)
	if err != nil {
		return BehavioralClusterFormationReport{}, fmt.Errorf("replay.ReplayBehavioralClusterFormation: read target blob: %w", err)
	}
	original := &eventsv1.BehavioralClusterFormation{}
	if err := proto.Unmarshal(payload, original); err != nil {
		return BehavioralClusterFormationReport{}, fmt.Errorf("replay.ReplayBehavioralClusterFormation: unmarshal target: %w", err)
	}

	pattern, err := ResolveBCFormationPattern(original.PatternSignature, original.PatternParameters)
	if err != nil {
		return BehavioralClusterFormationReport{}, err
	}
	if pattern.Parameters() != original.PatternParameters {
		return BehavioralClusterFormationReport{}, fmt.Errorf("%w: pattern %q produced %q, original carried %q",
			ErrPatternParameterMismatch, pattern.Signature(),
			pattern.Parameters(), original.PatternParameters)
	}

	// Filter the FormationContext to substrate state visible at the
	// original formation's commit time.
	fctx, err := hypothesis.CollectFormationContextAt(ctx, sub, row.CommittedAt)
	if err != nil {
		return BehavioralClusterFormationReport{}, fmt.Errorf("replay.ReplayBehavioralClusterFormation: collect formation context: %w", err)
	}

	formations := pattern.Form(fctx, original.FormationAt)

	report := BehavioralClusterFormationReport{
		TargetHashHex:                canonical.HashHex(targetHash),
		PatternSignature:             original.PatternSignature,
		PatternParameters:            original.PatternParameters,
		ReconstructedFormationCount:  len(formations),
		ContributingObservationCount: len(fctx.DeclaredSessions()),
		MaxCommittedAtNs:             row.CommittedAt,
	}

	// Search the reconstructed formations for a hash match. The
	// pattern sets the per-formation fields except pattern_signature
	// + pattern_parameters (which FormAll normally fills in); we
	// replicate that step here for the marshal+hash to be byte-
	// equivalent to what the original FormAll would have committed.
	for _, ev := range formations {
		ev.PatternSignature = pattern.Signature()
		ev.PatternParameters = pattern.Parameters()
		_, recomputedHash, err := canonical.MarshalAndHash(ev)
		if err != nil {
			return report, fmt.Errorf("replay.ReplayBehavioralClusterFormation: marshal candidate: %w", err)
		}
		if bytes.Equal(targetHash[:], recomputedHash[:]) {
			report.Match = true
			report.RecomputedHashHex = canonical.HashHex(recomputedHash)
			return report, nil
		}
	}

	return report, nil
}

// ResolveBCFormationPattern maps a BehavioralClusterFormation's
// (pattern_signature, pattern_parameters) tuple back to a concrete
// hypothesis.FormationPattern implementation suitable for replay.
//
// Currently supports session-descriptor-shared-v1
// ("min_cluster_size=N"). Other BC formation patterns register
// here as they land. ErrPatternUnknown for unrecognized signatures;
// format errors for malformed parameters.
func ResolveBCFormationPattern(signature, parameters string) (hypothesis.FormationPattern, error) {
	switch signature {
	case hypothesis.SessionDescriptorSharedV1Signature:
		minClusterSize, err := parseIntParam(parameters, "min_cluster_size")
		if err != nil {
			return nil, fmt.Errorf("parse %s parameters %q: %w",
				hypothesis.SessionDescriptorSharedV1Signature, parameters, err)
		}
		return hypothesis.SessionDescriptorSharedV1{MinClusterSize: minClusterSize}, nil

	default:
		return nil, fmt.Errorf("%w: %q (known: %s)",
			ErrPatternUnknown, signature,
			hypothesis.SessionDescriptorSharedV1Signature)
	}
}

