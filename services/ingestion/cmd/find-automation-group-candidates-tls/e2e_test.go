// End-to-end integration test for the §0221 TLS fingerprint vertical
// slice: ingest a JA3+JA4 observation as Cat I → evaluate the
// tls_ja4_automation_v1 signature → commit an AutomationGroupFormation
// (Cat III, scored) from the candidate → reconstruct the
// observation→inference provenance chain for audit.
//
// Covers the three constitutional layers the slice implements:
//   - Collection: the fingerprint enters the substrate as an OBSERVATION
//     (NetworkObservation / tls_ja4), never a verdict.
//   - Inference: the scored hypothesis (confidence + evidential_-
//     independence as SEPARATE dimensions) carries provenance back to
//     the observation that produced it.
//   - Replay: the chain observation→inference is reconstructible from
//     the substrate alone, surfacing the exact JA3/JA4 that grounded it.
//
// The Decision layer is intentionally absent — it is deferred to the
// ontology-revision RFC opened alongside §0221 (no decision record type
// exists in any category yet). This test asserts the chain that IS
// constitutional today; it does not fabricate a decision event.
package main

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/adapters/tls_fingerprint"
	commonv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/common/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/hypothesis"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/observationcollector"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/replay"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/signatures"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

const (
	botJA4 = "t13d1516h2_8daaf6152771_b186095e22b6"
	botJA3 = "ada70206e40642a3e4461f35503241d5"
)

func TestTLSFingerprintSlice_ObservationToInferenceToReplay(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })
	clock := func() time.Time { return time.Unix(0, 1716120000000000777) }
	in := ingest.New(sub, clock)

	// --- Collection: ingest JA3+JA4 as Cat I NetworkObservation -------
	const obsAt = int64(1716120000_000_000_000)
	input := `{"actor_ref":"actor-bot-1","endpoint_ref":"10.0.0.1:443","observed_at":1716120000000000000,` +
		`"ja4":"` + botJA4 + `","ja4_raw":"t13d1516h2_x_y","ja3":"` + botJA3 + `","ja3_raw":"769,47-53,0-10,23,0",` +
		`"sni_present":true,"alpn_protocols":["h2","http/1.1"]}` + "\n" +
		// A second actor whose fingerprint is NOT in the known set — must
		// NOT become a candidate (negative control).
		`{"actor_ref":"actor-human-1","endpoint_ref":"10.0.0.2:443","observed_at":1716120001000000000,` +
		`"ja4":"t13d1517h2_aaaaaaaaaaaa_bbbbbbbbbbbb","ja3":"00000000000000000000000000000000"}` + "\n"

	report, err := tls_fingerprint.Ingest(ctx, in, strings.NewReader(input), "tls-fingerprint-adapter:test", ingest.Envelope{Channel: "test"})
	if err != nil {
		t.Fatalf("tls_fingerprint.Ingest: %v", err)
	}
	if report.ObservationsCommitted != 2 {
		t.Fatalf("ObservationsCommitted: got %d want 2", report.ObservationsCommitted)
	}

	// --- Inference: evaluate the signature against the substrate ------
	observations, err := observationcollector.CollectNetwork(ctx, sub)
	if err != nil {
		t.Fatalf("CollectNetwork: %v", err)
	}
	sig := &signatures.TLSJa4AutomationV1{
		KnownJA4: map[string]struct{}{botJA4: {}},
		KnownJA3: map[string]struct{}{botJA3: {}},
	}
	result, err := sig.EvaluateNetwork(ctx, observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("candidates: got %d want 1 (only the known-automation fingerprint should match)", len(result.Candidates))
	}
	cand := result.Candidates[0]
	if len(cand.ActorRefs) != 1 || cand.ActorRefs[0] != "actor-bot-1" {
		t.Fatalf("candidate actor_refs: got %v want [actor-bot-1]", cand.ActorRefs)
	}
	if cand.ConfidenceHint <= 0 {
		t.Fatalf("ConfidenceHint: got %v want > 0 (scored hypothesis)", cand.ConfidenceHint)
	}

	// --- Commit the scored AutomationGroupFormation (Cat III) ---------
	// confidence + evidential_independence committed as SEPARATE
	// dimensions per §2.6 paired-dimension commitment.
	formationAt := int64(1716120100_000_000_000)
	formationHash, already, err := hypothesis.AutomationGroupFormationFromCandidate(ctx, sub, cand,
		hypothesis.AutomationGroupFormationFromCandidateOptions{
			PatternParameters:      "known_set_size=1",
			FormationAt:            formationAt,
			Confidence:             float32(cand.ConfidenceHint),
			EvidentialIndependence: &commonv1.EvidentialIndependence{Numerator: 1, Denominator: 1},
		}, clock)
	if err != nil {
		t.Fatalf("AutomationGroupFormationFromCandidate: %v", err)
	}
	if already {
		t.Fatalf("expected a newly committed formation, got alreadyPresent")
	}

	// --- Replay: reconstruct observation→inference provenance ---------
	prov, err := replay.ReconstructAutomationGroupProvenance(ctx, sub, formationHash)
	if err != nil {
		t.Fatalf("ReconstructAutomationGroupProvenance: %v", err)
	}
	if prov.PatternSignature != "tls_ja4_automation_v1" {
		t.Errorf("PatternSignature: got %q want tls_ja4_automation_v1", prov.PatternSignature)
	}
	// Paired dimensions present + separate.
	if prov.Confidence <= 0 {
		t.Errorf("Confidence: got %v want > 0 (scored)", prov.Confidence)
	}
	if prov.EvidentialDen == 0 {
		t.Errorf("EvidentialIndependence denominator is 0; paired-dimension not committed")
	}
	if prov.SourceCount != 1 {
		t.Fatalf("SourceCount: got %d want 1", prov.SourceCount)
	}

	// The provenance chain must resolve to the Cat I observation carrying
	// the exact JA3/JA4 that grounded the inference.
	src := prov.ResolvedSources[0]
	if !src.Found {
		t.Fatalf("source not resolved against substrate; chain broken")
	}
	if src.MessageType != "ghosttrace.events.v1.NetworkObservation" {
		t.Errorf("source MessageType: got %q want NetworkObservation", src.MessageType)
	}
	if src.JA4 != botJA4 {
		t.Errorf("reconstructed JA4: got %q want %q", src.JA4, botJA4)
	}
	if src.JA3 != botJA3 {
		t.Errorf("reconstructed JA3: got %q want %q", src.JA3, botJA3)
	}
	if src.ActorRef != "actor-bot-1" {
		t.Errorf("reconstructed actor_ref: got %q want actor-bot-1", src.ActorRef)
	}

	_ = obsAt // documents the ingested observation time for readers
	t.Logf("slice OK: obs(JA4=%s JA3=%s actor=%s) -> AutomationGroup(conf=%.3f EI=%d/%d) -> chain reconstructed",
		src.JA4, src.JA3, src.ActorRef, prov.Confidence, prov.EvidentialNum, prov.EvidentialDen)
}
