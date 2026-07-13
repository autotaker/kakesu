# Memory Plane Schema カタログ — draft-v0

証跡、Taskエピソード、記憶コンテキスト、意味 Wikiとの入出力を所有する。

## P0

以下は`draft-v0:r1`として追加済みである。

| Schema | 固定する内容 |
|---|---|
| `evidence-record.schema.json` | 種別、Task/Workspace、ダイジェスト、サイズ、保持、秘匿化 |
| `evidence-query-tool.schema.json` | 読み取り専用 SQL、params、行/バイト上限、カーソル |
| `evidence-query-result.schema.json` | columns、rows、切り詰め済み、next カーソル |
| `episode-agent-input.schema.json` | 終端 Task スナップショット、証跡 views、予算、スキーマ 参照 |
| `task-episode.schema.json` | situation、course、結果、判断、surprise、unresolved、証跡 |
| `memory-context-request.schema.json` | Task フェーズ、契約、Workspace 要約、トークン 予算 |
| `memory-context.schema.json` | 意味 excerpts、エピソード excerpts、contested items、起点 参照 |

## P1

| Schema | 固定する内容 |
|---|---|
| `wiki-query-result.schema.json` | Taskへ注入する意味 記憶 結果 |
| `wiki-maintenance-input.schema.json` | エピソード群、既存Wiki、エラー report、ウォーターマーク |
| `wiki-change-proposal.schema.json` | 概念/Schema/スクリプト/ケース パターン変更と証跡 |
| `memory-error.schema.json` | Work Agentからの誤り報告と証跡 |

`task-episode.schema.json`と`evidence-query-tool.schema.json`は設計文書で既に利用を前提としているため、Memory Planeの最初の実装対象とする。
