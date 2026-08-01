---
task_id: "TASK-0036"
title: "Development Agent HarnessのUbuntu provision manifestを実装する"
status: done
created_at: "2026-08-01"
---

# TASK-0036 Development Agent HarnessのUbuntu provision manifestを実装する

## `Planning input packet`（Main Agent所有）

このsectionだけをPlannerとQAへ渡す。MainがTask分割、要件、受け入れ条件を所有し、Plannerは本Task内の計画だけを作る。

### 目的

`tools/dev-agent-harness`に、UbuntuへprovisionすべきOS user、directory、systemd serviceの望ましい状態を、外部作用なしで決定的JSONLへ出力する読み取り専用dry-runを追加する。実OSを変更する前に、ownerが設定とprovision対象を機械検査できる境界を固定する。

### 対象と対象外

#### 対象

- `dev-agent-harness-setup plan-provision --config PATH --target-root PATH`。
- 1行のmanifest header、3 user、4 directory、3 serviceの計10 actionを固定順序で出すversion 1 JSONL。
- userはhome `/nonexistent`、shell `/usr/sbin/nologin`、locked、home非作成。directoryはconfig/state/runtime/auditとmode/owner/group。serviceはbroker/egress/approvalをdisabled/stoppedで表す。
- logical absolute pathをabsoluteかつcleanな`target-root`配下へescape不能に写像する純粋関数。
- byte-for-byte決定性、writer error、引数/config/root不正、非漏洩、外部作用不在のunit/CLI test。

#### 対象外

- user/group/directory/fileの作成・変更、`systemd-sysusers`、`systemd-tmpfiles`、`systemctl`、sudo/PAM、service enable/start/restart。
- systemd unit、sysusers/tmpfiles template、configure/build/install targetの変更。
- namespace、mount、cgroup、firewall、network、Credential、IPC、永続state、audit書込み。
- Ubuntu実環境でのlive-e2e。executor実装時の別Taskへ残す。
- Kakesu本体のruntime、module、配布物、Schema、Plane契約への組込み。

### 受け入れ条件

- [x] AC-1: validなversion 1設定とabsolute/cleanな`target-root`でCLIはexit 0となり、header 1行とaction 10行のcanonical JSONLだけをstdoutへ出し、stderrを空にする。同一入力の複数実行はbyte-for-byte一致する。
- [x] AC-2: headerは`kind=manifest`、`version=1`、`platform=ubuntu`、`default=deny`、target root、`action_count=10`を持ち、actionは連番1〜10かつuser→directory→serviceの固定順序になる。
- [x] AC-3: 3 user actionは設定のagent/runtime/brokerをroleへ一意に対応させ、home `/nonexistent`、shell `/usr/sbin/nologin`、`locked=true`、`create_home=false`を必須とする。
- [x] AC-4: config/state/runtime/audit directory actionはlogical path、target-root配下のtarget path、4桁mode、owner/groupを持つ。configは`root:broker`、他3件は`broker:broker`、全modeは`0750`、auditはstate配下とする。
- [x] AC-5: 3 service actionはbroker、egress、approvalの固定名とbroker userを持ち、`enabled=false`、`started=false`である。Agent/Runtime userとしてserviceを起動するactionを出さない。
- [x] AC-6: target rootが空、相対、非clean、NULを含む場合、引数が不足/余剰の場合、または設定が不正な場合はnon-zeroかつstdout空となる。writer失敗もnon-zeroとし、同じmanifestをretry/re-emitしない。いずれもstderrへ入力path/user/config本文を出さない。
- [x] AC-7: CLI前後でtarget root配下とhost状態にfile作成・mode/owner変更がなく、外部process、network、IPCを開始しないことをtest seamと一時directory snapshotで証明する。他のsetup操作と5 binaryは既存どおりfail-closedする。
- [x] AC-8: `go test ./...`、harness `make check`、`make distcheck`、root docs lintがPASSし、negative mutationでorder、disabled/stopped、path containment、writer errorの各guardを弱めると対応testが失敗する。
- [x] AC-9: 手書き実装＋testは700〜1,200行を目安とする。1,200行超過またはexecutor/OS変更という新security boundaryが必要なら、Mainへ戻し、Task分割はプロセス改善で解決できない場合の最後の手段とする。

### 安定した参照

| 参照ID | 対象 | 固定改訂 | 用途 |
|---|---|---|---|
| REF-1 | `docs/development/development-agent-harness.md` | commit `14af73e`時点のPhase 0/配置境界 | 外部基盤、Ubuntu user/systemd、installとprovision分離 |
| REF-2 | TASK-0035 config foundation | main commit `fb60ba5`時点 | strict設定読込、CLI非漏洩、deny既定 |
| REF-3 | scaffold deploy template | main commit `fb60ba5`時点 | user名、directory、service名とdisabled start境界 |
| REF-4 | repository bootstrap manifest | commit `a063f6d461bbc6ce752d93306f83e4939e299d1e` / digest `279dc69dba63337208ac4d0dd065db8055e7bb0b00fb8df5e0f9024d9f283329` | candidate証跡binding |

### 依存状態

| 依存 | 状態 | planning参照 | 扱い |
|---|---|---|---|
| config foundation | `ready` | REF-2 | version 1 Configを再利用する |
| actual Ubuntu executor/live host | `pending` | 対象外 | 本Taskへ追加せず後続Taskで固定する |

### 許可パス

- `tools/dev-agent-harness/internal/provision/`
- `tools/dev-agent-harness/internal/command/`
- `tools/dev-agent-harness/README.md`

### 完了経路preflight

| 確認対象 | 結果 | 根拠 |
|---|---|---|
| hooks / clean main | `ready` | `core.hooksPath=.githooks`、TASK開始時main clean/current |
| bootstrap binding | `ready` | REF-4をHANDOVER/REVIEW/QAへcandidate固定時に継承する |
| 生成物 | `ready` | `configure`その他生成物を変更しない |
| docs lint | `ready` | README変更直後、candidate固定前にroot textlintを実行する |
| root check | `ready` | candidate固定前に`PYTEST_ADDOPTS=--ignore=worktrees make check`を一度実行する |
| worktree | `ready` | `worktrees/TASK-0036-dev-agent-provision-manifest` |

### 未決事項

- JSON field名と内部型名は、上記観測値を変えない範囲でPlannerが決める。

### `Dependency-ready reconciliation`

- actual Ubuntu executorがreadyになっても本Taskを拡張せず、manifest consumerの別Taskで再審査する。

## 背景と設計観点

scaffoldはsystemd/sysusers/tmpfilesを配布するが、OS変更前に「何を作るか」を機械的に確認する入口がない。直接executorを実装するとdry-runと実行経路が同時に増え、単体QAと権限境界が曖昧になる。まず副作用のないcanonical manifestを正本にし、後続executorが同じ望ましい状態だけをconsumeできるようにする。

- JSONLは1 record 1 actionとし、配列全体の再serializeやYAML parserを不要にする。
- encode前に全recordをmemoryへ組み立ててvalidateし、writer error前のpartial outputを避ける。
- target pathはstring連結せず、logical absolute pathからseparatorを除いてroot配下へjoinし、containmentを再確認する。
- dry-runという名称で将来の実行権限を暗黙付与しない。serviceは常にdisabled/stoppedで計画する。

## 完成の定義

- [x] AC-1〜AC-9、独立REVIEW/QA、final root check、Wiki ingest、no-ff merge、main pushが完了している。

## 関連コンテキスト

### 意味 Wiki

- 完了後、provision manifestのrecord/order、不変条件、外部作用なし、executorとの境界をSemantic Wikiへ取り込む。

### 判断

- manifestはJSONL version 1、executorは対象外、serviceはdisabled/stoppedを既定とする。

### 適用しなかった重要な判断

- shell command列の出力、YAML、実root変更、template編集は採用しない。
