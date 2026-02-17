# ADR-0003: React Native for Mobile

- **Status**: Accepted
- **Date**: 2026-02-16

## Context

Sovereign needs a mobile client for iOS and Android. The mobile client is the primary user-facing interface — it handles messaging, MLS encryption/decryption, key management, passkey authentication, and multi-server connections.

Requirements:

- Support both iOS and Android from a single codebase where possible.
- Access to native platform APIs: hardware-backed key storage (Keychain/Keystore), biometric authentication, push notifications.
- Ability to integrate native cryptographic libraries for MLS operations.
- TypeScript support for type safety, especially around protocol buffer message types.

Alternatives considered:

- **Flutter (Dart)**: Cross-platform with good performance, but the Dart ecosystem is less mature for cryptographic operations and native module interop is more cumbersome via platform channels.
- **Native per-platform (Swift + Kotlin)**: Best performance and platform integration, but doubles the development effort and creates two codebases to maintain.
- **PWA**: No access to hardware-backed key storage, limited push notification support on iOS, cannot interact with the platform keychain for passkey storage.

## Decision

We will use **React Native with TypeScript** for the Sovereign mobile client.

React Native allows us to write the UI and business logic once in TypeScript while accessing native platform APIs through the native module bridge. TypeScript provides type safety that aligns with our protocol buffer generated types. The native module bridge enables integration with platform-specific cryptographic APIs and MLS libraries.

## Consequences

### Positive

- **Single codebase**: One TypeScript codebase targets both iOS and Android, reducing development and maintenance effort.
- **TypeScript type safety**: Protocol buffer generated TypeScript types flow through the entire client codebase, catching type errors at compile time.
- **Large ecosystem**: Access to npm packages for WebSocket management, state management, UI components, and more.
- **Native module bridge**: Critical for Sovereign — allows calling into native MLS libraries, platform keychain, and biometric APIs from TypeScript.
- **Hot reload**: Fast development iteration with React Native's hot reload during UI development.

### Negative

- **Native module complexity for MLS**: The MLS implementation will require native modules (likely wrapping a Rust or C MLS library), which adds build complexity and platform-specific code.
- **React Native version churn**: The React Native ecosystem evolves rapidly, requiring periodic upgrades that can be disruptive.
- **Performance overhead**: JavaScript bridge introduces overhead compared to fully native apps. For a messaging app (I/O-bound, not GPU-bound), this is acceptable.
- **Debugging complexity**: Issues can span JavaScript, native bridge, and native code layers, making debugging more involved.

### Neutral

- The team must maintain familiarity with both the React Native/TypeScript ecosystem and the native build toolchains (Xcode, Android Studio/Gradle).
- React Native's "New Architecture" (Fabric, TurboModules) is now stable and should be adopted to reduce bridge overhead for crypto operations.
