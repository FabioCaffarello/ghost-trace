package decision

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/FabioCaffarello/ghost-trace/libs/wire"
)

// Mount registers the two endpoints on mux.
//
// A host mounts these rather than writing them. That is the whole point
// of the package: the collector and the decision engine differ in where
// the state comes from and in nothing a caller can see.
func (s *Service) Mount(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/decisions", s.handleDecisions)
	mux.HandleFunc("POST /v1/outcomes", s.handleOutcomes)
}

// ---------------------------------------------------------------
// POST /v1/decisions — application server
// ---------------------------------------------------------------

func (s *Service) handleDecisions(w http.ResponseWriter, r *http.Request) {
	// secret_key authenticates the application server. This is the only
	// endpoint that accepts subject_id and action, and it is the reason
	// they are never read from a browser.
	if !s.authorizedSecret(r) {
		writeError(w, http.StatusUnauthorized, "invalid secret_key")
		return
	}

	var req wire.DecisionsRequest
	if !s.decode(w, r, &req) {
		return
	}

	out, err := s.Decide(r.Context(), Input{
		SessionToken: req.SessionToken,
		Action:       req.Action,
		SubjectID:    req.SubjectID,
	})
	if err != nil {
		s.writeError(w, err, "decision failed")
		return
	}

	resp := wire.DecisionsResponse{
		EvaluationID:   out.EvaluationID,
		Decision:       out.Decision,
		ShadowDecision: out.ShadowDecision,
		Score:          round3(out.Score),
		Confidence:     round3(out.Confidence),
		Reasons:        make([]wire.DecisionReason, 0, len(out.Reasons)),
		Evidence:       wire.DecisionEvidence{Events: out.EvidenceEvents, DurationMs: out.EvidenceMs},
		Mode:           out.Mode,
	}
	for _, rs := range out.Reasons {
		resp.Reasons = append(resp.Reasons, wire.DecisionReason{Code: rs.Code, Weight: rs.Weight})
	}
	writeJSON(w, http.StatusOK, resp)
}

// ---------------------------------------------------------------
// POST /v1/outcomes — application server
// ---------------------------------------------------------------

func (s *Service) handleOutcomes(w http.ResponseWriter, r *http.Request) {
	if !s.authorizedSecret(r) {
		writeError(w, http.StatusUnauthorized, "invalid secret_key")
		return
	}

	var req wire.OutcomesRequest
	if !s.decode(w, r, &req) {
		return
	}

	in := OutcomeInput{
		EvaluationID: req.EvaluationID,
		Outcome:      req.Outcome,
	}
	if req.ObservedAt != "" {
		t, err := time.Parse(time.RFC3339, req.ObservedAt)
		if err != nil {
			// Reject rather than fall back to the server clock:
			// observed_at is the application's claim and recorded_at the
			// server's observation, and the gap between them is itself a
			// signal. Substituting "now" on a parse failure collapses
			// that gap to zero and silently corrupts the labels channel.
			writeError(w, http.StatusBadRequest, "observed_at must be RFC 3339")
			return
		}
		in.ObservedAt = t
	}

	if err := s.RecordOutcome(r.Context(), in); err != nil {
		s.writeError(w, err, "could not record outcome")
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

// ---------------------------------------------------------------
// helpers
// ---------------------------------------------------------------

func (s *Service) decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, s.maxBody)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "malformed JSON body")
		return false
	}
	return true
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if len(h) > 7 && strings.EqualFold(h[:7], "bearer ") {
		return h[7:]
	}
	return ""
}

// authorizedSecret reports whether the request carries the secret_key.
// The comparison is constant-time: the secret authenticates the
// application server on an internet-facing endpoint, and a byte-wise
// == would leak prefix length through response timing.
func (s *Service) authorizedSecret(r *http.Request) bool {
	return subtle.ConstantTimeCompare([]byte(bearerToken(r)), []byte(s.cfg.SecretKey)) == 1
}

// writeError is the single translation from this package's error
// vocabulary to HTTP.
func (s *Service) writeError(w http.ResponseWriter, err error, fallback string) {
	switch {
	case errors.Is(err, ErrActionRequired):
		writeError(w, http.StatusBadRequest, "action is required")
	case errors.Is(err, ErrEvaluationIDRequired):
		writeError(w, http.StatusBadRequest, "evaluation_id is required")
	case errors.Is(err, ErrUnknownOutcome):
		writeError(w, http.StatusBadRequest, "unknown outcome")
	case errors.Is(err, ErrArchiveUnavailable):
		// A label with nowhere durable to live is worse than a refusal:
		// the caller would believe it had reported an outcome.
		writeError(w, http.StatusServiceUnavailable, "outcome storage not configured")
	default:
		s.log.Error(fallback, "err", err)
		writeError(w, http.StatusInternalServerError, fallback)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, wire.ErrorResponse{Error: msg})
}

func round3(f float64) float64 {
	return float64(int(f*1000+0.5)) / 1000
}
