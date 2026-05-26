#!/usr/bin/env bash
# pin-base-images: pull the floating tags for the Dockerfile base
# images, resolve their content-addressed sha256 digests, and write
# them as pinned references into the .env file. Operator runs this
# once at deployment time so subsequent rebuilds use byte-identical
# base layers regardless of registry retag activity.
#
# Per decision-log §0205 refinement (c): reproducibility honesta tem
# dimensão temporal — floating tags break rebuild six months later.

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

GOLANG_TAG="${GOLANG_TAG:-golang:1.23-alpine}"
DEBIAN_TAG="${DEBIAN_TAG:-debian:bookworm-slim}"

echo "pin-base-images: resolving ${GOLANG_TAG}..."
golang_pinned=$(resolve_digest "${GOLANG_TAG}")
echo "  -> ${golang_pinned}"

echo "pin-base-images: resolving ${DEBIAN_TAG}..."
debian_pinned=$(resolve_digest "${DEBIAN_TAG}")
echo "  -> ${debian_pinned}"

# Rewrite GOLANG_IMAGE and DEBIAN_IMAGE entries in the env file. Any
# pre-existing entries are replaced; missing entries are appended.
update_env() {
    local key="$1"
    local value="$2"
    local file="$3"
    if grep -q "^${key}=" "${file}"; then
        # macOS BSD sed needs the empty-suffix form for -i.
        sed -i.bak -e "s|^${key}=.*$|${key}=${value}|" "${file}" && rm -f "${file}.bak"
    else
        echo "${key}=${value}" >> "${file}"
    fi
}

update_env GOLANG_IMAGE "${golang_pinned}" "${ENV_FILE}"
update_env DEBIAN_IMAGE "${debian_pinned}" "${ENV_FILE}"

echo "pin-base-images: wrote pinned digests to ${ENV_FILE}"
