---
task_id: "TASK-0068"
change_class: product
status: approved
qa_agent: qa-agent-terra-medium
approved_by: main-agent-sol-high
approved_at: "2026-08-02T04:42:25Z"
revision: 2
implementation_reviewed_at: "2026-08-02T04:53:53Z"
expectation_changed: true
expectation_change_approved_by: main-agent-sol-high
---

# QA plan — TASK-0068

## QA independence and candidate boundary

This plan is derived solely from TASK-0068's `Planning input packet`. QA evaluates one DEV-produced `candidate_commit` identified in `HANDOVER.md`; it does not accept a moving worktree as evidence and does not rely on PLAN or DEV reasoning.

The candidate gate invariants are:

- the candidate changes only the six permitted paths: `internal/controlclient/client.go`, `internal/controlclient/client_test.go`, `internal/capabilitycontrol/control.go`, `internal/capabilitycontrol/control_test.go`, `internal/egressservice/service_test.go`, and `README.md` under `tools/dev-agent-harness`;
- its scale is approximately 750–1,050 changed lines and it contains no launcher, environment/config, Makefile, dependency, Schema, Kakesu runtime, generated/live-state, or real token/key change;
- the only new client operations are explicit GitHub REST-read and OpenAI Responses-text issue operations over the existing fixed Unix control socket. No agent-controlled provider, operation, body, model, repository, path, runtime socket, retry, fallback, TCP, cache, or persistence surface is introduced;
- API scopes receive fixed 16-use accounting while TTL and all existing subject/workspace/provider/repository/operation/destination bindings remain intact; `github-git-read` remains one use;
- QA audits the candidate hash in HANDOVER and its recorded root/harness full-check command/results. QA does not rerun those full checks and does not require raw logs, digests, or duplicated candidate hashes.

## Candidate test execution

QA's default independent execution is run from `tools/dev-agent-harness` on the fixed candidate:

```sh
GOCACHE=$PWD/.build/go-cache go test -count=1 -race ./internal/controlclient ./internal/capabilitycontrol ./internal/gitcredential ./internal/egressservice
```

The focused suite must be hermetic: it uses controlled Unix-socket peers and registry/control fixtures, never real credentials, `gh`, an OpenAI SDK, external HTTP/DNS/TLS, systemd, or VPS state. It must cover malformed peers and failure seams without exposing transport internals to diagnostics.

The root/harness full checks are not rerun by QA. QA evidence-reviews the HANDOVER command/result for focused race testing, harness `make check`/distcheck when recorded, root candidate-gate `make check`, and `git diff --check`, confirming that the record is bound to the fixed `candidate_commit`. Missing, failing, or candidate-mismatched records fail that gate; a passing record alone does not prove capability behavior.

## Acceptance-case matrix

| Case | Mode | Independent verification | Pass condition |
|---|---|---|---|
| QA-001 → AC-1 — exact GitHub REST/OpenAI issue wire and transport lifetime | focused-rerun | Use a controlled fixed Unix peer to capture each public explicit issue operation separately. Assert GitHub REST sends exactly one request to `POST /v1/capabilities` with body `{"provider":"github","repository":"owner/repo"}` and OpenAI sends exactly `{"provider":"openai"}`. Inspect dial/request lifecycle seams and invalid socket/repository inputs. | Each valid operation makes one dial and one request only, applies a deadline, then closes. Only an absolute clean fixed Unix socket and canonical `owner/repo` are accepted where applicable; no caller input can select provider, operation, body, model, or path. |
| QA-002 → AC-2 — strict response matrix and non-leak | focused-rerun | Table-drive peer responses through every acceptance boundary: only bounded `200 application/json`, canonical Content-Length/header order, one exact JSON `{"handle":"cap_..."}`, and immediate EOF are success. Exercise non-200, bad/missing/duplicate/reordered headers or length, oversized/truncated/extra body, malformed/noncanonical JSON or handle, extra byte, and dial/deadline/read/write/close errors. | Every invalid response and every transport failure returns nil/empty plus the fixed error, with no retry or fallback. Error text/diagnostics contain no socket, repository, handle, wire content, or lower-level error. |
| QA-003 → AC-3 — API scope budget and atomic 16-use boundary | focused-rerun | Issue peer-bound GitHub REST and OpenAI capabilities through controller/registry fixtures, then consume a valid scope 16 times. Include concurrent consumption where the existing atomic registry path is exercised. | Each valid consume atomically decrements by one; the sixteenth has remaining `0` and the seventeenth is rejected. The fixed budget is not caller-configurable and applies only to the intended API scopes. |
| QA-004 → AC-3/AC-4 — mismatch non-consumption and scope isolation | focused-rerun | Before and after mismatched consume attempts, inspect the remaining budget. Cover mismatched subject, workspace, provider, repository, operation, and destination, plus Git-read/push/write/other-provider/repository/host attempts against API handles. | Every mismatch is rejected without consuming use count. API handles cannot authorize Git read, push, write, or a different provider/repository/host; existing revoke, expiry, and epoch behavior remains effective. |
| QA-005 → AC-4 — Git-read single-use regression and existing protocol compatibility | focused-rerun | Directly run the included `./internal/gitcredential` and `./internal/egressservice` tests with the focused command. The former verifies Git credential helper get/erase and `github-git-read` issue/consume twice, with the second use rejected. The latter directly verifies shared-Registry integration: GitHub REST/OpenAI API capability consumption succeeds 16 times, the seventeenth is rejected, and revoke remains effective. CONNECT/server control wire is outside the permitted diff; QA verifies it only through the candidate-bound HANDOVER harness-check record and adds no separate full suite. | Git-read remains single-use; API scopes retain the existing 16-success/17th-rejection and revoke behavior. Helper get/erase remains unchanged; the HANDOVER audit shows no regression in the out-of-scope CONNECT/server control wire. |
| QA-006 → AC-5 — bounded scope, documentation, and candidate gates | evidence-review | Inspect the candidate diff and README statement against the six permitted paths, scope exclusions, and expected scale. Audit HANDOVER command/results as described above. | No prohibited product surface or secret is present; README records the launcher-prelude boundary. The recorded focused, harness/root, and diff checks pass for this candidate. |

## Required focused-test failure detection

The focused suite is adequate only if a deliberate mutation of each property would fail at least one case: changing GitHub/OpenAI request path, method, JSON body, socket/repository validation, dial count, deadline, or close; accepting any noncanonical status/header/length/body/handle/EOF condition; returning lower-level diagnostics or retrying; making 17 API consumptions succeed, decrementing non-atomically, or consuming a mismatch; and changing Git-read from one use or altering existing helper/CONNECT/control semantics. QA records the relevant test names and assertions on the candidate. It does not require mutation commits, raw output, or artifact digests.

## Environment-dependent cases — live-e2e, blocked/not run

The following are `live-e2e: blocked/not run` unless an approved environment and safe cleanup procedure are supplied after merge:

- real GitHub or OpenAI credentials, GitHub REST/Responses calls, `gh`, or OpenAI SDK behavior;
- real Unix socket ownership/peer identity and host permission behavior;
- systemd activation/deployment, separate UID behavior, or VPS operation.

Hermetic focused results must not be presented as evidence for these environment-dependent properties. Their blocked/not-run status does not prevent a candidate PASS for QA-001 through QA-006.

## Result rules

Mark an AC PASS only when its assigned mode passes on the candidate fixed in HANDOVER. Classify before attributing: a reproducible focused-test failure or invariant violation is candidate evidence; a missing, failing, or candidate-mismatched HANDOVER record is an evidence/gate failure; unavailable real credentials or live infrastructure is blocked/not-run, not a DEV defect. Report the case ID, mode, command or reviewed HANDOVER result, and concise observed outcome.
