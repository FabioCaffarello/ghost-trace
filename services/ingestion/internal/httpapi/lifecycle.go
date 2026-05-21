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
