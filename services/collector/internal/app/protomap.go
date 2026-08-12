// protomap.go is the anti-corruption boundary between domain values and
// the durable proto records this package archives. Every proto literal
// it writes is built here and nowhere else — the M3 drift (a feature
// scored but never persisted) happened precisely because this mapping
// was inlined at a call site.
//
// The feature vector itself is NOT built here. It has one builder,
// snapshot.FromState, shared with the evaluation record libs/decision
// writes; a second copy is what this file's own history warns about.
package app

import (
	eventsv1 "github.com/FabioCaffarello/ghost-trace/libs/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/collector/internal/session"
)

func buildSessionStart(id session.Identity, c session.Client) *eventsv1.SessionStart {
	return &eventsv1.SessionStart{
		TenantId:      id.TenantID,
		SessionId:     id.ID,
		StartedAt:     id.StartedAt.UnixNano(),
		PagePath:      id.PagePath,
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
