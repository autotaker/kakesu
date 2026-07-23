---
task_id: "TASK-0015"
title: "完了候補と独立Acceptance Reviewを実装する"
status: ready
created_at: "2026-07-15"
---

# TASK-0015 完了候補と独立Acceptance Reviewを実装する

## 目的

ownerの`complete_candidate`を前検査し、不変入力を用いる独立Acceptance Reviewerの証跡付き判定でのみTaskをcompletedにする。

## 背景

Taskの完了はowner自己申告ではなく受け入れ条件への独立判定が必要である。TASK-0011のRun記録、TASK-0013の未解決操作/メールボックス、TASK-0014の子Task状態を前検査へ統合する。

## スコープ

### 対象

- candidate snapshot/digest、前検査、review job lease/retry、reviewer入力・read-only evidence tool、構造化decision永続化。
- accept/reject/insufficient_evidenceの原子的Task遷移とevidence validation。
### 対象外

- 人間の責任者判断、PRマージ、Episode/Wiki生成、レビュー以外のbuilt-in agent、reviewerの会話履歴永続化。

## 受け入れ条件

- [ ] complete_candidateはowner/Task状態/contract version/成果物・evidence ref/未解決required child・asyncを前検査し、失敗時にreview jobを作らない。
- [ ] reviewerはowner/AgentRunから独立した固定入力snapshot、profile/schema version、read-only evidence参照だけを用いる。
- [ ] 全decisionは検証済みevidence_refsを持ち、rejectはunmet_acceptance、insufficientはrequired_evidenceを保存する。
- [ ] acceptだけがreview/job/eventとTask completedを同一transactionで確定し、reject/insufficientはrunningへ戻して再候補を許す。

## 検討すべき設計観点

- reviewer API sessionは揮発、入力digestと構造化判定が監査正本。candidate後の変更を入力へ混ぜない。
- 前検査は機械的整合、ReviewerはAcceptance意味判定を所有する。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] 必要な実装、テスト、文書が完成している。
- [ ] 独立レビューと`make check`がPASSしている。
- [ ] マージ後QAが完了している。
- [ ] HANDOVERがWiki Agentに取り込みされている。

## 関連コンテキスト

### 意味 Wiki

- docs/01 §5、docs/02 §7、docs/04 §3、docs/06 §9、docs/11-data-model.md、docs/12 §3
### 判断

- reviewerへTask更新toolを渡さず、構造化出力をHarnessが検証・適用する。
### 適用しなかった重要な判断

- ownerの自己完了、live mutable context、reviewer session/historyの永続化を採用しない。
