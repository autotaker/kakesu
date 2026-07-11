---
kind: concept
title: Task
---

# Task

Taskは、単一のOwnerが責任を持ち、ObjectiveとAcceptanceに基づいて完了候補を提出する作業単位である。([Task所有単位の設計Episode](episode://design/task-owner-unit))

## 特徴

- Ownerは必ず一人である。
- 一つのOwnerは同時に一つの非終端Taskだけを処理する。waiting中もOwnerは占有される。([Owner排他の設計Episode](episode://design/task-owner-exclusivity))
- Ownerが完成したと判断してCompletion Candidateを提出する。
- 独立したAcceptance Reviewerが元のAcceptanceとの整合を軽量確認する。([Completion Review導入Episode](episode://design/completion-review))
- 別Ownerへ完了責任を移す場合はSubtaskを生成する。

## Taskではないもの

shell command、ファイル読取、一回のTool Call、同じOwnerによる一時的な仮説検証は通常Activityである。長時間かどうかではなく、独立Ownerへ責任を移したかで区別する。([Task境界の設計Episode](episode://design/task-boundary))

## 境界事例

短い原因調査でも、L1 OwnerへObjectiveとAcceptanceを渡した場合はTaskになる。同じ調査を現在Ownerが自分のTask内で行うならActivityである。

## 関連

- [Task Ownership](../schemas/task-ownership.md)
- [Task Completion](../scripts/task-completion.md)
