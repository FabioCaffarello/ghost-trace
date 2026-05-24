// Package signatures is the F3 inference layer per Domain Pack v0.1
// §0143: signature engines that detect anti-bot patterns in Cat I
// observations and produce Cat III formation candidates. This is the
// first F3 work establishing the structural surface; concrete
// signatures live in sibling files and follow the §0152 cdp_marker_density_v1
// precedent.
//
// Per §3 N3 (no autonomous irreversible action): signature engines
// produce CANDIDATES; they do NOT commit formation events directly.
// An orchestrator (CLI command, scheduled job, or operator-invoked
// path) consumes candidates and decides whether to commit. The
// separation preserves the operator-elected-substrate-commit
// discipline established at §0119 + §0141 E1.
//
// Per §2.4 + §0021 substrate-time generation: when a signature reads
// from prior promoted hypotheses (forming a hypothesis under their
// influence), the orchestrator that commits the formation event must
// declare the `influenced_by` chain at formation time. Signatures
// surface the source-hashes that contributed; the orchestrator
// resolves those into typed `subject_ref_*` chain commitments.
package signatures

import (
	"context"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
)

// FormationCandidate is the output of a signature evaluation: a
// proposed Cat III hypothesis formation that the orchestrator may
// commit to substrate. The candidate carries the structural inputs
// the orchestrator needs to materialize the formation event:
//
//   - ActorRefs: the actors the hypothesis pertains to (per ontology
//     for AutomationGroup: "set of actors whose behavioral patterns
//     match a signature of automated operation"). MVP signatures may
//     emit single-actor candidates (set size = 1).
//
//   - SourceHashes: the Cat I observation hashes that contributed to
//     the candidate's recognition. Surfaced for the orchestrator to
//     thread into the formation event's source_event_hashes field per
//     §0139 hash-list discipline.
//
//   - EvidenceCount: count of Cat I roots reachable via the
//     observation chain; used by the orchestrator to compute
//     `evidential_independence` per §0133 Q3-α source-count ratio at
//     commit time.
//
//   - ConfidenceHint: a producer-side numeric in [0, 1] suggesting
//     the signature's own confidence in the candidate. Advisory
//     only — the orchestrator is the authoritative source for the
//     committed `confidence` value per §2.6 paired-dimension at the
//     marshalling boundary.
type FormationCandidate struct {
	// SignatureName identifies the signature that produced the
	// candidate (e.g., "cdp_marker_density_v1"). Versioned per the
	// op-def-versioning OMQ surface (lifecycle-semantics.md);
	// inception-phase: producer-supplied string.
	SignatureName string

	// HypothesisSubtype is the Cat III subtype the candidate proposes
	// to form. Inception-phase MVP: only AutomationGroup. Future
	// signatures may produce BehavioralCluster / CampaignHypothesis /
	// CoordinationRing candidates.
	HypothesisSubtype HypothesisSubtype

	// ActorRefs are the actors the hypothesis pertains to.
	ActorRefs []string

	// SourceHashes are the BLAKE3 content-hashes of the Cat I
	// observations that contributed to recognition. Orchestrator
	// threads into formation event's source_event_hashes field.
	SourceHashes [][]byte

	// EvidenceCount is the count of Cat I roots reachable via the
	// observation chain. Used by orchestrator for §0133 EI computation.
	EvidenceCount uint32

	// ConfidenceHint is the signature's advisory confidence in [0, 1].
	// Orchestrator is authoritative for committed confidence.
	ConfidenceHint float64
}

// HypothesisSubtype identifies the Cat III subtype a formation
// candidate proposes. Mirrors the ontology subtype enumeration per
// §0010 Q2-A.2. MVP scope (§0152): AutomationGroup only.
type HypothesisSubtype int

const (
	// HypothesisSubtypeUnknown is the zero value; signatures that
	// emit this value are structurally malformed.
	HypothesisSubtypeUnknown HypothesisSubtype = iota
	// HypothesisSubtypeAutomationGroup — set of actors whose
	// behavioral patterns match a signature of automated operation.
	HypothesisSubtypeAutomationGroup
	// HypothesisSubtypeBehavioralCluster — set of actors operating
	// under a common underlying entity (future signature scope).
	HypothesisSubtypeBehavioralCluster
	// HypothesisSubtypeCampaignHypothesis — set of events forming
	// a unified operation (future signature scope).
	HypothesisSubtypeCampaignHypothesis
	// HypothesisSubtypeCoordinationRing — set of actors coordinating
	// action (future signature scope).
	HypothesisSubtypeCoordinationRing
)

// Signature is the F3 inference engine interface. Concrete signatures
// implement Evaluate against the Cat I observation surface relevant
// to their detection axis. Stateless across invocations (orchestrator
// is responsible for window selection); deterministic given input.
//
// Per §0143 instrumentation-by-subtype-fonte-morfologia discipline:
// signatures should be testable in isolation per subtype/source/
// chain-morphology. The Evaluate interface accepts a typed observation
// slice; orchestrator selects the slice per its windowing strategy.
type Signature interface {
	// Name returns the signature's name for instrumentation +
	// versioning (e.g., "cdp_marker_density_v1").
	Name() string

	// Subtype returns the Cat III hypothesis subtype this signature
	// produces candidates for. MVP: AutomationGroup only.
	Subtype() HypothesisSubtype

	// EvaluateBrowser evaluates the signature against a slice of
	// BrowserObservation records. Returns formation candidates (may
	// be empty); error indicates a structural failure (e.g.,
	// malformed input), not "no candidates found".
	EvaluateBrowser(ctx context.Context, observations []*eventsv1.BrowserObservation) ([]*FormationCandidate, error)
}

// BrowserSignature is a convenience marker interface for signatures
// that consume BrowserObservation input only. Allows the orchestrator
// to dispatch by input class without type assertions.
type BrowserSignature interface {
	Signature
	isBrowserSignature()
}
