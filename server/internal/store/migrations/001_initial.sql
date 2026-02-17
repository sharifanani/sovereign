-- 001_initial.sql
-- Initial database schema for Sovereign

PRAGMA journal_mode=WAL;
PRAGMA foreign_keys=ON;

-- Users
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    username TEXT UNIQUE NOT NULL,
    display_name TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- WebAuthn credentials
CREATE TABLE IF NOT EXISTS credentials (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credential_id BLOB NOT NULL,
    public_key BLOB NOT NULL,
    sign_count INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_credentials_user_id ON credentials(user_id);

-- Sessions
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash BLOB NOT NULL,
    created_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL,
    last_seen_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);

-- Conversations (1:1 and group)
CREATE TABLE IF NOT EXISTS conversations (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL CHECK(type IN ('1:1', 'group')),
    title TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- Conversation members
CREATE TABLE IF NOT EXISTS conversation_members (
    conversation_id TEXT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL DEFAULT 'member' CHECK(role IN ('member', 'admin')),
    joined_at INTEGER NOT NULL,
    PRIMARY KEY (conversation_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_conversation_members_user_id ON conversation_members(user_id);

-- Messages (encrypted payloads only)
CREATE TABLE IF NOT EXISTS messages (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id TEXT NOT NULL REFERENCES users(id),
    encrypted_payload BLOB NOT NULL,
    server_timestamp INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_messages_conversation_timestamp ON messages(conversation_id, server_timestamp);

-- MLS group state
CREATE TABLE IF NOT EXISTS mls_group_state (
    conversation_id TEXT PRIMARY KEY REFERENCES conversations(id) ON DELETE CASCADE,
    group_id BLOB NOT NULL,
    epoch INTEGER NOT NULL DEFAULT 0,
    state_data BLOB NOT NULL,
    updated_at INTEGER NOT NULL
);

-- MLS key packages
CREATE TABLE IF NOT EXISTS key_packages (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key_package_data BLOB NOT NULL,
    uploaded_at INTEGER NOT NULL,
    consumed INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_key_packages_user_id_consumed ON key_packages(user_id, consumed);

-- Server configuration (key-value store)
CREATE TABLE IF NOT EXISTS server_config (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
