---
task_id: "TASK-0044"
change_class: product
status: approved
qa_agent: qa-agent-terra-medium
qa_role: independent-qa
approved_by: main-agent-sol-high
approved_at: "2026-08-01T08:17:42Z"
revision: 2
implementation_reviewed_at: "2026-08-01T08:26:30Z"
expectation_changed: true
expectation_change_approved_by: main-agent-sol-high
---

# QA Plan: TASK-0044

## Independence and scope

- Basis: the `Planning input packet` in `TASK.md` only.
- Candidate under test: the single `candidate_commit` named in the DEV handover. Do not combine base, working-tree, or later evidence; candidate hash/tree/digest remain canonical in that handover and are not transcribed here.
- This plan verifies ordering and fail-fast behavior. It does not add a permanent test, script, Make rule, digest, or requirement, and it does not alter repository files.
- The normal root `make check` is **evidence-review** only: QA audits the DEV candidate-bound result and its command log rather than rerunning the comprehensive check.

## Preconditions

1. Record the base and `candidate_commit` from the approved handover, and run all candidate checks from a clean checkout of that exact commit.
2. Confirm the candidate diff is limited to root `Makefile`, with at most 10 added plus deleted lines in total. Treat any other path or a changed public target/recipe as an out-of-scope failure requiring Main classification.
3. Use the already-provisioned local tool/dependency state. Do not run an installer, package manager install, dependency update, network-capable bootstrap, or generated-artifact update.

## Cases

| Case | Mode | Procedure and pass evidence |
|---|---|---|
| QA-1: candidate diff and target preservation | evidence-review | Independently inspect `base...candidate_commit` for the root Makefile. Confirm the change only rearranges `check` orchestration; it neither changes the commands/meaning of `lint-docs`, `lint`, `build`, `test`, or subtargets nor adds/removes public targets, rules, scripts, tests, glossary entries, or documents. Record the candidate diff identity and line counts. |
| QA-2: dry-run command-set and order | focused-rerun | Capture `make -n check` at the base and at the exact candidate. Compare the emitted command occurrences as a multiset (each existing check command once) and their sequence. The candidate must place `validate-terminology.py`, `pnpm lint:docs`, and the documentation `git diff --check` before core/memory/governance build, test, and lint commands. Confirm viewer-data generation and the final root `git diff --check` remain last. This dry run must not execute commands, install dependencies, access the network, or modify repository files. |
| QA-3: injected first-doc-command failure | focused-rerun | From the exact candidate, use `make -o node_modules/.modules.yaml check UV=false`, capturing stdout/stderr and exit status in a temporary location outside the repository. `-o node_modules/.modules.yaml` (`--old-file`) treats only the existing dependency marker as old/do-not-remake so the invocation cannot install dependencies; `UV=false` makes the first existing UV-backed documentation-lint command fail immediately. Pass only when the invocation is nonzero, its trace shows the `false` command, and no product build or test command appears in the trace. Do not create a lint violation, run any product command, install/update dependencies, or use the network. If the candidate dry-run evidence does not show a UV-backed first docs command, stop this case and report the injection mechanism as unsupported rather than substituting a new command or changing requirements. |
| QA-4: normal comprehensive result | evidence-review | Audit that the candidate launcher created the HANDOVER-bound candidate only after its root `make check` returned success and left candidate bytes unchanged. The launcher does not persist successful stdout/stderr; QA-2 independently proves command ordering and nonduplication, so a complete log is not required and the comprehensive check is not rerun. Independently run `git diff --check` against the candidate worktree (or audit an equivalent candidate-bound result) and require PASS. |

## Acceptance mapping and failure classification

| Acceptance condition | Evidence cases | Failure classification boundary |
|---|---|---|
| AC-1 | QA-2, QA-4 | Candidate ordering/duplication disagreement is a candidate defect; an unavailable local make/tool environment is environmental until reproduced against the candidate in a ready environment. |
| AC-2 | QA-1, QA-2 | Any changed command set, target semantics, or added artifact is scope/contract divergence, not presumed DEV fault. |
| AC-3 | QA-3 | Nonzero caused by the deliberate `UV=false` fault is expected. Build/test trace after that failure is a candidate fail-fast defect. A different first failure, missing UV override point, or unavailable runner is inconclusive/blocked and must be classified before remediation. |
| AC-4 | QA-1, QA-4 | A failing normal `make check` or `git diff --check` must be attributed from its log (candidate, pre-existing baseline, environment, or evidence defect); do not assume implementation fault. |

## Reporting

For every case, report the DEV handover's canonical candidate reference, command, mode, exit status, relevant ordered trace excerpts, and whether a command was intentionally not run. Report fault-injection output as negative-test evidence only; it is not a normal-check failure. Escalate any mismatch in candidate identity, scope, command set, acceptance meaning, or safe local dependency state to Main for classification.
