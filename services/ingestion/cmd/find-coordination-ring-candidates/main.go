// Command find-coordination-ring-candidates is the F3 orchestrator
// for the CoordinationRing subtype per decision-log §0186: reads
// NetworkObservation records from the substrate, invokes the
// endpoint_co_visit_v1 signature (§0185), and emits FormationCandidates
// to stdout as structured JSON for operator review.
//
// Mirrors find-automation-group-candidates / find-automation-group-
// candidates-network / find-behavioral-cluster-candidates / find-
// campaign-hypothesis-candidates on the CoordinationRing emission side
// (the FOURTH and FINAL Cat III subtype F3-loop CLI; closes the F3
// orchestrator CLI corpus to 4-of-4 subtypes).
//
// Per §0070 + §0185 CoordinationRing interaction-centric ontology: the
// emitted candidate carries Interactions as the structural-edge field
// alongside the ActorRefs vertex set. Operator's downstream form-
// coordination-ring CLI commits the formation event WITH interactions
// converted into CoordinationRingInteraction protos (actor_a=edge[0],
// actor_b=edge[1] per §0070 within-edge lex canonicalization).
//
// Wire-contract extension per §0163 + §0186 MO1: candidateJSON gains
// an optional Interactions field (omitempty) — interaction-centric
// subtypes populate it; other CLIs' candidateJSON definitions leave
// it absent. Common-field discipline preserved.
//
// Per §3 N3 + §0152: this command does NOT commit formation events
// directly. Operator reviews candidates + decides whether to commit.
//
// Read-only over substrate; safe to invoke concurrently with
// ingestion paths.
//
// Exit codes: 0 success (including empty candidate set); 2
// tool/config error.
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
		fmt.Fprintf(os.Stderr, "find-coordination-ring-candidates: %v\n", err)
		os.Exit(exitToolError)
	}
}

func run() error {
	dbPath := flag.String("db", "./ghost-trace.db", "SQLite primary-event-log path")
	blobDir := flag.String("blobs", "./blobs", "content-addressed blob-store directory")
	threshold := flag.Uint("threshold", 0, "endpoint_co_visit_v1 threshold override (0 = signature default = 3 distinct actors per cluster)")
	bucketSeconds := flag.Uint("bucket-seconds", 0, "endpoint_co_visit_v1 time-bucket width in seconds (0 = signature default = 60s)")
	limitCandidates := flag.Int("limit", 0, "maximum candidates to emit (0 = unlimited)")
	useAttribution := flag.Bool("with-attribution", false, "consult Cat II DerivedActorAttribution records (§0168) for actor enrichment on observations with empty actor_ref")
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

	sig := &signatures.EndpointCoVisitV1{
		Threshold:     uint32(*threshold),
		BucketSeconds: uint64(*bucketSeconds),
	}

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
	fmt.Fprintf(os.Stderr, "find-coordination-ring-candidates: scanned=%d observations; candidates=%d (signature=%s threshold=%d bucket_seconds=%d with_attribution=%v; actors_aggregated=%d; actors_above_threshold=%d; skipped_no_actor=%d; skipped_wrong_modality=%d)\n",
		result.Stats.ObservationsScanned,
		len(candidates),
		sig.Name(),
		thresholdOrDefault(sig),
		bucketSecondsOrDefault(sig),
		*useAttribution,
		result.Stats.ActorsAggregated,
		result.Stats.ActorsAboveThreshold,
		result.Stats.ObservationsSkippedNoActor,
		result.Stats.ObservationsSkippedWrongModality)
	return nil
}

// candidateJSON / statsJSON / emissionEnvelope mirror the shared
// wire-contract per §0163 stable-wire-contract discipline across all
// 5 F3-loop orchestrator CLIs. Per §0186 MO1: candidateJSON gains
// optional Interactions field (omitempty) for interaction-centric
// subtypes; other CLIs' candidateJSON definitions leave it absent.
type candidateJSON struct {
	SignatureName     string      `json:"signature_name"`
	HypothesisSubtype string      `json:"hypothesis_subtype"`
	ActorRefs         []string    `json:"actor_refs"`
	SourceHashesHex   []string    `json:"source_event_hashes_hex"`
	EvidenceCount     uint32      `json:"evidence_count"`
	ConfidenceHint    float64     `json:"confidence_hint"`
	Interactions      [][2]string `json:"interactions,omitempty"`
}

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
			Interactions:      c.Interactions,
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

func thresholdOrDefault(sig *signatures.EndpointCoVisitV1) uint32 {
	if sig.Threshold != 0 {
		return sig.Threshold
	}
	return 3
}

func bucketSecondsOrDefault(sig *signatures.EndpointCoVisitV1) uint64 {
	if sig.BucketSeconds != 0 {
		return sig.BucketSeconds
	}
	return 60
}
