import {
  MessageType,
  encodeEnvelope,
  decodeEnvelope,
  encodeAuthRequest,
  decodeAuthRequest,
  encodeAuthChallenge,
  decodeAuthChallenge,
  encodeAuthResponse,
  decodeAuthResponse,
  decodeAuthSuccess,
  decodeAuthError,
  encodeAuthRegisterRequest,
  decodeAuthRegisterRequest,
  decodeAuthRegisterChallenge,
  encodeAuthRegisterResponse,
  decodeAuthRegisterResponse,
  decodeAuthRegisterSuccess,
  type AuthRequestMessage,
  type AuthChallengeMessage,
  type AuthResponseMessage,
  type AuthRegisterRequestMessage,
  type AuthRegisterResponseMessage,
  type Envelope,
} from '../src/services/protocol';

describe('auth protocol codec', () => {
  describe('AuthRequest round-trip', () => {
    it('encodes and decodes with matching fields', () => {
      const original: AuthRequestMessage = { username: 'alice' };
      const encoded = encodeAuthRequest(original);
      expect(encoded).toBeInstanceOf(Uint8Array);
      expect(encoded.length).toBeGreaterThan(0);

      const decoded = decodeAuthRequest(encoded);
      expect(decoded.username).toBe('alice');
    });

    it('handles empty username', () => {
      const original: AuthRequestMessage = { username: '' };
      const decoded = decodeAuthRequest(encodeAuthRequest(original));
      expect(decoded.username).toBe('');
    });

    it('handles unicode username', () => {
      const original: AuthRequestMessage = { username: 'alice\u00e9\ud83d\ude00' };
      const decoded = decodeAuthRequest(encodeAuthRequest(original));
      expect(decoded.username).toBe('alice\u00e9\ud83d\ude00');
    });
  });

  describe('AuthChallenge round-trip', () => {
    it('encodes and decodes with matching fields', () => {
      const original: AuthChallengeMessage = {
        challenge: new Uint8Array([1, 2, 3, 4, 5]),
        credentialRequestOptions: new Uint8Array([10, 20, 30]),
      };
      const encoded = encodeAuthChallenge(original);
      const decoded = decodeAuthChallenge(encoded);

      expect(new Uint8Array(decoded.challenge)).toEqual(original.challenge);
      expect(new Uint8Array(decoded.credentialRequestOptions)).toEqual(
        original.credentialRequestOptions,
      );
    });

    it('handles empty byte arrays', () => {
      const original: AuthChallengeMessage = {
        challenge: new Uint8Array([]),
        credentialRequestOptions: new Uint8Array([]),
      };
      const decoded = decodeAuthChallenge(encodeAuthChallenge(original));
      expect(decoded.challenge.length).toBe(0);
      expect(decoded.credentialRequestOptions.length).toBe(0);
    });
  });

  describe('AuthResponse round-trip', () => {
    it('encodes and decodes with matching fields', () => {
      const original: AuthResponseMessage = {
        credentialId: new Uint8Array([1, 2, 3]),
        authenticatorData: new Uint8Array([4, 5, 6]),
        clientDataJson: new Uint8Array([7, 8, 9]),
        signature: new Uint8Array([10, 11, 12]),
      };
      const encoded = encodeAuthResponse(original);
      const decoded = decodeAuthResponse(encoded);

      expect(new Uint8Array(decoded.credentialId)).toEqual(original.credentialId);
      expect(new Uint8Array(decoded.authenticatorData)).toEqual(original.authenticatorData);
      expect(new Uint8Array(decoded.clientDataJson)).toEqual(original.clientDataJson);
      expect(new Uint8Array(decoded.signature)).toEqual(original.signature);
    });
  });

  describe('AuthSuccess decode', () => {
    it('decodes all fields', () => {
      // Build an AuthSuccess using protobufjs directly
      const protobuf = require('protobufjs');
      const root = new protobuf.Root();
      const AuthSuccessProto = new protobuf.Type('AuthSuccess')
        .add(new protobuf.Field('sessionToken', 1, 'string'))
        .add(new protobuf.Field('userId', 2, 'string'))
        .add(new protobuf.Field('username', 3, 'string'))
        .add(new protobuf.Field('displayName', 4, 'string'));
      root.add(AuthSuccessProto);
      root.resolveAll();

      const msg = AuthSuccessProto.create({
        sessionToken: 'tok-123',
        userId: 'user-1',
        username: 'alice',
        displayName: 'Alice Wonderland',
      });
      const encoded = AuthSuccessProto.encode(msg).finish();
      const decoded = decodeAuthSuccess(new Uint8Array(encoded));

      expect(decoded.sessionToken).toBe('tok-123');
      expect(decoded.userId).toBe('user-1');
      expect(decoded.username).toBe('alice');
      expect(decoded.displayName).toBe('Alice Wonderland');
    });
  });

  describe('AuthError decode', () => {
    it('decodes error code and message', () => {
      const protobuf = require('protobufjs');
      const root = new protobuf.Root();
      const AuthErrorProto = new protobuf.Type('AuthError')
        .add(new protobuf.Field('errorCode', 1, 'int32'))
        .add(new protobuf.Field('message', 2, 'string'));
      root.add(AuthErrorProto);
      root.resolveAll();

      const msg = AuthErrorProto.create({
        errorCode: 4001,
        message: 'authentication failed',
      });
      const encoded = AuthErrorProto.encode(msg).finish();
      const decoded = decodeAuthError(new Uint8Array(encoded));

      expect(decoded.errorCode).toBe(4001);
      expect(decoded.message).toBe('authentication failed');
    });

    it('handles zero error code', () => {
      const protobuf = require('protobufjs');
      const root = new protobuf.Root();
      const AuthErrorProto = new protobuf.Type('AuthError')
        .add(new protobuf.Field('errorCode', 1, 'int32'))
        .add(new protobuf.Field('message', 2, 'string'));
      root.add(AuthErrorProto);
      root.resolveAll();

      const msg = AuthErrorProto.create({ errorCode: 0, message: '' });
      const encoded = AuthErrorProto.encode(msg).finish();
      const decoded = decodeAuthError(new Uint8Array(encoded));

      expect(decoded.errorCode).toBe(0);
      expect(decoded.message).toBe('');
    });
  });

  describe('AuthRegisterRequest round-trip', () => {
    it('encodes and decodes with matching fields', () => {
      const original: AuthRegisterRequestMessage = {
        username: 'alice',
        displayName: 'Alice Wonderland',
      };
      const encoded = encodeAuthRegisterRequest(original);
      const decoded = decodeAuthRegisterRequest(encoded);

      expect(decoded.username).toBe('alice');
      expect(decoded.displayName).toBe('Alice Wonderland');
    });

    it('handles empty display name', () => {
      const original: AuthRegisterRequestMessage = {
        username: 'alice',
        displayName: '',
      };
      const decoded = decodeAuthRegisterRequest(encodeAuthRegisterRequest(original));
      expect(decoded.username).toBe('alice');
      expect(decoded.displayName).toBe('');
    });
  });

  describe('AuthRegisterChallenge decode', () => {
    it('decodes challenge and creation options', () => {
      const protobuf = require('protobufjs');
      const root = new protobuf.Root();
      const AuthRegisterChallengeProto = new protobuf.Type('AuthRegisterChallenge')
        .add(new protobuf.Field('challenge', 1, 'bytes'))
        .add(new protobuf.Field('credentialCreationOptions', 2, 'bytes'));
      root.add(AuthRegisterChallengeProto);
      root.resolveAll();

      const challengeBytes = new Uint8Array([1, 2, 3, 4]);
      const optionsBytes = new Uint8Array([5, 6, 7, 8]);

      const msg = AuthRegisterChallengeProto.create({
        challenge: challengeBytes,
        credentialCreationOptions: optionsBytes,
      });
      const encoded = AuthRegisterChallengeProto.encode(msg).finish();
      const decoded = decodeAuthRegisterChallenge(new Uint8Array(encoded));

      expect(new Uint8Array(decoded.challenge)).toEqual(challengeBytes);
      expect(new Uint8Array(decoded.credentialCreationOptions)).toEqual(optionsBytes);
    });
  });

  describe('AuthRegisterResponse round-trip', () => {
    it('encodes and decodes with matching fields', () => {
      const original: AuthRegisterResponseMessage = {
        credentialId: new Uint8Array([1, 2]),
        authenticatorData: new Uint8Array([3, 4]),
        clientDataJson: new Uint8Array([5, 6]),
        attestationObject: new Uint8Array([7, 8]),
      };
      const encoded = encodeAuthRegisterResponse(original);
      const decoded = decodeAuthRegisterResponse(encoded);

      expect(new Uint8Array(decoded.credentialId)).toEqual(original.credentialId);
      expect(new Uint8Array(decoded.authenticatorData)).toEqual(original.authenticatorData);
      expect(new Uint8Array(decoded.clientDataJson)).toEqual(original.clientDataJson);
      expect(new Uint8Array(decoded.attestationObject)).toEqual(original.attestationObject);
    });
  });

  describe('AuthRegisterSuccess decode', () => {
    it('decodes userId and sessionToken', () => {
      const protobuf = require('protobufjs');
      const root = new protobuf.Root();
      const AuthRegisterSuccessProto = new protobuf.Type('AuthRegisterSuccess')
        .add(new protobuf.Field('userId', 1, 'string'))
        .add(new protobuf.Field('sessionToken', 2, 'string'));
      root.add(AuthRegisterSuccessProto);
      root.resolveAll();

      const msg = AuthRegisterSuccessProto.create({
        userId: 'user-1',
        sessionToken: 'tok-abc',
      });
      const encoded = AuthRegisterSuccessProto.encode(msg).finish();
      const decoded = decodeAuthRegisterSuccess(new Uint8Array(encoded));

      expect(decoded.userId).toBe('user-1');
      expect(decoded.sessionToken).toBe('tok-abc');
    });
  });

  describe('Auth message envelope integration', () => {
    it('wraps AuthRequest in envelope and round-trips', () => {
      const authPayload = encodeAuthRequest({ username: 'alice' });
      const envelope: Envelope = {
        type: MessageType.AUTH_REQUEST,
        requestId: 'req-auth-1',
        payload: authPayload,
      };
      const encoded = encodeEnvelope(envelope);
      const decoded = decodeEnvelope(encoded);

      expect(decoded.type).toBe(MessageType.AUTH_REQUEST);
      expect(decoded.requestId).toBe('req-auth-1');
      const authMsg = decodeAuthRequest(decoded.payload);
      expect(authMsg.username).toBe('alice');
    });

    it('wraps AuthRegisterRequest in envelope and round-trips', () => {
      const regPayload = encodeAuthRegisterRequest({
        username: 'bob',
        displayName: 'Bob Builder',
      });
      const envelope: Envelope = {
        type: MessageType.AUTH_REGISTER_REQUEST,
        requestId: 'req-reg-1',
        payload: regPayload,
      };
      const encoded = encodeEnvelope(envelope);
      const decoded = decodeEnvelope(encoded);

      expect(decoded.type).toBe(MessageType.AUTH_REGISTER_REQUEST);
      expect(decoded.requestId).toBe('req-reg-1');
      const regMsg = decodeAuthRegisterRequest(decoded.payload);
      expect(regMsg.username).toBe('bob');
      expect(regMsg.displayName).toBe('Bob Builder');
    });

    it('wraps AuthResponse in envelope and round-trips', () => {
      const respPayload = encodeAuthResponse({
        credentialId: new Uint8Array([1]),
        authenticatorData: new Uint8Array([2]),
        clientDataJson: new Uint8Array([3]),
        signature: new Uint8Array([4]),
      });
      const envelope: Envelope = {
        type: MessageType.AUTH_RESPONSE,
        requestId: 'req-resp-1',
        payload: respPayload,
      };
      const encoded = encodeEnvelope(envelope);
      const decoded = decodeEnvelope(encoded);

      expect(decoded.type).toBe(MessageType.AUTH_RESPONSE);
      const respMsg = decodeAuthResponse(decoded.payload);
      expect(new Uint8Array(respMsg.credentialId)).toEqual(new Uint8Array([1]));
    });
  });
});
