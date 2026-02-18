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

// ============================================================================
// Messaging message types
// ============================================================================

const MessageSendType = new protobuf.Type('MessageSend')
  .add(new protobuf.Field('conversationId', 1, 'string'))
  .add(new protobuf.Field('encryptedPayload', 2, 'bytes'))
  .add(new protobuf.Field('messageType', 3, 'string'));

root.add(MessageSendType);

const MessageReceiveType = new protobuf.Type('MessageReceive')
  .add(new protobuf.Field('messageId', 1, 'string'))
  .add(new protobuf.Field('conversationId', 2, 'string'))
  .add(new protobuf.Field('senderId', 3, 'string'))
  .add(new protobuf.Field('encryptedPayload', 4, 'bytes'))
  .add(new protobuf.Field('serverTimestamp', 5, 'int64'))
  .add(new protobuf.Field('messageType', 6, 'string'));

root.add(MessageReceiveType);

const MessageAckType = new protobuf.Type('MessageAck')
  .add(new protobuf.Field('messageId', 1, 'string'));

root.add(MessageAckType);

const MessageDeliveredType = new protobuf.Type('MessageDelivered')
  .add(new protobuf.Field('messageId', 1, 'string'))
  .add(new protobuf.Field('deliveredTo', 2, 'string'));

root.add(MessageDeliveredType);

// ============================================================================
// Group message types
// ============================================================================

const GroupMemberType = new protobuf.Type('GroupMember')
  .add(new protobuf.Field('userId', 1, 'string'))
  .add(new protobuf.Field('username', 2, 'string'))
  .add(new protobuf.Field('displayName', 3, 'string'))
  .add(new protobuf.Field('role', 4, 'string'));

root.add(GroupMemberType);

const GroupCreateType = new protobuf.Type('GroupCreate')
  .add(new protobuf.Field('title', 1, 'string'))
  .add(new protobuf.Field('memberIds', 2, 'string', 'repeated'));

root.add(GroupCreateType);

const GroupCreatedType = new protobuf.Type('GroupCreated')
  .add(new protobuf.Field('conversationId', 1, 'string'))
  .add(new protobuf.Field('title', 2, 'string'))
  .add(new protobuf.Field('members', 3, 'GroupMember', 'repeated'));

root.add(GroupCreatedType);

const GroupInviteType = new protobuf.Type('GroupInvite')
  .add(new protobuf.Field('conversationId', 1, 'string'))
  .add(new protobuf.Field('userId', 2, 'string'));

root.add(GroupInviteType);

const GroupMemberAddedType = new protobuf.Type('GroupMemberAdded')
  .add(new protobuf.Field('conversationId', 1, 'string'))
  .add(new protobuf.Field('userId', 2, 'string'))
  .add(new protobuf.Field('addedBy', 3, 'string'));

root.add(GroupMemberAddedType);

const GroupMemberRemovedType = new protobuf.Type('GroupMemberRemoved')
  .add(new protobuf.Field('conversationId', 1, 'string'))
  .add(new protobuf.Field('userId', 2, 'string'))
  .add(new protobuf.Field('removedBy', 3, 'string'));

root.add(GroupMemberRemovedType);

const GroupLeaveType = new protobuf.Type('GroupLeave')
  .add(new protobuf.Field('conversationId', 1, 'string'));

root.add(GroupLeaveType);

// ============================================================================
// MLS Key Management message types
// ============================================================================

const MLSKeyPackageUploadType = new protobuf.Type('MLSKeyPackageUpload')
  .add(new protobuf.Field('keyPackageData', 1, 'bytes'));

root.add(MLSKeyPackageUploadType);

const MLSKeyPackageFetchType = new protobuf.Type('MLSKeyPackageFetch')
  .add(new protobuf.Field('userId', 1, 'string'));

root.add(MLSKeyPackageFetchType);

const MLSKeyPackageResponseType = new protobuf.Type('MLSKeyPackageResponse')
  .add(new protobuf.Field('userId', 1, 'string'))
  .add(new protobuf.Field('keyPackageData', 2, 'bytes'));

root.add(MLSKeyPackageResponseType);

const MLSWelcomeType = new protobuf.Type('MLSWelcome')
  .add(new protobuf.Field('conversationId', 1, 'string'))
  .add(new protobuf.Field('recipientId', 2, 'string'))
  .add(new protobuf.Field('welcomeData', 3, 'bytes'));

root.add(MLSWelcomeType);

const MLSWelcomeReceiveType = new protobuf.Type('MLSWelcomeReceive')
  .add(new protobuf.Field('conversationId', 1, 'string'))
  .add(new protobuf.Field('senderId', 2, 'string'))
  .add(new protobuf.Field('welcomeData', 3, 'bytes'));

root.add(MLSWelcomeReceiveType);

const MLSCommitType = new protobuf.Type('MLSCommit')
  .add(new protobuf.Field('conversationId', 1, 'string'))
  .add(new protobuf.Field('commitData', 2, 'bytes'));

root.add(MLSCommitType);

const MLSCommitBroadcastType = new protobuf.Type('MLSCommitBroadcast')
  .add(new protobuf.Field('conversationId', 1, 'string'))
  .add(new protobuf.Field('senderId', 2, 'string'))
  .add(new protobuf.Field('commitData', 3, 'bytes'));

root.add(MLSCommitBroadcastType);

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

// ============================================================================
// Messaging message interfaces
// ============================================================================

export interface MessageSendMessage {
  conversationId: string;
  encryptedPayload: Uint8Array;
  messageType: string;
}

export interface MessageReceiveMessage {
  messageId: string;
  conversationId: string;
  senderId: string;
  encryptedPayload: Uint8Array;
  serverTimestamp: number;
  messageType: string;
}

export interface MessageAckMessage {
  messageId: string;
}

export interface MessageDeliveredMessage {
  messageId: string;
  deliveredTo: string;
}

// ============================================================================
// Group message interfaces
// ============================================================================

export interface GroupMemberInfo {
  userId: string;
  username: string;
  displayName: string;
  role: string;
}

export interface GroupCreateMessage {
  title: string;
  memberIds: string[];
}

export interface GroupCreatedMessage {
  conversationId: string;
  title: string;
  members: GroupMemberInfo[];
}

export interface GroupInviteMessage {
  conversationId: string;
  userId: string;
}

export interface GroupMemberAddedMessage {
  conversationId: string;
  userId: string;
  addedBy: string;
}

export interface GroupMemberRemovedMessage {
  conversationId: string;
  userId: string;
  removedBy: string;
}

export interface GroupLeaveMessage {
  conversationId: string;
}

// ============================================================================
// MLS Key Management interfaces
// ============================================================================

export interface MLSKeyPackageUploadMessage {
  keyPackageData: Uint8Array;
}

export interface MLSKeyPackageFetchMessage {
  userId: string;
}

export interface MLSKeyPackageResponseMessage {
  userId: string;
  keyPackageData: Uint8Array;
}

export interface MLSWelcomeMessage {
  conversationId: string;
  recipientId: string;
  welcomeData: Uint8Array;
}

export interface MLSWelcomeReceiveMessage {
  conversationId: string;
  senderId: string;
  welcomeData: Uint8Array;
}

export interface MLSCommitMessage {
  conversationId: string;
  commitData: Uint8Array;
}

export interface MLSCommitBroadcastMessage {
  conversationId: string;
  senderId: string;
  commitData: Uint8Array;
}

// ============================================================================
// Messaging encode/decode functions
// ============================================================================

export function encodeMessageSend(msg: MessageSendMessage): Uint8Array {
  const message = MessageSendType.create({
    conversationId: msg.conversationId,
    encryptedPayload: msg.encryptedPayload,
    messageType: msg.messageType,
  });
  return MessageSendType.encode(message).finish();
}

export function decodeMessageReceive(data: Uint8Array): MessageReceiveMessage {
  const decoded = MessageReceiveType.decode(data);
  const obj = MessageReceiveType.toObject(decoded, { bytes: Uint8Array, longs: Number });
  return {
    messageId: obj.messageId as string,
    conversationId: obj.conversationId as string,
    senderId: obj.senderId as string,
    encryptedPayload: obj.encryptedPayload as Uint8Array,
    serverTimestamp: obj.serverTimestamp as number,
    messageType: obj.messageType as string,
  };
}

export function encodeMessageAck(msg: MessageAckMessage): Uint8Array {
  const message = MessageAckType.create({ messageId: msg.messageId });
  return MessageAckType.encode(message).finish();
}

export function decodeMessageDelivered(data: Uint8Array): MessageDeliveredMessage {
  const decoded = MessageDeliveredType.decode(data);
  const obj = MessageDeliveredType.toObject(decoded);
  return {
    messageId: obj.messageId as string,
    deliveredTo: obj.deliveredTo as string,
  };
}

// ============================================================================
// Group encode/decode functions
// ============================================================================

export function encodeGroupCreate(msg: GroupCreateMessage): Uint8Array {
  const message = GroupCreateType.create({
    title: msg.title,
    memberIds: msg.memberIds,
  });
  return GroupCreateType.encode(message).finish();
}

export function decodeGroupCreated(data: Uint8Array): GroupCreatedMessage {
  const decoded = GroupCreatedType.decode(data);
  const obj = GroupCreatedType.toObject(decoded);
  const members = (obj.members as Array<Record<string, unknown>> | undefined) ?? [];
  return {
    conversationId: obj.conversationId as string,
    title: obj.title as string,
    members: members.map((m) => ({
      userId: m.userId as string,
      username: m.username as string,
      displayName: m.displayName as string,
      role: m.role as string,
    })),
  };
}

export function encodeGroupInvite(msg: GroupInviteMessage): Uint8Array {
  const message = GroupInviteType.create({
    conversationId: msg.conversationId,
    userId: msg.userId,
  });
  return GroupInviteType.encode(message).finish();
}

export function decodeGroupMemberAdded(data: Uint8Array): GroupMemberAddedMessage {
  const decoded = GroupMemberAddedType.decode(data);
  const obj = GroupMemberAddedType.toObject(decoded);
  return {
    conversationId: obj.conversationId as string,
    userId: obj.userId as string,
    addedBy: obj.addedBy as string,
  };
}

export function decodeGroupMemberRemoved(data: Uint8Array): GroupMemberRemovedMessage {
  const decoded = GroupMemberRemovedType.decode(data);
  const obj = GroupMemberRemovedType.toObject(decoded);
  return {
    conversationId: obj.conversationId as string,
    userId: obj.userId as string,
    removedBy: obj.removedBy as string,
  };
}

export function encodeGroupLeave(msg: GroupLeaveMessage): Uint8Array {
  const message = GroupLeaveType.create({ conversationId: msg.conversationId });
  return GroupLeaveType.encode(message).finish();
}

// ============================================================================
// MLS Key Management encode/decode functions
// ============================================================================

export function encodeMLSKeyPackageUpload(msg: MLSKeyPackageUploadMessage): Uint8Array {
  const message = MLSKeyPackageUploadType.create({ keyPackageData: msg.keyPackageData });
  return MLSKeyPackageUploadType.encode(message).finish();
}

export function encodeMLSKeyPackageFetch(msg: MLSKeyPackageFetchMessage): Uint8Array {
  const message = MLSKeyPackageFetchType.create({ userId: msg.userId });
  return MLSKeyPackageFetchType.encode(message).finish();
}

export function decodeMLSKeyPackageResponse(data: Uint8Array): MLSKeyPackageResponseMessage {
  const decoded = MLSKeyPackageResponseType.decode(data);
  const obj = MLSKeyPackageResponseType.toObject(decoded, { bytes: Uint8Array });
  return {
    userId: obj.userId as string,
    keyPackageData: obj.keyPackageData as Uint8Array,
  };
}

export function encodeMLSWelcome(msg: MLSWelcomeMessage): Uint8Array {
  const message = MLSWelcomeType.create({
    conversationId: msg.conversationId,
    recipientId: msg.recipientId,
    welcomeData: msg.welcomeData,
  });
  return MLSWelcomeType.encode(message).finish();
}

export function decodeMLSWelcomeReceive(data: Uint8Array): MLSWelcomeReceiveMessage {
  const decoded = MLSWelcomeReceiveType.decode(data);
  const obj = MLSWelcomeReceiveType.toObject(decoded, { bytes: Uint8Array });
  return {
    conversationId: obj.conversationId as string,
    senderId: obj.senderId as string,
    welcomeData: obj.welcomeData as Uint8Array,
  };
}

export function encodeMLSCommit(msg: MLSCommitMessage): Uint8Array {
  const message = MLSCommitType.create({
    conversationId: msg.conversationId,
    commitData: msg.commitData,
  });
  return MLSCommitType.encode(message).finish();
}

export function decodeMLSCommitBroadcast(data: Uint8Array): MLSCommitBroadcastMessage {
  const decoded = MLSCommitBroadcastType.decode(data);
  const obj = MLSCommitBroadcastType.toObject(decoded, { bytes: Uint8Array });
  return {
    conversationId: obj.conversationId as string,
    senderId: obj.senderId as string,
    commitData: obj.commitData as Uint8Array,
  };
}
