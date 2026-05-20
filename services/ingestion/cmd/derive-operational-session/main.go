// Command derive-operational-session derives Category II
// OperationalSession constructs over every DeclaredSession in the
// substrate per decision-log §0043 (first Cat II construct landing).
// Wraps the internal/derivation package's DeriveAll entry point with
// command-line configuration + structured output.
//
// Operationally:
//
//   - Run after an ingestion window when operators want to materialize
//     the current operational-definition's view of session boundaries.
//   - Re-run is idempotent under the same (definition_version,
//     definition_parameters) tuple: zero new substrate rows produced.
//     A change to either tuple element produces a new derivation
//     alongside the prior records per entity-model.md line 45.
//
// Output: structured JSON to stdout + a brief human summary to stderr.
// Exit code: 0 on success (including zero-newly-derived); 2 on tool /
// configuration error (e.g. database open failure).
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/derivation"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "derive-operational-session: %v\n", err)
		os.Exit(2)
	}
}

func run() error {
	dbPath := flag.String("db", "./ghost-trace.db", "SQLite primary-event-log path")
	blobDir := flag.String("blobs", "./blobs", "content-addressed blob-store directory")
	definitionVersion := flag.String("definition-version", derivation.PaddedV1Version, "operational-definition version identifier")
	padSeconds := flag.Int64("pad-seconds", 300, "padded-v1 boundary-padding parameter (seconds added to declared_at to derive operational_end_at)")
	flag.Parse()

	def, err := resolveDefinition(*definitionVersion, *padSeconds)
	if err != nil {
		return err
	}

	ctx := context.Background()
	sub, err := substrate.Open(ctx, *dbPath, *blobDir)
	if err != nil {
		return fmt.Errorf("open substrate: %w", err)
	}
	defer func() { _ = sub.Close() }()

	report, err := derivation.DeriveAll(ctx, sub, def, time.Now)
	if err != nil {
		return fmt.Errorf("derive: %w", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(payload{
		DefinitionVersion:    def.Version(),
		DefinitionParameters: def.Parameters(),
		Examined:             report.Examined,
		NewlyDerived:         report.NewlyDerived,
		AlreadyDerived:       report.AlreadyDerived,
	}); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}

	fmt.Fprintf(os.Stderr,
		"derive-operational-session: definition=%s params=%q examined=%d newly_derived=%d already_derived=%d\n",
		def.Version(), def.Parameters(), report.Examined, report.NewlyDerived, report.AlreadyDerived)
	return nil
}

// resolveDefinition selects an OperationalDefinition by version
// identifier + binds its parameters. New definitions register here.
func resolveDefinition(version string, padSeconds int64) (derivation.OperationalDefinition, error) {
	switch version {
	case derivation.PaddedV1Version:
		return derivation.PaddedV1{PadSeconds: padSeconds}, nil
	default:
		return nil, fmt.Errorf("unknown definition-version %q; known versions: %s", version, derivation.PaddedV1Version)
	}
}

type payload struct {
	DefinitionVersion    string `json:"definition_version"`
	DefinitionParameters string `json:"definition_parameters"`
	Examined             int64  `json:"examined"`
	NewlyDerived         int64  `json:"newly_derived"`
	AlreadyDerived       int64  `json:"already_derived"`
}
