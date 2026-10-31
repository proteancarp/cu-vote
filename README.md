# Cu-Vote

Cu-Vote is an open-source electronic voting platform written in Go for universities, institutions, associations, and other organizations conducting internal elections. It is being built to give organizations election infrastructure they can inspect, self-host, and adapt to their workflows, with optional integration into existing identity systems.

The intended product supports deployment as a standalone application without an existing identity platform, or integration with organizational portals and identity infrastructure. Its goals include customizable election workflows, election-specific eligibility verification, anonymous ballot submission, and at most one accepted ballot per voting authorization. These voting capabilities are planned, not available today.

**Cu-Vote is in early development and is not ready for real elections.**

## Current status

The repository currently implements:

- A Go executable with an HTTP health endpoint and graceful shutdown.
- Election construction and validation of identifiers, titles, and opening/closing times.
- An in-memory lifecycle state machine: `DRAFT → FROZEN → OPEN → CLOSED → FINALIZED`.
- In-memory contest and choice configuration, with validation and configuration locking after freezing. Freezing requires at least one contest and at least one choice per contest.
- A Create Election application use case that generates an ID through a supplied dependency, validates a new draft election, and persists it through a repository interface.
- PostgreSQL persistence for unconfigured draft election metadata, with duplicate-ID rejection.
- A Get Election application use case that retrieves validated, read-only metadata: ID, title, schedule, and stored lifecycle state.
- Unit tests and opt-in PostgreSQL integration tests.

**Contests and choices are not yet persisted.** Metadata retrieval does not load a complete election definition or establish readiness to freeze or vote. Lifecycle transitions currently operate only in memory and are explicit; the schedule does not automatically open or close an election.

The executable exposes no election-management endpoints and does not yet wire the application use cases or database pool into server startup. It cannot conduct voting.

## Privacy and security principles

The central design rule is:

> Identity may authorize participation, but identity must not become part of the ballot.

The intended workflow separates authentication, election eligibility, authorization issuance, and ballot acceptance. Identity is known when checking eligibility and issuing authority. The ballot subsystem should validate election-specific authority and enforce single use without receiving voter identity or storing a durable voter-to-ballot relationship. Atomic database operations and constraints must eventually enforce at most one accepted ballot per authorization, including under concurrent requests.

This is an intended privacy property, not an implemented anonymity guarantee. The blind-signature authorization protocol remains under investigation in [ADR 0001](docs/adr/0001-anonymous-voting-authorization.md), whose status is **Proposed**. It is neither an accepted design nor implemented functionality. Protocol validation, cryptographic validation, and security review remain necessary before production voting use.

The [security model](docs/SECURITY_MODEL.md) describes the goals and limitations. The initial threat model does not promise secrecy against a malicious operator who modifies the software or host, coercion resistance, or end-to-end verifiability. Logging and timing metadata also need careful treatment to avoid recreating identity-to-ballot links.

## Technology and architecture

Cu-Vote follows a **modular-monolith** architecture: one Go application with focused internal domain boundaries. HTTP uses Go's standard-library `net/http` server. PostgreSQL is the intended production database; the existing persistence code uses `pgx/v5` and its connection pool.

The current source layout is:

```text
cmd/cuvote/                    Executable startup and shutdown
internal/election/             Election, contest, choice, and lifecycle rules
internal/election/application/ Create/Get Election use cases and interfaces
internal/election/postgres/    Election metadata persistence
internal/platform/config/      Environment configuration
internal/platform/database/    PostgreSQL pool management
internal/platform/httpserver/  HTTP server and /healthz
migrations/                    Versioned SQL migrations
```

These packages exist today; startup currently connects only configuration and the HTTP server. See the [architecture document](docs/ARCHITECTURE.md) for the broader direction, including modules that remain to be built.

## Getting started

Install **Go 1.26.6 or later**, as required by [go.mod](go.mod). Clone the repository and start the server:

```bash
git clone https://github.com/proteancarp/cu-vote.git
cd cu-vote
go run ./cmd/cuvote
```

No database configuration is required to start the current server. In another terminal:

```bash
curl --fail http://localhost:8080/healthz
```

Expected response:

```json
{"status":"ok"}
```

This endpoint reports HTTP server health; it does not check database connectivity or election readiness. Stop the server with `Ctrl+C`.

### HTTP configuration

The address is selected in this order, from highest to lowest precedence:

| Configuration | Behavior |
| --- | --- |
| Non-empty `PORT` | Uses `:<PORT>`, overriding `CUVOTE_HTTP_ADDR` |
| Non-empty `CUVOTE_HTTP_ADDR` | Uses the supplied listen address |
| Neither set | Uses `:8080` |

The default binds to all available interfaces. To bind locally on a different port, clear any inherited `PORT`:

```bash
PORT= CUVOTE_HTTP_ADDR=127.0.0.1:3000 go run ./cmd/cuvote
```

Then request `http://127.0.0.1:3000/healthz`.

## PostgreSQL and migrations

PostgreSQL currently stores only election metadata. The repository creates unconfigured drafts and retrieves metadata snapshots; it does not persist contest/choice definitions or lifecycle changes.

Configuration reads `DATABASE_URL`, but the executable does not use it to open a database connection. Migrations are **not applied automatically**. See [migrations/README.md](migrations/README.md) for the standalone migration CLI, schema details, and operational commands.

## Testing

From the repository root:

```bash
go test ./...
go vet ./...
```

With `TEST_DATABASE_URL` unset, PostgreSQL integration tests skip and the suite needs no database. To run them, use a dedicated, disposable PostgreSQL database with a role allowed to create and drop schemas. With Docker installed and running, this example starts PostgreSQL 17 on localhost port 5433, waits for readiness, runs the tests, and removes the container and its anonymous volumes on exit:

```bash
(
  set -e
  docker run --rm -d --name cuvote-postgres-test \
    -p 127.0.0.1:5433:5432 \
    -e POSTGRES_USER=cuvote_test \
    -e POSTGRES_PASSWORD=cuvote_test \
    -e POSTGRES_DB=cuvote_test \
    postgres:17
  trap 'docker rm -f -v cuvote-postgres-test >/dev/null' EXIT

  for attempt in {1..30}; do
    if docker exec cuvote-postgres-test pg_isready -h 127.0.0.1 -U cuvote_test -d cuvote_test; then
      break
    fi
    sleep 1
  done
  docker exec cuvote-postgres-test pg_isready -h 127.0.0.1 -U cuvote_test -d cuvote_test

  export TEST_DATABASE_URL='postgres://cuvote_test:cuvote_test@127.0.0.1:5433/cuvote_test?sslmode=disable'
  go test -count=1 -v ./internal/election/postgres -run Integration
)
```

The example credentials and disabled TLS are for local testing only. Never reuse test credentials or disposable test databases for production.

Integration tests create an isolated schema, apply the actual migration, exercise Create/Get persistence and error cases, test rollback/reapplication, and remove the schema. No pre-applied migrations are needed. An unreachable configured database causes failure, not a skip; tests never fall back to `DATABASE_URL`.

## Planned functionality

Major outstanding work includes:

- Complete election configuration persistence, including contests and choices.
- Durable lifecycle operations and election-management HTTP endpoints.
- Administrative authentication, access control, and privacy-aware auditing.
- Voter authentication, election-specific eligibility, and identity integrations.
- Anonymous authorization, subject to protocol investigation and validation.
- Ballot validation, atomic single-use authorization consumption, and ballot storage.
- Tallying, result finalization, and controlled publication.
- Standalone user interfaces and documented integration APIs/adapters.

This roadmap describes direction without release dates or guarantees.

## Documentation and contributions

- [Product definition](docs/PRODUCT.md): intended users, workflows, scope, and non-goals.
- [Architecture](docs/ARCHITECTURE.md): modular boundaries and implementation direction.
- [Security model](docs/SECURITY_MODEL.md): privacy goals, assumptions, and open questions.
- [Architecture decision records](docs/adr/): proposals and decision status.
- [Repository development instructions](AGENTS.md): conventions and change discipline.

Contributions to implementation, tests, documentation, and security analysis are welcome. Read these documents before proposing architectural or security-sensitive changes, respect each ADR's status, and keep changes focused and reviewable. For Go code changes, run `gofmt -w .`, `go test ./...`, and `go vet ./...`.

## License

Cu-Vote is licensed under the [Apache License 2.0 (Apache-2.0)](LICENSE).
