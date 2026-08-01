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

この計画はTASKの受け入れ条件を基準に、DEV開始前にQAが独立作成する。各ケースには一つの実施モードと、その理由を記録する。

## 受け入れ条件との対応

| ケース ID | AC-ID | 観測方法 | 実施モード / 理由 |
|---|---|---|---|
| QA-001 | AC-1 | TODO | `evidence-review` / TODO |

実施モードは `evidence-review`、`focused-rerun`、`live-e2e` のいずれかとする。実施不能なケースは理由を記録し、別モードのPASSで置き換えない。

## 境界・異常・回帰

- TODO

## 実装後の再確認

- [ ] 実装差分とレビュー結果を確認した。
- [ ] 操作手順と期待結果を現行実装に合わせた。
- [ ] 期待結果または範囲を変更した場合、main Agentの承認を得た。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | {{DATE}} | | 初版 | `pending` |
