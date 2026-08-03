# Ghost Trace — architecture contract

This document is the architectural contract of the system: the external
surface everything else is derived from, and the recorded answers to the
design questions that surface raised. It replaces `integration-contract.md`,
which the code has cited since M1 but which was never committed to the
repository. Section numbers §0–§9 are preserved from that document so that
every existing `contract §N` citation in code and schemas continues to
resolve.

Unlike everything else in `docs/` — written artifacts about work that has
run — this file is deliberately upstream of the code. It is the one
standing exception to the directory's rule.

---

## §0 — Scope

The question is **does this session behave like a human?** — answered from
interaction dynamics alone. Not *which browser is this*: that question is
fingerprinting, its evasions are freely available, and signals that answer
it are out of scope even when they are easy to collect. Canvas, WebGL,
font enumeration and audio fingerprinting are explicitly excluded.

**Collected:** pointer geometry, key *timing* and a coarse six-way class,
scroll displacement and mode, focus transitions, page visibility, form
paste/autofill/submit.

**Never collected:** keystroke content, field values (field identities are
hashed), any persistent client identifier. The session token lives in a
closure and dies with the page. All timestamps are session-relative.

A narrow set of device properties is collected — pointer type, touch,
viewport, timezone offset, reduced-motion. The admission test for a
client property: **does it change how behaviour should be interpreted?**
A trackpad and a mouse produce structurally different traces; comparing
them without normalising measures the input device, not the operator. A
property that merely helps tell one browser from another fails the test.

## §1 — Trust boundary

Everything arriving from the browser is hostile: the SDK runs in the
adversary's runtime. Telemetry is evidence to be weighed, never a claim
to be trusted. `subject_id` and `action` enter only from the application
server, authenticated with the `secret_key`; nothing the browser sends
can bind a session to an identity.

The strongest known attack in this class is **telemetry replay** — record
a genuine human session, replay its event stream under a fresh token.
Partial mitigations raise the cost; none close it. This is stated as a
limitation, not solved.

## §2 — Telemetry wire schema

Telemetry travels in envelopes: `{session_token, seq, sent_at_ms,
page{path, viewport}, events[]}`.

- **Batches are accepted out of order** and may be retried after
  timeouts. `seq` is a client-side monotonic counter; the server tracks a
  high-water mark, never a gap list. Idempotency is by content hash.
- Pointer wire form is `[x, y, dt]` — `dt_ms` is the gap since the
  previous point, so idle time is explicit and meaningful.
- Time is milliseconds since session start. Client wall-clock is
  untrustworthy and leaks more than it gives.
- Key events carry `class` (six-way) and a hashed `target` — never
  content, never key identity.
- Events carry `src` (mouse / trackpad / touch / pen) where the platform
  exposes it, because §0 admits properties that change interpretation.

The canonical schema is `schemas/events/v1/*.proto`; the JSON accepted at
the HTTP boundary mirrors it field-for-field.

## §3 — The external surface

Four endpoints. This section is the contract's core.

| Endpoint | Caller | Auth |
| --- | --- | --- |
| `POST /v1/sessions` | browser | `site_key` (public) |
| `POST /v1/telemetry` | browser | `Bearer <session_token>` |
| `POST /v1/decisions` | app server | `Bearer <secret_key>` |
| `POST /v1/outcomes` | app server | `Bearer <secret_key>` |

`GET /healthz` is unauthenticated liveness.

**`score` and `confidence` are separate fields.** This is the contract's
most important commitment. `score` is belief (how bot-like), `confidence`
is evidence (how much the session has shown that bears on this action).
Collapsing them into one number makes cold start unanswerable: a session
eleven events old can look maximally bot-like and must not be blocked.
Every consumer handles cold start the same way because the contract
answers it, not the integration.

**The evaluation record is self-contained.** It must carry enough feature
state to be reinterpreted without the original events — raw events expire
(§8.3), evaluations outlive them, and a recalibrated policy must be able
to re-judge an old evaluation from its record alone. Every feature the
policy consumes MUST therefore exist in the persisted `FeatureState`.

**`collect` is server-driven.** `POST /v1/sessions` returns
`collect{pointer_hz, batch_ms, types}`; collection policy changes without
an SDK release, and the client cannot infer decision timing from its own
behaviour.

**Outcomes** close the loop: `{evaluation_id, outcome, observed_at}` with
the enumeration `login_success`, `login_failure`, `challenge_passed`,
`challenge_failed`, `fraud_confirmed`, `user_appealed`, `abandoned`.
Without this channel, every number in §9 is uncomputable.

## §4 — Operating modes

`monitor` — score and record, never enforce. `enforce` — decisions carry
consequences. In both, `shadow_decision` reports what the other mode
would have done, so a tenant can watch enforcement before enabling it.

## §5 — Latency

**p99 = 80ms for `POST /v1/decisions`, measured server-side at ingress.**
Client timeout 250ms, then **fail-open**: an unavailable detector admits,
never blocks. The call sits on a user-visible login path whose own budget
is 200–500ms; a risk call that regresses it gets switched off.

80ms was chosen because it separates two architectures: a remote OLAP
round trip does not fit inside it, an in-memory read plus a policy
evaluation does. The original contract asserted that a tight p99 *forces*
continuous feature maintenance; measurement corrected this — at login
volume the compute is microseconds and the real force is **session
duration × concurrency** (memory held, not time to answer). The
correction and the measurement are in
[`duration-forces-the-architecture.md`](duration-forces-the-architecture.md).

## §6 — Demo surface

Non-contract, but load-bearing for the project: `GET /` (demo login
page), `GET /sdk.js`, `POST /demo/login` (a stand-in application server
that calls `/v1/decisions` like any integrator). The demo page is also
the capture instrument for the human study — it cannot break while
recruitment is open.

## §7 — Stability and failure semantics

`reasons[].code` is a **stable enumeration**. Adding a code is a feature;
changing the meaning of an existing one is a breaking change and does not
happen. Consumers may build logic on codes.

Fail-open is uniform: unknown session tokens at `/v1/telemetry` are
accepted and dropped (202 — the token may be expired, the client is not
misbehaving); an unknown token at `/v1/decisions` yields a zero-evidence
judgement, not an error — a missing session is a cold start, and the
confidence dimension already models it.

## §8 — Recorded decisions

Answers to the contract's open questions, decided during the v2 replan
and unchanged since. The measurements that tested them are in `docs/`.

- **§8.1 — p99 budget: 80ms** (see §5). Architecture argument relocated
  from per-request arithmetic to the duration × concurrency grid, and
  measured rather than asserted.
- **§8.2 — Pointer decimation: 20Hz fixed-rate**, as a `collect` policy
  field so the algorithm is server-swappable. RDP (ε≈2px, 40Hz cap,
  forced idle sample) is the measured upgrade path.
- **§8.3 — Retention: raw events 7 days; evaluation records 13 months.**
  This is why §3 requires the evaluation record to be self-contained.
- **§8.4 — Multi-tenant from day one, shallow.** `tenant_id` on every
  record and every key from the first commit; no provisioning flow, no
  isolation machinery until a second tenant exists.
- **§8.5 — Collection continues after the decision, at reduced rate**
  (5Hz pointer, 10s batches, server-driven). Post-decision behaviour is
  most of the signal for `fraud_confirmed`; the long-session consequence
  is what makes the maintained-state architecture real.
- **§8.6 — State is per-session; scoring is per-action.** `confidence` is
  a function of evidence available × evidence relevant to *this* action.
  A session with 200 pointer points and no typing has high confidence for
  a pointer-driven action and near-zero for a typing-driven one.

## §9 — The numbers

The project's definition of done is six numbers, each reproducible by one
command (`python3 experiments/numbers.py`), each published in the README
with its uncertainty:

1. Detection rate per adversarial tier (Wilson intervals)
2. False-positive rate on human traffic — person-level counts, estimated
   ICC, never a bare session-level rate; **blocking a human is far worse
   than admitting a bot**, and the FPR is the governing number
3. p99 decision latency against the 80ms budget
4. Time to confident decision (events and seconds)
5. Cold-start behaviour (a first visit is never blocked)
6. Memory and latency by concurrency × duration, both architectures

Refactoring PRs must not move these numbers. A change that does is either
a bug in the change or a declared recalibration — never an accident.

---

## §10 — Target architecture (forward-looking)

The refactor program in progress evolves the single binary in two phases
without changing the §3 surface:

**Phase 1 — Clean Architecture inside the service.** Domain
(`session`, `feature`, `policy`, `canonical`) ← application use-cases
behind ports (`SessionRepository`, `EventArchive`, `Clock`) ← adapters
(HTTP, in-memory store, substrate archive, proto mapping). Contracts
versioned with buf; generated code moves to an importable module
(`libs/genproto`); the service and the experiment layer become
containerized (`docker compose`); CI gains lint, vulnerability scanning,
image build/publish, and schema breaking-change detection.

**Phase 2 — Physical decomposition.** Four services with single
responsibilities: **collector** (`/v1/sessions`, `/v1/telemetry`; sole
writer of session state, snapshots to NATS KV), **decision-engine**
(`/v1/decisions`, `/v1/outcomes`; stateless, reads KV), **archive**
(consumes the event stream from NATS JetStream into the substrate;
idempotent by canonical hash), **demo-web** (a pure HTTP client of the
public contract — dogfooding it). Internal synchronous calls are gRPC;
the event flow is JetStream subjects partitioned by tenant. The split
ships only if the §5 budget holds in the composed topology, re-measured
by the same experiments that produced §9; the all-in-one binary remains
as the development composition and the cheap rollback.
