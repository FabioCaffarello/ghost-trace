package morphology

import (
	"bytes"
	"context"
	"path/filepath"
	"sort"
	"testing"
	"time"

	commonv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/common/v1"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

func newTestSubstrate(t *testing.T) (*substrate.Substrate, *ingest.Ingester) {
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
// shape discipline (ascending lex). Required for substrate commit.
func sortHashListAscending(hashes [][]byte) [][]byte {
	out := make([][]byte, len(hashes))
	copy(out, hashes)
	sort.Slice(out, func(i, j int) bool { return bytes.Compare(out[i], out[j]) < 0 })
	return out
}

// appendAutomationGroupFormation commits an AutomationGroupFormation
// with the given fields. Returns the committed event's hash for use as
// direct_influenced_by reference in subsequent formations. Hash lists
// auto-sorted per §0139 hash-list element-shape discipline.
func appendAutomationGroupFormation(t *testing.T, in *ingest.Ingester, actorRefs []string, directInfluencedBy [][]byte, sourceHashes [][]byte) [32]byte {
	t.Helper()
	// Per §0139 hash-list discipline: lists must be sorted ascending
	// at substrate commit. Test helper handles sorting transparently.
	sortedInfluencedBy := sortHashListAscending(directInfluencedBy)
	sortedSources := sortHashListAscending(sourceHashes)
	// Per §2.6 BC3 paired-dimension enforcement at marshalling: Cat III
	// formation must carry both confidence + evidential_independence.
	formation := &eventsv1.AutomationGroupFormation{
		PatternSignature:   "test-signature-v1",
		PatternParameters:  "{}",
		ActorRefs:          actorRefs,
		FormationAt:        1716120000000000000,
		SourceEventHashes:  sortedSources,
		DirectInfluencedBy: sortedInfluencedBy,
		ClosureHashes:      sortedInfluencedBy, // simplified for test
		Confidence:         0.5,
		EvidentialIndependence: &commonv1.EvidentialIndependence{
			Numerator:   1,
			Denominator: 1,
		},
	}
	rep, err := in.Append(context.Background(), formation, formation.FormationAt, ingest.Envelope{Channel: "test"})
	if err != nil {
		t.Fatalf("Append AutomationGroupFormation: %v", err)
	}
	var h [32]byte
	if _, err := hexToBytes(rep.EventHashHex, h[:]); err != nil {
		t.Fatalf("hex parse: %v", err)
	}
	return h
}

func hexToBytes(hexStr string, dst []byte) (int, error) {
	if len(hexStr) != 2*len(dst) {
		return 0, &lengthErr{want: 2 * len(dst), got: len(hexStr)}
	}
	for i := 0; i < len(dst); i++ {
		hi, lo := hexNibble(hexStr[2*i]), hexNibble(hexStr[2*i+1])
		dst[i] = byte(hi<<4 | lo)
	}
	return len(dst), nil
}

type lengthErr struct {
	want, got int
}

func (e *lengthErr) Error() string { return "hex length mismatch" }

func hexNibble(c byte) int {
	switch {
	case '0' <= c && c <= '9':
		return int(c - '0')
	case 'a' <= c && c <= 'f':
		return int(c-'a') + 10
	case 'A' <= c && c <= 'F':
		return int(c-'A') + 10
	}
	return 0
}

func TestMeasure_EmptySubstrate(t *testing.T) {
	sub, _ := newTestSubstrate(t)
	m, err := Measure(context.Background(), sub)
	if err != nil {
		t.Fatalf("Measure: %v", err)
	}
	if len(m.Hypotheses) != 0 {
		t.Errorf("Hypotheses: got %d want 0", len(m.Hypotheses))
	}
	if m.Stats.TotalFormations != 0 {
		t.Errorf("TotalFormations: got %d want 0", m.Stats.TotalFormations)
	}
}

func TestMeasure_SingleRootFormation_DepthZero(t *testing.T) {
	sub, in := newTestSubstrate(t)
	// Formation with no direct_influenced_by → depth 0, breadth 0.
	appendAutomationGroupFormation(t, in, []string{"actor-a"}, nil, [][]byte{
		make([]byte, 32), // single Cat I source (placeholder hash)
	})

	m, err := Measure(context.Background(), sub)
	if err != nil {
		t.Fatalf("Measure: %v", err)
	}
	if len(m.Hypotheses) != 1 {
		t.Fatalf("Hypotheses: got %d want 1", len(m.Hypotheses))
	}
	h := m.Hypotheses[0]
	if h.ChainDepthMax != 0 {
		t.Errorf("ChainDepthMax: got %d want 0 (no direct_influenced_by → root)", h.ChainDepthMax)
	}
	if h.ChainBreadthAtRoot != 0 {
		t.Errorf("ChainBreadthAtRoot: got %d want 0", h.ChainBreadthAtRoot)
	}
	if h.SourceEventCount != 1 {
		t.Errorf("SourceEventCount: got %d want 1", h.SourceEventCount)
	}
	// chains-fracas: depth 0 <= 2 OR breadth 0 <= 3 → true.
	if m.Stats.ChainsFracasCount != 1 {
		t.Errorf("ChainsFracasCount: got %d want 1 (root formation is chains-fraca)", m.Stats.ChainsFracasCount)
	}
	if m.Stats.ChainsFortesCount != 0 {
		t.Errorf("ChainsFortesCount: got %d want 0", m.Stats.ChainsFortesCount)
	}
}

func TestMeasure_ParentChildChain_DepthOne(t *testing.T) {
	sub, in := newTestSubstrate(t)
	// Parent: root formation
	parentHash := appendAutomationGroupFormation(t, in, []string{"actor-a"}, nil, [][]byte{make([]byte, 32)})
	// Child: references parent via direct_influenced_by
	childHash := appendAutomationGroupFormation(t, in, []string{"actor-b"}, [][]byte{parentHash[:]}, [][]byte{make([]byte, 32)})
	_ = childHash

	m, err := Measure(context.Background(), sub)
	if err != nil {
		t.Fatalf("Measure: %v", err)
	}
	if len(m.Hypotheses) != 2 {
		t.Fatalf("Hypotheses: got %d want 2", len(m.Hypotheses))
	}
	// Find child + parent by hash
	var parent, child *ChainMorphology
	for _, h := range m.Hypotheses {
		if h.HypothesisHash == parentHash {
			parent = h
		} else if h.HypothesisHash == childHash {
			child = h
		}
	}
	if parent == nil || child == nil {
		t.Fatalf("could not find parent or child in Hypotheses")
	}
	if parent.ChainDepthMax != 0 {
		t.Errorf("parent ChainDepthMax: got %d want 0", parent.ChainDepthMax)
	}
	if child.ChainDepthMax != 1 {
		t.Errorf("child ChainDepthMax: got %d want 1", child.ChainDepthMax)
	}
	if child.ChainBreadthAtRoot != 1 {
		t.Errorf("child ChainBreadthAtRoot: got %d want 1", child.ChainBreadthAtRoot)
	}
}

func TestMeasure_DiamondGraph_MaxPathDepth(t *testing.T) {
	sub, in := newTestSubstrate(t)
	// A → B, A → C, B → D, C → D
	// D's depth = 2 (max path A→B→D or A→C→D)
	a := appendAutomationGroupFormation(t, in, []string{"a"}, nil, [][]byte{make([]byte, 32)})
	b := appendAutomationGroupFormation(t, in, []string{"b"}, [][]byte{a[:]}, [][]byte{make([]byte, 32)})
	c := appendAutomationGroupFormation(t, in, []string{"c"}, [][]byte{a[:]}, [][]byte{make([]byte, 32)})
	d := appendAutomationGroupFormation(t, in, []string{"d"}, [][]byte{b[:], c[:]}, [][]byte{make([]byte, 32)})

	m, err := Measure(context.Background(), sub)
	if err != nil {
		t.Fatalf("Measure: %v", err)
	}
	depthByHash := make(map[[32]byte]uint32)
	for _, h := range m.Hypotheses {
		depthByHash[h.HypothesisHash] = h.ChainDepthMax
	}
	if got := depthByHash[a]; got != 0 {
		t.Errorf("A depth: got %d want 0", got)
	}
	if got := depthByHash[b]; got != 1 {
		t.Errorf("B depth: got %d want 1", got)
	}
	if got := depthByHash[c]; got != 1 {
		t.Errorf("C depth: got %d want 1", got)
	}
	if got := depthByHash[d]; got != 2 {
		t.Errorf("D depth: got %d want 2 (max path A→B→D or A→C→D)", got)
	}
	// D ChainBreadthAtRoot = 2 (direct_influenced_by has B + C)
	for _, h := range m.Hypotheses {
		if h.HypothesisHash == d {
			if h.ChainBreadthAtRoot != 2 {
				t.Errorf("D ChainBreadthAtRoot: got %d want 2", h.ChainBreadthAtRoot)
			}
		}
	}
}

func TestMeasure_AggregateStats_ChainsFortes(t *testing.T) {
	sub, in := newTestSubstrate(t)
	// Build a chains-fortes formation: depth >= 3 AND breadth >= 4.
	// Construct chain: root → l1 → l2 → l3 with breadth 4 at l3.
	root := appendAutomationGroupFormation(t, in, []string{"r"}, nil, [][]byte{make([]byte, 32)})
	l1a := appendAutomationGroupFormation(t, in, []string{"l1a"}, [][]byte{root[:]}, [][]byte{make([]byte, 32)})
	l1b := appendAutomationGroupFormation(t, in, []string{"l1b"}, [][]byte{root[:]}, [][]byte{make([]byte, 32)})
	l1c := appendAutomationGroupFormation(t, in, []string{"l1c"}, [][]byte{root[:]}, [][]byte{make([]byte, 32)})
	l1d := appendAutomationGroupFormation(t, in, []string{"l1d"}, [][]byte{root[:]}, [][]byte{make([]byte, 32)})
	l2 := appendAutomationGroupFormation(t, in, []string{"l2"}, [][]byte{l1a[:], l1b[:], l1c[:], l1d[:]}, [][]byte{make([]byte, 32)})
	// chains-fortes formation: depth 3 (root→l1→l2→target) AND breadth 4 (4 l1 predecessors via l2 ... wait no)
	// Actually breadth at target = len(direct_influenced_by of target).
	// Let target have 4 direct predecessors at depth 2 → breadth 4, depth 3.
	target := appendAutomationGroupFormation(t, in, []string{"target"}, [][]byte{l2[:], l1a[:], l1b[:], l1c[:]}, [][]byte{make([]byte, 32)})
	_ = target

	m, err := Measure(context.Background(), sub)
	if err != nil {
		t.Fatalf("Measure: %v", err)
	}
	// target should be chains-fortes: depth_max(target) = depth(l2)+1 = 2+1 = 3 > 2 AND breadth = 4 > 3.
	var targetMorph *ChainMorphology
	for _, h := range m.Hypotheses {
		if h.HypothesisHash == target {
			targetMorph = h
		}
	}
	if targetMorph == nil {
		t.Fatalf("could not find target formation")
	}
	if targetMorph.ChainDepthMax <= 2 || targetMorph.ChainBreadthAtRoot <= 3 {
		t.Errorf("target should be chains-fortes; got depth=%d breadth=%d", targetMorph.ChainDepthMax, targetMorph.ChainBreadthAtRoot)
	}
	if m.Stats.ChainsFortesCount == 0 {
		t.Errorf("ChainsFortesCount: got 0; expected at least 1 (target is chains-fortes)")
	}
}

func TestMeasure_PerSubtypeBreakdown(t *testing.T) {
	sub, in := newTestSubstrate(t)
	appendAutomationGroupFormation(t, in, []string{"a"}, nil, [][]byte{make([]byte, 32)})
	appendAutomationGroupFormation(t, in, []string{"b"}, nil, [][]byte{make([]byte, 32)})

	m, err := Measure(context.Background(), sub)
	if err != nil {
		t.Fatalf("Measure: %v", err)
	}
	if got := m.Stats.PerSubtype["ghosttrace.events.v1.AutomationGroupFormation"]; got != 2 {
		t.Errorf("PerSubtype[AutomationGroupFormation]: got %d want 2", got)
	}
	if got := m.Stats.PerSubtype["ghosttrace.events.v1.BehavioralClusterFormation"]; got != 0 {
		t.Errorf("PerSubtype[BehavioralClusterFormation]: got %d want 0 (none committed)", got)
	}
	if m.Stats.DepthHistogram[0] != 2 {
		t.Errorf("DepthHistogram[0]: got %d want 2 (both formations are roots)", m.Stats.DepthHistogram[0])
	}
}

func TestMeasure_ContextCancellation(t *testing.T) {
	sub, _ := newTestSubstrate(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Measure(ctx, sub)
	if err == nil {
		t.Fatalf("expected ctx.Err() on canceled context, got nil")
	}
}
