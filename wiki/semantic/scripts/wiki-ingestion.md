---
kind: script
title: Wiki Ingestion
---

# Wiki Ingestion

## Trigger

QA完了後に`HANDOVER.md`がcompleteとなる。

## 標準進行

1. Wiki AgentがHANDOVERと関連証跡を読む。
2. HANDOVERのdigestで重複ingestを検出する。
3. 既存WikiとDecisionへ同化できるか確認する。
4. 再利用可能な知識だけを調節または新規作成する。
5. ingest記録と索引を更新する。
6. Schema検査後に運用リポジトリの`main`へ直接コミットする。

## 典型的な失敗

- 一Taskの要約を一般則にする。
- 旧Decisionの本文を書き換えて履歴を失う。
- Task証跡やバックログまで変更する。
- 同じHANDOVERを重複して取り込む。

## 終了条件

ingest記録がHANDOVERのdigestと更新ページを示し、Wiki索引と全検査がPASSしている。
