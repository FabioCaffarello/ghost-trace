#!/usr/bin/env bash
# Ghost Trace Sub-benchmark 1 orchestrator per decision-log §0205.
#
# Pipeline (this scope):
#   1. ingest        — ingest-cic-ids against the operator-supplied CSV
#   2. derive-all    — replay-all-derived-actor-attributions (Cat II)
#   3. signatures    — 5x find-*-candidates (F3 corpus, read-only)
#   4. manifest      — emit per-run manifest aggregating step outputs
#
# Lifecycle steps (form + promote + measure-chain-morphology +
# demotion-evaluation) are DEFERRED to the §0206 follow-on per §4
# falsifiability — their bash shape depends on observing the empirical
# shape of signature JSON output on CIC-IDS-2017 substrate, evidence
# this script's first run produces.
#
# Per §0162 reachability-claim discipline + §4 falsifiability:
# instrumented non-firing IS a valid outcome. The manifest captures
# EvaluationStats per signature (ObservationsScanned +
# ObservationsSkippedNoActor + ObservationsSkippedWrongModality) so
# zero-candidate outcomes remain operator-diagnosable by subtype x
# signature x skip-reason without inspection of substrate by hand.
#
# Designed to run inside the `cli` container per infra/docker/compose.yml.
# Environment variables are populated by compose `environment:` block;
# the script does not read .env directly.

set -euo pipefail

# --- Environment + per-run directory --------------------------------

: "${GHOST_TRACE_SUBSTRATE_DB:?env unset}"
: "${GHOST_TRACE_BLOB_DIR:?env unset}"
: "${GHOST_TRACE_CIC_IDS_SAMPLE:?env unset}"
: "${GHOST_TRACE_RUNS_DIR:?env unset}"

RUN_ID="${RUN_ID:-$(date -u +%Y%m%dT%H%M%SZ)}"
RUN_DIR="${GHOST_TRACE_RUNS_DIR}/${RUN_ID}"
SIG_DIR="${RUN_DIR}/signatures"
mkdir -p "${SIG_DIR}"

LOG="${RUN_DIR}/run.log"
exec > >(tee -a "${LOG}") 2>&1

log() {
    printf '[%(%Y-%m-%dT%H:%M:%SZ)T] sub-benchmark-1: %s\n' -1 "$*"
}

# Invoke a CLI step. Captures stdout to ${step_path}.stdout, stderr
# to .stderr, and exit code. Aborts the run on non-zero exit.
invoke() {
    local label="$1"; shift
    local step_path="${RUN_DIR}/${label}"
    mkdir -p "$(dirname "${step_path}")"
    log "step=${label} cmd=$*"
    local rc=0
    "$@" > "${step_path}.stdout" 2> "${step_path}.stderr" || rc=$?
    if [ "${rc}" -ne 0 ]; then
        log "step=${label} FAILED exit=${rc} (see ${step_path}.stderr)"
        return "${rc}"
    fi
    log "step=${label} ok"
}

# Common substrate flags reused at every CLI invocation.
DB_FLAGS=(-db "${GHOST_TRACE_SUBSTRATE_DB}" -blobs "${GHOST_TRACE_BLOB_DIR}")

log "RUN_ID=${RUN_ID}"
log "substrate=${GHOST_TRACE_SUBSTRATE_DB}"
log "blobs=${GHOST_TRACE_BLOB_DIR}"
log "cic_ids_sample=${GHOST_TRACE_CIC_IDS_SAMPLE}"

# --- Step 1: ingest --------------------------------------------------

invoke ingest \
    ingest-cic-ids "${DB_FLAGS[@]}" \
    -channel cic-ids-file \
    -collector "cic-ids-2017-adapter:v1" \
    -progress 10000 \
    "${GHOST_TRACE_CIC_IDS_SAMPLE}"

# --- Step 2: derive-attributions ------------------------------------
#
# Per §0209: invokes attribution.DeriveAll via cmd/derive-actor-
# attribution, producing DerivedActorAttribution Cat II records from
# the NetworkObservation substrate under the network_5tuple_actor_v1
# operational definition (§0168). This is the operator-workflow
# closure of §0162 Gap (1) — the integration test at
# cic_ids_full_loop_integration_test.go:116 traversed this step
# manually; §0205's first real run surfaced the missing step at
# §0208; §0209 wires it.

invoke derive-attributions \
    derive-actor-attribution "${DB_FLAGS[@]}"

# --- Step 3: replay-attributions ------------------------------------
#
# Per §0208 diagnosis: this step is REPLAY semantic (verify-not-
# derive). It walks the DerivedActorAttribution records committed at
# Step 2 and re-validates each by re-deriving under the recorded
# operational definition, surfacing any drift between committed
# records and re-derivation outcome. Operationally meaningful only
# AFTER derive-attributions has populated the substrate; before §0209
# this step was misnamed "derive-all" in the orchestrator label and
# operationally a no-op against an empty Cat II surface.

invoke replay-attributions \
    replay-all-derived-actor-attributions "${DB_FLAGS[@]}"

# --- Step 4: signatures ---------------------------------------------
#
# F3 corpus: invoke each candidate-finder. CIC-IDS-only substrate has
# known reachability gaps (§0162) closed structurally at §0168 (Cat II
# attribution) + §0169 (alternative network signature
# tcp_flow_features_clustering_v1). The network finder is invoked with
# both -with-attribution and -signature flow-features per §0169 so its
# evaluation actually surfaces candidates from CIC-IDS substrate; the
# other finders are still invoked for instrumented-non-firing capture
# per §0162 discipline.

invoke signatures/find-automation-group-candidates \
    find-automation-group-candidates "${DB_FLAGS[@]}"

invoke signatures/find-automation-group-candidates-network \
    find-automation-group-candidates-network "${DB_FLAGS[@]}" \
        -signature flow-features \
        -with-attribution

invoke signatures/find-behavioral-cluster-candidates \
    find-behavioral-cluster-candidates "${DB_FLAGS[@]}"

invoke signatures/find-campaign-hypothesis-candidates \
    find-campaign-hypothesis-candidates "${DB_FLAGS[@]}"

invoke signatures/find-coordination-ring-candidates \
    find-coordination-ring-candidates "${DB_FLAGS[@]}"

# --- Step 4: manifest emission --------------------------------------
#
# Manifest schema: see infra/docker/manifest.schema.json. Captures
# inputs (git + image + sample), parameters, per-step status, and the
# F3 evaluation surface in the §0143 mandatory instrumentation axes
# (by_subtype x by_source x by_signature).
#
# Sample size + BLAKE3 are captured here: size via stat, BLAKE3 from
# operator-declared CIC_IDS_SAMPLE_BLAKE3 env var (manifest field
# notes whether it was declared or empty).

log "step=manifest assembling"

sample_size=$(stat -c '%s' "${GHOST_TRACE_CIC_IDS_SAMPLE}" 2>/dev/null || echo 0)

# Aggregate per-signature envelopes. Each find-* CLI emits JSON to
# stdout per §0163 envelope shape: {signature_name, candidate_count,
# candidates[], stats{...}}. We slurp each into a named subobject so
# the manifest preserves the full diagnostic surface per signature.
sig_aggregate=$(
    jq -n \
        --slurpfile a "${SIG_DIR}/find-automation-group-candidates.stdout" \
        --slurpfile b "${SIG_DIR}/find-automation-group-candidates-network.stdout" \
        --slurpfile c "${SIG_DIR}/find-behavioral-cluster-candidates.stdout" \
        --slurpfile d "${SIG_DIR}/find-campaign-hypothesis-candidates.stdout" \
        --slurpfile e "${SIG_DIR}/find-coordination-ring-candidates.stdout" \
        '{
            "automation_group_browser":   ($a[0] // null),
            "automation_group_network":   ($b[0] // null),
            "behavioral_cluster":         ($c[0] // null),
            "campaign_hypothesis":        ($d[0] // null),
            "coordination_ring":          ($e[0] // null)
         }'
)

# Total candidate count across all signatures. Drives the verdict
# field: if zero candidates emerged from any signature, the manifest
# records instrumented_non_firing = true per §0162 discipline.
total_candidates=$(
    echo "${sig_aggregate}" | jq '[.[] | .candidate_count // 0] | add // 0'
)
instrumented_non_firing=$( [ "${total_candidates}" -eq 0 ] && echo "true" || echo "false" )

# Per §0143 mandatory instrumentation axis (subtype x source x
# chain-morphology). Chain-morphology is absent in this scope because
# no formations are committed by this script (deferred to §0206); the
# manifest field is present as a typed null so the schema doesn't
# silently drop the axis.
manifest_path="${RUN_DIR}/manifest.json"
jq -n \
    --arg run_id              "${RUN_ID}" \
    --arg timestamp_utc       "$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    --arg ghost_trace_commit  "${GHOST_TRACE_GIT_COMMIT:-unknown}" \
    --arg image_digest        "${GHOST_TRACE_IMAGE_DIGEST:-unknown}" \
    --arg sample_path         "${GHOST_TRACE_CIC_IDS_SAMPLE}" \
    --arg sample_name         "${CIC_IDS_SAMPLE_NAME:-}" \
    --arg sample_blake3       "${CIC_IDS_SAMPLE_BLAKE3:-}" \
    --argjson sample_size     "${sample_size}" \
    --argjson formation_top_n "${FORMATION_TOP_N:-10}" \
    --argjson signatures      "${sig_aggregate}" \
    --argjson total_cands     "${total_candidates}" \
    --argjson non_firing      "${instrumented_non_firing}" \
    --arg layer_b_t_b_num     "${LAYER_B_T_B_NUMERATOR:-1}" \
    --arg layer_b_t_b_den     "${LAYER_B_T_B_DENOMINATOR:-2}" \
    --arg layer_b_k_c_num     "${LAYER_B_K_C_NUMERATOR:-1}" \
    --arg layer_b_k_c_den     "${LAYER_B_K_C_DENOMINATOR:-2}" \
    '{
        manifest_version: "f6-operational-v0.1",
        run_id: $run_id,
        timestamp_utc: $timestamp_utc,
        ghost_trace: {
            git_commit: $ghost_trace_commit,
            image_digest: $image_digest
        },
        inputs: {
            by_source: {
                cic_ids: {
                    sample_path: $sample_path,
                    sample_name: $sample_name,
                    blake3_declared: $sample_blake3,
                    size_bytes: $sample_size
                },
                synthetic: null,
                honeypot:  null
            }
        },
        parameters: {
            formation_top_n: $formation_top_n,
            layer_b: {
                t_b: { numerator: ($layer_b_t_b_num | tonumber), denominator: ($layer_b_t_b_den | tonumber) },
                k_c: { numerator: ($layer_b_k_c_num | tonumber), denominator: ($layer_b_k_c_den | tonumber) }
            }
        },
        pipeline_scope: {
            included_steps: ["ingest", "derive-attributions", "replay-attributions", "signatures"],
            deferred_steps: ["form", "promote", "measure-chain-morphology", "demotion-evaluation"],
            deferral_reason: "depends on empirical signature-output shape per §4 falsifiability; lands at §0206 follow-on"
        },
        signatures: $signatures,
        verdict: {
            total_candidates: $total_cands,
            instrumented_non_firing: $non_firing,
            demotion_fired: null,
            non_firing_diagnosis_axis: {
                by_subtype:          "see signatures.* for per-subtype EvaluationStats",
                by_source:           "cic_ids only this scope",
                by_chain_morphology: null
            }
        }
    }' > "${manifest_path}"

log "step=manifest emitted at ${manifest_path}"
log "verdict total_candidates=${total_candidates} instrumented_non_firing=${instrumented_non_firing}"
log "RUN_ID=${RUN_ID} complete"
