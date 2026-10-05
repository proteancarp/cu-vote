# Cu-Vote — Architecture

**Status:** Initial Architecture  
**Architecture Style:** Modular Monolith

## 1. Architectural Direction

Cu-Vote will initially be implemented as a modular monolith.

The application will run as a single deployable Go service while maintaining explicit internal boundaries between its major domains.

The goal is to obtain the operational simplicity of a monolith without collapsing the application into tightly coupled route handlers, database models, and utility functions.

Conceptually:

```text
                         Cu-Vote

                     ┌─────────────┐
                     │ HTTP / Web  │
                     │   Boundary  │
                     └──────┬──────┘
                            │
         ┌──────────────────┼──────────────────┐
         │                  │                  │
         ▼                  ▼                  ▼
  Administration       Identity &         Election
                       Eligibility        Management
         │                  │                  │
         │                  ▼                  │
         │          Authorization Issuer       │
         │                  │                  │
         │          identity boundary          │
         │                  │                  │
         │                  ▼                  │
         │             Ballot Box              │
         │                  │                  │
         │                  ▼                  │
         │              Results               │
         │                  │                  │
         └──────────────────┼──────────────────┘
                            │
                         PostgreSQL
```

These boxes represent application/domain boundaries, not independent network services.

Cu-Vote should not become a distributed system merely because it contains distinct security responsibilities.

---

## 2. Why a Modular Monolith

Cu-Vote does not currently require microservices.

A modular monolith provides:

- one deployment artifact,
- simpler self-hosting,
- straightforward database transactions,
- lower operational overhead,
- easier local development,
- easier debugging,
- lower resource consumption,
- and clearer development for an early open-source project.

At the same time, internal boundaries are important because Cu-Vote contains responsibilities that must not accidentally become coupled, particularly voter identity and ballot storage.

A module should own its domain behavior and data access.

Cross-module behavior should occur through explicit application interfaces rather than arbitrary database access or imports into another module's internals.

---

## 3. Deployment Unit

The initial application should build into one Go executable:

```text
cuvote
```

A basic deployment may therefore consist of:

```text
┌───────────────────┐
│     Cu-Vote       │
│    Go binary      │
└─────────┬─────────┘
          │
          ▼
┌───────────────────┐
│    PostgreSQL     │
└───────────────────┘
```

Optional external systems may later include:

```text
SMTP
OIDC provider
institutional identity API
object storage
reverse proxy
```

These must not be mandatory unless required by the chosen deployment configuration.

---

## 4. Major Modules

### 4.1 Election

Owns the definition and lifecycle of an election.

Responsibilities include:

- election creation,
- election configuration,
- contests,
- choices/candidates,
- election schedule,
- election state transitions,
- configuration freezing,
- election public-key association,
- result publication policy.

The election module does not authenticate voters and does not store ballots.

---

### 4.2 Identity

Represents the authenticated identity presented to Cu-Vote.

The identity module answers:

```text
Who is this actor?
```

It must not answer:

```text
How did this actor vote?
```

Standalone deployments may use Cu-Vote-owned authentication.

Integrated deployments may derive identity from an external system.

The core election and ballot domains must not depend on university-specific identifiers such as matric numbers.

An authenticated identity should eventually be represented internally by an opaque subject identifier.

---

### 4.3 Eligibility

Determines whether an authenticated subject may participate in a particular election.

It answers:

```text
Is subject S eligible for election E?
```

Possible eligibility sources may eventually include:

- imported voter registry,
- CSV,
- institutional API,
- employee directory,
- membership database,
- OIDC claims,
- custom integration.

The core voting system should depend on an eligibility contract rather than a particular source.

Eligibility is election-specific.

---

## 5. Authorization Issuer

The authorization module forms the security boundary between known identity and anonymous voting.

It operates while voter identity is still known.

Its responsibilities include:

- confirming authentication,
- confirming election eligibility,
- checking election state,
- enforcing at-most-one authorization issuance per voter/election,
- blind-signing the client's authorization request,
- maintaining election signing keys,
- returning the blinded signature.

It may know:

```text
Voter X received authorization for Election Y.
```

It must not persist:

```text
Voter X received Authorization Token Z.
```

The final unblinded authorization must not be available to this module.

---

## 6. Identity Boundary

The identity boundary is one of Cu-Vote's core architectural constraints.

Before the boundary:

```text
identity is known
eligibility is known
```

After the boundary:

```text
identity must no longer be required
```

Conceptually:

```text
Authenticated Subject
         │
         ▼
Eligibility Check
         │
         ▼
Authorization Issuance
         │
═════════╪════════════════════════
         │     IDENTITY BOUNDARY
═════════╪════════════════════════
         │
         ▼
Anonymous Authorization
         │
         ▼
Ballot Submission
```

No internal convenience feature should casually cross this boundary.

---

## 7. Ballot Box

The ballot module owns ballot acceptance and persistence.

It must not require voter identity.

Its responsibilities include:

- validating authorization structure,
- validating election scope,
- verifying the authorization signature,
- validating ballot structure,
- recomputing the canonical ballot hash,
- comparing the ballot hash to the signed authorization,
- ensuring the authorization has not been consumed,
- atomically consuming the authorization,
- storing the ballot.

A request to the ballot submission boundary should contain conceptually:

```text
election
authorization
signature
ballot
```

and not:

```text
voter_id
email
matric_number
employee_id
identity session
```

---

## 8. Single-Use Authorization

Authorization consumption must rely on a database-enforced uniqueness invariant.

For example:

```text
authorization_fingerprint UNIQUE
```

Ballot acceptance should conceptually occur as:

```text
BEGIN TRANSACTION

    validate authorization

    claim authorization fingerprint
        -> UNIQUE constraint

    persist ballot

COMMIT
```

If two concurrent requests attempt to redeem the same authorization, only one may commit successfully.

This invariant must hold regardless of the number of Go goroutines, HTTP requests, or application instances processing traffic.

Application-level mutexes must not be relied upon for election integrity because multiple Cu-Vote instances may eventually exist.

---

## 9. Ballot Storage

Ballots must not contain voter identity.

A ballot record may contain information such as:

```text
ballot_id
election_id
authorization_fingerprint
ballot_content
```

but must not contain:

```text
voter_id
identity_subject
email
matric_number
employee_number
authentication_session_id
```

Random identifiers should be preferred where sequential identifiers would unnecessarily expose voting order.

Exact timestamps should only be retained if required.

Timing metadata can become a correlation mechanism between authentication activity and ballot submission.

---

## 10. Results / Tally

The results module operates only on anonymous ballots.

Responsibilities include:

- tallying accepted ballots,
- calculating contest results,
- handling ties,
- enforcing result-publication policy,
- exposing results after the appropriate election state.

The tally system must never need access to voter identities.

Conceptually:

```text
Ballot Box
    │
    ▼
Anonymous ballots
    │
    ▼
Tally
    │
    ▼
Results
```

---

## 11. Administration

The administration module coordinates privileged election management.

Responsibilities may include:

- administrator authentication,
- election creation,
- contest management,
- choice/candidate management,
- eligibility configuration,
- election lifecycle transitions,
- result publication,
- key lifecycle operations.

Administrative access to election management must not imply access to identity-to-ballot mappings because no such mapping should exist in normal application state.

---

## 12. Audit

Cu-Vote should maintain administrative and security audit information where useful.

Examples include:

```text
election created
contest added
candidate changed
election frozen
election opened
election closed
results published
administrator changed configuration
```

Audit events must not recreate ballot linkage.

The system should not log:

```text
voter X submitted ballot Y
```

or raw anonymous authorization credentials.

Identity-side and ballot-side logs must be designed so that correlation is not accidentally reintroduced through common identifiers.

---

## 13. Election Lifecycle

The initial conceptual lifecycle is:

```text
DRAFT
  │
  ▼
FROZEN
  │
  ▼
OPEN
  │
  ▼
CLOSED
  │
  ▼
FINALIZED
```

### Draft

Election configuration may change.

Examples:

- contests may be added,
- candidates may be added,
- eligibility configuration may change,
- schedule may change.

No ballots may be accepted.

### Frozen

The ballot definition has been finalized.

Ballot-affecting configuration becomes immutable.

This provides a point at which the election configuration can be reviewed before voting begins.

### Open

Authorization may be issued and ballots may be cast.

Ballot-affecting configuration is immutable.

### Closed

No new authorization should be issued and no new ballots should be accepted.

Tallying and result preparation may occur.

### Finalized

The election result has been finalized according to the configured publication policy.

The precise allowed transitions and recovery procedures require further design.

---

## 14. Standalone Mode

In standalone mode Cu-Vote owns the end-to-end application workflow.

Conceptually:

```text
Cu-Vote UI
    │
    ▼
Cu-Vote authentication
    │
    ▼
Cu-Vote eligibility
    │
    ▼
Authorization issuance
    │
    ▼
Ballot box
```

A built-in voter registry may act as the initial eligibility provider.

This is the closest successor to CopperVote's original operating model without preserving its matric-number-specific assumptions.

---

## 15. Integrated Mode

In integrated mode another system may own identity.

For example:

```text
Student Portal
Employee Portal
Membership Platform
       │
       ▼
authenticated subject
       │
       ▼
Cu-Vote integration boundary
       │
       ▼
eligibility
       │
       ▼
anonymous authorization
       │
       ▼
ballot
```

Cu-Vote should not require an integrating system to understand its internal database structure.

Integration should occur through explicit APIs or adapters.

The exact trust and authentication mechanism for external systems remains to be designed.

---

## 16. Database Direction

PostgreSQL will be the primary database for the initial production architecture.

The modular monolith may use one database while preserving domain ownership of tables.

Conceptually:

```text
PostgreSQL
│
├── administrative/election state
├── identity and eligibility state
├── authorization issuance state
├── anonymous ballot state
└── audit state
```

A single database does not imply that arbitrary joins between domains are acceptable.

In particular:

```text
identity ─────X───── ballot
```

There should be no foreign-key path intentionally connecting voter identity to ballots.

Schema boundaries or database roles may later provide additional defense in depth.

---

## 17. HTTP Boundary

Cu-Vote should expose explicit HTTP boundaries for:

```text
administration
authentication
eligibility / authorization
ballot submission
results
integration
```

The exact URL structure remains undecided.

Identity-bearing session credentials should not be required by the ballot-submission endpoint.

Where cookie-based identity sessions are used, cookie scope should be designed so that identity cookies are not unnecessarily sent to anonymous ballot endpoints.

---

## 18. Client Responsibilities

Cu-Vote's anonymity protocol requires limited trusted client-side behavior.

The client will eventually be responsible for:

- constructing the canonical ballot,
- hashing the ballot,
- generating secure randomness,
- constructing the authorization message,
- blinding it,
- unblinding the issuer's response,
- submitting the anonymous authorization and ballot.

The cryptographic primitive must come from established implementations.

Cu-Vote must not implement cryptographic mathematics from scratch.

Client protocol behavior should eventually be specified independently from the first-party UI so other clients can interoperate.

---

## 19. Go Package Direction

The expected project shape is approximately:

```text
cu-vote/
├── cmd/
│   └── cuvote/
│       └── main.go
│
├── internal/
│   ├── admin/
│   ├── election/
│   ├── identity/
│   ├── eligibility/
│   ├── authorization/
│   ├── ballot/
│   ├── results/
│   ├── audit/
│   └── platform/
│       ├── config/
│       ├── database/
│       ├── http/
│       └── logging/
│
├── migrations/
├── docs/
└── ...
```

This structure is directional rather than frozen.

Packages should be created because the domain requires them, not merely because they appear in this document.

Cu-Vote should avoid creating generic layers such as:

```text
controllers/
services/
repositories/
utils/
```

for the entire application if doing so destroys domain ownership.

Infrastructure abstractions should remain close to the domain that consumes them.

---

## 20. Go Interface Principle

Interfaces should be introduced where a consumer genuinely requires an abstraction.

Cu-Vote should not create interfaces for every concrete type merely for architectural appearance.

Where an interface is appropriate, it should generally be defined by the consuming package.

Examples of likely abstraction boundaries include:

```text
EligibilityProvider
IdentityProvider
MailSender
Clock
ElectionRepository
AuthorizationRepository
BallotRepository
```

The exact interfaces should emerge while implementing concrete use cases.

---

## 21. Transaction Ownership

Application operations that enforce critical invariants must explicitly own their transactions.

Examples include:

```text
issue authorization once
redeem authorization once
open election
close election
finalize election
```

Transaction behavior must not be hidden behind unrelated HTTP handlers or implicit ORM behavior.

Election integrity belongs to the application/domain operation, not to the transport layer.

---

## 22. Concurrency Model

Go's concurrency features may be used where they improve throughput or simplify independent work.

However, correctness must not depend on process-local concurrency primitives.

For example:

```go
sync.Mutex
```

may prevent two goroutines in one process from entering a critical section but cannot protect election integrity when two Cu-Vote instances are running.

Durable invariants such as single-use authorization must therefore be enforced using database transactions, constraints, or equivalent shared coordination mechanisms.

---

## 23. Initial Non-Goals

The initial architecture does not require:

- microservices,
- Kubernetes,
- event sourcing,
- CQRS,
- distributed message brokers,
- blockchain,
- service mesh infrastructure,
- multiple independent databases,
- a generic plugin framework.

Such technologies should only be introduced if a concrete requirement justifies them.

---

## 24. Architectural Evolution

The modular monolith should preserve boundaries that allow future extraction if deployment requirements eventually justify it.

For example:

```text
              Modular Monolith

Identity ─ Authorization ─ Ballot ─ Results
                        │
                        ▼

               Possible future

 Identity Service
       │
 Authorization Service
       │
 Ballot Service
       │
 Result Service
```

Extraction is an option, not an objective.

Cu-Vote should prefer the simplest deployment architecture that can enforce its security model and product requirements.

---

## 25. Architectural Rule

The central architectural rule of Cu-Vote is:

> Identity may authorize participation, but identity must not become part of the ballot.

Every future feature should be evaluated against this boundary.