---
task_id: "TASK-0051"
change_class: "product"
status: approved
qa_agent: "qa-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T13:32:36Z"
revision: 2
implementation_reviewed_at: "2026-08-01T13:51:40Z"
expectation_changed: true
expectation_change_approved_by: "main-agent-sol-high"
---

# TASK-0051 QA PLAN

## QA scope

TASK.md の `Planning input packet` だけを期待値正本とする。candidate の
`tools/dev-agent-harness/internal/connectsession/` と許可される README 差分を独立確認し、
既存 `proxyca`、`brokerhttp`、Exchange、Policy、credential/transport の意味や API を変更していないことも確認する。

実 listener bind、OS peer identity、CA file/trust、real gh/OpenAI proxy/VPS は安全な実環境と cleanup が
未定義のため live-e2e を blocked とする。`net.Pipe` の PASS は実 client、trust、配置または外部接続を保証しない。

注入 Handler は trusted `brokerhttp.Handler` 相当で、request context の cancellation/deadline を協調して
return する依存契約とする。任意・悪性の non-returning Go callback を Session が外側から強制終了することは
対象外かつ無効な依存であり、QA はそれを要求又は PASS と主張しない。

## Cases

| Case ID | 対象AC | 確認内容 | Mode | Evidence |
|---|---|---|---|---|
| QA-001 | AC-1 | `New` が non-nil Authority interface と、request context を協調して honor する trusted `brokerhttp.Handler` 相当の Handler だけを受理し、immutable Session を返すことを確認する。invalid Rules、nil/typed-nil conn/context、zero/corrupt Session が panic せず fixed non-leak error となり、Authority/Handler 以外の long-lived state を保持しないことを source/test で監査する。任意の non-returning callback 強制停止は依存契約外である。 | evidence-review | candidate source/test、HANDOVER、DEV check evidence |
| QA-002 | AC-2 | 16 KiB・deadline 付きの HTTP/1.1 CONNECT 一件だけを読み、exact two host `:443` の RequestURI/Host 一致だけを通すことを確認する。outer header は単一の visible-ASCII `User-Agent`（256 bytes以下）と、sole value が case-insensitive `keep-alive` の単一 `Proxy-Connection` だけを任意で無視できる。duplicate、invalid、overlong User-Agent、duplicate/non-`keep-alive` Proxy-Connection、method/version/authority、body/framing、upgrade/auth、その他の unknown/control header、early/pipelined byte は Authority/Handler 前に fixed empty 403 へ fail closed することを確認する。 | focused-rerun | bundled race test |
| QA-003 | AC-3 | valid CONNECT で target leaf を response 前に一回だけ発行し、fixed empty 200 後に TLS 1.2+、exact SNI、ALPN `http/1.1`、issued leaf だけを使うことを確認する。Issue/SNI/ALPN/TLS failure が retry、fallback、diagnostic HTTP response なしで close することを確認する。 | focused-rerun | bundled race test |
| QA-004 | AC-4 | TLS 後の一 HTTP/1.1 request が caller context とその deadline/cancellation を変更せず real trusted `brokerhttp.Handler` へ同期一回だけ渡ることを確認する。context-only Subject resolution と Exchange single call、outer CONNECT/RemoteAddr/SNI/header 由来の identity 禁止、keep-alive/pipeline/HTTP2/retry/fallback なしを確認する。 | focused-rerun | bundled race test |
| QA-005 | AC-5 | trusted cooperative Handler へ同じ deadline context が伝播し、CONNECT/TLS/HTTP の connection I/O stall、cancel、EOF、panic、dependency/connection failure と cooperative Handler return の全 path で Session 自身の goroutine と conn を cleanup することを確認する。任意の non-returning callback を強制停止することは要求しない。public error/Format/wire response の fixed non-leak と concurrent session の certificate/context/request/response/buffer/deadline isolation を確認する。 | focused-rerun | bundled race test |
| QA-006 | AC-6 | hermetic test が trusted cooperative `brokerhttp.Handler` 相当への context/deadline propagation、two-host CONNECT→TLS→real Handler response、403/200、strict framing/header/early-byte rejection、single Issue、SNI/ALPN/TLS failure、context inheritance/non-self-asserted identity、single Handler、connection I/O stall/cancel/panic/close、Session-owned goroutine/conn cleanup、concurrent isolation を失敗検出できることを監査する。arbitrary non-returning callback の forced termination を要求又は主張しない。HANDOVER から candidate-bound root/harness checks、distcheck、README変更時 lint、diff scope/size を監査する。 | evidence-review | candidate source/test、HANDOVER、DEV command/result |
| QA-007 | 対象外 / AC-6の制限 | real listener bind/accept、OS peer identity、CA private file lifecycle/OS trust、real client と gh/OpenAI proxy、network namespace/VPS を実環境で確認する。 | live-e2e — blocked | 実環境、authority、real secret/trust、safe cleanup、listener/identity/VPS path が未定義。この blocked は hermetic PASS で代替しない。 |

## Execution rule

QA-002〜005 は同じ一回の focused-rerun 観測に束ねる。QA は
`tools/dev-agent-harness` を cwd として、candidate に対し次だけを一回実行する。

```sh
GOCACHE=$PWD/.build/go-cache go test -count=1 -race ./internal/connectsession
```

QA は root/harness `make check`、`make distcheck`、lint、追加 test 又は rerun を実行しない。QA-001 と
QA-006 は candidate source/test、HANDOVER、DEV command/result の独立 evidence audit だけを行う。

## Result criteria

各 case を planning input packet と candidate-bound evidence に照らして記録する。focused-rerun は command、exit
status、test coverage と failure-detection evidence を記録する。non-cooperative/non-returning Handler は invalid dependency と
分類し、Session の forced termination requirement には転換しない。失敗は実装不具合と決めつけず、candidate、environment、
dependency、requirement、evidence のいずれかに分類する。QA-007 は実行可能になるまで blocked のままとする。

## 実装後の再確認

- [x] candidate source/test、HANDOVER、DEV check evidence を独立確認した。
- [x] 指定 race test を candidate で一回実行した。
- [x] 変更が許可 path と差分 1,000 行以内に収まり、既存 dependency/package の意味を変えていないことを確認した。
- [x] trusted cooperative Handler への context/deadline propagation、connection I/O stall、panic、Session-owned goroutine/conn cleanup を確認し、arbitrary non-returning callback の forced termination を要求していない。
- [x] real listener/identity/CA trust/client/VPS live-e2e を PASS に置換せず、期待値又は scope を変更していないことを確認した。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | 2026-08-01 | qa-agent-terra-medium | planning input packet に基づく独立QA計画 | `approved` |
| 2 | 2026-08-01 | qa-agent-terra-medium | trusted cooperative Handler、context/deadline伝播、Session-owned cleanup と non-returning callback の依存境界を明確化 | `approved` |
