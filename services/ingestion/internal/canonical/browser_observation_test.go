// Focused tests for BrowserObservation per decision-log §0151 + RFC
// Domain Pack v0.1 anti-bot atlas F1.BrowserObservation. Follows
// §0144 + §0146 + §0147 test patterns and additionally exercises
// the §0150 authentication_class envelope field that this modality
// inherits natively at landing time.
//
// Paired-dimension exclusion for BrowserObservation + its 5
// sub-modality types is asserted by
// TestPairedDimensionSubjectSet_StructuralDetection in coverage_test.go
// (auto-picks-up the messageFactory registration).
package canonical

import (
	"bytes"
	"testing"

	commonv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/common/v1"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
)

func newBrowserObservationWithEachModality() []struct {
	name string
	obs  *eventsv1.BrowserObservation
} {
	const (
		observedAt   = int64(1716120000000000000)
		actorRef     = "actor-browser-test"
		sessionRef   = "session-browser-001"
		collectorRef = "browser-sdk:v1"
	)
	envelope := func() *eventsv1.BrowserObservation {
		return &eventsv1.BrowserObservation{
			ObservedAt:          observedAt,
			ActorRef:            actorRef,
			SessionRef:          sessionRef,
			CollectorRef:        collectorRef,
			AuthenticationClass: commonv1.AuthenticationClass_AUTHENTICATION_CLASS_CLIENT_WITNESSED,
		}
	}
	withCanvas := envelope()
	withCanvas.Modality = &eventsv1.BrowserObservation_CanvasFingerprint{
		CanvasFingerprint: &eventsv1.BrowserCanvasFingerprint{
			CanvasHash:         bytes.Repeat([]byte{0x01}, 32),
			TextBaselineOffset: 12,
			NoiseLevel:         0,
		},
	}
	withWebGL := envelope()
	withWebGL.Modality = &eventsv1.BrowserObservation_WebglFingerprint{
		WebglFingerprint: &eventsv1.BrowserWebGLFingerprint{
			Vendor:         "Google Inc. (Apple)",
			Renderer:       "ANGLE (Apple, Apple M1 Pro, OpenGL 4.1)",
			ExtensionsHash: bytes.Repeat([]byte{0x02}, 32),
			MaxTextureSize: 16384,
		},
	}
	withFonts := envelope()
	withFonts.Modality = &eventsv1.BrowserObservation_FontEnumeration{
		FontEnumeration: &eventsv1.BrowserFontEnumeration{
			FontListHash: bytes.Repeat([]byte{0x03}, 32),
			FontCount:    142,
		},
	}
	withCDP := envelope()
	withCDP.Modality = &eventsv1.BrowserObservation_CdpMarker{
		CdpMarker: &eventsv1.BrowserCDPMarker{
			MarkersObserved: []string{
				"navigator.webdriver=true",
				"$cdc_asdjflasutopfhvcZLmcfl_",
			},
			DetectionCount: 2,
		},
	}
	withHeaders := envelope()
	withHeaders.Modality = &eventsv1.BrowserObservation_HeaderOrder{
		HeaderOrder: &eventsv1.BrowserHeaderOrder{
			HeaderSequence: []string{
				"Host", "Connection", "Cache-Control",
				"sec-ch-ua", "User-Agent", "Accept",
			},
			HeaderCount: 6,
		},
	}
	return []struct {
		name string
		obs  *eventsv1.BrowserObservation
	}{
		{"canvas_fingerprint", withCanvas},
		{"webgl_fingerprint", withWebGL},
		{"font_enumeration", withFonts},
		{"cdp_marker", withCDP},
		{"header_order", withHeaders},
	}
}

// TestBrowserObservation_RoundTripPerModality asserts deterministic
// marshal + stable hash per sub-modality. Mirrors §0144 + §0146 +
// §0147 tests.
func TestBrowserObservation_RoundTripPerModality(t *testing.T) {
	for _, tc := range newBrowserObservationWithEachModality() {
		t.Run(tc.name, func(t *testing.T) {
			b1, err := Marshal(tc.obs)
			if err != nil {
				t.Fatalf("first Marshal: %v", err)
			}
			b2, err := Marshal(tc.obs)
			if err != nil {
				t.Fatalf("second Marshal: %v", err)
			}
			if !bytes.Equal(b1, b2) {
				t.Fatalf("non-deterministic marshal:\n  b1=%x\n  b2=%x", b1, b2)
			}
			_, h, err := MarshalAndHash(tc.obs)
			if err != nil {
				t.Fatalf("MarshalAndHash: %v", err)
			}
			h2 := Hash(b1)
			if h != h2 {
				t.Fatalf("hash unstable: %x != %x", h, h2)
			}
		})
	}
}

// TestBrowserObservation_OneofExhaustiveness asserts that all five
// sub-modalities are constructable + marshalable + produce distinct
// canonical bytes from the same envelope-level field values.
func TestBrowserObservation_OneofExhaustiveness(t *testing.T) {
	hashes := map[string][32]byte{}
	for _, tc := range newBrowserObservationWithEachModality() {
		_, h, err := MarshalAndHash(tc.obs)
		if err != nil {
			t.Fatalf("MarshalAndHash %s: %v", tc.name, err)
		}
		for prior, priorHash := range hashes {
			if h == priorHash {
				t.Fatalf("hash collision between sub-modalities %s and %s: %x", prior, tc.name, h)
			}
		}
		hashes[tc.name] = h
	}
	if got, want := len(hashes), 5; got != want {
		t.Fatalf("expected %d distinct sub-modality hashes, got %d", want, got)
	}
}

// TestBrowserObservation_InheritsAuthenticationClassNatively asserts
// the §0150 envelope field is present on BrowserObservation at landing
// time, NOT retrofitted. This is the structural mechanization of §0147
// Methodological observation 1 (resolution-before-fourth-modality-
// landing avoids retrofit). Fourth-modality landing inherits the
// authentication-class field with no schemas-evolution event needed
// to add it.
func TestBrowserObservation_InheritsAuthenticationClassNatively(t *testing.T) {
	obs := &eventsv1.BrowserObservation{
		ObservedAt:          1716120000000000000,
		AuthenticationClass: commonv1.AuthenticationClass_AUTHENTICATION_CLASS_CLIENT_WITNESSED,
		Modality: &eventsv1.BrowserObservation_CdpMarker{
			CdpMarker: &eventsv1.BrowserCDPMarker{
				MarkersObserved: []string{"navigator.webdriver=true"},
				DetectionCount:  1,
			},
		},
	}
	if got := obs.GetAuthenticationClass(); got != commonv1.AuthenticationClass_AUTHENTICATION_CLASS_CLIENT_WITNESSED {
		t.Errorf("authentication_class not set; got %v want CLIENT_WITNESSED", got)
	}
	if _, _, err := MarshalAndHash(obs); err != nil {
		t.Fatalf("MarshalAndHash: %v", err)
	}
	// Distinct hashes when authentication_class differs (mirrors
	// §0150 invariant).
	obsServer := &eventsv1.BrowserObservation{
		ObservedAt:          1716120000000000000,
		AuthenticationClass: commonv1.AuthenticationClass_AUTHENTICATION_CLASS_SERVER_AUTHENTICATED,
		Modality: &eventsv1.BrowserObservation_CdpMarker{
			CdpMarker: &eventsv1.BrowserCDPMarker{
				MarkersObserved: []string{"navigator.webdriver=true"},
				DetectionCount:  1,
			},
		},
	}
	_, hWit, err := MarshalAndHash(obs)
	if err != nil {
		t.Fatalf("MarshalAndHash witnessed: %v", err)
	}
	_, hSrv, err := MarshalAndHash(obsServer)
	if err != nil {
		t.Fatalf("MarshalAndHash server: %v", err)
	}
	if hWit == hSrv {
		t.Fatalf("different authentication_class produced same hash: %x", hWit)
	}
}

// TestBrowserObservation_HashFieldsParticipateInHash asserts the
// privacy-preserving hash fields (canvas_hash, extensions_hash,
// font_list_hash) participate in envelope hash derivation.
func TestBrowserObservation_HashFieldsParticipateInHash(t *testing.T) {
	base := func() *eventsv1.BrowserObservation {
		return &eventsv1.BrowserObservation{
			ObservedAt:          1716120000000000000,
			ActorRef:            "actor-x",
			AuthenticationClass: commonv1.AuthenticationClass_AUTHENTICATION_CLASS_CLIENT_WITNESSED,
		}
	}
	withCanvasA := base()
	withCanvasA.Modality = &eventsv1.BrowserObservation_CanvasFingerprint{
		CanvasFingerprint: &eventsv1.BrowserCanvasFingerprint{
			CanvasHash: bytes.Repeat([]byte{0x01}, 32),
		},
	}
	withCanvasB := base()
	withCanvasB.Modality = &eventsv1.BrowserObservation_CanvasFingerprint{
		CanvasFingerprint: &eventsv1.BrowserCanvasFingerprint{
			CanvasHash: bytes.Repeat([]byte{0x02}, 32),
		},
	}
	_, hA, err := MarshalAndHash(withCanvasA)
	if err != nil {
		t.Fatalf("MarshalAndHash A: %v", err)
	}
	_, hB, err := MarshalAndHash(withCanvasB)
	if err != nil {
		t.Fatalf("MarshalAndHash B: %v", err)
	}
	if hA == hB {
		t.Fatalf("different canvas_hash produced same envelope hash: %x", hA)
	}
}
