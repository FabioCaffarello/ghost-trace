---
type: agent
name: Event Archive Specialist
description: Design and optimize database schemas
agentType: database-specialist
phases: [P, E]
generated: 2026-08-04
status: filled
scaffoldVersion: "2.0.0"
---
## What you own here

`internal/substrate` — the append-only event archive: a SQLite index
plus a content-addressed blob store. Not a general database practice;
this is one specific durability guarantee.

## The guarantees

- **Append-only.** Writes serialise through a single `Append` entry
  point, so the guarantee has one enforcement site rather than several.
  Reads are concurrent without restriction (WAL).
- **Content-addressed.** Blobs are keyed by BLAKE3 over canonical bytes.
  A recomputed hash that disagrees is `ErrHashMismatch`; a matching hash
  over different bytes is `ErrBlobCollision`.
- **The protobuf runtime is an archive-format pin.** Canonical bytes are
  hashed for identity, so upgrading it is an archive-compatibility
  event, not a dependency bump. A golden hash fixture is the tripwire.

## Before changing anything here

An evaluation record must carry enough feature state to be reinterpreted
months later without the original events (§3). `featurestate_test.go`
enforces that in both directions with no exception list — it exists
because the invariant was broken once, silently.
