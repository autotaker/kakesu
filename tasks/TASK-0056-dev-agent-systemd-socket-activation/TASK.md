---
task_id: "TASK-0056"
title: "systemd socket activationでAgent接続endpointを固定する"
status: plan
created_at: "2026-08-02"
---

# TASK-0056 systemd socket activationでAgent接続endpointを固定する

## `Planning input packet`（Main Agent所有）

このsectionをPlannerとQAへ渡す唯一の`planning input packet`とし、各内容をPLAN/QA_PLANへ複製しない。

### 目的

TASK-0055のLinux PeerBinderへ渡す受理済みUnix listenerを、processによるpath作成・stale unlinkではなくsystemd socket activationから一回だけ取得する境界を作る。systemdが固定path、owner/group/mode、停止時cleanupを所有し、broker userのserviceは環境とFD 3、Unix listener種別、path、filesystem metadataを検証してから既存`brokerlistener.Server`へ渡せる状態にする。Agentにはsocketへのconnectだけを許し、runtime directoryの作成・削除権限を与えない。

### 対象と対象外

#### 対象

- `internal/socketactivation`のimmutable Receiver。固定`LISTEN_PID=current pid`、`LISTEN_FDS=1`、`LISTEN_FDNAMES=egress`とFD 3を一回だけ消費し、具体的な`*net.UnixListener`、固定`runtime_dir/egress.sock`だけを返す。成功/認識済み失敗ではactivation environmentを消去し、複製後のoriginal FDをcloseする。
- Linuxでは現在EUID、runtime directory、socket nodeを一descriptor/metadata境界で検証する。broker UIDは正数かつ現在EUIDと一致、runtime directoryはabsolute/cleanな実directory、owner broker UID、group agent GID、mode `0710`。socketはsymlinkでないUnix socket、owner broker UID、group agent GID、mode `0660`で、listener addressが同じ固定pathであることを要求する。
- 非Linux、missing/duplicate/malformed activation env、FD conversion/type/address/metadata不一致、root broker、UID/GID/path不正を固定errorでfail closedにする。TCP、追加FD、fallback bind、path unlink、retry、logを使わない。
- `deploy/systemd/dev-agent-egress.socket.in`を追加し、`ListenStream=@runstatedir@/dev-agent-harness/egress.sock`、`SocketUser=@broker_user@`、`SocketGroup=@agent_user@`、`SocketMode=0660`、`FileDescriptorName=egress`、`RemoveOnStop=yes`を固定する。installはunitを配置するだけでenable/startしない。
- runtime directoryのdesired stateを`broker:agent 0710`へ変更し、tmpfilesとprovision manifest/validator/testを一致させる。state/config/audit directoryの既存ownership/modeは変えない。
- configure/build/install/distへsocket unitを追加し、生成済み`configure`を正規に再生成する。platform-independent core test、Linux adapter/socket test source、Linux cross-compile、distribution test、README境界を追加する。外部dependencyは追加しない。

#### 対象外

- `cmd/dev-agent-egress`又は他binaryへのReceiver/PeerBinder/Server/Sessionのcomposition、service通常起動、signal handling、capability発行経路。
- process自身の`ListenUnix`、chmod/chown、stale socket検出/削除、Agentにruntime directory writeを与える設計、複数socket/FD、abstract namespace、TCP/vsock。
- systemd unitのenable/start/restart、実systemd manager、実Agent/broker別UID/GID、実socket connect/permission、namespace/VPSのlive E2E。unit/parser/hermetic test又はcross-compile PASSで代替しない。
- config schemaへのsocket path/mode/UID/GID field追加、username→numeric UID/GID解決、sysusersのuser/group生成意味変更。
- credentials、HTTP/TLS、policy、capability、provider、forwarder、approval/pushの変更。
<!-- safety_contractの場合: 製品コード、test、runtime/build設定、Schema、製品依存、生成製品入力/成果物、外部観測可能な挙動を変更しない。 -->

### 受け入れ条件

<!-- AC-IDはTask内で一意かつ安定させ、観測可能な結果をここに一度だけ記載する。 -->

- [x] AC-1: `New`はabsolute/clean runtime directory、正数かつplatform範囲内のbroker UID/agent GID、固定basenameからimmutable Receiverを作り、root broker、不正ID、invalid path、nil/zero/corrupt Receiverを固定非漏洩errorで拒否する。Format/errorへpath、UID/GID、environment、FD、下位errorを含めない。
- [x] AC-2: `Take`は`LISTEN_PID`が現在PIDのcanonical decimal、`LISTEN_FDS=1`、`LISTEN_FDNAMES=egress`の完全一致時だけFD 3を一回取得する。認識済みactivation environmentを一回消去し、`net.FileListener`が複製した後はoriginal FDをcloseする。concrete UnixListener以外、追加/欠落/不正env、conversion/close failureを拒否し、fallback listener、retry、cache、別goroutineを作らない。
- [x] AC-3: Linux metadata検証はcurrent EUID=broker UIDかつ非root、runtime directory=`directory, broker:agent, 0710`、固定socket node=`Unix socket, broker:agent, 0660`、listener network/path完全一致の場合だけlistenerを返す。検証後失敗では取得listenerをcloseし、pathをunlinkしない。非LinuxはFD/envからlistenerを返さない。
- [x] AC-4: socket unitは固定path/User/Group/Mode/FDName/RemoveOnStopの一socketだけを宣言し、runtime desired stateはtmpfilesとprovisionの双方で`broker:agent 0710`となる。他3 directory action、user/service action、deny default、install時非enable/startを維持し、configure/dist/install/uninstallへunitを過不足なく含める。
- [x] AC-5: hermetic testsはconstructor、env exactness/one-shot clear、single FD conversion、original/listener ownership、Unix type/address、directory/socket metadata、failure cleanup、fixed diagnosticsを失敗検出する。Linux build-tag testはactual Unix listener metadata readerを検査し、Linux cross-compileする。focused tests、harness `make check`/`make distcheck`、root `make lint-docs`、candidate launcher root `make check`、`git diff --check`がPASSし、許可path内の追加＋削除1,000行以下とする。

### 安定した参照

| 参照ID | 対象 | 固定改訂/ダイジェスト | 用途 |
|---|---|---|---|
| REF-1 | TASK-0055 PeerBinder | candidate `1a24ecf` / completion `e79306c` | concrete accepted UnixConn、exact peer UID、connection非所有の既存境界 |
| REF-2 | TASK-0036 provision manifest | completion `c69bafb` / main `e79306c`時点 | runtime directoryを含むcanonical desired-state正本 |
| REF-3 | harness build/deploy templates | main `e79306c`時点 | 配置のみでenable/startしないsystemd/tmpfiles/configure契約 |
| REF-4 | Development Agent harness設計 | main `e79306c`時点 | Agent user分離、privileged socket禁止、実OS/live境界 |

### 依存状態

| 依存 | 状態 (`ready` / `pending`) | planning参照 | `ready`後に固定する値 |
|---|---|---|---|
| TASK-0055 | `ready` | REF-1 | Receiverが返すUnixListenerを後続compositionでPeerBinder/Serverへ渡す |
| TASK-0036 | `ready` | REF-2 | action orderingを維持しruntime directoryのgroup/modeだけを変更する |

### 許可パス

- `tools/dev-agent-harness/internal/socketactivation/**`
- `tools/dev-agent-harness/internal/provision/provision.go`
- `tools/dev-agent-harness/internal/provision/provision_test.go`
- `tools/dev-agent-harness/deploy/systemd/dev-agent-egress.socket.in`
- `tools/dev-agent-harness/deploy/tmpfiles/dev-agent-harness.conf.in`
- `tools/dev-agent-harness/Makefile.in`
- `tools/dev-agent-harness/configure.ac`
- `tools/dev-agent-harness/configure`
- `tools/dev-agent-harness/README.md`

### 完了経路preflight

| 確認対象 | 結果 | コマンドまたは根拠 |
|---|---|---|
| 完了checker | `ready` | 標準planning/candidate/completionとpost-merge `task-check`を使用する |
| 権限 | `ready` | fake source/temporary Unix listenerとcross-compileだけ。実systemd、sudo、外部作用なし |
| 依存状態と参照 | `ready` | TASK-0036/0055完了。既存manifestとPeerBinder境界を固定済み |
| 生成物の有無と更新方法 | `ready` | `configure.ac`変更後にrepo所定の`autoconf`で`configure`だけ再生成する |
| 割当ワークツリー | `ready` | `worktrees/TASK-0056-dev-agent-systemd-socket-activation` |
| Lapログの書込・Schema・`repository annotation` | `ready` | 新規log/Schema/checkなし、標準3 commits |

### 未決事項

- なし

### `Dependency-ready reconciliation`

<!-- 依存ready時にMainがready参照、planning参照との差分、AC/設計/scope/QAへの影響、再承認結果を追記する。依存なし又は未readyならN/Aとする。 -->

- TASK-0055とTASK-0036はready。PeerBinderはsocketを作成/closeせずconcrete accepted UnixConnだけを認証し、provisionはruntime directoryを現在`broker:broker 0750`とする。本Taskはsystemd作成済みlistenerの取得と、Agentがconnect可能だがdirectoryを書けない`broker:agent 0710`へのdesired-state変更だけを行う。

## 背景

PeerBinderまで完成したが、Agentが到達できるUnix listenerがなく、現runtime directoryは`broker:broker 0750`のためAgentはsocket pathを探索できない。process-created socket方式は固定pathが公開された瞬間のmode、stale inode、unlinkの競合、Agentへのdirectory writeを新たに所有する。systemdにpath/mode/cleanupを委譲し、serviceは継承FDの検証と所有権移管だけを行う方が既存fail-closed境界を小さく保てる。

## 検討すべき設計観点

- runtime directoryは`broker:agent 0710`とし、brokerにowner全権、Agent groupにpath traverseだけを与える。socketはsystemdが`broker:agent 0660`で作り、Agentへconnectに必要な権限だけを与える。
- activation environmentは文字列を緩くparseせずcanonical exact値だけを受ける。recognized activationはone-shotで消費し、同じFDを二回取り込まない。
- public test injectionや任意FD/path APIを作らず、OS/env/FD seamはpackage-privateにする。
- filesystem metadataとlistener addressを両方確認し、TCP、renamed/wrong path、permission driftを拒否する。ただし実systemd managerの発行事実はlive E2Eまで主張しない。
- `configure`の機械差分を含めても1,000行を超える場合は、テストを削らずTaskを拡張せず、生成差分と実装scopeをMainへ再提示する。

## 完成の定義

- [x] 受け入れ条件を満たしている。
- [x] planning/candidate/completionを基準とし、独立REVIEW/QAが検出した手戻り修正candidateを履歴化して、final candidateのroot `make check`を満たしている。
- [x] 同一candidateの独立REVIEW/QAを完了し、実systemd/別UID/socket permission/VPSのlive E2E未実施境界をPASSと誤記していない。
- [ ] 安全契約変更の場合: 独立計画レビュー、契約検査、許可された統制文書差分の確認が完了している。

## 関連コンテキスト

### 意味 Wiki

- `wiki/semantic/schemas/development-agent-harness-provision-manifest.md`
- `wiki/semantic/schemas/development-agent-harness-egress-transaction.md`

### 判断

- socket path作成・mode・cleanupはsystemd、service processは継承FDの検証とGo listenerへの所有権移管だけを担当する。

### 適用しなかった重要な判断

- process-created socketはAgentにdirectory writeを与えるかbrokerのchmod/unlink/stale recoveryを必要とし、path差替え面を増やすため採用しない。
- runtime directoryをworld-traversable又はAgent書込可能にする案は、他runtime artifactの到達面とprecreate攻撃を増やすため採用しない。
