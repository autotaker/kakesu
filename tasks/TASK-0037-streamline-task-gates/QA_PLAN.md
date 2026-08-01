---
task_id: "TASK-0037"
change_class: product
status: approved
qa_agent: "qa-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T04:23:26Z"
implementation_reviewed_at: ""
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0037 QA PLAN

## 方針

現 main の TASK-0037 実装を baseline とし、固定した HANDOVER の `candidate_commit` を唯一の対象として同一 candidate から独立に確認する。対象差分は unified lifecycle と estimate の未使用算術/規則だけである。`make check` は DEV が candidate に対して実行した記録を監査し、同じ目的で QA が再実行しない。上限付きの既存 Node process tests を focused rerun する。外部サービス、実 OS 権限、配置または restart を変更しないため live E2E は不要である。

QA 結果の標準証跡は、各ケースの ID、HANDOVER の candidate、command、result とする。該当しない cache、exit 詳細、artifact digest、version/mode、CF checklist、全 Task Wiki receipt は求めない。

## 受け入れ条件との対応

| ケース ID | AC-ID | 観測方法 | 実施モード / 理由 |
|---|---|---|---|
| QA-001 | AC-7 | legacy qa→done の process fixture で、再利用可能な知識がない場合に Wiki receipt/Agent を自動要求・生成せず完了できることを確認する。 | `focused-rerun` / isolated lifecycle fixture が成功・回帰を決定的に再現できる。 |
| QA-002 | AC-6, AC-12 | development process test を実行し、未使用の `estimatePoints` 算術とその tests を削除しても `estimate_points` 自体は計画上の参考値として残ることを、source と PLAN template の差分で確認する。行数/ファイル数の数値見積と見積規則 check が残らないことも確認する。 | `focused-rerun` / deterministic test と candidate source/template audit で直接確認できる。 |
| QA-003 | AC-5, AC-9, AC-10, AC-12 | HANDOVER の `candidate_commit` と candidate diff を突合し、DEV の candidate-bound `make check` command/result 証跡を監査する。role・sandbox/権限境界、許可 path、秘密情報不在、Reviewer の独立性、QA focused rerun、P0/P1 拒否、Main merge 所有など既存の安全境界を候補差分が弱体化していないことを確認する。 | `evidence-review` / candidate と証跡の独立監査で、同じ root check を重複実行せずに確認する。 |

## 実行手順と期待結果

1. HANDOVER から `candidate_commit` を取得し、以後の diff と証跡をその commit に固定する。candidate が一意に読めない、または HANDOVER が candidate 側にある場合は FAIL とする。
2. QA-001 は `node --test scripts/task/unified-lifecycle.test.mjs` を candidate に対して一度だけ実行する。legacy qa→done が Wiki receipt/Agent を自動要求・生成せず PASS することを期待する。
3. QA-002 は `node --test scripts/task/development-process.test.mjs` を candidate に対して一度だけ実行し、candidate の `scripts/task/lib.mjs` と `templates/task/PLAN.md` を監査する。不要な算術/check は消え、`estimate_points` は参考値として維持されることを期待する。
4. QA-003 は HANDOVER の candidate、DEV `make check` 証跡、Reviewer 成果物、QA-001〜002 の結果を監査する。いずれも PASS し、role/sandbox/権限が不明、同一人物による DEV/Reviewer/QA、許可外変更または秘密情報があれば FAIL とする。`make task-check TASK=TASK-0037` は candidate pre-merge では成立しないため QA に要求せず、completion transaction と merge 後に Main が実行する。

## 境界・異常・回帰

- legacy qa→done の Wiki receipt/Agent 自動要求・生成以外の lifecycle 契約は baseline から弱体化しない。
- `estimate_points` は残し、未使用の算術・行数/ファイル数見積・見積規則 check だけを除く。
- Reviewer の独立監査と QA の focused rerun は、重複する `make check` の再実行を要求せずに残る。
- candidate の管理事実は Git から導出し、role・権限・秘密・scope の fail-closed 境界を候補差分が緩めない。

## 実装後の再確認

- [ ] fixed candidate の実装差分とレビュー結果を確認した。
- [ ] 上記 command と期待結果を candidate 実装に照らして確認した。
- [ ] 期待結果または範囲を変更した場合、main Agent の承認を得た。
