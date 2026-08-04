package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"context"
)

// HTTPDecisionClient implements DecisionClient the way a real
// integrator would: over the wire, authenticated with the secret key,
// against an explicit base URL. The loopback address the demo uses in
// the all-in-one binary is a choice the composition root makes — this
// client assumes nothing about where the engine is.
type HTTPDecisionClient struct {
	base      string
	secretKey string
	http      *http.Client
}

// NewHTTPDecisionClient constructs the client. A zero timeout gets the
// contract default: §5 sets the client-side budget at roughly 3× the
// p99 target, which at an 80ms p99 is 250ms.
func NewHTTPDecisionClient(base, secretKey string, timeout time.Duration) *HTTPDecisionClient {
	if timeout == 0 {
		timeout = 250 * time.Millisecond
	}
	return &HTTPDecisionClient{
		base:      base,
		secretKey: secretKey,
		http:      &http.Client{Timeout: timeout},
	}
}

// Decide posts the prepared request body to /v1/decisions.
func (c *HTTPDecisionClient) Decide(ctx context.Context, body []byte) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.base+"/v1/decisions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.secretKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}
