---
task_id: "TASK-0052"
title: "受理loopとtrusted peer context bindingを実装する"
status: plan
created_at: "2026-08-01"
---

# TASK-0052 受理loopとtrusted peer context bindingを実装する

## `Planning input packet`（Main Agent所有）

このsectionをPlannerとQAへ渡す唯一の`planning input packet`とし、各内容をPLAN/QA_PLANへ複製しない。

### 目的

既に受理loopの外側でsocket種別とbind先を決めた`net.Listener`について、同時接続数を固定上限へ抑えながらconnectionを受理し、trusted peer binderが返す主体を外部から偽装できないprivate context valueへ束縛して、TASK-0051のone-connection Sessionへ渡すserver境界を作る。実socket bind、Linux peer取得、network namespace、systemd、秘密fileとcompositionを混ぜず、accept、identity binding、connection所有権、cancel/drainをhermeticに固定する。

### 対象と対象外

#### 対象

- `internal/brokerlistener`のproduction `Server`とcontext-only `Resolver`。Rulesとしてtrustedでcontextに協調する`PeerBinder`、同じく協調するTASK-0051 Session相当interface、`1..64`の`MaxConcurrent`だけを受け、immutable Serverを返す。
- `Server.Serve(context.Context, net.Listener)`がlistenerを完全所有し、同時実行slotを取得してから`Accept`し、受理connectionごとにroot由来contextを作る。binderを一回だけ呼び、返された`egresstransaction.Subject`がUID正数かつ既存capability同等のidentifier上限を満たす場合だけprivate context keyへcopyしてSessionを一回呼ぶ。
- `Resolver.Resolve(context.Context)`はServerだけが設定できるprivate valueをcopyして返し、missing、wrong-type、invalid、cancelled contextを固定errorで拒否する。RemoteAddr、HTTP/CONNECT/TLS/header、公開context key又はcaller値から主体を作らない。
- 最大同時Session数を`MaxConcurrent`以下に保つ。binder拒否、invalid subject、binder/session panic又はSession errorはそのconnectionだけをcloseしてacceptを継続し、dependency detailを返却・Format・logへ出さない。予期しないAccept errorはlistenerをcloseし、全connection contextをcancelしてdrain後に固定server errorを返す。
- caller cancel/deadlineではlistenerをcloseしてAcceptを解除し、全connection contextをcancelし、協調的binder/Sessionがreturnするまでdrainして正常終了する。Server自身のwatcher/goroutine、accepted connection、listenerを残さない。任意の非協調callbackを強制停止せず、goroutineへ逃がしてtimeout leakを作らない。
- fake bounded listener、`net.Pipe`、fake binder、real `connectsession.Session`相当のcontext観測fixtureによるhermetic race test。binding順序、private Resolver、上限、拒否継続、panic隔離、accept failure、cancel/drain、並行identity隔離を検出する。

#### 対象外

- TCP/Unix socketの生成、address/port/path、listen/bind permission、socket option、systemd socket activation、accept fd継承、restart/rollback。
- Linux `SO_PEERCRED`、UID/PID/executable、source IP/veth/network namespace/Tailscale identityからSubjectを作るproduction PeerBinder。binderのtrusted input取得と実OS照合。
- Agentごとのlistener割当、workspace/Agent/UID mappingの永続化、revocation、rate limit、global load shedding、監査/log/metric。
- CA certificate/private keyのfile lifecycle、trust install、config/CLI/service binary、process composition、signal handling、daemonization。
- 既存`connectsession`、`brokerhttp`、Exchange/Transaction/Forwarder/Policy/capability/credential/transport package、build/generated artifactの変更。
- 実listener、実Agent user、実network namespace、実gh/OpenAI client、VPSでのlive E2E。fake listener PASSで実到達制御又はidentityを保証しない。
- contextを無視して永久にreturnしない任意・悪性のin-process PeerBinder/SessionをServerが強制停止すること。そのような依存は注入しない。

### 受け入れ条件

- [ ] AC-1: `New`はcontextに協調するtrusted PeerBinder、trusted Session、`1..64`のMaxConcurrentだけを受理し、Binder/Session/上限以外の長寿命stateを持たないimmutable Serverを返す。nil/typed-nil dependency、範囲外上限、nil/zero/破損Server、nil context/listenerはpanicせず固定errorとなり、Format/errorにdependency、subject、address、下位errorを含めない。
- [ ] AC-2: `Serve`はlistenerを所有し、slot取得後だけAcceptして同時Session数をMaxConcurrent以下に保つ。各accepted connでPeerBinderを一回だけSession前に呼び、UID正数、1〜128 byte、先頭英数字・残り英数字又は`._-`だけのAgentInstanceID/WorkspaceIDを検証してcopyし、root由来のprivate context valueとしてSessionへ一回だけ渡す。binder拒否/invalid subjectではSessionへ到達せずconnをcloseしてacceptを継続する。
- [ ] AC-3: `Resolver`はServerがprivate keyへ束縛したsubjectだけを独立copyで返す。missing、wrong type、invalid又はcancelled contextを固定errorで拒否し、公開setterを持たず、RemoteAddr、listener address、HTTP/CONNECT/TLS/header又はcaller自己申告値からSubjectを生成・補完しない。並行connectionのsubject/contextを共有しない。
- [ ] AC-4: Session error又はbinder/session panicは当該connだけをcloseしてslotを解放し、acceptを継続する。予期しないAccept errorは新規受理を停止し、listener close、全connection cancel、協調的処理のdrain後に固定server errorだけを返す。retry/backoff、別listener、default binder/session、診断logを作らない。
- [ ] AC-5: caller cancel/deadlineはlistener closeでAcceptを解除し、全connection contextへ同じcancelを伝播して、協調的binder/Sessionのreturn後にnilで終了する。正常cancel、accept failure、per-connection failure/panicの全経路でServer自身のgoroutine、accepted conn、listenerを残さず、任意の非協調callback強制停止又はtimeout用leak goroutineを導入しない。
- [ ] AC-6: hermetic race testがprivate Resolver、binder-before-session、invalid/rejected subject、MaxConcurrent上限、複数connのidentity隔離、Session/binder panic・error後のaccept継続、unexpected Accept failure、caller cancel/deadlineとdrain、listener/conn closeを失敗検出する。focused package race、harness check/distcheck、README変更時lint、candidate launcherのroot `make check`がPASSし、base...candidate差分は800〜900行を目標、追加＋削除1,000行以下とする。

### 安定した参照

| 参照ID | 対象 | 固定改訂/ダイジェスト | 用途 |
|---|---|---|---|
| REF-1 | TASK-0051 connect session | candidate `9fd8c1e` / completion `cb10da1` | contextを透過して一connectionを所有する既存Session境界 |
| REF-2 | TASK-0049 broker HTTP Handler | candidate `d5b94f2` / completion `5f5b0eb` | context-only SubjectResolverの既存consumer |
| REF-3 | TASK-0032 capability registry | main `cb10da1`時点 | Subject identifier 1〜128 byte、UID正数の既存検証意味 |
| REF-4 | Development Agent harness設計 | main `cb10da1`時点 | Agent自己申告でないidentity、listener/OS/live境界 |

### 依存状態

| 依存 | 状態 (`ready` / `pending`) | planning参照 | `ready`後に固定する値 |
|---|---|---|---|
| TASK-0051 | `ready` | REF-1 | `Serve(context.Context, net.Conn)`だけへ委譲しsessionを変更しない |

### 許可パス

- `tools/dev-agent-harness/internal/brokerlistener/`
- `tools/dev-agent-harness/README.md`

### 完了経路preflight

| 確認対象 | 結果 | コマンドまたは根拠 |
|---|---|---|
| 完了checker | `ready` | planning/candidate/completionの3 commit gateとpost-merge `task-check`を使う |
| 権限 | `ready` | fake listener、net.Pipe、in-memory contextだけ。実bind、OS credential、外部network、秘密を使わない |
| 依存状態と参照 | `ready` | TASK-0051完了。Session interfaceだけを注入し既存packageを変更しない |
| 生成物の有無と更新方法 | `ready` | Go source/testとREADMEだけ。dependency、config、build、生成物なし |
| 割当ワークツリー | `ready` | `worktrees/TASK-0052-dev-agent-listener` |
| Lapログの書込・Schema・`repository annotation` | `ready` | 標準3 commits。新Schema、digest転記、追加機械checkなし |

### 未決事項

- なし

### `Dependency-ready reconciliation`

<!-- 依存ready時にMainがready参照、planning参照との差分、AC/設計/scope/QAへの影響、再承認結果を追記する。依存なし又は未readyならN/Aとする。 -->

- TASK-0051はplanning開始時点でready。Sessionは受理済みconnのCONNECT/TLS/HTTPだけを所有する。本Taskはその前段のaccept/concurrency/private identity contextだけを追加し、実socket/OS identityを後続境界へ残す。

## 背景

strict CONNECT/TLS Sessionまで完成したが、connectionを受理し、Agent自己申告ではない主体contextへ結び付け、並行処理と停止を所有するserverがない。実bindやLinux peer取得と同時に作るとOS/network環境差とaccept lifecycleが混ざるため、まず注入listenerとtrusted binderの有限なcompositionだけを固定する。

## 検討すべき設計観点

- identity valueはprivate context keyでのみ運び、`Resolver`に公開setterを置かない。PeerBinderは後続のOS-specific実装が唯一の生産者となる。
- 標準`net.Listener`、`context`、bounded semaphore、`sync.WaitGroup`だけで構成し、独自protocol、queue、retry/backoff、worker poolを作らない。
- 同時接続slotはAccept前に取得する。満杯時はkernel/listener backlogへ自然に戻し、受理後に即時拒否する別挙動を作らない。
- per-connection errorはサービス停止理由にせずfail closedでcloseする。Accept自体の予期しないfailureだけをserver failureとし、cancel起因closeとはcontextで区別する。
- Binder/Sessionはtrustedかつcontext協調の依存契約にする。Go callbackを外側からkillするための追加goroutine/timeoutは作らない。
- 前Taskの手戻りを踏まえ、初回candidateは800〜900行を目標にし、標準型のまま実装してremediation余白を残す。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] planning/candidate/completionの3 commit経路とcandidate一回のroot `make check`を満たしている。
- [ ] 同一candidateの独立REVIEW/QAを完了し、実listener/OS identity/network namespace/client/VPSのlive E2E未実施境界をPASSと誤記していない。
- [ ] 再利用可能な知識が生じた場合だけ意味Wikiへ同化し、post-merge `task-check`をPASSしている。

## 関連コンテキスト

### 意味 Wiki

- `wiki/semantic/schemas/development-agent-harness-egress-transaction.md`

### 判断

- Session前段に、bounded acceptとprivate trusted identity contextだけを所有するServerを置く。

### 適用しなかった重要な判断

- actual TCP/Unix listener、Linux peer credential、network namespace mappingを同時実装する案はhermetic acceptanceとlive OS挙動を混在させるため採用しない。
- per-connection goroutineをtimeoutで強制終了する案は非協調dependency時にleakを隠すため採用しない。
- custom queue/worker pool/accept retry/backoffは初期要件に不要な状態と挙動を増やすため採用しない。
