---
task_id: "TASK-0058"
title: "外向き通信serviceのproduction compositionを実装する"
status: draft
created_at: "2026-08-02"
---

# TASK-0058 外向き通信serviceのproduction compositionを実装する

## `Planning input packet`（Main Agent所有）

このsectionをPlannerとQAへ渡す唯一の`planning input packet`とし、各内容をPLAN/QA_PLANへ複製しない。

### 目的

既存のpolicy、capability、credential、transport、HTTP/TLS session、Linux peer binding、systemd socket activation、runtime identityを一つの`dev-agent-egress` service起動へ接続する。設定とOS主体を検証してから秘密を一度だけ読み、固定した同じidentityとdependency graphでsocketを受領して、context cancelまでserveできるproduction compositionを作る。

### 対象と対象外

#### 対象

- config V1へ必須`egress.github_repositories`と`egress.openai_models`を追加する。各配列は1〜32件、重複なしで、値は既存`egresspolicy`と同じ規則だけを許す。unknown/duplicate/missing/empty/過剰件数/不正値を既存fixed classで拒否し、example、command/provision fixtureを同期する。
- config directory配下の固定`credentials` directoryをbroker-owned `0700`のdesired stateとしてprovision manifestへ追加する。action countは11へ更新するが、既存user、他directory、serviceの意味と順序を維持する。
- `internal/egressservice`へproduction compositionを追加する。configを一回読み、TASK-0057のruntime identityを一回解決してcurrent broker主体を確定した後だけ、`config_dir/credentials`からTASK-0053のbundleを一回読む。
- 一つのpolicy、空のcapability Registry、upstream transport、provider credential resolver、Exchange、context-only HTTP handler、proxy CA Session、PeerBinder、listener Serverを既存constructorだけで組み立てる。固定上限はpolicy version `egress-v1`、capability max TTL 10分/max uses 16/revocation epoch 1、body 64 KiB、output 4096 tokens、credential 4096 byte、provider timeout 10秒、forward timeout 30秒、response 1 MiB、同時connection 16とする。
- 同じresolved identityからsocket activationへbroker UID/agent GID、PeerBinderへagent UID/Subjectを渡す。全constructor成功後にsocket activationのFDを最後に一回だけ受領し、`brokerlistener.Server.Serve`へ渡す。失敗はpartial service/listenerを残さず固定非漏洩errorへ畳む。
- `dev-agent-egress serve --config PATH`を唯一のoperational起動面として追加する。mainはSIGINT/SIGTERMをcontext cancelへ変換する。systemd unitは固定config pathとsocket unitを宣言し、credential/env/capabilityをcommand lineやenvironmentへ渡さない。
- package/commandのhermetic test、Linux cross-compile、README境界を追加する。production constructor graphと起動順序、exact call、listener取得前失敗、cancel/Serve error、固定診断をfailure-detectする。

#### 対象外

- capabilityの発行・配布・管理IPC、起動時の既定capability、Approval連携。Registryは本Taskでは空で、未知handleはfail closedのままとする。
- `dev-agent-broker`/`dev-agent-approval`の実装、process間credential/registry IPC、永続化、audit、rotation/reload、restart復元。
- Agent側CA install、proxy environment、GH/OpenAI handle配布、Git credential helper、Git Smart HTTP、push/approval。
- 既存package内部のpolicy/credential/TLS/HTTP/socket/peer semantics変更、fallback/retry/cache、default HTTP client、追加goroutine、診断log。
- 実Linux NSS/UID/GID、systemd FD 3、socket permission、実秘密配置、実GitHub/OpenAI、DNS/TLS、Agent client、VPS live E2E。hermetic test/cross-compileで代替しない。
<!-- safety_contractの場合: 製品コード、test、runtime/build設定、Schema、製品依存、生成製品入力/成果物、外部観測可能な挙動を変更しない。 -->

### 受け入れ条件

<!-- AC-IDはTask内で一意かつ安定させ、観測可能な結果をここに一度だけ記載する。 -->

- [ ] AC-1: config V1は必須egress allowlistsをstrictにparse/validate/copyし、exampleとcommand/provision fixtureを同期する。provisionは`config_dir/credentials`をbroker-owned `0700`として一件だけ追加し、11 action以外の既存意味を変えない。
- [ ] AC-2: service startupはconfig load→runtime identity resolve→credential bundle load→trusted graph construction→socket Take→Serveの順序を一回だけ実行する。前段失敗では後段へ到達せず、socket取得後の所有権はServerへ一回だけ移る。
- [ ] AC-3: trusted graphは承認済み既存constructorと固定上限だけでpolicy/empty Registry/transport/resolver/exchange/handler/session/binder/serverを構成し、同じidentity resultをsocketとpeer境界へ供給する。default client、fallback、retry、cache、追加network/identity lookupを作らない。
- [ ] AC-4: `dev-agent-egress serve --config PATH`だけがoperational起動し、invalid args/config/identity/credentials/dependency/socket/Serve failureを固定exit/errorへ畳む。SIGINT/SIGTERM cancel、systemd config/socket wiring、no credential/env argumentを検証する。`--version`とno-args fail-closed契約を維持する。
- [ ] AC-5: hermetic testsは構築順序/exact call、identity共有、empty Registry denial、listener取得前後のfailure/cancel/ownership、fixed diagnosticsを失敗検出する。focused tests、Linux cross-compile、configured harness `make check`/`make distcheck`、root `make lint-docs`、candidate root `make check`、`git diff --check`がPASSし、許可path内の追加＋削除は概ね1,000行（上限1,100行）とする。

### 安定した参照

| 参照ID | 対象 | 固定改訂/ダイジェスト | 用途 |
|---|---|---|---|
| REF-1 | phase 1 egress core | main `14227b6` | capability/policy/credential/transport/exchange/HTTP/TLSの既存constructor契約 |
| REF-2 | TASK-0052/0055/0056/0057 | main `14227b6` | listener lifecycle、Linux PeerBinder、systemd socket activation、runtime identity |
| REF-3 | Development Agent Harness設計 | main `14227b6` | 外部開発基盤、phase 1、固定service/OS/秘密境界 |

### 依存状態

| 依存 | 状態 (`ready` / `pending`) | planning参照 | `ready`後に固定する値 |
|---|---|---|---|
| TASK-0053 | `ready` | REF-1 | proxy CAを含むbroker credential bundle |
| TASK-0057 | `ready` | REF-2 | 同一snapshotのbroker/agent UID/GIDとSubject |

### 許可パス

- `tools/dev-agent-harness/internal/egressservice/**`
- `tools/dev-agent-harness/internal/config/config.go`
- `tools/dev-agent-harness/internal/config/config_test.go`
- `tools/dev-agent-harness/internal/command/command.go`
- `tools/dev-agent-harness/internal/command/command_test.go`
- `tools/dev-agent-harness/internal/provision/provision.go`
- `tools/dev-agent-harness/internal/provision/provision_test.go`
- `tools/dev-agent-harness/cmd/dev-agent-egress/main.go`
- `tools/dev-agent-harness/config/harness.json.example.in`
- `tools/dev-agent-harness/deploy/systemd/dev-agent-egress.service.in`
- `tools/dev-agent-harness/README.md`

### 完了経路preflight

| 確認対象 | 結果 | コマンドまたは根拠 |
|---|---|---|
| 完了checker | `ready` | 標準planning/candidate/completionとpost-merge `task-check` |
| 権限 | `ready` | hermetic factory seamとcross-compile。実秘密/FD/systemd/networkは起動しない |
| 依存状態と参照 | `ready` | TASK-0053/0057と全transitive coreがmain `14227b6`で完了 |
| 生成物の有無と更新方法 | `ready` | `.in`だけを変更。`./configure`生成物はcandidateへ含めずdistcheck後にclean |
| 割当ワークツリー | `ready` | `worktrees/TASK-0058-dev-agent-egress-service` |
| Lapログの書込・Schema・`repository annotation` | `ready` | 新規log/Schema/checkなし、標準3 commits |

### 未決事項

- なし

### `Dependency-ready reconciliation`

<!-- 依存ready時にMainがready参照、planning参照との差分、AC/設計/scope/QAへの影響、再承認結果を追記する。依存なし又は未readyならN/Aとする。 -->

- TASK-0053/0057はready。bundleはproxy CA Authorityを、runtime identityはsocket/peer両境界の同一snapshotを供給する。本Taskでは意味を変更せずproduction compositionだけを追加する。

## 背景

外向き通信の各安全境界は個別に完成したが、配布済み`dev-agent-egress`は依然scaffoldで、service unitもconfig pathを渡さない。個別packageのPASSだけではconstructorの取り違え、異なるidentity snapshot、秘密読込前の主体未確認、socketの早すぎる取得を防げないため、capability配布より先に一つのfail-closed startup graphとして固定する。

## 検討すべき設計観点

- OS主体を秘密読込より先に検証し、外部作用を持つsocket Takeは全依存構築後の最後にする。
- config allowlistは既存egresspolicyの意味を再実装せず、そのconstructorで受理可能な値だけを許す。件数上限だけconfig境界で追加する。
- credential directoryはroot-owned config directoryの固定childとし、Agentに見えるconfig値やenvironmentから秘密pathを選ばせない。
- Registryは同じservice lifetimeに一つだけ作るが、本Taskではissuerを接続しない。空Registryのdenialを正常な安全境界としてtestし、静的な既定handleを作らない。
- production compositionのtest seamはpackage-privateに閉じ、任意factory/credential/identity APIを公開しない。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] 選択した`change_class`の完了経路と`make check`を満たしている。
- [ ] 製品変更の場合: 実装、テスト、文書、同一案の独立REVIEW/QA、完了後の環境依存ケース確認が完了している。
- [ ] 安全契約変更の場合: 独立計画レビュー、契約検査、許可された統制文書差分の確認が完了している。

## 関連コンテキスト

### 意味 Wiki

- `wiki/semantic/schemas/development-agent-harness-egress-transaction.md`
- `wiki/semantic/schemas/development-agent-harness-broker-credentials.md`
- `wiki/semantic/schemas/development-agent-harness-config.md`

### 判断

- phase 1の既存coreを`dev-agent-egress`一processへ合成する。別process broker IPCは後続で導入するまで暗黙実装しない。

### 適用しなかった重要な判断

- credentialsを`config_dir`直下へ置く案はroot-owned config fileとbroker-owned secret directoryのmode/owner契約を両立しにくいため、固定`config_dir/credentials`を使う。
- 起動時に既定capabilityを発行する案は配布先/主体/寿命のtrusted経路が未実装で、handleをenvironmentやfileへ漏らすため採用しない。
