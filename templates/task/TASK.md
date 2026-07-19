---
task_id: "{{TASK_ID}}"
title: {{TITLE_YAML}}
status: draft
created_at: "{{DATE}}"
---

# {{TASK_ID}} {{TITLE}}

## 目的

<!-- 達成したい結果を記載する。 -->

## 背景

<!-- なぜ今必要か、現在の問題、関連する制約を記載する。 -->

## スコープ

### 対象

- TODO

### 対象外

- TODO

## 受け入れ条件

- [ ] TODO
- [ ] 各QA ケースに`qa_execution_mode`（`evidence-review | focused-rerun | live-e2e`）と選択理由、fail-closed条件がある。
- [ ] DEV 案、独立REVIEW/QA、修正後のcarry-forwardまたはrerun、`merge_tree`確認を証跡化できる。

## 検討すべき設計観点

- TODO

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
