# Ghost Trace deployment scaffold — Sub-benchmark 1 reproducibility image
# per decision-log §0205. Single multi-stage image bundles all ~46 CLIs
# plus the main `ingestion` daemon binary.
#
# Base images are pinned by sha256 digest, NOT floating tag, per the
# §0205 refinement (c) reproducibility-honesty discipline: tag drift
# would silently break rebuilds six months from now. The default digest
# values below are placeholders; the operator MUST override via
# `--build-arg GOLANG_IMAGE=...` and `--build-arg DEBIAN_IMAGE=...`
# (or via `.env` consumed by `make pin-base-images`). See README §Base
# image pinning.

ARG GOLANG_IMAGE=golang:1.23-alpine@sha256:0000000000000000000000000000000000000000000000000000000000000000
ARG DEBIAN_IMAGE=debian:bookworm-slim@sha256:0000000000000000000000000000000000000000000000000000000000000000

# ---- builder stage ---------------------------------------------------

FROM ${GOLANG_IMAGE} AS builder

# git is needed by `go mod` to resolve `replace` directives that point
# at version-tagged repositories; alpine omits it from the base.
RUN apk add --no-cache git

WORKDIR /src

# Copy go module + sum first to maximize layer-cache hit rate on
# dependency-only changes.
COPY services/ingestion/go.mod services/ingestion/go.sum ./services/ingestion/
WORKDIR /src/services/ingestion
RUN go mod download

# Bring in the rest of the build inputs.
WORKDIR /src
COPY services/ingestion ./services/ingestion
COPY schemas ./schemas

WORKDIR /src/services/ingestion

# Install all CLI binaries under cmd/* in one pass. `go install
# ./cmd/...` builds every package whose path matches the pattern and
# places the resulting binary in $GOBIN. The main `ingestion` daemon
# binary lives at the package root and is built separately.
ENV GOBIN=/out/bin
RUN mkdir -p /out/bin \
 && go install -trimpath ./cmd/... \
 && go build -trimpath -o /out/bin/ingestion .

# Pre-stage the orchestrator script so the runtime stage can COPY it
# from a known path.
COPY infra/docker/run-sub-benchmark-1.sh /out/bin/run-sub-benchmark-1.sh
RUN chmod +x /out/bin/run-sub-benchmark-1.sh

# ---- runtime stage ---------------------------------------------------

FROM ${DEBIAN_IMAGE}

# debian-slim runtime so the operator can:
#   - drop into a shell and inspect substrate.db with sqlite3 directly
#   - run bash orchestrator scripts (the Sub-benchmark 1 pipeline is
#     bash + the bundled CLIs; see infra/docker/run-sub-benchmark-1.sh)
#   - parse signature JSON envelopes via jq inside the orchestrator
#
# This is the §0205 tier-3 auditability commitment: the runtime image
# is operator-inspectable line by line, not a scratch image.
RUN apt-get update \
 && apt-get install -y --no-install-recommends \
        bash \
        ca-certificates \
        jq \
        sqlite3 \
        tini \
 && rm -rf /var/lib/apt/lists/*

COPY --from=builder /out/bin/* /usr/local/bin/

# Volume mount-points. The compose topology binds named volumes here.
RUN mkdir -p \
        /var/lib/ghost-trace/substrate \
        /var/lib/ghost-trace/cic-ids \
        /var/lib/ghost-trace/runs

WORKDIR /var/lib/ghost-trace

ENTRYPOINT ["/usr/bin/tini", "--"]
CMD ["/bin/bash"]
