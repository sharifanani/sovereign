# Sovereign

Private messaging where you own the server.

Sovereign is a privacy-focused messaging application where users host their own servers. No third-party infrastructure, no metadata harvesting, no trust required beyond your own hardware.

## Key Features

- **Self-hosted**: Run your own server on any machine
- **End-to-end encrypted**: MLS (RFC 9420) for 1:1 and group messages
- **Passkey authentication**: No passwords, hardware-backed identity
- **Multi-server**: Connect to multiple Sovereign servers from one client
- **Admin panel**: Web-based server management embedded in the binary
- **Single binary**: One Go binary, zero external dependencies

## Architecture

| Component | Technology |
|-----------|------------|
| Server | Go (pure, no CGo) |
| Database | SQLite (modernc.org/sqlite) |
| Mobile Client | React Native (TypeScript) |
| Admin UI | React (Vite), embedded via `go:embed` |
| Protocol | Custom over WebSocket, Protocol Buffers |
| E2E Encryption | MLS (RFC 9420) |
| Authentication | Passkey / WebAuthn |

## Repository Structure

```
sovereign/
├── server/       # Go server + CLI wizard
├── mobile/       # React Native client
├── admin-ui/     # Admin panel (Vite, builds into server/web/dist/)
├── protocol/     # Canonical protobuf definitions
├── docs/         # RFCs, ADRs, design docs, API specs, guides
└── scripts/      # Build and development scripts
```

## Getting Started

### Prerequisites

- Go 1.22+
- Node.js 20+
- React Native development environment

### Build

```bash
make build        # Build server binary with embedded admin UI
make test         # Run all tests
make lint         # Lint all code
```

### Server Setup

```bash
./sovereign-cli setup    # Interactive setup wizard
./sovereign              # Start server
```

## Documentation

- [System Architecture](docs/design/system-architecture.md)
- [Protocol Specification](docs/api/protocol-spec.md)
- [Contributing Guide](docs/guides/contributing.md)
- [Development Setup](docs/guides/dev-setup.md)

## License

Business Source License (BSL). See [LICENSE](LICENSE) for details.
