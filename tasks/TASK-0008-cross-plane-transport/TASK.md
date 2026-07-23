---
task_id: "TASK-0008"
title: "Plane間の永続メッセージ配送とサービス監督を実装する"
status: ready
created_at: "2026-07-15"
---

# TASK-0008 Plane間の永続メッセージ配送とサービス監督を実装する

## 目的

各Planeの永続Inbox/OutboxとUDS dispatcherを実装し、重複許容・少なくとも一回配送を安全に収束させ、全サービスを監督下で起動・停止できるようにする。

## 背景

現在は3つのPlaneが個別のscaffoldで、CoreのStoreはportのみである。設計はDB横断transactionを禁止し、受信InboxのcommitをACK境界とするため、ネットワーク成功だけで配信済みにする実装は不正である。

## スコープ

### 対象

- Core `control.db`、Memory `evidence.db`、Governance `governance.db`のInbox/Outbox migrationとstore adapter。
- framed JSON over UDS、接続再試行、outbox claim/lease、commit後ACK、message_id/idempotency keyによる重複抑止。
- Core RunGroup、Python、Rustにおける登録済みservice supervisorとgraceful shutdown。
- 再送、重複、receiver crash、sender restart、ACK喪失を再現する統合試験。

### 対象外

- Task/Agent lifecycleのモデル設計（Task-0007）、payload意味処理、CASB enforcement、Memory episode処理。
- TCP、分散broker、exactly-once、DB間2PC。

## 受け入れ条件

- [ ] 送信側は状態変更とOutbox追加を同一ローカルtransactionで確定し、未commit messageを送信しない。
- [ ] 受信側はInbox insertがcommitするまでACKせず、同一`message_id`の再配送は副作用を再実行しない。
- [ ] ACK消失とreceiver再起動で再送されても、Inboxの一意制約により一件として収束する。
- [ ] UDS切断・不正frame・schema参照不正はsenderのOutboxを失わせず、観測可能な失敗として再試行または隔離する。
- [ ] Core/Memory/Governanceの長寿命workerはsupervisor登録され、起動失敗は部分Readyを残さず、shutdown時にleaseを安全に解放する。

## 検討すべき設計観点

- durable source of truthとsocket通知の分離、ACK境界、lease時間、idempotency scope。
- 認可されるUDS peer/pathとframe size上限、payloadを各Plane schema validatorへ渡す境界。
- service failureの伝播とbounded workerのbackpressure。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] 必要な実装、テスト、文書が完成している。
- [ ] 独立レビューと`make check`がPASSしている。
- [ ] マージ後QAが完了している。
- [ ] HANDOVERがWiki Agentに取り込みされている。

## 関連コンテキスト

### 意味 Wiki

- `docs/00-kakesu.md` §3.1: per-Plane queue、UDS、at-least-once、ACK。
- `docs/07-governance.md` §§1.1, 8: Plane DBの単独所有とGovernance Outbox→Control Inbox。
- `docs/13-technology-stack.md` §§5–7: 各DB、infrastructure ACK、wire検証。
- `core/internal/message/store.go`, `core/internal/plane/run_group.go`: 現行portと監督雛形。

### 判断

- Decision: at-least-once + receiver-side `message_id` dedupeを採用し、ACKは`PutInbox` commitの後だけ返す。

### 適用しなかった重要な判断

- exactly-once/2PCは複数SQLite所有境界を破るため不採用。in-memory channelだけの配送はクラッシュ時に正本を失うため不採用。
