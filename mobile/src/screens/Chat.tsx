import React, { useState, useCallback, useRef, useEffect } from 'react';
import {
  View,
  Text,
  TextInput,
  FlatList,
  TouchableOpacity,
  StyleSheet,
  type ListRenderItemInfo,
} from 'react-native';
import { useMessages } from '../state/messageStore';
import { conversationService, type StoredMessage } from '../services/conversation';
import { mlsService } from '../services/mls';
import MessageBubble from '../components/MessageBubble';

interface ChatProps {
  conversationId: string;
  onBack: () => void;
}

const Chat: React.FC<ChatProps> = ({ conversationId, onBack }) => {
  const { state, actions } = useMessages();
  const [inputText, setInputText] = useState('');
  const flatListRef = useRef<FlatList<StoredMessage>>(null);

  const conversation = state.conversations.find((c) => c.id === conversationId);
  const messages = state.messages[conversationId] ?? [];
  const currentUserId = mlsService.getUserId();

  // Mark conversation as read when opened
  useEffect(() => {
    actions.markAsRead(conversationId);
    conversationService.markAsRead(conversationId);
  }, [conversationId, actions]);

  // Sync messages from conversation service into state
  useEffect(() => {
    const storedMessages = conversationService.getMessages(conversationId);
    if (storedMessages.length > 0) {
      actions.setMessages(conversationId, storedMessages);
    }
  }, [conversationId, actions]);

  // Listen for conversation events
  useEffect(() => {
    const handler = (event: { type: string; conversationId: string; message?: StoredMessage; deliveredMessageId?: string }) => {
      if (event.conversationId !== conversationId) return;

      switch (event.type) {
        case 'message_received':
          if (event.message) {
            actions.addMessage(conversationId, event.message);
            actions.markAsRead(conversationId);
            conversationService.markAsRead(conversationId);
          }
          break;
        case 'message_sent':
          if (event.message) {
            // Refresh messages from service to get updated status
            actions.setMessages(conversationId, conversationService.getMessages(conversationId));
          }
          break;
        case 'message_delivered':
          if (event.deliveredMessageId) {
            actions.updateMessageStatus(event.deliveredMessageId, 'delivered');
          }
          break;
      }
    };

    conversationService.addEventListener(handler);
    return () => {
      conversationService.removeEventListener(handler);
    };
  }, [conversationId, actions]);

  const sendMessage = useCallback(() => {
    const text = inputText.trim();
    if (!text) return;

    const msg = conversationService.sendMessage(conversationId, text);
    if (msg) {
      actions.addMessage(conversationId, msg);
      setInputText('');
    }
  }, [inputText, conversationId, actions]);

  const renderMessage = useCallback(
    ({ item }: ListRenderItemInfo<StoredMessage>) => {
      const isMine = item.senderId === currentUserId;
      return (
        <MessageBubble
          content={item.text}
          isMine={isMine}
          timestamp={item.timestamp}
          senderName={isMine ? undefined : item.senderId}
          status={isMine ? item.status : undefined}
        />
      );
    },
    [currentUserId],
  );

  const keyExtractor = useCallback((item: StoredMessage) => item.id, []);

  const title = conversation?.title ?? 'Chat';
  const hasEncryption = mlsService.hasGroup(conversationId);

  return (
    <View style={styles.container}>
      {/* Header */}
      <View style={styles.header}>
        <TouchableOpacity style={styles.backButton} onPress={onBack}>
          <Text style={styles.backText}>{'\u2190'}</Text>
        </TouchableOpacity>
        <View style={styles.headerInfo}>
          <Text style={styles.headerTitle} numberOfLines={1}>{title}</Text>
          {hasEncryption ? (
            <Text style={styles.headerSubtitle}>End-to-end encrypted</Text>
          ) : null}
        </View>
      </View>

      {/* Encryption banner for new conversations */}
      {messages.length === 0 && hasEncryption ? (
        <View style={styles.encryptionBanner}>
          <Text style={styles.encryptionText}>
            End-to-end encrypted conversation.{'\n'}
            Messages are visible only to participants.
          </Text>
        </View>
      ) : null}

      {/* Message list */}
      <FlatList
        ref={flatListRef}
        data={messages}
        renderItem={renderMessage}
        keyExtractor={keyExtractor}
        style={styles.messageList}
        contentContainerStyle={styles.messageListContent}
        onContentSizeChange={() => {
          flatListRef.current?.scrollToEnd({ animated: true });
        }}
        inverted={false}
      />

      {/* Input area */}
      <View style={styles.inputContainer}>
        <TextInput
          style={styles.textInput}
          value={inputText}
          onChangeText={setInputText}
          placeholder={hasEncryption ? 'Type a message...' : 'Encryption not ready'}
          placeholderTextColor="#999"
          editable={hasEncryption}
          onSubmitEditing={sendMessage}
          returnKeyType="send"
          multiline={false}
        />
        <TouchableOpacity
          style={[styles.sendButton, (!hasEncryption || !inputText.trim()) && styles.sendButtonDisabled]}
          onPress={sendMessage}
          disabled={!hasEncryption || !inputText.trim()}
        >
          <Text style={styles.sendButtonText}>Send</Text>
        </TouchableOpacity>
      </View>
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#F5F5F5',
  },
  header: {
    backgroundColor: '#1A1A2E',
    paddingTop: 52,
    paddingHorizontal: 12,
    paddingBottom: 12,
    flexDirection: 'row',
    alignItems: 'center',
  },
  backButton: {
    paddingHorizontal: 8,
    paddingVertical: 4,
  },
  backText: {
    color: '#FFFFFF',
    fontSize: 24,
  },
  headerInfo: {
    flex: 1,
    marginLeft: 8,
  },
  headerTitle: {
    color: '#FFFFFF',
    fontSize: 18,
    fontWeight: '600',
  },
  headerSubtitle: {
    color: '#AAAACC',
    fontSize: 12,
    marginTop: 2,
  },
  encryptionBanner: {
    backgroundColor: '#E8F5E9',
    padding: 16,
    marginHorizontal: 12,
    marginTop: 12,
    borderRadius: 8,
  },
  encryptionText: {
    color: '#2E7D32',
    fontSize: 13,
    textAlign: 'center',
    lineHeight: 18,
  },
  messageList: {
    flex: 1,
  },
  messageListContent: {
    paddingVertical: 8,
  },
  inputContainer: {
    flexDirection: 'row',
    padding: 8,
    backgroundColor: '#FFFFFF',
    borderTopWidth: 1,
    borderTopColor: '#E0E0E0',
  },
  textInput: {
    flex: 1,
    backgroundColor: '#F0F0F0',
    borderRadius: 20,
    paddingHorizontal: 16,
    paddingVertical: 8,
    fontSize: 15,
    color: '#333333',
    marginRight: 8,
    maxHeight: 100,
  },
  sendButton: {
    backgroundColor: '#1A1A2E',
    borderRadius: 20,
    paddingHorizontal: 20,
    justifyContent: 'center',
  },
  sendButtonDisabled: {
    opacity: 0.4,
  },
  sendButtonText: {
    color: '#FFFFFF',
    fontSize: 15,
    fontWeight: '600',
  },
});

export default Chat;
