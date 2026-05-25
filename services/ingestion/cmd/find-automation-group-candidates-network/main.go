// Command find-automation-group-candidates-network is the F3
// orchestrator for the network-modality side per decision-log §0163:
// reads NetworkObservation records from the substrate, invokes the
// tcp_fingerprint_clustering_v1 signature (§0161), and emits
// FormationCandidates to stdout as structured JSON for operator
// review. Parallel structure to find-automation-group-candidates
// (browser-modality) per §0153 + §0161; together the two CLIs span
// the F3 corpus across both modalities defined by §0144 + §0151.
//
// Per §3 N3 + §0152: this command does NOT commit formation events
// directly. Operator reviews candidates + decides whether to commit
// each via the existing form-automation-group CLI (paired with the
// candidate's actor_ref + source-hash list).
//
// Per §0162 empirical finding: against CIC-IDS substrate, this CLI
// will surface 0 candidates + non-zero ObservationsSkippedNoActor
// reflecting CIC-IDS's flow-level (non-actor-attributed) shape.
// The diagnostic surface (skip counters) makes the §0144(e)
// phenomenon-vs-record gap operator-visible without inspection of
// substrate records by hand.
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

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/attribution"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/observationcollector"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/signatures"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

const exitToolError = 2

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "find-automation-group-candidates-network: %v\n", err)
		os.Exit(exitToolError)
	}
}

func run() error {
	dbPath := flag.String("db", "./ghost-trace.db", "SQLite primary-event-log path")
	blobDir := flag.String("blobs", "./blobs", "content-addressed blob-store directory")
	threshold := flag.Uint("threshold", 0, "signature threshold override (0 = signature default = 3)")
	limitCandidates := flag.Int("limit", 0, "maximum candidates to emit (0 = unlimited)")
	signatureName := flag.String("signature", "p0f", "signature to invoke: 'p0f' (tcp_fingerprint_clustering_v1) or 'flow-features' (tcp_flow_features_clustering_v1; closes §0162 gap (2) for CICFlowMeter-style adapters per §0169)")
	useAttribution := flag.Bool("with-attribution", false, "consult Cat II DerivedActorAttribution records (§0168) to fill empty actor_ref on Cat I observations; off by default for backward-compat")
	flag.Parse()

	ctx := context.Background()
	sub, err := substrate.Open(ctx, *dbPath, *blobDir)
	if err != nil {
		return fmt.Errorf("substrate.Open: %w", err)
	}
	defer sub.Close()

	observations, err := observationcollector.CollectNetwork(ctx, sub)
	if err != nil {
		return fmt.Errorf("collect: %w", err)
	}

	sig, sigThreshold, err := selectNetworkSignature(*signatureName, uint32(*threshold))
	if err != nil {
		return err
	}

	// AttributionView: nil unless explicitly requested. Per §0168
	// Decision A.1: signature consumption of Cat II derived
	// attribution is OPT-IN at orchestrator dispatch — operators
	// who have run attribution.DeriveAll on the substrate pass
	// -with-attribution to consume the resulting Cat II records.
	var attributionView signatures.AttributionLookup
	if *useAttribution {
		v, err := attribution.CollectAttributionView(ctx, sub)
		if err != nil {
			return fmt.Errorf("attribution.CollectAttributionView: %w", err)
		}
		attributionView = v
	}

	result, err := sig.EvaluateNetwork(ctx, observations, attributionView)
	if err != nil {
		return fmt.Errorf("signature.EvaluateNetwork: %w", err)
	}

	candidates := result.Candidates
	if *limitCandidates > 0 && len(candidates) > *limitCandidates {
		candidates = candidates[:*limitCandidates]
	}

	if err := emitCandidatesJSON(os.Stdout, sig.Name(), candidates, result.Stats); err != nil {
		return fmt.Errorf("emit JSON: %w", err)
	}
	fmt.Fprintf(os.Stderr, "find-automation-group-candidates-network: scanned=%d observations; candidates=%d (signature=%s threshold=%d with_attribution=%v; actors_aggregated=%d; actors_above_threshold=%d; skipped_no_actor=%d; skipped_wrong_modality=%d)\n",
		result.Stats.ObservationsScanned,
		len(candidates),
		sig.Name(),
		sigThreshold,
		*useAttribution,
		result.Stats.ActorsAggregated,
		result.Stats.ActorsAboveThreshold,
		result.Stats.ObservationsSkippedNoActor,
		result.Stats.ObservationsSkippedWrongModality)
	return nil
}

// selectNetworkSignature constructs the configured NetworkSignature
// instance and returns the effective threshold (signature's default
// when the override is 0). Per §0170: explicit name → signature
// mapping kept simple at inception; future signature additions extend
// the switch.
func selectNetworkSignature(name string, thresholdOverride uint32) (signatures.NetworkSignature, uint32, error) {
	switch name {
	case "p0f":
		s := &signatures.TCPFingerprintClusteringV1{Threshold: thresholdOverride}
		return s, thresholdOrDefaultP0F(s), nil
	case "flow-features":
		s := &signatures.TCPFlowFeaturesClusteringV1{Threshold: thresholdOverride}
		return s, thresholdOrDefaultFlowFeatures(s), nil
	default:
		return nil, 0, fmt.Errorf("unknown -signature value %q (valid: p0f, flow-features)", name)
	}
}

// candidateJSON is the operator-facing serialization shape. Distinct
// from signatures.FormationCandidate to keep the package-internal
// representation decoupled from the CLI's stable output contract.
// Shape matches find-automation-group-candidates' candidateJSON for
// operator-tooling reuse across the two F3-modality CLIs.
type candidateJSON struct {
	SignatureName     string   `json:"signature_name"`
	HypothesisSubtype string   `json:"hypothesis_subtype"`
	ActorRefs         []string `json:"actor_refs"`
	SourceHashesHex   []string `json:"source_event_hashes_hex"`
	EvidenceCount     uint32   `json:"evidence_count"`
	ConfidenceHint    float64  `json:"confidence_hint"`
}

// statsJSON is the operator-facing serialization of EvaluationStats.
// Shape matches find-automation-group-candidates' statsJSON for
// symmetric operator-tooling output across modalities.
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

func thresholdOrDefaultP0F(sig *signatures.TCPFingerprintClusteringV1) uint32 {
	if sig.Threshold != 0 {
		return sig.Threshold
	}
	return 3
}

func thresholdOrDefaultFlowFeatures(sig *signatures.TCPFlowFeaturesClusteringV1) uint32 {
	if sig.Threshold != 0 {
		return sig.Threshold
	}
	return 3
}
