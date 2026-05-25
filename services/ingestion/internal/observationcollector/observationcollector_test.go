package observationcollector

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
	return sub, ingest.New(sub, time.Now)
}

// TestCollectBehavioral_FiltersOnMessageType verifies CollectBehavioral
// returns only BehavioralObservation records, excluding paired
// IngestionEvent records emitted by Ingester.Append.
func TestCollectBehavioral_FiltersOnMessageType(t *testing.T) {
	sub, in := newTestSubstrate(t)
	ctx := context.Background()
	for i, actor := range []string{"actor-a", "actor-b", "actor-c"} {
		obs := &eventsv1.BehavioralObservation{
			ObservedAt:          int64(1000 + i),
			ActorRef:            actor,
			CollectorRef:        "browser-sdk:v1",
			AuthenticationClass: commonv1.AuthenticationClass_AUTHENTICATION_CLASS_CLIENT_WITNESSED,
			Modality: &eventsv1.BehavioralObservation_KeystrokeTiming{
				KeystrokeTiming: &eventsv1.BehavioralKeystrokeTiming{
					Intervals: []*eventsv1.KeystrokeInterval{
						{FlightNs: 50_000_000, DwellNs: 100_000_000},
					},
					TotalKeystrokeCount: 1,
				},
			},
		}
		if _, err := in.Append(ctx, obs, obs.ObservedAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("Append %d: %v", i, err)
		}
	}
	out, err := CollectBehavioral(ctx, sub)
	if err != nil {
		t.Fatalf("CollectBehavioral: %v", err)
	}
	if len(out) != 3 {
		t.Errorf("got %d BehavioralObservations, want 3 (IngestionEvent must be excluded)", len(out))
	}
}

// TestCollectBrowser_FiltersOnMessageType verifies CollectBrowser
// returns only BrowserObservation records.
func TestCollectBrowser_FiltersOnMessageType(t *testing.T) {
	sub, in := newTestSubstrate(t)
	ctx := context.Background()
	for i, actor := range []string{"actor-a", "actor-b"} {
		obs := &eventsv1.BrowserObservation{
			ObservedAt:          int64(1000 + i),
			ActorRef:            actor,
			CollectorRef:        "browser-sdk:v1",
			AuthenticationClass: commonv1.AuthenticationClass_AUTHENTICATION_CLASS_CLIENT_WITNESSED,
			Modality: &eventsv1.BrowserObservation_CdpMarker{
				CdpMarker: &eventsv1.BrowserCDPMarker{
					MarkersObserved: []string{"navigator.webdriver=true"},
					DetectionCount:  1,
				},
			},
		}
		if _, err := in.Append(ctx, obs, obs.ObservedAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("Append %d: %v", i, err)
		}
	}
	out, err := CollectBrowser(ctx, sub)
	if err != nil {
		t.Fatalf("CollectBrowser: %v", err)
	}
	if len(out) != 2 {
		t.Errorf("got %d BrowserObservations, want 2", len(out))
	}
}

// TestCollectNetwork_FiltersOnMessageType verifies CollectNetwork
// returns only NetworkObservation records.
func TestCollectNetwork_FiltersOnMessageType(t *testing.T) {
	sub, in := newTestSubstrate(t)
	ctx := context.Background()
	for i, actor := range []string{"actor-a", "actor-b", "actor-c"} {
		obs := &eventsv1.NetworkObservation{
			ObservedAt:          int64(1000 + i),
			ActorRef:            actor,
			EndpointRef:         "10.0.0.1:443",
			CollectorRef:        "test-collector:v1",
			AuthenticationClass: commonv1.AuthenticationClass_AUTHENTICATION_CLASS_SERVER_AUTHENTICATED,
			Modality: &eventsv1.NetworkObservation_TcpFingerprint{
				TcpFingerprint: &eventsv1.NetworkTcpFingerprint{WindowSize: 65535},
			},
		}
		if _, err := in.Append(ctx, obs, obs.ObservedAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("Append %d: %v", i, err)
		}
	}
	out, err := CollectNetwork(ctx, sub)
	if err != nil {
		t.Fatalf("CollectNetwork: %v", err)
	}
	if len(out) != 3 {
		t.Errorf("got %d NetworkObservations, want 3", len(out))
	}
}

// TestCollect_EmptySubstrate verifies all three Collect* functions
// return (nil, nil) on an empty substrate.
func TestCollect_EmptySubstrate(t *testing.T) {
	sub, _ := newTestSubstrate(t)
	ctx := context.Background()
	if out, err := CollectBehavioral(ctx, sub); err != nil || len(out) != 0 {
		t.Errorf("CollectBehavioral: got len=%d err=%v, want len=0 err=nil", len(out), err)
	}
	if out, err := CollectBrowser(ctx, sub); err != nil || len(out) != 0 {
		t.Errorf("CollectBrowser: got len=%d err=%v, want len=0 err=nil", len(out), err)
	}
	if out, err := CollectNetwork(ctx, sub); err != nil || len(out) != 0 {
		t.Errorf("CollectNetwork: got len=%d err=%v, want len=0 err=nil", len(out), err)
	}
}
