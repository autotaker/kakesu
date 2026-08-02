---
kind: decision
decision_id: DECISION-0005
title: Risk-based Candidate QA
status: superseded
decided_at: 2026-07-20
supersedes:
  - DECISION-0001
---

# Risk-based Candidate QA

## Context

全自動testをマージ後に再実行するだけでは、QAの独立性を十分に表せず、candidateに結び付くDEV証跡、negative case、testの弱体化を独立に監査する必要がある。一方、実権限、外部作用、実配置などの受け入れ真実は、実環境での確認を必要とする。

## Decision

QA_PLANはDEV開始前に各caseへ`evidence-review`、`focused-rerun`、`live-e2e`の一つと理由を割り当てる。QAはcandidate-bound証跡とtestの失敗検出能力を独立に評価する。高リスクcaseを`focused-rerun`にできるのは、受け入れ真実をhermetic、deterministic、boundedなfixtureで完全再現できる場合だけとする。実OS権限/auth、実配置、外部作用、実restart/rollback/cleanup、環境固有integrationは`live-e2e`とし、環境または安全なcleanupが不明ならblockedのままにする。

ReviewerとQAは同じcandidate commit/treeから独立かつ並行に開始し、相互のPASSを開始条件にしない。Mainだけがcandidate treeとmerge treeを照合し、REVIEW修正後に`qa_carry_forward`、focused rerun、full rerunを判断する。carry-forwardは非挙動かつ明示した低リスク条件を全て満たし、旧新candidate、tree/diff、影響case、再実行証拠、理由を記録できる場合だけ許可する。

candidate treeとmerge treeが同一で環境依存caseがない場合は、マージ後の全面QAを繰り返さない。環境依存caseはマージ後もcase単位で確認する。

## Consequences

- QAの独立性は、無差別な再実行回数ではなく、期待値、証跡、test妥当性の独立した判断として維持する。
- 証跡不足、candidate/tree不一致、高リスク信号、影響不明は`evidence-review`のPASSを許さない。
- QA FAIL、受け入れ条件またはQA_PLANの変更、認証認可・秘密・権限、IPC/Schema/設定/依存、並行性・lifecycle・persistence・error/fail-closed、test削除/弱体化、影響不明はcarry-forwardを禁止し、再実行へfail-closedする。
- FAIL分類と差し戻し先、candidate/merge treeの判断、Git操作はMainが所有する。
- DECISION-0001は履歴として不変のまま残る。

## Evidence

- [TASK-0024](../../tasks/TASK-0024-risk-based-qa-evidence/TASK.md)
- [TASK-0024 PLAN](../../tasks/TASK-0024-risk-based-qa-evidence/PLAN.md)
- [TASK-0024 QA PLAN](../../tasks/TASK-0024-risk-based-qa-evidence/QA_PLAN.md)
