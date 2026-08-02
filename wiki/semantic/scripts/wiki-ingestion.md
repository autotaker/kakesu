---
kind: script
title: Wiki Ingestion
---

# Wiki Ingestion

## Trigger

Mainが明示したWiki ingestだけを対象にする。Wiki依頼がないTaskはWiki Agentもreceiptもなしで完了できる。receiptは、明示的に取り込む場合だけ作る任意成果物である。

## 標準進行

1. Mainが対象Taskと編集許可Wiki pathを固定し、標準`agents.spawn_agent(agent_type="wiki")`で編集専用Wiki Agentを一件ずつ直列に起動する。
2. Wiki AgentがHANDOVERと関連証跡を読み、指定pathだけへ最小の知識更新を行う。独立`codex exec` launcherは使わない。
3. Mainが差分scopeを確認し、必要な索引生成・検証を実行する。
4. Mainだけが共通lock付きpublish transaction、stage、commit、pushを所有する。

## 典型的な失敗

- 一Taskの要約を一般則にする。
- 旧Decisionの本文を書き換えて履歴を失う。
- Main指定外のTask証跡、バックログ、Wiki pathを変更する。
- 同じHANDOVERを重複して取り込む。
- Wiki Agentがlock、検証、stage、commit、merge、push又は`.git`書込みを行う。

## 終了条件

明示的ingestでは、receiptがHANDOVERのdigestと更新ページを示す。索引生成、検証、公開はMainの責務であり、Wiki Agentの終了条件ではない。
