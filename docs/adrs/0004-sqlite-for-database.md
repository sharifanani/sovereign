# ADR-0004: SQLite for Database

- **Status**: Accepted
- **Date**: 2026-02-16

## Context

The Sovereign server needs a database for storing user accounts, key packages, message metadata, delivery state, and server configuration. A core design goal of Sovereign is easy self-hosting — the server should be a single binary with minimal operational dependencies.

Requirements:

- No external database server process to install, configure, or maintain.
- SQL query support for flexible data access patterns.
- Reliable and well-tested — data loss is unacceptable for a messaging system.
- Sufficient performance for a single-server deployment (expected scale: tens to hundreds of users).

Alternatives considered:

- **PostgreSQL**: Industry-standard relational database, but requires a separate server process, configuration, and maintenance. This directly contradicts the "single binary, zero-config" deployment goal.
- **LevelDB / Pebble**: Embedded key-value stores. No SQL support means we'd need to implement our own indexing and query logic, adding complexity without benefit.
- **BoltDB (bbolt)**: Embedded key-value store for Go. Same limitations as LevelDB — no SQL, and the single-writer model is more restrictive than SQLite's WAL mode.

## Decision

We will use **SQLite** as the sole database for the Sovereign server.

SQLite is an embedded SQL database engine that stores data in a single file. It requires no separate server process, no configuration, and no maintenance. It supports full SQL including joins, indexes, triggers, and CTEs. It is the most widely deployed database engine in the world, with decades of rigorous testing.

We will use WAL (Write-Ahead Logging) mode for improved concurrent read performance and configure appropriate busy timeouts for write contention.

## Consequences

### Positive

- **No external dependencies**: The database is compiled into the server binary. No `apt install`, no connection strings, no credentials to manage.
- **Single-file storage**: All data lives in one file (plus WAL and SHM files), simplifying backup (`cp sovereign.db sovereign.db.bak`) and migration.
- **Full SQL support**: Complex queries, joins, indexes, transactions, and CTEs are available for flexible data access patterns.
- **Battle-tested reliability**: SQLite is used in aviation, Android, iOS, browsers, and countless embedded systems. Its test suite has 100% branch coverage.
- **Excellent performance**: For the expected scale (single server, hundreds of users), SQLite is more than fast enough. Reads are essentially memory-mapped file access.

### Negative

- **Single-writer concurrency model**: Only one writer can operate at a time (WAL mode allows concurrent reads during writes). This limits write throughput, but is acceptable for the expected scale.
- **Single-server deployment**: SQLite does not support multi-server replication. Sovereign is designed for single-server deployment, so this is not a current concern. If multi-server is ever needed, this decision would need revisiting.
- **No built-in network access**: SQLite cannot be accessed remotely. All access must happen within the server process, which is the intended design.

### Neutral

- Requires careful schema migration management. We will use a versioned migration system embedded in the server binary.
- Database tuning (page size, cache size, journal mode) should be configured at server startup for optimal performance.
