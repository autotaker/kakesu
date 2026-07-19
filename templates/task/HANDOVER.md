---
task_id: "{{TASK_ID}}"
status: draft
completed_at: ""
---

# {{TASK_ID}} HANDOVER

## 成果

- TODO

## candidate-bound DEV証跡

- `candidate_commit`:
- `candidate_tree`:

| ケース ID | コマンド/テスト | 環境/フィクスチャ | cache条件 | exit | 成果物 ダイジェスト | 未実施理由 |
|---|---|---|---:|---:|---|---|
| QA-001 | TODO | TODO | TODO | TODO | TODO | なし |

- QAへ渡すネガティブ検出証拠、テスト弱体化の有無を判定できる差分ダイジェスト: TODO

## 主要な変更

- TODO

## 検証結果

- TODO

## 判断

- TODO
- 選択: `not-applicable | qa_carry_forward | focused-rerun | full-rerun`
- Main判断の旧新コミット/tree、diff、影響ケース、再実行証拠、理由: TODO
- carry-forwardの非挙動・明示低リスク条件と禁止条件の確認: TODO
- `merge_tree`と案 treeの比較: `pending`

## 既知の制約と未解決事項

- なし

環境依存ケースがある場合、install/deploy/config生成、実権限、外部作用、実restart/ロールバック/クリーンアップのマージ後確認を省略しない。実環境または安全なクリーンアップが不明なケースはblockedとして残す。

## 運用上の注意

- なし

## Wikiへ引き渡す知識

### 再利用可能な知識

- TODO

### 反例・失敗・注意点

- TODO

### 更新候補ページ

- TODO

## ブートストラップ例外

- 該当なし
