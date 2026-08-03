package decision

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	commonv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/common/v1"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/hypothesis"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/signatures"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

func TestAutomationTieredV1_Thresholds(t *testing.T) {
	p := AutomationTieredV1{}
	cases := []struct {
		conf float32
		want eventsv1.OperationalDecisionAudit_DecisionType
	}{
		{0.95, eventsv1.OperationalDecisionAudit_DECISION_TYPE_BLOCK},
		{0.80, eventsv1.OperationalDecisionAudit_DECISION_TYPE_BLOCK},
		{0.79, eventsv1.OperationalDecisionAudit_DECISION_TYPE_CHALLENGE},
		{0.50, eventsv1.OperationalDecisionAudit_DECISION_TYPE_CHALLENGE},
		{0.49, eventsv1.OperationalDecisionAudit_DECISION_TYPE_SHADOW},
		{0.01, eventsv1.OperationalDecisionAudit_DECISION_TYPE_SHADOW},
		{0.0, eventsv1.OperationalDecisionAudit_DECISION_TYPE_ALLOW},
	}
	for _, c := range cases {
		got, err := p.Evaluate(&eventsv1.AutomationGroupFormation{Confidence: c.conf})
		if err != nil {
			t.Fatalf("Evaluate(%v): %v", c.conf, err)
		}
		if got != c.want {
			t.Errorf("Evaluate(%v) = %v, want %v", c.conf, got, c.want)
		}
	}
}

func TestResolvePolicy(t *testing.T) {
	if _, err := ResolvePolicy(""); err != nil {
		t.Errorf("empty policy_ref should default, got %v", err)
	}
	if _, err := ResolvePolicy("automation-tiered-v1"); err != nil {
		t.Errorf("known policy_ref: %v", err)
	}
	if _, err := ResolvePolicy("nope"); err == nil {
		t.Error("unknown policy_ref should error")
	}
}

func newDecisionSubstrate(t *testing.T) *substrate.Substrate {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })
	return sub
}

// commitFormation commits an AutomationGroupFormation from a synthetic
// candidate with the given actors + confidence, returning its hash.
func commitFormation(t *testing.T, sub *substrate.Substrate, actors []string, confidence float32, sources [][]byte) [32]byte {
	t.Helper()
	ctx := context.Background()
	cand := &signatures.FormationCandidate{
		SignatureName:     "tls_ja4_automation_v1",
		HypothesisSubtype: signatures.HypothesisSubtypeAutomationGroup,
		ActorRefs:         actors,
		SourceHashes:      sources,
	}
	now := func() time.Time { return time.Unix(0, 1716120000000000000) }
	hash, _, err := hypothesis.AutomationGroupFormationFromCandidate(ctx, sub, cand,
		hypothesis.AutomationGroupFormationFromCandidateOptions{
			FormationAt:            1716120000000000000,
			Confidence:             confidence,
			EvidentialIndependence: &commonv1.EvidentialIndependence{Numerator: 1, Denominator: 1},
		}, now)
	if err != nil {
		t.Fatalf("commitFormation: %v", err)
	}
	return hash
}

func TestDecideFromAutomationGroup_CommitsAudit(t *testing.T) {
	sub := newDecisionSubstrate(t)
	ctx := context.Background()
	src := []byte("01234567890123456789012345678901") // 32 bytes
	formationHash := commitFormation(t, sub, []string{"actor-bot-1"}, 0.9, [][]byte{src})

	now := func() time.Time { return time.Unix(0, 1716120100000000000) }
	rep, err := DecideFromAutomationGroup(ctx, sub, DecideOptions{
		FormationEventHash: formationHash,
		OperatorRef:        "operator-test",
		DecidedAt:          1716120100000000000,
	}, now)
	if err != nil {
		t.Fatalf("DecideFromAutomationGroup: %v", err)
	}
	if rep.Verdict != "BLOCK" {
		t.Errorf("verdict: got %q want BLOCK (confidence 0.9)", rep.Verdict)
	}
	if rep.SubjectActorRef != "actor-bot-1" {
		t.Errorf("subject: got %q", rep.SubjectActorRef)
	}
	if rep.PolicyRef != "automation-tiered-v1" {
		t.Errorf("policy_ref: got %q", rep.PolicyRef)
	}
	if rep.AlreadyPresent {
		t.Error("first decision should not be alreadyPresent")
	}

	// Re-decide: idempotent (content-addressed).
	rep2, err := DecideFromAutomationGroup(ctx, sub, DecideOptions{
		FormationEventHash: formationHash,
		OperatorRef:        "operator-test",
		DecidedAt:          1716120100000000000,
	}, now)
	if err != nil {
		t.Fatalf("re-decide: %v", err)
	}
	if !rep2.AlreadyPresent {
		t.Error("re-decide should be alreadyPresent (idempotent)")
	}

	// Inspect the committed audit.
	var auditHash [32]byte
	if _, err := hexInto(rep.AuditEventHashHex, auditHash[:]); err != nil {
		t.Fatalf("decode audit hash: %v", err)
	}
	row, err := sub.LookupRow(ctx, auditHash)
	if err != nil {
		t.Fatalf("lookup audit: %v", err)
	}
	if row.MessageType != "ghosttrace.events.v1.OperationalDecisionAudit" {
		t.Errorf("audit MessageType: got %q", row.MessageType)
	}
	payload, err := sub.ReadBlob(ctx, auditHash)
	if err != nil {
		t.Fatalf("read audit: %v", err)
	}
	audit := &eventsv1.OperationalDecisionAudit{}
	if err := proto.Unmarshal(payload, audit); err != nil {
		t.Fatalf("unmarshal audit: %v", err)
	}
	if audit.GetDecisionType() != eventsv1.OperationalDecisionAudit_DECISION_TYPE_BLOCK {
		t.Errorf("audit decision_type: got %v", audit.GetDecisionType())
	}
	if audit.GetOperatorRef() != "operator-test" {
		t.Errorf("audit operator_ref: got %q (§3 N3 evidence missing)", audit.GetOperatorRef())
	}
	if len(audit.GetSourceObservationHashes()) != 1 || !bytes.Equal(audit.GetSourceObservationHashes()[0], src) {
		t.Errorf("audit source_observation_hashes: got %v want [%x]", audit.GetSourceObservationHashes(), src)
	}
	if len(audit.GetInfluencingHypothesisHashes()) != 1 || !bytes.Equal(audit.GetInfluencingHypothesisHashes()[0], formationHash[:]) {
		t.Errorf("audit influencing_hypothesis_hashes does not reference the formation")
	}
}

func TestDecideFromAutomationGroup_RequiresOperatorViaCaller(t *testing.T) {
	// The package allows empty OperatorRef (the CLI enforces non-empty);
	// confirm a low-confidence formation yields ALLOW.
	sub := newDecisionSubstrate(t)
	ctx := context.Background()
	formationHash := commitFormation(t, sub, []string{"actor-1"}, 0.0, nil)
	rep, err := DecideFromAutomationGroup(ctx, sub, DecideOptions{
		FormationEventHash: formationHash,
		OperatorRef:        "op",
		DecidedAt:          1,
	}, func() time.Time { return time.Unix(0, 2) })
	if err != nil {
		t.Fatalf("DecideFromAutomationGroup: %v", err)
	}
	if rep.Verdict != "ALLOW" {
		t.Errorf("verdict: got %q want ALLOW (confidence 0.0)", rep.Verdict)
	}
}

func TestDecideFromAutomationGroup_MultiActorRejected(t *testing.T) {
	sub := newDecisionSubstrate(t)
	ctx := context.Background()
	formationHash := commitFormation(t, sub, []string{"a1", "a2"}, 0.9, nil)
	_, err := DecideFromAutomationGroup(ctx, sub, DecideOptions{
		FormationEventHash: formationHash,
		OperatorRef:        "op",
	}, func() time.Time { return time.Unix(0, 1) })
	if err == nil {
		t.Fatal("multi-actor formation should be rejected at inception scope")
	}
}

// hexInto decodes a hex string into dst.
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
