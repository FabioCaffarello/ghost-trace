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

// AutomationGroupMergeOptions configures a single within-subtype
// AutomationGroup merge. Mirrors §0049's MergeOptions.
type AutomationGroupMergeOptions struct {
	// AntecedentAFormationHash is the content-hash of the FIRST
	// AutomationGroupFormation being merged. The "A"/"B" labels are
	// caller-facing only — Merge sorts the two hashes ascending
	// before recording (merge is symmetric).
	AntecedentAFormationHash [32]byte

	// AntecedentBFormationHash is the content-hash of the SECOND
	// AutomationGroupFormation being merged. MUST differ from
	// AntecedentAFormationHash.
	AntecedentBFormationHash [32]byte

	// ProducedFormationHash is the content-hash of a separately-
	// committed AutomationGroupFormation representing the merged
	// hypothesis (per §0049 Option B mirrored here).
	ProducedFormationHash [32]byte

	// MergedAt is the Unix-nanoseconds time at which the merge
	// recognition was recorded. Zero defaults to now().UnixNano().
	MergedAt int64

	// Reason is an operator-supplied free-form note. Optional.
	Reason string

	// Actor per §0109 — non-empty triggers AppendPair.
	Actor string
}

// AutomationGroupMergeReport is the per-MergeAutomationGroup
// outcome.
type AutomationGroupMergeReport struct {
	// MergeEventHashHex is the content-hash (hex) of the recorded
	// AutomationGroupMerge event.
	MergeEventHashHex string

	// AlreadyMerged is true when an identical merge event was
	// already in the substrate.
	AlreadyMerged bool

	// IngestionEventHashHex non-empty when Actor was supplied.
	IngestionEventHashHex string
}

// MergeAutomationGroup records an AutomationGroupMerge lifecycle
// event recognizing that the two antecedent AutomationGroupFormation
// hypotheses describe the same underlying phenomenon, with the
// produced hypothesis identified by a third (separately-committed)
// AutomationGroupFormation. Per Charter §2.5 BC5 the merge event is
// a Category I record committed via substrate.Append.
//
// Within-subtype only at this layer: all three target hashes MUST
// resolve to AutomationGroupFormation rows.
//
// Errors:
//   - ErrMergeAntecedentsIdentical: the two antecedent hashes are
//     byte-identical (sentinel reused from §0049 — merge-relation
//     symmetry constraint applies across subtypes).
//   - ErrTargetNotFound: one of the three formation hashes does not
//     resolve to any substrate row.
//   - ErrTargetWrongType: one of the three target hashes resolves
//     to a row whose message_type is NOT AutomationGroupFormation.
func MergeAutomationGroup(ctx context.Context, sub *substrate.Substrate, opts AutomationGroupMergeOptions, now func() time.Time) (AutomationGroupMergeReport, error) {
	if opts.AntecedentAFormationHash == opts.AntecedentBFormationHash {
		return AutomationGroupMergeReport{}, fmt.Errorf("%w: %x", ErrMergeAntecedentsIdentical, opts.AntecedentAFormationHash)
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
				return AutomationGroupMergeReport{}, fmt.Errorf("%w: %s %x", ErrTargetNotFound, target.label, target.hash)
			}
			return AutomationGroupMergeReport{}, fmt.Errorf("hypothesis.MergeAutomationGroup: lookup %s: %w", target.label, err)
		}
		if row.MessageType != automationGroupFormationMessageType {
			return AutomationGroupMergeReport{}, fmt.Errorf("%w: %s %x is %q", ErrTargetWrongType, target.label, target.hash, row.MessageType)
		}
	}

	mergedAt := opts.MergedAt
	if mergedAt == 0 {
		mergedAt = now().UnixNano()
	}

	// Ascending-sort antecedents (symmetric-relation idempotency
	// per §0049).
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

	ev := &eventsv1.AutomationGroupMerge{
		AntecedentFormationEventHashes: antecedents,
		ProducedFormationEventHash:     opts.ProducedFormationHash[:],
		MergedAt:                       mergedAt,
		Reason:                         opts.Reason,
	}
	payload, hash, err := canonical.MarshalAndHash(ev)
	if err != nil {
		return AutomationGroupMergeReport{}, fmt.Errorf("hypothesis.MergeAutomationGroup: marshal merge: %w", err)
	}
	hex := canonical.HashHex(hash)

	_, lookupErr := sub.LookupRow(ctx, hash)
	alreadyPresent := lookupErr == nil
	if lookupErr != nil && !errors.Is(lookupErr, sql.ErrNoRows) {
		return AutomationGroupMergeReport{}, fmt.Errorf("hypothesis.MergeAutomationGroup: lookup merge %s: %w", hex, lookupErr)
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
			return AutomationGroupMergeReport{}, fmt.Errorf("hypothesis.MergeAutomationGroup: append merge %s: %w", hex, err)
		}
		return AutomationGroupMergeReport{
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
		return AutomationGroupMergeReport{}, fmt.Errorf("hypothesis.MergeAutomationGroup: marshal ingestion event: %w", err)
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
		return AutomationGroupMergeReport{}, fmt.Errorf("hypothesis.MergeAutomationGroup: append pair (merge %s, ingestion %s): %w", hex, ingHex, err)
	}

	return AutomationGroupMergeReport{
		MergeEventHashHex:     hex,
		AlreadyMerged:         alreadyPresent,
		IngestionEventHashHex: ingHex,
	}, nil
}
