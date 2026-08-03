package api

import (
	"encoding/json"
	"flag"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/policy"
)

var updateGolden = flag.Bool("update", false, "rewrite golden files from current output")

// TestDecisionResponseGolden freezes the /v1/decisions wire response.
//
// The upcoming restructuring moves the decision path behind use-cases
// and adapters; this file is the tripwire that proves the externally
// visible contract did not move with it. The input is deterministic
// (fixed linear pointer batches), so everything except evaluation_id —
// which is random by design and normalized here — must be byte-stable.
func TestDecisionResponseGolden(t *testing.T) {
	srv := newTestServer(t, policy.ModeMonitor)
	token := startSession(t, srv)
	for seq := 1; seq <= 3; seq++ {
		status, _ := post(t, srv, "/v1/telemetry", token, linearBatch(token, seq, 40))
		if status != http.StatusAccepted {
			t.Fatalf("telemetry = %d", status)
		}
	}

	status, body := post(t, srv, "/v1/decisions", testSecretKey, map[string]any{
		"session_token": token,
		"action":        "login",
		"subject_id":    "user_golden",
	})
	if status != http.StatusOK {
		t.Fatalf("decisions = %d, want 200", status)
	}

	if id, _ := body["evaluation_id"].(string); len(id) < 8 {
		t.Fatalf("evaluation_id looks wrong: %q", id)
	}
	body["evaluation_id"] = "ev_NORMALIZED"

	got, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got = append(got, '\n')

	goldenPath := filepath.Join("testdata", "decision_response.golden.json")
	if *updateGolden {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden (run with -update to create): %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("decision response drifted from golden.\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}
