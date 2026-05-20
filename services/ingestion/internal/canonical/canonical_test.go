package canonical

import (
	"bytes"
	"testing"

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
