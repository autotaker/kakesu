---
task_id: "TASK-0026"
title: "ReviewerとQAの軽微修正コミットを許可する"
status: done
created_at: "2026-07-20"
---

# TASK-0026 ReviewerとQAの軽微修正コミットを許可する

## 目的

Reviewer/QA Agentが軽微と判断した指摘を自ら修正・コミットし、Task branchへの取り込みをもって指摘解消とPASSにできるようにする。

## 背景

NitsまでDEVへ差し戻し、再REVIEW・再QA・carry-forward判定を要求する現行運用が遅延を生んだ。各Agentの専門判断を信頼し、Mainはmainへの統合所有だけを維持する。

## スコープ

### 対象

- Reviewer/QA Agentによる軽微修正のstage/commit許可。
- Task branchへ取り込まれた後のPASS確定。
- AGENTS、Agent責務、レビュー/QA文書、Task template、関連skillの整合。

### 対象外

- 軽微か否かの細かな許可リスト、SLOC上限、追加承認チェックリスト。
- mainへのmerge/push権限の子Agentへの移譲。

## 受け入れ条件

- [ ] Reviewer/QAは自ら軽微と判断した指摘をTask worktreeで修正・stage・commitできる。
- [ ] MainがTask branchへの取り込みを確認すれば、その指摘を解消済みとしてPASSにできる。
- [ ] DEV差し戻し、再REVIEW、再QA、`qa_carry_forward`は軽微修正には要求しない。
- [ ] 挙動、要件、安全境界を変えると担当Agentが判断した場合だけ通常の差し戻しへ戻す。
- [ ] Mainだけがmainへのmergeを所有し、子Agentはmainへの直接merge/pushをしない。
- [ ] 各QA ケースに`qa_execution_mode`（`evidence-review | focused-rerun | live-e2e`）と選択理由、fail-closed条件がある。
- [ ] DEV 案、独立REVIEW/QA、修正後のcarry-forwardまたはrerun、`merge_tree`確認を証跡化できる。

## 検討すべき設計観点

- 「軽微」の判断を閉じた列挙にせず、担当Reviewer/QAの裁量とする。
- commit作成権限とmain統合権限を分離する。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] 必要な実装、テスト、文書が完成している。
- [ ] 独立レビューと`make check`がPASSしている。
- [ ] REVIEWとQAが同一案から独立に評価され、未実施/blocked理由とMain判断が記録されている。
- [ ] `merge_tree`と承認案 treeを比較し、環境依存ケースのマージ後確認を完了している。
- [ ] HANDOVERがWiki Agentに取り込みされている。

## 関連コンテキスト

製品コード、製品test、runtime/build設定、製品Schema、製品依存、製品挙動を変更しない。

## 運用上のブロッカー

- 規約変更はmerge commit `c541dde2d6fb2c4c2c6abacb18a46daf65c55527`として完了した。
- 安全契約Taskを製品QA証跡なしでDoneにする`task-check`対応はTASK-0025で扱う。

### 意味 Wiki

- 未調査

### 判断

- 未調査

### 適用しなかった重要な判断

- なし
