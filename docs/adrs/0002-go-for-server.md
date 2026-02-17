# ADR-0002: Go for Server

- **Status**: Accepted
- **Date**: 2026-02-16

## Context

The Sovereign server is the central component responsible for message routing, authentication, key package management, and admin functionality. It must be easy to self-host, meaning minimal runtime dependencies and straightforward deployment.

Requirements for the server language/runtime:

- Produce a single, statically-linked binary with no runtime dependencies.
- Strong standard library for HTTP, TLS, and cryptographic operations.
- Mature WebSocket library support.
- Easy cross-compilation for Linux (amd64, arm64), macOS, and potentially other targets.
- Reasonable development velocity for a small team.

Alternatives considered:

- **Rust**: Excellent performance and memory safety, but steeper learning curve, slower iteration speed, and longer compile times. Overkill for a server that is I/O-bound rather than CPU-bound.
- **Node.js (TypeScript)**: Familiar ecosystem but requires a runtime, complicates single-binary distribution, and is less suitable for long-running server processes with strict memory behavior.
- **Python**: Not suitable for single-binary distribution, runtime dependency, weaker concurrency story.

## Decision

We will use **Go** for the Sovereign server.

Go produces a single statically-linked binary with no runtime dependencies. Its standard library includes production-quality `net/http`, `crypto`, and `encoding` packages. The goroutine-based concurrency model is well-suited for managing many concurrent WebSocket connections. Cross-compilation is a first-class feature (`GOOS`/`GOARCH` environment variables). The language is simple enough to maintain high development velocity.

We will additionally enforce a **no-CGo** policy (see ADR-0009) to preserve the single-binary, cross-compilation benefits.

## Consequences

### Positive

- **Single binary deployment**: `go build` produces one executable. Deployment is `scp` + `systemctl restart`.
- **Fast builds**: Full server builds complete in seconds, enabling tight development loops.
- **Strong concurrency model**: Goroutines and channels are natural for managing per-connection WebSocket handlers, background tasks, and fan-out delivery.
- **Excellent standard library**: `net/http` for the admin API, `crypto/subtle` for constant-time comparisons, `embed` for the admin UI — all built in.
- **Easy cross-compilation**: Build for `linux/arm64` from a macOS development machine with one command.
- **Broad ecosystem**: Mature libraries for WebSocket (`gorilla/websocket` or `nhooyr.io/websocket`), WebAuthn (`go-webauthn/webauthn`), and Protocol Buffers (`google.golang.org/protobuf`).

### Negative

- **Verbose error handling**: Go's explicit `if err != nil` pattern adds verbosity, though it makes error paths explicit and reviewable.
- **Limited generics usage**: While Go now has generics, the ecosystem hasn't fully embraced generic-heavy patterns. This is an acceptable trade-off for simplicity.
- **No sum types**: Error and state modeling is less expressive than Rust's `enum` types. Mitigated by disciplined use of constants and interfaces.

### Neutral

- Go's garbage collector introduces occasional, small pause times. For a messaging server at the expected scale (single server, hundreds of connections), this is imperceptible.
- The Go module system handles dependency management. `go.sum` provides reproducible builds.
