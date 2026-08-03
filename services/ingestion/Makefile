# Ghost Trace ingestion service — build contract.
#
# internal/genproto is generated, not tracked. `generate` must run before
# build or test on a fresh checkout.
#
# The v1 Makefile enumerated all 58 protos by hand across 422 lines of
# per-CLI targets; this globs the schema tree instead, so adding or
# removing a proto needs no Makefile edit. M1 rewrites the schemas
# against integration-contract.md §2.

.DEFAULT_GOAL := help

TOOLBIN       := $(shell go env GOPATH)/bin
PROTOC_GEN_GO := $(TOOLBIN)/protoc-gen-go
GEN_OUT       := internal/genproto
SCHEMAS_DIR   := ../../schemas
PROTOBUF_VER  := v1.36.0
RUN_DATA      := .run-data

PROTO_FILES := $(shell cd $(SCHEMAS_DIR) && find . -name '*.proto' | sed 's|^\./||' | sort)

.PHONY: help
help:
	@echo "Targets:"
	@echo "  tools     install protoc-gen-go $(PROTOBUF_VER)"
	@echo "  generate  run protoc to produce $(GEN_OUT)/**/*.pb.go"
	@echo "  test      go test -race ./..."
	@echo "  build     go build ./..."
	@echo "  vet       go vet ./..."
	@echo "  run       serve the M1 slice on 127.0.0.1:8080"

.PHONY: run
run: generate
	go run ./cmd/ghost-trace -data $(RUN_DATA)

.PHONY: tools
tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@$(PROTOBUF_VER)

.PHONY: generate
generate:
	@mkdir -p $(GEN_OUT)
	protoc \
	  --plugin=protoc-gen-go="$(PROTOC_GEN_GO)" \
	  --proto_path=$(SCHEMAS_DIR) \
	  --go_out=$(GEN_OUT) \
	  --go_opt=paths=source_relative \
	  $(PROTO_FILES)

.PHONY: test
test:
	go test -race ./...

.PHONY: build
build:
	go build ./...

.PHONY: vet
vet:
	go vet ./...
