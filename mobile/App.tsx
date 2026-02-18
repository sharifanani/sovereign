import React, { useState, useCallback, useRef, useEffect } from 'react';
import {
  View,
  Text,
  TextInput,
  TouchableOpacity,
  StyleSheet,
  StatusBar,
  SafeAreaView,
  KeyboardAvoidingView,
  Platform,
} from 'react-native';
import { registerRootComponent } from 'expo';

import { AuthContext, useAuthReducer } from './src/state/authStore';
import { MessageContext, useMessageReducer } from './src/state/messageStore';
import { WebSocketClient, type ConnectionState } from './src/services/websocket';
import { mlsService } from './src/services/mls';
import { conversationService } from './src/services/conversation';
import { saveAuthState } from './src/services/storage';
import type {
  AuthChallengeMessage,
  AuthSuccessMessage,
  AuthErrorMessage,
  AuthRegisterChallengeMessage,
  AuthRegisterSuccessMessage,
} from './src/services/protocol';

import Login from './src/screens/Login';
import Register from './src/screens/Register';
import ConversationList from './src/screens/ConversationList';
import Chat from './src/screens/Chat';

// ============================================================================
// Navigation state
// ============================================================================

type Screen =
  | { name: 'connect' }
  | { name: 'login' }
  | { name: 'register' }
  | { name: 'conversations' }
  | { name: 'chat'; conversationId: string };

// ============================================================================
// Server Connect Screen
// ============================================================================

function ConnectScreen({ onConnect }: { onConnect: (url: string) => void }) {
  const [serverUrl, setServerUrl] = useState('ws://localhost:8080/ws');

  return (
    <SafeAreaView style={styles.connectContainer}>
      <View style={styles.connectContent}>
        <Text style={styles.connectLogo}>Sovereign</Text>
        <Text style={styles.connectTagline}>
          Private messaging, your server.
        </Text>

        <View style={styles.connectForm}>
          <Text style={styles.connectLabel}>Server URL</Text>
          <TextInput
            style={styles.connectInput}
            value={serverUrl}
            onChangeText={setServerUrl}
            autoCapitalize="none"
            autoCorrect={false}
            placeholder="ws://your-server:8080/ws"
            placeholderTextColor="#999"
            keyboardType="url"
          />
          <TouchableOpacity
            style={styles.connectButton}
            onPress={() => onConnect(serverUrl.trim())}
            disabled={!serverUrl.trim()}
          >
            <Text style={styles.connectButtonText}>Connect</Text>
          </TouchableOpacity>
        </View>
      </View>
    </SafeAreaView>
  );
}

// ============================================================================
// Main App
// ============================================================================

function AppContent() {
  const auth = useAuthReducer();
  const messages = useMessageReducer();
  const [screen, setScreen] = useState<Screen>({ name: 'connect' });
  const [connectionState, setConnectionState] = useState<ConnectionState>('disconnected');
  const [serverUrl, setServerUrl] = useState('');
  const clientRef = useRef<WebSocketClient | null>(null);

  // Clean up WebSocket on unmount
  useEffect(() => {
    return () => {
      clientRef.current?.disconnect();
    };
  }, []);

  const handleConnect = useCallback((url: string) => {
    setServerUrl(url);

    const client = new WebSocketClient({
      url,
      onMessage: () => {
        // Generic handler for unhandled message types
      },
      onStateChange: (state) => {
        setConnectionState(state);
      },
      authCallbacks: {
        onAuthRequired: () => {
          // Server connected, need to authenticate
        },
        onAuthSuccess: (msg: AuthSuccessMessage) => {
          // Initialize MLS and conversation service
          mlsService.initialize(msg.userId);
          conversationService.setClient(client);

          // Upload initial key packages
          conversationService.uploadKeyPackages(5);

          saveAuthState(url, {
            userId: msg.userId,
            username: msg.username,
            displayName: msg.displayName,
            sessionToken: msg.sessionToken,
          });

          auth.actions.completeLogin({
            userId: msg.userId,
            sessionToken: msg.sessionToken,
            username: msg.username,
            displayName: msg.displayName,
          });

          client.setSessionToken(msg.sessionToken);
          setScreen({ name: 'conversations' });
        },
        onAuthError: (msg: AuthErrorMessage) => {
          auth.actions.setError(msg.message || 'Authentication failed');
        },
        onAuthChallenge: (_msg: AuthChallengeMessage) => {
          // Passkey challenge — in a real app, we'd invoke the platform
          // authenticator here. For now, this is a placeholder.
          auth.actions.setError(
            'Passkey authentication not yet supported in Expo. Use registration to create a test account.',
          );
        },
        onRegisterChallenge: (_msg: AuthRegisterChallengeMessage) => {
          // WebAuthn credential creation — platform authenticator needed.
          auth.actions.setError(
            'Passkey registration requires a platform authenticator (not available in Expo Go).',
          );
        },
        onRegisterSuccess: (msg: AuthRegisterSuccessMessage) => {
          mlsService.initialize(msg.userId);
          conversationService.setClient(client);
          conversationService.uploadKeyPackages(5);

          auth.actions.completeRegistration({
            userId: msg.userId,
            sessionToken: msg.sessionToken,
            username: '',
            displayName: '',
          });

          client.setSessionToken(msg.sessionToken);
          setScreen({ name: 'conversations' });
        },
      },
      messagingCallbacks: {
        onMessageReceive: (envelope, msg) => {
          conversationService.handleMessageReceive(envelope, msg);
        },
        onMessageDelivered: (msg) => {
          conversationService.handleMessageDelivered(msg);
        },
        onGroupCreated: (envelope, msg) => {
          conversationService.handleGroupCreated(envelope, msg);
        },
        onGroupMemberAdded: (msg) => {
          conversationService.handleMemberAdded(msg.conversationId, msg.userId);
        },
        onGroupMemberRemoved: (msg) => {
          conversationService.handleMemberRemoved(msg.conversationId, msg.userId);
        },
        onKeyPackageResponse: (envelope, msg) => {
          conversationService.handleKeyPackageResponse(envelope, msg);
        },
        onWelcomeReceive: (msg) => {
          conversationService.handleWelcomeReceive(msg);
        },
        onCommitBroadcast: (msg) => {
          conversationService.handleCommitBroadcast(msg.conversationId, msg.commitData);
        },
      },
    });

    clientRef.current = client;
    conversationService.setClient(client);
    client.connect();
    setScreen({ name: 'login' });
  }, [auth.actions]);

  const handleNavigateToChat = useCallback((conversationId: string) => {
    setScreen({ name: 'chat', conversationId });
  }, []);

  const handleNewConversation = useCallback(
    async (peerId: string, peerDisplayName: string) => {
      try {
        const conversation = await conversationService.createDirectConversation(
          peerId,
          peerDisplayName,
        );
        messages.actions.addConversation(conversation);
        setScreen({ name: 'chat', conversationId: conversation.id });
      } catch (err: unknown) {
        const message = err instanceof globalThis.Error ? err.message : 'Failed to create conversation';
        auth.actions.setError(message);
      }
    },
    [messages.actions, auth.actions],
  );

  const renderScreen = () => {
    const client = clientRef.current;

    switch (screen.name) {
      case 'connect':
        return <ConnectScreen onConnect={handleConnect} />;

      case 'login':
        if (!client) return null;
        return (
          <Login
            serverUrl={serverUrl}
            client={client}
            onSuccess={() => setScreen({ name: 'conversations' })}
            onNavigateToRegister={() => setScreen({ name: 'register' })}
          />
        );

      case 'register':
        if (!client) return null;
        return (
          <Register
            serverUrl={serverUrl}
            client={client}
            onSuccess={() => setScreen({ name: 'conversations' })}
            onNavigateToLogin={() => setScreen({ name: 'login' })}
          />
        );

      case 'conversations':
        return (
          <ConversationList
            serverId={serverUrl}
            connectionState={connectionState}
            onSelectConversation={handleNavigateToChat}
            onNewConversation={handleNewConversation}
          />
        );

      case 'chat':
        return (
          <Chat
            conversationId={screen.conversationId}
            onBack={() => setScreen({ name: 'conversations' })}
          />
        );
    }
  };

  return (
    <AuthContext.Provider value={auth}>
      <MessageContext.Provider value={messages}>
        <KeyboardAvoidingView
          style={styles.root}
          behavior={Platform.OS === 'ios' ? 'padding' : undefined}
        >
          <StatusBar barStyle="light-content" backgroundColor="#1A1A2E" />
          {renderScreen()}
        </KeyboardAvoidingView>
      </MessageContext.Provider>
    </AuthContext.Provider>
  );
}

export default function App() {
  return <AppContent />;
}

registerRootComponent(App);

// ============================================================================
// Styles
// ============================================================================

const styles = StyleSheet.create({
  root: {
    flex: 1,
    backgroundColor: '#1A1A2E',
  },
  connectContainer: {
    flex: 1,
    backgroundColor: '#1A1A2E',
  },
  connectContent: {
    flex: 1,
    justifyContent: 'center',
    paddingHorizontal: 32,
  },
  connectLogo: {
    fontSize: 42,
    fontWeight: '700',
    color: '#FFFFFF',
    textAlign: 'center',
    marginBottom: 8,
  },
  connectTagline: {
    fontSize: 16,
    color: '#AAAACC',
    textAlign: 'center',
    marginBottom: 48,
  },
  connectForm: {
    backgroundColor: 'rgba(255,255,255,0.08)',
    borderRadius: 16,
    padding: 24,
  },
  connectLabel: {
    fontSize: 14,
    fontWeight: '600',
    color: '#AAAACC',
    marginBottom: 8,
  },
  connectInput: {
    backgroundColor: 'rgba(255,255,255,0.12)',
    borderRadius: 8,
    paddingHorizontal: 14,
    paddingVertical: 12,
    fontSize: 15,
    color: '#FFFFFF',
    marginBottom: 16,
  },
  connectButton: {
    backgroundColor: '#4CAF50',
    borderRadius: 8,
    paddingVertical: 14,
    alignItems: 'center',
  },
  connectButtonText: {
    color: '#FFFFFF',
    fontSize: 16,
    fontWeight: '600',
  },
});
