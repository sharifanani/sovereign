# Data Model

## Overview

Sovereign uses a single SQLite database file (`sovereign.db`) managed via the pure-Go driver `modernc.org/sqlite`. All tables use strict typing where possible and follow consistent conventions for identifiers, timestamps, and foreign keys.

### Conventions

- **Primary keys**: TEXT type containing UUIDs (v4), except for `ServerConfig` which uses a plain text key.
- **Timestamps**: All timestamps are stored as INTEGER values representing Unix epoch seconds (not milliseconds, not ISO 8601 strings). See the Notes section for rationale.
- **Foreign keys**: Enforced via `FOREIGN KEY` constraints. `PRAGMA foreign_keys = ON` must be set on every connection.
- **Encrypted data**: Stored as BLOB. The server never interprets encrypted payloads.
- **Boolean values**: Represented as INTEGER (0 = false, 1 = true), following SQLite convention.

---

## Entity Relationship Diagram

```mermaid
erDiagram
    User ||--o{ Credential : "has"
    User ||--o{ Session : "has"
    User ||--o{ ConversationMember : "participates in"
    User ||--o{ Message : "sends"
    User ||--o{ KeyPackage : "uploads"
    Conversation ||--o{ ConversationMember : "has"
    Conversation ||--o{ Message : "contains"
    Conversation ||--|| MLSGroupState : "has"

    User {
        TEXT id PK "UUID"
        TEXT username UK "unique, lowercase"
        TEXT display_name "human-readable name"
        INTEGER created_at "unix timestamp"
        INTEGER updated_at "unix timestamp"
    }

    Credential {
        TEXT id PK "internal ID"
        TEXT user_id FK "references User.id"
        BLOB credential_id "WebAuthn credential ID"
        BLOB public_key "WebAuthn public key"
        INTEGER sign_count "replay counter"
        INTEGER created_at "unix timestamp"
    }

    Session {
        TEXT id PK "internal ID"
        TEXT user_id FK "references User.id"
        BLOB token_hash "SHA-256 of session token"
        INTEGER created_at "unix timestamp"
        INTEGER expires_at "unix timestamp"
        INTEGER last_seen_at "unix timestamp"
    }

    Conversation {
        TEXT id PK "UUID"
        TEXT type "1:1 or group"
        TEXT title "nullable, group name"
        INTEGER created_at "unix timestamp"
        INTEGER updated_at "unix timestamp"
    }

    ConversationMember {
        TEXT conversation_id PK_FK "references Conversation.id"
        TEXT user_id PK_FK "references User.id"
        TEXT role "member or admin"
        INTEGER joined_at "unix timestamp"
    }

    Message {
        TEXT id PK "UUID"
        TEXT conversation_id FK "references Conversation.id"
        TEXT sender_id FK "references User.id"
        BLOB encrypted_payload "MLS ciphertext"
        INTEGER server_timestamp "unix timestamp, server-assigned"
    }

    MLSGroupState {
        TEXT conversation_id PK_FK "references Conversation.id"
        BLOB group_id "MLS group identifier"
        INTEGER epoch "MLS epoch counter"
        BLOB state_data "serialized MLS group state"
        INTEGER updated_at "unix timestamp"
    }

    KeyPackage {
        TEXT id PK "internal ID"
        TEXT user_id FK "references User.id"
        BLOB key_package_data "serialized MLS KeyPackage"
        INTEGER uploaded_at "unix timestamp"
        INTEGER consumed "0 = available, 1 = consumed"
    }

    ServerConfig {
        TEXT key PK "config key"
        TEXT value "config value"
    }
```

---

## Table Definitions

### User

Stores registered users on this Sovereign server.

```sql
CREATE TABLE user (
    id          TEXT PRIMARY KEY,
    username    TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);

CREATE UNIQUE INDEX idx_user_username ON user (username);
```

### Credential

Stores WebAuthn/Passkey credentials. A user may have multiple credentials (e.g., multiple devices).

```sql
CREATE TABLE credential (
    id            TEXT PRIMARY KEY,
    user_id       TEXT NOT NULL,
    credential_id BLOB NOT NULL,
    public_key    BLOB NOT NULL,
    sign_count    INTEGER NOT NULL DEFAULT 0,
    created_at    INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES user (id) ON DELETE CASCADE
);

CREATE INDEX idx_credential_user_id ON credential (user_id);
CREATE UNIQUE INDEX idx_credential_credential_id ON credential (credential_id);
```

### Session

Stores active user sessions. The raw session token is never stored; only its SHA-256 hash.

```sql
CREATE TABLE session (
    id           TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL,
    token_hash   BLOB NOT NULL,
    created_at   INTEGER NOT NULL,
    expires_at   INTEGER NOT NULL,
    last_seen_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES user (id) ON DELETE CASCADE
);

CREATE INDEX idx_session_user_id ON session (user_id);
CREATE UNIQUE INDEX idx_session_token_hash ON session (token_hash);
CREATE INDEX idx_session_expires_at ON session (expires_at);
```

### Conversation

Represents a messaging conversation (1:1 or group).

```sql
CREATE TABLE conversation (
    id         TEXT PRIMARY KEY,
    type       TEXT NOT NULL CHECK (type IN ('1:1', 'group')),
    title      TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);
```

### ConversationMember

Join table linking users to conversations with role information.

```sql
CREATE TABLE conversation_member (
    conversation_id TEXT NOT NULL,
    user_id         TEXT NOT NULL,
    role            TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('member', 'admin')),
    joined_at       INTEGER NOT NULL,
    PRIMARY KEY (conversation_id, user_id),
    FOREIGN KEY (conversation_id) REFERENCES conversation (id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES user (id) ON DELETE CASCADE
);

CREATE INDEX idx_conversation_member_user_id ON conversation_member (user_id);
```

### Message

Stores encrypted messages. The `encrypted_payload` is an opaque blob that the server cannot interpret.

```sql
CREATE TABLE message (
    id                TEXT PRIMARY KEY,
    conversation_id   TEXT NOT NULL,
    sender_id         TEXT NOT NULL,
    encrypted_payload BLOB NOT NULL,
    server_timestamp  INTEGER NOT NULL,
    FOREIGN KEY (conversation_id) REFERENCES conversation (id) ON DELETE CASCADE,
    FOREIGN KEY (sender_id) REFERENCES user (id) ON DELETE SET NULL
);

CREATE INDEX idx_message_conversation_timestamp ON message (conversation_id, server_timestamp);
CREATE INDEX idx_message_sender_id ON message (sender_id);
```

### MLSGroupState

Stores server-side MLS group metadata for each conversation. The `state_data` contains serialized group state that the server may need for processing MLS protocol messages (e.g., validating commits, tracking epoch). This does NOT contain private key material.

```sql
CREATE TABLE mls_group_state (
    conversation_id TEXT PRIMARY KEY,
    group_id        BLOB NOT NULL,
    epoch           INTEGER NOT NULL DEFAULT 0,
    state_data      BLOB NOT NULL,
    updated_at      INTEGER NOT NULL,
    FOREIGN KEY (conversation_id) REFERENCES conversation (id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX idx_mls_group_state_group_id ON mls_group_state (group_id);
```

### KeyPackage

Stores MLS KeyPackages uploaded by users. Other users fetch these when creating a group or adding a member.

```sql
CREATE TABLE key_package (
    id               TEXT PRIMARY KEY,
    user_id          TEXT NOT NULL,
    key_package_data BLOB NOT NULL,
    uploaded_at      INTEGER NOT NULL,
    consumed         INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (user_id) REFERENCES user (id) ON DELETE CASCADE
);

CREATE INDEX idx_key_package_user_available ON key_package (user_id, consumed);
```

### ServerConfig

Key-value store for server configuration. Used for settings that can change at runtime (server display name, limits, feature flags).

```sql
CREATE TABLE server_config (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
```

---

## Indexes

### Performance-Critical Indexes

| Index | Table | Columns | Purpose |
|-------|-------|---------|---------|
| `idx_message_conversation_timestamp` | `message` | `(conversation_id, server_timestamp)` | Primary query pattern: fetch messages in a conversation ordered by time. Used for initial load and sync. |
| `idx_session_token_hash` | `session` | `(token_hash)` | Every WebSocket message triggers a session lookup by token hash. Must be fast. |
| `idx_credential_credential_id` | `credential` | `(credential_id)` | WebAuthn authentication requires lookup by credential ID. |
| `idx_key_package_user_available` | `key_package` | `(user_id, consumed)` | Fetching available key packages for a user when creating or joining a group. |
| `idx_conversation_member_user_id` | `conversation_member` | `(user_id)` | Listing all conversations for a user. |
| `idx_session_expires_at` | `session` | `(expires_at)` | Periodic cleanup of expired sessions. |

### Query Patterns

The following are the most common queries the server executes and the indexes that support them:

1. **Authenticate a WebSocket connection**: `SELECT * FROM session WHERE token_hash = ? AND expires_at > ?` — uses `idx_session_token_hash`.
2. **Fetch recent messages for a conversation**: `SELECT * FROM message WHERE conversation_id = ? AND server_timestamp > ? ORDER BY server_timestamp ASC LIMIT ?` — uses `idx_message_conversation_timestamp`.
3. **List conversations for a user**: `SELECT c.* FROM conversation c JOIN conversation_member cm ON c.id = cm.conversation_id WHERE cm.user_id = ?` — uses `idx_conversation_member_user_id`.
4. **Fetch available key packages for a user**: `SELECT * FROM key_package WHERE user_id = ? AND consumed = 0 LIMIT 1` — uses `idx_key_package_user_available`.
5. **WebAuthn credential lookup**: `SELECT * FROM credential WHERE credential_id = ?` — uses `idx_credential_credential_id`.

---

## Schema Migrations

Migrations are embedded in the server binary and run automatically on startup. The migration system uses a simple `schema_version` table:

```sql
CREATE TABLE IF NOT EXISTS schema_version (
    version    INTEGER PRIMARY KEY,
    applied_at INTEGER NOT NULL
);
```

Each migration is a sequentially numbered SQL file. The server checks the current version, then applies any unapplied migrations in order within a transaction.

---

## Notes

### Why Unix Integer Timestamps Instead of DATETIME

1. **SQLite has no native datetime type.** SQLite's `DATETIME` is stored as TEXT, REAL, or INTEGER internally. Using INTEGER explicitly avoids ambiguity about the storage format.
2. **Consistent sorting and comparison.** Integer comparison is unambiguous and fast. No timezone parsing, no locale issues, no format string bugs.
3. **Protocol Buffers compatibility.** Protobuf uses `int64` for timestamps (as seconds or milliseconds since epoch). Storing the same format in the database avoids conversion at the protocol boundary.
4. **Simplicity.** Every language and platform can trivially work with Unix timestamps. There is no risk of timezone-related bugs.

### Why encrypted_payload Is BLOB

The `encrypted_payload` column in the `message` table stores the raw MLS ciphertext bytes. It is BLOB (not TEXT) because:

1. **It is binary data.** MLS ciphertext is not valid UTF-8 or any text encoding. Storing it as TEXT would risk corruption or encoding errors.
2. **The server must not interpret it.** Using BLOB makes it explicit that this is opaque data the server merely stores and forwards.
3. **Efficiency.** BLOB storage avoids any encoding/decoding overhead (e.g., base64) that would be needed if stored as TEXT.

### Why UUIDs Are TEXT Not BLOB

While UUIDs are technically 16 bytes of binary data, they are stored as TEXT (the standard hyphenated string format, e.g., `550e8400-e29b-41d4-a716-446655440000`) for the following reasons:

1. **SQLite tooling compatibility.** TEXT UUIDs are human-readable in `sqlite3` CLI, DB Browser for SQLite, and other tools. BLOB UUIDs would appear as hex dumps.
2. **Debugging.** Log messages, error reports, and API responses all benefit from human-readable IDs.
3. **Protocol Buffers compatibility.** Protobuf represents UUIDs as `string`, not `bytes`. Storing as TEXT avoids conversion.
4. **Negligible overhead.** A TEXT UUID is 36 bytes vs. 16 bytes for a BLOB UUID. For the expected scale of a self-hosted server (hundreds, not millions of users), this difference is immaterial. A million messages would add roughly 20 MB of overhead, which is insignificant for modern storage.

### SQLite Configuration

The following PRAGMAs are set on every database connection:

```sql
PRAGMA journal_mode = WAL;          -- Write-ahead logging for concurrent reads
PRAGMA busy_timeout = 5000;         -- Wait up to 5 seconds for locks
PRAGMA synchronous = NORMAL;        -- Balance durability and performance
PRAGMA foreign_keys = ON;           -- Enforce foreign key constraints
PRAGMA cache_size = -64000;         -- 64 MB page cache
PRAGMA temp_store = MEMORY;         -- Keep temp tables in memory
```

WAL mode is essential because the server needs concurrent read access (multiple goroutines querying conversations) while a single writer inserts new messages. SQLite in WAL mode supports exactly this pattern.
