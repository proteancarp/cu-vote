# Cu-Vote — Product Definition

## 1. Overview

Cu-Vote is an open-source electronic voting platform designed primarily for universities and other organizations that need a credible system for conducting internal elections.

The project originates from CopperVote, a university voting system originally developed as a final-year project. CopperVote demonstrated the core use case but was closely coupled to university-specific identity and deployment assumptions.

Cu-Vote revisits the problem from first principles.

The goal is to provide a voting system that organizations can either:

1. deploy as a standalone, self-hosted application, or
2. integrate into an existing platform through clearly defined interfaces and APIs.

Universities are the primary use case, but the core system must remain organization-neutral so that companies, associations, clubs, professional bodies, and similar organizations can use it.

---

## 2. Problem

Many organizations conduct internal elections without access to voting infrastructure that is simultaneously:

- easy to deploy,
- adaptable to their existing identity systems,
- resistant to duplicate voting,
- capable of protecting ballot secrecy,
- administratively manageable,
- self-hostable,
- and transparent about its security guarantees.

Existing organizations may also already have portals, identity systems, membership databases, or employee directories and should not be required to replace those systems merely to conduct an election.

Cu-Vote aims to provide the voting capability without forcing a particular identity model onto the organization.

---

## 3. Primary Users

### Universities

Typical elections may include:

- student union elections,
- faculty elections,
- departmental elections,
- class representative elections,
- student association elections.

### Organizations

Possible users include:

- companies,
- professional associations,
- clubs,
- cooperatives,
- societies,
- internal committees,
- membership organizations.

---

## 4. Deployment Modes

Cu-Vote supports two conceptual operating modes.

### 4.1 Standalone Mode

Cu-Vote provides the complete election workflow.

This may include:

- voter authentication,
- voter eligibility management,
- election administration,
- contest and candidate management,
- ballot presentation,
- ballot submission,
- result calculation,
- result publication.

The organization hosts Cu-Vote on infrastructure it controls or chooses, such as:

- a VPS,
- a cloud virtual machine,
- a Platform-as-a-Service provider,
- or other compatible infrastructure.

Standalone deployment should not require an organization to already possess its own identity platform.

### 4.2 Integrated Mode

Cu-Vote provides voting capabilities to an existing system.

Examples include:

- a university student portal,
- an employee portal,
- an institutional ERP,
- a membership platform,
- an existing authentication or SSO system.

In this mode, the external system may remain responsible for identifying or authenticating users while Cu-Vote handles election-specific authorization and voting.

Integration must occur through explicit and documented boundaries rather than requiring modifications to Cu-Vote's core election logic.

---

## 5. Core Domain Concepts

Cu-Vote should use organization-neutral terminology.

Initial concepts include:

### Organization

The entity operating one or more elections.

### Election

A defined voting event with its own configuration, eligibility rules, voting period, contests, and results.

### Voter

An entity whose identity can be authenticated and whose eligibility for an election can be determined.

### Eligibility

The determination that a voter is permitted to participate in a particular election.

Eligibility is election-specific.

### Voting Authorization

Authority issued after successful authentication and eligibility verification that permits participation in an election.

A voting authorization must be specific to one election and usable at most once.

### Contest

A question or position within an election for which voters make a choice.

Examples include:

- President,
- Treasurer,
- Department Representative,
- approval/rejection questions.

### Choice

An option available within a contest.

A candidate is one possible type of choice.

### Ballot

The set of selections submitted by a voter during an election.

The stored ballot must not contain information that identifies the voter who cast it.

### Administrator

An authorized operator who manages elections.

Administrative authority should eventually be capable of being scoped rather than assumed to apply to an entire installation.

---

## 6. Core Product Principles

### Election-specific eligibility

Being known to Cu-Vote does not automatically make a voter eligible for every election.

### Election isolation

Authorization or participation in one election must not authorize participation in another.

### Organization-neutral identity

Cu-Vote must not assume that identities are represented by matric numbers, employee IDs, email addresses, or any other specific identifier.

Those belong to identity or eligibility integrations.

### API-first boundaries

Core capabilities should be usable independently of the bundled user interface.

The first-party interface should consume the same clearly defined application boundaries that external systems can integrate with where practical.

### Self-hostability

An organization must be able to operate Cu-Vote without relying on a mandatory proprietary hosted service.

External services may be supported as optional integrations.

### Explicit security guarantees

Cu-Vote must describe what security and privacy properties it provides rather than relying on vague claims such as "secure" or "anonymous."

---

## 7. Initial Functional Scope

The initial system should eventually support:

- organization and administrator management,
- election creation and configuration,
- election lifecycle management,
- contests,
- candidates or other choices,
- voter eligibility,
- authentication or integration with an external identity source,
- election-specific voting authorization,
- anonymous ballot submission,
- prevention of repeated authorization use,
- election opening and closing,
- result calculation,
- controlled result publication.

Exact implementation details remain subject to architecture design.

---

## 8. Initial Non-Goals

The first version of Cu-Vote does not intend to provide:

- national or governmental election infrastructure,
- public political election infrastructure,
- coercion-resistant remote voting,
- protection against a malicious infrastructure operator who modifies the deployed application,
- cryptographically verifiable end-to-end elections unless explicitly introduced by a later design,
- support for every possible electoral system,
- blockchain-based voting merely for the sake of using blockchain technology.

These capabilities may require significantly different threat models and protocols.

---

## 9. Relationship to CopperVote

Cu-Vote is inspired by CopperVote but is not a continuation of its repository or architecture.

CopperVote remains a separate historical project.

Cu-Vote has:

- a separate repository,
- independent Git history,
- a redesigned architecture,
- generalized identity and eligibility concepts,
- stronger voting-integrity requirements,
- and a more explicit ballot-secrecy model.

Concepts learned from CopperVote may inform Cu-Vote, but implementation should not be copied without reevaluating the underlying assumptions.

---

## 10. Project Direction

Cu-Vote should ultimately make this workflow possible:

```text
Authenticate voter
        ↓
Determine election eligibility
        ↓
Issue election-specific voting authorization
        ↓
Cross the identity boundary
        ↓
Validate anonymous authorization
        ↓
Accept ballot
        ↓
Consume authorization exactly once
        ↓
Store ballot without voter identity
        ↓
Calculate and publish results
```

The architectural challenge is to make each boundary explicit, testable, and enforceable.