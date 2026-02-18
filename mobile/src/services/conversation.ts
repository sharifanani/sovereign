// Conversation service
// Manages 1:1 encrypted conversations: creation, message send/receive, and local state.

import naclUtil from 'tweetnacl-util';
import { mlsService } from './mls';
import { type WebSocketClient } from './websocket';
import {
  MessageType,
  generateRequestId,
  encodeGroupCreate,
  encodeMessageSend,
  encodeMessageAck,
  encodeMLSKeyPackageFetch,
  encodeMLSKeyPackageUpload,
  encodeMLSWelcome,
  type Envelope,
  type MessageReceiveMessage,
  type MessageDeliveredMessage,
  type GroupCreatedMessage,
  type GroupMemberInfo,
  type MLSKeyPackageResponseMessage,
  type MLSWelcomeReceiveMessage,
} from './protocol';

// ============================================================================
// Types
// ============================================================================

export type DeliveryStatus = 'sending' | 'sent' | 'delivered';

export interface StoredMessage {
  id: string;
  conversationId: string;
  senderId: string;
  text: string;
  timestamp: number;
  status: DeliveryStatus;
  messageType: string;
}

export interface Conversation {
  id: string;
  title: string;
  members: GroupMemberInfo[];
  lastMessage: StoredMessage | null;
  unreadCount: number;
  createdAt: number;
}

export type ConversationEventType =
  | 'conversation_created'
  | 'message_received'
  | 'message_sent'
  | 'message_delivered'
  | 'conversations_updated';

export interface ConversationEvent {
  type: ConversationEventType;
  conversationId: string;
  message?: StoredMessage;
  deliveredMessageId?: string;
}

export type ConversationEventListener = (event: ConversationEvent) => void;

// ============================================================================
// Conversation Service
// ============================================================================

export class ConversationService {
  private conversations: Map<string, Conversation> = new Map();
  private messages: Map<string, StoredMessage[]> = new Map();
  private client: WebSocketClient | null = null;
  private listeners: Set<ConversationEventListener> = new Set();
  private pendingKeyPackageCallbacks: Map<string, (response: MLSKeyPackageResponseMessage) => void> = new Map();
  private pendingGroupCreateCallbacks: Map<string, (response: GroupCreatedMessage) => void> = new Map();

  setClient(client: WebSocketClient): void {
    this.client = client;
  }

  // ============================================================================
  // Event system
  // ============================================================================

  addEventListener(listener: ConversationEventListener): void {
    this.listeners.add(listener);
  }

  removeEventListener(listener: ConversationEventListener): void {
    this.listeners.delete(listener);
  }

  private emit(event: ConversationEvent): void {
    for (const listener of this.listeners) {
      listener(event);
    }
  }

  // ============================================================================
  // Key package management
  // ============================================================================

  uploadKeyPackages(count: number): void {
    if (!this.client) return;
    for (let i = 0; i < count; i++) {
      const keyPackageData = mlsService.generateKeyPackage();
      const payload = encodeMLSKeyPackageUpload({ keyPackageData });
      this.client.send({
        type: MessageType.MLS_KEY_PACKAGE_UPLOAD,
        requestId: generateRequestId(),
        payload,
      });
    }
  }

  // ============================================================================
  // Conversation creation
  // ============================================================================

  async createDirectConversation(
    peerId: string,
    peerDisplayName: string,
  ): Promise<Conversation> {
    if (!this.client) {
      throw new globalThis.Error('WebSocket client not set');
    }

    // Check for existing 1:1 with this peer
    for (const conv of this.conversations.values()) {
      if (conv.members.length === 2 && conv.members.some((m) => m.userId === peerId)) {
        return conv;
      }
    }

    // Fetch peer's key package
    const keyPackageResponse = await this.fetchKeyPackage(peerId);
    const peerKeyPackage = mlsService.parseKeyPackage(keyPackageResponse.keyPackageData);

    // Create group on server (1:1 = group with 2 members)
    const groupCreated = await this.createGroupOnServer(peerDisplayName, [peerId]);

    // Create MLS group and generate Welcome
    const welcomeData = mlsService.createGroup(groupCreated.conversationId, peerKeyPackage);

    // Send Welcome to peer
    const welcomePayload = encodeMLSWelcome({
      conversationId: groupCreated.conversationId,
      recipientId: peerId,
      welcomeData,
    });
    this.client.send({
      type: MessageType.MLS_WELCOME,
      requestId: generateRequestId(),
      payload: welcomePayload,
    });

    // Store conversation locally
    const conversation: Conversation = {
      id: groupCreated.conversationId,
      title: peerDisplayName,
      members: groupCreated.members,
      lastMessage: null,
      unreadCount: 0,
      createdAt: Date.now(),
    };
    this.conversations.set(conversation.id, conversation);
    this.messages.set(conversation.id, []);

    this.emit({ type: 'conversation_created', conversationId: conversation.id });
    return conversation;
  }

  private fetchKeyPackage(userId: string): Promise<MLSKeyPackageResponseMessage> {
    return new Promise((resolve, reject) => {
      if (!this.client) {
        reject(new globalThis.Error('WebSocket client not set'));
        return;
      }
      const requestId = generateRequestId();
      this.pendingKeyPackageCallbacks.set(requestId, resolve);
      const payload = encodeMLSKeyPackageFetch({ userId });
      this.client.send({
        type: MessageType.MLS_KEY_PACKAGE_FETCH,
        requestId,
        payload,
      });
      // Timeout after 10 seconds
      setTimeout(() => {
        if (this.pendingKeyPackageCallbacks.has(requestId)) {
          this.pendingKeyPackageCallbacks.delete(requestId);
          reject(new globalThis.Error('Key package fetch timed out'));
        }
      }, 10_000);
    });
  }

  private createGroupOnServer(title: string, memberIds: string[]): Promise<GroupCreatedMessage> {
    return new Promise((resolve, reject) => {
      if (!this.client) {
        reject(new globalThis.Error('WebSocket client not set'));
        return;
      }
      const requestId = generateRequestId();
      this.pendingGroupCreateCallbacks.set(requestId, resolve);
      const payload = encodeGroupCreate({ title, memberIds });
      this.client.send({
        type: MessageType.GROUP_CREATE,
        requestId,
        payload,
      });
      setTimeout(() => {
        if (this.pendingGroupCreateCallbacks.has(requestId)) {
          this.pendingGroupCreateCallbacks.delete(requestId);
          reject(new globalThis.Error('Group creation timed out'));
        }
      }, 10_000);
    });
  }

  // ============================================================================
  // Sending messages
  // ============================================================================

  sendMessage(conversationId: string, text: string): StoredMessage | null {
    if (!this.client || !mlsService.hasGroup(conversationId)) {
      return null;
    }

    const plaintext = naclUtil.decodeUTF8(text);
    const encryptedPayload = mlsService.encrypt(conversationId, plaintext);

    const requestId = generateRequestId();
    const payload = encodeMessageSend({
      conversationId,
      encryptedPayload,
      messageType: 'text',
    });
    this.client.send({
      type: MessageType.MESSAGE_SEND,
      requestId,
      payload,
    });

    const msg: StoredMessage = {
      id: requestId,
      conversationId,
      senderId: mlsService.getUserId(),
      text,
      timestamp: Date.now(),
      status: 'sending',
      messageType: 'text',
    };

    const convMessages = this.messages.get(conversationId) ?? [];
    convMessages.push(msg);
    this.messages.set(conversationId, convMessages);

    const conversation = this.conversations.get(conversationId);
    if (conversation) {
      conversation.lastMessage = msg;
    }

    this.emit({ type: 'message_sent', conversationId, message: msg });
    return msg;
  }

  // ============================================================================
  // Handling incoming messages from WebSocket
  // ============================================================================

  handleMessageReceive(envelope: Envelope, msgData: MessageReceiveMessage): void {
    const { conversationId, messageId, senderId, serverTimestamp, messageType } = msgData;

    // If this is our own message echoed back, update status to 'sent'
    if (senderId === mlsService.getUserId()) {
      const convMessages = this.messages.get(conversationId);
      if (convMessages) {
        // Find the pending message by request ID matching the envelope
        const pending = convMessages.find((m) => m.status === 'sending' && m.conversationId === conversationId);
        if (pending) {
          pending.id = messageId;
          pending.status = 'sent';
          pending.timestamp = serverTimestamp / 1000; // Convert microseconds to milliseconds
          const conversation = this.conversations.get(conversationId);
          if (conversation) {
            conversation.lastMessage = pending;
          }
          this.emit({ type: 'message_sent', conversationId, message: pending });
        }
      }
      // ACK our own message
      this.sendAck(messageId);
      return;
    }

    // Decrypt incoming message
    let text: string;
    try {
      const plaintext = mlsService.decrypt(conversationId, msgData.encryptedPayload);
      text = naclUtil.encodeUTF8(plaintext);
    } catch {
      text = '[Decryption failed]';
    }

    const msg: StoredMessage = {
      id: messageId,
      conversationId,
      senderId,
      text,
      timestamp: serverTimestamp / 1000,
      status: 'delivered',
      messageType,
    };

    const convMessages = this.messages.get(conversationId) ?? [];
    convMessages.push(msg);
    this.messages.set(conversationId, convMessages);

    const conversation = this.conversations.get(conversationId);
    if (conversation) {
      conversation.lastMessage = msg;
      conversation.unreadCount++;
    }

    // ACK the message
    this.sendAck(messageId);

    this.emit({ type: 'message_received', conversationId, message: msg });
  }

  handleMessageDelivered(msg: MessageDeliveredMessage): void {
    // Update the message status to 'delivered'
    for (const [convId, convMessages] of this.messages.entries()) {
      const found = convMessages.find((m) => m.id === msg.messageId);
      if (found) {
        found.status = 'delivered';
        this.emit({
          type: 'message_delivered',
          conversationId: convId,
          deliveredMessageId: msg.messageId,
        });
        break;
      }
    }
  }

  handleGroupCreated(envelope: Envelope, msg: GroupCreatedMessage): void {
    // Check if this is a response to a pending create
    const callback = this.pendingGroupCreateCallbacks.get(envelope.requestId);
    if (callback) {
      this.pendingGroupCreateCallbacks.delete(envelope.requestId);
      callback(msg);
      return;
    }

    // Otherwise it's a group we were added to by someone else
    if (!this.conversations.has(msg.conversationId)) {
      const conversation: Conversation = {
        id: msg.conversationId,
        title: msg.title,
        members: msg.members,
        lastMessage: null,
        unreadCount: 0,
        createdAt: Date.now(),
      };
      this.conversations.set(msg.conversationId, conversation);
      this.messages.set(msg.conversationId, []);
      this.emit({ type: 'conversation_created', conversationId: msg.conversationId });
    }
  }

  handleKeyPackageResponse(envelope: Envelope, msg: MLSKeyPackageResponseMessage): void {
    const callback = this.pendingKeyPackageCallbacks.get(envelope.requestId);
    if (callback) {
      this.pendingKeyPackageCallbacks.delete(envelope.requestId);
      callback(msg);
    }
  }

  handleWelcomeReceive(msg: MLSWelcomeReceiveMessage): void {
    // Process the Welcome to join the MLS group
    const conversationId = mlsService.processWelcome(msg.welcomeData);

    // If we don't have the conversation yet, create a placeholder
    if (!this.conversations.has(conversationId)) {
      const conversation: Conversation = {
        id: conversationId,
        title: `Chat`, // Will be updated when we get group info
        members: [],
        lastMessage: null,
        unreadCount: 0,
        createdAt: Date.now(),
      };
      this.conversations.set(conversationId, conversation);
      this.messages.set(conversationId, []);
      this.emit({ type: 'conversation_created', conversationId });
    }
  }

  handleCommitBroadcast(conversationId: string, commitData: Uint8Array): void {
    mlsService.processCommit(conversationId, commitData);
  }

  handleMemberAdded(conversationId: string, userId: string): void {
    const conversation = this.conversations.get(conversationId);
    if (conversation && !conversation.members.some((m) => m.userId === userId)) {
      conversation.members.push({
        userId,
        username: '',
        displayName: '',
        role: 'member',
      });
      this.emit({ type: 'conversations_updated', conversationId });
    }
  }

  handleMemberRemoved(conversationId: string, userId: string): void {
    const conversation = this.conversations.get(conversationId);
    if (conversation) {
      conversation.members = conversation.members.filter((m) => m.userId !== userId);
      // If we were removed, clean up
      if (userId === mlsService.getUserId()) {
        mlsService.removeGroup(conversationId);
      }
      this.emit({ type: 'conversations_updated', conversationId });
    }
  }

  // ============================================================================
  // Queries
  // ============================================================================

  getConversations(): Conversation[] {
    return Array.from(this.conversations.values()).sort((a, b) => {
      const aTime = a.lastMessage?.timestamp ?? a.createdAt;
      const bTime = b.lastMessage?.timestamp ?? b.createdAt;
      return bTime - aTime;
    });
  }

  getConversation(conversationId: string): Conversation | undefined {
    return this.conversations.get(conversationId);
  }

  getMessages(conversationId: string): StoredMessage[] {
    return this.messages.get(conversationId) ?? [];
  }

  markAsRead(conversationId: string): void {
    const conversation = this.conversations.get(conversationId);
    if (conversation) {
      conversation.unreadCount = 0;
      this.emit({ type: 'conversations_updated', conversationId });
    }
  }

  // ============================================================================
  // Helpers
  // ============================================================================

  private sendAck(messageId: string): void {
    if (!this.client) return;
    const payload = encodeMessageAck({ messageId });
    this.client.send({
      type: MessageType.MESSAGE_ACK,
      requestId: generateRequestId(),
      payload,
    });
  }

  reset(): void {
    this.conversations.clear();
    this.messages.clear();
    this.pendingKeyPackageCallbacks.clear();
    this.pendingGroupCreateCallbacks.clear();
    this.listeners.clear();
  }
}

// Singleton instance
export const conversationService = new ConversationService();
