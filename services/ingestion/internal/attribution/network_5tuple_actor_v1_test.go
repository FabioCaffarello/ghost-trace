package attribution

import (
	"context"
	"path/filepath"
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

// newNetworkObservation returns a NetworkObservation with the given
// endpoint_ref + modality choice. actor_ref is left empty (the
// derivation target case).
func newNetworkObservationForAttribution(endpointRef string, modality interface{}, observedAt int64) *eventsv1.NetworkObservation {
	obs := &eventsv1.NetworkObservation{
		ObservedAt:          observedAt,
		EndpointRef:         endpointRef,
		CollectorRef:        "test-collector:v1",
		AuthenticationClass: commonv1.AuthenticationClass_AUTHENTICATION_CLASS_SERVER_AUTHENTICATED,
	}
	switch m := modality.(type) {
	case *eventsv1.NetworkObservation_IpAsn:
		obs.Modality = m
	case *eventsv1.NetworkObservation_TcpFingerprint:
		obs.Modality = m
	}
	return obs
}

func TestNetwork5TupleActorV1_ContractFields(t *testing.T) {
	def := Network5TupleActorV1{}
	if def.Version() != "network-5tuple-actor-v1" {
		t.Errorf("Version: got %q want network-5tuple-actor-v1", def.Version())
	}
	if def.Parameters() != "" {
		t.Errorf("Parameters: got %q want empty (v1 takes no parameters)", def.Parameters())
	}
}

func TestNetwork5TupleActorV1_Derive_TcpFingerprintModality(t *testing.T) {
	def := Network5TupleActorV1{}
	obs := newNetworkObservationForAttribution(
		"192.0.2.10:49152",
		&eventsv1.NetworkObservation_TcpFingerprint{
			TcpFingerprint: &eventsv1.NetworkTcpFingerprint{WindowSize: 65535},
		},
		1716120000000000000,
	)
	att, ok := def.Derive(obs)
	if !ok {
		t.Fatal("Derive: ok=false; expected true for valid tcp_fingerprint obs")
	}
	if att.DerivedActorRef != "192.0.2.10:49152/tcp" {
		t.Errorf("DerivedActorRef: got %q want 192.0.2.10:49152/tcp", att.DerivedActorRef)
	}
	if att.Confidence != 1.0 {
		t.Errorf("Confidence: got %v want 1.0", att.Confidence)
	}
	if att.EvidentialIndependence == nil {
		t.Fatal("EvidentialIndependence is nil; paired-dimension violation")
	}
	if att.EvidentialIndependence.Numerator != 1 || att.EvidentialIndependence.Denominator != 1 {
		t.Errorf("EvidentialIndependence: got %d/%d want 1/1",
			att.EvidentialIndependence.Numerator, att.EvidentialIndependence.Denominator)
	}
}

func TestNetwork5TupleActorV1_Derive_IpAsnModality(t *testing.T) {
	def := Network5TupleActorV1{}
	obs := newNetworkObservationForAttribution(
		"198.51.100.5:443",
		&eventsv1.NetworkObservation_IpAsn{
			IpAsn: &eventsv1.NetworkIpAsn{IpAddress: "198.51.100.5"},
		},
		1716120000000000000,
	)
	att, ok := def.Derive(obs)
	if !ok {
		t.Fatal("Derive: ok=false; expected true for valid ip_asn obs with endpoint_ref")
	}
	if att.DerivedActorRef != "198.51.100.5:443/ip" {
		t.Errorf("DerivedActorRef: got %q want 198.51.100.5:443/ip (ip_asn → ip)", att.DerivedActorRef)
	}
}

func TestNetwork5TupleActorV1_Derive_EmptyEndpoint_Skip(t *testing.T) {
	def := Network5TupleActorV1{}
	obs := newNetworkObservationForAttribution(
		"",
		&eventsv1.NetworkObservation_TcpFingerprint{
			TcpFingerprint: &eventsv1.NetworkTcpFingerprint{},
		},
		1716120000000000000,
	)
	_, ok := def.Derive(obs)
	if ok {
		t.Error("Derive: ok=true for empty endpoint_ref; expected false (skip)")
	}
}

func TestNetwork5TupleActorV1_Derive_UnparseableEndpoint_Skip(t *testing.T) {
	def := Network5TupleActorV1{}
	cases := []string{
		"no-colon-here",
		":443",
		"host:not-numeric",
		"host:",
	}
	for _, ep := range cases {
		obs := newNetworkObservationForAttribution(
			ep,
			&eventsv1.NetworkObservation_TcpFingerprint{
				TcpFingerprint: &eventsv1.NetworkTcpFingerprint{},
			},
			1716120000000000000,
		)
		if _, ok := def.Derive(obs); ok {
			t.Errorf("Derive(%q): ok=true; expected false (unparseable endpoint)", ep)
		}
	}
}

func TestNetwork5TupleActorV1_Derive_Determinism(t *testing.T) {
	def := Network5TupleActorV1{}
	obs := newNetworkObservationForAttribution(
		"10.0.0.1:8080",
		&eventsv1.NetworkObservation_TcpFingerprint{
			TcpFingerprint: &eventsv1.NetworkTcpFingerprint{},
		},
		1716120000000000000,
	)
	a, _ := def.Derive(obs)
	b, _ := def.Derive(obs)
	if a.DerivedActorRef != b.DerivedActorRef {
		t.Errorf("non-deterministic Derive: a=%q b=%q", a.DerivedActorRef, b.DerivedActorRef)
	}
	if a.Confidence != b.Confidence {
		t.Errorf("non-deterministic Confidence: a=%v b=%v", a.Confidence, b.Confidence)
	}
}

// TestDeriveAll_EndToEnd commits 3 NetworkObservation records (2 with
// distinct endpoint_ref + valid modalities; 1 with empty endpoint_ref
// → skipped), runs DeriveAll, verifies report counts + substrate has
// the expected 2 Cat II records.
func TestDeriveAll_EndToEnd(t *testing.T) {
	sub, in := newTestSubstrate(t)
	ctx := context.Background()
	clock := func() time.Time { return time.Unix(0, 1716120000000000777) }

	// Two valid sources + one skipped.
	obs1 := newNetworkObservationForAttribution(
		"192.0.2.10:49152",
		&eventsv1.NetworkObservation_TcpFingerprint{TcpFingerprint: &eventsv1.NetworkTcpFingerprint{}},
		1716120000000000000,
	)
	obs2 := newNetworkObservationForAttribution(
		"192.0.2.11:53124",
		&eventsv1.NetworkObservation_TcpFingerprint{TcpFingerprint: &eventsv1.NetworkTcpFingerprint{}},
		1716120000000000001,
	)
	obs3 := newNetworkObservationForAttribution(
		"", // empty endpoint → skip
		&eventsv1.NetworkObservation_TcpFingerprint{TcpFingerprint: &eventsv1.NetworkTcpFingerprint{}},
		1716120000000000002,
	)
	for _, o := range []*eventsv1.NetworkObservation{obs1, obs2, obs3} {
		if _, err := in.Append(ctx, o, o.ObservedAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}

	def := Network5TupleActorV1{}
	rep, err := DeriveAll(ctx, sub, def, clock)
	if err != nil {
		t.Fatalf("DeriveAll: %v", err)
	}
	if rep.Examined != 3 {
		t.Errorf("Examined: got %d want 3", rep.Examined)
	}
	if rep.Skipped != 1 {
		t.Errorf("Skipped: got %d want 1 (obs3 empty endpoint)", rep.Skipped)
	}
	if rep.NewlyDerived != 2 {
		t.Errorf("NewlyDerived: got %d want 2", rep.NewlyDerived)
	}
	if rep.AlreadyDerived != 0 {
		t.Errorf("AlreadyDerived: got %d want 0 (first run)", rep.AlreadyDerived)
	}

	// Second run → all idempotent.
	rep2, err := DeriveAll(ctx, sub, def, clock)
	if err != nil {
		t.Fatalf("DeriveAll second: %v", err)
	}
	if rep2.NewlyDerived != 0 {
		t.Errorf("second run NewlyDerived: got %d want 0", rep2.NewlyDerived)
	}
	if rep2.AlreadyDerived != 2 {
		t.Errorf("second run AlreadyDerived: got %d want 2", rep2.AlreadyDerived)
	}
}

// TestCollectAttributionView_LookupAfterDerive runs DeriveAll then
// CollectAttributionView and confirms For(sourceHash) returns the
// expected derived_actor_ref for each Cat I source.
func TestCollectAttributionView_LookupAfterDerive(t *testing.T) {
	sub, in := newTestSubstrate(t)
	ctx := context.Background()
	clock := func() time.Time { return time.Unix(0, 1716120000000000777) }

	obs := newNetworkObservationForAttribution(
		"203.0.113.5:443",
		&eventsv1.NetworkObservation_TcpFingerprint{TcpFingerprint: &eventsv1.NetworkTcpFingerprint{}},
		1716120000000000000,
	)
	rep, err := in.Append(ctx, obs, obs.ObservedAt, ingest.Envelope{Channel: "test"})
	if err != nil {
		t.Fatalf("Append: %v", err)
	}
	var srcHash [32]byte
	if _, err := hexDecodeInto(rep.EventHashHex, srcHash[:]); err != nil {
		t.Fatalf("decode source hash: %v", err)
	}

	if _, err := DeriveAll(ctx, sub, Network5TupleActorV1{}, clock); err != nil {
		t.Fatalf("DeriveAll: %v", err)
	}

	view, err := CollectAttributionView(ctx, sub)
	if err != nil {
		t.Fatalf("CollectAttributionView: %v", err)
	}
	derivedRef, attHash, ok := view.For(srcHash)
	if !ok {
		t.Fatal("view.For: ok=false; expected true after DeriveAll committed Cat II record")
	}
	if derivedRef != "203.0.113.5:443/tcp" {
		t.Errorf("derived ref: got %q want 203.0.113.5:443/tcp", derivedRef)
	}
	if attHash == ([32]byte{}) {
		t.Error("attribution hash is zero; expected populated")
	}
}

func TestCollectAttributionView_NoDerivations(t *testing.T) {
	sub, _ := newTestSubstrate(t)
	ctx := context.Background()

	view, err := CollectAttributionView(ctx, sub)
	if err != nil {
		t.Fatalf("CollectAttributionView: %v", err)
	}
	derivedRef, _, ok := view.For([32]byte{0x01, 0x02, 0x03})
	if ok {
		t.Errorf("view.For on empty substrate: ok=true derivedRef=%q; expected false", derivedRef)
	}
}

func TestEmptyAttributionView(t *testing.T) {
	v := EmptyAttributionView{}
	if _, _, ok := v.For([32]byte{0xff}); ok {
		t.Error("EmptyAttributionView.For: ok=true; expected always false")
	}
}

// hexDecodeInto is a local hex helper for test (parallel to the one in
// find-automation-group-candidates package; kept package-local).
func hexDecodeInto(hexStr string, dst []byte) (int, error) {
	if len(hexStr) != 2*len(dst) {
		return 0, &hexLenErr{}
	}
	for i := 0; i < len(dst); i++ {
		hi, lo := hexNibble(hexStr[2*i]), hexNibble(hexStr[2*i+1])
		dst[i] = byte(hi<<4 | lo)
	}
	return len(dst), nil
}

type hexLenErr struct{}

func (e *hexLenErr) Error() string { return "hex length mismatch" }

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
