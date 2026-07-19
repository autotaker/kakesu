---
task_id: "{{TASK_ID}}"
change_class: ""
status: draft
qa_agent: ""
approved_by: ""
approved_at: ""
revision: 1
implementation_reviewed_at: ""
expectation_changed: false
expectation_change_approved_by: ""
---

# {{TASK_ID}} QA PLAN

## 方針

この計画はTASKだけからDEV開始前に作成し、各ケースへ`qa_execution_mode`を一つだけ理由付きで割り当てる。`evidence-review`はcandidate-bound証跡の独立監査、`focused-rerun`はhermetic・deterministic・上限付き フィクスチャで完全再現できる高リスクケース、`live-e2e`は実OS権限/auth（sudo/PAMを含む）、実配置、外部作用、実restart/ロールバック/クリーンアップ、環境固有integrationを必要とするケースに使う。条件が不明な場合は強いモードまたはblockedへfail-closedし、PASSで代替しない。

## 前提と環境

- TODO

## 受け入れ条件との対応

| ケースID | 受け入れ条件 | `qa_execution_mode` / 理由 | 操作 | 期待結果 | 必要証跡 |
|---|---|---|---|---|---|
| QA-001 | TODO | `evidence-review` / TODO | TODO | TODO | ケース、案 コミット/tree、コマンド/テスト、環境/フィクスチャ、cache、exit、成果物 ダイジェスト、未実施理由、ネガティブ検出能力、テスト弱体化の有無 |

## 境界・異常・回帰

- 高リスク、証跡欠落、ダイジェスト/コミット/tree不一致、テスト削除/弱体化、影響不明は`evidence-review` PASSにしない。
- `qa_carry_forward`は`QA_RESULT.md`の`CF-1`から`CF-7`を全て証明できる場合だけ許可する。影響QAケース集合が空でなければ該当ケースを再実行し、影響を限定できなければ全面再実行とする。QA FAIL、受け入れ条件/QA_PLAN変更、認証認可、秘密、sudo/PAM、IPC/Schema/設定/依存、並行性/ライフサイクル/persistence/エラー/fail-closed、テスト削除/弱体化、影響不明、証跡と評価対象の案/tree不一致では禁止する。
- `live-e2e`の環境または安全なクリーンアップが用意できない場合はblockedとして残す。

## 実施不能時の扱い

- 未実施コマンド、環境、原因、残余リスクを記録し、別モードの結果でPASSにしない。FAIL候補の分類と復帰先はMainが判断する。

## 案とマージ後確認

- DEVは`candidate_commit`、`candidate_tree`、ケース ID、コマンド/テスト、環境/フィクスチャ、cache条件、exit、成果物 ダイジェスト、未実施理由をHANDOVERへ記録する。
- REVIEWとQAは同一案から相互のPASSを前提にせず独立に開始する。
- 修正後の`qa_carry_forward`はMainだけが`QA_RESULT.md`の`CF-1`から`CF-7`を全て記録して選択する。
- `merge_tree == candidate_tree`かつ環境依存ケースなしなら全面確認を省略できるが、環境依存ケースはマージ後もケース単位で確認する。

## 実装後の再確認

- [ ] 実装差分とレビュー結果を確認した。
- [ ] 操作手順を現行実装に合わせた。
- [ ] 期待結果または試験範囲の変更有無を確認した。
- [ ] 期待結果または範囲を変更した場合、main Agentの承認を得た。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | {{DATE}} | | 初版 | `pending` |
