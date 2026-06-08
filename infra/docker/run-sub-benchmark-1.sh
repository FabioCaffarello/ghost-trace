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
# to .stderr, wall-clock nanoseconds to .duration_ns, and exit code.
# Aborts the run on non-zero exit.
#
# Per §0211: wall-clock per step captured so the §0212 +/- §0213
# diagnoses (DeriveAll O(n²); per-column coercion) have an audit
# surface at the orchestrator tier. The .duration_ns file is the
# operational complement to the per-CLI elapsed_ns fields in the
# ingest + derive Report JSON — both axes are emitted because they
# answer different questions (CLI-internal cost vs. CLI-launch +
# CLI-internal + CLI-teardown total).
invoke() {
    local label="$1"; shift
    local step_path="${RUN_DIR}/${label}"
    mkdir -p "$(dirname "${step_path}")"
    log "step=${label} cmd=$*"
    local rc=0
    local start_ns end_ns
    start_ns=$(date +%s%N)
    "$@" > "${step_path}.stdout" 2> "${step_path}.stderr" || rc=$?
    end_ns=$(date +%s%N)
    printf '%s\n' "$(( end_ns - start_ns ))" > "${step_path}.duration_ns"
    if [ "${rc}" -ne 0 ]; then
        log "step=${label} FAILED exit=${rc} duration_ns=$(( end_ns - start_ns )) (see ${step_path}.stderr)"
        return "${rc}"
    fi
    log "step=${label} ok duration_ns=$(( end_ns - start_ns ))"
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

# --- Step 5: form-from-candidates -----------------------------------
#
# Per §0213: bridge from F3 candidate envelope to committed
# AutomationGroupFormation events. Reads the Network signature's
# stdout (the §0163 envelope JSON) and commits the top-N=10
# AutomationGroup formations selected by --rank-by=actor-count (the
# §0213 default; deterministic with content-hash tiebreak).
#
# This step is the operator-tier lift of the §0157 test helper
# pattern. Pre-§0213 the F3-candidate → formation path existed only
# in integration tests via test-only helpers; the §0212 first-real-
# run surfaced this gap empirically (252 candidates emitted with no
# operator path to materialize them).
#
# Other signatures' envelopes are NOT bridged in this scope. Only
# the Network applicable signature (per §0208 modality-applicability
# finding) emits non-zero candidates today; bridges for other
# subtypes lift via subsequent §-entries when those signatures
# produce non-trivial output.

invoke form-from-candidates \
    form-automation-group-from-candidate "${DB_FLAGS[@]}" \
        -top-n "${FORMATION_TOP_N:-10}" \
        -rank-by actor-count \
        "${SIG_DIR}/find-automation-group-candidates-network.stdout"

# --- Step 6: promote-formations -------------------------------------
#
# Per §0213: loop over the formation hashes emitted at Step 5; invoke
# promote-automation-group for each. Layer B parameters are NOT
# populated at promotion time in this scope (§0213 is candidacy
# materialization; Layer B candidacy gating + chain morphology +
# demote evaluation are §0214 candidacy-evaluation scope per the user-
# cravado split).
#
# The orchestrator loop preserves §0204 single-responsibility CLI
# discipline (each promotion is its own audit-grade invocation; no
# --auto-promote flag was added to the form-from-candidates CLI per
# §0213 path-rejection rationale).
#
# Per-promotion .stdout/.stderr/.duration_ns captured by invoke() into
# ${RUN_DIR}/promote-formations/<index>.*. Manifest assembly slurps the
# per-index .stdout files into the promotions_report array.

PROMOTE_DIR="${RUN_DIR}/promote-formations"
mkdir -p "${PROMOTE_DIR}"

# Extract formation hashes from form-from-candidates stdout JSON. jq
# emits one hash per line; bash mapfile reads into array.
mapfile -t FORMATION_HASHES < <(
    jq -r '.formations_committed[].formation_event_hash' \
        "${RUN_DIR}/form-from-candidates.stdout"
)

log "promote-formations: ${#FORMATION_HASHES[@]} formation hash(es) to promote"

PROMOTE_INDEX=0
for formation_hash in "${FORMATION_HASHES[@]}"; do
    invoke "promote-formations/${PROMOTE_INDEX}" \
        promote-automation-group "${DB_FLAGS[@]}" \
            -formation-event-hash "${formation_hash}" \
            -cadence-seconds 86400
    PROMOTE_INDEX=$(( PROMOTE_INDEX + 1 ))
done

log "promote-formations: ${PROMOTE_INDEX} promotion(s) committed"

# --- Step 7: manifest emission --------------------------------------
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

# Per §0211 Bug A fix: write the per-signature aggregate JSON to a file
# instead of carrying it in a bash variable that subsequently flows
# through `--argjson` (argv-bound; ARG_MAX-bounded). The pre-§0211
# orchestrator failed at this exact site under the §0208 RUN_ID
# 20260528T030751Z real-world sample size (signature candidate lists
# at 644K-observation scale exceeded the kernel ARG_MAX limit on the
# manifest jq invocation, dropping the manifest entirely).
#
# --slurpfile wraps the file content in a JSON array, even when the
# file contains a single object. The .signatures[0] in the jq
# expression below is the unwrap. Per §0211 Bug A fix — DO NOT
# remove [0] without also removing --slurpfile.
sig_aggregate_path="${RUN_DIR}/.sig_aggregate.json"
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
     }' > "${sig_aggregate_path}"

# Total candidate count across all signatures. Drives the verdict
# field: if zero candidates emerged from any signature, the manifest
# records instrumented_non_firing = true per §0162 discipline. Reads
# from the file (not the bash variable) per the §0211 Bug A pattern.
total_candidates=$(
    jq '[.[] | .candidate_count // 0] | add // 0' "${sig_aggregate_path}"
)
instrumented_non_firing=$( [ "${total_candidates}" -eq 0 ] && echo "true" || echo "false" )

# Read a per-step .duration_ns file (written by invoke()). Returns 0
# when the file does not exist (step did not run / step failed before
# .duration_ns was written — defensive). Uses bash $(<file) which
# strips the trailing newline command substitution adds.
read_duration_ns() {
    local f="$1"
    if [ -f "$f" ]; then
        local v
        v=$(<"$f")
        printf '%s' "$v"
    else
        printf '0'
    fi
}

# Aggregate per-step wall-clock durations (nanoseconds). Small JSON;
# --argjson is safe. The CLI-internal elapsed_ns fields inside
# ingest_report + derive_attributions_report (see below) measure
# different surfaces: ingest_report.elapsed_ns is the cic_ids.Ingest
# function's own elapsed (header read + parse + commit loop);
# step_durations.ingest is the orchestrator-observed wall-clock
# (process launch + CLI-internal + process teardown). The two axes
# answer different questions per §0211 Methodological observation 2.
step_durations_json=$(
    jq -n \
        --argjson ingest               "$(read_duration_ns "${RUN_DIR}/ingest.duration_ns")" \
        --argjson derive_attributions  "$(read_duration_ns "${RUN_DIR}/derive-attributions.duration_ns")" \
        --argjson replay_attributions  "$(read_duration_ns "${RUN_DIR}/replay-attributions.duration_ns")" \
        --argjson sig_ag_browser       "$(read_duration_ns "${SIG_DIR}/find-automation-group-candidates.duration_ns")" \
        --argjson sig_ag_network       "$(read_duration_ns "${SIG_DIR}/find-automation-group-candidates-network.duration_ns")" \
        --argjson sig_bc               "$(read_duration_ns "${SIG_DIR}/find-behavioral-cluster-candidates.duration_ns")" \
        --argjson sig_ch               "$(read_duration_ns "${SIG_DIR}/find-campaign-hypothesis-candidates.duration_ns")" \
        --argjson sig_cr               "$(read_duration_ns "${SIG_DIR}/find-coordination-ring-candidates.duration_ns")" \
        --argjson form_from_cands      "$(read_duration_ns "${RUN_DIR}/form-from-candidates.duration_ns")" \
        '{
            ingest:                  $ingest,
            derive_attributions:     $derive_attributions,
            replay_attributions:     $replay_attributions,
            signatures: {
                automation_group_browser: $sig_ag_browser,
                automation_group_network: $sig_ag_network,
                behavioral_cluster:       $sig_bc,
                campaign_hypothesis:      $sig_ch,
                coordination_ring:        $sig_cr
            },
            form_from_candidates:    $form_from_cands
         }'
)

# Per §0213: per-promotion durations into an array (one entry per
# committed formation hash). Reads each PROMOTE_DIR/<i>.duration_ns
# file in order, builds a JSON array. Empty array when no formations
# committed (e.g., zero AG candidates in envelope).
promote_durations_json=$(
    if [ -d "${PROMOTE_DIR}" ] && compgen -G "${PROMOTE_DIR}"/*.duration_ns >/dev/null; then
        for f in "${PROMOTE_DIR}"/*.duration_ns; do
            read_duration_ns "$f"
        done | jq -s '.'
    else
        echo "[]"
    fi
)

# Per §0213: per-promotion stdout JSON aggregated into array (one
# entry per committed formation; mirrors promote_durations_json shape).
promotions_report_json=$(
    if [ -d "${PROMOTE_DIR}" ] && compgen -G "${PROMOTE_DIR}"/*.stdout >/dev/null; then
        jq -s '.' "${PROMOTE_DIR}"/*.stdout
    else
        echo "[]"
    fi
)

# Per §0143 mandatory instrumentation axis (subtype x source x
# chain-morphology). Chain-morphology is absent in this scope because
# no formations are committed by this script (deferred to §0206); the
# manifest field is present as a typed null so the schema doesn't
# silently drop the axis.
#
# Per §0211: ingest_report + derive_attributions_report read from the
# respective step's .stdout via --slurpfile (same pattern as $a-$e
# above for sig_aggregate; .[0] unwrap below for the same reason). The
# ingest.stdout file is the cic_ids.Report JSON; derive-attributions.
# stdout is the derive payload JSON. Both flow through file-based jq
# input to remain ARG_MAX-safe regardless of dataset size.
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
    --slurpfile sig_agg       "${sig_aggregate_path}" \
    --slurpfile ingest_rep    "${RUN_DIR}/ingest.stdout" \
    --slurpfile derive_rep    "${RUN_DIR}/derive-attributions.stdout" \
    --slurpfile form_rep      "${RUN_DIR}/form-from-candidates.stdout" \
    --argjson promotions_rep  "${promotions_report_json}" \
    --argjson promote_durs    "${promote_durations_json}" \
    --argjson step_durations  "${step_durations_json}" \
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
            included_steps: ["ingest", "derive-attributions", "replay-attributions", "signatures", "form-from-candidates", "promote-formations"],
            deferred_steps: ["measure-chain-morphology", "demotion-evaluation"],
            deferral_reason: "Layer B candidacy evaluation + chain morphology + demote evaluation are §0214 candidacy-evaluation scope per §0213 user-cravado split (candidacy materialization vs candidacy evaluation as distinct lifecycle shapes)"
        },
        signatures: $sig_agg[0],
        ingest_report: ($ingest_rep[0] // null),
        derive_attributions_report: ($derive_rep[0] // null),
        formations_report: ($form_rep[0] // null),
        promotions_report: $promotions_rep,
        step_durations: ($step_durations + {promote_formations: $promote_durs}),
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
