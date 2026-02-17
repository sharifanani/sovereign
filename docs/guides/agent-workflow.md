# Agent Workflow Guide

## Overview

Sovereign uses 7 specialized AI subagents to develop the project. Each agent has defined responsibilities, owned directories, and a consistent workflow for contributing to the codebase.

## Agent Roster

| Agent | Model | Primary Focus |
|-------|-------|---------------|
| architect | opus | Design, RFCs, ADRs, protocol specification |
| backend-engineer | opus | Go server, WebSocket, SQLite, admin API |
| mobile-engineer | opus | React Native client, multi-server support |
| security-engineer | opus | MLS encryption, passkey auth, threat model |
| devops-engineer | sonnet | Build scripts, CI/CD, cross-compilation |
| technical-writer | sonnet | Documentation, guides, README |
| qa-engineer | sonnet | Test strategy, integration tests, conformance |

## Workflow

### 1. Task Assignment

Agents receive tasks through GitHub issues or direct instruction. Before starting work:
- Read the relevant RFC or design document
- Check for blocking dependencies
- Verify the task is within your owned directories

### 2. Design Phase (if required)

For significant changes:
1. Check existing ADRs and RFCs for relevant decisions
2. If a new architectural decision is needed, create an ADR
3. If a new feature is being proposed, draft an RFC
4. Get the appropriate level of review (see Review Process)

### 3. Implementation

1. Create a feature branch: `{issue-number}-{description}`
2. Implement the change following project coding standards
3. Write tests alongside implementation
4. Ensure all linting and build checks pass

### 4. Review Request

1. Create a PR with a clear description of what changed and why
2. Tag the appropriate reviewers based on the review process
3. Address review feedback
4. Merge after approval

## Directory Ownership

Each agent has primary ownership of specific directories. Agents should:
- **Own**: Make changes freely within owned directories
- **Coordinate**: Request review when modifying another agent's owned directories
- **Never**: Modify files outside owned directories without coordination

## Cross-Agent Coordination

When a change spans multiple agents' domains:
1. The architect agent designs the cross-component interface
2. Each agent implements their side of the interface
3. The qa-engineer verifies the integration
4. The security-engineer reviews any auth/crypto implications

## Memory and Context

- Agents should reference `CLAUDE.md` for project rules
- Check `docs/design/` for architectural context
- Read relevant RFCs before implementing features
- Update documentation when implementations diverge from designs
