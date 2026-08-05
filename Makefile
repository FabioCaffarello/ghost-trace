# Ghost Trace — the developer entrypoint.
#
# Every CI step is `make <target>`. The workflow files hold no build
# knowledge of their own, so "green on my machine" and "green in CI"
# cannot drift into meaning two different things: to know what CI will
# do, run `make ci`.
#
# Ten Go modules under a go.work workspace. The per-module targets
# loop over $(GO_MODULES) so none can be quietly left out of a gate.

SHELL := /bin/bash

# .SHELLFLAGS needs GNU Make 3.82; macOS still ships 3.81, where it is
# silently ignored. Setting it and relying on `set -e` would therefore
# abort a failing loop on Ubuntu and swallow it on a Mac — a gate that
# means two different things on two machines, which is the one outcome
# this file exists to prevent. So it is not set, and every loop below
# ends its body with an explicit `|| exit 1`.

.DEFAULT_GOAL := help

# --- layout ----------------------------------------------------------
GO_MODULES  := services/collector services/archive services/decision-engine services/demo-web libs/archive libs/canonical libs/decision libs/eventstream libs/feature libs/genproto libs/id libs/metrics libs/middleware libs/policy libs/snapshot libs/substrate libs/tenant libs/wire
SERVICE     := services/collector
EXPERIMENTS := experiments
COVER_DIR   := .coverage

# The pinned Go tools are invoked by explicit path, quoted because this
# repository can live on a volume whose path contains a space.
#
# Prepending GOPATH/bin to PATH for the whole file would be shorter and
# is wrong: it hijacks the resolution of every tool, not just the
# pinned ones. It was tried, and it silently promoted a months-old
# `go install`ed buf over the current Homebrew one — the pin enforced
# on protoc-gen-go quietly broken for the tool that drives it. Only
# `buf generate` gets the prepend, and only because buf resolves its
# plugins by name from PATH.
TOOLBIN   := $(shell go env GOPATH)/bin
GOLANGCI  := "$(TOOLBIN)/golangci-lint"
GOVULN    := "$(TOOLBIN)/govulncheck"

# --- pinned tool versions --------------------------------------------
# Pinned, not floating: a linter that changes underneath you turns a
# clean tree red on an unrelated PR. Dependabot proposes the bumps.
PROTOC_GEN_GO_VER := v1.36.0
GOLANGCI_VER      := v2.12.2
GOVULNCHECK_VER   := v1.1.4
BUF_VER           := 1.68.4
VACUUM_VER        := v0.30.0
DOTCONTEXT_VER    := 1.1.1

# The agent-harness exporter, named once because `make context-sync`
# invokes it three times. Three separately pinned call sites is three
# chances to export half the harness with one version and half with
# another, and nothing downstream would notice.
DOTCONTEXT        := npx -y @dotcontext/cli@$(DOTCONTEXT_VER)

# Minimum toolchain versions asserted by `make bootstrap`.
GO_MIN     := 1.26
NODE_MIN   := 20
PYTHON_MIN := 3.12

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nGhost Trace — make targets\n"} \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } \
		/^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@echo ""

##@ Setup

.PHONY: bootstrap
bootstrap: ## Assert the toolchain versions this repo is built against
	@echo "== toolchain =="
	@command -v go >/dev/null || { echo "missing: go >= $(GO_MIN) — https://go.dev/dl"; exit 1; }
	@go version | awk '{print "  go        " $$3}'
	@go version | sed 's/.*go\([0-9]*\.[0-9]*\).*/\1/' | \
		awk -v min=$(GO_MIN) '{ if ($$1+0 < min+0) { print "  go " $$1 " < " min " — upgrade"; exit 1 } }'
	@command -v buf >/dev/null || { echo "missing: buf $(BUF_VER) — brew install bufbuild/buf/buf"; exit 1; }
	@buf --version | awk -v want=$(BUF_VER) '{ printf "  buf       %s%s\n", $$1, ($$1 == want ? "" : "  (pinned: " want ")") }'
	@command -v node >/dev/null || { echo "missing: node >= $(NODE_MIN) — experiment tiers 1,2,4,5,6"; exit 1; }
	@node --version | awk '{print "  node      " $$1}'
	@command -v python3 >/dev/null || { echo "missing: python3 >= $(PYTHON_MIN) — statistics + orchestrator"; exit 1; }
	@python3 --version | awk '{print "  python3   " $$2}'
	@if command -v docker >/dev/null; then docker --version | awk '{print "  docker    " $$3}'; \
		else echo "  docker    absent — only the container targets need it"; fi
	@echo "ok"

# A tool is acceptable only when GOPATH/bin holds a copy built from the
# pinned module version BY the Go toolchain currently on PATH. Both
# conditions have bitten this repository:
#
#   version   protoc-gen-go decides the bytes in libs/genproto, and a
#             different local build makes `make genproto-sync` fail with
#             a diff that looks like someone edited generated code.
#   toolchain golangci-lint and govulncheck parse Go source using the
#             compiler's own packages, so a copy built against an older
#             release cannot read a newer language version. It reports
#             "package requires newer Go version" and does not say which
#             binary is at fault. A Homebrew govulncheck built with Go
#             1.25 did exactly this against the 1.26 tree.
#
# GOPATH/bin is where these are installed and $(GOLANGCI)/$(GOVULN)
# invoke them from there by absolute path, so whatever a system package
# manager put on PATH is neither used nor fought with.
define ensure_tool
	bin="$(TOOLBIN)/$(1)"; \
	want_go=$$(go env GOVERSION); \
	want_mod="$(word 2,$(subst @, ,$(2)))"; \
	info=$$(go version -m "$$bin" 2>/dev/null || true); \
	if ! printf '%s' "$$info" | grep -q "$$want_go" || \
	   ! printf '%s' "$$info" | grep -q "$$want_mod"; then \
		echo "installing $(1) $$want_mod (built with $$want_go)"; \
		go install $(2); \
	fi
endef

.PHONY: tools
tools: tool-protoc-gen-go tool-golangci tool-govulncheck ## Install the pinned Go tools into GOPATH/bin

.PHONY: tool-protoc-gen-go
tool-protoc-gen-go:
	@$(call ensure_tool,protoc-gen-go,google.golang.org/protobuf/cmd/protoc-gen-go@$(PROTOC_GEN_GO_VER))

.PHONY: tool-golangci
tool-golangci:
	@$(call ensure_tool,golangci-lint,github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VER))

.PHONY: tool-govulncheck
tool-govulncheck:
	@$(call ensure_tool,govulncheck,golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VER))

.PHONY: setup
setup: ## Install experiment dependencies (node + python)
	cd $(EXPERIMENTS) && npm ci
	cd $(EXPERIMENTS) && python3 -m venv .venv && .venv/bin/pip install -q -r requirements.txt
	@echo "tier 3 additionally needs real Chrome — see experiments/README.md"

.PHONY: hooks
hooks: ## Install the lefthook git hooks
	@command -v lefthook >/dev/null || { echo "missing: lefthook — brew install lefthook"; exit 1; }
	lefthook install

##@ Build and test

.PHONY: build
build: ## Compile every module
	@for m in $(GO_MODULES); do echo "== build $$m"; (cd $$m && go build ./...) || exit 1; done

.PHONY: test
test: ## go test (no race detector — the fast loop)
	@for m in $(GO_MODULES); do echo "== test $$m"; (cd $$m && go test ./...) || exit 1; done

.PHONY: test-race
test-race: ## go test -race (what CI runs)
	@for m in $(GO_MODULES); do echo "== test -race $$m"; (cd $$m && go test -race ./...) || exit 1; done

# -coverpkg=./... so the wire-level characterization tests get credit for
# the packages they actually drive; without it `internal/app` reads as
# untested when it is exercised through every HTTP test in the suite.
.PHONY: coverage
coverage: ## Test with coverage; write .coverage/coverage.out and print the total
	@mkdir -p $(COVER_DIR)
	cd $(SERVICE) && go test -race -covermode=atomic -coverpkg=./... \
		-coverprofile="$(CURDIR)/$(COVER_DIR)/coverage.out" ./...
	@echo ""
	@# Reported from inside the module: the profile names packages by
	@# import path, which only resolves where a go.mod is in scope.
	@cd $(SERVICE) && go tool cover -func="$(CURDIR)/$(COVER_DIR)/coverage.out" | tail -1

.PHONY: bench
bench: ## Run the architecture benchmark (number 6)
	cd $(SERVICE) && go run ./cmd/bench-architecture

.PHONY: run
run: ## Serve the slice on 127.0.0.1:8080
	$(MAKE) -C $(SERVICE) run

##@ Quality gates

.PHONY: fmt
fmt: ## Rewrite Go sources with gofmt -s
	@for m in $(GO_MODULES); do (cd $$m && gofmt -s -w .) || exit 1; done

.PHONY: fmt-check
fmt-check: ## Fail if any Go source is not gofmt-clean
	@bad=$$(for m in $(GO_MODULES); do (cd $$m && gofmt -s -l .) | sed "s|^|$$m/|"; done); \
	if [ -n "$$bad" ]; then echo "not gofmt-clean:"; echo "$$bad"; echo "fix: make fmt"; exit 1; fi
	@echo "gofmt clean"

.PHONY: vet
vet: ## go vet every module
	@for m in $(GO_MODULES); do echo "== vet $$m"; (cd $$m && go vet ./...) || exit 1; done

.PHONY: lint
lint: tool-golangci ## golangci-lint over every module (config: .golangci.yml)
	@for m in $(GO_MODULES); do echo "== lint $$m"; (cd $$m && $(GOLANGCI) run ./...) || exit 1; done

# govulncheck exits 3 when it finds something reachable, so the failure
# is the `|| exit 1` and not a non-empty report: a run that prints five
# advisories and returns success is worse than no scan at all.
.PHONY: vuln
vuln: tool-govulncheck ## govulncheck every module against the Go vulnerability database
	@for m in $(GO_MODULES); do echo "== govulncheck $$m"; (cd $$m && $(GOVULN) ./...) || exit 1; done

# Conventional Commits, so semantic-release can derive a version from
# the log. PR_TITLE is read from the ENVIRONMENT, never interpolated
# into the recipe: it is attacker-controlled text on a fork PR, and
# `run: make lint-commit TITLE=${{ ... }}` is a shell injection.
.PHONY: lint-commit
lint-commit: ## Check PR_TITLE (or MSG_FILE) against Conventional Commits
	@if [ -n "$${PR_TITLE:-}" ]; then \
		scripts/check-commit-message.sh --text "$$PR_TITLE"; \
	elif [ -n "$${MSG_FILE:-}" ]; then \
		scripts/check-commit-message.sh --file "$$MSG_FILE"; \
	else \
		echo "set PR_TITLE=... or MSG_FILE=... (or run: make lint-commit-selftest)"; \
		exit 2; \
	fi

.PHONY: lint-commit-selftest
lint-commit-selftest: ## Prove the commit-message rule accepts and rejects the right set
	@scripts/check-commit-message.sh --selftest

# `go mod tidy` is run against a copy of the manifests and the originals
# are restored, so the check never mutates the tree it is judging.
.PHONY: tidy-check
tidy-check: ## Fail if `go mod tidy` would change any go.mod/go.sum
	@for m in $(GO_MODULES); do \
		cp $$m/go.mod $$m/go.mod.bak; \
		if [ -f $$m/go.sum ]; then cp $$m/go.sum $$m/go.sum.bak; fi; \
		(cd $$m && go mod tidy) || exit 1; \
		dirty=""; \
		diff -q $$m/go.mod $$m/go.mod.bak >/dev/null || dirty=1; \
		if [ -f $$m/go.sum.bak ] && ! diff -q $$m/go.sum $$m/go.sum.bak >/dev/null; then dirty=1; fi; \
		mv $$m/go.mod.bak $$m/go.mod; \
		if [ -f $$m/go.sum.bak ]; then mv $$m/go.sum.bak $$m/go.sum; fi; \
		if [ -n "$$dirty" ]; then echo "$$m is not tidy — fix: (cd $$m && go mod tidy)"; exit 1; fi; \
	done
	@echo "modules tidy"

##@ Schemas

# buf is not installed from here — it is a large binary and CI takes a
# prebuilt one — but the version is still asserted, because buf decides
# both the lint rules and the generated bytes that `genproto-sync`
# compares against what is committed. Exact equality, not a minimum:
# "newer" is not a safe default when the output is content-addressed.
#
# The check earns its keep on the shadowing case. GOPATH/bin precedes
# PATH (above) so a stale `go install`ed buf silently wins over a
# current Homebrew one, and the symptom would be an unexplained diff in
# generated code rather than anything pointing at buf.
.PHONY: require-buf
require-buf:
	@have=$$(buf --version 2>/dev/null || echo none); \
	if [ "$$have" != "$(BUF_VER)" ]; then \
		echo "buf $$have is on PATH ($$(command -v buf)); $(BUF_VER) is pinned"; \
		echo "fix: brew install bufbuild/buf/buf"; \
		exit 1; \
	fi

.PHONY: generate
generate: require-buf tool-protoc-gen-go ## buf generate -> libs/genproto (committed)
	PATH="$(TOOLBIN):$$PATH" buf generate

.PHONY: buf-lint
buf-lint: require-buf ## buf lint over schemas/
	buf lint

# subdir=schemas pins the against-input root so the comparison aligns
# with refs that predate buf.yaml (module root used to be schemas/).
.PHONY: buf-breaking
buf-breaking: require-buf ## buf breaking against origin/$(BASE_REF) (default: main)
	buf breaking --against ".git#branch=origin/$(or $(BASE_REF),main),subdir=schemas"

# Generated to a temporary tree and diffed, so the check depends on
# nothing but the generator's output.
#
# This gate used `git status --porcelain` for three milestones, and the
# same defect surfaced three times: git reports a file as changed
# relative to HEAD, which is not the question. It called a correctly
# regenerated but uncommitted binding "drift", and it called a newly
# added path drift on the very commit introducing it (R1.14 openapi,
# R1.15 fixtures). Diffing against a fresh generation asks the actual
# question, and it catches what git ALSO caught: a new proto whose
# .pb.go was never committed shows up as a file present in the temp
# tree and absent here.
.PHONY: genproto-sync
genproto-sync: require-buf tool-protoc-gen-go ## Fail if libs/genproto differs from a fresh buf generate
	@tmp=$$(mktemp -d "$${TMPDIR:-/tmp}/gtgenproto.XXXXXX"); \
	trap 'rm -rf "$$tmp"' EXIT; \
	PATH="$(TOOLBIN):$$PATH" buf generate -o "$$tmp" >/dev/null || \
		{ echo "buf generate failed — run: make generate"; exit 1; }; \
	if ! diff -ru libs/genproto/events "$$tmp/libs/genproto/events" >/dev/null 2>&1; then \
		diff -ru libs/genproto/events "$$tmp/libs/genproto/events" | head -40; \
		echo ""; \
		echo "libs/genproto is out of sync with schemas/"; \
		echo "fix: make generate && git add libs/genproto"; \
		exit 1; \
	fi; \
	echo "libs/genproto in sync"

##@ Contract

# The published HTTP contract is GENERATED from the Go types the
# handlers decode into, then committed and drift-gated — the same
# discipline libs/genproto is under, for the same reason: a
# specification nothing checks is a specification that describes last
# quarter's API.
.PHONY: openapi
openapi: ## Regenerate contract/openapi.yaml from the wire types
	cd $(SERVICE) && go run ./cmd/gen-openapi "$(CURDIR)/contract/openapi.yaml"

# Generates to a temporary file and diffs, rather than regenerating in
# place and asking git what changed. Two reasons: the check never
# mutates the tree it is judging, and it does not depend on git's
# staging state — a newly added file reads as "A" in `git status` and
# the git-based version called that drift on the very commit that
# introduced the file.
.PHONY: openapi-sync
openapi-sync: ## Fail if contract/openapi.yaml differs from a fresh generation
	@# An explicit XXXXXX template, not `mktemp -t openapi`: BSD mktemp
	@# treats -t as a prefix and GNU requires the X's, so the short form
	@# works on a Mac and fails on the runner. Errors are NOT swallowed —
	@# the first version hid the real cause behind a generic message.
	@tmp=$$(mktemp "$${TMPDIR:-/tmp}/openapi.XXXXXX"); \
	trap 'rm -f "$$tmp"' EXIT; \
	(cd $(SERVICE) && go run ./cmd/gen-openapi "$$tmp") >/dev/null || \
		{ echo "gen-openapi failed — see above"; exit 1; }; \
	if ! diff -u contract/openapi.yaml "$$tmp" >/dev/null 2>&1; then \
		diff -u contract/openapi.yaml "$$tmp" | head -60; \
		echo ""; \
		echo "contract/openapi.yaml is out of sync with the wire types"; \
		echo "fix: make openapi && git add contract/openapi.yaml"; \
		exit 1; \
	fi; \
	echo "contract/openapi.yaml in sync"

# The request fixtures are what the harness's wire modules produce —
# not hand-written, because a hand-written fixture is a third
# description of the wire, free to drift from both the clients and the
# server. Emitting them also asserts the JavaScript and Python halves
# of the harness agree byte for byte.
.PHONY: contract-fixtures
contract-fixtures: ## Regenerate contract/fixtures/ from the harness wire modules
	cd $(EXPERIMENTS) && python3 emit_fixtures.py

.PHONY: contract-fixtures-sync
contract-fixtures-sync: ## Fail if contract/fixtures/ drifts from the wire modules
	@# Emitted to a temporary directory and diffed, for the reason
	@# openapi-sync learned the hard way: git calls a newly added path
	@# untracked, and a git-based check reads that as drift on the very
	@# commit that introduces it.
	@tmp=$$(mktemp -d "$${TMPDIR:-/tmp}/gtfixtures.XXXXXX"); \
	trap 'rm -rf "$$tmp"' EXIT; \
	(cd $(EXPERIMENTS) && python3 emit_fixtures.py "$$tmp") >/dev/null || \
		{ echo "emitting fixtures failed — run: make contract-fixtures"; exit 1; }; \
	if ! diff -ru contract/fixtures/requests "$$tmp" >/dev/null 2>&1; then \
		diff -ru contract/fixtures/requests "$$tmp" | head -40; \
		echo ""; \
		echo "contract/fixtures is out of sync with the harness wire modules"; \
		echo "fix: make contract-fixtures && git add contract/fixtures"; \
		exit 1; \
	fi; \
	echo "contract/fixtures in sync"

.PHONY: openapi-lint
openapi-lint: tool-vacuum ## Lint the specification (structure, descriptions, operation ids)
	@# --fail-severity warn, because the ruleset is curated: anything it
	@# still reports is something we decided we care about.
	@"$(TOOLBIN)/vacuum" lint --details --no-banner --fail-severity warn \
		--ruleset .vacuum.yaml contract/openapi.yaml

.PHONY: tool-vacuum
tool-vacuum:
	@$(call ensure_tool,vacuum,github.com/daveshanley/vacuum@$(VACUUM_VER))

##@ Agent harness

# .context/ is the versioned source of agent knowledge; CLAUDE.md and
# .claude/ are generated exports of it (the latter gitignored). Same
# discipline as every other generated artifact here: edit the source,
# regenerate, and let a gate catch the drift.
.PHONY: context-sync
context-sync: ## Regenerate CLAUDE.md and .claude/ from .context/
	@# THREE exports, not one. `sync` handles agent playbooks and
	@# nothing else — it does not write CLAUDE.md and it does not touch
	@# .claude/skills/. CLAUDE.md comes from `export-rules`; the skills
	@# come from `admin skill export`.
	@#
	@# The first version of this target ran only the first command, so
	@# editing .context/docs/README.md turned context-sync-check red and
	@# `make context-sync` — the fix that check itself prints — could not
	@# turn it green. .claude/skills/ had been a frozen copy of
	@# .context/skills/ since the bootstrap for the same reason: an agent
	@# was reading instructions the repository had already revised.
	@#
	@# One version for all three. Three call sites pinned separately is
	@# three chances to export half the harness with a different tool.
	@tmp=$$(mktemp "$${TMPDIR:-/tmp}/claude-md.XXXXXX"); \
	trap 'rm -f "$$tmp"' EXIT; \
	cp CLAUDE.md "$$tmp" 2>/dev/null || :; \
	$(DOTCONTEXT) sync --preset claude --force || exit 1; \
	$(DOTCONTEXT) export-rules --preset claude --force || exit 1; \
	$(DOTCONTEXT) admin skill export --preset claude --force || exit 1; \
	python3 -c 'import pathlib, re, sys; \
	old = pathlib.Path(sys.argv[1]); \
	new = pathlib.Path(sys.argv[2]); \
	a = old.read_text() if old.exists() else None; \
	b = new.read_text(); \
	norm = lambda s: re.sub(r"^(> Auto-generated from .* on ).*", r"\1", s, flags=re.M); \
	sys.exit(0) if a is None or a == b or norm(a) != norm(b) else \
	  (new.write_text(a), print("CLAUDE.md content unchanged; kept the committed timestamp"))' \
	  "$$tmp" CLAUDE.md

.PHONY: context-sync-check
context-sync-check: ## Fail if CLAUDE.md has gone stale against .context/docs/README.md
	@# Checked in Python, not shell. The first version reconstructed the
	@# exporter's output format with sed and compared byte-wise: it was
	@# a second definition of what the exporter does, and it relied on
	@# GNU sed semantics, so it passed on macOS and failed on the
	@# runner. Third portability trap in this Makefile after
	@# .SHELLFLAGS and `mktemp -t`.
	@#
	@# The invariant that actually matters is containment: whatever else
	@# the export adds, the current source must be inside it.
	@python3 -c "import pathlib, sys; \
	src = pathlib.Path('.context/docs/README.md').read_text().strip(); \
	out = pathlib.Path('CLAUDE.md').read_text(); \
	sys.exit(0) if src in out else sys.exit('CLAUDE.md is stale against .context/docs/README.md\nfix: make context-sync && git add CLAUDE.md')"
	@echo "CLAUDE.md in sync"

# Deliberately outside `make ci`, for the same reason parity and shadow
# are: it needs the network. The check it exercises is offline on
# purpose — putting npm in the gate would make every push depend on a
# registry being up.
.PHONY: context-sync-selftest
context-sync-selftest: ## Prove `make context-sync` fixes what context-sync-check rejects (needs npm)
	@# The bug this guards survived a whole milestone: context-sync-check
	@# went red exactly as designed, while `make context-sync` — the fix
	@# that check prints in its own failure message — left it red,
	@# because the target ran one of the exporter's three commands. A
	@# check nobody can act on is worse than no check, because it reads
	@# as a working gate.
	@#
	@# So this asserts both halves: red on an edited source, and green
	@# after the documented fix. It edits a TRACKED file, so the restore
	@# is a trap rather than a final line — same reason tidy-check works
	@# on copies.
	@# Both files are saved and both are restored. Restoring only the
	@# source and regenerating would leave CLAUDE.md carrying a fresh
	@# timestamp — a selftest that dirties a tracked file every time it
	@# passes.
	@src=.context/docs/README.md; \
	tmp=$$(mktemp "$${TMPDIR:-/tmp}/ctxsrc.XXXXXX"); \
	out=$$(mktemp "$${TMPDIR:-/tmp}/ctxout.XXXXXX"); \
	cp "$$src" "$$tmp"; cp CLAUDE.md "$$out"; \
	trap 'cp "$$tmp" "$$src"; cp "$$out" CLAUDE.md; rm -f "$$tmp" "$$out"' EXIT; \
	printf '\n## context-sync-selftest\n\nWritten and removed by the selftest.\n' >> "$$src"; \
	if $(MAKE) -s context-sync-check >/dev/null 2>&1; then \
		echo "context-sync-selftest: FAIL — the check passed on an edited"; \
		echo "source, so it is not measuring anything"; \
		exit 1; \
	fi; \
	$(MAKE) -s context-sync >/dev/null || \
		{ echo "context-sync-selftest: FAIL — context-sync itself errored"; exit 1; }; \
	if ! $(MAKE) -s context-sync-check >/dev/null; then \
		echo "context-sync-selftest: FAIL — context-sync-check is still red"; \
		echo "after running the fix its own failure message prints"; \
		exit 1; \
	fi; \
	echo "context-sync-selftest: red on an edited source, green after its own fix"

# The archive parity check: what the collector wrote locally, the
# archive holds too. It needs a real broker and SKIPS without one, which
# is why it is its own target rather than part of `make ci` — a skipped
# parity test reported as a pass is the vacuous green this repository
# keeps finding.
#
#   docker run --rm -d -p 4222:4222 --name gt-nats nats:alpine -js
#   GT_NATS_URL=nats://127.0.0.1:4222 make parity
# The snapshot shadow: a session judged through the KV store decides
# exactly what the collector decides in process. Needs a real broker for
# the same reason parity does, and refuses to run without one.
.PHONY: shadow
shadow: ## Snapshot-shadow equivalence against a real broker (needs GT_NATS_URL)
	@if [ -z "$${GT_NATS_URL:-}" ]; then \
		echo "GT_NATS_URL is not set — the shadow test would skip, which is not a pass."; \
		echo "start a broker:  docker run --rm -d -p 4222:4222 --name gt-nats nats:alpine -js"; \
		exit 1; \
	fi
	cd $(SERVICE) && go test -count=1 ./internal/app/ -run TestDecisionThroughTheSnapshotStore -v

# The same equivalence one layer out: not through the KV store in one
# process, but through the topology a client actually meets. Two
# services running, the same decision request put to each, the answers
# compared field by field.
#
# Defaults point at the compose ports. It refuses to run without both
# services rather than skipping, for the reason above.
#
#   docker compose --profile core up -d
#   make shadow-http
.PHONY: shadow-http
shadow-http: ## A/B the collector against the engine, and check the demo is wired (needs the topology up)
	@if [ -z "$${GT_COLLECTOR_URL:-}" ] && [ -z "$${GT_ENGINE_URL:-}" ]; then \
		echo "using the compose defaults: collector :8080, engine :8082"; \
		echo "(override with GT_COLLECTOR_URL and GT_ENGINE_URL)"; \
	fi
	@for pair in "$${GT_COLLECTOR_URL:-http://127.0.0.1:8080}" \
	             "$${GT_ENGINE_URL:-http://127.0.0.1:8082}" \
	             "$${GT_DEMO_URL:-http://127.0.0.1:8083}"; do \
		curl -fsS -o /dev/null --max-time 3 "$$pair/healthz" || { \
			echo "$$pair is not answering /healthz — the shadow test would skip, which is not a pass."; \
			echo "bring both up:  docker compose --profile core up -d"; \
			exit 1; \
		}; \
	done
	cd services/decision-engine && \
		GT_COLLECTOR_URL="$${GT_COLLECTOR_URL:-http://127.0.0.1:8080}" \
		GT_ENGINE_URL="$${GT_ENGINE_URL:-http://127.0.0.1:8082}" \
		GT_DEMO_URL="$${GT_DEMO_URL:-http://127.0.0.1:8083}" \
		go test -count=1 ./internal/shadow/ -v

# The kill-test: take one service away and check what the rest
# promises. Each scenario is a degradation an ADR asserts, and until
# this existed all of them were prose. It needs the composed topology
# and REFUSES without it, for the reason parity and shadow do.
#
#   docker compose --profile core up -d
#   make kill-test
.PHONY: kill-test
kill-test: ## Take each service away and check the degradation promises (needs the topology up)
	@python3 deploy/kill-test.py

# The accounting phase's gate. Deliberately outside `make ci`: it stops
# and starts containers, which a pull-request runner should not be doing
# to a shared daemon, and it takes minutes rather than seconds.
#
# It refuses when the topology is down. A gate that skips is the exact
# failure this phase was opened to remove — `make shadow` skipping
# without GT_NATS_URL is how a broken tenant lookup reached CI.
.PHONY: loss-audit
loss-audit: ## Drive traffic, break things, and make the archive's books balance (needs the topology up)
	@python3 deploy/loss-audit.py

.PHONY: parity
parity: ## Archive parity against a real broker (needs GT_NATS_URL)
	@if [ -z "$${GT_NATS_URL:-}" ]; then \
		echo "GT_NATS_URL is not set — the parity test would skip, which is not a pass."; \
		echo "start a broker:  docker run --rm -d -p 4222:4222 --name gt-nats nats:alpine -js"; \
		exit 1; \
	fi
	cd services/archive && go test -count=1 ./internal/consumer/ -run Archive -v

##@ Experiments

.PHONY: experiments-check
experiments-check: ## Syntax-check every tier and run the asserted statistics selftest
	@echo "== python syntax"
	@cd $(EXPERIMENTS) && python3 -m compileall -q analyze.py numbers.py run.py make_links.py \
		publish_manifest.py schema/ \
		tiers/tier3_undetected_chromedriver.py testdata/make_synthetic_human.py
	@echo "== node syntax"
	@cd $(EXPERIMENTS) && for f in tiers/*.js lib/*.js *.mjs; do node --check "$$f" || exit 1; done
	@echo "== statistics selftest (asserted)"
	@cd $(EXPERIMENTS) && python3 analyze.py --selftest
	@echo "== numbers.json schema selftest"
	@cd $(EXPERIMENTS) && python3 -m schema --selftest
	@echo "== numbers-check selftest (asserted)"
	@python3 $(EXPERIMENTS)/numbers_check.py --selftest
	@echo "== release-derivation selftest (asserted)"
	@python3 scripts/next-release.py --selftest
	@echo "== workflow-guard selftest (asserted)"
	@python3 scripts/check-workflows.py --selftest
	@echo "== the required checks actually require every job"
	@python3 scripts/check-workflows.py

# Measuring and CHECKING are one target, because they were two habits
# and the second one kept being skipped. The run prints the six numbers
# either way; the check is what decides whether they reproduced.
.PHONY: numbers
numbers: ## Reproduce the six numbers and check them (the invariant; needs browsers)
	python3 $(EXPERIMENTS)/numbers.py
	@$(MAKE) --no-print-directory numbers-check

# The invariant CONTRIBUTING has always stated, finally enforced.
# Separate from `numbers` so a run already on disk can be re-checked
# against a newly published manifest without paying seven minutes for
# it.
.PHONY: numbers-check
numbers-check: ## Check the last run against the newest published manifest
	@python3 $(EXPERIMENTS)/numbers_check.py

# Promoting a run to docs/results/ is a deliberate act, not a side
# effect of measuring: the manifest is the record someone else will
# cite. It refuses a dirty tree, because a number produced from
# uncommitted code cannot be reproduced by anyone, including its
# author, and a manifest saying otherwise is worse than no manifest.
.PHONY: numbers-manifest
numbers-manifest: ## Publish the last run to docs/results/ (requires a clean tree)
	cd $(EXPERIMENTS) && python3 publish_manifest.py

##@ Containers

.PHONY: docker-build
docker-build: ## Build every image defined in compose.yml
	docker compose --profile core --profile experiments --profile observability build

.PHONY: pin-images
pin-images: ## Resolve the base-image tags in .env to sha256 digests
	deploy/docker/pin-base-images.sh

.PHONY: up
up: ## Start the core profile
	docker compose --profile core up -d

.PHONY: down
down: ## Stop every profile and drop the volumes
	docker compose --profile core --profile demo --profile experiments --profile observability down -v

.PHONY: restart
restart: down up ## Recreate the core profile

.PHONY: logs
logs: ## Follow the core profile logs
	docker compose --profile core logs -f

##@ Composite

# verify: what a human runs before pushing (and what lefthook's pre-push
# hook runs). Fast enough to be habitual.
.PHONY: verify
verify: fmt-check vet lint test-race ## Pre-push gate: format, vet, lint, race tests

# ci: the whole gate. `make ci` green locally and a green CI run are the
# same statement — that equivalence is the reason this file exists.
.PHONY: ci
ci: fmt-check tidy-check lint-commit-selftest buf-lint genproto-sync openapi-sync openapi-lint contract-fixtures-sync context-sync-check vet lint test-race coverage experiments-check vuln ## Everything CI runs

##@ Housekeeping

.PHONY: clean
clean: ## Remove coverage output and local run data
	rm -rf $(COVER_DIR)
	rm -rf $(SERVICE)/.run-data
	@echo "clean"
