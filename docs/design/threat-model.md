# Threat Model

## Overview

This document enumerates the threat landscape for a Sovereign deployment. Sovereign is a self-hosted messaging application that uses MLS (RFC 9420) for end-to-end encryption and WebAuthn/Passkey for authentication. The security model is designed so that the server operator does not need to be trusted with message content.

---

## 1. Trust Boundaries

### Client to Server (WebSocket, TLS)

```
┌─────────────────┐         TLS / WebSocket          ┌─────────────────┐
│  Mobile Client   │◄───────────────────────────────►│  Sovereign Server│
│  (trusted)       │                                  │  (semi-trusted)  │
└─────────────────┘                                  └─────────────────┘
```

- The client trusts the server to faithfully route messages but does NOT trust it with message content.
- The server authenticates clients via WebAuthn session tokens.
- Transport is protected by TLS. If the user runs the server without TLS (e.g., behind a reverse proxy), the segment between client and proxy is TLS-protected but the segment between proxy and server is plaintext on localhost.
- The client is considered trusted: it holds private keys, performs encryption/decryption, and manages MLS state.

### Server to Database (Local, Same Machine)

```
┌─────────────────┐         Local file I/O           ┌──────────────┐
│  Sovereign Server│◄──────────────────────────────►│  SQLite DB     │
│                  │                                  │  (sovereign.db)│
└─────────────────┘                                  └──────────────┘
```

- The database is a local file on the same machine as the server. There is no network boundary.
- Protection depends on OS-level file permissions. The database file should be readable only by the server process user.
- If the machine is compromised, the database is compromised. This gives the attacker access to encrypted blobs and metadata but not message plaintext.

### Admin to Server (HTTP, Same Binary)

```
┌─────────────────┐         HTTP (localhost or TLS)   ┌─────────────────┐
│  Admin Browser   │◄───────────────────────────────►│  Sovereign Server│
│  (trusted)       │                                  │  (admin API)     │
└─────────────────┘                                  └─────────────────┘
```

- The admin UI is served by the same binary as the messaging server.
- Admin authentication uses the same WebAuthn/Passkey system, with the admin role checked server-side.
- If the admin panel is exposed to the network, it must be protected by TLS. The recommended deployment is to access it from localhost or over a VPN.

### Client to Client (E2E Encrypted, Never Direct)

```
┌──────────┐                                          ┌──────────┐
│  Client A │──── E2E encrypted (via server) ────────│  Client B │
│           │     Clients never connect directly      │           │
└──────────┘                                          └──────────┘
```

- Clients never communicate directly. All messages pass through the server.
- Message content is end-to-end encrypted using MLS. The server handles only encrypted ciphertext.
- MLS provides forward secrecy and post-compromise security.

---

## 2. Assets

| Asset | Location | Sensitivity | Protection |
|-------|----------|-------------|------------|
| **MLS private keys** | Client device only | Critical — compromise allows impersonation and message decryption | Device secure storage (Keychain/Keystore), never transmitted |
| **Message plaintext** | Client device only (in memory during display) | Critical — the core user data being protected | E2E encryption, never stored in plaintext on server |
| **MLS group session state** | Client device | High — contains key schedule for current epoch | Device secure storage, rotated with each epoch |
| **WebAuthn private key** | Client device (hardware-bound if available) | High — compromise allows impersonation | Hardware security module or platform authenticator |
| **Session tokens** | Client device (memory/keychain), server (hashed) | Medium — compromise allows session hijacking | Short-lived, stored hashed on server, transmitted only over TLS |
| **Encrypted message blobs** | Server database | Low (content is encrypted) — but metadata is exposed | Database file permissions, E2E encryption of content |
| **User credentials (public keys)** | Server database | Low — public keys are not secret | Standard database protection |
| **Group membership metadata** | Server database | Medium — reveals social graph | Database file permissions, acknowledged v1 limitation |
| **Server configuration** | Config file and database | Medium — contains server settings | File system permissions |

---

## 3. Threat Actors

### Compromised Server Operator

**Capability**: Full access to the server binary, database, config, network traffic at the server, and runtime memory.

**Motivation**: Curiosity, coercion (legal or otherwise), malice.

**What they can do**:
- Read all metadata: who talks to whom, when, group membership, message sizes.
- Read encrypted message blobs (but cannot decrypt them without MLS keys).
- Modify or delete messages in the database (but clients can detect tampering if message IDs/hashes are verified).
- Deny service by shutting down the server.
- Attempt to inject fake messages (but clients verify MLS group membership and signatures).
- Serve a malicious client update (if the client auto-updates from the server, which is NOT the design — clients are installed from app stores).

**What they cannot do**:
- Read message plaintext (protected by MLS E2E encryption).
- Obtain MLS private keys (never sent to server).
- Forge messages from other users (MLS signatures prevent this).

### Network Attacker (MITM)

**Capability**: Can observe, modify, or inject network traffic between client and server.

**What they can do** (without TLS):
- Read all traffic including encrypted blobs and metadata.
- Modify messages in transit.
- Hijack sessions by stealing tokens.

**What they can do** (with TLS):
- Nothing, assuming proper TLS configuration and certificate validation.

**Mitigation**: TLS is required for production deployments. The client should validate server certificates and optionally support certificate pinning.

### Malicious Group Member

**Capability**: Legitimate member of an MLS group with valid keys.

**What they can do**:
- Read all messages in the group (they are a member, this is expected).
- Share message content outside the group (screenshot, copy — this is an inherent limitation of any messaging system).
- Attempt to send forged messages (prevented by MLS authentication).

**What they cannot do**:
- Read messages from groups they are not a member of.
- Read messages from before they joined (if forward secrecy is maintained through proper epoch advancement).
- Read messages after they are removed (post-compromise security through key rotation on removal).

### Compromised Client Device

**Capability**: Full access to the client app's storage, memory, and keychain.

**What they can do**:
- Read all MLS private keys and group session state.
- Read all decrypted message history stored on the device.
- Impersonate the user on all servers the device is connected to.
- Read future messages until the compromise is detected and keys are rotated.

**Mitigation**: Device-level security (screen lock, full-disk encryption, secure enclave), MLS forward secrecy limits the blast radius, user can revoke sessions from another device or through the admin panel.

---

## 4. Server-Side Visibility

### What the Server CAN See

The server necessarily has access to the following information in order to function:

| Data | Why the Server Sees It |
|------|----------------------|
| **Encrypted message blobs** | The server stores and forwards them. It cannot read the plaintext. |
| **Sender identity** | The server must know who sent a message to route it and enforce authorization. |
| **Recipient identity / group membership** | The server must know group members to route messages to the right WebSocket connections. |
| **Timestamps** | The server assigns `server_timestamp` for ordering and sync. |
| **Message sizes** | The server handles the encrypted bytes and knows their length. |
| **Connection metadata** | IP addresses, connection times, TLS handshake metadata. |
| **User accounts** | Usernames, display names, creation times. |
| **WebAuthn public keys** | Needed for authentication verification. Not secret. |
| **MLS key packages (public)** | Needed for group creation. Contains only public key material. |

### What the Server CANNOT See

| Data | Why Not |
|------|---------|
| **Message plaintext** | Encrypted by MLS on the client before transmission. Server never has the decryption keys. |
| **MLS private keys** | Generated and stored exclusively on client devices. Never transmitted. |
| **MLS group session keys** | Derived on each client from the MLS key schedule. Server does not participate in key derivation. |
| **Passkey private keys** | Stored in the device's hardware security module or platform authenticator. The server only has the public key. |

### Metadata Mitigations (Future)

The following techniques are acknowledged as valuable but deferred beyond v1:

- **Message padding**: Pad all encrypted messages to fixed size buckets (e.g., 256B, 1KB, 4KB, 16KB) to prevent message size analysis.
- **Traffic shaping**: Send dummy messages at regular intervals to obscure real messaging patterns.
- **Sealed sender**: Encrypt the sender identity so the server cannot see who sent a message within a group (requires protocol changes).
- **Private group membership**: Use cryptographic techniques to hide group membership from the server (significant complexity).

---

## 5. Key Compromise Scenarios

### Scenario: Client Private Key Compromise

**Impact**: The attacker can impersonate the compromised user and decrypt messages in all groups the user is a member of, starting from the current MLS epoch.

**Blast Radius**:
- All current group conversations are readable by the attacker.
- Future messages remain readable until the compromise is detected and the user is removed/re-added to groups.
- Past messages from before the current epoch are protected by forward secrecy (the attacker cannot derive old keys from the current key).

**Detection**: Anomalous sign-in from a new device (if the server tracks device metadata), concurrent sessions from multiple locations.

**Recovery**:
1. Revoke all sessions for the compromised user.
2. Remove the user from all groups (triggers MLS Remove + Commit, rotating keys).
3. User registers a new passkey on a clean device.
4. Re-add the user to groups with new key material.

### Scenario: Server Compromise

**Impact**: The attacker gains access to the database and server memory.

**What is exposed**:
- All metadata (user list, group memberships, who talks to whom, timestamps).
- Encrypted message blobs (unreadable without MLS keys).
- Session token hashes (attacker could attempt to brute-force tokens, but they are 256-bit random values, making this infeasible).
- WebAuthn public keys (not useful for impersonation — the attacker needs the private key on the client device).

**What is NOT exposed**:
- Message plaintext.
- MLS private keys.
- Passkey private keys.

**Recovery**:
1. Take the server offline.
2. Investigate the breach and patch the vulnerability.
3. Rotate the server's TLS certificate.
4. Invalidate all sessions (force all users to re-authenticate).
5. Optionally: advance the MLS epoch in all groups to ensure post-compromise security.

### Scenario: MLS Group State Compromise

**Impact**: If an attacker obtains the MLS group state for a specific group (e.g., from a compromised client), they can read messages in that group for the current epoch.

**Blast Radius**: Limited to the specific group and the current epoch. Other groups and past epochs are not affected.

**Recovery**:
1. Remove the compromised member from the group.
2. The MLS Remove + Commit triggers a key rotation, starting a new epoch.
3. Messages in the new epoch are protected with fresh key material.

**Mitigation**: Frequent epoch advancement (e.g., on every member change, or periodically) limits the window of exposure.

---

## 6. Replay Attacks

### Server-Side Mitigation

- The server assigns a monotonically increasing `server_timestamp` to each message upon receipt.
- Each message receives a unique server-generated UUID.
- The server does not re-deliver messages that have already been delivered to a connected client (deduplication by message ID).

### Client-Side Mitigation

- Clients should maintain a set of recently seen message IDs and reject duplicates.
- MLS provides its own replay protection through epoch-based key derivation and message counters. A replayed MLS ciphertext will fail decryption because the MLS message counter will not match.

### WebAuthn Replay Protection

- WebAuthn includes a signature counter (`sign_count`) that increments with each authentication. The server rejects assertions with a counter value less than or equal to the stored value, detecting cloned authenticators.

---

## 7. Metadata Exposure

### Acknowledged v1 Limitations

Sovereign v1 makes the following metadata visible to the server operator. This is an explicit, acknowledged limitation:

| Metadata | Exposure | Risk |
|----------|----------|------|
| **Social graph** | Server knows which users communicate and in which groups | Reveals relationships |
| **Communication patterns** | Server sees when messages are sent | Reveals activity patterns |
| **Message frequency** | Server sees how many messages are exchanged | Reveals conversation intensity |
| **Message sizes** | Server sees the size of encrypted blobs | May reveal content type (e.g., text vs. image) |
| **Online/offline status** | Server knows when users connect/disconnect | Reveals availability patterns |
| **IP addresses** | Server sees client IP addresses | Reveals approximate location |

### Mitigation Strategy (Post-v1)

1. **Message padding**: Normalize message sizes to fixed buckets.
2. **Traffic shaping**: Inject cover traffic to obscure real messaging patterns.
3. **Connection mixing**: Allow connections through Tor or VPN (already possible at the network layer).
4. **Sealed sender**: Encrypt sender identity within group messages.
5. **Private membership**: Cryptographic group membership that the server cannot enumerate.

### Why Metadata Exposure Is Acceptable in v1

Sovereign is a self-hosted application. The server operator is typically the user themselves or someone they trust (a family member, a friend, a small organization). In this deployment model, the server operator already knows the social graph because they invited the users. The metadata exposure is far less concerning than in a centralized messaging service where the operator is a corporation with commercial interests in user data.

---

## 8. Authentication Threats

### Passkey / WebAuthn Specific Threats

#### Passkey Theft

**Threat**: An attacker obtains the user's passkey private key.

**Likelihood**: Low. Hardware-bound passkeys (e.g., YubiKey, Secure Enclave, Android Keystore) are designed to prevent key extraction. Platform authenticators (Touch ID, Face ID, Windows Hello) store keys in hardware-backed secure storage.

**Mitigation**:
- Prefer hardware-bound passkeys over synced passkeys where possible.
- Server cannot mitigate this directly — it is a client-side concern.
- If compromise is suspected, user can revoke all sessions and register a new passkey.

#### Session Hijacking

**Threat**: An attacker obtains a valid session token and uses it to impersonate the user.

**Mitigation**:
- Session tokens are transmitted only over TLS.
- Session tokens are stored hashed (SHA-256) on the server. A database leak does not expose usable tokens.
- Sessions have configurable expiry (default: 30 days).
- `last_seen_at` tracking enables detection of concurrent usage from unexpected locations.
- Users can revoke sessions from the admin panel or by re-authenticating.

#### Credential Stuffing

**Threat**: Not applicable. Sovereign uses passkeys exclusively — there are no passwords to stuff. An attacker cannot attempt to log in with credentials from other breached services.

#### Phishing

**Threat**: An attacker sets up a fake Sovereign server to trick users into authenticating.

**Mitigation**: WebAuthn credentials are scoped to the server's origin (domain). A credential registered on `sovereign.example.com` cannot be used on `evil.example.com`. This is a fundamental property of the WebAuthn protocol and provides strong phishing resistance.

---

## 9. Denial of Service

### Attack Vectors

| Vector | Description | Mitigation |
|--------|-------------|------------|
| **WebSocket connection flood** | Attacker opens thousands of WebSocket connections | Connection limit per IP address (configurable, default: 10). Global connection limit (configurable, default: 1000). |
| **Message flood** | Authenticated attacker sends messages at high rate | Per-user rate limiting (configurable, default: 60 messages/minute). Server-side message queue with backpressure. |
| **Large message attack** | Attacker sends very large messages to exhaust memory/storage | Maximum message size limit (configurable, default: 64 KB for encrypted payload). WebSocket frame size limit. |
| **Key package exhaustion** | Attacker consumes all of a user's key packages, preventing them from being added to groups | Rate limiting on key package fetching. Minimum key package reserve (server refuses to give out the last key package). |
| **Authentication abuse** | Attacker floods registration/login endpoints | Rate limiting on auth endpoints per IP. Challenge expiry (60 seconds). |
| **Database growth** | Attacker sends many messages to grow the database file | Storage quota per conversation (configurable). Periodic cleanup of expired sessions and consumed key packages. |

### Infrastructure-Level DoS

Since Sovereign runs on user-owned hardware (often a home server or Raspberry Pi), it is inherently vulnerable to network-level DoS (bandwidth exhaustion, SYN floods). This is outside the application's control and must be mitigated at the network level (ISP, firewall, router).

**Recommendation**: Users who expose their server to the public internet should consider:
- Running behind Cloudflare or similar DDoS protection service.
- Using a VPN (e.g., WireGuard, Tailscale) to restrict access to known users.
- Firewall rules to restrict access to known IP ranges.

---

## 10. Out of Scope for v1

The following threats and mitigations are explicitly deferred to future versions:

| Item | Reason for Deferral |
|------|-------------------|
| **Federation** | No server-to-server communication in v1. Each server is an isolated island. Federation introduces a large attack surface (trust between servers, identity verification, message routing). |
| **Anonymous registration** | v1 requires a username. Anonymous or pseudonymous registration (e.g., via Tor, with no username) is a future consideration. |
| **Traffic analysis resistance** | Padding, cover traffic, and timing obfuscation are complex and have performance implications. Deferred to post-v1. |
| **Plausible deniability** | MLS provides authentication (non-repudiation). Deniable messaging (where a recipient cannot prove to a third party that the sender wrote a message) requires different cryptographic constructions. |
| **Device verification** | Cross-device verification (e.g., comparing safety numbers/QR codes) to detect MITM on key exchange. Important for high-security deployments but adds UX complexity. |
| **Message franking** | The ability for a recipient to prove to the server that a specific message was sent (for abuse reporting). Not needed in the self-hosted trust model. |
| **Post-quantum cryptography** | MLS currently uses classical cryptographic primitives. Post-quantum key exchange and signatures are an active area of research and standardization. |
| **Secure backup and restore** | Encrypted backup of message history and key material. Complex to implement securely (key management for backups). |
