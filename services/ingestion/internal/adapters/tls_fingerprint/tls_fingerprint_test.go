package tls_fingerprint

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/observationcollector"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

func TestToObservation_MapsFields(t *testing.T) {
	rec := Record{
		ActorRef:      "actor-1",
		EndpointRef:   "10.0.0.1:443",
		JA4:           "ja4-x",
		JA4Raw:        "ja4-raw-x",
		JA3:           "ja3-x",
		JA3Raw:        "ja3-raw-x",
		SNIPresent:    true,
		ALPNProtocols: []string{"h2"},
	}
	obs := ToObservation(rec, 12345, "collector-x")
	if obs.GetActorRef() != "actor-1" || obs.GetEndpointRef() != "10.0.0.1:443" {
		t.Fatalf("envelope mapping wrong: %+v", obs)
	}
	if obs.GetObservedAt() != 12345 || obs.GetCollectorRef() != "collector-x" {
		t.Fatalf("observed_at/collector mapping wrong: %+v", obs)
	}
	tls := obs.GetTlsJa4()
	if tls == nil {
		t.Fatal("tls_ja4 modality not set")
	}
	if tls.GetJa4() != "ja4-x" || tls.GetJa4Raw() != "ja4-raw-x" {
		t.Errorf("JA4 fields wrong: %+v", tls)
	}
	if tls.GetJa3() != "ja3-x" || tls.GetJa3Raw() != "ja3-raw-x" {
		t.Errorf("JA3 fields wrong: %+v", tls)
	}
	if !tls.GetSniPresent() || len(tls.GetAlpnProtocols()) != 1 {
		t.Errorf("sni/alpn fields wrong: %+v", tls)
	}
}

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

func TestIngest_CommitsAndRejects(t *testing.T) {
	sub, in := newTestSubstrate(t)
	ctx := context.Background()

	input := strings.Join([]string{
		`{"actor_ref":"a1","ja4":"ja4-1","observed_at":100}`,    // valid (JA4 only)
		`{"actor_ref":"a2","ja3":"ja3-2","observed_at":200}`,    // valid (JA3 only)
		`{"actor_ref":"a3","observed_at":300}`,                  // reject: no fingerprint
		`not-json`,                                              // reject: malformed
		``,                                                      // blank: skipped, not parsed
		`{"actor_ref":"a4","ja4":"ja4-4","ja3":"ja3-4"}`,        // valid (both)
	}, "\n")

	report, err := Ingest(ctx, in, strings.NewReader(input), "collector-test", ingest.Envelope{Channel: "test"})
	if err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	if report.RowsParsed != 5 {
		t.Errorf("RowsParsed: got %d want 5 (blank line not counted)", report.RowsParsed)
	}
	if report.ObservationsCommitted != 3 {
		t.Errorf("ObservationsCommitted: got %d want 3", report.ObservationsCommitted)
	}
	if report.RowsRejected != 2 {
		t.Errorf("RowsRejected: got %d want 2 (no-fingerprint + malformed)", report.RowsRejected)
	}

	// Confirm the committed observations are retrievable as Cat I.
	obs, err := observationcollector.CollectNetwork(ctx, sub)
	if err != nil {
		t.Fatalf("CollectNetwork: %v", err)
	}
	if len(obs) != 3 {
		t.Fatalf("collected observations: got %d want 3", len(obs))
	}
}

func TestIngest_PerRecordCollectorOverride(t *testing.T) {
	sub, in := newTestSubstrate(t)
	ctx := context.Background()
	input := `{"actor_ref":"a1","ja4":"ja4-1","collector_ref":"per-record","observed_at":100}`
	if _, err := Ingest(ctx, in, strings.NewReader(input), "default-collector", ingest.Envelope{Channel: "test"}); err != nil {
		t.Fatalf("Ingest: %v", err)
	}
	obs, err := observationcollector.CollectNetwork(ctx, sub)
	if err != nil {
		t.Fatalf("CollectNetwork: %v", err)
	}
	if len(obs) != 1 || obs[0].GetCollectorRef() != "per-record" {
		t.Fatalf("per-record collector override not applied: %+v", obs)
	}
}
