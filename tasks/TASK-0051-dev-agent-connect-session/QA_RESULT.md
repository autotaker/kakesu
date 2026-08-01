---
task_id: "TASK-0051"
status: pass
qa_agent: "qa-agent-terra-medium"
decision: pass
tested_at: "2026-08-01T13:45:00Z"
---

# TASK-0051 QA RESULT

## 結果

| ケース ID | モード | 結果 | 独立証跡 |
|---|---|---|---|
| QA-001 | evidence-review | pass | `New` accepts only non-nil dependencies. The injected Handler is evaluated as the revision-2 trusted, cooperative brokerhttp-equivalent dependency; Session retains only Authority/Handler and fixed errors/Format do not disclose dependency detail. Arbitrary non-returning callbacks remain invalid dependencies, not a forced-termination requirement. |
| QA-002 | focused-rerun | pass | The prescribed race test exits 0 (6.701s). Exact 16 KiB CONNECT/Host and only the permitted optional outer headers are accepted; framing/header/authority/early-byte failures are denied before dependencies with fixed empty 403. |
| QA-003 | focused-rerun | pass | The run exits 0. Issue is single and before 200; TLS enforces TLS 1.2+, exact SNI, and http/1.1 ALPN. Issue/SNI/version/ALPN failures close without an added HTTP response. |
| QA-004 | focused-rerun | pass | The run exits 0. The trusted real brokerhttp fixture verifies caller context propagation, context-only identity resolution, single Handler/Exchange use, and concurrent isolation. Source synchronously gives the Handler the derived request context. |
| QA-005 | focused-rerun | pass | Fixed I/O stall, cancellation with a cooperative Handler, panic, and Session-owned watcher/connection cleanup are covered. Response body/header limits, invalid/case-variant header framing, and no-inner-byte failures remain fail closed. No claim is made to kill an arbitrary non-cooperative callback. |
| QA-006 | evidence-review | pass | The prior QA-F-001 gap is resolved: `TestHTTPPhaseDeadlinePropagation` has a cooperative Handler observe a non-zero request deadline, verifies a caller-earlier deadline within tolerance, and rejects a phase deadline later than the five-second cap. The hermetic suite also covers the required protocol, TLS, context, cancellation/stall, cleanup, and response-bound negatives. HANDOVER records candidate-bound root/harness checks, distcheck, lint, and diff-check PASS; scope is the three permitted paths at exactly 1,000 additions/0 deletions. |
| QA-007 | live-e2e | blocked | Real listener bind/accept, OS peer identity, CA file/rotation/trust, real client, gh/OpenAI proxy, network namespace/VPS, and safe cleanup remain outside the approved hermetic environment. |

## 発見事項

- なし。Revision-2 deadline-propagation evidence now covers both caller-earlier and fixed phase-cap cases; the prior QA-F-001 is resolved.
- Approved QA_PLAN に従い、`tools/dev-agent-harness` で `GOCACHE=$PWD/.build/go-cache go test -count=1 -race ./internal/connectsession` を一回だけ実行し、exit 0（6.701s）。root/harness checks、distcheck、lint、追加 test、rerun は実行していない。

## 結論

`pass` — QA-001〜QA-006 は revision-2 planning input and approved QA_PLAN に適合する。QA-007 は live-e2e のまま blocked であり、PASS に置換していない。
