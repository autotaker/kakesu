# Memory Plane Schema catalog — draft-v0

Evidence、Task Episode、Memory Context、Semantic Wikiとの入出力を所有する。

## P0

以下は`draft-v0:r1`として追加済みである。

| Schema | 固定する内容 |
|---|---|
| `evidence-record.schema.json` | kind、task/workspace、digest、size、retention、redaction |
| `evidence-query-tool.schema.json` | read-only SQL、params、row/byte上限、cursor |
| `evidence-query-result.schema.json` | columns、rows、truncated、next cursor |
| `episode-agent-input.schema.json` | terminal Task snapshot、Evidence views、budget、schema ref |
| `task-episode.schema.json` | situation、course、outcome、decision、surprise、unresolved、Evidence |
| `memory-context-request.schema.json` | Task phase、Contract、Workspace summary、token budget |
| `memory-context.schema.json` | semantic excerpts、Episode excerpts、contested items、source refs |

## P1

| Schema | 固定する内容 |
|---|---|
| `wiki-query-result.schema.json` | Taskへ注入するSemantic Memory result |
| `wiki-maintenance-input.schema.json` | Episode群、既存Wiki、error report、watermark |
| `wiki-change-proposal.schema.json` | Concept/Schema/Script/Case Pattern変更とEvidence |
| `memory-error.schema.json` | Work Agentからの誤り報告とEvidence |

`task-episode.schema.json`と`evidence-query-tool.schema.json`は設計文書で既に利用を前提としているため、Memory Planeの最初の実装対象とする。
