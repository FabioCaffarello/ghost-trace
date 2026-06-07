package hypothesis

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	commonv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/common/v1"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/signatures"
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

// AutomationGroupFormationFromCandidateOptions carries the operator-
// elected commit parameters for AutomationGroupFormationFromCandidate.
// Per §0213: the bridge between F3 candidate envelope (signatures.
// FormationCandidate) and committed AutomationGroupFormation event.
// Defaults reflect §0011 inception-phase discipline (Confidence=0.0;
// EI=1/1); operator override via these options.
type AutomationGroupFormationFromCandidateOptions struct {
	// PatternParameters is the formation event's pattern_parameters
	// field. Defaults to empty string when zero-value; operator can
	// override via CLI flag for non-default signature parameter
	// captures.
	PatternParameters string

	// FormationAt is the formation event's formation_at field (Unix
	// nanoseconds). When zero, AutomationGroupFormationFromCandidate
	// substitutes now().UnixNano() — same pattern as
	// FormAutomationGroupAll's now() use.
	FormationAt int64

	// Confidence is the formation event's confidence field. Default
	// 0.0 per §0213 inception discipline (operator-overridable via
	// CLI). Departs from §0157 helper which used candidate.
	// ConfidenceHint; §0213 cravou 0.0 explicitly to preserve §0011
	// inception phase + register the §0214 Layer B gating empirical
	// prediction.
	Confidence float32

	// EvidentialIndependence is the formation event's
	// evidential_independence field. Required at marshalling
	// boundary per §0140 canonical-package paired-dimension
	// commitment check. Default {1, 1} per §0157 helper precedent
	// (full operator-asserted independence at inception).
	EvidentialIndependence *commonv1.EvidentialIndependence
}

// AutomationGroupFormationFromCandidate commits a single
// AutomationGroupFormation event derived from an F3 signature
// candidate (signatures.FormationCandidate). Returns the formation's
// content-hash + a flag indicating whether the formation already
// existed in the substrate (idempotency via content-addressed
// immutability per §0027 AP6).
//
// Per decision-log §0213: this function is the operator-tier lift
// of the §0157 test helper (commitFormationFromCandidate). The
// pre-§0213 pattern was: F3 signatures emit candidates;
// integration tests committed formations from candidates via test-
// only helpers; operator workflow had no path from emission to
// commit. §0213 closes the operational gap by lifting the helper
// shape to a package-public function consumable by an operator-
// facing CLI (cmd/form-automation-group-from-candidate).
//
// The function intentionally does NOT walk the substrate (unlike
// FormAutomationGroupAll which collects DeclaredSessions). The
// candidate carries everything required for the formation event;
// the substrate write is direct.
//
// Per §2.4 + §2.6 + §0140 marshalling boundary:
//   - source_event_hashes = candidate.SourceHashes verbatim
//     (per §2.3 chain; the candidate already threaded Cat I + Cat II
//     hashes per the §0169 + §0157 patterns).
//   - direct_influenced_by = nil (inception-phase; no prior
//     hypothesis influencing this formation).
//   - closure_hashes = nil (per §0157 precedent; not applicable
//     at formation stage).
//   - confidence = opts.Confidence (default 0.0 per §0213).
//   - evidential_independence = opts.EvidentialIndependence
//     (default {1, 1} per §0157 precedent; REQUIRED at
//     marshalling boundary per §0140 paired-dimension check).
//
// Cross-subtype guard: returns an error if
// candidate.HypothesisSubtype != HypothesisSubtypeAutomationGroup.
// Future bridge CLIs for other subtypes (BehavioralCluster,
// CampaignHypothesis, CoordinationRing) will have their own
// FromCandidate functions per the §0045 typed-subtype-specific
// commitment.
func AutomationGroupFormationFromCandidate(
	ctx context.Context,
	sub *substrate.Substrate,
	candidate *signatures.FormationCandidate,
	opts AutomationGroupFormationFromCandidateOptions,
	now func() time.Time,
) (hash [32]byte, alreadyPresent bool, err error) {
	if candidate == nil {
		return [32]byte{}, false, errors.New("hypothesis.AutomationGroupFormationFromCandidate: candidate must not be nil")
	}
	if candidate.HypothesisSubtype != signatures.HypothesisSubtypeAutomationGroup {
		return [32]byte{}, false, fmt.Errorf("hypothesis.AutomationGroupFormationFromCandidate: candidate.HypothesisSubtype = %v; want AutomationGroup", candidate.HypothesisSubtype)
	}
	if now == nil {
		now = time.Now
	}

	formationAt := opts.FormationAt
	if formationAt == 0 {
		formationAt = now().UnixNano()
	}

	ei := opts.EvidentialIndependence
	if ei == nil {
		ei = &commonv1.EvidentialIndependence{Numerator: 1, Denominator: 1}
	}

	formation := &eventsv1.AutomationGroupFormation{
		PatternSignature:       candidate.SignatureName,
		PatternParameters:      opts.PatternParameters,
		ActorRefs:              candidate.ActorRefs,
		FormationAt:            formationAt,
		Confidence:             opts.Confidence,
		SourceEventHashes:      candidate.SourceHashes,
		EvidentialIndependence: ei,
	}

	payload, formationHash, err := canonical.MarshalAndHash(formation)
	if err != nil {
		return [32]byte{}, false, fmt.Errorf("marshal formation: %w", err)
	}
	hex := canonical.HashHex(formationHash)

	_, lookupErr := sub.LookupRow(ctx, formationHash)
	alreadyPresent = lookupErr == nil
	if lookupErr != nil && !errors.Is(lookupErr, sql.ErrNoRows) {
		return [32]byte{}, false, fmt.Errorf("lookup formation %s: %w", hex, lookupErr)
	}

	row := substrate.EventRow{
		EventHash:   formationHash,
		EventTime:   formationAt,
		MessageType: string(formation.ProtoReflect().Descriptor().FullName()),
		PayloadRef:  hex[:2] + "/" + hex[2:],
		CommittedAt: now().UnixNano(),
	}
	if err := sub.Append(ctx, row, payload); err != nil {
		return [32]byte{}, false, fmt.Errorf("append formation %s: %w", hex, err)
	}

	return formationHash, alreadyPresent, nil
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
	return FormAutomationGroupAllWithActor(ctx, sub, pattern, now, "")
}

// FormAutomationGroupAllWithActor is FormAutomationGroupAll with
// per-actor attribution. Mirrors FormAllWithActor per §0111 T4 form
// landing.
func FormAutomationGroupAllWithActor(ctx context.Context, sub *substrate.Substrate, pattern AutomationGroupFormationPattern, now func() time.Time, actor string) (AutomationGroupReport, error) {
	if pattern == nil {
		return AutomationGroupReport{}, errors.New("hypothesis.FormAutomationGroupAllWithActor: pattern must not be nil")
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
