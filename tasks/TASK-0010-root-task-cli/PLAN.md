---
task_id: "TASK-0010"
status: draft
planner_agent: "planner-agent-terra-medium"
approved_by: ""
approved_at: ""
approved_dev_profile: "luna-xhigh"
planned_implementation_files: 12
planned_implementation_lines: 720
estimate_points: 5
---

# TASK-0010 PLAN

## 受け入れ条件の具体化

| 条件 | 観測方法 | 根拠 |
|---|---|---|
| input safety | invalid CLI input exits nonzero and leaves DB/filesystem unchanged | `docs/00` §6 |
| atomic allocation | success exposes Task+Agent+Workspace; each injected failure exposes none and returns Agent idle | `docs/01`, `docs/03` |
| root identity | root Task has null parent, human authority reference, empty workspace and no inherited grants | `docs/00` §§2.4/4 |
| concurrency | competing creates for one idle Agent produce exactly one assignment | Task-0007 owner invariant |

## 関連Wikiと判断

- `docs/00-kakesu.md` §§2.4/6/8、`docs/01-domain-model.md` §§1–3。
- `docs/02-task-lifecycle.md`, `docs/03-agent-lifecycle.md`, `docs/06-tools-and-async.md`のTask入力。
- TASK-0007 atomic Control store、TASK-0009 Linux runtime manager、TASK-0022 fail-closed filesystem isolation。安全なroot workspace開始にはTASK-0022が必要である。

## 設計

### 選択案

`kakesu task create`はCobra/stdlibのthin adapterとしてJSONまたはflagsをparseし、`RootTaskService.Create`を呼ぶ。Serviceはinput validation、idempotency key、idle Agent reservation、TASK-0022がadmitした`empty` Linux P0 workspace prepare、TASK-0007のatomic persistを順に実行する。DB commit前の物理resourceはcompensation cleanup対象、commit後の実行開始は別Taskに委ねる。出力はstableなJSON resultとhuman summary、失敗はtyped error→exit codeへ変換する。

### 代替案と不採用理由

- CLIがSQLiteを直接更新: transaction/compensationを迂回するため不採用。
- TASK-0022未admitのruntimeを安全なworkspaceとして開始: FS escapeを許すため不採用。
- Agent自動作成: profile capacity運用を隠すためMVPでは不採用。

### 責務と境界

- CLI: parse/output/exit code。RootTaskService: workflow/compensation。Control repository: atomic DB persist/owner reservation。Workspace manager: prepare/cleanupだけを提供。
- Human root authorityとAgent execution ownerは別field/概念に保ち、CLIは人間との直接通信経路を実装しない。

### 不変条件

- root Taskは`parent_task_id = null`、objective/acceptanceは非空、ownerは一人、workspaceは一つで`empty`。
- reservation中/assigned Agentは他Taskへ割り当てない。失敗時はDB record・reservation・filesystem resourceを残さない。
- workspaceにgrantを継承しない。idempotency key同一再実行は同じ結果または明示conflictを返し二重Taskを作らない。

### 失敗時・移行・互換性

- validation failureはworkspace prepare前に返す。reservation failureはTask作成しない。prepare failureはAgent reservationをrollbackする。
- DB commit failureはworkspace cleanupを試み、cleanup failureはoperator-visible orphanとして記録する。
- CLI outputはversioned result shapeにする。MVPに旧CLI互換はない。

## 変更予定

見積もり対象は実装コード、Schema、設定ファイルだけとする。

| ファイル | 種別 | 概算変更行数 | 変更内容 |
|---|---|---:|---|
| `core/cmd/kakesu/main.go` | implementation | 70 | command registration/exit mapping |
| `core/cmd/kakesu/task_create.go` | implementation | 120 | flags/JSON parse and output |
| `core/cmd/kakesu/task_create_test.go` | implementation | 90 | CLI integration tests |
| `core/internal/control/root_task_service.go` | implementation | 130 | reserve/prepare/persist/compensate workflow |
| `core/internal/control/root_task_service_test.go` | implementation | 120 | failure injection/concurrency |
| `core/internal/control/repository.go` | implementation | 45 | atomic root-create port |
| `core/internal/control/sqlite_repository.go` | implementation | 65 | reservation/persist transaction |
| `core/internal/control/model.go` | implementation | 35 | root authority/idempotency result |
| `core/internal/execution/workspace.go` | implementation | 45 | prepare/cleanup contract |
| `core/internal/execution/linux/workspace.go` | implementation | 50 | Task-0009 adapter integration |
| `core/internal/app/app.go` | implementation | 25 | service dependency wiring |
| `core/README.md` | implementation | 25 | invocation and failure semantics |

## 見積もり

```text
file_score = ceil(planned_implementation_files / 3)
line_score = ceil(planned_implementation_lines / 200)
estimate_points = 1, 2, 3, 5, 8, 13のうちmax(1, file_score, line_score)以上の最小値
```

## 実装手順

1. Task-0007 root-create transaction portとTask-0009 workspace prepare/cleanup portを確定する。
2. input/result/error modelsとRootTaskServiceを実装し、failure injection pointsを設ける。
3. CLI adapter、JSON/human output、exit mappingを実装する。
4. invalid input、owner contention、prepare failure、commit failure、idempotent retryを統合試験する。

## 検証計画

- `go test ./...`とCLI subprocess integrationでDB/filesystemの事後状態を検査する。
- Linux P0対応runnerでTASK-0022のFS profile admission後だけcreate成功し、workspace identity/no inherited grantを確認する。
- `make check`、`git diff --check`。Task-0007/0009のconsumer contract testsを必須にする。

## 未解決事項

- root human authorityの永続field名とCLI入力のidempotency key形式は、既存schemaに同名fieldがないためDEVでSchema ownerと照合して固定する。TASK-0022のprofile admission contractとTask/owner/workspace原子性は変更しない。

## main Agentレビュー

- [ ] 受け入れ条件が検証可能である。
- [ ] 設計観点と代替案を検討している。
- [ ] QA計画を作成できる。
- [ ] 見積もりが規則どおりである。
- [ ] DEV開始を承認した。
