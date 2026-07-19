---
task_id: "{{TASK_ID}}"
status: pending
reviewer_agent: ""
reviewed_commit: ""
candidate_commit: ""
candidate_tree: ""
decision: pending
make_check: pending
reviewed_at: ""
---

# {{TASK_ID}} REVIEW RESULT

## 対象

- ブランチ:
- 案 コミット/tree:
- Task / PLAN / QA PLAN:

## 実行した検査

| コマンド | 結果 | 備考 |
|---|---|---|
| `make check` | `pending` | |

## 受け入れ条件の確認

| 条件 | 結果 | 根拠 |
|---|---|---|
| TODO | `pending` | TODO |

## QAとの独立性

- QAと同一案から評価を開始した: `pending`
- 相互のPASSを開始条件にしていない: `pending`
- 案が変わった場合の再評価/再束縛: `pending`

## 指摘

| ID | 重大度 | 状態 | 内容 | 根拠 |
|---|---|---|---|---|
| - | - | - | 指摘なし | - |

## 残存リスク

- TODO

## 結論

`pending`
