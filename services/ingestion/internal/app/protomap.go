// protomap.go is the single anti-corruption boundary between domain
// values and the durable proto records. Every proto literal the
// application archives is built here and nowhere else — the M3 drift
// (a feature scored but never persisted) happened precisely because
// this mapping was inlined at a call site, and
// TestFeatureStateProtoCoversFeatureVector guards the FeatureState
// half of it permanently.
package app

import (
	"github.com/FabioCaffarello/ghost-trace/libs/feature"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/libs/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/libs/policy"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/session"
)

func buildSessionStart(st *session.State, c session.Client) *eventsv1.SessionStart {
	return &eventsv1.SessionStart{
		TenantId:      st.TenantID,
		SessionId:     st.ID,
		StartedAt:     st.StartedAt.UnixNano(),
		PagePath:      st.PagePath,
		PointerType:   c.PointerType,
		Touch:         c.Touch,
		ViewportW:     uint32(c.Viewport[0]),
		ViewportH:     uint32(c.Viewport[1]),
		TzOffsetMin:   int32(c.TZOffsetMin),
		ReducedMotion: c.ReducedMotion,
	}
}

func buildTelemetryBatch(env TelemetryEnvelope, tenantID, sessionID string, receivedAt int64) *eventsv1.TelemetryBatch {
	batch := &eventsv1.TelemetryBatch{
		TenantId:   tenantID,
		SessionId:  sessionID,
		Seq:        env.Seq,
		SentAtMs:   env.SentAtMs,
		PagePath:   env.PagePath,
		ViewportW:  uint32(env.Viewport[0]),
		ViewportH:  uint32(env.Viewport[1]),
		ReceivedAt: receivedAt,
	}

	for i := range env.Events {
		ev := &env.Events[i]
		switch ev.Type {
		case "pointer":
			pe := &eventsv1.PointerEvent{TMs: ev.T, Src: ev.Src}
			for _, p := range ev.Pts {
				pe.Pts = append(pe.Pts, &eventsv1.PointerPoint{
					X: p[0], Y: p[1], DtMs: uint32(p[2]),
				})
			}
			batch.PointerEvents = append(batch.PointerEvents, pe)
		case "key":
			batch.KeyEvents = append(batch.KeyEvents, &eventsv1.KeyEvent{
				TMs: ev.T, Phase: ev.Phase, KeyClass: ev.KeyClass, Target: ev.Target,
			})
		case "scroll":
			batch.ScrollEvents = append(batch.ScrollEvents, &eventsv1.ScrollEvent{
				TMs: ev.T, Dy: ev.Dy, Mode: ev.Mode,
			})
		case "focus":
			batch.FocusEvents = append(batch.FocusEvents, &eventsv1.FocusEvent{
				TMs: ev.T, State: ev.State, Target: ev.Target,
			})
		case "visibility":
			batch.VisibilityEvents = append(batch.VisibilityEvents, &eventsv1.VisibilityEvent{
				TMs: ev.T, State: ev.State,
			})
		case "form":
			batch.FormEvents = append(batch.FormEvents, &eventsv1.FormEvent{
				TMs: ev.T, Target: ev.Target, Action: ev.Action,
			})
		}
	}
	return batch
}

// batchHasEvents reports whether a batch carries anything worth
// archiving. An empty batch still updates session counters but is not a
// record.
func batchHasEvents(b *eventsv1.TelemetryBatch) bool {
	return len(b.PointerEvents) > 0 || len(b.KeyEvents) > 0 ||
		len(b.ScrollEvents) > 0 || len(b.FocusEvents) > 0 ||
		len(b.VisibilityEvents) > 0 || len(b.FormEvents) > 0
}

func buildEvaluation(tenantID, sessionID string, in DecideInput, out DecideOutput,
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
		Features:           buildFeatureState(st),
	}
	for _, rs := range j.Reasons() {
		rec.Reasons = append(rec.Reasons, &eventsv1.Reason{
			Code: rs.Code, Weight: float32(rs.Weight),
		})
	}
	return rec
}

func buildFeatureState(st policy.State) *eventsv1.FeatureState {
	return &eventsv1.FeatureState{
		PointerStraightness:     float32(st.Pointer.Straightness),
		PointerSegments:         st.Pointer.Segments,
		PointerPathPx:           float32(st.Pointer.PathPx),
		PointerPoints:           st.Pointer.Points,
		KeyFlightCv:             float32(st.Keystroke.FlightCV),
		KeyDwellCv:              float32(st.Keystroke.DwellCV),
		KeyMeanDwellMs:          float32(st.Keystroke.MeanDwellMs),
		KeyExactRepeatRatio:     float32(st.Keystroke.ExactRepeatRatio),
		KeyIntervals:            st.Keystroke.Intervals,
		ProgrammaticScrollRatio: float32(st.Interaction.ProgrammaticScrollRatio),
		ScrollEvents:            st.Interaction.ScrollEvents,
		FocusTransitions:        st.Interaction.FocusTransitions,
		HiddenPeriods:           st.Interaction.HiddenPeriods,
		Pastes:                  st.Interaction.Pastes,
		Injections:              st.Interaction.Injections,
		InjectedFields:          st.Interaction.InjectedFields,
		Autofills:               st.Interaction.Autofills,
		DistinctFocusTargets:    st.Interaction.DistinctFocusTargets,
		KeyEvents:               st.Keystroke.Keys,
	}
}
