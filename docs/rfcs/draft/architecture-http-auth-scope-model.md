# RFC — Architecture: HTTP auth-scope model (T3/T4 unblock)

- **Status:** accepted
- **Authors:** committee
- **Date:** 2026-05-21
- **Type:** architecture
- **Affects:** [`services/ingestion/internal/httpapi/`](../../../services/ingestion/internal/httpapi/) (auth + handler dispatch); [`services/ingestion/main.go`](../../../services/ingestion/main.go) (CLI options for per-tier token files); [`services/ingestion/README.md`](../../../services/ingestion/README.md) (deployment shape); [`docs/charter/decision-log.md` §0035](../../charter/decision-log.md) (single-token model — superseded for multi-tier deployments; preserved as the zero-config default); [`§0037`](../../charter/decision-log.md)–[`§0038`](../../charter/decision-log.md) (mTLS + IngestionEvent identity recording — extended to T3/T4 lifecycle-event pairs); [`§0094`](../../charter/decision-log.md) (operation-tier classification — this RFC selects the wire format the classification anticipated). No Charter section amended.

> RFCs in Ghost Trace are proposals subject to constitutional review. The [Constitutional Charter](../../charter/constitutional-charter.md) is the authoritative reference. RFCs that conflict with the Charter must either be rejected or accompanied by a corresponding charter amendment (see [`../../charter/amendments.md`](../../charter/amendments.md)).

---

## Summary

Select **α composed with γ** from [`§0094`](../../charter/decision-log.md)'s candidate evolutions for the HTTP auth-scope wire format: **multi-token files mapped to tier scopes (α)** as the primary mechanism, with **mTLS cert subject-name-to-scope mapping (γ)** as the optional per-actor-attribution upgrade. The current [`§0035`](../../charter/decision-log.md) single-token shape is preserved as a backward-compatible zero-config default — when only one token is configured, all protected paths require that token (current behavior). When per-tier token files are configured, each path is gated by its tier's token; T3 + T4 endpoints additionally require a recorded per-actor attribution on the corresponding `IngestionEvent` (extending [`§0038`](../../charter/decision-log.md)'s mTLS-identity-threading to lifecycle-event pairs). The RFC unblocks HTTP `orphan-cleanup` (T3) and HTTP write-side for the 24 Cat III lifecycle CLIs (T4) named at [`§0093`](../../charter/decision-log.md) + [`§0094`](../../charter/decision-log.md).

## Motivation

[`§0094`](../../charter/decision-log.md) classified HTTP operations into five tiers (T0 public, T1 producer, T2 operator-read, T3 substrate-admin, T4 constitutional-act). The current bearer-token model ([`§0035`](../../charter/decision-log.md)) collapses T1+T2+T3+T4 into one equivalence class — sufficient at single-tenant inception but insufficient for T3/T4 operations whose substrate consequences are destructive ([`cmd/orphan-cleanup`](../../../services/ingestion/cmd/orphan-cleanup/) at T3) or constitutional (the 24 Cat III lifecycle CLIs at T4). The cost of not making the change: HTTP write-side cannot extend beyond T1 ingestion; remote operators continue to need shell access for substrate maintenance + Cat III lifecycle work; the §0094-recorded gap remains an open blocker rather than a closed RFC.

## Constitutional Review

Verbatim output of the rfc-author §1 pre-authorship analysis (Q1–Q6).

### Q1 — Charter invariants touched

- **§2.1 Observational Integrity (frozen):** preserved. T3 (`orphan-cleanup`) operates on the blob-store half of the substrate; the events table remains append-only ([`§0027`](../../charter/decision-log.md) AP4–AP6 unchanged). T4 (lifecycle-event writes) is structurally identical to existing CLI-driven Cat III lifecycle ops ([`§0044`](../../charter/decision-log.md)–[`§0076`](../../charter/decision-log.md)); HTTP is a thin transport over the same `hypothesis.{Form,Promote,...}` library entry points. No §2.1 mechanism modified.
- **§2.2 Epistemic Separation (frozen):** preserved. T1 commits Cat I primary observations; T4 commits Cat I lifecycle events documenting Cat III lifecycle operations (per §2.5 BC5 — lifecycle events ARE Cat I records by storage classification; they document Cat III operations by domain). T3 commits a new Cat I audit record per Proposal item 4 (none today). No category boundary crosses at this RFC.
- **§2.3 Provenance Integrity (frozen v0.4):** preserved with extension obligation. Per-actor attribution on T4 lifecycle-event pairs uses the [`§0038`](../../charter/decision-log.md) `IngestionEvent` shape (already enriched with mTLS-CN/SAN/cert-SHA fields for T1). Extending that recording to T3/T4 IngestionEvent pairs is structurally subordinate to §2.3's existing commitment.
- **§2.5 Hypothesis Lifecycle Explicitness (frozen v0.3):** preserved. The §2.5 lifecycle event chain is the same chain T4 endpoints expose; per-actor attribution becomes a load-bearing field on the IngestionEvent paired with each lifecycle event.
- **§2.4 + §2.6 (pending — empirical pressure phase):** no interaction. Auth is service-tier access control; per [`§0035`](../../charter/decision-log.md) precedent, no pending-invariant working stub is modified.

### Q2 — Glossary redefinition

No. The terms `bearer token`, `mTLS`, `scope`, `tier` are technology + service-tier vocabulary. The canonical project terms (`substrate`, `observation`, `hypothesis`, lifecycle terms) are used unchanged. The §0094 term `operation tier` is reused as written; not promoted to canonical glossary at this RFC.

### Q3 — Implicit resolution of open Ontology questions

None. Auth-scope is wire-format selection at the service tier; the five open Ontology questions (`ontology.md` Q3 independence formal definition, Q5 transitive half; `provenance-model.md` OMQ #1 Granularity, OMQ #4 Cross-domain) are not touched.

### Q4 — Charter amendment required

No. Auth is service-tier access-control per [`§0035`](../../charter/decision-log.md) precedent ("no §2.1/§2.2/§2.3/§2.5 commitment is affected"). No Charter prose modified.

### Q5 — New invariant introduced

No. The RFC operationalizes a [`§0094`](../../charter/decision-log.md) classification into a wire format. The cross-tier per-actor-attribution requirement at [`§0094`](../../charter/decision-log.md) is already a Charter-level obligation under §2.5 BC5 + §2.3 + §2.1 (the [`§0038`](../../charter/decision-log.md) precedent for T1); the RFC extends the mechanism, not the obligation.

### Q6 — Ceremony without behavioral consequence

No. Falsifiable by deletion: without the RFC, T3 + T4 cannot be exposed over HTTP (per [`§0093`](../../charter/decision-log.md) + [`§0094`](../../charter/decision-log.md) named exclusions); operators continue to need shell access for substrate maintenance + Cat III work; the §0094-recorded gap remains a blocker.

## Proposal

Concrete commitments:

1. **Multi-tier token files (α).** Four new options at handler construction:
   - `WithAuthTierToken(tier, token)` — accepts `tier ∈ {"producer", "operator-read", "substrate-admin", "constitutional-act"}` (string-typed for forward-compatibility with [`§0094`](../../charter/decision-log.md)'s tier names; T0 has no token).
   - Corresponding CLI options: `--http-auth-{producer,operator-read,substrate-admin,constitutional-act}-token-file <path>`. Same file-precedence + whitespace-trim + empty-rejection contract as [`§0035`](../../charter/decision-log.md) `--http-auth-token-file`.
   - **Backward compatibility.** `WithAuthToken` / `--http-auth-token{,-file}` remain. When the single-token form is configured, the handler treats it as the union of all four tiers (current behavior — preserves [`§0035`](../../charter/decision-log.md) deployments). When per-tier tokens are configured, the single-token form is rejected at configuration time (handler construction returns an error / main exits non-zero).

2. **Handler-side tier dispatch.** Each route is annotated with its tier (the [`§0094`](../../charter/decision-log.md) classification is the authoritative reference). `ServeHTTP` looks up the route's tier; auth-check loads the per-tier token; constant-time compare against the request's `Authorization: Bearer <token>` value. 401 on mismatch with `WWW-Authenticate: Bearer realm="ghost-trace-ingestion"` per [`§0035`](../../charter/decision-log.md) convention. T0 (`/healthz`) remains exempt.

3. **mTLS subject-to-scope mapping (γ, optional upgrade).** New option `WithMTLSScopePolicy(path)` — points at a policy file mapping verified cert subjects to tier scopes. Format: one line per mapping, `<CommonName-or-SAN> <tier> [<tier>...]`. When configured AND the request presents a verified client cert AND the cert's subject is in the policy, the cert-derived scope set takes precedence over bearer-token scope. Cert-only auth (no bearer token) is permitted when the policy fully covers the request's required tier. Per-actor attribution (item 4) draws from the mTLS subject when γ is active.

4. **Per-actor attribution on T3 + T4 substrate writes.** For HTTP-channel T3 + T4 endpoints, the substrate write carries the actor identity via the `IngestionEvent` paired with the substrate-committed record:
   - **T4 (lifecycle ops):** the existing Cat I lifecycle event (BehavioralClusterFormation, BehavioralClusterPromotion, ...) is paired with an `IngestionEvent` via [`§0038`](../../charter/decision-log.md) `AppendPair`. Per-actor attribution lands on that IngestionEvent's identity fields. No new record type required.
   - **T3 (orphan-cleanup):** currently commits NO substrate record (verified at [`services/ingestion/internal/orphan/cleanup.go`](../../../services/ingestion/internal/orphan/cleanup.go) — `Report` is in-memory only). HTTP-channel T3 introduces a new Cat I record type `OrphanCleanupAudit` capturing: the operator-supplied invocation parameters (`dry_run`, `confirm`, `keep_newer_than_seconds`, `max_deletions`); the list of orphan hashes inspected; the subset to be deleted. The audit + its paired IngestionEvent commit BEFORE any blob is deleted; deletion follows. Partial-deletion failure leaves the audit + the surviving blobs recoverable (operator re-runs the deletion against the audit's hash list). CLI-channel orphan-cleanup remains unchanged (no audit record committed) per the local-shell-trust assumption — see Open Question 4.
   - **Per-actor identity source.** (a) Verified mTLS subject (CN + SAN + cert-SHA-256) when γ is active. (b) The matched bearer-token's `token_id` (a new optional field in the per-tier token file: `<token>\n<token_id>\n`) when only α is active. (c) The fallback `unattributed-token-<tier>` literal when only α is active AND no `token_id` is supplied — operationally discouraged (logged as a warning at startup) but not forbidden, to preserve [`§0035`](../../charter/decision-log.md) single-line-token-file backward compatibility.

5. **HTTP T3 + T4 routes (introduced by follow-on PRs, not this RFC).** Once this RFC is accepted, the named follow-ons unblock:
   - **T3:** `POST /v1/admin/orphan-cleanup` mirroring [`cmd/orphan-cleanup`](../../../services/ingestion/cmd/orphan-cleanup/) (with the same `dry-run` + `confirm` + `keep-newer-than` + `max-deletions` safety belts; explicit `confirm=true` query parameter required for non-dry-run).
   - **T4:** 24 endpoints mirroring the Cat III lifecycle CLIs (`POST /v1/hypotheses/<subtype>/{form,promote,demote,dissolve,merge,split}`). Each accepts `application/x-protobuf` with the same canonical-serialization-contract enforcement as [`§0034`](../../charter/decision-log.md) `POST /v1/events`.
   These routes are out of scope for this RFC; they ship under ordinary RFC/PR discipline once the auth model lands.

6. **Configuration validation at handler construction.** `httpapi.New` returns an error (currently panics; change is contained to inception-phase construction) when: (a) per-tier tokens are configured alongside the single-token option; (b) `WithMTLSScopePolicy` is configured without TLS termination; (c) the policy file contains a tier name not in the [`§0094`](../../charter/decision-log.md) classification. Main exits non-zero on construction error.

## Alternatives Considered

Three alternatives evaluated; two retained as composable; one rejected.

- **β — JWT with scope claims.** Standard mechanism with native scope support. Rejected at this RFC: introduces a JWT library dependency; key-rotation policy is itself a follow-on; revocation-list complexity is operationally premature at single-host inception. Admissible-but-deferred per the [`§0035`](../../charter/decision-log.md)-style rationale; reversal trigger is when token revocation becomes load-bearing (long-lived JWTs across a federated operator surface).
- **δ — HMAC request signing with scope-bound keys.** Replay-resistant. Rejected at this RFC for the same reason [`§0035`](../../charter/decision-log.md) deferred it: "operational complexity disproportionate at inception." Admissible-but-deferred; reversal trigger is when persistent attackers with packet capture become operationally relevant.
- **Single-token + ACL list (file-based authorization without per-tier tokens).** Rejected on subordination grounds: this is α with the per-tier-token-file structure replaced by a path-prefix ACL. The [`§0094`](../../charter/decision-log.md) classification IS the natural unit of per-tier authorization; collapsing it back into a path-prefix ACL re-creates the equivalence-class collapse [`§0093`](../../charter/decision-log.md) named as the gap.

## Open Questions

1. **Token rotation under multi-tier.** [`§0035`](../../charter/decision-log.md) deferred online rotation. Multi-tier amplifies the surface (N tokens to rotate). Rotation policy is out of scope at this RFC; named follow-on when operational pressure surfaces.
2. **CLI-driven lifecycle ops per-actor attribution.** CLI invocations of Cat III lifecycle CLIs ([`cmd/promote-*`](../../../services/ingestion/cmd/), etc.) currently record no per-actor attribution beyond the optional `Reason` string. The cross-tier per-actor-attribution requirement at [`§0094`](../../charter/decision-log.md) is scoped to HTTP at this RFC; CLI attribution is a separate scope question (the CLI operator is trusted by virtue of local shell access). Named follow-on.
3. **Tier promotion via cert chain.** A cert chain with intermediate CAs could naturally express tier delegation (the intermediate CA's scope is the maximum scope its issued leaves may claim). Deferred; current γ is flat (one-line subject→tier mapping).
4. **CLI orphan-cleanup symmetry with HTTP T3 audit.** Proposal item 4 introduces `OrphanCleanupAudit` for the HTTP T3 path only; CLI invocations continue committing no substrate audit record. Whether to extend the audit-on-commit discipline to the CLI (for symmetry + uniform forensic record) or keep the local-shell-trust asymmetry is unresolved at this RFC. Named follow-on; orthogonal to the wire-format selection.

## Anti-Patterns to Avoid

- **AP1 — Tier conflation at the handler.** A T4 endpoint accepting a T2 token is a §2.5 violation (operator-read scope confers constitutional-act authority). Mitigation: per-route tier annotation is mandatory; CI lint rejects new routes without a tier annotation.
- **AP2 — Per-actor attribution dropped on retry or partial failure.** Mitigation by tier: **T4** — the lifecycle event AND its paired `IngestionEvent` commit in a single substrate transaction ([`§0038`](../../charter/decision-log.md) `AppendPair`); partial failure rolls both back. **T3** — the `OrphanCleanupAudit` + its paired `IngestionEvent` commit BEFORE any blob deletion (Proposal item 4); audit-commit failure aborts the operation (no deletion); deletion-loop failure mid-flight leaves the audit + surviving blobs recoverable. The audit's hash list IS the recovery contract.
- **AP3 — γ activated without TLS.** mTLS cert verification requires TLS. Configuring `WithMTLSScopePolicy` over plain HTTP is a structural error (no verified cert chain). Mitigation: handler-construction validation (item 6c).
- **AP4 — Backward-compatibility silence.** A deployment upgrading from [`§0035`](../../charter/decision-log.md) single-token to multi-tier may continue using the single-token configuration unaware that T3/T4 routes are now reachable via the same token. Mitigation: the single-token form is treated as the union of all four tiers (preserves [`§0035`](../../charter/decision-log.md) behavior); but the README documents the upgrade path + warns about T3/T4 reachability when new write-side routes ship in the follow-on PRs.

## Migration and Backward Compatibility

- **[`§0035`](../../charter/decision-log.md) single-token deployments continue to work unchanged.** No configuration change required. All currently-shipped routes (T0–T2) retain identical behavior.
- **Multi-tier deployments are opt-in.** Operators opt in by configuring at least one per-tier token; the single-token option is then rejected at construction time (forcing explicit migration rather than silent precedence rules).
- **Existing IngestionEvent records are unchanged.** Per-actor attribution extends the [`§0038`](../../charter/decision-log.md) field set (CN/SAN/cert-SHA already present); new optional `token_id` field added to the IngestionEvent proto. Optional → forward-compatible at the canonical-serialization-contract layer.
- **New `OrphanCleanupAudit` proto.** First new Cat I record type introduced since [`§0042`](../../charter/decision-log.md)-era `NetworkEvent`. Registered via the same dispatch registry the existing Cat I types use; commits via the same `AppendPair` path. Schema authored alongside the T3 endpoint follow-on, not in this RFC.
- **Replay contract.** [`replay-model.md` §Phase 1](../../architecture/replay-model.md) deterministic replay over Cat I records is unchanged. T3 + T4 writes produce Category I lifecycle event + IngestionEvent pairs; Phase 1 replay re-derives the IngestionEvent's content-hash from its payload, same as today. Per-actor attribution becomes part of the payload — same replay invariant.

## References

- [`§0034`](../../charter/decision-log.md) — HTTP interface introduction.
- [`§0035`](../../charter/decision-log.md) — bearer-token model; multi-token/scoped-tokens carry-forward this RFC discharges.
- [`§0036`](../../charter/decision-log.md)–[`§0038`](../../charter/decision-log.md) — TLS / mTLS / IngestionEvent identity threading.
- [`§0093`](../../charter/decision-log.md) — `GET /v1/verify`; named the auth-model gap.
- [`§0094`](../../charter/decision-log.md) — operation-tier classification; α/β/γ/δ candidate space this RFC selects from.
- [`docs/architecture/replay-model.md`](../../architecture/replay-model.md) — Phase 1 replay contract; preserved.

## Decision Record

Resolved at [`decision-log §0098`](../../charter/decision-log.md): **α (multi-token files mapped to tier scopes) composed with γ (mTLS cert subject-name-to-scope mapping)** adopted as the HTTP auth-scope wire format. The discussion-phase recommendation is adopted **unmodified** — no committee extensions; the three structural moves in §Proposal stand as written (multi-tier token files; handler-side tier dispatch; mTLS subject-to-scope mapping as optional γ upgrade) along with the four refinements that landed during the discussion-phase discipline pass (per [`§0097`](../../charter/decision-log.md) precedent — though §0097 was a separate code landing, not part of this RFC's content): Q1 phrasing precision (lifecycle events are Cat I per §2.5 BC5); T3 OrphanCleanupAudit introduction (orphan-cleanup commits no substrate record today; HTTP T3 introduces a new Cat I proto committed via AppendPair BEFORE blob deletion); AP2 split by tier; Open Question 4 added (CLI orphan-cleanup symmetry).

Per [`§0094`](../../charter/decision-log.md): the "Auth-model wire-format RFC" carry-forward is discharged. T3 + T4 HTTP work proceeds under ordinary RFC/PR discipline against the selected wire format.

### Reversal conditions

The selection stands subject to four named reversal conditions; any single condition firing triggers a follow-on RFC reconsidering the selection in scope.

- **R-auth-1 — JWT-warranted token-revocation pressure (β reversal).** Per §Alternatives: when token revocation becomes load-bearing (long-lived JWTs across a federated operator surface). Trigger: explicit RFC characterizing the revocation requirement that file-based multi-token rotation cannot meet. Reversal scope: replace α (or add JWT as a peer mechanism) for token issuance + scope claims; γ unaffected.

- **R-auth-2 — HMAC-warranted replay-resistance pressure (δ reversal).** Per §Alternatives: when persistent attackers with packet capture become operationally relevant. Trigger: explicit RFC characterizing the replay-attack pressure that bearer-token-over-TLS cannot meet. Reversal scope: add δ as per-request signing layer; α + γ unaffected as authentication mechanisms (HMAC is signature, not authentication).

- **R-auth-3 — `cli_actor` proto split (Proposal item 3 reversal trigger).** Per [`§0097`](../../charter/decision-log.md) + this RFC's Proposal item 3: a consumer needs to distinguish mTLS-CN from CLI-actor at read time AND the `channel` discriminator is insufficient. Trigger: explicit RFC characterizing the read-time consumer that cannot disambiguate via `channel`. Reversal scope: proto change adding a distinct `cli_actor` field; α + γ runtime mechanisms unaffected.

- **R-auth-4 — CLI orphan-cleanup symmetry (Open Question 4 reversal).** When the local-shell-trust asymmetry between HTTP T3 (commits OrphanCleanupAudit) and CLI orphan-cleanup (commits no audit) becomes operationally untenable — typically when CLI invocations in production cron jobs require forensic-record symmetry with HTTP. Trigger: explicit RFC extending audit-on-commit to CLI orphan-cleanup. Reversal scope: extends T3 mechanism to a second channel; wire-format selection unaffected.

No reversal condition fires at acceptance. The three named follow-on landings (per [`§0098`](../../charter/decision-log.md) Consequences) — multi-tier token plumbing; T3 OrphanCleanupAudit endpoint; T4 24 lifecycle endpoints — ship under ordinary PR discipline.
