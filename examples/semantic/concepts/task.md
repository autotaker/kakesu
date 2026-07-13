---
kind: concept
title: Task
---

# Task

Taskは、単一のオーナーが責任を持ち、目的と受け入れ条件に基づいて完了候補を提出する作業単位である。([Task所有単位の設計エピソード](episode://design/task-owner-unit))

## 特徴

- オーナーは必ず一人である。
- 1つのオーナーは同時に1つの非終端Taskだけを処理する。`waiting`中もオーナーは占有される。([オーナー排他の設計エピソード](episode://design/task-owner-exclusivity))
- オーナーが完成したと判断して完了案を提出する。
- 独立した受け入れ条件レビュアーが元の受け入れ条件との整合を軽量確認する。([完了レビュー導入エピソード](episode://design/completion-review))
- 別オーナーへ完了責任を移す場合はSubtaskを生成する。

## Taskではないもの

シェル コマンド、ファイル読取、1回のツール 呼び出し、同じオーナーによる一時的な仮説検証は通常作業である。長時間かどうかではなく、独立オーナーへ責任を移したかで区別する。([Task境界の設計エピソード](episode://design/task-boundary))

## 境界事例

短い原因調査でも、L1 オーナーへ目的と受け入れ条件を渡した場合はTaskになる。同じ調査を現在オーナーが自分のTask内で行うなら作業である。

## 関連

- [Task 所有権](../schemas/task-ownership.md)
- [Task 完了](../scripts/task-completion.md)
