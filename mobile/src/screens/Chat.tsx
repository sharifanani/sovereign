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
import { WebSocketClient, type ConnectionState } from '../services/websocket';
import {
  type Envelope,
  MessageType,
  encodePing,
  decodePing,
  generateRequestId,
  messageTypeName,
} from '../services/protocol';

interface ChatMessage {
  id: string;
  text: string;
  direction: 'sent' | 'received';
  timestamp: number;
}

const DEFAULT_SERVER_URL = 'ws://localhost:8080/ws';

const Chat: React.FC = () => {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [inputText, setInputText] = useState('');
  const [connectionState, setConnectionState] = useState<ConnectionState>('disconnected');
  const [serverUrl, setServerUrl] = useState(DEFAULT_SERVER_URL);
  const [editingUrl, setEditingUrl] = useState(false);
  const [urlDraft, setUrlDraft] = useState(DEFAULT_SERVER_URL);
  const clientRef = useRef<WebSocketClient | null>(null);
  const flatListRef = useRef<FlatList<ChatMessage>>(null);

  const handleMessage = useCallback((envelope: Envelope) => {
    // For Phase A echo testing: display whatever comes back
    let displayText: string;

    if (envelope.type === MessageType.PING && envelope.payload.length > 0) {
      const ping = decodePing(envelope.payload);
      displayText = `[PING] timestamp=${ping.timestamp}`;
    } else if (envelope.type === MessageType.ERROR) {
      // Just show raw info for errors
      displayText = `[ERROR] payload=${envelope.payload.length} bytes`;
    } else {
      displayText = `[${messageTypeName(envelope.type)}] reqId=${envelope.requestId} payload=${envelope.payload.length}b`;
    }

    const msg: ChatMessage = {
      id: `recv-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
      text: displayText,
      direction: 'received',
      timestamp: Date.now(),
    };
    setMessages((prev) => [...prev, msg]);
  }, []);

  const handleStateChange = useCallback((state: ConnectionState) => {
    setConnectionState(state);
  }, []);

  const connectToServer = useCallback(() => {
    if (clientRef.current) {
      clientRef.current.disconnect();
    }
    const client = new WebSocketClient({
      url: serverUrl,
      onMessage: handleMessage,
      onStateChange: handleStateChange,
    });
    clientRef.current = client;
    client.connect();
  }, [serverUrl, handleMessage, handleStateChange]);

  const disconnectFromServer = useCallback(() => {
    clientRef.current?.disconnect();
    clientRef.current = null;
  }, []);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      clientRef.current?.disconnect();
    };
  }, []);

  const sendMessage = useCallback(() => {
    const text = inputText.trim();
    if (!text || connectionState !== 'connected') {
      return;
    }

    // Wrap text in a Ping envelope for Phase A echo testing
    const timestamp = Date.now() * 1000; // Unix microseconds
    const payload = encodePing({ timestamp });
    const requestId = generateRequestId();

    const envelope: Envelope = {
      type: MessageType.PING,
      requestId,
      payload,
    };

    clientRef.current?.send(envelope);

    const msg: ChatMessage = {
      id: `sent-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
      text: `[PING] ${text} (ts=${timestamp})`,
      direction: 'sent',
      timestamp: Date.now(),
    };
    setMessages((prev) => [...prev, msg]);
    setInputText('');
  }, [inputText, connectionState]);

  const renderMessage = useCallback(({ item }: ListRenderItemInfo<ChatMessage>) => {
    const isSent = item.direction === 'sent';
    return (
      <View style={[styles.messageBubble, isSent ? styles.sentBubble : styles.receivedBubble]}>
        <Text style={[styles.messageText, isSent ? styles.sentText : styles.receivedText]}>
          {item.text}
        </Text>
        <Text style={styles.timestampText}>
          {new Date(item.timestamp).toLocaleTimeString()}
        </Text>
      </View>
    );
  }, []);

  const keyExtractor = useCallback((item: ChatMessage) => item.id, []);

  const statusColor =
    connectionState === 'connected'
      ? '#4CAF50'
      : connectionState === 'connecting'
        ? '#FF9800'
        : '#F44336';

  return (
    <View style={styles.container}>
      {/* Header with connection status */}
      <View style={styles.header}>
        <View style={styles.statusRow}>
          <View style={[styles.statusDot, { backgroundColor: statusColor }]} />
          <Text style={styles.statusText}>{connectionState.toUpperCase()}</Text>
        </View>
        {editingUrl ? (
          <View style={styles.urlEditRow}>
            <TextInput
              style={styles.urlInput}
              value={urlDraft}
              onChangeText={setUrlDraft}
              autoCapitalize="none"
              autoCorrect={false}
              placeholder="ws://host:port/ws"
            />
            <TouchableOpacity
              style={styles.urlButton}
              onPress={() => {
                setServerUrl(urlDraft);
                setEditingUrl(false);
              }}
            >
              <Text style={styles.urlButtonText}>Save</Text>
            </TouchableOpacity>
          </View>
        ) : (
          <TouchableOpacity onPress={() => setEditingUrl(true)}>
            <Text style={styles.serverUrlText}>{serverUrl}</Text>
          </TouchableOpacity>
        )}
        <View style={styles.controlRow}>
          <TouchableOpacity
            style={[styles.controlButton, connectionState === 'connected' && styles.controlButtonActive]}
            onPress={connectionState === 'disconnected' ? connectToServer : disconnectFromServer}
          >
            <Text style={styles.controlButtonText}>
              {connectionState === 'disconnected' ? 'Connect' : 'Disconnect'}
            </Text>
          </TouchableOpacity>
        </View>
      </View>

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
      />

      {/* Input area */}
      <View style={styles.inputContainer}>
        <TextInput
          style={styles.textInput}
          value={inputText}
          onChangeText={setInputText}
          placeholder={connectionState === 'connected' ? 'Type a message...' : 'Connect to send'}
          placeholderTextColor="#999"
          editable={connectionState === 'connected'}
          onSubmitEditing={sendMessage}
          returnKeyType="send"
        />
        <TouchableOpacity
          style={[styles.sendButton, connectionState !== 'connected' && styles.sendButtonDisabled]}
          onPress={sendMessage}
          disabled={connectionState !== 'connected'}
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
    paddingTop: 48,
    paddingHorizontal: 16,
    paddingBottom: 12,
  },
  statusRow: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: 4,
  },
  statusDot: {
    width: 10,
    height: 10,
    borderRadius: 5,
    marginRight: 8,
  },
  statusText: {
    color: '#FFFFFF',
    fontSize: 13,
    fontWeight: '600',
    letterSpacing: 1,
  },
  serverUrlText: {
    color: '#AAAACC',
    fontSize: 12,
    marginBottom: 8,
  },
  urlEditRow: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: 8,
  },
  urlInput: {
    flex: 1,
    backgroundColor: '#2A2A4E',
    color: '#FFFFFF',
    paddingHorizontal: 8,
    paddingVertical: 4,
    borderRadius: 4,
    fontSize: 12,
    marginRight: 8,
  },
  urlButton: {
    backgroundColor: '#4CAF50',
    paddingHorizontal: 12,
    paddingVertical: 6,
    borderRadius: 4,
  },
  urlButtonText: {
    color: '#FFFFFF',
    fontSize: 12,
    fontWeight: '600',
  },
  controlRow: {
    flexDirection: 'row',
  },
  controlButton: {
    backgroundColor: '#4CAF50',
    paddingHorizontal: 16,
    paddingVertical: 8,
    borderRadius: 6,
  },
  controlButtonActive: {
    backgroundColor: '#F44336',
  },
  controlButtonText: {
    color: '#FFFFFF',
    fontSize: 14,
    fontWeight: '600',
  },
  messageList: {
    flex: 1,
  },
  messageListContent: {
    padding: 12,
  },
  messageBubble: {
    maxWidth: '80%',
    padding: 10,
    borderRadius: 12,
    marginBottom: 8,
  },
  sentBubble: {
    alignSelf: 'flex-end',
    backgroundColor: '#1A1A2E',
  },
  receivedBubble: {
    alignSelf: 'flex-start',
    backgroundColor: '#FFFFFF',
    borderWidth: 1,
    borderColor: '#E0E0E0',
  },
  messageText: {
    fontSize: 14,
  },
  sentText: {
    color: '#FFFFFF',
  },
  receivedText: {
    color: '#333333',
  },
  timestampText: {
    fontSize: 10,
    color: '#999999',
    marginTop: 4,
    alignSelf: 'flex-end',
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
    fontSize: 14,
    color: '#333333',
    marginRight: 8,
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
    fontSize: 14,
    fontWeight: '600',
  },
});

export default Chat;
