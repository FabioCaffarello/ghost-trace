// Focused tests for AttestationObservation per decision-log §0147 + RFC
// Domain Pack v0.1 anti-bot atlas F1.AttestationObservation. Follows
// §0144 NetworkObservation + §0146 BehavioralObservation test patterns.
//
// Paired-dimension exclusion for AttestationObservation + its 5
// sub-modality types is asserted by
// TestPairedDimensionSubjectSet_StructuralDetection in coverage_test.go
// (auto-picks-up the messageFactory registration in corpus_test.go).
package canonical

import (
	"bytes"
	"testing"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
)

func newAttestationObservationWithEachModality() []struct {
	name string
	obs  *eventsv1.AttestationObservation
} {
	const (
		observedAt   = int64(1716120000000000000)
		actorRef     = "actor-attestation-test"
		sessionRef   = "session-attest-001"
		collectorRef = "browser-sdk:v1"
	)
	envelope := func() *eventsv1.AttestationObservation {
		return &eventsv1.AttestationObservation{
			ObservedAt:   observedAt,
			ActorRef:     actorRef,
			SessionRef:   sessionRef,
			CollectorRef: collectorRef,
		}
	}
	withApple := envelope()
	withApple.Modality = &eventsv1.AttestationObservation_ApplePat{
		ApplePat: &eventsv1.AttestationApplePAT{
			TokenHash:     make([]byte, 32),
			IssuerKeyId:   "apple-issuer-2026Q2",
			ChallengeHash: make([]byte, 32),
			RedeemedAt:    1716120000000000000,
		},
	}
	for i := range withApple.GetApplePat().TokenHash {
		withApple.GetApplePat().TokenHash[i] = byte(i)
	}
	for i := range withApple.GetApplePat().ChallengeHash {
		withApple.GetApplePat().ChallengeHash[i] = byte(255 - i)
	}

	withGoogle := envelope()
	withGoogle.Modality = &eventsv1.AttestationObservation_GooglePrivacyPass{
		GooglePrivacyPass: &eventsv1.AttestationGooglePrivacyPass{
			TokenHash:     bytes.Repeat([]byte{0xAB}, 32),
			IssuerKeyId:   "google-pp-issuer-2026Q2",
			ChallengeHash: bytes.Repeat([]byte{0xCD}, 32),
			RedeemedAt:    1716120000000000000,
		},
	}

	withTurnstile := envelope()
	withTurnstile.Modality = &eventsv1.AttestationObservation_CloudflareTurnstile{
		CloudflareTurnstile: &eventsv1.AttestationCloudflareTurnstile{
			ResponseTokenHash: bytes.Repeat([]byte{0xEF}, 32),
			SiteKey:           "1x00000000000000000000AA",
			Action:            "login",
			Cdata:             "session-marker-xyz",
			IssuedAt:          1716120000000000000,
		},
	}

	withWebAuthn := envelope()
	withWebAuthn.Modality = &eventsv1.AttestationObservation_WebauthnAssertion{
		WebauthnAssertion: &eventsv1.AttestationWebAuthnAssertion{
			CredentialIdHash:   bytes.Repeat([]byte{0x01}, 32),
			AuthenticatorData:  []byte{0x01, 0x02, 0x03, 0x04, 0x05},
			ClientDataJsonHash: bytes.Repeat([]byte{0x02}, 32),
			Signature:          []byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF},
			UserHandleHash:     bytes.Repeat([]byte{0x03}, 32),
		},
	}

	withFido2 := envelope()
	withFido2.Modality = &eventsv1.AttestationObservation_Fido2Attestation{
		Fido2Attestation: &eventsv1.AttestationFido2{
			CredentialIdHash:   bytes.Repeat([]byte{0x10}, 32),
			AttestationObject:  []byte{0xA0, 0xA1, 0xA2, 0xA3},
			Format:             "packed",
			Aaguid:             bytes.Repeat([]byte{0x55}, 16),
			RegisteredAt:       1716120000000000000,
		},
	}

	return []struct {
		name string
		obs  *eventsv1.AttestationObservation
	}{
		{"apple_pat", withApple},
		{"google_privacy_pass", withGoogle},
		{"cloudflare_turnstile", withTurnstile},
		{"webauthn_assertion", withWebAuthn},
		{"fido2_attestation", withFido2},
	}
}

// TestAttestationObservation_RoundTripPerModality asserts deterministic
// marshal + stable hash per sub-modality. Mirrors §0144 + §0146 tests.
func TestAttestationObservation_RoundTripPerModality(t *testing.T) {
	for _, tc := range newAttestationObservationWithEachModality() {
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

// TestAttestationObservation_OneofExhaustiveness asserts that all five
// sub-modalities are constructable + marshalable + produce distinct
// canonical bytes from the same envelope-level field values.
func TestAttestationObservation_OneofExhaustiveness(t *testing.T) {
	hashes := map[string][32]byte{}
	for _, tc := range newAttestationObservationWithEachModality() {
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

// TestAttestationObservation_HashFieldsParticipateInHash asserts that
// the BLAKE3 hash fields (token_hash, challenge_hash, etc.) participate
// in canonical hash derivation — two AttestationApplePAT records with
// different token_hash values produce distinct envelope hashes. This is
// the substrate-level guarantee that hash-rather-than-raw-token design
// preserves uniqueness across distinct tokens.
func TestAttestationObservation_HashFieldsParticipateInHash(t *testing.T) {
	base := func() *eventsv1.AttestationObservation {
		return &eventsv1.AttestationObservation{
			ObservedAt:   1716120000000000000,
			ActorRef:     "actor-x",
			SessionRef:   "session-y",
			CollectorRef: "browser-sdk:v1",
		}
	}
	withTokenA := base()
	withTokenA.Modality = &eventsv1.AttestationObservation_ApplePat{
		ApplePat: &eventsv1.AttestationApplePAT{
			TokenHash:   bytes.Repeat([]byte{0x01}, 32),
			IssuerKeyId: "apple-issuer-2026Q2",
		},
	}
	withTokenB := base()
	withTokenB.Modality = &eventsv1.AttestationObservation_ApplePat{
		ApplePat: &eventsv1.AttestationApplePAT{
			TokenHash:   bytes.Repeat([]byte{0x02}, 32),
			IssuerKeyId: "apple-issuer-2026Q2",
		},
	}
	_, hA, err := MarshalAndHash(withTokenA)
	if err != nil {
		t.Fatalf("MarshalAndHash A: %v", err)
	}
	_, hB, err := MarshalAndHash(withTokenB)
	if err != nil {
		t.Fatalf("MarshalAndHash B: %v", err)
	}
	if hA == hB {
		t.Fatalf("different token_hash produced same envelope hash: %x", hA)
	}
}

// TestAttestationObservation_CollectorRefDiscipline asserts collector_ref
// participation in hash derivation (parallel to NetworkObservation +
// BehavioralObservation tests). Required for phenomenon-vs-record OMQ
// resolution path.
func TestAttestationObservation_CollectorRefDiscipline(t *testing.T) {
	base := func() *eventsv1.AttestationObservation {
		return &eventsv1.AttestationObservation{
			ObservedAt:   1716120000000000000,
			ActorRef:     "actor-x",
			SessionRef:   "session-y",
			Modality: &eventsv1.AttestationObservation_CloudflareTurnstile{
				CloudflareTurnstile: &eventsv1.AttestationCloudflareTurnstile{
					ResponseTokenHash: bytes.Repeat([]byte{0x01}, 32),
					SiteKey:           "1x00000000000000000000AA",
					Action:            "login",
				},
			},
		}
	}
	fromBrowser := base()
	fromBrowser.CollectorRef = "browser-sdk:v1"
	fromHoneypot := base()
	fromHoneypot.CollectorRef = "honeypot-attestation-collector:hp-01"
	_, hBr, err := MarshalAndHash(fromBrowser)
	if err != nil {
		t.Fatalf("MarshalAndHash browser: %v", err)
	}
	_, hHp, err := MarshalAndHash(fromHoneypot)
	if err != nil {
		t.Fatalf("MarshalAndHash honeypot: %v", err)
	}
	if hBr == hHp {
		t.Fatalf("different collector_ref produced same hash: %x", hBr)
	}
}
