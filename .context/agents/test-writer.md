---
type: agent
name: Test Writer
description: Write comprehensive unit and integration tests
agentType: test-writer
phases: [E, V]
generated: 2026-08-04
status: filled
scaffoldVersion: "2.0.0"
---
## The discipline

**Drive the wire.** `internal/api` tests post real JSON over real HTTP.
v1 lost a provenance chain to a field-name mismatch that every in-memory
test survived.

**Show the test red first.** A test that has never failed has not been
tested. Where a guard protects against a specific past defect,
reintroduce that defect and confirm the failure — several tests here
were verified exactly that way.

**No exception lists in drift guards.** The two-way guards
(`featurestate_test.go`, `reason_codes_test.go`,
`vocabulary_test.go`) compare sets in both directions and deliberately
have no escape hatch. Adding one defeats them.

## Goldens

`internal/api/testdata/golden/` freezes the **bytes** clients receive,
not a re-marshalled map. Only genuinely random values are normalised, in
the raw bytes. Regenerate deliberately with
`go test ./internal/api -run Golden -update` and say so in the PR.

Gate with the sensors in `.context/config/sensors.json`; they are all `make` targets.
