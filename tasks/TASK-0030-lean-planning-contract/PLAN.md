---
task_id: "TASK-0030"
change_class: safety_contract
status: approved
safety_contract_version: 2
safety_contract_planned_paths:
  - ".agents/skills/run-efficient-task-delivery/SKILL.md"
  - "docs/development/development-process.md"
  - "docs/development/agent-roles.md"
  - "docs/development/task-management.md"
  - "templates/task/TASK.md"
  - "templates/task/PLAN.md"
  - "templates/task/QA_PLAN.md"
  - "docs/glossary.yml"
safety_contract_generated_paths: []
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "sol-high"
approved_dev_profile_reason: "必須開発統制と複数の規範文書・skillの責務境界を横断して変更するため"
approved_dev_profile_risk_signals: ["safety_contract", "cross_cutting"]
approved_by: "main-agent-sol-high"
approved_at: "2026-07-20T10:24:56+09:00"
planning_reviewed_by: "reviewer-agent-terra-medium"
planning_review_decision: pass
planning_reviewed_at: "2026-07-20T10:24:30+09:00"
classification_approved_by: "main-agent-sol-high"
classification_approved_at: "2026-07-20T10:24:57+09:00"
classification_approval_reason: "v2 preflightで製品成果物非変更と承認済み8 pathを検証し、計画入力の必須統制だけを変更するため"
planned_implementation_files: 0
planned_implementation_lines: 0
estimate_points: 1
---

# TASK-0030 PLAN

## 分類と境界

これは必須開発統制を更新する `safety_contract` である。対象は製品側の規範文書、Task template、delivery skill、および本Taskの証跡に限る。製品コード、test、runtime/build設定、Schema、依存、生成製品入力/成果物、既存Lap Schema/JSONL、運用リポジトリの`run-lap30`は変更しない。独立したTASK-first `QA_PLAN.md`、計画レビュー、Mainの分類承認を要するが、製品DEV、製品`REVIEW_RESULT.md`、製品`QA_RESULT.md`のPASSは作成しない。

## AC対応

| AC-ID | 設計判断 | 変更path |
|---|---|---|
| AC-1, AC-2 | Main所有packetを唯一のplanning入力とし、PLANは設計判断、QA_PLANは観測だけをAC-IDで対応させる。 | `templates/task/{TASK.md,PLAN.md,QA_PLAN.md}`、`docs/development/{agent-roles.md,task-management.md}` |
| AC-3 | dependency-independent planningとdependency-ready reconciliationを分離し、後者の設計・scope影響を再承認へ戻す。 | `docs/development/development-process.md`、`templates/task/{TASK.md,PLAN.md,QA_PLAN.md}` |
| AC-4, AC-5 | active planning、dependency wait、preflight、Lap開始の境界を正本へ寄せる。 | `docs/development/development-process.md`、`.agents/skills/run-efficient-task-delivery/SKILL.md` |
| AC-6 | delivery skillを実行判断と正本参照へ縮約する。 | `.agents/skills/run-efficient-task-delivery/SKILL.md` |
| AC-7 | 既存統制と記録契約を参照で保持する。 | 全変更path |

## 未決設計判断

1. packetは新規証跡ファイルではなくTask template内のMain所有sectionとする。TASKが受け入れ条件の正本であり、PLAN/QA_PLANはpacket又は条件本文を複製しない。
2. PLAN/QA_PLANの対応表は同じAC-ID集合を使い、設計判断と観測を分離する。
3. dependency-ready reconciliationが設計、scope、期待結果を動かす場合は、DEV前にPLAN/QA_PLANの再承認を必須にする。
4. active planning、wait、preflight、Lap開始を別の状態として記録し、未解決preflightでLapを開始しない。

## dependency-ready reconciliation

TASK-0031のmerge commit `b60e691182127de73fff36660f7d3cfe26bc01e9` により安全契約v2の事前scope検査が利用可能になった。既存ACと設計判断は変更せず、予定scopeを上記8 pathへ固定する。用語sourceとして`docs/glossary.yml`を追加するが、generator再実行後も`docs/99-glossary-index.md`は内容不変のため生成pathは空とする。

## 変更予定

| path | 変更責務 |
|---|---|
| `docs/development/{development-process.md,agent-roles.md,task-management.md}` | packetのMain所有、PLAN/QAのAC-ID参照分離、reconciliation、preflight、active/wait分類を正本化する。 |
| `templates/task/{TASK.md,PLAN.md,QA_PLAN.md}` | packet sectionとAC-ID対応欄を追加し、重複した受け入れ条件本文を要求しない。 |
| `.agents/skills/run-efficient-task-delivery/SKILL.md` | 上記正本を参照し、planning input/preflight/依存準備を短く整合させる。 |
| `docs/glossary.yml` | planning packet、dependency reconciliation、active/wait、preflightに関する変更後の用語を正本へ整合させる。 |

`agents/openai.yaml`、運用リポジトリの`run-lap30`、Lap30 event Schema、JSONL、既存Task証跡、製品pathは変更予定に含めない。見積り対象の実装コード、Schema、設定ファイルはないため、見積りは既存算式どおり1 pointである。

## 実施順序

1. MainがTask templateに従うpacketを用意し、PlannerとQAへ同一入力を渡す。QAはPLANを入力にせずTASK/packetから独立にQA_PLANを作成する。
2. 規範文書とtemplateへpacket、AC-ID参照、reconciliation、計測/preflight境界を定義する。
3. delivery skillを正本参照へ寄せる。
4. 変更pathだけを検査し、独立計画レビューとMainの分類承認を得る。依存が後からreadyになった場合はDEV前にreconciliationを完了する。

## 失敗時・互換性

- packet、AC-ID対応、preflight又は依存状態が欠けるか矛盾する場合はPLAN/QA_PLANを承認せず、`requirement_gap`、`qa_plan_defect`、又は`environment_issue`候補としてMainへ戻す。
- dependency-ready差分がAC、設計、scope、QA期待値を変える場合は、差分だけで静かに進めずPLAN/QA_PLANの再承認へ戻す。
- preflight失敗はLap外の`not_started`又はblockedとして残し、active planningや開始済みLapへ算入しない。
- 文書の縮約で既存ゲート、ロール分離、Main所有Git、安全境界を弱める、又は既存Schema/JSONLの意味を変える差分が出た場合は安全契約変更としてFAILし、該当正本へ戻す。既存Task証跡とLapログは遡及変更しない。

## 検証

- `rg`で対象規範、template、delivery skillに残る受け入れ条件本文の複製と旧preflight/計測表現を確認し、AC-ID参照と正本参照が一貫することをレビューする。
- packet、dependency-ready差分、計測境界、preflight欠落を文書/template上で追跡し、各々が承認停止又は正しい計測分類になることを確認する。
- 用語generatorを実行し、`docs/glossary.yml`が妥当で`docs/99-glossary-index.md`に内容差分が生じないことを確認する。
- safety-contract対象の文書/template/skill検査、`git diff --check`、`make check`、`make work-check`を実行する。製品テスト又は製品PASSをこのTaskの検証証拠にしない。

## main Agentレビュー

- [x] AC-ID参照だけでTASK本文を複製していない。
- [x] packet、reconciliation、preflight、Lap計測の責務とfail-closed条件が明確である。
- [x] QA AgentがTASK-firstで独立した観測計画を作成できる。
- [x] safety-contractのpath境界、見積り、検証が妥当である。
- [x] 独立計画レビューと分類承認後にのみ次工程へ進める。
