# ADR-0005: MLS over Signal Protocol

- **Status**: Accepted
- **Date**: 2026-02-16

## Context

Sovereign requires end-to-end encryption (E2E) for all messages — both 1:1 and group conversations. The encryption protocol must provide forward secrecy (compromise of current keys does not reveal past messages) and post-compromise security (the system self-heals after a key compromise).

Requirements:

- E2E encryption for both 1:1 and group messaging.
- Forward secrecy and post-compromise security.
- Efficient group operations (add/remove members, key rotation) that do not scale linearly with group size.
- Based on a well-reviewed, standardized protocol — never roll our own cryptography.
- Server must be unable to read message content (acts only as a delivery service).

Alternatives considered:

- **Signal Protocol (Double Ratchet)**: Excellent for 1:1 messaging, widely proven in Signal, WhatsApp, and others. However, for group messaging, Signal Protocol requires the sender to encrypt the message individually for each member (O(n) per message send). Group key management is complex and bolted on rather than natively supported.
- **Custom protocol**: Categorically rejected. Rolling custom cryptographic protocols is a well-established anti-pattern that leads to exploitable vulnerabilities.

## Decision

We will use **MLS (Messaging Layer Security), as defined in RFC 9420**, for all end-to-end encryption in Sovereign.

MLS is an IETF standard designed specifically for group messaging. It uses a tree-based key management structure (a ratchet tree) where group operations — adding members, removing members, updating keys — are O(log n) rather than O(n). It natively provides both forward secrecy and post-compromise security. 1:1 conversations are simply groups with two members, eliminating the need for a separate 1:1 encryption protocol.

## Consequences

### Positive

- **Efficient group operations**: Adding or removing a member from a group of 1000 users requires ~10 operations (log₂(1000)) rather than 1000. This makes large groups practical.
- **IETF standard**: RFC 9420 has been reviewed by the cryptographic community. Using a standard means future interoperability is possible and security analysis is shared.
- **Unified protocol**: Both 1:1 and group messaging use the same protocol, simplifying the codebase and security analysis.
- **Forward secrecy**: Compromise of current keys does not reveal past messages.
- **Post-compromise security**: After a compromise, the protocol self-heals through regular key updates (Commits), locking out the attacker.
- **Server as delivery service**: The server stores and forwards opaque encrypted blobs. It has no ability to read message content or forge group operations.

### Negative

- **Implementation complexity**: MLS is more complex than the Signal Protocol's Double Ratchet. The tree-based ratchet, Commit/Welcome message handling, and epoch management require careful implementation.
- **Fewer battle-tested libraries**: MLS is newer than the Signal Protocol. While implementations exist (OpenMLS, MLS++), they are less battle-tested than libsignal.
- **Newer standard**: RFC 9420 was published in 2023. While it has been rigorously reviewed, it has less real-world deployment history than the Signal Protocol.
- **Client-side state management**: Each client must maintain MLS group state (ratchet tree, epoch secrets). This state must be persisted securely and synchronized correctly.

### Neutral

- 1:1 chats are modeled as 2-member MLS groups. This simplifies the protocol stack but means 1:1 chats carry the overhead of MLS group machinery. In practice, this overhead is negligible.
- The specific MLS library choice (OpenMLS, or a custom implementation wrapping MLS primitives) is a separate decision that depends on the mobile platform's native capabilities.
