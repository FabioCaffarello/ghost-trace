// Integration test exercising the morphology measurement package
// (§0155) against BehavioralClusterFormation events derived end-to-end
// from F3 candidate output per §0177. Extends the §0176 BehavioralCluster
// arc baseline (Layer B firing per evaluation axis) along the measurement
// axis. Mirrors §0158's equivalent for AutomationGroup on the
// BehavioralCluster subtype side.
//
// Scope per §0177: connectivity + per-formation-value correctness for
// a constructed 3-formation chain at BehavioralCluster subtype.
// Aggregate stats correctness (per-bucket histogram math) covered by
// morphology package's own unit tests; this test validates only the
// integration path + a single shape's per-formation morphology values
// AT THE BehavioralCluster subtype level.
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
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/morphology"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/signatures"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/observationcollector"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

func newBehavioralMorphologySubstrate(t *testing.T) (*substrate.Substrate, *ingest.Ingester) {
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
// shape discipline. Local copy mirroring §0158's helper.
func sortHashListAscending(hashes [][]byte) [][]byte {
	out := make([][]byte, len(hashes))
	copy(out, hashes)
	sort.Slice(out, func(i, j int) bool { return bytes.Compare(out[i], out[j]) < 0 })
	return out
}

// commitBehavioralClusterFormationFromCandidateWithChain extends §0176's
// commitBehavioralClusterFormationFromCandidate with explicit
// direct_influenced_by + closure_hashes parameters for chain
// construction. BehavioralCluster equivalent of §0158's
// commitFormationFromCandidateWithChain.
func commitBehavioralClusterFormationFromCandidateWithChain(t *testing.T, sub *substrate.Substrate, c *signatures.FormationCandidate, formationAt int64, directInfluencedBy [][]byte, closureHashes [][]byte) [32]byte {
	t.Helper()
	ctx := context.Background()

	formation := &eventsv1.BehavioralClusterFormation{
		PatternSignature:   c.SignatureName,
		PatternParameters:  "threshold=3",
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

// TestMorphology_AgainstBehavioralClusterF3DerivedFormationChain
// exercises the path from F3 keystroke_timing_clustering_v1 candidates
// through a 3-formation BehavioralClusterFormation chain commit to
// morphology measurement. Verifies all stages connect + per-formation
// morphology values match the constructed chain shape:
//
//	formation_a (root)         depth=0 breadth=0
//	formation_b → a            depth=1 breadth=1
//	formation_c → b            depth=2 breadth=1
//
// All three formations meet §0143 chains-fracas predicate; aggregate
// stats reflect 3 BehavioralClusterFormation rows.
//
// Mirrors §0158's TestMorphology_AgainstF3DerivedFormationChain on
// the BehavioralCluster subtype side.
func TestMorphology_AgainstBehavioralClusterF3DerivedFormationChain(t *testing.T) {
	sub, in := newBehavioralMorphologySubstrate(t)
	ctx := context.Background()

	// Step 1: Inject BehavioralObservation for 3 sets of 3 actors,
	// each set with a distinct quantized keystroke fingerprint so
	// F3 emits 3 distinct multi-actor candidates.
	ivsA := []uint64{50_000_000, 100_000_000, 0, 100_000_000, 100_000_000, 150_000_000}
	ivsB := []uint64{150_000_000, 200_000_000, 100_000_000, 200_000_000, 200_000_000, 250_000_000}
	ivsC := []uint64{350_000_000, 400_000_000, 300_000_000, 400_000_000, 400_000_000, 450_000_000}
	for i, actor := range []string{"actor-a-1", "actor-a-2", "actor-a-3"} {
		appendKeystrokeObs(t, in, actor, ivsA, int64(1+i))
	}
	for i, actor := range []string{"actor-b-1", "actor-b-2", "actor-b-3"} {
		appendKeystrokeObs(t, in, actor, ivsB, int64(10+i))
	}
	for i, actor := range []string{"actor-c-1", "actor-c-2", "actor-c-3"} {
		appendKeystrokeObs(t, in, actor, ivsC, int64(20+i))
	}

	// Step 2: Run F3 → expect 3 candidates (one per fingerprint
	// cluster). Use first-actor-of-set as the lookup key.
	observations, err := observationcollector.CollectBehavioral(ctx, sub)
	if err != nil {
		t.Fatalf("observationcollector.CollectBehavioral: %v", err)
	}
	sig := &signatures.KeystrokeTimingClusteringV1{}
	result, err := sig.EvaluateBehavioral(ctx, observations)
	if err != nil {
		t.Fatalf("EvaluateBehavioral: %v", err)
	}
	if got, want := len(result.Candidates), 3; got != want {
		t.Fatalf("candidates count: got %d want %d", got, want)
	}
	candA := findCandidateByActor(t, result.Candidates, "actor-a-1")
	candB := findCandidateByActor(t, result.Candidates, "actor-b-1")
	candC := findCandidateByActor(t, result.Candidates, "actor-c-1")

	// Step 3: Commit 3-formation chain.
	hashA := commitBehavioralClusterFormationFromCandidateWithChain(t, sub, candA, 1716120100000000000, nil, nil)
	hashB := commitBehavioralClusterFormationFromCandidateWithChain(t, sub, candB, 1716120200000000000,
		[][]byte{hashA[:]}, [][]byte{hashA[:]})
	hashC := commitBehavioralClusterFormationFromCandidateWithChain(t, sub, candC, 1716120300000000000,
		[][]byte{hashB[:]}, [][]byte{hashA[:], hashB[:]})

	// Step 4: Run morphology.Measure.
	m, err := morphology.Measure(ctx, sub)
	if err != nil {
		t.Fatalf("morphology.Measure: %v", err)
	}

	// Step 5: Verify aggregate stats.
	if got, want := m.Stats.TotalFormations, uint32(3); got != want {
		t.Errorf("TotalFormations: got %d want %d", got, want)
	}
	if got, want := m.Stats.PerSubtype["ghosttrace.events.v1.BehavioralClusterFormation"], uint32(3); got != want {
		t.Errorf("PerSubtype[BehavioralClusterFormation]: got %d want %d", got, want)
	}
	if got, want := m.Stats.ChainsFracasCount, uint32(3); got != want {
		t.Errorf("ChainsFracasCount: got %d want %d", got, want)
	}
	if got, want := m.Stats.ChainsFortesCount, uint32(0); got != want {
		t.Errorf("ChainsFortesCount: got %d want %d", got, want)
	}

	// Step 6: Verify per-formation morphology values via hash lookup.
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
		if got.SubtypeName != "ghosttrace.events.v1.BehavioralClusterFormation" {
			t.Errorf("%s: SubtypeName got %q want BehavioralClusterFormation", exp.name, got.SubtypeName)
		}
	}

	t.Logf("morphology measurement (BehavioralCluster chain): total=%d fracas=%d fortes=%d", m.Stats.TotalFormations, m.Stats.ChainsFracasCount, m.Stats.ChainsFortesCount)
}

// findCandidateByActor locates the FormationCandidate whose ActorRefs[0]
// matches actorRef. Local copy mirroring §0158's helper.
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
