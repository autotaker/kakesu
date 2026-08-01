---
task_id: "TASK-0038"
change_class: "product"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "luna-xhigh"
approved_dev_profile_reason: "既存`provision.Build`を唯一のV1正本として再利用し、single-FD file policy、bounded byte read、完全byte照合、setup CLIへの局所接続だけを追加する。executor、root、外部process/network/IPC、依存変更を含まないため"
approved_dev_profile_risk_signals: []
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T03:36:53Z"
planning_reviewed_by: "reviewer-agent-terra-medium"
planning_review_decision: "pass"
planning_reviewed_at: "2026-08-01T03:36:53Z"
classification_approved_by: ""
classification_approved_at: ""
classification_approval_reason: ""
# safety_contract_version: 2
# safety_contract_planned_paths: []
# safety_contract_generated_paths: []
---

# TASK-0038 PLAN

## AC対応

TASKの条件本文を再掲せず、`planning input packet`のAC-IDに設計を対応させる。

| AC-ID | 設計判断 | 変更パス | 実施順序 | 失敗時の扱い |
|---|---|---|---|---|
| AC-1 | setupだけに、順序固定の`verify-provision --config PATH --manifest PATH --target-root PATH`を追加する。valid config/rootと`provision.Build`由来の安全なregular manifestだけを成功させ、固定summaryをstdoutへ出す。 | `internal/command/`、`internal/provision/` | 1, 3 | config、manifest read、Build、比較、stdout writeの失敗ではstdoutを空にし、固定classだけをstderrへ出してnon-zeroにする。 |
| AC-2 | verificationのaccept decisionは、FDから得たbounded raw bytesと、同じconfig/rootから一度だけ再構築した`provision.Build` resultの`bytes.Equal`だけにする。入力の独立解釈、別schema、別desired-state builderは追加しない。 | `internal/provision/` とtest | 2 | 行数、field、型、空白、順序、値、config、root、終端改行を含む任意のbyte差はmanifest-mismatch classで拒否する。 |
| AC-3 | manifest readerは`O_RDONLY|O_CLOEXEC|O_NOFOLLOW|O_NONBLOCK`で一度だけopenし、同じFDをread前後にstatする。regular、128 KiB以下、group/world non-writable、同一size/modeとbyte-countを確認してbounded readする。pathの事前stat/reopenはしない。 | `internal/provision/` とtest | 1, 2 | symlink、non-regular、上限超過、unsafe mode、size/mode変化、read errorはfile-policyまたはI/O classでfail-closedにする。 |
| AC-4 | command adapterは正確な6 argsだけを受理し、argument/config/file-policy/manifest-mismatch/I/Oを固定診断へ変換する。入力path、user、config/manifest本文、JSON値、OS error本文を出さない。 | `internal/command/` とtest | 3 | 不足・余剰・順不同を含む失敗はnon-zeroかつstdout空とする。 |
| AC-5 | production依存をconfig load、FD read/stat、pure Build、byte compare、stdout/stderr writerに限定する。manifest/config/target rootを書き換えず、process/network/IPCを導入しない。既存setup操作と5 binaryのfail-closed behaviorを保持する。 | 許可された3 pathとtest | 2, 3, 4 | 副作用、禁止依存、既存動作の回帰を検出したらPASSにせずMainへ返す。executor/root/live OSは追加しない。 |
| AC-6 | unit/CLI testsはvalid case、1 byte追加・変更・削除、config/root mismatch、symlink/type/mode/size、read中metadata変更、非漏洩、副作用不在を通常のpositive/negative testsで観測する。 | `internal/provision/*_test.go`、`internal/command/*_test.go` | 1, 2, 3 | 各negative caseが受理された場合、またはread中変更を確実に観測できない場合はFAILとしてMainへ報告する。 |
| AC-7 | READMEにverify command、固定success summary、FD policy/canonical equality/read-only境界、executor非対象を記載する。生成済み`configure`とinstall surfaceは変更しない。 | `README.md` | 4 | `./configure`後のharness `make check`、`make distcheck`、root `make check`、`git diff --check`のいずれかがFAILなら完了へ進めない。 |
| AC-8 | standard libraryと許可3 pathだけに留める。executor、実OS/root権限、許可外path、1,200行超過を必要とする要求はこのTaskの外とする。 | 許可された3 pathのみ | 全工程 | 境界追加または上限超過の見込みで実装を停止し、Mainへ戻す。 |

## 関連Wikiと判断

- Mainは完了後、single-FD file policyとcanonical byte equalityを既存provision manifest Semantic Wikiへ取り込む必要性だけを判断する。DEVはWikiとTask証跡を編集しない。
- REF-2のstrict config FD policyを読取安全性の出発点として再利用する。REF-3の`provision.Build`はV1 encodingとdesired stateの唯一の正本であり、consumerはそれを解釈・複製しない。

## 補足設計

### 代替案と不採用理由

- 入力の独立解釈: canonical bytesの唯一正本を二重化し、受理面を増やすため不採用。
- decoded objectの意味比較: field order、whitespace、終端改行を落とすため不採用。raw bytes完全一致を用いる。
- `os.Stat(path)`後に`os.Open(path)`: check-to-reopen raceを作るため不採用。O_NOFOLLOWの一回open後、そのFDだけを検査・読取する。
- executorやroot dry-run: read-only verificationへOS副作用を混在させるため不採用。後続Taskへ残す。

### 責務・境界・不変条件

- `internal/provision`はFD-bound manifest file policy、bounded byte read、`Build`とのbyte equality、stable error classを所有する。既存manifest builderは変更せず、V1の構造を再実装しない。
- FD lifecycleはopen、before stat/policy、bounded read、after stat/policy、size/mode/byte-count comparison、canonical byte comparisonの順である。open済みdescriptorをcloseし、named pathへ再アクセスしない。
- `internal/command`はargument order、config load、verification、固定summary/diagnostic/exitだけを所有する。成功stdoutは`provision manifest version=1 actions=10 verified\n`のみでstderrは空、失敗stdoutは空である。
- read-onlyはmanifest/config/target rootとその他host stateを作成・変更しない意味である。production codeはwrite/create/chmod/chown/rename/unlink、external process、network、IPCを行わない。

### 移行・互換性

- `check-config`、`plan-provision`、`--version`、`--help`、通常起動拒否を保持する。新規許可操作はsetupの`verify-provision`だけである。
- version 1の曖昧な拡張、normalization、partial manifest受理を導入しない。byte equalityにより非canonical入力を拒否する。
- `configure`、install program list、systemd/sysusers/tmpfiles、example config、go.modは変更しない。

## 変更予定

| ファイル | 種別 | 変更内容 |
|---|---|---|
| `tools/dev-agent-harness/internal/provision/provision.go` | implementation | single-FD manifest read/policy、128 KiB bounded bytes、stable verify errors、`Build` bytesとの完全一致。 |
| `tools/dev-agent-harness/internal/command/command.go` | implementation | strict `verify-provision` adapter、安全なsummary/error/exit。 |
| `tools/dev-agent-harness/internal/provision/provision_test.go` | test | positive fixture、byte difference、FD policy/read-change、副作用不在の通常unit tests。 |
| `tools/dev-agent-harness/internal/command/command_test.go` | test | CLI positive/arguments/errors/non-leak/既存fail-closed regression。 |
| `tools/dev-agent-harness/README.md` | documentation | verify usage、read-only/canonical/file-policy boundary、executor非対象。 |

## 実装手順

1. `internal/provision`にmanifest専用のsingle-FD readerを置く。既存configのFD patternを参照し、open/stat/read/statで128 KiB、file type、mode、size、byte-countをfail-closedに確認する。
2. 同packageに、読込bytesと`provision.Build(config, targetRoot)` resultを完全比較するverify入口を追加する。byte differenceをmanifest mismatchへ統一する。
3. `internal/command`へverify subcommandを接続し、6 args、固定summary、固定かつ非漏洩のdiagnostic、stdout/stderr/exitを追加する。既存commandのfail-closed behaviorを回帰する。
4. READMEを更新し、通常のpositive/negative unit/CLI testsを追加する。`./configure`後のharness `make check`、`make distcheck`、root `make check`、`git diff --check`を実行する。

## 検証計画

| ケース群 | 受け入れ条件 | 実施・観測 |
|---|---|---|
| valid canonical CLI | AC-1, AC-2, AC-4 | `Build`から書いた0600 regular manifestとvalid config/rootでexit 0、stdout exact summary、stderr emptyを確認する。 |
| canonical byte difference | AC-2, AC-6 | canonical fileへ1 byteを追加・変更・削除し、別config/rootでbuildしたfileも含めnon-zero/stdout emptyを確認する。 |
| FD policy / read change | AC-3, AC-6 | symlink、directory/FIFO、unsafe mode、128 KiB超、read前後のsize/mode変化を拒否し、path reopenなしをtest seamとsource確認で観測する。 |
| diagnostics / compatibility | AC-4, AC-5 | args、config、manifest mismatch、I/Oでpath/user/content/OS error textが出ないこと、existing setup commandsと5 binaryが従来どおりfail-closedであることを確認する。 |
| side-effect boundary | AC-5, AC-6 | manifest/config/target root snapshot不変と、production importsにprocess/network/IPC/write APIを加えないことを確認する。 |
| repository checks | AC-7 | `./configure`後にharness `make check`、`make distcheck`、root `make check`、`git diff --check`をPASSさせる。live Ubuntu、sudo/PAM、executor/restart/rollbackは対象外のため実行しない。 |

## リスクと停止条件

| リスク | 抑制/検出 | 停止条件 |
|---|---|---|
| path re-open又はFD raceが別fileを検証する | one-open FD helperとpre/post size/mode equality、read-change test | single-FD invariantを保てない場合は実装を停止してMainへ返す。 |
| desired stateがconsumerへ複製されdriftする | `Build` resultとのraw byte equalityだけをaccept decisionに使う | 独立解釈やbuilder複製が必要になればscope外として停止する。 |
| diagnosticから入力/OS情報が漏れる | exact stdout/stderr assertions | 漏洩を検出したら固定classへ縮退するまで完了へ進めない。 |
| read-only verifierがexecutor/OS作用へ拡大する | allow-path/import/source auditとsnapshot | root/process/systemd/network/IPC/許可外pathが必要ならMainへ戻す。 |

## 未解決事項

- なし。公開するGo API名とerror classの正確な綴りは、AC-4の固定・非漏洩診断と既存package APIを満たす範囲でDEVが決める。

## main Agentレビュー

- [x] TASKの全AC-IDへ設計判断、パス、順序、失敗時の扱いを対応させ、条件本文を複製していない。
- [x] 設計観点と代替案を検討している。
- [x] QA_PLANがTASK-firstで独立作成されている。
- [x] `dependency-ready reconciliation`と完了経路preflightが完了している。
- [x] DEV開始を承認した。

安全契約変更でv2契約を選ぶ場合は、コメントを外して`safety_contract_version: 2`と予定パス・生成パスの配列を記録し、DEV前に`make task-preflight TASK=TASK-0038`を実行する。変更しない種別は空配列とし、通常の予定パスと生成パスを重複させない。独立計画レビューのPASSとMainの分類承認をフロントマターへ記録する。分類変更時はTask、PLAN、QA_PLANを再承認し、承認者と時刻を更新する。
