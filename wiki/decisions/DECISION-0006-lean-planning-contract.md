---
kind: decision
decision_id: DECISION-0006
title: Lean Planning Contract
status: accepted
decided_at: 2026-07-20
supersedes: []
---

# Lean Planning Contract

## Context

Task、PLAN、QA_PLANに同じ受け入れ条件を再記述すると、計画時間と不整合の機会が増え、依存待ちや完了経路の不足がplanning作業と混同される。

## Decision

Main所有のplanning input packetをPlannerとQAの共通かつ唯一の入力とする。packetは目的、非対象、AC-ID、安定参照、依存状態、許可パス、preflight、未決事項を定義する。PLANは設計判断を、QA_PLANはAC-IDごとの観測を所有し、Taskの条件本文を複製しない。

dependency waitはactive planningから分ける。依存ready後に設計、scope、期待結果、QA観測が変わる場合は、Mainがreconciliationを記録し、DEV前に影響する計画を再承認する。DEV前のpreflightで完了checker、権限、依存、生成物、worktree、Lapログの書込・Schema・repository annotationを確認する。

## Consequences

- PLAN/QA_PLANは相互に待たず、同じpacketから独立に作成できる。
- packet、依存状態、preflightの不明確さは承認を止める根拠であり、推測で埋めない。
- active planningが10分を超える場合は原因を記録し、文章の磨き込みを継続しない。
- 新たなLap Schema/JSONLや既存証跡の遡及変換は要求しない。

## Evidence

- [TASK-0030](../../tasks/TASK-0030-lean-planning-contract/TASK.md)
- [TASK-0030 PLAN](../../tasks/TASK-0030-lean-planning-contract/PLAN.md)
- [TASK-0030 QA PLAN](../../tasks/TASK-0030-lean-planning-contract/QA_PLAN.md)
