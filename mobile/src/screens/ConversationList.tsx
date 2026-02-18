import React, { useState, useCallback } from 'react';
import {
  View,
  Text,
  FlatList,
  TouchableOpacity,
  TextInput,
  StyleSheet,
  type ListRenderItemInfo,
} from 'react-native';
import { useMessages } from '../state/messageStore';
import type { Conversation } from '../services/conversation';
import type { ConnectionState } from '../services/websocket';

interface ConversationListProps {
  serverId: string;
  connectionState: ConnectionState;
  onSelectConversation: (conversationId: string) => void;
  onNewConversation: (peerId: string, peerDisplayName: string) => void;
}

function formatTimestamp(timestamp: number): string {
  const now = Date.now();
  const diff = now - timestamp;
  const minutes = Math.floor(diff / 60_000);
  if (minutes < 1) return 'now';
  if (minutes < 60) return `${minutes}m`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h`;
  const days = Math.floor(hours / 24);
  return `${days}d`;
}

function truncateText(text: string, maxLength: number): string {
  if (text.length <= maxLength) return text;
  return text.slice(0, maxLength) + '...';
}

const ConversationList: React.FC<ConversationListProps> = ({
  serverId,
  connectionState,
  onSelectConversation,
  onNewConversation,
}) => {
  const { state } = useMessages();
  const [showNewChat, setShowNewChat] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [peerIdInput, setPeerIdInput] = useState('');

  const statusColor =
    connectionState === 'connected'
      ? '#4CAF50'
      : connectionState === 'connecting'
        ? '#FF9800'
        : '#F44336';

  const handleStartNewChat = useCallback(() => {
    const trimmed = peerIdInput.trim();
    if (!trimmed) return;
    onNewConversation(trimmed, trimmed);
    setShowNewChat(false);
    setPeerIdInput('');
  }, [peerIdInput, onNewConversation]);

  const renderConversation = useCallback(
    ({ item }: ListRenderItemInfo<Conversation>) => {
      const lastMsg = item.lastMessage;
      const previewText = lastMsg ? truncateText(lastMsg.text, 40) : 'No messages yet';
      const timestamp = lastMsg?.timestamp ?? item.createdAt;

      return (
        <TouchableOpacity
          style={styles.conversationRow}
          onPress={() => onSelectConversation(item.id)}
        >
          <View style={styles.avatar}>
            <Text style={styles.avatarText}>
              {item.title.charAt(0).toUpperCase()}
            </Text>
          </View>
          <View style={styles.conversationInfo}>
            <View style={styles.topRow}>
              <Text style={styles.conversationTitle} numberOfLines={1}>
                {item.title}
              </Text>
              <Text style={styles.timestamp}>{formatTimestamp(timestamp)}</Text>
            </View>
            <View style={styles.bottomRow}>
              <Text style={styles.previewText} numberOfLines={1}>
                {previewText}
              </Text>
              {item.unreadCount > 0 ? (
                <View style={styles.unreadBadge}>
                  <Text style={styles.unreadText}>{item.unreadCount}</Text>
                </View>
              ) : null}
            </View>
          </View>
        </TouchableOpacity>
      );
    },
    [onSelectConversation],
  );

  const keyExtractor = useCallback((item: Conversation) => item.id, []);

  const filteredConversations = searchQuery
    ? state.conversations.filter((c) =>
        c.title.toLowerCase().includes(searchQuery.toLowerCase()),
      )
    : state.conversations;

  return (
    <View style={styles.container}>
      {/* Header */}
      <View style={styles.header}>
        <View style={styles.headerTopRow}>
          <Text style={styles.title}>Messages</Text>
          <View style={styles.statusRow}>
            <View style={[styles.statusDot, { backgroundColor: statusColor }]} />
            <Text style={styles.statusText}>{connectionState}</Text>
          </View>
        </View>
        <Text style={styles.serverLabel}>{serverId}</Text>
      </View>

      {/* Search bar */}
      {state.conversations.length > 0 ? (
        <View style={styles.searchContainer}>
          <TextInput
            style={styles.searchInput}
            value={searchQuery}
            onChangeText={setSearchQuery}
            placeholder="Search conversations..."
            placeholderTextColor="#999"
          />
        </View>
      ) : null}

      {/* New conversation form */}
      {showNewChat ? (
        <View style={styles.newChatContainer}>
          <Text style={styles.newChatLabel}>Enter user ID:</Text>
          <View style={styles.newChatRow}>
            <TextInput
              style={styles.newChatInput}
              value={peerIdInput}
              onChangeText={setPeerIdInput}
              placeholder="User ID"
              placeholderTextColor="#999"
              autoCapitalize="none"
              autoCorrect={false}
            />
            <TouchableOpacity style={styles.newChatSend} onPress={handleStartNewChat}>
              <Text style={styles.newChatSendText}>Start</Text>
            </TouchableOpacity>
          </View>
          <TouchableOpacity onPress={() => setShowNewChat(false)}>
            <Text style={styles.cancelText}>Cancel</Text>
          </TouchableOpacity>
        </View>
      ) : null}

      {/* Conversation list */}
      {filteredConversations.length > 0 ? (
        <FlatList
          data={filteredConversations}
          renderItem={renderConversation}
          keyExtractor={keyExtractor}
          style={styles.list}
          contentContainerStyle={styles.listContent}
        />
      ) : (
        <View style={styles.emptyContainer}>
          <Text style={styles.emptyTitle}>No conversations yet.</Text>
          <Text style={styles.emptySubtitle}>
            Tap + to start a new message.
          </Text>
        </View>
      )}

      {/* New conversation FAB */}
      {!showNewChat ? (
        <TouchableOpacity
          style={styles.fab}
          onPress={() => setShowNewChat(true)}
        >
          <Text style={styles.fabText}>+</Text>
        </TouchableOpacity>
      ) : null}
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
    paddingTop: 56,
    paddingHorizontal: 20,
    paddingBottom: 16,
  },
  headerTopRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  title: {
    fontSize: 24,
    fontWeight: '700',
    color: '#FFFFFF',
  },
  statusRow: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  statusDot: {
    width: 8,
    height: 8,
    borderRadius: 4,
    marginRight: 6,
  },
  statusText: {
    color: '#AAAACC',
    fontSize: 12,
    textTransform: 'capitalize',
  },
  serverLabel: {
    color: '#AAAACC',
    fontSize: 12,
    marginTop: 4,
  },
  searchContainer: {
    paddingHorizontal: 16,
    paddingVertical: 8,
    backgroundColor: '#FFFFFF',
    borderBottomWidth: 1,
    borderBottomColor: '#E0E0E0',
  },
  searchInput: {
    backgroundColor: '#F0F0F0',
    borderRadius: 8,
    paddingHorizontal: 12,
    paddingVertical: 8,
    fontSize: 14,
    color: '#333333',
  },
  newChatContainer: {
    backgroundColor: '#FFFFFF',
    padding: 16,
    borderBottomWidth: 1,
    borderBottomColor: '#E0E0E0',
  },
  newChatLabel: {
    fontSize: 14,
    fontWeight: '600',
    color: '#333333',
    marginBottom: 8,
  },
  newChatRow: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  newChatInput: {
    flex: 1,
    backgroundColor: '#F0F0F0',
    borderRadius: 8,
    paddingHorizontal: 12,
    paddingVertical: 8,
    fontSize: 14,
    color: '#333333',
    marginRight: 8,
  },
  newChatSend: {
    backgroundColor: '#1A1A2E',
    borderRadius: 8,
    paddingHorizontal: 16,
    paddingVertical: 10,
  },
  newChatSendText: {
    color: '#FFFFFF',
    fontSize: 14,
    fontWeight: '600',
  },
  cancelText: {
    color: '#999999',
    fontSize: 13,
    marginTop: 8,
    textAlign: 'center',
  },
  list: {
    flex: 1,
  },
  listContent: {
    paddingVertical: 4,
  },
  conversationRow: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: '#FFFFFF',
    paddingVertical: 14,
    paddingHorizontal: 16,
    borderBottomWidth: 1,
    borderBottomColor: '#F0F0F0',
  },
  avatar: {
    width: 48,
    height: 48,
    borderRadius: 24,
    backgroundColor: '#1A1A2E',
    justifyContent: 'center',
    alignItems: 'center',
    marginRight: 12,
  },
  avatarText: {
    color: '#FFFFFF',
    fontSize: 20,
    fontWeight: '600',
  },
  conversationInfo: {
    flex: 1,
  },
  topRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 4,
  },
  conversationTitle: {
    fontSize: 16,
    fontWeight: '600',
    color: '#333333',
    flex: 1,
    marginRight: 8,
  },
  timestamp: {
    fontSize: 12,
    color: '#999999',
  },
  bottomRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  previewText: {
    fontSize: 14,
    color: '#666666',
    flex: 1,
    marginRight: 8,
  },
  unreadBadge: {
    backgroundColor: '#1A1A2E',
    borderRadius: 10,
    minWidth: 20,
    height: 20,
    justifyContent: 'center',
    alignItems: 'center',
    paddingHorizontal: 6,
  },
  unreadText: {
    color: '#FFFFFF',
    fontSize: 11,
    fontWeight: '700',
  },
  emptyContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: 32,
  },
  emptyTitle: {
    fontSize: 18,
    fontWeight: '600',
    color: '#333333',
    marginBottom: 8,
  },
  emptySubtitle: {
    fontSize: 14,
    color: '#999999',
    textAlign: 'center',
  },
  fab: {
    position: 'absolute',
    right: 20,
    bottom: 24,
    width: 56,
    height: 56,
    borderRadius: 28,
    backgroundColor: '#1A1A2E',
    justifyContent: 'center',
    alignItems: 'center',
    elevation: 4,
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.25,
    shadowRadius: 4,
  },
  fabText: {
    color: '#FFFFFF',
    fontSize: 28,
    fontWeight: '300',
    lineHeight: 30,
  },
});

export default ConversationList;
