# The six numbers reproduce after Phase 4

**Date** 2026-08-06 · **Target** `make numbers` · **Topology** local,
monolith mode · **Load** idle

Phase 4 changed `services/`: the substrate now stores small payloads in
the `events` row rather than in a content-addressed file (ADR-0009), and
the archive's accounting changed shape (ADR-0010). The repository's own
rule is that `make numbers` runs after any such change, and that the six
numbers reproduce or the prose around them is re-read.

They reproduce.

```
numbers-check: against docs/results/numbers-2026-08-04-0b0af2d1.json
               topology monolith -> monolith

  tier1_playwright_naive             12/12 -> 12/12
  tier2_puppeteer_stealth            12/12 -> 12/12
  tier3_undetected_chromedriver      8/8 -> 8/8
  tier4_synthetic_linear             100/100 -> 100/100
  tier5_humanised_bow1.0             10/10 -> 10/10
  tier6_value_injection              10/10 -> 10/10
  p99 decision latency               0.739ms / 80ms budget
  cold start never blocks            True
  false-positive rate                None

  the six numbers reproduce
```

Nothing moved. That is the expected result and it is worth having
rather than assuming: the storage layer under every archived record was
rewritten, and a content-addressed store that had started answering with
different bytes would have shown up as a detection change long before it
showed up anywhere else.

## What this run added: the load condition

Every number this harness has ever produced was taken against a system
doing nothing else. That was never hidden — the schema calls the latency
a floor — but it was never **recorded**, so an idle baseline and a run
taken under load could be compared and nothing would object.

Phase 4 made that concrete. [The decision curve](decision-under-load-2026-08-05.md)
measured the same p99, from the same code, at **4.7× the published
figure** under sustained concurrency. Both readings are true. They are
not the same measurement.

So `provenance.run.load` now records the condition, and `numbers_check`
treats it exactly as it already treats topology:

- **It refuses to compare across conditions.** The precedent is
  `check_topology`, which already refuses to judge an in-process
  decision against one crossing a network hop, because they are
  *different measurements wearing the same name*.
- **The baseline picker filters on it too.** A picker that ignored the
  condition would hand a loaded run an idle baseline and then the check
  would fail — reporting the picker's mistake as the run's.
- **A manifest with no `load` field reads as `idle`**, which is what
  every run before Phase 4 was. Same treatment `topology` gets for
  manifests published before the split.

## The schema refused the run first

Worth recording, because it is the guard working rather than a
hindrance. The first `make numbers` completed every tier and then
**refused to write the manifest**:

```
numbers.json does NOT satisfy experiments/schema/numbers.schema.json:
  provenance.run: unexpected property 'load'
refusing to write it — the run's data is in results/ and the schema or
the producer is wrong.
```

Seven minutes of browsers, and the output was withheld because the
producer had learned a field the schema had not. That is the correct
order of events: a manifest is what somebody else will cite, and one
carrying a field nothing validates is a manifest nobody can check.

## What is not published here

**No new manifest.** Publishing is a deliberate act that requires a
clean tree, for a good reason — a number produced from uncommitted code
cannot be reproduced by anyone, including its author. None is needed
either: existing manifests read as `idle` and new runs record `idle`, so
they compare without objection.

**The false-positive rate is still `null`.** It governs every other
number, it is calendar-bound rather than effort-bound, and no amount of
work in this phase moves it.
