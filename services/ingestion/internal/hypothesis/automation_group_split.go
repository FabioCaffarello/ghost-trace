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

// AutomationGroupSplitOptions configures a single within-subtype
// AutomationGroup split. Mirrors §0050's SplitOptions.
type AutomationGroupSplitOptions struct {
	// AntecedentFormationHash is the content-hash of the
	// AutomationGroupFormation being split.
	AntecedentFormationHash [32]byte

	// SuccessorFormationHashes is the set of separately-committed
	// AutomationGroupFormation hashes representing the successors.
	// Length MUST be ≥ 2; entries MUST be byte-distinct from each
	// other AND from AntecedentFormationHash.
	SuccessorFormationHashes [][32]byte

	// SplitAt is the Unix-nanoseconds time. Zero defaults to
	// now().UnixNano().
	SplitAt int64

	// Reason is an operator-supplied free-form note.
	Reason string

	// Actor per §0110 — non-empty triggers AppendPair.
	Actor string
}

// AutomationGroupSplitReport is the per-SplitAutomationGroup
// outcome.
type AutomationGroupSplitReport struct {
	// SplitEventHashHex is the content-hash (hex) of the recorded
	// AutomationGroupSplit event.
	SplitEventHashHex string

	// AlreadySplit is true when an identical split event was
	// already in the substrate.
	AlreadySplit bool

	// IngestionEventHashHex non-empty when Actor was supplied.
	IngestionEventHashHex string
}

// SplitAutomationGroup records an AutomationGroupSplit lifecycle
// event recognizing that the antecedent AutomationGroupFormation
// hypothesis contained multiple distinct phenomena and has been
// divided into the supplied successor AutomationGroupFormation
// hypotheses. Per Charter §2.5 BC5 the split event is a Cat I
// record committed via substrate.Append.
//
// Within-subtype only at this layer: antecedent and all successor
// hashes MUST resolve to AutomationGroupFormation rows.
//
// Errors:
//   - ErrSplitInsufficientSuccessors: fewer than 2 successors.
//   - ErrSplitSuccessorsNotDistinct: duplicate successors or
//     antecedent-overlap (sentinel shared with §0050).
//   - ErrTargetNotFound: antecedent OR one successor hash missing.
//   - ErrTargetWrongType: antecedent OR one successor is NOT an
//     AutomationGroupFormation.
func SplitAutomationGroup(ctx context.Context, sub *substrate.Substrate, opts AutomationGroupSplitOptions, now func() time.Time) (AutomationGroupSplitReport, error) {
	if len(opts.SuccessorFormationHashes) < 2 {
		return AutomationGroupSplitReport{}, fmt.Errorf("%w: got %d", ErrSplitInsufficientSuccessors, len(opts.SuccessorFormationHashes))
	}

	seen := make(map[[32]byte]struct{}, len(opts.SuccessorFormationHashes)+1)
	seen[opts.AntecedentFormationHash] = struct{}{}
	for _, succ := range opts.SuccessorFormationHashes {
		if _, dup := seen[succ]; dup {
			return AutomationGroupSplitReport{}, fmt.Errorf("%w: %x", ErrSplitSuccessorsNotDistinct, succ)
		}
		seen[succ] = struct{}{}
	}

	if now == nil {
		now = time.Now
	}

	row, err := sub.LookupRow(ctx, opts.AntecedentFormationHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AutomationGroupSplitReport{}, fmt.Errorf("%w: antecedent %x", ErrTargetNotFound, opts.AntecedentFormationHash)
		}
		return AutomationGroupSplitReport{}, fmt.Errorf("hypothesis.SplitAutomationGroup: lookup antecedent: %w", err)
	}
	if row.MessageType != automationGroupFormationMessageType {
		return AutomationGroupSplitReport{}, fmt.Errorf("%w: antecedent %x is %q", ErrTargetWrongType, opts.AntecedentFormationHash, row.MessageType)
	}
	for i, succ := range opts.SuccessorFormationHashes {
		srow, err := sub.LookupRow(ctx, succ)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return AutomationGroupSplitReport{}, fmt.Errorf("%w: successor[%d] %x", ErrTargetNotFound, i, succ)
			}
			return AutomationGroupSplitReport{}, fmt.Errorf("hypothesis.SplitAutomationGroup: lookup successor[%d]: %w", i, err)
		}
		if srow.MessageType != automationGroupFormationMessageType {
			return AutomationGroupSplitReport{}, fmt.Errorf("%w: successor[%d] %x is %q", ErrTargetWrongType, i, succ, srow.MessageType)
		}
	}

	splitAt := opts.SplitAt
	if splitAt == 0 {
		splitAt = now().UnixNano()
	}

	successorBytes := make([][]byte, len(opts.SuccessorFormationHashes))
	for i, h := range opts.SuccessorFormationHashes {
		successorBytes[i] = append([]byte(nil), h[:]...)
	}
	sort.Slice(successorBytes, func(i, j int) bool {
		return bytes.Compare(successorBytes[i], successorBytes[j]) < 0
	})

	ev := &eventsv1.AutomationGroupSplit{
		AntecedentFormationEventHash:  opts.AntecedentFormationHash[:],
		SuccessorFormationEventHashes: successorBytes,
		SplitAt:                       splitAt,
		Reason:                        opts.Reason,
	}
	payload, hash, err := canonical.MarshalAndHash(ev)
	if err != nil {
		return AutomationGroupSplitReport{}, fmt.Errorf("hypothesis.SplitAutomationGroup: marshal split: %w", err)
	}
	hex := canonical.HashHex(hash)

	_, lookupErr := sub.LookupRow(ctx, hash)
	alreadyPresent := lookupErr == nil
	if lookupErr != nil && !errors.Is(lookupErr, sql.ErrNoRows) {
		return AutomationGroupSplitReport{}, fmt.Errorf("hypothesis.SplitAutomationGroup: lookup split %s: %w", hex, lookupErr)
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
			return AutomationGroupSplitReport{}, fmt.Errorf("hypothesis.SplitAutomationGroup: append split %s: %w", hex, err)
		}
		return AutomationGroupSplitReport{
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
		return AutomationGroupSplitReport{}, fmt.Errorf("hypothesis.SplitAutomationGroup: marshal ingestion event: %w", err)
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
		return AutomationGroupSplitReport{}, fmt.Errorf("hypothesis.SplitAutomationGroup: append pair (split %s, ingestion %s): %w", hex, ingHex, err)
	}

	return AutomationGroupSplitReport{
		SplitEventHashHex:     hex,
		AlreadySplit:          alreadyPresent,
		IngestionEventHashHex: ingHex,
	}, nil
}
