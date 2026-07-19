---
task_id: "{{TASK_ID}}"
status: pending
qa_agent: ""
tested_commit: ""
candidate_commit: ""
candidate_tree: ""
merge_tree: ""
decision: pending
tested_at: ""
---

# {{TASK_ID}} QA RESULT

## 対象

- 案 コミット/tree:
- `main` / merge tree:
- `merge_tree`はマージ後にMainが記録し、案 QAでは未設定とする:
- QA PLAN 改訂:
- 環境:

## 結果

| ケースID | モード | 対象案 コミット/tree | 結果 | 証跡（コマンド/テスト、環境/フィクスチャ、cache、exit、成果物 ダイジェスト、ネガティブ検出能力、テスト弱体化の有無） | 未実施/blocked理由 |
|---|---|---|---|---|---|
| QA-001 | `evidence-review` | TODO | `pending` | TODO | なし |

## 発見事項

| ID | FAIL分類 | 影響 | 差し戻し候補 | 内容 |
|---|---|---|---|---|
| - | - | - | - | なし |

## main Agent判断

- 結論: `pending`
- 差し戻し先:
- revert / バグ化:
- 判断理由:

## 未実施項目

- なし

## Main-owned `qa_carry_forward` / 再実行判断

- 選択: `not-applicable | qa_carry_forward | focused-rerun | full-rerun`
- Main記録（旧新コミット/tree、diff、影響ケース、再実行証拠、理由）: TODO
- 非挙動かつ明示した低リスク条件を全て証明した: `pending`
- 禁止条件（QA FAIL、受け入れ/QA_PLAN変更、認証認可・秘密・sudo/PAM・IPC/Schema・設定/依存・並行性/ライフサイクル/persistence/エラー/fail-closed、テスト削除/弱体化、影響不明、案/tree不一致）を確認した: `pending`

## 結論

`pending`
