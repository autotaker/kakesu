# Task管理

Taskは`backlog.yaml`と`tasks/TASK-NNNN-.../`の証跡で管理する。Task 雛形にはTASK、PLAN、QA_PLAN、REVIEW_RESULT、QA_RESULT、HANDOVERを置く。

## 所有と編集

Main Agentは`backlog`、`planning-gate`、`completion-gate`を所有する。PlannerはPLAN、QAはQA_PLAN/QA_RESULT、レビュアーはREVIEW_RESULT、DEVはTask ワークツリーの製品コードを編集する。子Agentはstage/コミットせず、起動処理の親が共通ロック、スコープ、検証、コミットを所有する。

## 証跡の最小契約

- HANDOVER: `task_id`、`status`、`completed_at`、`candidate_commit`。candidate-bound DEV証跡はコマンド/結果の表で記録する。
- REVIEW_RESULT: レビュアー 識別情報、判断、reviewed_at、案 diffとDEV checkを監査した記録。
- QA_RESULT: QA 識別情報、判断、tested_at、ケースIDごとのコマンド/結果。
- QA_PLAN: DEV開始前の承認、QA 識別情報、各ケースの実施モードと理由。

案 tree/ダイジェスト、tested コミット、reviewed コミット、CFチェックリスト、Wiki receiptは標準完了の必須転記項目ではない。estimate_pointsはバックログの規模感を示す参考値で、PLANの算術ゲートではない。

## 安全契約変更

製品成果物を変更しない安全契約Taskは、TASK-firstの独立QA_PLAN、Mainによる意図・スコープ・受け入れ経路確認、契約検査を行う。安全契約専用の既存証跡があるTaskでは、そのTaskの検査に必要なfieldsをPLAN承認時に明示する。製品Taskのtemplateへ安全契約専用fieldsを戻さない。

既存の振り返りを10 Taskごとに実施し、ルールの検出価値、誤検知、時間、保守費を確認する。低価値のルールは削除または警告化する。
