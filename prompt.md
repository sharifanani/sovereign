# Sovereign — Agent Entrypoint

Use this document to orient yourself when starting a new session on this project.

## What Is Sovereign?

A privacy-focused self-hosted messaging application. Users run their own server (single Go binary), connect with a React Native mobile client, and communicate via MLS E2E encrypted messages with passkey authentication. The target experience is akin to WhatsApp, but self-hosted and private — no third-party infrastructure, no metadata harvesting.

## Current State

### Bootstrap Complete (Steps 1-8)

The project has been fully bootstrapped. The following is in place:

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
- **Go server scaffold**: Compiles (`go build ./...`), has placeholder packages for all internal components, initial DB migration
- **React Native scaffold**: TypeScript strict mode, compiles (`npx tsc --noEmit`), placeholder screens and services
- **Admin UI scaffold**: Vite + React, builds to `server/web/dist/` for Go embedding
- **Makefile**: `build`, `test`, `lint`, `clean`, `dev-server`, `proto`, `cross-compile` targets

### What Has NOT Been Done

No application logic has been implemented yet. All Go packages, React components, and services are stubs/placeholders. The MVP development phases have not started.

## MVP Development Phases (Next Steps)

Execute these in order (D, E, F can be parallelized after C):

| Phase | Description | Agents Involved |
|-------|-------------|-----------------|
| **A** | WebSocket echo server + React Native client that connects | backend, mobile, devops |
| **B** | Passkey registration and login (server + client) | security, backend, mobile |
| **C** | 1:1 E2E encrypted text messaging | all engineering agents |
| **D** | Group chat with MLS key management | security, backend, mobile |
| **E** | Multi-server client (multiple simultaneous connections) | mobile |
| **F** | Admin panel (embedded web UI for user/server management) | backend, devops |
| **G** | CLI wizard, cross-compilation, docs, final security review | all agents |

**Phase A is the next phase to implement.**

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
