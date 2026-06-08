// Command form-automation-group-from-candidate is the operator-facing
// bridge CLI between an F3 signature candidate envelope (§0163 stable-
// wire-contract emitted by find-*-candidates CLIs) and committed
// AutomationGroupFormation events. Per decision-log §0213: closes the
// operator-tier gap between the F3 corpus (which emits candidates) and
// the lifecycle corpus (which historically consumed substrate-walking
// patterns like uniform-cadence-v1, NOT F3 candidates).
//
// Pre-§0213 the only path from F3 candidate → AutomationGroupFormation
// was via test-only helpers (commitFormationFromCandidate per §0157 +
// commitBehavioralClusterFormationFromCandidate per §0176). The §0212
// first-real-run shakedown surfaced this gap empirically: 252 candidates
// emitted by tcp_flow_features_clustering_v1 with no operator path to
// materialize them into formations.
//
// Operationally:
//
//   - Input: a §0163 candidate envelope JSON (file via positional arg,
//     stdin fallback). The envelope shape:
//     {signature_name, candidate_count, candidates[], stats{...}}.
//   - Top-N selection: ranks the candidates by a deterministic
//     criterion (default: actor-count descending; tiebreak: candidate
//     content-hash ascending) and commits the top N (default 10).
//   - Per candidate: commits one AutomationGroupFormation event via
//     hypothesis.AutomationGroupFormationFromCandidate. Confidence
//     defaults to 0.0 + EvidentialIndependence defaults to 1/1 per
//     §0213 inception discipline; both operator-overridable.
//   - Output: structured JSON to stdout with per-formation hash +
//     candidate-source-index + counts. Exit codes: 0 success;
//     2 tool / config error; 3 substrate error.
//
// Per §0213 cross-subtype scope: this CLI commits ONLY AutomationGroup
// formations. Candidates carrying HypothesisSubtype != AutomationGroup
// are skipped with a counter (operator can decide whether to invoke a
// different bridge CLI for those — e.g., form-behavioral-cluster-from-
// candidate would parallel this when §0220+ lifts that bridge).
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"time"

	commonv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/common/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/hypothesis"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/signatures"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

const (
	exitToolError       = 2
	exitTargetIntegrity = 3
)

const (
	rankByActorCount = "actor-count"
	rankByPosition   = "position"
	rankByHash       = "hash"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// run is the testable entry point. Returns the process exit code so
// the test harness can assert exit semantics without launching a
// subprocess.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("form-automation-group-from-candidate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "./ghost-trace.db", "SQLite primary-event-log path")
	blobDir := fs.String("blobs", "./blobs", "content-addressed blob-store directory")
	topN := fs.Int("top-n", 10, "commit at most this many formations from the candidate envelope (selected by --rank-by)")
	rankBy := fs.String("rank-by", rankByActorCount, "ranking criterion: actor-count (descending len(ActorRefs); tiebreak content-hash ascending), position (envelope order), hash (content-hash ascending)")
	patternParameters := fs.String("pattern-parameters", "", "AutomationGroupFormation.pattern_parameters value; default empty preserves §0213 inception phase")
	confidenceDefault := fs.Float64("confidence-default", 0.0, "AutomationGroupFormation.confidence value per committed formation; default 0.0 per §0213 inception discipline. §0214 Layer B gating empirical prediction: Layer B may operate over Confidence=0.0 baseline with surprising behavior (all-passing or all-failing depending on gating formula) — override available for §0214 tier-3 experimentation if needed")
	eiNumerator := fs.Int64("ei-numerator", 1, "AutomationGroupFormation.evidential_independence numerator; default 1/1 per §0157 helper precedent + §0140 marshalling-boundary paired-dimension requirement")
	eiDenominator := fs.Int64("ei-denominator", 1, "AutomationGroupFormation.evidential_independence denominator; default 1/1 per §0157 helper precedent")
	formationAtNs := fs.Int64("formation-at-ns", 0, "explicit formation_at as Unix nanoseconds; 0 = wall-clock now()")

	if err := fs.Parse(args); err != nil {
		return exitToolError
	}
	if fs.NArg() > 1 {
		fmt.Fprintf(stderr, "form-automation-group-from-candidate: too many positional arguments (expected 0 or 1, got %d)\n", fs.NArg())
		return exitToolError
	}
	if *topN < 1 {
		fmt.Fprintf(stderr, "form-automation-group-from-candidate: --top-n must be >= 1; got %d\n", *topN)
		return exitToolError
	}
	if *eiDenominator < 1 {
		fmt.Fprintf(stderr, "form-automation-group-from-candidate: --ei-denominator must be >= 1; got %d\n", *eiDenominator)
		return exitToolError
	}

	// Read the candidate envelope: positional arg path or stdin.
	var envelopeReader io.Reader
	if fs.NArg() == 1 {
		f, err := os.Open(fs.Arg(0))
		if err != nil {
			fmt.Fprintf(stderr, "form-automation-group-from-candidate: open envelope: %v\n", err)
			return exitToolError
		}
		defer func() { _ = f.Close() }()
		envelopeReader = f
	} else {
		envelopeReader = stdin
	}

	envelope, err := decodeEnvelope(envelopeReader)
	if err != nil {
		fmt.Fprintf(stderr, "form-automation-group-from-candidate: decode envelope: %v\n", err)
		return exitToolError
	}

	// Convert envelope candidates to signatures.FormationCandidate
	// slice (the API-public structural type the bridge consumes).
	allCandidates, err := envelopeToCandidates(envelope)
	if err != nil {
		fmt.Fprintf(stderr, "form-automation-group-from-candidate: convert envelope: %v\n", err)
		return exitToolError
	}

	// Filter to AutomationGroup-only (the rest are skipped per §0213
	// cross-subtype scope). Counted; surfaced in output JSON.
	agCandidates, crossSubtypeSkipped := filterAutomationGroup(allCandidates)

	// Rank + truncate to top-N.
	ranked, err := rankCandidates(agCandidates, *rankBy)
	if err != nil {
		fmt.Fprintf(stderr, "form-automation-group-from-candidate: rank: %v\n", err)
		return exitToolError
	}
	selected := ranked
	if len(selected) > *topN {
		selected = selected[:*topN]
	}

	// Open substrate + commit each selected candidate.
	ctx := context.Background()
	sub, err := substrate.Open(ctx, *dbPath, *blobDir)
	if err != nil {
		fmt.Fprintf(stderr, "form-automation-group-from-candidate: open substrate: %v\n", err)
		return exitToolError
	}
	defer func() { _ = sub.Close() }()

	opts := hypothesis.AutomationGroupFormationFromCandidateOptions{
		PatternParameters: *patternParameters,
		FormationAt:       *formationAtNs,
		Confidence:        float32(*confidenceDefault),
		EvidentialIndependence: &commonv1.EvidentialIndependence{
			Numerator:   uint64(*eiNumerator),
			Denominator: uint64(*eiDenominator),
		},
	}

	out := outputPayload{
		RankingAlgorithm:      *rankBy,
		TopN:                  *topN,
		CandidatesIngested:    len(allCandidates),
		CandidatesAGEligible:  len(agCandidates),
		CrossSubtypeSkipped:   crossSubtypeSkipped,
		FormationsCommitted:   make([]formationRecord, 0, len(selected)),
		AlreadyPresentCount:   0,
	}

	for _, rc := range selected {
		hash, alreadyPresent, err := hypothesis.AutomationGroupFormationFromCandidate(ctx, sub, rc.candidate, opts, time.Now)
		if err != nil {
			fmt.Fprintf(stderr, "form-automation-group-from-candidate: commit candidate %d: %v\n", rc.envelopeIndex, err)
			return exitTargetIntegrity
		}
		out.FormationsCommitted = append(out.FormationsCommitted, formationRecord{
			CandidateEnvelopeIndex: rc.envelopeIndex,
			FormationEventHashHex:  hex.EncodeToString(hash[:]),
			ActorRefsCount:         len(rc.candidate.ActorRefs),
			SourceHashesCount:      len(rc.candidate.SourceHashes),
			AlreadyPresent:         alreadyPresent,
		})
		if alreadyPresent {
			out.AlreadyPresentCount++
		}
	}

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fmt.Fprintf(stderr, "form-automation-group-from-candidate: encode json: %v\n", err)
		return exitToolError
	}

	fmt.Fprintf(stderr,
		"form-automation-group-from-candidate: rank_by=%s top_n=%d ingested=%d ag_eligible=%d cross_subtype_skipped=%d committed=%d already_present=%d\n",
		*rankBy, *topN, len(allCandidates), len(agCandidates), crossSubtypeSkipped, len(out.FormationsCommitted), out.AlreadyPresentCount)
	return 0
}

// envelopeCandidate is the §0163 envelope's per-candidate shape as
// emitted by find-*-candidates CLIs. Field names match the JSON
// field names used in the production signatures package's candidate
// emission verbatim.
//
// The struct is local to this CLI (decode boundary) because the
// production type signatures.FormationCandidate uses [][]byte for
// SourceHashes (raw bytes); the signature CLIs emit hex-encoded
// strings under the wire field name `source_event_hashes_hex` (all
// 5 find-* CLIs share this convention; verified at decision-log
// §0214 Finding 1 cross-CLI cravamento). The conversion is in
// envelopeToCandidates below.
//
// Per §0215 + §0214 MO1 (wire-contract integration testing
// pattern): the JSON field name MUST match the upstream CLI's emit
// form verbatim. Pre-§0215 this struct used `json:"source_hashes"`
// which silently mis-decoded the upstream `source_event_hashes_hex`
// field as null → 10 broken-chain formations committed at §0213
// first-real-run (RUN_ID 20260608T045037Z). §0215 corrects the
// field name; the §0214 MO1 real-wire fixture testing discipline
// prevents recurrence at the test-coverage tier.
type envelopeCandidate struct {
	SignatureName     string     `json:"signature_name"`
	HypothesisSubtype string     `json:"hypothesis_subtype"`
	ActorRefs         []string   `json:"actor_refs"`
	SourceHashes      []string   `json:"source_event_hashes_hex"`
	EvidenceCount     uint32     `json:"evidence_count"`
	ConfidenceHint    float64    `json:"confidence_hint"`
	Interactions      [][]string `json:"interactions"`
}

// envelope is the §0163 envelope shape emitted by find-*-candidates
// CLIs to stdout. Only the candidates slice is required by this
// bridge; signature_name / candidate_count / stats are preserved as
// auditable surface but not consumed in selection.
type envelope struct {
	SignatureName  string              `json:"signature_name"`
	CandidateCount int                 `json:"candidate_count"`
	Candidates     []envelopeCandidate `json:"candidates"`
}

func decodeEnvelope(r io.Reader) (envelope, error) {
	// Read all bytes first so a re-decode on validation failure is
	// possible without re-reading from stdin (which is non-seekable).
	bs, err := io.ReadAll(r)
	if err != nil {
		return envelope{}, fmt.Errorf("read envelope: %w", err)
	}
	var env envelope
	if err := json.NewDecoder(bytes.NewReader(bs)).Decode(&env); err != nil {
		return envelope{}, fmt.Errorf("json decode envelope: %w", err)
	}
	return env, nil
}

func envelopeToCandidates(env envelope) ([]*signatures.FormationCandidate, error) {
	out := make([]*signatures.FormationCandidate, 0, len(env.Candidates))
	for i, ec := range env.Candidates {
		subtype, err := parseSubtype(ec.HypothesisSubtype)
		if err != nil {
			return nil, fmt.Errorf("candidate %d: %w", i, err)
		}
		hashes := make([][]byte, 0, len(ec.SourceHashes))
		for j, hh := range ec.SourceHashes {
			raw, err := hex.DecodeString(hh)
			if err != nil {
				return nil, fmt.Errorf("candidate %d source-hash %d: hex decode %q: %w", i, j, hh, err)
			}
			if len(raw) != 32 {
				return nil, fmt.Errorf("candidate %d source-hash %d: got %d bytes want 32 per §0139", i, j, len(raw))
			}
			hashes = append(hashes, raw)
		}
		interactions := make([][2]string, 0, len(ec.Interactions))
		for j, edge := range ec.Interactions {
			if len(edge) != 2 {
				return nil, fmt.Errorf("candidate %d interaction %d: got %d endpoints want 2", i, j, len(edge))
			}
			interactions = append(interactions, [2]string{edge[0], edge[1]})
		}
		out = append(out, &signatures.FormationCandidate{
			SignatureName:     ec.SignatureName,
			HypothesisSubtype: subtype,
			ActorRefs:         ec.ActorRefs,
			SourceHashes:      hashes,
			EvidenceCount:     ec.EvidenceCount,
			ConfidenceHint:    ec.ConfidenceHint,
			Interactions:      interactions,
		})
	}
	return out, nil
}

// parseSubtype converts the JSON-serialized HypothesisSubtype string
// (per the signatures package's stringer output) back into the typed
// enum. The §0163 envelope emits the stringer form (e.g.
// "AutomationGroup"); the bridge reverses for typed comparison.
func parseSubtype(s string) (signatures.HypothesisSubtype, error) {
	switch s {
	case "AutomationGroup":
		return signatures.HypothesisSubtypeAutomationGroup, nil
	case "BehavioralCluster":
		return signatures.HypothesisSubtypeBehavioralCluster, nil
	case "CampaignHypothesis":
		return signatures.HypothesisSubtypeCampaignHypothesis, nil
	case "CoordinationRing":
		return signatures.HypothesisSubtypeCoordinationRing, nil
	case "", "Unknown":
		return signatures.HypothesisSubtypeUnknown, nil
	default:
		return signatures.HypothesisSubtypeUnknown, fmt.Errorf("unknown HypothesisSubtype %q", s)
	}
}

// filterAutomationGroup splits the candidate slice into AutomationGroup-
// eligible candidates + a count of skipped cross-subtype candidates.
// Per §0213 cross-subtype scope.
func filterAutomationGroup(in []*signatures.FormationCandidate) ([]*signatures.FormationCandidate, int) {
	ag := make([]*signatures.FormationCandidate, 0, len(in))
	skipped := 0
	for _, c := range in {
		if c.HypothesisSubtype == signatures.HypothesisSubtypeAutomationGroup {
			ag = append(ag, c)
		} else {
			skipped++
		}
	}
	return ag, skipped
}

// rankedCandidate pairs a candidate with its envelope-source index
// for audit-grade output (operator can trace each formation back to
// the candidate's position in the input envelope).
type rankedCandidate struct {
	candidate     *signatures.FormationCandidate
	envelopeIndex int
}

// rankCandidates produces a deterministic ordering per the requested
// criterion. Returns a new slice; does not mutate the input.
func rankCandidates(in []*signatures.FormationCandidate, criterion string) ([]rankedCandidate, error) {
	rcs := make([]rankedCandidate, len(in))
	for i, c := range in {
		rcs[i] = rankedCandidate{candidate: c, envelopeIndex: i}
	}
	switch criterion {
	case rankByActorCount:
		// Primary: len(ActorRefs) descending. Tiebreak:
		// candidate-content-hash ascending. The §0213 default per
		// the user's cravamento; deterministic + interpretable.
		// Sub-cuidado §0219+: if HypothesisSubtype semantic
		// resolves to "actors_above_threshold = membership-count
		// not distinct-actors" (Finding 6 hypothesis (a)), the
		// len(ActorRefs) here may proxy memberships not distinct
		// actors; default may shift to "hash" for semantic
		// neutrality. Operator-overridable today via --rank-by.
		sort.SliceStable(rcs, func(i, j int) bool {
			li, lj := len(rcs[i].candidate.ActorRefs), len(rcs[j].candidate.ActorRefs)
			if li != lj {
				return li > lj
			}
			return bytes.Compare(candidateContentHash(rcs[i].candidate), candidateContentHash(rcs[j].candidate)) < 0
		})
		return rcs, nil
	case rankByPosition:
		// No-op: envelope order preserved.
		return rcs, nil
	case rankByHash:
		// Candidate-content-hash ascending. Semantic-neutral
		// deterministic ordering; useful when actor-count
		// semantics is ambiguous (see §0219+ above).
		sort.SliceStable(rcs, func(i, j int) bool {
			return bytes.Compare(candidateContentHash(rcs[i].candidate), candidateContentHash(rcs[j].candidate)) < 0
		})
		return rcs, nil
	default:
		return nil, fmt.Errorf("unknown --rank-by %q; known: %s | %s | %s", criterion, rankByActorCount, rankByPosition, rankByHash)
	}
}

// candidateContentHash produces a deterministic hash of the
// candidate's identity-bearing fields for tiebreak ordering. NOT
// the same as the committed AutomationGroupFormation content-hash
// (which depends on PatternParameters + FormationAt + EI). The
// tiebreak hash uses only fields present in the candidate envelope
// to keep ranking stable regardless of formation-time options.
func candidateContentHash(c *signatures.FormationCandidate) []byte {
	h := sha256.New()
	_, _ = io.WriteString(h, c.SignatureName)
	_, _ = io.WriteString(h, "|")
	for _, ar := range c.ActorRefs {
		_, _ = io.WriteString(h, ar)
		_, _ = io.WriteString(h, ",")
	}
	_, _ = io.WriteString(h, "|")
	for _, sh := range c.SourceHashes {
		_, _ = h.Write(sh)
	}
	return h.Sum(nil)
}

type outputPayload struct {
	RankingAlgorithm     string            `json:"ranking_algorithm"`
	TopN                 int               `json:"top_n"`
	CandidatesIngested   int               `json:"candidates_ingested"`
	CandidatesAGEligible int               `json:"candidates_automation_group_eligible"`
	CrossSubtypeSkipped  int               `json:"cross_subtype_skipped"`
	FormationsCommitted  []formationRecord `json:"formations_committed"`
	AlreadyPresentCount  int               `json:"already_present_count"`
}

type formationRecord struct {
	CandidateEnvelopeIndex int    `json:"candidate_envelope_index"`
	FormationEventHashHex  string `json:"formation_event_hash"`
	ActorRefsCount         int    `json:"actor_refs_count"`
	SourceHashesCount      int    `json:"source_hashes_count"`
	AlreadyPresent         bool   `json:"already_present"`
}
