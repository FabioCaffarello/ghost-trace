// Integration test exercising chain morphology against
// CoordinationRingFormation events derived end-to-end from F3
// candidate output per §0186. Mirrors §0158 + §0177 + §0184 patterns
// on the CoordinationRing subtype side.
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
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

func newCoordinationMorphologySubstrate(t *testing.T) (*substrate.Substrate, *ingest.Ingester) {
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

func sortHashListAscending(hashes [][]byte) [][]byte {
	out := make([][]byte, len(hashes))
	copy(out, hashes)
	sort.Slice(out, func(i, j int) bool { return bytes.Compare(out[i], out[j]) < 0 })
	return out
}

// commitCoordinationRingFormationFromCandidateWithChain extends the
// §0186 foundation helper with explicit direct_influenced_by +
// closure_hashes parameters for chain construction. Mirrors §0158's
// + §0177's + §0184's equivalents on the CoordinationRing subtype side.
func commitCoordinationRingFormationFromCandidateWithChain(t *testing.T, sub *substrate.Substrate, c *signatures.FormationCandidate, formationAt int64, directInfluencedBy [][]byte, closureHashes [][]byte) [32]byte {
	t.Helper()
	ctx := context.Background()

	interactions := make([]*eventsv1.CoordinationRingInteraction, len(c.Interactions))
	for i, edge := range c.Interactions {
		interactions[i] = &eventsv1.CoordinationRingInteraction{
			ActorA: edge[0],
			ActorB: edge[1],
		}
	}

	formation := &eventsv1.CoordinationRingFormation{
		PatternSignature:   c.SignatureName,
		PatternParameters:  "endpoint_window_seconds=60;min_cohort_size=3",
		FormationAt:        formationAt,
		SourceEventHashes:  c.SourceHashes,
		Interactions:       interactions,
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

// TestMorphology_AgainstCoordinationRingF3DerivedFormationChain
// verifies morphology.Measure walks CoordinationRingFormation chains
// correctly + computes per-formation depth + breadth + closure.
func TestMorphology_AgainstCoordinationRingF3DerivedFormationChain(t *testing.T) {
	sub, in := newCoordinationMorphologySubstrate(t)
	ctx := context.Background()

	// Inject 3 distinct (endpoint, time-bucket) actor cohorts via
	// separate endpoint addresses → F3 emits 3 distinct candidates.
	const bucketStart = int64(1716120000_000_000_000)
	for i, endpoint := range []string{"10.0.0.1:443", "10.0.0.2:443", "10.0.0.3:443"} {
		actorBase := []string{"a", "b", "c"}[i]
		for j := 0; j < 3; j++ {
			appendNetworkObs(t, in,
				actorBase+"-"+string(rune('1'+j)),
				endpoint,
				bucketStart+int64(i*60_000_000_000)+int64(j*1_000_000_000))
		}
	}

	observations, err := collectNetworkObservations(ctx, sub)
	if err != nil {
		t.Fatalf("collectNetworkObservations: %v", err)
	}
	sig := &signatures.EndpointCoVisitV1{}
	result, err := sig.EvaluateNetwork(ctx, observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if got, want := len(result.Candidates), 3; got != want {
		t.Fatalf("candidates count: got %d want %d", got, want)
	}

	hashA := commitCoordinationRingFormationFromCandidateWithChain(t, sub, result.Candidates[0], 1716120100000000000, nil, nil)
	hashB := commitCoordinationRingFormationFromCandidateWithChain(t, sub, result.Candidates[1], 1716120200000000000,
		[][]byte{hashA[:]}, [][]byte{hashA[:]})
	hashC := commitCoordinationRingFormationFromCandidateWithChain(t, sub, result.Candidates[2], 1716120300000000000,
		[][]byte{hashB[:]}, [][]byte{hashA[:], hashB[:]})

	m, err := morphology.Measure(ctx, sub)
	if err != nil {
		t.Fatalf("morphology.Measure: %v", err)
	}

	if got, want := m.Stats.TotalFormations, uint32(3); got != want {
		t.Errorf("TotalFormations: got %d want %d", got, want)
	}
	if got, want := m.Stats.PerSubtype["ghosttrace.events.v1.CoordinationRingFormation"], uint32(3); got != want {
		t.Errorf("PerSubtype[CoordinationRingFormation]: got %d want %d", got, want)
	}
	if got, want := m.Stats.ChainsFracasCount, uint32(3); got != want {
		t.Errorf("ChainsFracasCount: got %d want %d", got, want)
	}

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
		if got.SubtypeName != "ghosttrace.events.v1.CoordinationRingFormation" {
			t.Errorf("%s: SubtypeName got %q want CoordinationRingFormation", exp.name, got.SubtypeName)
		}
	}
	t.Logf("morphology measurement (CoordinationRing chain): total=%d fracas=%d fortes=%d", m.Stats.TotalFormations, m.Stats.ChainsFracasCount, m.Stats.ChainsFortesCount)
}
