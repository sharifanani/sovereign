import React, { useState, useCallback } from 'react';
import {
  View,
  Text,
  TextInput,
  TouchableOpacity,
  StyleSheet,
  ActivityIndicator,
} from 'react-native';
import { useAuth } from '../state/authStore';
import { sendLoginRequest, handleAuthChallenge } from '../services/auth';
import { saveAuthState } from '../services/storage';
import { type WebSocketClient } from '../services/websocket';
import type {
  AuthChallengeMessage,
  AuthSuccessMessage,
  AuthErrorMessage,
} from '../services/protocol';

interface LoginProps {
  serverUrl: string;
  client: WebSocketClient;
  onSuccess: () => void;
  onNavigateToRegister: () => void;
}

const Login: React.FC<LoginProps> = ({ serverUrl, client, onSuccess, onNavigateToRegister }) => {
  const { state: authState, actions } = useAuth();
  const [username, setUsername] = useState('');
  const [usernameError, setUsernameError] = useState<string | null>(null);

  const validateUsername = useCallback((value: string): boolean => {
    if (!value.trim()) {
      setUsernameError('Username is required');
      return false;
    }
    setUsernameError(null);
    return true;
  }, []);

  const handleLogin = useCallback(() => {
    if (!validateUsername(username)) {
      return;
    }
    actions.startLogin();
    sendLoginRequest(client, username.trim());
  }, [username, client, actions, validateUsername]);

  const onAuthChallenge = useCallback(
    async (challenge: AuthChallengeMessage) => {
      try {
        await handleAuthChallenge(client, challenge);
      } catch (err: unknown) {
        const message = err instanceof globalThis.Error ? err.message : 'Authentication failed';
        actions.setError(message);
      }
    },
    [client, actions],
  );

  const onAuthSuccess = useCallback(
    async (msg: AuthSuccessMessage) => {
      await saveAuthState(serverUrl, {
        userId: msg.userId,
        username: msg.username,
        displayName: msg.displayName,
        sessionToken: msg.sessionToken,
      });
      actions.completeLogin({
        userId: msg.userId,
        sessionToken: msg.sessionToken,
        username: msg.username,
        displayName: msg.displayName,
      });
      onSuccess();
    },
    [serverUrl, actions, onSuccess],
  );

  const onAuthError = useCallback(
    (msg: AuthErrorMessage) => {
      actions.setError(msg.message || 'Login failed');
    },
    [actions],
  );

  const isLoading = authState.status === 'authenticating';

  // Expose callbacks for parent to wire to WebSocket
  React.useEffect(() => {
    loginCallbacksRef.current = { onAuthChallenge, onAuthSuccess, onAuthError };
  }, [onAuthChallenge, onAuthSuccess, onAuthError]);

  return (
    <View style={styles.container}>
      <View style={styles.header}>
        <Text style={styles.title}>Login</Text>
        <Text style={styles.subtitle}>{serverUrl}</Text>
      </View>

      <View style={styles.form}>
        <Text style={styles.welcomeText}>Welcome back.</Text>

        <Text style={styles.label}>Username</Text>
        <TextInput
          style={[styles.input, usernameError ? styles.inputError : null]}
          value={username}
          onChangeText={(text) => {
            setUsername(text);
            if (usernameError) validateUsername(text);
          }}
          autoCapitalize="none"
          autoCorrect={false}
          placeholder="Enter your username"
          placeholderTextColor="#999"
          editable={!isLoading}
        />
        {usernameError ? <Text style={styles.errorText}>{usernameError}</Text> : null}

        {authState.error ? (
          <View style={styles.errorBanner}>
            <Text style={styles.errorBannerText}>{authState.error}</Text>
          </View>
        ) : null}

        <TouchableOpacity
          style={[styles.loginButton, isLoading && styles.buttonDisabled]}
          onPress={handleLogin}
          disabled={isLoading}
        >
          {isLoading ? (
            <ActivityIndicator color="#FFFFFF" />
          ) : (
            <Text style={styles.loginButtonText}>Login with Passkey</Text>
          )}
        </TouchableOpacity>

        <TouchableOpacity
          style={styles.linkButton}
          onPress={onNavigateToRegister}
          disabled={isLoading}
        >
          <Text style={styles.linkText}>Don't have an account? Register</Text>
        </TouchableOpacity>
      </View>
    </View>
  );
};

// Ref for parent components to access auth callbacks
export const loginCallbacksRef: React.MutableRefObject<{
  onAuthChallenge: (challenge: AuthChallengeMessage) => Promise<void>;
  onAuthSuccess: (msg: AuthSuccessMessage) => Promise<void>;
  onAuthError: (msg: AuthErrorMessage) => void;
} | null> = { current: null };

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#F5F5F5',
  },
  header: {
    backgroundColor: '#1A1A2E',
    paddingTop: 60,
    paddingHorizontal: 24,
    paddingBottom: 24,
  },
  title: {
    fontSize: 28,
    fontWeight: '700',
    color: '#FFFFFF',
    marginBottom: 4,
  },
  subtitle: {
    fontSize: 14,
    color: '#AAAACC',
  },
  form: {
    padding: 24,
  },
  welcomeText: {
    fontSize: 18,
    color: '#333333',
    marginBottom: 24,
  },
  label: {
    fontSize: 14,
    fontWeight: '600',
    color: '#333333',
    marginBottom: 6,
  },
  input: {
    backgroundColor: '#FFFFFF',
    borderWidth: 1,
    borderColor: '#E0E0E0',
    borderRadius: 8,
    paddingHorizontal: 14,
    paddingVertical: 12,
    fontSize: 16,
    color: '#333333',
  },
  inputError: {
    borderColor: '#F44336',
  },
  errorText: {
    color: '#F44336',
    fontSize: 12,
    marginTop: 4,
  },
  errorBanner: {
    backgroundColor: '#FFEBEE',
    borderRadius: 8,
    padding: 12,
    marginTop: 16,
  },
  errorBannerText: {
    color: '#C62828',
    fontSize: 14,
  },
  loginButton: {
    backgroundColor: '#1A1A2E',
    borderRadius: 8,
    paddingVertical: 14,
    alignItems: 'center',
    marginTop: 24,
  },
  buttonDisabled: {
    opacity: 0.6,
  },
  loginButtonText: {
    color: '#FFFFFF',
    fontSize: 16,
    fontWeight: '600',
  },
  linkButton: {
    alignItems: 'center',
    marginTop: 16,
  },
  linkText: {
    color: '#1A1A2E',
    fontSize: 14,
  },
});

export default Login;
