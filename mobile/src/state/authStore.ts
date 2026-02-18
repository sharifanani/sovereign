// Auth state management
// Minimal React Context-based store for per-server authentication state

import React, { useCallback, useContext, useMemo, useReducer } from 'react';

export type AuthStatus =
  | 'unauthenticated'
  | 'registering'
  | 'authenticating'
  | 'authenticated';

export interface AuthState {
  userId: string;
  username: string;
  displayName: string;
  sessionToken: string;
  status: AuthStatus;
  error: string | null;
}

const INITIAL_AUTH_STATE: AuthState = {
  userId: '',
  username: '',
  displayName: '',
  sessionToken: '',
  status: 'unauthenticated',
  error: null,
};

type AuthAction =
  | { type: 'START_REGISTRATION' }
  | { type: 'COMPLETE_REGISTRATION'; userId: string; sessionToken: string; username: string; displayName: string }
  | { type: 'START_LOGIN' }
  | { type: 'COMPLETE_LOGIN'; userId: string; sessionToken: string; username: string; displayName: string }
  | { type: 'LOGOUT' }
  | { type: 'SET_ERROR'; error: string }
  | { type: 'CLEAR_ERROR' };

function authReducer(state: AuthState, action: AuthAction): AuthState {
  switch (action.type) {
    case 'START_REGISTRATION':
      return { ...state, status: 'registering', error: null };
    case 'COMPLETE_REGISTRATION':
      return {
        ...state,
        status: 'authenticated',
        userId: action.userId,
        sessionToken: action.sessionToken,
        username: action.username,
        displayName: action.displayName,
        error: null,
      };
    case 'START_LOGIN':
      return { ...state, status: 'authenticating', error: null };
    case 'COMPLETE_LOGIN':
      return {
        ...state,
        status: 'authenticated',
        userId: action.userId,
        sessionToken: action.sessionToken,
        username: action.username,
        displayName: action.displayName,
        error: null,
      };
    case 'LOGOUT':
      return { ...INITIAL_AUTH_STATE };
    case 'SET_ERROR':
      return {
        ...state,
        status: state.status === 'registering' || state.status === 'authenticating'
          ? 'unauthenticated'
          : state.status,
        error: action.error,
      };
    case 'CLEAR_ERROR':
      return { ...state, error: null };
  }
}

export interface AuthActions {
  startRegistration: () => void;
  completeRegistration: (data: { userId: string; sessionToken: string; username: string; displayName: string }) => void;
  startLogin: () => void;
  completeLogin: (data: { userId: string; sessionToken: string; username: string; displayName: string }) => void;
  logout: () => void;
  setError: (error: string) => void;
  clearError: () => void;
}

interface AuthContextValue {
  state: AuthState;
  actions: AuthActions;
}

export const AuthContext = React.createContext<AuthContextValue | null>(null);

export function useAuthReducer(): AuthContextValue {
  const [state, dispatch] = useReducer(authReducer, INITIAL_AUTH_STATE);

  const startRegistration = useCallback(() => dispatch({ type: 'START_REGISTRATION' }), []);
  const completeRegistration = useCallback(
    (data: { userId: string; sessionToken: string; username: string; displayName: string }) =>
      dispatch({ type: 'COMPLETE_REGISTRATION', ...data }),
    [],
  );
  const startLogin = useCallback(() => dispatch({ type: 'START_LOGIN' }), []);
  const completeLogin = useCallback(
    (data: { userId: string; sessionToken: string; username: string; displayName: string }) =>
      dispatch({ type: 'COMPLETE_LOGIN', ...data }),
    [],
  );
  const logout = useCallback(() => dispatch({ type: 'LOGOUT' }), []);
  const setError = useCallback((error: string) => dispatch({ type: 'SET_ERROR', error }), []);
  const clearError = useCallback(() => dispatch({ type: 'CLEAR_ERROR' }), []);

  const actions = useMemo(
    () => ({ startRegistration, completeRegistration, startLogin, completeLogin, logout, setError, clearError }),
    [startRegistration, completeRegistration, startLogin, completeLogin, logout, setError, clearError],
  );

  return useMemo(() => ({ state, actions }), [state, actions]);
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new globalThis.Error('useAuth must be used within an AuthContext.Provider');
  }
  return ctx;
}
