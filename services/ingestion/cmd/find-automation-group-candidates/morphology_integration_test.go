// Integration test exercising the morphology measurement package
// (§0155) against AutomationGroupFormation events derived end-to-end
// from F3 candidate output per §0158. Extends the §0157 baseline
// (which exercised one F3-derived formation against Layer B) with a
// multi-formation chain: three formations committed in chain shape,
// then morphology.Measure walks the substrate and emits per-hypothesis
// chain_depth_max + chain_breadth_at_root + aggregate stats.
//
// This is the first integration test exercising morphology against
// substrate state derived end-to-end from F3 signature output (not
// hand-built fixtures). Validates that the F3 → operator-formation →
// morphology path connects structurally + morphology accepts
// substrate state produced via the F3 inference path.
//
// Scope per §0158: connectivity + per-formation-value correctness for
// a constructed 3-formation chain. Aggregate stats correctness (the
// per-bucket histogram math) is exhaustively covered by the morphology
// package's own unit test suite (internal/morphology/morphology_test.go);
// this test validates only the integration path + a single shape's
// per-formation morphology values.
package main

import (
	"bytes"
	"context"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	commonv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/common/v1"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/observationcollector"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/morphology"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/signatures"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

func newMorphologyIntegrationSubstrate(t *testing.T) (*substrate.Substrate, *ingest.Ingester) {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })
	clock := func() time.Time { return time.Unix(0, 1716120000000000777) }
	return sub, ingest.New(sub, clock)
}

// sortHashListAscending sorts a hash-list per §0139 hash-list element-
// shape discipline (ascending lex). Required for substrate commit
// when direct_influenced_by + closure_hashes carry >1 entry.
func sortHashListAscending(hashes [][]byte) [][]byte {
	out := make([][]byte, len(hashes))
	copy(out, hashes)
	sort.Slice(out, func(i, j int) bool { return bytes.Compare(out[i], out[j]) < 0 })
	return out
}

// commitFormationFromCandidateWithChain extends commitFormationFromCandidate
// (layerb_firing_test.go) with explicit direct_influenced_by +
// closure_hashes parameters for chain construction. Hash lists are
// auto-sorted ascending per §0139.
func commitFormationFromCandidateWithChain(t *testing.T, sub *substrate.Substrate, c *signatures.FormationCandidate, formationAt int64, directInfluencedBy [][]byte, closureHashes [][]byte) [32]byte {
	t.Helper()
	ctx := context.Background()

	formation := &eventsv1.AutomationGroupFormation{
		PatternSignature:   c.SignatureName,
		PatternParameters:  "threshold=2",
		ActorRefs:          c.ActorRefs,
		FormationAt:        formationAt,
		SourceEventHashes:  c.SourceHashes,
		DirectInfluencedBy: sortHashListAscending(directInfluencedBy),
		ClosureHashes:      sortHashListAscending(closureHashes),
		Confidence:         float32(c.ConfidenceHint),
		EvidentialIndependence: &commonv1.EvidentialIndependence{
			Numerator:   1,
			Denominator: 1,
		},
	}

	payload, hash, err := canonical.MarshalAndHash(formation)
	if err != nil {
		t.Fatalf("MarshalAndHash formation: %v", err)
	}
	hex := canonical.HashHex(hash)
	row := substrate.EventRow{
		EventHash:   hash,
		EventTime:   formationAt,
		MessageType: string(formation.ProtoReflect().Descriptor().FullName()),
		PayloadRef:  hex[:2] + "/" + hex[2:],
		CommittedAt: formationAt,
	}
	if err := sub.Append(ctx, row, payload); err != nil {
		t.Fatalf("substrate.Append formation: %v", err)
	}
	return hash
}

// findCandidateByActor locates the FormationCandidate whose ActorRefs[0]
// matches actorRef. Helper for tests that aggregate multiple candidates
// from a single signature pass.
func findCandidateByActor(t *testing.T, candidates []*signatures.FormationCandidate, actorRef string) *signatures.FormationCandidate {
	t.Helper()
	for _, c := range candidates {
		if len(c.ActorRefs) > 0 && c.ActorRefs[0] == actorRef {
			return c
		}
	}
	t.Fatalf("candidate for actor %q not found", actorRef)
	return nil
}

// TestMorphology_AgainstF3DerivedFormationChain exercises the path
// from F3 signature candidates through a 3-formation chain commit to
// morphology measurement. Verifies all stages connect + per-formation
// morphology values match the constructed chain shape:
//
//	formation_a (root)      depth=0 breadth=0
//	formation_b → a         depth=1 breadth=1
//	formation_c → b         depth=2 breadth=1
//
// All three formations meet the §0143 chains-fracas predicate
// (depth_max <= 2 OR breadth_at_root <= 3), so ChainsFracasCount=3 +
// ChainsFortesCount=0.
func TestMorphology_AgainstF3DerivedFormationChain(t *testing.T) {
	sub, in := newMorphologyIntegrationSubstrate(t)
	ctx := context.Background()

	// Step 1: Inject BrowserObservations for 3 actors, each above
	// signature default threshold (3 detections each).
	appendBrowserObs(t, in, "actor-a", []string{"navigator.webdriver=true"}, 3)
	appendBrowserObs(t, in, "actor-b", []string{"$cdc_test"}, 3)
	appendBrowserObs(t, in, "actor-c", []string{"missing-chrome.csi"}, 3)

	// Step 2: Run F3 signature → expect 3 candidates.
	observations, err := observationcollector.CollectBrowser(ctx, sub)
	if err != nil {
		t.Fatalf("observationcollector.CollectBrowser: %v", err)
	}
	if got, want := len(observations), 3; got != want {
		t.Fatalf("observations count: got %d want %d", got, want)
	}
	sig := &signatures.CDPMarkerDensityV1{}
	result, err := sig.EvaluateBrowser(ctx, observations)
	if err != nil {
		t.Fatalf("EvaluateBrowser: %v", err)
	}
	if got, want := len(result.Candidates), 3; got != want {
		t.Fatalf("candidates count: got %d want %d", got, want)
	}
	candA := findCandidateByActor(t, result.Candidates, "actor-a")
	candB := findCandidateByActor(t, result.Candidates, "actor-b")
	candC := findCandidateByActor(t, result.Candidates, "actor-c")

	// Step 3: Commit 3-formation chain. Each formation references the
	// prior via direct_influenced_by + accumulates closure_hashes per
	// §0134-τ transitive closure semantics. Distinct formationAt
	// timestamps ensure unique content-hashes.
	hashA := commitFormationFromCandidateWithChain(t, sub, candA, 1716120100000000000, nil, nil)
	hashB := commitFormationFromCandidateWithChain(t, sub, candB, 1716120200000000000,
		[][]byte{hashA[:]}, [][]byte{hashA[:]})
	hashC := commitFormationFromCandidateWithChain(t, sub, candC, 1716120300000000000,
		[][]byte{hashB[:]}, [][]byte{hashA[:], hashB[:]})

	// Step 4: Run morphology.Measure against the substrate.
	m, err := morphology.Measure(ctx, sub)
	if err != nil {
		t.Fatalf("morphology.Measure: %v", err)
	}

	// Step 5: Verify aggregate stats.
	if got, want := m.Stats.TotalFormations, uint32(3); got != want {
		t.Errorf("TotalFormations: got %d want %d", got, want)
	}
	if got, want := m.Stats.PerSubtype["ghosttrace.events.v1.AutomationGroupFormation"], uint32(3); got != want {
		t.Errorf("PerSubtype[AutomationGroupFormation]: got %d want %d", got, want)
	}
	// All 3 formations meet chains-fracas predicate (depth<=2 OR breadth<=3).
	if got, want := m.Stats.ChainsFracasCount, uint32(3); got != want {
		t.Errorf("ChainsFracasCount: got %d want %d", got, want)
	}
	if got, want := m.Stats.ChainsFortesCount, uint32(0); got != want {
		t.Errorf("ChainsFortesCount: got %d want %d", got, want)
	}

	// Step 6: Verify per-formation morphology values. Lookup by hash
	// since Measurement.Hypotheses ordering is hash-sorted (not
	// commit-order).
	morphByHash := make(map[[32]byte]*morphology.ChainMorphology, len(m.Hypotheses))
	for _, h := range m.Hypotheses {
		morphByHash[h.HypothesisHash] = h
	}
	expectations := []struct {
		name     string
		hash     [32]byte
		depthMax uint32
		breadth  uint32
		closure  uint32
	}{
		{"formation_a (root)", hashA, 0, 0, 0},
		{"formation_b (depth 1)", hashB, 1, 1, 1},
		{"formation_c (depth 2)", hashC, 2, 1, 2},
	}
	for _, exp := range expectations {
		got, ok := morphByHash[exp.hash]
		if !ok {
			t.Errorf("%s: hypothesis hash not found in morphology output", exp.name)
			continue
		}
		if got.ChainDepthMax != exp.depthMax {
			t.Errorf("%s: ChainDepthMax got %d want %d", exp.name, got.ChainDepthMax, exp.depthMax)
		}
		if got.ChainBreadthAtRoot != exp.breadth {
			t.Errorf("%s: ChainBreadthAtRoot got %d want %d", exp.name, got.ChainBreadthAtRoot, exp.breadth)
		}
		if got.ClosureCount != exp.closure {
			t.Errorf("%s: ClosureCount got %d want %d", exp.name, got.ClosureCount, exp.closure)
		}
		if got.SubtypeName != "ghosttrace.events.v1.AutomationGroupFormation" {
			t.Errorf("%s: SubtypeName got %q want AutomationGroupFormation", exp.name, got.SubtypeName)
		}
	}

	t.Logf("morphology measurement: total=%d fracas=%d fortes=%d", m.Stats.TotalFormations, m.Stats.ChainsFracasCount, m.Stats.ChainsFortesCount)
}
