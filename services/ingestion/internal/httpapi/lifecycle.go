// T4 constitutional-act endpoints per decision-log §0094 + §0098 +
// §0105. HTTP analogs of the Cat III lifecycle CLIs, accepting
// application/x-protobuf with canonical-serialization-contract
// enforcement matching §0034.
//
// Per RFC architecture-http-auth-scope-model item 4(a): the existing
// lifecycle event (BehavioralClusterPromotion et al.) is paired with
// an IngestionEvent via substrate.AppendPair through the hypothesis
// package's Actor-non-empty path; per-actor attribution lands on the
// IngestionEvent's identity fields. The HTTP path populates Actor
// from the verified mTLS subject when γ is active; from the bearer-
// token's tier as a fallback literal when only α is active.
package httpapi

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"google.golang.org/protobuf/proto"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/hypothesis"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
)

// promoteResponse is the wire-shape of the T4 promote endpoint's HTTP
// response body. Mirrors the structured fields the CLI emits on stdout
// (the HTTP path returns JSON; both channels carry the same field set).
type promoteResponse struct {
	// PromotionEventHash is the lowercase-hex content-hash of the
	// committed BehavioralClusterPromotion event.
	PromotionEventHash string `json:"promotion_event_hash"`

	// IngestionEventHash is the lowercase-hex content-hash of the
	// paired IngestionEvent. Always non-empty for HTTP T4 — the
	// AppendPair path is unconditional under HTTP-channel writes per
	// the auth-scope RFC's cross-tier per-actor-attribution
	// requirement.
	IngestionEventHash string `json:"ingestion_event_hash"`

	// AlreadyPromoted is true when an identical promotion event was
	// already in the substrate (idempotent commit per §0027 AP6).
	// False when this invocation committed a new row.
	AlreadyPromoted bool `json:"already_promoted"`
}

// handlePromoteBehavioralCluster implements POST
// /v1/hypotheses/behavioral-cluster/promote per decision-log §0105.
// T4 constitutional-act tier per §0094 + §0098.
//
// Request body: application/x-protobuf encoding a
// BehavioralClusterPromotion message. The four payload fields
// (formation_event_hash, promoted_at, cadence_seconds, reason) become
// the PromoteOptions parameters; the Actor field is populated from
// the verified mTLS subject (when γ active) or the per-tier fallback
// literal "unattributed-token-constitutional-act" (when only α).
//
// Response: 200 + promoteResponse JSON. 400 on body decode failure or
// invalid parameters (cadence_seconds <= 0; formation hash wrong
// length). 401 on auth (handled at ServeHTTP boundary). 404 on target
// formation not found OR wrong type (§2.5 lifecycle integrity). 405
// non-POST. 415 wrong Content-Type. 500 on substrate failure. 503
// when the handler was constructed without WithSubstrate.
func (h *Handler) handlePromoteBehavioralCluster(w http.ResponseWriter, r *http.Request) {
	var msg eventsv1.BehavioralClusterPromotion
	status, errMsg, _, ok := h.decodePromotePayload(r, &msg)
	if !ok {
		writeIngestError(w, status, errMsg)
		return
	}
	if s, m, ok := validatePromoteParams(msg.GetFormationEventHash(), msg.GetCadenceSeconds()); !ok {
		writeIngestError(w, s, m)
		return
	}

	var formationHash [32]byte
	copy(formationHash[:], msg.GetFormationEventHash())

	env := envelopeForRequest(r)
	opts := hypothesis.PromoteOptions{
		FormationEventHash: formationHash,
		PromotedAt:         msg.GetPromotedAt(),
		CadenceSeconds:     msg.GetCadenceSeconds(),
		Reason:             msg.GetReason(),
		Actor:              resolveT4Actor(env, TierConstitutionalAct),
	}

	report, err := hypothesis.Promote(r.Context(), h.sub, opts, time.Now)
	if err != nil {
		s, m := promoteErrToHTTPStatus(err, formationHash)
		if s == http.StatusInternalServerError && h.fatal != nil {
			h.fatal.ReportFatal(fmt.Errorf("T4 promote-behavioral-cluster: %w", err))
		}
		writeIngestError(w, s, m)
		return
	}

	writeJSON(w, http.StatusOK, promoteResponse{
		PromotionEventHash: report.PromotionEventHashHex,
		IngestionEventHash: report.IngestionEventHashHex,
		AlreadyPromoted:    report.AlreadyPromoted,
	})
}

// resolveT4Actor returns the per-actor attribution string for a T4
// HTTP request. Precedence per RFC architecture-http-auth-scope-model
// item 4(c):
//
//  1. Verified mTLS subject CN (when γ active — env.ClientCommonName
//     populated by envelopeForRequest).
//  2. Fallback literal "unattributed-token-<tier>" (when only α
//     active OR no client cert presented). Operationally discouraged
//     (the hypothesis package logs nothing here; structured logging
//     is the operator's responsibility); preserves §0035 single-
//     line-token-file backward compatibility per the RFC item 4(c)
//     "discouraged-but-permitted" clause.
//
// The bearer-token's `token_id` precedence rung (RFC item 4(b)) is
// not exercised at this T4 pilot landing — per-tier token files are
// currently single-line `<token>` only. A future per-tier token file
// format extension (`<token>\n<token_id>\n`) will land before multi-
// admin α deployments make the fallback literal load-bearing.
func resolveT4Actor(env ingest.Envelope, tier Tier) string {
	if env.ClientCommonName != "" {
		return env.ClientCommonName
	}
	return fmt.Sprintf("unattributed-token-%s", tier)
}

// errIs is a thin wrapper to keep the lifecycle handler's
// switch-on-error syntax readable. Returns true iff err matches target
// via errors.Is.
func errIs(err, target error) bool {
	for cur := err; cur != nil; {
		if cur == target {
			return true
		}
		un, ok := cur.(interface{ Unwrap() error })
		if !ok {
			break
		}
		cur = un.Unwrap()
	}
	return false
}

// decodePromotePayload reads + length-limits the request body and
// unmarshals it into msg under the §0034 canonical-serialization-
// contract enforcement (application/x-protobuf required; 413 when
// body exceeds requestBodyLimit). Returns a non-nil error message
// string when validation fails; the caller writes the HTTP error.
// Shared across the four per-subtype promote handlers per §0105
// pilot-then-replicate pattern.
func (h *Handler) decodePromotePayload(r *http.Request, msg proto.Message) (status int, errMsg string, body []byte, ok bool) {
	if r.Method != http.MethodPost {
		return http.StatusMethodNotAllowed, "method not allowed; POST required", nil, false
	}
	if h.sub == nil {
		return http.StatusServiceUnavailable, "substrate access not configured (WithSubstrate)", nil, false
	}
	if ct := r.Header.Get("Content-Type"); ct != "application/x-protobuf" {
		return http.StatusUnsupportedMediaType, fmt.Sprintf("Content-Type must be application/x-protobuf (got %q)", ct), nil, false
	}
	b, err := io.ReadAll(io.LimitReader(r.Body, h.requestBodyLimit+1))
	if err != nil {
		return http.StatusBadRequest, fmt.Sprintf("read body: %v", err), nil, false
	}
	if int64(len(b)) > h.requestBodyLimit {
		return http.StatusRequestEntityTooLarge, fmt.Sprintf("request body exceeds %d-byte limit", h.requestBodyLimit), nil, false
	}
	if err := proto.Unmarshal(b, msg); err != nil {
		return http.StatusBadRequest, fmt.Sprintf("decode payload: %v", err), nil, false
	}
	return 0, "", b, true
}

// validatePromoteParams enforces the shared validation gate across
// the four per-subtype promote handlers: 32-byte formation hash +
// positive cadence_seconds. Returns (status, msg, ok=false) on
// failure; otherwise (_, _, ok=true).
func validatePromoteParams(formationHash []byte, cadenceSeconds int64) (int, string, bool) {
	if len(formationHash) != 32 {
		return http.StatusBadRequest, fmt.Sprintf("formation_event_hash must be 32 bytes (got %d)", len(formationHash)), false
	}
	if cadenceSeconds <= 0 {
		return http.StatusBadRequest, fmt.Sprintf("cadence_seconds must be positive (got %d)", cadenceSeconds), false
	}
	return 0, "", true
}

// promoteErrToHTTPStatus maps a hypothesis.Promote* error to its HTTP
// status + body message. Shared across the four per-subtype promote
// handlers per §0105 pilot-then-replicate pattern.
func promoteErrToHTTPStatus(err error, formationHash [32]byte) (int, string) {
	switch {
	case errIs(err, hypothesis.ErrTargetNotFound):
		return http.StatusNotFound, fmt.Sprintf("formation event hash not found: %x", formationHash)
	case errIs(err, hypothesis.ErrTargetWrongType):
		return http.StatusNotFound, fmt.Sprintf("formation event hash resolves to wrong type: %x", formationHash)
	}
	return http.StatusInternalServerError, fmt.Sprintf("promote: %v", err)
}

// handlePromoteAutomationGroup implements POST
// /v1/hypotheses/automation-group/promote. Per §0105 pilot-then-
// replicate pattern: mechanical replication of handlePromoteBehavioral
// Cluster with AutomationGroupPromotion proto + hypothesis.Promote
// AutomationGroup helper.
func (h *Handler) handlePromoteAutomationGroup(w http.ResponseWriter, r *http.Request) {
	var msg eventsv1.AutomationGroupPromotion
	status, errMsg, _, ok := h.decodePromotePayload(r, &msg)
	if !ok {
		writeIngestError(w, status, errMsg)
		return
	}
	if s, m, ok := validatePromoteParams(msg.GetFormationEventHash(), msg.GetCadenceSeconds()); !ok {
		writeIngestError(w, s, m)
		return
	}

	var formationHash [32]byte
	copy(formationHash[:], msg.GetFormationEventHash())

	env := envelopeForRequest(r)
	opts := hypothesis.AutomationGroupPromoteOptions{
		FormationEventHash: formationHash,
		PromotedAt:         msg.GetPromotedAt(),
		CadenceSeconds:     msg.GetCadenceSeconds(),
		Reason:             msg.GetReason(),
		Actor:              resolveT4Actor(env, TierConstitutionalAct),
	}

	report, err := hypothesis.PromoteAutomationGroup(r.Context(), h.sub, opts, time.Now)
	if err != nil {
		s, m := promoteErrToHTTPStatus(err, formationHash)
		if s == http.StatusInternalServerError && h.fatal != nil {
			h.fatal.ReportFatal(fmt.Errorf("T4 promote-automation-group: %w", err))
		}
		writeIngestError(w, s, m)
		return
	}

	writeJSON(w, http.StatusOK, promoteResponse{
		PromotionEventHash: report.PromotionEventHashHex,
		IngestionEventHash: report.IngestionEventHashHex,
		AlreadyPromoted:    report.AlreadyPromoted,
	})
}

// handlePromoteCampaignHypothesis implements POST
// /v1/hypotheses/campaign-hypothesis/promote per §0105 pilot-then-
// replicate pattern.
func (h *Handler) handlePromoteCampaignHypothesis(w http.ResponseWriter, r *http.Request) {
	var msg eventsv1.CampaignHypothesisPromotion
	status, errMsg, _, ok := h.decodePromotePayload(r, &msg)
	if !ok {
		writeIngestError(w, status, errMsg)
		return
	}
	if s, m, ok := validatePromoteParams(msg.GetFormationEventHash(), msg.GetCadenceSeconds()); !ok {
		writeIngestError(w, s, m)
		return
	}

	var formationHash [32]byte
	copy(formationHash[:], msg.GetFormationEventHash())

	env := envelopeForRequest(r)
	opts := hypothesis.CampaignHypothesisPromoteOptions{
		FormationEventHash: formationHash,
		PromotedAt:         msg.GetPromotedAt(),
		CadenceSeconds:     msg.GetCadenceSeconds(),
		Reason:             msg.GetReason(),
		Actor:              resolveT4Actor(env, TierConstitutionalAct),
	}

	report, err := hypothesis.PromoteCampaignHypothesis(r.Context(), h.sub, opts, time.Now)
	if err != nil {
		s, m := promoteErrToHTTPStatus(err, formationHash)
		if s == http.StatusInternalServerError && h.fatal != nil {
			h.fatal.ReportFatal(fmt.Errorf("T4 promote-campaign-hypothesis: %w", err))
		}
		writeIngestError(w, s, m)
		return
	}

	writeJSON(w, http.StatusOK, promoteResponse{
		PromotionEventHash: report.PromotionEventHashHex,
		IngestionEventHash: report.IngestionEventHashHex,
		AlreadyPromoted:    report.AlreadyPromoted,
	})
}

// formResponse is the wire-shape of the T4 form endpoints' JSON
// response. Form is the divergent T4 op: pattern-based, not target-
// hash-based; one invocation forms zero or more hypotheses. Per
// §0111.
type formResponse struct {
	// Examined is the total DeclaredSession count walked.
	Examined int64 `json:"examined"`

	// NewlyFormed is the count of new formation events committed by
	// this invocation.
	NewlyFormed int64 `json:"newly_formed"`

	// AlreadyFormed is the count of formations whose content-hash was
	// already in the substrate (idempotent commit per §0027 AP6).
	AlreadyFormed int64 `json:"already_formed"`
}

// formCommonGate enforces the shared method + substrate gate across
// the four T4 form handlers. Form endpoints accept query parameters
// for pattern config (divergent from other T4 ops which use proto
// body) — operationally consistent with T3 orphan-cleanup.
func (h *Handler) formCommonGate(r *http.Request) (int, string, bool) {
	if r.Method != http.MethodPost {
		return http.StatusMethodNotAllowed, "method not allowed; POST required", false
	}
	if h.sub == nil {
		return http.StatusServiceUnavailable, "substrate access not configured (WithSubstrate)", false
	}
	return 0, "", true
}

// parseIntQueryParam parses an int query parameter from q.GetXxx;
// returns the default when absent. Returns an error message string +
// false on parse failure.
func parseIntQueryParam(q map[string][]string, key string, def int) (int, string, bool) {
	raw := ""
	if vs, ok := q[key]; ok && len(vs) > 0 {
		raw = vs[0]
	}
	if raw == "" {
		return def, "", true
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Sprintf("%s: %v", key, err), false
	}
	return v, "", true
}

// parseFloatQueryParam mirrors parseIntQueryParam for float64.
func parseFloatQueryParam(q map[string][]string, key string, def float64) (float64, string, bool) {
	raw := ""
	if vs, ok := q[key]; ok && len(vs) > 0 {
		raw = vs[0]
	}
	if raw == "" {
		return def, "", true
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Sprintf("%s: %v", key, err), false
	}
	return v, "", true
}

// handleFormBehavioralCluster implements POST
// /v1/hypotheses/behavioral-cluster/form per §0111. Query parameters:
//   - min_cluster_size (int, required, ≥ 2): SessionDescriptorSharedV1
//     pattern config.
func (h *Handler) handleFormBehavioralCluster(w http.ResponseWriter, r *http.Request) {
	if status, errMsg, ok := h.formCommonGate(r); !ok {
		writeIngestError(w, status, errMsg)
		return
	}
	q := r.URL.Query()
	minClusterSize, em, ok := parseIntQueryParam(q, "min_cluster_size", 0)
	if !ok {
		writeIngestError(w, http.StatusBadRequest, em)
		return
	}
	if minClusterSize < 2 {
		writeIngestError(w, http.StatusBadRequest, fmt.Sprintf("min_cluster_size must be ≥ 2 (got %d)", minClusterSize))
		return
	}
	env := envelopeForRequest(r)
	rep, err := hypothesis.FormAllWithActor(r.Context(), h.sub,
		hypothesis.SessionDescriptorSharedV1{MinClusterSize: int64(minClusterSize)},
		time.Now, resolveT4Actor(env, TierConstitutionalAct))
	if err != nil {
		if h.fatal != nil {
			h.fatal.ReportFatal(fmt.Errorf("T4 form-behavioral-cluster: %w", err))
		}
		writeIngestError(w, http.StatusInternalServerError, fmt.Sprintf("form: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, formResponse{
		Examined:      rep.Examined,
		NewlyFormed:   rep.NewlyFormed,
		AlreadyFormed: rep.AlreadyFormed,
	})
}

// handleFormAutomationGroup implements POST
// /v1/hypotheses/automation-group/form per §0111. Query parameters:
//   - min_observation_count (int, ≥ 2): UniformCadenceV1 config.
//   - max_co_v_threshold (float, ≥ 0): UniformCadenceV1 config.
func (h *Handler) handleFormAutomationGroup(w http.ResponseWriter, r *http.Request) {
	if status, errMsg, ok := h.formCommonGate(r); !ok {
		writeIngestError(w, status, errMsg)
		return
	}
	q := r.URL.Query()
	minObsCount, em, ok := parseIntQueryParam(q, "min_observation_count", 0)
	if !ok {
		writeIngestError(w, http.StatusBadRequest, em)
		return
	}
	if minObsCount < 2 {
		writeIngestError(w, http.StatusBadRequest, fmt.Sprintf("min_observation_count must be ≥ 2 (got %d)", minObsCount))
		return
	}
	maxCoV, em, ok := parseFloatQueryParam(q, "max_co_v_threshold", 0)
	if !ok {
		writeIngestError(w, http.StatusBadRequest, em)
		return
	}
	if maxCoV < 0 {
		writeIngestError(w, http.StatusBadRequest, fmt.Sprintf("max_co_v_threshold must be ≥ 0 (got %f)", maxCoV))
		return
	}
	env := envelopeForRequest(r)
	rep, err := hypothesis.FormAutomationGroupAllWithActor(r.Context(), h.sub,
		hypothesis.UniformCadenceV1{MinObservationCount: int64(minObsCount), MaxCoVThreshold: maxCoV},
		time.Now, resolveT4Actor(env, TierConstitutionalAct))
	if err != nil {
		if h.fatal != nil {
			h.fatal.ReportFatal(fmt.Errorf("T4 form-automation-group: %w", err))
		}
		writeIngestError(w, http.StatusInternalServerError, fmt.Sprintf("form: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, formResponse{
		Examined:      rep.Examined,
		NewlyFormed:   rep.NewlyFormed,
		AlreadyFormed: rep.AlreadyFormed,
	})
}

// handleFormCampaignHypothesis implements POST
// /v1/hypotheses/campaign-hypothesis/form per §0111. Query parameters:
//   - min_campaign_size (int, ≥ 2): TemporalDescriptorCohortV1 config.
//   - max_intra_event_gap_seconds (int, ≥ 0): TemporalDescriptorCohortV1
//     config.
func (h *Handler) handleFormCampaignHypothesis(w http.ResponseWriter, r *http.Request) {
	if status, errMsg, ok := h.formCommonGate(r); !ok {
		writeIngestError(w, status, errMsg)
		return
	}
	q := r.URL.Query()
	minSize, em, ok := parseIntQueryParam(q, "min_campaign_size", 0)
	if !ok {
		writeIngestError(w, http.StatusBadRequest, em)
		return
	}
	if minSize < 2 {
		writeIngestError(w, http.StatusBadRequest, fmt.Sprintf("min_campaign_size must be ≥ 2 (got %d)", minSize))
		return
	}
	maxGap, em, ok := parseIntQueryParam(q, "max_intra_event_gap_seconds", 0)
	if !ok {
		writeIngestError(w, http.StatusBadRequest, em)
		return
	}
	if maxGap < 0 {
		writeIngestError(w, http.StatusBadRequest, fmt.Sprintf("max_intra_event_gap_seconds must be ≥ 0 (got %d)", maxGap))
		return
	}
	env := envelopeForRequest(r)
	rep, err := hypothesis.FormCampaignHypothesisAllWithActor(r.Context(), h.sub,
		hypothesis.TemporalDescriptorCohortV1{MinCampaignSize: int64(minSize), MaxIntraEventGapSeconds: int64(maxGap)},
		time.Now, resolveT4Actor(env, TierConstitutionalAct))
	if err != nil {
		if h.fatal != nil {
			h.fatal.ReportFatal(fmt.Errorf("T4 form-campaign-hypothesis: %w", err))
		}
		writeIngestError(w, http.StatusInternalServerError, fmt.Sprintf("form: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, formResponse{
		Examined:      rep.Examined,
		NewlyFormed:   rep.NewlyFormed,
		AlreadyFormed: rep.AlreadyFormed,
	})
}

// handleFormCoordinationRing implements POST
// /v1/hypotheses/coordination-ring/form per §0111. Query parameters:
//   - min_edge_support (int, ≥ 2): CoOccurrenceWindowV1 config.
//   - max_window_seconds (int, ≥ 0): CoOccurrenceWindowV1 config.
func (h *Handler) handleFormCoordinationRing(w http.ResponseWriter, r *http.Request) {
	if status, errMsg, ok := h.formCommonGate(r); !ok {
		writeIngestError(w, status, errMsg)
		return
	}
	q := r.URL.Query()
	minEdgeSupport, em, ok := parseIntQueryParam(q, "min_edge_support", 0)
	if !ok {
		writeIngestError(w, http.StatusBadRequest, em)
		return
	}
	if minEdgeSupport < 2 {
		writeIngestError(w, http.StatusBadRequest, fmt.Sprintf("min_edge_support must be ≥ 2 (got %d)", minEdgeSupport))
		return
	}
	maxWindow, em, ok := parseIntQueryParam(q, "max_window_seconds", 0)
	if !ok {
		writeIngestError(w, http.StatusBadRequest, em)
		return
	}
	if maxWindow < 0 {
		writeIngestError(w, http.StatusBadRequest, fmt.Sprintf("max_window_seconds must be ≥ 0 (got %d)", maxWindow))
		return
	}
	env := envelopeForRequest(r)
	rep, err := hypothesis.FormCoordinationRingAllWithActor(r.Context(), h.sub,
		hypothesis.CoOccurrenceWindowV1{MinEdgeSupport: int64(minEdgeSupport), MaxWindowSeconds: int64(maxWindow)},
		time.Now, resolveT4Actor(env, TierConstitutionalAct))
	if err != nil {
		if h.fatal != nil {
			h.fatal.ReportFatal(fmt.Errorf("T4 form-coordination-ring: %w", err))
		}
		writeIngestError(w, http.StatusInternalServerError, fmt.Sprintf("form: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, formResponse{
		Examined:      rep.Examined,
		NewlyFormed:   rep.NewlyFormed,
		AlreadyFormed: rep.AlreadyFormed,
	})
}

// splitResponse is the wire-shape of the T4 split endpoints' JSON
// response. Split identity: antecedent hash + N successor hashes;
// response surfaces the split event hash + idempotency + ingestion
// event hash. Per §0110.
type splitResponse struct {
	SplitEventHash     string `json:"split_event_hash"`
	IngestionEventHash string `json:"ingestion_event_hash"`
	AlreadySplit       bool   `json:"already_split"`
}

func (h *Handler) decodeSplitPayload(r *http.Request, msg proto.Message) (int, string, bool) {
	if r.Method != http.MethodPost {
		return http.StatusMethodNotAllowed, "method not allowed; POST required", false
	}
	if h.sub == nil {
		return http.StatusServiceUnavailable, "substrate access not configured (WithSubstrate)", false
	}
	if ct := r.Header.Get("Content-Type"); ct != "application/x-protobuf" {
		return http.StatusUnsupportedMediaType, fmt.Sprintf("Content-Type must be application/x-protobuf (got %q)", ct), false
	}
	b, err := io.ReadAll(io.LimitReader(r.Body, h.requestBodyLimit+1))
	if err != nil {
		return http.StatusBadRequest, fmt.Sprintf("read body: %v", err), false
	}
	if int64(len(b)) > h.requestBodyLimit {
		return http.StatusRequestEntityTooLarge, fmt.Sprintf("request body exceeds %d-byte limit", h.requestBodyLimit), false
	}
	if err := proto.Unmarshal(b, msg); err != nil {
		return http.StatusBadRequest, fmt.Sprintf("decode payload: %v", err), false
	}
	return 0, "", true
}

func validateSplitParams(antecedent []byte, successors [][]byte) (int, string, [32]byte, [][32]byte, bool) {
	var a [32]byte
	if len(antecedent) != 32 {
		return http.StatusBadRequest, fmt.Sprintf("antecedent_formation_event_hash must be 32 bytes (got %d)", len(antecedent)), a, nil, false
	}
	if len(successors) < 2 {
		return http.StatusBadRequest, fmt.Sprintf("successor_formation_event_hashes must have at least 2 entries (got %d)", len(successors)), a, nil, false
	}
	out := make([][32]byte, len(successors))
	for i, succ := range successors {
		if len(succ) != 32 {
			return http.StatusBadRequest, fmt.Sprintf("successor[%d] must be 32 bytes (got %d)", i, len(succ)), a, nil, false
		}
		copy(out[i][:], succ)
	}
	copy(a[:], antecedent)
	return 0, "", a, out, true
}

func splitErrToHTTPStatus(err error) (int, string) {
	switch {
	case errIs(err, hypothesis.ErrSplitInsufficientSuccessors):
		return http.StatusBadRequest, fmt.Sprintf("split: %v", err)
	case errIs(err, hypothesis.ErrSplitSuccessorsNotDistinct):
		return http.StatusBadRequest, fmt.Sprintf("split: %v", err)
	case errIs(err, hypothesis.ErrTargetNotFound):
		return http.StatusNotFound, fmt.Sprintf("split: target not found: %v", err)
	case errIs(err, hypothesis.ErrTargetWrongType):
		return http.StatusNotFound, fmt.Sprintf("split: target wrong type: %v", err)
	}
	return http.StatusInternalServerError, fmt.Sprintf("split: %v", err)
}

func (h *Handler) handleSplitBehavioralCluster(w http.ResponseWriter, r *http.Request) {
	var msg eventsv1.BehavioralClusterSplit
	if status, errMsg, ok := h.decodeSplitPayload(r, &msg); !ok {
		writeIngestError(w, status, errMsg)
		return
	}
	s, m, a, succs, ok := validateSplitParams(msg.GetAntecedentFormationEventHash(), msg.GetSuccessorFormationEventHashes())
	if !ok {
		writeIngestError(w, s, m)
		return
	}
	env := envelopeForRequest(r)
	opts := hypothesis.SplitOptions{
		AntecedentFormationHash:  a,
		SuccessorFormationHashes: succs,
		SplitAt:                  msg.GetSplitAt(),
		Reason:                   msg.GetReason(),
		Actor:                    resolveT4Actor(env, TierConstitutionalAct),
	}
	report, err := hypothesis.Split(r.Context(), h.sub, opts, time.Now)
	if err != nil {
		st, mm := splitErrToHTTPStatus(err)
		if st == http.StatusInternalServerError && h.fatal != nil {
			h.fatal.ReportFatal(fmt.Errorf("T4 split-behavioral-cluster: %w", err))
		}
		writeIngestError(w, st, mm)
		return
	}
	writeJSON(w, http.StatusOK, splitResponse{
		SplitEventHash:     report.SplitEventHashHex,
		IngestionEventHash: report.IngestionEventHashHex,
		AlreadySplit:       report.AlreadySplit,
	})
}

func (h *Handler) handleSplitAutomationGroup(w http.ResponseWriter, r *http.Request) {
	var msg eventsv1.AutomationGroupSplit
	if status, errMsg, ok := h.decodeSplitPayload(r, &msg); !ok {
		writeIngestError(w, status, errMsg)
		return
	}
	s, m, a, succs, ok := validateSplitParams(msg.GetAntecedentFormationEventHash(), msg.GetSuccessorFormationEventHashes())
	if !ok {
		writeIngestError(w, s, m)
		return
	}
	env := envelopeForRequest(r)
	opts := hypothesis.AutomationGroupSplitOptions{
		AntecedentFormationHash:  a,
		SuccessorFormationHashes: succs,
		SplitAt:                  msg.GetSplitAt(),
		Reason:                   msg.GetReason(),
		Actor:                    resolveT4Actor(env, TierConstitutionalAct),
	}
	report, err := hypothesis.SplitAutomationGroup(r.Context(), h.sub, opts, time.Now)
	if err != nil {
		st, mm := splitErrToHTTPStatus(err)
		if st == http.StatusInternalServerError && h.fatal != nil {
			h.fatal.ReportFatal(fmt.Errorf("T4 split-automation-group: %w", err))
		}
		writeIngestError(w, st, mm)
		return
	}
	writeJSON(w, http.StatusOK, splitResponse{
		SplitEventHash:     report.SplitEventHashHex,
		IngestionEventHash: report.IngestionEventHashHex,
		AlreadySplit:       report.AlreadySplit,
	})
}

func (h *Handler) handleSplitCampaignHypothesis(w http.ResponseWriter, r *http.Request) {
	var msg eventsv1.CampaignHypothesisSplit
	if status, errMsg, ok := h.decodeSplitPayload(r, &msg); !ok {
		writeIngestError(w, status, errMsg)
		return
	}
	s, m, a, succs, ok := validateSplitParams(msg.GetAntecedentFormationEventHash(), msg.GetSuccessorFormationEventHashes())
	if !ok {
		writeIngestError(w, s, m)
		return
	}
	env := envelopeForRequest(r)
	opts := hypothesis.CampaignHypothesisSplitOptions{
		AntecedentFormationHash:  a,
		SuccessorFormationHashes: succs,
		SplitAt:                  msg.GetSplitAt(),
		Reason:                   msg.GetReason(),
		Actor:                    resolveT4Actor(env, TierConstitutionalAct),
	}
	report, err := hypothesis.SplitCampaignHypothesis(r.Context(), h.sub, opts, time.Now)
	if err != nil {
		st, mm := splitErrToHTTPStatus(err)
		if st == http.StatusInternalServerError && h.fatal != nil {
			h.fatal.ReportFatal(fmt.Errorf("T4 split-campaign-hypothesis: %w", err))
		}
		writeIngestError(w, st, mm)
		return
	}
	writeJSON(w, http.StatusOK, splitResponse{
		SplitEventHash:     report.SplitEventHashHex,
		IngestionEventHash: report.IngestionEventHashHex,
		AlreadySplit:       report.AlreadySplit,
	})
}

func (h *Handler) handleSplitCoordinationRing(w http.ResponseWriter, r *http.Request) {
	var msg eventsv1.CoordinationRingSplit
	if status, errMsg, ok := h.decodeSplitPayload(r, &msg); !ok {
		writeIngestError(w, status, errMsg)
		return
	}
	s, m, a, succs, ok := validateSplitParams(msg.GetAntecedentFormationEventHash(), msg.GetSuccessorFormationEventHashes())
	if !ok {
		writeIngestError(w, s, m)
		return
	}
	env := envelopeForRequest(r)
	opts := hypothesis.CoordinationRingSplitOptions{
		AntecedentFormationHash:  a,
		SuccessorFormationHashes: succs,
		SplitAt:                  msg.GetSplitAt(),
		Reason:                   msg.GetReason(),
		Actor:                    resolveT4Actor(env, TierConstitutionalAct),
	}
	report, err := hypothesis.SplitCoordinationRing(r.Context(), h.sub, opts, time.Now)
	if err != nil {
		st, mm := splitErrToHTTPStatus(err)
		if st == http.StatusInternalServerError && h.fatal != nil {
			h.fatal.ReportFatal(fmt.Errorf("T4 split-coordination-ring: %w", err))
		}
		writeIngestError(w, st, mm)
		return
	}
	writeJSON(w, http.StatusOK, splitResponse{
		SplitEventHash:     report.SplitEventHashHex,
		IngestionEventHash: report.IngestionEventHashHex,
		AlreadySplit:       report.AlreadySplit,
	})
}

// mergeResponse is the wire-shape of the T4 merge endpoints' JSON
// response. Merge identity carries the two antecedent hashes + the
// produced formation hash; the response surfaces the merge event hash
// + idempotency + ingestion event hash. Per §0109.
type mergeResponse struct {
	// MergeEventHash is the lowercase-hex content-hash of the
	// committed <Subtype>Merge event.
	MergeEventHash string `json:"merge_event_hash"`

	// IngestionEventHash is the lowercase-hex content-hash of the
	// paired IngestionEvent. Always non-empty for HTTP T4.
	IngestionEventHash string `json:"ingestion_event_hash"`

	// AlreadyMerged true on idempotent re-commit. Merge is symmetric
	// (ascending-sort normalization per §0049); identical antecedent
	// pairs in either order produce the same content-hash.
	AlreadyMerged bool `json:"already_merged"`
}

// decodeMergePayload reads + length-limits the request body and
// unmarshals into msg. Shared across the four T4 merge handlers.
func (h *Handler) decodeMergePayload(r *http.Request, msg proto.Message) (int, string, bool) {
	if r.Method != http.MethodPost {
		return http.StatusMethodNotAllowed, "method not allowed; POST required", false
	}
	if h.sub == nil {
		return http.StatusServiceUnavailable, "substrate access not configured (WithSubstrate)", false
	}
	if ct := r.Header.Get("Content-Type"); ct != "application/x-protobuf" {
		return http.StatusUnsupportedMediaType, fmt.Sprintf("Content-Type must be application/x-protobuf (got %q)", ct), false
	}
	b, err := io.ReadAll(io.LimitReader(r.Body, h.requestBodyLimit+1))
	if err != nil {
		return http.StatusBadRequest, fmt.Sprintf("read body: %v", err), false
	}
	if int64(len(b)) > h.requestBodyLimit {
		return http.StatusRequestEntityTooLarge, fmt.Sprintf("request body exceeds %d-byte limit", h.requestBodyLimit), false
	}
	if err := proto.Unmarshal(b, msg); err != nil {
		return http.StatusBadRequest, fmt.Sprintf("decode payload: %v", err), false
	}
	return 0, "", true
}

// validateMergeParams enforces the shared validation gate across the
// four T4 merge handlers: exactly two 32-byte antecedent hashes + one
// 32-byte produced formation hash.
func validateMergeParams(antecedents [][]byte, produced []byte) (int, string, [32]byte, [32]byte, [32]byte, bool) {
	var a, b, p [32]byte
	if len(antecedents) != 2 {
		return http.StatusBadRequest, fmt.Sprintf("antecedent_formation_event_hashes must have exactly 2 entries (got %d)", len(antecedents)), a, b, p, false
	}
	if len(antecedents[0]) != 32 {
		return http.StatusBadRequest, fmt.Sprintf("antecedent A must be 32 bytes (got %d)", len(antecedents[0])), a, b, p, false
	}
	if len(antecedents[1]) != 32 {
		return http.StatusBadRequest, fmt.Sprintf("antecedent B must be 32 bytes (got %d)", len(antecedents[1])), a, b, p, false
	}
	if len(produced) != 32 {
		return http.StatusBadRequest, fmt.Sprintf("produced_formation_event_hash must be 32 bytes (got %d)", len(produced)), a, b, p, false
	}
	copy(a[:], antecedents[0])
	copy(b[:], antecedents[1])
	copy(p[:], produced)
	return 0, "", a, b, p, true
}

// mergeErrToHTTPStatus maps a hypothesis.Merge* error to its HTTP
// status + body message. Shared across the four T4 merge handlers.
func mergeErrToHTTPStatus(err error) (int, string) {
	switch {
	case errIs(err, hypothesis.ErrMergeAntecedentsIdentical):
		return http.StatusBadRequest, "merge antecedents must differ (identical antecedent hashes)"
	case errIs(err, hypothesis.ErrTargetNotFound):
		return http.StatusNotFound, fmt.Sprintf("merge: target not found: %v", err)
	case errIs(err, hypothesis.ErrTargetWrongType):
		return http.StatusNotFound, fmt.Sprintf("merge: target wrong type: %v", err)
	}
	return http.StatusInternalServerError, fmt.Sprintf("merge: %v", err)
}

// handleMergeBehavioralCluster implements POST
// /v1/hypotheses/behavioral-cluster/merge per §0109.
func (h *Handler) handleMergeBehavioralCluster(w http.ResponseWriter, r *http.Request) {
	var msg eventsv1.BehavioralClusterMerge
	if status, errMsg, ok := h.decodeMergePayload(r, &msg); !ok {
		writeIngestError(w, status, errMsg)
		return
	}
	s, m, a, b, p, ok := validateMergeParams(msg.GetAntecedentFormationEventHashes(), msg.GetProducedFormationEventHash())
	if !ok {
		writeIngestError(w, s, m)
		return
	}
	env := envelopeForRequest(r)
	opts := hypothesis.MergeOptions{
		AntecedentAFormationHash: a,
		AntecedentBFormationHash: b,
		ProducedFormationHash:    p,
		MergedAt:                 msg.GetMergedAt(),
		Reason:                   msg.GetReason(),
		Actor:                    resolveT4Actor(env, TierConstitutionalAct),
	}
	report, err := hypothesis.Merge(r.Context(), h.sub, opts, time.Now)
	if err != nil {
		st, mm := mergeErrToHTTPStatus(err)
		if st == http.StatusInternalServerError && h.fatal != nil {
			h.fatal.ReportFatal(fmt.Errorf("T4 merge-behavioral-cluster: %w", err))
		}
		writeIngestError(w, st, mm)
		return
	}
	writeJSON(w, http.StatusOK, mergeResponse{
		MergeEventHash:     report.MergeEventHashHex,
		IngestionEventHash: report.IngestionEventHashHex,
		AlreadyMerged:      report.AlreadyMerged,
	})
}

// handleMergeAutomationGroup implements POST
// /v1/hypotheses/automation-group/merge per §0109.
func (h *Handler) handleMergeAutomationGroup(w http.ResponseWriter, r *http.Request) {
	var msg eventsv1.AutomationGroupMerge
	if status, errMsg, ok := h.decodeMergePayload(r, &msg); !ok {
		writeIngestError(w, status, errMsg)
		return
	}
	s, m, a, b, p, ok := validateMergeParams(msg.GetAntecedentFormationEventHashes(), msg.GetProducedFormationEventHash())
	if !ok {
		writeIngestError(w, s, m)
		return
	}
	env := envelopeForRequest(r)
	opts := hypothesis.AutomationGroupMergeOptions{
		AntecedentAFormationHash: a,
		AntecedentBFormationHash: b,
		ProducedFormationHash:    p,
		MergedAt:                 msg.GetMergedAt(),
		Reason:                   msg.GetReason(),
		Actor:                    resolveT4Actor(env, TierConstitutionalAct),
	}
	report, err := hypothesis.MergeAutomationGroup(r.Context(), h.sub, opts, time.Now)
	if err != nil {
		st, mm := mergeErrToHTTPStatus(err)
		if st == http.StatusInternalServerError && h.fatal != nil {
			h.fatal.ReportFatal(fmt.Errorf("T4 merge-automation-group: %w", err))
		}
		writeIngestError(w, st, mm)
		return
	}
	writeJSON(w, http.StatusOK, mergeResponse{
		MergeEventHash:     report.MergeEventHashHex,
		IngestionEventHash: report.IngestionEventHashHex,
		AlreadyMerged:      report.AlreadyMerged,
	})
}

// handleMergeCampaignHypothesis implements POST
// /v1/hypotheses/campaign-hypothesis/merge per §0109.
func (h *Handler) handleMergeCampaignHypothesis(w http.ResponseWriter, r *http.Request) {
	var msg eventsv1.CampaignHypothesisMerge
	if status, errMsg, ok := h.decodeMergePayload(r, &msg); !ok {
		writeIngestError(w, status, errMsg)
		return
	}
	s, m, a, b, p, ok := validateMergeParams(msg.GetAntecedentFormationEventHashes(), msg.GetProducedFormationEventHash())
	if !ok {
		writeIngestError(w, s, m)
		return
	}
	env := envelopeForRequest(r)
	opts := hypothesis.CampaignHypothesisMergeOptions{
		AntecedentAFormationHash: a,
		AntecedentBFormationHash: b,
		ProducedFormationHash:    p,
		MergedAt:                 msg.GetMergedAt(),
		Reason:                   msg.GetReason(),
		Actor:                    resolveT4Actor(env, TierConstitutionalAct),
	}
	report, err := hypothesis.MergeCampaignHypothesis(r.Context(), h.sub, opts, time.Now)
	if err != nil {
		st, mm := mergeErrToHTTPStatus(err)
		if st == http.StatusInternalServerError && h.fatal != nil {
			h.fatal.ReportFatal(fmt.Errorf("T4 merge-campaign-hypothesis: %w", err))
		}
		writeIngestError(w, st, mm)
		return
	}
	writeJSON(w, http.StatusOK, mergeResponse{
		MergeEventHash:     report.MergeEventHashHex,
		IngestionEventHash: report.IngestionEventHashHex,
		AlreadyMerged:      report.AlreadyMerged,
	})
}

// handleMergeCoordinationRing implements POST
// /v1/hypotheses/coordination-ring/merge per §0109.
func (h *Handler) handleMergeCoordinationRing(w http.ResponseWriter, r *http.Request) {
	var msg eventsv1.CoordinationRingMerge
	if status, errMsg, ok := h.decodeMergePayload(r, &msg); !ok {
		writeIngestError(w, status, errMsg)
		return
	}
	s, m, a, b, p, ok := validateMergeParams(msg.GetAntecedentFormationEventHashes(), msg.GetProducedFormationEventHash())
	if !ok {
		writeIngestError(w, s, m)
		return
	}
	env := envelopeForRequest(r)
	opts := hypothesis.CoordinationRingMergeOptions{
		AntecedentAFormationHash: a,
		AntecedentBFormationHash: b,
		ProducedFormationHash:    p,
		MergedAt:                 msg.GetMergedAt(),
		Reason:                   msg.GetReason(),
		Actor:                    resolveT4Actor(env, TierConstitutionalAct),
	}
	report, err := hypothesis.MergeCoordinationRing(r.Context(), h.sub, opts, time.Now)
	if err != nil {
		st, mm := mergeErrToHTTPStatus(err)
		if st == http.StatusInternalServerError && h.fatal != nil {
			h.fatal.ReportFatal(fmt.Errorf("T4 merge-coordination-ring: %w", err))
		}
		writeIngestError(w, st, mm)
		return
	}
	writeJSON(w, http.StatusOK, mergeResponse{
		MergeEventHash:     report.MergeEventHashHex,
		IngestionEventHash: report.IngestionEventHashHex,
		AlreadyMerged:      report.AlreadyMerged,
	})
}

// dissolveResponse is the wire-shape of the T4 dissolve endpoints'
// JSON response. Per §0108 T4 dissolve landing — dissolution targets
// the formation directly (no cadence-gate state to surface, unlike
// demote).
type dissolveResponse struct {
	// DissolutionEventHash is the lowercase-hex content-hash of the
	// committed <Subtype>Dissolution event.
	DissolutionEventHash string `json:"dissolution_event_hash"`

	// IngestionEventHash is the lowercase-hex content-hash of the
	// paired IngestionEvent. Always non-empty for HTTP T4.
	IngestionEventHash string `json:"ingestion_event_hash"`

	// AlreadyDissolved true on idempotent re-commit.
	AlreadyDissolved bool `json:"already_dissolved"`
}

// decodeDissolvePayload is the dissolve analog of decodePromote
// Payload / decodeDemotePayload.
func (h *Handler) decodeDissolvePayload(r *http.Request, msg proto.Message) (int, string, bool) {
	if r.Method != http.MethodPost {
		return http.StatusMethodNotAllowed, "method not allowed; POST required", false
	}
	if h.sub == nil {
		return http.StatusServiceUnavailable, "substrate access not configured (WithSubstrate)", false
	}
	if ct := r.Header.Get("Content-Type"); ct != "application/x-protobuf" {
		return http.StatusUnsupportedMediaType, fmt.Sprintf("Content-Type must be application/x-protobuf (got %q)", ct), false
	}
	b, err := io.ReadAll(io.LimitReader(r.Body, h.requestBodyLimit+1))
	if err != nil {
		return http.StatusBadRequest, fmt.Sprintf("read body: %v", err), false
	}
	if int64(len(b)) > h.requestBodyLimit {
		return http.StatusRequestEntityTooLarge, fmt.Sprintf("request body exceeds %d-byte limit", h.requestBodyLimit), false
	}
	if err := proto.Unmarshal(b, msg); err != nil {
		return http.StatusBadRequest, fmt.Sprintf("decode payload: %v", err), false
	}
	return 0, "", true
}

// validateDissolveParams enforces the shared validation gate across
// the four T4 dissolve handlers: 32-byte formation_event_hash.
func validateDissolveParams(formationHash []byte) (int, string, bool) {
	if len(formationHash) != 32 {
		return http.StatusBadRequest, fmt.Sprintf("formation_event_hash must be 32 bytes (got %d)", len(formationHash)), false
	}
	return 0, "", true
}

// dissolveErrToHTTPStatus maps a hypothesis.Dissolve* error to its
// HTTP status + body message.
func dissolveErrToHTTPStatus(err error, formationHash [32]byte) (int, string) {
	switch {
	case errIs(err, hypothesis.ErrTargetNotFound):
		return http.StatusNotFound, fmt.Sprintf("formation event hash not found: %x", formationHash)
	case errIs(err, hypothesis.ErrTargetWrongType):
		return http.StatusNotFound, fmt.Sprintf("formation event hash resolves to wrong type: %x", formationHash)
	}
	return http.StatusInternalServerError, fmt.Sprintf("dissolve: %v", err)
}

// handleDissolveBehavioralCluster implements POST
// /v1/hypotheses/behavioral-cluster/dissolve per §0108.
func (h *Handler) handleDissolveBehavioralCluster(w http.ResponseWriter, r *http.Request) {
	var msg eventsv1.BehavioralClusterDissolution
	if status, errMsg, ok := h.decodeDissolvePayload(r, &msg); !ok {
		writeIngestError(w, status, errMsg)
		return
	}
	if s, m, ok := validateDissolveParams(msg.GetFormationEventHash()); !ok {
		writeIngestError(w, s, m)
		return
	}
	var formationHash [32]byte
	copy(formationHash[:], msg.GetFormationEventHash())

	env := envelopeForRequest(r)
	opts := hypothesis.DissolveOptions{
		FormationEventHash: formationHash,
		DissolvedAt:        msg.GetDissolvedAt(),
		Reason:             msg.GetReason(),
		Actor:              resolveT4Actor(env, TierConstitutionalAct),
	}
	report, err := hypothesis.Dissolve(r.Context(), h.sub, opts, time.Now)
	if err != nil {
		s, m := dissolveErrToHTTPStatus(err, formationHash)
		if s == http.StatusInternalServerError && h.fatal != nil {
			h.fatal.ReportFatal(fmt.Errorf("T4 dissolve-behavioral-cluster: %w", err))
		}
		writeIngestError(w, s, m)
		return
	}
	writeJSON(w, http.StatusOK, dissolveResponse{
		DissolutionEventHash: report.DissolutionEventHashHex,
		IngestionEventHash:   report.IngestionEventHashHex,
		AlreadyDissolved:     report.AlreadyDissolved,
	})
}

// handleDissolveAutomationGroup implements POST
// /v1/hypotheses/automation-group/dissolve per §0108.
func (h *Handler) handleDissolveAutomationGroup(w http.ResponseWriter, r *http.Request) {
	var msg eventsv1.AutomationGroupDissolution
	if status, errMsg, ok := h.decodeDissolvePayload(r, &msg); !ok {
		writeIngestError(w, status, errMsg)
		return
	}
	if s, m, ok := validateDissolveParams(msg.GetFormationEventHash()); !ok {
		writeIngestError(w, s, m)
		return
	}
	var formationHash [32]byte
	copy(formationHash[:], msg.GetFormationEventHash())

	env := envelopeForRequest(r)
	opts := hypothesis.AutomationGroupDissolveOptions{
		FormationEventHash: formationHash,
		DissolvedAt:        msg.GetDissolvedAt(),
		Reason:             msg.GetReason(),
		Actor:              resolveT4Actor(env, TierConstitutionalAct),
	}
	report, err := hypothesis.DissolveAutomationGroup(r.Context(), h.sub, opts, time.Now)
	if err != nil {
		s, m := dissolveErrToHTTPStatus(err, formationHash)
		if s == http.StatusInternalServerError && h.fatal != nil {
			h.fatal.ReportFatal(fmt.Errorf("T4 dissolve-automation-group: %w", err))
		}
		writeIngestError(w, s, m)
		return
	}
	writeJSON(w, http.StatusOK, dissolveResponse{
		DissolutionEventHash: report.DissolutionEventHashHex,
		IngestionEventHash:   report.IngestionEventHashHex,
		AlreadyDissolved:     report.AlreadyDissolved,
	})
}

// handleDissolveCampaignHypothesis implements POST
// /v1/hypotheses/campaign-hypothesis/dissolve per §0108.
func (h *Handler) handleDissolveCampaignHypothesis(w http.ResponseWriter, r *http.Request) {
	var msg eventsv1.CampaignHypothesisDissolution
	if status, errMsg, ok := h.decodeDissolvePayload(r, &msg); !ok {
		writeIngestError(w, status, errMsg)
		return
	}
	if s, m, ok := validateDissolveParams(msg.GetFormationEventHash()); !ok {
		writeIngestError(w, s, m)
		return
	}
	var formationHash [32]byte
	copy(formationHash[:], msg.GetFormationEventHash())

	env := envelopeForRequest(r)
	opts := hypothesis.CampaignHypothesisDissolveOptions{
		FormationEventHash: formationHash,
		DissolvedAt:        msg.GetDissolvedAt(),
		Reason:             msg.GetReason(),
		Actor:              resolveT4Actor(env, TierConstitutionalAct),
	}
	report, err := hypothesis.DissolveCampaignHypothesis(r.Context(), h.sub, opts, time.Now)
	if err != nil {
		s, m := dissolveErrToHTTPStatus(err, formationHash)
		if s == http.StatusInternalServerError && h.fatal != nil {
			h.fatal.ReportFatal(fmt.Errorf("T4 dissolve-campaign-hypothesis: %w", err))
		}
		writeIngestError(w, s, m)
		return
	}
	writeJSON(w, http.StatusOK, dissolveResponse{
		DissolutionEventHash: report.DissolutionEventHashHex,
		IngestionEventHash:   report.IngestionEventHashHex,
		AlreadyDissolved:     report.AlreadyDissolved,
	})
}

// handleDissolveCoordinationRing implements POST
// /v1/hypotheses/coordination-ring/dissolve per §0108.
func (h *Handler) handleDissolveCoordinationRing(w http.ResponseWriter, r *http.Request) {
	var msg eventsv1.CoordinationRingDissolution
	if status, errMsg, ok := h.decodeDissolvePayload(r, &msg); !ok {
		writeIngestError(w, status, errMsg)
		return
	}
	if s, m, ok := validateDissolveParams(msg.GetFormationEventHash()); !ok {
		writeIngestError(w, s, m)
		return
	}
	var formationHash [32]byte
	copy(formationHash[:], msg.GetFormationEventHash())

	env := envelopeForRequest(r)
	opts := hypothesis.CoordinationRingDissolveOptions{
		FormationEventHash: formationHash,
		DissolvedAt:        msg.GetDissolvedAt(),
		Reason:             msg.GetReason(),
		Actor:              resolveT4Actor(env, TierConstitutionalAct),
	}
	report, err := hypothesis.DissolveCoordinationRing(r.Context(), h.sub, opts, time.Now)
	if err != nil {
		s, m := dissolveErrToHTTPStatus(err, formationHash)
		if s == http.StatusInternalServerError && h.fatal != nil {
			h.fatal.ReportFatal(fmt.Errorf("T4 dissolve-coordination-ring: %w", err))
		}
		writeIngestError(w, s, m)
		return
	}
	writeJSON(w, http.StatusOK, dissolveResponse{
		DissolutionEventHash: report.DissolutionEventHashHex,
		IngestionEventHash:   report.IngestionEventHashHex,
		AlreadyDissolved:     report.AlreadyDissolved,
	})
}

// demoteResponse is the wire-shape of the T4 demote endpoints' JSON
// response. Cadence fields surface §0011 Layer A gate state for
// operator-facing reporting (Layer A is CANDIDACY, not hard barrier).
type demoteResponse struct {
	// DemotionEventHash is the lowercase-hex content-hash of the
	// committed <Subtype>Demotion event.
	DemotionEventHash string `json:"demotion_event_hash"`

	// IngestionEventHash is the lowercase-hex content-hash of the
	// paired IngestionEvent. Always non-empty for HTTP T4.
	IngestionEventHash string `json:"ingestion_event_hash"`

	// AlreadyDemoted true on idempotent re-commit.
	AlreadyDemoted bool `json:"already_demoted"`

	// CadenceSatisfied reports whether Layer A cadence had elapsed
	// at demoted_at. Per §0011 CANDIDACY criterion (not hard barrier);
	// surfaced for operator-facing reporting.
	CadenceSatisfied bool `json:"cadence_satisfied"`

	// CadenceElapsedSeconds reports how many seconds elapsed between
	// the source promotion's promoted_at and this demotion's
	// demoted_at.
	CadenceElapsedSeconds int64 `json:"cadence_elapsed_seconds"`
}

// decodeDemotePayload is the demote analog of decodePromotePayload —
// shared body decode + length cap + Content-Type check for the four
// T4 demote handlers per §0107.
func (h *Handler) decodeDemotePayload(r *http.Request, msg proto.Message) (status int, errMsg string, ok bool) {
	if r.Method != http.MethodPost {
		return http.StatusMethodNotAllowed, "method not allowed; POST required", false
	}
	if h.sub == nil {
		return http.StatusServiceUnavailable, "substrate access not configured (WithSubstrate)", false
	}
	if ct := r.Header.Get("Content-Type"); ct != "application/x-protobuf" {
		return http.StatusUnsupportedMediaType, fmt.Sprintf("Content-Type must be application/x-protobuf (got %q)", ct), false
	}
	b, err := io.ReadAll(io.LimitReader(r.Body, h.requestBodyLimit+1))
	if err != nil {
		return http.StatusBadRequest, fmt.Sprintf("read body: %v", err), false
	}
	if int64(len(b)) > h.requestBodyLimit {
		return http.StatusRequestEntityTooLarge, fmt.Sprintf("request body exceeds %d-byte limit", h.requestBodyLimit), false
	}
	if err := proto.Unmarshal(b, msg); err != nil {
		return http.StatusBadRequest, fmt.Sprintf("decode payload: %v", err), false
	}
	return 0, "", true
}

// validateDemoteParams enforces the shared validation gate across the
// four T4 demote handlers: 32-byte promotion_event_hash.
func validateDemoteParams(promotionHash []byte) (int, string, bool) {
	if len(promotionHash) != 32 {
		return http.StatusBadRequest, fmt.Sprintf("promotion_event_hash must be 32 bytes (got %d)", len(promotionHash)), false
	}
	return 0, "", true
}

// demoteErrToHTTPStatus maps a hypothesis.Demote* error to its HTTP
// status + body message. Shared across the four T4 demote handlers.
func demoteErrToHTTPStatus(err error, promotionHash [32]byte) (int, string) {
	switch {
	case errIs(err, hypothesis.ErrTargetNotFound):
		return http.StatusNotFound, fmt.Sprintf("promotion event hash not found: %x", promotionHash)
	case errIs(err, hypothesis.ErrTargetWrongType):
		return http.StatusNotFound, fmt.Sprintf("promotion event hash resolves to wrong type: %x", promotionHash)
	}
	return http.StatusInternalServerError, fmt.Sprintf("demote: %v", err)
}

// handleDemoteBehavioralCluster implements POST
// /v1/hypotheses/behavioral-cluster/demote per §0107.
func (h *Handler) handleDemoteBehavioralCluster(w http.ResponseWriter, r *http.Request) {
	var msg eventsv1.BehavioralClusterDemotion
	status, errMsg, ok := h.decodeDemotePayload(r, &msg)
	if !ok {
		writeIngestError(w, status, errMsg)
		return
	}
	if s, m, ok := validateDemoteParams(msg.GetPromotionEventHash()); !ok {
		writeIngestError(w, s, m)
		return
	}
	var promotionHash [32]byte
	copy(promotionHash[:], msg.GetPromotionEventHash())

	env := envelopeForRequest(r)
	opts := hypothesis.DemoteOptions{
		PromotionEventHash: promotionHash,
		DemotedAt:          msg.GetDemotedAt(),
		Reason:             msg.GetReason(),
		Actor:              resolveT4Actor(env, TierConstitutionalAct),
	}
	report, err := hypothesis.Demote(r.Context(), h.sub, opts, time.Now)
	if err != nil {
		s, m := demoteErrToHTTPStatus(err, promotionHash)
		if s == http.StatusInternalServerError && h.fatal != nil {
			h.fatal.ReportFatal(fmt.Errorf("T4 demote-behavioral-cluster: %w", err))
		}
		writeIngestError(w, s, m)
		return
	}
	writeJSON(w, http.StatusOK, demoteResponse{
		DemotionEventHash:     report.DemotionEventHashHex,
		IngestionEventHash:    report.IngestionEventHashHex,
		AlreadyDemoted:        report.AlreadyDemoted,
		CadenceSatisfied:      report.CadenceSatisfied,
		CadenceElapsedSeconds: report.CadenceElapsedSeconds,
	})
}

// handleDemoteAutomationGroup implements POST
// /v1/hypotheses/automation-group/demote per §0107.
func (h *Handler) handleDemoteAutomationGroup(w http.ResponseWriter, r *http.Request) {
	var msg eventsv1.AutomationGroupDemotion
	status, errMsg, ok := h.decodeDemotePayload(r, &msg)
	if !ok {
		writeIngestError(w, status, errMsg)
		return
	}
	if s, m, ok := validateDemoteParams(msg.GetPromotionEventHash()); !ok {
		writeIngestError(w, s, m)
		return
	}
	var promotionHash [32]byte
	copy(promotionHash[:], msg.GetPromotionEventHash())

	env := envelopeForRequest(r)
	opts := hypothesis.AutomationGroupDemoteOptions{
		PromotionEventHash: promotionHash,
		DemotedAt:          msg.GetDemotedAt(),
		Reason:             msg.GetReason(),
		Actor:              resolveT4Actor(env, TierConstitutionalAct),
	}
	report, err := hypothesis.DemoteAutomationGroup(r.Context(), h.sub, opts, time.Now)
	if err != nil {
		s, m := demoteErrToHTTPStatus(err, promotionHash)
		if s == http.StatusInternalServerError && h.fatal != nil {
			h.fatal.ReportFatal(fmt.Errorf("T4 demote-automation-group: %w", err))
		}
		writeIngestError(w, s, m)
		return
	}
	writeJSON(w, http.StatusOK, demoteResponse{
		DemotionEventHash:     report.DemotionEventHashHex,
		IngestionEventHash:    report.IngestionEventHashHex,
		AlreadyDemoted:        report.AlreadyDemoted,
		CadenceSatisfied:      report.CadenceSatisfied,
		CadenceElapsedSeconds: report.CadenceElapsedSeconds,
	})
}

// handleDemoteCampaignHypothesis implements POST
// /v1/hypotheses/campaign-hypothesis/demote per §0107.
func (h *Handler) handleDemoteCampaignHypothesis(w http.ResponseWriter, r *http.Request) {
	var msg eventsv1.CampaignHypothesisDemotion
	status, errMsg, ok := h.decodeDemotePayload(r, &msg)
	if !ok {
		writeIngestError(w, status, errMsg)
		return
	}
	if s, m, ok := validateDemoteParams(msg.GetPromotionEventHash()); !ok {
		writeIngestError(w, s, m)
		return
	}
	var promotionHash [32]byte
	copy(promotionHash[:], msg.GetPromotionEventHash())

	env := envelopeForRequest(r)
	opts := hypothesis.CampaignHypothesisDemoteOptions{
		PromotionEventHash: promotionHash,
		DemotedAt:          msg.GetDemotedAt(),
		Reason:             msg.GetReason(),
		Actor:              resolveT4Actor(env, TierConstitutionalAct),
	}
	report, err := hypothesis.DemoteCampaignHypothesis(r.Context(), h.sub, opts, time.Now)
	if err != nil {
		s, m := demoteErrToHTTPStatus(err, promotionHash)
		if s == http.StatusInternalServerError && h.fatal != nil {
			h.fatal.ReportFatal(fmt.Errorf("T4 demote-campaign-hypothesis: %w", err))
		}
		writeIngestError(w, s, m)
		return
	}
	writeJSON(w, http.StatusOK, demoteResponse{
		DemotionEventHash:     report.DemotionEventHashHex,
		IngestionEventHash:    report.IngestionEventHashHex,
		AlreadyDemoted:        report.AlreadyDemoted,
		CadenceSatisfied:      report.CadenceSatisfied,
		CadenceElapsedSeconds: report.CadenceElapsedSeconds,
	})
}

// handleDemoteCoordinationRing implements POST
// /v1/hypotheses/coordination-ring/demote per §0107.
func (h *Handler) handleDemoteCoordinationRing(w http.ResponseWriter, r *http.Request) {
	var msg eventsv1.CoordinationRingDemotion
	status, errMsg, ok := h.decodeDemotePayload(r, &msg)
	if !ok {
		writeIngestError(w, status, errMsg)
		return
	}
	if s, m, ok := validateDemoteParams(msg.GetPromotionEventHash()); !ok {
		writeIngestError(w, s, m)
		return
	}
	var promotionHash [32]byte
	copy(promotionHash[:], msg.GetPromotionEventHash())

	env := envelopeForRequest(r)
	opts := hypothesis.CoordinationRingDemoteOptions{
		PromotionEventHash: promotionHash,
		DemotedAt:          msg.GetDemotedAt(),
		Reason:             msg.GetReason(),
		Actor:              resolveT4Actor(env, TierConstitutionalAct),
	}
	report, err := hypothesis.DemoteCoordinationRing(r.Context(), h.sub, opts, time.Now)
	if err != nil {
		s, m := demoteErrToHTTPStatus(err, promotionHash)
		if s == http.StatusInternalServerError && h.fatal != nil {
			h.fatal.ReportFatal(fmt.Errorf("T4 demote-coordination-ring: %w", err))
		}
		writeIngestError(w, s, m)
		return
	}
	writeJSON(w, http.StatusOK, demoteResponse{
		DemotionEventHash:     report.DemotionEventHashHex,
		IngestionEventHash:    report.IngestionEventHashHex,
		AlreadyDemoted:        report.AlreadyDemoted,
		CadenceSatisfied:      report.CadenceSatisfied,
		CadenceElapsedSeconds: report.CadenceElapsedSeconds,
	})
}

// handlePromoteCoordinationRing implements POST
// /v1/hypotheses/coordination-ring/promote per §0105 pilot-then-
// replicate pattern.
func (h *Handler) handlePromoteCoordinationRing(w http.ResponseWriter, r *http.Request) {
	var msg eventsv1.CoordinationRingPromotion
	status, errMsg, _, ok := h.decodePromotePayload(r, &msg)
	if !ok {
		writeIngestError(w, status, errMsg)
		return
	}
	if s, m, ok := validatePromoteParams(msg.GetFormationEventHash(), msg.GetCadenceSeconds()); !ok {
		writeIngestError(w, s, m)
		return
	}

	var formationHash [32]byte
	copy(formationHash[:], msg.GetFormationEventHash())

	env := envelopeForRequest(r)
	opts := hypothesis.CoordinationRingPromoteOptions{
		FormationEventHash: formationHash,
		PromotedAt:         msg.GetPromotedAt(),
		CadenceSeconds:     msg.GetCadenceSeconds(),
		Reason:             msg.GetReason(),
		Actor:              resolveT4Actor(env, TierConstitutionalAct),
	}

	report, err := hypothesis.PromoteCoordinationRing(r.Context(), h.sub, opts, time.Now)
	if err != nil {
		s, m := promoteErrToHTTPStatus(err, formationHash)
		if s == http.StatusInternalServerError && h.fatal != nil {
			h.fatal.ReportFatal(fmt.Errorf("T4 promote-coordination-ring: %w", err))
		}
		writeIngestError(w, s, m)
		return
	}

	writeJSON(w, http.StatusOK, promoteResponse{
		PromotionEventHash: report.PromotionEventHashHex,
		IngestionEventHash: report.IngestionEventHashHex,
		AlreadyPromoted:    report.AlreadyPromoted,
	})
}
