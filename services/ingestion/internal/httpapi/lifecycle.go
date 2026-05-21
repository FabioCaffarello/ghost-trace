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
	if r.Method != http.MethodPost {
		writeIngestError(w, http.StatusMethodNotAllowed, "method not allowed; POST required")
		return
	}
	if h.sub == nil {
		writeIngestError(w, http.StatusServiceUnavailable, "substrate access not configured (WithSubstrate)")
		return
	}
	if ct := r.Header.Get("Content-Type"); ct != "application/x-protobuf" {
		writeIngestError(w, http.StatusUnsupportedMediaType, fmt.Sprintf("Content-Type must be application/x-protobuf (got %q)", ct))
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, h.requestBodyLimit+1))
	if err != nil {
		writeIngestError(w, http.StatusBadRequest, fmt.Sprintf("read body: %v", err))
		return
	}
	if int64(len(body)) > h.requestBodyLimit {
		writeIngestError(w, http.StatusRequestEntityTooLarge, fmt.Sprintf("request body exceeds %d-byte limit", h.requestBodyLimit))
		return
	}

	var msg eventsv1.BehavioralClusterPromotion
	if err := proto.Unmarshal(body, &msg); err != nil {
		writeIngestError(w, http.StatusBadRequest, fmt.Sprintf("decode BehavioralClusterPromotion: %v", err))
		return
	}

	if len(msg.GetFormationEventHash()) != 32 {
		writeIngestError(w, http.StatusBadRequest, fmt.Sprintf("formation_event_hash must be 32 bytes (got %d)", len(msg.GetFormationEventHash())))
		return
	}
	if msg.GetCadenceSeconds() <= 0 {
		writeIngestError(w, http.StatusBadRequest, fmt.Sprintf("cadence_seconds must be positive (got %d)", msg.GetCadenceSeconds()))
		return
	}

	var formationHash [32]byte
	copy(formationHash[:], msg.GetFormationEventHash())

	env := envelopeForRequest(r)
	actor := resolveT4Actor(env, TierConstitutionalAct)

	opts := hypothesis.PromoteOptions{
		FormationEventHash: formationHash,
		PromotedAt:         msg.GetPromotedAt(),
		CadenceSeconds:     msg.GetCadenceSeconds(),
		Reason:             msg.GetReason(),
		Actor:              actor,
	}

	report, err := hypothesis.Promote(r.Context(), h.sub, opts, time.Now)
	if err != nil {
		switch {
		case errIs(err, hypothesis.ErrTargetNotFound):
			writeIngestError(w, http.StatusNotFound, fmt.Sprintf("formation event hash not found: %x", formationHash))
			return
		case errIs(err, hypothesis.ErrTargetWrongType):
			writeIngestError(w, http.StatusNotFound, fmt.Sprintf("formation event hash resolves to wrong type: %x", formationHash))
			return
		}
		if h.fatal != nil {
			h.fatal.ReportFatal(fmt.Errorf("T4 promote-behavioral-cluster: %w", err))
		}
		writeIngestError(w, http.StatusInternalServerError, fmt.Sprintf("promote: %v", err))
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
