# PostgreSQL migrations

Apply versioned SQL migrations operationally using the standalone
[golang-migrate CLI](https://github.com/golang-migrate/migrate/tree/master/cmd/migrate)
with its PostgreSQL driver enabled. It tracks applied versions and dirty migration
state. It is an operator tool, not an application dependency. Run from the
repository root against the database configured through `DATABASE_URL`:

```bash
migrate -path migrations -database "$DATABASE_URL" up
migrate -path migrations -database "$DATABASE_URL" version
```

The first migration creates `elections` with a text primary key, non-null title,
opening/closing times, and lifecycle state (default `DRAFT`). Constraints enforce
the stored time range and known lifecycle states. ID/title validation remains in
the domain. No contests, choices, identities, or ballots are stored here.

`timestamptz` represents absolute instants rather than local wall-clock times.
Unlike `timestamp without time zone`, it accounts for UTC offsets. PostgreSQL
stores the instant rather than the original timezone name or offset, with
microsecond precision; reads should compare instants using `time.Time.Equal`.
Sub-microsecond precision is not retained. If conversion collapses a very short
schedule to equal instants, the time-range constraint rejects the insert.

The create repository accepts only unconfigured draft elections, preventing
silent loss of contest/choice configuration. It executes one parameterized INSERT.
PostgreSQL makes that statement atomic and the primary key prevents overwrites,
so no explicit transaction is needed. Duplicate errors retain PostgreSQL SQLSTATE
`23505` and can be inspected as `*pgconn.PgError` through `errors.As`.

Retrieval uses a parameterized SELECT by ID and returns only an application-owned
`GetElectionResult` metadata snapshot. Shared `election.ValidateMetadata` rules
validate the ID, title, and schedule; `election.ParseState` validates the exact
lifecycle state. Values are preserved without trimming. Missing rows return
`application.ErrElectionNotFound` rather than exposing `pgx.ErrNoRows`. Invalid
persisted metadata returns a contextual error wrapping the domain validation
error; it is never repaired automatically. Retrieval does not construct a mutable
Election aggregate, load contests or choices, or establish freeze readiness.
Creation and retrieval have separate application-owned repository contracts.

Rollback of this migration deletes the election table and its contents. Use only
when that data loss is intended:

```bash
migrate -path migrations -database "$DATABASE_URL" down 1
```

Migrations are never run automatically by the domain, application, repository,
or pool component. `cmd/cuvote` does not yet open a database pool.

## Integration tests

Use a dedicated PostgreSQL test database with a role allowed to create and drop
schemas. For example, configure `TEST_DATABASE_URL` with a connection URL for an
existing local test database, then run:

```bash
go test -count=1 -v ./internal/election/postgres -run Integration
```

Each run creates a uniquely named schema, applies the actual up migration there,
tests Create/Get Election persistence, duplicate rejection without overwriting,
missing IDs, valid lifecycle metadata, invalid persisted titles, caller cancellation,
database query errors, and migration rollback/reapplication, then removes that
schema. No pre-applied migrations are required in the test database. Unit tests
validate corrupt metadata including states and time ranges that the SQL CHECK
constraints prohibit; the schema is not weakened for these tests.

Without `TEST_DATABASE_URL`, integration tests skip and `go test ./...` requires
no database. A configured but unreachable database causes a test failure, not a
skip. Tests never fall back to `DATABASE_URL`.

Production configuration reads `DATABASE_URL` through `config.Load()` without
connecting or requiring the variable to be present. When wiring database startup,
pass `cfg.DatabaseURL` and the caller's context to `database.Open`, handle its
error, and close the returned pool on shutdown. An empty URL is rejected by
`database.Open` rather than choosing pgx connection defaults.
