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

// CampaignHypothesisFormationContext is the Cat III formation
// context for the CampaignHypothesis subtype per §0063. Parallel
// to FormationContext (§0045) + AutomationGroupFormationContext
// (§0056) — each concrete subtype carries its own context
// interface per the typed-subtype-landings discipline.
//
// At inception, exposes the same Cat I surface as the other two
// subtypes (DeclaredSessions). Subtype distinction is in the
// FORMATION ALGORITHM (event-centric inference, not actor-centric)
// rather than the context surface.
type CampaignHypothesisFormationContext interface {
	DeclaredSessions() []SourceDeclaredSession
}

// CampaignHypothesisFormationPattern is the formation contract
// for the CampaignHypothesis subtype. Returns
// CampaignHypothesisFormation events; the return type is the
// hypothesis-identity-bearing structure, so it MUST be subtype-
// specific per §0056.
type CampaignHypothesisFormationPattern interface {
	Signature() string
	Parameters() string
	Form(fctx CampaignHypothesisFormationContext, formationAt int64) []*eventsv1.CampaignHypothesisFormation
}

// CampaignHypothesisReport is the per-FormCampaignHypothesisAll
// outcome.
type CampaignHypothesisReport struct {
	Examined      int64
	NewlyFormed   int64
	AlreadyFormed int64
}

// campaignHypothesisWalkerContext implements
// CampaignHypothesisFormationContext over a pre-collected slice of
// (DeclaredSession, content-hash) pairs.
type campaignHypothesisWalkerContext struct {
	sessions []SourceDeclaredSession
}

func (w *campaignHypothesisWalkerContext) DeclaredSessions() []SourceDeclaredSession {
	return w.sessions
}

// FormCampaignHypothesisAll walks every DeclaredSession in the
// substrate, applies pattern.Form to the collected context, and
// commits each resulting CampaignHypothesisFormation event via
// substrate.Append. Idempotency + concurrency model identical to
// §0045's FormAll and §0056's FormAutomationGroupAll.
func FormCampaignHypothesisAll(ctx context.Context, sub *substrate.Substrate, pattern CampaignHypothesisFormationPattern, now func() time.Time) (CampaignHypothesisReport, error) {
	return FormCampaignHypothesisAllWithActor(ctx, sub, pattern, now, "")
}

// FormCampaignHypothesisAllWithActor is FormCampaignHypothesisAll with
// per-actor attribution per §0111 T4 form landing.
func FormCampaignHypothesisAllWithActor(ctx context.Context, sub *substrate.Substrate, pattern CampaignHypothesisFormationPattern, now func() time.Time, actor string) (CampaignHypothesisReport, error) {
	if pattern == nil {
		return CampaignHypothesisReport{}, errors.New("hypothesis.FormCampaignHypothesisAllWithActor: pattern must not be nil")
	}
	if now == nil {
		now = time.Now
	}

	fctx := &campaignHypothesisWalkerContext{}
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
		return CampaignHypothesisReport{}, fmt.Errorf("hypothesis.FormCampaignHypothesisAll: collect declared sessions: %w", err)
	}

	rep := CampaignHypothesisReport{Examined: int64(len(fctx.sessions))}
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
