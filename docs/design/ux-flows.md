# UX Flows

## Overview

This document describes the user-facing journeys through Sovereign, from server setup to daily messaging. Each flow is described as a step-by-step sequence with the actors, preconditions, and outcomes specified.

---

## 1. Server Setup (CLI Wizard)

**Actor**: Server operator (self-hosting user).

**Precondition**: The operator has downloaded the `sovereign` and `sovereign-cli` binaries to their machine.

**Goal**: Initialize the Sovereign server with a configuration file, database, and admin account.

### Flow

```
Step 1: Run the wizard
    $ ./sovereign-cli setup

    ┌─────────────────────────────────────────────┐
    │  Sovereign Server Setup                      │
    │  ─────────────────────────────────────────── │
    │  This wizard will configure your server.     │
    └─────────────────────────────────────────────┘

Step 2: Server display name
    > Enter a display name for your server: My Home Server

    This name is shown to users when they connect.

Step 3: Listen address and port
    > Listen address [default: :8080]:

    The server will accept connections on this address.
    For external access, ensure this port is forwarded on your router.

Step 4: Data directory
    > Data directory [default: ./data]:

    The SQLite database and config file will be stored here.

Step 5: TLS configuration
    > Enable TLS? (y/n) [default: n]: y
    > Path to TLS certificate: /path/to/cert.pem
    > Path to TLS private key: /path/to/key.pem

    If TLS is disabled, the server will listen on plain HTTP/WS.
    You should use a reverse proxy (e.g., Caddy, nginx) for TLS
    termination in production.

Step 6: Create admin user
    > Admin username: alice
    > Admin display name: Alice

Step 7: Register admin passkey
    ┌─────────────────────────────────────────────┐
    │  Passkey Registration                        │
    │  ─────────────────────────────────────────── │
    │  The wizard will start a temporary local     │
    │  web server for passkey registration.        │
    │                                              │
    │  Open this URL in your browser:              │
    │  http://localhost:9090/setup/passkey          │
    │                                              │
    │  Follow the prompts to register your         │
    │  passkey (biometric, PIN, or security key).  │
    └─────────────────────────────────────────────┘

    The operator opens the URL in their browser. The browser
    prompts for passkey creation (e.g., Touch ID, Face ID,
    Windows Hello, or a hardware security key). The browser
    sends the attestation response back to the temporary server.

    > Passkey registered successfully.

Step 8: Generate configuration
    The wizard writes:
    - data/sovereign.toml      (server configuration)
    - data/sovereign.db        (initialized SQLite database with schema)
    - Admin user and credential stored in the database

    ┌─────────────────────────────────────────────┐
    │  Setup Complete                              │
    │  ─────────────────────────────────────────── │
    │  Server name:   My Home Server               │
    │  Listen:        :8080                        │
    │  Admin user:    alice                        │
    │  Database:      data/sovereign.db            │
    │  Config:        data/sovereign.toml          │
    │                                              │
    │  Start your server:                          │
    │    $ ./sovereign --config data/sovereign.toml│
    │                                              │
    │  Admin panel will be available at:            │
    │    http://localhost:8080/admin                │
    └─────────────────────────────────────────────┘
```

### Outcome

- A config file exists at the specified path.
- The SQLite database is initialized with all tables.
- An admin user exists with a registered passkey credential.
- The server is ready to start.

---

## 2. User Registration

**Actor**: New user on an existing, running Sovereign server.

**Precondition**: The Sovereign server is running and reachable. The user has the Sovereign mobile app installed.

**Goal**: Create an account on the server and authenticate.

### Flow

```
Step 1: Add server
    User opens the Sovereign app.
    The app shows the server list (empty for a new user).
    User taps "Add Server".

Step 2: Enter server URL
    ┌─────────────────────────────────────────────┐
    │  Add Server                                  │
    │                                              │
    │  Server URL:                                 │
    │  ┌─────────────────────────────────────────┐ │
    │  │ sovereign.example.com                    │ │
    │  └─────────────────────────────────────────┘ │
    │                                              │
    │  [ Connect ]                                 │
    └─────────────────────────────────────────────┘

    User enters the server's URL and taps "Connect".
    The app establishes a WebSocket connection to the server.

Step 3: Receive server info
    The server responds with its display name and capabilities.

    ┌─────────────────────────────────────────────┐
    │  My Home Server                              │
    │  sovereign.example.com                       │
    │                                              │
    │  [ Register ]     [ Login ]                  │
    └─────────────────────────────────────────────┘

    User taps "Register".

Step 4: Enter user information
    ┌─────────────────────────────────────────────┐
    │  Create Account                              │
    │                                              │
    │  Username:                                   │
    │  ┌─────────────────────────────────────────┐ │
    │  │ bob                                      │ │
    │  └─────────────────────────────────────────┘ │
    │                                              │
    │  Display Name:                               │
    │  ┌─────────────────────────────────────────┐ │
    │  │ Bob Smith                                │ │
    │  └─────────────────────────────────────────┘ │
    │                                              │
    │  [ Continue ]                                │
    └─────────────────────────────────────────────┘

Step 5: Passkey creation
    The app initiates the WebAuthn registration flow.
    The OS presents the passkey creation prompt:

    ┌─────────────────────────────────────────────┐
    │  Create a passkey for                        │
    │  sovereign.example.com?                      │
    │                                              │
    │  Your passkey will be saved to iCloud        │
    │  Keychain and available on all your          │
    │  devices.                                    │
    │                                              │
    │  [ Continue with Face ID ]                   │
    └─────────────────────────────────────────────┘

    User authenticates with biometric (Face ID, Touch ID,
    fingerprint) or PIN.

Step 6: Server creates account
    The server:
    - Verifies the WebAuthn attestation response.
    - Creates the User record (id, username, display_name).
    - Stores the Credential (public key, credential ID).
    - Creates a Session and returns the session token.

Step 7: Registration complete
    The app:
    - Stores the session token securely (device keychain).
    - Generates MLS key material (identity key, leaf key).
    - Uploads initial key packages to the server.
    - Navigates to the conversation list.

    ┌─────────────────────────────────────────────┐
    │  My Home Server                              │
    │  ─────────────────────────────────────────── │
    │                                              │
    │  No conversations yet.                       │
    │  Tap + to start a new message.               │
    │                                              │
    │                           [ + New Message ]  │
    └─────────────────────────────────────────────┘
```

### Outcome

- The user has an account on the server.
- A passkey credential is stored on their device and the public key is on the server.
- The user is authenticated with a valid session.
- MLS key packages are uploaded and available for other users to fetch.

---

## 3. User Login

**Actor**: Returning user who already has an account on the server.

**Precondition**: The user previously registered on this server and has their passkey available.

**Goal**: Authenticate and resume messaging.

### Flow

```
Step 1: Select server
    User opens the Sovereign app.
    The app shows the server list.
    User taps on the server they want to connect to.

Step 2: Initiate login
    ┌─────────────────────────────────────────────┐
    │  My Home Server                              │
    │  sovereign.example.com                       │
    │                                              │
    │  Welcome back.                               │
    │                                              │
    │  [ Login with Passkey ]                      │
    └─────────────────────────────────────────────┘

    User taps "Login with Passkey".

Step 3: WebAuthn challenge
    The app sends a login request to the server.
    The server generates a WebAuthn challenge and returns
    PublicKeyCredentialRequestOptions (including the
    allowCredentials list for this user's registered credentials).

Step 4: Passkey authentication
    The OS presents the passkey prompt:

    ┌─────────────────────────────────────────────┐
    │  Sign in to                                  │
    │  sovereign.example.com                       │
    │                                              │
    │  bob                                         │
    │                                              │
    │  [ Continue with Face ID ]                   │
    └─────────────────────────────────────────────┘

    User authenticates with biometric or PIN.

Step 5: Server verifies
    The server:
    - Verifies the assertion signature against the stored public key.
    - Checks and updates the sign_count.
    - Creates a new Session and returns the session token.

Step 6: Login complete
    The app:
    - Stores the session token.
    - Connects WebSocket with the token.
    - Syncs any missed messages since last connection.
    - Navigates to the conversation list with updated conversations.

    ┌─────────────────────────────────────────────┐
    │  My Home Server                              │
    │  ─────────────────────────────────────────── │
    │                                              │
    │  ┌─────────────────────────────────────────┐ │
    │  │ Alice                           2m ago  │ │
    │  │ Hey, are you coming tonight?            │ │
    │  └─────────────────────────────────────────┘ │
    │  ┌─────────────────────────────────────────┐ │
    │  │ Family Group                    1h ago  │ │
    │  │ Mom: Dinner at 7                        │ │
    │  └─────────────────────────────────────────┘ │
    │                                              │
    │                           [ + New Message ]  │
    └─────────────────────────────────────────────┘
```

### Outcome

- The user is authenticated with a fresh session token.
- Missed messages are synced and displayed.
- The WebSocket connection is active for real-time messaging.

---

## 4. Adding a Server (Multi-Server)

**Actor**: User who already has at least one server configured and wants to add another.

**Precondition**: The user has the Sovereign app with at least one server already connected.

**Goal**: Connect to an additional Sovereign server.

### Flow

```
Step 1: Open server list
    From any screen, user navigates to the server list
    (sidebar on tablet, bottom tab or hamburger menu on phone).

    ┌─────────────────────────────────────────────┐
    │  Servers                                     │
    │  ─────────────────────────────────────────── │
    │                                              │
    │  ● My Home Server                    3 ●     │
    │    sovereign.example.com                     │
    │                                              │
    │  ─────────────────────────────────────────── │
    │                                              │
    │  [ + Add Server ]                            │
    └─────────────────────────────────────────────┘

    The green dot indicates the server is connected.
    The badge "3" indicates 3 unread messages.
    User taps "+ Add Server".

Step 2: Enter new server URL
    ┌─────────────────────────────────────────────┐
    │  Add Server                                  │
    │                                              │
    │  Server URL:                                 │
    │  ┌─────────────────────────────────────────┐ │
    │  │ work.sovereign.io                        │ │
    │  └─────────────────────────────────────────┘ │
    │                                              │
    │  [ Connect ]                                 │
    └─────────────────────────────────────────────┘

Step 3: Validate connection
    The app connects to the new server and fetches server info.
    If the connection fails, an error is shown:

    "Could not connect to work.sovereign.io.
     Please check the URL and try again."

    If the connection succeeds:

    ┌─────────────────────────────────────────────┐
    │  Work Sovereign                              │
    │  work.sovereign.io                           │
    │                                              │
    │  [ Register ]     [ Login ]                  │
    └─────────────────────────────────────────────┘

Step 4: Authenticate
    User chooses "Register" (if new to this server) or "Login"
    (if they already have an account). The flow follows the
    Registration or Login flows described above.

Step 5: Server added
    After successful authentication, the new server appears in
    the server list:

    ┌─────────────────────────────────────────────┐
    │  Servers                                     │
    │  ─────────────────────────────────────────── │
    │                                              │
    │  ● My Home Server                    3 ●     │
    │    sovereign.example.com                     │
    │                                              │
    │  ● Work Sovereign                            │
    │    work.sovereign.io                         │
    │                                              │
    │  ─────────────────────────────────────────── │
    │                                              │
    │  [ + Add Server ]                            │
    └─────────────────────────────────────────────┘

    The user can now switch between servers or view
    a unified conversation list across all servers.
```

### Outcome

- The new server is added to the client's server list.
- The user is authenticated on the new server.
- Conversations from the new server appear in the unified view.
- The app maintains simultaneous WebSocket connections to all configured servers.

---

## 5. 1:1 Messaging

**Actor**: Two registered users on the same server.

**Precondition**: Both users have accounts, registered passkeys, and uploaded MLS key packages.

**Goal**: Exchange end-to-end encrypted messages.

### Flow

```
Step 1: Start new conversation
    User taps "+ New Message" from the conversation list.

    ┌─────────────────────────────────────────────┐
    │  New Message                                 │
    │  ─────────────────────────────────────────── │
    │  Search users on this server                 │
    │  ┌─────────────────────────────────────────┐ │
    │  │ 🔍                                       │ │
    │  └─────────────────────────────────────────┘ │
    │                                              │
    │  Alice                                       │
    │  Charlie                                     │
    │  Diana                                       │
    └─────────────────────────────────────────────┘

Step 2: Select recipient
    User taps on "Alice".

    If a 1:1 conversation with Alice already exists,
    the app navigates directly to it.

    If no conversation exists, the app creates one:

Step 3: Create MLS group (behind the scenes)
    The app:
    a) Creates a new Conversation on the server (type: '1:1').
    b) Fetches Alice's MLS key package from the server.
    c) Creates a new MLS group with two members (self + Alice).
    d) Sends an MLS Welcome message to Alice via the server.
    e) Stores the MLS group state locally.

Step 4: Conversation opens
    ┌─────────────────────────────────────────────┐
    │  ← Alice                                     │
    │  ─────────────────────────────────────────── │
    │                                              │
    │  End-to-end encrypted conversation.          │
    │  Messages are visible only to you and Alice. │
    │                                              │
    │                                              │
    │                                              │
    │                                              │
    │  ─────────────────────────────────────────── │
    │  ┌─────────────────────────────────────────┐ │
    │  │ Type a message...                 [Send]│ │
    │  └─────────────────────────────────────────┘ │
    └─────────────────────────────────────────────┘

Step 5: Send a message
    User types "Hey Alice, are you free tonight?" and taps Send.

    The app:
    a) Serializes the message as a protobuf MessageContent.
    b) Encrypts with MLS → produces ciphertext.
    c) Wraps in a protobuf Envelope.
    d) Sends over WebSocket.

    The message appears in the chat with a "sending" indicator:

    │                                              │
    │                  Hey Alice, are you free      │
    │                  tonight?            ○ 3:42p │
    │                                              │

    The ○ indicator means "sent to server" (filled = delivered).

Step 6: Server routes the message
    The server:
    a) Receives the Envelope.
    b) Validates the sender's session.
    c) Assigns server_timestamp and message ID.
    d) Stores the encrypted message in the database.
    e) Looks up Alice's WebSocket connection.
    f) Forwards the Envelope to Alice.

Step 7: Recipient receives
    Alice's app:
    a) Receives the Envelope over WebSocket.
    b) Extracts the MLS ciphertext.
    c) Decrypts with her MLS group session.
    d) Displays the message.

    On Alice's screen:

    │                                              │
    │  Hey Alice, are you free            3:42p    │
    │  tonight?                                    │
    │                                              │

    Alice's app sends a delivery acknowledgment to the server.

    On the sender's screen, the indicator updates:

    │                  Hey Alice, are you free      │
    │                  tonight?            ● 3:42p │

    The ● indicator means "delivered to recipient".

Step 8: Alice replies
    The same flow occurs in reverse. Alice types a reply,
    her app encrypts and sends, the server routes, and
    the original sender receives and decrypts.

    │                                              │
    │                  Hey Alice, are you free      │
    │                  tonight?            ● 3:42p │
    │                                              │
    │  Sure! What time?                   3:43p    │
    │                                              │
```

### Outcome

- A 1:1 conversation exists between the two users.
- Messages are end-to-end encrypted via MLS.
- Both users see the conversation in real time.
- The server stored only encrypted blobs and metadata.

---

## 6. Group Chat

**Actor**: Multiple registered users on the same server.

**Precondition**: All users have accounts with uploaded MLS key packages.

**Goal**: Create a group conversation and exchange encrypted messages.

### Flow

#### Creating a Group

```
Step 1: Start new group
    User taps "+ New Message" → "New Group".

    ┌─────────────────────────────────────────────┐
    │  New Group                                   │
    │  ─────────────────────────────────────────── │
    │                                              │
    │  Group Name:                                 │
    │  ┌─────────────────────────────────────────┐ │
    │  │ Weekend Hiking                           │ │
    │  └─────────────────────────────────────────┘ │
    │                                              │
    │  Add Members:                                │
    │  ┌─────────────────────────────────────────┐ │
    │  │ 🔍                                       │ │
    │  └─────────────────────────────────────────┘ │
    │                                              │
    │  ☑ Alice                                     │
    │  ☑ Charlie                                   │
    │  ☐ Diana                                     │
    │  ☐ Eve                                       │
    │                                              │
    │  [ Create Group ]                            │
    └─────────────────────────────────────────────┘

Step 2: Create MLS group (behind the scenes)
    The app:
    a) Creates a Conversation on the server (type: 'group',
       title: 'Weekend Hiking').
    b) Adds self as ConversationMember with role 'admin'.
    c) Fetches MLS key packages for Alice and Charlie.
    d) Creates a new MLS group.
    e) Generates MLS Add proposals for Alice and Charlie.
    f) Commits the proposals.
    g) Sends MLS Welcome messages to Alice and Charlie via
       the server.
    h) Server stores ConversationMember records for all members.

Step 3: Group is ready
    ┌─────────────────────────────────────────────┐
    │  ← Weekend Hiking (3 members)                │
    │  ─────────────────────────────────────────── │
    │                                              │
    │  End-to-end encrypted group.                 │
    │  You created this group with Alice and       │
    │  Charlie.                                    │
    │                                              │
    │  ─────────────────────────────────────────── │
    │  ┌─────────────────────────────────────────┐ │
    │  │ Type a message...                 [Send]│ │
    │  └─────────────────────────────────────────┘ │
    └─────────────────────────────────────────────┘
```

#### Sending Messages in a Group

```
Step 4: Send a message
    User types "Trail suggestions for Saturday?" and taps Send.

    The app encrypts the message with the MLS group session.
    The ciphertext is sent to the server.

    The server:
    a) Receives the Envelope.
    b) Looks up all ConversationMembers for this conversation.
    c) Forwards the encrypted Envelope to all members with
       active WebSocket connections.
    d) Stores the encrypted message.

    Each recipient's app decrypts and displays the message.

    ┌─────────────────────────────────────────────┐
    │  ← Weekend Hiking (3 members)                │
    │  ─────────────────────────────────────────── │
    │                                              │
    │                  Trail suggestions for       │
    │                  Saturday?          ● 10:15a │
    │                                              │
    │  Alice                              10:16a   │
    │  How about Eagle Creek?                      │
    │                                              │
    │  Charlie                            10:17a   │
    │  +1 for Eagle Creek!                         │
    │                                              │
    │  ─────────────────────────────────────────── │
    │  ┌─────────────────────────────────────────┐ │
    │  │ Type a message...                 [Send]│ │
    │  └─────────────────────────────────────────┘ │
    └─────────────────────────────────────────────┘
```

#### Adding a Member

```
Step 5: Add a new member
    Group admin taps the group name → "Add Member" → selects Diana.

    The app:
    a) Fetches Diana's MLS key package from the server.
    b) Creates an MLS Add proposal for Diana.
    c) Commits the proposal → new epoch begins.
    d) Sends an MLS Welcome message to Diana.
    e) Server adds Diana as ConversationMember.

    All existing members receive the MLS Commit and update
    their group state to the new epoch.

    Diana's app:
    a) Receives the MLS Welcome message.
    b) Initializes her MLS group state.
    c) The conversation appears in her conversation list.

    A system message appears in the group:

    │  ─── Diana was added to the group ───       │

    Diana can see messages sent after she joined but
    NOT messages sent before (forward secrecy).
```

#### Removing a Member

```
Step 6: Remove a member
    Group admin taps the group name → taps Charlie →
    "Remove from Group".

    ┌─────────────────────────────────────────────┐
    │  Remove Charlie from Weekend Hiking?         │
    │                                              │
    │  Charlie will no longer be able to send      │
    │  or receive messages in this group.          │
    │                                              │
    │  [ Cancel ]     [ Remove ]                   │
    └─────────────────────────────────────────────┘

    Admin confirms.

    The app:
    a) Creates an MLS Remove proposal for Charlie.
    b) Commits the proposal → new epoch with key rotation.
    c) All remaining members update their group state.
    d) Server removes Charlie from ConversationMember.

    Key rotation ensures Charlie cannot decrypt any messages
    sent after his removal (post-compromise security).

    A system message appears:

    │  ─── Charlie was removed from the group ─── │

    Charlie's app:
    a) Receives notification of removal.
    b) The conversation is marked as "left" or removed
       from the active list.
    c) Charlie retains access to messages received before
       removal (local history).
```

### Outcome

- A group conversation exists with MLS E2E encryption.
- All members can send and receive encrypted messages.
- Adding members sends Welcome messages and advances the epoch.
- Removing members triggers key rotation, ensuring the removed member cannot read future messages.
- The server only handles encrypted blobs and membership metadata.

---

## 7. Admin Panel

**Actor**: Server administrator (user with admin role).

**Precondition**: The Sovereign server is running. The admin has a registered account with admin privileges.

**Goal**: Monitor and manage the server through a browser-based UI.

### Flow

#### Accessing the Admin Panel

```
Step 1: Navigate to admin URL
    Admin opens a web browser and navigates to:
    http://localhost:8080/admin
    (or https://sovereign.example.com/admin if accessed remotely)

Step 2: Login
    ┌─────────────────────────────────────────────┐
    │  Sovereign Admin                             │
    │  ─────────────────────────────────────────── │
    │                                              │
    │  Sign in with your passkey to continue.      │
    │                                              │
    │  [ Sign In ]                                 │
    └─────────────────────────────────────────────┘

    Admin clicks "Sign In".
    Browser prompts for passkey authentication
    (biometric, PIN, or security key).

    Server verifies the assertion AND checks that the
    user has admin role. Non-admin users are rejected.
```

#### Dashboard

```
Step 3: Dashboard view
    ┌─────────────────────────────────────────────────────────┐
    │  Sovereign Admin       My Home Server                    │
    │  ─────────────────────────────────────────────────────── │
    │                                                          │
    │  ┌──────────┐  ┌──────────┐  ┌──────────┐               │
    │  │ Users    │  │ Active   │  │ Messages │               │
    │  │    12    │  │ Conns: 8 │  │ Today: 347│              │
    │  └──────────┘  └──────────┘  └──────────┘               │
    │                                                          │
    │  Server Status                                           │
    │  ─────────────────────────────                           │
    │  Uptime:         3d 14h 22m                              │
    │  Memory:         48 MB                                   │
    │  Database Size:  12 MB                                   │
    │  Go Version:     1.23.0                                  │
    │  Server Version: 0.1.0                                   │
    │                                                          │
    │  Recent Activity                                         │
    │  ─────────────────────────────                           │
    │  bob connected                         2 minutes ago     │
    │  alice disconnected                    15 minutes ago    │
    │  charlie registered                    1 hour ago        │
    │                                                          │
    │  ──────────────────────────────────────────────────────  │
    │  [ Users ]  [ Settings ]  [ Dashboard ]                  │
    └─────────────────────────────────────────────────────────┘
```

#### User Management

```
Step 4: User management
    Admin clicks "Users" in the navigation.

    ┌─────────────────────────────────────────────────────────┐
    │  Users (12)                                              │
    │  ─────────────────────────────────────────────────────── │
    │                                                          │
    │  Username    Display Name    Registered     Status       │
    │  ──────────  ──────────────  ─────────────  ──────────  │
    │  alice       Alice           2025-01-15     Active  ⚙   │
    │  bob         Bob Smith       2025-01-16     Active  ⚙   │
    │  charlie     Charlie         2025-01-20     Active  ⚙   │
    │  diana       Diana Prince    2025-01-22     Disabled ⚙  │
    │  ...                                                     │
    └─────────────────────────────────────────────────────────┘

    Admin clicks ⚙ next to a user to manage them:

    ┌─────────────────────────────────────────────┐
    │  User: diana                                 │
    │  ─────────────────────────────────────────── │
    │                                              │
    │  Display Name: Diana Prince                  │
    │  Registered:   2025-01-22                    │
    │  Last Seen:    2025-01-25 14:30              │
    │  Status:       Disabled                      │
    │                                              │
    │  Sessions: 0 active                          │
    │  Credentials: 1 passkey                      │
    │  Conversations: 3                            │
    │                                              │
    │  Actions:                                    │
    │  [ Enable Account ]                          │
    │  [ Revoke All Sessions ]                     │
    │  [ Delete Account ]                          │
    └─────────────────────────────────────────────┘

    Available actions:
    - Enable/Disable account: Prevents the user from logging
      in and disconnects active sessions.
    - Revoke All Sessions: Forces the user to re-authenticate
      on all devices.
    - Delete Account: Permanently removes the user and their
      credentials. Messages are retained (sender_id set to NULL)
      but the user can no longer be identified.
```

#### Server Settings

```
Step 5: Server settings
    Admin clicks "Settings" in the navigation.

    ┌─────────────────────────────────────────────────────────┐
    │  Server Settings                                         │
    │  ─────────────────────────────────────────────────────── │
    │                                                          │
    │  General                                                 │
    │  ─────────────────────────────                           │
    │  Server Name:    [ My Home Server          ]             │
    │                                                          │
    │  Limits                                                  │
    │  ─────────────────────────────                           │
    │  Max Connections:       [ 1000 ]                         │
    │  Max Connections/IP:    [ 10   ]                         │
    │  Max Message Size (KB): [ 64   ]                         │
    │  Rate Limit (msg/min):  [ 60   ]                         │
    │  Session Expiry (days): [ 30   ]                         │
    │                                                          │
    │  Registration                                            │
    │  ─────────────────────────────                           │
    │  Open Registration:     [ ● On  ○ Off ]                  │
    │  (When off, only admins can create new accounts)         │
    │                                                          │
    │  [ Save Changes ]                                        │
    └─────────────────────────────────────────────────────────┘

    Changes to settings are saved to the server_config table
    and take effect immediately (no server restart required).
```

### Outcome

- The admin can monitor server health and activity.
- The admin can manage users (enable, disable, delete, revoke sessions).
- The admin can configure server settings in real time.
- All admin actions are authenticated via WebAuthn and restricted to users with the admin role.
- The admin panel is a static SPA embedded in the server binary, requiring no separate deployment.
