---
task_id: "TASK-0067"
change_class: product
status: approved
qa_agent: qa-agent-terra-medium
approved_by: main-agent-sol-high
approved_at: "2026-08-02T03:49:42Z"
revision: 1
implementation_reviewed_at: "2026-08-02T04:15:00Z"
expectation_changed: false
expectation_change_approved_by: ""
---

# QA plan — TASK-0067

## QA independence and candidate boundary

This plan is derived solely from TASK-0067's `Planning input packet`.  QA evaluates one DEV-produced `candidate_commit` identified in `HANDOVER.md`; it does not accept a moving worktree as evidence.  QA remains independent of PLAN and DEV reasoning.

The candidate gate invariants are:

- the candidate contains only the three permitted paths: the new bridge source, its test, and `tools/dev-agent-harness/README.md`;
- no launcher, command, Makefile, config, dependency, schema, generated/live-state, or secret change is present;
- the implementation keeps the Unix peer-bound egress service as the authorization authority and adds no HTTP/TLS/CONNECT parsing or fallback route;
- QA refers to the candidate recorded in HANDOVER and audits the exact command/result recorded there.  QA does not request raw logs or artifact digests.

## Candidate test execution

QA's default independent execution is run from `tools/dev-agent-harness` on the fixed candidate:

```sh
GOCACHE=$PWD/.build/go-cache go test -count=1 -race ./internal/proxybridge
```

The package tests must be hermetic.  They use fake listener and dialer seams plus `net.Pipe`; no real egress service, external network, credentials, privileged namespace, or host socket permission is required.  A test timeout appropriate to the existing Go test harness may be used only to turn a deadlock/leak into a deterministic failure.

The root/harness full checks are not rerun by QA.  QA performs evidence review of the corresponding HANDOVER command/result (`make check`, harness distcheck where recorded, root candidate-gate `make check`, and `git diff --check`) and verifies they refer to this same candidate.  Their absence, failure, or candidate mismatch is a gate failure; a passing record is not treated as proof of the bridge behavior.

## Acceptance-case matrix

| Case | Mode | Independent verification | Pass condition |
|---|---|---|---|
| AC-1 — fixed loopback listener and canonical endpoint | focused-rerun | The race command covers constructor/start tests with a fake listener.  Assert exactly one `tcp4` listen request for exact `127.0.0.1:0`; the returned URL is exactly canonical `http://127.0.0.1:<OS-assigned-port>`.  Table-driven invalid inputs cover wildcard, IPv6, non-loopback, fixed/invalid port, nil listener, and typed-nil listener. | Invalid rules fail before listen with a fixed/sanitized rejection; an invalid returned listener/address is closed and fixed-rejected before Serve/accept; no caller/CLI/environment-selected bind value is accepted; no second listen occurs. |
| AC-2 — one fixed Unix dial per accepted client | focused-rerun | With a fake listener/dialer, accept a client and inspect dial calls.  Verify one timeout-bounded `unix` dial to the trusted absolute clean path.  Inject a dial failure and verify the client closes, receives no forwarded bytes, and no retry/alternate network/path occurs. | Exactly one allowed dial on success.  Dial failure is connection-local and sanitized: no lower error, address, or path escapes; no payload forwarding or fallback happens. |
| AC-3 — raw bidirectional streaming, half-close, cancellation, cleanup | focused-rerun | Use paired `net.Pipe` endpoints for client/upstream to prove byte-for-byte forwarding in both directions without HTTP, CONNECT, or TLS interpretation.  Exercise client EOF and upstream EOF separately and observe the opposite write-half close while the reverse direction can finish.  Exercise context cancellation and one-direction copy failure, then assert both endpoints close and Serve waits for all per-connection goroutines. | Both raw streams and both half-close directions complete without mutation or retained payload; cancellation/failure closes both sides and leaves no blocked connection goroutine. |
| AC-4 — bounded concurrency and shutdown/drain | focused-rerun | Use a blocking fake dialer/listener and `net.Pipe` clients.  At the 1–64 configured cap, assert an additional accepted client cannot start a Unix dial until capacity is released.  Inject an unexpected accept failure and assert new acceptance stops, active sessions are cancelled/drained, and the returned server error is fixed/sanitized.  Cancel the parent context with active streams and assert listener and connections close and Serve returns normally. | Concurrency never exceeds the configured bound; no extra dial begins while saturated; accept and parent-cancel paths drain all active work with the required fixed result and no hang/race. |
| AC-5 — bounded scope, documentation, and candidate checks | evidence-review | Inspect the candidate diff and README boundary statement against the permitted paths and prohibited behavior.  Audit HANDOVER's recorded command/results for the focused race test and the required root/harness checks, including `git diff --check`, bound to `candidate_commit`. | Diff is limited to the three allowed paths, contains the expected package/test/README material, and has no prohibited surface.  All recorded gate commands pass for that exact candidate. |

## Required focused-test failure detection

The focused suite is adequate only if deliberate mutations to each property would fail at least one case above: changing the listen network/address or listening twice; dialing a second time, another network/path, or forwarding after dial failure; replacing raw copies with parsing/buffering, omitting either half-close, or not closing on cancellation/failure; and starting a dial above the concurrency bound or returning before drain.  QA records the existing test names and their relevant assertions when reviewing the candidate; it does not require mutation commits or raw output.

## Environment-dependent cases — live-e2e, blocked/not run

The following are deliberately outside candidate PASS and are recorded as `live-e2e: blocked/not run` unless an approved environment and safe cleanup procedure are supplied after merge:

- real OS network namespace and actual loopback reachability/isolation;
- real Unix socket ownership, permission, and peer-UID behavior;
- real Git, gh, or OpenAI traffic and credentials;
- systemd activation/deployment and VPS behavior.

Hermetic fake listener/dialer and `net.Pipe` results must not be represented as evidence for these environment properties.  Their blocked/not-run status does not prevent a candidate PASS for AC-1 through AC-5.

## Result rules

Mark an AC PASS only when its assigned mode passes on the candidate fixed in HANDOVER.  Classify failures before attribution: a reproducible focused-test failure or invariant violation is candidate evidence; a missing/mismatched HANDOVER record is evidence/gate failure; unavailable live environment is blocked/not-run, not a DEV defect.  Report the case ID, mode, command or reviewed HANDOVER result, and concise observed outcome.
