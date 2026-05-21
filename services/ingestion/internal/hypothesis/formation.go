// Package hypothesis implements Category III hypothesis lifecycle per
// Charter §2.5 (Hypothesis Lifecycle Explicitness, frozen v0.3) +
// entity-model.md §Category III + decision-log §0010 (Q2 subtype
// resolution).
//
// This package implements the FIRST lifecycle operation — formation —
// for the FIRST Category III subtype — BehavioralCluster (per
// entity-model.md line 65). The other five lifecycle operations
// (merge, split, promotion, demotion, dissolution per Charter §2.5
// + decision-log §0011 + §0013) and the other three subtypes
// (CoordinationRing, CampaignHypothesis, AutomationGroup per §0010)
// are follow-on landings using the same FormationPattern + walker
// pathway established here.
//
// Constitutional shape per Charter §2.5 Boundary Condition 3: the
// substrate stores ONLY lifecycle events (BehavioralClusterFormation,
// future BehavioralClusterPromotion, etc.), NEVER a "current state"
// BehavioralCluster record. The hypothesis's current state is a
// projection over the lifecycle event chain.
//
// Constitutional shape per Charter §2.5 Boundary Condition 5:
// lifecycle events ARE Category I records under §2.1 immutability.
// FormAll commits via substrate.Append (acquires writeMu per
// concurrency-pattern.md §Substrate-Writer Serialization).
//
// Determinism contract: the formation event recording is
// deterministic with respect to (substrate-state, pattern_signature,
// pattern_parameters) — re-running with identical inputs produces
// content-hash-identical formation events (the substrate's INSERT OR
// IGNORE rejects duplicates per §0027 AP6). The INFERENCE that
// triggered formation is non-deterministic per Charter §2.2 Category
// III requirement; non-determinism lives in the choice of
// FormationPattern, not in the recording mechanism.
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

// declaredSessionMessageType is the substrate's message_type
// discriminator for DeclaredSession rows. The walker pre-collects
// these to provide to the FormationPattern.
const declaredSessionMessageType = "ghosttrace.events.v1.DeclaredSession"

// SourceDeclaredSession pairs a substrate-resident DeclaredSession
// with its content-hash. The hash is the observational-provenance
// reference per Charter §2.3 that any resulting formation event
// records in its source_event_hashes field.
type SourceDeclaredSession struct {
	Hash    [32]byte
	Session *eventsv1.DeclaredSession
}

// FormationContext provides typed access to the substrate's Category I
// observations a FormationPattern may consult while evaluating whether
// formation conditions are met. The walker (FormAll) implements this
// interface by pre-collecting the relevant Cat I rows; patterns query
// it without touching the substrate directly.
//
// New typed accessors are added here as new patterns require new
// Category I observation types — the same incremental-extension
// procedure that §0042 established for adding Cat I types and §0044
// established for adding Cat II derivation context.
type FormationContext interface {
	// DeclaredSessions returns every DeclaredSession in the substrate,
	// each paired with its content-hash. Order is substrate-walk
	// order (commit-ascending per substrate.WalkEvents). The returned
	// slice is shared with the walker and MUST NOT be mutated.
	DeclaredSessions() []SourceDeclaredSession
}

// FormationPattern is the Category III formation contract — the
// pattern's Form method returns a slice of BehavioralClusterFormation
// events, one per cluster the pattern infers from the FormationContext.
// Implementations MUST be deterministic with respect to (FormationContext
// state, Parameters) so re-running with identical inputs produces
// content-hash-identical events.
//
// Adding a new formation pattern:
//  1. Implement FormationPattern (Signature, Parameters, Form).
//  2. Register it in cmd/form-hypothesis/main.go's resolvePattern.
//  3. Author canonical-corpus entries if the pattern produces a new
//     surface beyond what existing entries cover.
type FormationPattern interface {
	// Signature is the stable identifier of this formation pattern
	// (e.g. "session-descriptor-shared-v1"). Part of the produced
	// BehavioralClusterFormation's identity per entity-model.md line 44.
	Signature() string

	// Parameters returns the canonical-form string encoding of the
	// pattern's parameters. Canonical form: lowercase key=value pairs
	// sorted by key, separated by semicolons.
	Parameters() string

	// Form evaluates fctx and returns zero or more
	// BehavioralClusterFormation events representing the clusters the
	// pattern inferred. The pattern_signature + pattern_parameters
	// fields are set by the caller (FormAll); the implementation
	// populates actor_refs (sorted ascending), formation_at,
	// confidence, and source_event_hashes (sorted ascending).
	Form(fctx FormationContext, formationAt int64) []*eventsv1.BehavioralClusterFormation
}

// Report is the per-FormAll outcome. Examined counts every
// DeclaredSession enumerated by the walker; NewlyFormed counts the
// BehavioralClusterFormation events committed in this run;
// AlreadyFormed counts the formation events whose content-hash
// collided with an existing substrate row (per content-addressed
// idempotency).
type Report struct {
	Examined      int64
	NewlyFormed   int64
	AlreadyFormed int64
}

// walkerContext implements FormationContext over a pre-collected
// slice of (DeclaredSession, content-hash) pairs.
type walkerContext struct {
	sessions []SourceDeclaredSession
}

// DeclaredSessions implements FormationContext.
func (w *walkerContext) DeclaredSessions() []SourceDeclaredSession {
	return w.sessions
}

// FormAll walks every DeclaredSession in the substrate, applies
// pattern.Form to the collected context, and commits each resulting
// BehavioralClusterFormation event via substrate.Append. The commit
// is idempotent: re-running with the same pattern + substrate state
// produces zero new rows (NewlyFormed=0, AlreadyFormed equals the
// number of formation events the pattern returned) because content-
// addressed immutability rejects duplicate content via INSERT OR
// IGNORE per §0027 AP6.
//
// Concurrency: FormAll reads via WalkEvents (no writeMu) + writes via
// substrate.Append (acquires writeMu). Safe to run alongside the
// ingestion service's write path; the substrate serializes all
// writers per concurrency-pattern.md §Substrate-Writer Serialization.
func FormAll(ctx context.Context, sub *substrate.Substrate, pattern FormationPattern, now func() time.Time) (Report, error) {
	return FormAllWithActor(ctx, sub, pattern, now, "")
}

// FormAllWithActor is FormAll with per-actor attribution. When actor
// is non-empty, each formed BehavioralClusterFormation event commits
// paired with an IngestionEvent via substrate.AppendPair (channel=
// "cli", client_common_name=actor); per-actor attribution lands on
// the IngestionEvent. Empty actor preserves the FormAll single-Append
// path (backward compatible).
//
// Used by HTTP T4 form endpoint per §0111 (cross-tier per-actor-
// attribution requirement for HTTP-channel writes per §0094); CLI
// continues using FormAll (no actor; §0033 local-shell-trust).
func FormAllWithActor(ctx context.Context, sub *substrate.Substrate, pattern FormationPattern, now func() time.Time, actor string) (Report, error) {
	if pattern == nil {
		return Report{}, errors.New("hypothesis.FormAllWithActor: pattern must not be nil")
	}
	if now == nil {
		now = time.Now
	}

	fctx, err := CollectFormationContextAt(ctx, sub, 0)
	if err != nil {
		return Report{}, fmt.Errorf("hypothesis.FormAllWithActor: %w", err)
	}

	rep := Report{Examined: int64(len(fctx.DeclaredSessions()))}
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

// CollectFormationContextAt walks the substrate to pre-collect every
// DeclaredSession whose substrate row's committed_at is ≤
// maxCommittedAt. Returns a FormationContext suitable for passing to
// FormationPattern.Form.
//
// maxCommittedAt == 0 disables the bound (collect ALL DeclaredSessions
// regardless of commit time) — matches the unrestricted behavior
// FormAll uses for normal formation.
//
// Bounded collection is the substrate-time-filter primitive Phase 3
// reconstructive replay consumes per decision-log §0086 +
// docs/architecture/replay-model.md L25-28: "given the substrate as
// it existed at time T, what would the formation pattern have
// concluded?". The bound by committed_at (NOT event_time) matches the
// OMQ #3 resolution (§0021 substrate-time generation) — substrate
// authority is at substrate time.
//
// Cost: one substrate walk + one blob-read per DeclaredSession.
// Reused across FormAll + replay paths per §0083 single-source-of-
// truth pattern.
func CollectFormationContextAt(ctx context.Context, sub *substrate.Substrate, maxCommittedAt int64) (FormationContext, error) {
	fctx := &walkerContext{}
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != declaredSessionMessageType {
			return nil
		}
		if maxCommittedAt != 0 && row.CommittedAt > maxCommittedAt {
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
		return nil, fmt.Errorf("CollectFormationContextAt: %w", err)
	}
	return fctx, nil
}
