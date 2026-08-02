---
task_id: "TASK-0069"
change_class: product
status: approved
qa_agent: qa-agent-terra-medium
approved_by: "main-agent-sol-high"
approved_at: "2026-08-02T05:08:18Z"
revision: 2
implementation_reviewed_at: ""
expectation_changed: false
expectation_change_approved_by: ""
---

# QA plan — TASK-0069

## QA independence and candidate boundary

This plan is derived solely from TASK-0069's `Planning input packet`. QA evaluates one DEV-produced `candidate_commit` identified in `HANDOVER.md`; it neither accepts a moving worktree nor relies on PLAN or DEV reasoning.

The candidate gate invariants are:

- the candidate changes only the seven permitted paths: `internal/launchsession/session.go`, `internal/launchsession/session_test.go`, `internal/command/command.go`, `internal/command/command_test.go`, `cmd/dev-agent-launcher/main.go`, `Makefile.in`, and `README.md` under `tools/dev-agent-harness`;
- its scale is approximately 850–1,200 changed lines, with no real credential or external communication, and no change to existing egress/control/helper/policy implementation, config, dependency, Schema, deployment/provisioning, generated/live-state, or Kakesu runtime;
- launcher inputs cannot select a runtime socket, helper, provider, operation, model, proxy endpoint, or other routing/auth value; fixed link-time paths and the existing strict components remain their authority;
- QA audits the candidate hash in HANDOVER and its recorded candidate gate results. QA does not request raw logs, digests, duplicate candidate identifiers, or a rerun of all checks.

## Candidate test execution

QA independently runs this one focused race suite from `tools/dev-agent-harness` on the fixed candidate:

```sh
GOCACHE=$PWD/.build/go-cache go test -count=1 -race ./internal/launchsession ./internal/command
```

The suite must be hermetic and failure-detecting. It uses controlled child, control-client, bridge, filesystem, and cancellation seams; it never uses real GitHub/OpenAI/Codex credentials, `git`, `gh`, SDKs, DNS, TLS, external HTTP, systemd, or host permission assumptions. It must use bounded synchronization only to make cleanup, child wait, drain, and non-leak failures deterministic.

QA does not rerun harness/root full checks, distcheck, or installation staging. It evidence-reviews the candidate-bound HANDOVER results for this focused race test, harness `make check`, harness `make distcheck`, install staging that proves launcher/helper link-time paths, root candidate-gate `make check`, and `git diff --check`. Missing, failed, or candidate-mismatched records fail that evidence gate; passing records do not prove launcher behavior.

## Acceptance-case matrix

| Case | Mode | Independent verification | Pass condition |
|---|---|---|---|
| QA-001 → AC-1 — exact CLI, pre-dial rejection, exec/stdio | focused-rerun | Table-drive `--help`, `--version`, and exact `run --repository owner/repo -- COMMAND [ARG...]`. Cover empty command; missing, unknown, duplicate, and reordered options; added launcher options; NUL; and noncanonical/invalid repository values. With a controlled child seam, inspect argv, shell non-use, and stdio wiring. Assert invalid input cannot reach control dial or child start. | Only the exact operational form and help/version are accepted. Repository is lowercase canonical `owner/repo`; child gets unchanged argument boundaries and transparent stdio through direct exec. Every malformed form fails as usage before dial/start. |
| QA-002 → AC-2 — fixed-path, ordered partial initialization and revoke | focused-rerun | Inject clean/unclean/relative paths and setup failures at each seam: fixed socket/helper validation, public-CA issue, GitHub REST issue, OpenAI issue, CA write, and bridge start. Record calls and issued handles without exposing their values. Assert each allowed issue occurs once in order, subsequent initialization/child start stops at first failure, and only previously issued handles receive one revoke request. | Only absolute clean link-time fixed socket/helper paths are used; no runtime routing/provider/operation/model/endpoint input is accepted. Public CA, GitHub REST, and OpenAI handles are issued exactly once on success. Each partial-init path has no later initialization/child, and revokes precisely the handles actually issued once. |
| QA-003 → AC-3 — literal temporary CA and bridge lifecycle | focused-rerun | Use controlled filesystem and bridge seams to assert one IPv4 loopback bridge endpoint; a fresh literal `/tmp` child directory with mode `0700`; and one `0600` regular CA file. Exercise existing path, symlink, non-regular path, parent `TMPDIR`, CA write/bridge/child-start failures, unexpected bridge failure while the child is running, normal/nonzero exit, and cancellation. Observe child stop/wait when bridge fails, stop-new-accept, drain/wait, one directory removal attempt, and revoke calls. | No symlink, existing path, non-`/tmp` parent, or repeated CA write is used. Every started bridge stops accepting and drains before return; unexpected bridge failure stops and waits for the child; the CA directory is removed once without retry; all issued API handles are each revoked once on every stated exit path. |
| QA-004 → AC-4 — allowlisted environment and deterministic Git command scope | focused-rerun | Start with hostile parent values covering `GH_*`, `OPENAI_*`, proxy/CA, Git config/credential, SSH, loader/runtime injection, arbitrary variables, and permitted `HOME`, `PATH`, `TERM`, `LANG`/`LC_*`, `CODEX_HOME`. Capture the child environment/config invocation without reading credential files. Assert Opaque GitHub/OpenAI handles; session-only `HTTP_PROXY`/`HTTPS_PROXY` and lowercase pair; `SSL_CERT_FILE`/`CURL_CA_BUNDLE`/`REQUESTS_CA_BUNDLE`/`NODE_EXTRA_CA_CERTS`/`GIT_SSL_CAINFO`; and `GIT_TERMINAL_PROMPT=0` are each set once, while parent API/proxy/Git/etc. values are absent. Inspect command-scope Git config ordering/content. | The result is an allowlist reconstruction only; Codex locations may pass through but launcher neither reads nor expands secret contents. Git prompt is disabled; helper entries reset with an empty value then use only the absolute fixed helper; GitHub uses `useHttpPath=true`; proxy and CA are fixed; no duplicate or inherited dangerous value remains. |
| QA-005 → AC-5 — child exit, cancel, fixed diagnostics, and non-leak | focused-rerun | Simulate normal zero, ordinary nonzero, signal, child start, initialization, cleanup, cancellation, and revoke-unknown/expired failures. Assert cancel signals/stops and waits for child, then bridge cleanup completes before launcher return. Capture stderr/error strings under adversarial handles, repository, paths, environment, command, and lower errors; inspect no live child/bridge/file resource remains. | Zero is retained for normal child exit and ordinary child nonzero is retained; signal/start/init/cleanup failures collapse to the fixed nonzero outcome. Diagnostics are usage or fixed `session failed` only. They contain no handle, repository, path, environment, command, or lower error. Revoke failure does not replace a child result; no child, bridge activity, or CA directory leaks. |
| QA-006 → AC-6 — scope, build/install wiring, and recorded candidate gates | evidence-review | Inspect the fixed candidate diff, README guarantee-limit statement, and scale against the seven permitted paths and exclusions. Audit the candidate-bound HANDOVER command/results named above, including install staging's launcher/helper link-time path evidence. | The diff stays in scope; docs do not claim the launcher/proxy environment is OS network isolation; no secret/prohibited surface appears. Focused, harness/root, distcheck, install-staging, and diff-check records pass for the same candidate. |

## Required focused-test failure detection

The focused suite is adequate only if a deliberate mutation of each property would fail at least one case: accepting a malformed/reordered CLI or dialing/starting a child before usage rejection; using a caller-controlled or unclean fixed path, issuing a component twice/out of order, continuing after partial initialization, or revoking the wrong count; using `TMPDIR`, an existing/symlink/non-regular CA path, skipping bridge drain, cleanup, or a lifecycle exit path; inheriting a hostile environment, omitting the helper reset, using non-fixed Git config, or duplicating session values; and leaking a child/bridge/file, changing exit-code behavior, or revealing a protected value through diagnostics. QA records relevant test names/assertions on the candidate; it needs neither mutation commits nor raw output.

## Environment-dependent cases — live-e2e, blocked/not run

The following are `live-e2e: blocked/not run` unless an approved environment and safe cleanup procedure are supplied after merge:

- real GitHub REST/OpenAI Responses/Git Smart HTTP authentication and real Codex credential discovery;
- actual `git`, `gh`, Codex, or OpenAI SDK proxy/CA/credential behavior;
- real OS default-deny firewall or network namespace enforcement, loopback isolation, Unix-socket ownership/peer UID behavior, or loader/process containment;
- systemd/VPS deployment, restart, rollback, and cleanup behavior.

Hermetic results must not be represented as proof of these properties or of network isolation. Their blocked/not-run status does not prevent a candidate PASS for QA-001 through QA-006.

## Result rules

Mark an AC PASS only when its assigned mode passes on the candidate fixed in HANDOVER. Classify before attribution: a reproducible focused-test failure or invariant violation is candidate evidence; a missing, failing, or candidate-mismatched HANDOVER record is an evidence/gate failure; unavailable approved live infrastructure is blocked/not-run, not a DEV defect. Report the case ID, mode, command or reviewed HANDOVER result, and concise observed outcome.

## Implementation follow-up

- [ ] Verify the implementation and candidate evidence against this plan without changing the plan's expectations.
- [ ] If a required test cannot observe its assigned property, report an evidence gap; do not substitute a full-check rerun or a format-only artifact.
- [ ] Escalate any needed change to acceptance scope or execution mode to Main for approval before revising this plan.

## Revision history

| Revision | Date | Author | Change | Main approval |
|---:|---|---|---|---|
| 1 | 2026-08-02 | | Placeholder created | `pending` |
| 2 | 2026-08-02 | qa-agent-terra-medium | Independent QA plan from TASK Planning input packet | `approved 2026-08-02T05:08:18Z` |
