// Auth service
// Passkey/WebAuthn authentication for Sovereign mobile client
//
// NOTE: The actual WebAuthn/passkey API calls require a native module
// (e.g., react-native-passkey) that bridges to iOS AuthenticationServices /
// Android FIDO2 API. This module provides a placeholder abstraction.
// Replace the `performPasskeyRegistration` and `performPasskeyAuthentication`
// functions with real native module calls when integrating.

import {
  encodeAuthRequest,
  encodeAuthResponse,
  encodeAuthRegisterRequest,
  encodeAuthRegisterResponse,
  MessageType,
  generateRequestId,
  type AuthChallengeMessage,
  type AuthRegisterChallengeMessage,
  type Envelope,
} from './protocol';
import { type WebSocketClient } from './websocket';

// Result from the platform's WebAuthn registration ceremony
export interface PasskeyRegistrationResult {
  credentialId: Uint8Array;
  authenticatorData: Uint8Array;
  clientDataJson: Uint8Array;
  attestationObject: Uint8Array;
}

// Result from the platform's WebAuthn authentication ceremony
export interface PasskeyAuthenticationResult {
  credentialId: Uint8Array;
  authenticatorData: Uint8Array;
  clientDataJson: Uint8Array;
  signature: Uint8Array;
}

// Placeholder: perform passkey registration via the platform authenticator.
// In production, this calls into a native module that invokes
// ASAuthorizationPlatformPublicKeyCredentialProvider (iOS) or
// Fido2ApiClient (Android).
async function performPasskeyRegistration(
  _challenge: AuthRegisterChallengeMessage,
): Promise<PasskeyRegistrationResult> {
  // TODO: Replace with react-native-passkey or equivalent native module.
  // The native module should:
  // 1. Parse credentialCreationOptions from the challenge
  // 2. Present the system passkey creation UI
  // 3. Return the attestation response
  throw new globalThis.Error(
    'Passkey registration requires a native module (react-native-passkey). ' +
    'This is a placeholder for development.',
  );
}

// Placeholder: perform passkey authentication via the platform authenticator.
async function performPasskeyAuthentication(
  _challenge: AuthChallengeMessage,
): Promise<PasskeyAuthenticationResult> {
  // TODO: Replace with react-native-passkey or equivalent native module.
  // The native module should:
  // 1. Parse credentialRequestOptions from the challenge
  // 2. Present the system passkey authentication UI
  // 3. Return the assertion response
  throw new globalThis.Error(
    'Passkey authentication requires a native module (react-native-passkey). ' +
    'This is a placeholder for development.',
  );
}

// Send a registration request to the server
export function sendRegisterRequest(
  client: WebSocketClient,
  username: string,
  displayName: string,
): void {
  const payload = encodeAuthRegisterRequest({ username, displayName });
  const envelope: Envelope = {
    type: MessageType.AUTH_REGISTER_REQUEST,
    requestId: generateRequestId(),
    payload,
  };
  client.send(envelope);
}

// Handle registration challenge from server: perform passkey creation and send response
export async function handleRegisterChallenge(
  client: WebSocketClient,
  challenge: AuthRegisterChallengeMessage,
): Promise<void> {
  const result = await performPasskeyRegistration(challenge);
  const payload = encodeAuthRegisterResponse({
    credentialId: result.credentialId,
    authenticatorData: result.authenticatorData,
    clientDataJson: result.clientDataJson,
    attestationObject: result.attestationObject,
  });
  const envelope: Envelope = {
    type: MessageType.AUTH_REGISTER_RESPONSE,
    requestId: generateRequestId(),
    payload,
  };
  client.send(envelope);
}

// Send a login request to the server
export function sendLoginRequest(
  client: WebSocketClient,
  username: string,
): void {
  const payload = encodeAuthRequest({ username });
  const envelope: Envelope = {
    type: MessageType.AUTH_REQUEST,
    requestId: generateRequestId(),
    payload,
  };
  client.send(envelope);
}

// Handle auth challenge from server: perform passkey authentication and send response
export async function handleAuthChallenge(
  client: WebSocketClient,
  challenge: AuthChallengeMessage,
): Promise<void> {
  const result = await performPasskeyAuthentication(challenge);
  const payload = encodeAuthResponse({
    credentialId: result.credentialId,
    authenticatorData: result.authenticatorData,
    clientDataJson: result.clientDataJson,
    signature: result.signature,
  });
  const envelope: Envelope = {
    type: MessageType.AUTH_RESPONSE,
    requestId: generateRequestId(),
    payload,
  };
  client.send(envelope);
}
