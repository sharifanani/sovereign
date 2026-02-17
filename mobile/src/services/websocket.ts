// WebSocket client service
// Manages connection to a single Sovereign server

import {
  encodeEnvelope,
  decodeEnvelope,
  encodePing,
  decodePong,
  encodePong,
  decodePing,
  MessageType,
  generateRequestId,
  type Envelope,
  type MessageTypeValue,
} from './protocol';

export type ConnectionState = 'disconnected' | 'connecting' | 'connected';

export interface WebSocketClientConfig {
  url: string;
  onMessage: (envelope: Envelope) => void;
  onStateChange: (state: ConnectionState) => void;
}

const PING_INTERVAL_MS = 30_000;
const PONG_TIMEOUT_MS = 10_000;
const MAX_BACKOFF_MS = 30_000;
const BACKOFF_BASE_DELAYS = [1000, 2000, 4000, 8000, 16000, MAX_BACKOFF_MS];

function computeBackoffDelay(attempt: number): number {
  const index = Math.min(attempt, BACKOFF_BASE_DELAYS.length - 1);
  const baseDelay = BACKOFF_BASE_DELAYS[index];
  // Jitter: actual_delay = base_delay * (0.5 + random() * 0.5)
  return baseDelay * (0.5 + Math.random() * 0.5);
}

export class WebSocketClient {
  private ws: WebSocket | null = null;
  private config: WebSocketClientConfig;
  private state: ConnectionState = 'disconnected';
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

  getUrl(): string {
    return this.config.url;
  }

  connect(): void {
    this.intentionalClose = false;
    this.doConnect();
  }

  disconnect(): void {
    this.intentionalClose = true;
    this.cleanup();
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

  private doConnect(): void {
    this.cleanup();
    this.setState('connecting');

    const ws = new WebSocket(this.config.url, 'sovereign.v1');
    ws.binaryType = 'arraybuffer';
    this.ws = ws;

    ws.onopen = () => {
      this.setState('connected');
      this.reconnectAttempt = 0;
      this.startPingInterval();
    };

    ws.onmessage = (event) => {
      this.handleMessage(event);
    };

    ws.onerror = () => {
      // onerror is always followed by onclose; handle reconnect there
    };

    ws.onclose = () => {
      this.stopPingInterval();
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

    // Deliver all other messages to the callback
    this.config.onMessage(envelope);
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
      // Pong not received — connection is dead
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
