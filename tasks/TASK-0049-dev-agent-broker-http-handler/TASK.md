---
task_id: "TASK-0049"
title: "TLS終端後HTTP交換handlerを実装する"
status: plan
created_at: "2026-08-01"
---

# TASK-0049 TLS終端後HTTP交換handlerを実装する

## `Planning input packet`（Main Agent所有）

このsectionをPlannerとQAへ渡す唯一の`planning input packet`とし、各内容をPLAN/QA_PLANへ複製しない。

### 目的

TLSを既に終端して正規の`net/http` requestへ変換したbroker内境界で、trustedな呼出元解決とHTTP request/response変換をTASK-0048のin-memory Exchangeへ接続する。Agent由来requestから許可されたmethod、host/path、Content-Type、Authorization、上限付きbodyだけを新しい値へcopyし、成功時だけ縮退済みresponseを固定headerで返す。protocol不整合、呼出元解決失敗、Exchange拒否では秘密又は入力detailを返さずfail closedにする。

### 対象と対象外

#### 対象

- `internal/brokerhttp`のproduction `http.Handler` constructor。RulesとしてTASK-0048 Exchange interface、request contextだけを受けるtrusted呼出元resolver、1 byte〜1 MiBのrequest body上限だけを受け取る。
- HTTP/1.1のorigin-form requestだけを受理し、空scheme/authority/userinfo/fragment/query/raw path、exact Host、既知Content-Length、transfer coding/trailer/upgrade不在を検査する。TLS、SNI、socket peer情報は既にtrusted前段で処理済みとし、Handlerはrequest headerやremote addressから呼出元を自己申告させない。
- 呼出元resolverをbody/headerを含まない`context.Context`だけで一回呼び、`egresstransaction.Subject`を得る。method、`https://` + Host + canonical path、Content-Type単値、Authorization values、上限内bodyを独立copyしたRequestへ変換し、Exchangeを一回だけ同期呼出する。
- 成功時だけExchangeのstatus、空又は`application/json`、独立bodyを、固定`Cache-Control: no-store`、`X-Content-Type-Options: nosniff`、正確なContent-Length以外を持たないresponseへ書く。全拒否は同一の空`403`へ畳み、retry、redirect、challenge、diagnostic body/headerを作らない。
- fake呼出元resolver/Exchangeと`httptest`、およびreal Exchange + fake上流dependencyを使うhermetic race test。両provider成功、HTTP mapping、protocol/framing/header/body拒否、単回呼出、input/output copy、response header allowlist、固定empty deny、並行request隔離を検出する。

#### 対象外

- TCP/Unix listener、TLS handshake/CA/private key、CONNECT受付/トンネル、SNI証明書生成、HTTP/2、absolute-form proxy request、transparent proxy、peer credential又はTailscale identityからのproduction呼出元resolver。
- `http.Server` timeout/header size設定、process/config/service wiring、privilege drop、socket activation、production Exchange/resolver/transport/credential bundle生成。
- request header/bodyの監査、challenge/approval、push grant、Git Smart HTTP、GraphQL、streaming/upload、redirect/retry、rate/cost、audit/persistence。
- 既存`brokerexchange`、Transaction、Forwarder、Policy、Registry、credential/transport packageの変更又はHandler内での再実装。
- 実TLSクライアント、実gh/OpenAI SDK、実GitHub/OpenAI、実credential、DNS/system trust、Agent network namespaceを使うlive E2E。hermetic Handler PASSで実配置を保証しない。

### 受け入れ条件

- [ ] AC-1: `New`は非nil Exchange、非nil trusted呼出元resolver、1〜1,048,576 byteのrequest body上限だけを受理し、immutable Handlerを返す。不正Rulesは固定errorになり、nil/zero Handlerもpanic、dependency detail、入力値を公開せず空403を返す。
- [ ] AC-2: HandlerはHTTP/1.1 origin-formだけを受理する。nil URL、absolute/opaque/userinfo、query/fragment/raw又はpercent-encoded path、空/過長Host、HTTP/1.0/2、CONNECT/upgrade、transfer coding、trailer、未知又はbody長と一致しないContent-LengthをExchange前に拒否し、requestのHost/path/header/bodyを変更・保持しない。
- [ ] AC-3: protocol検査後、呼出元resolverをrequest contextだけで一回呼ぶ。成功時はmethod、`https://` + Host + path、Content-Typeの0又は1値、Authorizationの全値、上限内bodyを新しい`egresstransaction.Request`へcopyし、解決済みSubjectとともにExchangeへ同期一回だけ渡す。RemoteAddr、Forwarded系header、Agent指定identity headerからSubjectを作らず、default Exchange、retry又はredirectを選ばない。
- [ ] AC-4: Exchange成功時だけ2xx status、空又は正規`application/json` Content-Type、独立bodyを返す。response headerは必要時のContent-Type、固定Cache-Control/X-Content-Type-Options、正確なContent-Lengthだけに限定し、Exchange又はcaller bufferとaliasせず、次又は並行requestとstateを共有しない。
- [ ] AC-5: mapping、body read、呼出元resolver又はExchangeの全失敗は、空body、Content-Length 0、固定no-store/nosniffだけの403へ畳む。opaque handle、credential、URL/path、Host、request/response body、呼出元、provider、下位errorをresponse、error又はformatへ出さず、Exchangeを複数回呼ばない。
- [ ] AC-6: fake resolver/Exchange + `httptest`とreal Exchange + fake上流dependencyによるhermetic race testが両provider成功、HTTP/1.1 canonical mapping、protocol/framing/content/header/body拒否、呼出元非自己申告、resolver/Exchange単回、zero/empty deny、success header allowlist、input/output copy、並行隔離、fixed non-leakを検出する。`go test -count=1 -race ./internal/brokerhttp`、harness `make check`/`make distcheck`、README変更時のroot `make lint-docs`、candidate launcherのroot `make check`がPASSし、base...candidate差分は追加＋削除1,000行以下である。

### 安定した参照

| 参照ID | 対象 | 固定改訂/ダイジェスト | 用途 |
|---|---|---|---|
| REF-1 | TASK-0048 broker Exchange | candidate `4536a19` / completion `474cae2` | `Do(Subject, Request)`の単回同期呼出、safe response、zero/fixed failure境界 |
| REF-2 | TASK-0041 egress transaction | candidate `c369ec4` / completion `1296427` | SubjectとRequestの型、Authorization/capability消費順序 |
| REF-3 | Development Agent harness設計 | main `474cae2`時点 | TLS終端後にHost/method/path/bodyを検査し、呼出元をOpaque capabilityへ束縛する境界 |
| REF-4 | egress transaction意味Wiki | main `474cae2`時点 | Handler、Exchange、Transaction、Forwarderの責務分離とlive未確認境界 |

### 依存状態

| 依存 | 状態 (`ready` / `pending`) | planning参照 | `ready`後に固定する値 |
|---|---|---|---|
| TASK-0048 | `ready` | REF-1 | Exchange APIを変更せず、Handlerから一回呼ぶ |

### 許可パス

- `tools/dev-agent-harness/internal/brokerhttp/`
- `tools/dev-agent-harness/README.md`

### 完了経路preflight

| 確認対象 | 結果 | コマンドまたは根拠 |
|---|---|---|
| 完了checker | `ready` | planning/candidate/completionの3 commit gateとpost-merge `task-check`を使う |
| 権限 | `ready` | DEV/QAは`httptest`、fake resolver/Exchange/transportとin-memory Registryだけを使い、listener、TLS、外部network、実credentialへ到達しない |
| 依存状態と参照 | `ready` | TASK-0048完了。既存Exchange/Transaction APIと責務をREF-1/2で固定 |
| 生成物の有無と更新方法 | `ready` | Go source/testとREADMEだけ。dependency、config、生成物なし |
| 割当ワークツリー | `ready` | `worktrees/TASK-0049-dev-agent-broker-http-handler` |
| Lapログの書込・Schema・`repository annotation` | `ready` | 標準3 commits。新Schema、digest転記、追加機械checkなし |

### 未決事項

- なし

### `Dependency-ready reconciliation`

- TASK-0048はplanning開始時点でready。Exchangeの公開APIとfixed failure境界をそのまま使い、HTTP変換だけを新packageへ追加する。listener/TLS/production identity resolverと実クライアントE2Eは後続/live blockedのままとする。

## 背景

ExchangeはtypedなSubject/Requestを受けてsafe responseを返せるが、TLS終端後の`net/http` requestを安全にtyped値へ変換し、Agentへ最小responseを書き戻す入口がない。listenerや証明書生成まで同時に加えると、HTTP parser境界、identity境界、OS/TLS配置の失敗原因が混ざるため、まずhermeticなHandlerだけを固定する。

## 検討すべき設計観点

- 呼出元resolverにはrequest header/body/RemoteAddrを渡さずcontextだけを渡し、前段が束縛したtrusted identityとAgent自己申告を分離する。production resolverは後続Taskの責務とする。
- net/httpが解析済みでも、Handlerはorigin-form、HTTP/1.1、no query/fragment/raw path、既知Content-Length、no transfer/trailer/upgradeへsurfaceを狭める。HTTP/2やchunked対応は実クライアント観測後に別Taskで追加する。
- Host/pathを便利にnormalizeせず、明示的に`https://`へ合成したraw値を既存Policy/Transactionへ渡す。policy/capability/credential/response検査は再実装しない。
- failure responseは理由を分類せず同一empty 403にする。challenge/approvalや診断は後続の監査/非同期制御面へ分離し、Handlerで秘密を含む本文を作らない。
- response writerへ書く前にheaderを固定allowlistへ初期化し、Exchange成功後だけstatus/bodyを公開する。write failureをretryせず、partial write診断をAgentへ返さない。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] planning/candidate/completionの3 commit経路とcandidate一回のroot `make check`を満たしている。
- [ ] 同一candidateの独立REVIEW/QAを完了し、実TLS/Agent/client/providerのlive E2E未実施境界をPASSと誤記していない。
- [ ] 再利用可能な知識が生じた場合だけ意味Wikiを既存ページへ同化し、post-merge `task-check`をPASSしている。

## 関連コンテキスト

### 意味 Wiki

- `wiki/semantic/schemas/development-agent-harness-egress-transaction.md`

### 判断

- listener/TLSより先に、trusted context identityとHTTP/1.1 origin-formをExchangeへ渡すhermetic Handlerを固定する。

### 適用しなかった重要な判断

- HandlerがRemoteAddr又はidentity headerからSubjectを生成する案は、前段proxy/peer境界との信頼関係を暗黙化するため採用しない。
- listener、TLS証明書、production resolver、server timeoutを同時実装する案はhermetic QAとlive配置QAを混在させるため採用しない。
- HTTP/2、chunked、absolute-formを初期surfaceへ含める案はparser差異とframing ambiguityを増やすため、実クライアント観測まで採用しない。
