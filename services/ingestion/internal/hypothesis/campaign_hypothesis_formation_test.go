package hypothesis

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// campaignSubstrate constructs a fresh substrate and ingests one
// DeclaredSession per (descriptor, declaredAt) tuple. Returns the
// substrate. Mirrors §0045's populateSubstrate but parameterized
// over descriptor (since CampaignHypothesis groups by descriptor,
// not by actor).
func campaignSubstrate(t *testing.T, items []struct {
	Descriptor []byte
	DeclaredAt int64
}) *substrate.Substrate {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	in := ingest.New(sub, func() time.Time { return time.Unix(0, 1716120000000000777) })
	for i, it := range items {
		msg := &eventsv1.DeclaredSession{
			DeclaredAt:        it.DeclaredAt,
			ActorRef:          "actor-" + string(rune('a'+i%26)),
			SessionDescriptor: it.Descriptor,
		}
		if _, err := in.Append(ctx, msg, msg.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("Append %d: %v", i, err)
		}
	}
	return sub
}

func walkCampaignFormations(t *testing.T, sub *substrate.Substrate) []*eventsv1.CampaignHypothesisFormation {
	t.Helper()
	ctx := context.Background()
	var out []*eventsv1.CampaignHypothesisFormation
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != "ghosttrace.events.v1.CampaignHypothesisFormation" {
			return nil
		}
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return err
		}
		ev := &eventsv1.CampaignHypothesisFormation{}
		if err := proto.Unmarshal(payload, ev); err != nil {
			return err
		}
		out = append(out, ev)
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	return out
}

func TestTemporalDescriptorCohortV1HappyPath(t *testing.T) {
	// Five events sharing descriptor "alpha", arriving within
	// 60-second gaps each → one cohort of size 5.
	// Three events sharing descriptor "beta" with the same gap
	// pattern → second cohort of size 3.
	// Two events sharing descriptor "gamma" → cohort of size 2,
	// below min_campaign_size=3 → excluded.
	gap := int64(60 * 1e9) // 60s in ns
	sub := campaignSubstrate(t, []struct {
		Descriptor []byte
		DeclaredAt int64
	}{
		{[]byte("alpha"), 1000}, {[]byte("alpha"), 1000 + gap},
		{[]byte("alpha"), 1000 + 2*gap}, {[]byte("alpha"), 1000 + 3*gap},
		{[]byte("alpha"), 1000 + 4*gap},
		{[]byte("beta"), 5000}, {[]byte("beta"), 5000 + gap},
		{[]byte("beta"), 5000 + 2*gap},
		{[]byte("gamma"), 9000}, {[]byte("gamma"), 9000 + gap},
	})

	rep, err := FormCampaignHypothesisAll(context.Background(), sub,
		TemporalDescriptorCohortV1{MinCampaignSize: 3, MaxIntraEventGapSeconds: 300},
		func() time.Time { return time.Unix(0, 1) })
	if err != nil {
		t.Fatalf("FormCampaignHypothesisAll: %v", err)
	}
	if rep.NewlyFormed != 2 {
		t.Errorf("NewlyFormed: got %d, want 2 (alpha + beta cohorts; gamma excluded)", rep.NewlyFormed)
	}
	forms := walkCampaignFormations(t, sub)
	if len(forms) != 2 {
		t.Fatalf("substrate carries %d formations, want 2", len(forms))
	}
	for _, f := range forms {
		if f.PatternSignature != TemporalDescriptorCohortV1Signature {
			t.Errorf("pattern_signature: got %q", f.PatternSignature)
		}
		if f.PatternParameters != "max_intra_event_gap_seconds=300;min_campaign_size=3" {
			t.Errorf("pattern_parameters: got %q", f.PatternParameters)
		}
		if len(f.SourceEventHashes) < 3 {
			t.Errorf("source_event_hashes: got %d, want >= 3", len(f.SourceEventHashes))
		}
	}
}

func TestTemporalDescriptorCohortV1RespectsGapBreak(t *testing.T) {
	// Five events sharing descriptor "alpha"; the gap between #3
	// and #4 EXCEEDS the threshold, so they split into TWO
	// cohorts. With min_campaign_size=3, only cohorts of size >=3
	// qualify. Sub-3 cohorts are excluded.
	smallGap := int64(60 * 1e9)
	bigGap := int64(1000 * 1e9) // 1000s > 300s threshold
	sub := campaignSubstrate(t, []struct {
		Descriptor []byte
		DeclaredAt int64
	}{
		{[]byte("alpha"), 1000},
		{[]byte("alpha"), 1000 + smallGap},
		{[]byte("alpha"), 1000 + 2*smallGap},
		{[]byte("alpha"), 1000 + 2*smallGap + bigGap},
		{[]byte("alpha"), 1000 + 2*smallGap + bigGap + smallGap},
	})

	rep, err := FormCampaignHypothesisAll(context.Background(), sub,
		TemporalDescriptorCohortV1{MinCampaignSize: 3, MaxIntraEventGapSeconds: 300},
		nil)
	if err != nil {
		t.Fatalf("FormCampaignHypothesisAll: %v", err)
	}
	// First cohort: 3 events → qualifies.
	// Second cohort: 2 events → excluded.
	if rep.NewlyFormed != 1 {
		t.Errorf("NewlyFormed: got %d, want 1 (first cohort only)", rep.NewlyFormed)
	}
	forms := walkCampaignFormations(t, sub)
	if len(forms) != 1 || len(forms[0].SourceEventHashes) != 3 {
		t.Errorf("expected 1 formation with 3 source events; got %v", forms)
	}
}

func TestTemporalDescriptorCohortV1Idempotent(t *testing.T) {
	gap := int64(60 * 1e9)
	sub := campaignSubstrate(t, []struct {
		Descriptor []byte
		DeclaredAt int64
	}{
		{[]byte("alpha"), 1000}, {[]byte("alpha"), 1000 + gap}, {[]byte("alpha"), 1000 + 2*gap},
	})

	ctx := context.Background()
	pattern := TemporalDescriptorCohortV1{MinCampaignSize: 3, MaxIntraEventGapSeconds: 300}
	rep1, _ := FormCampaignHypothesisAll(ctx, sub, pattern, nil)
	rep2, _ := FormCampaignHypothesisAll(ctx, sub, pattern, nil)
	if rep1.NewlyFormed != 1 {
		t.Fatalf("first run NewlyFormed: got %d, want 1", rep1.NewlyFormed)
	}
	if rep2.NewlyFormed != 0 {
		t.Errorf("second run NewlyFormed: got %d, want 0 (idempotent)", rep2.NewlyFormed)
	}
	if rep2.AlreadyFormed != 1 {
		t.Errorf("second run AlreadyFormed: got %d, want 1", rep2.AlreadyFormed)
	}
}

func TestTemporalDescriptorCohortV1VersioningProducesNewRecord(t *testing.T) {
	gap := int64(60 * 1e9)
	sub := campaignSubstrate(t, []struct {
		Descriptor []byte
		DeclaredAt int64
	}{
		{[]byte("alpha"), 1000}, {[]byte("alpha"), 1000 + gap}, {[]byte("alpha"), 1000 + 2*gap}, {[]byte("alpha"), 1000 + 3*gap},
	})

	ctx := context.Background()
	if _, err := FormCampaignHypothesisAll(ctx, sub,
		TemporalDescriptorCohortV1{MinCampaignSize: 3, MaxIntraEventGapSeconds: 300}, nil); err != nil {
		t.Fatalf("first FormCampaignHypothesisAll: %v", err)
	}
	rep, err := FormCampaignHypothesisAll(ctx, sub,
		TemporalDescriptorCohortV1{MinCampaignSize: 4, MaxIntraEventGapSeconds: 300}, nil)
	if err != nil {
		t.Fatalf("second FormCampaignHypothesisAll: %v", err)
	}
	if rep.NewlyFormed != 1 {
		t.Errorf("changing MinCampaignSize did not produce new record; NewlyFormed=%d, want 1", rep.NewlyFormed)
	}
	if got := len(walkCampaignFormations(t, sub)); got != 2 {
		t.Errorf("substrate should hold 2 formations after two parameterizations; got %d", got)
	}
}

func TestTemporalDescriptorCohortV1EmptySubstrate(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	rep, err := FormCampaignHypothesisAll(ctx, sub,
		TemporalDescriptorCohortV1{MinCampaignSize: 3, MaxIntraEventGapSeconds: 300}, nil)
	if err != nil {
		t.Fatalf("FormCampaignHypothesisAll on empty: %v", err)
	}
	if rep.Examined != 0 || rep.NewlyFormed != 0 {
		t.Errorf("empty substrate: %+v, want zero", rep)
	}
}

func TestTemporalDescriptorCohortV1FormationAtIsMaxDeclaredAt(t *testing.T) {
	gap := int64(60 * 1e9)
	sub := campaignSubstrate(t, []struct {
		Descriptor []byte
		DeclaredAt int64
	}{
		{[]byte("alpha"), 1000}, {[]byte("alpha"), 1000 + gap}, {[]byte("alpha"), 1000 + 2*gap},
	})

	if _, err := FormCampaignHypothesisAll(context.Background(), sub,
		TemporalDescriptorCohortV1{MinCampaignSize: 3, MaxIntraEventGapSeconds: 300}, nil); err != nil {
		t.Fatalf("FormCampaignHypothesisAll: %v", err)
	}
	forms := walkCampaignFormations(t, sub)
	if len(forms) != 1 {
		t.Fatalf("expected 1 formation; got %d", len(forms))
	}
	want := int64(1000 + 2*gap)
	if forms[0].FormationAt != want {
		t.Errorf("formation_at: got %d, want %d", forms[0].FormationAt, want)
	}
}

func TestTemporalDescriptorCohortV1CoexistsWithBCAndAG(t *testing.T) {
	// Substrate carrying all three Cat III subtypes' formations.
	// Confirms the three subtypes coexist as distinct
	// message_types per the §0056 + §0063 typed-subtype-landings
	// commitment.
	gap := int64(60 * 1e9)
	sub := campaignSubstrate(t, []struct {
		Descriptor []byte
		DeclaredAt int64
	}{
		{[]byte("alpha"), 1000}, {[]byte("alpha"), 1000 + gap}, {[]byte("alpha"), 1000 + 2*gap},
	})
	ctx := context.Background()

	if _, err := FormCampaignHypothesisAll(ctx, sub,
		TemporalDescriptorCohortV1{MinCampaignSize: 3, MaxIntraEventGapSeconds: 300}, nil); err != nil {
		t.Fatalf("FormCampaignHypothesisAll: %v", err)
	}

	// Add observations to support a BehavioralCluster (two actors
	// sharing a descriptor) and an AutomationGroup (uniform-cadence
	// single actor) alongside.
	in := ingest.New(sub, func() time.Time { return time.Unix(0, 99999) })
	for _, actor := range []string{"share-a", "share-b"} {
		msg := &eventsv1.DeclaredSession{
			DeclaredAt:        99999,
			ActorRef:          actor,
			SessionDescriptor: []byte("shared-bc"),
		}
		if _, err := in.Append(ctx, msg, msg.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("Append BC: %v", err)
		}
	}
	for i := 0; i < 5; i++ {
		msg := &eventsv1.DeclaredSession{
			DeclaredAt:        int64(200000 + i*1000),
			ActorRef:          "bot-ag",
			SessionDescriptor: []byte("ag-bot"),
		}
		if _, err := in.Append(ctx, msg, msg.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("Append AG: %v", err)
		}
	}
	if _, err := FormAll(ctx, sub,
		SessionDescriptorSharedV1{MinClusterSize: 2}, func() time.Time { return time.Unix(0, 300000) }); err != nil {
		t.Fatalf("FormAll BC: %v", err)
	}
	if _, err := FormAutomationGroupAll(ctx, sub,
		UniformCadenceV1{MinObservationCount: 5, MaxCoVThreshold: 0.15}, nil); err != nil {
		t.Fatalf("FormAutomationGroupAll: %v", err)
	}

	typeCounts := map[string]int{}
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		typeCounts[row.MessageType]++
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	for _, mt := range []string{
		"ghosttrace.events.v1.BehavioralClusterFormation",
		"ghosttrace.events.v1.AutomationGroupFormation",
		"ghosttrace.events.v1.CampaignHypothesisFormation",
	} {
		if typeCounts[mt] < 1 {
			t.Errorf("substrate missing %s", mt)
		}
	}
}

func TestTemporalDescriptorCohortV1SourceEventHashesSorted(t *testing.T) {
	gap := int64(60 * 1e9)
	sub := campaignSubstrate(t, []struct {
		Descriptor []byte
		DeclaredAt int64
	}{
		{[]byte("alpha"), 1000}, {[]byte("alpha"), 1000 + gap}, {[]byte("alpha"), 1000 + 2*gap},
	})
	if _, err := FormCampaignHypothesisAll(context.Background(), sub,
		TemporalDescriptorCohortV1{MinCampaignSize: 3, MaxIntraEventGapSeconds: 300}, nil); err != nil {
		t.Fatalf("FormCampaignHypothesisAll: %v", err)
	}
	forms := walkCampaignFormations(t, sub)
	if len(forms) != 1 {
		t.Fatalf("expected 1 formation; got %d", len(forms))
	}
	hashes := forms[0].SourceEventHashes
	for i := 1; i < len(hashes); i++ {
		if bytes.Compare(hashes[i-1], hashes[i]) >= 0 {
			t.Errorf("source_event_hashes not sorted ascending at index %d", i)
		}
	}
}
