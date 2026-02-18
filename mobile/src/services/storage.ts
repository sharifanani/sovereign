// Storage service
// Placeholder using in-memory storage for development.
// NOTE: For production, replace with react-native-keychain or expo-secure-store
// for session tokens and sensitive data. AsyncStorage is NOT secure for credentials.

interface StoredAuthState {
  userId: string;
  username: string;
  displayName: string;
  sessionToken: string;
}

// In-memory storage as a development placeholder.
// In production this must use the platform keychain (iOS Keychain / Android Keystore).
const authStorage = new Map<string, StoredAuthState>();

function serverKey(serverUrl: string): string {
  return serverUrl.replace(/\/+$/, '').toLowerCase();
}

export async function saveAuthState(
  serverUrl: string,
  state: StoredAuthState,
): Promise<void> {
  authStorage.set(serverKey(serverUrl), state);
}

export async function loadAuthState(
  serverUrl: string,
): Promise<StoredAuthState | null> {
  return authStorage.get(serverKey(serverUrl)) ?? null;
}

export async function clearAuthState(serverUrl: string): Promise<void> {
  authStorage.delete(serverKey(serverUrl));
}

export async function getSessionToken(
  serverUrl: string,
): Promise<string | null> {
  const state = authStorage.get(serverKey(serverUrl));
  return state?.sessionToken ?? null;
}

export async function saveSessionToken(
  serverUrl: string,
  token: string,
): Promise<void> {
  const existing = authStorage.get(serverKey(serverUrl));
  if (existing) {
    existing.sessionToken = token;
    authStorage.set(serverKey(serverUrl), existing);
  }
}
