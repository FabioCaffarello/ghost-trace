package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// envelopeFixture builds a §0163 envelope JSON with N AutomationGroup
// candidates of varying ActorRefs lengths. Used to exercise top-N
// ranking + cross-subtype filter behavior.
//
// Per §0215 + §0214 MO1 (wire-contract integration testing pattern):
// the field name `source_event_hashes_hex` MUST match the upstream
// CLI's emit form verbatim. Pre-§0215 this fixture used
// `source_hashes` which matched the bridge envelope struct's (also
// wrong) JSON tag — both sides shared the same drift, so tests
// passed while the cross-package wire-contract failed silently at
// the §0213 first-real-run. §0215 fixes the bridge tag AND the
// fixture field name in lockstep; in addition, the new test
// TestRun_RealWireFixture_ChainIntact loads testdata/sample-
// envelope.json (real-wire fixture per MO1) for cross-package
// boundary assertion.
func envelopeFixture(t *testing.T, actorCounts []int, crossSubtypeCount int) string {
	t.Helper()
	candidates := []map[string]any{}
	for i, n := range actorCounts {
		actors := make([]string, n)
		for j := 0; j < n; j++ {
			actors[j] = fmt.Sprintf("actor-%d-%d:%d/tcp", i, j, 1000+j)
		}
		candidates = append(candidates, map[string]any{
			"signature_name":          "tcp_flow_features_clustering_v1",
			"hypothesis_subtype":      "AutomationGroup",
			"actor_refs":              actors,
			"source_event_hashes_hex": []string{hex.EncodeToString(makeHash(t, fmt.Sprintf("src-%d", i)))},
			"evidence_count":          n,
			"confidence_hint":         0.5,
		})
	}
	for i := 0; i < crossSubtypeCount; i++ {
		candidates = append(candidates, map[string]any{
			"signature_name":          "keystroke_timing_clustering_v1",
			"hypothesis_subtype":      "BehavioralCluster",
			"actor_refs":              []string{fmt.Sprintf("bc-actor-%d", i)},
			"source_event_hashes_hex": []string{hex.EncodeToString(makeHash(t, fmt.Sprintf("bc-src-%d", i)))},
		})
	}
	env := map[string]any{
		"signature_name":  "tcp_flow_features_clustering_v1",
		"candidate_count": len(candidates),
		"candidates":      candidates,
	}
	bs, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	return string(bs)
}

// makeHash produces a 32-byte deterministic hash from a label. The
// candidate envelope encoding requires hex-encoded 32-byte values per
// §0139.
func makeHash(t *testing.T, label string) []byte {
	t.Helper()
	out := make([]byte, 32)
	copy(out, label)
	return out
}

func runWith(t *testing.T, dir string, extraArgs []string, stdin string) (int, string, string) {
	t.Helper()
	args := []string{
		"-db", filepath.Join(dir, "test.db"),
		"-blobs", filepath.Join(dir, "blobs"),
	}
	args = append(args, extraArgs...)
	var stdout, stderr bytes.Buffer
	code := run(args, strings.NewReader(stdin), &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func decodeOutput(t *testing.T, stdout string) outputPayload {
	t.Helper()
	var p outputPayload
	if err := json.NewDecoder(strings.NewReader(stdout)).Decode(&p); err != nil {
		t.Fatalf("decode output: %v\nstdout=%q", err, stdout)
	}
	return p
}

// TestRun_RealWireFixture_ChainIntact is the §0215 + §0214 MO1
// prospective application: bridge consumes a real-wire-shape
// envelope fixture (testdata/sample-envelope.json) that carries the
// upstream CLI's emit field name `source_event_hashes_hex` verbatim;
// asserts that committed formations have NON-EMPTY source_event_
// hashes (the §2.3 chain reconstruction restored from broken state).
//
// THIS IS THE ASSERTION THAT WOULD HAVE CAUGHT §0214 SURFACE A
// PRE-DEPLOYMENT. Pre-§0215 the bridge envelope struct used
// `json:"source_hashes"` (wrong field name); the upstream emits
// `source_event_hashes_hex`; mismatched decode produced silent
// null source_hashes → 10 broken-chain formations committed at
// §0213 first-real-run. This test fails on the pre-§0215 bridge
// code (source_hashes_count = 0); passes on the post-§0215 bridge.
//
// Test fixture provenance: testdata/sample-envelope.json was hand-
// crafted at §0215 to mirror the upstream find-automation-group-
// candidates-network CLI's emit JSON shape verbatim (signature_name
// + candidate_count + candidates[] with signature_name +
// hypothesis_subtype + actor_refs + source_event_hashes_hex +
// evidence_count + confidence_hint + stats{} keys exactly as the
// upstream Go struct's JSON tags). Future bridge tests for BC/CH/CR
// signatures (§0223+ scope) should follow this fixture-pattern per
// the §0214 MO1 discipline.
func TestRun_RealWireFixture_ChainIntact(t *testing.T) {
	dir := t.TempDir()
	envelopeBytes, err := os.ReadFile(filepath.Join("testdata", "sample-envelope.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	code, stdout, stderr := runWith(t, dir, []string{"-top-n", "10"}, string(envelopeBytes))
	if code != 0 {
		t.Fatalf("exit code: got %d want 0 (stderr=%q)", code, stderr)
	}
	out := decodeOutput(t, stdout)

	if out.CandidatesIngested != 3 {
		t.Errorf("CandidatesIngested: got %d want 3 (fixture carries 3 AG candidates)", out.CandidatesIngested)
	}
	if len(out.FormationsCommitted) != 3 {
		t.Fatalf("FormationsCommitted: got %d want 3", len(out.FormationsCommitted))
	}

	// THE §0215 ASSERTION — would have caught §0214 Surface A.
	// Every formation MUST carry non-empty source_event_hashes
	// reflecting the candidate's source_event_hashes_hex field
	// decoded successfully.
	for i, f := range out.FormationsCommitted {
		if f.SourceHashesCount == 0 {
			t.Errorf("formation[%d] SourceHashesCount: got 0 — §2.3 chain broken (§0214 Surface A regression; the bridge envelope JSON field name must match upstream `source_event_hashes_hex` per §0214 Finding 1 cross-CLI cravamento)", i)
		}
	}

	// Per-candidate source-hash counts from fixture: 3, 5, 1
	// (in the actor-count-descending order: 5-actor → 5 hashes;
	// 3-actor → 3 hashes; 1-actor → 1 hash).
	expectedHashCounts := []int{5, 3, 1}
	for i, want := range expectedHashCounts {
		if got := out.FormationsCommitted[i].SourceHashesCount; got != want {
			t.Errorf("formation[%d] SourceHashesCount: got %d want %d (per fixture cardinality)", i, got, want)
		}
	}
}

// TestRun_TopNRankByActorCount witnesses the §0213 default ranking
// algorithm: ActorRefs length descending + tiebreak by content-hash.
// Three AG candidates with ActorRefs counts [3, 5, 1]; top-N=2 selects
// the two largest (5, 3).
func TestRun_TopNRankByActorCount(t *testing.T) {
	dir := t.TempDir()
	envelope := envelopeFixture(t, []int{3, 5, 1}, 0)

	code, stdout, stderr := runWith(t, dir, []string{"-top-n", "2"}, envelope)
	if code != 0 {
		t.Fatalf("exit code: got %d want 0 (stderr=%q)", code, stderr)
	}
	out := decodeOutput(t, stdout)
	if out.RankingAlgorithm != "actor-count" {
		t.Errorf("RankingAlgorithm: got %q want actor-count default", out.RankingAlgorithm)
	}
	if out.CandidatesIngested != 3 {
		t.Errorf("CandidatesIngested: got %d want 3", out.CandidatesIngested)
	}
	if out.CandidatesAGEligible != 3 {
		t.Errorf("CandidatesAGEligible: got %d want 3", out.CandidatesAGEligible)
	}
	if len(out.FormationsCommitted) != 2 {
		t.Fatalf("FormationsCommitted count: got %d want 2", len(out.FormationsCommitted))
	}
	// Top-2 should be the candidates with 5 and 3 actor refs (index 1 + index 0).
	if out.FormationsCommitted[0].ActorRefsCount != 5 {
		t.Errorf("formation[0] ActorRefsCount: got %d want 5 (largest)", out.FormationsCommitted[0].ActorRefsCount)
	}
	if out.FormationsCommitted[1].ActorRefsCount != 3 {
		t.Errorf("formation[1] ActorRefsCount: got %d want 3 (second largest)", out.FormationsCommitted[1].ActorRefsCount)
	}
	if out.FormationsCommitted[0].CandidateEnvelopeIndex != 1 {
		t.Errorf("formation[0] envelope index: got %d want 1 (the 5-actor candidate)", out.FormationsCommitted[0].CandidateEnvelopeIndex)
	}
}

// TestRun_TopNExceedsAvailable witnesses the edge case: --top-n > count
// commits all available without error. Documents the contract.
func TestRun_TopNExceedsAvailable(t *testing.T) {
	dir := t.TempDir()
	envelope := envelopeFixture(t, []int{2, 4}, 0)

	code, stdout, _ := runWith(t, dir, []string{"-top-n", "10"}, envelope)
	if code != 0 {
		t.Fatalf("exit code: got %d want 0", code)
	}
	out := decodeOutput(t, stdout)
	if len(out.FormationsCommitted) != 2 {
		t.Errorf("FormationsCommitted: got %d want 2 (all available, even when top-n=10)", len(out.FormationsCommitted))
	}
}

// TestRun_CrossSubtypeSkipped witnesses §0213 cross-subtype scope:
// candidates with HypothesisSubtype != AutomationGroup are skipped and
// counted; the bridge only commits AG formations.
func TestRun_CrossSubtypeSkipped(t *testing.T) {
	dir := t.TempDir()
	// 2 AG candidates + 3 cross-subtype (BehavioralCluster).
	envelope := envelopeFixture(t, []int{2, 3}, 3)

	code, stdout, _ := runWith(t, dir, []string{"-top-n", "10"}, envelope)
	if code != 0 {
		t.Fatalf("exit code: got %d want 0", code)
	}
	out := decodeOutput(t, stdout)
	if out.CandidatesIngested != 5 {
		t.Errorf("CandidatesIngested: got %d want 5", out.CandidatesIngested)
	}
	if out.CandidatesAGEligible != 2 {
		t.Errorf("CandidatesAGEligible: got %d want 2", out.CandidatesAGEligible)
	}
	if out.CrossSubtypeSkipped != 3 {
		t.Errorf("CrossSubtypeSkipped: got %d want 3", out.CrossSubtypeSkipped)
	}
	if len(out.FormationsCommitted) != 2 {
		t.Errorf("FormationsCommitted: got %d want 2 (AG-only)", len(out.FormationsCommitted))
	}
}

// TestRun_Idempotent witnesses content-addressed immutability per §0027
// AP6: re-invoking with the SAME envelope + SAME options produces the
// SAME formation hashes; second invocation reports already_present > 0.
func TestRun_Idempotent(t *testing.T) {
	dir := t.TempDir()
	envelope := envelopeFixture(t, []int{2, 4}, 0)

	formationAt := "1716120100000000000"
	code1, stdout1, _ := runWith(t, dir, []string{"-top-n", "10", "-formation-at-ns", formationAt}, envelope)
	if code1 != 0 {
		t.Fatalf("first run exit code: got %d want 0", code1)
	}
	out1 := decodeOutput(t, stdout1)

	code2, stdout2, _ := runWith(t, dir, []string{"-top-n", "10", "-formation-at-ns", formationAt}, envelope)
	if code2 != 0 {
		t.Fatalf("second run exit code: got %d want 0", code2)
	}
	out2 := decodeOutput(t, stdout2)

	if out2.AlreadyPresentCount != 2 {
		t.Errorf("AlreadyPresentCount on second run: got %d want 2 (full idempotency)", out2.AlreadyPresentCount)
	}
	for i := range out1.FormationsCommitted {
		if out1.FormationsCommitted[i].FormationEventHashHex != out2.FormationsCommitted[i].FormationEventHashHex {
			t.Errorf("formation[%d] hash mismatch: first=%s second=%s (content-hash should be identical)",
				i, out1.FormationsCommitted[i].FormationEventHashHex, out2.FormationsCommitted[i].FormationEventHashHex)
		}
	}
}

// TestRun_RankByHash witnesses the --rank-by hash alternative (semantic-
// neutral ordering for the §0219+ sub-stat ambiguity case where
// len(ActorRefs) may proxy memberships not distinct actors).
func TestRun_RankByHash(t *testing.T) {
	dir := t.TempDir()
	envelope := envelopeFixture(t, []int{3, 5, 1}, 0)

	code, stdout, _ := runWith(t, dir, []string{"-top-n", "3", "-rank-by", "hash"}, envelope)
	if code != 0 {
		t.Fatalf("exit code: got %d want 0", code)
	}
	out := decodeOutput(t, stdout)
	if out.RankingAlgorithm != "hash" {
		t.Errorf("RankingAlgorithm: got %q want hash", out.RankingAlgorithm)
	}
	if len(out.FormationsCommitted) != 3 {
		t.Fatalf("FormationsCommitted: got %d want 3", len(out.FormationsCommitted))
	}
}

// TestRun_FileInputPath witnesses the positional-arg file-input path
// (alternative to stdin). Mirrors the cmd/ingest-cic-ids positional-arg
// pattern per §0204.
func TestRun_FileInputPath(t *testing.T) {
	dir := t.TempDir()
	envelope := envelopeFixture(t, []int{2}, 0)
	envelopePath := filepath.Join(dir, "envelope.json")
	if err := os.WriteFile(envelopePath, []byte(envelope), 0o644); err != nil {
		t.Fatalf("write envelope file: %v", err)
	}

	code, stdout, _ := runWith(t, dir, []string{"-top-n", "5", envelopePath}, "")
	if code != 0 {
		t.Fatalf("exit code: got %d want 0", code)
	}
	out := decodeOutput(t, stdout)
	if len(out.FormationsCommitted) != 1 {
		t.Errorf("FormationsCommitted: got %d want 1", len(out.FormationsCommitted))
	}
}

// TestRun_UnknownRankBy witnesses the --rank-by validation surface:
// unknown criterion exits with exitToolError.
func TestRun_UnknownRankBy(t *testing.T) {
	dir := t.TempDir()
	envelope := envelopeFixture(t, []int{2}, 0)
	code, _, stderr := runWith(t, dir, []string{"-rank-by", "bogus"}, envelope)
	if code != exitToolError {
		t.Fatalf("exit code: got %d want %d", code, exitToolError)
	}
	if !strings.Contains(stderr, "unknown --rank-by") {
		t.Errorf("stderr should explain unknown rank-by; got %q", stderr)
	}
}

// TestRun_TopNZeroRejected witnesses --top-n must be >= 1; defensive
// validation per §0205 tier-3 audibility (operator gets clear error,
// not silent zero-output).
func TestRun_TopNZeroRejected(t *testing.T) {
	dir := t.TempDir()
	envelope := envelopeFixture(t, []int{2}, 0)
	code, _, stderr := runWith(t, dir, []string{"-top-n", "0"}, envelope)
	if code != exitToolError {
		t.Fatalf("exit code: got %d want %d", code, exitToolError)
	}
	if !strings.Contains(stderr, "--top-n must be") {
		t.Errorf("stderr should explain top-n validation; got %q", stderr)
	}
}

// TestRun_OperatorOverrides witnesses --confidence-default + --ei-*
// flags reach the committed formation. Operator override is named for
// §0214 Layer B experimentation tier-3 if needed per §0213 entry.
func TestRun_OperatorOverrides(t *testing.T) {
	dir := t.TempDir()
	envelope := envelopeFixture(t, []int{2}, 0)
	code, stdout, _ := runWith(t, dir, []string{
		"-top-n", "1",
		"-confidence-default", "0.75",
		"-ei-numerator", "3",
		"-ei-denominator", "4",
		"-pattern-parameters", "min-cluster=3",
	}, envelope)
	if code != 0 {
		t.Fatalf("exit code: got %d want 0", code)
	}
	out := decodeOutput(t, stdout)
	if len(out.FormationsCommitted) != 1 {
		t.Fatalf("FormationsCommitted: got %d want 1", len(out.FormationsCommitted))
	}
	// The committed formation's content is opaque from the CLI's
	// stdout payload (would require substrate inspection); the
	// hypothesis-package test
	// TestAutomationGroupFormationFromCandidate_OperatorOverrides
	// already verifies the override path at the substrate level. This
	// test verifies the CLI accepts + forwards the flags without error
	// (the structural connection).
}
