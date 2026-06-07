.DEFAULT_GOAL := help

# Ghost Trace Sub-benchmark 1 deployment scaffold per decision-log
# §0205. Convenience layer over docker compose; the orchestrator script
# (run-sub-benchmark-1.sh) is the actual pipeline.

COMPOSE      ?= docker compose
ENV_FILE     ?= .env
RUNS_HOST    ?= ./runs

.PHONY: help
help:
	@echo "Sub-benchmark 1 scaffold targets:"
	@echo "  help               this message"
	@echo "  pin-base-images    pull current golang + debian base images and write their"
	@echo "                     sha256 digests into $(ENV_FILE) (overwrites GOLANG_IMAGE +"
	@echo "                     DEBIAN_IMAGE keys); operator runs this once before build"
	@echo "  build              build the ghost-trace runtime image (multi-stage)"
	@echo "  run                run the Sub-benchmark 1 pipeline end-to-end against the"
	@echo "                     operator-supplied CIC-IDS sample; emits a per-RUN_ID"
	@echo "                     manifest.json + per-step stdout/stderr capture"
	@echo "  observability-up   start the optional HTTP ingestion-api daemon on 127.0.0.1"
	@echo "                     (--profile observability); pipeline does not require this"
	@echo "  observability-down stop the optional HTTP daemon"
	@echo "  list-runs          list run-artifact directories in the runs volume"
	@echo "  copy-runs          copy all run-artifact directories to $(RUNS_HOST)/"
	@echo "  clean-old-runs N=10  delete all but the N most recent run-artifact dirs"
	@echo "                     in the named volume (operator-invoked; never automatic)"
	@echo "  clean              docker compose down + remove named volumes (DESTRUCTIVE:"
	@echo "                     drops substrate-data, cic-ids-input, run-artifacts)"

.PHONY: pin-base-images
pin-base-images:
	@./pin-base-images.sh $(ENV_FILE)

.PHONY: build
build:
	$(COMPOSE) --env-file $(ENV_FILE) build

.PHONY: run
run:
	$(COMPOSE) --env-file $(ENV_FILE) run --rm \
		-e GHOST_TRACE_GIT_COMMIT="$$(git rev-parse HEAD 2>/dev/null || echo unknown)" \
		cli /usr/local/bin/run-sub-benchmark-1.sh

.PHONY: observability-up
observability-up:
	$(COMPOSE) --env-file $(ENV_FILE) --profile observability up -d ingestion-api

.PHONY: observability-down
observability-down:
	$(COMPOSE) --env-file $(ENV_FILE) --profile observability down

.PHONY: list-runs
list-runs:
	$(COMPOSE) --env-file $(ENV_FILE) run --rm cli ls -1 /var/lib/ghost-trace/runs

.PHONY: copy-runs
copy-runs:
	@mkdir -p "$(RUNS_HOST)"
	$(COMPOSE) --env-file $(ENV_FILE) run --rm \
		-v "$(abspath $(RUNS_HOST))":/host-runs \
		cli /bin/bash -c 'cp -r /var/lib/ghost-trace/runs/. /host-runs/'

# clean-old-runs: opt-in retention policy per §0205 refinement (preserve
# run-artifacts by RUN_ID; cleanup is operator responsibility, never
# automatic). N defaults to 10; override on the command line:
#   make clean-old-runs N=20
N ?= 10
.PHONY: clean-old-runs
clean-old-runs:
	$(COMPOSE) --env-file $(ENV_FILE) run --rm cli /bin/bash -c '\
		cd /var/lib/ghost-trace/runs && \
		ls -1t | tail -n +$$(( $(N) + 1 )) | xargs -r rm -rf && \
		echo "retained $$(ls -1 | wc -l) run(s) (target: $(N))"'

.PHONY: clean
clean:
	$(COMPOSE) --env-file $(ENV_FILE) down -v
