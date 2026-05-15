# Infrastructure

This directory contains infrastructure definitions for Ghost Trace deployments.

## Status

Not yet defined. Infrastructure decisions are gated on technology selection via RFC.

## Directory Layout

- [`docker/`](./docker/) — local development environment.
- [`terraform/`](./terraform/) — cloud infrastructure (when applicable).
- [`k8s/`](./k8s/) — Kubernetes manifests (when applicable).

## Open Questions

1. Whether the reference deployment targets a single cloud, multi-cloud, or self-hosted scenario.
2. Whether Kubernetes is the default orchestration target or one of several supported targets.

These will be resolved via RFC.
