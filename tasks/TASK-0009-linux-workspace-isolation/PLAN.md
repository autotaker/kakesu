---
task_id: "TASK-0009"
status: draft
planner_agent: "planner-agent-terra-medium"
approved_by: ""
approved_at: ""
approved_dev_profile: "sol-high"
planned_implementation_files: 14
planned_implementation_lines: 860
estimate_points: 5
---

# TASK-0009 PLAN

## 受け入れ条件の具体化

| 条件 | 観測方法 | 根拠 |
|---|---|---|
| profile admission | required Linux capabilitiesを欠くhostでsandbox起動が拒否される | `docs/07` §2 |
| workspace identity | workspace IDごとのnamespace/cgroup/process identityが一意に生成される | `docs/07` §2 |
| process lifecycle | cancel/failure後にprocess group/cgroup内の子孫が残らない | `docs/07` P0表 |
| fail closed | namespace/cgroup/create/exec失敗時にworkloadを開始せずresourceを回収する | `docs/07` §2 |

## 関連Wikiと判断

- `docs/00-kakesu.md` §§2.1–2.2, `docs/01-domain-model.md` §3。
- `docs/07-governance.md` §2のLinux P0表（namespace/cgroup/process lifecycle）。FS強制はTASK-0022、DNS/proxy/firewallはTASK-0017、HTTPS正規化はTASK-0023の正本とする。
- `docs/13-technology-stack.md`のGo Core/Execution boundary、Task-0006 wire schema。

## 設計

### 選択案

Go Execution PlaneにLinux専用`WorkspaceRuntimeManager`を置く。profile probeがuser/mount/PID/IPC/network namespacesとcgroup v2を確認する。createは`empty` workspace root、秘密値を継承しないprocess environment、namespace/cgroup、rootless/no_new_privs/capability drop、workspace-scoped process identityを準備してからexecする。一つでも失敗したらworkloadを開始せず全resourceをcleanupする。

### 代替案と不採用理由

- container runtimeへの丸投げ: namespace/cgroup/process identityの証拠を明示できないため不採用。
- partial isolation fallback: P0を満たさないので不採用。
- Landlock/LSM FS強制やfirewall/DNS/proxyを同居: 各強制境界の負例と所有者を曖昧にするため不採用。

### 責務と境界

- profile probe/admission、workspace layout、namespace/cgroup/process lifecycle、audit recordを別packageにする。
- Controlはlogical workspaceを所有し、Executionは物理resourceを所有する。Governanceはpolicy/enforcementを所有し、本Taskはproxyそのものを実装しない。

### 不変条件

- `workspace_id`は一つのnamespace/cgroup/audit identityへ対応し、Task/Agent変更でpolicy主体を変えない。
- FS allowlist/escape、DNS/proxy/firewall route、HTTPS canonical requestはこのTaskの保証に含めない。それぞれTASK-0022、TASK-0017、TASK-0023が強制・試験する。
- isolate/create/execが一つでも失敗すればcommandをexecしない。cleanupは冪等で子孫全体に及ぶ。
- P0は`empty`だけで、grant継承は空、fork/shared-readonlyは許可しない。

### 失敗時・移行・互換性

- capability probe失敗はunsupported profileとして明示し、best-effortで起動しない。
- namespace/cgroup途中失敗はreverse-order cleanup、cleanup失敗は監査しretry可能なorphanとして残す。
- process exitはaudit/eventを記録し、cancel/timeout/shutdownは同じkill/reap pathを使う。既存workspace形式の移行はない。

## 変更予定

見積もり対象は実装コード、Schema、設定ファイルだけとする。

| ファイル | 種別 | 概算変更行数 | 変更内容 |
|---|---|---:|---|
| `core/internal/execution/linux/profile.go` | implementation | 75 | P0 capability probe/admission |
| `core/internal/execution/linux/profile_test.go` | implementation | 60 | probe failure tests |
| `core/internal/execution/linux/workspace.go` | implementation | 130 | empty root/layout/create cleanup |
| `core/internal/execution/linux/namespaces.go` | implementation | 115 | user/PID/IPC namespace setup |
| `core/internal/execution/linux/process.go` | implementation | 105 | exec/no_new_privs/cap drop |
| `core/internal/execution/linux/cgroup.go` | implementation | 75 | cgroup tracking/reap |
| `core/internal/execution/linux/audit.go` | implementation | 60 | lifecycle/violation records |
| `core/internal/execution/linux/cleanup.go` | implementation | 70 | idempotent reverse cleanup |
| `core/internal/execution/linux/integration_test.go` | implementation | 130 | namespace/cgroup/process lifecycle tests |
| `core/internal/execution/service.go` | implementation | 65 | manager lifecycle wiring |
| `core/internal/execution/workspace.go` | implementation | 75 | platform-neutral port/models |
| `core/cmd/kakesu/main.go` | implementation | 35 | Linux profile config bootstrap |
| `schemas/draft-v0/execution-plane/workspace-created.schema.json` | schema | 40 | empty/profile/runtime identity binding validation |

## 見積もり

```text
file_score = ceil(planned_implementation_files / 3)
line_score = ceil(planned_implementation_lines / 200)
estimate_points = 1, 2, 3, 5, 8, 13のうちmax(1, file_score, line_score)以上の最小値
```

## 実装手順

1. Linux P0 acceptance matrixをtestable probe・isolation casesへ変換する。
2. platform-neutral portとLinux probe/workspace/cgroup primitivesを実装する。
3. namespace/cgroup/process identityを失敗時cleanup込みで接続する。
4. process supervisor/auditを追加し、probe/identity/descendant/cleanup testsを実行する。
5. Task-0010 consumerが`empty` workspaceをprepare/cleanupできる契約を確定する。

## 検証計画

- Linux runner上でrootless profile admission、namespace/cgroup identity、cancel descendant reaping、create/cleanup失敗を実行する。
- unit testsはprivileged host依存をmockし、integrationはP0機能を実測する。機能不足をPASS扱いにしない。
- `go test ./...`、schema validation、`make check`、`git diff --check`。

## 未解決事項

- rootless user/mount/PID/IPC/network namespaceとcgroup v2の正確なCI要件はDEV開始前にprobeして固定する。Landlock/LSMとfirewall backendは後続Taskの前提として別途固定する。

## main Agentレビュー

- [ ] 受け入れ条件が検証可能である。
- [ ] 設計観点と代替案を検討している。
- [ ] QA計画を作成できる。
- [ ] 見積もりが規則どおりである。
- [ ] DEV開始を承認した。
