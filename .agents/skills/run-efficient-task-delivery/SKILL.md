---
name: run-efficient-task-delivery
description: Run repository Tasks efficiently while preserving PLAN, DEV, REVIEW, and QA gates, role separation, and parent-owned Git boundaries. Use when the main Agent starts or continues a Task, or when it needs to diagnose a delayed delivery and plan a non-binding efficiency improvement.
---

# Run efficient Task delivery

Use this workflow as the main Agent. Keep the required gates and ownership boundaries fixed; optimize handoffs, context, polling, and verification around them.

## Establish the execution boundary

1. For a new Task, first read the Task, every applicable `AGENTS.md`, and the required Wiki/Decision references. Have the Planner and QA Agent create `PLAN.md` and `QA_PLAN.md`, and do not start DEV until the main Agent approves both. On continuation, read the approved `PLAN.md` and `QA_PLAN.md` before resuming.
2. Record the Task branch/worktree, approved DEV profile, role split, allowed files, acceptance checks, and completion evidence. Refuse scope expansion until the main Agent records a new approval.
3. Preflight known permission, lock, dependency, and generated-output requirements. Classify a predictable failure before retrying it; do not spend turns repeating a permission or dependency failure.

## Run the gates in order

1. Keep `PLAN → DEV → REVIEW → QA` sequential. Keep DEV, Reviewer, and QA on independent Agents.
2. Start one child at a time with native `agents.spawn_agent`. Treat `task_name` as a tracking identifier and `agent_type` as the role selector; never infer one from the other.
3. When the selected role differs from the caller, pass `fork_turns="none"`. Observe the requested and effective model, reasoning effort, and permission/runtime conditions. If `agent_type` is missing, native spawn is unavailable, or model/effort differs, stop, record requested/observed values and runtime evidence, and let the main Agent decide whether the existing, narrowly scoped fallback is allowed. Do not make CLI or `make work-agent` fallback the normal path.
4. Give each child one owned responsibility and a completion contract: changed paths, prohibited operations, local checks, next-gate prerequisites, and a short evidence summary. Require children to avoid stage, commit, merge, and `.git` writes.
5. Keep the main Agent responsible for shared locks, scope and hook checks, staging, commits, post-checks, merge, and QA-failure classification. Only the main Agent approves and merges to `main`.

## Bound context and verification

1. Ask for concise summaries, bounded command output, and exact failure excerpts. Poll for state changes with adaptive waits; do not repeat unchanged full output or use a fixed 60-second poll by habit.
2. Let the changing Agent run focused checks first. In each required gate, run the complete verification planned for that gate once and avoid purposeless duplicates within the same gate. Let Reviewer and QA perform their independent checks; never omit the Reviewer's `make check` or the QA `make check` required by the approved `QA_PLAN.md`.
3. Preserve evidence for each gate. On failure, classify it as implementation, QA-plan, requirement, environment, or regression evidence before assigning blame or retrying.

## Review a delay

When delivery is delayed, read [the TASK-0003 retrospective](references/2026-07-15-task-0003-retrospective.md) and compare the session's waits, tool-output volume, full-check count, late lint discoveries, and permission/dependency waits. Treat its 22–28 minute target as a hypothesis, not an SLO, acceptance gate, or Agent score. Record what changed and what remained invariant; never trade away a gate, independent role, native-spawn contract, or parent-owned Git operation for speed.
