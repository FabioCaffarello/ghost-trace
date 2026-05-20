package derivation

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// newPopulatedSubstrate constructs a substrate + ingests `count`
// DeclaredSession messages via the full ingest.Ingester path. Returns
// the substrate plus the slice of source content-hashes (primary
// observation hashes only; not the IngestionEvent enrichment hashes).
func newPopulatedSubstrate(t *testing.T, count int) (*substrate.Substrate, [][32]byte) {
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
	for i := 0; i < count; i++ {
		msg := &eventsv1.DeclaredSession{
			DeclaredAt:        int64(1716120000000000000 + int64(i)*int64(time.Second)),
			ActorRef:          "actor-" + string(rune('a'+i)),
			SessionDescriptor: []byte("session-" + string(rune('A'+i))),
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

func TestPaddedV1Deterministic(t *testing.T) {
	source := &eventsv1.DeclaredSession{
		DeclaredAt:        1716120000000000000,
		ActorRef:          "actor-determinism",
		SessionDescriptor: []byte("s"),
	}
	def := PaddedV1{PadSeconds: 300}

	out1 := def.Derive(source, [32]byte{})
	out1.DefinitionVersion = def.Version()
	out1.DefinitionParameters = def.Parameters()

	out2 := def.Derive(source, [32]byte{})
	out2.DefinitionVersion = def.Version()
	out2.DefinitionParameters = def.Parameters()

	_, h1, err := canonical.MarshalAndHash(out1)
	if err != nil {
		t.Fatalf("MarshalAndHash: %v", err)
	}
	_, h2, err := canonical.MarshalAndHash(out2)
	if err != nil {
		t.Fatalf("MarshalAndHash: %v", err)
	}
	if h1 != h2 {
		t.Errorf("Derive is non-deterministic: hash1 %x != hash2 %x", h1, h2)
	}
}

func TestPaddedV1BoundaryDivergence(t *testing.T) {
	source := &eventsv1.DeclaredSession{DeclaredAt: 1716120000000000000, ActorRef: "actor-divergence"}
	def := PaddedV1{PadSeconds: 300}

	out := def.Derive(source, [32]byte{})
	if out.OperationalStartAt != source.DeclaredAt {
		t.Errorf("operational_start_at: got %d, want %d (=declared_at)", out.OperationalStartAt, source.DeclaredAt)
	}
	want := source.DeclaredAt + 300*int64(time.Second)
	if out.OperationalEndAt != want {
		t.Errorf("operational_end_at: got %d, want %d (=declared_at + pad)", out.OperationalEndAt, want)
	}
	if out.OperationalEndAt == source.DeclaredAt {
		t.Error("operational boundary did not diverge from source declared_at — Cat II divergence missing")
	}
}

func TestPaddedV1ActorRefInherits(t *testing.T) {
	source := &eventsv1.DeclaredSession{DeclaredAt: 1, ActorRef: "actor-inherits"}
	def := PaddedV1{PadSeconds: 1}
	out := def.Derive(source, [32]byte{})
	if out.ActorRef != source.ActorRef {
		t.Errorf("actor_ref: got %q, want %q (inheritance per entity-model.md line 36)", out.ActorRef, source.ActorRef)
	}
}

func TestDeriveAllHappyPath(t *testing.T) {
	ctx := context.Background()
	sub, sourceHashes := newPopulatedSubstrate(t, 3)
	def := PaddedV1{PadSeconds: 300}

	rep, err := DeriveAll(ctx, sub, def, func() time.Time { return time.Unix(0, 1) })
	if err != nil {
		t.Fatalf("DeriveAll: %v", err)
	}
	if rep.Examined != 3 {
		t.Errorf("Examined: got %d, want 3", rep.Examined)
	}
	if rep.NewlyDerived != 3 {
		t.Errorf("NewlyDerived: got %d, want 3", rep.NewlyDerived)
	}
	if rep.AlreadyDerived != 0 {
		t.Errorf("AlreadyDerived: got %d, want 0", rep.AlreadyDerived)
	}

	// Substrate now contains: 3 DeclaredSession + 3 IngestionEvent + 3 OperationalSession = 9.
	n, err := sub.Count(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 9 {
		t.Errorf("substrate Count: got %d, want 9 (3 primary + 3 enrichment + 3 derived)", n)
	}

	typeCounts := map[string]int{}
	walkedSourceHashes := map[[32]byte]bool{}
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		typeCounts[row.MessageType]++
		if row.MessageType == "ghosttrace.events.v1.OperationalSession" {
			payload, err := sub.ReadBlob(ctx, row.EventHash)
			if err != nil {
				return err
			}
			var os eventsv1.OperationalSession
			if err := proto.Unmarshal(payload, &os); err != nil {
				return err
			}
			if got := os.GetDefinitionVersion(); got != def.Version() {
				t.Errorf("derived definition_version: got %q, want %q", got, def.Version())
			}
			if got := os.GetDefinitionParameters(); got != def.Parameters() {
				t.Errorf("derived definition_parameters: got %q, want %q", got, def.Parameters())
			}
			var srcHash [32]byte
			copy(srcHash[:], os.GetSourceEventHash())
			walkedSourceHashes[srcHash] = true
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	if got := typeCounts["ghosttrace.events.v1.OperationalSession"]; got != 3 {
		t.Errorf("OperationalSession rows: got %d, want 3", got)
	}
	for _, srcHash := range sourceHashes {
		if !walkedSourceHashes[srcHash] {
			t.Errorf("no OperationalSession references source hash %x", srcHash)
		}
	}
}

func TestDeriveAllIdempotent(t *testing.T) {
	ctx := context.Background()
	sub, _ := newPopulatedSubstrate(t, 2)
	def := PaddedV1{PadSeconds: 300}

	rep1, err := DeriveAll(ctx, sub, def, func() time.Time { return time.Unix(0, 1) })
	if err != nil {
		t.Fatalf("first DeriveAll: %v", err)
	}
	if rep1.NewlyDerived != 2 {
		t.Fatalf("first run NewlyDerived: got %d, want 2", rep1.NewlyDerived)
	}

	// Re-run with the IDENTICAL definition. Content-hash collision
	// means substrate.Append is a no-op for each row; the report
	// distinguishes via pre-flight LookupRow.
	rep2, err := DeriveAll(ctx, sub, def, func() time.Time { return time.Unix(0, 999) })
	if err != nil {
		t.Fatalf("second DeriveAll: %v", err)
	}
	if rep2.NewlyDerived != 0 {
		t.Errorf("second run NewlyDerived: got %d, want 0 (idempotent)", rep2.NewlyDerived)
	}
	if rep2.AlreadyDerived != 2 {
		t.Errorf("second run AlreadyDerived: got %d, want 2", rep2.AlreadyDerived)
	}

	// Substrate row count unchanged after the re-run (2 DeclaredSession
	// + 2 IngestionEvent + 2 OperationalSession = 6).
	n, err := sub.Count(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 6 {
		t.Errorf("substrate Count after re-run: got %d, want 6", n)
	}
}

func TestDeriveAllVersioningProducesNewRecord(t *testing.T) {
	// Re-derivation under a NEW (definition_version, parameters)
	// tuple produces NEW substrate records per entity-model.md line 45.
	// The prior derivation is preserved.
	ctx := context.Background()
	sub, _ := newPopulatedSubstrate(t, 1)

	defA := PaddedV1{PadSeconds: 300}
	defB := PaddedV1{PadSeconds: 600}

	if _, err := DeriveAll(ctx, sub, defA, func() time.Time { return time.Unix(0, 1) }); err != nil {
		t.Fatalf("DeriveAll defA: %v", err)
	}
	repB, err := DeriveAll(ctx, sub, defB, func() time.Time { return time.Unix(0, 2) })
	if err != nil {
		t.Fatalf("DeriveAll defB: %v", err)
	}
	if repB.NewlyDerived != 1 {
		t.Errorf("changing parameters did not produce new record: NewlyDerived=%d, want 1", repB.NewlyDerived)
	}
	if repB.AlreadyDerived != 0 {
		t.Errorf("changing parameters wrongly counted as AlreadyDerived: %d", repB.AlreadyDerived)
	}

	osCount := 0
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType == "ghosttrace.events.v1.OperationalSession" {
			osCount++
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	if osCount != 2 {
		t.Errorf("after two derivations under different params, OperationalSession rows: got %d, want 2 (both preserved)", osCount)
	}
}

func TestDeriveAllEmptySubstrate(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	rep, err := DeriveAll(ctx, sub, PaddedV1{PadSeconds: 300}, nil)
	if err != nil {
		t.Fatalf("DeriveAll on empty substrate: %v", err)
	}
	if rep.Examined != 0 || rep.NewlyDerived != 0 || rep.AlreadyDerived != 0 {
		t.Errorf("empty substrate Report: got %+v, want zero", rep)
	}
}

func TestDeriveAllSkipsNonDeclaredSessionRows(t *testing.T) {
	// Substrate populated with 2 DeclaredSession (each producing 1 paired
	// IngestionEvent enrichment row = 4 rows total). Then DeriveAll runs;
	// Examined should be 2 (only DeclaredSession rows), not 4.
	ctx := context.Background()
	sub, _ := newPopulatedSubstrate(t, 2)

	rep, err := DeriveAll(ctx, sub, PaddedV1{PadSeconds: 1}, func() time.Time { return time.Unix(0, 1) })
	if err != nil {
		t.Fatalf("DeriveAll: %v", err)
	}
	if rep.Examined != 2 {
		t.Errorf("Examined: got %d, want 2 (IngestionEvent rows should be skipped)", rep.Examined)
	}
}

