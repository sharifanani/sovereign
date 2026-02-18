import {
  encodeEnvelope,
  decodeEnvelope,
  encodeAuthRequest,
  encodeAuthRegisterRequest,
  decodeAuthRegisterRequest,
  decodeAuthRequest,
  MessageType,
  type Envelope,
} from '../src/services/protocol';
import {
  sendRegisterRequest,
  sendLoginRequest,
  handleRegisterChallenge,
  handleAuthChallenge,
  type PasskeyRegistrationResult,
  type PasskeyAuthenticationResult,
} from '../src/services/auth';

// Mock WebSocket client
class MockWebSocketClient {
  sent: Envelope[] = [];
  send(envelope: Envelope): void {
    this.sent.push(envelope);
  }
  getState() { return 'connected' as const; }
  getAuthState() { return 'unauthenticated' as const; }
  getUrl() { return 'wss://example.com/ws'; }
  connect() {}
  disconnect() {}
  setSessionToken(_token: string) {}
  sendPing() {}
  sendAuthWithSessionToken(_token: string) {}
}

describe('auth service', () => {
  describe('sendRegisterRequest', () => {
    it('sends AUTH_REGISTER_REQUEST envelope with username and displayName', () => {
      const client = new MockWebSocketClient();
      sendRegisterRequest(client as any, 'alice', 'Alice Wonderland');

      expect(client.sent.length).toBe(1);
      const env = client.sent[0];
      expect(env.type).toBe(MessageType.AUTH_REGISTER_REQUEST);
      expect(env.requestId).toBeTruthy();
      expect(env.payload).toBeInstanceOf(Uint8Array);

      // Decode the payload to verify contents
      const decoded = decodeAuthRegisterRequest(env.payload);
      expect(decoded.username).toBe('alice');
      expect(decoded.displayName).toBe('Alice Wonderland');
    });

    it('generates unique request IDs for each call', () => {
      const client = new MockWebSocketClient();
      sendRegisterRequest(client as any, 'alice', 'Alice');
      sendRegisterRequest(client as any, 'bob', 'Bob');

      expect(client.sent.length).toBe(2);
      expect(client.sent[0].requestId).not.toBe(client.sent[1].requestId);
    });
  });

  describe('sendLoginRequest', () => {
    it('sends AUTH_REQUEST envelope with username', () => {
      const client = new MockWebSocketClient();
      sendLoginRequest(client as any, 'alice');

      expect(client.sent.length).toBe(1);
      const env = client.sent[0];
      expect(env.type).toBe(MessageType.AUTH_REQUEST);
      expect(env.requestId).toBeTruthy();

      const decoded = decodeAuthRequest(env.payload);
      expect(decoded.username).toBe('alice');
    });
  });

  describe('handleRegisterChallenge', () => {
    it('throws because passkey native module is a placeholder', async () => {
      const client = new MockWebSocketClient();
      const challenge = {
        challenge: new Uint8Array([1, 2, 3]),
        credentialCreationOptions: new Uint8Array([4, 5, 6]),
      };

      await expect(
        handleRegisterChallenge(client as any, challenge),
      ).rejects.toThrow('Passkey registration requires a native module');
    });
  });

  describe('handleAuthChallenge', () => {
    it('throws because passkey native module is a placeholder', async () => {
      const client = new MockWebSocketClient();
      const challenge = {
        challenge: new Uint8Array([1, 2, 3]),
        credentialRequestOptions: new Uint8Array([4, 5, 6]),
      };

      await expect(
        handleAuthChallenge(client as any, challenge),
      ).rejects.toThrow('Passkey authentication requires a native module');
    });
  });

  describe('message encoding correctness', () => {
    it('register request payload is valid protobuf', () => {
      const payload = encodeAuthRegisterRequest({
        username: 'test-user',
        displayName: 'Test User',
      });

      // Should not throw when decoding
      const decoded = decodeAuthRegisterRequest(payload);
      expect(decoded.username).toBe('test-user');
      expect(decoded.displayName).toBe('Test User');
    });

    it('login request payload is valid protobuf', () => {
      const payload = encodeAuthRequest({ username: 'test-user' });
      const decoded = decodeAuthRequest(payload);
      expect(decoded.username).toBe('test-user');
    });
  });
});

describe('storage service', () => {
  // Import inline to avoid module-level side effects
  let storage: typeof import('../src/services/storage');

  beforeEach(async () => {
    storage = await import('../src/services/storage');
  });

  describe('saveAuthState and loadAuthState', () => {
    it('round-trips auth state for a server', async () => {
      const state = {
        userId: 'user-1',
        username: 'alice',
        displayName: 'Alice',
        sessionToken: 'tok-123',
      };

      await storage.saveAuthState('https://example.com', state);
      const loaded = await storage.loadAuthState('https://example.com');

      expect(loaded).toEqual(state);
    });

    it('returns null for unknown server', async () => {
      const loaded = await storage.loadAuthState('https://unknown.example.com');
      expect(loaded).toBeNull();
    });

    it('normalizes server URLs (trailing slash, case)', async () => {
      const state = {
        userId: 'user-1',
        username: 'alice',
        displayName: 'Alice',
        sessionToken: 'tok-123',
      };

      await storage.saveAuthState('HTTPS://EXAMPLE.COM/', state);
      const loaded = await storage.loadAuthState('https://example.com');

      expect(loaded).toEqual(state);
    });
  });

  describe('clearAuthState', () => {
    it('removes stored auth state', async () => {
      const state = {
        userId: 'user-1',
        username: 'alice',
        displayName: 'Alice',
        sessionToken: 'tok-123',
      };

      await storage.saveAuthState('https://example.com', state);
      await storage.clearAuthState('https://example.com');
      const loaded = await storage.loadAuthState('https://example.com');

      expect(loaded).toBeNull();
    });
  });

  describe('getSessionToken', () => {
    it('returns session token for stored server', async () => {
      const state = {
        userId: 'user-1',
        username: 'alice',
        displayName: 'Alice',
        sessionToken: 'tok-123',
      };

      await storage.saveAuthState('https://example.com', state);
      const token = await storage.getSessionToken('https://example.com');

      expect(token).toBe('tok-123');
    });

    it('returns null for unknown server', async () => {
      const token = await storage.getSessionToken('https://unknown.example.com');
      expect(token).toBeNull();
    });
  });

  describe('saveSessionToken', () => {
    it('updates session token for existing server', async () => {
      const state = {
        userId: 'user-1',
        username: 'alice',
        displayName: 'Alice',
        sessionToken: 'old-token',
      };

      await storage.saveAuthState('https://example.com', state);
      await storage.saveSessionToken('https://example.com', 'new-token');
      const token = await storage.getSessionToken('https://example.com');

      expect(token).toBe('new-token');
    });

    it('does nothing for unknown server (no state exists)', async () => {
      // Should not throw
      await storage.saveSessionToken('https://unknown.example.com', 'tok');
      const token = await storage.getSessionToken('https://unknown.example.com');
      expect(token).toBeNull();
    });
  });
});
