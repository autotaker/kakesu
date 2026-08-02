---
kind: decision
decision_id: DECISION-0007
title: Streamlined Task Gates
status: accepted
decided_at: 2026-08-01
supersedes:
  - DECISION-0001
  - DECISION-0005
---

# Streamlined Task Gates

## Context

品質gateを証跡の重複、同じcommandの反復、または転記値の形式照合で増やすと、独立したREVIEW/QAの検出力を高めずにTaskの完了を遅らせる。Wiki knowledgeは再利用可能な場合にだけ明示ingestされるため、全Taskの完了前提にはならない。

## Decision

標準経路はplanning commit、製品candidate commit、candidateを第2親に持つcompletion no-ff mergeの三つのtransactionとする。planning/completionはlock、scope検査、全体validation、commitを一回ずつ行い、同一内容を親が検証済みならhookでの重複validationを省略できる。不一致又は通常commitはfail-closedでhook検証する。

candidateの正本はmain側HANDOVERの`candidate_commit`だけとし、candidate treeとmergeの関係はGitから導出する。Reviewerはcandidate差分とDEVのcandidate-bound check証跡を独立監査し、同じ`make check`を目的なく再実行しない。QAはcaseごとに`evidence-review`、`focused-rerun`、`live-e2e`を選ぶ。標準証跡はcase ID、HANDOVER candidate、command、結果とし、追加の実行環境・cache・exit・artifact情報は該当する場合だけ記録する。

candidateが変わった場合は影響するfocused testを再実行し、影響を限定できない場合だけfull rerunを行う。根拠チェックリストに基づくQA結果の転用はしない。Wiki ingestとreceiptは明示依頼時の任意成果物であり、完了の必須gateにしない。見積pointsは参考情報に残すが、行数・ファイル数との算術一致で完了を止めない。

## Consequences

- PLAN、QA_PLAN、独立REVIEW/QA、candidate固定、Mainのmerge所有、fail-closedなscope/権限境界は維持する。
- 新しい必須rule又は機械gateは、反復したfailure、又は重大なsecurity/permission/不可逆復旧failureを直接検出できる場合だけ追加する。
- 10 Taskごとの既存retrospectiveで検出価値、false positive、時間、保守費を見直し、低価値のruleを削除又は警告化する。専用の棚卸し証跡は作らない。
- DECISION-0005は履歴として不変のまま残る。

## Evidence

- [TASK-0037](../../tasks/TASK-0037-streamline-task-gates/TASK.md)
- [TASK-0037 HANDOVER](../../tasks/TASK-0037-streamline-task-gates/HANDOVER.md)
- [TASK-0037 QA PLAN](../../tasks/TASK-0037-streamline-task-gates/QA_PLAN.md)
