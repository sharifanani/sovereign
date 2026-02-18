import {
  encodeEnvelope,
  decodeEnvelope,
  encodePing,
  encodePong,
  MessageType,
  generateRequestId,
  type Envelope,
} from '../src/services/protocol';
import { WebSocketClient, type ConnectionState, type WebSocketClientConfig } from '../src/services/websocket';

// --- Mock WebSocket ---

type WSListener = ((event: Record<string, unknown>) => void) | null;

class MockWebSocket {
  static readonly CONNECTING = 0;
  static readonly OPEN = 1;
  static readonly CLOSING = 2;
  static readonly CLOSED = 3;

  url: string;
  protocol: string;
  binaryType = 'blob';
  readyState = MockWebSocket.CONNECTING;

  onopen: WSListener = null;
  onclose: WSListener = null;
  onmessage: WSListener = null;
  onerror: WSListener = null;

  sent: ArrayBuffer[] = [];
  private closed = false;

  constructor(url: string, protocol?: string) {
    this.url = url;
    this.protocol = protocol ?? '';
    MockWebSocket.instances.push(this);
  }

  send(data: ArrayBuffer): void {
    this.sent.push(data);
  }

  close(): void {
    if (this.closed) return;
    this.closed = true;
    this.readyState = MockWebSocket.CLOSED;
    if (this.onclose) {
      this.onclose({});
    }
  }

  // --- Test helpers ---

  simulateOpen(): void {
    this.readyState = MockWebSocket.OPEN;
    if (this.onopen) {
      this.onopen({});
    }
  }

  simulateMessage(data: ArrayBuffer): void {
    if (this.onmessage) {
      this.onmessage({ data });
    }
  }

  simulateClose(): void {
    this.readyState = MockWebSocket.CLOSED;
    if (this.onclose) {
      this.onclose({});
    }
  }

  simulateError(): void {
    if (this.onerror) {
      this.onerror({});
    }
  }

  static instances: MockWebSocket[] = [];
  static clear(): void {
    MockWebSocket.instances = [];
  }
  static latest(): MockWebSocket {
    return MockWebSocket.instances[MockWebSocket.instances.length - 1];
  }
}

// Install mock
(globalThis as Record<string, unknown>).WebSocket = MockWebSocket;

function makeEnvelopeBuffer(envelope: Envelope): ArrayBuffer {
  const bytes = encodeEnvelope(envelope);
  return bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength) as ArrayBuffer;
}

function createClient(overrides?: Partial<WebSocketClientConfig>): {
  client: WebSocketClient;
  states: ConnectionState[];
  messages: Envelope[];
} {
  const states: ConnectionState[] = [];
  const messages: Envelope[] = [];
  const config: WebSocketClientConfig = {
    url: 'wss://example.com/ws',
    onMessage: (env) => messages.push(env),
    onStateChange: (s) => states.push(s),
    ...overrides,
  };
  return { client: new WebSocketClient(config), states, messages };
}

describe('WebSocketClient', () => {
  beforeEach(() => {
    jest.useFakeTimers();
    MockWebSocket.clear();
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  describe('connection state transitions', () => {
    it('transitions from disconnected -> connecting -> connected', () => {
      const { client, states } = createClient();
      expect(client.getState()).toBe('disconnected');

      client.connect();
      expect(client.getState()).toBe('connecting');
      expect(states).toEqual(['connecting']);

      MockWebSocket.latest().simulateOpen();
      expect(client.getState()).toBe('connected');
      expect(states).toEqual(['connecting', 'connected']);

      client.disconnect();
    });
  });

  describe('disconnect', () => {
    it('sets state to disconnected and closes WebSocket', () => {
      const { client, states } = createClient();
      client.connect();
      const ws = MockWebSocket.latest();
      ws.simulateOpen();

      states.length = 0;
      client.disconnect();

      expect(client.getState()).toBe('disconnected');
      expect(states).toEqual(['disconnected']);
      expect(ws.readyState).toBe(MockWebSocket.CLOSED);
    });
  });

  describe('send', () => {
    it('sends binary data when connected', () => {
      const { client } = createClient();
      client.connect();
      const ws = MockWebSocket.latest();
      ws.simulateOpen();

      const envelope: Envelope = {
        type: MessageType.MESSAGE_SEND,
        requestId: 'req-1',
        payload: new Uint8Array([10, 20]),
      };
      client.send(envelope);

      expect(ws.sent.length).toBe(1);
      const decoded = decodeEnvelope(new Uint8Array(ws.sent[0]));
      expect(decoded.type).toBe(MessageType.MESSAGE_SEND);
      expect(decoded.requestId).toBe('req-1');

      client.disconnect();
    });

    it('silently drops messages when disconnected', () => {
      const { client } = createClient();
      // Never connect

      const envelope: Envelope = {
        type: MessageType.MESSAGE_SEND,
        requestId: 'req-drop',
        payload: new Uint8Array([]),
      };
      client.send(envelope);

      // No WebSocket instance exists, so nothing should throw
      expect(MockWebSocket.instances.length).toBe(0);
    });

    it('silently drops messages when connecting but not yet open', () => {
      const { client } = createClient();
      client.connect();
      // ws exists but onopen hasn't fired yet, state is 'connecting'

      const envelope: Envelope = {
        type: MessageType.MESSAGE_SEND,
        requestId: 'req-drop-2',
        payload: new Uint8Array([]),
      };
      client.send(envelope);

      const ws = MockWebSocket.latest();
      expect(ws.sent.length).toBe(0);

      client.disconnect();
    });
  });

  describe('ping scheduling', () => {
    it('sends a ping on interval after connection', () => {
      const { client } = createClient();
      client.connect();
      const ws = MockWebSocket.latest();
      ws.simulateOpen();

      // No pings sent yet
      expect(ws.sent.length).toBe(0);

      // Advance by 30 seconds (PING_INTERVAL_MS)
      jest.advanceTimersByTime(30_000);

      // A ping should have been sent
      expect(ws.sent.length).toBe(1);
      const decoded = decodeEnvelope(new Uint8Array(ws.sent[0]));
      expect(decoded.type).toBe(MessageType.PING);

      client.disconnect();
    });

    it('sends multiple pings on subsequent intervals', () => {
      const { client } = createClient();
      client.connect();
      const ws = MockWebSocket.latest();
      ws.simulateOpen();

      // Send pong responses to prevent timeout closure
      const sendPongResponse = (): void => {
        if (ws.sent.length > 0) {
          const lastSent = decodeEnvelope(new Uint8Array(ws.sent[ws.sent.length - 1]));
          if (lastSent.type === MessageType.PING) {
            const pongPayload = encodePong({ timestamp: Date.now() });
            const pongEnvelope = makeEnvelopeBuffer({
              type: MessageType.PONG,
              requestId: lastSent.requestId,
              payload: pongPayload,
            });
            ws.simulateMessage(pongEnvelope);
          }
        }
      };

      jest.advanceTimersByTime(30_000);
      expect(ws.sent.length).toBe(1);
      sendPongResponse();

      jest.advanceTimersByTime(30_000);
      expect(ws.sent.length).toBe(2);
      sendPongResponse();

      jest.advanceTimersByTime(30_000);
      expect(ws.sent.length).toBe(3);

      client.disconnect();
    });
  });

  describe('pong timeout', () => {
    it('closes connection if no pong received within timeout', () => {
      const { client, states } = createClient();
      client.connect();
      const ws = MockWebSocket.latest();
      ws.simulateOpen();
      states.length = 0;

      // Trigger ping
      jest.advanceTimersByTime(30_000);
      expect(ws.sent.length).toBe(1);

      // Advance past pong timeout (10s) without sending pong
      jest.advanceTimersByTime(10_000);

      // WebSocket should have been closed
      expect(ws.readyState).toBe(MockWebSocket.CLOSED);
      expect(states).toContain('disconnected');
    });

    it('does not close connection if pong is received in time', () => {
      const { client } = createClient();
      client.connect();
      const ws = MockWebSocket.latest();
      ws.simulateOpen();

      // Trigger ping
      jest.advanceTimersByTime(30_000);

      // Send pong before timeout
      const pongPayload = encodePong({ timestamp: Date.now() });
      ws.simulateMessage(makeEnvelopeBuffer({
        type: MessageType.PONG,
        requestId: 'pong-1',
        payload: pongPayload,
      }));

      // Advance past what would have been the timeout
      jest.advanceTimersByTime(10_000);

      // Connection should still be open (ws was not closed by the client)
      // The mock's close was not called by the client, so readyState is still OPEN
      expect(ws.readyState).toBe(MockWebSocket.OPEN);

      client.disconnect();
    });
  });

  describe('reconnection backoff', () => {
    it('schedules reconnect with exponential backoff delays', () => {
      // Mock Math.random to return 0.5 so jitter multiplier = 0.5 + 0.5*0.5 = 0.75
      const randomSpy = jest.spyOn(Math, 'random').mockReturnValue(0.5);

      const { client, states } = createClient();
      const expectedBaseDelays = [1000, 2000, 4000, 8000, 16000, 30000];

      client.connect();
      let ws = MockWebSocket.latest();

      for (let attempt = 0; attempt < expectedBaseDelays.length; attempt++) {
        // Simulate connection failure
        ws.simulateClose();

        const expectedDelay = expectedBaseDelays[attempt] * 0.75; // jitter with random=0.5
        const instancesBefore = MockWebSocket.instances.length;

        // Advance just before the delay - should not reconnect yet
        jest.advanceTimersByTime(expectedDelay - 1);
        expect(MockWebSocket.instances.length).toBe(instancesBefore);

        // Advance to complete the delay - should trigger reconnect
        jest.advanceTimersByTime(1);
        expect(MockWebSocket.instances.length).toBe(instancesBefore + 1);

        ws = MockWebSocket.latest();
      }

      // After max attempts, delay should cap at 30000 * 0.75
      ws.simulateClose();
      const instancesBefore = MockWebSocket.instances.length;
      jest.advanceTimersByTime(30000 * 0.75);
      expect(MockWebSocket.instances.length).toBe(instancesBefore + 1);

      randomSpy.mockRestore();
      client.disconnect();
    });

    it('resets backoff counter after successful connection', () => {
      const randomSpy = jest.spyOn(Math, 'random').mockReturnValue(0.5);

      const { client } = createClient();
      client.connect();
      let ws = MockWebSocket.latest();

      // First disconnect triggers reconnect at attempt=0
      ws.simulateClose();
      jest.advanceTimersByTime(1000 * 0.75);
      ws = MockWebSocket.latest();

      // Second disconnect triggers attempt=1
      ws.simulateClose();
      jest.advanceTimersByTime(2000 * 0.75);
      ws = MockWebSocket.latest();

      // Now connect successfully
      ws.simulateOpen();

      // Disconnect again - should reset to attempt=0 (1000ms base)
      ws.simulateClose();
      const instancesBefore = MockWebSocket.instances.length;

      // Should reconnect at 1000 * 0.75 = 750ms, not 4000 * 0.75 = 3000ms
      jest.advanceTimersByTime(750);
      expect(MockWebSocket.instances.length).toBe(instancesBefore + 1);

      randomSpy.mockRestore();
      client.disconnect();
    });
  });

  describe('intentional disconnect', () => {
    it('does not schedule reconnection after intentional disconnect', () => {
      const { client } = createClient();
      client.connect();
      const ws = MockWebSocket.latest();
      ws.simulateOpen();

      const instancesBefore = MockWebSocket.instances.length;
      client.disconnect();

      // Advance time significantly
      jest.advanceTimersByTime(60_000);

      // No new WebSocket instances should have been created
      expect(MockWebSocket.instances.length).toBe(instancesBefore);
    });
  });

  describe('message handling', () => {
    it('delivers non-Ping/Pong envelopes to onMessage callback', () => {
      const { client, messages } = createClient();
      client.connect();
      const ws = MockWebSocket.latest();
      ws.simulateOpen();

      const envelope: Envelope = {
        type: MessageType.MESSAGE_RECEIVE,
        requestId: 'msg-1',
        payload: new Uint8Array([42]),
      };
      ws.simulateMessage(makeEnvelopeBuffer(envelope));

      expect(messages.length).toBe(1);
      expect(messages[0].type).toBe(MessageType.MESSAGE_RECEIVE);
      expect(messages[0].requestId).toBe('msg-1');

      client.disconnect();
    });

    it('does not deliver Ping messages to onMessage callback', () => {
      const { client, messages } = createClient();
      client.connect();
      const ws = MockWebSocket.latest();
      ws.simulateOpen();

      const pingPayload = encodePing({ timestamp: Date.now() });
      ws.simulateMessage(makeEnvelopeBuffer({
        type: MessageType.PING,
        requestId: 'ping-1',
        payload: pingPayload,
      }));

      expect(messages.length).toBe(0);

      client.disconnect();
    });

    it('does not deliver Pong messages to onMessage callback', () => {
      const { client, messages } = createClient();
      client.connect();
      const ws = MockWebSocket.latest();
      ws.simulateOpen();

      const pongPayload = encodePong({ timestamp: Date.now() });
      ws.simulateMessage(makeEnvelopeBuffer({
        type: MessageType.PONG,
        requestId: 'pong-1',
        payload: pongPayload,
      }));

      expect(messages.length).toBe(0);

      client.disconnect();
    });

    it('responds to server Ping with a Pong', () => {
      const { client } = createClient();
      client.connect();
      const ws = MockWebSocket.latest();
      ws.simulateOpen();

      const pingPayload = encodePing({ timestamp: 12345 });
      ws.simulateMessage(makeEnvelopeBuffer({
        type: MessageType.PING,
        requestId: 'server-ping-1',
        payload: pingPayload,
      }));

      // Client should have sent a Pong
      expect(ws.sent.length).toBe(1);
      const decoded = decodeEnvelope(new Uint8Array(ws.sent[0]));
      expect(decoded.type).toBe(MessageType.PONG);
      expect(decoded.requestId).toBe('server-ping-1');

      client.disconnect();
    });

    it('delivers multiple messages in order', () => {
      const { client, messages } = createClient();
      client.connect();
      const ws = MockWebSocket.latest();
      ws.simulateOpen();

      const types = [
        MessageType.MESSAGE_RECEIVE,
        MessageType.MESSAGE_ACK,
        MessageType.GROUP_CREATED,
      ] as const;

      for (const t of types) {
        ws.simulateMessage(makeEnvelopeBuffer({
          type: t,
          requestId: `msg-${t}`,
          payload: new Uint8Array([]),
        }));
      }

      expect(messages.length).toBe(3);
      expect(messages[0].type).toBe(MessageType.MESSAGE_RECEIVE);
      expect(messages[1].type).toBe(MessageType.MESSAGE_ACK);
      expect(messages[2].type).toBe(MessageType.GROUP_CREATED);

      client.disconnect();
    });
  });
});
