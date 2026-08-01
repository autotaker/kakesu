---
task_id: "TASK-0057"
title: "workspace identityとLinux user/group解決境界を実装する"
status: plan
created_at: "2026-08-02"
---

# TASK-0057 workspace identityとLinux user/group解決境界を実装する

## `Planning input packet`（Main Agent所有）

このsectionをPlannerとQAへ渡す唯一の`planning input packet`とし、各内容をPLAN/QA_PLANへ複製しない。

### 目的

設定済みworkspace ID、Linuxのbroker/agent user・agent primary group、service起動ごとに生成するAgent instance IDを、一つのimmutable identityへ解決する。後続service compositionが同じ結果から`socketactivation`のbroker UID/agent GIDと`peerbinder`のexpected agent UID/Subjectを構築できるようにし、username、乱数、numeric IDの解決をservice本体へ散らさない。

### 対象と対象外

#### 対象

- config V1へ必須`identity.workspace_id`を追加する。1〜128 byte、先頭ASCII英数字、残りASCII英数字又は`._-`だけを許し、未知・重複・欠落・空・不正値を既存fixed error classで拒否する。exampleとconfig testを同期する。
- `internal/runtimeidentity`へimmutable Resolverを追加する。固定agent/broker usernameとworkspace IDをconstructorでcopy・検証し、`Resolve`ごとにLinux user/group lookupを各一回だけ行う。
- Linuxではbroker usernameのUIDがcurrent EUIDと完全一致してnon-root、agent usernameのUIDが正数でbrokerと異なり、そのprimary GIDが同名agent groupのGIDと完全一致する場合だけ受理する。numeric textはcanonical decimalかつGo int/Linux uint32へlosslessな値だけを許す。
- `crypto/rand`の16 byteからservice-lifetime用の新しい`agent-`+lowercase hex instance IDを一回生成する。結果はbroker UID、agent UID/GID、`brokerlistener.Subject{AgentInstanceID, AgentUID, WorkspaceID}`の相互整合したcopyだけをaccessorで返す。
- lookup、entropy、platform、receiver corruptionの失敗はpartial identityを返さず固定非漏洩errorへ畳む。Format/errorへusername、workspace、UID/GID、lookup/entropy errorを含めない。非Linuxは常にfail closedとする。
- platform-independent hermetic test、Linux adapter test source、Linux cross-compile、README境界を追加する。外部dependencyは追加しない。

#### 対象外

- service binary、socket activation、PeerBinder、brokerlistener/Session/Exchangeのcomposition又は起動・signal handling。
- capability発行・配布、AgentInstanceIDの永続化/復元、複数workspace/user mapping、runtime userの解決、username/GIDのfallback。
- `/etc/passwd`等の独自parser、LDAP/remote NSS設定、cache/retry/timeout goroutine、shell/getent command、root/sudo、sysusers変更。
- config version変更、既存paths/users/networkの意味変更、systemd unit、provision、credential、provider、HTTP/TLS、approval/push。
- 実Linuxの別broker/agent user/group、NSS、service restart、VPS live E2E。cross-compile又はfake lookupのPASSで代替しない。
<!-- safety_contractの場合: 製品コード、test、runtime/build設定、Schema、製品依存、生成製品入力/成果物、外部観測可能な挙動を変更しない。 -->

### 受け入れ条件

<!-- AC-IDはTask内で一意かつ安定させ、観測可能な結果をここに一度だけ記載する。 -->

- [ ] AC-1: config V1は必須`identity.workspace_id`をstrictにparse/validate/copyし、exampleと既存config command/provision testsを同期する。未知・重複・欠落・不正workspace IDを既存fixed classで拒否し、他fieldの意味とversionを変えない。
- [ ] AC-2: Resolver constructorはagent/broker usernameとworkspace IDをcopyして検証し、nil/zero/corrupt receiverを固定errorで拒否する。Linux `Resolve`はagent user、broker user、agent groupを各一回だけlookupし、canonical/lossless positive IDs、current non-root broker EUID、distinct agent UID、agent primary GID=同名group GIDだけを受理する。
- [ ] AC-3: `Resolve`は16 byteのcrypto entropyを一回だけ使う新しい`agent-`+32 lowercase hex IDと、相互整合するbroker UID、agent UID/GID、fresh Subject copyを返す。lookup/entropy/identity mismatchではpartial resultなし、retry/cache/goroutine/logなし、非Linuxはfail closedである。
- [ ] AC-4: hermetic testsはconstructor/copy/corruption、lookup exact call count/order非依存、numeric境界、EUID/root/distinct user/primary group mismatch、entropy exact length/call/failure、fresh ID/Subject copy、fixed diagnosticsを失敗検出する。Linux adapter sourceをcross-compileし、実NSS/別UIDを実行したとは扱わない。
- [ ] AC-5: focused tests、harness `make check`/`make distcheck`、root `make lint-docs`、candidate root `make check`、`git diff --check`がPASSする。許可path内の追加＋削除1,000行以下、外部dependency/config version/service compositionなしとする。

### 安定した参照

| 参照ID | 対象 | 固定改訂/ダイジェスト | 用途 |
|---|---|---|---|
| REF-1 | TASK-0055 PeerBinder | candidate `1a24ecf` / completion `e79306c` | expected agent UIDと一つのSubjectを受ける境界 |
| REF-2 | TASK-0056 socket activation | candidate `40ccc20` / completion `b65a686` | broker UID、agent GIDを受ける境界 |
| REF-3 | config V1 | main `b65a686`時点 | users/path/networkのstrict config契約 |
| REF-4 | Development Agent Harness設計 | main `b65a686`時点 | OS主体、workspace/Agent instance identity、実NSS/live境界 |

### 依存状態

| 依存 | 状態 (`ready` / `pending`) | planning参照 | `ready`後に固定する値 |
|---|---|---|---|
| TASK-0055 | `ready` | REF-1 | agent UIDとSubjectの出力契約 |
| TASK-0056 | `ready` | REF-2 | broker UIDとagent GIDの出力契約 |

### 許可パス

- `tools/dev-agent-harness/internal/runtimeidentity/**`
- `tools/dev-agent-harness/internal/config/config.go`
- `tools/dev-agent-harness/internal/config/config_test.go`
- `tools/dev-agent-harness/internal/command/command_test.go`
- `tools/dev-agent-harness/internal/provision/provision.go`
- `tools/dev-agent-harness/internal/provision/provision_test.go`
- `tools/dev-agent-harness/config/harness.json.example.in`
- `tools/dev-agent-harness/README.md`

### 完了経路preflight

| 確認対象 | 結果 | コマンドまたは根拠 |
|---|---|---|
| 完了checker | `ready` | 標準planning/candidate/completionとpost-merge `task-check` |
| 権限 | `ready` | fake lookup/entropy/EUIDとcross-compileのみ。root、実user変更、外部作用なし |
| 依存状態と参照 | `ready` | TASK-0055/0056完了、main `b65a686`へ固定 |
| 生成物の有無と更新方法 | `ready` | example templateのみ。generated `harness.json.example`はcandidateへ含めない |
| 割当ワークツリー | `ready` | `worktrees/TASK-0057-dev-agent-runtime-identity` |
| Lapログの書込・Schema・`repository annotation` | `ready` | 新規log/Schema/checkなし、標準3 commits |

### 未決事項

- なし

### `Dependency-ready reconciliation`

<!-- 依存ready時にMainがready参照、planning参照との差分、AC/設計/scope/QAへの影響、再承認結果を追記する。依存なし又は未readyならN/Aとする。 -->

- TASK-0055/0056はready。PeerBinderはexpected agent UID+Subject、socket activationはbroker UID+agent GIDを別々に要求する。本Taskは同じ解決結果から両方を供給し、service compositionは後続Taskへ残す。

## 背景

listener入口まで完成したが、configはusernameだけを持ち、OS numeric identityとworkspace/Agent instance identityを同じ起動境界で固定する実装がない。service mainで都度lookup・乱数生成すると、socket metadataとpeer UIDが異なる解決結果を使う余地が生じるため、composition前に小さいimmutable境界として確定する。

## 検討すべき設計観点

- workspace IDはroot-owned configに固定するが、Agent instance IDはservice起動ごとに新規生成し、restart後の古いcapabilityを別instanceへ持ち越さない。
- usernameは既存config規則を再利用し、production lookupはLinux adapterだけに置く。core test seamはpackage-privateで任意identity APIを公開しない。
- resolved resultは実credentialではないが、ordinary diagnosticsへusername/UID/workspaceを出さない。accessorは後続trusted compositionに必要な値だけをcopyして返す。
- real NSS、別UID/GID、sysusersとの一致はlive E2Eまで未確認とし、macOSの`os/user`結果をproduction保証にしない。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] planning/candidate/completionの通常3 commitsとfinal candidateのroot `make check`を満たしている。
- [ ] 同一candidateの独立REVIEW/QAを完了し、実Linux NSS/別UID/GID/VPS未実施境界をPASSと誤記していない。
- [ ] 安全契約変更の場合: 独立計画レビュー、契約検査、許可された統制文書差分の確認が完了している。

## 関連コンテキスト

### 意味 Wiki

- `wiki/semantic/schemas/development-agent-harness-egress-transaction.md`
- `wiki/semantic/schemas/development-agent-harness-capability-registry.md`

### 判断

- workspace IDはconfig固定、Agent instance IDはservice起動ごとに生成し、一つのresolved identityからsocket/peer両境界へnumeric identityを供給する。

### 適用しなかった重要な判断

- Agent instance IDをconfigへ固定するとrestart前後のcapability subjectを区別できないため採用しない。
- service mainが複数箇所でusername lookupする案はsocket metadataとpeer UIDに異なるsnapshotを使い得るため採用しない。
