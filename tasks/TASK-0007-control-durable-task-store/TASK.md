---
task_id: "TASK-0007"
title: "Control Planeの永続Taskストアとライフサイクルを実装する"
status: cancelled
created_at: "2026-07-15"
---

# TASK-0007 Control Planeの永続Taskストアとライフサイクルを実装する

> 1〜2 LapのTask境界へ合わせるため、TASK-0027、TASK-0028、TASK-0029へ受け入れ条件を分割して置き換えた。後続Taskの依存先は最終sliceのTASK-0029とする。

## 目的

Go Coreが`control.db`を単独で所有し、Task、Agent、Task contract、進捗、状態遷移をクラッシュ復旧可能に永続化する。

## 背景

TaskとResponses APIレスポンスは別の寿命を持ち、API状態は正本にできない。現行Control PlaneはIdleServiceのみであり、Task-0008の配送とTask-0010の原子的割当には、順序付きイベントとトランザクション境界を持つControl集約が必要である。

## スコープ

### 対象

- SQLite migration、Task/Agent/contract/progress/eventのrepositoryとControl service。
- Task contractの版管理、現在進捗と履歴、Task lifecycleの許可遷移、ownerの単一同時Task制約。
- 再起動後のread model復元、schema参照を持つ不変contract/event snapshot、repository/serviceのユニット・統合テスト。

### 対象外

- Inbox/Outboxの配送実装（Task-0008）、Linux Sandbox生成（Task-0009）、root CLI（Task-0010）。
- Agent Run、Continuation、Async、Authority、Completion reviewの完全実装。ただし将来tableの予約は許す。
- Control DBを他Planeが直接開くこと、分散トランザクション。

## 受け入れ条件

- [ ] 新規Taskは目的、受け入れ条件、単一owner、workspace参照、contract versionを一つのSQLite transactionで保存する。
- [ ] Agentは非終端Taskを同時に2件所有できず、競合割当は一方だけ成功する。
- [ ] status transitionは`docs/02-task-lifecycle.md`の許可辺だけを受理し、拒否時は状態・進捗・eventを変更しない。
- [ ] contract更新はversionを単調増加させ、旧snapshotとTaskEventを残す。
- [ ] progressの現在値と履歴は分離して保存され、再起動後も同じread modelを返す。
- [ ] migration済み空DB、既存DB再open、競合更新、違法遷移を自動試験する。

## 検討すべき設計観点

- Task aggregateとAgent aggregateの排他、SQLite transactionの粒度、状態遷移の明示表。
- current progressとappend-only履歴、contract snapshotの不変性、時刻とIDの生成責任。
- 書込み失敗時にeventだけ・状態だけを残さない原子性。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] 必要な実装、テスト、文書が完成している。
- [ ] 独立レビューと`make check`がPASSしている。
- [ ] マージ後QAが完了している。
- [ ] HANDOVERがWiki Agentに取り込みされている。

## 関連コンテキスト

### 意味 Wiki

- `docs/01-domain-model.md` §§1–4: Agent/Task/進捗/Workspaceの所有関係。
- `docs/02-task-lifecycle.md`: TaskStatusと遷移・終端条件。
- `docs/03-agent-lifecycle.md`: owner割当・解放の責務。
- `docs/11-data-model.md`, `docs/13-technology-stack.md` §6: `control.db`の所有データ。

### 判断

- Decision: `control.db`だけがControl集約を書き、状態変更と対応TaskEvent/contract snapshotを同一transactionで確定する。

### 適用しなかった重要な判断

- in-memory storeは再起動と競合の受け入れ条件を満たさないため不採用。ORMはMVPのmigrationとSQL制約を隠すため採用しない。
