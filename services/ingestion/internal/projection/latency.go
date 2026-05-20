package projection

// computeLatencies populates the per-projection latency fields from
// the materialized LifecycleHistory + per-type lifecycle pointers.
// Per decision-log §0055 the projection layer derives three
// latencies on the §2.5 lifecycle surface:
//
//   - Formation → first promotion latency. The first promotion
//     against this formation is the earliest entry in LifecycleHistory
//     whose Type equals promotionMessageType (the history is sorted
//     ascending by event_time so the first such entry is the
//     earliest by definition).
//   - Latest-promotion → latest-demotion latency. Uses the already-
//     computed LatestPromotion + LatestDemotion pointers; their
//     pairing is per §0051's per-formation projection semantics
//     (LatestDemotion only populated when it targets LatestPromotion).
//   - Formation → dissolution latency. Uses the Dissolution pointer
//     (§0048 sole-dissolution semantic; per-event-time pick when
//     multiple are present).
//
// Latency values may be negative if the underlying events arrived
// out of order (substrate accepts events regardless of operator-
// supplied timestamp; the projection does not reject negative
// latencies — that would be a category claim about producer
// correctness rather than a §2.1 invariant violation).
//
// computeLatencies is pure: same input projection produces the
// same latency fields. Idempotent under repeated invocation.
func computeLatencies(proj *HypothesisProjection) {
	if len(proj.LifecycleHistory) == 0 {
		return
	}
	formationEventTime := proj.LifecycleHistory[0].EventTime
	// LifecycleHistory[0] is the formation entry (history is
	// ascending by event_time and the formation is the FIRST event
	// in the chain — every subsequent event references it).
	// If the substrate somehow contained a lifecycle event whose
	// event_time PRECEDED the formation, LifecycleHistory[0] would
	// be that earlier event, not the formation. That would be a
	// pathological substrate (the writer-side hypothesis package
	// rejects such records), so the assumption is structurally
	// sound. Defensive code below uses the formation hash check.
	for _, entry := range proj.LifecycleHistory {
		if entry.EventHash == proj.FormationHash {
			formationEventTime = entry.EventTime
			break
		}
	}

	// Formation → first promotion.
	for _, entry := range proj.LifecycleHistory {
		if entry.Type == promotionMessageType {
			latency := entry.EventTime - formationEventTime
			proj.FormationToFirstPromotionLatencyNs = &latency
			break
		}
	}

	// Latest promotion → latest demotion. Both pointers reuse the
	// §0051 single-projection semantics: LatestDemotion is only set
	// when it targets LatestPromotion.
	if proj.LatestPromotion != nil && proj.LatestDemotion != nil {
		latency := proj.LatestDemotion.DemotedAt - proj.LatestPromotion.PromotedAt
		proj.LatestPromotionToLatestDemotionLatencyNs = &latency
	}

	// Formation → dissolution.
	if proj.Dissolution != nil {
		latency := proj.Dissolution.DissolvedAt - formationEventTime
		proj.FormationToDissolutionLatencyNs = &latency
	}
}
