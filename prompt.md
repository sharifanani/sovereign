# Sovereign — Agent Entrypoint

Use this document to orient yourself when starting a new session on this project.

## What Is Sovereign?

A privacy-focused self-hosted messaging application. Users run their own server (single Go binary), connect with a React Native mobile client, and communicate via MLS E2E encrypted messages with passkey authentication. The target experience is akin to WhatsApp, but self-hosted and private — no third-party infrastructure, no metadata harvesting.

## Current State

### Bootstrap (Steps 1-8) — Complete

- **Repository structure**: Monorepo with `server/`, `mobile/`, `admin-ui/`, `protocol/`, `docs/`
- **Project rules**: `CLAUDE.md` at the root defines all development rules and conventions
- **7 AI subagents**: Defined in `.claude/agents/` (architect, backend-engineer, mobile-engineer, security-engineer, devops-engineer, technical-writer, qa-engineer)
- **Design documents**: System architecture, data model, threat model, UX flows in `docs/design/`
- **Protocol specification**: Full protocol spec, message types, admin API, error codes in `docs/api/`
- **Protobuf definitions**: `protocol/messages.proto` with all message types
- **9 ADRs**: All architectural decisions recorded in `docs/adrs/`
- **6 RFCs**: All feature designs documented in `docs/rfcs/`
- **Templates**: RFC and ADR templates in `docs/rfcs/_template.md` and `docs/adrs/_template.md`
- **Guides**: Agent workflow, review process, contributing guides in `docs/guides/`
- **Admin UI scaffold**: Vite + React, builds to `server/web/dist/` for Go embedding
- **Makefile**: `build`, `test`, `lint`, `clean`, `dev-server`, `proto`, `cross-compile` targets

### Phase A (WebSocket echo + client connection) — Complete, needs tests

**Server (`server/`):**
- Protobuf Go code generated from `protocol/messages.proto` into `server/internal/protocol/messages.pb.go`
- WebSocket echo server implemented: accepts connections at `/ws`, binary frames only, `sovereign.v1` subprotocol
- `internal/ws/hub.go` — Connection registry with goroutine-safe register/unregister
- `internal/ws/conn.go` — Read/write pump goroutines, Ping/Pong handling, echo for all other message types, error responses, 64KB message limit
- `internal/ws/upgrade.go` — HTTP-to-WebSocket upgrade with subprotocol negotiation
- `internal/config/config.go` — `DefaultConfig()` with defaults (`:8080`, 64KB max message, etc.)
- `cmd/sovereign/main.go` — Wires up Hub, HTTP server, embedded admin UI at `/admin/`, graceful shutdown
- Dependencies: `nhooyr.io/websocket` (pure Go), `google.golang.org/protobuf`
- `go build ./...` and `go vet ./...` pass

**Mobile (`mobile/`):**
- `src/services/protocol.ts` — Protobuf codec via `protobufjs` reflection API (Envelope, Ping, Pong, Error encode/decode, UUIDv4 request ID generation)
- `src/services/websocket.ts` — WebSocket client with binary frames, Ping/Pong heartbeat (30s/10s), exponential backoff reconnection with jitter, connection state callbacks
- `src/screens/Chat.tsx` — Chat UI with connection status, configurable server URL, message list, send/receive echo messages
- `npx tsc --noEmit` passes
- **Not yet runnable as a mobile app** — missing `index.js`, `app.json`, Metro config, and `ios/`/`android/` native directories. The scaffold was TypeScript-only. Needs proper React Native or Expo initialization before it can run on a device/simulator.

**Tests (added in Phase A.1):**
- `server/internal/ws/conn_test.go` — Echo, Ping/Pong, invalid message, non-binary rejection, message size limit, send buffer full (15 test cases)
- `server/internal/ws/hub_test.go` — Register/unregister, count, stop (5 subtests)
- `server/internal/ws/upgrade_test.go` — Subprotocol negotiation (3 subtests)
- `server/internal/config/config_test.go` — Default values, no zero values (6 subtests)
- `mobile/__tests__/protocol.test.ts` — Envelope round-trip, Ping/Pong/Error codec, MessageType enum values, generateRequestId, messageTypeName (50 tests)
- `mobile/__tests__/websocket.test.ts` — Connection state, send, ping/pong, reconnection backoff, intentional disconnect, message handling (17 tests)
- All pass: `go test ./...` and `cd mobile && npx jest`

## MVP Development Phases

Execute these in order (D, E, F can be parallelized after C):

| Phase | Description | Status |
|-------|-------------|--------|
| **A** | WebSocket echo server + React Native client that connects | **Done** |
| **A.1** | Tests for Phase A (server + mobile) | **Done** |
| **B** | Passkey registration and login (server + client) | **Next** |
| **C** | 1:1 E2E encrypted text messaging | Pending |
| **D** | Group chat with MLS key management | Pending |
| **E** | Multi-server client (multiple simultaneous connections) | Pending |
| **F** | Admin panel (embedded web UI for user/server management) | Pending |
| **G** | CLI wizard, cross-compilation, docs, final security review | Pending |

**Phase A.1 (tests) is the next step.**

## Workflow Rules

When implementing any phase, **always include the qa-engineer agent** to write tests alongside the implementation. Every phase must have tests before it's considered complete. Go code needs table-driven tests; TypeScript needs Jest tests.

## Key Files to Read First

| File | Purpose |
|------|---------|
| `CLAUDE.md` | Project rules, tech stack, coding standards |
| `docs/design/system-architecture.md` | Component architecture and data flow |
| `docs/design/data-model.md` | Database entities and schema |
| `docs/api/protocol-spec.md` | WebSocket protocol design |
| `docs/api/message-types.md` | All protocol message types |
| `protocol/messages.proto` | Canonical protobuf definitions |
| `docs/design/ux-flows.md` | User journeys and screen flows |
| `docs/design/threat-model.md` | Security boundaries and threat analysis |

## Build Verification

Run these to verify the project is in a healthy state:

```bash
# Go server builds
cd server && go build ./...

# Mobile TypeScript compiles
cd mobile && npx tsc --noEmit

# Admin UI builds and embeds
cd admin-ui && npm run build

# Full build (admin-ui + server binary)
make build
```

## Agent Invocation

Agents are defined in `.claude/agents/` and can be invoked to work on specific areas:

- **architect** — For design questions, RFCs, ADRs, protocol changes
- **backend-engineer** — For Go server implementation
- **mobile-engineer** — For React Native client work
- **security-engineer** — For MLS, auth, and security review
- **devops-engineer** — For build, CI/CD, and infrastructure
- **technical-writer** — For documentation
- **qa-engineer** — For testing strategy and implementation
