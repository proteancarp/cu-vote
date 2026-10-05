# AGENTS.md

## Project

Cu-Vote is an open-source, self-hostable electronic voting system written primarily in Go.

Read these documents before making architectural or security-sensitive changes:

- `docs/PRODUCT.md`
- `docs/SECURITY_MODEL.md`
- `docs/ARCHITECTURE.md`
- `docs/adr/`

Architectural decisions in accepted ADRs take precedence over informal assumptions.

## Current Architecture

Cu-Vote is a modular monolith.

Do not introduce microservices, message brokers, distributed infrastructure, or additional deployment units without an explicit architectural decision that requires them.

The primary backend implementation is Go.

PostgreSQL is the intended production database.

## Core Security Rule

Identity may authorize participation, but identity must not become part of the ballot.

Do not introduce persistent relationships between voter identity and ballot data.

In particular, ballot-domain code must not require or persist identity information such as:

- voter IDs,
- email addresses,
- matric numbers,
- employee IDs,
- authentication session IDs.

Read `docs/SECURITY_MODEL.md` before modifying authentication, authorization, eligibility, voting, logging, auditing, or ballot persistence.

## Cryptography

Do not invent cryptographic primitives or implement cryptographic mathematics from scratch.

Use established specifications and reviewed libraries.

The anonymous voting-authorization protocol is documented in:

`docs/adr/0001-anonymous-voting-authorization.md`

Its status must be respected. Do not silently treat a Proposed ADR as finalized architecture.

## Go Conventions

Prefer the Go standard library unless an external dependency solves a concrete requirement.

Keep packages focused on domain responsibilities.

Do not create global `controllers`, `services`, `repositories`, or `utils` packages merely to imitate layered architectures.

Define interfaces where a consuming package genuinely requires an abstraction, not automatically for every concrete implementation.

Critical integrity guarantees must not depend solely on process-local synchronization such as `sync.Mutex`.

## Commands

Format:

```bash
gofmt -w .
```

Test:

```bash
go test ./...
```

Static analysis:

```bash
go vet ./...
```

Run locally:

```bash
go run ./cmd/cuvote
```

Before considering a code change complete, run formatting, tests, and static analysis.

## Repository Boundaries

Application entry points belong under `cmd/`.

Private application code belongs under `internal/`.

Infrastructure code shared across domains belongs under `internal/platform/` only when it is genuinely platform-level.

Create domain packages when implementation work actually requires them rather than pre-creating empty architectural placeholders.

## Change Discipline

Prefer small, reviewable changes.

Do not combine unrelated refactors with feature work.

When behavior changes, add or update tests.

When an architectural or security assumption changes, update the relevant documentation or ADR as part of the same change.

Do not weaken security invariants merely to simplify implementation.