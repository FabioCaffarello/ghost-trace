// Package decision is the operator-elected enforcement-decision layer
// per decision-log §0222 (Framing A: Decision = Cat I audit record).
// It is the fourth layer of the anti-bot vertical slice (§0221 added
// collection + inference + replay).
//
// Constitutional posture (§3 N3 — no autonomous irreversible action):
// nothing in this package commits a decision on its own. DecideFrom-
// AutomationGroup is invoked by an operator-elected path (the
// decide-from-automation-group CLI) that names an operator_ref; the
// committed OperationalDecisionAudit is the substrate evidence that the
// action was operator-initiated, not autonomous. Signatures never reach
// this package.
//
// The policy evaluation (inference → verdict) is DETERMINISTIC given a
// versioned policy + the referenced hypothesis. Under Framing A the
// evaluation is applied in-process here and only the RESULT (the audit)
// is committed; the audit records policy_ref so the verdict is re-
// derivable by looking up that policy's versioned thresholds against the
// referenced hypothesis. Modelling the evaluation itself as a Cat II
// DecisionConstruct (Framing B) is deferred per §0222.
package decision

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

const automationGroupFormationMessageType = "ghosttrace.events.v1.AutomationGroupFormation"

// Policy maps a Cat III AutomationGroup inference to an enforcement
// verdict, deterministically, under a versioned identifier. The policy
// is operator-owned; it is NOT a substrate record under Framing A.
type Policy interface {
	// Ref is the versioned policy identifier recorded in the audit's
	// policy_ref field.
	Ref() string
	// Evaluate maps the formation to a verdict. Deterministic.
	Evaluate(formation *eventsv1.AutomationGroupFormation) (eventsv1.OperationalDecisionAudit_DecisionType, error)
}

// AutomationTieredV1 maps the formation's committed `confidence` to a
// verdict via fixed thresholds. The CONFIDENCE dimension is the sole
// input — `evidential_independence` is deliberately NOT consulted here,
// honoring the §2.6 separation of the two dimensions (a policy that
// wanted to gate on independence would be a distinct versioned policy).
//
//	confidence >= 0.8 -> BLOCK
//	confidence >= 0.5 -> CHALLENGE
//	confidence >  0.0 -> SHADOW
//	confidence == 0.0 -> ALLOW
type AutomationTieredV1 struct{}

// Ref identifies the policy version.
func (AutomationTieredV1) Ref() string { return "automation-tiered-v1" }

// Evaluate applies the tiered thresholds to the formation's confidence.
func (AutomationTieredV1) Evaluate(f *eventsv1.AutomationGroupFormation) (eventsv1.OperationalDecisionAudit_DecisionType, error) {
	if f == nil {
		return eventsv1.OperationalDecisionAudit_DECISION_TYPE_UNSPECIFIED, errors.New("decision.AutomationTieredV1: nil formation")
	}
	c := f.GetConfidence()
	switch {
	case c >= 0.8:
		return eventsv1.OperationalDecisionAudit_DECISION_TYPE_BLOCK, nil
	case c >= 0.5:
		return eventsv1.OperationalDecisionAudit_DECISION_TYPE_CHALLENGE, nil
	case c > 0.0:
		return eventsv1.OperationalDecisionAudit_DECISION_TYPE_SHADOW, nil
	default:
		return eventsv1.OperationalDecisionAudit_DECISION_TYPE_ALLOW, nil
	}
}

// ResolvePolicy maps a versioned policy_ref to its implementation.
// Inception corpus carries one policy; new policies register here.
func ResolvePolicy(ref string) (Policy, error) {
	switch ref {
	case "", AutomationTieredV1{}.Ref():
		return AutomationTieredV1{}, nil
	default:
		return nil, fmt.Errorf("decision.ResolvePolicy: unknown policy_ref %q (known: %s)", ref, AutomationTieredV1{}.Ref())
	}
}

// DecideOptions carries the operator-elected inputs for
// DecideFromAutomationGroup.
type DecideOptions struct {
	// FormationEventHash is the AutomationGroupFormation the decision
	// acts on (the inference "this actor is automated").
	FormationEventHash [32]byte
	// PolicyRef selects the versioned policy; empty = the default
	// (automation-tiered-v1).
	PolicyRef string
	// OperatorRef names the elector (§3 N3 operator-initiated evidence).
	OperatorRef string
	// DecidedAt is the decision time (Unix nanoseconds); 0 = now().
	DecidedAt int64
}

// DecideReport is the per-DecideFromAutomationGroup outcome.
type DecideReport struct {
	AuditEventHashHex string `json:"audit_event_hash_hex"`
	Verdict           string `json:"verdict"`
	SubjectActorRef   string `json:"subject_actor_ref"`
	PolicyRef         string `json:"policy_ref"`
	AlreadyPresent    bool   `json:"already_present"`
}

// DecideFromAutomationGroup evaluates the configured policy against an
// AutomationGroupFormation and commits a single OperationalDecisionAudit
// recording the verdict. Idempotent via content-addressed immutability
// (re-running identical inputs commits no new row).
//
// Inception scope: single-actor AutomationGroup formations only (the
// TLS slice's shape). A multi-actor formation has no single subject
// actor for one decision; the function returns an error directing the
// operator to a per-actor decision path (deferred).
func DecideFromAutomationGroup(ctx context.Context, sub *substrate.Substrate, opts DecideOptions, now func() time.Time) (DecideReport, error) {
	if now == nil {
		now = time.Now
	}
	policy, err := ResolvePolicy(opts.PolicyRef)
	if err != nil {
		return DecideReport{}, err
	}

	row, err := sub.LookupRow(ctx, opts.FormationEventHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return DecideReport{}, fmt.Errorf("decision.DecideFromAutomationGroup: formation %x not found", opts.FormationEventHash)
		}
		return DecideReport{}, fmt.Errorf("decision.DecideFromAutomationGroup: lookup formation: %w", err)
	}
	if row.MessageType != automationGroupFormationMessageType {
		return DecideReport{}, fmt.Errorf("decision.DecideFromAutomationGroup: target %x is %q, want AutomationGroupFormation", opts.FormationEventHash, row.MessageType)
	}

	payload, err := sub.ReadBlob(ctx, opts.FormationEventHash)
	if err != nil {
		return DecideReport{}, fmt.Errorf("decision.DecideFromAutomationGroup: read formation blob: %w", err)
	}
	formation := &eventsv1.AutomationGroupFormation{}
	if err := proto.Unmarshal(payload, formation); err != nil {
		return DecideReport{}, fmt.Errorf("decision.DecideFromAutomationGroup: unmarshal formation: %w", err)
	}

	actors := formation.GetActorRefs()
	if len(actors) != 1 {
		return DecideReport{}, fmt.Errorf("decision.DecideFromAutomationGroup: formation has %d actor_refs; inception scope decides single-actor AutomationGroups only", len(actors))
	}
	subjectActor := actors[0]

	verdict, err := policy.Evaluate(formation)
	if err != nil {
		return DecideReport{}, fmt.Errorf("decision.DecideFromAutomationGroup: policy evaluate: %w", err)
	}

	decidedAt := opts.DecidedAt
	if decidedAt == 0 {
		decidedAt = now().UnixNano()
	}

	// source_observation_hashes: the §2.3 chain inherited from the
	// formation. influencing_hypothesis_hashes: the formation itself.
	// Both follow the §0139 element-shape discipline (32 bytes, ascending,
	// no duplicates).
	srcHashes := sortedDedupedHashes(formation.GetSourceEventHashes())
	hypHash := make([]byte, 32)
	copy(hypHash, opts.FormationEventHash[:])

	audit := &eventsv1.OperationalDecisionAudit{
		DecisionType:                verdict,
		SubjectActorRef:             subjectActor,
		DecidedAt:                   decidedAt,
		SourceObservationHashes:     srcHashes,
		InfluencingHypothesisHashes: [][]byte{hypHash},
		PolicyRef:                   policy.Ref(),
		OperatorRef:                 opts.OperatorRef,
	}

	auditPayload, auditHash, err := canonical.MarshalAndHash(audit)
	if err != nil {
		return DecideReport{}, fmt.Errorf("decision.DecideFromAutomationGroup: marshal audit: %w", err)
	}
	hex := canonical.HashHex(auditHash)

	_, lookupErr := sub.LookupRow(ctx, auditHash)
	alreadyPresent := lookupErr == nil
	if lookupErr != nil && !errors.Is(lookupErr, sql.ErrNoRows) {
		return DecideReport{}, fmt.Errorf("decision.DecideFromAutomationGroup: lookup audit %s: %w", hex, lookupErr)
	}

	auditRow := substrate.EventRow{
		EventHash:   auditHash,
		EventTime:   decidedAt,
		MessageType: string(audit.ProtoReflect().Descriptor().FullName()),
		PayloadRef:  hex[:2] + "/" + hex[2:],
		CommittedAt: now().UnixNano(),
	}
	if err := sub.Append(ctx, auditRow, auditPayload); err != nil {
		return DecideReport{}, fmt.Errorf("decision.DecideFromAutomationGroup: append audit %s: %w", hex, err)
	}

	return DecideReport{
		AuditEventHashHex: hex,
		Verdict:           VerdictName(verdict),
		SubjectActorRef:   subjectActor,
		PolicyRef:         policy.Ref(),
		AlreadyPresent:    alreadyPresent,
	}, nil
}

// VerdictName renders a DecisionType as its short operator-facing name
// (ALLOW / CHALLENGE / BLOCK / SHADOW / UNSPECIFIED).
func VerdictName(v eventsv1.OperationalDecisionAudit_DecisionType) string {
	switch v {
	case eventsv1.OperationalDecisionAudit_DECISION_TYPE_ALLOW:
		return "ALLOW"
	case eventsv1.OperationalDecisionAudit_DECISION_TYPE_CHALLENGE:
		return "CHALLENGE"
	case eventsv1.OperationalDecisionAudit_DECISION_TYPE_BLOCK:
		return "BLOCK"
	case eventsv1.OperationalDecisionAudit_DECISION_TYPE_SHADOW:
		return "SHADOW"
	default:
		return "UNSPECIFIED"
	}
}

// sortedDedupedHashes returns a copy of the input hash list in ascending
// lexicographic order with duplicates removed, per the §0139 hash-list
// element-shape discipline. Non-32-byte entries are preserved as-is
// (the substrate's content-hash references are always 32 bytes; defensive).
func sortedDedupedHashes(in [][]byte) [][]byte {
	if len(in) == 0 {
		return nil
	}
	out := make([][]byte, 0, len(in))
	for _, h := range in {
		c := make([]byte, len(h))
		copy(c, h)
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return bytesLess(out[i], out[j]) })
	deduped := out[:0]
	var prev []byte
	for _, h := range out {
		if prev != nil && bytesEqual(prev, h) {
			continue
		}
		deduped = append(deduped, h)
		prev = h
	}
	return deduped
}

func bytesLess(a, b []byte) bool {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return len(a) < len(b)
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
