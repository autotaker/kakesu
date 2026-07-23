---
task_id: "TASK-0025"
title: "task-checkを安全契約変更の軽量経路へ対応させる"
status: done
created_at: "2026-07-20"
---

# TASK-0025 task-checkを安全契約変更の軽量経路へ対応させる

## 目的

`task-check`が製品変更と安全契約変更を明示的に分類し、安全契約変更では製品`REVIEW_RESULT.md`/`QA_RESULT.md` PASS、完全HANDOVER、Wiki receiptを捏造せずにDone判定できるようにする。製品変更の既存ゲートは緩めない。

## 背景

TASK-0024で安全契約変更の軽量経路を規範化し、製品変更と検証を完了した。しかし現行`task-check`は全Taskへ製品QA経路を一律適用するため、安全契約Taskを正直な証跡のままDoneにできなかった。TASK-0024は架空PASSを作らずblockedとしており、本Task完了後に再検査して閉じる。

## スコープ

### 対象

- Taskの変更分類を機械判定できる正本fieldと互換性規則。
- `scripts/task/check-task.mjs`とprocess testの分類別Done gate。
- 安全契約変更に必要なPLAN、TASK-first QA_PLAN、独立計画レビュー、Main承認、統制文書検査、merge tree証拠の検証。
- TASK-0024をfixtureとして使う回帰testと閉鎖手順。

### 対象外

- 製品Taskの既存REVIEW/QA/HANDOVER/Wiki要件の削除または緩和。
- 純粋証跡保守を架空Taskとして必須化すること。
- QA mode、carry-forward、役割分離、Main所有Gitの変更。

## 受け入れ条件

- [ ] 変更分類は未指定・未知・矛盾時にfail-closedし、既存Taskの移行規則が明記される。
- [ ] 製品変更のDone gateは現行のREVIEW PASS、QA PASSまたはaccepted-with-bugs、HANDOVER、Wiki receipt、commit/tree検査を維持する。
- [ ] 安全契約変更のDone gateは承認済みPLAN/QA_PLAN、独立計画レビュー、Main承認、対象検査、merged commit/treeを要求し、製品REVIEW/QA/HANDOVER/Wikiの架空PASSを要求しない。
- [ ] 分類を偽って製品変更を安全契約経路で閉じるnegative testと、TASK-0024を正しく閉じる回帰testがPASSする。
- [ ] 既存Task証跡を遡及破壊せず、移行前Taskの解釈が決定的である。
- [ ] 各QA ケースに`qa_execution_mode`（`evidence-review | focused-rerun | live-e2e`）と選択理由、fail-closed条件がある。
- [ ] DEV 案、独立REVIEW/QA、修正後のcarry-forwardまたはrerun、`merge_tree`確認を証跡化できる。

## 検討すべき設計観点

- 分類fieldの所有者、承認時点、途中変更時の再承認を閉じた契約にする。
- ファイル拡張子やSLOCだけで安全契約と判定せず、Task契約と実差分の両方を照合する。
- 軽量経路の導入を製品ゲート回避に使えないnegative fixtureを先に固定する。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] 必要な実装、テスト、文書が完成している。
- [ ] 独立レビューと`make check`がPASSしている。
- [ ] REVIEWとQAが同一案から独立に評価され、未実施/blocked理由とMain判断が記録されている。
- [ ] `merge_tree`と承認案 treeを比較し、環境依存ケースのマージ後確認を完了している。
- [ ] HANDOVERがWiki Agentに取り込みされている。

## 関連コンテキスト

### 意味 Wiki

- 未調査

### 判断

- 未調査

### 適用しなかった重要な判断

- なし
