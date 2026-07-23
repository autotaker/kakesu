---
kind: decision
decision_id: DECISION-0004
title: MultiAgentV2 Role Startup
status: accepted
decided_at: 2026-07-15
supersedes:
  - DECISION-0003
---

# MultiAgentV2 Role Startup

## Context

Role-specific model and effort contracts can be bypassed or rejected when launcher metadata is confused: `task_name` is not a role selector, default context inheritance can conflict with a named child role, and a role configuration's sandbox declaration does not itself prove its runtime enforcement.

## Decision

Use internal `agents.spawn_agent` as the standard child-launch path. Treat `task_name` as a tracking identifier and use `agent_type` to select the role. For a child with a role different from its caller, explicitly set `fork_turns="none"`.

If `agent_type` is unavailable, internal spawning cannot be used, or observed model/effort differs from the contract, stop and record requested and observed values plus runtime conditions before deciding whether the restricted `make *-agent` fallback is available. Do not adopt output from a mismatched child.

Keep sequential PLAN→DEV→REVIEW→QA gates, role separation, the child's Git prohibition, and the parent's lock, scope-check, hook, staging, commit, and post-check ownership. Explorer remains limited to one bounded read-only question with no redelegation.

## Consequences

- A `task_name` used as a role selector is rejected as a routing error.
- Default `fork_turns="all"` is not used for cross-role launches.
- Declared sandbox settings are treated as intent until effective runtime permissions are observed; they do not relax Git and lock boundaries.
- The prior routing decision remains an immutable historical record.

## Evidence

- [TASK-0003](../../tasks/TASK-0003-multi-agent-v2-nested-explorer/HANDOVER.md)
