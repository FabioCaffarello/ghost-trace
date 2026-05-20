.DEFAULT_GOAL := help

TOOLBIN     := $(shell go env GOPATH)/bin
PROTOC_GEN_GO := $(TOOLBIN)/protoc-gen-go
GEN_OUT     := internal/genproto
SCHEMAS_DIR := ../../schemas
BIN_DIR     := bin

.PHONY: help
help:
	@echo "Targets:"
	@echo "  tools          install protoc-gen-go locally per tools.go pin"
	@echo "  generate       run protoc to produce $(GEN_OUT)/**/*.pb.go"
	@echo "  build          go build -trimpath -o $(BIN_DIR)/ingestion ."
	@echo "  test           go test -race ./..."
	@echo "  golden-corpus  regenerate canonical-corpus golden files (schemas-evolution event)"
	@echo "  clean          remove $(GEN_OUT) + $(BIN_DIR)"
	@echo ""
	@echo "Typical first build: make tools && make generate && make test"

.PHONY: tools
tools:
	cd ./internal/tools && go install google.golang.org/protobuf/cmd/protoc-gen-go

.PHONY: generate
generate: $(PROTOC_GEN_GO)
	@mkdir -p $(GEN_OUT)
	protoc \
	  --plugin=protoc-gen-go=$(PROTOC_GEN_GO) \
	  --proto_path=$(SCHEMAS_DIR) \
	  --go_out=$(GEN_OUT) \
	  --go_opt=paths=source_relative \
	  events/v1/declared_session.proto

$(PROTOC_GEN_GO):
	$(MAKE) tools

.PHONY: build
build:
	@mkdir -p $(BIN_DIR)
	go build -trimpath -o $(BIN_DIR)/ingestion .

.PHONY: test
test:
	go test -race ./...

.PHONY: golden-corpus
golden-corpus:
	go test ./internal/canonical/ -run TestCanonicalCorpus -update -v

.PHONY: clean
clean:
	rm -rf $(GEN_OUT) $(BIN_DIR)
