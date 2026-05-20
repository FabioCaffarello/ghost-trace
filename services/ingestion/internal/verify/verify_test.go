package verify

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/derivation"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/hypothesis"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// newPopulatedSubstrate constructs a substrate + ingests `count` test
// DeclaredSession messages via the full ingest.Ingester path. Returns
// the substrate plus the slice of primary content-hashes committed.
func newPopulatedSubstrate(t *testing.T, count int) (*substrate.Substrate, []string) {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	in := ingest.New(sub, time.Now)
	var hashes []string
	for i := 0; i < count; i++ {
		msg := &eventsv1.DeclaredSession{
			DeclaredAt:        int64(1000 + i),
			ActorRef:          "actor-" + string(rune('a'+i)),
			SessionDescriptor: []byte("session-" + string(rune('A'+i))),
		}
		rep, err := in.Append(ctx, msg, msg.DeclaredAt, ingest.Envelope{Channel: "test"})
		if err != nil {
			t.Fatalf("Append %d: %v", i, err)
		}
		hashes = append(hashes, rep.EventHashHex, rep.IngestionEventHashHex)
	}
	return sub, hashes
}

func TestVerifyHappyPath(t *testing.T) {
	ctx := context.Background()
	sub, hashes := newPopulatedSubstrate(t, 3)

	report, err := Verify(ctx, sub, Options{})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if report.Failed() {
		t.Errorf("happy-path Verify reported failure: %+v", report)
	}
	// 3 ingest calls × 2 records (primary + enrichment) = 6 walked.
	if report.VerifiedCount != int64(len(hashes)) {
		t.Errorf("VerifiedCount: got %d, want %d", report.VerifiedCount, len(hashes))
	}
	if report.HashMismatchCount != 0 {
		t.Errorf("HashMismatchCount: got %d, want 0", report.HashMismatchCount)
	}
	if report.MissingBlobCount != 0 {
		t.Errorf("MissingBlobCount: got %d, want 0", report.MissingBlobCount)
	}
}

func TestVerifyDetectsCorruption(t *testing.T) {
	ctx := context.Background()
	sub, _ := newPopulatedSubstrate(t, 3)

	// Locate the blob-store directory + corrupt one blob.
	// Walk to find any blob file.
	var blobPath string
	dir := sub.BlobDir()
	if dir == "" {
		t.Skip("substrate does not expose BlobDir() — instrumentation needed")
	}
	if err := filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || blobPath != "" {
			return nil
		}
		blobPath = p
		return nil
	}); err != nil {
		t.Fatalf("walk blobs: %v", err)
	}
	if blobPath == "" {
		t.Fatal("no blob file found to corrupt")
	}
	// Overwrite with garbage; hash recomputation must surface a mismatch.
	if err := os.WriteFile(blobPath, []byte("corrupted-on-purpose"), 0o644); err != nil {
		t.Fatalf("write corrupted blob: %v", err)
	}

	report, err := Verify(ctx, sub, Options{})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !report.Failed() {
		t.Fatal("Verify did not detect blob corruption")
	}
	if report.HashMismatchCount < 1 {
		t.Errorf("HashMismatchCount: got %d, want >= 1", report.HashMismatchCount)
	}
	if len(report.HashMismatchHashes) != int(report.HashMismatchCount) {
		t.Errorf("HashMismatchHashes len %d != HashMismatchCount %d",
			len(report.HashMismatchHashes), report.HashMismatchCount)
	}
}

func TestVerifyDetectsMissingBlob(t *testing.T) {
	ctx := context.Background()
	sub, _ := newPopulatedSubstrate(t, 3)

	dir := sub.BlobDir()
	if dir == "" {
		t.Skip("substrate does not expose BlobDir()")
	}
	// Delete one blob to simulate filesystem corruption / partial backup.
	var deletedHex string
	if err := filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || deletedHex != "" {
			return err
		}
		if err := os.Remove(p); err != nil {
			return err
		}
		// Recover the hex hash from the path: <dir>/<2>/<62>.
		rel, _ := filepath.Rel(dir, p)
		deletedHex = strings.ReplaceAll(filepath.ToSlash(rel), "/", "")
		return nil
	}); err != nil {
		t.Fatalf("walk blobs: %v", err)
	}
	if deletedHex == "" {
		t.Fatal("no blob file deleted")
	}

	report, err := Verify(ctx, sub, Options{})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !report.Failed() {
		t.Fatal("Verify did not detect missing blob")
	}
	if report.MissingBlobCount < 1 {
		t.Errorf("MissingBlobCount: got %d, want >= 1", report.MissingBlobCount)
	}

	// The deleted hash should appear in MissingBlobHashes.
	found := false
	for _, h := range report.MissingBlobHashes {
		if h == deletedHex {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("deleted hash %s not in MissingBlobHashes %v", deletedHex, report.MissingBlobHashes)
	}
}

func TestVerifyEmptySubstrate(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	report, err := Verify(ctx, sub, Options{})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if report.Failed() {
		t.Errorf("empty substrate should pass: %+v", report)
	}
	if report.VerifiedCount != 0 {
		t.Errorf("VerifiedCount: got %d, want 0", report.VerifiedCount)
	}
}

// Sanity: the helper produces deterministic test data.
func TestPopulatedSubstrateHashes(t *testing.T) {
	_, hashes := newPopulatedSubstrate(t, 2)
	if len(hashes) != 4 {
		t.Errorf("expected 4 hashes (2 primary + 2 enrichment), got %d", len(hashes))
	}
	for _, h := range hashes {
		if len(h) != 64 {
			t.Errorf("hash length: got %d, want 64 — %q", len(h), h)
		}
	}
}

func TestVerifyCheckOrphansDetectsOrphan(t *testing.T) {
	ctx := context.Background()
	sub, _ := newPopulatedSubstrate(t, 2)

	// Plant an orphan blob: write a file in the blob-store with a
	// hash that does NOT appear in the events table.
	orphanContent := []byte("this-blob-has-no-index-row")
	orphanHash := canonical.Hash(orphanContent)
	orphanHex := canonical.HashHex(orphanHash)
	orphanDir := filepath.Join(sub.BlobDir(), orphanHex[:2])
	if err := os.MkdirAll(orphanDir, 0o755); err != nil {
		t.Fatalf("mkdir orphan shard: %v", err)
	}
	orphanPath := filepath.Join(orphanDir, orphanHex[2:])
	if err := os.WriteFile(orphanPath, orphanContent, 0o644); err != nil {
		t.Fatalf("write orphan blob: %v", err)
	}

	report, err := Verify(ctx, sub, Options{CheckOrphans: true})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	// Orphans are NOT failures per §0033 + §0040.
	if report.Failed() {
		t.Errorf("orphan should not cause Failed(): %+v", report)
	}
	if report.OrphanBlobCount != 1 {
		t.Errorf("OrphanBlobCount: got %d, want 1", report.OrphanBlobCount)
	}
	if len(report.OrphanBlobPaths) != 1 {
		t.Fatalf("OrphanBlobPaths len: got %d, want 1", len(report.OrphanBlobPaths))
	}
	if report.OrphanBlobPaths[0] != orphanPath {
		t.Errorf("OrphanBlobPaths[0]: got %q, want %q", report.OrphanBlobPaths[0], orphanPath)
	}
}

func TestVerifyOrphanDetectionDisabledByDefault(t *testing.T) {
	ctx := context.Background()
	sub, _ := newPopulatedSubstrate(t, 2)

	// Plant an orphan blob.
	orphanContent := []byte("orphan-not-checked-when-off")
	orphanHash := canonical.Hash(orphanContent)
	orphanHex := canonical.HashHex(orphanHash)
	orphanDir := filepath.Join(sub.BlobDir(), orphanHex[:2])
	_ = os.MkdirAll(orphanDir, 0o755)
	_ = os.WriteFile(filepath.Join(orphanDir, orphanHex[2:]), orphanContent, 0o644)

	// Default Options{CheckOrphans: false} skips the blob-store walk.
	report, err := Verify(ctx, sub, Options{})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if report.OrphanBlobCount != 0 {
		t.Errorf("OrphanBlobCount: got %d, want 0 (orphan-check disabled)", report.OrphanBlobCount)
	}
	if report.Failed() {
		t.Errorf("happy-path with orphan present (and check disabled) should not fail: %+v", report)
	}
}

func TestVerifyCheckOrphansHappyPath(t *testing.T) {
	// No orphans planted; OrphanBlobCount must be 0.
	ctx := context.Background()
	sub, _ := newPopulatedSubstrate(t, 3)

	report, err := Verify(ctx, sub, Options{CheckOrphans: true})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if report.Failed() {
		t.Errorf("clean substrate should not fail: %+v", report)
	}
	if report.OrphanBlobCount != 0 {
		t.Errorf("OrphanBlobCount: got %d, want 0 (no orphans planted)", report.OrphanBlobCount)
	}
}

// TestVerifyHeterogeneousCatITypes proves verify walks a substrate
// containing multiple Cat I primary-observation types (per decision-log
// §0042) and reports success. The substrate's hash-chain consistency
// is type-agnostic; verify recomputes hashes per the canonical-
// serialization-contract for each row regardless of message_type.
func TestVerifyHeterogeneousCatITypes(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	in := ingest.New(sub, time.Now)
	decl := &eventsv1.DeclaredSession{DeclaredAt: 1000, ActorRef: "het-decl", SessionDescriptor: []byte("s")}
	if _, err := in.Append(ctx, decl, decl.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
		t.Fatalf("Append DeclaredSession: %v", err)
	}
	netEvt := &eventsv1.NetworkEvent{ObservedAt: 1001, ActorRef: "het-net", EndpointRef: "10.0.0.1:80", EventDescriptor: []byte("f")}
	if _, err := in.Append(ctx, netEvt, netEvt.ObservedAt, ingest.Envelope{Channel: "test"}); err != nil {
		t.Fatalf("Append NetworkEvent: %v", err)
	}

	report, err := Verify(ctx, sub, Options{})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if report.Failed() {
		t.Errorf("heterogeneous substrate Verify reported failure: %+v", report)
	}
	if report.VerifiedCount != 4 {
		t.Errorf("VerifiedCount: got %d, want 4 (2 primary + 2 enrichment)", report.VerifiedCount)
	}
}

// TestVerifyWithCatIIRecords proves verify passes over a substrate
// that contains Category II OperationalSession records alongside Cat I
// primary observations (per decision-log §0043). The substrate-
// integrity audit is type-agnostic: hash-chain consistency does not
// depend on which Category a record belongs to.
func TestVerifyWithCatIIRecords(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	in := ingest.New(sub, time.Now)
	decl := &eventsv1.DeclaredSession{DeclaredAt: 1000, ActorRef: "actor-cat-ii", SessionDescriptor: []byte("s")}
	if _, err := in.Append(ctx, decl, decl.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
		t.Fatalf("Append DeclaredSession: %v", err)
	}
	if _, err := derivation.DeriveAll(ctx, sub, derivation.PaddedV1{PadSeconds: 300}, time.Now); err != nil {
		t.Fatalf("DeriveAll: %v", err)
	}

	report, err := Verify(ctx, sub, Options{})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if report.Failed() {
		t.Errorf("Cat-I+Cat-II substrate Verify reported failure: %+v", report)
	}
	if report.VerifiedCount != 3 {
		t.Errorf("VerifiedCount: got %d, want 3 (1 primary + 1 enrichment + 1 derived)", report.VerifiedCount)
	}
}

// TestVerifyWithCatIIIRecords proves verify passes over a substrate
// that contains Category III lifecycle event records
// (BehavioralClusterFormation) alongside Cat I primary observations
// and Cat II operational constructs (per decision-log §0045). The
// substrate-integrity audit is type-agnostic across all three
// Categories per Charter §2.5 BC5 (lifecycle events are Cat I records).
func TestVerifyWithCatIIIRecords(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	in := ingest.New(sub, time.Now)
	for _, ar := range []string{"actor-cat-iii-a", "actor-cat-iii-b"} {
		decl := &eventsv1.DeclaredSession{
			DeclaredAt:        1000,
			ActorRef:          ar,
			SessionDescriptor: []byte("shared"),
		}
		if _, err := in.Append(ctx, decl, decl.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("Append DeclaredSession %s: %v", ar, err)
		}
	}
	if _, err := hypothesis.FormAll(ctx, sub, hypothesis.SessionDescriptorSharedV1{MinClusterSize: 2}, time.Now); err != nil {
		t.Fatalf("FormAll: %v", err)
	}

	report, err := Verify(ctx, sub, Options{})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if report.Failed() {
		t.Errorf("Cat-III substrate Verify reported failure: %+v", report)
	}
	// 2 DeclaredSession + 2 IngestionEvent + 1 BehavioralClusterFormation = 5.
	if report.VerifiedCount != 5 {
		t.Errorf("VerifiedCount: got %d, want 5", report.VerifiedCount)
	}
}

// TestVerifyWithCatIIIPromotion proves verify passes over a substrate
// that contains BOTH a BehavioralClusterFormation AND a subsequent
// BehavioralClusterPromotion lifecycle event (per decision-log §0046).
// The substrate-integrity audit is type-agnostic across all Cat III
// lifecycle operations per Charter §2.5 BC5.
func TestVerifyWithCatIIIPromotion(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	in := ingest.New(sub, time.Now)
	for _, ar := range []string{"actor-prom-a", "actor-prom-b"} {
		decl := &eventsv1.DeclaredSession{DeclaredAt: 1000, ActorRef: ar, SessionDescriptor: []byte("shared")}
		if _, err := in.Append(ctx, decl, decl.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("Append DeclaredSession %s: %v", ar, err)
		}
	}
	if _, err := hypothesis.FormAll(ctx, sub, hypothesis.SessionDescriptorSharedV1{MinClusterSize: 2}, time.Now); err != nil {
		t.Fatalf("FormAll: %v", err)
	}

	// Find the formation event hash to promote against.
	var formationHash [32]byte
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType == "ghosttrace.events.v1.BehavioralClusterFormation" {
			formationHash = row.EventHash
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}

	if _, err := hypothesis.Promote(ctx, sub, hypothesis.PromoteOptions{
		FormationEventHash: formationHash,
		PromotedAt:         1716120000000000000,
		CadenceSeconds:     3600,
		Reason:             "integration test",
	}, time.Now); err != nil {
		t.Fatalf("Promote: %v", err)
	}

	report, err := Verify(ctx, sub, Options{})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if report.Failed() {
		t.Errorf("formation+promotion substrate Verify reported failure: %+v", report)
	}
	// 2 DeclaredSession + 2 IngestionEvent + 1 Formation + 1 Promotion = 6.
	if report.VerifiedCount != 6 {
		t.Errorf("VerifiedCount: got %d, want 6", report.VerifiedCount)
	}
}

// TestVerifyWithCatIIIFullLoop proves verify passes over a substrate
// containing the full Cat III lifecycle chain — formation + promotion
// + demotion (per decision-log §0047 closing the promote/demote loop).
// The substrate-integrity audit is type-agnostic across all Cat III
// lifecycle operations per Charter §2.5 BC5.
func TestVerifyWithCatIIIFullLoop(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	in := ingest.New(sub, time.Now)
	for _, ar := range []string{"actor-demote-a", "actor-demote-b"} {
		decl := &eventsv1.DeclaredSession{DeclaredAt: 1000, ActorRef: ar, SessionDescriptor: []byte("shared")}
		if _, err := in.Append(ctx, decl, decl.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("Append DeclaredSession %s: %v", ar, err)
		}
	}
	if _, err := hypothesis.FormAll(ctx, sub, hypothesis.SessionDescriptorSharedV1{MinClusterSize: 2}, time.Now); err != nil {
		t.Fatalf("FormAll: %v", err)
	}

	var formationHash [32]byte
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType == "ghosttrace.events.v1.BehavioralClusterFormation" {
			formationHash = row.EventHash
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}

	if _, err := hypothesis.Promote(ctx, sub, hypothesis.PromoteOptions{
		FormationEventHash: formationHash,
		PromotedAt:         1716120000000000000,
		CadenceSeconds:     60,
		Reason:             "integration",
	}, time.Now); err != nil {
		t.Fatalf("Promote: %v", err)
	}

	var promotionHash [32]byte
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType == "ghosttrace.events.v1.BehavioralClusterPromotion" {
			promotionHash = row.EventHash
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}

	if _, err := hypothesis.Demote(ctx, sub, hypothesis.DemoteOptions{
		PromotionEventHash: promotionHash,
		DemotedAt:          1716120120000000000,
		Reason:             "integration close",
	}, time.Now); err != nil {
		t.Fatalf("Demote: %v", err)
	}

	report, err := Verify(ctx, sub, Options{})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if report.Failed() {
		t.Errorf("full-loop substrate Verify reported failure: %+v", report)
	}
	// 2 DeclaredSession + 2 IngestionEvent + 1 Formation + 1 Promotion + 1 Demotion = 7.
	if report.VerifiedCount != 7 {
		t.Errorf("VerifiedCount: got %d, want 7", report.VerifiedCount)
	}
}

// TestVerifyWithCatIIIFullLifecycle proves verify passes over a
// substrate containing the full Cat III lifecycle chain — formation
// + promotion + demotion + dissolution (per decision-log §0048
// landing the fourth lifecycle operation). The substrate-integrity
// audit is type-agnostic across all Cat III lifecycle operations per
// Charter §2.5 BC5.
func TestVerifyWithCatIIIFullLifecycle(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	in := ingest.New(sub, time.Now)
	for _, ar := range []string{"actor-dissolve-a", "actor-dissolve-b"} {
		decl := &eventsv1.DeclaredSession{DeclaredAt: 1000, ActorRef: ar, SessionDescriptor: []byte("shared")}
		if _, err := in.Append(ctx, decl, decl.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("Append DeclaredSession %s: %v", ar, err)
		}
	}
	if _, err := hypothesis.FormAll(ctx, sub, hypothesis.SessionDescriptorSharedV1{MinClusterSize: 2}, time.Now); err != nil {
		t.Fatalf("FormAll: %v", err)
	}

	var formationHash [32]byte
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType == "ghosttrace.events.v1.BehavioralClusterFormation" {
			formationHash = row.EventHash
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}

	if _, err := hypothesis.Promote(ctx, sub, hypothesis.PromoteOptions{
		FormationEventHash: formationHash,
		PromotedAt:         1716120000000000000,
		CadenceSeconds:     60,
		Reason:             "integration",
	}, time.Now); err != nil {
		t.Fatalf("Promote: %v", err)
	}

	var promotionHash [32]byte
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType == "ghosttrace.events.v1.BehavioralClusterPromotion" {
			promotionHash = row.EventHash
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}

	if _, err := hypothesis.Demote(ctx, sub, hypothesis.DemoteOptions{
		PromotionEventHash: promotionHash,
		DemotedAt:          1716120120000000000,
		Reason:             "integration close",
	}, time.Now); err != nil {
		t.Fatalf("Demote: %v", err)
	}

	if _, err := hypothesis.Dissolve(ctx, sub, hypothesis.DissolveOptions{
		FormationEventHash: formationHash,
		DissolvedAt:        1716120180000000000,
		Reason:             "integration terminal",
	}, time.Now); err != nil {
		t.Fatalf("Dissolve: %v", err)
	}

	report, err := Verify(ctx, sub, Options{})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if report.Failed() {
		t.Errorf("full-lifecycle substrate Verify reported failure: %+v", report)
	}
	// 2 DeclaredSession + 2 IngestionEvent + 1 Formation + 1 Promotion + 1 Demotion + 1 Dissolution = 8.
	if report.VerifiedCount != 8 {
		t.Errorf("VerifiedCount: got %d, want 8", report.VerifiedCount)
	}
}

// TestVerifyWithCatIIIFullLifecyclePlusMerge proves verify passes
// over a substrate containing FIVE Cat III lifecycle event types —
// formation, promotion, demotion, dissolution, and merge (per
// decision-log §0049 landing the fifth lifecycle operation). The
// substrate-integrity audit is type-agnostic across all Cat III
// lifecycle operations per Charter §2.5 BC5.
//
// The substrate is populated with THREE separate formations (alpha,
// beta, gamma) so the merge has two distinct antecedents and a
// distinct produced formation per the §0049 structural choice that
// preserves §0045 invariant (hypothesis identity IS the formation
// event's content-hash).
func TestVerifyWithCatIIIFullLifecyclePlusMerge(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	in := ingest.New(sub, time.Now)
	// Three descriptor groups, each meeting min-cluster-size=2 →
	// three formations.
	for _, item := range []struct {
		actor string
		desc  []byte
	}{
		{"actor-merge-a1", []byte("alpha")},
		{"actor-merge-a2", []byte("alpha")},
		{"actor-merge-b1", []byte("beta")},
		{"actor-merge-b2", []byte("beta")},
		{"actor-merge-g1", []byte("gamma")},
		{"actor-merge-g2", []byte("gamma")},
	} {
		decl := &eventsv1.DeclaredSession{DeclaredAt: 1000, ActorRef: item.actor, SessionDescriptor: item.desc}
		if _, err := in.Append(ctx, decl, decl.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("Append DeclaredSession %s: %v", item.actor, err)
		}
	}
	if _, err := hypothesis.FormAll(ctx, sub, hypothesis.SessionDescriptorSharedV1{MinClusterSize: 2}, time.Now); err != nil {
		t.Fatalf("FormAll: %v", err)
	}

	// Identify the three formations by walking their payloads.
	type fmEntry struct {
		hash [32]byte
		desc string
	}
	var entries []fmEntry
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
		var desc string
		for _, ar := range ev.ActorRefs {
			if strings.Contains(ar, "-a") {
				desc = "alpha"
				break
			}
			if strings.Contains(ar, "-b") {
				desc = "beta"
				break
			}
			if strings.Contains(ar, "-g") {
				desc = "gamma"
				break
			}
		}
		entries = append(entries, fmEntry{hash: row.EventHash, desc: desc})
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 formations; got %d", len(entries))
	}
	var alpha, beta, gamma [32]byte
	for _, e := range entries {
		switch e.desc {
		case "alpha":
			alpha = e.hash
		case "beta":
			beta = e.hash
		case "gamma":
			gamma = e.hash
		}
	}

	// Promote → demote → dissolve alpha; then merge alpha + beta → gamma.
	promRep, err := hypothesis.Promote(ctx, sub, hypothesis.PromoteOptions{
		FormationEventHash: alpha,
		PromotedAt:         1716120000000000000,
		CadenceSeconds:     60,
		Reason:             "integration merge",
	}, time.Now)
	if err != nil {
		t.Fatalf("Promote: %v", err)
	}
	var promotionHash [32]byte
	if raw, err := decodeHex(promRep.PromotionEventHashHex); err == nil {
		copy(promotionHash[:], raw)
	} else {
		t.Fatalf("decode promotion hash: %v", err)
	}
	if _, err := hypothesis.Demote(ctx, sub, hypothesis.DemoteOptions{
		PromotionEventHash: promotionHash,
		DemotedAt:          1716120120000000000,
		Reason:             "integration merge",
	}, time.Now); err != nil {
		t.Fatalf("Demote: %v", err)
	}
	if _, err := hypothesis.Dissolve(ctx, sub, hypothesis.DissolveOptions{
		FormationEventHash: alpha,
		DissolvedAt:        1716120180000000000,
		Reason:             "integration merge",
	}, time.Now); err != nil {
		t.Fatalf("Dissolve: %v", err)
	}
	if _, err := hypothesis.Merge(ctx, sub, hypothesis.MergeOptions{
		AntecedentAFormationHash: alpha,
		AntecedentBFormationHash: beta,
		ProducedFormationHash:    gamma,
		MergedAt:                 1716120240000000000,
		Reason:                   "integration merge — recognized as same phenomenon",
	}, time.Now); err != nil {
		t.Fatalf("Merge: %v", err)
	}

	report, err := Verify(ctx, sub, Options{})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if report.Failed() {
		t.Errorf("full-lifecycle-plus-merge substrate Verify reported failure: %+v", report)
	}
	// 6 DeclaredSession + 6 IngestionEvent + 3 Formation + 1 Promotion
	// + 1 Demotion + 1 Dissolution + 1 Merge = 19.
	if report.VerifiedCount != 19 {
		t.Errorf("VerifiedCount: got %d, want 19", report.VerifiedCount)
	}
}

func decodeHex(s string) ([]byte, error) {
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

// Sanity: every proto.Message field on the test message round-trips
// through canonical.Marshal without error.
func TestMessageRoundtrip(t *testing.T) {
	msg := &eventsv1.DeclaredSession{
		DeclaredAt:        42,
		ActorRef:          "round-trip",
		SessionDescriptor: []byte("rt"),
	}
	b, err := proto.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	want := canonical.Hash(b)
	if want == [32]byte{} {
		t.Error("expected non-zero hash")
	}
}
