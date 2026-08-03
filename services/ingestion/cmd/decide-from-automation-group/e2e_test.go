// End-to-end integration test for the FULL anti-bot vertical slice
// across all four layers (§0221 collection+inference+replay; §0222
// decision): ingest a JA3+JA4 observation → form a scored AutomationGroup
// → decide an enforcement verdict under a versioned policy → reconstruct
// the entire observation→inference→decision chain from the substrate.
//
// This is the test the §0221 Definition-of-Done described and the §0222
// decision layer completes: replay reconstructs the complete chain for
// a session, for audit.
package main

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/adapters/tls_fingerprint"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/decision"
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

func TestAntibotSlice_FullChain_ObservationToDecisionToReplay(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })
	clock := func() time.Time { return time.Unix(0, 1716120000000000777) }
	in := ingest.New(sub, clock)

	// (1) Collection — ingest JA3+JA4 as Cat I.
	input := `{"actor_ref":"actor-bot-1","endpoint_ref":"10.0.0.1:443","observed_at":1716120000000000000,` +
		`"ja4":"` + botJA4 + `","ja3":"` + botJA3 + `","sni_present":true,"alpn_protocols":["h2"]}` + "\n"
	if _, err := tls_fingerprint.Ingest(ctx, in, strings.NewReader(input), "tls-fingerprint-adapter:test", ingest.Envelope{Channel: "test"}); err != nil {
		t.Fatalf("ingest: %v", err)
	}

	// (2) Inference — signature → AutomationGroup candidate → formation.
	observations, err := observationcollector.CollectNetwork(ctx, sub)
	if err != nil {
		t.Fatalf("CollectNetwork: %v", err)
	}
	sig := &signatures.TLSJa4AutomationV1{KnownJA4: map[string]struct{}{botJA4: {}}}
	res, err := sig.EvaluateNetwork(ctx, observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(res.Candidates) != 1 {
		t.Fatalf("candidates: got %d want 1", len(res.Candidates))
	}
	cand := res.Candidates[0]
	formationHash, _, err := hypothesis.AutomationGroupFormationFromCandidate(ctx, sub, cand,
		hypothesis.AutomationGroupFormationFromCandidateOptions{
			FormationAt:            1716120100000000000,
			Confidence:             float32(cand.ConfidenceHint), // 0.5 for a single match
			EvidentialIndependence: &commonv1.EvidentialIndependence{Numerator: 1, Denominator: 1},
		}, clock)
	if err != nil {
		t.Fatalf("form: %v", err)
	}

	// (3) Decision — operator-elected verdict under a versioned policy.
	decRep, err := decision.DecideFromAutomationGroup(ctx, sub, decision.DecideOptions{
		FormationEventHash: formationHash,
		PolicyRef:          "automation-tiered-v1",
		OperatorRef:        "operator-e2e",
		DecidedAt:          1716120200000000000,
	}, clock)
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	// confidence 0.5 → CHALLENGE tier.
	if decRep.Verdict != "CHALLENGE" {
		t.Fatalf("verdict: got %q want CHALLENGE (confidence 0.5)", decRep.Verdict)
	}

	var auditHash [32]byte
	if _, err := hexInto(decRep.AuditEventHashHex, auditHash[:]); err != nil {
		t.Fatalf("decode audit hash: %v", err)
	}

	// (4) Replay — reconstruct observation→inference→decision.
	prov, err := replay.ReconstructDecisionProvenance(ctx, sub, auditHash)
	if err != nil {
		t.Fatalf("ReconstructDecisionProvenance: %v", err)
	}
	if prov.Verdict != "CHALLENGE" || prov.SubjectActorRef != "actor-bot-1" {
		t.Errorf("decision: verdict=%q subject=%q", prov.Verdict, prov.SubjectActorRef)
	}
	if prov.PolicyRef != "automation-tiered-v1" || prov.OperatorRef != "operator-e2e" {
		t.Errorf("policy/operator: %q / %q", prov.PolicyRef, prov.OperatorRef)
	}
	if len(prov.InfluencingHypotheses) != 1 {
		t.Fatalf("influencing hypotheses: got %d want 1", len(prov.InfluencingHypotheses))
	}
	hyp := prov.InfluencingHypotheses[0]
	if hyp.PatternSignature != "tls_ja4_automation_v1" {
		t.Errorf("hypothesis pattern_signature: got %q", hyp.PatternSignature)
	}
	if len(hyp.ResolvedSources) != 1 {
		t.Fatalf("hypothesis sources: got %d want 1", len(hyp.ResolvedSources))
	}
	obs := hyp.ResolvedSources[0]
	if !obs.Found || obs.JA4 != botJA4 || obs.JA3 != botJA3 {
		t.Errorf("reconstructed observation: found=%v ja4=%q ja3=%q", obs.Found, obs.JA4, obs.JA3)
	}
	if obs.ActorRef != "actor-bot-1" {
		t.Errorf("reconstructed actor: %q", obs.ActorRef)
	}

	t.Logf("FULL CHAIN OK: obs(JA4=%s JA3=%s) -> AutomationGroup(conf=%.2f) -> decision(%s by %s under %s) -> chain reconstructed from substrate",
		obs.JA4, obs.JA3, cand.ConfidenceHint, prov.Verdict, prov.OperatorRef, prov.PolicyRef)
}

// hexInto decodes hex into dst.
func hexInto(s string, dst []byte) (int, error) {
	if len(s) != 2*len(dst) {
		return 0, errLen{}
	}
	for i := 0; i < len(dst); i++ {
		dst[i] = byte(nib(s[2*i])<<4 | nib(s[2*i+1]))
	}
	return len(dst), nil
}

type errLen struct{}

func (errLen) Error() string { return "hex length mismatch" }

func nib(c byte) int {
	switch {
	case '0' <= c && c <= '9':
		return int(c - '0')
	case 'a' <= c && c <= 'f':
		return int(c-'a') + 10
	case 'A' <= c && c <= 'F':
		return int(c-'A') + 10
	}
	return 0
}
