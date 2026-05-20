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

// populateSubstrate constructs a substrate + ingests the supplied
// (actor_ref, session_descriptor) pairs as DeclaredSession Cat I
// observations via the full ingest.Ingester path. Returns the
// substrate + the slice of source content-hashes in ingestion order.
func populateSubstrate(t *testing.T, items []struct {
	ActorRef   string
	Descriptor []byte
}) (*substrate.Substrate, [][32]byte) {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	in := ingest.New(sub, func() time.Time { return time.Unix(0, 1716120000000000777) })
	var hashes [][32]byte
	for i, it := range items {
		msg := &eventsv1.DeclaredSession{
			DeclaredAt:        int64(1716120000000000000 + int64(i)*int64(time.Second)),
			ActorRef:          it.ActorRef,
			SessionDescriptor: it.Descriptor,
		}
		rep, err := in.Append(ctx, msg, msg.DeclaredAt, ingest.Envelope{Channel: "test"})
		if err != nil {
			t.Fatalf("Append %d: %v", i, err)
		}
		var h [32]byte
		raw, err := hexDecode(rep.EventHashHex)
		if err != nil {
			t.Fatalf("decode hash %s: %v", rep.EventHashHex, err)
		}
		copy(h[:], raw)
		hashes = append(hashes, h)
	}
	return sub, hashes
}

func hexDecode(s string) ([]byte, error) {
	out := make([]byte, len(s)/2)
	for i := 0; i < len(out); i++ {
		var hi, lo byte
		c := s[2*i]
		switch {
		case '0' <= c && c <= '9':
			hi = c - '0'
		case 'a' <= c && c <= 'f':
			hi = c - 'a' + 10
		}
		c = s[2*i+1]
		switch {
		case '0' <= c && c <= '9':
			lo = c - '0'
		case 'a' <= c && c <= 'f':
			lo = c - 'a' + 10
		}
		out[i] = (hi << 4) | lo
	}
	return out, nil
}

func walkFormations(t *testing.T, sub *substrate.Substrate) []*eventsv1.BehavioralClusterFormation {
	t.Helper()
	ctx := context.Background()
	var out []*eventsv1.BehavioralClusterFormation
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != "ghosttrace.events.v1.BehavioralClusterFormation" {
			return nil
		}
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return err
		}
		ev := &eventsv1.BehavioralClusterFormation{}
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

func TestFormAllHappyPath(t *testing.T) {
	// Two actors share descriptor "alpha"; one actor has descriptor
	// "beta" alone. Only the "alpha" group meets min-cluster-size=2.
	sub, sourceHashes := populateSubstrate(t, []struct {
		ActorRef   string
		Descriptor []byte
	}{
		{"actor-a", []byte("alpha")},
		{"actor-b", []byte("alpha")},
		{"actor-c", []byte("beta")},
	})

	ctx := context.Background()
	rep, err := FormAll(ctx, sub, SessionDescriptorSharedV1{MinClusterSize: 2}, func() time.Time { return time.Unix(0, 1) })
	if err != nil {
		t.Fatalf("FormAll: %v", err)
	}
	if rep.Examined != 3 {
		t.Errorf("Examined: got %d, want 3", rep.Examined)
	}
	if rep.NewlyFormed != 1 {
		t.Errorf("NewlyFormed: got %d, want 1 (only the alpha-group qualifies)", rep.NewlyFormed)
	}

	formations := walkFormations(t, sub)
	if len(formations) != 1 {
		t.Fatalf("substrate carries %d formations, want 1", len(formations))
	}
	got := formations[0]
	if got.PatternSignature != SessionDescriptorSharedV1Signature {
		t.Errorf("pattern_signature: got %q, want %q", got.PatternSignature, SessionDescriptorSharedV1Signature)
	}
	if got.PatternParameters != "min_cluster_size=2" {
		t.Errorf("pattern_parameters: got %q, want min_cluster_size=2", got.PatternParameters)
	}
	if len(got.ActorRefs) != 2 {
		t.Fatalf("actor_refs: got %d, want 2", len(got.ActorRefs))
	}
	if got.ActorRefs[0] != "actor-a" || got.ActorRefs[1] != "actor-b" {
		t.Errorf("actor_refs not sorted ascending: %v", got.ActorRefs)
	}
	// source_event_hashes must reference the two alpha-group DeclaredSessions.
	wantHashes := [][32]byte{sourceHashes[0], sourceHashes[1]}
	if len(got.SourceEventHashes) != 2 {
		t.Fatalf("source_event_hashes: got %d, want 2", len(got.SourceEventHashes))
	}
	for _, want := range wantHashes {
		found := false
		for _, raw := range got.SourceEventHashes {
			if bytes.Equal(raw, want[:]) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing source_event_hash %x in formation provenance", want)
		}
	}
}

func TestFormAllRespectsMinClusterSize(t *testing.T) {
	// Only one actor shares descriptor "alpha". min-cluster-size=2
	// → zero formations.
	sub, _ := populateSubstrate(t, []struct {
		ActorRef   string
		Descriptor []byte
	}{
		{"actor-a", []byte("alpha")},
		{"actor-b", []byte("beta")},
		{"actor-c", []byte("gamma")},
	})

	rep, err := FormAll(context.Background(), sub, SessionDescriptorSharedV1{MinClusterSize: 2}, func() time.Time { return time.Unix(0, 1) })
	if err != nil {
		t.Fatalf("FormAll: %v", err)
	}
	if rep.NewlyFormed != 0 {
		t.Errorf("NewlyFormed: got %d, want 0 (no group meets size 2)", rep.NewlyFormed)
	}
	if got := len(walkFormations(t, sub)); got != 0 {
		t.Errorf("substrate carries %d formations, want 0", got)
	}
}

func TestFormAllIdempotent(t *testing.T) {
	sub, _ := populateSubstrate(t, []struct {
		ActorRef   string
		Descriptor []byte
	}{
		{"actor-a", []byte("alpha")},
		{"actor-b", []byte("alpha")},
	})
	def := SessionDescriptorSharedV1{MinClusterSize: 2}

	ctx := context.Background()
	rep1, err := FormAll(ctx, sub, def, func() time.Time { return time.Unix(0, 1) })
	if err != nil {
		t.Fatalf("first FormAll: %v", err)
	}
	if rep1.NewlyFormed != 1 {
		t.Fatalf("first run NewlyFormed: got %d, want 1", rep1.NewlyFormed)
	}

	rep2, err := FormAll(ctx, sub, def, func() time.Time { return time.Unix(0, 999) })
	if err != nil {
		t.Fatalf("second FormAll: %v", err)
	}
	if rep2.NewlyFormed != 0 {
		t.Errorf("second run NewlyFormed: got %d, want 0 (idempotent)", rep2.NewlyFormed)
	}
	if rep2.AlreadyFormed != 1 {
		t.Errorf("second run AlreadyFormed: got %d, want 1", rep2.AlreadyFormed)
	}
	if got := len(walkFormations(t, sub)); got != 1 {
		t.Errorf("after re-run, substrate carries %d formations, want 1", got)
	}
}

func TestFormAllVersioningProducesNewRecord(t *testing.T) {
	// Three actors share descriptor "alpha". min-cluster-size=2 → forms.
	// min-cluster-size=3 → still forms (3 ≥ 3). Different parameters
	// produce a DIFFERENT content-hash → new substrate record.
	sub, _ := populateSubstrate(t, []struct {
		ActorRef   string
		Descriptor []byte
	}{
		{"actor-a", []byte("alpha")},
		{"actor-b", []byte("alpha")},
		{"actor-c", []byte("alpha")},
	})

	ctx := context.Background()
	if _, err := FormAll(ctx, sub, SessionDescriptorSharedV1{MinClusterSize: 2}, func() time.Time { return time.Unix(0, 1) }); err != nil {
		t.Fatalf("FormAll defA: %v", err)
	}
	repB, err := FormAll(ctx, sub, SessionDescriptorSharedV1{MinClusterSize: 3}, func() time.Time { return time.Unix(0, 2) })
	if err != nil {
		t.Fatalf("FormAll defB: %v", err)
	}
	if repB.NewlyFormed != 1 {
		t.Errorf("changing parameters did not produce new record: NewlyFormed=%d, want 1", repB.NewlyFormed)
	}
	if got := len(walkFormations(t, sub)); got != 2 {
		t.Errorf("substrate should hold 2 formations after two parameterizations; got %d", got)
	}
}

func TestFormAllExcludesEmptyActorRef(t *testing.T) {
	// An anonymous DeclaredSession (empty actor_ref) is excluded
	// from cluster membership.
	sub, _ := populateSubstrate(t, []struct {
		ActorRef   string
		Descriptor []byte
	}{
		{"", []byte("alpha")},
		{"actor-b", []byte("alpha")},
	})

	rep, err := FormAll(context.Background(), sub, SessionDescriptorSharedV1{MinClusterSize: 2}, func() time.Time { return time.Unix(0, 1) })
	if err != nil {
		t.Fatalf("FormAll: %v", err)
	}
	if rep.NewlyFormed != 0 {
		t.Errorf("NewlyFormed: got %d, want 0 (anonymous member excluded leaves cluster size 1)", rep.NewlyFormed)
	}
}

func TestFormAllEmptySubstrate(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	rep, err := FormAll(ctx, sub, SessionDescriptorSharedV1{MinClusterSize: 2}, nil)
	if err != nil {
		t.Fatalf("FormAll on empty substrate: %v", err)
	}
	if rep.Examined != 0 || rep.NewlyFormed != 0 || rep.AlreadyFormed != 0 {
		t.Errorf("empty substrate Report: got %+v, want zero", rep)
	}
}

func TestFormAllMultipleDescriptorGroups(t *testing.T) {
	// Two distinct shared-descriptor groups each meet min-cluster-size=2.
	// Both should form their own BehavioralCluster.
	sub, _ := populateSubstrate(t, []struct {
		ActorRef   string
		Descriptor []byte
	}{
		{"actor-a", []byte("alpha")},
		{"actor-b", []byte("alpha")},
		{"actor-c", []byte("beta")},
		{"actor-d", []byte("beta")},
	})

	rep, err := FormAll(context.Background(), sub, SessionDescriptorSharedV1{MinClusterSize: 2}, func() time.Time { return time.Unix(0, 1) })
	if err != nil {
		t.Fatalf("FormAll: %v", err)
	}
	if rep.NewlyFormed != 2 {
		t.Errorf("NewlyFormed: got %d, want 2 (one per descriptor group)", rep.NewlyFormed)
	}
	formations := walkFormations(t, sub)
	if len(formations) != 2 {
		t.Fatalf("substrate carries %d formations, want 2", len(formations))
	}
	// Each formation has exactly 2 actor_refs.
	for _, f := range formations {
		if len(f.ActorRefs) != 2 {
			t.Errorf("formation has %d actors, want 2", len(f.ActorRefs))
		}
	}
}

func TestFormAllConfidencePlaceholder(t *testing.T) {
	// 3-actor cluster: confidence = 1.0 - 1.0/3 ≈ 0.666...
	sub, _ := populateSubstrate(t, []struct {
		ActorRef   string
		Descriptor []byte
	}{
		{"actor-a", []byte("alpha")},
		{"actor-b", []byte("alpha")},
		{"actor-c", []byte("alpha")},
	})

	if _, err := FormAll(context.Background(), sub, SessionDescriptorSharedV1{MinClusterSize: 2}, func() time.Time { return time.Unix(0, 1) }); err != nil {
		t.Fatalf("FormAll: %v", err)
	}
	formations := walkFormations(t, sub)
	if len(formations) != 1 {
		t.Fatalf("formations: %d, want 1", len(formations))
	}
	want := float32(1.0 - 1.0/3.0)
	if got := formations[0].Confidence; got != want {
		t.Errorf("confidence: got %v, want %v (placeholder per §2.6 pending)", got, want)
	}
}
