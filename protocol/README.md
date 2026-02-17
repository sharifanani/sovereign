# Sovereign Protocol Definitions

This directory contains the canonical protocol definitions for the Sovereign messaging system. All client and server implementations must conform to these definitions.

## Contents

| File              | Description                                                      |
|-------------------|------------------------------------------------------------------|
| `messages.proto`  | Protocol Buffers definitions for all message types and envelopes |
| `errors.json`     | Machine-readable error code definitions                          |

## Source of Truth

- **`messages.proto`** is the authoritative source of truth for all message types, field names, field numbers, and the `MessageType` enum. Generated Go and TypeScript code is derived from this file.
- **`errors.json`** defines all error codes, their categories, descriptions, and whether they cause connection termination. Both server and client implementations should reference this file for consistent error handling.

## Generating Code from Proto Definitions

### Prerequisites

Install the Protocol Buffers compiler and language-specific plugins:

```bash
# Install protoc (macOS)
brew install protobuf

# Install Go plugin
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

# Install TypeScript plugin
npm install -g ts-proto
```

### Generate Go Stubs

From the repository root:

```bash
protoc \
  --proto_path=protocol \
  --go_out=server/internal/protocol \
  --go_opt=paths=source_relative \
  protocol/messages.proto
```

This generates `messages.pb.go` in `server/internal/protocol/`.

### Generate TypeScript Stubs

From the repository root:

```bash
protoc \
  --proto_path=protocol \
  --plugin=protoc-gen-ts_proto=$(which protoc-gen-ts_proto) \
  --ts_proto_out=client/src/protocol \
  --ts_proto_opt=outputEncodeMethods=true \
  --ts_proto_opt=outputJsonMethods=true \
  protocol/messages.proto
```

This generates `messages.ts` in `client/src/protocol/`.

## Backward Compatibility Rules

To maintain backward compatibility between different versions of clients and servers:

1. **Never remove a field.** Mark deprecated fields with `[deprecated = true]` and stop using them, but do not delete the field definition.
2. **Never change a field number.** Field numbers are the wire-format identifiers. Changing them breaks all existing serialized data.
3. **Never renumber enum values.** Enum values are serialized as integers. Renumbering changes the meaning of existing data.
4. **Add new fields with new field numbers.** Use the next available field number when adding fields to a message.
5. **Add new enum values at the end.** New `MessageType` values should use the next available number in their category range.
6. **Reserve removed field numbers.** If a field is removed, add a `reserved` statement to prevent accidental reuse:
   ```protobuf
   message Example {
     reserved 3, 7;
     reserved "old_field_name";
   }
   ```

These rules ensure that a newer client can communicate with an older server (and vice versa) without deserialization failures.
