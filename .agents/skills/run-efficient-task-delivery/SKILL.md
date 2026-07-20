---
name: run-efficient-task-delivery
description: Classify repository work and apply proportionate delivery gates. Use full PLAN, DEV, REVIEW, and QA for product changes, PLAN gates for safety-contract changes, and a checklist plus one independent review for pure measurement, status, or evidence maintenance. Use when the main Agent starts or continues a Task, closes postmerge evidence, maintains a backlog, or diagnoses delayed delivery without weakening role separation or parent-owned Git boundaries.
---

# Run efficient Task delivery

Use this skill for execution decisions; do not copy the repository contracts into a prompt or evidence file. The authoritative path definitions and gates are in [`AGENTS.md`](../../../AGENTS.md) and the [development process](../../../docs/development/development-process.md); Task evidence ownership is in [Task management](../../../docs/development/task-management.md), and role/Git boundaries are in [Agent roles](../../../docs/development/agent-roles.md).

## Select the path

1. Classify the change as product, safety contract, or pure evidence maintenance using `AGENTS.md`. When scope or meaning is uncertain, stop the lighter path and use the safety-contract path.
2. For pure evidence maintenance, record the required Main checklist, use one independent reviewer, run only affected checks, and preserve append-only evidence. Do not create product PASS evidence.
3. For product or safety-contract work, require the path-specific approved evidence before editing. Never weaken role separation, Main-owned Git, candidate/tree binding, or postmerge checks.

## Start `planning`

1. Main owns the single `planning input packet` in `TASK.md` and gives the same `packet` to Planner and QA. Run the completion-path `preflight` defined by the development process before starting the measured `Lap`; unresolved `preflight` remains `not_started` or blocked.
2. Planner maps AC-ID to design decision, path, order, failure handling, and estimate. QA independently maps AC-ID to observation from TASK, without using PLAN as its input.
3. Separate `dependency-independent active planning` from dependency wait. When dependencies become ready, reconcile stable references and obtain every required reapproval before DEV.
4. If `active planning` exceeds ten minutes, record the prescribed cause and return the missing decision or input to Main instead of continuing prose polishing.

## Execute and close

1. Use native `agents.spawn_agent` with the approved role and profile. Stop on unavailable routing or model/effort mismatch; only Main may choose the documented fallback.
2. Keep work inside the assigned path/worktree. Run focused checks before the gate's required full check, preserve concise candidate-bound evidence, and classify failures before retrying.
3. For product work, Reviewer and QA start independently from the same candidate. Main owns correction routing, merge/push, candidate/merge-tree comparison, and postmerge environment-dependent confirmation.
4. For safety-contract work, use the PLAN review and contract checks; do not create product DEV/REVIEW/QA PASS evidence.
