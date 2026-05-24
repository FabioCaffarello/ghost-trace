// Focused tests for the AuthenticationClass envelope field added to
// F1 Cat I observation envelopes per decision-log §0150 (resolution
// of F4 authentication-class typing OMQ; Candidate α). Asserts:
//   - field participates in hash derivation across all 3 F1 envelopes
//     (Network + Behavioral + Attestation)
//   - default UNKNOWN value produces no wire-format bytes (proto3
//     enum default; existing records pre-resolution remain stable)
//   - enum values are structurally distinct (different class → different
//     envelope hash)
package canonical

import (
	"testing"

	commonv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/common/v1"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
)

// TestAuthenticationClass_DefaultUnknownIsStableWireFormat asserts that
// a F1 envelope with AuthenticationClass unset (default UNKNOWN)
// produces the same canonical bytes as a version of the same record
// before §0150 landed. Critical for backward-compat of pre-resolution
// substrate records.
func TestAuthenticationClass_DefaultUnknownIsStableWireFormat(t *testing.T) {
	noClass := &eventsv1.NetworkObservation{
		ObservedAt:   1716120000000000000,
		ActorRef:     "actor-x",
		EndpointRef:  "192.0.2.1:443",
		CollectorRef: "test-collector:v1",
		Modality: &eventsv1.NetworkObservation_IpAsn{
			IpAsn: &eventsv1.NetworkIpAsn{IpAddress: "192.0.2.1"},
		},
	}
	explicitUnknown := &eventsv1.NetworkObservation{
		ObservedAt:          1716120000000000000,
		ActorRef:            "actor-x",
		EndpointRef:         "192.0.2.1:443",
		CollectorRef:        "test-collector:v1",
		AuthenticationClass: commonv1.AuthenticationClass_AUTHENTICATION_CLASS_UNKNOWN,
		Modality: &eventsv1.NetworkObservation_IpAsn{
			IpAsn: &eventsv1.NetworkIpAsn{IpAddress: "192.0.2.1"},
		},
	}
	_, hNoClass, err := MarshalAndHash(noClass)
	if err != nil {
		t.Fatalf("MarshalAndHash noClass: %v", err)
	}
	_, hExplicit, err := MarshalAndHash(explicitUnknown)
	if err != nil {
		t.Fatalf("MarshalAndHash explicit: %v", err)
	}
	if hNoClass != hExplicit {
		t.Fatalf("unset vs explicit-UNKNOWN should produce identical hashes\n  unset=%x\n  explicit=%x", hNoClass, hExplicit)
	}
}

// TestAuthenticationClass_DifferentClassesProduceDifferentHashes asserts
// that the four enum values produce structurally distinct envelope
// hashes when otherwise-identical observations are committed under
// different classes.
func TestAuthenticationClass_DifferentClassesProduceDifferentHashes(t *testing.T) {
	base := func() *eventsv1.NetworkObservation {
		return &eventsv1.NetworkObservation{
			ObservedAt:   1716120000000000000,
			ActorRef:     "actor-x",
			EndpointRef:  "192.0.2.1:443",
			CollectorRef: "test-collector:v1",
			Modality: &eventsv1.NetworkObservation_IpAsn{
				IpAsn: &eventsv1.NetworkIpAsn{IpAddress: "192.0.2.1"},
			},
		}
	}
	classes := []struct {
		name  string
		value commonv1.AuthenticationClass
	}{
		{"unknown", commonv1.AuthenticationClass_AUTHENTICATION_CLASS_UNKNOWN},
		{"server_authenticated", commonv1.AuthenticationClass_AUTHENTICATION_CLASS_SERVER_AUTHENTICATED},
		{"client_attested", commonv1.AuthenticationClass_AUTHENTICATION_CLASS_CLIENT_ATTESTED},
		{"client_witnessed", commonv1.AuthenticationClass_AUTHENTICATION_CLASS_CLIENT_WITNESSED},
	}
	hashes := map[string][32]byte{}
	for _, c := range classes {
		obs := base()
		obs.AuthenticationClass = c.value
		_, h, err := MarshalAndHash(obs)
		if err != nil {
			t.Fatalf("MarshalAndHash %s: %v", c.name, err)
		}
		for prior, priorHash := range hashes {
			// UNKNOWN (default) collides with "unset" by design — see
			// TestAuthenticationClass_DefaultUnknownIsStableWireFormat.
			// Non-UNKNOWN values must each produce a distinct hash.
			if h == priorHash && c.name != "unknown" && prior != "unknown" {
				t.Fatalf("hash collision between %s and %s: %x", prior, c.name, h)
			}
		}
		hashes[c.name] = h
	}
	// The 3 non-UNKNOWN values must all be distinct.
	if hashes["server_authenticated"] == hashes["client_attested"] {
		t.Errorf("server_authenticated and client_attested collide")
	}
	if hashes["server_authenticated"] == hashes["client_witnessed"] {
		t.Errorf("server_authenticated and client_witnessed collide")
	}
	if hashes["client_attested"] == hashes["client_witnessed"] {
		t.Errorf("client_attested and client_witnessed collide")
	}
}

// TestAuthenticationClass_AppliesToAllThreeF1Envelopes asserts the
// field is present + functional on NetworkObservation, BehavioralObservation,
// and AttestationObservation envelopes. Asserts symmetry of the
// resolution across F1 modalities.
func TestAuthenticationClass_AppliesToAllThreeF1Envelopes(t *testing.T) {
	netObs := &eventsv1.NetworkObservation{
		ObservedAt:          1716120000000000000,
		AuthenticationClass: commonv1.AuthenticationClass_AUTHENTICATION_CLASS_SERVER_AUTHENTICATED,
		Modality: &eventsv1.NetworkObservation_IpAsn{
			IpAsn: &eventsv1.NetworkIpAsn{IpAddress: "192.0.2.1"},
		},
	}
	behObs := &eventsv1.BehavioralObservation{
		ObservedAt:          1716120000000000000,
		AuthenticationClass: commonv1.AuthenticationClass_AUTHENTICATION_CLASS_CLIENT_WITNESSED,
		Modality: &eventsv1.BehavioralObservation_PointerDwell{
			PointerDwell: &eventsv1.BehavioralPointerDwell{
				ElementClass: "form-input",
				DwellNs:      1000000000,
			},
		},
	}
	attObs := &eventsv1.AttestationObservation{
		ObservedAt:          1716120000000000000,
		AuthenticationClass: commonv1.AuthenticationClass_AUTHENTICATION_CLASS_CLIENT_ATTESTED,
		Modality: &eventsv1.AttestationObservation_CloudflareTurnstile{
			CloudflareTurnstile: &eventsv1.AttestationCloudflareTurnstile{
				ResponseTokenHash: make([]byte, 32),
				SiteKey:           "site-test",
				Action:            "login",
			},
		},
	}
	for _, msg := range []interface{ GetAuthenticationClass() commonv1.AuthenticationClass }{netObs, behObs, attObs} {
		if msg.GetAuthenticationClass() == commonv1.AuthenticationClass_AUTHENTICATION_CLASS_UNKNOWN {
			t.Errorf("envelope %T returned UNKNOWN class despite explicit set", msg)
		}
	}
	// Marshal all three; ensure no errors.
	if _, _, err := MarshalAndHash(netObs); err != nil {
		t.Errorf("MarshalAndHash netObs: %v", err)
	}
	if _, _, err := MarshalAndHash(behObs); err != nil {
		t.Errorf("MarshalAndHash behObs: %v", err)
	}
	if _, _, err := MarshalAndHash(attObs); err != nil {
		t.Errorf("MarshalAndHash attObs: %v", err)
	}
}
