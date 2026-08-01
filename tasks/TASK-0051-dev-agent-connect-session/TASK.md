---
task_id: "TASK-0051"
title: "CONNECT受付とTLS終端sessionを実装する"
status: plan
created_at: "2026-08-01"
---

# TASK-0051 CONNECT受付とTLS終端sessionを実装する

## `Planning input packet`（Main Agent所有）

このsectionをPlannerとQAへ渡す唯一の`planning input packet`とし、各内容をPLAN/QA_PLANへ複製しない。

### 目的

Agent側HTTPS clientから既に受理した一つの`net.Conn`について、strict CONNECTを一回だけ受け、TASK-0050のhost限定leafでTLSを終端し、TASK-0049のHTTP Handlerへconnection-bound contextを保った一つのHTTP/1.1 requestだけを渡すsession境界を作る。実listener bind、OS peer identity、秘密file、process wiringを混ぜず、CONNECT/TLS/HTTPの順序、fail-closed response、deadlineとconnection所有権をhermeticに固定する。

### 対象と対象外

#### 対象

- `internal/connectsession`のproduction `Session`。RulesとしてTASK-0050 Authority相当の`Issue(host)` interfaceと非nil `http.Handler`だけを受け、`Serve(context.Context, net.Conn)`で一connectionを完全所有して終了時にcloseする。
- 最大16 KiBのHTTP/1.1 CONNECT headerをdeadline付きで一回だけ読み、exact `api.github.com:443`又は`api.openai.com:443`のRequestURI/Hostだけを受理する。Host以外は省略可能な単一`User-Agent`（visible ASCII、256 byte以下）と単一`Proxy-Connection: keep-alive`だけを無視して受け、body、Content-Length、transfer/trailer/upgrade、Proxy-Authorization、その他header、early/pipelined byteを拒否する。無効入力には固定header・空bodyの403だけを返す。
- target hostのleafを成功応答前に一回だけ発行し、固定`200 Connection Established`後にTLS 1.2以上、SNI exact target、ALPN `http/1.1`だけでhandshakeする。client certificateを要求せず、SNI/ALPN/TLS不整合後は追加HTTP responseを作らずcloseする。
- callerから渡されたcontextだけをTLS後requestへ継承し、一sessionにつき一つのHTTP/1.1 requestを既存Handlerへ一回だけ渡す。outer CONNECT header/RemoteAddr/SNIから主体を生成せず、keep-alive、HTTP/2、retry、redirect、別handler fallbackを行わない。
- phase deadlineとcaller cancellationの早い方でCONNECT/TLS/HTTPを停止し、正常・拒否・panic/依存失敗の全経路でconnectionを閉じ、公開error/Format/responseにtarget、header、SNI、peer、下位errorを含めない。複数sessionはmutable stateを共有しない。
- `net.Pipe`とin-memory CA、real `brokerhttp.Handler` + fake Exchange/context resolverによるhermetic race test。両host成功、strict CONNECT、fixed 403/200、SNI/ALPN/TLS拒否、context継承、handler単回、cancel/timeout、並行session隔離を検出する。

#### 対象外

- TCP/Unix socketのbind/accept loop、listen address、systemd/socket activation、実配置、restart/rollback、複数connectionのaccept管理。
- Linux peer credential、Tailscale identity、UID/workspace/Agent mapping、production `SubjectResolver`、Agent自己申告でないidentityの実装・確認。
- CA certificate/private keyのfile読込・生成・rotate、Agent OS trust storeへのinstall、実証明書、certificate pinning対応。
- HTTP/2、generic CONNECT、任意host/port、absolute-form、plain HTTP proxy、WebSocket/Upgrade、chunked/streaming、multiple request/keep-alive tunnel、transparent proxy、SNI-only L4 proxy。
- 既存`proxyca`、`brokerhttp`、Exchange/Transaction/Forwarder/Policy/credential/transport package、config/CLI/service/build/generated artifactの変更。
- 実`gh`/OpenAI SDK、実proxy environment、実CA trust、実network namespace/VPS、実GitHub/OpenAIを通すlive E2E。`net.Pipe` PASSで実client/配置を保証しない。

### 受け入れ条件

- [ ] AC-1: `New`は非nil Authority interfaceと非nil Handlerだけを受理し、immutable Sessionを返す。不正Rules、nil/zero/破損Session、nil/typed-nil conn/contextはpanicせず固定errorとなる。SessionはAuthority/Handler以外の長寿命state、default handler/CA、接続、target、bufferを保持しない。
- [ ] AC-2: `Serve`はphase deadline内に最大16 KiBのHTTP/1.1 CONNECT一件だけを読み、RequestURIとHostがexact `api.github.com:443`又は`api.openai.com:443`で一致する場合だけ先へ進む。Host以外は省略可能な単一`User-Agent`（visible ASCII、256 byte以下）と単一`Proxy-Connection: keep-alive`だけを無視して受理する。method/version/authority差、body/Content-Length、transfer/trailer/upgrade、Proxy-Authorization、重複/その他header、control/過長値、CONNECT後のearly byteはAuthority/Handler前に拒否し、固定`HTTP/1.1 403 Forbidden`、`Content-Length: 0`、`Connection: close`、空bodyだけを返す。
- [ ] AC-3: valid CONNECTではtarget hostの`Issue`を一回だけ呼び、成功後だけ固定`HTTP/1.1 200 Connection Established`と空bodyを返す。そのconnectionでTLS 1.2以上、SNI exact target、ALPN `http/1.1`、発行済みleafだけを使い、client certificateを要求しない。Issue/SNI/ALPN/TLS failureはretry、別host、別certificate、診断responseを作らずconnectionを閉じる。
- [ ] AC-4: TLS成功後はcaller由来contextをrequestへ継承し、HTTP/1.1 request一件だけを注入Handlerへ同期一回渡してresponse完了後にcloseする。CONNECT header、RemoteAddr、SNI、HTTP headerからcontext/主体を生成又は変更せず、keep-alive、pipelining、HTTP/2、別handler/default server、retryを使わない。real `brokerhttp.Handler`とのfixtureでcontext-only Subject解決とExchange単回を確認する。
- [ ] AC-5: CONNECT read、TLS handshake、HTTP request/responseは固定上限とcaller context deadlineの早い方で停止し、caller cancellation、stall、EOF、handler panic/connection errorでもgoroutine又はconnectionを残さない。全終了経路で入力connを所有してcloseし、公開error/Format/outer responseは固定でtarget/header/SNI/peer/context/下位errorを含まない。並行session間でcertificate、context、request/response、buffer、deadlineを共有しない。
- [ ] AC-6: hermetic race testが両hostのCONNECT→TLS→real Handler response、fixed 403/200、strict authority/header/framing/early byte拒否、Issue単回、SNI/ALPN/TLS失敗、context継承・非自己申告、handler単回、cancel/stall/close、並行隔離を検出する。`go test -count=1 -race ./internal/connectsession`、harness `make check`/`make distcheck`、README変更時のTask worktree `make lint-docs`、candidate launcherのroot `make check`がPASSし、base...candidate差分は追加＋削除1,000行以下である。

### 安定した参照

| 参照ID | 対象 | 固定改訂/ダイジェスト | 用途 |
|---|---|---|---|
| REF-1 | TASK-0049 broker HTTP Handler | candidate `d5b94f2` / completion `5f5b0eb` | TLS後requestをcontext-only Subject resolverとExchangeへ一回渡す既存入口 |
| REF-2 | TASK-0050 proxy CA Authority | candidate `ffc220f` / completion `688e927` | exact two-host leaf発行と公開CA境界。SessionはAuthorityを変更しない |
| REF-3 | Development Agent harness設計 | main `688e927`時点 | custom CA対応clientだけ、strict HTTP検査、generic CONNECT/upgrade拒否、実identity/配置分離 |
| REF-4 | egress transaction / proxy CA意味Wiki | main `688e927`時点 | Handler/CA/session責務とlive E2E未確認境界 |

### 依存状態

| 依存 | 状態 (`ready` / `pending`) | planning参照 | `ready`後に固定する値 |
|---|---|---|---|
| TASK-0049 | `ready` | REF-1 | Handler API/context-only resolverを変更せず注入する |
| TASK-0050 | `ready` | REF-2 | Authorityの`Issue(host)`だけを使いCA signer/public APIを広げない |

### 許可パス

- `tools/dev-agent-harness/internal/connectsession/`
- `tools/dev-agent-harness/README.md`

### 完了経路preflight

| 確認対象 | 結果 | コマンドまたは根拠 |
|---|---|---|
| 完了checker | `ready` | planning/candidate/completionの3 commit gateとpost-merge `task-check`を使う |
| 権限 | `ready` | DEV/QAは`net.Pipe`、test生成CA、in-memory context/fakeだけを使い、socket bind、実秘密、OS trust、外部networkへ到達しない |
| 依存状態と参照 | `ready` | TASK-0049/0050完了。公開interfaceをtest側adapterで合成し既存packageを変更しない |
| 生成物の有無と更新方法 | `ready` | Go source/testとREADMEだけ。dependency、config、build、生成物なし |
| 割当ワークツリー | `ready` | `worktrees/TASK-0051-dev-agent-connect-session` |
| Lapログの書込・Schema・`repository annotation` | `ready` | 標準3 commits。新Schema、digest転記、追加機械checkなし |

### 未決事項

- なし

### `Dependency-ready reconciliation`

<!-- 依存ready時にMainがready参照、planning参照との差分、AC/設計/scope/QAへの影響、再承認結果を追記する。依存なし又は未readyならN/Aとする。 -->

- TASK-0049/0050はいずれもplanning開始時点でready。HandlerはTLS後HTTP、Authorityはleaf発行だけを所有する。本Taskは間のone-connection sessionだけを追加し、既存API/意味を変更せず、実listener/identity/file/trustを後続live境界へ残す。

## 背景

HTTP Handlerとhost限定CAは完成したが、Agent側clientが送るCONNECTをTLS後Handlerへ接続するnetwork sessionがない。listener bindやproduction identityまで同時実装すると、protocol smuggling、TLS host binding、OS peer attribution、service lifecycleの欠陥が混ざる。まず受理済み一connectionの有限state machineだけを固定する。

## 検討すべき設計観点

- CONNECT target、TLS SNI、leaf SAN、TLS後Hostは別層で同じ2 hostへ拘束する。いずれもnormalize又はfallbackせず、前段の成功を後段の認可根拠にしない。
- caller contextは後続production listenerがOS peer identityから作るtrusted入力とし、本Taskはその値を透過継承するだけにする。outer/inner HTTP header、RemoteAddr、SNIから主体を作らない。
- 一session一HTTP request・keep-aliveなしに狭め、connection reuseの性能最適化は実client観測後に検討する。初期GH/OpenAI操作はconnection close後の再接続を許容するが、live E2Eまでsupport済みとしない。
- CA発行は200前、TLS failureは200後なので、後段失敗時にHTTP診断を混ぜない。公開errorとwire denyは固定値へ畳む。
- deadline/cancellationと全path closeをstate machineの一部としてtestし、実listener accept loopやservice shutdownとは分離する。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] planning/candidate/completionの3 commit経路とcandidate一回のroot `make check`を満たしている。
- [ ] 同一candidateの独立REVIEW/QAを完了し、実listener/identity/CA trust/client/VPSのlive E2E未実施境界をPASSと誤記していない。
- [ ] 再利用可能な知識が生じた場合だけ意味Wikiへ同化し、post-merge `task-check`をPASSしている。

## 関連コンテキスト

### 意味 Wiki

- `wiki/semantic/schemas/development-agent-harness-egress-transaction.md`
- `wiki/semantic/schemas/development-agent-harness-proxy-ca.md`

### 判断

- 実listenerより先に、strict CONNECT→host-bound TLS→single HTTP requestのone-connection sessionをhermeticに固定する。

### 適用しなかった重要な判断

- listener bind、production peer identity、CA file/trust installを同時実装する案はOS/environment lifecycleとprotocol state machineを混在させるため採用しない。
- generic CONNECT tunnel又はSNIだけで上流へ中継する案はTLS後HTTP policyを迂回するため採用しない。
- 初期からkeep-alive/multiple request、HTTP/2、可変timeout/header limitを公開する案は未観測surfaceを増やすため採用しない。
