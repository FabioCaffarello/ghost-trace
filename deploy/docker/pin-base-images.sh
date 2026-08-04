#!/usr/bin/env bash
# pin-base-images: pull the floating tags for the compose base images,
# resolve their content-addressed sha256 digests, and write them as
# pinned references into the .env file. Run once at deployment time so
# subsequent rebuilds use byte-identical base layers regardless of
# registry retag activity — honest reproducibility has a temporal
# dimension, and floating tags break a rebuild six months later.
#
# Usage: deploy/docker/pin-base-images.sh [env-file]   (default: .env)

set -euo pipefail

ENV_FILE="${1:-.env}"

if [ ! -f "${ENV_FILE}" ]; then
    echo "pin-base-images: ${ENV_FILE} not found; copy .env.example to ${ENV_FILE} first" >&2
    exit 2
fi

resolve_digest() {
    local ref="$1"
    docker pull --quiet "${ref}" > /dev/null
    docker image inspect --format '{{ index .RepoDigests 0 }}' "${ref}"
}

update_env() {
    local key="$1" value="$2" file="$3"
    if grep -q "^${key}=" "${file}"; then
        # macOS BSD sed needs the empty-suffix form for -i.
        sed -i.bak -e "s|^${key}=.*$|${key}=${value}|" "${file}" && rm -f "${file}.bak"
    else
        echo "${key}=${value}" >> "${file}"
    fi
}

GOLANG_TAG="${GOLANG_TAG:-golang:1.26-alpine}"
DISTROLESS_TAG="${DISTROLESS_TAG:-gcr.io/distroless/static-debian12:nonroot}"
PLAYWRIGHT_TAG="${PLAYWRIGHT_TAG:-mcr.microsoft.com/playwright:v1.49.1-noble}"
PROMETHEUS_TAG="${PROMETHEUS_TAG:-prom/prometheus:v3.1.0}"

for pair in \
    "GOLANG_IMAGE ${GOLANG_TAG}" \
    "DISTROLESS_IMAGE ${DISTROLESS_TAG}" \
    "PLAYWRIGHT_IMAGE ${PLAYWRIGHT_TAG}" \
    "PROMETHEUS_IMAGE ${PROMETHEUS_TAG}"; do
    key="${pair%% *}"
    tag="${pair#* }"
    echo "pin-base-images: resolving ${tag}..."
    pinned=$(resolve_digest "${tag}")
    echo "  -> ${pinned}"
    update_env "${key}" "${pinned}" "${ENV_FILE}"
done

echo "pin-base-images: wrote pinned digests to ${ENV_FILE}"
