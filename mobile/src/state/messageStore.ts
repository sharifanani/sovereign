// Message and conversation state management
// React Context-based store for conversations and messages

import React, { useCallback, useContext, useMemo, useReducer } from 'react';
import type { Conversation, StoredMessage, DeliveryStatus } from '../services/conversation';

// ============================================================================
// State
// ============================================================================

export interface MessageState {
  conversations: Conversation[];
  currentConversationId: string | null;
  messages: Record<string, StoredMessage[]>;
}

const INITIAL_MESSAGE_STATE: MessageState = {
  conversations: [],
  currentConversationId: null,
  messages: {},
};

// ============================================================================
// Actions
// ============================================================================

type MessageAction =
  | { type: 'SET_CONVERSATIONS'; conversations: Conversation[] }
  | { type: 'ADD_CONVERSATION'; conversation: Conversation }
  | { type: 'UPDATE_CONVERSATION'; conversationId: string; updates: Partial<Conversation> }
  | { type: 'SET_CURRENT_CONVERSATION'; conversationId: string | null }
  | { type: 'SET_MESSAGES'; conversationId: string; messages: StoredMessage[] }
  | { type: 'ADD_MESSAGE'; conversationId: string; message: StoredMessage }
  | { type: 'UPDATE_MESSAGE_STATUS'; messageId: string; status: DeliveryStatus }
  | { type: 'UPDATE_MESSAGE'; messageId: string; updates: Partial<StoredMessage> }
  | { type: 'MARK_AS_READ'; conversationId: string }
  | { type: 'RESET' };

function messageReducer(state: MessageState, action: MessageAction): MessageState {
  switch (action.type) {
    case 'SET_CONVERSATIONS':
      return { ...state, conversations: action.conversations };

    case 'ADD_CONVERSATION': {
      const exists = state.conversations.some((c) => c.id === action.conversation.id);
      if (exists) return state;
      return { ...state, conversations: [action.conversation, ...state.conversations] };
    }

    case 'UPDATE_CONVERSATION':
      return {
        ...state,
        conversations: state.conversations.map((c) =>
          c.id === action.conversationId ? { ...c, ...action.updates } : c,
        ),
      };

    case 'SET_CURRENT_CONVERSATION':
      return { ...state, currentConversationId: action.conversationId };

    case 'SET_MESSAGES':
      return {
        ...state,
        messages: { ...state.messages, [action.conversationId]: action.messages },
      };

    case 'ADD_MESSAGE': {
      const existing = state.messages[action.conversationId] ?? [];
      // Avoid duplicates
      if (existing.some((m) => m.id === action.message.id)) {
        return state;
      }
      return {
        ...state,
        messages: {
          ...state.messages,
          [action.conversationId]: [...existing, action.message],
        },
      };
    }

    case 'UPDATE_MESSAGE_STATUS': {
      const newMessages: Record<string, StoredMessage[]> = {};
      let found = false;
      for (const [convId, msgs] of Object.entries(state.messages)) {
        newMessages[convId] = msgs.map((m) => {
          if (m.id === action.messageId) {
            found = true;
            return { ...m, status: action.status };
          }
          return m;
        });
      }
      return found ? { ...state, messages: newMessages } : state;
    }

    case 'UPDATE_MESSAGE': {
      const newMessages: Record<string, StoredMessage[]> = {};
      let found = false;
      for (const [convId, msgs] of Object.entries(state.messages)) {
        newMessages[convId] = msgs.map((m) => {
          if (m.id === action.messageId) {
            found = true;
            return { ...m, ...action.updates };
          }
          return m;
        });
      }
      return found ? { ...state, messages: newMessages } : state;
    }

    case 'MARK_AS_READ':
      return {
        ...state,
        conversations: state.conversations.map((c) =>
          c.id === action.conversationId ? { ...c, unreadCount: 0 } : c,
        ),
      };

    case 'RESET':
      return { ...INITIAL_MESSAGE_STATE };
  }
}

// ============================================================================
// Context and Hooks
// ============================================================================

export interface MessageActions {
  setConversations: (conversations: Conversation[]) => void;
  addConversation: (conversation: Conversation) => void;
  updateConversation: (conversationId: string, updates: Partial<Conversation>) => void;
  setCurrentConversation: (conversationId: string | null) => void;
  setMessages: (conversationId: string, messages: StoredMessage[]) => void;
  addMessage: (conversationId: string, message: StoredMessage) => void;
  updateMessageStatus: (messageId: string, status: DeliveryStatus) => void;
  updateMessage: (messageId: string, updates: Partial<StoredMessage>) => void;
  markAsRead: (conversationId: string) => void;
  reset: () => void;
}

interface MessageContextValue {
  state: MessageState;
  actions: MessageActions;
}

export const MessageContext = React.createContext<MessageContextValue | null>(null);

export function useMessageReducer(): MessageContextValue {
  const [state, dispatch] = useReducer(messageReducer, INITIAL_MESSAGE_STATE);

  const setConversations = useCallback(
    (conversations: Conversation[]) => dispatch({ type: 'SET_CONVERSATIONS', conversations }),
    [],
  );
  const addConversation = useCallback(
    (conversation: Conversation) => dispatch({ type: 'ADD_CONVERSATION', conversation }),
    [],
  );
  const updateConversation = useCallback(
    (conversationId: string, updates: Partial<Conversation>) =>
      dispatch({ type: 'UPDATE_CONVERSATION', conversationId, updates }),
    [],
  );
  const setCurrentConversation = useCallback(
    (conversationId: string | null) => dispatch({ type: 'SET_CURRENT_CONVERSATION', conversationId }),
    [],
  );
  const setMessages = useCallback(
    (conversationId: string, messages: StoredMessage[]) =>
      dispatch({ type: 'SET_MESSAGES', conversationId, messages }),
    [],
  );
  const addMessage = useCallback(
    (conversationId: string, message: StoredMessage) =>
      dispatch({ type: 'ADD_MESSAGE', conversationId, message }),
    [],
  );
  const updateMessageStatus = useCallback(
    (messageId: string, status: DeliveryStatus) =>
      dispatch({ type: 'UPDATE_MESSAGE_STATUS', messageId, status }),
    [],
  );
  const updateMessage = useCallback(
    (messageId: string, updates: Partial<StoredMessage>) =>
      dispatch({ type: 'UPDATE_MESSAGE', messageId, updates }),
    [],
  );
  const markAsRead = useCallback(
    (conversationId: string) => dispatch({ type: 'MARK_AS_READ', conversationId }),
    [],
  );
  const reset = useCallback(() => dispatch({ type: 'RESET' }), []);

  const actions = useMemo(
    () => ({
      setConversations,
      addConversation,
      updateConversation,
      setCurrentConversation,
      setMessages,
      addMessage,
      updateMessageStatus,
      updateMessage,
      markAsRead,
      reset,
    }),
    [
      setConversations,
      addConversation,
      updateConversation,
      setCurrentConversation,
      setMessages,
      addMessage,
      updateMessageStatus,
      updateMessage,
      markAsRead,
      reset,
    ],
  );

  return useMemo(() => ({ state, actions }), [state, actions]);
}

export function useMessages(): MessageContextValue {
  const ctx = useContext(MessageContext);
  if (!ctx) {
    throw new globalThis.Error('useMessages must be used within a MessageContext.Provider');
  }
  return ctx;
}
