# Sovereign Error Codes

**Version**: 1.0
**Last Updated**: 2026-02-16

All error codes used in Sovereign protocol error messages and REST API responses. Errors are communicated via the `error` message type (WebSocket) or JSON error response bodies (REST API).

---

## Error Format

### WebSocket Error (inside Envelope)

```
Error {
  code:    int32   — Numeric error code from the tables below
  message: string  — Human-readable description
  fatal:   bool    — If true, the server will close the connection
}
```

### REST API Error

```json
{
  "error": {
    "code": 2003,
    "message": "Not authorized as admin"
  }
}
```

---

## 1xxx -- Authentication

Errors related to user authentication and registration.

| Code | Name                | Description                                                                                  | HTTP Equivalent | Fatal |
|------|---------------------|----------------------------------------------------------------------------------------------|----------------|-------|
| 1001 | InvalidCredential   | The provided WebAuthn credential is invalid or does not match any registered credential.     | 401            | No    |
| 1002 | ExpiredSession      | The session token has expired. The client must re-authenticate with a full WebAuthn ceremony.| 401            | Yes   |
| 1003 | RegistrationFailed  | User registration failed. The username may already be taken, or attestation verification failed. | 409        | No    |
| 1004 | ChallengeFailed     | The WebAuthn challenge-response verification failed. The signature is invalid or the challenge has expired. | 401 | No    |
| 1005 | SessionRevoked      | The session was explicitly revoked by an administrator.                                      | 401            | Yes   |

### Details

**1001 InvalidCredential**: Returned when the `credential_id` in `auth.response` does not match any credential registered for the user, or the user does not exist. The client may retry with a different credential or prompt the user to re-register.

**1002 ExpiredSession**: Returned when a client attempts to reconnect with an expired session token. Sessions expire after the configured timeout period (default: 30 days). This is a fatal error -- the WebSocket connection is closed with code `4004`. The client must perform a full WebAuthn authentication.

**1003 RegistrationFailed**: Returned during the registration flow when:
- The requested username is already taken.
- The attestation object is malformed or cannot be verified.
- Server-side registration constraints are violated (e.g., registration is disabled).

**1004 ChallengeFailed**: Returned when the WebAuthn assertion signature verification fails. This may indicate:
- The challenge has expired (challenges are valid for 60 seconds).
- The authenticator data or client data is malformed.
- The signature does not match the stored public key.

**1005 SessionRevoked**: Returned when an administrator explicitly revokes a session via the admin API. This is a fatal error -- the connection is closed immediately with code `4004`. The user must re-authenticate.

---

## 2xxx -- Authorization

Errors related to permission checks after successful authentication.

| Code | Name            | Description                                                                    | HTTP Equivalent | Fatal |
|------|----------------|--------------------------------------------------------------------------------|----------------|-------|
| 2001 | NotGroupAdmin   | The operation requires group admin privileges, but the user is a regular member.| 403            | No    |
| 2002 | NotGroupMember  | The user is not a member of the specified group conversation.                   | 403            | No    |
| 2003 | NotAdmin        | The operation requires server admin privileges.                                 | 403            | No    |
| 2004 | AccountDisabled | The user's account has been disabled by an administrator.                       | 403            | Yes   |

### Details

**2001 NotGroupAdmin**: Returned when a non-admin member attempts admin-only operations such as inviting or removing members. Only the group creator and explicitly promoted admins can perform these actions.

**2002 NotGroupMember**: Returned when a user attempts to send a message to or interact with a group conversation they are not a member of. This includes attempting to read group metadata.

**2003 NotAdmin**: Returned when a non-admin user attempts to access the admin REST API endpoints. This error is used exclusively for REST API authorization.

**2004 AccountDisabled**: Returned when a disabled user attempts to authenticate or when an active session belongs to a newly disabled user. This is a fatal error -- the WebSocket connection is closed with code `4005`. The user cannot reconnect until an administrator re-enables their account.

---

## 3xxx -- Protocol

Errors related to protocol violations, malformed messages, and rate limiting.

| Code | Name              | Description                                                                              | HTTP Equivalent | Fatal |
|------|------------------|------------------------------------------------------------------------------------------|----------------|-------|
| 3001 | MalformedMessage  | The message payload could not be deserialized as the expected Protocol Buffer message.    | 400            | No    |
| 3002 | UnknownMessageType| The `MessageType` enum value in the Envelope is not recognized by the server.            | 400            | No    |
| 3003 | MessageTooLarge   | The serialized Envelope exceeds the maximum allowed size (default: 64KB).                | 413            | No    |
| 3004 | RateLimited       | The client has exceeded the message rate limit. Includes `retry_after_ms` in the message.| 429            | No    |
| 3005 | InvalidEnvelope   | The binary frame could not be deserialized as a valid Envelope.                          | 400            | No    |

### Details

**3001 MalformedMessage**: The Envelope was valid but the `payload` bytes could not be deserialized into the message type indicated by the `type` field. The client should check that it is serializing the correct message type.

**3002 UnknownMessageType**: The `type` field in the Envelope contains a value that the server does not recognize. This may occur when a newer client connects to an older server. The client should not retry the message.

**3003 MessageTooLarge**: The entire serialized Envelope (including type, request_id, and payload) exceeds 65,536 bytes. The client should reduce the payload size. For large file transfers, the client should use external storage and send a reference link in the message.

**3004 RateLimited**: The client's token bucket is empty. The error message includes a `retry_after_ms` field indicating the minimum number of milliseconds the client should wait before sending another message. Repeated rate limit violations within a short period may result in a temporary connection ban.

**3005 InvalidEnvelope**: The binary WebSocket frame could not be parsed as a Protocol Buffer Envelope at all. This typically indicates a serialization bug in the client, use of text frames instead of binary, or data corruption. If this error occurs repeatedly, the server may close the connection as a fatal error.

---

## 4xxx -- Group

Errors related to group conversation operations.

| Code | Name              | Description                                                                    | HTTP Equivalent | Fatal |
|------|------------------|--------------------------------------------------------------------------------|----------------|-------|
| 4001 | GroupNotFound      | The specified conversation ID does not exist or does not refer to a group.     | 404            | No    |
| 4002 | AlreadyMember      | The user being invited is already a member of the group.                       | 409            | No    |
| 4003 | NotMember          | The specified user is not a member of the group (for removal operations).      | 404            | No    |
| 4004 | CannotRemoveSelf   | Use `group.leave` instead of attempting to remove yourself from a group.       | 400            | No    |
| 4005 | GroupFull          | The group has reached the maximum number of members (configurable, default: 256). | 409         | No    |

### Details

**4001 GroupNotFound**: The `conversation_id` provided does not match any existing group conversation. It may have been deleted, or the ID may be incorrect. Note that 1:1 conversations also have a `conversation_id` but group-specific operations (invite, leave) will return this error for non-group conversations.

**4002 AlreadyMember**: Returned when `group.invite` is sent for a user who is already a member. The client should refresh its member list.

**4003 NotMember**: Returned when attempting to remove a user who is not in the group. This may occur if the user has already left or been removed by another admin.

**4004 CannotRemoveSelf**: Returned when a group admin attempts to remove themselves using a removal mechanism. Self-removal must use the `group.leave` message type, which handles admin succession.

**4005 GroupFull**: The group has reached the configured maximum member limit. The default is 256 members, which aligns with practical MLS group size limits. This limit is configurable by the server administrator.

---

## 5xxx -- MLS

Errors related to MLS (Messaging Layer Security) key management and group state operations.

| Code | Name                  | Description                                                                     | HTTP Equivalent | Fatal |
|------|-----------------------|---------------------------------------------------------------------------------|----------------|-------|
| 5001 | InvalidKeyPackage     | The uploaded KeyPackage is malformed or could not be parsed.                    | 400            | No    |
| 5002 | InvalidCommit         | The MLS Commit message is malformed or rejected by server-side validation.      | 400            | No    |
| 5003 | InvalidWelcome        | The MLS Welcome message is malformed or could not be parsed.                    | 400            | No    |
| 5004 | EpochMismatch         | The MLS operation references an epoch that does not match the server's current state. | 409       | No    |
| 5005 | NoKeyPackageAvailable | No KeyPackage is available for the requested user. The user needs to upload new KeyPackages. | 404 | No    |

### Details

**5001 InvalidKeyPackage**: The server performs basic structural validation on uploaded KeyPackages. This error is returned if the data cannot be parsed as a valid KeyPackage structure. The server does not perform full cryptographic verification of the KeyPackage contents.

**5002 InvalidCommit**: The Commit message could not be parsed or failed server-side structural validation. This does not imply cryptographic verification failure (the server cannot verify MLS content), but rather that the data structure is malformed.

**5003 InvalidWelcome**: Similar to InvalidCommit, the Welcome message could not be parsed. The server relays Welcome messages without cryptographic verification but validates the structure.

**5004 EpochMismatch**: The MLS Commit references a group epoch that does not match the server's tracked epoch for the conversation. This typically occurs when two members send concurrent Commits. The client should fetch the latest group state and retry.

**5005 NoKeyPackageAvailable**: No KeyPackages remain in the pool for the requested user. This prevents adding the user to a new group. The client should notify the user to come online so their client can upload new KeyPackages. The server sends low-KeyPackage warnings to connected clients.

---

## 9xxx -- Internal

Server-side errors that are not caused by client behavior.

| Code | Name                | Description                                                             | HTTP Equivalent | Fatal |
|------|---------------------|-------------------------------------------------------------------------|----------------|-------|
| 9001 | InternalError       | An unexpected internal error occurred. The request could not be processed.| 500           | No    |
| 9002 | DatabaseError       | The database is unreachable or returned an unexpected error.             | 500            | No    |
| 9003 | ServiceUnavailable  | The server is temporarily unavailable (maintenance, overloaded, etc.).   | 503            | Yes   |

### Details

**9001 InternalError**: A catch-all for unexpected server errors. The error message will not contain implementation details for security reasons. Server logs should be consulted for debugging. If this error occurs persistently, clients should back off and retry.

**9002 DatabaseError**: The database (SQLite) is unreachable, locked, or returned an unexpected error. This is typically transient and the client should retry the operation after a short delay. If persistent, it may indicate disk space issues or database corruption.

**9003 ServiceUnavailable**: The server is unable to process requests, typically due to graceful shutdown or maintenance mode. This is a fatal error -- the WebSocket connection is closed with code `4006`. The client should implement reconnection with backoff. During planned maintenance, the error message may include an estimated return time.

---

## Error Code Quick Reference

| Code | Name                  | Category       | Fatal |
|------|-----------------------|----------------|-------|
| 1001 | InvalidCredential     | Authentication | No    |
| 1002 | ExpiredSession        | Authentication | Yes   |
| 1003 | RegistrationFailed    | Authentication | No    |
| 1004 | ChallengeFailed       | Authentication | No    |
| 1005 | SessionRevoked        | Authentication | Yes   |
| 2001 | NotGroupAdmin         | Authorization  | No    |
| 2002 | NotGroupMember        | Authorization  | No    |
| 2003 | NotAdmin              | Authorization  | No    |
| 2004 | AccountDisabled       | Authorization  | Yes   |
| 3001 | MalformedMessage      | Protocol       | No    |
| 3002 | UnknownMessageType    | Protocol       | No    |
| 3003 | MessageTooLarge       | Protocol       | No    |
| 3004 | RateLimited           | Protocol       | No    |
| 3005 | InvalidEnvelope       | Protocol       | No    |
| 4001 | GroupNotFound          | Group          | No    |
| 4002 | AlreadyMember         | Group          | No    |
| 4003 | NotMember             | Group          | No    |
| 4004 | CannotRemoveSelf      | Group          | No    |
| 4005 | GroupFull             | Group          | No    |
| 5001 | InvalidKeyPackage     | MLS            | No    |
| 5002 | InvalidCommit         | MLS            | No    |
| 5003 | InvalidWelcome        | MLS            | No    |
| 5004 | EpochMismatch         | MLS            | No    |
| 5005 | NoKeyPackageAvailable | MLS            | No    |
| 9001 | InternalError         | Internal       | No    |
| 9002 | DatabaseError         | Internal       | No    |
| 9003 | ServiceUnavailable    | Internal       | Yes   |
