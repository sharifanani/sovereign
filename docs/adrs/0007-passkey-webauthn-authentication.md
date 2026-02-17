# ADR-0007: Passkey/WebAuthn Authentication

- **Status**: Accepted
- **Date**: 2026-02-16

## Context

Sovereign users need to authenticate with the server to establish their identity and authorize WebSocket connections. As a privacy-focused messaging application, the authentication mechanism must be strong, phishing-resistant, and must not create a database of secrets (like password hashes) that could be breached.

Requirements:

- No passwords — eliminate the risk of password reuse, credential stuffing, and password database breaches.
- Phishing-resistant — authentication should be bound to the server's origin.
- Hardware-backed where possible — leverage platform secure enclaves (iOS Secure Enclave, Android StrongBox).
- Good user experience — biometric unlock (Face ID, fingerprint) for daily use.
- Support for account recovery if a device is lost.

Alternatives considered:

- **Password + TOTP**: The traditional approach. Passwords are frequently reused, phishable, and create a high-value breach target (the password hash database). TOTP adds a second factor but is also phishable.
- **OAuth/OIDC (external identity provider)**: Delegates authentication to Google, Apple, etc. This creates a dependency on an external service, which contradicts the self-hosted, sovereign design. Also raises privacy concerns — the identity provider knows when users authenticate.
- **Client certificates (mTLS)**: Strong authentication but terrible user experience. Certificate management is complex and confusing for non-technical users.

## Decision

We will use **Passkey/WebAuthn** as the sole authentication mechanism for Sovereign.

WebAuthn (Web Authentication) is a W3C standard for public-key-based authentication. A passkey is a WebAuthn credential that can be synced across devices (via iCloud Keychain, Google Password Manager, etc.) or hardware-bound (YubiKey, platform authenticator). The server stores only public keys — there is no secret material on the server that could be breached.

On the server, we will use the `go-webauthn/webauthn` Go library. On the mobile client, we will use React Native's native module bridge to access platform WebAuthn APIs.

## Consequences

### Positive

- **No password database to breach**: The server stores only public keys. Even a complete database dump reveals no usable authentication material.
- **Phishing-resistant**: WebAuthn credentials are origin-bound. A credential registered for `sovereign.example.com` cannot be used on `evil-sovereign.example.com`.
- **Hardware-backed security**: On supported devices, the private key lives in a secure enclave and never leaves the hardware.
- **Excellent user experience**: Users authenticate with a biometric (Face ID, fingerprint) or device PIN. No passwords to remember or type.
- **Multi-device passkeys**: Modern passkey implementations sync credentials across devices (e.g., iCloud Keychain), reducing the risk of lockout if one device is lost.

### Negative

- **Platform dependency**: Requires a platform that supports WebAuthn/passkeys. As of 2026, support is broad (iOS 16+, Android 9+, all modern browsers), but very old devices are excluded.
- **Account recovery complexity**: If a user loses all devices and has no synced passkey, account recovery is challenging. Mitigation: support multiple registered passkeys, encourage cross-platform passkey sync, and consider a recovery flow (future work).
- **Implementation complexity**: WebAuthn's ceremony flow (challenge generation, attestation parsing, assertion verification) is more complex than password hashing. Mitigated by using a well-maintained library (`go-webauthn/webauthn`).

### Neutral

- Users can register multiple passkeys (e.g., phone + security key) for redundancy.
- The admin panel (embedded web UI) also uses WebAuthn for authentication, ensuring consistent security across all interfaces.
