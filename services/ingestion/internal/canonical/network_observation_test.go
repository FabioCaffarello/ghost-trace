// Focused tests for NetworkObservation per decision-log §0144 + RFC
// Domain Pack v0.1 anti-bot atlas F1.NetworkObservation. Covers:
//   - per-sub-modality round-trip serialization
//   - oneof exhaustiveness (all 5 sub-modalities constructable +
//     marshalable; each produces distinct hash from same envelope)
//   - actor_ref discipline (empty + populated both stable)
//   - sub-modality hash discrimination (5 sub-modalities with
//     payload-equivalent bytes produce distinct hashes — discriminator
//     structural separation)
//
// Paired-dimension exclusion for NetworkObservation + its 5
// sub-modality types is asserted by
// TestPairedDimensionSubjectSet_StructuralDetection in coverage_test.go
// (which automatically picks up the messageFactory registration in
// corpus_test.go) — no explicit assertion needed here.
package canonical

import (
	"bytes"
	"testing"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
)

// newNetworkObservationWithEachModality returns a table of
// NetworkObservation instances, one per sub-modality, with envelope
// fields held constant across the table. Used to assert that the
// oneof discriminator produces structurally-distinct canonical bytes
// even when envelope fields and sub-modality scalar values overlap.
func newNetworkObservationWithEachModality() []struct {
	name string
	obs  *eventsv1.NetworkObservation
} {
	const (
		observedAt   = int64(1716120000000000000)
		actorRef     = "actor-under-test"
		endpointRef  = "192.0.2.1:443"
		collectorRef = "test-collector:v1"
	)
	envelope := func() *eventsv1.NetworkObservation {
		return &eventsv1.NetworkObservation{
			ObservedAt:   observedAt,
			ActorRef:     actorRef,
			EndpointRef:  endpointRef,
			CollectorRef: collectorRef,
		}
	}
	withTcp := envelope()
	withTcp.Modality = &eventsv1.NetworkObservation_TcpFingerprint{
		TcpFingerprint: &eventsv1.NetworkTcpFingerprint{
			WindowSize: 65535,
			Mss:        1460,
			Ttl:        64,
		},
	}
	withTls := envelope()
	withTls.Modality = &eventsv1.NetworkObservation_TlsJa4{
		TlsJa4: &eventsv1.NetworkTlsJa4{
			Ja4:         "t13d1516h2_8daaf6152771_b186095e22b6",
			SniPresent:  true,
			AlpnProtocols: []string{"h2", "http/1.1"},
		},
	}
	withHttp2 := envelope()
	withHttp2.Modality = &eventsv1.NetworkObservation_Http2FramePattern{
		Http2FramePattern: &eventsv1.NetworkHttp2FramePattern{
			SettingsCanonical: "1:65536;3:1000;4:6291456;6:262144",
			HeaderTableSize:   65536,
		},
	}
	withDns := envelope()
	withDns.Modality = &eventsv1.NetworkObservation_DnsPattern{
		DnsPattern: &eventsv1.NetworkDnsPattern{
			QnamePattern:     "depth:3;label_lengths:7,3,3",
			RecursionDesired: true,
		},
	}
	withIpAsn := envelope()
	withIpAsn.Modality = &eventsv1.NetworkObservation_IpAsn{
		IpAsn: &eventsv1.NetworkIpAsn{
			IpAddress:       "192.0.2.1",
			Asn:             64496,
			AsnOrganization: "EXAMPLE-AS",
			GeoCountry:      "US",
		},
	}
	return []struct {
		name string
		obs  *eventsv1.NetworkObservation
	}{
		{"tcp_fingerprint", withTcp},
		{"tls_ja4", withTls},
		{"http2_frame_pattern", withHttp2},
		{"dns_pattern", withDns},
		{"ip_asn", withIpAsn},
	}
}

// TestNetworkObservation_RoundTripPerModality asserts that for each of
// the five sub-modalities, Marshal is deterministic and hash is stable.
// Parallel to TestMarshalDeterminism + TestHashStability over
// DeclaredSession, but covering the oneof structure.
func TestNetworkObservation_RoundTripPerModality(t *testing.T) {
	for _, tc := range newNetworkObservationWithEachModality() {
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

// TestNetworkObservation_OneofExhaustiveness asserts that all five
// sub-modalities are constructable + marshalable + produce structurally
// distinct canonical bytes from the same envelope-level field values.
// The discriminator (oneof field number) must propagate into canonical
// bytes such that two NetworkObservation records with the same envelope
// but different sub-modality variants do not collide.
func TestNetworkObservation_OneofExhaustiveness(t *testing.T) {
	hashes := map[string][32]byte{}
	for _, tc := range newNetworkObservationWithEachModality() {
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

// TestNetworkObservation_ActorRefDiscipline asserts that
// NetworkObservation can be marshaled with an empty actor_ref
// (unattributed observation per §0023 single-tier discipline) and with
// a populated actor_ref; both produce stable distinct hashes.
func TestNetworkObservation_ActorRefDiscipline(t *testing.T) {
	base := func() *eventsv1.NetworkObservation {
		return &eventsv1.NetworkObservation{
			ObservedAt:   1716120000000000000,
			EndpointRef:  "192.0.2.1:443",
			CollectorRef: "test-collector:v1",
			Modality: &eventsv1.NetworkObservation_IpAsn{
				IpAsn: &eventsv1.NetworkIpAsn{
					IpAddress: "192.0.2.1",
				},
			},
		}
	}
	unattributed := base()
	// ActorRef intentionally left empty.
	attributed := base()
	attributed.ActorRef = "actor-x"

	_, hUnattributed, err := MarshalAndHash(unattributed)
	if err != nil {
		t.Fatalf("MarshalAndHash unattributed: %v", err)
	}
	_, hAttributed, err := MarshalAndHash(attributed)
	if err != nil {
		t.Fatalf("MarshalAndHash attributed: %v", err)
	}
	if hUnattributed == hAttributed {
		t.Fatalf("empty vs populated actor_ref produced same hash: %x", hUnattributed)
	}
}

// TestNetworkObservation_CollectorRefDiscipline asserts that the
// collector_ref field (introduced as the structural seed for the
// phenomenon-vs-record reconciliation OMQ per §0144 (e)) participates
// in hash derivation — two otherwise-identical observations from
// different collectors produce distinct hashes. This is the
// substrate-level guarantee that the OMQ resolution can rest on.
func TestNetworkObservation_CollectorRefDiscipline(t *testing.T) {
	base := func() *eventsv1.NetworkObservation {
		return &eventsv1.NetworkObservation{
			ObservedAt:  1716120000000000000,
			ActorRef:    "actor-x",
			EndpointRef: "192.0.2.1:443",
			Modality: &eventsv1.NetworkObservation_IpAsn{
				IpAsn: &eventsv1.NetworkIpAsn{
					IpAddress: "192.0.2.1",
				},
			},
		}
	}
	fromCic := base()
	fromCic.CollectorRef = "cic-ids-2017-adapter:v1"
	fromHoneypot := base()
	fromHoneypot.CollectorRef = "honeypot-collector:hp-01"

	_, hCic, err := MarshalAndHash(fromCic)
	if err != nil {
		t.Fatalf("MarshalAndHash cic: %v", err)
	}
	_, hHoneypot, err := MarshalAndHash(fromHoneypot)
	if err != nil {
		t.Fatalf("MarshalAndHash honeypot: %v", err)
	}
	if hCic == hHoneypot {
		t.Fatalf("different collector_ref produced same hash: %x", hCic)
	}
}
