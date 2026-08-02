---
kind: concept
title: Development Task
---

# Development Task

Development Taskは、目的、受け入れ条件、設計観点、完成の定義を一人の進行責任者が判断可能な大きさへまとめた開発単位である。製品のTaskドメインモデルとは別に、Git上の変更を統制する運用上の単位である。

## 特徴

- `TASK-NNNN`で一意に識別する。
- 一つのトピックブランチとworktreeに対応する。
- Main所有のplanning input packetが、PlannerとQAに共通の入力を与える。
- PLAN、DEV、独立REVIEW/QAのフェーズを通る。QA_PLANはDEV前に作成する。
- Task本文と実行証跡を製品リポジトリ外へ保持する。

## Taskではないもの

- Epicは複数Taskを束ねるロードマップ単位であり、直接実装しない。
- Reviewerの個別指摘はTaskではない。先送りする作業として独立追跡が必要な場合にだけTask化する。
- 一時的なAgent実行やコマンドはTaskの証跡であり、Taskそのものではない。

## 関連

- [Task delivery](../scripts/task-delivery.md)
- [Planning input packet](planning-input-packet.md)
- [Work repository boundary](../schemas/work-repository-boundary.md)
- [Risk-based candidate QA](../../decisions/DECISION-0005-risk-based-candidate-qa.md)
