import {
  MessageType,
  encodeEnvelope,
  decodeEnvelope,
  encodePing,
  decodePing,
  encodePong,
  decodePong,
  decodeError,
  generateRequestId,
  messageTypeName,
  type Envelope,
  type MessageTypeValue,
} from '../src/services/protocol';

describe('protocol codec', () => {
  describe('Envelope round-trip', () => {
    it('encodes and decodes an Envelope with all fields matching', () => {
      const payload = new Uint8Array([1, 2, 3, 4]);
      const original: Envelope = {
        type: MessageType.MESSAGE_SEND,
        requestId: 'test-request-123',
        payload,
      };

      const encoded = encodeEnvelope(original);
      expect(encoded).toBeInstanceOf(Uint8Array);
      expect(encoded.length).toBeGreaterThan(0);

      const decoded = decodeEnvelope(encoded);
      expect(decoded.type).toBe(original.type);
      expect(decoded.requestId).toBe(original.requestId);
      expect(new Uint8Array(decoded.payload)).toEqual(payload);
    });

    it('round-trips every MessageType value', () => {
      for (const [name, typeValue] of Object.entries(MessageType)) {
        const original: Envelope = {
          type: typeValue as MessageTypeValue,
          requestId: `req-${name}`,
          payload: new Uint8Array([]),
        };
        const decoded = decodeEnvelope(encodeEnvelope(original));
        expect(decoded.type).toBe(typeValue);
        expect(decoded.requestId).toBe(original.requestId);
      }
    });
  });

  describe('Ping encode/decode', () => {
    it('round-trips a timestamp', () => {
      const timestamp = Date.now() * 1000;
      const encoded = encodePing({ timestamp });
      const decoded = decodePing(encoded);
      expect(decoded.timestamp).toBe(timestamp);
    });

    it('handles zero timestamp', () => {
      const decoded = decodePing(encodePing({ timestamp: 0 }));
      expect(decoded.timestamp).toBe(0);
    });
  });

  describe('Pong encode/decode', () => {
    it('round-trips a timestamp', () => {
      const timestamp = Date.now() * 1000;
      const encoded = encodePong({ timestamp });
      const decoded = decodePong(encoded);
      expect(decoded.timestamp).toBe(timestamp);
    });

    it('handles zero timestamp', () => {
      const decoded = decodePong(encodePong({ timestamp: 0 }));
      expect(decoded.timestamp).toBe(0);
    });
  });

  describe('Error decode', () => {
    it('decodes code, message, and fatal fields', () => {
      // Build an Error message using the protocol codec's internal type
      // We encode using protobufjs directly via the module's exported decoder
      const protobuf = require('protobufjs');
      const root = new protobuf.Root();
      const ErrorProto = new protobuf.Type('Error')
        .add(new protobuf.Field('code', 1, 'int32'))
        .add(new protobuf.Field('message', 2, 'string'))
        .add(new protobuf.Field('fatal', 3, 'bool'));
      root.add(ErrorProto);
      root.resolveAll();

      const msg = ErrorProto.create({ code: 404, message: 'not found', fatal: true });
      const encoded = ErrorProto.encode(msg).finish();
      const decoded = decodeError(new Uint8Array(encoded));

      expect(decoded.code).toBe(404);
      expect(decoded.message).toBe('not found');
      expect(decoded.fatal).toBe(true);
    });

    it('decodes non-fatal error with zero code', () => {
      const protobuf = require('protobufjs');
      const root = new protobuf.Root();
      const ErrorProto = new protobuf.Type('Error')
        .add(new protobuf.Field('code', 1, 'int32'))
        .add(new protobuf.Field('message', 2, 'string'))
        .add(new protobuf.Field('fatal', 3, 'bool'));
      root.add(ErrorProto);
      root.resolveAll();

      const msg = ErrorProto.create({ code: 0, message: 'ok', fatal: false });
      const encoded = ErrorProto.encode(msg).finish();
      const decoded = decodeError(new Uint8Array(encoded));

      expect(decoded.code).toBe(0);
      expect(decoded.message).toBe('ok');
      expect(decoded.fatal).toBe(false);
    });
  });

  describe('MessageType enum', () => {
    const expectedValues: Record<string, number> = {
      MESSAGE_TYPE_UNSPECIFIED: 0,
      AUTH_REQUEST: 1,
      AUTH_CHALLENGE: 2,
      AUTH_RESPONSE: 3,
      AUTH_SUCCESS: 4,
      AUTH_ERROR: 5,
      AUTH_REGISTER_REQUEST: 6,
      AUTH_REGISTER_CHALLENGE: 7,
      AUTH_REGISTER_RESPONSE: 8,
      AUTH_REGISTER_SUCCESS: 9,
      MESSAGE_SEND: 20,
      MESSAGE_RECEIVE: 21,
      MESSAGE_ACK: 22,
      MESSAGE_DELIVERED: 23,
      GROUP_CREATE: 30,
      GROUP_CREATED: 31,
      GROUP_INVITE: 32,
      GROUP_MEMBER_ADDED: 33,
      GROUP_MEMBER_REMOVED: 34,
      GROUP_LEAVE: 35,
      MLS_KEY_PACKAGE_UPLOAD: 40,
      MLS_KEY_PACKAGE_FETCH: 41,
      MLS_KEY_PACKAGE_RESPONSE: 42,
      MLS_WELCOME: 43,
      MLS_WELCOME_RECEIVE: 44,
      MLS_COMMIT: 45,
      MLS_COMMIT_BROADCAST: 46,
      PRESENCE_UPDATE: 50,
      PRESENCE_NOTIFY: 51,
      PING: 60,
      PONG: 61,
      ERROR: 62,
    };

    it.each(Object.entries(expectedValues))(
      '%s has value %d',
      (name, value) => {
        expect(MessageType[name as keyof typeof MessageType]).toBe(value);
      },
    );

    it('has the expected number of entries', () => {
      expect(Object.keys(MessageType).length).toBe(Object.keys(expectedValues).length);
    });
  });

  describe('generateRequestId', () => {
    const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;

    it('returns a valid UUID v4 format', () => {
      const id = generateRequestId();
      expect(id).toMatch(UUID_RE);
    });

    it('generates unique values across multiple calls', () => {
      const ids = new Set(Array.from({ length: 100 }, () => generateRequestId()));
      expect(ids.size).toBe(100);
    });

    it('sets version nibble to 4', () => {
      const id = generateRequestId();
      expect(id[14]).toBe('4');
    });

    it('sets variant nibble to 8, 9, a, or b', () => {
      const id = generateRequestId();
      expect(['8', '9', 'a', 'b']).toContain(id[19]);
    });
  });

  describe('messageTypeName', () => {
    it('returns correct name for known types', () => {
      expect(messageTypeName(MessageType.PING)).toBe('PING');
      expect(messageTypeName(MessageType.PONG)).toBe('PONG');
      expect(messageTypeName(MessageType.ERROR)).toBe('ERROR');
      expect(messageTypeName(MessageType.AUTH_REQUEST)).toBe('AUTH_REQUEST');
      expect(messageTypeName(MessageType.MESSAGE_SEND)).toBe('MESSAGE_SEND');
    });

    it('returns UNKNOWN(n) for unknown types', () => {
      expect(messageTypeName(999 as MessageTypeValue)).toBe('UNKNOWN(999)');
    });

    it('returns correct name for MESSAGE_TYPE_UNSPECIFIED', () => {
      expect(messageTypeName(MessageType.MESSAGE_TYPE_UNSPECIFIED)).toBe('MESSAGE_TYPE_UNSPECIFIED');
    });
  });

  describe('empty payload handling', () => {
    it('round-trips an Envelope with empty payload', () => {
      const original: Envelope = {
        type: MessageType.PING,
        requestId: 'empty-payload-test',
        payload: new Uint8Array([]),
      };

      const decoded = decodeEnvelope(encodeEnvelope(original));
      expect(decoded.type).toBe(MessageType.PING);
      expect(decoded.requestId).toBe('empty-payload-test');
      // Empty payload may decode as empty Uint8Array or be absent
      expect(decoded.payload.length).toBe(0);
    });
  });
});
