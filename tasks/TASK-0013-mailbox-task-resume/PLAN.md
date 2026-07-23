---
task_id: "TASK-0013"
status: draft
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "sol-high"
approved_by: ""
approved_at: ""
planned_implementation_files: 16
planned_implementation_lines: 1300
estimate_points: 8
---

# TASK-0013 PLAN

## 受け入れ条件の具体化

| 条件 | 観測方法 | 根拠 |
|---|---|---|
| effectively-once消費 | duplicate event試験で一entry/一作用 | docs/06 §13, docs/13 |
| await group | timeout/replayで元operation非複製を照合 | docs/06 §14 |
| 条件付き再開 | all/any、stale、cancelledの状態遷移試験 | docs/02 §4 |
| restart安全性 | DB再起動後の未消費entry/cursor試験 | docs/05 §9 |

## 関連Wikiと判断

- docs/02, 05, 06, 11, 12, 13。依存: TASK-0007、TASK-0011、TASK-0012。

## 設計

### 選択案

control.dbのunique event IDとTask sequenceでinboxを確定し、同一transactionでentry追加、WaitCondition評価、必要なTask遷移/continuation更新を行う。dispatcherは永続entryをwake-upしてcoordinatorへ渡す。
### 代替案と不採用理由

- process channelをMailboxにする: restart/重複に耐えない。
- event到着で常に再開: 依存未完了、終端Taskを壊す。
- awaitで元toolを再呼出し: 外部作用を複製する。
### 責務と境界

- control: mailbox、wait group、condition evaluator、Task transition。
- workagent: 再開対象をcoordinatorへenqueueする。
- async: 終端結果をeventとして発行し、Mailboxを直接状態変更しない。
### 不変条件

- event ID一意、Task内sequence単調、entryは一度だけconsumed、待機は明示condition、terminal Taskは再開しない。
### 失敗時・移行・互換性

dispatcher crashは未consumed entryを再claimし、condition競合はversion CASで再評価する。既存空DBにmigrationを追加し、旧インメモリ契約はない。

## 変更予定

| ファイル群 | 種別 | 概算行数 | 変更内容 |
|---|---|---:|---|
| core/internal/control/{mailbox,mailbox_repository,wait_condition,await_group}.go | implementation | 470 | durable inbox/condition |
| core/internal/control/{task_transition,async_repository}.go | implementation | 180 | 原子的状態接続 |
| core/internal/workagent/{resume_queue,coordinator}.go | implementation | 130 | 条件成立後の再開 |
| core/internal/message/{dispatcher,store}.go | implementation | 80 | inbox wake-up |
| schemas/draft-v0/{control-plane/mailbox-event,control-plane/mailbox-consumption,execution-plane/async-operation,api/work-agent-tools}.json | schema | 440 | event/await契約 |
| core/internal/**/**/*_test.go | test（見積外） | — | duplicate/restart/orphan |

実装対象16ファイル・1,300行。`ceil(16/3)=6`、`ceil(1300/200)=7`のため8pt。

## 実装手順

1. mailbox/condition schemaとmigration、dedupe repositoryを作る。
2. await groupとTask waiting/continuation transactionを実装する。
3. dispatcher/recoveryとcoordinator再開を接続する。
4. duplicate、all/any、restart、orphanの試験後に`make check`。

## 検証計画

- at-least-once配送、同一event競合、await timeout/replay、複数await、Task cancel/complete後の遅延結果、restart。
- core Go tests、schema/tabletop検証、`make check`。

## 未解決事項

- event payloadの最終列挙はTASK-0012のterminal結果schemaと同期して確定する。

## main Agentレビュー

- [ ] 受け入れ条件が検証可能である。
- [ ] 設計観点と代替案を検討している。
- [ ] QA計画を作成できる。
- [ ] 見積もりが規則どおりである。
- [ ] DEV開始を承認した。
