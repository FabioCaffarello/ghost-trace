# Ghost Trace ingestion service — build contract.
#
# Generated protobuf bindings live in the committed libs/genproto module,
# produced by `buf generate` at the repository root (see buf.gen.yaml).
# A fresh checkout builds with no proto toolchain installed; `generate`
# exists for schema changes and CI enforces that its output is committed.

.DEFAULT_GOAL := help

TOOLBIN      := $(shell go env GOPATH)/bin
REPO_ROOT    := ../..
PROTOBUF_VER := v1.36.0
RUN_DATA     := .run-data

.PHONY: help
help:
	@echo "Targets:"
	@echo "  tools     install protoc-gen-go $(PROTOBUF_VER) (buf itself: brew install buf)"
	@echo "  generate  buf generate -> libs/genproto (committed; run after schema edits)"
	@echo "  lint      buf lint over schemas/"
	@echo "  test      go test -race ./..."
	@echo "  build     go build ./..."
	@echo "  vet       go vet ./..."
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

.PHONY: tools
tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@$(PROTOBUF_VER)

.PHONY: generate
generate:
	cd $(REPO_ROOT) && PATH="$(TOOLBIN):$$PATH" buf generate

.PHONY: lint
lint:
	cd $(REPO_ROOT) && buf lint

.PHONY: test
test:
	go test -race ./...

.PHONY: build
build:
	go build ./...

.PHONY: vet
vet:
	go vet ./...
