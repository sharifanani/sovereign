// Protocol codec service
// Handles protobuf serialization/deserialization for the Sovereign protocol

import * as protobuf from 'protobufjs';

// MessageType enum values matching protocol/messages.proto
export const MessageType = {
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
} as const;

export type MessageTypeValue = (typeof MessageType)[keyof typeof MessageType];

// Build protobuf types using reflection API
const root = new protobuf.Root();

const MessageTypeEnum = new protobuf.Enum('MessageType', {
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
});

root.add(MessageTypeEnum);

const EnvelopeType = new protobuf.Type('Envelope')
  .add(new protobuf.Field('type', 1, 'MessageType'))
  .add(new protobuf.Field('requestId', 2, 'string'))
  .add(new protobuf.Field('payload', 3, 'bytes'));

root.add(EnvelopeType);

const PingType = new protobuf.Type('Ping')
  .add(new protobuf.Field('timestamp', 1, 'int64'));

root.add(PingType);

const PongType = new protobuf.Type('Pong')
  .add(new protobuf.Field('timestamp', 1, 'int64'));

root.add(PongType);

const ErrorType = new protobuf.Type('Error')
  .add(new protobuf.Field('code', 1, 'int32'))
  .add(new protobuf.Field('message', 2, 'string'))
  .add(new protobuf.Field('fatal', 3, 'bool'));

root.add(ErrorType);

// Authentication message types
const AuthRequestType = new protobuf.Type('AuthRequest')
  .add(new protobuf.Field('username', 1, 'string'));

root.add(AuthRequestType);

const AuthChallengeType = new protobuf.Type('AuthChallenge')
  .add(new protobuf.Field('challenge', 1, 'bytes'))
  .add(new protobuf.Field('credentialRequestOptions', 2, 'bytes'));

root.add(AuthChallengeType);

const AuthResponseType = new protobuf.Type('AuthResponse')
  .add(new protobuf.Field('credentialId', 1, 'bytes'))
  .add(new protobuf.Field('authenticatorData', 2, 'bytes'))
  .add(new protobuf.Field('clientDataJson', 3, 'bytes'))
  .add(new protobuf.Field('signature', 4, 'bytes'));

root.add(AuthResponseType);

const AuthSuccessType = new protobuf.Type('AuthSuccess')
  .add(new protobuf.Field('sessionToken', 1, 'string'))
  .add(new protobuf.Field('userId', 2, 'string'))
  .add(new protobuf.Field('username', 3, 'string'))
  .add(new protobuf.Field('displayName', 4, 'string'));

root.add(AuthSuccessType);

const AuthErrorType = new protobuf.Type('AuthError')
  .add(new protobuf.Field('errorCode', 1, 'int32'))
  .add(new protobuf.Field('message', 2, 'string'));

root.add(AuthErrorType);

const AuthRegisterRequestType = new protobuf.Type('AuthRegisterRequest')
  .add(new protobuf.Field('username', 1, 'string'))
  .add(new protobuf.Field('displayName', 2, 'string'));

root.add(AuthRegisterRequestType);

const AuthRegisterChallengeType = new protobuf.Type('AuthRegisterChallenge')
  .add(new protobuf.Field('challenge', 1, 'bytes'))
  .add(new protobuf.Field('credentialCreationOptions', 2, 'bytes'));

root.add(AuthRegisterChallengeType);

const AuthRegisterResponseType = new protobuf.Type('AuthRegisterResponse')
  .add(new protobuf.Field('credentialId', 1, 'bytes'))
  .add(new protobuf.Field('authenticatorData', 2, 'bytes'))
  .add(new protobuf.Field('clientDataJson', 3, 'bytes'))
  .add(new protobuf.Field('attestationObject', 4, 'bytes'));

root.add(AuthRegisterResponseType);

const AuthRegisterSuccessType = new protobuf.Type('AuthRegisterSuccess')
  .add(new protobuf.Field('userId', 1, 'string'))
  .add(new protobuf.Field('sessionToken', 2, 'string'));

root.add(AuthRegisterSuccessType);

// Resolve all type references
root.resolveAll();

// Exported types for use in other modules
export interface Envelope {
  type: MessageTypeValue;
  requestId: string;
  payload: Uint8Array;
}

export interface PingMessage {
  timestamp: number | Long;
}

export interface PongMessage {
  timestamp: number | Long;
}

export interface ErrorMessage {
  code: number;
  message: string;
  fatal: boolean;
}

// Long type for 64-bit integers (protobufjs uses number or { low: number, high: number, unsigned: boolean })
interface Long {
  low: number;
  high: number;
  unsigned: boolean;
}

// Encode an Envelope to binary
export function encodeEnvelope(envelope: Envelope): Uint8Array {
  const err = EnvelopeType.verify({
    type: envelope.type,
    requestId: envelope.requestId,
    payload: envelope.payload,
  });
  if (err) {
    throw new globalThis.Error(`Envelope verification failed: ${err}`);
  }
  const message = EnvelopeType.create({
    type: envelope.type,
    requestId: envelope.requestId,
    payload: envelope.payload,
  });
  return EnvelopeType.encode(message).finish();
}

// Decode binary data to an Envelope
export function decodeEnvelope(data: Uint8Array): Envelope {
  const decoded = EnvelopeType.decode(data);
  const obj = EnvelopeType.toObject(decoded, {
    bytes: Uint8Array,
    longs: Number,
  });
  return {
    type: obj.type as MessageTypeValue,
    requestId: obj.requestId as string,
    payload: obj.payload as Uint8Array,
  };
}

// Encode a Ping message payload
export function encodePing(ping: PingMessage): Uint8Array {
  const message = PingType.create({ timestamp: ping.timestamp });
  return PingType.encode(message).finish();
}

// Decode a Ping message payload
export function decodePing(data: Uint8Array): PingMessage {
  const decoded = PingType.decode(data);
  const obj = PingType.toObject(decoded, { longs: Number });
  return { timestamp: obj.timestamp as number };
}

// Encode a Pong message payload
export function encodePong(pong: PongMessage): Uint8Array {
  const message = PongType.create({ timestamp: pong.timestamp });
  return PongType.encode(message).finish();
}

// Decode a Pong message payload
export function decodePong(data: Uint8Array): PongMessage {
  const decoded = PongType.decode(data);
  const obj = PongType.toObject(decoded, { longs: Number });
  return { timestamp: obj.timestamp as number };
}

// Decode an Error message payload
export function decodeError(data: Uint8Array): ErrorMessage {
  const decoded = ErrorType.decode(data);
  const obj = ErrorType.toObject(decoded, { longs: Number });
  return {
    code: obj.code as number,
    message: obj.message as string,
    fatal: obj.fatal as boolean,
  };
}

// Generate a UUIDv4-style request ID
export function generateRequestId(): string {
  const bytes = new Uint8Array(16);
  for (let i = 0; i < 16; i++) {
    bytes[i] = Math.floor(Math.random() * 256);
  }
  // Set version (4) and variant (RFC 4122)
  bytes[6] = (bytes[6] & 0x0f) | 0x40;
  bytes[8] = (bytes[8] & 0x3f) | 0x80;

  const hex = Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('');
  return [
    hex.slice(0, 8),
    hex.slice(8, 12),
    hex.slice(12, 16),
    hex.slice(16, 20),
    hex.slice(20, 32),
  ].join('-');
}

// Helper: get a human-readable name for a message type
export function messageTypeName(type: MessageTypeValue): string {
  const entry = Object.entries(MessageType).find(([, v]) => v === type);
  return entry ? entry[0] : `UNKNOWN(${type})`;
}

// ============================================================================
// Auth message interfaces
// ============================================================================

export interface AuthRequestMessage {
  username: string;
}

export interface AuthChallengeMessage {
  challenge: Uint8Array;
  credentialRequestOptions: Uint8Array;
}

export interface AuthResponseMessage {
  credentialId: Uint8Array;
  authenticatorData: Uint8Array;
  clientDataJson: Uint8Array;
  signature: Uint8Array;
}

export interface AuthSuccessMessage {
  sessionToken: string;
  userId: string;
  username: string;
  displayName: string;
}

export interface AuthErrorMessage {
  errorCode: number;
  message: string;
}

export interface AuthRegisterRequestMessage {
  username: string;
  displayName: string;
}

export interface AuthRegisterChallengeMessage {
  challenge: Uint8Array;
  credentialCreationOptions: Uint8Array;
}

export interface AuthRegisterResponseMessage {
  credentialId: Uint8Array;
  authenticatorData: Uint8Array;
  clientDataJson: Uint8Array;
  attestationObject: Uint8Array;
}

export interface AuthRegisterSuccessMessage {
  userId: string;
  sessionToken: string;
}

// ============================================================================
// Auth message encode/decode functions
// ============================================================================

export function encodeAuthRequest(msg: AuthRequestMessage): Uint8Array {
  const message = AuthRequestType.create({ username: msg.username });
  return AuthRequestType.encode(message).finish();
}

export function decodeAuthRequest(data: Uint8Array): AuthRequestMessage {
  const decoded = AuthRequestType.decode(data);
  const obj = AuthRequestType.toObject(decoded);
  return { username: obj.username as string };
}

export function encodeAuthChallenge(msg: AuthChallengeMessage): Uint8Array {
  const message = AuthChallengeType.create({
    challenge: msg.challenge,
    credentialRequestOptions: msg.credentialRequestOptions,
  });
  return AuthChallengeType.encode(message).finish();
}

export function decodeAuthChallenge(data: Uint8Array): AuthChallengeMessage {
  const decoded = AuthChallengeType.decode(data);
  const obj = AuthChallengeType.toObject(decoded, { bytes: Uint8Array });
  return {
    challenge: obj.challenge as Uint8Array,
    credentialRequestOptions: obj.credentialRequestOptions as Uint8Array,
  };
}

export function encodeAuthResponse(msg: AuthResponseMessage): Uint8Array {
  const message = AuthResponseType.create({
    credentialId: msg.credentialId,
    authenticatorData: msg.authenticatorData,
    clientDataJson: msg.clientDataJson,
    signature: msg.signature,
  });
  return AuthResponseType.encode(message).finish();
}

export function decodeAuthResponse(data: Uint8Array): AuthResponseMessage {
  const decoded = AuthResponseType.decode(data);
  const obj = AuthResponseType.toObject(decoded, { bytes: Uint8Array });
  return {
    credentialId: obj.credentialId as Uint8Array,
    authenticatorData: obj.authenticatorData as Uint8Array,
    clientDataJson: obj.clientDataJson as Uint8Array,
    signature: obj.signature as Uint8Array,
  };
}

export function decodeAuthSuccess(data: Uint8Array): AuthSuccessMessage {
  const decoded = AuthSuccessType.decode(data);
  const obj = AuthSuccessType.toObject(decoded);
  return {
    sessionToken: obj.sessionToken as string,
    userId: obj.userId as string,
    username: obj.username as string,
    displayName: obj.displayName as string,
  };
}

export function decodeAuthError(data: Uint8Array): AuthErrorMessage {
  const decoded = AuthErrorType.decode(data);
  const obj = AuthErrorType.toObject(decoded);
  return {
    errorCode: obj.errorCode as number,
    message: obj.message as string,
  };
}

export function encodeAuthRegisterRequest(msg: AuthRegisterRequestMessage): Uint8Array {
  const message = AuthRegisterRequestType.create({
    username: msg.username,
    displayName: msg.displayName,
  });
  return AuthRegisterRequestType.encode(message).finish();
}

export function decodeAuthRegisterRequest(data: Uint8Array): AuthRegisterRequestMessage {
  const decoded = AuthRegisterRequestType.decode(data);
  const obj = AuthRegisterRequestType.toObject(decoded);
  return {
    username: obj.username as string,
    displayName: obj.displayName as string,
  };
}

export function decodeAuthRegisterChallenge(data: Uint8Array): AuthRegisterChallengeMessage {
  const decoded = AuthRegisterChallengeType.decode(data);
  const obj = AuthRegisterChallengeType.toObject(decoded, { bytes: Uint8Array });
  return {
    challenge: obj.challenge as Uint8Array,
    credentialCreationOptions: obj.credentialCreationOptions as Uint8Array,
  };
}

export function encodeAuthRegisterResponse(msg: AuthRegisterResponseMessage): Uint8Array {
  const message = AuthRegisterResponseType.create({
    credentialId: msg.credentialId,
    authenticatorData: msg.authenticatorData,
    clientDataJson: msg.clientDataJson,
    attestationObject: msg.attestationObject,
  });
  return AuthRegisterResponseType.encode(message).finish();
}

export function decodeAuthRegisterResponse(data: Uint8Array): AuthRegisterResponseMessage {
  const decoded = AuthRegisterResponseType.decode(data);
  const obj = AuthRegisterResponseType.toObject(decoded, { bytes: Uint8Array });
  return {
    credentialId: obj.credentialId as Uint8Array,
    authenticatorData: obj.authenticatorData as Uint8Array,
    clientDataJson: obj.clientDataJson as Uint8Array,
    attestationObject: obj.attestationObject as Uint8Array,
  };
}

export function decodeAuthRegisterSuccess(data: Uint8Array): AuthRegisterSuccessMessage {
  const decoded = AuthRegisterSuccessType.decode(data);
  const obj = AuthRegisterSuccessType.toObject(decoded);
  return {
    userId: obj.userId as string,
    sessionToken: obj.sessionToken as string,
  };
}
