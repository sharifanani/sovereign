# Sovereign Admin REST API

**Version**: 1.0
**Last Updated**: 2026-02-16
**Base Path**: `/admin/api/`

---

## Overview

The Admin REST API provides server administration capabilities for the Sovereign messaging server. All endpoints are served over HTTPS and require admin authentication.

### Authentication

All endpoints require a valid admin session. Authentication is provided via one of:

- **Cookie**: `sovereign_admin_session` cookie containing the session token.
- **Header**: `Authorization: Bearer <session_token>` header.

Unauthenticated requests receive a `401 Unauthorized` response. Requests from non-admin users receive a `403 Forbidden` response.

### Common Response Format

All responses use JSON with the following wrapper for errors:

```json
{
  "error": {
    "code": 2003,
    "message": "Not authorized as admin"
  }
}
```

Successful responses return the data directly (not wrapped in a success envelope) unless otherwise noted.

### Pagination

List endpoints support pagination via query parameters:

| Parameter | Type  | Default | Description                    |
|----------|-------|---------|--------------------------------|
| `offset` | `int` | `0`     | Number of records to skip.     |
| `limit`  | `int` | `20`    | Maximum records to return (max 100). |

Paginated responses include pagination metadata:

```json
{
  "data": [...],
  "pagination": {
    "offset": 0,
    "limit": 20,
    "total": 142
  }
}
```

---

## Server Info

### GET /admin/api/info

Returns current server status and statistics.

**Request**: No body required.

**Response** (`200 OK`):

```json
{
  "server_name": "My Sovereign Server",
  "version": "1.2.0",
  "uptime_seconds": 86400,
  "active_connections": 47,
  "total_users": 156,
  "total_messages": 23841,
  "started_at": "2026-02-15T10:30:00Z"
}
```

| Field                | Type     | Description                                      |
|---------------------|----------|--------------------------------------------------|
| `server_name`       | `string` | Configured display name of the server.           |
| `version`           | `string` | Server software version.                         |
| `uptime_seconds`    | `int`    | Seconds since server started.                    |
| `active_connections`| `int`    | Number of currently active WebSocket connections. |
| `total_users`       | `int`    | Total registered users.                          |
| `total_messages`    | `int`    | Total messages processed by the server.          |
| `started_at`        | `string` | ISO 8601 timestamp of when the server started.   |

**Error Responses**:

| Status | Description                    |
|--------|--------------------------------|
| `401`  | Not authenticated.             |
| `403`  | Authenticated but not an admin.|

---

## User Management

### GET /admin/api/users

List all registered users with pagination.

**Query Parameters**:

| Parameter | Type     | Default | Description                              |
|----------|----------|---------|------------------------------------------|
| `offset` | `int`    | `0`     | Number of records to skip.               |
| `limit`  | `int`    | `20`    | Maximum records to return (max 100).     |
| `search` | `string` | —       | Optional. Filter by username or display name (case-insensitive substring match). |

**Response** (`200 OK`):

```json
{
  "data": [
    {
      "user_id": "usr_01H8X9KPQR",
      "username": "alice",
      "display_name": "Alice Smith",
      "enabled": true,
      "created_at": "2026-01-10T14:22:00Z",
      "last_seen_at": "2026-02-16T08:15:33Z",
      "credential_count": 2
    },
    {
      "user_id": "usr_01H8X9KPQS",
      "username": "bob",
      "display_name": "Bob Jones",
      "enabled": true,
      "created_at": "2026-01-11T09:05:00Z",
      "last_seen_at": "2026-02-16T07:42:11Z",
      "credential_count": 1
    }
  ],
  "pagination": {
    "offset": 0,
    "limit": 20,
    "total": 156
  }
}
```

| Field              | Type     | Description                                   |
|-------------------|----------|-----------------------------------------------|
| `user_id`         | `string` | Unique user identifier.                       |
| `username`        | `string` | User's login username.                        |
| `display_name`    | `string` | User's display name.                          |
| `enabled`         | `bool`   | Whether the user account is active.           |
| `created_at`      | `string` | ISO 8601 timestamp of account creation.       |
| `last_seen_at`    | `string` | ISO 8601 timestamp of last activity. Null if never connected. |
| `credential_count`| `int`    | Number of registered WebAuthn credentials.    |

**Error Responses**:

| Status | Description                    |
|--------|--------------------------------|
| `401`  | Not authenticated.             |
| `403`  | Authenticated but not an admin.|

---

### GET /admin/api/users/:id

Get detailed information about a specific user.

**Path Parameters**:

| Parameter | Type     | Description             |
|----------|----------|-------------------------|
| `id`     | `string` | The user's unique ID.   |

**Response** (`200 OK`):

```json
{
  "user_id": "usr_01H8X9KPQR",
  "username": "alice",
  "display_name": "Alice Smith",
  "enabled": true,
  "is_admin": false,
  "created_at": "2026-01-10T14:22:00Z",
  "last_seen_at": "2026-02-16T08:15:33Z",
  "credentials": [
    {
      "credential_id": "cred_ABCdef123",
      "created_at": "2026-01-10T14:22:00Z",
      "last_used_at": "2026-02-16T08:15:33Z",
      "device_name": "MacBook Pro"
    }
  ],
  "conversations": [
    {
      "conversation_id": "conv_XYZ789",
      "title": "Project Chat",
      "member_count": 5,
      "last_message_at": "2026-02-16T08:10:00Z"
    }
  ],
  "active_sessions": 1,
  "key_package_count": 3,
  "message_count": 412
}
```

| Field               | Type       | Description                                    |
|--------------------|------------|------------------------------------------------|
| `user_id`          | `string`   | Unique user identifier.                        |
| `username`         | `string`   | User's login username.                         |
| `display_name`     | `string`   | User's display name.                           |
| `enabled`          | `bool`     | Whether the user account is active.            |
| `is_admin`         | `bool`     | Whether the user has admin privileges.         |
| `created_at`       | `string`   | ISO 8601 timestamp of account creation.        |
| `last_seen_at`     | `string`   | ISO 8601 timestamp of last activity.           |
| `credentials`      | `array`    | List of registered WebAuthn credentials.       |
| `conversations`    | `array`    | List of conversations the user is a member of. |
| `active_sessions`  | `int`      | Number of currently active sessions.           |
| `key_package_count`| `int`      | Number of available MLS KeyPackages.           |
| `message_count`    | `int`      | Total messages sent by this user.              |

**Error Responses**:

| Status | Description                    |
|--------|--------------------------------|
| `401`  | Not authenticated.             |
| `403`  | Authenticated but not an admin.|
| `404`  | User not found.                |

---

### PUT /admin/api/users/:id

Update a user's profile or account status.

**Path Parameters**:

| Parameter | Type     | Description             |
|----------|----------|-------------------------|
| `id`     | `string` | The user's unique ID.   |

**Request Body** (`application/json`):

```json
{
  "display_name": "Alice Johnson",
  "enabled": false
}
```

| Field          | Type     | Required | Description                                     |
|---------------|----------|----------|-------------------------------------------------|
| `display_name`| `string` | No       | New display name for the user.                  |
| `enabled`     | `bool`   | No       | Set to `false` to disable the account. Disabled accounts cannot authenticate and active sessions are revoked immediately. |

All fields are optional. Only provided fields are updated.

**Response** (`200 OK`):

```json
{
  "user_id": "usr_01H8X9KPQR",
  "username": "alice",
  "display_name": "Alice Johnson",
  "enabled": false,
  "updated_at": "2026-02-16T12:00:00Z"
}
```

**Error Responses**:

| Status | Description                                    |
|--------|------------------------------------------------|
| `400`  | Invalid request body.                          |
| `401`  | Not authenticated.                             |
| `403`  | Authenticated but not an admin.                |
| `404`  | User not found.                                |
| `422`  | Validation error (e.g., display_name too long).|

---

### DELETE /admin/api/users/:id

Delete a user and all associated data. This action is irreversible.

**Path Parameters**:

| Parameter | Type     | Description             |
|----------|----------|-------------------------|
| `id`     | `string` | The user's unique ID.   |

**Request**: No body required.

**Response** (`200 OK`):

```json
{
  "deleted": true,
  "user_id": "usr_01H8X9KPQR",
  "deleted_data": {
    "sessions_revoked": 2,
    "credentials_removed": 1,
    "key_packages_removed": 3,
    "conversations_affected": 4
  }
}
```

| Field                   | Type   | Description                                     |
|------------------------|--------|-------------------------------------------------|
| `deleted`              | `bool` | Always `true` on success.                       |
| `user_id`              | `string`| The deleted user's ID.                          |
| `deleted_data`         | `object`| Summary of data that was removed.               |
| `.sessions_revoked`    | `int`  | Number of active sessions that were terminated. |
| `.credentials_removed` | `int`  | Number of WebAuthn credentials removed.         |
| `.key_packages_removed`| `int`  | Number of MLS KeyPackages removed.              |
| `.conversations_affected`| `int`| Number of conversations the user was removed from. |

**Behavior**:
- All active sessions are immediately revoked (WebSocket connections closed).
- All WebAuthn credentials are deleted.
- All MLS KeyPackages are deleted.
- The user is removed from all group conversations (other members receive `group.member_removed` notifications).
- Message history is retained for other group members but the sender is marked as `[deleted user]`.

**Error Responses**:

| Status | Description                                         |
|--------|-----------------------------------------------------|
| `401`  | Not authenticated.                                  |
| `403`  | Authenticated but not an admin.                     |
| `404`  | User not found.                                     |
| `409`  | Cannot delete the last admin account.               |

---

## Server Settings

### GET /admin/api/settings

Retrieve all server configuration settings.

**Request**: No body required.

**Response** (`200 OK`):

```json
{
  "server_name": "My Sovereign Server",
  "max_connections": 10000,
  "max_connections_per_user": 5,
  "rate_limit_per_second": 30,
  "rate_limit_burst": 10,
  "max_message_size_bytes": 65536,
  "session_timeout_hours": 720,
  "registration_enabled": true,
  "min_key_packages": 5
}
```

| Field                      | Type     | Description                                               |
|---------------------------|----------|-----------------------------------------------------------|
| `server_name`             | `string` | Display name of the server.                               |
| `max_connections`         | `int`    | Maximum total concurrent WebSocket connections.            |
| `max_connections_per_user`| `int`    | Maximum concurrent connections per user.                   |
| `rate_limit_per_second`   | `int`    | Maximum messages per second per connection.                |
| `rate_limit_burst`        | `int`    | Burst allowance for rate limiting.                         |
| `max_message_size_bytes`  | `int`    | Maximum size of a single Envelope in bytes.                |
| `session_timeout_hours`   | `int`    | Hours before an idle session expires.                      |
| `registration_enabled`    | `bool`   | Whether new user registration is open.                     |
| `min_key_packages`        | `int`    | Minimum KeyPackages a client should maintain on the server.|

**Error Responses**:

| Status | Description                    |
|--------|--------------------------------|
| `401`  | Not authenticated.             |
| `403`  | Authenticated but not an admin.|

---

### PUT /admin/api/settings

Update server configuration settings. Only provided fields are updated.

**Request Body** (`application/json`):

```json
{
  "server_name": "Sovereign HQ",
  "max_connections": 5000,
  "rate_limit_per_second": 50,
  "registration_enabled": false
}
```

All fields are optional. Only provided fields are updated. See the GET endpoint for field descriptions.

**Response** (`200 OK`):

Returns the full settings object (same format as GET) reflecting the updated values.

```json
{
  "server_name": "Sovereign HQ",
  "max_connections": 5000,
  "max_connections_per_user": 5,
  "rate_limit_per_second": 50,
  "rate_limit_burst": 10,
  "max_message_size_bytes": 65536,
  "session_timeout_hours": 720,
  "registration_enabled": false,
  "min_key_packages": 5
}
```

**Behavior**:
- Settings changes take effect immediately for new connections.
- Existing connections are not affected by `max_connections` or `rate_limit` changes until they reconnect.
- Changing `registration_enabled` to `false` immediately prevents new registrations.

**Error Responses**:

| Status | Description                                           |
|--------|-------------------------------------------------------|
| `400`  | Invalid request body or invalid setting value.        |
| `401`  | Not authenticated.                                    |
| `403`  | Authenticated but not an admin.                       |
| `422`  | Validation error (e.g., max_connections below current).|

---

## Sessions

### GET /admin/api/sessions

List all active sessions.

**Query Parameters**:

| Parameter | Type     | Default | Description                              |
|----------|----------|---------|------------------------------------------|
| `offset` | `int`    | `0`     | Number of records to skip.               |
| `limit`  | `int`    | `20`    | Maximum records to return (max 100).     |
| `user_id`| `string` | —       | Optional. Filter sessions by user ID.    |

**Response** (`200 OK`):

```json
{
  "data": [
    {
      "session_id": "sess_ABC123",
      "user_id": "usr_01H8X9KPQR",
      "username": "alice",
      "created_at": "2026-02-16T08:00:00Z",
      "last_active_at": "2026-02-16T08:15:33Z",
      "ip_address": "192.168.1.42",
      "user_agent": "Sovereign/1.0 (macOS)",
      "connected": true
    },
    {
      "session_id": "sess_DEF456",
      "user_id": "usr_01H8X9KPQS",
      "username": "bob",
      "created_at": "2026-02-15T18:30:00Z",
      "last_active_at": "2026-02-16T07:42:11Z",
      "ip_address": "10.0.0.15",
      "user_agent": "Sovereign/1.0 (Windows)",
      "connected": true
    }
  ],
  "pagination": {
    "offset": 0,
    "limit": 20,
    "total": 53
  }
}
```

| Field            | Type     | Description                                       |
|-----------------|----------|---------------------------------------------------|
| `session_id`    | `string` | Unique session identifier.                        |
| `user_id`       | `string` | The user this session belongs to.                 |
| `username`      | `string` | The user's username.                              |
| `created_at`    | `string` | ISO 8601 timestamp of session creation.           |
| `last_active_at`| `string` | ISO 8601 timestamp of last activity.              |
| `ip_address`    | `string` | Client IP address.                                |
| `user_agent`    | `string` | Client user agent string.                         |
| `connected`     | `bool`   | Whether a WebSocket connection is currently active.|

**Error Responses**:

| Status | Description                    |
|--------|--------------------------------|
| `401`  | Not authenticated.             |
| `403`  | Authenticated but not an admin.|

---

### DELETE /admin/api/sessions/:id

Revoke a specific session, immediately disconnecting the client if connected.

**Path Parameters**:

| Parameter | Type     | Description                  |
|----------|----------|------------------------------|
| `id`     | `string` | The session ID to revoke.    |

**Request**: No body required.

**Response** (`200 OK`):

```json
{
  "revoked": true,
  "session_id": "sess_ABC123",
  "user_id": "usr_01H8X9KPQR",
  "was_connected": true
}
```

| Field            | Type     | Description                                              |
|-----------------|----------|----------------------------------------------------------|
| `revoked`       | `bool`   | Always `true` on success.                                |
| `session_id`    | `string` | The revoked session ID.                                  |
| `user_id`       | `string` | The user whose session was revoked.                      |
| `was_connected` | `bool`   | Whether the session had an active WebSocket connection.   |

**Behavior**:
- The session token is immediately invalidated.
- If the session has an active WebSocket connection, it is closed with code `4004 (Session Expired)` and an error message with code `1005 (SessionRevoked)` is sent before closing.
- The user must re-authenticate to establish a new session.

**Error Responses**:

| Status | Description                    |
|--------|--------------------------------|
| `401`  | Not authenticated.             |
| `403`  | Authenticated but not an admin.|
| `404`  | Session not found.             |
