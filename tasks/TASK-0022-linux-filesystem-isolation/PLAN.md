---
task_id: "TASK-0022"
status: draft
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "sol-high"
approved_by: ""
approved_at: ""
planned_implementation_files: 15
planned_implementation_lines: 1120
estimate_points: 8
---

# TASK-0022 PLAN

## 受け入れ条件の具体化

| 条件 | 観測方法 | 根拠 |
|---|---|---|
| profile admission | Landlock/LSMまたはmount capability欠落hostでexec前拒否 | docs/07 §2 |
| layered enforcement | workspace/runtime allowlist外のread/writeが拒否 | docs/07 §2 |
| escape resistance | symlink、procfs、bind、magic-link、credential/socket負例が拒否 | docs/07 §2 |
| fail closed | enforcement失敗でexecなし、cleanup/auditが残る | docs/07 §2 |

## 関連Wikiと判断

- docs/07 §2、docs/11 §2、docs/12 §5、docs/13 §§2,4。依存: TASK-0006、TASK-0009。

## 設計

### 選択案

TASK-0009のworkspace mount namespaceへFS profile compilerを接続する。compilerはworkspace write rootと最小read-only runtime allowlistをLandlock/LSM ruleへ変換し、mount boundaryと両方がprobe・適用・検証できた場合だけadmitする。

### 代替案と不採用理由

- mount namespaceのみ: path escapeを別強制点で拒否できないため不採用。
- userspace path filter: TOCTOUとkernel bypassを防げないため不採用。

### 責務と境界

- TASK-0009はnamespace/process lifecycle、当TaskはFS rule/profile/escape test、TASK-0017/0023はnetwork境界を所有する。

### 不変条件

- allowlist外はdefault deny、mountとLandlock/LSMの両方が必要、unsupported/enforcement failureはexec前fail-closed、ruleはworkspace identityへ束縛する。

### 失敗時・移行・互換性

probeまたはrule install失敗はprofileをadvertiseせずreverse cleanupする。旧best-effort profileへのfallbackはない。

## 変更予定

| ファイル群 | 種別 | 概算行数 | 変更内容 |
|---|---|---:|---|
| core/internal/execution/linux/{fs_profile,landlock,mount_policy,probe}.go | implementation | 430 | compiler/probe/enforcement |
| core/internal/execution/linux/{audit,workspace}.go | implementation | 150 | lifecycle binding |
| core/internal/execution/linux/*_test.go | test | 300 | negative escape/profile tests |
| core/internal/execution/testdata/fs-allowlist.json | fixture | 40 | allowlist cases |
| schemas/draft-v0/execution-plane/workspace-created.schema.json | schema | 60 | FS profile admission evidence |
| docs/runtime/linux-profile.md | documentation | 140 | supported host/profile contract |

実装対象15ファイル・1,120行。`ceil(15/3)=5`、`ceil(1120/200)=6`のため8pt。

## 実装手順

1. probeとFS profile schemaを確定する。
2. mount boundaryへLandlock/LSM compilerを接続する。
3. admission/audit/cleanupを追加する。
4. negative escapeとunsupported-host fail-closed試験を実行する。

## 検証計画

- allowlist外read/write、credential/socket、symlink/procfs/bind/magic-link、rule-install failure、unsupported kernel、cleanupをLinux integrationで検証する。
- Go tests、schema validation、`make check`、`git diff --check`。

## 未解決事項

- 対象CI kernelのLandlock ABI/LSM要件はDEV開始前にprobeで固定し、満たせないrunnerはP0として扱わない。

## main Agentレビュー

- [ ] 受け入れ条件が検証可能である。
- [ ] 設計観点と代替案を検討している。
- [ ] QA計画を作成できる。
- [ ] 見積もりが規則どおりである。
- [ ] DEV開始を承認した。
