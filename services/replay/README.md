# Replay Service

## Constitutional Role

Supports replay across the phase contracts specified in [`../../docs/architecture/replay-model.md`](../../docs/architecture/replay-model.md). Reconstructs historical assertion state from the substrate.

## Status

Not implemented.

## Required Properties

- Deterministic replay for Phase 1 and Phase 2 assertions.
- Reconstructive replay for Phase 3 assertions (with the qualifications described in the replay model).
- Honest contract reporting: a replay request must indicate the phase contract it is fulfilling.
