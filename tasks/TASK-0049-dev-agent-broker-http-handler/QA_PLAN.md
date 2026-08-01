---
task_id: "TASK-0049"
change_class: "product"
status: approved
qa_agent: "qa-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T11:31:40Z"
revision: 1
implementation_reviewed_at: ""
expectation_changed: false
expectation_change_approved_by: ""
---

# QA_PLAN: TASK-0049

## QA scope

TASK.mdの`Planning input packet`だけを期待値正本として、candidateの
`tools/dev-agent-harness/internal/brokerhttp/`と許可されるREADME差分を独立に確認する。
Handlerが既存broker Exchange、Transaction、Forwarder、Policy、Registry、credential/transportの
意味を変更又は再実装していないことも確認対象にする。

実TLS/listener、production identity resolver、実gh/OpenAI client/provider、credential、DNS/system
trust、Agent network namespaceはlive E2E対象だが、安全な実環境とcleanupが定義されていない。このため
blockedとし、hermetic結果で実配置又は実provider受理を主張しない。

## Cases

| Case ID | 対象AC | 確認内容 | Mode | Evidence |
|---|---|---|---|---|
| QA-001 | AC-1 | `New`がExchange/resolverのnil又はtyped nil、request body上限の1 byte〜1 MiB外を固定non-leak errorで拒否し、有効Rulesからimmutable Handlerを返すことを確認する。nil/zero Handlerはpanicやdependency/input detailを出さず空403となることを確認する。 | evidence-review | candidate source/test、HANDOVER、DEV check証跡 |
| QA-002 | AC-2 | HTTP/1.1 origin-formだけを受理し、nil URL、absolute/opaque/userinfo、query/fragment/raw又はpercent-encoded path、空/過長Host、HTTP/1.0/2、CONNECT/upgrade、transfer coding/trailer、未知又は不一致Content-LengthをExchange前に拒否することを確認する。caller requestのHost/path/header/bodyを変更・保持しないことを確認する。 | focused-rerun | `tools/dev-agent-harness`で一回だけ実行するrace test |
| QA-003 | AC-3 | protocol検査後にcontextだけを渡すtrusted resolverを一回呼び、method、`https://`+Host+canonical path、0又は1値Content-Type、Authorization全値、上限内bodyの独立copyとSubjectをExchangeへ同期一回だけ渡すことを確認する。RemoteAddr/Forwarded/Agent identity headerの自己申告を使わず、default Exchange/retry/redirectを導入しないことを確認する。 | focused-rerun | `tools/dev-agent-harness`で一回だけ実行するrace test |
| QA-004 | AC-4 | Exchange成功時だけ2xx、空又は正規`application/json`、独立bodyを返し、response headerを必要時Content-Type、固定Cache-Control/X-Content-Type-Options、正確なContent-Lengthだけに限定することを確認する。success outputがExchange/caller bufferとaliasせず、連続又は並行requestでstateを共有しないことを確認する。 | focused-rerun | `tools/dev-agent-harness`で一回だけ実行するrace test |
| QA-005 | AC-5 | mapping/body read/resolver/Exchangeの各失敗をempty body、Content-Length 0、no-store/nosniffだけの同一403へ畳み、opaque handle、credential、URL/path、Host、request/response body、subject、provider、dependency errorをresponse/error/formatへ出さないことを確認する。Exchangeの複数回呼出がないことを確認する。 | focused-rerun | `tools/dev-agent-harness`で一回だけ実行するrace test |
| QA-006 | AC-6 | hermetic race testがfake resolver/Exchange＋`httptest`、real Exchange＋fake upstream dependencyで、両provider、canonical HTTP mapping、protocol/framing/content/header/body拒否、identity非自己申告、resolver/Exchange単回、empty deny、success header allowlist、input/output copy、並行隔離、fixed non-leakを実際に失敗検出できることを確認する。source/test/HANDOVERからDEVがharness `make check`/`make distcheck`、README変更時root `make lint-docs`、candidate launcher root `make check`を実行済みであること、許可pathとbase...candidateの追加＋削除が1,000行以下であることを監査する。QAは包括checkを再実行しない。 | evidence-review | candidate source/test、HANDOVER、DEV command/result |
| QA-007 | 対象外 / AC-6の制限 | 実TLS client/listener、production identity resolver、実gh/OpenAI SDK/provider、credential、DNS/system trust、Agent network namespaceを通じた実配置を確認する。 | live-e2e — blocked | 実環境、権限、credentialの安全な取得・cleanup、production identity/resolver経路が未定義。このblockedは他caseのPASSで代替しない。 |

## Execution rule

focused-rerunのQA-002、QA-003、QA-004、QA-005は同じ一回のrace test観測に束ねる。QAは
`tools/dev-agent-harness`をcwdとして、次だけを一回実行する。

```sh
GOCACHE=$PWD/.build/go-cache go test -count=1 -race ./internal/brokerhttp
```

それ以外はcandidateのsource/test、HANDOVER、DEV command/resultを独立監査する。root `make check`、harness
`make check`、harness `make distcheck`、追加processはQAでは実行しない。

## Result criteria

各caseについてcandidate source/testとHANDOVERの事実をPlanning input packetに照らして記録し、focused-rerunはcommandと結果を記録する。失敗は実装不具合と決めつけず、QAガイドラインに従ってcandidate、環境、依存、要件又は証跡のいずれかへ分類する。QA-007は実施可能になるまでblockedのままとする。

## 実装後の再確認

- [ ] candidateのsource/test、HANDOVER、DEV check証跡を独立に確認した。
- [ ] 指定race testをcandidateで一回だけ実行した。
- [ ] live E2E blockedをPASSに置換せず、期待結果又は範囲を変更していないことを確認した。
