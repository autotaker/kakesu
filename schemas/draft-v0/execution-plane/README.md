# Execution Plane Schema カタログ — draft-v0

Agent実行、関数 呼び出し、ツール 結果、非同期操作、継続情報、リソース ランタイムを所有する。Task契約やセキュリティ ポリシーの意味判断は所有しない。

## P0

以下は`draft-v0:r1`として追加済みである。

| Schema | 固定する内容 |
|---|---|
| `tool-call.schema.json` | 呼び出し ID、ツール name、validated 引数、スキーマ 参照 |
| `tool-result.schema.json` | `completed` / accepted / `failed`共通共通形式 |
| `tool-result-values.schema.json` | 子Task、質問、上位判断依頼、許可、待機、キャンセル等の結果union |
| `async-operation.schema.json` | 操作 キー、状態、結果/エラー 参照、期限 |
| `agent-run-event.schema.json` | `completed` 出力 項目、ツール 呼び出し/出力、圧縮、メンテナンス イベント |
| `continuation.schema.json` | 論理 カーソル、待機 Condition、再開 ウォーターマーク |
| `workspace-created.schema.json` | 分岐/empty、親Workspace、ポリシー割り当て、許可非継承 |
| `resume-context.schema.json` | previous/新規 実行、継続情報、メールボックス、ウォーターマーク |

## P1

| Schema | 固定する内容 |
|---|---|
| `agent-run-snapshot.schema.json` | 実行再開に必要な実行状態 |
| `agent-resource.schema.json` | プロセス/サーバー/ワークツリー、lifetime、クリーンアップ 状態 |
| `workspace-runtime.schema.json` | ランタイム アダプター、ネットワーク 識別情報、マウント/リソース 参照 |

## 現在のAPI アダプター

Responses APIへ渡すWork Agent ツール バンドルは`../api/work-agent-tools.json`にある。関数 呼び出し 出力は構造化出力対象外でも、本ディレクトリの`tool-result*.schema.json`でハーネス生成時と再配送時に検証する。
