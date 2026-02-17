# ADR-0001: Monorepo Structure

- **Status**: Accepted
- **Date**: 2026-02-16

## Context

Sovereign consists of multiple components: a Go server, a React Native mobile client, an admin UI, protocol definitions (Protocol Buffers), and documentation. We need to decide whether to organize these in a single monorepo or split them across multiple repositories.

Key considerations:

- The server and mobile client share protocol definitions (protobuf schemas).
- Changes to the protocol often require coordinated updates across server and client.
- The admin UI is embedded into the server binary via `go:embed`, creating a build-time dependency.
- A single developer or small team will maintain all components.
- AI coding agents benefit from having full project context in one place.

## Decision

We will use a **monorepo** containing all components in a single Git repository with the following top-level structure:

```
sovereign/
├── server/          # Go server
├── mobile/          # React Native mobile client
├── admin-ui/        # React/Vite admin panel
├── proto/           # Protocol Buffer definitions
├── docs/            # Documentation, ADRs, RFCs
├── Makefile         # Orchestrates builds across components
└── README.md
```

## Consequences

### Positive

- **Atomic cross-component changes**: A single commit can update protobuf definitions, server handling, and client parsing together, ensuring consistency.
- **Shared protocol definitions**: The `proto/` directory serves as the single source of truth, with generated code output to each component.
- **Unified CI**: One CI pipeline can build, test, and validate all components, catching integration issues early.
- **Easier agent coordination**: AI coding agents have full project context without needing to clone or reference multiple repositories.
- **Simplified dependency management**: No need for Git submodules, multi-repo version pinning, or cross-repo release coordination.

### Negative

- **Larger repository size**: The repo will grow as all components evolve, though this is manageable for a project of this scale.
- **CI configuration complexity**: The CI pipeline must be configured to detect which components changed and run only the relevant build/test steps to avoid unnecessary work.
- **Toolchain breadth**: Contributors need Go, Node.js, and protobuf toolchains available locally, though the Makefile abstracts this.

### Neutral

- Requires a top-level `Makefile` (or similar orchestration) to manage builds, code generation, and cross-component tasks.
- Git history contains changes for all components, which can be filtered with path-scoped log queries.
