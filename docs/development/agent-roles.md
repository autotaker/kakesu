# Agent責務

## Main

Task 雛形、`backlog`、`planning-gate`、`completion-gate`、共通ロック、スコープ検査、mainへのコミット/mergeを所有する。DEV開始前にPLAN/QA_PLANの意図・スコープ・受け入れ経路を確認し、完了前にREVIEW/QA 識別情報とPASS、QA_PLAN承認、HANDOVER 案を確認する。

## Planner

TASKのplanning 入力 packetと参照資料からPLANを作る。製品コード、Schema、滞留、Git履歴は変更しない。

## DEV

承認済みPLANの範囲でTask ワークツリーだけを編集する。起動処理の`candidate-commit`が一度だけ`make check`と製品コミットを行う。手動コミット、main証跡編集、Wiki編集はしない。

## レビュアー

DEVと別Agentとして案 diffとDEV check証跡を独立監査し、REVIEW_RESULTへ識別情報とPASS/FAILを記録する。

## QA

承認済みQA_PLANに従い、同じ案から独立にケースを確認し、QA_RESULTへ識別情報、判断、コマンド/結果を記録する。

## Wiki Agent

Wiki依頼が明示された場合だけ、Mainが標準の`agents.spawn_agent(task_name=..., agent_type="wiki", fork_turns="none", ...)`でTerra/medium、workspace-writeのWiki担当を一つずつ直列に起動する。許可パスは起動前にMainが固定し、Wiki Agentはその範囲だけを編集する。Wiki Agentは別Agent起動、共通ロック、スコープ判定、検証、stage、コミット、merge、`.git`書込みを行わない。

子の終了後、Mainが差分スコープを確認し、索引変更時だけ`make wiki-index`、続いて`make work-check`を実行する。公開は既存の共通ロック付きMain publish トランザクションとコミットだけで行う。標準完了はWiki receiptを要求せず、Wiki依頼のないTaskはAgentもreceiptもなしで完了できる。receiptを作る場合は既存Schemaと検証規則に従う任意成果物とする。
