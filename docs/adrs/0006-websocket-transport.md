# ADR-0006: WebSocket Transport

- **Status**: Accepted
- **Date**: 2026-02-16

## Context

The Sovereign server and mobile client need a real-time, bidirectional communication channel. Messages must be delivered with low latency in both directions — the server pushes incoming messages to connected clients, and clients send outgoing messages to the server.

Requirements:

- True bidirectional communication (server can push to client without polling).
- Low latency for real-time messaging.
- Support for binary payloads (Protocol Buffer encoded messages).
- Wide client support (React Native, browsers for admin panel).
- Reasonable behavior through firewalls and proxies.

Alternatives considered:

- **HTTP long-polling**: Higher latency, more connection overhead, more server resource usage per client. A fallback option, but not a primary transport.
- **gRPC (bidirectional streaming)**: Powerful but complex on mobile. Requires HTTP/2, which can be problematic through some proxies. gRPC-Web adds another translation layer. Overkill for our needs.
- **Server-Sent Events (SSE)**: Server-to-client only. Would require a separate channel for client-to-server messages (e.g., HTTP POST), complicating the protocol and adding latency for sends.

## Decision

We will use **WebSocket** as the primary transport between the Sovereign mobile client and server.

WebSocket provides true bidirectional communication over a single TCP connection. After the initial HTTP upgrade handshake, both sides can send messages at any time with minimal framing overhead. WebSocket supports binary frames natively, which pairs well with Protocol Buffer serialization. It is widely supported across platforms and works through most firewalls and proxies.

All messages are framed using a Protocol Buffer `Envelope` message (see RFC-0002) sent as binary WebSocket frames.

## Consequences

### Positive

- **Real-time delivery**: Messages are delivered instantly in both directions without polling.
- **Low overhead**: After the initial handshake, WebSocket framing adds only 2-14 bytes per message, far less than HTTP request/response headers.
- **Binary frame support**: Protocol Buffer payloads are sent as binary frames without base64 encoding overhead.
- **Wide support**: React Native, all browsers, and Go all have mature WebSocket implementations.
- **Single connection**: One connection handles all message types (chat messages, typing indicators, presence, key packages), simplifying connection management.

### Negative

- **Stateful connections**: Each connected client holds a persistent TCP connection on the server. Requires connection lifecycle management (heartbeat, timeout, cleanup).
- **No built-in request-response**: WebSocket is a message-oriented protocol without request-response correlation. We solve this by including a `request_id` field in the Protocol Buffer envelope, allowing clients to match responses to requests.
- **Reconnection complexity**: Clients must handle disconnection and reconnection gracefully, including re-authentication, message gap detection, and backoff strategies.
- **Load balancer considerations**: WebSocket connections are long-lived, which can affect load balancer behavior. For single-server Sovereign deployments, this is not a concern.

### Neutral

- The WebSocket connection is established over HTTPS (WSS), inheriting TLS encryption for the transport layer. This is independent of the MLS end-to-end encryption.
- Heartbeat/ping-pong frames are used to detect dead connections and keep connections alive through intermediaries.
