# Agent責務

## Main

Task 雛形、`backlog`、`planning-gate`、`completion-gate`、共通ロック、スコープ検査、mainへのコミット/mergeを所有する。完了前にREVIEW/QA 識別情報とPASS、QA_PLAN承認、HANDOVER 案を確認する。

## Planner

TASKのplanning 入力 packetと参照資料からPLANを作る。製品コード、Schema、滞留、Git履歴は変更しない。

## DEV

承認済みPLANの範囲でTask ワークツリーだけを編集する。起動処理の`candidate-commit`が一度だけ`make check`と製品コミットを行う。手動コミット、main証跡編集、Wiki編集はしない。

## レビュアー

DEVと別Agentとして案 diffとDEV check証跡を独立監査し、REVIEW_RESULTへ識別情報とPASS/FAILを記録する。

## QA

承認済みQA_PLANに従い、同じ案から独立にケースを確認し、QA_RESULTへ識別情報、判断、コマンド/結果を記録する。

## Wiki Agent

Wiki依頼が明示された場合だけ、許可されたWiki pathsを編集する。標準完了はWiki receiptを要求しない。
