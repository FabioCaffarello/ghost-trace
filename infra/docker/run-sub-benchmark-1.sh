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
# Per §0213 + §0216 + §0220: loop over the formation hashes emitted at
# Step 5; invoke promote-automation-group for each WITH `-layer-b` so
# the promotion event carries §0138 inception-phase Layer B parameters
# (T_B=K_C=1/2; N_window=1000; N_A=cadence_seconds). The downstream
# demote-formations Step 8 requires these parameters per `layerb.go:132`
# `opts.Params == nil` early-return guard; without `-layer-b` at
# promotion, Layer B at demotion produces `evaluated: false` per §0219
# Finding 1.
#
# Pre-§0220 (per stale §0213 comment, now revised): Layer B parameters
# were deliberately scoped out at §0213 candidacy-materialization scope
# ("§0214 candidacy-evaluation scope per user-cravado split"). The §0216
# scope expansion added demote-formations Step 8 (which depends on Layer
# B parameters at promotion-time) but did NOT propagate the dependency
# to the §0213 scope comment here. §0219 Finding 1 surfaced this stale-
# scope-boundary subclass of §0022-emergence (seventh instance); §0220
# revises the scope comment + wires `-layer-b`.
#
# The parameter-propagation contract (docs/adapters/parameter-
# propagation-contract.md per §0220) documents this `-layer-b` wiring +
# any other opt-in option that affects downstream evaluation as new
# evidence surfaces (lazy-populated per §0220 MO: contract grows under
# empirical pressure, not via preventive audit).
#
# Stale-scope-boundary maintenance discipline: when an orchestrator step
# is added or modified, all `Per §NNNN:` scope comments within the
# script must be re-validated against the current step set. A scope
# comment that was correct at its original entry can become stale when
# a subsequent entry expands the step set in a way that depends on
# what the comment defers.
#
# The orchestrator loop preserves §0204 single-responsibility CLI
# discipline (each promotion is its own audit-grade invocation; no
# --auto-promote option was added to the form-from-candidates CLI per
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
            -cadence-seconds 86400 \
            -layer-b
    PROMOTE_INDEX=$(( PROMOTE_INDEX + 1 ))
done

log "promote-formations: ${PROMOTE_INDEX} promotion(s) committed"

# --- Step 7: measure-chain-morphology -------------------------------
#
# Per §0216 + §0143 D2 substrate-grounded comprovação criterion:
# substrate walk reads ALL Cat III formations and emits per-hypothesis
# chain morphology (chain_depth_max + chain_breadth_at_root) plus
# aggregate stats (chains_fracas vs chains_fortes per §0143).
#
# No input filter — measure-chain-morphology has no CLI option to
# restrict by hash. The substrate carries 20 formations after
# §0215 re-run (10 broken-chain historical preserved per §2.1 +
# 10 NEW clean-chain post-§0215 fix). Per §0216 + §0143 D2 audit-
# trail discipline: morphology_report contains ALL 20 for audit
# completeness; Sub-benchmark 1 comprovação claim applies to the
# 10 clean-chain subset only (broken-chain set has
# source_event_count=0 → fracas by construction; their morphology
# is historical artifact, not comprovação evidence).

invoke measure-chain-morphology \
    measure-chain-morphology "${DB_FLAGS[@]}"

# --- Step 8: demote-formations (shell-filtered to clean-chain) ------
#
# Per §0216 + user-cravado decision (c) — shell-based pre-filter
# preserving §0205 tier-3 audibility (each step shell-readable, not
# buried in Go). Selects promotion hashes whose formation has
# source_hashes_count > 0 (clean-chain set per §0215); skips the
# 10 broken-chain historical promotions (preserved in substrate per
# §2.1 but excluded from §0216 candidacy evaluation per §2.3 + §0143
# D2 chain-reconstructibility requirement).
#
# The filter joins formations_report (clean-chain identifier per
# source_hashes_count) with promotions_report (formation-to-promotion
# hash mapping). Only clean-chain promotion hashes flow into the
# demote loop.

DEMOTE_DIR="${RUN_DIR}/demote-formations"
mkdir -p "${DEMOTE_DIR}"

mapfile -t CLEAN_PROMOTION_HASHES < <(
    # Per §0217 Surface A + §0218 fix: each promote-formations/X.stdout
    # is a SINGLE OBJECT (not array). Pre-§0218 jq filter included
    # `.[] |` at the start which iterated the object's VALUES — first
    # value being a string caused the subsequent `.formation_event_hash`
    # access to fail with `Cannot index string with string
    # "formation_event_hash"`. Default jq per-file iteration processes
    # each input as one object; the `.[] |` was incorrect. §0217 first-
    # run RUN 20260608T154112Z surfaced this empirically: 0 clean-chain
    # promotions extracted → 0 demotions invoked → Layer B prediction
    # NOT empirically tested in that run.
    # DO NOT re-introduce `.[] |` to this filter — see §0217 Finding 1 +
    # §0218 fix entry for the postmortem.
    jq -r --slurpfile f "${RUN_DIR}/form-from-candidates.stdout" '
        select(
            .formation_event_hash as $fh |
            ($f[0].formations_committed[] |
             select(.formation_event_hash == $fh and .source_hashes_count > 0))
            | any
        ) |
        .promotion_event_hash
    ' "${PROMOTE_DIR}"/*.stdout
)

log "demote-formations: ${#CLEAN_PROMOTION_HASHES[@]} clean-chain promotion hash(es) to demote (broken-chain skipped per §0216 §2.3 + §0143 D2 audit-trail discipline)"

DEMOTE_INDEX=0
for promotion_hash in "${CLEAN_PROMOTION_HASHES[@]}"; do
    invoke "demote-formations/${DEMOTE_INDEX}" \
        demote-automation-group "${DB_FLAGS[@]}" \
            -promotion-event-hash "${promotion_hash}" \
            -reason "§0216 first candidacy evaluation; Layer B verdict inline; Layer A cadence ~0s (immediate demotion within shakedown per §0011 candidacy-not-barrier)"
    DEMOTE_INDEX=$(( DEMOTE_INDEX + 1 ))
done

log "demote-formations: ${DEMOTE_INDEX} demotion(s) committed"

# --- Step 9: manifest emission --------------------------------------
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
# strips the trailing newline command substitution adds; printf re-
# appends a newline so the promote-formations array assembly via
# `jq -s '.'` over a stdout stream of multiple read_duration_ns
# invocations slurps each value as an independent integer.
#
# Per §0215 + §0214 Finding 2 / Surface B: pre-§0215 printf used
# `'%s'` (no newline); the for-loop output concatenated 10 integers
# without separator (e.g. `19091375` + `17055083` + ... =
# `190913751705508318...`); `jq -s '.'` slurped the concatenation
# as a single number `~1.9e+79` corrupting the
# `step_durations.promote_formations` array in the manifest. The
# newline separator restores per-promotion timing observability.
read_duration_ns() {
    local f="$1"
    if [ -f "$f" ]; then
        local v
        v=$(<"$f")
        printf '%s\n' "$v"
    else
        printf '0\n'
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
        --argjson measure_morph        "$(read_duration_ns "${RUN_DIR}/measure-chain-morphology.duration_ns")" \
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
            form_from_candidates:    $form_from_cands,
            measure_chain_morphology: $measure_morph
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

# Per §0216: per-demotion stdout JSON aggregated into array (one entry
# per clean-chain promotion that was demoted). Each entry carries the
# demotion event hash + Layer B verdict inline (per cliutil.LayerBPayload).
# Empty array when no demotions invoked (e.g., zero clean-chain
# promotions from shell pre-filter).
demotions_report_json=$(
    if [ -d "${DEMOTE_DIR}" ] && compgen -G "${DEMOTE_DIR}"/*.stdout >/dev/null; then
        jq -s '.' "${DEMOTE_DIR}"/*.stdout
    else
        echo "[]"
    fi
)

# Per §0216: per-demotion durations into an array (one entry per
# demote-automation-group invocation). Empty array when no demotions
# invoked.
demote_durations_json=$(
    if [ -d "${DEMOTE_DIR}" ] && compgen -G "${DEMOTE_DIR}"/*.duration_ns >/dev/null; then
        for f in "${DEMOTE_DIR}"/*.duration_ns; do
            read_duration_ns "$f"
        done | jq -s '.'
    else
        echo "[]"
    fi
)

# Per §0216 + §0143 D2 audit-trail discipline: verdict.demotion_fired
# aggregation considers Layer B fired status across demotions_report
# entries. ANY demotion's Layer B fired → demotion_fired=true; ALL not-
# fired → demotion_fired=false. Empty demotions_report (no clean-chain
# promotions) → demotion_fired=false (null-safe). The aggregation
# considers ONLY clean-chain demotions (per §0216 shell pre-filter);
# broken-chain formations preserved per §2.1 but excluded per §0143 D2
# substrate-grounded comprovação criterion.
demotion_fired_aggregate=$(
    echo "${demotions_report_json}" | jq '[.[] | .layer_b.fired // false] | any // false'
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
    --slurpfile morph_rep     "${RUN_DIR}/measure-chain-morphology.stdout" \
    --argjson promotions_rep  "${promotions_report_json}" \
    --argjson promote_durs    "${promote_durations_json}" \
    --argjson demotions_rep   "${demotions_report_json}" \
    --argjson demote_durs     "${demote_durations_json}" \
    --argjson demotion_fired  "${demotion_fired_aggregate}" \
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
            included_steps: ["ingest", "derive-attributions", "replay-attributions", "signatures", "form-from-candidates", "promote-formations", "measure-chain-morphology", "demote-formations"],
            deferred_steps: [],
            deferral_reason: "§0216 closes the candidacy-evaluation scope (Layer B + chain morphology + demote); pipeline now end-to-end through demotion. Subsequent multi-run comprovação methodology scope is §0227+ named-but-non-binding (substrate accumulation over time produces influence chains that reference these formations; first non-trivial Layer B firing is the §0227+ pré-condição empírica)."
        },
        signatures: $sig_agg[0],
        ingest_report: ($ingest_rep[0] // null),
        derive_attributions_report: ($derive_rep[0] // null),
        formations_report: ($form_rep[0] // null),
        promotions_report: $promotions_rep,
        morphology_report: ($morph_rep[0] // null),
        demotions_report: $demotions_rep,
        step_durations: ($step_durations + {promote_formations: $promote_durs, demote_formations: $demote_durs}),
        verdict: {
            total_candidates: $total_cands,
            instrumented_non_firing: $non_firing,
            demotion_fired: $demotion_fired,
            non_firing_diagnosis_axis: {
                by_subtype:          "see signatures.* for per-subtype EvaluationStats",
                by_source:           "cic_ids only this scope",
                by_chain_morphology: "see morphology_report.stats for chains_fracas_count + chains_fortes_count per §0143 Sub-benchmark 1 definition; §0216 + §0143 D2 audit-trail discipline — clean-chain subset only for comprovação claim"
            }
        }
    }' > "${manifest_path}"

log "step=manifest emitted at ${manifest_path}"
log "verdict total_candidates=${total_candidates} instrumented_non_firing=${instrumented_non_firing} demotion_fired=${demotion_fired_aggregate}"
log "RUN_ID=${RUN_ID} complete"
