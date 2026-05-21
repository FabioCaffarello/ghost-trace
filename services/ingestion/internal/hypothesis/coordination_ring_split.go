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

// CoordinationRingSplitOptions configures a single within-subtype
// CoordinationRing split. Mirrors §0050+§0061+§0068.
type CoordinationRingSplitOptions struct {
	AntecedentFormationHash  [32]byte
	SuccessorFormationHashes [][32]byte
	SplitAt                  int64
	Reason                   string
}

// CoordinationRingSplitReport is the per-SplitCoordinationRing
// outcome.
type CoordinationRingSplitReport struct {
	SplitEventHashHex string
	AlreadySplit      bool
}

// SplitCoordinationRing records a CoordinationRingSplit lifecycle
// event recognizing that the antecedent CoordinationRingFormation
// contained multiple distinct coordinated-action phenomena and has
// been divided into the supplied successor CoordinationRingFormation
// hypotheses. Per Charter §2.5 BC5 the split event is a Cat I
// record committed via substrate.Append.
//
// Within-subtype only. Cross-subtype split per §Cross-subtype
// operations remains deferred.
//
// Errors:
//   - ErrSplitInsufficientSuccessors: fewer than 2 successors.
//   - ErrSplitSuccessorsNotDistinct: duplicate successors or
//     antecedent-overlap.
//   - ErrTargetNotFound: antecedent or successor not in substrate.
//   - ErrTargetWrongType: target NOT a CoordinationRingFormation.
func SplitCoordinationRing(ctx context.Context, sub *substrate.Substrate, opts CoordinationRingSplitOptions, now func() time.Time) (CoordinationRingSplitReport, error) {
	if len(opts.SuccessorFormationHashes) < 2 {
		return CoordinationRingSplitReport{}, fmt.Errorf("%w: got %d", ErrSplitInsufficientSuccessors, len(opts.SuccessorFormationHashes))
	}

	seen := make(map[[32]byte]struct{}, len(opts.SuccessorFormationHashes)+1)
	seen[opts.AntecedentFormationHash] = struct{}{}
	for _, succ := range opts.SuccessorFormationHashes {
		if _, dup := seen[succ]; dup {
			return CoordinationRingSplitReport{}, fmt.Errorf("%w: %x", ErrSplitSuccessorsNotDistinct, succ)
		}
		seen[succ] = struct{}{}
	}

	if now == nil {
		now = time.Now
	}

	row, err := sub.LookupRow(ctx, opts.AntecedentFormationHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CoordinationRingSplitReport{}, fmt.Errorf("%w: antecedent %x", ErrTargetNotFound, opts.AntecedentFormationHash)
		}
		return CoordinationRingSplitReport{}, fmt.Errorf("hypothesis.SplitCoordinationRing: lookup antecedent: %w", err)
	}
	if row.MessageType != coordinationRingFormationMessageType {
		return CoordinationRingSplitReport{}, fmt.Errorf("%w: antecedent %x is %q", ErrTargetWrongType, opts.AntecedentFormationHash, row.MessageType)
	}
	for i, succ := range opts.SuccessorFormationHashes {
		srow, err := sub.LookupRow(ctx, succ)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return CoordinationRingSplitReport{}, fmt.Errorf("%w: successor[%d] %x", ErrTargetNotFound, i, succ)
			}
			return CoordinationRingSplitReport{}, fmt.Errorf("hypothesis.SplitCoordinationRing: lookup successor[%d]: %w", i, err)
		}
		if srow.MessageType != coordinationRingFormationMessageType {
			return CoordinationRingSplitReport{}, fmt.Errorf("%w: successor[%d] %x is %q", ErrTargetWrongType, i, succ, srow.MessageType)
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

	ev := &eventsv1.CoordinationRingSplit{
		AntecedentFormationEventHash:  opts.AntecedentFormationHash[:],
		SuccessorFormationEventHashes: successorBytes,
		SplitAt:                       splitAt,
		Reason:                        opts.Reason,
	}
	payload, hash, err := canonical.MarshalAndHash(ev)
	if err != nil {
		return CoordinationRingSplitReport{}, fmt.Errorf("hypothesis.SplitCoordinationRing: marshal split: %w", err)
	}
	hex := canonical.HashHex(hash)

	_, lookupErr := sub.LookupRow(ctx, hash)
	alreadyPresent := lookupErr == nil
	if lookupErr != nil && !errors.Is(lookupErr, sql.ErrNoRows) {
		return CoordinationRingSplitReport{}, fmt.Errorf("hypothesis.SplitCoordinationRing: lookup split %s: %w", hex, lookupErr)
	}

	splitRow := substrate.EventRow{
		EventHash:   hash,
		EventTime:   splitAt,
		MessageType: string(ev.ProtoReflect().Descriptor().FullName()),
		PayloadRef:  hex[:2] + "/" + hex[2:],
		CommittedAt: now().UnixNano(),
	}
	if err := sub.Append(ctx, splitRow, payload); err != nil {
		return CoordinationRingSplitReport{}, fmt.Errorf("hypothesis.SplitCoordinationRing: append split %s: %w", hex, err)
	}

	return CoordinationRingSplitReport{
		SplitEventHashHex: hex,
		AlreadySplit:      alreadyPresent,
	}, nil
}
