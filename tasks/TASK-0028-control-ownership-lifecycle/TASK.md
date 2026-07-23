---
task_id: "TASK-0028"
title: "Control owner排他とTaskライフサイクルを実装する"
status: ready
created_at: "2026-07-20"
---

# TASK-0028 Control owner排他とTaskライフサイクルを実装する

## 目的

TASK-0027のControl-owned SQLite Storeを拡張し、1つのAgentが同時に所有できる非終端Taskを1件に制限する。Taskの許可されたlifecycle遷移だけを原子的に受理し、completed/cancelled確定時にはowner解放と対応eventを同じtransactionで永続化する。

## 背景

TASK-0027はTask作成時のowner割当とcreation evidenceを原子的に保存するが、競合する2 Taskの同一Agent割当、Task状態更新、終端時のowner解放は意図的に対象外とした。TASK-0007 PLAN 0007-Bを独立sliceとして実装しなければ、waiting/suspended中の責任保持、違法遷移拒否、終端後のAgent再利用をDB上で保証できない。

## スコープ

### 対象

- TASK-0027のStore APIとSQLite schema/migrationを拡張し、Agentごとの非終端owner assignmentをDB制約とcompare-and-set transactionで一件に制限する。
- 許可辺 ready→running、running→waiting|suspended|reviewing_completion、waiting|suspended→running、reviewing_completion→running|completed、および全非終端状態→cancelledを明示遷移表で実装する。
- waiting/suspended/reviewing_completionではownerを保持し、completed/cancelled確定時だけowner解放と対応TaskEventをcurrent Task更新と同一transactionで確定する。
- 同じAgentへの2 Task並行割当では一方だけ成功し、敗者にTask/owner/eventのpartial writeを残さない。
- 全許可辺、全拒否辺、終端状態からの再遷移、競合割当、transaction failureをtemporary SQLite integration testで検証する。

### 対象外

- expected version付きcontract/progress更新、immutable contract snapshot履歴、append-only progress history/current再構築、クラッシュ後のversioned recovery（TASK-0029）。
- owner移管、複数owner、Agent Run同時実行制御、Task tree cancellation cascade、Workspace/Resource cleanup、transport/CLI、Inbox/Outbox。
- Task/Agent event Schema変更またはruntime Schema検証、SQLite driver変更、Control以外からのDB直接書込み。

## 受け入れ条件

- [ ] 同一Agentへ2つの非終端Taskを並行割当すると正確に一方だけ成功し、敗者側にowner/event/Task状態のpartial writeがない。
- [ ] ready→running、running→waiting|suspended|reviewing_completion、waiting|suspended→running、reviewing_completion→running|completed、各非終端状態→cancelledだけを受理する。
- [ ] 許可されない全state pair、終端状態からの遷移、期待current state不一致を拒否し、Task状態、progress、owner、event sequenceが完全に不変である。
- [ ] waiting、suspended、reviewing_completion中はowner assignmentを保持し、Agentを別の非終端Taskへ割り当てられない。
- [ ] completed/cancelled確定はTask current state、terminal event、owner解放を同一transactionでcommitし、故意失敗時は全件rollbackする。
- [ ] owner解放後は同じAgentを別の非終端Taskへ割当可能である。
- [ ] 全QAケースにqa_execution_mode、理由、fail-closed条件があり、同一candidate_commit/treeを独立REVIEW/QAが評価できる。

## 検討すべき設計観点

- 排他はprocess-local mutexではなくSQLite制約とtransaction CASで保証する。
- event payloadの意味は既存event contractに従い、current stateとevent sequenceの更新を分離しない。
- busy/conflictを自動retryせずtyped conflictとして呼出側へ返し、再試行判断を上位へ残す。
- TASK-0029のversion column、contract/progress mutation、recovery read modelを先取りしない。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] 必要な実装、integration test、文書が完成している。
- [ ] Reviewerが同一candidateで独立レビューとmake checkを完了している。
- [ ] QAがReviewerと独立に同一candidateから開始し、全許可/拒否edgeと並行割当を検証している。
- [ ] Mainがmerge_treeと承認candidate_treeの同一性を確認している。
- [ ] HANDOVERがWiki Agentに取り込みされている。

## 関連コンテキスト

### 正本

- tasks/TASK-0007-control-durable-task-store/PLAN.md 0007-B。
- tasks/TASK-0027-control-durable-foundation/TASK.md とPLAN.md。
- docs/02-task-lifecycle.md §§1, 8, 11、docs/03-agent-lifecycle.md §§4, 9, 12。

### 判断

- DB制約とcompare-and-set transactionを併用し、owner排他、state更新、event、終端owner解放を一つのControl Store境界へ置く。1〜2 Lapで完結しない場合は0029の範囲を借りず再分割する。

### 適用しなかった重要な判断

- in-memory mutexのみ、遷移後の別transaction event、waiting/suspended時owner解放、自動競合retry、versioned updateの先取りは採用しない。
