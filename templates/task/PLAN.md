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
planned_implementation_files: 0
planned_implementation_lines: 0
estimate_points: 1
---

# {{TASK_ID}} PLAN

## 受け入れ条件の具体化

| 条件 | 観測方法 | 根拠 |
|---|---|---|
| TODO | TODO | TODO |

## 関連Wikiと判断

- TODO

## 設計

### 選択案

TODO

### 代替案と不採用理由

- TODO

### 責務と境界

- TODO

### 不変条件

- TODO

### 失敗時・移行・互換性

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

- [ ] 受け入れ条件が検証可能である。
- [ ] 設計観点と代替案を検討している。
- [ ] QA計画を作成できる。
- [ ] 見積もりが規則どおりである。
- [ ] DEV開始を承認した。

安全契約変更では、独立計画レビューのPASSとMainの分類承認をフロントマターへ記録する。分類変更時はTask、PLAN、QA_PLANを再承認し、承認者と時刻を更新する。
