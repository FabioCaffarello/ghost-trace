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

// ErrMergeAntecedentsIdentical is returned by Merge when the caller
// passes the same formation hash as both antecedents. Per
// lifecycle-semantics.md line 28 merge combines TWO hypotheses
// recognized as describing the same underlying phenomenon — merging
// a hypothesis with itself is not a valid §2.5 lifecycle move.
var ErrMergeAntecedentsIdentical = errors.New("hypothesis.Merge: antecedent formations must be distinct")

// MergeOptions configures a single within-subtype merge. The
// (AntecedentA, AntecedentB, ProducedFormation, MergedAt, Reason)
// tuple uniquely identifies a merge event's content-hash — re-running
// with identical values is idempotent.
//
// Within-subtype scope: all three formation hashes MUST resolve to
// BehavioralClusterFormation rows. Cross-subtype merge per
// entity-model.md §Cross-subtype operations remains deferred to
// lifecycle-semantics.md post-Q4 redaction and is not implemented at
// this layer.
type MergeOptions struct {
	// AntecedentAFormationHash is the content-hash of the FIRST
	// BehavioralClusterFormation being merged. Per §0045 the hash IS
	// the hypothesis identity. The "A"/"B" labels are caller-facing
	// only — Merge sorts the two hashes ascending before recording so
	// the merge event's content-hash is invariant under argument
	// order (merge is a symmetric relation).
	AntecedentAFormationHash [32]byte

	// AntecedentBFormationHash is the content-hash of the SECOND
	// BehavioralClusterFormation being merged. MUST differ from
	// AntecedentAFormationHash; Merge returns ErrMergeAntecedentsIdentical
	// otherwise.
	AntecedentBFormationHash [32]byte

	// ProducedFormationHash is the content-hash of a separately-
	// committed BehavioralClusterFormation event representing the
	// merged hypothesis. Per §0045 this hash IS the produced
	// hypothesis's identity; subsequent lifecycle operations
	// (promote, demote, dissolve, future merge/split) target THIS
	// hash, not the merge event's own hash.
	//
	// The substrate does NOT enforce structural-consistency
	// relationships between produced.source_event_hashes / actor_refs
	// and the union of the antecedents' corresponding fields — those
	// are operational conventions of the inference process and are
	// surfaced through the reason field and projection-time audits.
	ProducedFormationHash [32]byte

	// MergedAt is the Unix-nanoseconds time at which the merge
	// recognition was recorded. Zero defaults to now().UnixNano().
	// Explicit non-zero values permitted for forensic replay +
	// deterministic test recording.
	MergedAt int64

	// Reason is an operator-supplied free-form note explaining the
	// recognition that the two antecedent hypotheses describe the
	// same underlying phenomenon. Optional but strongly recommended.
	Reason string
}

// MergeReport is the per-Merge outcome.
type MergeReport struct {
	// MergeEventHashHex is the content-hash (hex) of the recorded
	// BehavioralClusterMerge event.
	MergeEventHashHex string

	// AlreadyMerged is true when an identical merge event was already
	// in the substrate (content-hash collision). False when this
	// Merge invocation committed a new row.
	AlreadyMerged bool
}

// Merge records a BehavioralClusterMerge lifecycle event recognizing
// that the two antecedent BehavioralClusterFormation hypotheses
// describe the same underlying phenomenon, with the produced
// hypothesis identified by a third (separately-committed)
// BehavioralClusterFormation. Per Charter §2.5 BC5 the merge event
// is a Category I record committed via substrate.Append (acquires
// writeMu per concurrency-pattern.md §Substrate-Writer Serialization).
//
// Within-subtype only at this layer: all three target hashes MUST
// resolve to BehavioralClusterFormation rows. Cross-subtype merge
// per entity-model.md §Cross-subtype operations is deferred to
// lifecycle-semantics.md post-Q4 redaction.
//
// Errors:
//   - ErrMergeAntecedentsIdentical: the two antecedent hashes are
//     byte-identical.
//   - ErrTargetNotFound: one of the three formation hashes does not
//     resolve to any substrate row.
//   - ErrTargetWrongType: one of the three target hashes resolves to
//     a row whose message_type is NOT BehavioralClusterFormation.
func Merge(ctx context.Context, sub *substrate.Substrate, opts MergeOptions, now func() time.Time) (MergeReport, error) {
	if opts.AntecedentAFormationHash == opts.AntecedentBFormationHash {
		return MergeReport{}, fmt.Errorf("%w: %x", ErrMergeAntecedentsIdentical, opts.AntecedentAFormationHash)
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
				return MergeReport{}, fmt.Errorf("%w: %s %x", ErrTargetNotFound, target.label, target.hash)
			}
			return MergeReport{}, fmt.Errorf("hypothesis.Merge: lookup %s: %w", target.label, err)
		}
		if row.MessageType != behavioralClusterFormationMessageType {
			return MergeReport{}, fmt.Errorf("%w: %s %x is %q", ErrTargetWrongType, target.label, target.hash, row.MessageType)
		}
	}

	mergedAt := opts.MergedAt
	if mergedAt == 0 {
		mergedAt = now().UnixNano()
	}

	// Merge is symmetric; sort the antecedents ascending so the
	// merge event's content-hash is invariant under caller argument
	// order. Two calls Merge(A, B) and Merge(B, A) MUST produce the
	// same substrate row.
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

	ev := &eventsv1.BehavioralClusterMerge{
		AntecedentFormationEventHashes: antecedents,
		ProducedFormationEventHash:     opts.ProducedFormationHash[:],
		MergedAt:                       mergedAt,
		Reason:                         opts.Reason,
	}
	payload, hash, err := canonical.MarshalAndHash(ev)
	if err != nil {
		return MergeReport{}, fmt.Errorf("hypothesis.Merge: marshal merge: %w", err)
	}
	hex := canonical.HashHex(hash)

	_, lookupErr := sub.LookupRow(ctx, hash)
	alreadyPresent := lookupErr == nil
	if lookupErr != nil && !errors.Is(lookupErr, sql.ErrNoRows) {
		return MergeReport{}, fmt.Errorf("hypothesis.Merge: lookup merge %s: %w", hex, lookupErr)
	}

	mergeRow := substrate.EventRow{
		EventHash:   hash,
		EventTime:   mergedAt,
		MessageType: string(ev.ProtoReflect().Descriptor().FullName()),
		PayloadRef:  hex[:2] + "/" + hex[2:],
		CommittedAt: now().UnixNano(),
	}
	if err := sub.Append(ctx, mergeRow, payload); err != nil {
		return MergeReport{}, fmt.Errorf("hypothesis.Merge: append merge %s: %w", hex, err)
	}

	return MergeReport{
		MergeEventHashHex: hex,
		AlreadyMerged:     alreadyPresent,
	}, nil
}
