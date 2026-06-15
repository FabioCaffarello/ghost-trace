// Command find-automation-group-candidates-tls is the F3 orchestrator
// for the TLS-fingerprint modality per decision-log §0221: reads
// NetworkObservation records from the substrate, invokes the
// tls_ja4_automation_v1 signature against an operator-supplied set of
// known-automation JA4/JA3 fingerprints, and emits FormationCandidates
// to stdout as structured JSON for operator review. This is the
// INFERENCE layer of the §0221 TLS vertical slice. Parallel structure
// to find-automation-group-candidates-network (§0163).
//
// Per §3 N3 + §0152: this command does NOT commit formation events
// directly. Operator reviews candidates + commits each via the existing
// form-automation-group-from-candidate CLI.
//
// The known-automation reference set is operator-supplied (see the
// tls_ja4_automation_v1 constitutional note): -known-ja4 / -known-ja3
// (comma-separated) and/or -known-ja4-file / -known-ja3-file (one
// fingerprint per line). With no reference set supplied, the signature
// emits zero candidates (it does not assert which fingerprints are
// automation).
//
// Read-only over the substrate; no AppendPair; safe concurrent with
// ingestion paths.
//
// Exit codes: 0 success (including empty candidate set); 2 tool/config
// error (invalid flags, substrate open failure).
package main

import (
	"bufio"
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/attribution"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/observationcollector"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/signatures"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

const exitToolError = 2

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "find-automation-group-candidates-tls: %v\n", err)
		os.Exit(exitToolError)
	}
}

func run() error {
	dbPath := flag.String("db", "./ghost-trace.db", "SQLite primary-event-log path")
	blobDir := flag.String("blobs", "./blobs", "content-addressed blob-store directory")
	threshold := flag.Uint("threshold", 0, "signature threshold override (0 = signature default = 1)")
	limitCandidates := flag.Int("limit", 0, "maximum candidates to emit (0 = unlimited)")
	knownJA4 := flag.String("known-ja4", "", "comma-separated JA4 fingerprints treated as automation")
	knownJA3 := flag.String("known-ja3", "", "comma-separated JA3 fingerprints treated as automation")
	knownJA4File := flag.String("known-ja4-file", "", "path to a file of JA4 fingerprints (one per line) treated as automation")
	knownJA3File := flag.String("known-ja3-file", "", "path to a file of JA3 fingerprints (one per line) treated as automation")
	useAttribution := flag.Bool("with-attribution", false, "consult Cat II DerivedActorAttribution records (§0168) to fill empty actor_ref")
	flag.Parse()

	ja4Set, err := buildFingerprintSet(*knownJA4, *knownJA4File)
	if err != nil {
		return fmt.Errorf("build JA4 set: %w", err)
	}
	ja3Set, err := buildFingerprintSet(*knownJA3, *knownJA3File)
	if err != nil {
		return fmt.Errorf("build JA3 set: %w", err)
	}

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

	sig := &signatures.TLSJa4AutomationV1{
		KnownJA4:  ja4Set,
		KnownJA3:  ja3Set,
		Threshold: uint32(*threshold),
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
	fmt.Fprintf(os.Stderr, "find-automation-group-candidates-tls: scanned=%d observations; candidates=%d (signature=%s threshold=%d known_ja4=%d known_ja3=%d with_attribution=%v; actors_aggregated=%d; actors_above_threshold=%d; skipped_no_actor=%d; skipped_wrong_modality=%d)\n",
		result.Stats.ObservationsScanned,
		len(candidates),
		sig.Name(),
		sig.EffectiveThreshold(),
		len(ja4Set),
		len(ja3Set),
		*useAttribution,
		result.Stats.ActorsAggregated,
		result.Stats.ActorsAboveThreshold,
		result.Stats.ObservationsSkippedNoActor,
		result.Stats.ObservationsSkippedWrongModality)
	return nil
}

// buildFingerprintSet merges a comma-separated inline list and an
// optional one-per-line file into a set. Returns nil when both are
// empty (signals "no reference set" to the signature, which then
// emits zero candidates).
func buildFingerprintSet(inline, filePath string) (map[string]struct{}, error) {
	set := make(map[string]struct{})
	for _, v := range strings.Split(inline, ",") {
		if v = strings.TrimSpace(v); v != "" {
			set[v] = struct{}{}
		}
	}
	if filePath != "" {
		f, err := os.Open(filePath)
		if err != nil {
			return nil, err
		}
		defer func() { _ = f.Close() }()
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			if v := strings.TrimSpace(sc.Text()); v != "" {
				set[v] = struct{}{}
			}
		}
		if err := sc.Err(); err != nil {
			return nil, err
		}
	}
	if len(set) == 0 {
		return nil, nil
	}
	return set, nil
}

type candidateJSON struct {
	SignatureName     string   `json:"signature_name"`
	HypothesisSubtype string   `json:"hypothesis_subtype"`
	ActorRefs         []string `json:"actor_refs"`
	SourceHashesHex   []string `json:"source_event_hashes_hex"`
	EvidenceCount     uint32   `json:"evidence_count"`
	ConfidenceHint    float64  `json:"confidence_hint"`
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
