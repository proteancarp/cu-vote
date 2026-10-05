# ADR 0001: Anonymous Voting Authorization

**Status:** Proposed

## Context

Cu-Vote must authenticate voters and determine election eligibility while preventing stored ballots from being linked back to voter identities.

A normal bearer token, session identifier, JWT, or server-generated opaque authorization is insufficient because the issuing system can observe the same identifier that is later presented when a ballot is submitted.

Cu-Vote therefore requires a boundary between identity/eligibility verification and ballot submission.

The ballot subsystem must be able to establish that:

- voting authority was legitimately issued,
- the authority applies to the current election,
- the authority has not previously been consumed,
- and the submitted ballot matches the authority,

without requiring the voter's identity.

## Decision

Cu-Vote will prototype publicly verifiable blind-signed voting authorizations based on RSA Blind Signatures as specified by RFC 9474.

The design is inspired by the issuance/redemption separation used by Privacy Pass but does not imply that Cu-Vote implements the Privacy Pass protocol itself.

### Election signing keys

Each election receives a dedicated blind-signing key pair.

The private key is available only to the authorization issuer.

The corresponding public key is available to clients and the ballot verifier.

An authorization issued for one election must not verify under another election's public key.

### Authorization message

At final ballot confirmation, the client constructs an authorization message containing conceptually:

```text id="ozbjyb"
protocol_version
election_id
random_nonce
ballot_hash
```

The protocol identifier/domain separator prevents the authorization from being interpreted as a signature for another Cu-Vote purpose.

`ballot_hash` is computed from a canonical representation of the voter's completed ballot.

The client blinds this message before submitting it to the authorization issuer.

### Issuance

The authorization issuer operates in the authenticated identity context.

Before blind-signing a request it must verify that:

1. the voter is authenticated,
2. the voter is eligible for the election,
3. the election currently permits authorization issuance,
4. voting authorization has not already been issued to that voter for the election.

The one-authorization-per-voter/election property must be enforced atomically.

The issuer receives only the blinded message.

It blind-signs the message using the election's private signing key and returns the blinded signature.

The client finalizes/unblinds the signature.

### Redemption

The client sends the following to the ballot subsystem:

```text id="a17lz2"
authorization message
signature
ballot
```

The ballot subsystem must:

1. validate the authorization message format and protocol version,
2. verify the election identifier,
3. verify the blind signature using the election's public key,
4. independently calculate the submitted ballot's canonical hash,
5. require that it matches the signed `ballot_hash`,
6. verify that the authorization has not previously been consumed,
7. atomically consume the authorization and persist the ballot.

The ballot subsystem does not require voter identity.

### Single-use enforcement

Authorization consumption must be enforced by a database uniqueness invariant or equivalent atomic mechanism.

Concurrent requests containing the same authorization must result in at most one accepted ballot.

Application-level read-then-write checks are insufficient.

### Identity separation

The issuance subsystem may persist that a voter has received voting authorization for an election.

It must not persist the final unblinded authorization alongside that identity.

The ballot subsystem may retain information required to verify that a legitimate authorization was consumed but must not receive or persist the authenticated voter identity.

Identity session credentials must not intentionally accompany ballot redemption requests.

### Logging

Authentication/issuance and ballot-redemption logging must be designed to avoid recreating the identity-to-ballot relationship through:

- request bodies,
- authorization values,
- session identifiers,
- IP-address logging,
- precise correlatable timestamps,
- tracing identifiers shared across the identity and ballot boundaries.

Infrastructure-level traffic observation by a malicious host operator remains outside the initial Cu-Vote threat model.

## Rationale

Blind signatures allow an issuer to authorize a client-controlled value without learning the value that will later be presented to the verifier.

Publicly verifiable authorization allows the ballot subsystem to validate authority using only the election public key rather than possession of the issuer's private key.

Binding the authorization to the ballot hash prevents a valid authorization from being reused to submit a different ballot.

Election-specific signing keys and explicit election identifiers provide defense-in-depth against cross-election authorization.

Atomic redemption addresses the concurrent double-submission weakness discovered in CopperVote.

## Alternatives Considered

### Server-generated opaque bearer token

Rejected because the issuer can trivially associate the generated token with the voter and later correlate it with ballot redemption.

### JWT or similar signed application token

Rejected for the anonymity boundary because the issued credential remains visible or reproducible by the identity system and therefore provides authentication without issuance/redemption unlinkability.

### VOPRF-based private-verification token

Potentially suitable, but public verification is preferable for the initial architecture because the ballot verifier does not need access to the issuer's secret verification material.

### BBS anonymous credentials

Potentially useful for richer anonymous credentials and zero-knowledge presentations, but introduce substantially greater protocol complexity than required for the initial one-election/one-authorization requirement.

### Anonymous Rate-Limited Credentials

Potentially valuable for later work involving reusable anonymous credentials and recovery semantics. Current standardization and implementation maturity will be reevaluated before adoption.

## Consequences

### Positive

- Identity and ballot authorization are cryptographically separated.
- The ballot verifier does not need voter identity.
- Verification requires only an election public key.
- Authorization can be scoped to one election.
- Authorization can be bound to a specific ballot.
- Duplicate redemption can be enforced atomically.
- The architecture avoids inventing a new cryptographic primitive.

### Negative

- Client-side cryptographic operations are required.
- Key generation, storage, rotation, and destruction become security-critical.
- Network and logging metadata can still weaken practical unlinkability.
- Deployment becomes more complex than ordinary session-based voting.
- Authorization recovery is difficult because unlinkability intentionally prevents the issuer from identifying the final credential.

## Unresolved Question: Lost Authorization

If authorization has been issued but the client loses its authorization state before successful ballot redemption, blindly issuing another authorization could allow the voter to possess multiple valid credentials.

The initial prototype should minimize this window by issuing authorization only during final ballot submission and by supporting idempotent retries of the same issuance attempt.

Safe recovery after complete client-state loss remains unresolved and must be addressed before this decision is considered final.

## Validation Required

This ADR remains Proposed until the project demonstrates, through a focused protocol prototype and tests, that:

- issuance transcripts cannot be used to identify redeemed authorizations under the selected blind-signature construction,
- election isolation holds,
- concurrent redemption accepts at most one ballot,
- ballot modifications invalidate the authorization,
- issuance retries can be safely made idempotent,
- application/session/logging boundaries do not trivially reconstruct voter-to-ballot linkage,
- selected cryptographic libraries correctly implement the required standardized construction.