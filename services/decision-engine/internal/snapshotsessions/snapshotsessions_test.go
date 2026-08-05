package snapshotsessions_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/FabioCaffarello/ghost-trace/libs/eventstream"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/libs/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/decision-engine/internal/snapshotsessions"
)

type fakeStore struct {
	snap *eventsv1.SessionSnapshot
	err  error
	// tenant records what the adapter asked for, so the test can check
	// the key is composed from configuration and not from the token
	// alone.
	tenant string
}

func (f *fakeStore) Get(_ context.Context, tenant, _ string) (*eventsv1.SessionSnapshot, error) {
	f.tenant = tenant
	return f.snap, f.err
}

func TestNoSnapshotIsAColdStart(t *testing.T) {
	store := &fakeStore{err: fmt.Errorf("%w: t.st_x", eventstream.ErrNoSnapshot)}

	sess, found, err := snapshotsessions.New(store).Lookup(context.Background(), "t_demo", "st_x")
	if err != nil {
		t.Fatalf("a session with no snapshot returned an error: %v", err)
	}
	if found {
		t.Error("found = true for a session that has no snapshot")
	}
	if sess.ID != "" {
		t.Errorf("session id = %q, want empty", sess.ID)
	}
}

func TestAStoreThatDidNotAnswerIsNotAColdStart(t *testing.T) {
	// The distinction this adapter exists to make. Reporting a broken
	// store as a missing session turns a broker outage into a stream of
	// brand-new sessions, every one of them judged innocent for lack of
	// evidence — a detector failing open, silently, at exactly the
	// moment its evidence supply breaks.
	boom := errors.New("connection refused")
	store := &fakeStore{err: boom}

	if _, found, err := snapshotsessions.New(store).
		Lookup(context.Background(), "t_demo", "st_x"); !errors.Is(err, boom) || found {
		t.Errorf("Lookup = (found %v, err %v), want (false, the store's error)", found, err)
	}
}

func TestSnapshotMapsToDecisionState(t *testing.T) {
	store := &fakeStore{snap: &eventsv1.SessionSnapshot{
		TenantId:    "t_demo",
		SessionId:   "s_1",
		LastEventMs: 4200,
		Features:    &eventsv1.FeatureState{PointerPoints: 30, KeyEvents: 12},
	}}

	sess, found, err := snapshotsessions.New(store).Lookup(context.Background(), "t_demo", "st_x")
	if err != nil || !found {
		t.Fatalf("Lookup = (found %v, err %v), want (true, nil)", found, err)
	}
	if sess.ID != "s_1" || sess.TenantID != "t_demo" {
		t.Errorf("identity = %q/%q, want s_1/t_demo", sess.ID, sess.TenantID)
	}
	if sess.LastEventMs != 4200 {
		t.Errorf("last event ms = %d, want 4200", sess.LastEventMs)
	}
	if sess.State.Pointer.Points != 30 || sess.State.Keystroke.Keys != 12 {
		t.Errorf("feature state did not survive the mapping: %+v", sess.State)
	}
	if store.tenant != "t_demo" {
		t.Errorf("looked up under tenant %q; the key is (tenant, token) and the "+
			"tenant is configuration, not something the request carries", store.tenant)
	}
}
