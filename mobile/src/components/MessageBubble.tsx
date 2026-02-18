import React from 'react';
import { View, Text, StyleSheet } from 'react-native';
import type { DeliveryStatus } from '../services/conversation';

interface MessageBubbleProps {
  content: string;
  isMine: boolean;
  timestamp: number;
  senderName?: string;
  status?: DeliveryStatus;
}

function formatTime(timestamp: number): string {
  const date = new Date(timestamp);
  const hours = date.getHours();
  const minutes = date.getMinutes();
  const ampm = hours >= 12 ? 'p' : 'a';
  const displayHours = hours % 12 || 12;
  const displayMinutes = minutes.toString().padStart(2, '0');
  return `${displayHours}:${displayMinutes}${ampm}`;
}

function statusIndicator(status: DeliveryStatus | undefined): string {
  switch (status) {
    case 'sending':
      return ' ...';
    case 'sent':
      return ' \u25CB'; // open circle
    case 'delivered':
      return ' \u25CF'; // filled circle
    default:
      return '';
  }
}

const MessageBubble: React.FC<MessageBubbleProps> = ({
  content,
  isMine,
  timestamp,
  senderName,
  status,
}) => {
  return (
    <View style={[styles.container, isMine ? styles.sentContainer : styles.receivedContainer]}>
      {!isMine && senderName ? (
        <Text style={styles.senderName}>{senderName}</Text>
      ) : null}
      <View style={[styles.bubble, isMine ? styles.sentBubble : styles.receivedBubble]}>
        <Text style={[styles.messageText, isMine ? styles.sentText : styles.receivedText]}>
          {content}
        </Text>
        <Text style={[styles.metaText, isMine ? styles.sentMeta : styles.receivedMeta]}>
          {formatTime(timestamp)}
          {isMine ? statusIndicator(status) : ''}
        </Text>
      </View>
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    marginBottom: 6,
    paddingHorizontal: 12,
  },
  sentContainer: {
    alignItems: 'flex-end',
  },
  receivedContainer: {
    alignItems: 'flex-start',
  },
  bubble: {
    maxWidth: '80%',
    padding: 10,
    borderRadius: 12,
  },
  sentBubble: {
    backgroundColor: '#1A1A2E',
    borderBottomRightRadius: 4,
  },
  receivedBubble: {
    backgroundColor: '#FFFFFF',
    borderWidth: 1,
    borderColor: '#E0E0E0',
    borderBottomLeftRadius: 4,
  },
  senderName: {
    fontSize: 12,
    fontWeight: '600',
    color: '#666666',
    marginBottom: 2,
    marginLeft: 4,
  },
  messageText: {
    fontSize: 15,
    lineHeight: 20,
  },
  sentText: {
    color: '#FFFFFF',
  },
  receivedText: {
    color: '#333333',
  },
  metaText: {
    fontSize: 11,
    marginTop: 4,
    alignSelf: 'flex-end',
  },
  sentMeta: {
    color: '#AAAACC',
  },
  receivedMeta: {
    color: '#999999',
  },
});

export default MessageBubble;
