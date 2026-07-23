---
task_id: "TASK-0012"
status: draft
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "sol-high"
approved_by: ""
approved_at: ""
planned_implementation_files: 18
planned_implementation_lines: 1400
estimate_points: 8
---

# TASK-0012 PLAN

## 受け入れ条件の具体化

| 条件 | 観測方法 | 根拠 |
|---|---|---|
| 隔離terminal | fixture workspace外へ作用しない実行試験 | docs/06 §3, docs/13 |
| 同期/非同期昇格 | deadline境界でcompleted/acceptedを検査 | docs/06, docs/12 §4 |
| durable/replay | restart・同一call再配送で一操作だけ | docs/11 |
| cancel収束 | process tree停止と一回の結果を検査 | docs/06 §15 |

## 関連Wikiと判断

- docs/06, 11, 12, 13。依存: TASK-0009、TASK-0011。

## 設計

### 選択案

Async repository/lease workerをcontrolに置き、executionのsandbox runnerをportとして呼ぶ。tool dispatchはoperation keyを原子的に取得し、sync deadlineまでは結果を待ち、以後同じrecordを返す。
### 代替案と不採用理由

- terminalごとのgoroutine待機: restart/replayを復旧できない。
- Responses background mode: OSプロセス取消・Task所有を表せない。
- shell PIDのみ保存: PID再利用と結果/証跡の整合を保証できない。
### 責務と境界

- control: operation状態、lease、idempotency、終端結果。
- execution: Workspaceに束縛したrunner/process cleanup。
- workagent: schema検証済みtool要求をdispatchするのみ。
### 不変条件

- operation keyは一実処理、processはTask Workspace内、結果永続化後だけcompleted/cancelled、Task終端後は新規起動不可。
### 失敗時・移行・互換性

worker crashはlease expiryから再照合し、PID生存不明は`outcome_unknown`として再実行しない。空DB migrationのみを対象にする。

## 変更予定

| ファイル群 | 種別 | 概算行数 | 変更内容 |
|---|---|---:|---|
| core/internal/control/{async,operation_repository,lease}.go | implementation | 350 | durable operation/lease/cancel |
| core/internal/execution/{service,terminal,runner,process}.go | implementation | 450 | sandbox runnerとcleanup |
| core/internal/workagent/{tools,terminal_dispatch}.go | implementation | 150 | tool境界 |
| core/internal/{app,message}/**.go | implementation | 90 | worker wiring/outbox |
| schemas/draft-v0/{execution-plane/async-operation,execution-plane/tool-result,api/work-agent-tools}.json | schema | 360 | terminal/async契約 |
| core/internal/**/**/*_test.go | test（見積外） | — | timeout/cancel/restart |

実装対象18ファイル・1,400行。`ceil(18/3)=6`、`ceil(1400/200)=7`のため8pt。

## 実装手順

1. operation schema/migration/repositoryとlease recoveryを実装する。
2. sandbox runner、process group timeout/cancel、artifact outputを追加する。
3. terminal dispatchと同期期限の返却を接続する。
4. failure/restart tests、`go test ./...`、`make check`を実行する。

## 検証計画

- 成功、非ゼロ終了、output上限、timeout、double cancel、replay、worker crash、Task終端後の結果。
- core Go tests、schema検証、`make check`。

## 未解決事項

- TASK-0009が提供するsandbox runner/agent resourceの具体port名に合わせる。

## main Agentレビュー

- [ ] 受け入れ条件が検証可能である。
- [ ] 設計観点と代替案を検討している。
- [ ] QA計画を作成できる。
- [ ] 見積もりが規則どおりである。
- [ ] DEV開始を承認した。
