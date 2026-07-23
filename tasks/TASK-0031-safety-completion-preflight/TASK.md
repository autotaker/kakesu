---
task_id: "TASK-0031"
title: "安全契約の完了経路をDEV前検査可能にする"
status: plan
created_at: "2026-07-20"
---

# TASK-0031 安全契約の完了経路をDEV前検査可能にする

## Planning input packet（Main Agent所有）

### 目的

安全契約Taskの予定pathと必須生成物が完了checkerで受理されるかをDEV前にfail-closedで検査する。

### 対象と対象外

- 対象: `task-check`のpreflight phase、PLANの機械可読な予定path、実差分との照合、用語集索引の許可、tests/Make target/template/docs。
- 対象外: 外部運用リポジトリcommitの束縛、Lap30 skill縮約、既存Done Taskへの遡及要求。

### 受け入れ条件

- AC-1: 新契約を選ぶ安全契約PLANは予定変更pathと生成pathを機械可読に宣言し、DEV前preflightが許可外path・欠落・重複を拒否する。
- AC-2: `docs/99-glossary-index.md`を用語検査から生成される安全契約pathとして明示的に扱える。
- AC-3: Done検査は実際の候補差分が承認済み予定path内であることと、宣言した生成pathが欠落していないことを検証する。
- AC-4: 旧Taskは従来どおり検証でき、新契約は明示的なversion opt-inでのみ適用される。
- AC-5: 正常、許可外path、生成path欠落、実差分逸脱、旧契約互換の自動テストがある。

### 安定した参照

- REF-1: `scripts/task/check-task.mjs`の安全契約allowlist/done検査。
- REF-2: `scripts/validate-terminology.py --write`が更新する用語集と索引。

### 依存状態

- 依存なし。TASK-0030をunblockするが、TASK-0030の未commit差分を入力にしない。

### 許可パス

- `scripts/task/check-task.mjs`
- `scripts/task/development-process.test.mjs`
- `Makefile`
- `templates/task/PLAN.md`
- `docs/development/{development-process.md,task-management.md}`
- `docs/glossary.yml`
- `docs/99-glossary-index.md`

### 完了経路preflight

- 現行tests、branch/worktree作成権限、node依存、生成物更新commandをMainが確認する。
- Lap30ログはこの障害修正Taskでは開始せず、通常のTask時間として証跡化する。

### 未決事項

- 新契約の識別field名とCLI引数はPLANで決める。

### Scope reconciliation

- 2026-07-20T10:04:00+09:00 Main承認: `make check`が既存の決定的生成commandで示した`docs/glossary.yml`と`docs/99-glossary-index.md`を追加する。AC・設計・QA期待値は変更せず、生成結果以外へscopeを広げない。

## 背景

TASK-0030は承認後の`make check`で初めて生成索引が必要と判明し、現行allowlistでは安全契約として閉じられなかった。TASK-0024と同じ「merge付近で完了不能が判明」を自動検査で防ぐ。

## 完成の定義

- [x] AC-1〜AC-5を満たす。
- [x] PLAN/QA_PLAN、独立REVIEW/QA、`make check`がPASSする。
- [x] Mainがno-ff mergeし、同一candidate treeを確認する。
