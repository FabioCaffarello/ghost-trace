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

// CampaignHypothesisSplitOptions configures a single within-subtype
// CampaignHypothesis split. Mirrors §0050's SplitOptions and §0061's
// AG variant.
type CampaignHypothesisSplitOptions struct {
	AntecedentFormationHash  [32]byte
	SuccessorFormationHashes [][32]byte
	SplitAt                  int64
	Reason                   string

	// Actor per §0110 — non-empty triggers AppendPair.
	Actor string
}

// CampaignHypothesisSplitReport is the per-SplitCampaignHypothesis
// outcome.
type CampaignHypothesisSplitReport struct {
	SplitEventHashHex string
	AlreadySplit      bool

	// IngestionEventHashHex non-empty when Actor was supplied.
	IngestionEventHashHex string
}

// SplitCampaignHypothesis records a CampaignHypothesisSplit
// lifecycle event recognizing that the antecedent
// CampaignHypothesisFormation hypothesis contained multiple
// distinct operations and has been divided into the supplied
// successor CampaignHypothesisFormation hypotheses. Per Charter
// §2.5 BC5 the split event is a Cat I record committed via
// substrate.Append.
//
// Within-subtype only. Cross-subtype split per §Cross-subtype
// operations remains deferred.
//
// Errors:
//   - ErrSplitInsufficientSuccessors: fewer than 2 successors.
//   - ErrSplitSuccessorsNotDistinct: duplicate successors or
//     antecedent-overlap.
//   - ErrTargetNotFound: antecedent or successor not in substrate.
//   - ErrTargetWrongType: target NOT a CampaignHypothesisFormation.
func SplitCampaignHypothesis(ctx context.Context, sub *substrate.Substrate, opts CampaignHypothesisSplitOptions, now func() time.Time) (CampaignHypothesisSplitReport, error) {
	if len(opts.SuccessorFormationHashes) < 2 {
		return CampaignHypothesisSplitReport{}, fmt.Errorf("%w: got %d", ErrSplitInsufficientSuccessors, len(opts.SuccessorFormationHashes))
	}

	seen := make(map[[32]byte]struct{}, len(opts.SuccessorFormationHashes)+1)
	seen[opts.AntecedentFormationHash] = struct{}{}
	for _, succ := range opts.SuccessorFormationHashes {
		if _, dup := seen[succ]; dup {
			return CampaignHypothesisSplitReport{}, fmt.Errorf("%w: %x", ErrSplitSuccessorsNotDistinct, succ)
		}
		seen[succ] = struct{}{}
	}

	if now == nil {
		now = time.Now
	}

	row, err := sub.LookupRow(ctx, opts.AntecedentFormationHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CampaignHypothesisSplitReport{}, fmt.Errorf("%w: antecedent %x", ErrTargetNotFound, opts.AntecedentFormationHash)
		}
		return CampaignHypothesisSplitReport{}, fmt.Errorf("hypothesis.SplitCampaignHypothesis: lookup antecedent: %w", err)
	}
	if row.MessageType != campaignHypothesisFormationMessageType {
		return CampaignHypothesisSplitReport{}, fmt.Errorf("%w: antecedent %x is %q", ErrTargetWrongType, opts.AntecedentFormationHash, row.MessageType)
	}
	for i, succ := range opts.SuccessorFormationHashes {
		srow, err := sub.LookupRow(ctx, succ)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return CampaignHypothesisSplitReport{}, fmt.Errorf("%w: successor[%d] %x", ErrTargetNotFound, i, succ)
			}
			return CampaignHypothesisSplitReport{}, fmt.Errorf("hypothesis.SplitCampaignHypothesis: lookup successor[%d]: %w", i, err)
		}
		if srow.MessageType != campaignHypothesisFormationMessageType {
			return CampaignHypothesisSplitReport{}, fmt.Errorf("%w: successor[%d] %x is %q", ErrTargetWrongType, i, succ, srow.MessageType)
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

	ev := &eventsv1.CampaignHypothesisSplit{
		AntecedentFormationEventHash:  opts.AntecedentFormationHash[:],
		SuccessorFormationEventHashes: successorBytes,
		SplitAt:                       splitAt,
		Reason:                        opts.Reason,
	}
	payload, hash, err := canonical.MarshalAndHash(ev)
	if err != nil {
		return CampaignHypothesisSplitReport{}, fmt.Errorf("hypothesis.SplitCampaignHypothesis: marshal split: %w", err)
	}
	hex := canonical.HashHex(hash)

	_, lookupErr := sub.LookupRow(ctx, hash)
	alreadyPresent := lookupErr == nil
	if lookupErr != nil && !errors.Is(lookupErr, sql.ErrNoRows) {
		return CampaignHypothesisSplitReport{}, fmt.Errorf("hypothesis.SplitCampaignHypothesis: lookup split %s: %w", hex, lookupErr)
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
			return CampaignHypothesisSplitReport{}, fmt.Errorf("hypothesis.SplitCampaignHypothesis: append split %s: %w", hex, err)
		}
		return CampaignHypothesisSplitReport{
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
		return CampaignHypothesisSplitReport{}, fmt.Errorf("hypothesis.SplitCampaignHypothesis: marshal ingestion event: %w", err)
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
		return CampaignHypothesisSplitReport{}, fmt.Errorf("hypothesis.SplitCampaignHypothesis: append pair (split %s, ingestion %s): %w", hex, ingHex, err)
	}

	return CampaignHypothesisSplitReport{
		SplitEventHashHex:     hex,
		AlreadySplit:          alreadyPresent,
		IngestionEventHashHex: ingHex,
	}, nil
}
