package hypothesis

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// ErrSplitInsufficientSuccessors is returned by Split when the
// caller supplies fewer than 2 successor formation hashes. Per
// lifecycle-semantics.md line 29 split divides a hypothesis into
// MULTIPLE successor hypotheses; a one-successor "split" is not a
// valid §2.5 lifecycle move (and is structurally indistinguishable
// from a no-op rename, which the substrate's content-addressing
// already prevents).
var ErrSplitInsufficientSuccessors = errors.New("hypothesis.Split: successors must contain at least 2 distinct formations")

// ErrSplitSuccessorsNotDistinct is returned by Split when the
// successor set contains duplicates or includes the antecedent
// itself. A split MUST produce strictly distinct successors, none
// of which is identity-equal to the antecedent — otherwise the
// operation is not a valid partition of the antecedent's underlying
// phenomena.
var ErrSplitSuccessorsNotDistinct = errors.New("hypothesis.Split: successors must be byte-distinct from each other and from the antecedent")

// SplitOptions configures a single within-subtype split. The
// (AntecedentFormation, SuccessorFormations, SplitAt, Reason) tuple
// uniquely identifies a split event's content-hash — re-running
// with identical values is idempotent.
//
// Within-subtype scope: the antecedent and all successor formation
// hashes MUST resolve to BehavioralClusterFormation rows.
// Cross-subtype split per entity-model.md §Cross-subtype operations
// remains deferred to lifecycle-semantics.md post-Q4 redaction.
type SplitOptions struct {
	// AntecedentFormationHash is the content-hash of the
	// BehavioralClusterFormation being split. Per §0045 the hash IS
	// the hypothesis identity.
	AntecedentFormationHash [32]byte

	// SuccessorFormationHashes is the set of separately-committed
	// BehavioralClusterFormation hashes representing the successor
	// hypotheses. Length MUST be ≥ 2; all entries MUST be
	// byte-distinct from each other AND from AntecedentFormationHash.
	// Split sorts ascending before recording so the substrate
	// identity is invariant under the caller's enumeration order
	// (successors form a SET, not a sequence).
	SuccessorFormationHashes [][32]byte

	// SplitAt is the Unix-nanoseconds time at which the split
	// recognition was recorded. Zero defaults to now().UnixNano().
	// Explicit non-zero values permitted for forensic replay +
	// deterministic test recording.
	SplitAt int64

	// Reason is an operator-supplied free-form note explaining the
	// recognition that the antecedent hypothesis contained multiple
	// distinct phenomena. Optional but strongly recommended.
	Reason string

	// Actor per §0110 — non-empty triggers AppendPair.
	Actor string
}

// SplitReport is the per-Split outcome.
type SplitReport struct {
	// SplitEventHashHex is the content-hash (hex) of the recorded
	// BehavioralClusterSplit event.
	SplitEventHashHex string

	// AlreadySplit is true when an identical split event was
	// already in the substrate (content-hash collision). False when
	// this Split invocation committed a new row.
	AlreadySplit bool

	// IngestionEventHashHex non-empty when Actor was supplied.
	IngestionEventHashHex string
}

// Split records a BehavioralClusterSplit lifecycle event recognizing
// that the antecedent BehavioralClusterFormation hypothesis contained
// multiple distinct phenomena and has been divided into the supplied
// successor BehavioralClusterFormation hypotheses. Per Charter §2.5
// BC5 the split event is a Category I record committed via
// substrate.Append (acquires writeMu per concurrency-pattern.md
// §Substrate-Writer Serialization).
//
// Within-subtype only at this layer: the antecedent and all
// successor hashes MUST resolve to BehavioralClusterFormation rows.
// Cross-subtype split per entity-model.md §Cross-subtype operations
// is deferred to lifecycle-semantics.md post-Q4 redaction.
//
// Errors:
//   - ErrSplitInsufficientSuccessors: fewer than 2 successors supplied.
//   - ErrSplitSuccessorsNotDistinct: the successor set contains
//     duplicates OR includes the antecedent.
//   - ErrTargetNotFound: the antecedent OR one of the successor
//     hashes does not resolve to any substrate row.
//   - ErrTargetWrongType: the antecedent OR one of the successors
//     resolves to a row whose message_type is NOT
//     BehavioralClusterFormation.
func Split(ctx context.Context, sub *substrate.Substrate, opts SplitOptions, now func() time.Time) (SplitReport, error) {
	if len(opts.SuccessorFormationHashes) < 2 {
		return SplitReport{}, fmt.Errorf("%w: got %d", ErrSplitInsufficientSuccessors, len(opts.SuccessorFormationHashes))
	}

	// Distinctness: successors mutually distinct + none equal to antecedent.
	seen := make(map[[32]byte]struct{}, len(opts.SuccessorFormationHashes)+1)
	seen[opts.AntecedentFormationHash] = struct{}{}
	for _, succ := range opts.SuccessorFormationHashes {
		if _, dup := seen[succ]; dup {
			return SplitReport{}, fmt.Errorf("%w: %x", ErrSplitSuccessorsNotDistinct, succ)
		}
		seen[succ] = struct{}{}
	}

	if now == nil {
		now = time.Now
	}

	// Validate antecedent.
	{
		row, err := sub.LookupRow(ctx, opts.AntecedentFormationHash)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return SplitReport{}, fmt.Errorf("%w: antecedent %x", ErrTargetNotFound, opts.AntecedentFormationHash)
			}
			return SplitReport{}, fmt.Errorf("hypothesis.Split: lookup antecedent: %w", err)
		}
		if row.MessageType != behavioralClusterFormationMessageType {
			return SplitReport{}, fmt.Errorf("%w: antecedent %x is %q", ErrTargetWrongType, opts.AntecedentFormationHash, row.MessageType)
		}
	}
	// Validate every successor.
	for i, succ := range opts.SuccessorFormationHashes {
		row, err := sub.LookupRow(ctx, succ)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return SplitReport{}, fmt.Errorf("%w: successor[%d] %x", ErrTargetNotFound, i, succ)
			}
			return SplitReport{}, fmt.Errorf("hypothesis.Split: lookup successor[%d]: %w", i, err)
		}
		if row.MessageType != behavioralClusterFormationMessageType {
			return SplitReport{}, fmt.Errorf("%w: successor[%d] %x is %q", ErrTargetWrongType, i, succ, row.MessageType)
		}
	}

	splitAt := opts.SplitAt
	if splitAt == 0 {
		splitAt = now().UnixNano()
	}

	// Successors form a SET; sort ascending so the split event's
	// content-hash is invariant under caller enumeration order.
	successorBytes := make([][]byte, len(opts.SuccessorFormationHashes))
	for i, h := range opts.SuccessorFormationHashes {
		successorBytes[i] = append([]byte(nil), h[:]...)
	}
	sort.Slice(successorBytes, func(i, j int) bool {
		return bytes.Compare(successorBytes[i], successorBytes[j]) < 0
	})

	ev := &eventsv1.BehavioralClusterSplit{
		AntecedentFormationEventHash:   opts.AntecedentFormationHash[:],
		SuccessorFormationEventHashes:  successorBytes,
		SplitAt:                        splitAt,
		Reason:                         opts.Reason,
	}
	payload, hash, err := canonical.MarshalAndHash(ev)
	if err != nil {
		return SplitReport{}, fmt.Errorf("hypothesis.Split: marshal split: %w", err)
	}
	hex := canonical.HashHex(hash)

	_, lookupErr := sub.LookupRow(ctx, hash)
	alreadyPresent := lookupErr == nil
	if lookupErr != nil && !errors.Is(lookupErr, sql.ErrNoRows) {
		return SplitReport{}, fmt.Errorf("hypothesis.Split: lookup split %s: %w", hex, lookupErr)
	}

	committedAt := now().UnixNano()
	splitRow := substrate.EventRow{
		EventHash:   hash,
		EventTime:   splitAt,
		MessageType: string(ev.ProtoReflect().Descriptor().FullName()),
		PayloadRef:  hex[:2] + "/" + hex[2:],
		CommittedAt: committedAt,
	}

	if opts.Actor == "" {
		if err := sub.Append(ctx, splitRow, payload); err != nil {
			return SplitReport{}, fmt.Errorf("hypothesis.Split: append split %s: %w", hex, err)
		}
		return SplitReport{
			SplitEventHashHex: hex,
			AlreadySplit:      alreadyPresent,
		}, nil
	}

	ingEv := &eventsv1.IngestionEvent{
		PrimaryEventHash: hash[:],
		ReceivedAt:       committedAt,
		IngestedAt:       committedAt,
		Channel:          "cli",
		ClientCommonName: opts.Actor,
	}
	ingPayload, ingHash, err := canonical.MarshalAndHash(ingEv)
	if err != nil {
		return SplitReport{}, fmt.Errorf("hypothesis.Split: marshal ingestion event: %w", err)
	}
	ingHex := canonical.HashHex(ingHash)
	ingRow := substrate.EventRow{
		EventHash:   ingHash,
		EventTime:   committedAt,
		MessageType: string(ingEv.ProtoReflect().Descriptor().FullName()),
		PayloadRef:  ingHex[:2] + "/" + ingHex[2:],
		CommittedAt: committedAt,
	}
	if err := sub.AppendPair(ctx, splitRow, payload, ingRow, ingPayload); err != nil {
		return SplitReport{}, fmt.Errorf("hypothesis.Split: append pair (split %s, ingestion %s): %w", hex, ingHex, err)
	}

	return SplitReport{
		SplitEventHashHex:     hex,
		AlreadySplit:          alreadyPresent,
		IngestionEventHashHex: ingHex,
	}, nil
}
