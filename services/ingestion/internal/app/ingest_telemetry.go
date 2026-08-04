package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/feature"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/session"
)

// TelemetryEnvelope is the §2 wire envelope, transport-free.
type TelemetryEnvelope struct {
	SessionToken string
	Seq          uint32
	SentAtMs     uint32
	PagePath     string
	Viewport     [2]int
	Events       []TelemetryEvent
}

// TelemetryEvent is one §2 event. Which fields are meaningful depends
// on Type; the rest stay zero.
type TelemetryEvent struct {
	Type string
	T    uint32

	// pointer
	Src string
	Pts [][3]int32

	// key — timing and coarse class only, never content (§2, §6)
	Phase    string
	KeyClass string
	Target   string

	// scroll
	Dy   int32
	Mode string

	// focus / visibility
	State string

	// form
	Action string
}

// IngestTelemetry feeds one envelope into the session's feature
// accumulators and archives the batch.
//
// Returns ErrSessionNotFound for an unknown or expired token — the
// transport decides what that means (for the SDK it is expected loss,
// not an error).
//
// The critical section is deliberately minimal: pointer polylines are
// converted before taking the session lock, and the archive record is
// built after releasing it. The store lock serializes every session in
// the process; transport and serialization work must never run under
// it.
func (a *App) IngestTelemetry(ctx context.Context, env TelemetryEnvelope) error {
	// Pre-convert pointer polylines outside the lock.
	points := make([][]feature.Point, len(env.Events))
	for i := range env.Events {
		ev := &env.Events[i]
		if ev.Type != "pointer" {
			continue
		}
		pts := make([]feature.Point, 0, len(ev.Pts))
		for _, p := range ev.Pts {
			pts = append(pts, feature.Point{X: p[0], Y: p[1], DtMs: uint32(p[2])})
		}
		points[i] = pts
	}

	var tenantID, sessionID string
	err := a.sessions.With(env.SessionToken, func(st *session.State) {
		tenantID, sessionID = st.TenantID, st.ID
		st.ObserveBatch(env.Seq, env.SentAtMs)

		for i := range env.Events {
			ev := &env.Events[i]
			st.ObserveEventTime(ev.T)

			// Unknown types are dropped silently rather than rejected:
			// the collect policy is server-driven and may change at any
			// time, so an SDK sending a type this build does not know is
			// expected behaviour, not a client error (contract §7).
			switch ev.Type {
			case "pointer":
				st.Pointer.Add(ev.T, points[i])
			case "key":
				st.Keystroke.AddKey(ev.T, ev.Phase, ev.KeyClass, ev.Target)
			case "scroll":
				st.Interaction.AddScroll(ev.Mode)
			case "focus":
				st.Interaction.AddFocus(ev.Target)
			case "visibility":
				st.Interaction.AddVisibility(ev.State)
			case "form":
				st.Interaction.AddForm(ev.Action, ev.Target)
			}
		}
	})
	if err != nil {
		if errors.Is(err, session.ErrNotFound) {
			return ErrSessionNotFound
		}
		return fmt.Errorf("ingest telemetry: %w", err)
	}

	batch := buildTelemetryBatch(env, tenantID, sessionID, a.now().UnixNano())
	if batchHasEvents(batch) {
		a.archiveBestEffort(ctx, batch, batch.ReceivedAt, "telemetry", "session_id", sessionID)
	}
	return nil
}
