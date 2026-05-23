// Package layerb implements the Layer B demotion-candidacy predicate
// per the canonical-serialization-contract §Demotion-Candidacy Predicate
// section + decision-log §0141 service-tier implementation resolution.
//
// Constitutional shape (per Charter §2.5 frozen v0.3 + decision-log
// §0011 Q4-staged-combination + §0135 Layer B form + §0138 parameter
// calibration):
//
//	DEMOTE-CANDIDATE(H) := Layer A(H) AND ((freshness_B(H) < T_B)
//	                                     OR (saturation_C(H) > K_C))
//
// Per §0141 sub-decision C resolved to C1 (substrate-global window),
// freshness_B + saturation_C operate over the LAST N substrate events
// in commit order — the global universe. The filter (H.hash ∈
// r.closure_hashes) selects relevant events from that universe; the
// divisor in saturation_C is N (the global count), not the filter count.
//
// Per §0141 sub-decision B resolved to B1 (on-the-fly from substrate),
// this package performs no persistent storage. Each Evaluate call
// computes freshness_B + saturation_C fresh from substrate state.
//
// Per §0141 sub-decision A resolved to A1 (internal package), this
// package is sibling to the lifecycle-operation files in
// internal/hypothesis/ and is called by demote-hypothesis +
// 4 subtype variants.
//
// Layer A (cadence gate) is NOT evaluated here — Layer A predates this
// package and remains the existing per-demote-function computation
// (CadenceSatisfied flag in *DemoteReport per §0011 advisory pattern).
// Layer B is the deep criterion this package operationalizes.
package layerb

import (
	"context"
	"fmt"
	"math/big"

	"google.golang.org/protobuf/proto"

	commonv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/common/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// EvaluateOptions configures a single Layer B evaluation.
type EvaluateOptions struct {
	// FormationEventHash is H.hash per the canonical-serialization-
	// contract — the content-hash of the originating formation event
	// (BehavioralClusterFormation / AutomationGroupFormation /
	// CampaignHypothesisFormation / CoordinationRingFormation). Per
	// decision-log §0045, this hash IS the hypothesis's stable
	// identifier; promotion events carry it as formation_event_hash.
	FormationEventHash [32]byte

	// Params is the LayerBParameters proto recorded on the promotion
	// event (per §0138 N_A bundling). Required: t_b, k_c, n_window.
	// nil → Evaluate returns a NotEvaluated verdict with an error;
	// the caller must surface the absence as a Layer-B-not-runnable
	// state.
	Params *commonv1.LayerBParameters
}

// Verdict is the per-Evaluate outcome.
//
// Per §0141 sub-decision D resolved to D1 (transient DemoteReport
// extension), this struct is the in-memory verdict surface. It is NOT
// committed to substrate; callers may include selected fields in their
// own report types (e.g., DemoteReport's LayerB* fields) for operator
// visibility.
type Verdict struct {
	// FreshnessBNumerator / FreshnessBDenominator encode the
	// computed freshness_B(H) value as a rational pair. Convention
	// matches the EvidentialIndependence proto: numerator/denominator
	// with denominator > 0. When the window contains zero filter-
	// matching events, FreshnessBDenominator is set to 1 and
	// FreshnessBNumerator to 0; the verdict's FreshnessUndefined
	// flag indicates this state.
	FreshnessBNumerator   uint64
	FreshnessBDenominator uint64

	// SaturationCNumerator / SaturationCDenominator encode the
	// computed saturation_C(H) value as a rational pair. Convention
	// matches FreshnessB. When the window contains zero events
	// (Substrate is empty post-promotion), SaturationCDenominator
	// is set to 1 and SaturationCNumerator to 0.
	SaturationCNumerator   uint64
	SaturationCDenominator uint64

	// FreshnessUndefined is true when the L-BC freshness component
	// could not be computed because zero events in the window matched
	// the filter (H.hash ∈ closure_hashes). Per the L-BC-OR semantic,
	// an undefined freshness component does NOT fire freshness_B <
	// T_B; the disjunction reduces to saturation_C > K_C alone.
	FreshnessUndefined bool

	// FreshnessFired is true when freshness_B(H) < T_B (per §0135 +
	// §0138). False if freshness is undefined or if the threshold is
	// not crossed.
	FreshnessFired bool

	// SaturationFired is true when saturation_C(H) > K_C (per §0135
	// + §0138).
	SaturationFired bool

	// Fired is the L-BC-OR disjunction: FreshnessFired OR
	// SaturationFired. This is the Layer B verdict — true means H
	// is a Layer B demotion candidate.
	Fired bool

	// WindowEventCount is the number of substrate events scanned
	// (≤ N). Equals N when substrate contains at least N events;
	// otherwise equals the total event count.
	WindowEventCount uint64

	// FilterMatchCount is the number of events in the window where
	// H.hash ∈ closure_hashes (the filter for freshness_B's
	// averaging set).
	FilterMatchCount uint64
}

// Evaluate computes the Layer B verdict for the hypothesis identified
// by opts.FormationEventHash against the substrate's recent N events,
// per the canonical-serialization-contract §Demotion-Candidacy Predicate.
//
// Returns an error if opts.Params is nil or carries invalid threshold
// values (denominator = 0); the caller should surface the error as a
// Layer-B-not-runnable state.
func Evaluate(ctx context.Context, sub *substrate.Substrate, opts EvaluateOptions) (Verdict, error) {
	if sub == nil {
		return Verdict{}, fmt.Errorf("layerb.Evaluate: substrate is nil")
	}
	if opts.Params == nil {
		return Verdict{}, fmt.Errorf("layerb.Evaluate: opts.Params is nil (LayerBParameters required per §0138)")
	}
	tB := opts.Params.GetTB()
	kC := opts.Params.GetKC()
	if tB == nil || kC == nil {
		return Verdict{}, fmt.Errorf("layerb.Evaluate: t_b or k_c is nil in LayerBParameters")
	}
	if tB.GetDenominator() == 0 || kC.GetDenominator() == 0 {
		return Verdict{}, fmt.Errorf("layerb.Evaluate: t_b or k_c denominator is zero (per §2.6 rational-pair invariant)")
	}
	n := opts.Params.GetNWindow()
	if n == 0 {
		return Verdict{}, fmt.Errorf("layerb.Evaluate: n_window is zero (per §0138 W-count window form)")
	}

	// Step 1: Walk substrate in commit order, retain the last N events
	// in a ring buffer. Substrate.WalkEvents iterates in commit order
	// ascending; the ring buffer captures the tail.
	window := make([]substrate.EventRow, 0, n)
	var ringPos uint64
	ringFull := false
	err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if uint64(len(window)) < n {
			window = append(window, row)
			return nil
		}
		ringFull = true
		window[ringPos] = row
		ringPos = (ringPos + 1) % n
		return nil
	})
	if err != nil {
		return Verdict{}, fmt.Errorf("layerb.Evaluate: walk substrate: %w", err)
	}

	// Step 2: Reorder ring buffer to commit-order ascending. If the
	// ring filled, the oldest entry is at ringPos; otherwise the buffer
	// is already in commit-order ascending.
	orderedWindow := window
	if ringFull && ringPos != 0 {
		orderedWindow = make([]substrate.EventRow, 0, len(window))
		orderedWindow = append(orderedWindow, window[ringPos:]...)
		orderedWindow = append(orderedWindow, window[:ringPos]...)
	}

	verdict := Verdict{
		WindowEventCount: uint64(len(orderedWindow)),
	}

	// Step 3: For each event in the window, read its payload + decode
	// the proto + extract closure_hashes + direct_influenced_by +
	// evidential_independence. Apply the filter (FormationEventHash ∈
	// closure_hashes) and the L-C exclusion (FormationEventHash ∉
	// direct_influenced_by). Accumulate freshness_B sum + saturation_C
	// count.
	//
	// freshness_B = avg(EI) over events where H.hash ∈ closure_hashes
	// saturation_C = count(H.hash ∈ closure_hashes AND H.hash ∉
	//                      direct_influenced_by) / N
	freshSum := new(big.Rat)
	var saturationCount uint64
	var filterMatchCount uint64

	target := opts.FormationEventHash

	for _, row := range orderedWindow {
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return Verdict{}, fmt.Errorf("layerb.Evaluate: read blob %x: %w", row.EventHash, err)
		}
		closure, direct, ei, ok := extractLayerBFields(row.MessageType, payload)
		if !ok {
			// Event has no closure_hashes field (Cat I primary, audit
			// record, etc.). Filter naturally excludes it; contributes
			// nothing to freshness_B sum or saturation_C numerator.
			continue
		}
		if !hashListContains(closure, target) {
			continue
		}
		filterMatchCount++

		// EI carrier (formation/OperationalSession with closure_hashes)
		// contributes to freshness_B avg.
		if ei != nil && ei.GetDenominator() > 0 {
			r := new(big.Rat).SetFrac(
				new(big.Int).SetUint64(ei.GetNumerator()),
				new(big.Int).SetUint64(ei.GetDenominator()),
			)
			freshSum.Add(freshSum, r)
		}

		// saturation_C: H ∈ closure_hashes AND H ∉ direct_influenced_by
		if !hashListContains(direct, target) {
			saturationCount++
		}
	}

	verdict.FilterMatchCount = filterMatchCount

	// Step 4: Compute freshness_B = freshSum / filterMatchCount, if any.
	if filterMatchCount == 0 {
		verdict.FreshnessUndefined = true
		verdict.FreshnessBNumerator = 0
		verdict.FreshnessBDenominator = 1
	} else {
		avg := new(big.Rat).Quo(
			freshSum,
			new(big.Rat).SetFrac(new(big.Int).SetUint64(filterMatchCount), big.NewInt(1)),
		)
		verdict.FreshnessBNumerator = avg.Num().Uint64()
		verdict.FreshnessBDenominator = avg.Denom().Uint64()
		// Compare avg < T_B
		tBRat := new(big.Rat).SetFrac(
			new(big.Int).SetUint64(tB.GetNumerator()),
			new(big.Int).SetUint64(tB.GetDenominator()),
		)
		if avg.Cmp(tBRat) < 0 {
			verdict.FreshnessFired = true
		}
	}

	// Step 5: Compute saturation_C = saturationCount / N.
	satRat := new(big.Rat).SetFrac(
		new(big.Int).SetUint64(saturationCount),
		new(big.Int).SetUint64(n),
	)
	verdict.SaturationCNumerator = satRat.Num().Uint64()
	verdict.SaturationCDenominator = satRat.Denom().Uint64()
	kCRat := new(big.Rat).SetFrac(
		new(big.Int).SetUint64(kC.GetNumerator()),
		new(big.Int).SetUint64(kC.GetDenominator()),
	)
	if satRat.Cmp(kCRat) > 0 {
		verdict.SaturationFired = true
	}

	verdict.Fired = verdict.FreshnessFired || verdict.SaturationFired
	return verdict, nil
}

// extractLayerBFields reads the closure_hashes, direct_influenced_by,
// and evidential_independence fields from a substrate event payload via
// protoreflect. Returns (closure, direct, ei, ok) where ok is false if
// the message type does not carry the closure_hashes field (in which
// case the event is filtered out of Layer B's computation by
// construction).
//
// The reflection-based reader is subtype-agnostic — it handles all four
// formation proto types + OperationalSession uniformly. Per §0136 these
// are the only message types subject to the paired-dimension commitment
// + closure-storage shape.
func extractLayerBFields(messageType string, payload []byte) (closure [][]byte, direct [][]byte, ei *commonv1.EvidentialIndependence, ok bool) {
	// Use proto.UnmarshalOptions with Discard:true to ignore unknown
	// fields gracefully; the proto must be unmarshaled into the correct
	// concrete type by message-type name.
	concrete := concreteMessageFor(messageType)
	if concrete == nil {
		return nil, nil, nil, false
	}
	if err := proto.Unmarshal(payload, concrete); err != nil {
		return nil, nil, nil, false
	}

	m := concrete.ProtoReflect()
	fields := m.Descriptor().Fields()
	closureFd := fields.ByName("closure_hashes")
	directFd := fields.ByName("direct_influenced_by")
	eiFd := fields.ByName("evidential_independence")
	if closureFd == nil {
		// Message type does not carry closure_hashes — by construction
		// not subject to Layer B's filter.
		return nil, nil, nil, false
	}

	closureList := m.Get(closureFd).List()
	for i := 0; i < closureList.Len(); i++ {
		closure = append(closure, closureList.Get(i).Bytes())
	}
	if directFd != nil {
		directList := m.Get(directFd).List()
		for i := 0; i < directList.Len(); i++ {
			direct = append(direct, directList.Get(i).Bytes())
		}
	}
	if eiFd != nil && m.Has(eiFd) {
		eiMsg := m.Get(eiFd).Message()
		// Extract numerator + denominator via reflection rather than
		// type-assertion — keeps the helper subtype-agnostic.
		numFd := eiMsg.Descriptor().Fields().ByName("numerator")
		denomFd := eiMsg.Descriptor().Fields().ByName("denominator")
		if numFd != nil && denomFd != nil {
			ei = &commonv1.EvidentialIndependence{
				Numerator:   eiMsg.Get(numFd).Uint(),
				Denominator: eiMsg.Get(denomFd).Uint(),
			}
		}
	}
	return closure, direct, ei, true
}

// hashListContains returns true if target appears in list. Linear scan;
// the per-event list is small (closure_hashes is per-record, typically
// O(1) to O(10) elements). Comparison via constant-time-style byte
// equality is unnecessary at this layer — these are public content
// hashes, not secrets.
func hashListContains(list [][]byte, target [32]byte) bool {
	for _, h := range list {
		if len(h) != 32 {
			continue
		}
		var hashed [32]byte
		copy(hashed[:], h)
		if hashed == target {
			return true
		}
	}
	return false
}

// concreteMessageFor returns a fresh proto.Message of the type named
// by messageType, or nil if the message type is not in the Layer-B-
// relevant set. The set is the 4 formation protos + OperationalSession
// per §0136 paired-dimension subject enumeration; all other message
// types are filtered out of Layer B's computation.
//
// Per protoreflect convention, returning a zero-value concrete message
// is sufficient for Unmarshal; ProtoReflect() yields the descriptor
// from there.
//
// The dispatch is intentionally explicit rather than registry-based to
// keep the Layer-B-relevant set visible at code-review time. A future
// schemas-evolution event adding a new closure-carrying message type
// requires explicit registration here.
func concreteMessageFor(messageType string) proto.Message {
	factory, ok := concreteMessageRegistry[messageType]
	if !ok {
		return nil
	}
	return factory()
}
