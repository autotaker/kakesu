---
task_id: "TASK-0065"
title: "Git read用credential helperを実装する"
status: draft
created_at: "2026-08-02"
---

# TASK-0065 Git read用credential helperを実装する

## `Planning input packet`（Main Agent所有）

このsectionをPlannerとQAへ渡す唯一の`planning input packet`とし、各内容をPLAN/QA_PLANへ複製しない。

### 目的

`git-credential-dev-agent`をGitのcustom credential helperとして実用化し、allowlist repositoryのHTTPS readに必要なGit-read Opaque capabilityだけを既存egress control socketから取得してGitへ返す。実GitHub tokenをAgent process、Git config、remote URL、credential cacheへ渡さず、TASK-0063のSmart HTTP read経路をGit clientから利用可能な直前まで成立させる。

### 対象と対象外

#### 対象

- Git helperの`get`でboundedなcredential inputを読み、`protocol=https`、exact `github.com[:443]`、canonical `owner/repo.git`だけからrepositoryを導出する。
- fixed control Unix socketへ一接続一操作でGit-read capability issue requestを送り、strictなhandle-only成功responseを検証する。
- 成功時はstdoutへ`username=x-access-token`と`password=cap_...`だけをcredential formatで返し、実token、socket path、repository、下位errorを診断へ出さない。
- `store`は何も保存せずsilent success、`erase`は入力のcanonical Opaque handleだけを同じcontrol socketで失効する。将来の未知operationはGit仕様どおりsilent ignoreする。
- default socket pathをconfigureの`runstatedir/dev-agent-harness/egress.sock`からlink時に固定し、Agent入力又はenvironmentで接続先を変更させない。
- net.Pipe/fake dialerとin-memory credential streamsによるhermetic testsで、wire bytes、順序、上限、close/deadline、output、拒否、非漏洩を確認する。

#### 対象外

- Git config/`credential.useHttpPath`/proxy/CA environmentの自動設定、launcher、実`git clone/fetch/pull`、redirectは後続Taskとする。
- `git-receive-pack`、push、Approval、push grant、Tailscale/Passkeyを実装しない。
- GitHub App token取得/置換、Capability Registry/control server/Smart HTTP policyを変更しない。
- credentialのdisk/cache/state保存、retry、redirect、別socket fallback、TCP、environment指定socket、prompt/askpassを追加しない。
- 実systemd socket、別UID、実GitHub/DNS/TLS、VPS配置をhermetic PASSで代替しない。
<!-- safety_contractの場合: 製品コード、test、runtime/build設定、Schema、製品依存、生成製品入力/成果物、外部観測可能な挙動を変更しない。 -->

### 受け入れ条件

<!-- AC-IDはTask内で一意かつ安定させ、観測可能な結果をここに一度だけ記載する。 -->

- [ ] AC-1: `get`は一個のoperation argと上限付きcredential inputだけを受理し、exact HTTPS GitHub hostとcanonical `owner/repo.git`から一意なrepositoryを導出する。duplicate/missing/conflicting context、URL属性、NUL/CR/overlong/extra bytes、別host/protocol/pathはbroker到達前にcredentialを返さない。
- [ ] AC-2: control clientはconfigure固定のabsolute Unix socketへtimeout/deadline付きで一回だけdialし、exact issue wireから`provider=github`、repository、`operation=github-git-read`を送り、唯一のbounded 200/JSON handle responseだけを受理する。chunked/extra header/body/bytes、malformed/noncanonical handle、204/403/5xx、early EOFは固定拒否する。
- [ ] AC-3: 成功`get`はGit credential formatでusername `x-access-token`とOpaque handleだけを返す。failure/unmatched contextは実token、handle、repository、path、socket、入力値、下位errorをstdout/stderrへ出さず、他helper/promptへcredential探索を広げないfail-closed結果を返す。
- [ ] AC-4: `store`は入力を永続化せずsilent success、`erase`はcanonical handle一件だけをexact DELETE wireで失効し、成功204だけを受理する。未知の一個operationはsilent ignoreし、zero/複数argは固定usage failureとする。
- [ ] AC-5: helperのsocket pathは`Makefile.in`のconfigure済み`runstatedir`からlink時に固定され、environment/credential input/CLI flagで変更できない。`--version`/`--help`、build/install/distcheckと他binaryの既存意味を維持する。
- [ ] AC-6: candidateは承認済み7パス・約900〜1,200行以内で、実token、push、launcher/config mutation、依存、Schema、Kakesu runtime、live stateを含まず、focused race、harness `make check`、root `make check`、`git diff --check`がPASSする。

### 安定した参照

| 参照ID | 対象 | 固定改訂/ダイジェスト | 用途 |
|---|---|---|---|
| REF-1 | Harness設計 §9.3、Phase 2 | main `db29a9c` | get/store/erase、Opaque password、Smart HTTP readの設計境界 |
| REF-2 | TASK-0061/0063 control protocolとGit read scope | main `db29a9c` | issue/revoke exact wireと既存single-use consume経路 |
| REF-3 | Git公式`gitcredentials`/`git-credential` manual | 2026-08-02確認 | operation arg、credential属性形式、blank/EOF終端、store/erase出力無視、未知operation silent ignore |

### 依存状態

| 依存 | 状態 (`ready` / `pending`) | planning参照 | `ready`後に固定する値 |
|---|---|---|---|
| なし | `ready` | N/A | N/A |

### 許可パス

- `tools/dev-agent-harness/cmd/git-credential-dev-agent/main.go`
- `tools/dev-agent-harness/internal/controlclient/client.go`（新規）
- `tools/dev-agent-harness/internal/controlclient/client_test.go`（新規）
- `tools/dev-agent-harness/internal/gitcredential/helper.go`（新規）
- `tools/dev-agent-harness/internal/gitcredential/helper_test.go`（新規）
- `tools/dev-agent-harness/Makefile.in`
- `tools/dev-agent-harness/README.md`

### 完了経路preflight

| 確認対象 | 結果 | コマンドまたは根拠 |
|---|---|---|
| 完了checker | `ready` | 通常product 3トランザクション、同一candidate独立REVIEW/QA、no-ff completionを使用する |
| 権限 | `ready` | Opaque auth出力・Unix protocol client・credential helper境界のため`dev-sol`、Mainだけがstage/commit/merge/pushする |
| 依存状態と参照 | `ready` | TASK-0061 control issue/revokeとTASK-0063 Git read scopeがmainへ反映済み |
| 生成物の有無と更新方法 | `ready` | Go source/testとMakefile.in/READMEのみ。configure再生成、dependency追加なし |
| 割当ワークツリー | `ready` | `worktrees/TASK-0065-git-credential-helper` |
| Lapログの書込・Schema・`repository annotation` | `not-applicable` | 新規log/Schema/annotationなし |

### 未決事項

- なし

### `Dependency-ready reconciliation`

<!-- 依存ready時にMainがready参照、planning参照との差分、AC/設計/スコープ/QAへの影響、再承認結果を追記する。依存なし又は未readyならN/Aとする。 -->

- N/A

## 背景

TASK-0063で`github.com`のupload-pack readとHTTP Basic Opaque capability置換は完成したが、scaffoldの`git-credential-dev-agent`はoperationを実装していない。Git clientはcapabilityを取得できないため、現状のread pathは直接fixture以外から利用できない。Git configやlive cloneを同時に扱うと環境依存範囲が広がるため、このTaskは標準helper protocolと既存control socket clientだけを単体QA可能な境界として完成させる。

## 検討すべき設計観点

- Git公式形式はblank line又はEOF終端と将来の未知operation silent ignoreを許す一方、security contextに使うprotocol/host/pathはduplicateやURL展開を許さず完全一致で決める。
- `get` failureで別helper/interactive promptへfallbackすると認証経路が広がるため、credentialを返さないだけでなくGitへ探索停止を伝える固定fail-closed outputを検討する。
- `store`はunsupported generatorとしてsilent ignoreし、入力credentialを解釈・保存・転送しない。`erase`だけは失敗した一時capabilityをboundedに失効する。
- server parserと規則を共有せず、client側は期待responseをstrictに検証する。実handle以外のserver bodyや下位errorを公開しない。
- control socket pathはinstall時に決まる値であり、Agent制御environmentで差し替えない。testabilityはproduction環境変数ではなくprivate dial seamで確保する。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] 選択した`change_class`の完了経路と`make check`を満たしている。
- [ ] 製品変更の場合: 実装、テスト、文書、同一案の独立REVIEW/QA、完了後の環境依存ケース確認が完了している。
- [ ] 安全契約変更の場合: Mainの意図・スコープ・受け入れ経路確認、契約検査、許可された統制文書差分の確認が完了している。

## 関連コンテキスト

### 意味 Wiki

- 未調査

### 判断

- 未調査

### 適用しなかった重要な判断

- なし
