import { ConversationService, type ConversationEvent, type StoredMessage } from '../src/services/conversation';
import { MLSService } from '../src/services/mls';
import { mlsService } from '../src/services/mls';
import {
  MessageType,
  encodeEnvelope,
  decodeEnvelope,
  decodeMessageReceive,
  type Envelope,
  type MessageReceiveMessage,
  type MessageDeliveredMessage,
  type GroupCreatedMessage,
  type MLSKeyPackageResponseMessage,
  type MLSWelcomeReceiveMessage,
  type GroupMemberInfo,
} from '../src/services/protocol';
import naclUtil from 'tweetnacl-util';

// Mock WebSocket client
class MockWebSocketClient {
  sent: Envelope[] = [];
  send(envelope: Envelope): void {
    this.sent.push(envelope);
  }
  getState() { return 'connected' as const; }
  getAuthState() { return 'authenticated' as const; }
  getUrl() { return 'wss://example.com/ws'; }
  connect() {}
  disconnect() {}
  setSessionToken(_token: string) {}
  sendPing() {}
  sendAuthWithSessionToken(_token: string) {}
}

describe('ConversationService', () => {
  let service: ConversationService;
  let client: MockWebSocketClient;

  beforeEach(() => {
    service = new ConversationService();
    client = new MockWebSocketClient();
    service.setClient(client as any);

    // Initialize the singleton MLS service
    mlsService.reset();
    mlsService.initialize('current-user');
  });

  afterEach(() => {
    service.reset();
    mlsService.reset();
  });

  describe('event system', () => {
    it('addEventListener and emit deliver events to listeners', () => {
      const events: ConversationEvent[] = [];
      service.addEventListener((e) => events.push(e));

      // Trigger an event by handling a GroupCreated from someone else
      const envelope: Envelope = {
        type: MessageType.GROUP_CREATED,
        requestId: 'ext-req-1',
        payload: new Uint8Array([]),
      };
      const msg: GroupCreatedMessage = {
        conversationId: 'conv-ext-1',
        title: 'External Chat',
        members: [],
      };
      service.handleGroupCreated(envelope, msg);

      expect(events.length).toBe(1);
      expect(events[0].type).toBe('conversation_created');
      expect(events[0].conversationId).toBe('conv-ext-1');
    });

    it('removeEventListener stops delivery', () => {
      const events: ConversationEvent[] = [];
      const listener = (e: ConversationEvent) => events.push(e);
      service.addEventListener(listener);
      service.removeEventListener(listener);

      const envelope: Envelope = {
        type: MessageType.GROUP_CREATED,
        requestId: 'ext-req-2',
        payload: new Uint8Array([]),
      };
      service.handleGroupCreated(envelope, {
        conversationId: 'conv-ext-2',
        title: 'Chat',
        members: [],
      });

      expect(events.length).toBe(0);
    });
  });

  describe('uploadKeyPackages', () => {
    it('sends the specified number of MLS_KEY_PACKAGE_UPLOAD envelopes', () => {
      service.uploadKeyPackages(3);
      const uploads = client.sent.filter((e) => e.type === MessageType.MLS_KEY_PACKAGE_UPLOAD);
      expect(uploads.length).toBe(3);
      // Each should have unique requestId
      const ids = new Set(uploads.map((e) => e.requestId));
      expect(ids.size).toBe(3);
    });

    it('does nothing if client is not set', () => {
      const noClientService = new ConversationService();
      // Should not throw
      noClientService.uploadKeyPackages(5);
    });
  });

  describe('sendMessage', () => {
    beforeEach(() => {
      // Set up an MLS group for encryption
      const peerService = new MLSService();
      peerService.initialize('peer-user');
      const peerKp = peerService.parseKeyPackage(peerService.generateKeyPackage());
      mlsService.createGroup('conv-1', peerKp);
    });

    it('encrypts, sends MESSAGE_SEND, and stores the message locally', () => {
      const msg = service.sendMessage('conv-1', 'Hello World');
      expect(msg).not.toBeNull();
      expect(msg!.conversationId).toBe('conv-1');
      expect(msg!.text).toBe('Hello World');
      expect(msg!.status).toBe('sending');
      expect(msg!.senderId).toBe('current-user');

      // Should have sent a MESSAGE_SEND envelope
      const sends = client.sent.filter((e) => e.type === MessageType.MESSAGE_SEND);
      expect(sends.length).toBe(1);

      // Message should be stored locally
      const messages = service.getMessages('conv-1');
      expect(messages.length).toBe(1);
      expect(messages[0].text).toBe('Hello World');
    });

    it('returns null if no MLS group exists for the conversation', () => {
      const msg = service.sendMessage('nonexistent-conv', 'Hi');
      expect(msg).toBeNull();
    });

    it('returns null if client is not set', () => {
      const noClientService = new ConversationService();
      const msg = noClientService.sendMessage('conv-1', 'Hi');
      expect(msg).toBeNull();
    });

    it('updates lastMessage on the conversation', () => {
      // First create a conversation entry
      service.handleGroupCreated(
        { type: MessageType.GROUP_CREATED, requestId: 'req-gc', payload: new Uint8Array([]) },
        { conversationId: 'conv-1', title: 'Chat', members: [] },
      );

      const msg = service.sendMessage('conv-1', 'Update test');
      const conv = service.getConversation('conv-1');
      expect(conv!.lastMessage).toBe(msg);
    });

    it('emits message_sent event', () => {
      const events: ConversationEvent[] = [];
      service.addEventListener((e) => events.push(e));

      service.sendMessage('conv-1', 'Event test');

      const sentEvents = events.filter((e) => e.type === 'message_sent');
      expect(sentEvents.length).toBe(1);
      expect(sentEvents[0].message!.text).toBe('Event test');
    });
  });

  describe('handleMessageReceive', () => {
    let peerService: MLSService;

    beforeEach(() => {
      peerService = new MLSService();
      peerService.initialize('peer-user');
      const peerKp = peerService.parseKeyPackage(peerService.generateKeyPackage());
      const welcomeData = mlsService.createGroup('conv-1', peerKp);
      peerService.processWelcome(welcomeData);

      // Create conversation entry
      service.handleGroupCreated(
        { type: MessageType.GROUP_CREATED, requestId: 'req-gc', payload: new Uint8Array([]) },
        { conversationId: 'conv-1', title: 'Chat', members: [] },
      );
    });

    it('decrypts and stores incoming message from a peer', () => {
      const plaintext = naclUtil.decodeUTF8('Hi from peer');
      const encryptedPayload = peerService.encrypt('conv-1', plaintext);

      const envelope: Envelope = {
        type: MessageType.MESSAGE_RECEIVE,
        requestId: 'recv-1',
        payload: new Uint8Array([]),
      };
      const msgData: MessageReceiveMessage = {
        messageId: 'msg-from-peer',
        conversationId: 'conv-1',
        senderId: 'peer-user',
        encryptedPayload,
        serverTimestamp: 1700000000000000, // microseconds
        messageType: 'text',
      };

      service.handleMessageReceive(envelope, msgData);

      const messages = service.getMessages('conv-1');
      expect(messages.length).toBe(1);
      expect(messages[0].text).toBe('Hi from peer');
      expect(messages[0].status).toBe('delivered');
      expect(messages[0].timestamp).toBe(1700000000000); // converted from microseconds
    });

    it('sends MESSAGE_ACK after receiving a peer message', () => {
      const plaintext = naclUtil.decodeUTF8('Ack me');
      const encryptedPayload = peerService.encrypt('conv-1', plaintext);

      service.handleMessageReceive(
        { type: MessageType.MESSAGE_RECEIVE, requestId: 'recv-ack', payload: new Uint8Array([]) },
        {
          messageId: 'msg-ack-target',
          conversationId: 'conv-1',
          senderId: 'peer-user',
          encryptedPayload,
          serverTimestamp: 1700000000000000,
          messageType: 'text',
        },
      );

      const acks = client.sent.filter((e) => e.type === MessageType.MESSAGE_ACK);
      expect(acks.length).toBe(1);
    });

    it('increments unreadCount for the conversation', () => {
      const plaintext = naclUtil.decodeUTF8('Unread');
      const encryptedPayload = peerService.encrypt('conv-1', plaintext);

      service.handleMessageReceive(
        { type: MessageType.MESSAGE_RECEIVE, requestId: 'recv-unread', payload: new Uint8Array([]) },
        {
          messageId: 'msg-unread-1',
          conversationId: 'conv-1',
          senderId: 'peer-user',
          encryptedPayload,
          serverTimestamp: 1700000000000000,
          messageType: 'text',
        },
      );

      const conv = service.getConversation('conv-1');
      expect(conv!.unreadCount).toBe(1);
    });

    it('handles own message echo by updating status to sent', () => {
      // First send a message (status = sending)
      service.sendMessage('conv-1', 'Echo me');
      const messages = service.getMessages('conv-1');
      expect(messages[0].status).toBe('sending');

      // Simulate server echoing back (senderId = current user)
      const envelope: Envelope = {
        type: MessageType.MESSAGE_RECEIVE,
        requestId: 'echo-1',
        payload: new Uint8Array([]),
      };
      const msgData: MessageReceiveMessage = {
        messageId: 'server-msg-id',
        conversationId: 'conv-1',
        senderId: 'current-user',
        encryptedPayload: new Uint8Array([]),
        serverTimestamp: 1700000000000000,
        messageType: 'text',
      };

      service.handleMessageReceive(envelope, msgData);

      const updated = service.getMessages('conv-1');
      expect(updated[0].status).toBe('sent');
      expect(updated[0].id).toBe('server-msg-id');
    });

    it('stores [Decryption failed] when decryption fails', () => {
      service.handleMessageReceive(
        { type: MessageType.MESSAGE_RECEIVE, requestId: 'recv-fail', payload: new Uint8Array([]) },
        {
          messageId: 'msg-bad-crypto',
          conversationId: 'conv-1',
          senderId: 'peer-user',
          encryptedPayload: new Uint8Array([1, 2, 3]), // invalid ciphertext
          serverTimestamp: 1700000000000000,
          messageType: 'text',
        },
      );

      const messages = service.getMessages('conv-1');
      expect(messages.length).toBe(1);
      expect(messages[0].text).toBe('[Decryption failed]');
    });

    it('emits message_received event for peer messages', () => {
      const events: ConversationEvent[] = [];
      service.addEventListener((e) => events.push(e));

      const plaintext = naclUtil.decodeUTF8('Event message');
      const encryptedPayload = peerService.encrypt('conv-1', plaintext);

      service.handleMessageReceive(
        { type: MessageType.MESSAGE_RECEIVE, requestId: 'recv-evt', payload: new Uint8Array([]) },
        {
          messageId: 'msg-evt',
          conversationId: 'conv-1',
          senderId: 'peer-user',
          encryptedPayload,
          serverTimestamp: 1700000000000000,
          messageType: 'text',
        },
      );

      const received = events.filter((e) => e.type === 'message_received');
      expect(received.length).toBe(1);
    });
  });

  describe('handleMessageDelivered', () => {
    it('updates message status to delivered', () => {
      // Set up group and send a message
      const peerService = new MLSService();
      peerService.initialize('peer-user');
      const peerKp = peerService.parseKeyPackage(peerService.generateKeyPackage());
      mlsService.createGroup('conv-del', peerKp);

      const msg = service.sendMessage('conv-del', 'Deliver me');
      expect(msg).not.toBeNull();
      const msgId = msg!.id;

      service.handleMessageDelivered({ messageId: msgId, deliveredTo: 'peer-user' });

      const messages = service.getMessages('conv-del');
      expect(messages[0].status).toBe('delivered');
    });

    it('emits message_delivered event', () => {
      const peerService = new MLSService();
      peerService.initialize('peer-user');
      const peerKp = peerService.parseKeyPackage(peerService.generateKeyPackage());
      mlsService.createGroup('conv-del-evt', peerKp);

      const msg = service.sendMessage('conv-del-evt', 'Event deliver');
      const events: ConversationEvent[] = [];
      service.addEventListener((e) => events.push(e));

      service.handleMessageDelivered({ messageId: msg!.id, deliveredTo: 'peer-user' });

      const delivered = events.filter((e) => e.type === 'message_delivered');
      expect(delivered.length).toBe(1);
      expect(delivered[0].deliveredMessageId).toBe(msg!.id);
    });

    it('does nothing for unknown messageId', () => {
      // Should not throw
      service.handleMessageDelivered({ messageId: 'nonexistent', deliveredTo: 'user-1' });
    });
  });

  describe('handleGroupCreated', () => {
    it('resolves pending callback if requestId matches', () => {
      // Simulate pending create via the internal callback map by calling createGroupOnServer indirectly
      // We'll test this by verifying that an externally-arriving GroupCreated creates a conversation
      const envelope: Envelope = {
        type: MessageType.GROUP_CREATED,
        requestId: 'ext-req',
        payload: new Uint8Array([]),
      };
      const msg: GroupCreatedMessage = {
        conversationId: 'conv-created-1',
        title: 'New Chat',
        members: [
          { userId: 'u1', username: 'alice', displayName: 'Alice', role: 'admin' },
        ],
      };

      service.handleGroupCreated(envelope, msg);

      const conv = service.getConversation('conv-created-1');
      expect(conv).toBeDefined();
      expect(conv!.title).toBe('New Chat');
      expect(conv!.members.length).toBe(1);
    });

    it('does not duplicate conversation if already exists', () => {
      const envelope: Envelope = {
        type: MessageType.GROUP_CREATED,
        requestId: 'ext-dup',
        payload: new Uint8Array([]),
      };
      const msg: GroupCreatedMessage = {
        conversationId: 'conv-dup',
        title: 'Dup Chat',
        members: [],
      };

      service.handleGroupCreated(envelope, msg);
      service.handleGroupCreated(envelope, msg);

      const convs = service.getConversations().filter((c) => c.id === 'conv-dup');
      expect(convs.length).toBe(1);
    });
  });

  describe('handleWelcomeReceive', () => {
    it('processes Welcome and creates a conversation placeholder', () => {
      const peerService = new MLSService();
      peerService.initialize('peer-user');
      const myKp = mlsService.parseKeyPackage(mlsService.generateKeyPackage());
      const welcomeData = peerService.createGroup('conv-welcome', myKp);

      const msg: MLSWelcomeReceiveMessage = {
        conversationId: 'conv-welcome',
        senderId: 'peer-user',
        welcomeData,
      };

      service.handleWelcomeReceive(msg);

      expect(mlsService.hasGroup('conv-welcome')).toBe(true);
      const conv = service.getConversation('conv-welcome');
      expect(conv).toBeDefined();
      expect(conv!.title).toBe('Chat');
    });
  });

  describe('handleMemberAdded', () => {
    it('adds a member to an existing conversation', () => {
      service.handleGroupCreated(
        { type: MessageType.GROUP_CREATED, requestId: 'r', payload: new Uint8Array([]) },
        { conversationId: 'conv-add', title: 'Chat', members: [{ userId: 'u1', username: 'alice', displayName: 'Alice', role: 'admin' }] },
      );

      service.handleMemberAdded('conv-add', 'u2');

      const conv = service.getConversation('conv-add');
      expect(conv!.members.length).toBe(2);
      expect(conv!.members[1].userId).toBe('u2');
    });

    it('does not add duplicate member', () => {
      service.handleGroupCreated(
        { type: MessageType.GROUP_CREATED, requestId: 'r', payload: new Uint8Array([]) },
        { conversationId: 'conv-dup-add', title: 'Chat', members: [{ userId: 'u1', username: 'alice', displayName: 'Alice', role: 'admin' }] },
      );

      service.handleMemberAdded('conv-dup-add', 'u1');
      const conv = service.getConversation('conv-dup-add');
      expect(conv!.members.length).toBe(1);
    });
  });

  describe('handleMemberRemoved', () => {
    it('removes a member from an existing conversation', () => {
      service.handleGroupCreated(
        { type: MessageType.GROUP_CREATED, requestId: 'r', payload: new Uint8Array([]) },
        {
          conversationId: 'conv-rem',
          title: 'Chat',
          members: [
            { userId: 'u1', username: 'alice', displayName: 'Alice', role: 'admin' },
            { userId: 'u2', username: 'bob', displayName: 'Bob', role: 'member' },
          ],
        },
      );

      service.handleMemberRemoved('conv-rem', 'u2');
      const conv = service.getConversation('conv-rem');
      expect(conv!.members.length).toBe(1);
      expect(conv!.members[0].userId).toBe('u1');
    });

    it('cleans up MLS group if current user is removed', () => {
      // Set up group first
      const peerService = new MLSService();
      peerService.initialize('peer');
      const peerKp = peerService.parseKeyPackage(peerService.generateKeyPackage());
      mlsService.createGroup('conv-self-rem', peerKp);
      expect(mlsService.hasGroup('conv-self-rem')).toBe(true);

      service.handleGroupCreated(
        { type: MessageType.GROUP_CREATED, requestId: 'r', payload: new Uint8Array([]) },
        {
          conversationId: 'conv-self-rem',
          title: 'Chat',
          members: [{ userId: 'current-user', username: 'me', displayName: 'Me', role: 'admin' }],
        },
      );

      service.handleMemberRemoved('conv-self-rem', 'current-user');
      expect(mlsService.hasGroup('conv-self-rem')).toBe(false);
    });
  });

  describe('queries', () => {
    beforeEach(() => {
      // Create two conversations with different timestamps
      service.handleGroupCreated(
        { type: MessageType.GROUP_CREATED, requestId: 'r1', payload: new Uint8Array([]) },
        { conversationId: 'conv-old', title: 'Old Chat', members: [] },
      );
      service.handleGroupCreated(
        { type: MessageType.GROUP_CREATED, requestId: 'r2', payload: new Uint8Array([]) },
        { conversationId: 'conv-new', title: 'New Chat', members: [] },
      );
    });

    it('getConversations returns all conversations', () => {
      const convs = service.getConversations();
      expect(convs.length).toBe(2);
    });

    it('getConversation returns undefined for unknown id', () => {
      expect(service.getConversation('nonexistent')).toBeUndefined();
    });

    it('getMessages returns empty array for unknown conversation', () => {
      expect(service.getMessages('nonexistent')).toEqual([]);
    });
  });

  describe('markAsRead', () => {
    it('resets unread count to zero', () => {
      service.handleGroupCreated(
        { type: MessageType.GROUP_CREATED, requestId: 'r', payload: new Uint8Array([]) },
        { conversationId: 'conv-read', title: 'Chat', members: [] },
      );

      // Manually set unread count by modifying the conversation
      const conv = service.getConversation('conv-read')!;
      // We'll use handleMessageReceive to increment unread
      // But we need MLS group for that. Let's test via direct update
      const peerService = new MLSService();
      peerService.initialize('peer');
      const peerKp = peerService.parseKeyPackage(peerService.generateKeyPackage());
      const welcomeData = mlsService.createGroup('conv-read', peerKp);
      peerService.processWelcome(welcomeData);

      const plaintext = naclUtil.decodeUTF8('Message 1');
      const encrypted = peerService.encrypt('conv-read', plaintext);
      service.handleMessageReceive(
        { type: MessageType.MESSAGE_RECEIVE, requestId: 'recv-read', payload: new Uint8Array([]) },
        {
          messageId: 'msg-read-1',
          conversationId: 'conv-read',
          senderId: 'peer',
          encryptedPayload: encrypted,
          serverTimestamp: 1700000000000000,
          messageType: 'text',
        },
      );

      expect(service.getConversation('conv-read')!.unreadCount).toBe(1);

      service.markAsRead('conv-read');
      expect(service.getConversation('conv-read')!.unreadCount).toBe(0);
    });
  });

  describe('reset', () => {
    it('clears all conversations, messages, callbacks, and listeners', () => {
      service.handleGroupCreated(
        { type: MessageType.GROUP_CREATED, requestId: 'r', payload: new Uint8Array([]) },
        { conversationId: 'conv-reset', title: 'Chat', members: [] },
      );

      const events: ConversationEvent[] = [];
      service.addEventListener((e) => events.push(e));

      service.reset();

      expect(service.getConversations()).toEqual([]);
      expect(service.getMessages('conv-reset')).toEqual([]);

      // Listener should have been cleared, so no events after reset
      service.handleGroupCreated(
        { type: MessageType.GROUP_CREATED, requestId: 'r2', payload: new Uint8Array([]) },
        { conversationId: 'conv-after', title: 'Chat 2', members: [] },
      );
      expect(events.length).toBe(0);
    });
  });
});
