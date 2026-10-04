# Cu-Vote — Security Model

## 1. Purpose

This document defines the security and privacy properties Cu-Vote intends to provide.

The purpose is not to claim that the system is universally "secure" or "anonymous."

Instead, Cu-Vote should define specific guarantees, assumptions, adversaries, and limitations that can later be verified against the implementation.

---

## 2. Security Goals

Cu-Vote initially targets five primary security invariants.

### 2.1 Eligibility

Every accepted ballot must originate from valid election-specific voting authorization issued only after the voter has been authenticated and determined to be eligible.

An ineligible person must not be able to obtain valid voting authority.

---

### 2.2 Uniqueness

A voting authorization may result in at most one accepted ballot.

This property must remain true even when:

- requests are repeated,
- users refresh or retry,
- multiple requests arrive concurrently,
- application instances process requests simultaneously.

This guarantee must ultimately be enforced transactionally or through an equivalent atomic mechanism rather than relying only on application-level checks.

---

### 2.3 Ballot Secrecy

Persistent ballot data must not contain the identity of the voter who cast it.

The system must not store a normal application relationship equivalent to:

```text
Voter → Ballot
```

A ballot should demonstrate that valid voting authority was used without identifying the voter to whom that authority was issued.

---

### 2.4 Election Isolation

Voting authority must be scoped to a particular election.

Authorization issued for Election A must never be accepted by Election B.

Election-specific state must not accidentally leak authority between elections.

---

### 2.5 Ballot Integrity

Once a ballot has been accepted, normal administrative actions must not silently change or destroy its meaning.

Election configuration must therefore become sufficiently immutable before or when voting begins.

Changes to candidates, choices, contests, or voting rules after this boundary must not invalidate or reinterpret already-cast ballots.

---

## 3. Meaning of Anonymity

Cu-Vote does not define anonymity as merely hiding voter information from the application's user interface.

The intended property is stronger.

After authentication and eligibility verification, the voting process should cross an identity boundary.

The voting subsystem should be able to establish:

> This ballot was submitted using valid election-specific voting authority.

It should not need to establish:

> This ballot was submitted by voter X.

Conceptually:

```text
IDENTITY DOMAIN

Voter identity
      │
      ▼
Authentication
      │
      ▼
Eligibility
      │
      ▼
Voting authorization
      │
════════════════════════
      Identity boundary
════════════════════════
      │
      ▼

VOTING DOMAIN

Authorization validation
      │
      ▼
Ballot submission
      │
      ▼
Anonymous ballot storage
```

The exact mechanism used to cross this boundary has not yet been selected.

---

## 4. Operator Privacy Guarantee

Cu-Vote intends to protect ballot secrecy from ordinary system administrators and operators.

An administrator with legitimate access to:

- the administrative UI,
- election configuration,
- voter records,
- eligibility records,
- the application database,
- stored ballots,
- and election results

should not be able to determine which ballot was cast by a particular voter through normal application data.

The database architecture must therefore avoid storing a durable voter-to-ballot mapping.

---

## 5. Threat Model

### 5.1 Adversaries initially considered

Cu-Vote should defend against reasonable attempts involving:

- an ineligible person attempting to vote,
- an eligible voter attempting to vote more than once,
- replay of previously used voting authority,
- submission of choices that do not belong to the relevant contest or election,
- attempts to use authorization across elections,
- accidental administrative modification of an active election,
- ordinary administrators attempting to identify voter choices from stored application data,
- concurrent ballot submissions attempting to bypass duplicate-voting protection,
- unauthorized use of administrative functionality.

---

## 6. Trusted Components

The initial Cu-Vote model assumes that the software deployed by the organization is the authentic Cu-Vote implementation.

The operating organization is trusted not to maliciously alter the source code, runtime, or host infrastructure for the purpose of surveilling voters.

This is an explicit limitation.

---

## 7. Malicious Host Limitation

Cu-Vote v1 does not initially claim ballot secrecy against an infrastructure operator who deliberately modifies or instruments the running application.

A malicious operator with complete control over:

- the host operating system,
- application binary,
- runtime memory,
- network termination,
- database,
- or deployed frontend

could potentially alter the application to record identity-to-ballot information during voting.

Preventing this requires a substantially stronger architecture and may involve technologies such as:

- separation of authorities,
- client-side cryptography,
- blind signatures,
- anonymous credential systems,
- mix networks,
- homomorphic tallying,
- zero-knowledge proofs,
- independently verifiable client software.

Cu-Vote must not claim these guarantees unless they are explicitly implemented and analyzed.

---

## 8. Authentication Versus Ballot Secrecy

Authentication and anonymity are not contradictory if they are separated appropriately.

Cu-Vote must know enough during the identity stage to answer:

```text
Is this person who they claim to be?
```

and:

```text
Is this person eligible for this election?
```

The ballot subsystem should only need to answer:

```text
Was this submission authorized?
```

and:

```text
Has this authorization already been used?
```

The ballot subsystem should not require the voter's identity.

---

## 9. Voting Authorization

The voting authorization mechanism is a security-critical component that remains to be designed.

It must eventually satisfy at least the following properties:

- issued only after authentication and eligibility verification,
- scoped to exactly one election,
- accepted at most once,
- infeasible for unauthorized parties to forge,
- capable of being validated by the voting system,
- designed so ballot storage does not identify the voter,
- resistant to replay,
- safe under concurrent submission.

A random token alone must not automatically be assumed to provide anonymity.

The architecture must consider whether the issuer can later correlate an issued authorization with the ballot that consumes it.

No custom cryptographic protocol should be invented without appropriate analysis and established primitives.

---

## 10. Election Lifecycle Integrity

Cu-Vote should have explicit election states.

The exact lifecycle remains an architectural decision, but it will likely distinguish concepts such as:

```text
Draft
  ↓
Configured
  ↓
Published / Frozen
  ↓
Open
  ↓
Closed
  ↓
Finalized
```

Before voting begins, Cu-Vote must define the point after which ballot-affecting configuration becomes immutable.

Changes that could alter previously cast ballots must not be silently permitted.

---

## 11. Administrative Security

Administrative access must eventually support:

- authenticated administrators,
- explicit authorization checks,
- scoped privileges where appropriate,
- secure password handling or external authentication,
- CSRF protection where browser sessions are used,
- secure session management,
- auditing of sensitive administrative actions.

Administrative auditability must be designed carefully so that logs do not accidentally create voter-to-ballot correlations.

---

## 12. Data Minimization

Cu-Vote should avoid storing data merely because it might be useful later.

Particularly sensitive data includes:

- voter identity,
- authentication information,
- voting authorization issuance data,
- ballot content,
- timestamps capable of correlating identity activity with ballot submission,
- IP addresses,
- application logs.

Retention of such information must be justified against both operational requirements and ballot-secrecy guarantees.

---

## 13. Auditability Versus Anonymity

Election systems need operational evidence, but excessive logging can weaken anonymity.

Cu-Vote therefore must distinguish between:

```text
Evidence that an authorized ballot was accepted
```

and:

```text
Evidence identifying who submitted that ballot
```

The first may be useful.

The second conflicts with the intended privacy model.

The logging and auditing architecture must respect this distinction.

---

## 14. Security Claims Cu-Vote Must Not Make Yet

Until demonstrated by architecture, implementation, testing, and where appropriate cryptographic analysis, Cu-Vote must not claim:

- complete anonymity,
- end-to-end verifiability,
- coercion resistance,
- receipt freeness,
- protection from malicious host operators,
- cryptographic tally correctness,
- national-election-grade security,
- perfect resistance to traffic or timing analysis.

Claims should remain narrower than the actual guarantees rather than broader.

---

## 15. Security Design Questions Still Open

Before implementation, Cu-Vote must answer:

1. How is anonymous voting authority issued?
2. Can the authority issuer correlate issuance with later ballot use?
3. Should authentication and ballot acceptance run as separate components?
4. What information may legally and operationally be logged?
5. How is voting authority consumed atomically?
6. How is concurrent double submission prevented?
7. How are authorization secrets stored?
8. What database constraints support the invariants?
9. When exactly does an election become immutable?
10. How should authentication adapters interact with eligibility providers?
11. What trust is placed in external identity systems?
12. What election and organization boundaries require isolation?
13. What level of voter-verifiable auditing is appropriate?
14. What recovery mechanisms are allowed after election opening?
15. What happens if a voting authorization is issued but never used?

These questions must be resolved before Cu-Vote's security architecture can be considered stable.