---
name: Bug report
about: Something behaves differently from what the contract says
labels: bug
---

## What happened

## What the contract says should happen

<!-- If it is about the HTTP surface, `contract/openapi.yaml` and
     `contract/architecture.md` §3 are the authority. If a number is
     wrong, the manifests in `docs/results/` say what was measured,
     on which commit and with which seed. -->

## Reproducing it

<!-- The seed matters for tiers 5 and 6: every result row carries its
     label (e.g. ghost-trace-v1:tier6_value_injection:3), and one
     session can be replayed on its own. -->

```
GT_SEED=...
```

## Environment

- Go / Node / Python:
- OS and architecture:
- Commit:
