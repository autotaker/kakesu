# 開発用Wiki Schema

## Semantic Wiki

Semantic Wikiは複数Taskで再利用する知識をMarkdownで表す。frontmatterは`kind`と`title`だけを正とする。

```yaml
---
kind: concept
title: Development Task
---
```

種別は次の4つである。

- `concept`: 対象の意味、境界、近い概念との差。
- `schema`: 主体、関係、静的制約。
- `script`: 標準的な時間進行、分岐、終了条件。
- `case-pattern`: 複数Taskから得た条件付きの経験則、反例、適用限界。

関連はMarkdownリンクで表し、Task証跡を根拠としてリンクする。

## Decision

Decisionは確定時点の判断を不変記録として残す。

```yaml
---
kind: decision
decision_id: DECISION-0001
title: PLAN DEV QA delivery process
status: accepted
decided_at: 2026-07-14
supersedes: []
---
```

`status`は`accepted | superseded`とする。置換では旧Decisionを編集して新しい結論にせず、新Decisionを作って`supersedes`へ旧IDを記録する。

## Ingestion Receipt

`ingestions/TASK-NNNN.json`は`../schemas/ingestion-receipt.schema.json`に従う。同じTaskでもHANDOVERのdigestが変わった場合は、同じreceiptを新しい処理結果へ更新できる。Git履歴が過去のdigestと変更内容を保持する。
