import * as protobuf from 'protobufjs';
import {
  // Messaging
  encodeMessageSend,
  decodeMessageReceive,
  encodeMessageAck,
  decodeMessageDelivered,
  // Group
  encodeGroupCreate,
  decodeGroupCreated,
  encodeGroupInvite,
  decodeGroupMemberAdded,
  decodeGroupMemberRemoved,
  encodeGroupLeave,
  // MLS
  encodeMLSKeyPackageUpload,
  encodeMLSKeyPackageFetch,
  decodeMLSKeyPackageResponse,
  encodeMLSWelcome,
  decodeMLSWelcomeReceive,
  encodeMLSCommit,
  decodeMLSCommitBroadcast,
  // Types
  type MessageSendMessage,
  type MessageReceiveMessage,
  type MessageAckMessage,
  type MessageDeliveredMessage,
  type GroupCreateMessage,
  type GroupCreatedMessage,
  type GroupInviteMessage,
  type GroupMemberAddedMessage,
  type GroupMemberRemovedMessage,
  type GroupLeaveMessage,
  type MLSKeyPackageUploadMessage,
  type MLSKeyPackageFetchMessage,
  type MLSKeyPackageResponseMessage,
  type MLSWelcomeMessage,
  type MLSWelcomeReceiveMessage,
  type MLSCommitMessage,
  type MLSCommitBroadcastMessage,
} from '../src/services/protocol';

// Helper: build protobuf types for server-side encoding (types the client only decodes)
function buildServerTypes(): Record<string, protobuf.Type> {
  const root = new protobuf.Root();

  const MessageReceiveProto = new protobuf.Type('MessageReceive')
    .add(new protobuf.Field('messageId', 1, 'string'))
    .add(new protobuf.Field('conversationId', 2, 'string'))
    .add(new protobuf.Field('senderId', 3, 'string'))
    .add(new protobuf.Field('encryptedPayload', 4, 'bytes'))
    .add(new protobuf.Field('serverTimestamp', 5, 'int64'))
    .add(new protobuf.Field('messageType', 6, 'string'));
  root.add(MessageReceiveProto);

  const MessageDeliveredProto = new protobuf.Type('MessageDelivered')
    .add(new protobuf.Field('messageId', 1, 'string'))
    .add(new protobuf.Field('deliveredTo', 2, 'string'));
  root.add(MessageDeliveredProto);

  const GroupMemberProto = new protobuf.Type('GroupMember')
    .add(new protobuf.Field('userId', 1, 'string'))
    .add(new protobuf.Field('username', 2, 'string'))
    .add(new protobuf.Field('displayName', 3, 'string'))
    .add(new protobuf.Field('role', 4, 'string'));
  root.add(GroupMemberProto);

  const GroupCreatedProto = new protobuf.Type('GroupCreated')
    .add(new protobuf.Field('conversationId', 1, 'string'))
    .add(new protobuf.Field('title', 2, 'string'))
    .add(new protobuf.Field('members', 3, 'GroupMember', 'repeated'));
  root.add(GroupCreatedProto);

  const GroupMemberAddedProto = new protobuf.Type('GroupMemberAdded')
    .add(new protobuf.Field('conversationId', 1, 'string'))
    .add(new protobuf.Field('userId', 2, 'string'))
    .add(new protobuf.Field('addedBy', 3, 'string'));
  root.add(GroupMemberAddedProto);

  const GroupMemberRemovedProto = new protobuf.Type('GroupMemberRemoved')
    .add(new protobuf.Field('conversationId', 1, 'string'))
    .add(new protobuf.Field('userId', 2, 'string'))
    .add(new protobuf.Field('removedBy', 3, 'string'));
  root.add(GroupMemberRemovedProto);

  const MLSKeyPackageResponseProto = new protobuf.Type('MLSKeyPackageResponse')
    .add(new protobuf.Field('userId', 1, 'string'))
    .add(new protobuf.Field('keyPackageData', 2, 'bytes'));
  root.add(MLSKeyPackageResponseProto);

  const MLSWelcomeReceiveProto = new protobuf.Type('MLSWelcomeReceive')
    .add(new protobuf.Field('conversationId', 1, 'string'))
    .add(new protobuf.Field('senderId', 2, 'string'))
    .add(new protobuf.Field('welcomeData', 3, 'bytes'));
  root.add(MLSWelcomeReceiveProto);

  const MLSCommitBroadcastProto = new protobuf.Type('MLSCommitBroadcast')
    .add(new protobuf.Field('conversationId', 1, 'string'))
    .add(new protobuf.Field('senderId', 2, 'string'))
    .add(new protobuf.Field('commitData', 3, 'bytes'));
  root.add(MLSCommitBroadcastProto);

  // Client-only encode types — we decode them on the "server side" to verify encoding
  const MessageSendProto = new protobuf.Type('MessageSend')
    .add(new protobuf.Field('conversationId', 1, 'string'))
    .add(new protobuf.Field('encryptedPayload', 2, 'bytes'))
    .add(new protobuf.Field('messageType', 3, 'string'));
  root.add(MessageSendProto);

  const MessageAckProto = new protobuf.Type('MessageAck')
    .add(new protobuf.Field('messageId', 1, 'string'));
  root.add(MessageAckProto);

  const GroupCreateProto = new protobuf.Type('GroupCreate')
    .add(new protobuf.Field('title', 1, 'string'))
    .add(new protobuf.Field('memberIds', 2, 'string', 'repeated'));
  root.add(GroupCreateProto);

  const GroupInviteProto = new protobuf.Type('GroupInvite')
    .add(new protobuf.Field('conversationId', 1, 'string'))
    .add(new protobuf.Field('userId', 2, 'string'));
  root.add(GroupInviteProto);

  const GroupLeaveProto = new protobuf.Type('GroupLeave')
    .add(new protobuf.Field('conversationId', 1, 'string'));
  root.add(GroupLeaveProto);

  const MLSKeyPackageUploadProto = new protobuf.Type('MLSKeyPackageUpload')
    .add(new protobuf.Field('keyPackageData', 1, 'bytes'));
  root.add(MLSKeyPackageUploadProto);

  const MLSKeyPackageFetchProto = new protobuf.Type('MLSKeyPackageFetch')
    .add(new protobuf.Field('userId', 1, 'string'));
  root.add(MLSKeyPackageFetchProto);

  const MLSWelcomeProto = new protobuf.Type('MLSWelcome')
    .add(new protobuf.Field('conversationId', 1, 'string'))
    .add(new protobuf.Field('recipientId', 2, 'string'))
    .add(new protobuf.Field('welcomeData', 3, 'bytes'));
  root.add(MLSWelcomeProto);

  const MLSCommitProto = new protobuf.Type('MLSCommit')
    .add(new protobuf.Field('conversationId', 1, 'string'))
    .add(new protobuf.Field('commitData', 2, 'bytes'));
  root.add(MLSCommitProto);

  root.resolveAll();
  return {
    MessageReceive: MessageReceiveProto,
    MessageDelivered: MessageDeliveredProto,
    GroupCreated: GroupCreatedProto,
    GroupMemberAdded: GroupMemberAddedProto,
    GroupMemberRemoved: GroupMemberRemovedProto,
    MLSKeyPackageResponse: MLSKeyPackageResponseProto,
    MLSWelcomeReceive: MLSWelcomeReceiveProto,
    MLSCommitBroadcast: MLSCommitBroadcastProto,
    MessageSend: MessageSendProto,
    MessageAck: MessageAckProto,
    GroupCreate: GroupCreateProto,
    GroupInvite: GroupInviteProto,
    GroupLeave: GroupLeaveProto,
    MLSKeyPackageUpload: MLSKeyPackageUploadProto,
    MLSKeyPackageFetch: MLSKeyPackageFetchProto,
    MLSWelcome: MLSWelcomeProto,
    MLSCommit: MLSCommitProto,
  };
}

function encodeServerMessage(types: Record<string, protobuf.Type>, typeName: string, obj: Record<string, unknown>): Uint8Array {
  const t = types[typeName];
  const msg = t.create(obj);
  return new Uint8Array(t.encode(msg).finish());
}

describe('Phase C protocol codec — messaging types', () => {
  const serverTypes = buildServerTypes();

  // ============================================================================
  // MessageSend (client encode)
  // ============================================================================

  describe('MessageSend encode', () => {
    it('encodes a MessageSend payload that the server can decode', () => {
      const msg: MessageSendMessage = {
        conversationId: 'conv-123',
        encryptedPayload: new Uint8Array([10, 20, 30]),
        messageType: 'text',
      };
      const encoded = encodeMessageSend(msg);
      expect(encoded).toBeInstanceOf(Uint8Array);
      expect(encoded.length).toBeGreaterThan(0);

      // Verify by decoding with protobufjs
      const decoded = serverTypes.MessageSend.decode(encoded);
      const obj = serverTypes.MessageSend.toObject(decoded, { bytes: Uint8Array });
      expect(obj.conversationId).toBe('conv-123');
      expect(new Uint8Array(obj.encryptedPayload as Uint8Array)).toEqual(new Uint8Array([10, 20, 30]));
      expect(obj.messageType).toBe('text');
    });

    it('handles empty encrypted payload', () => {
      const msg: MessageSendMessage = {
        conversationId: 'conv-empty',
        encryptedPayload: new Uint8Array([]),
        messageType: 'text',
      };
      const encoded = encodeMessageSend(msg);
      const decoded = serverTypes.MessageSend.decode(encoded);
      const obj = serverTypes.MessageSend.toObject(decoded, { bytes: Uint8Array });
      expect(obj.conversationId).toBe('conv-empty');
    });
  });

  // ============================================================================
  // MessageReceive (client decode)
  // ============================================================================

  describe('MessageReceive decode', () => {
    it('decodes all fields correctly', () => {
      const payload = new Uint8Array([99, 100, 101]);
      const encoded = encodeServerMessage(serverTypes, 'MessageReceive', {
        messageId: 'msg-001',
        conversationId: 'conv-123',
        senderId: 'user-42',
        encryptedPayload: payload,
        serverTimestamp: 1700000000000,
        messageType: 'text',
      });

      const decoded = decodeMessageReceive(encoded);
      expect(decoded.messageId).toBe('msg-001');
      expect(decoded.conversationId).toBe('conv-123');
      expect(decoded.senderId).toBe('user-42');
      expect(new Uint8Array(decoded.encryptedPayload)).toEqual(payload);
      expect(decoded.serverTimestamp).toBe(1700000000000);
      expect(decoded.messageType).toBe('text');
    });

    it('handles missing optional-like fields with defaults', () => {
      const encoded = encodeServerMessage(serverTypes, 'MessageReceive', {
        messageId: 'msg-002',
        conversationId: 'conv-123',
        senderId: 'user-1',
        encryptedPayload: new Uint8Array([]),
        serverTimestamp: 0,
        messageType: '',
      });
      const decoded = decodeMessageReceive(encoded);
      expect(decoded.serverTimestamp).toBe(0);
      expect(decoded.messageType).toBe('');
    });
  });

  // ============================================================================
  // MessageAck (client encode)
  // ============================================================================

  describe('MessageAck encode', () => {
    it('encodes a MessageAck that the server can decode', () => {
      const encoded = encodeMessageAck({ messageId: 'msg-ack-1' });
      const decoded = serverTypes.MessageAck.decode(encoded);
      const obj = serverTypes.MessageAck.toObject(decoded);
      expect(obj.messageId).toBe('msg-ack-1');
    });
  });

  // ============================================================================
  // MessageDelivered (client decode)
  // ============================================================================

  describe('MessageDelivered decode', () => {
    it('decodes messageId and deliveredTo', () => {
      const encoded = encodeServerMessage(serverTypes, 'MessageDelivered', {
        messageId: 'msg-del-1',
        deliveredTo: 'user-99',
      });
      const decoded = decodeMessageDelivered(encoded);
      expect(decoded.messageId).toBe('msg-del-1');
      expect(decoded.deliveredTo).toBe('user-99');
    });
  });

  // ============================================================================
  // GroupCreate (client encode)
  // ============================================================================

  describe('GroupCreate encode', () => {
    it('encodes title and memberIds', () => {
      const msg: GroupCreateMessage = {
        title: 'Test Group',
        memberIds: ['user-1', 'user-2', 'user-3'],
      };
      const encoded = encodeGroupCreate(msg);
      const decoded = serverTypes.GroupCreate.decode(encoded);
      const obj = serverTypes.GroupCreate.toObject(decoded);
      expect(obj.title).toBe('Test Group');
      expect(obj.memberIds).toEqual(['user-1', 'user-2', 'user-3']);
    });

    it('handles empty memberIds', () => {
      const encoded = encodeGroupCreate({ title: 'Solo Group', memberIds: [] });
      const decoded = serverTypes.GroupCreate.decode(encoded);
      const obj = serverTypes.GroupCreate.toObject(decoded);
      expect(obj.title).toBe('Solo Group');
      expect(obj.memberIds ?? []).toEqual([]);
    });
  });

  // ============================================================================
  // GroupCreated (client decode)
  // ============================================================================

  describe('GroupCreated decode', () => {
    it('decodes conversationId, title, and members array', () => {
      const encoded = encodeServerMessage(serverTypes, 'GroupCreated', {
        conversationId: 'conv-group-1',
        title: 'Friends',
        members: [
          { userId: 'u1', username: 'alice', displayName: 'Alice', role: 'admin' },
          { userId: 'u2', username: 'bob', displayName: 'Bob', role: 'member' },
        ],
      });
      const decoded = decodeGroupCreated(encoded);
      expect(decoded.conversationId).toBe('conv-group-1');
      expect(decoded.title).toBe('Friends');
      expect(decoded.members).toHaveLength(2);
      expect(decoded.members[0].userId).toBe('u1');
      expect(decoded.members[0].role).toBe('admin');
      expect(decoded.members[1].userId).toBe('u2');
      expect(decoded.members[1].displayName).toBe('Bob');
    });

    it('handles empty members list', () => {
      const encoded = encodeServerMessage(serverTypes, 'GroupCreated', {
        conversationId: 'conv-empty',
        title: 'Empty',
        members: [],
      });
      const decoded = decodeGroupCreated(encoded);
      expect(decoded.members).toEqual([]);
    });
  });

  // ============================================================================
  // GroupInvite (client encode)
  // ============================================================================

  describe('GroupInvite encode', () => {
    it('encodes conversationId and userId', () => {
      const encoded = encodeGroupInvite({ conversationId: 'conv-1', userId: 'user-invite' });
      const decoded = serverTypes.GroupInvite.decode(encoded);
      const obj = serverTypes.GroupInvite.toObject(decoded);
      expect(obj.conversationId).toBe('conv-1');
      expect(obj.userId).toBe('user-invite');
    });
  });

  // ============================================================================
  // GroupMemberAdded (client decode)
  // ============================================================================

  describe('GroupMemberAdded decode', () => {
    it('decodes conversationId, userId, and addedBy', () => {
      const encoded = encodeServerMessage(serverTypes, 'GroupMemberAdded', {
        conversationId: 'conv-1',
        userId: 'new-user',
        addedBy: 'admin-user',
      });
      const decoded = decodeGroupMemberAdded(encoded);
      expect(decoded.conversationId).toBe('conv-1');
      expect(decoded.userId).toBe('new-user');
      expect(decoded.addedBy).toBe('admin-user');
    });
  });

  // ============================================================================
  // GroupMemberRemoved (client decode)
  // ============================================================================

  describe('GroupMemberRemoved decode', () => {
    it('decodes conversationId, userId, and removedBy', () => {
      const encoded = encodeServerMessage(serverTypes, 'GroupMemberRemoved', {
        conversationId: 'conv-1',
        userId: 'removed-user',
        removedBy: 'admin-user',
      });
      const decoded = decodeGroupMemberRemoved(encoded);
      expect(decoded.conversationId).toBe('conv-1');
      expect(decoded.userId).toBe('removed-user');
      expect(decoded.removedBy).toBe('admin-user');
    });
  });

  // ============================================================================
  // GroupLeave (client encode)
  // ============================================================================

  describe('GroupLeave encode', () => {
    it('encodes conversationId', () => {
      const encoded = encodeGroupLeave({ conversationId: 'conv-leaving' });
      const decoded = serverTypes.GroupLeave.decode(encoded);
      const obj = serverTypes.GroupLeave.toObject(decoded);
      expect(obj.conversationId).toBe('conv-leaving');
    });
  });

  // ============================================================================
  // MLSKeyPackageUpload (client encode)
  // ============================================================================

  describe('MLSKeyPackageUpload encode', () => {
    it('encodes keyPackageData bytes', () => {
      const kpData = new Uint8Array([1, 2, 3, 4, 5]);
      const encoded = encodeMLSKeyPackageUpload({ keyPackageData: kpData });
      const decoded = serverTypes.MLSKeyPackageUpload.decode(encoded);
      const obj = serverTypes.MLSKeyPackageUpload.toObject(decoded, { bytes: Uint8Array });
      expect(new Uint8Array(obj.keyPackageData as Uint8Array)).toEqual(kpData);
    });
  });

  // ============================================================================
  // MLSKeyPackageFetch (client encode)
  // ============================================================================

  describe('MLSKeyPackageFetch encode', () => {
    it('encodes userId', () => {
      const encoded = encodeMLSKeyPackageFetch({ userId: 'target-user' });
      const decoded = serverTypes.MLSKeyPackageFetch.decode(encoded);
      const obj = serverTypes.MLSKeyPackageFetch.toObject(decoded);
      expect(obj.userId).toBe('target-user');
    });
  });

  // ============================================================================
  // MLSKeyPackageResponse (client decode)
  // ============================================================================

  describe('MLSKeyPackageResponse decode', () => {
    it('decodes userId and keyPackageData', () => {
      const kpData = new Uint8Array([10, 20, 30]);
      const encoded = encodeServerMessage(serverTypes, 'MLSKeyPackageResponse', {
        userId: 'target-user',
        keyPackageData: kpData,
      });
      const decoded = decodeMLSKeyPackageResponse(encoded);
      expect(decoded.userId).toBe('target-user');
      expect(new Uint8Array(decoded.keyPackageData)).toEqual(kpData);
    });
  });

  // ============================================================================
  // MLSWelcome (client encode)
  // ============================================================================

  describe('MLSWelcome encode', () => {
    it('encodes conversationId, recipientId, and welcomeData', () => {
      const welcomeData = new Uint8Array([7, 8, 9]);
      const encoded = encodeMLSWelcome({
        conversationId: 'conv-welcome',
        recipientId: 'peer-1',
        welcomeData,
      });
      const decoded = serverTypes.MLSWelcome.decode(encoded);
      const obj = serverTypes.MLSWelcome.toObject(decoded, { bytes: Uint8Array });
      expect(obj.conversationId).toBe('conv-welcome');
      expect(obj.recipientId).toBe('peer-1');
      expect(new Uint8Array(obj.welcomeData as Uint8Array)).toEqual(welcomeData);
    });
  });

  // ============================================================================
  // MLSWelcomeReceive (client decode)
  // ============================================================================

  describe('MLSWelcomeReceive decode', () => {
    it('decodes conversationId, senderId, and welcomeData', () => {
      const welcomeData = new Uint8Array([11, 22, 33]);
      const encoded = encodeServerMessage(serverTypes, 'MLSWelcomeReceive', {
        conversationId: 'conv-welcome-rx',
        senderId: 'sender-1',
        welcomeData,
      });
      const decoded = decodeMLSWelcomeReceive(encoded);
      expect(decoded.conversationId).toBe('conv-welcome-rx');
      expect(decoded.senderId).toBe('sender-1');
      expect(new Uint8Array(decoded.welcomeData)).toEqual(welcomeData);
    });
  });

  // ============================================================================
  // MLSCommit (client encode)
  // ============================================================================

  describe('MLSCommit encode', () => {
    it('encodes conversationId and commitData', () => {
      const commitData = new Uint8Array([44, 55, 66]);
      const encoded = encodeMLSCommit({
        conversationId: 'conv-commit',
        commitData,
      });
      const decoded = serverTypes.MLSCommit.decode(encoded);
      const obj = serverTypes.MLSCommit.toObject(decoded, { bytes: Uint8Array });
      expect(obj.conversationId).toBe('conv-commit');
      expect(new Uint8Array(obj.commitData as Uint8Array)).toEqual(commitData);
    });
  });

  // ============================================================================
  // MLSCommitBroadcast (client decode)
  // ============================================================================

  describe('MLSCommitBroadcast decode', () => {
    it('decodes conversationId, senderId, and commitData', () => {
      const commitData = new Uint8Array([77, 88, 99]);
      const encoded = encodeServerMessage(serverTypes, 'MLSCommitBroadcast', {
        conversationId: 'conv-commit-bc',
        senderId: 'sender-commit',
        commitData,
      });
      const decoded = decodeMLSCommitBroadcast(encoded);
      expect(decoded.conversationId).toBe('conv-commit-bc');
      expect(decoded.senderId).toBe('sender-commit');
      expect(new Uint8Array(decoded.commitData)).toEqual(commitData);
    });
  });
});
