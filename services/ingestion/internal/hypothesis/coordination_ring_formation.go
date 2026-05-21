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

// coordinationRingFormationMessageType is the substrate's
// MessageType discriminator for CoordinationRingFormation rows.
// Used by the within-subtype lifecycle operators (promotion,
// demotion, dissolution, merge, split — follow-on landings) to
// confirm a referenced formation belongs to this subtype before
// committing the operation. Per the §0045+§0056+§0063+§0070
// typed-subtype-landings discipline, the constant is package-
// private to the hypothesis package; the projection package
// declares its own equivalent at the read layer.
const coordinationRingFormationMessageType = "ghosttrace.events.v1.CoordinationRingFormation"

// CoordinationRingFormationContext is the Cat III formation
// context for the CoordinationRing subtype per §0070. Parallel to
// FormationContext (§0045) + AutomationGroupFormationContext
// (§0056) + CampaignHypothesisFormationContext (§0063) — each
// concrete subtype carries its own context interface per the
// typed-subtype-landings discipline.
//
// At inception, exposes the same Cat I surface as the other three
// subtypes (DeclaredSessions). Subtype distinction is in the
// FORMATION ALGORITHM (interaction/edge inference, neither
// flat-set-of-actors like BC/AG nor flat-set-of-events like CH)
// rather than the context surface.
type CoordinationRingFormationContext interface {
	DeclaredSessions() []SourceDeclaredSession
}

// CoordinationRingFormationPattern is the formation contract for
// the CoordinationRing subtype. Returns CoordinationRingFormation
// events; the return type is the hypothesis-identity-bearing
// structure, so it MUST be subtype-specific per §0056+§0063.
type CoordinationRingFormationPattern interface {
	Signature() string
	Parameters() string
	Form(fctx CoordinationRingFormationContext, formationAt int64) []*eventsv1.CoordinationRingFormation
}

// CoordinationRingReport is the per-FormCoordinationRingAll
// outcome.
type CoordinationRingReport struct {
	Examined      int64
	NewlyFormed   int64
	AlreadyFormed int64
}

// coordinationRingWalkerContext implements
// CoordinationRingFormationContext over a pre-collected slice of
// (DeclaredSession, content-hash) pairs.
type coordinationRingWalkerContext struct {
	sessions []SourceDeclaredSession
}

func (w *coordinationRingWalkerContext) DeclaredSessions() []SourceDeclaredSession {
	return w.sessions
}

// FormCoordinationRingAll walks every DeclaredSession in the
// substrate, applies pattern.Form to the collected context, and
// commits each resulting CoordinationRingFormation event via
// substrate.Append. Idempotency + concurrency model identical to
// §0045+§0056+§0063 formation entry points.
func FormCoordinationRingAll(ctx context.Context, sub *substrate.Substrate, pattern CoordinationRingFormationPattern, now func() time.Time) (CoordinationRingReport, error) {
	return FormCoordinationRingAllWithActor(ctx, sub, pattern, now, "")
}

// FormCoordinationRingAllWithActor is FormCoordinationRingAll with
// per-actor attribution per §0111 T4 form landing.
func FormCoordinationRingAllWithActor(ctx context.Context, sub *substrate.Substrate, pattern CoordinationRingFormationPattern, now func() time.Time, actor string) (CoordinationRingReport, error) {
	if pattern == nil {
		return CoordinationRingReport{}, errors.New("hypothesis.FormCoordinationRingAllWithActor: pattern must not be nil")
	}
	if now == nil {
		now = time.Now
	}

	fctx := &coordinationRingWalkerContext{}
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
		return CoordinationRingReport{}, fmt.Errorf("hypothesis.FormCoordinationRingAll: collect declared sessions: %w", err)
	}

	rep := CoordinationRingReport{Examined: int64(len(fctx.sessions))}
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

		committedAt := now().UnixNano()
		row := substrate.EventRow{
			EventHash:   hash,
			EventTime:   ev.GetFormationAt(),
			MessageType: string(ev.ProtoReflect().Descriptor().FullName()),
			PayloadRef:  hex[:2] + "/" + hex[2:],
			CommittedAt: committedAt,
		}

		if actor == "" {
			if err := sub.Append(ctx, row, payload); err != nil {
				return rep, fmt.Errorf("append formation %s: %w", hex, err)
			}
		} else {
			ingEv := &eventsv1.IngestionEvent{
				PrimaryEventHash: hash[:],
				ReceivedAt:       committedAt,
				IngestedAt:       committedAt,
				Channel:          "cli",
				ClientCommonName: actor,
			}
			ingPayload, ingHash, err := canonical.MarshalAndHash(ingEv)
			if err != nil {
				return rep, fmt.Errorf("marshal ingestion event: %w", err)
			}
			ingHex := canonical.HashHex(ingHash)
			ingRow := substrate.EventRow{
				EventHash:   ingHash,
				EventTime:   committedAt,
				MessageType: string(ingEv.ProtoReflect().Descriptor().FullName()),
				PayloadRef:  ingHex[:2] + "/" + ingHex[2:],
				CommittedAt: committedAt,
			}
			if err := sub.AppendPair(ctx, row, payload, ingRow, ingPayload); err != nil {
				return rep, fmt.Errorf("append pair formation %s + ingestion %s: %w", hex, ingHex, err)
			}
		}

		if alreadyPresent {
			rep.AlreadyFormed++
		} else {
			rep.NewlyFormed++
		}
	}

	return rep, nil
}
