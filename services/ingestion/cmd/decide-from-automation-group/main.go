// Command decide-from-automation-group is the operator-elected
// enforcement-decision CLI per decision-log §0222 (Framing A). Given an
// AutomationGroupFormation (the inference "this actor is automated"), it
// evaluates a versioned policy and commits an OperationalDecisionAudit
// (Cat I) recording the verdict (ALLOW / CHALLENGE / BLOCK / SHADOW).
// This is the DECISION layer of the anti-bot vertical slice; it
// completes obs→inference→decision in the substrate.
//
// Per §3 N3: the substrate is NOT the actor. This CLI is the operator-
// elected path; -operator names the elector and is recorded in the
// audit as the structural evidence that the decision was operator-
// initiated, not autonomous. Signatures and inference paths never
// invoke this.
//
// Exit codes: 0 success; 2 tool/config error; 3 substrate-precondition
// failure (formation not found / wrong type / multi-actor).
package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/decision"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

const (
	exitToolError       = 2
	exitTargetIntegrity = 3
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "decide-from-automation-group: %v\n", err)
		// Substrate-precondition failures (not-found / wrong-type /
		// multi-actor) are operator-correctable; distinguish them.
		msg := err.Error()
		if strings.Contains(msg, "not found") || strings.Contains(msg, "want AutomationGroupFormation") || strings.Contains(msg, "single-actor") {
			os.Exit(exitTargetIntegrity)
		}
		os.Exit(exitToolError)
	}
}

func run() error {
	dbPath := flag.String("db", "./ghost-trace.db", "SQLite primary-event-log path")
	blobDir := flag.String("blobs", "./blobs", "content-addressed blob-store directory")
	formationHashHex := flag.String("formation-event-hash", "", "REQUIRED: hex BLAKE3-256 of the AutomationGroupFormation to decide on")
	policyRef := flag.String("policy", "", "versioned policy_ref (empty = automation-tiered-v1)")
	operatorRef := flag.String("operator", "", "operator identity electing the decision (§3 N3 operator-initiated evidence)")
	decidedAt := flag.Int64("decided-at", 0, "decision time as Unix nanoseconds (0 = now)")
	flag.Parse()

	if *formationHashHex == "" {
		return fmt.Errorf("--formation-event-hash is required")
	}
	raw, err := hex.DecodeString(*formationHashHex)
	if err != nil {
		return fmt.Errorf("decode --formation-event-hash: %w", err)
	}
	if len(raw) != 32 {
		return fmt.Errorf("--formation-event-hash must be 32 bytes (64 hex chars); got %d bytes", len(raw))
	}
	if *operatorRef == "" {
		return fmt.Errorf("--operator is required (§3 N3: a decision audit must name the operator that elected it)")
	}
	var formationHash [32]byte
	copy(formationHash[:], raw)

	ctx := context.Background()
	sub, err := substrate.Open(ctx, *dbPath, *blobDir)
	if err != nil {
		return fmt.Errorf("substrate.Open: %w", err)
	}
	defer sub.Close()

	report, err := decision.DecideFromAutomationGroup(ctx, sub, decision.DecideOptions{
		FormationEventHash: formationHash,
		PolicyRef:          *policyRef,
		OperatorRef:        *operatorRef,
		DecidedAt:          *decidedAt,
	}, time.Now)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	fmt.Fprintf(os.Stderr, "decide-from-automation-group: verdict=%s subject=%s policy=%s audit=%s already_present=%v\n",
		report.Verdict, report.SubjectActorRef, report.PolicyRef, report.AuditEventHashHex, report.AlreadyPresent)
	return nil
}
