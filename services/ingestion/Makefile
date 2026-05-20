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
	@echo "  verify-build   go build -trimpath -o $(BIN_DIR)/verify ./cmd/verify"
	@echo "  orphan-cleanup-build  go build -trimpath -o $(BIN_DIR)/orphan-cleanup ./cmd/orphan-cleanup"
	@echo "  derive-build   go build -trimpath -o $(BIN_DIR)/derive-operational-session ./cmd/derive-operational-session"
	@echo "  form-hypothesis-build  go build -trimpath -o $(BIN_DIR)/form-hypothesis ./cmd/form-hypothesis"
	@echo "  promote-hypothesis-build  go build -trimpath -o $(BIN_DIR)/promote-hypothesis ./cmd/promote-hypothesis"
	@echo "  demote-hypothesis-build  go build -trimpath -o $(BIN_DIR)/demote-hypothesis ./cmd/demote-hypothesis"
	@echo "  dissolve-hypothesis-build  go build -trimpath -o $(BIN_DIR)/dissolve-hypothesis ./cmd/dissolve-hypothesis"
	@echo "  merge-hypotheses-build  go build -trimpath -o $(BIN_DIR)/merge-hypotheses ./cmd/merge-hypotheses"
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
	  events/v1/declared_session.proto \
	  events/v1/ingestion_event.proto \
	  events/v1/network_event.proto \
	  events/v1/operational_session.proto \
	  events/v1/behavioral_cluster_formation.proto \
	  events/v1/behavioral_cluster_promotion.proto \
	  events/v1/behavioral_cluster_demotion.proto \
	  events/v1/behavioral_cluster_dissolution.proto \
	  events/v1/behavioral_cluster_merge.proto

$(PROTOC_GEN_GO):
	$(MAKE) tools

.PHONY: build
build:
	@mkdir -p $(BIN_DIR)
	go build -trimpath -o $(BIN_DIR)/ingestion .

.PHONY: verify-build
verify-build:
	@mkdir -p $(BIN_DIR)
	go build -trimpath -o $(BIN_DIR)/verify ./cmd/verify

.PHONY: orphan-cleanup-build
orphan-cleanup-build:
	@mkdir -p $(BIN_DIR)
	go build -trimpath -o $(BIN_DIR)/orphan-cleanup ./cmd/orphan-cleanup

.PHONY: derive-build
derive-build:
	@mkdir -p $(BIN_DIR)
	go build -trimpath -o $(BIN_DIR)/derive-operational-session ./cmd/derive-operational-session

.PHONY: form-hypothesis-build
form-hypothesis-build:
	@mkdir -p $(BIN_DIR)
	go build -trimpath -o $(BIN_DIR)/form-hypothesis ./cmd/form-hypothesis

.PHONY: promote-hypothesis-build
promote-hypothesis-build:
	@mkdir -p $(BIN_DIR)
	go build -trimpath -o $(BIN_DIR)/promote-hypothesis ./cmd/promote-hypothesis

.PHONY: demote-hypothesis-build
demote-hypothesis-build:
	@mkdir -p $(BIN_DIR)
	go build -trimpath -o $(BIN_DIR)/demote-hypothesis ./cmd/demote-hypothesis

.PHONY: dissolve-hypothesis-build
dissolve-hypothesis-build:
	@mkdir -p $(BIN_DIR)
	go build -trimpath -o $(BIN_DIR)/dissolve-hypothesis ./cmd/dissolve-hypothesis

.PHONY: merge-hypotheses-build
merge-hypotheses-build:
	@mkdir -p $(BIN_DIR)
	go build -trimpath -o $(BIN_DIR)/merge-hypotheses ./cmd/merge-hypotheses

.PHONY: test
test:
	go test -race ./...

.PHONY: golden-corpus
golden-corpus:
	go test ./internal/canonical/ -run TestCanonicalCorpus -update -v

.PHONY: clean
clean:
	rm -rf $(GEN_OUT) $(BIN_DIR)
