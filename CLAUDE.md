# Sovereign — Project Instructions

## Overview

Sovereign is a privacy-focused private messaging application where users host their own servers. Single Go binary, React Native mobile client, MLS E2E encryption, passkey authentication.

## Tech Stack

| Component | Technology |
|-----------|------------|
| Server | Go 1.22+ (pure, no CGo, single binary) |
| Database | SQLite via `modernc.org/sqlite` (CGo-free) |
| Mobile | React Native (TypeScript, strict mode) |
| Admin UI | React (Vite), embedded in Go binary via `//go:embed` |
| Protocol | Custom over WebSocket, Protocol Buffers wire format |
| E2E Encryption | MLS (RFC 9420) |
| Authentication | Passkey / WebAuthn |

## Repository Structure

- `server/` — Go server and CLI setup wizard
- `mobile/` — React Native client app
- `admin-ui/` — Admin panel React app (output builds to `server/web/dist/`)
- `protocol/` — Canonical protobuf definitions (source of truth for all message types)
- `docs/rfcs/` — RFC design proposals
- `docs/adrs/` — Architecture Decision Records
- `docs/design/` — Architecture, data model, threat model, UX flows
- `docs/api/` — Protocol spec, message types, admin API, error codes
- `docs/guides/` — Contributing, agent workflow, review process, dev setup
- `scripts/` — Build, lint, test automation

## Development Rules

### Process

- **Design before code**: All significant features require an RFC or ADR before implementation begins.
- **Review tiers**: Minor changes (naming, formatting, small fixes) can be agent-reviewed. Major changes (architecture, security, protocol) require human approval.
- **ADRs**: Record all architectural decisions in `docs/adrs/` using the template.
- **RFCs**: Propose all significant features in `docs/rfcs/` using the template.

### Go (Server)

- **No CGo**: All dependencies must be pure Go. Never use CGo or libraries that require it.
- **Build check**: `go build ./...` must pass before committing.
- **Testing**: `go test ./...` must pass. Write table-driven tests.
- **Linting**: `go vet ./...` must pass.
- **Error handling**: Always handle errors explicitly. No `_` for error returns.
- **Naming**: Follow standard Go naming conventions (camelCase for unexported, PascalCase for exported).
- **Module path**: `github.com/sovereign-im/sovereign/server`

### TypeScript (Mobile & Admin UI)

- **Strict mode**: `tsconfig.json` must have `strict: true`.
- **Functional components**: Use React functional components with hooks. No class components.
- **Type safety**: No `any` types except in type assertion boundaries with external libraries.
- **Naming**: PascalCase for components, camelCase for functions/variables, UPPER_SNAKE for constants.

### Protocol

- **Source of truth**: `protocol/messages.proto` is the canonical definition of all message types.
- **Backward compatibility**: Never remove or renumber protobuf fields. Mark deprecated fields as reserved.
- **Both platforms**: Changes to protobuf definitions must be reflected in both Go and TypeScript generated code.

### Security

- **No custom crypto**: Use established, audited cryptographic libraries only.
- **No plaintext keys**: Never store private keys or credentials in plaintext. Use platform-specific secure storage.
- **MLS compliance**: Follow RFC 9420 strictly. No shortcuts or simplifications to the protocol.
- **Input validation**: Validate all external input at system boundaries (WebSocket messages, HTTP requests, user input).
- **Threat model**: All security-relevant changes must be evaluated against `docs/design/threat-model.md`.

### Git

- **Commit messages**: Use conventional commits (`feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`).
- **Branch naming**: Use descriptive branch names. When working on issues, use `{issue-number}-{description}`.
- **No secrets**: Never commit `.env` files, private keys, or credentials.

## Build & Test Commands

**Always use `make` targets** for building, testing, and linting. They handle working directories correctly and prevent mistakes. Run all make commands from the project root.

```bash
make build         # Build server binary with embedded admin UI
make test          # Run ALL tests (Go + TypeScript)
make test-server   # Run Go tests only
make test-mobile   # Run TypeScript/Jest tests only
make lint          # Lint all code (go vet + tsc --noEmit)
make lint-server   # Go vet only
make lint-mobile   # Mobile tsc --noEmit only
make lint-admin    # Admin UI tsc --noEmit only
make clean         # Remove build artifacts
make dev-server    # Run server in development mode
make proto         # Regenerate protobuf stubs
```

Do **not** run `go test`, `npx jest`, `npx tsc`, or `npm test` directly — use the make targets instead.

## Agent System

This project uses 7 specialized AI subagents defined in `.claude/agents/`. Each agent owns specific directories and has defined responsibilities. See `docs/guides/agent-workflow.md` for the full workflow.

| Agent | Responsibilities |
|-------|-----------------|
| architect | RFCs, ADRs, system design, protocol spec, cross-component integration |
| backend-engineer | Go server code, SQLite DAL, WebSocket, admin API, CLI wizard |
| mobile-engineer | React Native app, WebSocket client, protocol codec, multi-server |
| security-engineer | MLS integration, passkey auth, threat model, security reviews |
| devops-engineer | Build scripts, CI/CD, cross-compilation, admin UI build pipeline |
| technical-writer | Documentation, README, guides, setup docs |
| qa-engineer | Test strategy, integration tests, protocol conformance tests |
