---
task_id: "TASK-0047"
title: "provider上流response Forwarderを実装する"
status: plan
created_at: "2026-08-01"
---

# TASK-0047 provider上流response Forwarderを実装する

## `Planning input packet`（Main Agent所有）

このsectionをPlannerとQAへ渡す唯一の`planning input packet`とし、各内容をPLAN/QA_PLANへ複製しない。

### 目的

TASK-0041のtrusted `Forwarder`境界とTASK-0045の上流HTTPS `http.RoundTripper`を接続する、broker内の同期request/response変換を実装する。`PreparedRequest`を再検証して実認証情報を上流へ一度だけ送信し、成功responseからstatus、JSON本文だけを上限付きで独立コピーして注入sinkへ渡す。認証情報、上流header、下位error又は未検証本文をAgent側へ返さない小さい境界にする。

### 対象と対象外

#### 対象

- `internal/upstreamforwarder`のproduction constructorと、`egresstransaction.Forwarder`を実装する同期Forwarder。Rulesとして同じ`egresspolicy.Policy`、注入`http.RoundTripper`、request単位のresponse sink、全体timeout、response本文上限だけを受け取る。
- `PreparedRequest`を同じPolicyで再評価し、得た正規scopeと保持scopeが完全一致した場合だけ送信する。実認証情報を含むAuthorization、method、URL、content type、本文も固定境界で検査し、policy/scope不一致をtransport前に拒否する。
- 上流requestはcaller入力を変更せず独立本文を持ち、固定`Accept`/`User-Agent`、必要時の`Content-Type`と実`Authorization`以外のAgent由来headerを持たない。context timeout内で注入transportを同期的に一回だけ呼び、redirect/retryを実装しない。
- 2xx responseだけを受理する。HEAD又は空responseは空本文として扱い、本文がある場合は上限内のUTF-8かつ有効なJSONとJSON media typeだけを受理する。上流headerは転送せず、sinkへstatus、正規化content type、独立本文だけを一回渡す。
- response/error全経路のbody close、context cancellation、read/close/sink failure、responseとerrorの同時返却、固定non-leak error、input/sink間のalias不在をfake transport/body/sinkでhermeticに検証する。

#### 対象外

- Agent向けHTTP/HTTPS proxy listener、CONNECT、TLS interception、CA、Agent trust store、socket/IPC、downstream HTTP response writer、CLI/config/service wiring。
- `egresspolicy`、`egresstransaction`、`upstreamtransport`、resolver/credential store/capabilityの変更又は統合。Transaction APIからresponseを返す変更も行わず、後続gatewayがrequest単位sinkを合成する。
- Git Smart HTTP、Git push/pull、GraphQL、OpenAI streaming/tool/file/image、provider error本文、redirect/retry、response streaming、compression、cache、rate/cost計測。
- `Set-Cookie`、redirect location、pagination/rate-limit/request ID等の上流header転送。必要な安全header allowlistは実クライアント統合で観測して別Taskにする。
- 実GitHub/OpenAI、実Internet DNS/TLS/system trust、実認証情報、外部networkを使うlive E2E。fake transportのPASSで実provider受理を保証しない。
- 新dependency、build/install/config/生成物、恒久外部network test。

### 受け入れ条件

- [ ] AC-1: `New`はPolicy、RoundTripper、ResponseSink、1ms以上30秒以下のtimeout、1 byte以上1 MiB以下のresponse上限を検査・保持し、不正Rulesへ固定errorを返す。Forwarderは`egresstransaction.Forwarder`を実装し、nil/zero receiverを含む失敗をpanic又は入力detailなしの固定errorにする。
- [ ] AC-2: `Forward`は`PreparedRequest`のmethod/URL/content type/bodyを同じPolicyで再評価し、正規scopeが`PreparedRequest.Scope`と完全一致し、Authorizationが`Bearer `に続く1〜4,096 byteのvisible ASCII値である場合だけtransportへ到達する。拒否時はtransport/sinkを0回とし、caller所有sliceを変更・保持しない。
- [ ] AC-3: 許可requestはtimeout付きcontextと独立本文で一つの`http.Request`へ変換し、上流へ渡すheaderを実`Authorization`、固定`Accept: application/json`、固定`User-Agent`、OpenAI時の`Content-Type: application/json`だけに限定する。注入RoundTripperを同期的に一回だけ呼び、Forwarder自身はredirect、retry、environment proxy又はdefault transportを使わない。
- [ ] AC-4: 2xxかつ非nil bodyのresponseだけを読み、必ずcloseする。HEAD/204は0-byte本文だけを空responseとして受理し、それ以外の0-byte本文も空responseとする。本文ありは上限内、UTF-8、有効なJSON、`application/json`又は`application/*+json` media typeだけを受理する。HEAD/204の非empty本文、上限超過、非2xx、invalid status/content type/JSON、read/close error、response+error又はtimeout/cancelはsinkへ渡さず固定errorにする。
- [ ] AC-5: 成功時だけResponseSinkを一回呼び、status code、空又は正規`application/json` content type、caller/transportとaliasしない本文だけを渡す。上流header、Authorization、URL、scope、provider error本文及びunderlying errorはsink、error、formatへ出さず、sink失敗も再試行しない。
- [ ] AC-6: fake RoundTripper/body/sinkによるhermetic testが両provider成功、scope再評価・不一致、header allowlist、単回transport/sink、timeout/cancel、2xx/HEAD/204/JSON media type、size/UTF-8/JSON/status/read/close/sink異常、response+error、body close、copy ownership、固定error/non-leakを検出する。`go test -race ./internal/upstreamforwarder`、harness `make check`/`make distcheck`、candidate launcherのroot `make check`がPASSし、base...candidate差分は追加＋削除1,000行以下である。

### 安定した参照

| 参照ID | 対象 | 固定改訂/ダイジェスト | 用途 |
|---|---|---|---|
| REF-1 | TASK-0041 egress transaction | candidate `c369ec4` / completion `1296427` | policy、capability、credential解決後に同期一回だけ呼ばれる`PreparedRequest`/`Forwarder`契約 |
| REF-2 | TASK-0045 upstream transport | candidate `929ad89` / completion `0e8136a` | 注入RoundTripper、固定origin/DNS/IP/TLS/HTTP1、一接続・no retry境界 |
| REF-3 | egress transaction意味Wiki | main `ea2a49e`時点 | Forwarderが未実装である境界、実認証情報と上流responseをAgentへ漏らさない責務 |
| REF-4 | Development Agent harness設計 | main `ea2a49e`時点 | phase 1のGitHub REST read/OpenAI非streamingと、proxy/Git/承認の後続境界 |

### 依存状態

| 依存 | 状態 (`ready` / `pending`) | planning参照 | `ready`後に固定する値 |
|---|---|---|---|
| TASK-0041 | `ready` | REF-1 | Forwarder interfaceとPreparedRequestを変更せず実装する |
| TASK-0045 | `ready` | REF-2 | productionで注入可能なRoundTripperと所有権契約をそのまま利用する |

### 許可パス

- `tools/dev-agent-harness/internal/upstreamforwarder/`
- `tools/dev-agent-harness/README.md`

### 完了経路preflight

| 確認対象 | 結果 | コマンドまたは根拠 |
|---|---|---|
| 完了checker | `ready` | planning/candidate/completionの3 commit gateとpost-merge `task-check`を使う |
| 権限 | `ready` | DEV/QAはfake RoundTripperだけを使い、外部network、実認証情報、OS権限へ到達しない |
| 依存状態と参照 | `ready` | TASK-0041/0045は完了済み。interfaceとtransport境界をREF-1/2で固定 |
| 生成物の有無と更新方法 | `ready` | Go source/testとREADMEだけ。dependency、config、生成物なし |
| 割当ワークツリー | `ready` | `worktrees/TASK-0047-dev-agent-upstream-forwarder` |
| Lapログの書込・Schema・`repository annotation` | `ready` | 標準3 commits。新Schema、digest転記、追加機械checkなし |

### 未決事項

- なし

### `Dependency-ready reconciliation`

- TASK-0041/0045はいずれもplanning開始時点でready。既存interface/packageを変更せず、新packageの合成だけに限定する。実provider受理とAgent向けresponse writerはこのTaskでもlive E2E blockedのままとする。

## 背景

policy、capability消費、credential解決、上流DNS/TLS transportは個別に実装済みだが、それらの間でcredential-bearing `PreparedRequest`をHTTP requestへ変換し、上流responseを安全な最小値へ縮退するForwarderは未実装である。これをAgent proxyと同時に作ると、TLS終端、downstream writer、上流header、provider error、secret所有権の不具合を切り分けにくい。先にfake transportで閉じる同期交換を実装する。

## 検討すべき設計観点

- Transactionが一度認可した値であっても、Forwarder単体が直接呼ばれた場合に実認証情報を別repo/hostへ送らないよう、同じPolicyによる再評価とscope完全一致をnetwork前に行う。この再評価は転記形式checkではなくcredential送信先の安全境界である。
- Forwarderはrequest単位で生成されるsinkへresponseを渡し、Transactionの戻り値/interfaceを変更しない。long-lived Forwarderでresponseを共有せず、後続gatewayがper-requestに合成する。
- provider success responseだけを扱い、error本文やheaderを先回りして許可しない。gh pagination/rate-limit等で必要なheaderは実統合の観測後に最小allowlistを別Taskで加える。
- full-body bufferingは1 MiB上限のphase 1限定とし、streaming、compression、retryを実装しない。bodyを完全検証・closeしてからsinkを呼ぶため、失敗時にpartial responseを公開しない。
- 公開error/formatは固定値だけにし、secret、URL、scope、response、underlying errorをwrap/記録しない。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] planning/candidate/completionの3 commit経路とcandidate一回のroot `make check`を満たしている。
- [ ] 同一candidateの独立REVIEW/QAを完了し、実provider/Agent proxyのlive E2E未実施境界をPASSと誤記していない。
- [ ] 再利用可能な知識が生じた場合だけ意味Wikiを既存ページへ同化し、post-merge `task-check`をPASSしている。

## 関連コンテキスト

### 意味 Wiki

- `wiki/semantic/schemas/development-agent-harness-egress-transaction.md`

### 判断

- Agent proxyより先に、credential-bearing requestと上流success responseの同期交換を新しいinternal packageへ閉じる。

### 適用しなかった重要な判断

- Transactionにresponse戻り値を追加する案は既存契約を広げ、同一transactionの再利用・保持境界を複雑にするため採用しない。
- provider error本文、任意response header、streaming、redirect/retryを初版で許可する案は、Agentへ返す情報面と副作用を拡張するため採用しない。
