# Ghost Trace ingestion — the targets specific to running this service.
#
# Everything shared with the rest of the repository (build, test, vet,
# lint, coverage, generate, the schema gates) now lives in the root
# Makefile and is driven from there, so a gate cannot be strengthened at
# the root and quietly left behind here. From the repository root:
#
#   make help        the full target list
#   make build       compile every module
#   make test-race   the race suite
#   make generate    buf generate -> libs/genproto (committed)
#   make ci          everything CI runs
#
# Generated protobuf bindings live in the committed libs/genproto
# module, so a fresh checkout builds with no proto toolchain installed.

.DEFAULT_GOAL := help

RUN_DATA := .run-data

.PHONY: help
help:
	@echo "Service-local targets (everything else: 'make help' at the repository root):"
	@echo "  run       serve the slice on 127.0.0.1:8080"
	@echo "  capture   serve on 0.0.0.0:8080 with human capture enabled"

.PHONY: run
run:
	go run ./cmd/ghost-trace -data $(RUN_DATA)

# Bound to all interfaces so participant links reach the machine from
# other devices. Human capture is calendar-bound rather than
# effort-bound, so it starts as early as possible and runs while the
# rest of the work continues.
.PHONY: capture
capture:
	go run ./cmd/ghost-trace -addr 0.0.0.0:8080 -data $(RUN_DATA) \
	  -capture-log ../../experiments/results/human_sessions.jsonl
