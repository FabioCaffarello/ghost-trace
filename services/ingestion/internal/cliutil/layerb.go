// Package cliutil contains small helpers shared across the
// services/ingestion/cmd/ CLI mains. Per CLAUDE.md §7 constitutional
// minimalism: factored out only when at least two CLIs need the same
// surface (the §0141 LayerB CLI-surface PR introduces this need across
// the 4 demote CLIs + 4 promote CLIs).
package cliutil

import (
	commonv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/common/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/hypothesis"
)

// LayerBPayload is the JSON serialization shape for the Layer B
// verdict produced by hypothesis.LayerBReport. Field-level omitempty
// keeps legacy-promotion output (Evaluated=false) compact — only the
// `evaluated` key appears for the legacy path.
//
// Per §0141 sub-decision D1 (transient DemoteReport extension), the
// verdict is informational; demote records the demotion regardless of
// Fired per sub-decision E1 (advisory like Layer A).
type LayerBPayload struct {
	// Evaluated is true when Layer B's predicate was computed. False
	// when the source promotion lacks LayerBParameters (pre-§0138
	// legacy promotion or promote without --layer-b opt-in).
	Evaluated bool `json:"evaluated"`

	// Fired is the L-BC-OR disjunction (FreshnessFired OR
	// SaturationFired). Omitted when Evaluated=false.
	Fired bool `json:"fired,omitempty"`

	// FreshnessFired is true when freshness_B(H) < T_B.
	FreshnessFired bool `json:"freshness_fired,omitempty"`

	// SaturationFired is true when saturation_C(H) > K_C.
	SaturationFired bool `json:"saturation_fired,omitempty"`

	// FreshnessUndefined is true when filter matched zero events.
	FreshnessUndefined bool `json:"freshness_undefined,omitempty"`

	// FreshnessBNumerator + FreshnessBDenominator encode the computed
	// freshness_B value as a rational pair. Omitted when zero.
	FreshnessBNumerator   uint64 `json:"freshness_b_numerator,omitempty"`
	FreshnessBDenominator uint64 `json:"freshness_b_denominator,omitempty"`

	// SaturationCNumerator + SaturationCDenominator encode saturation_C.
	SaturationCNumerator   uint64 `json:"saturation_c_numerator,omitempty"`
	SaturationCDenominator uint64 `json:"saturation_c_denominator,omitempty"`

	// WindowEventCount is the number of substrate events scanned.
	WindowEventCount uint64 `json:"window_event_count,omitempty"`

	// FilterMatchCount is the number of events in the window where
	// H.hash ∈ closure_hashes.
	FilterMatchCount uint64 `json:"filter_match_count,omitempty"`
}

// LayerBPayloadFromReport converts a hypothesis.LayerBReport into its
// CLI JSON shape. Used by all 4 demote CLIs.
func LayerBPayloadFromReport(r hypothesis.LayerBReport) LayerBPayload {
	return LayerBPayload{
		Evaluated:              r.Evaluated,
		Fired:                  r.Fired,
		FreshnessFired:         r.FreshnessFired,
		SaturationFired:        r.SaturationFired,
		FreshnessUndefined:     r.FreshnessUndefined,
		FreshnessBNumerator:    r.FreshnessBNumerator,
		FreshnessBDenominator:  r.FreshnessBDenominator,
		SaturationCNumerator:   r.SaturationCNumerator,
		SaturationCDenominator: r.SaturationCDenominator,
		WindowEventCount:       r.WindowEventCount,
		FilterMatchCount:       r.FilterMatchCount,
	}
}

// LayerBSummary returns a one-token human summary suitable for inline
// stderr reporting in the CLI summary line. Variants:
//
//   - "layer_b=not_evaluated" — promotion lacks LayerBParameters.
//   - "layer_b=fired"         — L-BC-OR fired (freshness < T_B or saturation > K_C).
//   - "layer_b=not_fired"     — predicate ran, did not fire.
func LayerBSummary(r hypothesis.LayerBReport) string {
	if !r.Evaluated {
		return "layer_b=not_evaluated"
	}
	if r.Fired {
		return "layer_b=fired"
	}
	return "layer_b=not_fired"
}

// InceptionPhaseLayerBParameters returns a fresh *commonv1.LayerBParameters
// populated with §0138 inception-phase resolved values: T_B = K_C = 1/2;
// N_window = 1000; N_A_duration_nanoseconds = 1 day. Used by the promote
// CLIs when --layer-b is set without per-parameter overrides — per
// §0141 sub-decision F3 (CLI operator-supplied with bundled defaults).
//
// cadenceSeconds (when non-zero) overrides the N_A bundling default;
// CLI's --cadence-seconds flag is the operator-supplied override per
// §0141 F3.
func InceptionPhaseLayerBParameters(cadenceSeconds int64) *commonv1.LayerBParameters {
	var naNs uint64
	if cadenceSeconds <= 0 {
		// §0138 default: 1 day = 86400 seconds = 86400e9 nanoseconds.
		naNs = uint64(86400) * secondInNanos
	} else {
		naNs = uint64(cadenceSeconds) * secondInNanos
	}
	return &commonv1.LayerBParameters{
		TB:                    &commonv1.EvidentialIndependence{Numerator: 1, Denominator: 2},
		KC:                    &commonv1.EvidentialIndependence{Numerator: 1, Denominator: 2},
		NWindow:               1000,
		NADurationNanoseconds: naNs,
	}
}

const secondInNanos uint64 = 1_000_000_000
