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

// formAutomationGroup populates the substrate with five
// uniform-cadence DeclaredSessions for the supplied actor,
// runs FormAutomationGroupAll, and returns the resulting
// AutomationGroupFormation hash.
func formAutomationGroup(t *testing.T, sub *substrate.Substrate, actor string, declaredAtBase int64) [32]byte {
	t.Helper()
	ctx := context.Background()
	in := ingest.New(sub, func() time.Time { return time.Unix(0, declaredAtBase) })
	for i := 0; i < 5; i++ {
		msg := &eventsv1.DeclaredSession{
			DeclaredAt:        declaredAtBase + int64(i)*1000,
			ActorRef:          actor,
			SessionDescriptor: []byte("ag-" + actor),
		}
		if _, err := in.Append(ctx, msg, msg.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("ingest %d: %v", i, err)
		}
	}
	if _, err := hypothesis.FormAutomationGroupAll(ctx, sub,
		hypothesis.UniformCadenceV1{MinObservationCount: 5, MaxCoVThreshold: 0.15},
		func() time.Time { return time.Unix(0, declaredAtBase+10000) }); err != nil {
		t.Fatalf("FormAutomationGroupAll: %v", err)
	}
	var formationHash [32]byte
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != automationGroupFormationMessageType {
			return nil
		}
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return err
		}
		ev := &eventsv1.AutomationGroupFormation{}
		if err := proto.Unmarshal(payload, ev); err != nil {
			return err
		}
		for _, ar := range ev.ActorRefs {
			if ar == actor {
				formationHash = row.EventHash
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	if formationHash == ([32]byte{}) {
		t.Fatalf("no formation found for actor %q", actor)
	}
	return formationHash
}

func TestProjectAutomationGroupForming(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	formation := formAutomationGroup(t, sub, "bot-form", 1000)
	proj, err := ProjectAutomationGroup(ctx, sub, formation)
	if err != nil {
		t.Fatalf("ProjectAutomationGroup: %v", err)
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

func TestProjectAutomationGroupPromoted(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	formation := formAutomationGroup(t, sub, "bot-prom", 1000)
	if _, err := hypothesis.PromoteAutomationGroup(ctx, sub, hypothesis.AutomationGroupPromoteOptions{
		FormationEventHash: formation,
		PromotedAt:         50000,
		CadenceSeconds:     60,
		Reason:             "operational pilot",
	}, nil); err != nil {
		t.Fatalf("PromoteAutomationGroup: %v", err)
	}

	proj, err := ProjectAutomationGroup(ctx, sub, formation)
	if err != nil {
		t.Fatalf("ProjectAutomationGroup: %v", err)
	}
	if proj.State != StatePromoted {
		t.Errorf("state: got %q, want %q", proj.State, StatePromoted)
	}
	if proj.LatestPromotion == nil || proj.LatestPromotion.Reason != "operational pilot" {
		t.Errorf("LatestPromotion: %+v", proj.LatestPromotion)
	}
}

func TestProjectAutomationGroupDissolved(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	formation := formAutomationGroup(t, sub, "bot-diss", 1000)
	if _, err := hypothesis.DissolveAutomationGroup(ctx, sub, hypothesis.AutomationGroupDissolveOptions{
		FormationEventHash: formation,
		DissolvedAt:        50000,
		Reason:             "signature misattributed",
	}, nil); err != nil {
		t.Fatalf("DissolveAutomationGroup: %v", err)
	}

	proj, err := ProjectAutomationGroup(ctx, sub, formation)
	if err != nil {
		t.Fatalf("ProjectAutomationGroup: %v", err)
	}
	if proj.State != StateDissolved {
		t.Errorf("state: got %q, want %q", proj.State, StateDissolved)
	}
}

func TestProjectAutomationGroupUnknownFormation(t *testing.T) {
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
	_, err = ProjectAutomationGroup(ctx, sub, bogus)
	if !errors.Is(err, ErrFormationNotFound) {
		t.Errorf("expected ErrFormationNotFound; got %v", err)
	}
}

func TestProjectAutomationGroupCrossSubtypeRejected(t *testing.T) {
	// A BehavioralClusterFormation hash passed to
	// ProjectAutomationGroup MUST return ErrTargetNotFormation
	// (the AG projection's wrong-type check is subtype-specific).
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

	_, err = ProjectAutomationGroup(ctx, sub, bcFormation)
	if !errors.Is(err, ErrTargetNotFormation) {
		t.Errorf("expected ErrTargetNotFormation for BC formation; got %v", err)
	}
}

func TestListAutomationGroupsEmptyAndPopulated(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	got, err := ListAutomationGroups(ctx, sub, AutomationGroupListOptions{})
	if err != nil {
		t.Fatalf("ListAutomationGroups: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("empty substrate: got %d", len(got))
	}

	_ = formAutomationGroup(t, sub, "bot-list-a", 1000)
	_ = formAutomationGroup(t, sub, "bot-list-b", 100000)
	_ = formAutomationGroup(t, sub, "bot-list-c", 200000)

	got, err = ListAutomationGroups(ctx, sub, AutomationGroupListOptions{})
	if err != nil {
		t.Fatalf("ListAutomationGroups: %v", err)
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

func TestCountAutomationGroupsByStateEquivalentToListFilter(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	_ = formAutomationGroup(t, sub, "bot-c-a", 1000)
	bot2 := formAutomationGroup(t, sub, "bot-c-b", 100000)
	bot3 := formAutomationGroup(t, sub, "bot-c-c", 200000)

	if _, err := hypothesis.PromoteAutomationGroup(ctx, sub, hypothesis.AutomationGroupPromoteOptions{
		FormationEventHash: bot2,
		PromotedAt:         150000,
		CadenceSeconds:     60,
	}, nil); err != nil {
		t.Fatalf("Promote: %v", err)
	}
	if _, err := hypothesis.DissolveAutomationGroup(ctx, sub, hypothesis.AutomationGroupDissolveOptions{
		FormationEventHash: bot3,
		DissolvedAt:        250000,
	}, nil); err != nil {
		t.Fatalf("Dissolve: %v", err)
	}

	counts, err := CountAutomationGroupsByState(ctx, sub, AutomationGroupListOptions{})
	if err != nil {
		t.Fatalf("CountAutomationGroupsByState: %v", err)
	}
	if counts.Total != 3 {
		t.Errorf("Total: got %d, want 3", counts.Total)
	}
	// Equivalence invariant per state.
	for state := range counts.ByState {
		list, err := ListAutomationGroups(ctx, sub, AutomationGroupListOptions{StateFilter: state})
		if err != nil {
			t.Fatalf("ListAutomationGroups(%q): %v", state, err)
		}
		if len(list) != counts.ByState[state] {
			t.Errorf("equivalence violated for state=%q: counts=%d, list=%d",
				state, counts.ByState[state], len(list))
		}
	}
}

func TestProjectAllAutomationGroupsEquivalentToProjectAutomationGroup(t *testing.T) {
	// Same equivalence invariant as §0052's BehavioralCluster
	// equivalence — per-formation projection from ProjectAll equals
	// ProjectAutomationGroup(sameFormation).
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	alpha := formAutomationGroup(t, sub, "bot-eq-a", 1000)
	beta := formAutomationGroup(t, sub, "bot-eq-b", 100000)

	if _, err := hypothesis.PromoteAutomationGroup(ctx, sub, hypothesis.AutomationGroupPromoteOptions{
		FormationEventHash: alpha,
		PromotedAt:         50000,
		CadenceSeconds:     60,
	}, nil); err != nil {
		t.Fatalf("Promote: %v", err)
	}
	if _, err := hypothesis.DissolveAutomationGroup(ctx, sub, hypothesis.AutomationGroupDissolveOptions{
		FormationEventHash: beta,
		DissolvedAt:        150000,
	}, nil); err != nil {
		t.Fatalf("Dissolve: %v", err)
	}

	all, err := ProjectAllAutomationGroups(ctx, sub)
	if err != nil {
		t.Fatalf("ProjectAllAutomationGroups: %v", err)
	}
	for _, formHash := range [][32]byte{alpha, beta} {
		single, err := ProjectAutomationGroup(ctx, sub, formHash)
		if err != nil {
			t.Fatalf("ProjectAutomationGroup(%x): %v", formHash, err)
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

func TestProjectAutomationGroupLatencies(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	formation := formAutomationGroup(t, sub, "bot-lat", 1000)
	// formation_at = max(declared_at) = 1000+4*1000 = 5000.

	if _, err := hypothesis.PromoteAutomationGroup(ctx, sub, hypothesis.AutomationGroupPromoteOptions{
		FormationEventHash: formation,
		PromotedAt:         50000,
		CadenceSeconds:     60,
	}, nil); err != nil {
		t.Fatalf("Promote: %v", err)
	}
	if _, err := hypothesis.DissolveAutomationGroup(ctx, sub, hypothesis.AutomationGroupDissolveOptions{
		FormationEventHash: formation,
		DissolvedAt:        80000,
	}, nil); err != nil {
		t.Fatalf("Dissolve: %v", err)
	}

	proj, err := ProjectAutomationGroup(ctx, sub, formation)
	if err != nil {
		t.Fatalf("ProjectAutomationGroup: %v", err)
	}
	if proj.FormationToFirstPromotionLatencyNs == nil {
		t.Fatal("FormationToFirstPromotionLatencyNs: got nil")
	}
	// formation_at=5000, first promotion=50000 → latency=45000.
	want := int64(50000 - 5000)
	if *proj.FormationToFirstPromotionLatencyNs != want {
		t.Errorf("FormationToFirstPromotionLatencyNs: got %d, want %d",
			*proj.FormationToFirstPromotionLatencyNs, want)
	}
	if proj.FormationToDissolutionLatencyNs == nil {
		t.Fatal("FormationToDissolutionLatencyNs: got nil")
	}
	want = int64(80000 - 5000)
	if *proj.FormationToDissolutionLatencyNs != want {
		t.Errorf("FormationToDissolutionLatencyNs: got %d, want %d",
			*proj.FormationToDissolutionLatencyNs, want)
	}
}
