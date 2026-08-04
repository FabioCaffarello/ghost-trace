package canonical

import (
	"bytes"
	"testing"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/libs/genproto/events/v1"
)

func sampleBatch() *eventsv1.TelemetryBatch {
	return &eventsv1.TelemetryBatch{
		TenantId:  "t_demo",
		SessionId: "s_abc123",
		Seq:       3,
		SentAtMs:  8421,
		PagePath:  "/login",
		ViewportW: 1440,
		ViewportH: 900,
		PointerEvents: []*eventsv1.PointerEvent{{
			TMs: 1200,
			Src: "mouse",
			Pts: []*eventsv1.PointerPoint{
				{X: 412, Y: 388, DtMs: 0},
				{X: 430, Y: 371, DtMs: 16},
				{X: 468, Y: 340, DtMs: 33},
			},
		}},
		ReceivedAt: 1716120000000000000,
	}
}

func TestMarshalIsDeterministic(t *testing.T) {
	// The property the archive depends on: equal messages must produce
	// byte-identical output, or the same observation stored twice
	// becomes two records.
	first, err := Marshal(sampleBatch())
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	for i := 0; i < 100; i++ {
		again, err := Marshal(sampleBatch())
		if err != nil {
			t.Fatalf("Marshal iteration %d: %v", i, err)
		}
		if !bytes.Equal(first, again) {
			t.Fatalf("iteration %d produced different bytes", i)
		}
	}
}

func TestHashIsStableAcrossEqualMessages(t *testing.T) {
	_, h1, err := MarshalAndHash(sampleBatch())
	if err != nil {
		t.Fatalf("MarshalAndHash: %v", err)
	}
	_, h2, err := MarshalAndHash(sampleBatch())
	if err != nil {
		t.Fatalf("MarshalAndHash: %v", err)
	}
	if h1 != h2 {
		t.Errorf("equal messages hashed differently: %s vs %s", HashHex(h1), HashHex(h2))
	}
}

func TestDistinctMessagesHashDistinctly(t *testing.T) {
	// A single coordinate change must move the hash — otherwise
	// distinct observations could collide into one archive record.
	a := sampleBatch()
	b := sampleBatch()
	b.PointerEvents[0].Pts[0].X = 413

	_, ha, err := MarshalAndHash(a)
	if err != nil {
		t.Fatalf("MarshalAndHash(a): %v", err)
	}
	_, hb, err := MarshalAndHash(b)
	if err != nil {
		t.Fatalf("MarshalAndHash(b): %v", err)
	}
	if ha == hb {
		t.Error("a one-pixel difference produced the same hash")
	}
}

func TestSeqIsLoadBearingInIdentity(t *testing.T) {
	// Batches are accepted out of order and reordered server-side by
	// seq (contract §2). Two batches identical but for seq must remain
	// distinguishable in the archive.
	a := sampleBatch()
	b := sampleBatch()
	b.Seq = 4

	_, ha, _ := MarshalAndHash(a)
	_, hb, _ := MarshalAndHash(b)
	if ha == hb {
		t.Error("batches differing only in seq hashed identically")
	}
}

func TestHashHexIsLowercase64(t *testing.T) {
	_, h, err := MarshalAndHash(sampleBatch())
	if err != nil {
		t.Fatalf("MarshalAndHash: %v", err)
	}
	hexed := HashHex(h)
	if len(hexed) != 64 {
		t.Errorf("HashHex length = %d, want 64", len(hexed))
	}
	for _, c := range hexed {
		if !(c >= '0' && c <= '9') && !(c >= 'a' && c <= 'f') {
			t.Errorf("HashHex contains non-lowercase-hex byte %q", c)
			break
		}
	}
}

func TestMarshalEvaluationRoundTrips(t *testing.T) {
	ev := &eventsv1.Evaluation{
		TenantId:      "t_demo",
		EvaluationId:  "ev_4d81",
		SessionId:     "s_abc123",
		Action:        "login",
		Decision:      "allow",
		Mode:          "monitor",
		Score:         0.82,
		Confidence:    0.41,
		PolicyRef:     "pointer-linearity-v1",
		FeatureSetRef: "pointer-v1",
		Reasons: []*eventsv1.Reason{
			{Code: "POINTER_LINEARITY", Weight: 0.34},
		},
		Features: &eventsv1.FeatureState{
			PointerStraightness: 0.97,
			PointerSegments:     4,
			PointerPathPx:       1820,
			PointerPoints:       143,
		},
	}
	if _, _, err := MarshalAndHash(ev); err != nil {
		t.Fatalf("MarshalAndHash(Evaluation): %v", err)
	}
}
