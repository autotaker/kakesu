---
task_id: "TASK-0055"
title: "Linux SO_PEERCRED PeerBinderを実装する"
status: plan
created_at: "2026-08-02"
---

# TASK-0055 Linux SO_PEERCRED PeerBinderを実装する

## `Planning input packet`（Main Agent所有）

このsectionをPlannerとQAへ渡す唯一の`planning input packet`とし、各内容をPLAN/QA_PLANへ複製しない。

### 目的

既存`brokerlistener.PeerBinder`のproduction実装として、受理済みLinux Unix domain socketからkernelが返す`SO_PEERCRED`のUIDを一回取得し、listenerごとに固定したtrustedなAgent instance/workspace主体へ結び付ける。Agentが送るprotocol、header、address文字列又は自己申告値をidentity根拠にせず、UID不一致と取得不能を接続単位でfail closedにする。

### 対象と対象外

#### 対象

- `internal/peerbinder`のimmutable Binder。`New`は正数のexpected UIDと、既存`brokerlistener.Subject`互換のAgentInstanceID/WorkspaceIDを一つだけ固定し、デフォルトidentity又は複数UIDの曖昧なfallbackを持たない。
- `Bind(context.Context, net.Conn)`は具体的な受理済み`*net.UnixConn`だけを受け、Linuxでは`SyscallConn`経由で`SOL_SOCKET/SO_PEERCRED`を一回読む。kernel UIDがexpected UIDと完全一致し、contextが前後とも有効な場合だけ固定subjectの独立copyを返す。
- nil/typed-nil/corrupt Binder、nil/typed-nil/cancelled context、nil/non-Unix connection、peer取得失敗、UID不一致をpanicや下位詳細の露出なしに固定errorへ畳む。PID/GID、socket path、RemoteAddr、payloadはidentity又は公開結果へ使わない。
- Linux adapter、非Linux fail-closed adapter、platform-independentなreader seamによるhermetic unit test、Linux上だけで実Unix socket pairのpeer UIDを確認するintegration test。追加dependencyは使わない。
- READMEへ「listener単位の静的subject + kernel UID照合」という保証と、実UID分離・socket permission・namespace/VPSが未検証である境界を短く追記する。

#### 対象外

- Unix socketの作成、path、owner/mode、listen/bind、unlink、systemd socket activation、listener lifecycle、UIDのusername解決、config/CLI/service composition。
- 一つのsocketで複数Agent/workspaceを多重化するmapping、PID/executable/cgroup/namespaceによる識別、agent instance IDの生成・永続化・revocation。呼出元は一つのlistener/Binderを一つのsubject lifetimeへ対応させる。
- `brokerlistener`、Session、HTTP/TLS、capability、credential、provider、forwarder、policyの意味変更。
- TCP、vsock、Windows/macOSをproduction対応に見せるfallback。非Linuxは固定errorで拒否する。
- 実`dev-agent`/broker UID、実socket permission、systemd、network namespace、実クライアント、VPSでのlive E2E。Linux unit testのPASSを配置境界のPASSに置き換えない。
<!-- safety_contractの場合: 製品コード、test、runtime/build設定、Schema、製品依存、生成製品入力/成果物、外部観測可能な挙動を変更しない。 -->

### 受け入れ条件

<!-- AC-IDはTask内で一意かつ安定させ、観測可能な結果をここに一度だけ記載する。 -->

- [x] AC-1: `New`はUIDが正数かつplatformのUID範囲内で、AgentInstanceID/WorkspaceIDが1〜128 byte、先頭ASCII英数字、残りASCII英数字又は`._-`の一つのBindingだけを受理し、値とreaderを内部copyしたimmutable Binderを返す。invalid rules、nil/zero/corrupt Binderはpanicせず固定errorとなり、Format/errorへidentity、UID、socket、下位errorを含めない。
- [x] AC-2: `Bind`は具体的なnon-nil `*net.UnixConn`に対してpeer readerを一回だけ同期呼出しし、取得UIDとexpected UIDが完全一致する場合だけ固定した`brokerlistener.Subject`の独立copyを返す。non-Unix/wrapped/nil connection、reader failure、UID不一致では空subjectと固定errorを返し、RemoteAddr、socket path、payload、PID/GID又はcaller自己申告値からidentityを生成・補完しない。
- [x] AC-3: nil/typed-nil又は既にcancelled/deadline超過のcontextはreader前に拒否し、reader中にcancelされた場合もreader後にsubjectを返さない。readerをgoroutineへ逃がすtimeout、retry、cache、logを追加せず、呼出しごとに一回のboundedなOS照合を行う。
- [x] AC-4: Linux adapterは`*net.UnixConn.SyscallConn`の`Control`内で標準libraryの`GetsockoptUcred(fd, SOL_SOCKET, SO_PEERCRED)`だけを一回使い、control/getsockopt failureを固定errorへ畳む。非Linux adapterは常にfail closedとする。hermetic testはconstructor、exact UID、拒否、copy、context前後cancel、single call、固定診断を失敗検出し、Linux限定testは実Unix socketで接続peer UIDを確認する。Linux cross-compileもPASSする。
- [x] AC-5: focused package test、Linux cross-compile、harness `make check`/`make distcheck`、README lint、candidate launcherのroot `make check`、`git diff --check`がPASSする。変更は許可pathだけ、追加＋削除700行以下とし、外部dependency、設定、build/generated artifactを増やさない。

### 安定した参照

| 参照ID | 対象 | 固定改訂/ダイジェスト | 用途 |
|---|---|---|---|
| REF-1 | TASK-0052 broker listener | completion `c1b00a9` / candidate `0d8ccc8` | production PeerBinder interface、subject validation/copy、受理connection lifecycle |
| REF-2 | Development Agent harness設計 | main `ee19537`時点 | Agent自己申告でないOS identity、ユーザー分離、live境界 |
| REF-3 | Go `syscall` Linux API | Go 1.24 module toolchain | `UnixConn.SyscallConn`と`GetsockoptUcred`によるdependencyなしの実装境界 |

### 依存状態

| 依存 | 状態 (`ready` / `pending`) | planning参照 | `ready`後に固定する値 |
|---|---|---|---|
| TASK-0052 | `ready` | REF-1 | `PeerBinder` interface、Subject型、Server側の二重検証を固定する |

### 許可パス

- `tools/dev-agent-harness/internal/peerbinder/**`
- `tools/dev-agent-harness/README.md`

### 完了経路preflight

| 確認対象 | 結果 | コマンドまたは根拠 |
|---|---|---|
| 完了checker | `ready` | 標準planning/candidate/completionとpost-merge `task-check`を使用する |
| 権限 | `ready` | hermetic socket/testとcross-compileだけ。network、secret、sudo、外部作用なし |
| 依存状態と参照 | `ready` | TASK-0052 completionと現mainのinterfaceを確認済み |
| 生成物の有無と更新方法 | `ready` | Go source/testとREADMEだけ。生成物・依存更新なし |
| 割当ワークツリー | `ready` | `worktrees/TASK-0055-dev-agent-linux-peer-binder` |
| Lapログの書込・Schema・`repository annotation` | `ready` | 新規log/Schema/checkなし、標準3 commits |

### 未決事項

- なし

### `Dependency-ready reconciliation`

<!-- 依存ready時にMainがready参照、planning参照との差分、AC/設計/scope/QAへの影響、再承認結果を追記する。依存なし又は未readyならN/Aとする。 -->

- TASK-0052はready。既存ServerはPeerBinderをaccepted connectionごとに一回呼び、返却Subjectを別途検証・copyする。本Taskはそのinterfaceを変更せず、production binderによるOS UID取得とlistener単位のtrusted静的mappingだけを追加する。

## 背景

TASK-0052でaccept/concurrency/private context lifecycleは完成したが、唯一のidentity producerは注入interfaceのままであり、実Linux connectionをAgentのUIDへ束縛できない。socket作成・権限・service compositionまで同時に扱うとOS配置の副作用とidentity判定が混ざるため、まず受理済みUnix connectionに対するkernel UID照合を小さな独立境界として完成させる。

## 検討すべき設計観点

- UIDだけでは複数workspace/instanceを区別できないため、Binderをlistenerごとに一つのtrusted subjectへ固定する。多重化や自己申告mappingは採用しない。
- concrete `*net.UnixConn`以外を拒否し、TCPや任意wrapperが`SyscallConn`を実装していてもidentity根拠にしない。
- Linux syscallは小さなbuild-tag adapterへ隔離し、policy、copy、context、diagnosticはplatform-independent coreで検査する。非Linuxでdevelopment buildが成功してもproduction supportとは表現しない。
- context cancellationはOS call前後で確認する。即時のsocket option読取を別goroutineへ移したりretryしたりしない。
- 本Taskで新しい設定field、CLI、service、dependency、機械gateを追加しない。

## 完成の定義

- [x] 受け入れ条件を満たしている。
- [x] planning/candidate/completionの3 commitsとcandidate一回のroot `make check`を満たしている。
- [x] 同一candidateの独立REVIEW/QAを完了し、実UID分離/socket permission/namespace/VPSのlive E2E未実施境界をPASSと誤記していない。
- [ ] 安全契約変更の場合: 独立計画レビュー、契約検査、許可された統制文書差分の確認が完了している。

## 関連コンテキスト

### 意味 Wiki

- `wiki/semantic/schemas/development-agent-harness-egress-transaction.md`を候補とする。再利用可能な知識がなければreceiptだけとし、意味Wiki更新を強制しない。

### 判断

- OSが認証したUIDと、listener ownerが固定した一つのsubjectを組み合わせる。connection内容から主体を作らない。

### 適用しなかった重要な判断

- UIDからAgentInstanceID/WorkspaceIDをglobal mapで引く案は、一socketの多重化・lifetime/revocation stateを持ち込み本Taskの安全境界を広げるため採用しない。
- `RemoteAddr().String()`やsocket pathをidentityに使う案はkernel認証されたpeer主体を表さないため採用しない。
