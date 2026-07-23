---
kind: decision
decision_id: DECISION-0003
title: Codex Agent Model Routing
status: superseded
decided_at: 2026-07-14
supersedes: []
---

# Codex Agent Model Routing

## Context

役割ごとの能力、調査権限、二つのrepository root間の設定、adapter commitの責務を明示しなければ、設定driftと権限逸脱を検出できない。

## Decision

製品repositoryのproject-scoped TOMLをrole routingの正本とし、固定roleは指定されたmodelとeffort以外を起動前にfail closedする。DEVのLuna利用は局所的・機械検証可能・risk signalなしの場合に限り、Solへのpromotionは履歴とmain Agent承認を要する。Explorerは明示launcherによる一問のread-only調査に限り、depth 2、threads 2、no-childとする。

work repositoryの設定は正本から決定的に生成するadapterとし、digest付き完全一致検査でdriftを拒否する。adapterの生成、scope検査、hook付きcommit、post-check、rollbackはlock所有の専用launcher親だけが行う。

## Consequences

- fixed roleの互換overrideはroleを経由しない明示legacy経路に限定する。
- Explorerの遵守は自己申告ではなくlauncher traceと負例試験で確認する。
- 子の失敗やscope、stage、commit、hook、検証のdriftはfail closedし、開始前状態へrollbackして`commit:null`を記録する。
- QA実行環境の権限制約は、再実行結果と区別して環境所見として帰属する。

## Evidence

- [TASK-0002](../../tasks/TASK-0002-codex-agent-model-routing/TASK.md)
