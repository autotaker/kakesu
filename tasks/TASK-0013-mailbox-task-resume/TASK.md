---
task_id: "TASK-0013"
title: "Mailbox待機とTask再開を実装する"
status: ready
created_at: "2026-07-15"
---

# TASK-0013 Mailbox待機とTask再開を実装する

## 目的

Mailboxイベントを重複なく受信・消費し、`await_async`の待機条件が解決したTaskだけを安全に`waiting`から再開する。

## 背景

非同期処理（TASK-0012）とResponses Run（TASK-0011）を、イベント到着で論理的に接続する。待機理由を状態名に埋めず、durable WaitConditionとして扱う。

## スコープ

### 対象

- event ID重複排除、Task単調sequence、未消費Mailboxの永続化/消費。
- `await_async` all/any待機group、timeout、continuationと`waiting`遷移、条件成立時のRun再開。
### 対象外

- 子Task作成・結果配送（TASK-0014）、ask/escalate、human authority、terminal実行本体。

## 受け入れ条件

- [ ] 同一event IDの少なくとも一回配送は一回だけ適用され、sequence順と消費記録が保持される。
- [ ] `await_async`は元操作を複製せず、期限超過時に冪等な待機group `async_id`を返す。
- [ ] ownerが停止待ちを選んだ場合だけTaskは`waiting`となり、all/any条件成立時だけ一度`running`へ戻る。
- [ ] restart、stale event、Task終端後のorphan結果が状態を不正に再開させない。

## 検討すべき設計観点

- Mailboxは正本、channelはwake-upのみ。event適用・condition照合・Task遷移は同一transactionで行う。
- continuation cursorとRun再開をTask-0011の境界に従わせる。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] 必要な実装、テスト、文書が完成している。
- [ ] 独立レビューと`make check`がPASSしている。
- [ ] マージ後QAが完了している。
- [ ] HANDOVERがWiki Agentに取り込みされている。

## 関連コンテキスト

### 意味 Wiki

- docs/02 §4、docs/05 §8-9、docs/06 §13-14、docs/11-data-model.md
### 判断

- `waiting`は理由でなく状態。WaitConditionが再開条件を所有する。
### 適用しなかった重要な判断

- イベント受信だけでTaskを無条件に再開しない。
