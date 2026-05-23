package canonical

import (
	"bytes"
	"strings"
	"testing"

	commonv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/common/v1"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
)

func newTestMessage() *eventsv1.DeclaredSession {
	return &eventsv1.DeclaredSession{
		DeclaredAt:        1716120000000000000,
		ActorRef:          "actor-under-test",
		SessionDescriptor: []byte("session-descriptor-under-test"),
	}
}

func TestMarshalDeterminism(t *testing.T) {
	msg := newTestMessage()

	b1, err := Marshal(msg)
	if err != nil {
		t.Fatalf("first Marshal: %v", err)
	}
	b2, err := Marshal(msg)
	if err != nil {
		t.Fatalf("second Marshal: %v", err)
	}
	if !bytes.Equal(b1, b2) {
		t.Fatalf("non-deterministic marshal:\n  b1=%x\n  b2=%x", b1, b2)
	}
}

func TestHashStability(t *testing.T) {
	msg := newTestMessage()

	b, h, err := MarshalAndHash(msg)
	if err != nil {
		t.Fatalf("MarshalAndHash: %v", err)
	}
	h2 := Hash(b)
	if h != h2 {
		t.Fatalf("hash unstable: %x != %x", h, h2)
	}
}

func TestHashHexFormat(t *testing.T) {
	msg := newTestMessage()
	_, h, err := MarshalAndHash(msg)
	if err != nil {
		t.Fatal(err)
	}
	hex := HashHex(h)
	if got, want := len(hex), 64; got != want {
		t.Fatalf("HashHex length: got %d, want %d", got, want)
	}
	for _, c := range hex {
		switch {
		case c >= '0' && c <= '9':
		case c >= 'a' && c <= 'f':
		default:
			t.Fatalf("HashHex contains non-lowercase-hex character %q (full: %s)", c, hex)
		}
	}
}

func TestFieldChangeChangesHash(t *testing.T) {
	a := newTestMessage()
	b := newTestMessage()
	b.ActorRef = "different-actor"

	_, ha, err := MarshalAndHash(a)
	if err != nil {
		t.Fatal(err)
	}
	_, hb, err := MarshalAndHash(b)
	if err != nil {
		t.Fatal(err)
	}
	if ha == hb {
		t.Fatal("distinct field values produced identical hashes")
	}
}

// hash returns a fresh 32-byte slice with the first byte set to b. Used to
// build sorted / unsorted / duplicate hash lists in influence-validation
// tests. Distinct b values produce distinct lexicographic orderings; equal
// b values produce duplicates.
func hash(b byte) []byte {
	out := make([]byte, 32)
	out[0] = b
	return out
}

func TestMarshalAcceptsWellFormedClosureHashes(t *testing.T) {
	msg := &eventsv1.BehavioralClusterFormation{
		PatternSignature:   "session-descriptor-shared-v1",
		PatternParameters:  "min_cluster_size=2",
		ActorRefs:          []string{"actor-a", "actor-b"},
		FormationAt:        1716120000000000000,
		Confidence:         0.8,
		SourceEventHashes:  [][]byte{hash(0x10), hash(0x20)},
		DirectInfluencedBy: [][]byte{hash(0xa0)},
		ClosureHashes:      [][]byte{hash(0xa0), hash(0xb0)},
	}
	if _, err := Marshal(msg); err != nil {
		t.Fatalf("well-formed message rejected: %v", err)
	}
}

func TestMarshalRejectsShortClosureHash(t *testing.T) {
	msg := &eventsv1.BehavioralClusterFormation{
		ClosureHashes: [][]byte{[]byte("too-short")},
	}
	_, err := Marshal(msg)
	if err == nil {
		t.Fatal("Marshal accepted a closure_hashes element shorter than 32 bytes")
	}
	if !strings.Contains(err.Error(), "closure_hashes") || !strings.Contains(err.Error(), "want 32") {
		t.Fatalf("error %q does not name the field + expected length", err)
	}
}

func TestMarshalRejectsOversizedDirectInfluencedBy(t *testing.T) {
	tooLong := make([]byte, 33)
	msg := &eventsv1.BehavioralClusterFormation{
		DirectInfluencedBy: [][]byte{tooLong},
	}
	_, err := Marshal(msg)
	if err == nil {
		t.Fatal("Marshal accepted a direct_influenced_by element longer than 32 bytes")
	}
	if !strings.Contains(err.Error(), "direct_influenced_by") {
		t.Fatalf("error %q does not name direct_influenced_by", err)
	}
}

func TestMarshalRejectsOutOfOrderClosureHashes(t *testing.T) {
	msg := &eventsv1.BehavioralClusterFormation{
		ClosureHashes: [][]byte{hash(0xb0), hash(0xa0)}, // descending
	}
	_, err := Marshal(msg)
	if err == nil {
		t.Fatal("Marshal accepted descending closure_hashes")
	}
	if !strings.Contains(err.Error(), "ascending") {
		t.Fatalf("error %q does not mention ascending ordering", err)
	}
}

func TestMarshalRejectsDuplicateClosureHashes(t *testing.T) {
	msg := &eventsv1.BehavioralClusterFormation{
		ClosureHashes: [][]byte{hash(0xa0), hash(0xa0)},
	}
	_, err := Marshal(msg)
	if err == nil {
		t.Fatal("Marshal accepted duplicate closure_hashes")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("error %q does not mention duplicates", err)
	}
}

func TestMarshalAcceptsEmptyInfluenceLists(t *testing.T) {
	// Cat II OperationalSession with empty influence lists is the
	// inception-phase typical case (Cat I-only chain). Must marshal.
	msg := &eventsv1.OperationalSession{
		DefinitionVersion:    "padded-v1",
		DefinitionParameters: "pad_seconds=300",
		SourceEventHash:      hash(0x01),
		ActorRef:             "actor-empty-chain",
		Confidence:           1.0,
		EvidentialIndependence: &commonv1.EvidentialIndependence{
			Numerator:   1,
			Denominator: 1,
		},
	}
	if _, err := Marshal(msg); err != nil {
		t.Fatalf("empty-influence-list message rejected: %v", err)
	}
}

func TestMarshalAcceptsValidEvidentialIndependence(t *testing.T) {
	// numerator=0 / denominator=1 → α = 0 (fully influenced; valid).
	zero := &commonv1.EvidentialIndependence{Numerator: 0, Denominator: 1}
	if _, err := Marshal(zero); err != nil {
		t.Fatalf("α = 0 (0/1) rejected: %v", err)
	}
	// numerator=denominator → α = 1 (fully independent; valid).
	one := &commonv1.EvidentialIndependence{Numerator: 7, Denominator: 7}
	if _, err := Marshal(one); err != nil {
		t.Fatalf("α = 1 (7/7) rejected: %v", err)
	}
	// Intermediate value 2/3 — valid.
	half := &commonv1.EvidentialIndependence{Numerator: 2, Denominator: 3}
	if _, err := Marshal(half); err != nil {
		t.Fatalf("α = 2/3 rejected: %v", err)
	}
}

func TestMarshalRejectsZeroDenominatorEvidentialIndependence(t *testing.T) {
	bad := &commonv1.EvidentialIndependence{Numerator: 1, Denominator: 0}
	_, err := Marshal(bad)
	if err == nil {
		t.Fatal("Marshal accepted EvidentialIndependence with denominator=0")
	}
	if !strings.Contains(err.Error(), "denominator must be > 0") {
		t.Fatalf("error %q does not mention denominator > 0 invariant", err)
	}
}

func TestMarshalRejectsOutOfRangeEvidentialIndependence(t *testing.T) {
	bad := &commonv1.EvidentialIndependence{Numerator: 5, Denominator: 3}
	_, err := Marshal(bad)
	if err == nil {
		t.Fatal("Marshal accepted EvidentialIndependence with numerator > denominator (α > 1)")
	}
	if !strings.Contains(err.Error(), "α ∈ [0, 1]") {
		t.Fatalf("error %q does not mention the α range invariant", err)
	}
}

func TestMarshalChecksNestedEvidentialIndependence(t *testing.T) {
	// LayerBParameters carries multiple EvidentialIndependence sub-fields
	// (t_b, k_c) — the walk must recurse into the nested-message
	// structure and reject when any nested instance is malformed.
	bad := &commonv1.LayerBParameters{
		TB: &commonv1.EvidentialIndependence{Numerator: 1, Denominator: 2},
		KC: &commonv1.EvidentialIndependence{Numerator: 5, Denominator: 3}, // α > 1
		NWindow:               1000,
		NADurationNanoseconds: 86400000000000,
	}
	_, err := Marshal(bad)
	if err == nil {
		t.Fatal("Marshal accepted LayerBParameters with k_c α > 1")
	}
	if !strings.Contains(err.Error(), "α ∈ [0, 1]") {
		t.Fatalf("nested-walk error %q does not mention the α range invariant", err)
	}
}

func TestMarshalAcceptsUnsetEvidentialIndependence(t *testing.T) {
	// A message that declares an EvidentialIndependence field but leaves
	// it nil must NOT be rejected by the rational-pair check (the check
	// is on PRESENT instances). Paired-dimension presence-enforcement is
	// a substrate-tier concern per the canonical-serialization-contract.
	msg := &eventsv1.BehavioralClusterFormation{
		PatternSignature:  "p",
		PatternParameters: "k=v",
		FormationAt:       1,
		Confidence:        0.5,
		// EvidentialIndependence intentionally nil.
	}
	if _, err := Marshal(msg); err != nil {
		t.Fatalf("unset evidential_independence rejected by rational-pair check: %v", err)
	}
}

// Per docs/architecture/canonical-serialization-contract.md §Hash-List
// Field Discipline + decision-log §0139: the marshalling-boundary
// element-shape check covers FIVE field families. PRs #143 enforced two
// (closure_hashes + direct_influenced_by); §0139 generalizes to three
// additional families (source_event_hashes per formation protos,
// antecedent_formation_event_hashes per merge protos,
// successor_formation_event_hashes per split protos). The tests below
// exercise each new family with positive (ascending well-formed) +
// negative (descending / duplicate / short) cases.

func TestMarshalRejectsShortSourceEventHash(t *testing.T) {
	msg := &eventsv1.BehavioralClusterFormation{
		SourceEventHashes: [][]byte{[]byte("too-short")},
	}
	_, err := Marshal(msg)
	if err == nil {
		t.Fatal("Marshal accepted a source_event_hashes element shorter than 32 bytes")
	}
	if !strings.Contains(err.Error(), "source_event_hashes") || !strings.Contains(err.Error(), "want 32") {
		t.Fatalf("error %q does not name source_event_hashes + expected length", err)
	}
}

func TestMarshalRejectsOutOfOrderSourceEventHashes(t *testing.T) {
	msg := &eventsv1.BehavioralClusterFormation{
		SourceEventHashes: [][]byte{hash(0xb0), hash(0xa0)}, // descending
	}
	_, err := Marshal(msg)
	if err == nil {
		t.Fatal("Marshal accepted descending source_event_hashes")
	}
	if !strings.Contains(err.Error(), "source_event_hashes") || !strings.Contains(err.Error(), "ascending") {
		t.Fatalf("error %q does not name source_event_hashes + ascending discipline", err)
	}
}

func TestMarshalRejectsDuplicateSourceEventHashes(t *testing.T) {
	msg := &eventsv1.BehavioralClusterFormation{
		SourceEventHashes: [][]byte{hash(0xa0), hash(0xa0)},
	}
	_, err := Marshal(msg)
	if err == nil {
		t.Fatal("Marshal accepted duplicate source_event_hashes")
	}
	if !strings.Contains(err.Error(), "source_event_hashes") || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("error %q does not name source_event_hashes + duplicate discipline", err)
	}
}

func TestMarshalAcceptsWellFormedSourceEventHashes(t *testing.T) {
	msg := &eventsv1.BehavioralClusterFormation{
		PatternSignature:  "p",
		PatternParameters: "k=v",
		FormationAt:       1,
		Confidence:        0.5,
		SourceEventHashes: [][]byte{hash(0x01), hash(0x02), hash(0x03)},
	}
	if _, err := Marshal(msg); err != nil {
		t.Fatalf("well-formed ascending source_event_hashes rejected: %v", err)
	}
}

func TestMarshalRejectsOutOfOrderAntecedentFormationEventHashes(t *testing.T) {
	msg := &eventsv1.BehavioralClusterMerge{
		AntecedentFormationEventHashes: [][]byte{hash(0xb0), hash(0xa0)},
	}
	_, err := Marshal(msg)
	if err == nil {
		t.Fatal("Marshal accepted descending antecedent_formation_event_hashes")
	}
	if !strings.Contains(err.Error(), "antecedent_formation_event_hashes") || !strings.Contains(err.Error(), "ascending") {
		t.Fatalf("error %q does not name antecedent_formation_event_hashes + ascending discipline", err)
	}
}

func TestMarshalRejectsDuplicateAntecedentFormationEventHashes(t *testing.T) {
	msg := &eventsv1.CampaignHypothesisMerge{
		AntecedentFormationEventHashes: [][]byte{hash(0x10), hash(0x10)},
	}
	_, err := Marshal(msg)
	if err == nil {
		t.Fatal("Marshal accepted duplicate antecedent_formation_event_hashes")
	}
	if !strings.Contains(err.Error(), "antecedent_formation_event_hashes") || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("error %q does not name antecedent_formation_event_hashes + duplicate discipline", err)
	}
}

func TestMarshalRejectsShortSuccessorFormationEventHash(t *testing.T) {
	msg := &eventsv1.BehavioralClusterSplit{
		SuccessorFormationEventHashes: [][]byte{[]byte("not-32-bytes")},
	}
	_, err := Marshal(msg)
	if err == nil {
		t.Fatal("Marshal accepted a successor_formation_event_hashes element shorter than 32 bytes")
	}
	if !strings.Contains(err.Error(), "successor_formation_event_hashes") || !strings.Contains(err.Error(), "want 32") {
		t.Fatalf("error %q does not name successor_formation_event_hashes + expected length", err)
	}
}

func TestMarshalRejectsOutOfOrderSuccessorFormationEventHashes(t *testing.T) {
	msg := &eventsv1.AutomationGroupSplit{
		SuccessorFormationEventHashes: [][]byte{hash(0xc0), hash(0xa0), hash(0xb0)}, // not ascending
	}
	_, err := Marshal(msg)
	if err == nil {
		t.Fatal("Marshal accepted out-of-order successor_formation_event_hashes")
	}
	if !strings.Contains(err.Error(), "successor_formation_event_hashes") || !strings.Contains(err.Error(), "ascending") {
		t.Fatalf("error %q does not name successor_formation_event_hashes + ascending discipline", err)
	}
}

func TestMarshalAcceptsWellFormedSuccessorFormationEventHashes(t *testing.T) {
	msg := &eventsv1.CoordinationRingSplit{
		AntecedentFormationEventHash:  hash(0x01),
		SuccessorFormationEventHashes: [][]byte{hash(0x10), hash(0x20), hash(0x30)},
		SplitAt:                       1716120000000000000,
		Reason:                        "well-formed split fixture",
	}
	if _, err := Marshal(msg); err != nil {
		t.Fatalf("well-formed ascending successor_formation_event_hashes rejected: %v", err)
	}
}
