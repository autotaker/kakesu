---
kind: decision
decision_id: DECISION-0002
title: Work Repository and Wiki Ownership
status: accepted
decided_at: 2026-07-14
supersedes: []
---

# Work Repository and Wiki Ownership

## Context

バックログとTask証跡を製品コードから分離しつつ、変更履歴と過去の判断をPlanner AgentとTask起票者が再利用できる必要がある。

## Decision

`agent-harness-work`を製品リポジトリに隣接する独立Gitリポジトリとし、`main`一本で運用する。バックログ、Task証跡、Semantic Wiki、DecisionをGit管理し、製品worktreeと生成HTMLは除外する。

Wiki本文とDecisionはWiki AgentがSchemaに従って自律保守し、検査後に`main`へ直接コミットする。main AgentはWiki本文を通常レビューせず、Wiki Schema、検証規則、権限境界を管理する。

## Consequences

- 運用リポジトリへの書き込みをロックで直列化する。
- HANDOVER ingestはdigest付きのreceiptで冪等にする。
- Wiki AgentはTask証跡とバックログを変更しない。
- Schema変更が必要な知識は保留として記録する。

## Evidence

- [TASK-0001](../../tasks/TASK-0001-development-process-foundation/TASK.md)
