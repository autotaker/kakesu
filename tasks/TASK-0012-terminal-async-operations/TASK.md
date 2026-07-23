---
task_id: "TASK-0012"
title: "terminalツールと永続非同期操作を実装する"
status: ready
created_at: "2026-07-15"
---

# TASK-0012 terminalツールと永続非同期操作を実装する

## 目的

隔離Workspace内の`terminal`を、durable async operation、同期期限、取消、再起動復旧付きで提供する。

## 背景

TASK-0009の隔離実行基盤とTASK-0011のdurable stepに、長時間プロセスをTask責任・AgentRun・API連鎖から分離する共通非同期基盤を接続する必要がある。

## スコープ

### 対象

- terminal commandのWorkspace境界、stdout/stderr artifact参照、timeout、process lifecycle。
- operation key付き`async_operations`、worker lease、同期期限超過時の`accepted(async_id)`、cancel/restart recovery。

### 対象外

- ネットワーク許可/CASB、任意host tool、Mailbox待機群（TASK-0013）、delegate（TASK-0014）。

## 受け入れ条件

- [ ] commandは対象Taskの隔離Workspaceでのみ開始し、終了コードと上限付き出力参照を返す。
- [ ] 期限内完了はcompleted、超過は同一操作を継続した`accepted(async_id)`となる。
- [ ] 再配送・再起動・二重cancelでプロセスまたは終端イベントを重複させない。
- [ ] cancel/timeout/shutdownは子プロセスを停止し、operation状態と監査可能な結果を収束させる。

## 検討すべき設計観点

- OSプロセスはAgent resource、async recordはcontrol.db正本。context cancellationを全境界へ伝播する。
- 外部実行はDB transaction外、開始/終了意図と結果は永続化してから進める。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] 必要な実装、テスト、文書が完成している。
- [ ] 独立レビューと`make check`がPASSしている。
- [ ] マージ後QAが完了している。
- [ ] HANDOVERがWiki Agentに取り込みされている。

## 関連コンテキスト

### 意味 Wiki

- docs/06-tools-and-async.md、docs/11-data-model.md、docs/12-implementation-and-tests.md、docs/13-technology-stack.md
### 判断

- Async OperationはHarness tool処理の寿命であり、単一LLM背景実行やTask寿命と別IDにする。
### 適用しなかった重要な判断

- goroutine、PID、in-memory channelだけを非同期操作の正本にしない。
