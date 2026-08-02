---
task_id: "TASK-0069"
status: complete
qa_agent: "qa-agent-terra-medium"
decision: pass
tested_at: "2026-08-02T05:49:49Z"
---

# TASK-0069 QA RESULT

## Candidate binding

- Candidate worktree HEAD matched `HANDOVER.md`; the worktree was clean at QA start.
- QA used TASK.md and QA_PLAN.md, and did not use PLAN or reviewer results as inputs.

## Results

| Case | Mode | Command or review | Result |
|---|---|---|---|
| QA-001 | `focused-rerun` | focused race suite; `TestLauncherExactRunAndStdio`, `TestLauncherRejectsNonExactArgumentsBeforeSession`, and help/version/fixed-diagnostic assertions | `pass` — exact CLI, pre-session rejection, direct argv/stdio, and fixed diagnostics are covered. |
| QA-002 | `focused-rerun` | focused race suite; fixed-path rejection and partial-acquisition/revoke assertions | `pass` — fixed clean paths, ordered single issue calls, stop-at-first-failure, and owned-handle revocation are covered. |
| QA-003 | `focused-rerun` | focused race suite; CA setup/start failure and drain/lifecycle assertions | `pass` — literal `/tmp` lifecycle, mode/type/symlink defenses, bridge drain, cleanup, cancellation, and bridge-failure paths are covered. |
| QA-004 | `focused-rerun` | focused race suite; hostile-parent environment and Git command-scope assertions | `pass` — allowlist rebuild, Opaque session values, proxy/CA values, prompt disablement, helper reset, fixed Git settings, and forced `GIT_CONFIG_NOSYSTEM=1`/`GIT_CONFIG_GLOBAL=/dev/null` are covered. |
| QA-005 | `focused-rerun` | focused race suite; child exit/cancel/bridge failure/cleanup and diagnostic assertions | `pass` — ordinary child status retention, launcher failure folding, child wait before return, non-leak, and revoke-failure handling are covered. |
| QA-006 | `evidence-review` | candidate diff, README, and HANDOVER candidate-bound records | `pass` — seven permitted paths only (1,199 changed lines); README states the isolation limit and system/global Git-config exclusion; HANDOVER records passing focused, harness/root, distcheck, install-staging, and diff checks for the final candidate. |

The single permitted focused command was run once in the candidate worktree:

`cd tools/dev-agent-harness && GOCACHE=$PWD/.build/go-cache go test -count=1 -race ./internal/launchsession ./internal/command` — `pass`.

## Live E2E boundary

`blocked/not run`: real GitHub/OpenAI/Codex credentials; actual Git/gh/Codex/OpenAI client behavior; OS default-deny firewall or network namespace, loopback isolation, Unix-socket ownership/peer UID, loader/process containment; and systemd/VPS deployment, restart, rollback, and cleanup. Hermetic results are not evidence for those properties.

## Conclusion

`pass` — QA-001 through QA-006 satisfy the approved QA plan. The R-2 Git system/global configuration inheritance issue is fixed and covered by hostile-parent replacement assertions. The candidate identifier and full gate evidence are referenced through `HANDOVER.md`.
