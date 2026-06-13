package replay

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// FormationProvenanceReport is the outcome of reconstructing a Cat III
// formation's observational-provenance chain per decision-log §0221.
//
// Distinct from ReplayAutomationGroupFormation (Phase-3 DETERMINISTIC
// replay), which re-derives a formation from a REGISTERED formation
// pattern (uniform-cadence-v1) and compares hashes. F3-candidate-derived
// formations (e.g. from tls_ja4_automation_v1) carry a pattern_signature
// that is a SIGNATURE name, not a registered substrate-walk pattern, so
// deterministic replay reports ErrPatternUnknown for them. This function
// covers the complementary §replay-model Phase-3 RECONSTRUCTIVE guarantee
// for that path: it resolves the formation's committed source_event_hashes
// (§2.3 observational provenance) back to the Cat I observations that
// produced the hypothesis, reconstructing the observation→inference chain
// for audit — the auditability the §0221 vertical slice requires.
//
// Non-redundancy with measure-chain-morphology (§0155): morphology
// computes chain depth/breadth metrics; this resolves the actual source
// records and surfaces the observed TLS fingerprint, answering "which
// observations grounded this hypothesis, and what did they observe?"
type FormationProvenanceReport struct {
	FormationHashHex  string                   `json:"formation_hash_hex"`
	MessageType       string                   `json:"message_type"`
	PatternSignature  string                   `json:"pattern_signature"`
	PatternParameters string                   `json:"pattern_parameters"`
	ActorRefs         []string                 `json:"actor_refs"`
	Confidence        float32                  `json:"confidence"`
	EvidentialNum     uint64                   `json:"evidential_independence_numerator"`
	EvidentialDen     uint64                   `json:"evidential_independence_denominator"`
	SourceCount       int                      `json:"source_count"`
	ResolvedSources   []ResolvedSource         `json:"resolved_sources"`
}

// ResolvedSource is one entry in the provenance chain: a source event
// hash from the formation's source_event_hashes resolved against the
// substrate. When the source is a Cat I NetworkObservation carrying the
// tls_ja4 sub-modality, the observed fingerprint is surfaced so an
// auditor sees the exact JA3/JA4 that grounded the inference.
type ResolvedSource struct {
	HashHex     string `json:"hash_hex"`
	Found       bool   `json:"found"`
	MessageType string `json:"message_type,omitempty"`
	ActorRef    string `json:"actor_ref,omitempty"`
	EndpointRef string `json:"endpoint_ref,omitempty"`
	JA4         string `json:"ja4,omitempty"`
	JA3         string `json:"ja3,omitempty"`
}

// ReconstructAutomationGroupProvenance resolves the observational
// provenance chain of an AutomationGroupFormation. Returns ErrTargetNotFound
// when the formation hash is absent, ErrTargetWrongType when the hash
// addresses a non-AutomationGroupFormation record.
func ReconstructAutomationGroupProvenance(ctx context.Context, sub *substrate.Substrate, targetHash [32]byte) (FormationProvenanceReport, error) {
	row, err := sub.LookupRow(ctx, targetHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return FormationProvenanceReport{}, fmt.Errorf("%w: %x", ErrTargetNotFound, targetHash)
		}
		return FormationProvenanceReport{}, fmt.Errorf("replay.ReconstructAutomationGroupProvenance: lookup target: %w", err)
	}
	if row.MessageType != automationGroupFormationMessageType {
		return FormationProvenanceReport{}, fmt.Errorf("%w: %x is %q (expected %s)",
			ErrTargetWrongType, targetHash, row.MessageType, automationGroupFormationMessageType)
	}

	payload, err := sub.ReadBlob(ctx, targetHash)
	if err != nil {
		return FormationProvenanceReport{}, fmt.Errorf("replay.ReconstructAutomationGroupProvenance: read target blob: %w", err)
	}
	formation := &eventsv1.AutomationGroupFormation{}
	if err := proto.Unmarshal(payload, formation); err != nil {
		return FormationProvenanceReport{}, fmt.Errorf("replay.ReconstructAutomationGroupProvenance: unmarshal target: %w", err)
	}

	report := FormationProvenanceReport{
		FormationHashHex:  canonical.HashHex(targetHash),
		MessageType:       row.MessageType,
		PatternSignature:  formation.GetPatternSignature(),
		PatternParameters: formation.GetPatternParameters(),
		ActorRefs:         formation.GetActorRefs(),
		Confidence:        formation.GetConfidence(),
		SourceCount:       len(formation.GetSourceEventHashes()),
		ResolvedSources:   make([]ResolvedSource, 0, len(formation.GetSourceEventHashes())),
	}
	if ei := formation.GetEvidentialIndependence(); ei != nil {
		report.EvidentialNum = ei.GetNumerator()
		report.EvidentialDen = ei.GetDenominator()
	}

	for _, sh := range formation.GetSourceEventHashes() {
		report.ResolvedSources = append(report.ResolvedSources, resolveSource(ctx, sub, sh))
	}
	return report, nil
}

// resolveSource looks up a single source event hash and, when it
// addresses a Cat I NetworkObservation, surfaces the observed TLS
// fingerprint. Unknown / unresolvable hashes are recorded with
// Found=false rather than failing the whole reconstruction (a missing
// source is itself an auditable signal).
func resolveSource(ctx context.Context, sub *substrate.Substrate, sourceHash []byte) ResolvedSource {
	out := ResolvedSource{HashHex: hex.EncodeToString(sourceHash)}
	if len(sourceHash) != 32 {
		return out
	}
	var h [32]byte
	copy(h[:], sourceHash)

	row, err := sub.LookupRow(ctx, h)
	if err != nil {
		return out // Found stays false (sql.ErrNoRows or lookup failure).
	}
	out.Found = true
	out.MessageType = row.MessageType

	if row.MessageType != networkObservationMessageType {
		return out
	}
	payload, err := sub.ReadBlob(ctx, h)
	if err != nil {
		return out
	}
	obs := &eventsv1.NetworkObservation{}
	if err := proto.Unmarshal(payload, obs); err != nil {
		return out
	}
	out.ActorRef = obs.GetActorRef()
	out.EndpointRef = obs.GetEndpointRef()
	if tls := obs.GetTlsJa4(); tls != nil {
		out.JA4 = tls.GetJa4()
		out.JA3 = tls.GetJa3()
	}
	return out
}
