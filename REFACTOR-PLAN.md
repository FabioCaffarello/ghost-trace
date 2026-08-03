# Ghost Trace v2 — Refactor Plan

Derived from `integration-contract.md` v0.1. Planning only; no code, no decisions
recorded as formal artifacts until something runs.

---

## 1. Verdict

The repo is 365 commits, ~338 Go files, 58 protos, 53 CLI binaries, and 27,651
lines of documentation. Almost none of it is on the critical path of the
contract.

The reason is structural and worth stating precisely, because it is the same
inversion the brief identifies: **v1 built a substrate for population-level
inference about sets of actors. The contract asks for per-session scoring of one
visitor in under 100ms.** Those are different products, not different maturity
levels of the same product.

Concretely: `observationcollector.CollectBehavioral` — the only path from stored
behavioural data to a signature — does a full table scan of the event log and one
blob read per event, with no index on `message_type` and no session index at all.
Every consumer above it is an operator-invoked CLI. There is no online path, no
per-session state, and no notion of a request arriving that must be answered.

Three things survive and are genuinely worth keeping. The rest goes.

**Scale of the excision: roughly 85% of the working tree.** I recommend
`git tag v1-archive` before the first commit so nothing is lost and the working
tree stops exerting gravity.

---

## 2. Inventory — keep / adapt / delete

### Keep

| Component | LOC (src/test) | Justification |
| --- | --- | --- |
| `internal/canonical` | 336 / 1897 | Deterministic proto serialization + BLAKE3 content hashing. `evaluation_id` must be durable and a decision must be reproducible from its record — this is exactly that machinery, already tested hard. |
| `internal/substrate` | 413 / 362 | Append-only SQLite + content-addressed blob store. Correct as the *raw event archive* (contract §8.3) and as M4's cold tier. Explicitly **not** the online path. |
| `internal/ingest` | 274 / 281 | Idempotent append keyed by content hash. Telemetry batches arrive out of order and may be retried (§2: "batches are accepted out of order"); hash-idempotency is the right primitive, unchanged. |
| Paired-dimension enforcement | ~80 within `canonical` | The rule "never emit a belief without its evidence measure" enforced at the marshalling boundary. Contract §3 `score`/`confidence` is the same shape with renamed axes. Strongest single artifact v1 produced. |
| Go + protobuf toolchain, `Makefile` | — | Working, boring, no reason to touch. |

### Adapt

| Component | LOC (src/test) | Justification |
| --- | --- | --- |
| `schemas/events/v1/behavioral_*.proto` | 5 files | Closest existing thing to contract §2 — mouse trajectory, keystroke timing, scroll cadence, pointer dwell. **But wire-incompatible**: `offset_ns` vs `t` in ms-since-session-start; no key `class`/`target`; no `src` (mouse/trackpad/touch); no `focus`/`visibility`/`form` types; no envelope, no `seq`. Rewrite against §2; keep the modality-oneof shape. |
| `internal/httpapi` | 5703 / 6032 | The mux, bearer auth, request-id correlation, structured logging and `/metrics` middleware are reusable — roughly 800 lines. The 40+ hypothesis-lifecycle routes are not. Keep the scaffolding, delete the surface. |
| `internal/decision` | 290 / 224 | The *idea* is right and maps directly onto `/v1/decisions`: a `Policy` interface, resolution by ref, and an immutable audit record as the decision's durable handle. The *implementation* is not — it thresholds a single `confidence` float off an `AutomationGroupFormation`, takes no action, has no policy config, and has no `evaluation_id`. Rewrite; keep the interface shape. |
| `internal/observationcollector` | 108 / 135 | Right idea (typed read of one modality), wrong access pattern (full scan). Becomes a session-scoped read against a session index. |
| `infra/docker` | — | Compose + Dockerfile adapt; the CIC-IDS pipeline scaffold and 41MB of `runs/` artifacts go. |
| `.github/workflows` | 4 files | Keep CI; strip the three constitutional-check jobs. |

### Delete

| Component | LOC (src/test) | Justification |
| --- | --- | --- |
| `internal/hypothesis` | 5993 / 7231 | Cat III lifecycle — form/promote/demote/merge/split/dissolve across four subtypes. Models durable claims about *sets of actors across sessions*. The contract scores one session. No path from here to there. |
| `internal/projection` | 3127 / 3161 | Materialized views of hypothesis lifecycle state. Follows `hypothesis`. |
| `internal/replay` | 1869 / 2005 | Replay of formation provenance chains. The contract's replay need (§8.3, session replay in a dashboard) is a different and far simpler thing. |
| `internal/signatures` | 2005 / 2241 | Seven signatures, all cross-actor clustering over TLS/network/CIC-IDS data. `keystroke_timing_clustering_v1` is the only behavioural one and it answers *"is this the same operator across accounts?"* — not *"is this a human?"*. Different question. Salvage the 50ms quantization note; delete the code. |
| `internal/{morphology,attribution,derivation,orphan,verify}` | 1580 / 2827 | All serve the hypothesis/provenance apparatus. |
| `cmd/*` — 53 binaries | 17,442 | Operator-invoked batch tools over the deleted machinery. M1 needs one server binary. |
| ~40 of 58 protos | — | Cat III lifecycle + TLS/network observation types. **The browser-fingerprint protos (`browser_canvas_fingerprint`, `browser_webgl_fingerprint`, `browser_font_enumeration`, `browser_header_order`) are explicitly out of scope per contract §0** — they identify the browser, not the behaviour, and contradict the thesis. Delete outright rather than park. |
| `docs/` | 27,651 | Charter (frozen v0.7), 10,776-line decision log, 44 RFCs, ontology, architecture. This is the v1 failure in physical form. **Archive to a tag, do not carry forward** — except `docs/v1-retrospective.md`, extracted first and kept (see M0). |
| `.claude/` skills, agents, hooks | 3,356 | Infrastructure enforcing constitutional discipline over documents that no longer exist. |
| `infra/{terraform,k8s}`, `infra/docker/{runs,seed}` | 41MB | Empty scaffolds and run artifacts. |
| `services/{assertion-engine,graph,projections,replay}` | — | Empty directories. |
| `services/ingestion/bin/` | — | Committed binaries. |

---

## 3. The epistemic model — argued both ways

The brief asks me to argue the case. Here it is.

### Against keeping it

The four-layer model did not *cause* the v1 failure, but it invited it. Every
layer boundary is a place where a document about the boundary can be written, and
27,651 lines say that invitation was accepted. Category separation is a claim
that requires enforcement machinery — marshalling validators, paired-dimension
checks, frozen-section hooks, a `charter-guardian` skill — and all of that is
pure overhead until something runs.

More damning: **the contract never asks for it.** It asks for `score`,
`confidence`, `reasons[]`, `evidence`, and an `evaluation_id`. You could satisfy
every line of the contract with a flat pipeline and no categories at all. A
system with no boundaries to govern would have shipped a year ago.

### For keeping it

Two contract requirements cannot be met without the distinction.

**§3, `/v1/outcomes`:** *"the evaluation record must carry enough feature state to
be reinterpreted without the original events."* Reinterpretation requires knowing
which parts of the record are facts that replay identically and which are
judgements that a recalibrated model will revise. That line is exactly
observation-vs-inference. Without it, retuning a feature extractor silently
rewrites history, and every label collected before the change becomes
uninterpretable — which destroys the calibration loop that the contract calls the
difference between a detector and a platform.

**§3, `score` vs `confidence`:** a system that collapses belief and evidence into
one number cannot answer cold-start, and the contract calls this "the most
important decision in the document." v1 already enforces this pairing at the
marshalling boundary. That enforcement transfers directly.

Secondarily, **§7** requires `reasons[].code` to be a stable enumeration whose
meaning may not change — that is versioned operational definitions, the Cat II
discipline, restated.

### Verdict

**Keep the distinction. Delete the apparatus.**

The mapping the brief proposes is right for three of four layers and wrong for
one, and the wrong one is where all the code is:

| Contract | v1 category | Fit |
| --- | --- | --- |
| raw events | Cat I observation | Clean. Immutable, replayable, never revised. |
| features | Cat II operational construct | Clean, and genuinely good — features *are* deterministic functions of raw events under a versioned definition. |
| signals | Cat III hypothesis | **Broken.** Cat III is a durable entity with a lifecycle: formed, promoted, demoted, merged, split, dissolved. That is correct for "this set of actors is an automation group" — a claim that outlives any session and must be retired when wrong. It is wrong for "this session's pointer path is linear" — a per-session scalar with no lifecycle, which dies when the session does. |
| decision | Cat I decision audit | Clean, and worth keeping deliberately: an immutable record carrying feature state at decision time is precisely what `/v1/outcomes` needs when the chargeback lands ten weeks later. |

The operative rule: **a category boundary earns its keep when something on the
other side of it gets revised.** Features get recalibrated. Policies get retuned.
Observations never change. That is three struct types and a naming discipline in
one package — call it 200 lines — not 6,000 lines of lifecycle events across 24
protos and 53 CLIs.

Cross-session automation groups are a real product. They are v2.

---

## 4. Gap analysis — what the contract requires and nothing provides

| Contract surface | Status |
| --- | --- |
| Browser SDK | **Nothing.** No JS/TS in the repo at all. |
| Demo page | **Nothing.** |
| `POST /v1/sessions` — token issuance, expiry, server-driven `collect` policy | **Nothing.** |
| `POST /v1/telemetry` — envelope, `seq` reordering, batch acceptance | **Nothing.** `/v1/events` is one proto per request, operator-authenticated, synchronous. |
| `POST /v1/decisions` | **Nothing matching.** `decide-from-automation-group` takes a formation hash, not a session token, and runs from a CLI. |
| `POST /v1/outcomes` | **Nothing.** No label channel exists. Without it §9's numbers are uncomputable. |
| Per-session feature state | **Nothing.** Full-scan batch only. This is the central gap. |
| Feature extractors — curvature, acceleration, dwell, interval variance, digraph timing, scroll dynamics | **Nothing.** The signatures cluster actors; they do not score sessions. |
| Tenancy — `site_key` / `secret_key` | **Nothing.** Auth is a single global bearer token with tier classes. |
| Policy as configuration | **Nothing.** Thresholds are hardcoded Go constants. |
| Operating modes — `monitor` / `enforce`, `shadow_decision` | **Nothing.** |
| Fail-open + client timeout semantics | **Nothing.** |
| Normalization inputs — pointer type, touch, tz offset, reduced-motion | **Partial.** Mouse trajectory carries viewport only. |
| Adversarial harness — 4 tiers | **Nothing.** |
| §9 metrics | **Nothing.** `/metrics` counts HTTP requests, not detection outcomes. |

---

## 5. Answers to §8

### §8.1 — p99 budget for `/v1/decisions` — **APPROVED: 80ms**

**p99 = 80ms, measured server-side at ingress, excluding network. Client timeout
250ms (≈3×), then fail-open.**

Reasoning:

- **Market bar.** reCAPTCHA Enterprise assessment lands ~50–100ms; Castle and Sift
  publish ~100–200ms. 80ms server-side puts you inside ~100ms wall-clock for a
  co-located caller, which is defensible without being heroic.
- **Caller budget.** The call site is a login POST. A typical login is 200–500ms
  p99 end-to-end, of which Argon2/bcrypt alone is 50–100ms. A 150ms risk call is a
  30–50% regression on a user-visible path and will get switched off. 80ms is
  absorbable.
- **What makes 80ms the *discriminating* number.** A remote OLAP round trip is
  20–80ms p99 under load — it does not fit inside 80ms with a policy evaluation on
  top. An in-memory read of maintained session state plus a policy evaluation fits
  with room to spare. 80ms is the threshold that separates those two
  architectures, which is exactly what §5 says this number is for. 100ms would
  leave room for a store hop and quietly weaken the argument; 50ms would force
  in-process-only and add engineering risk to M1 for no product gain.

**One flag, and it is the most important thing in this document.**

Contract §5 asserts that a tight p99 means features *cannot* be computed on call.
That is true at scale and for long sessions. It is **not** true for v1's actual
workload: an 8-second login session at 20Hz is ~160 pointer points and perhaps 50
key events. Computing curvature, acceleration and interval variance over that is
microseconds of CPU. The cost is *fetching* the events, not computing on them.

So the honest position is that continuous feature maintenance is not forced by
arithmetic at v1 volume. Picking an artificially tight budget to force stream
processing would be the v1 inversion running in reverse — letting the desired
architecture dictate the requirement. 80ms is chosen because it is defensible on
its own terms against the caller's budget and the market, and continuous
maintenance falls out of it as the natural rather than the coerced answer. That
distinction is what makes the article honest, and it is worth being able to say
out loud.

The real engineering — and the real article — is not hitting 80ms. A single Go
binary with a `map[session]state` hits it trivially. It is what happens when that
map must survive a deploy, span replicas, and hold 100k concurrent sessions.

#### Consequence: the architecture argument moves to concurrency

Accepting the correction relocates the load-bearing claim. If continuous feature
maintenance is not forced by single-request arithmetic, then the case for it rests
entirely on **concurrency** — and that must therefore be measured, not asserted.
This is a strictly better position: a claim about one request is a claim about
CPU, which is boring and roughly the same either way. A claim about ten thousand
concurrent sessions is a claim about memory, GC pressure, and eviction, which is
where the two architectures genuinely diverge.

The divergence is specific, and naming it correctly matters more than naming it
loudly. The two designs scale differently in *different variables*:

- **Maintained state** holds a fixed-size accumulator per session — running sums,
  counts, extrema, a small histogram. Order of ~500 bytes. **Constant in session
  duration.** Cost is `O(sessions)`.
- **Compute-on-call** must retain the raw event buffer until the decision arrives,
  because it cannot know in advance which features it will need. Cost is
  `O(sessions × duration)`.

**Correction to an earlier version of this plan:** it claimed ~200MB for
compute-on-call at N=10,000. That was wrong by 10×. An 8-second session at 20Hz
is ~160 points ≈ 2KB, so 10,000 of them is **~20MB against ~5MB** — a difference
of no operational consequence whatsoever. Concurrency alone does not make the
argument.

**The axis that does is session duration.** Working the arithmetic properly, with
§8.5's post-decision policy (20Hz until the decision, 5Hz after):

| Session duration | Points/session | Compute-on-call @ N=10,000 | Maintained @ N=10,000 |
| --- | --- | --- | --- |
| 10s | ~200 | ~24MB | ~5MB |
| 5min | ~1,650 | ~200MB | ~5MB |
| 30min | ~9,150 | ~1.1GB | ~5MB |

At 10 seconds the two are indistinguishable. At 30 minutes one of them does not
fit in a container. **The divergence is on the diagonal** — concurrency × duration
— which is why M5 #6 becomes a two-dimensional grid rather than a ladder.

This has a consequence that should be stated in the article rather than buried:
**for a login-only deployment, compute-on-call is genuinely fine.** The
architecture earns its keep only because §8.5 commits to collection continuing
after the decision, which is what makes long sessions real. The stream-processing
case and the continuous-authentication case are the same case. If the product were
only ever scoring an 8-second login, the honest recommendation would be to compute
on call and skip the machinery entirely — and saying that plainly is what makes the
rest of the argument credible.

### §8.2 — Pointer decimation — **APPROVED as default; the measurement is M4**

**Ship 20Hz fixed-rate for M1** (matches the contract's own `collect` default,
simplest possible SDK). **Make the algorithm a `collect` policy field** so it is
swappable server-side without an SDK release, which §3 already requires.

**Target for M4: Ramer–Douglas–Peucker, ε ≈ 2px, capped at 40Hz, with a forced
sample every 100ms so idle periods still register.**

Fixed-rate decimation destroys precisely the signal the model depends on:
direction changes and click-approach deceleration are where the pointer moves
fastest through state-space, so uniform sampling under-represents them. RDP keeps
points where the path bends and drops them on straight runs — curvature-preserving
by construction.

There is a second-order effect worth measuring: **under RDP, a linear-mouse bot
compresses to almost nothing.** The compression ratio is itself a discriminative
feature that fixed-rate sampling cannot produce. That is the standalone article
the contract anticipates, and the harness from M2 makes it measurable.

### §8.3 — Raw event retention — **APPROVED as default**

- **Raw events: 7 days.** Enough for dashboard session replay and red-team
  debugging; short enough to be a defensible privacy posture under GDPR/LGPD.
- **Evaluation record + feature state: 13 months.** Chargebacks land 120+ days
  out; 13 months covers an annual review cycle.

This falls straight out of §3's requirement that `evaluation_id` outlive raw
events, and it is the reason the evaluation record must be self-contained.
Flag it only if you have a specific compliance regime in mind.

### §8.4 — Multi-tenant from day one — **APPROVED: yes, but shallow**

`tenant_id` on every record and every query from the first commit. Adding the
column now is near-free; retrofitting it is a migration of every store, every
index and every query in the system.

**Shallow** means: no tenant provisioning flow, no per-tenant config UI, no
billing, no storage or compute isolation. One tenant exists (the demo) and the
second costs an INSERT.

### §8.5 — Collection after the decision — **APPROVED: yes, at reduced rate**

Continue collecting and continue scoring; issue no new `decision` unless the
application asks for one.

Stopping collection makes the outcome channel far weaker — if you stop at the
login decision you can never observe the post-login behaviour that would
corroborate or contradict the label, which is most of the signal for
`fraud_confirmed` and `user_appealed`. And it is nearly free: the SDK is already
resident.

What must be bounded is volume. After the first decision, drop `pointer_hz` to 5
and `batch_ms` to 10000. This is exactly why `collect` is server-driven — no SDK
change required, and the red team cannot infer that a decision was made from a
change in client behaviour, because the server controls the timing.

### §8.6 — `score`/`confidence` per-action or per-session — **APPROVED**

**State is per-session. Scoring is per-action.**

There is one pointer trace and one typing cadence in a session, so feature state
is naturally session-scoped and maintained once. But `score` and `confidence` must
be computed per action, because the action determines which features are relevant
and how they are weighted. A `login` weights keystroke cadence heavily — there was
typing. A `checkout` weights pointer and scroll — there may have been no typing at
all.

Confidence *especially* must be per-action, for exactly the reason the contract
gives. A session with 200 pointer points and zero keystrokes has high confidence
for a pointer-driven action and near-zero for a typing-driven one. A single
session-level scalar throws that away and produces the worst possible failure:
confident-looking blocks on evidence that does not bear on the action taken.

This also resolves cold-start cleanly and in one line: **confidence is a function
of evidence available × evidence relevant to this action.** `reasons[]` becomes
per-action for free, since it is the decomposition of a per-action score.

### Sign-off summary — all resolved

| Decision | Disposition |
| --- | --- |
| §8.1 p99 = 80ms | **Approved.** Contract §5 correction accepted; architecture argument relocated to concurrency and measured at M5 #6. |
| §8.6 per-session state, per-action scoring | **Approved without reservation.** |
| Docs archive | **Approved with condition:** extract `docs/v1-retrospective.md` before deleting. See M0. |
| M2 human traffic | **Replanned.** Repeated-measures design, smaller N, FPR always as an interval. See M2. |
| §8.2 decimation | Default now, measure at M4. |
| §8.3 retention 7d / 13mo | Default. |
| §8.4 shallow multi-tenant | Default. |
| §8.5 continue at reduced rate | Default. |

---

## 6. Milestones

Each milestone is demonstrable. None is a framework.

### Delivery framing — M2 is the public milestone

Budget: **20h/week, public artifact in 4–6 weeks.** Nothing is cut from the plan;
what changes is where the line marked *ready to show* sits.

**M0 + M1 + M2 is the public milestone.** Vertical slice, adversary, first
numbers. The M2 write-up is the external deliverable. M3–M5 continue the series
in public, each with its own artifact.

M5 remains the **definition of done for the project**. M2 is the definition of
done for *going public*. Those are different lines and conflating them is what
keeps projects private for a year.

| Week | Work | Hours |
| --- | --- | --- |
| 1 | M0 — retrospective, tag, delete | ~8h (one day, timeboxed) |
| 1–3 | M1 — slice: demo page, SDK, 3 endpoints, one feature | ~35h |
| 3 | **Recruitment opens** — the M1 demo page *is* the capture instrument | ~2h |
| 3–5 | M2 — four bot tiers, harness, statistics | ~25h |
| 5–6 | M2 write-up | ~10h |

**The scheduling constraint that matters: human capture is calendar-bound, not
effort-bound.** Recruiting 20 people to run 3–5 sessions takes days of waiting no
matter how many hours are worked that week. It is the one item on the critical
path that more effort cannot accelerate.

So it starts the moment M1 renders a page — which is the whole reason M1's demo
page and M2's capture instrument are the same artifact. Recruitment then runs in
the background for the two weeks while the bot tiers are built, and the two arrive
together. Building the harness first and recruiting afterwards adds two idle weeks
to a six-week budget.

**Contingency, decided now rather than under deadline pressure:** if Arm B is
short of ~20 people at week 5, the write-up **ships anyway** with the person count
it has and states it plainly. Per the statistics above the headline is already a
person-level count with an interval, so a smaller sample widens the interval
rather than invalidating the piece. Waiting for a better number is how a six-week
public milestone becomes a six-month one, and the bot-side results — which are
tight, reproducible, and unlimited — carry the article regardless.

### M0 — Excision — **timebox: one day**

Three steps, in order:

1. **Write `docs/v1-retrospective.md`** — two pages, drafted while the code is
   still in front of me, not reconstructed later. What v1 set out to build, why
   the abstraction was wrong, what survived and why. This is also M0's
   communication artifact (§7); it is the only piece of writing in this plan that
   can only be written now, because after the delete commit it becomes an
   archaeology exercise against a git tag.
2. `git tag v1-archive`.
3. One commit that deletes. No new code.

Repo goes from ~570k lines to a Go module with `canonical`, `substrate`,
`ingest`, an HTTP skeleton, and one markdown file.

**Explicitly not in scope for M0** — each of these is tempting during a cleanup
and each is how a one-day timebox becomes a two-week one:

- No package renaming or module restructuring.
- No adding the missing `message_type` / session index to `substrate` — that is
  M1 work, driven by a real query.
- No `go.mod` changes, no dependency upgrades.
- No test refactoring beyond deleting tests for deleted code.
- No rewriting the behavioural protos. That is M1, against §2.

*Done when:* `go test ./...` passes, the tree fits on one screen, the
retrospective exists, and the day is over. Stop regardless of what looks untidy.

### M1 — The vertical slice

Demo login page → SDK → ingest → decision → visible result on the page.

- Static HTML login form with a result banner.
- SDK: `POST /v1/sessions`, pointer capture at 20Hz, batched `POST /v1/telemetry`.
- Server: one Go binary. Session state in an in-process map. `POST /v1/decisions`.
- **One feature** — pointer linearity. **One reason code** — `POINTER_LINEARITY`.
- `monitor` mode. Fail-open. Real `score`; `confidence` present but crude.

Deliberately excluded: persistence of session state, transport, OLAP store,
tenancy beyond a hardcoded `tenant_id`, all other event types.

*Done when:* you open the page, move the mouse, submit, and see a score that is
different when you move the mouse in a straight line — and
`docs/vertical-slice.md` exists (§7).

### M2 — The adversary

Before more features, because features without an adversary are unfalsifiable.

#### The bot side — unchanged

Four tiers, per the brief: Playwright with no humanisation;
`puppeteer-extra-stealth`; `undetected-chromedriver`; a synthetic linear-mouse
script. These are reproducible and unlimited — detection rate per tier is a clean
number with a tight interval, because I can run 10,000 sessions per tier
overnight for free.

The asymmetry between the two sides of this experiment is the point, and it
should be stated rather than smoothed over: **the bot numbers are strong and the
human number is weak, and no amount of engineering fixes that.**

#### The human side — two arms, because the design serves two questions

An earlier version of this plan proposed one pooled sample (8 people × 30
sessions) and assumed ρ=0.2. **Both were wrong, and they were wrong for the same
reason.**

The ρ=0.2 assumption contradicts the argument made two sections below. If false
positives concentrate in individuals with atypical motor patterns — and that is
precisely the claim — then whether a session is falsely flagged is close to a
*property of the person*, not of the session. That is high intraclass
correlation, plausibly 0.5–0.8, not 0.2.

The limiting case is the important one. **As ρ → 1, effective sample size → the
number of people, regardless of how many sessions each contributes.** Eight
people running a thousand sessions each still gives n_eff ≈ 8, and the rule of
three then caps any claim at ~37%. No amount of session volume rescues it.

That single fact reorganises the whole design, because the two questions have
opposite optima:

| Question | Wants | Because |
| --- | --- | --- |
| Does habituation / device / fatigue move the score? | **Few people × many sessions** | It is a within-person contrast; the person is the control. |
| What is the false-positive rate? | **Many people × few sessions** | The person is the unit of variance; people are the sample. |

So the budget splits into two explicit arms rather than one compromise that serves
neither.

**Arm A — condition sensitivity (deep).** 3 people × ~40 sessions ≈ 120 sessions,
systematically crossed:

| Axis | Levels |
| --- | --- |
| Input device | mouse / trackpad / touch |
| Viewport | desktop / mobile |
| Familiarity | 1st encounter / 10th / 30th |
| State | alert / tired / distracted |
| Assistive settings | `prefers-reduced-motion` on / off |

Answers: does the same person's score drift as they habituate? Does switching
trackpad→mouse move it more than switching alert→tired? **Reports effect sizes,
never an FPR.** Arm A is explicitly not evidence about false-positive rate and the
harness will not compute one from it.

The habituation contrast is the one I expect to matter most: the 30th visit to a
form produces fast, ballistic, low-correction movement that looks markedly more
synthetic than the 1st. If that effect is large, it means the system degrades
against exactly its most loyal users — a finding worth more than any single rate.

**Arm B — false-positive breadth (wide).** As many distinct people as can be
recruited × 3–5 sessions each. Target ~20 people ≈ 80 sessions.

This changes the recruitment ask in a way that makes it easier, not harder: **not
"30 sessions from 8 people" but "five minutes each from as many people as
possible."** Five minutes is an ask you can make of a group chat; thirty sessions
is an ask you can make of a friend. Arm B's precision is governed by person count
alone, so trading depth for breadth is strictly correct here.

#### Statistics — the person is the unit of analysis

**The primary reported figure is person-level, not session-level:**

> *k of 20 people were falsely flagged at least once in 3–5 sessions.*

This is honest by construction. It sidesteps the clustering problem entirely
rather than correcting for it, it is directly interpretable, and it cannot be
inflated by running more sessions per person. It is reported with a Wilson
interval over the person count.

Session-level FPR is reported as **secondary**, with two guards:

**1. Rule of three.** With zero events in `n` independent units, the 95% upper
bound is ≈ `3/n`:

| n (independent) | 95% upper bound |
| --- | --- |
| 8 | 37% |
| 20 | 15% |
| 100 | 3% |
| 3,000 | 0.1% |

Production anti-bot systems operate in the last row. This project reaches the
second at best, and the README says so in those words.

**2. ρ is estimated, never assumed.** Computed from Arm B's between-person vs
within-person variance in flag rate, and reported **with its own interval** —
estimating ρ from ~20 clusters is itself noisy, and an unqualified ρ would
reintroduce exactly the false precision this section exists to prevent.

There is no code path in the harness that emits a bare session-level FPR scalar.

#### Why FPR is the weak metric — for the README, stated plainly

Beyond the sample size, there is a directional bias that no interval captures:

**The humans most likely to be falsely flagged are the ones whose motor patterns
differ most from the norm** — tremor, motor impairment, switch or eye-tracking
input, non-dominant-hand use, very old or very young users, assistive overlays. A
volunteer pool drawn from one person's social graph systematically under-samples
precisely that population.

This is also the mechanism behind the high ρ established above: it is the *same
claim*, seen from the statistical side. If false positives were spread evenly
across people, ρ would be low and pooled session counts would be honest. They are
not spread evenly, which is simultaneously why ρ is high, why the person is the
right unit of analysis, and why the sample is biased. One phenomenon, three
consequences.

So the FPR figure is not merely imprecise, it is **optimistic in a predictable
direction**, and the direction is the one with the highest human cost — the
contract's own §9 says blocking a human is far worse than admitting a bot. The
README states this as a limitation of the result, not a footnote, alongside the
telemetry-replay attack from contract §1. A number that is wrong in a known
direction and says so is worth more than a tighter number that pretends
otherwise.

*Done when:* detection rate per bot tier is a number with a tight interval; Arm A
reports condition effect sizes; Arm B reports person-level false-flag counts with
a Wilson interval and an estimated ρ; and the M2 write-up (§7) is published — even
if the numbers are bad. Especially if they are bad.

### M3 — Feature depth and the real `confidence`

Now that the harness can score a change, add:

- Event types: `key` (class + timing only), `scroll`, `focus`, `visibility`,
  `form`.
- Features: digraph timing, keystroke interval variance, scroll dynamics,
  focus-blur patterns, paste/autofill.
- Per-action scoring and real per-action confidence, per §8.6.
- `reasons[].code` as a stable documented enumeration.
- `enforce` mode and `shadow_decision`.

*Done when:* **time-to-confident-decision** is measurable — the §9 number almost
nobody publishes — and `docs/time-to-confident-decision.md` exists (§7).

### M4 — Durability and outcomes

Only now, because until M3 you do not know the shape of the feature state you are
persisting.

- Session state survives restart and spans replicas.
- Evaluation records persisted, self-contained, 13-month retention.
- `POST /v1/outcomes`. Labels start accumulating.
- `tenant_id` threaded end-to-end.
- Latency measured under load against the 80ms budget.

**Transport and store land here, one each:**

- **Transport: NATS JetStream.** Single binary, subject-per-session is a natural
  fit, and its built-in KV gives session feature state without adding Redis — one
  dependency instead of two. Kafka's stronger long-retention replay story does not
  pay for itself when §8.3 sets raw retention to 7 days.
- **OLAP: ClickHouse.** Evaluation records, outcomes, and harness analytics.

Both picks are M4 decisions. Do not install either before M4 — M1–M3 need no
transport at all.

M4 also builds the **compute-on-call variant** for the M5 #6 comparison. It is
roughly a day, and without it the architecture claim stays an assertion.

*Done when:* session state survives a restart, outcomes are arriving, both
architectures run under load, and `docs/concurrency-forces-the-architecture.md`
exists (§7).

### M5 — The numbers

§9 plus the concurrency number is the definition of done. A README that leads
with six numbers, each reproducible by a single harness command:

1. Detection rate per adversarial tier
2. False-positive rate on human traffic — **as a cluster-adjusted interval, per M2**
3. p99 decision latency (single session, idle system)
4. Time to confident decision (events and seconds)
5. Cold-start behaviour on a first visit
6. **p99 decision latency at N concurrent sessions under sustained ingest**

#### Number 6 — a grid, not a ladder

Per the §8.1 correction, concurrency alone does not separate the two
architectures. **Number 6 is a two-dimensional sweep:**

- **Concurrency:** 100 / 1,000 / 10,000 / 50,000
- **Session duration:** 10s / 5min / 30min

Twelve cells, both architectures, same hardware, same ingest load. The headline
cell is **10,000 × 5min**; the finding is the diagonal.

**Why these anchors.** Concurrency is anchored on the workload: a customer at 10M
logins/day peaks near 500/sec, and at ~30s of collection that is ~15,000
concurrent; 1M/day sits near 1,500. Duration is anchored on the product: 10s is a
login-only deployment, 5min is a checkout or account-management flow, 30min is
continuous authentication — the case §8.5 commits to.

**What the grid is expected to show,** stated in advance so it is a prediction and
not a post-hoc reading:

- **Bottom-left (low N, short sessions):** no difference. Compute-on-call wins on
  simplicity.
- **The diagonal:** divergence appears and grows.
- **Top-right (50,000 × 30min):** compute-on-call is predicted to fail outright —
  ~5.5GB of retained buffers against ~25MB.

If the divergence does *not* appear on the diagonal, the architecture is wrong and
the article says so. That is the falsifiable form, and writing the prediction down
before running it is what makes it one.

Concurrency is meaningless without ingest underneath it, so every cell sustains
real load. At 10,000 × 20Hz with 2s batches: ~5,000 batches/sec, ~200,000 points/sec,
with decision calls issued against live sessions rather than a quiesced system.

**Both architectures are benchmarked in every cell.** Number 6 is only an argument
if the comparison exists — a single-architecture latency curve proves the system
is fast, not that the design choice was necessary. The on-call variant is ~a day
at M4 and it converts the central claim from assertion into measurement.

*Done when:* someone can clone the repo, run one command, and reproduce all six —
including the full two-architecture grid.

---

## 7. Communication track

Every milestone ships a written artifact. The artifact is part of the milestone,
not a follow-up — a milestone with green tests and no write-up is not done.

| Milestone | Artifact | What it argues |
| --- | --- | --- |
| M0 | `docs/v1-retrospective.md` | What v1 set out to build, why the abstraction was wrong, what survived. |
| M1 | `docs/vertical-slice.md` | The smallest thing that can score a session. One feature, one reason code, end to end. |
| M2 | `docs/adversary-and-uncertainty.md` | **Priority.** Four tiers, and why FPR is the weak metric of the set. |
| M3 | `docs/time-to-confident-decision.md` | The §9 number almost nobody publishes, and what the curve looks like. |
| M4 | `docs/concurrency-forces-the-architecture.md` | The two-architecture comparison. Stream processing as a consequence of concurrency, measured. |
| M5 | `README.md` | Six numbers, reproducible by one command. |

### Drafted in the moment — the mechanism

"Written during, not after" fails if it is only an intention, so it gets a
mechanism:

- The file is **created on the first day of the milestone** and appended to as
  work happens — a running log of what was tried, what surprised, what broke. At
  the end it is edited *down*, not written *up*.
- It lands in the **same PR series as the code**, never a documentation PR
  afterwards. A separate writing PR is how "later" becomes "never".
- The interesting content is the part that memory destroys first: the thing that
  did not work, the number that came back wrong, the assumption that survived
  three days longer than it deserved. That is unrecoverable a month later, and it
  is the only content that distinguishes this from a feature list.

### Why M2's has priority

Two reasons. It is the falsifiability piece — until it exists, every other number
in the project is unaudited. And it is the hardest one to write honestly: it has
to report a wide interval, a biased sample, and a limitation with real human cost.
That write-up must be drafted **while the disappointment of a wide confidence
interval is fresh**. Written three months later, against numbers that have become
familiar, the wording softens by itself and nobody notices it happening.

### One rule for `docs/`

`docs/` holds **written artifacts about work that has run.** Never specifications,
never governance, never a plan for something not yet built. That single rule is
the difference between this directory and the 27,651 lines being deleted at M0. If
a file in `docs/` describes something that does not execute, it is the failure
mode returning.

`integration-contract.md` is the one standing exception — it is the external
surface and it is upstream of the code by design.

---

## 8. Status

All four adjustments are incorporated. No approvals outstanding.

Ready to start M0 on your word: retrospective first, then `git tag v1-archive`,
then the delete commit — one day, then stop.
