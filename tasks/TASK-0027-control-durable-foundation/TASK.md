---
task_id: "TASK-0027"
title: "Control永続ストア基盤と原子的Task作成を実装する"
status: ready
created_at: "2026-07-20"
---

# TASK-0027 Control永続ストア基盤と原子的Task作成を実装する

## 目的

Control所有のSQLite永続ストア最小基盤を提供し、root Task作成に必要なcurrent Task、owner Agent、workspace参照、contract v1 snapshot、progress v0、作成/owner eventを一つのtransactionで全件commitまたは全件rollbackできるようにする。

## 背景

TASK-0007 PLANの0007-Aは、durable lifecycle全体を一括実装せず、最小DB migrationと原子的作成を独立した一Lap sliceと定めた。この基盤がなければ再open後の整合性を証明できず、TASK-0028のowner lifecycleとTASK-0029のversioned recoveryを安全に拡張できない。

## スコープ

### 対象

- core/go.modへpure-Go SQLite driverを明示依存として追加する。
- control.db open時の最小schema migration、schema version確認、WAL / foreign keys / busy timeout設定。
- Task、owner Agent、workspace reference、schema ID/revision/digestを含むcontract v1 JSON snapshot、progress v0、sequence 1 TaskCreatedとsequence 2 OwnerAssignedの原子的永続化。
- temporary SQLiteによる新規DB migration、同一schema version reopen、正常作成、故意SQL失敗時rollbackのhermetic integration test。
- typed conflict/storage error。open/migration failureとSQL競合はfail-fastで、silent overwrite/autoretryしない。

### 対象外

- owner単一同時Task制約、owner解放、lifecycle遷移、Task更新（TASK-0028）。
- expected version付きcontract/progress更新、immutable contract履歴、progress history/current read model、versioned recovery、schema実行時検証（TASK-0029）。
- transport/CLI、Inbox/Outbox、Agent Run、Plane間原子性、Schema変更、contract JSON意味検証、CGO driver、in-memory代替。

## 受け入れ条件

- [ ] 新規temporary SQLite DBで最小migrationが一度だけ適用される。
- [ ] 同一schema version DBのclose/reopenはmigrationを重複適用せず、同じ作成read modelを返す。
- [ ] 正常作成はTask、owner Agent、workspace reference、contract v1 JSON snapshot、progress v0、sequence 1/2 eventsを同一transactionで観測できる。
- [ ] 故意SQL失敗時、Task/owner/workspace/contract/progress/eventのpartial writeが残らない。
- [ ] Controlだけがcontrol.dbを書き、接続はWAL、foreign keys、busy timeoutを設定する。
- [ ] 実装実測はcore/go.modとcore/internal/control/store.goの2 files / 252 physical lines / 2 points。testと生成go.sumは見積対象外。

## 検討すべき設計観点

- current Taskとappend-only creation evidenceを一transactionへ閉じる。
- pure-Go driverのGo 1.23互換性、取得可能性、licenseをDEV前に確認し、不可能ならCGOへ自動置換せずPLANへ戻す。
- 初期190行見積はmigration SQL、typed model/error、故障注入seamの物理行を過小評価した。全体SLOC上限内なので受け入れ機能を削らず、実測へ補正する。
- TASK-0028/0029の範囲を先取りしない。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] 必要な実装、テスト、文書が完成している。
- [ ] 独立レビューとmake checkがPASSしている。
- [ ] REVIEWとQAが同一candidate treeから独立に評価されている。
- [ ] merge_treeと承認candidate treeを比較している。

## 関連コンテキスト

### 正本

- tasks/TASK-0007-control-durable-task-store/PLAN.md の0007-A。
- docs/01-domain-model.md、docs/11-data-model.md、docs/13-technology-stack.md §§5–6。

### 判断

- pure-Go SQLiteのControl-owned storeを第一sliceとする。migration SQLはstore.goの埋め込み定数に閉じ、複数migrationが必要になった場合だけ後続Taskで抽出する。

### 適用しなかった重要な判断

- 一括実装、CGO driver、in-memory store/mutex、event sourcingのみ、mutable contract一行は採用しない。
