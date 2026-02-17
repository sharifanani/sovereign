import React from 'react';

interface MessageBubbleProps {
  content: string;
  isMine: boolean;
  timestamp: number;
  senderName?: string;
}

const MessageBubble: React.FC<MessageBubbleProps> = ({ content, isMine, timestamp, senderName }) => {
  // TODO: Render message bubble with alignment, timestamp, sender name
  return null;
};

export default MessageBubble;
