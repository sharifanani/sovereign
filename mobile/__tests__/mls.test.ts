import { MLSService } from '../src/services/mls';

describe('MLSService', () => {
  let service: MLSService;

  beforeEach(() => {
    service = new MLSService();
  });

  describe('initialize', () => {
    it('generates an identity key pair and stores the userId', () => {
      service.initialize('user-1');
      expect(service.isInitialized()).toBe(true);
      expect(service.getUserId()).toBe('user-1');
    });

    it('getPublicKey returns a 32-byte Curve25519 public key', () => {
      service.initialize('user-1');
      const pk = service.getPublicKey();
      expect(pk).toBeInstanceOf(Uint8Array);
      expect(pk.length).toBe(32);
    });

    it('preserves the same key pair across multiple initialize calls', () => {
      service.initialize('user-1');
      const pk1 = service.getPublicKey();
      service.initialize('user-1');
      const pk2 = service.getPublicKey();
      expect(pk1).toEqual(pk2);
    });

    it('throws if getPublicKey is called before initialization', () => {
      expect(() => service.getPublicKey()).toThrow('MLS service not initialized');
    });
  });

  describe('isInitialized', () => {
    it('returns false before initialization', () => {
      expect(service.isInitialized()).toBe(false);
    });

    it('returns false after reset', () => {
      service.initialize('user-1');
      service.reset();
      expect(service.isInitialized()).toBe(false);
    });
  });

  describe('generateKeyPackage / parseKeyPackage', () => {
    it('round-trips a key package through serialize and parse', () => {
      service.initialize('alice');
      const kpBytes = service.generateKeyPackage();
      expect(kpBytes).toBeInstanceOf(Uint8Array);
      expect(kpBytes.length).toBeGreaterThan(0);

      const kp = service.parseKeyPackage(kpBytes);
      expect(kp.userId).toBe('alice');
      expect(kp.publicKey).toEqual(service.getPublicKey());
      expect(kp.createdAt).toBeGreaterThan(0);
    });

    it('throws if generateKeyPackage is called before initialization', () => {
      expect(() => service.generateKeyPackage()).toThrow('MLS service not initialized');
    });

    it('generates key packages with increasing createdAt timestamps', () => {
      service.initialize('alice');
      const kp1 = service.parseKeyPackage(service.generateKeyPackage());
      const kp2 = service.parseKeyPackage(service.generateKeyPackage());
      expect(kp2.createdAt).toBeGreaterThanOrEqual(kp1.createdAt);
    });
  });

  describe('createGroup', () => {
    it('creates a group and returns Welcome data', () => {
      const alice = new MLSService();
      alice.initialize('alice');

      const bob = new MLSService();
      bob.initialize('bob');
      const bobKp = bob.parseKeyPackage(bob.generateKeyPackage());

      const welcomeData = alice.createGroup('conv-1', bobKp);
      expect(welcomeData).toBeInstanceOf(Uint8Array);
      expect(welcomeData.length).toBeGreaterThan(0);
      expect(alice.hasGroup('conv-1')).toBe(true);
    });

    it('throws if not initialized', () => {
      const bob = new MLSService();
      bob.initialize('bob');
      const bobKp = bob.parseKeyPackage(bob.generateKeyPackage());

      expect(() => service.createGroup('conv-1', bobKp)).toThrow('MLS service not initialized');
    });
  });

  describe('processWelcome', () => {
    it('joins a group from a Welcome and returns the conversationId', () => {
      const alice = new MLSService();
      alice.initialize('alice');

      const bob = new MLSService();
      bob.initialize('bob');
      const bobKp = bob.parseKeyPackage(bob.generateKeyPackage());

      const welcomeData = alice.createGroup('conv-1', bobKp);
      const conversationId = bob.processWelcome(welcomeData);

      expect(conversationId).toBe('conv-1');
      expect(bob.hasGroup('conv-1')).toBe(true);
    });

    it('throws if not initialized', () => {
      const alice = new MLSService();
      alice.initialize('alice');

      const bob = new MLSService();
      bob.initialize('bob');
      const bobKp = bob.parseKeyPackage(bob.generateKeyPackage());
      const welcomeData = alice.createGroup('conv-1', bobKp);

      const uninit = new MLSService();
      expect(() => uninit.processWelcome(welcomeData)).toThrow('MLS service not initialized');
    });
  });

  describe('encrypt / decrypt', () => {
    let alice: MLSService;
    let bob: MLSService;

    beforeEach(() => {
      alice = new MLSService();
      alice.initialize('alice');

      bob = new MLSService();
      bob.initialize('bob');

      const bobKp = bob.parseKeyPackage(bob.generateKeyPackage());
      const welcomeData = alice.createGroup('conv-1', bobKp);
      bob.processWelcome(welcomeData);
    });

    it('round-trips plaintext through encrypt then decrypt (same party)', () => {
      const plaintext = new Uint8Array([72, 101, 108, 108, 111]); // "Hello"
      const ciphertext = alice.encrypt('conv-1', plaintext);
      const decrypted = alice.decrypt('conv-1', ciphertext);
      expect(decrypted).toEqual(plaintext);
    });

    it('cross-party: Alice encrypts, Bob decrypts', () => {
      const plaintext = new Uint8Array([72, 101, 108, 108, 111]);
      const ciphertext = alice.encrypt('conv-1', plaintext);
      const decrypted = bob.decrypt('conv-1', ciphertext);
      expect(decrypted).toEqual(plaintext);
    });

    it('cross-party: Bob encrypts, Alice decrypts', () => {
      const plaintext = new Uint8Array([87, 111, 114, 108, 100]); // "World"
      const ciphertext = bob.encrypt('conv-1', plaintext);
      const decrypted = alice.decrypt('conv-1', ciphertext);
      expect(decrypted).toEqual(plaintext);
    });

    it('produces unique ciphertexts for the same plaintext (random nonce)', () => {
      const plaintext = new Uint8Array([1, 2, 3]);
      const ct1 = alice.encrypt('conv-1', plaintext);
      const ct2 = alice.encrypt('conv-1', plaintext);
      // Different nonces mean different ciphertexts
      expect(ct1).not.toEqual(ct2);
      // But both decrypt to the same plaintext
      expect(alice.decrypt('conv-1', ct1)).toEqual(plaintext);
      expect(alice.decrypt('conv-1', ct2)).toEqual(plaintext);
    });

    it('ciphertext includes nonce prefix (24 bytes)', () => {
      const plaintext = new Uint8Array([42]);
      const ciphertext = alice.encrypt('conv-1', plaintext);
      // NaCl secretbox nonce is 24 bytes, ciphertext adds MAC overhead (16 bytes)
      expect(ciphertext.length).toBe(24 + 1 + 16);
    });

    it('handles empty plaintext', () => {
      const plaintext = new Uint8Array([]);
      const ciphertext = alice.encrypt('conv-1', plaintext);
      const decrypted = alice.decrypt('conv-1', ciphertext);
      expect(decrypted).toEqual(plaintext);
    });

    it('throws on invalid ciphertext (tampered data)', () => {
      const plaintext = new Uint8Array([1, 2, 3, 4]);
      const ciphertext = alice.encrypt('conv-1', plaintext);
      // Tamper with the ciphertext
      ciphertext[ciphertext.length - 1] ^= 0xff;
      expect(() => bob.decrypt('conv-1', ciphertext)).toThrow('Decryption failed');
    });

    it('throws on ciphertext that is too short (less than nonce length)', () => {
      const shortData = new Uint8Array([1, 2, 3]);
      expect(() => bob.decrypt('conv-1', shortData)).toThrow();
    });

    it('throws when no group state exists for the conversationId', () => {
      const plaintext = new Uint8Array([1]);
      expect(() => alice.encrypt('nonexistent', plaintext)).toThrow('No group state for conversation');
      expect(() => alice.decrypt('nonexistent', new Uint8Array([]))).toThrow('No group state for conversation');
    });
  });

  describe('hasGroup / removeGroup', () => {
    it('hasGroup returns false for unknown conversations', () => {
      service.initialize('user-1');
      expect(service.hasGroup('nonexistent')).toBe(false);
    });

    it('removeGroup deletes group state', () => {
      const alice = new MLSService();
      alice.initialize('alice');
      const bob = new MLSService();
      bob.initialize('bob');
      const bobKp = bob.parseKeyPackage(bob.generateKeyPackage());

      alice.createGroup('conv-1', bobKp);
      expect(alice.hasGroup('conv-1')).toBe(true);

      alice.removeGroup('conv-1');
      expect(alice.hasGroup('conv-1')).toBe(false);
    });

    it('removeGroup for nonexistent group does not throw', () => {
      service.initialize('user-1');
      expect(() => service.removeGroup('nonexistent')).not.toThrow();
    });
  });

  describe('processCommit', () => {
    it('is a no-op in Phase C (does not throw)', () => {
      const alice = new MLSService();
      alice.initialize('alice');
      const bob = new MLSService();
      bob.initialize('bob');
      const bobKp = bob.parseKeyPackage(bob.generateKeyPackage());
      alice.createGroup('conv-1', bobKp);

      expect(() => alice.processCommit('conv-1', new Uint8Array([]))).not.toThrow();
    });
  });

  describe('reset', () => {
    it('clears identity key pair, userId, and all group state', () => {
      service.initialize('user-1');
      const bob = new MLSService();
      bob.initialize('bob');
      const bobKp = bob.parseKeyPackage(bob.generateKeyPackage());
      service.createGroup('conv-1', bobKp);

      service.reset();

      expect(service.isInitialized()).toBe(false);
      expect(service.getUserId()).toBe('');
      expect(service.hasGroup('conv-1')).toBe(false);
    });
  });
});
