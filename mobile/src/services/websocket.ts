// WebSocket client service
// Manages connection to a single Sovereign server

import {
  encodeEnvelope,
  decodeEnvelope,
  encodePing,
  decodePong,
  encodePong,
  decodePing,
  encodeAuthRequest,
  decodeAuthSuccess,
  decodeAuthError,
  decodeAuthChallenge,
  decodeAuthRegisterChallenge,
  decodeAuthRegisterSuccess,
  decodeMessageReceive,
  decodeMessageDelivered,
  decodeGroupCreated,
  decodeGroupMemberAdded,
  decodeGroupMemberRemoved,
  decodeMLSKeyPackageResponse,
  decodeMLSWelcomeReceive,
  decodeMLSCommitBroadcast,
  MessageType,
  generateRequestId,
  type Envelope,
  type MessageTypeValue,
  type AuthSuccessMessage,
  type AuthErrorMessage,
  type AuthChallengeMessage,
  type AuthRegisterChallengeMessage,
  type AuthRegisterSuccessMessage,
  type MessageReceiveMessage,
  type MessageDeliveredMessage,
  type GroupCreatedMessage,
  type GroupMemberAddedMessage,
  type GroupMemberRemovedMessage,
  type MLSKeyPackageResponseMessage,
  type MLSWelcomeReceiveMessage,
  type MLSCommitBroadcastMessage,
} from './protocol';

export type ConnectionState = 'disconnected' | 'connecting' | 'connected';

export type AuthState = 'unauthenticated' | 'authenticating' | 'authenticated';

export interface AuthCallbacks {
  onAuthSuccess: (msg: AuthSuccessMessage) => void;
  onAuthError: (msg: AuthErrorMessage) => void;
  onAuthChallenge: (msg: AuthChallengeMessage) => void;
  onRegisterChallenge: (msg: AuthRegisterChallengeMessage) => void;
  onRegisterSuccess: (msg: AuthRegisterSuccessMessage) => void;
  onAuthRequired: () => void;
}

export interface MessagingCallbacks {
  onMessageReceive: (envelope: Envelope, msg: MessageReceiveMessage) => void;
  onMessageDelivered: (msg: MessageDeliveredMessage) => void;
  onGroupCreated: (envelope: Envelope, msg: GroupCreatedMessage) => void;
  onGroupMemberAdded: (msg: GroupMemberAddedMessage) => void;
  onGroupMemberRemoved: (msg: GroupMemberRemovedMessage) => void;
  onKeyPackageResponse: (envelope: Envelope, msg: MLSKeyPackageResponseMessage) => void;
  onWelcomeReceive: (msg: MLSWelcomeReceiveMessage) => void;
  onCommitBroadcast: (msg: MLSCommitBroadcastMessage) => void;
}

export interface WebSocketClientConfig {
  url: string;
  onMessage: (envelope: Envelope) => void;
  onStateChange: (state: ConnectionState) => void;
  sessionToken?: string;
  authCallbacks?: AuthCallbacks;
  messagingCallbacks?: MessagingCallbacks;
}

const PING_INTERVAL_MS = 30_000;
const PONG_TIMEOUT_MS = 10_000;
const MAX_BACKOFF_MS = 30_000;
const BACKOFF_BASE_DELAYS = [1000, 2000, 4000, 8000, 16000, MAX_BACKOFF_MS];

function computeBackoffDelay(attempt: number): number {
  const index = Math.min(attempt, BACKOFF_BASE_DELAYS.length - 1);
  const baseDelay = BACKOFF_BASE_DELAYS[index];
  return baseDelay * (0.5 + Math.random() * 0.5);
}

export class WebSocketClient {
  private ws: WebSocket | null = null;
  private config: WebSocketClientConfig;
  private state: ConnectionState = 'disconnected';
  private authState: AuthState = 'unauthenticated';
  private reconnectAttempt = 0;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private pingTimer: ReturnType<typeof setInterval> | null = null;
  private pongTimer: ReturnType<typeof setTimeout> | null = null;
  private intentionalClose = false;

  constructor(config: WebSocketClientConfig) {
    this.config = config;
  }

  getState(): ConnectionState {
    return this.state;
  }

  getAuthState(): AuthState {
    return this.authState;
  }

  getUrl(): string {
    return this.config.url;
  }

  setSessionToken(token: string): void {
    this.config.sessionToken = token;
  }

  connect(): void {
    this.intentionalClose = false;
    this.doConnect();
  }

  disconnect(): void {
    this.intentionalClose = true;
    this.cleanup();
    this.authState = 'unauthenticated';
    this.setState('disconnected');
  }

  send(envelope: Envelope): void {
    if (this.state !== 'connected' || !this.ws) {
      return;
    }
    const binary = encodeEnvelope(envelope);
    const buffer = binary.buffer.slice(binary.byteOffset, binary.byteOffset + binary.byteLength) as ArrayBuffer;
    this.ws.send(buffer);
  }

  sendPing(): void {
    const payload = encodePing({ timestamp: Date.now() * 1000 });
    this.send({
      type: MessageType.PING,
      requestId: generateRequestId(),
      payload,
    });
  }

  sendAuthWithSessionToken(token: string): void {
    this.authState = 'authenticating';
    const payload = encodeAuthRequest({ username: '' });
    // For session token reconnection, we send an AUTH_REQUEST with the token
    // embedded. The server recognizes empty username + session token means reconnect.
    // The actual session token is sent as part of the request envelope pattern.
    // For now, send username as empty and rely on the server to check the token.
    this.send({
      type: MessageType.AUTH_REQUEST,
      requestId: generateRequestId(),
      payload,
    });
  }

  private doConnect(): void {
    this.cleanup();
    this.setState('connecting');
    this.authState = 'unauthenticated';

    const ws = new WebSocket(this.config.url, 'sovereign.v1');
    ws.binaryType = 'arraybuffer';
    this.ws = ws;

    ws.onopen = () => {
      this.setState('connected');
      this.reconnectAttempt = 0;
      this.startPingInterval();

      // If we have a session token, attempt automatic re-authentication
      if (this.config.sessionToken) {
        this.sendAuthWithSessionToken(this.config.sessionToken);
      } else if (this.config.authCallbacks) {
        this.config.authCallbacks.onAuthRequired();
      }
    };

    ws.onmessage = (event) => {
      this.handleMessage(event);
    };

    ws.onerror = () => {
      // onerror is always followed by onclose; handle reconnect there
    };

    ws.onclose = () => {
      this.stopPingInterval();
      this.authState = 'unauthenticated';
      if (!this.intentionalClose) {
        this.setState('disconnected');
        this.scheduleReconnect();
      }
    };
  }

  private handleMessage(event: WebSocketMessageEvent): void {
    const data: unknown = event.data;
    let bytes: Uint8Array;
    if (data instanceof ArrayBuffer) {
      bytes = new Uint8Array(data);
    } else {
      return;
    }

    const envelope = decodeEnvelope(bytes);

    // Handle Pong responses
    if (envelope.type === MessageType.PONG) {
      this.clearPongTimeout();
      if (envelope.payload.length > 0) {
        decodePong(envelope.payload);
      }
      return;
    }

    // Handle server-initiated Ping
    if (envelope.type === MessageType.PING) {
      let timestamp = Date.now() * 1000;
      if (envelope.payload.length > 0) {
        const ping = decodePing(envelope.payload);
        timestamp = ping.timestamp as number;
      }
      const pongPayload = encodePong({ timestamp });
      this.send({
        type: MessageType.PONG,
        requestId: envelope.requestId,
        payload: pongPayload,
      });
      return;
    }

    // Handle auth messages
    if (this.handleAuthMessage(envelope)) {
      return;
    }

    // Handle messaging/group/MLS messages
    if (this.handleMessagingMessage(envelope)) {
      return;
    }

    // Deliver all other messages to the callback
    this.config.onMessage(envelope);
  }

  private handleAuthMessage(envelope: Envelope): boolean {
    const callbacks = this.config.authCallbacks;
    if (!callbacks) {
      return false;
    }

    switch (envelope.type) {
      case MessageType.AUTH_SUCCESS: {
        const msg = decodeAuthSuccess(envelope.payload);
        this.authState = 'authenticated';
        callbacks.onAuthSuccess(msg);
        return true;
      }
      case MessageType.AUTH_ERROR: {
        const msg = decodeAuthError(envelope.payload);
        this.authState = 'unauthenticated';
        callbacks.onAuthError(msg);
        return true;
      }
      case MessageType.AUTH_CHALLENGE: {
        const msg = decodeAuthChallenge(envelope.payload);
        this.authState = 'authenticating';
        callbacks.onAuthChallenge(msg);
        return true;
      }
      case MessageType.AUTH_REGISTER_CHALLENGE: {
        const msg = decodeAuthRegisterChallenge(envelope.payload);
        callbacks.onRegisterChallenge(msg);
        return true;
      }
      case MessageType.AUTH_REGISTER_SUCCESS: {
        const msg = decodeAuthRegisterSuccess(envelope.payload);
        this.authState = 'authenticated';
        callbacks.onRegisterSuccess(msg);
        return true;
      }
      default:
        return false;
    }
  }

  private handleMessagingMessage(envelope: Envelope): boolean {
    const callbacks = this.config.messagingCallbacks;
    if (!callbacks) {
      return false;
    }

    switch (envelope.type) {
      case MessageType.MESSAGE_RECEIVE: {
        const msg = decodeMessageReceive(envelope.payload);
        callbacks.onMessageReceive(envelope, msg);
        return true;
      }
      case MessageType.MESSAGE_DELIVERED: {
        const msg = decodeMessageDelivered(envelope.payload);
        callbacks.onMessageDelivered(msg);
        return true;
      }
      case MessageType.GROUP_CREATED: {
        const msg = decodeGroupCreated(envelope.payload);
        callbacks.onGroupCreated(envelope, msg);
        return true;
      }
      case MessageType.GROUP_MEMBER_ADDED: {
        const msg = decodeGroupMemberAdded(envelope.payload);
        callbacks.onGroupMemberAdded(msg);
        return true;
      }
      case MessageType.GROUP_MEMBER_REMOVED: {
        const msg = decodeGroupMemberRemoved(envelope.payload);
        callbacks.onGroupMemberRemoved(msg);
        return true;
      }
      case MessageType.MLS_KEY_PACKAGE_RESPONSE: {
        const msg = decodeMLSKeyPackageResponse(envelope.payload);
        callbacks.onKeyPackageResponse(envelope, msg);
        return true;
      }
      case MessageType.MLS_WELCOME_RECEIVE: {
        const msg = decodeMLSWelcomeReceive(envelope.payload);
        callbacks.onWelcomeReceive(msg);
        return true;
      }
      case MessageType.MLS_COMMIT_BROADCAST: {
        const msg = decodeMLSCommitBroadcast(envelope.payload);
        callbacks.onCommitBroadcast(msg);
        return true;
      }
      default:
        return false;
    }
  }

  private setState(newState: ConnectionState): void {
    if (this.state !== newState) {
      this.state = newState;
      this.config.onStateChange(newState);
    }
  }

  private startPingInterval(): void {
    this.stopPingInterval();
    this.pingTimer = setInterval(() => {
      this.sendPing();
      this.startPongTimeout();
    }, PING_INTERVAL_MS);
  }

  private stopPingInterval(): void {
    if (this.pingTimer !== null) {
      clearInterval(this.pingTimer);
      this.pingTimer = null;
    }
    this.clearPongTimeout();
  }

  private startPongTimeout(): void {
    this.clearPongTimeout();
    this.pongTimer = setTimeout(() => {
      this.ws?.close();
    }, PONG_TIMEOUT_MS);
  }

  private clearPongTimeout(): void {
    if (this.pongTimer !== null) {
      clearTimeout(this.pongTimer);
      this.pongTimer = null;
    }
  }

  private scheduleReconnect(): void {
    if (this.intentionalClose) {
      return;
    }
    const delay = computeBackoffDelay(this.reconnectAttempt);
    this.reconnectAttempt++;
    this.reconnectTimer = setTimeout(() => {
      this.doConnect();
    }, delay);
  }

  private cleanup(): void {
    if (this.reconnectTimer !== null) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    this.stopPingInterval();
    if (this.ws) {
      this.ws.onopen = null;
      this.ws.onmessage = null;
      this.ws.onerror = null;
      this.ws.onclose = null;
      if (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING) {
        this.ws.close();
      }
      this.ws = null;
    }
  }
}
