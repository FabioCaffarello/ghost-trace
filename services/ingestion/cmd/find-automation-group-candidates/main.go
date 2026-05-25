// Command find-automation-group-candidates is the F3 orchestrator
// per decision-log §0153: reads BrowserObservation records from the
// substrate, invokes the cdp_marker_density_v1 signature (§0152),
// and emits FormationCandidates to stdout as structured JSON for
// operator review.
//
// Per §3 N3 + §0152: this command does NOT commit formation events
// directly. Operator reviews candidates + decides whether to commit
// each via the existing `form-automation-group` CLI (paired with
// the candidate's actor_ref + source-hash list).
//
// Per §0143 + §0151 + §0152: this is the first F3 inference loop
// end-to-end exercise — Cat I ingestion (CIC-IDS via §0145; F1 atlas
// via §0144+§0146+§0147+§0151) → F3 signature (§0152) →
// operator-reviewed candidate (this command). Subsequent operator
// commit + Layer A/B candidacy + demotion close the loop per §0143
// Sub-benchmark 1 criterion.
//
// Read-only over the substrate; no AppendPair invocation; no writeMu
// contention; safe to invoke against a substrate concurrently being
// written by ingestion paths.
//
// Exit codes: 0 success (including empty candidate set); 2
// tool/config error (invalid flags, substrate open failure).
package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/observationcollector"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/signatures"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

const exitToolError = 2

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "find-automation-group-candidates: %v\n", err)
		os.Exit(exitToolError)
	}
}

func run() error {
	dbPath := flag.String("db", "./ghost-trace.db", "SQLite primary-event-log path")
	blobDir := flag.String("blobs", "./blobs", "content-addressed blob-store directory")
	threshold := flag.Uint("threshold", 0, "cdp_marker_density_v1 threshold override (0 = signature default = 2)")
	limitCandidates := flag.Int("limit", 0, "maximum candidates to emit (0 = unlimited)")
	flag.Parse()

	ctx := context.Background()
	sub, err := substrate.Open(ctx, *dbPath, *blobDir)
	if err != nil {
		return fmt.Errorf("substrate.Open: %w", err)
	}
	defer sub.Close()

	observations, err := observationcollector.CollectBrowser(ctx, sub)
	if err != nil {
		return fmt.Errorf("collect: %w", err)
	}

	sig := &signatures.CDPMarkerDensityV1{Threshold: uint32(*threshold)}
	result, err := sig.EvaluateBrowser(ctx, observations)
	if err != nil {
		return fmt.Errorf("signature.EvaluateBrowser: %w", err)
	}

	candidates := result.Candidates
	if *limitCandidates > 0 && len(candidates) > *limitCandidates {
		candidates = candidates[:*limitCandidates]
	}

	if err := emitCandidatesJSON(os.Stdout, sig.Name(), candidates, result.Stats); err != nil {
		return fmt.Errorf("emit JSON: %w", err)
	}
	fmt.Fprintf(os.Stderr, "find-automation-group-candidates: scanned=%d observations; candidates=%d (signature=%s threshold=%d; actors_aggregated=%d; actors_above_threshold=%d)\n",
		result.Stats.ObservationsScanned,
		len(candidates),
		sig.Name(),
		thresholdOrDefault(sig),
		result.Stats.ActorsAggregated,
		result.Stats.ActorsAboveThreshold)
	return nil
}

// candidateJSON is the operator-facing serialization shape. Distinct
// from signatures.FormationCandidate to keep the package-internal
// representation decoupled from the CLI's stable output contract.
// Per §0152: candidates surfaced for operator review; operator commits
// via existing form-automation-group CLI.
type candidateJSON struct {
	SignatureName     string   `json:"signature_name"`
	HypothesisSubtype string   `json:"hypothesis_subtype"`
	ActorRefs         []string `json:"actor_refs"`
	SourceHashesHex   []string `json:"source_event_hashes_hex"`
	EvidenceCount     uint32   `json:"evidence_count"`
	ConfidenceHint    float64  `json:"confidence_hint"`
}

// statsJSON is the operator-facing serialization of EvaluationStats.
// Distinct from signatures.EvaluationStats to keep package-internal
// representation decoupled from CLI's stable output contract per
// §0153 Methodological observation 4.
type statsJSON struct {
	ObservationsScanned              uint32            `json:"observations_scanned"`
	ObservationsSkippedNoActor       uint32            `json:"observations_skipped_no_actor"`
	ObservationsSkippedWrongModality uint32            `json:"observations_skipped_wrong_modality"`
	ActorsAggregated                 uint32            `json:"actors_aggregated"`
	ActorsAboveThreshold             uint32            `json:"actors_above_threshold"`
	CandidatesEmitted                uint32            `json:"candidates_emitted"`
	PerCollector                     map[string]uint32 `json:"per_collector,omitempty"`
}

type emissionEnvelope struct {
	SignatureName  string          `json:"signature_name"`
	CandidateCount int             `json:"candidate_count"`
	Candidates     []candidateJSON `json:"candidates"`
	Stats          statsJSON       `json:"stats"`
}

func emitCandidatesJSON(w *os.File, signatureName string, candidates []*signatures.FormationCandidate, stats signatures.EvaluationStats) error {
	out := emissionEnvelope{
		SignatureName:  signatureName,
		CandidateCount: len(candidates),
		Candidates:     make([]candidateJSON, 0, len(candidates)),
		Stats: statsJSON{
			ObservationsScanned:              stats.ObservationsScanned,
			ObservationsSkippedNoActor:       stats.ObservationsSkippedNoActor,
			ObservationsSkippedWrongModality: stats.ObservationsSkippedWrongModality,
			ActorsAggregated:                 stats.ActorsAggregated,
			ActorsAboveThreshold:             stats.ActorsAboveThreshold,
			CandidatesEmitted:                stats.CandidatesEmitted,
			PerCollector:                     stats.PerCollector,
		},
	}
	for _, c := range candidates {
		hashesHex := make([]string, len(c.SourceHashes))
		for i, h := range c.SourceHashes {
			hashesHex[i] = hex.EncodeToString(h)
		}
		out.Candidates = append(out.Candidates, candidateJSON{
			SignatureName:     c.SignatureName,
			HypothesisSubtype: subtypeName(c.HypothesisSubtype),
			ActorRefs:         c.ActorRefs,
			SourceHashesHex:   hashesHex,
			EvidenceCount:     c.EvidenceCount,
			ConfidenceHint:    c.ConfidenceHint,
		})
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func subtypeName(s signatures.HypothesisSubtype) string {
	switch s {
	case signatures.HypothesisSubtypeAutomationGroup:
		return "AutomationGroup"
	case signatures.HypothesisSubtypeBehavioralCluster:
		return "BehavioralCluster"
	case signatures.HypothesisSubtypeCampaignHypothesis:
		return "CampaignHypothesis"
	case signatures.HypothesisSubtypeCoordinationRing:
		return "CoordinationRing"
	default:
		return "Unknown"
	}
}

func thresholdOrDefault(sig *signatures.CDPMarkerDensityV1) uint32 {
	if sig.Threshold != 0 {
		return sig.Threshold
	}
	return 2
}
