// Command derive-actor-attribution derives Category II
// DerivedActorAttribution records over every NetworkObservation in
// the substrate per decision-log §0209 (operational closure of
// §0162 Gap (1), §0168 Cat II construct). Wraps the
// internal/attribution package's DeriveAll entry point with
// command-line configuration + structured output.
//
// Mirrors cmd/derive-operational-session (the precedent Cat II
// derivation CLI per §0043): same flag surface convention, same
// exit-code semantic, same JSON output shape.
//
// Operationally:
//
//   - Run after ingest when operators want to materialize per-
//     observation actor attribution under the chosen attribution
//     definition.
//   - Re-run is idempotent under the same (definition_version,
//     definition_parameters) tuple: zero new substrate rows produced
//     on the second pass; existing records are counted in
//     already_derived.
//   - Required as a pipeline step BEFORE F3 signatures that consume
//     Cat II AttributionView (§0168 Decision A.1: explicit
//     parameter, not silent back-fill).
//
// Output: structured JSON to stdout + brief human summary to stderr.
// Exit code: 0 on success (including zero newly_derived); 2 on tool
// or configuration error (unknown definition-version, substrate open
// failure, etc.).
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/attribution"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

const exitToolError = 2

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run is the testable entry point. Returns the process exit code so
// the test harness can assert exit semantics without launching a
// subprocess.
func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("derive-actor-attribution", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "./ghost-trace.db", "SQLite primary-event-log path")
	blobDir := fs.String("blobs", "./blobs", "content-addressed blob-store directory")
	definitionVersion := fs.String("definition-version", attribution.Network5TupleActorV1Version, "attribution-definition version identifier")

	if err := fs.Parse(args); err != nil {
		return exitToolError
	}

	def, err := resolveDefinition(*definitionVersion)
	if err != nil {
		fmt.Fprintf(stderr, "derive-actor-attribution: %v\n", err)
		return exitToolError
	}

	ctx := context.Background()
	sub, err := substrate.Open(ctx, *dbPath, *blobDir)
	if err != nil {
		fmt.Fprintf(stderr, "derive-actor-attribution: open substrate: %v\n", err)
		return exitToolError
	}
	defer func() { _ = sub.Close() }()

	report, err := attribution.DeriveAll(ctx, sub, def, time.Now)
	if err != nil {
		fmt.Fprintf(stderr, "derive-actor-attribution: derive: %v\n", err)
		return exitToolError
	}

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(payload{
		DefinitionVersion:    def.Version(),
		DefinitionParameters: def.Parameters(),
		Examined:             report.Examined,
		Skipped:              report.Skipped,
		NewlyDerived:         report.NewlyDerived,
		AlreadyDerived:       report.AlreadyDerived,
	}); err != nil {
		fmt.Fprintf(stderr, "derive-actor-attribution: encode json: %v\n", err)
		return exitToolError
	}

	fmt.Fprintf(stderr,
		"derive-actor-attribution: definition=%s params=%q examined=%d skipped=%d newly_derived=%d already_derived=%d\n",
		def.Version(), def.Parameters(), report.Examined, report.Skipped, report.NewlyDerived, report.AlreadyDerived)
	return 0
}

// resolveDefinition selects an AttributionDefinition by version
// identifier. New attribution definitions register here per the
// precedent established by cmd/derive-operational-session
// resolveDefinition. v1 has no operator-supplied parameters per
// §0168 inception-phase scope (Network5TupleActorV1.Parameters()
// returns ""), so no parameter-binding step is needed.
func resolveDefinition(version string) (attribution.AttributionDefinition, error) {
	switch version {
	case attribution.Network5TupleActorV1Version:
		return attribution.Network5TupleActorV1{}, nil
	default:
		return nil, fmt.Errorf("unknown definition-version %q; known versions: %s",
			version, attribution.Network5TupleActorV1Version)
	}
}

type payload struct {
	DefinitionVersion    string `json:"definition_version"`
	DefinitionParameters string `json:"definition_parameters"`
	Examined             int64  `json:"examined"`
	Skipped              int64  `json:"skipped"`
	NewlyDerived         int64  `json:"newly_derived"`
	AlreadyDerived       int64  `json:"already_derived"`
}
