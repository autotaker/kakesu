---
kind: decision
decision_id: DECISION-0001
title: PLAN DEV QA Process
status: accepted
decided_at: 2026-07-14
supersedes: []
---

# PLAN DEV QA Process

## Context

Task単位のGit開発で、設計、実装、受け入れ確認の責務と証跡を分離する必要がある。

## Decision

開発をPLAN、DEV、QAの3フェーズで進める。Planner AgentがPLANを作り、main Agentが承認する。DEV Agentは専用worktreeで実装し、独立Reviewer Agentと`make check`を通す。main Agentが`--no-ff`でマージした後、独立QA Agentが受け入れレビューする。

QA計画は実装前に作り、実装後に再確認する。QAのFAILは原因分類し、main AgentがPLAN、DEV、QAの差し戻し先、revert、バグ化を判断する。

## Consequences

- DEV AgentとReviewer Agent、DEV AgentとQA Agentを分離する。
- マージ後QAによりmainへ一時的なリスクが生じるため、重大問題のrevert基準を持つ。
- フェーズ内の詳細状態はバックログへ増やさず、証跡ファイルで確認する。

## Evidence

- [TASK-0001](../../tasks/TASK-0001-development-process-foundation/TASK.md)
