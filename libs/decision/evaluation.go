package decision

import (
	"github.com/FabioCaffarello/ghost-trace/libs/feature"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/libs/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/libs/policy"
	"github.com/FabioCaffarello/ghost-trace/libs/snapshot"
)

// buildEvaluation is the anti-corruption boundary between domain values
// and the durable Evaluation record. Every proto literal this package
// archives is built here and nowhere else — the M3 drift (a feature
// scored but never persisted) happened precisely because this mapping
// was inlined at a call site.
//
// The feature vector comes from snapshot.FromState rather than a second
// copy of the same nineteen assignments. It had BEEN a second copy: the
// evaluation record and the session snapshot each built a FeatureState
// with identical code, and nothing compared them. The guard test checks
// that every feature field maps to a proto field — not that both
// builders set it — so a field added to one and forgotten in the other
// would have passed. That divergence is precisely what a decision engine
// judging from snapshots while the archive stores evaluations would turn
// into a service deciding on one feature vector and recording another.
func buildEvaluation(tenantID, sessionID string, in Input, out Output,
	st policy.State, j policy.Judgement, decidedAt int64) *eventsv1.Evaluation {

	rec := &eventsv1.Evaluation{
		TenantId:           tenantID,
		EvaluationId:       out.EvaluationID,
		SessionId:          sessionID,
		Action:             in.Action,
		SubjectId:          in.SubjectID,
		DecidedAt:          decidedAt,
		Decision:           out.Decision,
		ShadowDecision:     out.ShadowDecision,
		Mode:               out.Mode,
		Score:              float32(j.Score()),
		Confidence:         float32(j.Confidence()),
		EvidenceEvents:     out.EvidenceEvents,
		EvidenceDurationMs: out.EvidenceMs,
		PolicyRef:          policy.Ref,
		FeatureSetRef:      feature.SetRef,
		Features:           snapshot.FromState(st),
	}
	for _, rs := range j.Reasons() {
		rec.Reasons = append(rec.Reasons, &eventsv1.Reason{
			Code: rs.Code, Weight: float32(rs.Weight),
		})
	}
	return rec
}
