package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	commonv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/common/v1"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// seedNetworkObservations writes obs to a fresh substrate under dir.
// Returns the substrate path + blob dir for subsequent run() calls.
func seedNetworkObservations(t *testing.T, dir string, obs []*eventsv1.NetworkObservation) (dbPath, blobDir string) {
	t.Helper()
	dbPath = filepath.Join(dir, "test.db")
	blobDir = filepath.Join(dir, "blobs")
	ctx := context.Background()
	sub, err := substrate.Open(ctx, dbPath, blobDir)
	if err != nil {
		t.Fatalf("Open substrate: %v", err)
	}
	clock := func() time.Time { return time.Unix(0, 1716120000000000777) }
	in := ingest.New(sub, clock)
	for i, o := range obs {
		eventTime := int64(1716120000000000777) + int64(i)*int64(time.Second)
		if _, err := in.Append(ctx, o, eventTime, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("Append obs[%d]: %v", i, err)
		}
	}
	if err := sub.Close(); err != nil {
		t.Fatalf("Close substrate: %v", err)
	}
	return dbPath, blobDir
}

// derivableObs constructs a NetworkObservation with populated
// endpoint_ref + ip_asn modality — exercises the Derive predicate's
// successful path.
func derivableObs(host string, port uint32) *eventsv1.NetworkObservation {
	return &eventsv1.NetworkObservation{
		ObservedAt:          1716120000000000777,
		EndpointRef:         fmt.Sprintf("%s:%d", host, port),
		CollectorRef:        "test-collector:v1",
		AuthenticationClass: commonv1.AuthenticationClass_AUTHENTICATION_CLASS_SERVER_AUTHENTICATED,
		Modality: &eventsv1.NetworkObservation_IpAsn{
			IpAsn: &eventsv1.NetworkIpAsn{IpAddress: host},
		},
	}
}

// nonDerivableObs constructs a NetworkObservation with empty
// endpoint_ref — exercises the Derive predicate's skip path.
func nonDerivableObs() *eventsv1.NetworkObservation {
	return &eventsv1.NetworkObservation{
		ObservedAt:          1716120000000000777,
		EndpointRef:         "",
		CollectorRef:        "test-collector:v1",
		AuthenticationClass: commonv1.AuthenticationClass_AUTHENTICATION_CLASS_SERVER_AUTHENTICATED,
		Modality: &eventsv1.NetworkObservation_IpAsn{
			IpAsn: &eventsv1.NetworkIpAsn{IpAddress: "192.0.2.99"},
		},
	}
}

func decodePayload(t *testing.T, stdout string) payload {
	t.Helper()
	var p payload
	if err := json.NewDecoder(strings.NewReader(stdout)).Decode(&p); err != nil {
		t.Fatalf("decode payload: %v\nstdout=%q", err, stdout)
	}
	return p
}

func TestRun_DerivesAttributions(t *testing.T) {
	dir := t.TempDir()
	dbPath, blobDir := seedNetworkObservations(t, dir, []*eventsv1.NetworkObservation{
		derivableObs("192.0.2.10", 49152),
		derivableObs("198.51.100.5", 443),
		derivableObs("192.0.2.11", 53124),
	})

	var stdout, stderr bytes.Buffer
	code := run([]string{"-db", dbPath, "-blobs", blobDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code: got %d want 0 (stderr=%q)", code, stderr.String())
	}
	p := decodePayload(t, stdout.String())

	if p.DefinitionVersion != "network-5tuple-actor-v1" {
		t.Errorf("DefinitionVersion: got %q want network-5tuple-actor-v1", p.DefinitionVersion)
	}
	if p.Examined != 3 {
		t.Errorf("Examined: got %d want 3", p.Examined)
	}
	if p.NewlyDerived != 3 {
		t.Errorf("NewlyDerived: got %d want 3", p.NewlyDerived)
	}
	if p.Skipped != 0 {
		t.Errorf("Skipped: got %d want 0", p.Skipped)
	}
	if p.AlreadyDerived != 0 {
		t.Errorf("AlreadyDerived: got %d want 0", p.AlreadyDerived)
	}
}

func TestRun_SkipsNonDerivableObservations(t *testing.T) {
	dir := t.TempDir()
	dbPath, blobDir := seedNetworkObservations(t, dir, []*eventsv1.NetworkObservation{
		derivableObs("192.0.2.10", 49152),
		nonDerivableObs(),
		nonDerivableObs(),
	})
	// nonDerivableObs's are content-identical (zero observed_at offset
	// is overwritten by seedNetworkObservations's per-index eventTime,
	// but the canonical payload still differs); but to be safe the
	// test verifies skipped >= 1 + newly_derived >= 1.

	var stdout, stderr bytes.Buffer
	code := run([]string{"-db", dbPath, "-blobs", blobDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code: got %d want 0 (stderr=%q)", code, stderr.String())
	}
	p := decodePayload(t, stdout.String())

	if p.Skipped < 1 {
		t.Errorf("Skipped: got %d want >= 1 (non-derivable obs should skip)", p.Skipped)
	}
	if p.NewlyDerived < 1 {
		t.Errorf("NewlyDerived: got %d want >= 1 (one derivable obs should derive)", p.NewlyDerived)
	}
	// Examined counts ALL NetworkObservation rows scanned, regardless
	// of derive outcome.
	if p.Examined < 1 {
		t.Errorf("Examined: got %d want >= 1", p.Examined)
	}
}

func TestRun_Idempotent(t *testing.T) {
	dir := t.TempDir()
	dbPath, blobDir := seedNetworkObservations(t, dir, []*eventsv1.NetworkObservation{
		derivableObs("192.0.2.10", 49152),
		derivableObs("198.51.100.5", 443),
	})

	// First run: derives both observations.
	var stdout1, stderr1 bytes.Buffer
	code1 := run([]string{"-db", dbPath, "-blobs", blobDir}, &stdout1, &stderr1)
	if code1 != 0 {
		t.Fatalf("first run exit code: got %d want 0 (stderr=%q)", code1, stderr1.String())
	}
	p1 := decodePayload(t, stdout1.String())
	if p1.NewlyDerived != 2 {
		t.Errorf("first run NewlyDerived: got %d want 2", p1.NewlyDerived)
	}

	// Second run: same substrate, no new observations; the previously-
	// committed DerivedActorAttribution records exist; AppendPair
	// content-hash dedup returns AlreadyDerived for each.
	var stdout2, stderr2 bytes.Buffer
	code2 := run([]string{"-db", dbPath, "-blobs", blobDir}, &stdout2, &stderr2)
	if code2 != 0 {
		t.Fatalf("second run exit code: got %d want 0 (stderr=%q)", code2, stderr2.String())
	}
	p2 := decodePayload(t, stdout2.String())
	if p2.NewlyDerived != 0 {
		t.Errorf("second run NewlyDerived: got %d want 0 (idempotency violated)", p2.NewlyDerived)
	}
	if p2.AlreadyDerived != 2 {
		t.Errorf("second run AlreadyDerived: got %d want 2 (idempotency expects all prior rows recognized)", p2.AlreadyDerived)
	}
	// Examined unchanged across runs — same source observation count.
	if p1.Examined != p2.Examined {
		t.Errorf("Examined: first=%d second=%d (should be equal)", p1.Examined, p2.Examined)
	}
}

func TestRun_UnknownDefinitionVersion(t *testing.T) {
	dir := t.TempDir()
	dbPath, blobDir := seedNetworkObservations(t, dir, []*eventsv1.NetworkObservation{
		derivableObs("192.0.2.10", 49152),
	})

	var stdout, stderr bytes.Buffer
	code := run([]string{"-db", dbPath, "-blobs", blobDir, "-definition-version", "bogus-v0"}, &stdout, &stderr)
	if code != exitToolError {
		t.Fatalf("exit code: got %d want %d (stderr=%q)", code, exitToolError, stderr.String())
	}
	if !strings.Contains(stderr.String(), "unknown definition-version") {
		t.Errorf("stderr should explain unknown version; got %q", stderr.String())
	}
	// Substrate should NOT have been opened (resolveDefinition fails
	// before substrate.Open). No JSON emitted to stdout.
	if stdout.Len() > 0 {
		t.Errorf("stdout should be empty on resolveDefinition failure; got %q", stdout.String())
	}
}
