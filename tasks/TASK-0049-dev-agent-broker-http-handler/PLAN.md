---
task_id: "TASK-0049"
change_class: "product"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "luna-xhigh"
approved_dev_profile_reason: "既存Exchange APIを変更せず、新規HTTP変換packageとfake/httptest中心のhermetic testだけに閉じるため。"
approved_dev_profile_risk_signals: []
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T11:31:40Z"
planning_reviewed_by: ""
planning_review_decision: "pending"
planning_reviewed_at: ""
classification_approved_by: ""
classification_approved_at: ""
classification_approval_reason: ""
# safety_contract_version: 2
# safety_contract_planned_paths: []
# safety_contract_generated_paths: []
---

# TASK-0049 PLAN

## AC対応

TASKの条件本文を再掲せず、`planning input packet`のAC-IDに設計を対応させる。

| AC-ID | 設計判断 | 変更パス | 実施順序 | 失敗時の扱い |
|---|---|---|---|---|
| AC-1 | `Rules` は非nil `brokerexchange.Exchange`、trusted resolver、1〜1,048,576 byte のbody上限だけを受ける。`New` はtyped nilを含め固定errorで拒否し、不変のHandlerだけを保持する。nil/zero/破損Handlerの`ServeHTTP`も安全にdenyする。 | `tools/dev-agent-harness/internal/brokerhttp/` | 1 | constructorは固定non-leak error、request処理は空403に畳み、panic、入力、依存詳細を公開しない。 |
| AC-2 | HTTP/1.1かつorigin-formだけを、URLのscheme/host/opaque/userinfo/query/fragment/raw-or-percent-encoded path不在、空でない固定上限内のraw Host、既知Content-Length、transfer coding/trailer/upgrade不在として検査する。Host/pathを正規化・補正・保持しない。 | `tools/dev-agent-harness/internal/brokerhttp/` | 2 | 任意のprotocol/framing不整合はbodyをExchangeへ渡さず、resolver/Exchange未到達の空403にする。 |
| AC-3 | protocol通過後にresolverを`context.Context`だけで一回呼び、得た`Subject`と、新規コピーしたmethod、`https://`+raw Host+canonical path、0又は1個のContent-Type、全Authorization、上限内bodyを一回だけ`Exchange.Do`へ同期送信する。identity header、Forwarded、RemoteAddrは読まず、Exchange/Transaction/Policy検査を複製しない。 | `tools/dev-agent-harness/internal/brokerhttp/` | 3 | body読取、resolver、mapping、Exchangeの全失敗はretry、redirect、default Exchangeを選ばず空403にする。 |
| AC-4 | Writer headerを毎回初期化し、Exchange成功の2xxだけを、空又は正規`application/json`、独立copy body、正確なContent-Length、固定`Cache-Control: no-store`と`X-Content-Type-Options: nosniff`だけで書く。request-local bufferだけを使う。 | `tools/dev-agent-harness/internal/brokerhttp/` | 4 | 非2xx又は縮退response不整合、writer errorでは成功を拡張・再送せず、後続/並行requestへstateを残さない。 |
| AC-5 | protocol、copy、read、resolver、Exchange、response mappingの拒否を同一のempty 403 responseへ正規化する。denyはContent-Length 0と固定no-store/nosniff以外のheader、body、challenge、diagnosticを持たない。 | `tools/dev-agent-harness/internal/brokerhttp/` | 2–4 | opaque handle、credential、URL/path、Host、body、Subject、provider、下位errorをresponse又はformatへ出さない。 |
| AC-6 | package-local fake resolver/Exchangeと`httptest`で入口を検出し、real Exchange + fake credential resolver/RoundTripperで両providerまでのhermetic連鎖を検出する。counterとrace fixtureで単回呼出、copy、header allowlist、固定deny、並行隔離を確認する。 | `tools/dev-agent-harness/internal/brokerhttp/`、`tools/dev-agent-harness/README.md` | 2–5 | fakeのPASSをlistener、TLS、production resolver、実credential/provider/networkの成功根拠にしない。 |

## 責務・境界・不変条件

- HandlerはTLS終端後のHTTP parser境界だけを所有する。trusted identityはcontext-only resolverだけから受け、header、body、URL、RemoteAddrをresolverへ渡さず自己申告から作らない。
- 成功順序は Rules/receiver検証 → HTTP/1.1 origin-form/framing検証 → bounded body独立copy → resolver一回 → request独立copy → `Exchange.Do`一回 → response独立copy/allowlist write である。検査前にresolver/Exchangeへ到達しない。
- `brokerexchange` がPolicy、Authorization/capability消費、credential解決、Forwarder、upstream responseの検査を所有する。HandlerはURLを `https://`+raw Host+canonical path として渡すだけで、provider allowlist、scope、Authorizationの意味、retryを再実装しない。
- Handlerの長寿命stateは検証済み依存と上限だけとし、request/response/body/Subjectを保持しない。bodyは読取時、Exchange入力時、Exchange出力時に所有権を分離し、並行requestに共有mutable stateを置かない。
- responseは成功でもdenyでも先にwriter headerをclearし、明示allowlist以外を残さない。write失敗はAgentへ診断せずretryしない。READMEはこのHTTP入口の範囲とlive E2E未確認境界だけを既存説明へ追記する。

## 変更予定

| パス | 変更内容 |
|---|---|
| `tools/dev-agent-harness/internal/brokerhttp/` | immutable Handler Rules/New、context-only trusted resolver、strict HTTP/1.1 origin-form/framing検査、bounded copy mapping、fixed empty deny、success response allowlist、およびhermetic race testsを追加する。 |
| `tools/dev-agent-harness/README.md` | broker HTTP handlerがTLS終端後のstrict requestを既存Exchangeへ接続する範囲と、listener/TLS/production identity/live E2Eが対象外であることだけを追記する。 |

## 実装手順

1. 新packageのRules、resolver interface、Handler、固定errorとnil/zero fail-closed面を定義し、body上限と依存の検証試験を置く。
2. HTTP/1.1 origin-form、URL/Host、Content-Length、transfer/trailer/upgrade、Content-Type/Authorizationの入力検査とbounded body copyを実装し、拒否時のresolver/Exchangeゼロ到達を試験する。
3. context-only resolver一回と、raw Host/canonical pathからの新規`egresstransaction.Request`組立て、`Exchange.Do`同期一回を実装する。identity自己申告と入力aliasをcounter/copy fixtureで拒否・検出する。
4. writer header初期化、2xx/JSONのみのsuccess縮退、正確なContent-Length、固定empty 403、output copyを実装し、header/body/non-leak試験を置く。
5. fake resolver/Exchangeの`httptest`とreal Exchange + fake上流dependencyで、GitHub/OpenAI成功、protocol/framing/header/body拒否、単回性、copy、並行隔離をrace fixtureへまとめ、READMEを責務境界に合わせる。
6. candidate前に `go test -count=1 -race ./internal/brokerhttp`、harness `make check`、`make distcheck`、README変更時のroot `make lint-docs`、`git diff --check`、許可2pathとbase...candidate追加＋削除1,000行以下を確認する。candidate launcherのroot `make check`は一回だけとし、planning/candidate/completion以外のcommit、追加gate/process/digest/candidate重複記録は作らない。

## 検証計画

- `go test -count=1 -race ./internal/brokerhttp` をfocused testとして、Rules/receiver、HTTP/1.1 canonical mapping、origin/URL/Host/framing/body/headerの拒否、context-only identity、resolver/Exchange各一回、both provider成功、input/output copy、success header allowlist、empty deny、parallel isolationとfixed non-leakを検出する。
- real Exchange経路はreal Policy/Registryとfake credential resolver/RoundTripperだけを使い、既存Transaction/Forwarder/Policy検査をHandlerに複製せず、両providerの許可が一回のHandler→Exchange連鎖で成立することを確認する。
- 実listener/TLS、peer credential/Tailscale resolver、実credential/provider、DNS/system trust、Agent namespaceは`live-e2e` blockedのまま残し、hermetic PASSで代替しない。

## 停止条件

- Exchange API、Transaction、Policy、Forwarder、Registry又はcredential/transport packageの変更、provider policy/Authorization/capability検査のHandler内再実装が必要になれば停止する。
- listener、TLS、production resolver、config/process wiring、external dependency、実network/credential、許可外path、又は1,000行上限超過が必要になれば停止してMainへ戻す。
- chunked/HTTP/2/absolute-form、redirect/retry、streaming、任意request/response header、diagnostic/challengeを受理する必要が生じた場合は本Taskのstrict HTTP/1.1 contractを広げず停止する。

## 未解決事項

- なし

## main Agentレビュー

- [x] TASKの全AC-IDへ設計判断、パス、順序、失敗時の扱いを対応させ、条件本文を複製していない。
- [x] 設計観点と代替案を検討している。
- [x] QA_PLANがTASK-firstで独立作成されている。
- [x] `dependency-ready reconciliation`と完了経路preflightが完了している。
- [x] DEV開始を承認した。

安全契約変更でv2契約を選ぶ場合は、コメントを外して`safety_contract_version: 2`と予定パス・生成パスの配列を記録し、DEV前に`make task-preflight TASK=TASK-0049`を実行する。変更しない種別は空配列とし、通常の予定パスと生成パスを重複させない。独立計画レビューのPASSとMainの分類承認をフロントマターへ記録する。分類変更時はTask、PLAN、QA_PLANを再承認し、承認者と時刻を更新する。
