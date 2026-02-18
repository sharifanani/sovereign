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
import { sendRegisterRequest, handleRegisterChallenge } from '../services/auth';
import { saveAuthState } from '../services/storage';
import { type WebSocketClient } from '../services/websocket';
import type {
  AuthRegisterChallengeMessage,
  AuthRegisterSuccessMessage,
  AuthErrorMessage,
} from '../services/protocol';

interface RegisterProps {
  serverUrl: string;
  client: WebSocketClient;
  onSuccess: () => void;
  onNavigateToLogin: () => void;
}

const USERNAME_REGEX = /^[a-zA-Z0-9]{3,20}$/;

const Register: React.FC<RegisterProps> = ({ serverUrl, client, onSuccess, onNavigateToLogin }) => {
  const { state: authState, actions } = useAuth();
  const [username, setUsername] = useState('');
  const [displayName, setDisplayName] = useState('');
  const [usernameError, setUsernameError] = useState<string | null>(null);
  const [displayNameError, setDisplayNameError] = useState<string | null>(null);

  const validateUsername = useCallback((value: string): boolean => {
    if (!value) {
      setUsernameError('Username is required');
      return false;
    }
    if (!USERNAME_REGEX.test(value)) {
      setUsernameError('3-20 alphanumeric characters');
      return false;
    }
    setUsernameError(null);
    return true;
  }, []);

  const validateDisplayName = useCallback((value: string): boolean => {
    if (!value.trim()) {
      setDisplayNameError('Display name is required');
      return false;
    }
    setDisplayNameError(null);
    return true;
  }, []);

  const handleRegister = useCallback(() => {
    const validUser = validateUsername(username);
    const validDisplay = validateDisplayName(displayName);
    if (!validUser || !validDisplay) {
      return;
    }

    actions.startRegistration();
    sendRegisterRequest(client, username, displayName.trim());
  }, [username, displayName, client, actions, validateUsername, validateDisplayName]);

  // These callbacks are called from the parent component which wires them
  // to the WebSocket auth callbacks
  const onRegisterChallenge = useCallback(
    async (challenge: AuthRegisterChallengeMessage) => {
      try {
        await handleRegisterChallenge(client, challenge);
      } catch (err: unknown) {
        const message = err instanceof globalThis.Error ? err.message : 'Registration failed';
        actions.setError(message);
      }
    },
    [client, actions],
  );

  const onRegisterSuccess = useCallback(
    async (msg: AuthRegisterSuccessMessage) => {
      await saveAuthState(serverUrl, {
        userId: msg.userId,
        username,
        displayName: displayName.trim(),
        sessionToken: msg.sessionToken,
      });
      actions.completeRegistration({
        userId: msg.userId,
        sessionToken: msg.sessionToken,
        username,
        displayName: displayName.trim(),
      });
      onSuccess();
    },
    [serverUrl, username, displayName, actions, onSuccess],
  );

  const onAuthError = useCallback(
    (msg: AuthErrorMessage) => {
      actions.setError(msg.message || 'Registration failed');
    },
    [actions],
  );

  const isLoading = authState.status === 'registering';

  // Expose callbacks for parent to wire to WebSocket
  React.useEffect(() => {
    registerCallbacksRef.current = { onRegisterChallenge, onRegisterSuccess, onAuthError };
  }, [onRegisterChallenge, onRegisterSuccess, onAuthError]);

  return (
    <View style={styles.container}>
      <View style={styles.header}>
        <Text style={styles.title}>Create Account</Text>
        <Text style={styles.subtitle}>{serverUrl}</Text>
      </View>

      <View style={styles.form}>
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
          placeholder="alphanumeric, 3-20 chars"
          placeholderTextColor="#999"
          editable={!isLoading}
        />
        {usernameError ? <Text style={styles.errorText}>{usernameError}</Text> : null}

        <Text style={styles.label}>Display Name</Text>
        <TextInput
          style={[styles.input, displayNameError ? styles.inputError : null]}
          value={displayName}
          onChangeText={(text) => {
            setDisplayName(text);
            if (displayNameError) validateDisplayName(text);
          }}
          placeholder="Your display name"
          placeholderTextColor="#999"
          editable={!isLoading}
        />
        {displayNameError ? <Text style={styles.errorText}>{displayNameError}</Text> : null}

        {authState.error ? (
          <View style={styles.errorBanner}>
            <Text style={styles.errorBannerText}>{authState.error}</Text>
          </View>
        ) : null}

        <TouchableOpacity
          style={[styles.registerButton, isLoading && styles.buttonDisabled]}
          onPress={handleRegister}
          disabled={isLoading}
        >
          {isLoading ? (
            <ActivityIndicator color="#FFFFFF" />
          ) : (
            <Text style={styles.registerButtonText}>Register</Text>
          )}
        </TouchableOpacity>

        <TouchableOpacity
          style={styles.linkButton}
          onPress={onNavigateToLogin}
          disabled={isLoading}
        >
          <Text style={styles.linkText}>Already have an account? Login</Text>
        </TouchableOpacity>
      </View>
    </View>
  );
};

// Ref for parent components to access auth callbacks
export const registerCallbacksRef: React.MutableRefObject<{
  onRegisterChallenge: (challenge: AuthRegisterChallengeMessage) => Promise<void>;
  onRegisterSuccess: (msg: AuthRegisterSuccessMessage) => Promise<void>;
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
  label: {
    fontSize: 14,
    fontWeight: '600',
    color: '#333333',
    marginBottom: 6,
    marginTop: 16,
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
  registerButton: {
    backgroundColor: '#1A1A2E',
    borderRadius: 8,
    paddingVertical: 14,
    alignItems: 'center',
    marginTop: 24,
  },
  buttonDisabled: {
    opacity: 0.6,
  },
  registerButtonText: {
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

export default Register;
