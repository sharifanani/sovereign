# Sovereign WebSocket Protocol Specification

**Version**: 1.0
**Status**: Draft
**Last Updated**: 2026-02-16

---

## 1. Transport

Sovereign uses **WebSocket (RFC 6455)** over TLS as its sole transport protocol. All client-server communication flows through a single persistent WebSocket connection.

- **Protocol**: `wss://` (WebSocket Secure). Plain `ws://` is not supported in production.
- **Port**: Configurable via server settings. Default: `443` (behind reverse proxy) or `8443` (direct).
- **Endpoint**: Single WebSocket endpoint at `/ws`.
- **TLS**: TLS 1.3 required. TLS 1.2 accepted with strong cipher suites only.
- **Subprotocol**: The client SHOULD request the `sovereign.v1` WebSocket subprotocol during the handshake. The server MUST reject connections that request an unsupported subprotocol version.

### Connection URL

```
wss://your-server.example.com/ws
```

No query parameters are required for the initial connection. Authentication occurs after the WebSocket handshake completes.

---

## 2. Framing

All WebSocket messages use **binary frames**. Text frames MUST NOT be used and will be rejected by the server with a close code of `1003 (Unsupported Data)`.

Each binary frame contains exactly one serialized Protocol Buffer `Envelope` message. Fragmented WebSocket frames are handled transparently by the WebSocket layer; at the application layer, each complete message corresponds to one `Envelope`.

### Binary Format

```
[WebSocket Binary Frame]
  └─ [Serialized Protobuf Envelope]
       ├─ type: MessageType enum
       ├─ request_id: string
       └─ payload: bytes (type-specific protobuf message)
```

---

## 3. Envelope Format

Every message exchanged between client and server is wrapped in an `Envelope` Protocol Buffer message. The Envelope provides a uniform structure for routing, correlation, and deserialization.

### Envelope Fields

| Field        | Type          | Description                                                                 |
|-------------|---------------|-----------------------------------------------------------------------------|
| `type`      | `MessageType` | Enum identifying the payload type. Determines how `payload` is deserialized. |
| `request_id`| `string`      | Client-generated unique identifier for request-response correlation. Responses echo the `request_id` from the originating request. Server-initiated messages (e.g., `message.receive`, `presence.notify`) use an empty string. |
| `payload`   | `bytes`       | Serialized Protocol Buffer message specific to the `type`. The receiver deserializes this using the message definition corresponding to `type`. |

### Request-Response Correlation

- The client MUST generate a unique `request_id` for each request it sends. UUIDv4 is recommended.
- The server MUST echo the `request_id` in any direct response to that request.
- Server-initiated messages (pushes, broadcasts) set `request_id` to an empty string.
- The client SHOULD maintain a map of pending `request_id` values with timeouts. If no response is received within 30 seconds, the client SHOULD consider the request failed.

### Example Flow

```
Client                                    Server
  │                                         │
  │─── Envelope{type=AUTH_REQUEST,         │
  │    request_id="abc-123",               │
  │    payload=AuthRequest{...}}  ────────►│
  │                                         │
  │◄──── Envelope{type=AUTH_CHALLENGE,     │
  │      request_id="abc-123",             │
  │      payload=AuthChallenge{...}} ──────│
  │                                         │
```

---

## 4. Connection Lifecycle

A WebSocket connection progresses through the following states:

```
CONNECTING ──► AUTHENTICATING ──► READY ──► DISCONNECTED
                     │                        ▲
                     └──── (auth failure) ─────┘
```

### 4.1 Connect

The client opens a WebSocket connection to `wss://<host>/ws`.

- The server accepts the connection and waits for an `AuthRequest` message.
- The server MUST close the connection if no `AuthRequest` is received within 10 seconds of the handshake completing.
- Close code for timeout: `4001 (Authentication Timeout)`.

### 4.2 Authenticate

Authentication uses the Passkey/WebAuthn protocol, adapted for WebSocket transport.

**Login Flow:**

```
Client                                    Server
  │                                         │
  │── AuthRequest{username} ──────────────►│
  │                                         │  (lookup user, generate challenge)
  │◄── AuthChallenge{challenge,            │
  │    credential_request_options} ────────│
  │                                         │
  │  (user performs WebAuthn ceremony)      │
  │                                         │
  │── AuthResponse{credential_id,          │
  │   authenticator_data,                  │
  │   client_data_json, signature} ───────►│
  │                                         │  (verify assertion)
  │◄── AuthSuccess{session_token,          │
  │    user_id, username,                  │
  │    display_name} ──────────────────────│
  │                                         │
```

**Registration Flow:**

```
Client                                    Server
  │                                         │
  │── RegisterRequest{username,            │
  │   display_name} ──────────────────────►│
  │                                         │  (create pending user, generate challenge)
  │◄── RegisterChallenge{challenge,        │
  │    credential_creation_options} ───────│
  │                                         │
  │  (user creates new credential)          │
  │                                         │
  │── RegisterResponse{credential_id,      │
  │   authenticator_data,                  │
  │   client_data_json,                    │
  │   attestation_object} ────────────────►│
  │                                         │  (verify attestation, store credential)
  │◄── RegisterSuccess{user_id,            │
  │    session_token} ─────────────────────│
  │                                         │
```

**Failure:**

At any point during authentication, the server may send an `AuthError` with an error code and message. Fatal authentication errors result in the WebSocket being closed with code `4002 (Authentication Failed)`.

### 4.3 Ready

After receiving `AuthSuccess` or `RegisterSuccess`, the connection enters the READY state. In this state:

- The client may send and receive any non-auth message type.
- The server delivers any messages that were queued while the client was offline.
- The server sends presence notifications for contacts who are currently online.

### 4.4 Messaging

Once in the READY state, the client and server exchange typed messages bidirectionally:

- **Client-to-server**: Sending messages, creating groups, uploading key packages, updating presence, etc.
- **Server-to-client**: Delivering messages, notifying of group changes, broadcasting MLS commits, forwarding presence updates, etc.

Each message is wrapped in an `Envelope` with the appropriate `MessageType` and serialized payload.

### 4.5 Heartbeat

Heartbeat messages ensure connection liveness and detect network failures promptly.

| Parameter       | Value  |
|----------------|--------|
| Ping interval  | 30 seconds |
| Pong timeout   | 10 seconds |
| Initiator      | Client |

**Behavior:**

1. The client sends a `Ping` message every 30 seconds, containing the current timestamp.
2. The server responds with a `Pong` message, echoing the timestamp.
3. If the client does not receive a `Pong` within 10 seconds of sending a `Ping`, it MUST consider the connection dead and initiate a reconnection.
4. The server MAY also send unsolicited `Ping` messages. The client MUST respond with a `Pong`.
5. The server MUST close connections that have not sent any message (including `Ping`) for 90 seconds.

**Note**: These are application-level ping/pong messages (inside `Envelope`), distinct from WebSocket-level ping/pong frames. Implementations SHOULD also handle WebSocket-level ping/pong frames as required by RFC 6455.

### 4.6 Disconnect

Either side can close the WebSocket connection:

- **Graceful**: Send a WebSocket Close frame with an appropriate close code.
- **Ungraceful**: Network failure, process crash, etc.

On disconnect, the server:
- Marks the user's presence as `offline`.
- Queues any subsequent messages addressed to the user for later delivery.
- Retains the session token for reconnection (subject to session expiry).

---

## 5. Reconnection

When a connection drops, the client MUST implement automatic reconnection with exponential backoff and jitter.

### Backoff Schedule

| Attempt | Base Delay | Max Delay |
|---------|-----------|-----------|
| 1       | 1s        | —         |
| 2       | 2s        | —         |
| 3       | 4s        | —         |
| 4       | 8s        | —         |
| 5       | 16s       | —         |
| 6+      | 30s       | 30s       |

### Jitter

Each delay is randomized with jitter to prevent thundering herd:

```
actual_delay = base_delay * (0.5 + random(0, 1) * 0.5)
```

This produces a delay between 50% and 100% of the base delay.

### Reconnection Flow

1. Client detects disconnect (Pong timeout, WebSocket close, or network error).
2. Client waits for the backoff delay.
3. Client opens a new WebSocket connection to `/ws`.
4. Client sends `AuthRequest` with the stored `session_token` instead of initiating a full WebAuthn ceremony.
5. Server validates the session token and responds with `AuthSuccess`.
6. Server delivers any messages queued since the client's last acknowledged message.
7. Backoff counter resets on successful reconnection.

If the session token has expired, the server responds with `AuthError` (code `1002 ExpiredSession`). The client MUST then perform a full re-authentication with WebAuthn.

---

## 6. Message Ordering

### Server Timestamps

- The server assigns a monotonically increasing `server_timestamp` (Unix microseconds) to every message it processes.
- The `server_timestamp` is the canonical ordering key for messages.
- The server guarantees that within a single conversation, messages have strictly increasing `server_timestamp` values.

### Client Handling

- Clients MUST order messages within a conversation by `server_timestamp`.
- Messages MAY arrive out of order due to network conditions or queued delivery. Clients SHOULD handle this by inserting messages at the correct position based on `server_timestamp`.
- Clients SHOULD NOT rely on message arrival order matching `server_timestamp` order.

### Delivery Guarantees

- **At-least-once delivery**: The server will attempt to deliver each message at least once. Clients MUST be prepared to receive duplicate messages (identified by `message_id`).
- **No guaranteed delivery order**: While the server assigns ordered timestamps, delivery order is not guaranteed. Use `server_timestamp` for display ordering.
- **Acknowledgment**: Clients send `message.ack` after processing a received message. The server uses acknowledgments to track delivery progress and avoid re-sending acknowledged messages on reconnection.

---

## 7. Flow Control

### Message Size Limit

| Limit                | Value  |
|---------------------|--------|
| Max message size    | 64 KB  |
| Max envelope size   | 65,536 bytes (serialized Envelope) |

Messages exceeding the size limit are rejected with error code `3003 (MessageTooLarge)`. The WebSocket connection remains open.

### Rate Limiting

| Limit                       | Default | Configurable |
|----------------------------|---------|-------------|
| Messages per second per connection | 30      | Yes         |
| Burst allowance            | 10      | Yes         |

Rate limiting uses a token bucket algorithm:
- Each connection has a bucket with capacity equal to the burst allowance.
- Tokens are added at the rate limit per second.
- Each sent message consumes one token.
- When the bucket is empty, messages are rejected with error code `3004 (RateLimited)`.
- Rate limit errors are non-fatal; the client SHOULD wait before sending more messages.
- The `error` payload includes a `retry_after_ms` field indicating when the client may retry.

### Connection Limits

| Limit                          | Default | Configurable |
|-------------------------------|---------|-------------|
| Max concurrent connections per user | 5       | Yes         |
| Max total connections          | 10,000  | Yes         |

When a user exceeds the per-user connection limit, the oldest connection is closed with code `4003 (Too Many Connections)`.

---

## 8. Error Handling

### Error Messages

The server sends typed `Error` messages within an `Envelope` to communicate problems to the client.

| Field     | Type    | Description                                              |
|----------|---------|----------------------------------------------------------|
| `code`   | `int32` | Numeric error code (see error-codes.md)                  |
| `message`| `string`| Human-readable error description                         |
| `fatal`  | `bool`  | If true, the server will close the connection after sending this error |

### Fatal vs Non-Fatal Errors

- **Non-fatal errors**: The connection remains open. The client should handle the error and may continue sending messages. Examples: rate limiting, invalid message format, group not found.
- **Fatal errors**: The server closes the WebSocket connection immediately after sending the error. The client should not attempt to send further messages on this connection. Examples: authentication failure, session revoked, account disabled.

### WebSocket Close Codes

Sovereign uses custom WebSocket close codes in the 4000-4999 range:

| Code | Name                    | Description                                       |
|------|------------------------|---------------------------------------------------|
| 4001 | Authentication Timeout | No auth message received within timeout           |
| 4002 | Authentication Failed  | WebAuthn verification failed                      |
| 4003 | Too Many Connections   | User exceeded max concurrent connections           |
| 4004 | Session Expired        | Session token is no longer valid                   |
| 4005 | Account Disabled       | User account has been disabled by admin            |
| 4006 | Server Shutdown        | Server is shutting down gracefully                 |
| 4007 | Protocol Error         | Unrecoverable protocol violation                   |

### Standard WebSocket Close Codes Used

| Code | Name              | Usage                                            |
|------|------------------|--------------------------------------------------|
| 1000 | Normal Closure   | Clean disconnect initiated by either side        |
| 1001 | Going Away       | Server shutdown or client navigating away        |
| 1003 | Unsupported Data | Client sent a text frame instead of binary       |
| 1009 | Message Too Big  | Message exceeds 64KB limit                       |
| 1011 | Internal Error   | Unexpected server error                          |

---

## Appendix A: Wire Format Summary

```
┌─────────────────────────────────┐
│     TLS 1.3 Connection          │
│  ┌───────────────────────────┐  │
│  │   WebSocket (RFC 6455)    │  │
│  │  ┌─────────────────────┐  │  │
│  │  │  Binary Frame       │  │  │
│  │  │  ┌───────────────┐  │  │  │
│  │  │  │  Envelope      │  │  │  │
│  │  │  │  (protobuf)    │  │  │  │
│  │  │  │  ┌───────────┐ │  │  │  │
│  │  │  │  │ type      │ │  │  │  │
│  │  │  │  │ request_id│ │  │  │  │
│  │  │  │  │ payload   │ │  │  │  │
│  │  │  │  └───────────┘ │  │  │  │
│  │  │  └───────────────┘  │  │  │
│  │  └─────────────────────┘  │  │
│  └───────────────────────────┘  │
└─────────────────────────────────┘
```

## Appendix B: Version History

| Version | Date       | Changes                    |
|---------|-----------|----------------------------|
| 1.0     | 2026-02-16 | Initial protocol specification |
