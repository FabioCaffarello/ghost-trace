package hypothesis

import (
	"context"

	commonv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/common/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/hypothesis/layerb"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// LayerBReport is the per-Demote verdict surface populated by
// evaluateLayerB. Embedded as a field in each DemoteReport variant
// (Demote / DemoteAutomationGroup / DemoteCampaignHypothesis /
// DemoteCoordinationRing) per §0141 sub-decision D1 (transient
// DemoteReport extension).
//
// Convention mirrors the existing CadenceSatisfied advisory pattern per
// §0141 sub-decision E1 (advisory like Layer A): the fields are
// informational; demote records the demotion regardless of Fired.
// Operator and downstream projections consume the verdict for surfacing
// demote-candidacy state in CLI output and reporting tools.
//
// Evaluated=false signals that Layer B evaluation could NOT run for
// this demote — typically because the source promotion event predates
// the §0138 LayerBParameters bundling and the promotion's
// layer_b_parameters field is unset. All other fields are zero-valued
// when Evaluated=false; callers should not interpret Fired=false as
// "Layer B passed" when Evaluated=false.
type LayerBReport struct {
	// Evaluated is true when Layer B's predicate was computed for this
	// demote. False if the source promotion lacks LayerBParameters
	// (pre-§0138 legacy promotion); the remaining fields are zero in
	// that case.
	Evaluated bool

	// Fired is the L-BC-OR disjunction (FreshnessFired OR
	// SaturationFired) per §0135 + §0136. True means the demote target
	// satisfies Layer B's deep criterion at evaluation time.
	Fired bool

	// FreshnessFired is true when freshness_B(H) < T_B at evaluation
	// time. False when freshness is undefined or above the threshold.
	FreshnessFired bool

	// SaturationFired is true when saturation_C(H) > K_C at evaluation
	// time.
	SaturationFired bool

	// FreshnessUndefined is true when zero events in the recent-N
	// window matched the filter (H.hash ∈ closure_hashes), making
	// freshness_B's average undefined. Per the L-BC-OR semantic this
	// state cannot fire the freshness component; the disjunction
	// reduces to saturation_C > K_C alone.
	FreshnessUndefined bool

	// FreshnessBNumerator + FreshnessBDenominator encode the computed
	// freshness_B value as a rational pair. Both zero when
	// FreshnessUndefined is true (denominator is 1, numerator 0).
	FreshnessBNumerator   uint64
	FreshnessBDenominator uint64

	// SaturationCNumerator + SaturationCDenominator encode the computed
	// saturation_C value as a rational pair.
	SaturationCNumerator   uint64
	SaturationCDenominator uint64

	// WindowEventCount is the number of substrate events scanned by
	// the evaluation (≤ N from LayerBParameters.n_window).
	WindowEventCount uint64

	// FilterMatchCount is the number of events in the window where
	// H.hash ∈ closure_hashes.
	FilterMatchCount uint64
}

// evaluateLayerB runs the layerb predicate against substrate for the
// promoted hypothesis identified by formationEventHash + params.
// Returns a LayerBReport with Evaluated=false (and zero-value other
// fields) if params is nil — legacy promotions that predate §0138
// LayerBParameters bundling are not evaluated.
//
// Per §0141 sub-decision B1 (on-the-fly from substrate), the helper
// performs no persistent storage; each call recomputes the verdict
// from current substrate state.
func evaluateLayerB(ctx context.Context, sub *substrate.Substrate, formationEventHash [32]byte, params *commonv1.LayerBParameters) (LayerBReport, error) {
	if params == nil {
		return LayerBReport{Evaluated: false}, nil
	}
	verdict, err := layerb.Evaluate(ctx, sub, layerb.EvaluateOptions{
		FormationEventHash: formationEventHash,
		Params:             params,
	})
	if err != nil {
		return LayerBReport{Evaluated: false}, err
	}
	return LayerBReport{
		Evaluated:              true,
		Fired:                  verdict.Fired,
		FreshnessFired:         verdict.FreshnessFired,
		SaturationFired:        verdict.SaturationFired,
		FreshnessUndefined:     verdict.FreshnessUndefined,
		FreshnessBNumerator:    verdict.FreshnessBNumerator,
		FreshnessBDenominator:  verdict.FreshnessBDenominator,
		SaturationCNumerator:   verdict.SaturationCNumerator,
		SaturationCDenominator: verdict.SaturationCDenominator,
		WindowEventCount:       verdict.WindowEventCount,
		FilterMatchCount:       verdict.FilterMatchCount,
	}, nil
}
