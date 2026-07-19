---
name: run-efficient-task-delivery
description: Classify repository work and apply proportionate delivery gates. Use full PLAN, DEV, REVIEW, and QA for product changes, planning gates for safety-contract changes, and a checklist plus one independent review for pure measurement, status, or evidence maintenance. Use when the main Agent starts or continues a Task, closes postmerge evidence, maintains a backlog, or diagnoses delayed delivery without weakening role separation or parent-owned Git boundaries.
---

# Run efficient Task delivery

Use this workflow as the main Agent. First select the lightest safe path, then optimize handoffs, context, polling, and verification without weakening the gates required by that path.

Repository `AGENTS.md` and development contracts take precedence. Do not use this skill to bypass a stricter repository rule. If the repository does not yet permit the proportionate path, change that contract through the safety-contract path before using it.

## Classify the work before choosing gates

Use the full product path when any product code, test, runtime or build configuration, schema, dependency, generated input, or externally observable behavior changes. Preserve the repository's complete PLAN, QA_PLAN, DEV, same-candidate independent REVIEW/QA, and postmerge environment-dependent confirmation gates.

Use the safety-contract path when no product artifact changes, but the work changes a security or authority boundary, threat model, acceptance condition, feature scope, dependency, resource cap, shedding order, or mandatory development control. Require a TASK and PLAN, an independent TASK-first QA_PLAN, and an independent planning review. Do not create DEV, REVIEW, or QA evidence that claims a product implementation passed when no product implementation exists.

Use the lightweight evidence path only when all of the following are true:

- No product code, test, configuration, schema, dependency, build input, or generated product artifact changes.
- No security, authority, threat-model, acceptance, feature-scope, dependency, cap, shedding-order, or mandatory-control decision changes.
- Every edit is limited to backlog status, measurement arithmetic, SLOC or timing data, retry or failure classification, evidence linkage, or an append-only correction of those facts.

When uncertain, choose the safety-contract path. If the independent reviewer finds a contradiction, unsupported provenance, changed acceptance meaning, or a safety implication, stop the lightweight path and reclassify the work. Do not stretch a measurement label around a design decision.

## Run the lightweight evidence path

1. Do not create a new Task ID, dedicated worktree, PLAN, QA_PLAN, DEV Agent, QA Agent, counted Lap, or standalone PR solely for this maintenance. Bundle it into the related product Task's postmerge closure whenever that Task remains open.
2. Have Main record a short checklist containing the purpose, source and provenance, exact paths, arithmetic or mapping being checked, exclusions, affected validation, and rollback or correction method.
3. Use one independent reviewer. Ask that reviewer to verify provenance, arithmetic, status transitions, cross-file identifiers and metadata, schema or parser validity, diff scope, and absence of secrets. A review that discovers changed behavior or policy must reclassify the work rather than approve it.
4. Run only checks that the edited evidence can affect. Do not repeat a full product test suite when the edited files are not product or build inputs. Preserve append-only history and use the repository's correction mechanism instead of rewriting published evidence.
5. Keep staging, commit, lock, hook, and scope checks with Main. Create a standalone branch or PR only when the change cannot be bundled and the repository requires one. Do not write product `REVIEW_RESULT` or `QA_RESULT` files that imply product verification.

Target five to ten minutes as a non-binding diagnostic. Exceeding it is a reason to inspect classification, source quality, or tooling; it is not permission to skip the checklist or independent review.

## Establish the full-path execution boundary

1. For a new Task, first read the Task, every applicable `AGENTS.md`, and the required Wiki/Decision references. Have the Planner and QA Agent create `PLAN.md` and `QA_PLAN.md`, and do not start DEV until the main Agent approves both. On continuation, read the approved `PLAN.md` and `QA_PLAN.md` before resuming.
2. Record the Task branch/worktree, approved DEV profile, role split, allowed files, acceptance checks, and completion evidence. Refuse scope expansion until the main Agent records a new approval.
3. Preflight known permission, lock, dependency, and generated-output requirements. Classify a predictable failure before retrying it; do not spend turns repeating a permission or dependency failure.

## Run the full product gates in order

1. Keep `PLAN → DEV` as the implementation gate. After DEV fixes `candidate_commit` and `candidate_tree`, start Reviewer and QA independently and in parallel from that same candidate; neither PASS is the other's start condition.
2. Start one child at a time with native `agents.spawn_agent`. Treat `task_name` as a tracking identifier and `agent_type` as the role selector; never infer one from the other.
3. When the selected role differs from the caller, pass `fork_turns="none"`. Observe the requested and effective model, reasoning effort, and permission/runtime conditions. If `agent_type` is missing, native spawn is unavailable, or model/effort differs, stop, record requested/observed values and runtime evidence, and let the main Agent decide whether the existing, narrowly scoped fallback is allowed. Do not make CLI or `make work-agent` fallback the normal path.
4. Give each child one owned responsibility and a completion contract: changed paths, prohibited operations, local checks, next-gate prerequisites, and a short evidence summary. Require children to avoid stage, commit, merge, and `.git` writes.
5. Keep the main Agent responsible for shared locks, scope and hook checks, staging, commits, post-checks, merge, and QA-failure classification. Only the main Agent approves and merges to `main`.

### Risk-based QA and candidate evidence

Before DEV, QA assigns every case exactly one `qa_execution_mode`: `evidence-review`, `focused-rerun`, or `live-e2e`, with a rationale and fail-closed condition. `evidence-review` independently audits candidate-bound DEV evidence rather than accepting self-approval; the audit includes case ID, `candidate_commit`, `candidate_tree`, command/test, environment or fixture, cache condition, exit, artifact digest, unexecuted reason, negative detection, and test weakening. `focused-rerun` is allowed for a high-risk case only when hermetic, deterministic, and bounded fixtures fully reproduce the acceptance truth. Cases depending on real OS privilege/auth (including sudo/PAM), install/deploy/generated-config placement, external services or side effects, real restart/rollback/cleanup, or environment-specific integration require `live-e2e`. Unknown environment or unsafe cleanup remains blocked and cannot PASS through another mode.

If a review fix changes the candidate, Main alone chooses `qa_carry_forward`, focused rerun, or full rerun. Carry-forward is allowed only for a non-behavioral change whose explicit low-risk conditions are all proven; it records old/new commit/tree, diff, affected cases, rerun evidence, and reason. It is forbidden for QA FAIL, acceptance or QA_PLAN changes, auth/secrets/sudo/PAM, IPC/Schema/config/dependency, concurrency/lifecycle/persistence/error/fail-closed, test deletion/weakening, unknown impact, or candidate/tree mismatch. After merge Main compares `merge_tree` with the approved candidate tree; only when equal and no environment-dependent case exists may duplicate full confirmation be omitted. Environment-dependent cases retain case-level post-merge confirmation. Existing Task evidence and Lap30 event Schema/JSONL remain valid and are not rewritten.

## Bound context and verification

1. Ask for concise summaries, bounded command output, and exact failure excerpts. Poll for state changes with adaptive waits; do not repeat unchanged full output or use a fixed 60-second poll by habit.
2. On the full product path, let the changing Agent run focused checks first. In each required gate, run the complete verification planned for that gate once and avoid purposeless duplicates within the same gate. Let Reviewer and QA perform their independent checks; never omit the Reviewer's `make check` or the QA `make check` required by the approved `QA_PLAN.md`. On the lightweight evidence path, run the narrower affected checks defined by its checklist.
3. Preserve evidence for each gate. On failure, classify it as implementation, QA-plan, requirement, environment, or regression evidence before assigning blame or retrying.

## Review a delay

When delivery is delayed, read [the TASK-0003 retrospective](references/2026-07-15-task-0003-retrospective.md) and compare the session's waits, tool-output volume, full-check count, late lint discoveries, and permission/dependency waits. Treat its 22–28 minute target as a hypothesis, not an SLO, acceptance gate, or Agent score. Record what changed and what remained invariant; never trade away a gate, independent role, native-spawn contract, or parent-owned Git operation for speed.
