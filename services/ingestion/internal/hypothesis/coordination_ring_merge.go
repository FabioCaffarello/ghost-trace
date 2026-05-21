package hypothesis

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// CoordinationRingMergeOptions configures a single within-subtype
// CoordinationRing merge. Mirrors §0049+§0060+§0067 for the
// fourth-subtype landing.
type CoordinationRingMergeOptions struct {
	AntecedentAFormationHash [32]byte
	AntecedentBFormationHash [32]byte
	ProducedFormationHash    [32]byte
	MergedAt                 int64
	Reason                   string

	// Actor per §0109 — non-empty triggers AppendPair.
	Actor string
}

// CoordinationRingMergeReport is the per-MergeCoordinationRing
// outcome.
type CoordinationRingMergeReport struct {
	MergeEventHashHex string
	AlreadyMerged     bool

	// IngestionEventHashHex non-empty when Actor was supplied.
	IngestionEventHashHex string
}

// MergeCoordinationRing records a CoordinationRingMerge lifecycle
// event recognizing that the two antecedent CoordinationRingFormation
// hypotheses describe the same underlying coordinated-action
// phenomenon, with the produced hypothesis identified by a third
// (separately-committed) CoordinationRingFormation. Per Charter §2.5
// BC5 the merge event is a Cat I record committed via
// substrate.Append.
//
// Within-subtype only at this layer. The merge-relation symmetry
// constraint (`ErrMergeAntecedentsIdentical`) applies per §0049.
// Ascending-sort normalization preserves byte-equality under caller
// argument order.
//
// Errors:
//   - ErrMergeAntecedentsIdentical: identical antecedents.
//   - ErrTargetNotFound: missing target row.
//   - ErrTargetWrongType: non-CoordinationRingFormation row.
func MergeCoordinationRing(ctx context.Context, sub *substrate.Substrate, opts CoordinationRingMergeOptions, now func() time.Time) (CoordinationRingMergeReport, error) {
	if opts.AntecedentAFormationHash == opts.AntecedentBFormationHash {
		return CoordinationRingMergeReport{}, fmt.Errorf("%w: %x", ErrMergeAntecedentsIdentical, opts.AntecedentAFormationHash)
	}
	if now == nil {
		now = time.Now
	}

	for _, target := range []struct {
		label string
		hash  [32]byte
	}{
		{"antecedent A", opts.AntecedentAFormationHash},
		{"antecedent B", opts.AntecedentBFormationHash},
		{"produced", opts.ProducedFormationHash},
	} {
		row, err := sub.LookupRow(ctx, target.hash)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return CoordinationRingMergeReport{}, fmt.Errorf("%w: %s %x", ErrTargetNotFound, target.label, target.hash)
			}
			return CoordinationRingMergeReport{}, fmt.Errorf("hypothesis.MergeCoordinationRing: lookup %s: %w", target.label, err)
		}
		if row.MessageType != coordinationRingFormationMessageType {
			return CoordinationRingMergeReport{}, fmt.Errorf("%w: %s %x is %q", ErrTargetWrongType, target.label, target.hash, row.MessageType)
		}
	}

	mergedAt := opts.MergedAt
	if mergedAt == 0 {
		mergedAt = now().UnixNano()
	}

	a := opts.AntecedentAFormationHash[:]
	b := opts.AntecedentBFormationHash[:]
	first, second := a, b
	if bytes.Compare(a, b) > 0 {
		first, second = b, a
	}
	antecedents := [][]byte{
		append([]byte(nil), first...),
		append([]byte(nil), second...),
	}

	ev := &eventsv1.CoordinationRingMerge{
		AntecedentFormationEventHashes: antecedents,
		ProducedFormationEventHash:     opts.ProducedFormationHash[:],
		MergedAt:                       mergedAt,
		Reason:                         opts.Reason,
	}
	payload, hash, err := canonical.MarshalAndHash(ev)
	if err != nil {
		return CoordinationRingMergeReport{}, fmt.Errorf("hypothesis.MergeCoordinationRing: marshal merge: %w", err)
	}
	hex := canonical.HashHex(hash)

	_, lookupErr := sub.LookupRow(ctx, hash)
	alreadyPresent := lookupErr == nil
	if lookupErr != nil && !errors.Is(lookupErr, sql.ErrNoRows) {
		return CoordinationRingMergeReport{}, fmt.Errorf("hypothesis.MergeCoordinationRing: lookup merge %s: %w", hex, lookupErr)
	}

	committedAt := now().UnixNano()
	mergeRow := substrate.EventRow{
		EventHash:   hash,
		EventTime:   mergedAt,
		MessageType: string(ev.ProtoReflect().Descriptor().FullName()),
		PayloadRef:  hex[:2] + "/" + hex[2:],
		CommittedAt: committedAt,
	}

	if opts.Actor == "" {
		if err := sub.Append(ctx, mergeRow, payload); err != nil {
			return CoordinationRingMergeReport{}, fmt.Errorf("hypothesis.MergeCoordinationRing: append merge %s: %w", hex, err)
		}
		return CoordinationRingMergeReport{
			MergeEventHashHex: hex,
			AlreadyMerged:     alreadyPresent,
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
		return CoordinationRingMergeReport{}, fmt.Errorf("hypothesis.MergeCoordinationRing: marshal ingestion event: %w", err)
	}
	ingHex := canonical.HashHex(ingHash)
	ingRow := substrate.EventRow{
		EventHash:   ingHash,
		EventTime:   committedAt,
		MessageType: string(ingEv.ProtoReflect().Descriptor().FullName()),
		PayloadRef:  ingHex[:2] + "/" + ingHex[2:],
		CommittedAt: committedAt,
	}
	if err := sub.AppendPair(ctx, mergeRow, payload, ingRow, ingPayload); err != nil {
		return CoordinationRingMergeReport{}, fmt.Errorf("hypothesis.MergeCoordinationRing: append pair (merge %s, ingestion %s): %w", hex, ingHex, err)
	}

	return CoordinationRingMergeReport{
		MergeEventHashHex:     hex,
		AlreadyMerged:         alreadyPresent,
		IngestionEventHashHex: ingHex,
	}, nil
}
