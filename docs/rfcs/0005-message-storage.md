# RFC-0005: Message Storage

- **Status**: Accepted
- **Author**: architect
- **Created**: 2026-02-16
- **Updated**: 2026-02-16
- **Review**: backend-engineer

## Summary

This RFC defines how encrypted messages are stored and retrieved on the Sovereign server. It covers the SQLite schema design, indexing strategy, cursor-based pagination, delivery tracking, offline message queuing, retention policies, and storage limits. A critical design principle: the server stores only encrypted payloads and never has access to plaintext message content.

## Motivation

The Sovereign server must persist messages for two primary reasons:

1. **Offline delivery**: When a recipient is not connected, their messages must be queued for delivery when they reconnect.
2. **History sync**: When a user gets a new device or reinstalls the app, they need to retrieve their message history.

Since all messages are end-to-end encrypted with MLS, the server stores opaque encrypted blobs. The storage layer does not need to understand message content — it only needs to store, index, retrieve, and expire these blobs efficiently.

## Detailed Design

### Storage Principles

1. **Encrypted only**: The server stores MLS-encrypted message payloads. It never sees or stores plaintext.
2. **Metadata minimal**: The server stores only the metadata necessary for routing and retrieval (group ID, sender ID, timestamp, message ID).
3. **Append-mostly**: Messages are appended. Updates are limited to delivery status tracking. Deletes happen only through retention policy enforcement.
4. **Server-authoritative ordering**: The server assigns timestamps and message IDs. These are the canonical ordering for message retrieval.

### Database Schema

```sql
-- Messages table: stores encrypted message blobs
CREATE TABLE messages (
    id              TEXT PRIMARY KEY,       -- Server-generated ULID (sortable, unique)
    group_id        TEXT NOT NULL,          -- MLS group identifier
    sender_id       TEXT NOT NULL,          -- User who sent the message
    server_timestamp INTEGER NOT NULL,      -- Server-assigned Unix microseconds
    payload         BLOB NOT NULL,          -- MLS-encrypted message (opaque to server)
    payload_size    INTEGER NOT NULL,       -- Size in bytes (for storage accounting)
    message_type    INTEGER NOT NULL DEFAULT 0,  -- 0=application, 1=commit, 2=welcome, 3=proposal
    epoch           INTEGER NOT NULL DEFAULT 0,  -- MLS epoch number
    created_at      INTEGER NOT NULL        -- Row creation time (Unix seconds)
);

-- Primary index for conversation retrieval (group + time ordering)
CREATE INDEX idx_messages_group_timestamp
    ON messages(group_id, server_timestamp);

-- Index for per-user message retrieval (offline delivery)
CREATE INDEX idx_messages_sender
    ON messages(sender_id, server_timestamp);

-- Index for retention policy cleanup
CREATE INDEX idx_messages_created_at
    ON messages(created_at);

-- Index for storage accounting
CREATE INDEX idx_messages_group_size
    ON messages(group_id, payload_size);


-- Delivery tracking: per-recipient delivery state
CREATE TABLE delivery_status (
    message_id      TEXT NOT NULL,          -- References messages.id
    recipient_id    TEXT NOT NULL,          -- Target user
    status          INTEGER NOT NULL DEFAULT 0,  -- 0=pending, 1=delivered, 2=read
    delivered_at    INTEGER,               -- Unix microseconds, NULL if not yet delivered
    read_at         INTEGER,               -- Unix microseconds, NULL if not yet read
    PRIMARY KEY (message_id, recipient_id),
    FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE
);

-- Index for finding undelivered messages for a user
CREATE INDEX idx_delivery_pending
    ON delivery_status(recipient_id, status)
    WHERE status = 0;


-- Group membership: tracks which users are in which groups
-- (Server-side view, derived from MLS operations)
CREATE TABLE group_members (
    group_id        TEXT NOT NULL,
    user_id         TEXT NOT NULL,
    joined_at       INTEGER NOT NULL,       -- Unix seconds
    PRIMARY KEY (group_id, user_id)
);

CREATE INDEX idx_group_members_user
    ON group_members(user_id);
```

### Message IDs: ULIDs

Messages are identified by [ULIDs](https://github.com/ulid/spec) (Universally Unique Lexicographically Sortable Identifiers). ULIDs are:

- **Sortable**: Lexicographic sort order matches chronological order.
- **Unique**: 128-bit with millisecond timestamp prefix and 80 bits of randomness.
- **Compact**: 26-character base32 encoding.

Using ULIDs as the primary key means the primary index is naturally sorted by creation time, which aligns with the most common access pattern (recent messages first).

### Message Retrieval: Cursor-Based Pagination

Clients retrieve messages using cursor-based pagination, which is more efficient and consistent than offset-based pagination for append-heavy workloads.

**Request (via WebSocket):**

```protobuf
message HistoryRequest {
  string group_id = 1;          // Which conversation
  string cursor = 2;            // Last message ID from previous page (empty for latest)
  int32 limit = 3;              // Max messages to return (default 50, max 200)
  Direction direction = 4;      // BACKWARD (older) or FORWARD (newer)
}

enum Direction {
  BACKWARD = 0;  // Older messages (default)
  FORWARD = 1;   // Newer messages
}
```

**Response:**

```protobuf
message HistoryResponse {
  repeated StoredMessage messages = 1;  // Ordered by server_timestamp
  string next_cursor = 2;              // Cursor for next page (empty if no more)
  bool has_more = 3;                   // Whether more messages exist
}

message StoredMessage {
  string id = 1;                      // ULID
  string group_id = 2;
  string sender_id = 3;
  int64 server_timestamp = 4;
  bytes payload = 5;                  // Encrypted blob
  int32 message_type = 6;
}
```

**SQL query (backward pagination):**

```sql
SELECT id, group_id, sender_id, server_timestamp, payload, message_type
FROM messages
WHERE group_id = ?
  AND id < ?           -- cursor: messages older than cursor
ORDER BY id DESC       -- newest first
LIMIT ?;               -- page size
```

The `idx_messages_group_timestamp` index makes this query efficient. The ULID primary key's sortability means `id < cursor` correctly filters chronologically.

### Delivery Tracking

The server tracks delivery state for each message-recipient pair:

```
                ┌─────────┐    Client connects,     ┌───────────┐    Client sends    ┌──────┐
Message sent    │ PENDING │    message forwarded     │ DELIVERED │    read receipt    │ READ │
──────────────► │  (0)    │ ──────────────────────►  │    (1)    │ ────────────────►  │ (2)  │
                └─────────┘                          └───────────┘                    └──────┘
```

**Delivery flow:**

1. Server receives a message and inserts it into `messages`.
2. Server creates a `delivery_status` row (status=PENDING) for each group member except the sender.
3. For each currently connected recipient, the server forwards the message and updates status to DELIVERED.
4. For offline recipients, the status remains PENDING.
5. When an offline recipient connects, the server queries for PENDING messages and delivers them, updating status to DELIVERED.
6. When the client sends a read receipt (`MSG_READ_RECEIPT`), the server updates the status to READ.

**Offline delivery query:**

```sql
SELECT m.id, m.group_id, m.sender_id, m.server_timestamp, m.payload, m.message_type
FROM delivery_status ds
JOIN messages m ON m.id = ds.message_id
WHERE ds.recipient_id = ?
  AND ds.status = 0  -- PENDING
ORDER BY m.server_timestamp ASC;
```

### Retention Policy

The server supports configurable message retention to manage storage growth:

```yaml
# Server configuration
storage:
  retention_days: 90          # Delete messages older than 90 days (0 = keep forever)
  max_storage_mb: 1024        # Maximum total storage in MB (0 = unlimited)
  cleanup_interval_hours: 6   # How often to run cleanup
```

**Retention enforcement:**

A background goroutine runs periodically and:

1. Deletes messages older than `retention_days`:
   ```sql
   DELETE FROM messages
   WHERE created_at < ? -- (now - retention_days)
   ```
   The `ON DELETE CASCADE` on `delivery_status` automatically cleans up delivery records.

2. If `max_storage_mb` is set and total storage exceeds the limit, deletes the oldest messages until storage is below the limit:
   ```sql
   DELETE FROM messages
   WHERE id IN (
     SELECT id FROM messages
     ORDER BY created_at ASC
     LIMIT ?  -- batch size
   );
   ```

3. Runs `VACUUM` periodically (e.g., weekly) to reclaim disk space from deleted rows.

### Storage Limits

Per-message and per-group limits prevent abuse:

| Limit | Default | Configurable |
|-------|---------|-------------|
| Max message payload size | 256 KB | Yes |
| Max messages per group per day | 10,000 | Yes |
| Max total storage | 1 GB | Yes |
| Max groups per user | 500 | Yes |

These limits are enforced at the WebSocket handler level before messages reach the storage layer.

### MLS Control Messages

MLS control messages (Commits, Proposals, Welcomes) are stored in the same `messages` table with a different `message_type` value. This ensures:

- Clients can retrieve missed MLS operations during offline catchup.
- The Commit history is available for clients that need to fast-forward through missed epochs.
- Retention policies apply uniformly to all message types.

Control messages are not subject to the same rate limits as application messages (see RFC-0002) because they are essential for group state consistency.

## Security Considerations

- **Encrypted-at-rest**: The server stores only MLS-encrypted payloads. Even without additional at-rest encryption, message content is protected. For defense-in-depth, server operators are encouraged to use full-disk encryption (LUKS, FileVault) on the server's filesystem.

- **Metadata minimization**: The server stores the minimum metadata needed for routing: group ID, sender ID, timestamp, and message type. It does not store recipient lists in the message record (this is derived from group membership). Message content types, reactions, and other application-level data are inside the encrypted payload and invisible to the server.

- **Deletion semantics**: When messages are deleted (via retention policy), both the encrypted payload and metadata are removed (`DELETE` from SQLite). After `VACUUM`, the data is physically removed from the database file. However, this does not guarantee deletion from filesystem-level artifacts (journal files, backups, SSD wear leveling). Server operators should use encrypted filesystems for stronger deletion guarantees.

- **Delivery status as metadata**: The delivery tracking system reveals when a user received and read a message. This is inherent to delivery receipts. Users should be able to opt out of read receipts (client-side configuration, with the client simply not sending `MSG_READ_RECEIPT`).

- **Storage-based denial of service**: Without limits, a malicious user could flood the server with messages to exhaust storage. Per-group rate limits and total storage caps mitigate this. The server should monitor storage usage and alert operators when approaching limits.

- **Timing attacks on delivery status**: The pattern of PENDING-to-DELIVERED transitions reveals when users come online. This is a metadata leakage concern. Mitigation is deferred (potential approaches: delayed delivery status updates, batched status changes).

## Alternatives Considered

- **Separate storage for MLS control messages**: Store Commits, Proposals, and Welcomes in a separate table from application messages. Rejected because the access patterns are the same (retrieve by group + time range), and a unified table simplifies the storage layer.

- **Client-side storage only (no server history)**: Do not store messages on the server; deliver them once and discard. This eliminates server-side storage concerns but makes message history sync (new devices) and offline delivery significantly more complex. Rejected in favor of server-side encrypted blob storage.

- **Object storage (S3-compatible) for payloads**: Store encrypted blobs in S3/MinIO and only metadata in SQLite. This would scale storage better but adds an external dependency, contradicting the single-binary goal. Rejected for the initial design; could be revisited if storage requirements grow significantly.

- **Offset-based pagination**: Simpler to implement but suffers from inconsistency when new messages are inserted between page requests (items shift, leading to duplicates or gaps). Cursor-based pagination is more reliable for append-heavy workloads.

## Open Questions

- **Message editing and deletion by sender**: Should the protocol support sender-initiated message editing or deletion? This requires careful design: the server would store an "edit" or "delete" marker, but the original encrypted content has already been delivered to recipients. This is deferred to a future RFC.

- **Media/file storage**: Large files (images, videos, documents) should not be inlined in the message payload (which is limited to 256 KB). A separate encrypted file upload/download mechanism is needed. Deferred to a future RFC.

- **Database sharding**: For extremely large deployments, a single SQLite file may become a bottleneck. Options include per-group database files or migration to a different storage backend. Deferred — the current design is sufficient for the target scale.

- **Backup and restore**: How should server operators back up the message database? Simple file copy (with WAL checkpoint) works for SQLite, but a more robust solution (streaming backup, point-in-time recovery) may be desirable.

## References

- [ADR-0004: SQLite for Database](../adrs/0004-sqlite-for-database.md)
- [ADR-0009: modernc.org/sqlite](../adrs/0009-modernc-sqlite.md)
- [RFC-0002: WebSocket Protocol](./0002-websocket-protocol.md)
- [RFC-0003: MLS Integration](./0003-mls-integration.md)
- [ULID Specification](https://github.com/ulid/spec)
- [SQLite WAL Mode](https://www.sqlite.org/wal.html)
