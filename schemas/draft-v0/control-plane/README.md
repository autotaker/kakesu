# Control Plane Schema カタログ — draft-v0

Task責任、契約、親子関係、メールボックス ルーティング、完了、および人間・外部責任者との唯一の通信境界である責任者ゲートウェイを所有する。ゲートウェイは認証、配送、期限、重複排除、判断永続化を担うが、各Planeが所有する判断の意味やスコープは変更しない。Agent実行やCASB ルール本文は所有しない。

## P0

以下は`draft-v0:r1`として追加済みである。

| Schema | 固定する内容 |
|---|---|
| `task-contract.schema.json` | 目的、受け入れ条件、指示、契約バージョン |
| `task-command.schema.json` | `delegate`、質問、`Escalate`、reply、キャンセル 子、完了案 |
| `task-event.schema.json` | ライフサイクル、契約 変更、完了、キャンセル ペイロード |
| `mailbox-event.schema.json` | 共通共通形式と非同期/子/質問/上位判断依頼/統治 ペイロード union |
| `mailbox-consumption.schema.json` | consumer、イベント シーケンス、ウォーターマーク、冪等消費結果 |
| `completion-review-input.schema.json` | レビュアーへ渡す固定案 スナップショット |
| `completion-review-output.schema.json` | 受理 / 拒否 / 不十分 証跡 |
| `task-authority-request.schema.json` | ルートTask 上位判断依頼を責任者へ提示するペイロード |
| `task-authority-decision.schema.json` | 契約 パッチ、terminate、回答者 来歴 |
| `task-containment-command.schema.json` | 封じ込め集合のTask別停止と保存済み状態への再開要求 |

## P1

| Schema | 固定する内容 |
|---|---|
| `task-progress.schema.json` | TODO ledger、ウォーターマーク、バージョン |
| `resume-context.schema.json` | 契約、進捗、メールボックス、過去実行 イベントの再開スナップショット |
| `suspension.schema.json` | 起点、エラー、再試行 ポリシー、next 再試行 |

## 現在のAPI アダプター

Control Plane由来のWork Agent ツール、受け入れ条件レビュー出力、Task 上位判断依頼 判断は`../api/`の合成バンドルに含まれる。実装時は本ディレクトリの正規 Schemaから生成する。
