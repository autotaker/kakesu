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

軽微指摘をQA Agentが直接修正した場合は、修正コミットとTask ブランチへの取り込みを記録する。取り込み後は解消済みとしてPASSにでき、再QAまたは`qa_carry_forward`を要求しない。

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
- `CF-1` 旧QA PASSと旧`candidate_commit`/`candidate_tree`の束縛: `pending` / 証拠: TODO
- `CF-2` 旧新案の全差分と差分ダイジェスト: `pending` / 証拠: TODO
- `CF-3` 変更は実行されない誤字、空白、コメント、リンク、証跡メタデータだけで、製品挙動、ランタイム、テスト、Schema、設定、依存、生成物、外部公開契約または安全契約、受け入れ条件、QA_PLANの意味変更がない: `pending` / 証拠: TODO
- `CF-4` 影響QAケース集合: `[]`。空でなければcarry-forwardせず該当ケースを再実行する。
- `CF-5` 独立レビュアーによる挙動、テスト、安全性、契約への影響なしの確認と、新案の`make check` PASS: `pending` / レビュアー証拠、コマンド、結果: TODO
- `CF-6` QA FAIL、受け入れ条件/QA_PLAN変更、認証認可、秘密、sudo/PAM、IPC/Schema/設定/依存、並行性/ライフサイクル/persistence/エラー/fail-closed、テスト削除/弱体化、影響不明、証跡と評価対象の案/tree不一致が全て偽: `pending`
- `CF-7` Main記録（旧新コミット/tree、全差分とダイジェスト、空の影響ケース集合、レビュアー/`make check`証拠、理由）: TODO

## 結論

`pending`
