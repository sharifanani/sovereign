# Review Process

## Overview

Sovereign uses a tiered review process to balance velocity with quality. Changes are categorized by risk and reviewed accordingly.

## Review Tiers

### Tier 1: Minor Changes (Agent-Reviewed)

Changes that are low-risk and localized:
- Bug fixes with clear root cause
- Naming and formatting improvements
- Documentation typos and clarifications
- Test additions (without behavior changes)
- Dependency updates (patch versions)

**Process**: Another agent reviews and approves. No human approval required.

### Tier 2: Standard Changes (Agent-Reviewed + Verification)

Changes that modify behavior but follow established patterns:
- New API endpoints following existing patterns
- UI screen additions following existing design
- Database migration additions
- New test coverage for existing features

**Process**: Relevant engineering agent reviews. QA engineer verifies tests. Architect reviews if component boundaries are involved.

### Tier 3: Major Changes (Human-Approved)

Changes that affect architecture, security, or protocol:
- New RFCs or significant RFC amendments
- Protocol message type additions or modifications
- Authentication or encryption changes
- Database schema changes affecting existing data
- Cross-component interface changes
- Dependency additions (new libraries)

**Process**: Relevant agents review first, then human approval required before merge.

## Security Review Requirements

All changes touching the following MUST be reviewed by the security-engineer:
- `server/internal/auth/` — Authentication logic
- `server/internal/mls/` — MLS group management
- `mobile/src/services/crypto.ts` — Client-side cryptography
- `mobile/src/services/auth.ts` — Client-side authentication
- Any code handling key material, session tokens, or credentials
- Any changes to the WebSocket authentication handshake

## Review Checklist

Reviewers should verify:
- [ ] Code follows project coding standards (`CLAUDE.md`)
- [ ] Tests are included and pass
- [ ] Changes match the relevant RFC/design doc
- [ ] No security regressions (check threat model)
- [ ] Build passes (`make build && make test`)
- [ ] Documentation is updated if behavior changes
