# Contributing to Sovereign

## Development Setup

### Prerequisites

- Go 1.22+
- Node.js 20+
- Protocol Buffers compiler (`protoc`)
- React Native development environment (Xcode for iOS, Android Studio for Android)

### Clone and Build

```bash
git clone <repo-url>
cd sovereign

# Build everything
make build

# Run tests
make test

# Lint
make lint
```

### Server Development

```bash
cd server
go mod download
go build ./...
go test ./...
```

### Mobile Development

```bash
cd mobile
npm install
npx tsc --noEmit    # Type check

# iOS
npx react-native run-ios

# Android
npx react-native run-android
```

### Admin UI Development

```bash
cd admin-ui
npm install
npm run dev          # Development server with hot reload
npm run build        # Production build → server/web/dist/
```

## Coding Standards

### Go

- No CGo dependencies
- Handle all errors explicitly
- Write table-driven tests
- Use `context.Context` for cancellation
- Run `go vet ./...` before committing

### TypeScript

- Strict mode enabled
- No `any` types (except at library boundaries)
- Functional components with hooks
- Explicit Props interfaces for all components

### Protocol

- `protocol/messages.proto` is the source of truth
- Never remove or renumber protobuf fields
- Changes must be reflected in both Go and TypeScript

## Process

### Before Writing Code

1. Check for an existing RFC or design document
2. For new features, propose an RFC in `docs/rfcs/`
3. For architectural decisions, create an ADR in `docs/adrs/`

### Making Changes

1. Create a branch: `{issue-number}-{description}`
2. Make your changes following coding standards
3. Write or update tests
4. Ensure `make build` and `make test` pass
5. Create a PR with a clear description

### Commit Messages

Use conventional commits:
- `feat:` — New feature
- `fix:` — Bug fix
- `docs:` — Documentation
- `refactor:` — Code restructuring
- `test:` — Test additions or changes
- `chore:` — Build, CI, or tooling changes

### Pull Requests

- Keep PRs focused on a single change
- Reference the relevant issue or RFC
- Include test results in the PR description
- Request review from the appropriate agent/person

## Security

- Never commit secrets, keys, or `.env` files
- Use platform-specific secure storage for key material
- Report security issues privately — do not open public issues
