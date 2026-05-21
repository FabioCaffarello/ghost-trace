package projection

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/hypothesis"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// formCampaignHypothesis populates the substrate with three
// temporal-descriptor-cohort DeclaredSessions for the supplied
// descriptor, runs FormCampaignHypothesisAll, and returns the
// hash of the newly-created CampaignHypothesisFormation. Mirrors
// §0062's formAutomationGroup but parameterized by descriptor since
// CampaignHypothesis groups by descriptor (event-centric subtype).
// Pre-walks the substrate to record the existing formation set;
// the new formation is the one that's added.
func formCampaignHypothesis(t *testing.T, sub *substrate.Substrate, descriptor string, declaredAtBase int64) [32]byte {
	t.Helper()
	ctx := context.Background()

	existing := map[[32]byte]bool{}
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType == campaignHypothesisFormationMessageType {
			existing[row.EventHash] = true
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents (pre): %v", err)
	}

	in := ingest.New(sub, func() time.Time { return time.Unix(0, declaredAtBase) })
	gap := int64(60 * 1e9)
	for i := 0; i < 3; i++ {
		msg := &eventsv1.DeclaredSession{
			DeclaredAt:        declaredAtBase + int64(i)*gap,
			ActorRef:          "actor-" + descriptor + "-" + string(rune('a'+i)),
			SessionDescriptor: []byte(descriptor),
		}
		if _, err := in.Append(ctx, msg, msg.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("ingest %d: %v", i, err)
		}
	}
	if _, err := hypothesis.FormCampaignHypothesisAll(ctx, sub,
		hypothesis.TemporalDescriptorCohortV1{MinCampaignSize: 3, MaxIntraEventGapSeconds: 300},
		func() time.Time { return time.Unix(0, declaredAtBase+10000) }); err != nil {
		t.Fatalf("FormCampaignHypothesisAll: %v", err)
	}
	var formationHash [32]byte
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != campaignHypothesisFormationMessageType {
			return nil
		}
		if existing[row.EventHash] {
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
		_ = ev
		formationHash = row.EventHash
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents (post): %v", err)
	}
	if formationHash == ([32]byte{}) {
		t.Fatalf("no new formation found for descriptor %q", descriptor)
	}
	return formationHash
}

func TestProjectCampaignHypothesisForming(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	formation := formCampaignHypothesis(t, sub, "camp-form", 1000)
	proj, err := ProjectCampaignHypothesis(ctx, sub, formation)
	if err != nil {
		t.Fatalf("ProjectCampaignHypothesis: %v", err)
	}
	if proj.State != StateForming {
		t.Errorf("state: got %q, want %q", proj.State, StateForming)
	}
	if proj.LatestPromotion != nil || proj.LatestDemotion != nil ||
		proj.Dissolution != nil || proj.MergedInto != nil || proj.SplitInto != nil {
		t.Errorf("forming projection should have no lifecycle pointers")
	}
	if len(proj.LifecycleHistory) != 1 {
		t.Errorf("lifecycle history: got %d, want 1 (formation only)", len(proj.LifecycleHistory))
	}
}

func TestProjectCampaignHypothesisPromoted(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	formation := formCampaignHypothesis(t, sub, "camp-prom", 1000)
	if _, err := hypothesis.PromoteCampaignHypothesis(ctx, sub, hypothesis.CampaignHypothesisPromoteOptions{
		FormationEventHash: formation,
		PromotedAt:         500000,
		CadenceSeconds:     60,
		Reason:             "operational pilot",
	}, nil); err != nil {
		t.Fatalf("PromoteCampaignHypothesis: %v", err)
	}

	proj, err := ProjectCampaignHypothesis(ctx, sub, formation)
	if err != nil {
		t.Fatalf("ProjectCampaignHypothesis: %v", err)
	}
	if proj.State != StatePromoted {
		t.Errorf("state: got %q, want %q", proj.State, StatePromoted)
	}
	if proj.LatestPromotion == nil || proj.LatestPromotion.Reason != "operational pilot" {
		t.Errorf("LatestPromotion: %+v", proj.LatestPromotion)
	}
}

func TestProjectCampaignHypothesisDissolved(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	formation := formCampaignHypothesis(t, sub, "camp-diss", 1000)
	if _, err := hypothesis.DissolveCampaignHypothesis(ctx, sub, hypothesis.CampaignHypothesisDissolveOptions{
		FormationEventHash: formation,
		DissolvedAt:        500000,
		Reason:             "campaign signature misattributed",
	}, nil); err != nil {
		t.Fatalf("DissolveCampaignHypothesis: %v", err)
	}

	proj, err := ProjectCampaignHypothesis(ctx, sub, formation)
	if err != nil {
		t.Fatalf("ProjectCampaignHypothesis: %v", err)
	}
	if proj.State != StateDissolved {
		t.Errorf("state: got %q, want %q", proj.State, StateDissolved)
	}
}

func TestProjectCampaignHypothesisUnknownFormation(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	var bogus [32]byte
	for i := range bogus {
		bogus[i] = byte(0xff)
	}
	_, err = ProjectCampaignHypothesis(ctx, sub, bogus)
	if !errors.Is(err, ErrFormationNotFound) {
		t.Errorf("expected ErrFormationNotFound; got %v", err)
	}
}

func TestProjectCampaignHypothesisCrossSubtypeRejected(t *testing.T) {
	// A BehavioralClusterFormation hash passed to
	// ProjectCampaignHypothesis MUST return ErrTargetNotFormation —
	// the CH projection's wrong-type check is subtype-specific.
	// Mirrors §0062's TestProjectAutomationGroupCrossSubtypeRejected.
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	// Form a BehavioralCluster.
	in := ingest.New(sub, func() time.Time { return time.Unix(0, 1000) })
	for _, actor := range []string{"actor-a", "actor-b"} {
		msg := &eventsv1.DeclaredSession{
			DeclaredAt:        1000,
			ActorRef:          actor,
			SessionDescriptor: []byte("shared"),
		}
		if _, err := in.Append(ctx, msg, msg.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}
	if _, err := hypothesis.FormAll(ctx, sub,
		hypothesis.SessionDescriptorSharedV1{MinClusterSize: 2}, func() time.Time { return time.Unix(0, 2000) }); err != nil {
		t.Fatalf("FormAll: %v", err)
	}
	var bcFormation [32]byte
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType == "ghosttrace.events.v1.BehavioralClusterFormation" {
			bcFormation = row.EventHash
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}

	_, err = ProjectCampaignHypothesis(ctx, sub, bcFormation)
	if !errors.Is(err, ErrTargetNotFormation) {
		t.Errorf("expected ErrTargetNotFormation for BC formation; got %v", err)
	}
}

func TestListCampaignHypothesesEmptyAndPopulated(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	got, err := ListCampaignHypotheses(ctx, sub, CampaignHypothesisListOptions{})
	if err != nil {
		t.Fatalf("ListCampaignHypotheses: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("empty substrate: got %d", len(got))
	}

	_ = formCampaignHypothesis(t, sub, "camp-list-a", 1000)
	_ = formCampaignHypothesis(t, sub, "camp-list-b", 1000000000)
	_ = formCampaignHypothesis(t, sub, "camp-list-c", 2000000000)

	got, err = ListCampaignHypotheses(ctx, sub, CampaignHypothesisListOptions{})
	if err != nil {
		t.Fatalf("ListCampaignHypotheses: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("populated: got %d, want 3", len(got))
	}
	// Ordering: ascending by FormationHash.
	for i := 1; i < len(got); i++ {
		if bytes.Compare(got[i-1].FormationHash[:], got[i].FormationHash[:]) >= 0 {
			t.Errorf("not ascending at %d", i)
		}
	}
}

func TestCountCampaignHypothesesByStateEquivalentToListFilter(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	_ = formCampaignHypothesis(t, sub, "camp-c-a", 1000)
	camp2 := formCampaignHypothesis(t, sub, "camp-c-b", 1000000000)
	camp3 := formCampaignHypothesis(t, sub, "camp-c-c", 2000000000)

	if _, err := hypothesis.PromoteCampaignHypothesis(ctx, sub, hypothesis.CampaignHypothesisPromoteOptions{
		FormationEventHash: camp2,
		PromotedAt:         1500000000,
		CadenceSeconds:     60,
	}, nil); err != nil {
		t.Fatalf("Promote: %v", err)
	}
	if _, err := hypothesis.DissolveCampaignHypothesis(ctx, sub, hypothesis.CampaignHypothesisDissolveOptions{
		FormationEventHash: camp3,
		DissolvedAt:        2500000000,
	}, nil); err != nil {
		t.Fatalf("Dissolve: %v", err)
	}

	counts, err := CountCampaignHypothesesByState(ctx, sub, CampaignHypothesisListOptions{})
	if err != nil {
		t.Fatalf("CountCampaignHypothesesByState: %v", err)
	}
	if counts.Total != 3 {
		t.Errorf("Total: got %d, want 3", counts.Total)
	}
	for state := range counts.ByState {
		list, err := ListCampaignHypotheses(ctx, sub, CampaignHypothesisListOptions{StateFilter: state})
		if err != nil {
			t.Fatalf("ListCampaignHypotheses(%q): %v", state, err)
		}
		if len(list) != counts.ByState[state] {
			t.Errorf("equivalence violated for state=%q: counts=%d, list=%d",
				state, counts.ByState[state], len(list))
		}
	}
}

func TestProjectAllCampaignHypothesesEquivalentToProjectCampaignHypothesis(t *testing.T) {
	// Same equivalence invariant as §0052 BC and §0062 AG —
	// per-formation projection from ProjectAll equals
	// ProjectCampaignHypothesis(sameFormation).
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	alpha := formCampaignHypothesis(t, sub, "camp-eq-a", 1000)
	beta := formCampaignHypothesis(t, sub, "camp-eq-b", 1000000000)

	if _, err := hypothesis.PromoteCampaignHypothesis(ctx, sub, hypothesis.CampaignHypothesisPromoteOptions{
		FormationEventHash: alpha,
		PromotedAt:         500000,
		CadenceSeconds:     60,
	}, nil); err != nil {
		t.Fatalf("Promote: %v", err)
	}
	if _, err := hypothesis.DissolveCampaignHypothesis(ctx, sub, hypothesis.CampaignHypothesisDissolveOptions{
		FormationEventHash: beta,
		DissolvedAt:        1500000000,
	}, nil); err != nil {
		t.Fatalf("Dissolve: %v", err)
	}

	all, err := ProjectAllCampaignHypotheses(ctx, sub)
	if err != nil {
		t.Fatalf("ProjectAllCampaignHypotheses: %v", err)
	}
	for _, formHash := range [][32]byte{alpha, beta} {
		single, err := ProjectCampaignHypothesis(ctx, sub, formHash)
		if err != nil {
			t.Fatalf("ProjectCampaignHypothesis(%x): %v", formHash, err)
		}
		multi, ok := all[formHash]
		if !ok {
			t.Errorf("ProjectAll missing %x", formHash)
			continue
		}
		if single.State != multi.State {
			t.Errorf("%x: single=%q multi=%q", formHash, single.State, multi.State)
		}
		if len(single.LifecycleHistory) != len(multi.LifecycleHistory) {
			t.Errorf("%x: single history=%d, multi=%d",
				formHash, len(single.LifecycleHistory), len(multi.LifecycleHistory))
		}
	}
}

func TestProjectCampaignHypothesisLatencies(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	formation := formCampaignHypothesis(t, sub, "camp-lat", 1000)

	// Read the formation's recorded formation_at to compute expected latencies.
	formationRow, err := sub.LookupRow(ctx, formation)
	if err != nil {
		t.Fatalf("LookupRow: %v", err)
	}
	formationAt := formationRow.EventTime

	if _, err := hypothesis.PromoteCampaignHypothesis(ctx, sub, hypothesis.CampaignHypothesisPromoteOptions{
		FormationEventHash: formation,
		PromotedAt:         500000,
		CadenceSeconds:     60,
	}, nil); err != nil {
		t.Fatalf("Promote: %v", err)
	}
	if _, err := hypothesis.DissolveCampaignHypothesis(ctx, sub, hypothesis.CampaignHypothesisDissolveOptions{
		FormationEventHash: formation,
		DissolvedAt:        800000,
	}, nil); err != nil {
		t.Fatalf("Dissolve: %v", err)
	}

	proj, err := ProjectCampaignHypothesis(ctx, sub, formation)
	if err != nil {
		t.Fatalf("ProjectCampaignHypothesis: %v", err)
	}
	if proj.FormationToFirstPromotionLatencyNs == nil {
		t.Fatal("FormationToFirstPromotionLatencyNs: got nil")
	}
	wantPromo := int64(500000) - formationAt
	if *proj.FormationToFirstPromotionLatencyNs != wantPromo {
		t.Errorf("FormationToFirstPromotionLatencyNs: got %d, want %d",
			*proj.FormationToFirstPromotionLatencyNs, wantPromo)
	}
	if proj.FormationToDissolutionLatencyNs == nil {
		t.Fatal("FormationToDissolutionLatencyNs: got nil")
	}
	wantDiss := int64(800000) - formationAt
	if *proj.FormationToDissolutionLatencyNs != wantDiss {
		t.Errorf("FormationToDissolutionLatencyNs: got %d, want %d",
			*proj.FormationToDissolutionLatencyNs, wantDiss)
	}
}
