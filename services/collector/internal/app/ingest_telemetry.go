package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/FabioCaffarello/ghost-trace/libs/feature"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/libs/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/libs/policy"
	"github.com/FabioCaffarello/ghost-trace/libs/snapshot"
	"github.com/FabioCaffarello/ghost-trace/services/collector/internal/session"
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
	// Captured inside the store callback, published after it returns.
	var (
		snapshotState policy.State
		snapshotMs    uint32
	)

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

		// Copied by value while the lock is held, for the same reason
		// Decide does it: concurrent batches keep mutating this state,
		// and reading it after With returns would race with them.
		snapshotState = policy.State{
			Pointer:     st.Pointer.State(),
			Keystroke:   st.Keystroke.State(),
			Interaction: st.Interaction.State(),
		}
		snapshotMs = st.LastEventMs
	})
	if err != nil {
		if errors.Is(err, session.ErrNotFound) {
			return ErrSessionNotFound
		}
		return fmt.Errorf("ingest telemetry: %w", err)
	}

	// Published after the state is durable in memory and before the
	// archive write, so a reader sees the batch's effect as soon as this
	// call returns. Best effort: a snapshot store that is down makes
	// another process's view stale, which is not a reason to reject
	// telemetry here.
	snapCtx, cancelSnap := bestEffort(ctx)
	defer cancelSnap()
	if err := a.snapshots.Put(snapCtx, env.SessionToken, &eventsv1.SessionSnapshot{
		TenantId:    tenantID,
		SessionId:   sessionID,
		LastEventMs: snapshotMs,
		WrittenAt:   a.now().UnixNano(),
		Features:    snapshot.FromState(snapshotState),
	}); err == nil {
		a.loss.Written(KindSnapshot)
	} else if !errors.Is(err, ErrNoSnapshotStore) {
		// No store to publish to is not a loss, exactly as it is not
		// one for the archive.
		a.loss.Dropped(KindSnapshot, reasonFor(err))
		a.log.WarnContext(ctx, "session snapshot not published",
			"err", err, "session_id", sessionID)
	}

	batch := buildTelemetryBatch(env, tenantID, sessionID, a.now().UnixNano())
	if batchHasEvents(batch) {
		a.archiveBestEffort(ctx, batch, batch.ReceivedAt, KindTelemetry, "session_id", sessionID)
	}
	return nil
}
