package replay

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

const operationalDecisionAuditMessageType = "ghosttrace.events.v1.OperationalDecisionAudit"

// DecisionProvenanceReport reconstructs the full observation→inference→
// decision chain for an OperationalDecisionAudit per decision-log §0222.
// It is the fourth hop of the §0221 replay capability: from the
// committed decision, resolve the influencing Cat III hypotheses and,
// through each, the Cat I observations that grounded the inference —
// answering "why was this verdict reached, on what evidence?" entirely
// from the substrate (audit-grade navigability).
type DecisionProvenanceReport struct {
	DecisionHashHex string `json:"decision_hash_hex"`
	Verdict         string `json:"verdict"`
	SubjectActorRef string `json:"subject_actor_ref"`
	PolicyRef       string `json:"policy_ref"`
	OperatorRef     string `json:"operator_ref"`
	DecidedAt       int64  `json:"decided_at"`

	// InfluencingHypotheses is the per-hypothesis provenance reconstruction
	// (each resolving back to its Cat I source observations).
	InfluencingHypotheses []FormationProvenanceReport `json:"influencing_hypotheses"`

	// DirectSourceObservations resolves the audit's own
	// source_observation_hashes (the §2.3 chain it carried directly).
	DirectSourceObservations []ResolvedSource `json:"direct_source_observations"`
}

// ReconstructDecisionProvenance resolves an OperationalDecisionAudit's
// full provenance chain. Returns ErrTargetNotFound / ErrTargetWrongType
// for an absent or mistyped target. Influencing hypotheses that are not
// AutomationGroupFormations (or are absent) are recorded with a
// resolution note rather than failing the whole reconstruction (a
// missing antecedent is itself an auditable signal).
func ReconstructDecisionProvenance(ctx context.Context, sub *substrate.Substrate, decisionHash [32]byte) (DecisionProvenanceReport, error) {
	row, err := sub.LookupRow(ctx, decisionHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return DecisionProvenanceReport{}, fmt.Errorf("%w: %x", ErrTargetNotFound, decisionHash)
		}
		return DecisionProvenanceReport{}, fmt.Errorf("replay.ReconstructDecisionProvenance: lookup target: %w", err)
	}
	if row.MessageType != operationalDecisionAuditMessageType {
		return DecisionProvenanceReport{}, fmt.Errorf("%w: %x is %q (expected %s)",
			ErrTargetWrongType, decisionHash, row.MessageType, operationalDecisionAuditMessageType)
	}

	payload, err := sub.ReadBlob(ctx, decisionHash)
	if err != nil {
		return DecisionProvenanceReport{}, fmt.Errorf("replay.ReconstructDecisionProvenance: read target blob: %w", err)
	}
	audit := &eventsv1.OperationalDecisionAudit{}
	if err := proto.Unmarshal(payload, audit); err != nil {
		return DecisionProvenanceReport{}, fmt.Errorf("replay.ReconstructDecisionProvenance: unmarshal target: %w", err)
	}

	report := DecisionProvenanceReport{
		DecisionHashHex: canonical.HashHex(decisionHash),
		Verdict:         verdictName(audit.GetDecisionType()),
		SubjectActorRef: audit.GetSubjectActorRef(),
		PolicyRef:       audit.GetPolicyRef(),
		OperatorRef:     audit.GetOperatorRef(),
		DecidedAt:       audit.GetDecidedAt(),
	}

	// Resolve each influencing Cat III hypothesis to its own provenance.
	for _, hh := range audit.GetInfluencingHypothesisHashes() {
		if len(hh) != 32 {
			continue
		}
		var h [32]byte
		copy(h[:], hh)
		prov, err := ReconstructAutomationGroupProvenance(ctx, sub, h)
		if err != nil {
			// Record a stub for an unresolved / non-AG antecedent rather
			// than aborting; the absence is the auditable signal.
			report.InfluencingHypotheses = append(report.InfluencingHypotheses, FormationProvenanceReport{
				FormationHashHex: canonical.HashHex(h),
			})
			continue
		}
		report.InfluencingHypotheses = append(report.InfluencingHypotheses, prov)
	}

	// Resolve the audit's own source observations.
	for _, sh := range audit.GetSourceObservationHashes() {
		report.DirectSourceObservations = append(report.DirectSourceObservations, resolveSource(ctx, sub, sh))
	}

	return report, nil
}

// verdictName renders the enum as its short operator-facing name.
func verdictName(v eventsv1.OperationalDecisionAudit_DecisionType) string {
	switch v {
	case eventsv1.OperationalDecisionAudit_DECISION_TYPE_ALLOW:
		return "ALLOW"
	case eventsv1.OperationalDecisionAudit_DECISION_TYPE_CHALLENGE:
		return "CHALLENGE"
	case eventsv1.OperationalDecisionAudit_DECISION_TYPE_BLOCK:
		return "BLOCK"
	case eventsv1.OperationalDecisionAudit_DECISION_TYPE_SHADOW:
		return "SHADOW"
	default:
		return "UNSPECIFIED"
	}
}
