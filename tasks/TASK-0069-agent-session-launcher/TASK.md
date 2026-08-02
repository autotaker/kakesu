---
task_id: "TASK-0069"
title: "Agent session launcherを実装する"
status: draft
created_at: "2026-08-02"
---

# TASK-0069 Agent session launcherを実装する

## `Planning input packet`（Main Agent所有）

このsectionをPlannerとQAへ渡す唯一の`planning input packet`とし、各内容をPLAN/QA_PLANへ複製しない。

### 目的

`dev-agent-launcher`から一つのcoding-agent sessionを起動し、GitHub REST、OpenAI Responses、Git Smart HTTP readを実認証情報なしで利用できるようにする。既存control client、loopback bridge、Git credential helperを組み合わせ、Opaque capability、公開proxy CA、限定environment、child process、失効と一時ファイルcleanupを一つのfail-closed lifecycleへ束縛する。Codex自身の認証情報は例外としてagent userの`HOME`又は`CODEX_HOME`からchildが直接読むことを許すが、launcherは内容を読取・複製・出力しない。

### 対象と対象外

#### 対象

- `dev-agent-launcher run --repository owner/repo -- COMMAND [ARG...]`だけをoperational CLIとして追加し、repositoryと引数をstrictに検査してshellを介さず一つのchildを起動する。
- link-timeで固定したegress Unix socketとGit credential helper pathだけを使い、公開proxy CA、GitHub REST capability、OpenAI capabilityを各一回取得する。runtime path、provider、operation、model、proxy endpointをAgent入力にしない。
- 固定IPv4 loopback bridgeを一つ起動し、session専用の0700 temporary directoryと0600 CA fileを固定`/tmp`配下に作る。child終了・起動失敗・cancel・初期化失敗の全経路でbridgeをcancel/drainし、CA directoryを削除し、発行済みAPI handleを各一回失効要求する。
- 親environmentから必要最小限の非API値だけを選び、`GH_TOKEN`と`OPENAI_API_KEY`にはOpaque handle、HTTP(S) proxyとCA変数にはsession専用値を設定する。Gitへhelper reset＋固定helper、GitHub path-aware credential、proxy、CAをcommand-scope configとして渡し、対話credential promptを無効化する。
- context cancelをchildへ反映し、stdioを透過し、正常終了・child exit・固定launcher failureを区別する。errorとdiagnosticへhandle、repository、socket/helper/temp path、environment値、child/lower-level errorを出さない。
- hermetic testsでCLI、environment再構築、初期化順序、child/bridge lifecycle、全failure cleanup、non-leakを検出し、build/install wiringとREADMEの保証限界を更新する。

#### 対象外

- launcher又はproxy environmentをnetwork隔離の強制境界とみなさない。実OSのdefault-deny firewall/network namespace、loopback isolation、Unix socket ownership/peer UID、systemd/VPS配置は後続live E2Eで扱う。
- 実GitHub/OpenAI/Codex credential、実`git`/`gh`/Codex/OpenAI SDK、DNS/TLS、外部HTTPをhermetic PASSで代替しない。
- Git push/write、GitHub GraphQL/write、OpenAI upload/admin/files、Approval、承認待ち、grant更新、長期session renew、retry、cache、永続化、auditを追加しない。
- Git credential helper、control protocol、capability budget/policy、proxy bridge、egress service、provider credential置換の既存意味を変更しない。
- Codex認証内容をlauncherが取得、検証、移動、environmentへ展開しない。親から任意credential/proxy/Git/loader environmentを継承しない。
- config Schema、依存、Kakesu本体runtime/Schema、systemd/sysusers/tmpfiles/provision manifest、生成済み`configure`、live stateを変更しない。
<!-- safety_contractの場合: 製品コード、test、runtime/build設定、Schema、製品依存、生成製品入力/成果物、外部観測可能な挙動を変更しない。 -->

### 受け入れ条件

<!-- AC-IDはTask内で一意かつ安定させ、観測可能な結果をここに一度だけ記載する。 -->

- [ ] AC-1: CLIはexact `run --repository owner/repo -- COMMAND [ARG...]`と`--help`/`--version`だけを受理する。repositoryはcanonical lowercase `owner/repo`、commandは空でないことを要求し、unknown/missing/duplicate/reordered option、追加launcher option、NUL、不正repositoryはchild又はcontrol dial前にusage failureとなる。childはshellを介さず引数境界とstdioを保持する。
- [ ] AC-2: sessionはabsolute cleanなlink-time fixed control socket/helper pathを起動前に検査し、公開CA、GitHub REST handle、OpenAI handleを既存strict clientから各一回だけ取得する。任意socket/provider/operation/model/endpointを入力できず、途中失敗時は以後の初期化又はchildを開始せず、既に発行したhandleだけを各一回失効要求する。
- [ ] AC-3: bridgeは一つのIPv4 loopback endpointだけを返し、CAはliteral `/tmp`直下のfresh 0700 directory内の0600 regular fileへ一回だけ書く。symlink/既存path/親`TMPDIR`を使わない。setup failure、child start failure、nonzero/normal exit、context cancelの全経路で新規acceptを止め、active bridgeをdrainし、CA directoryを再試行なしで削除し、発行済みAPI handleを各一回失効要求する。
- [ ] AC-4: child environmentは親から`HOME`、`PATH`、`TERM`、`LANG`/`LC_*`、任意の`CODEX_HOME`だけを選択して再構築し、他の`GH_*`、`OPENAI_*`、proxy/CA、Git config、credential、SSH、loader/runtime injection値を継承しない。sessionはOpaque `GH_TOKEN`/`OPENAI_API_KEY`、upper/lower HTTP(S) loopback proxy、CA trust変数、credential prompt無効化、command-scope Git configだけを決定的かつ重複なしで設定する。Git configは既存helper列をempty valueでresetしてからabsolute fixed helperだけを使い、GitHub `useHttpPath=true`、proxy、CAを固定する。
- [ ] AC-5: context cancelはchildを停止してwaitし、bridge cleanup後だけlauncherが戻る。正常childは0、通常のnonzero childはそのexit code、signal/開始/初期化/cleanup failureは固定非zeroへ畳む。diagnosticはusage又は固定`session failed`だけで、handle、repository、socket/helper/temp path、environment、command、lower errorを含めない。revokeがunknown/期限切れ等で失敗してもchildの既存exit結果を秘密付き診断へ置換せず、TTLを残余fail-safeとする。
- [ ] AC-6: candidateは承認済み7パス・約850〜1,200 changed linesを目安とし、実credential、外部通信、既存egress/control/helper/policy実装、config/dependency/Schema/deploy/generated/live stateを含まない。focused race、harness `make check`/`make distcheck`、install stagingでlauncher/helperのlink-time path、candidate gate root `make check`、`git diff --check`がPASSする。

### 安定した参照

| 参照ID | 対象 | 固定改訂/ダイジェスト | 用途 |
|---|---|---|---|
| REF-1 | Harness設計 §7.2、§9 | main `c584160` | Opaque auth、Codex credential例外、proxy置換、default-deny外側境界 |
| REF-2 | TASK-0065 / TASK-0066 | main `c584160` | Git helper、fixed control socket、公開proxy CA取得client |
| REF-3 | TASK-0067 / TASK-0068 | main `c584160` | loopback bridge、GitHub REST/OpenAI API capability clientとbudget |

### 依存状態

| 依存 | 状態 (`ready` / `pending`) | planning参照 | `ready`後に固定する値 |
|---|---|---|---|
| TASK-0065 / TASK-0066 / TASK-0067 / TASK-0068 | `ready` | main `c584160` | helper/control socket、CA client、bridge、API issue clientの公開contract |

### 許可パス

- `tools/dev-agent-harness/internal/launchsession/session.go`
- `tools/dev-agent-harness/internal/launchsession/session_test.go`
- `tools/dev-agent-harness/internal/command/command.go`
- `tools/dev-agent-harness/internal/command/command_test.go`
- `tools/dev-agent-harness/cmd/dev-agent-launcher/main.go`
- `tools/dev-agent-harness/Makefile.in`
- `tools/dev-agent-harness/README.md`

### 完了経路preflight

| 確認対象 | 結果 | コマンドまたは根拠 |
|---|---|---|
| 完了checker | `ready` | product 3トランザクション、同一candidate独立REVIEW/QA、no-ff completionを使用する |
| 権限 | `ready` | Opaque credential environmentとprocess cleanupを扱う高リスク境界なので`dev-sol`/highを使う。Mainだけがstage/commit/merge/pushする |
| 依存状態と参照 | `ready` | TASK-0065〜0068がmainへ反映済み |
| 生成物の有無と更新方法 | `ready` | Go source/test、Makefile.in、READMEのみ。`configure`又は配布生成物は更新せず`make distcheck`で再生成可能性を確認する |
| 割当ワークツリー | `ready` | `worktrees/TASK-0069-agent-session-launcher` |
| Lapログの書込・Schema・`repository annotation` | `not-applicable` | 新規log/Schema/annotationなし |

### 未決事項

- なし

### `Dependency-ready reconciliation`

<!-- 依存ready時にMainがready参照、planning参照との差分、AC/設計/スコープ/QAへの影響、再承認結果を追記する。依存なし又は未readyならN/Aとする。 -->

- N/A

## 背景

TASK-0065〜0068でGit credential helper、公開CA取得、loopback bridge、GitHub REST/OpenAI capability取得が個別に完成したが、現在の`dev-agent-launcher`はscaffoldであり、Agent clientへそれらを安全に束縛するsession lifecycleがない。部品単体のstrictnessだけでは、親credential environmentの漏洩、初期化途中のhandle残留、一時CAの残留、bridge goroutineの残留、shell経由の引数改変を防げない。このTaskは外側network isolationを偽って保証せず、既存部品を一回のprocess lifecycleへ統合する最小境界を完成させる。

## 検討すべき設計観点

- API handleはchild environmentにのみ置き、launcherのerror、log、argv、diskへ書かない。公開CAだけをdiskへ置き、handleとは異なるcleanup特性を明示する。
- cleanupは逆順かつboundedに行い、partial initializationでも作成済み資源だけを一回処理する。revoke失敗とlocal bridge/file cleanup失敗を混同しない。
- environment allowlistは「危険な既知キーを削るdenylist」ではなく、必要な名前だけをcopyするallowlistにする。Codex credential例外は`HOME`/`CODEX_HOME`の場所を保つだけで、secret bytesをlauncherに読ませない。
- Git/gh/OpenAI clientがproxy設定を無視できるため、このlauncherは利便的なroutingとsecret置換境界であり、network enforcementは外側default-denyが所有する。READMEとQAはこの限界を過大評価しない。
- Git credential helperは操作ごとにsingle-use handleを取得する既存設計を維持し、launcherがGit-read handleを事前発行又はcacheしない。Git pushはこのsessionでも拒否されたままにする。

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
