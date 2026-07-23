---
task_id: "TASK-0026"
change_class: safety_contract
status: approved
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "sol-high"
approved_dev_profile_reason: "AgentのGit権限境界を変更する安全契約のため"
approved_dev_profile_risk_signals: ["contract", "authority"]
approved_by: "main-agent-sol-high"
approved_at: "2026-07-20"
planning_reviewed_by: "reviewer-agent-terra-medium"
planning_review_decision: pass
planning_reviewed_at: "2026-07-20"
classification_approved_by: "main-agent-sol-high"
classification_approved_at: "2026-07-20"
classification_approval_reason: "製品成果物を変更せず、ReviewerとQAの軽微修正に関する必須開発統制だけを変更したため"
planned_implementation_files: 0
planned_implementation_lines: 0
estimate_points: 1
---

# PLAN — TASK-0026

## 分類と根拠

安全契約変更。製品コード、テスト、runtime/build設定、Schema、依存、生成物は変更しない。一方で、Reviewer/QA Agentのcommit権限と、軽微修正後に要求するゲートを変更するため、権限境界および必須開発統制に影響する。

## 変更方針

Reviewer/QAが自ら軽微と判断した指摘は、当該Task worktreeで修正、stage、commitできるようにする。MainはTask branchへの取り込み確認により当該指摘を解消済みとしてPASSにでき、軽微修正にDEV差戻し、再REVIEW、再QA、`qa_carry_forward`を要求しない。

軽微でない（挙動、要件、安全境界を変える）と担当Agentが判断した場合は通常経路へ戻す。Mainだけが`main`へのmerge/pushを所有し、子Agentは直接行わない。軽微の閉じた許可リスト、SLOC上限、追加承認チェックリストは設けない。

## 想定変更範囲

- `AGENTS.md`
- `docs/development/development-process.md`
- `docs/development/agent-roles.md`
- `docs/development/code-review.md`
- `docs/development/qa.md`
- `templates/task/REVIEW_RESULT.md`
- `templates/task/QA_RESULT.md`
- `.agents/skills/run-efficient-task-delivery/SKILL.md`
- `../agent-harness-work/AGENTS.md`
- `../agent-harness-work/.agents/skills/run-lap30/SKILL.md`

範囲外: 製品実装、テスト、依存、Schema、生成物、および子Agentへの`main`統合・push権限の移譲。

## 実施順序と証跡

1. 独立QAがTASK.mdから先に`QA_PLAN.md`を作成し、全QAケースの既存`qa_execution_mode`、理由、fail-closed条件を保つ前提で、軽微修正のPASS証跡と通常経路へ戻す条件を定義する。
2. 独立した計画レビューで、TASK受け入れ条件との整合、権限分離、全対象文書・template・skillの一貫性、製品成果物非変更を確認する。不整合または安全上の含意があれば実装へ進めずPLANを修正する。
3. MainがPLANとQA_PLANを承認後、承認済み範囲だけを実装する。子Agent間の変更を戻さず、`main`へのmerge/pushは行わない。
4. 実装時は、通常のcandidate/独立REVIEW/QA/`merge_tree`証跡の規定を、軽微修正のTask-branch commitを解消済みとする新しい統制と矛盾なく更新する。製品DEV/REVIEW/QAのPASS証跡はこの安全契約変更自体には作らない。

## 受け入れ確認

- 対象文書、template、skillが同じ権限分離と軽微修正後のPASS規則を示す。
- 非軽微と判断された変更は通常の差戻し経路に戻る。
- Mainのみが`main`へのmerge/pushを行う。
- `git diff --check`、変更した契約/証跡に対応する所定のscope・hook検査を実行する。製品テストは製品PASSの証拠として実行・記録しない。

## Main 承認

- [x] PLAN承認
- [x] QA_PLAN承認
- [x] 実装開始承認
