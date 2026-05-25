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

// EvaluationStats carries per-evaluation instrumentation counters
// per §0154 + §0143 Sub-benchmark 1 mandatory-instrumentation
// requirement. Counters are populated by the signature implementation
// during evaluation; orchestrator surfaces them in operator-facing
// output. The structure is intentionally per-evaluation (not
// cumulative across invocations); cumulative aggregation is
// orchestrator territory.
//
// Per-source breakdown via PerCollector map: collector_ref → counters.
// Enables §0143 instrumentation-by-source axis without forcing
// callers to know all collector_ref namespaces in advance.
//
// Per-chain-morphology field captured as best-effort at the signature
// layer; full morphology is observable at the hypothesis substrate
// level (post-formation-commit). Signature layer captures
// pre-formation morphology indicators (e.g., per-actor source-hash
// count is a proxy for chain-breadth-at-root).
type EvaluationStats struct {
	// ObservationsScanned is the total count of observations passed
	// to EvaluateBrowser (or equivalent).
	ObservationsScanned uint32

	// ObservationsSkippedNoActor is the count of observations skipped
	// because actor_ref was empty (signature cannot anchor a
	// hypothesis without an actor_ref per the ontology).
	ObservationsSkippedNoActor uint32

	// ObservationsSkippedWrongModality is the count of observations
	// skipped because the signature's required sub-modality was not
	// populated (e.g., cdp_marker signature skipping
	// canvas_fingerprint observations).
	ObservationsSkippedWrongModality uint32

	// ActorsAggregated is the count of distinct actor_refs the
	// signature aggregated across the input window (regardless of
	// whether they met the threshold).
	ActorsAggregated uint32

	// ActorsAboveThreshold is the count of distinct actor_refs that
	// produced a FormationCandidate (i.e., met the signature's
	// emission criteria).
	ActorsAboveThreshold uint32

	// CandidatesEmitted equals len(EvaluationResult.Candidates);
	// preserved explicitly for telemetry-readable summary without
	// requiring callers to inspect the candidates slice.
	CandidatesEmitted uint32

	// PerCollector breaks down observation counts by collector_ref
	// (§0144 (e) phenomenon-vs-record reconciliation slot). Enables
	// §0143 instrumentation-by-source axis. Key: collector_ref;
	// value: count of observations scanned with that collector_ref.
	// Empty when no observations carried a non-empty collector_ref.
	PerCollector map[string]uint32
}

// EvaluationResult is the signature evaluation output: candidates +
// instrumentation. Per §0154: returning both in a single struct keeps
// signature interface simple (one return value) while exposing the
// instrumentation surface mandated by §0143 Sub-benchmark 1.
type EvaluationResult struct {
	// Candidates are the formation candidates the signature emits.
	// May be empty; empty != error.
	Candidates []*FormationCandidate

	// Stats are the per-evaluation instrumentation counters.
	Stats EvaluationStats
}

// Signature is the F3 inference engine common-surface interface.
// Concrete signatures embed Signature + add a per-modality Evaluate*
// method via the BrowserSignature or NetworkSignature interfaces below.
// Stateless across invocations (orchestrator is responsible for window
// selection); deterministic given input.
//
// Per §0143 instrumentation-by-subtype-fonte-morfologia discipline:
// signatures populate EvaluationStats during evaluation, surfacing
// per-source + per-skip-reason counters via the EvaluationResult
// returned to the orchestrator. Per §0154: single return value
// carrying both candidates + stats.
type Signature interface {
	// Name returns the signature's name for instrumentation +
	// versioning (e.g., "cdp_marker_density_v1").
	Name() string

	// Subtype returns the Cat III hypothesis subtype this signature
	// produces candidates for.
	Subtype() HypothesisSubtype
}

// BrowserSignature is the interface for signatures that consume
// BrowserObservation input. Embeds Signature for the common surface
// + adds the modality-specific Evaluate method.
type BrowserSignature interface {
	Signature
	// EvaluateBrowser evaluates the signature against a slice of
	// BrowserObservation records. Returns EvaluationResult (candidates
	// + stats); error indicates a structural failure (e.g., malformed
	// input), not "no candidates found".
	EvaluateBrowser(ctx context.Context, observations []*eventsv1.BrowserObservation) (*EvaluationResult, error)
	isBrowserSignature()
}

// AttributionLookup is the read-only Cat II actor-attribution lookup
// surface F3 NetworkSignatures consume per decision-log §0168 Decision
// A.1 (signature-aware consumption). When a Cat I NetworkObservation
// lacks declared actor_ref (e.g., flow-level records from CIC-IDS),
// the signature consults this lookup against the source observation's
// content-hash to obtain the Cat II derived actor_ref.
//
// The interface is satisfied by attribution.AttributionView. Defined
// here (in the signatures package) to avoid an upstream dependency
// from attribution → signatures or vice-versa; the signature consumer
// depends on a small read-only interface, not on the attribution
// package's full implementation. The attribution package's
// AttributionView satisfies this interface by structural method match.
//
// Per §2.2 + §0168 epistemic-separation discipline: the lookup
// returns the Cat II derivation record's content-hash alongside the
// derived actor_ref. Consuming signatures thread the Cat II hash
// through the resulting hypothesis's source_event_hashes preserving
// the §2.3 provenance chain (Cat III hypothesis → Cat II derivation
// → Cat I observation).
type AttributionLookup interface {
	// For returns the derived actor_ref and the Cat II derivation
	// record's content-hash for the given Cat I source hash. Returns
	// ("", [32]byte{}, false) when no Cat II derivation exists for
	// the source.
	For(sourceHash [32]byte) (derivedActorRef string, attributionHash [32]byte, ok bool)
}

// NetworkSignature is the interface for signatures that consume
// NetworkObservation input. Mirrors BrowserSignature for the
// network-modality side per §0144 discriminated-union framing —
// strong typing pressures F3 signature engines to name explicitly
// which sub-modality class each signature consumes.
//
// Per §0168 Decision A.1: EvaluateNetwork accepts an optional
// AttributionLookup parameter for Cat II derived-attribution
// consumption. nil = no Cat II layer (current behavior; Cat I records
// without actor_ref are skipped at ObservationsSkippedNoActor).
// non-nil = signatures may consult the lookup for unattributed Cat I
// records + emit hypotheses whose source_event_hashes include both
// the Cat I source hash AND the Cat II derivation hash.
type NetworkSignature interface {
	Signature
	// EvaluateNetwork evaluates the signature against a slice of
	// NetworkObservation records. The attribution parameter is the
	// Cat II derived-actor lookup (nil for backward-compat behavior
	// pre-§0168). Returns EvaluationResult (candidates + stats);
	// error indicates a structural failure, not "no candidates found".
	EvaluateNetwork(ctx context.Context, observations []*eventsv1.NetworkObservation, attribution AttributionLookup) (*EvaluationResult, error)
	isNetworkSignature()
}
