---
task_id: "TASK-0029"
title: "Control versioned状態とクラッシュ復旧を実装する"
status: ready
created_at: "2026-07-20"
---

# TASK-0029 Control versioned状態とクラッシュ復旧を実装する

## 目的

TASK-0027/0028後のControl-owned SQLite Storeへexpected-version付きcontract/progress更新を追加し、immutable historyとcurrent read modelを同一transactionで保つ。DB close/reopen後も同じversioned Task read modelと履歴を復旧できるようにする。

## 背景

TASK-0027は原子的Task作成とcontract v1/progress v0を保存し、TASK-0028はowner排他とTask lifecycleを追加する。TASK-0007 PLAN 0007-Cで残されたcontract/progressの楽観version更新、旧snapshot/history保持、再open recoveryを完成しなければ、競合更新のlost update防止とクラッシュ後の耐久read modelを証明できない。

## 依存とDEV開始条件

- TASK-0027およびTASK-0028の独立REVIEW/QA PASS、Main merge、candidate_treeとmerge_treeの一致を必須とする。
- TASK-0028は現時点で未mergeである。DEV開始前に実際のStore/lifecycle API、schema migration version、typed conflict、event sequence契約を再読し、本TASKのPLAN前提と一致することをMainが確認する。
- TASK-0028 merge後APIが本TASK前提と異なる場合、実装で吸収せずPLAN/QA_PLANを改訂・再承認する。

## スコープ

### 対象

- contract updateはexpected current version一致とnew versionの単調な+1を検査し、新しいimmutable snapshotとContractChanged eventをcurrent pointer更新と同一transactionで保存する。
- progress updateはexpected current version一致とnew versionの単調な+1を検査し、append-only progress history、新current progress、対応eventを同一transactionで保存する。
- contract/progress/eventごとにschema ID、revision、digestを不変参照として保存し、旧snapshot/history/eventを削除・上書きしない。
- 同じexpected versionからの並行更新では正確に一方だけ成功し、敗者はtyped conflictとなりpartial writeを残さない。
- DB close/reopen後、Task current state、active owner、current contract/progress、全immutable history、event sequenceを同じread modelとして取得する。
- temporary SQLite integration testでcontract/progress正常更新、競合、故意SQL失敗rollback、旧履歴保持、close/reopen recoveryを検証する。

### 対象外

- runtime JSON Schema意味検証、schema resolution library、Schema変更。必要なら依存追加前に再計画する。
- owner割当/解放、Task lifecycle遷移、owner移管、Agent Run、Task tree、Workspace/Resource cleanup（TASK-0027/0028の責務）。
- transport/CLI、Inbox/Outbox、Plane間原子性、event replayだけを正本とするevent-sourced再構築、backup/replication/PITR。
- contract/progress以外のgeneric patch API、自動競合retry、silent overwrite。

## 受け入れ条件

- [ ] expected contract version一致時だけversionを1増やし、immutable snapshot、current contract、ContractChanged eventを原子的に保存する。
- [ ] expected progress version一致時だけversionを1増やし、append-only history、current progress、対応eventを原子的に保存する。
- [ ] stale/future/skipped version、同一versionの内容差替え、並行競合を拒否し、current/history/event sequenceを完全に不変に保つ。
- [ ] contract/progress/eventの各recordがschema ID/revision/digestを保持し、過去snapshot/history/eventを更新・削除しない。
- [ ] 故意のSQL失敗ではcurrent更新、history append、event appendの全てをrollbackする。
- [ ] close/reopen後にcurrent Task/owner、current contract/progress、全履歴、event sequenceがclose前と一致する。
- [ ] 全QAケースにqa_execution_mode、理由、fail-closed条件があり、同一candidate_commit/treeを独立REVIEW/QAが評価できる。

## 検討すべき設計観点

- expected-version検査とINSERT/UPDATE/event appendを一つのSQLite transactionへ閉じる。
- immutable historyとcurrent read modelを分離し、recoveryをevent再生だけに依存させない。
- DB conflict/busyを自動retryせずtyped conflict/storage errorとして上位へ返す。
- schema参照は不変bytes/metadataとして保存し、runtime validator導入を本sliceへ混在させない。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] 必要な実装、temporary SQLite integration test、文書が完成している。
- [ ] Reviewerが同一candidateで独立レビューとmake checkを完了している。
- [ ] QAがReviewerと独立に同一candidateから開始し、競合/rollback/reopenを検証している。
- [ ] Mainがmerge_treeと承認candidate_treeの同一性を確認している。
- [ ] HANDOVERがWiki Agentに取り込みされている。

## 関連コンテキスト

### 正本

- tasks/TASK-0007-control-durable-task-store/PLAN.md 0007-C。
- tasks/TASK-0027-control-durable-foundation/TASK.md とPLAN.md。
- tasks/TASK-0028-control-ownership-lifecycle/TASK.md とPLAN.md。
- docs/01-domain-model.md、docs/11-data-model.md、docs/13-technology-stack.md §§5–6。

### 判断

- expected-version CAS、immutable history、current read model、schema reference、close/reopen recoveryだけを最大2 Lapの最終sliceとする。

### 適用しなかった重要な判断

- event replayのみ、mutable contract一行、history上書き、自動retry、runtime schema validatorの無承認追加は採用しない。
