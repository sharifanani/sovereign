# ADR-0008: Embedded Admin UI

- **Status**: Accepted
- **Date**: 2026-02-16

## Context

Sovereign needs an administration interface for server operators to manage users, view server status, configure settings, and perform maintenance tasks. This admin UI must be accessible via a web browser, as server operators may not always have SSH access or prefer a graphical interface.

A core deployment goal is that the Sovereign server is a single binary with no external dependencies. The admin UI must not require a separate web server, CDN, or static file hosting.

Requirements:

- Web-based admin panel accessible from any modern browser.
- No separate deployment or hosting — must ship with the server binary.
- Always version-matched with the server (no version skew between admin UI and server API).
- Interactive enough for configuration, user management, and monitoring tasks.

Alternatives considered:

- **Separate admin server**: Run the admin UI as its own process or container. Adds operational complexity — another thing to deploy, version, and secure.
- **CLI-only admin**: Administer the server via command-line tools. No separate deployment needed, but poor UX for tasks like browsing user lists or viewing charts.
- **Server-rendered HTML (Go templates)**: Simpler than a JavaScript SPA, but limited interactivity. Poor experience for real-time updates or complex forms.

## Decision

We will build the admin UI as a **React/Vite single-page application** and embed it into the Go server binary using the `//go:embed` directive.

The build process:

1. The admin UI source lives in `admin-ui/` in the monorepo.
2. `npm run build` (Vite) outputs static assets (HTML, JS, CSS) to `server/web/dist/`.
3. The Go server embeds `server/web/dist/` using `//go:embed dist/*`.
4. At runtime, the server serves the embedded files under `/admin/` via `http.FileServer`.

The admin UI communicates with the server via a REST API (separate from the WebSocket messaging API) served on the same port under `/api/admin/`.

## Consequences

### Positive

- **Single binary deployment**: The admin UI is compiled into the server binary. No CDN, no static file server, no Docker sidecar.
- **Always version-matched**: The admin UI and server API are built from the same commit. There is zero risk of version skew between the admin frontend and the server backend.
- **No CORS complexity**: The admin UI and its API are served from the same origin, eliminating cross-origin issues.
- **Simple security model**: Admin access is a single endpoint on the server, protected by the same WebAuthn authentication used for the messaging API.

### Negative

- **Larger binary size**: Embedding the admin UI increases the server binary size. In practice, a production-built React/Vite app (with code splitting and compression) is typically 200-500 KB gzipped — a negligible addition to the Go binary.
- **Build step dependency**: The admin UI must be built (`npm run build`) before the Go binary can be compiled. The Makefile orchestrates this, but it adds a Node.js dependency to the build environment (not the runtime).
- **No hot reload in production**: The embedded files are static at compile time. During development, the admin UI can be served by Vite's dev server with hot reload, proxying API requests to the Go server.

### Neutral

- The `//go:embed` directive requires the embedded directory to exist at build time. The Makefile ensures the admin UI is built before the Go build step.
- The admin UI uses the same authentication system (WebAuthn/passkeys) as the mobile client, but with an additional role check to verify the user has admin privileges.
