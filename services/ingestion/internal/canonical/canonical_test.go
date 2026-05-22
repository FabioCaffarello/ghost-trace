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
