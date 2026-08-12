package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

// Report posts the prepared label to /v1/outcomes.
//
// Unlike Decide it checks the STATUS, and that asymmetry is the point.
// A decision that comes back malformed still leaves the caller with a
// fail-open verdict it can act on. A label is only ever worth the
// storage it reached: `libs/decision` deliberately surfaces write
// failures here rather than swallowing them, because reporting success
// for a label that was not stored silently poisons the calibration loop.
// A client that then discarded the status would put the swallow back one
// layer out.
func (c *HTTPDecisionClient) Report(ctx context.Context, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.base+"/v1/outcomes", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.secretKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<10))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("outcomes: %s", resp.Status)
	}
	return nil
}
