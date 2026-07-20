---
task_id: "{{TASK_ID}}"
change_class: ""
status: draft
planner_agent: ""
approved_by: ""
approved_at: ""
planning_reviewed_by: ""
planning_review_decision: "pending"
planning_reviewed_at: ""
classification_approved_by: ""
classification_approved_at: ""
classification_approval_reason: ""
# safety_contract_version: 2
# safety_contract_planned_paths: []
# safety_contract_generated_paths: []
planned_implementation_files: 0
planned_implementation_lines: 0
estimate_points: 1
---

# {{TASK_ID}} PLAN

## AC対応

TASKの条件本文を再掲せず、`planning input packet`のAC-IDに設計を対応させる。

| AC-ID | 設計判断 | 変更パス | 実施順序 | 失敗時の扱い |
|---|---|---|---|---|
| AC-1 | TODO | TODO | TODO | TODO |

## 関連Wikiと判断

- TODO

## 補足設計

### 代替案と不採用理由

- TODO

### 責務・境界・不変条件

- TODO

### 移行・互換性

- TODO

## 変更予定

見積もり対象は実装コード、Schema、設定ファイルだけとする。

| ファイル | 種別 | 概算変更行数 | 変更内容 |
|---|---|---:|---|
| TODO | implementation | 0 | TODO |

## 見積もり

```text
file_score = ceil(planned_implementation_files / 3)
line_score = ceil(planned_implementation_lines / 200)
estimate_points = 1, 2, 3, 5, 8, 13のうちmax(1, file_score, line_score)以上の最小値
```

## 実装手順

1. TODO

## 検証計画

- TODO

## 未解決事項

- なし

## main Agentレビュー

- [ ] TASKの全AC-IDへ設計判断、パス、順序、失敗時の扱いを対応させ、条件本文を複製していない。
- [ ] 設計観点と代替案を検討している。
- [ ] QA_PLANがTASK-firstで独立作成されている。
- [ ] `dependency-ready reconciliation`と完了経路preflightが完了している。
- [ ] 見積もりが規則どおりである。
- [ ] DEV開始を承認した。

安全契約変更でv2契約を選ぶ場合は、コメントを外して`safety_contract_version: 2`と予定パス・生成パスの配列を記録し、DEV前に`make task-preflight TASK={{TASK_ID}}`を実行する。変更しない種別は空配列とし、通常の予定パスと生成パスを重複させない。独立計画レビューのPASSとMainの分類承認をフロントマターへ記録する。分類変更時はTask、PLAN、QA_PLANを再承認し、承認者と時刻を更新する。
