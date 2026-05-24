// CLI-level test exercising the measure-chain-morphology emission
// surface per §0159: builds a substrate with a known multi-formation
// shape, runs morphology.Measure, invokes emitJSON, decodes the JSON
// output, and verifies the envelope shape + values. Symmetric coverage
// to find-automation-group-candidates' main_test.go TestFullPipeline.
//
// Scope per §0159: CLI emission contract (JSON shape, field values).
// Per-formation morphology correctness is covered by §0158
// (TestMorphology_AgainstF3DerivedFormationChain); aggregate
// histogram math is covered by the morphology package unit tests.
// This test validates only the CLI's emission surface — the layer
// between morphology.Measurement (typed) and operator-readable JSON.
package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	commonv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/common/v1"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/morphology"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

func newCLITestSubstrate(t *testing.T) *substrate.Substrate {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })
	return sub
}

// sortHashListAscending sorts a hash-list per §0139 hash-list element-
// shape discipline (ascending lex). Required for substrate commit
// when multi-element hash lists are present.
func sortHashListAscending(hashes [][]byte) [][]byte {
	out := make([][]byte, len(hashes))
	copy(out, hashes)
	sort.Slice(out, func(i, j int) bool { return bytes.Compare(out[i], out[j]) < 0 })
	return out
}

// commitTestFormation commits a minimal AutomationGroupFormation to
// the substrate with the given chain references. Hash lists are
// auto-sorted per §0139. Returns the committed hash for chain
// references in subsequent commits.
func commitTestFormation(t *testing.T, sub *substrate.Substrate, actorRefs []string, formationAt int64, directInfluencedBy [][]byte, closureHashes [][]byte) [32]byte {
	t.Helper()
	ctx := context.Background()

	formation := &eventsv1.AutomationGroupFormation{
		PatternSignature:   "test-cli-sig-v1",
		PatternParameters:  "{}",
		ActorRefs:          actorRefs,
		FormationAt:        formationAt,
		SourceEventHashes:  nil,
		DirectInfluencedBy: sortHashListAscending(directInfluencedBy),
		ClosureHashes:      sortHashListAscending(closureHashes),
		Confidence:         0.5,
		EvidentialIndependence: &commonv1.EvidentialIndependence{
			Numerator:   1,
			Denominator: 1,
		},
	}

	payload, hash, err := canonical.MarshalAndHash(formation)
	if err != nil {
		t.Fatalf("MarshalAndHash: %v", err)
	}
	hx := canonical.HashHex(hash)
	row := substrate.EventRow{
		EventHash:   hash,
		EventTime:   formationAt,
		MessageType: string(formation.ProtoReflect().Descriptor().FullName()),
		PayloadRef:  hx[:2] + "/" + hx[2:],
		CommittedAt: formationAt,
	}
	if err := sub.Append(ctx, row, payload); err != nil {
		t.Fatalf("substrate.Append: %v", err)
	}
	return hash
}

// TestEmitJSON_FromMeasurement exercises emitJSON's contract: build a
// substrate with a 2-formation chain (formation_a root + formation_b
// influenced by a), run morphology.Measure, pass to emitJSON, decode
// the JSON output, verify envelope + hypothesis + stats shape and
// values.
func TestEmitJSON_FromMeasurement(t *testing.T) {
	sub := newCLITestSubstrate(t)
	ctx := context.Background()

	hashA := commitTestFormation(t, sub, []string{"actor-a"}, 1716120100000000000, nil, nil)
	hashB := commitTestFormation(t, sub, []string{"actor-b"}, 1716120200000000000, [][]byte{hashA[:]}, [][]byte{hashA[:]})

	m, err := morphology.Measure(ctx, sub)
	if err != nil {
		t.Fatalf("morphology.Measure: %v", err)
	}

	tmp, err := os.CreateTemp("", "measure-chain-morphology-*.json")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer os.Remove(tmp.Name())
	if err := emitJSON(tmp, m); err != nil {
		t.Fatalf("emitJSON: %v", err)
	}
	tmp.Close()

	raw, err := os.ReadFile(tmp.Name())
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var got emissionEnvelope
	if err := json.NewDecoder(bytes.NewReader(raw)).Decode(&got); err != nil {
		t.Fatalf("Decode: %v", err)
	}

	// Envelope-level checks.
	if len(got.Hypotheses) != 2 {
		t.Fatalf("Hypotheses count: got %d want 2", len(got.Hypotheses))
	}
	if got.Stats.TotalFormations != 2 {
		t.Errorf("Stats.TotalFormations: got %d want 2", got.Stats.TotalFormations)
	}
	if got.Stats.PerSubtype["ghosttrace.events.v1.AutomationGroupFormation"] != 2 {
		t.Errorf("Stats.PerSubtype[AutomationGroupFormation]: got %d want 2", got.Stats.PerSubtype["ghosttrace.events.v1.AutomationGroupFormation"])
	}
	// Both formations are chains-fracas (depth_max <= 2 OR breadth <= 3).
	if got.Stats.ChainsFracasCount != 2 {
		t.Errorf("Stats.ChainsFracasCount: got %d want 2", got.Stats.ChainsFracasCount)
	}
	if got.Stats.ChainsFortesCount != 0 {
		t.Errorf("Stats.ChainsFortesCount: got %d want 0", got.Stats.ChainsFortesCount)
	}

	// Per-hypothesis shape checks: each hypothesis_hash_hex is valid
	// BLAKE3-256 hex; subtype_name is the canonical full name.
	for i, h := range got.Hypotheses {
		if len(h.HypothesisHashHex) != 64 {
			t.Errorf("Hypotheses[%d].HypothesisHashHex: got len %d want 64", i, len(h.HypothesisHashHex))
		}
		if _, err := hex.DecodeString(h.HypothesisHashHex); err != nil {
			t.Errorf("Hypotheses[%d].HypothesisHashHex: not valid hex: %v", i, err)
		}
		if h.SubtypeName != "ghosttrace.events.v1.AutomationGroupFormation" {
			t.Errorf("Hypotheses[%d].SubtypeName: got %q want AutomationGroupFormation", i, h.SubtypeName)
		}
	}

	// Locate per-formation values by hash.
	byHash := make(map[string]hypothesisJSON, len(got.Hypotheses))
	for _, h := range got.Hypotheses {
		byHash[h.HypothesisHashHex] = h
	}
	hashAHex := hex.EncodeToString(hashA[:])
	hashBHex := hex.EncodeToString(hashB[:])

	gotA, ok := byHash[hashAHex]
	if !ok {
		t.Fatalf("formation_a hash %s not found in emitted JSON", hashAHex)
	}
	if gotA.ChainDepthMax != 0 || gotA.ChainBreadthAtRoot != 0 || gotA.ClosureCount != 0 {
		t.Errorf("formation_a morphology: got depth=%d breadth=%d closure=%d want 0/0/0", gotA.ChainDepthMax, gotA.ChainBreadthAtRoot, gotA.ClosureCount)
	}
	if gotA.FormationAt != 1716120100000000000 {
		t.Errorf("formation_a FormationAt: got %d want 1716120100000000000", gotA.FormationAt)
	}

	gotB, ok := byHash[hashBHex]
	if !ok {
		t.Fatalf("formation_b hash %s not found in emitted JSON", hashBHex)
	}
	if gotB.ChainDepthMax != 1 || gotB.ChainBreadthAtRoot != 1 || gotB.ClosureCount != 1 {
		t.Errorf("formation_b morphology: got depth=%d breadth=%d closure=%d want 1/1/1", gotB.ChainDepthMax, gotB.ChainBreadthAtRoot, gotB.ClosureCount)
	}

	// JSON output is indented (per emitJSON's `enc.SetIndent("", "  ")`);
	// verify the indent marker is present in the raw bytes.
	if !bytes.Contains(raw, []byte("\n  \"hypotheses\":")) && !bytes.Contains(raw, []byte("  \"hypotheses\":")) {
		t.Errorf("expected JSON output to be indented with 2-space; raw output: %s", string(raw))
	}
}

// TestEmitJSON_EmptyMeasurement verifies the emission contract for
// the empty-substrate case: zero hypotheses + all-zero stats + valid
// JSON envelope. Matches measure-chain-morphology's documented exit-0
// behavior for empty hypothesis set.
func TestEmitJSON_EmptyMeasurement(t *testing.T) {
	sub := newCLITestSubstrate(t)
	ctx := context.Background()

	m, err := morphology.Measure(ctx, sub)
	if err != nil {
		t.Fatalf("morphology.Measure: %v", err)
	}

	tmp, err := os.CreateTemp("", "measure-chain-morphology-empty-*.json")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer os.Remove(tmp.Name())
	if err := emitJSON(tmp, m); err != nil {
		t.Fatalf("emitJSON: %v", err)
	}
	tmp.Close()

	raw, err := os.ReadFile(tmp.Name())
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var got emissionEnvelope
	if err := json.NewDecoder(bytes.NewReader(raw)).Decode(&got); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(got.Hypotheses) != 0 {
		t.Errorf("Hypotheses count: got %d want 0", len(got.Hypotheses))
	}
	if got.Stats.TotalFormations != 0 {
		t.Errorf("Stats.TotalFormations: got %d want 0", got.Stats.TotalFormations)
	}
	if got.Stats.ChainsFracasCount != 0 {
		t.Errorf("Stats.ChainsFracasCount: got %d want 0", got.Stats.ChainsFracasCount)
	}
	if got.Stats.ChainsFortesCount != 0 {
		t.Errorf("Stats.ChainsFortesCount: got %d want 0", got.Stats.ChainsFortesCount)
	}
}
