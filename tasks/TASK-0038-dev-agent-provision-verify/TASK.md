---
task_id: "TASK-0038"
title: "Provision manifestのstrict consumerを実装する"
status: plan
created_at: "2026-08-01"
---

# TASK-0038 Provision manifestのstrict consumerを実装する

## `Planning input packet`（Main Agent所有）

このsectionをPlannerとQAへ渡す唯一の`planning input packet`とし、各内容をPLAN/QA_PLANへ複製しない。

### 目的

`TASK-0036`が生成するUbuntu provision manifestを、OSへ適用する前にstrictに読み取り、同じ設定とtarget rootから再構築したcanonical manifestとの完全一致を検証する副作用のないCLIを追加する。executorへ不正、非canonical、別設定由来のmanifestが渡ることをfail-closedで防ぐ境界を固定する。

### 対象と対象外

#### 対象

- `dev-agent-harness-setup verify-provision --config PATH --manifest PATH --target-root PATH`。
- manifestをfile descriptorから安全に読み、regular file、non-symlink、128 KiB以下、group/world writableでないことを検査する。
- `provision.Build(config, target-root)`を望ましい状態とV1 encodingの唯一の正本として再利用し、入力manifestとcanonical bytesを完全一致で照合する。独立parserや二重のV1 schemaは作らない。
- 成功時は固定された非機密summaryだけをstdoutへ出し、失敗時は固定error classだけをstderrへ出す。
- file policy、canonical equality、診断非漏洩、副作用不在をunit/CLI testで検証する。

#### 対象外

- manifestの適用、user/group/directory/fileの作成・変更、`systemd-sysusers`、`systemd-tmpfiles`、`systemctl`、sudo/PAM、service enable/start/restart。
- root権限、Ubuntu実環境、live-e2e、rollback、部分適用state、audit永続化。
- namespace、mount、cgroup、firewall、network、Credential、IPC、承認、push grant。
- systemd unit、sysusers/tmpfiles template、configure/build/install target、依存packageの変更。
- Kakesu本体のruntime、module、配布物、Schema、Plane契約への組込み。

### 受け入れ条件

<!-- AC-IDはTask内で一意かつ安定させ、観測可能な結果をここに一度だけ記載する。 -->

- [ ] AC-1: validなversion 1設定、absolute/cleanな`target-root`、TASK-0036が生成した安全なmanifest fileに対してCLIはexit 0となり、stdoutへ`provision manifest version=1 actions=10 verified`と改行だけを出し、stderrを空にする。
- [ ] AC-2: verifierは同じconfigとtarget rootから`provision.Build`で再構築したcanonical bytesとの完全一致だけを受理条件とする。したがって行数、field、型、空白、record順、値、config、target root、終端改行のいずれかが異なる入力をnon-zeroで拒否し、独立JSON parser、別schema、別の望ましい状態実装を追加しない。
- [ ] AC-3: manifest pathは一度だけopenし、そのfile descriptorから検査・読取する。symlink、non-regular file、128 KiB超、group/world writable file、読取前後のsize/mode変更を拒否し、検査済みpathを開き直さない。
- [ ] AC-4: 引数不足/余剰、設定不正、manifest不一致、file policy違反、read失敗はnon-zeroかつstdout空となる。stderrは固定error classだけを含み、入力path、user名、config/manifest本文、JSON値、OS error本文を出さない。
- [ ] AC-5: CLIはmanifest、config、target rootその他host stateを書き換えず、外部process、network、IPCを開始しない。他のsetup操作と5 binaryは既存どおりfail-closedする。
- [ ] AC-6: unit/CLI testはvalid case、代表的な1 byte追加/変更/削除、config/root mismatch、symlink/type/mode/size、読取中変更、非漏洩、副作用不在を検出する。JSON grammarごとの網羅matrixやproduction parserは要求しない。
- [ ] AC-7: harness `make check`、`make distcheck`、root `make check`がPASSし、生成済み`configure`とinstall surfaceは変わらない。
- [ ] AC-8: executor/実OS/root権限という新security boundary、許可外path、または1,200行超過が必要ならMainへ戻す。行数の下限、見積算術、mutation専用証跡は完了条件にしない。

### 安定した参照

| 参照ID | 対象 | 固定改訂/ダイジェスト | 用途 |
|---|---|---|---|
| REF-1 | `docs/development/development-agent-harness.md` | main commit `e15055b`時点 | Phase 0、OS identity、provisionとruntimeの境界 |
| REF-2 | TASK-0035 config foundation | main commit `fb60ba5`時点 | strict config読込、file policy、非漏洩、deny既定 |
| REF-3 | TASK-0036 provision manifest | main commit `c69bafb`時点 | canonical JSONL、`provision.Build`、副作用なしの望ましい状態 |
| REF-4 | TASK-0037 minimal lifecycle | main commit `e15055b`時点 | planning/candidate/completionの3 commit経路 |

### 依存状態

| 依存 | 状態 (`ready` / `pending`) | planning参照 | `ready`後に固定する値 |
|---|---|---|---|
| TASK-0036 | `ready` | REF-3 | `provision.Build`とversion 1 canonical JSONLを再利用する |
| actual Ubuntu executor/live host | `pending` | 対象外 | 後続Taskで実行権限、rollback、live QAを固定する |

### 許可パス

- `tools/dev-agent-harness/internal/provision/`
- `tools/dev-agent-harness/internal/command/`
- `tools/dev-agent-harness/README.md`

### 完了経路preflight

| 確認対象 | 結果 | コマンドまたは根拠 |
|---|---|---|
| 完了checker | `ready` | TASK-0037のplanning/candidate/completion gateを使用する |
| 権限 | `ready` | verifierはread-only、root/sudo/network/process起動を要求しない |
| 依存状態と参照 | `ready` | TASK-0036はmain commit `c69bafb`で完了、REF-3へ固定 |
| 生成物の有無と更新方法 | `ready` | `configure`その他生成物を変更しない |
| 割当ワークツリー | `ready` | `worktrees/TASK-0038-dev-agent-provision-verify` |
| Lapログの書込・Schema・`repository annotation` | `ready` | 新しい3 commit経路を使い、legacy bindingを新規作成しない |

### 未決事項

- なし

### `Dependency-ready reconciliation`

<!-- 依存ready時にMainがready参照、planning参照との差分、AC/設計/scope/QAへの影響、再承認結果を追記する。依存なし又は未readyならN/Aとする。 -->

- TASK-0036はplanning開始時点でready。REF-3との差分はなく、reconciliation追加作業は不要。

## 背景

TASK-0036は安全な望ましい状態をcanonical JSONLとして生成できるが、後続executorが受け取るfileをstrictに検証する入口はまだない。executorとfile policyを同時に実装すると、入力検証の欠陥とroot権限下の副作用が同じTaskへ混在する。先にread-only consumerを独立QAし、executorが受理できる唯一のmanifest境界を小さく固定する。

## 検討すべき設計観点

- manifestの意味論やgrammarを再実装せず、TASK-0036のcanonical bytesを正本にする。
- pathを検査してから再openするTOCTOUを避け、open済みdescriptorだけを読む。
- canonical bytesと一致しない入力は、そのJSONとしての解釈に関係なく拒否する。
- 入力由来の値やOS errorを診断へ含めない。
- success summaryはautomation向けに固定し、manifest本文を再出力しない。
- executor、root権限、live Ubuntuは別security boundaryとして後続Taskへ残す。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] 選択した`change_class`の完了経路と`make check`を満たしている。
- [ ] 製品変更の場合: 実装、テスト、文書、同一案の独立REVIEW/QA、完了後の環境依存ケース確認が完了している。
- [ ] 安全契約変更の場合: 独立計画レビュー、契約検査、許可された統制文書差分の確認が完了している。

## 関連コンテキスト

### 意味 Wiki

- 既存のprovision manifest Wikiへ、consumerのcanonical equalityとfile policyが再利用可能な知識として残る場合だけ追記する。Task固有の証跡だけなら更新しない。

### 判断

- strict consumerはread-onlyで実装し、canonical bytes完全一致を受理条件とする。
- executorを本Taskへ含めない。

### 適用しなかった重要な判断

- JSON/YAML parser、汎用schema validator、manifest内容の部分一致、path検査後の再open、root権限でのdry-runは採用しない。
