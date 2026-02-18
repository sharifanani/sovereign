// MLS service
// Phase C stepping-stone: NaCl box encryption for 1:1 E2E messaging.
// Uses tweetnacl X25519 Diffie-Hellman key exchange and secretbox symmetric encryption.
// Designed with a clean interface so a real MLS library can be swapped in for Phase D.

import nacl from 'tweetnacl';
import naclUtil from 'tweetnacl-util';

// ============================================================================
// Types
// ============================================================================

export interface KeyPair {
  publicKey: Uint8Array;
  secretKey: Uint8Array;
}

export interface KeyPackage {
  userId: string;
  publicKey: Uint8Array;
  createdAt: number;
}

interface GroupState {
  conversationId: string;
  sharedSecret: Uint8Array;
  peerPublicKey: Uint8Array;
  epoch: number;
}

// Serialized format for key packages transmitted over the wire
interface SerializedKeyPackage {
  userId: string;
  publicKey: string; // base64
  createdAt: number;
}

// Serialized format for Welcome messages transmitted over the wire
interface SerializedWelcome {
  conversationId: string;
  senderPublicKey: string; // base64
  epoch: number;
}

// ============================================================================
// MLS Service
// ============================================================================

export class MLSService {
  private identityKeyPair: KeyPair | null = null;
  private groups: Map<string, GroupState> = new Map();
  private userId = '';

  initialize(userId: string): void {
    this.userId = userId;
    if (!this.identityKeyPair) {
      this.identityKeyPair = nacl.box.keyPair();
    }
  }

  isInitialized(): boolean {
    return this.identityKeyPair !== null && this.userId !== '';
  }

  getUserId(): string {
    return this.userId;
  }

  getPublicKey(): Uint8Array {
    if (!this.identityKeyPair) {
      throw new globalThis.Error('MLS service not initialized');
    }
    return this.identityKeyPair.publicKey;
  }

  // Generate a key package for upload to the server.
  // In a real MLS implementation, key packages contain signed init keys.
  // Here we serialize our public key with metadata.
  generateKeyPackage(): Uint8Array {
    if (!this.identityKeyPair) {
      throw new globalThis.Error('MLS service not initialized');
    }
    const kp: SerializedKeyPackage = {
      userId: this.userId,
      publicKey: naclUtil.encodeBase64(this.identityKeyPair.publicKey),
      createdAt: Date.now(),
    };
    return naclUtil.decodeUTF8(JSON.stringify(kp));
  }

  // Parse a key package received from the server
  parseKeyPackage(data: Uint8Array): KeyPackage {
    const json = naclUtil.encodeUTF8(data);
    const parsed: unknown = JSON.parse(json);
    const kp = parsed as SerializedKeyPackage;
    return {
      userId: kp.userId,
      publicKey: naclUtil.decodeBase64(kp.publicKey),
      createdAt: kp.createdAt,
    };
  }

  // Create an MLS group for a 1:1 conversation.
  // Derives a shared secret from our secret key and the peer's public key.
  createGroup(conversationId: string, peerKeyPackage: KeyPackage): Uint8Array {
    if (!this.identityKeyPair) {
      throw new globalThis.Error('MLS service not initialized');
    }

    const sharedSecret = nacl.box.before(peerKeyPackage.publicKey, this.identityKeyPair.secretKey);

    this.groups.set(conversationId, {
      conversationId,
      sharedSecret,
      peerPublicKey: peerKeyPackage.publicKey,
      epoch: 1,
    });

    // Create a Welcome message for the peer
    const welcome: SerializedWelcome = {
      conversationId,
      senderPublicKey: naclUtil.encodeBase64(this.identityKeyPair.publicKey),
      epoch: 1,
    };
    return naclUtil.decodeUTF8(JSON.stringify(welcome));
  }

  // Process a Welcome message to join a group initiated by a peer
  processWelcome(welcomeData: Uint8Array): string {
    if (!this.identityKeyPair) {
      throw new globalThis.Error('MLS service not initialized');
    }

    const json = naclUtil.encodeUTF8(welcomeData);
    const parsed: unknown = JSON.parse(json);
    const welcome = parsed as SerializedWelcome;

    const peerPublicKey = naclUtil.decodeBase64(welcome.senderPublicKey);
    const sharedSecret = nacl.box.before(peerPublicKey, this.identityKeyPair.secretKey);

    this.groups.set(welcome.conversationId, {
      conversationId: welcome.conversationId,
      sharedSecret,
      peerPublicKey,
      epoch: welcome.epoch,
    });

    return welcome.conversationId;
  }

  // Encrypt plaintext for a conversation using NaCl secretbox
  encrypt(conversationId: string, plaintext: Uint8Array): Uint8Array {
    const group = this.groups.get(conversationId);
    if (!group) {
      throw new globalThis.Error(`No group state for conversation: ${conversationId}`);
    }

    const nonce = nacl.randomBytes(nacl.secretbox.nonceLength);
    const ciphertext = nacl.secretbox(plaintext, nonce, group.sharedSecret);

    // Prepend nonce to ciphertext
    const result = new Uint8Array(nonce.length + ciphertext.length);
    result.set(nonce, 0);
    result.set(ciphertext, nonce.length);
    return result;
  }

  // Decrypt ciphertext from a conversation using NaCl secretbox
  decrypt(conversationId: string, data: Uint8Array): Uint8Array {
    const group = this.groups.get(conversationId);
    if (!group) {
      throw new globalThis.Error(`No group state for conversation: ${conversationId}`);
    }

    const nonce = data.slice(0, nacl.secretbox.nonceLength);
    const ciphertext = data.slice(nacl.secretbox.nonceLength);
    const plaintext = nacl.secretbox.open(ciphertext, nonce, group.sharedSecret);

    if (!plaintext) {
      throw new globalThis.Error('Decryption failed: invalid ciphertext or key');
    }

    return plaintext;
  }

  // Check if we have group state for a conversation
  hasGroup(conversationId: string): boolean {
    return this.groups.has(conversationId);
  }

  // Remove group state (e.g., when leaving a conversation)
  removeGroup(conversationId: string): void {
    this.groups.delete(conversationId);
  }

  // Process an MLS Commit (epoch advancement).
  // In Phase C this is a no-op since 1:1 groups don't change membership.
  processCommit(_conversationId: string, _commitData: Uint8Array): void {
    // No-op for Phase C 1:1 conversations
  }

  // Reset all state (e.g., on logout)
  reset(): void {
    this.identityKeyPair = null;
    this.groups.clear();
    this.userId = '';
  }
}

// Singleton instance
export const mlsService = new MLSService();
