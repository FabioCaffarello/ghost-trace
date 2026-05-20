package hypothesis

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// AutomationGroupFormationContext is the Category III formation
// context for the AutomationGroup subtype, per decision-log §0056.
// Parallel to FormationContext (BehavioralCluster) per §0045 — each
// concrete Cat III subtype has its own FormationContext + pattern
// + walker triad, NOT a uniform abstract surface (per the §0045
// commitment to typed subtype-specific landings reaffirmed at
// §0050). The duplication is bounded (one file per subtype) and
// preferred over a generic Cat III abstraction that would invite
// the same kind of category-leak the Charter forbids in prose.
//
// At inception, AutomationGroupFormationContext exposes the SAME
// Cat I surface as the BehavioralCluster FormationContext (the
// DeclaredSessions slice). Future AutomationGroup patterns that
// need additional Cat I types (e.g. NetworkEvent for traffic-cadence
// signatures) extend this interface with new typed accessors —
// same incremental-extension procedure §0042/§0044/§0045 established.
type AutomationGroupFormationContext interface {
	DeclaredSessions() []SourceDeclaredSession
}

// AutomationGroupFormationPattern is the formation contract for the
// AutomationGroup subtype. Mirrors FormationPattern (per §0045) but
// returns AutomationGroupFormation events instead of
// BehavioralClusterFormation events — the return type is the
// hypothesis-identity-bearing structure, so it MUST be subtype-
// specific.
//
// Adding a new AutomationGroup formation pattern:
//  1. Implement AutomationGroupFormationPattern (Signature,
//     Parameters, Form).
//  2. Register it in cmd/form-automation-group/main.go's
//     resolvePattern.
//  3. Author canonical-corpus entries if the pattern produces a new
//     surface beyond what existing entries cover.
type AutomationGroupFormationPattern interface {
	Signature() string
	Parameters() string
	Form(fctx AutomationGroupFormationContext, formationAt int64) []*eventsv1.AutomationGroupFormation
}

// AutomationGroupReport is the per-FormAutomationGroupAll outcome.
// Mirrors the §0045 Report shape; separate type so future evolution
// of either pathway does not couple the two.
type AutomationGroupReport struct {
	Examined      int64
	NewlyFormed   int64
	AlreadyFormed int64
}

// automationGroupWalkerContext implements AutomationGroupFormationContext
// over a pre-collected slice of (DeclaredSession, content-hash)
// pairs. Parallel structure to walkerContext for BehavioralCluster
// per §0045.
type automationGroupWalkerContext struct {
	sessions []SourceDeclaredSession
}

func (w *automationGroupWalkerContext) DeclaredSessions() []SourceDeclaredSession {
	return w.sessions
}

// FormAutomationGroupAll walks every DeclaredSession in the
// substrate, applies pattern.Form to the collected context, and
// commits each resulting AutomationGroupFormation event via
// substrate.Append. The commit is idempotent: re-running with the
// same pattern + substrate state produces zero new rows because
// content-addressed immutability rejects duplicate content via
// INSERT OR IGNORE per §0027 AP6.
//
// Concurrency: same model as FormAll per §0045 — reads via
// WalkEvents (no writeMu) + writes via substrate.Append (acquires
// writeMu). Safe to run alongside the ingestion service's write
// path; the substrate serializes all writers per
// concurrency-pattern.md §Substrate-Writer Serialization.
func FormAutomationGroupAll(ctx context.Context, sub *substrate.Substrate, pattern AutomationGroupFormationPattern, now func() time.Time) (AutomationGroupReport, error) {
	if pattern == nil {
		return AutomationGroupReport{}, errors.New("hypothesis.FormAutomationGroupAll: pattern must not be nil")
	}
	if now == nil {
		now = time.Now
	}

	fctx := &automationGroupWalkerContext{}
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != declaredSessionMessageType {
			return nil
		}
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return fmt.Errorf("read declared session %x: %w", row.EventHash, err)
		}
		ds := &eventsv1.DeclaredSession{}
		if err := proto.Unmarshal(payload, ds); err != nil {
			return fmt.Errorf("unmarshal declared session %x: %w", row.EventHash, err)
		}
		fctx.sessions = append(fctx.sessions, SourceDeclaredSession{Hash: row.EventHash, Session: ds})
		return nil
	}); err != nil {
		return AutomationGroupReport{}, fmt.Errorf("hypothesis.FormAutomationGroupAll: collect declared sessions: %w", err)
	}

	rep := AutomationGroupReport{Examined: int64(len(fctx.sessions))}
	formationAt := now().UnixNano()
	formations := pattern.Form(fctx, formationAt)

	for _, ev := range formations {
		ev.PatternSignature = pattern.Signature()
		ev.PatternParameters = pattern.Parameters()

		payload, hash, err := canonical.MarshalAndHash(ev)
		if err != nil {
			return rep, fmt.Errorf("marshal formation: %w", err)
		}
		hex := canonical.HashHex(hash)

		_, lookupErr := sub.LookupRow(ctx, hash)
		alreadyPresent := lookupErr == nil
		if lookupErr != nil && !errors.Is(lookupErr, sql.ErrNoRows) {
			return rep, fmt.Errorf("lookup formation %s: %w", hex, lookupErr)
		}

		row := substrate.EventRow{
			EventHash:   hash,
			EventTime:   ev.GetFormationAt(),
			MessageType: string(ev.ProtoReflect().Descriptor().FullName()),
			PayloadRef:  hex[:2] + "/" + hex[2:],
			CommittedAt: now().UnixNano(),
		}
		if err := sub.Append(ctx, row, payload); err != nil {
			return rep, fmt.Errorf("append formation %s: %w", hex, err)
		}

		if alreadyPresent {
			rep.AlreadyFormed++
		} else {
			rep.NewlyFormed++
		}
	}

	return rep, nil
}
