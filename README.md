# 階層型エージェントハーネス設計書一式 V4

このパッケージは、Responses APIを推論基盤として使う階層型エージェントハーネスの統合設計書である。作業エージェント、Task、Workspace、非同期ツール、CASB型Egress統治、完了レビュー、長期記憶Wikiを一つの整合したモデルにまとめている。

## 最初に読む文書

1. [エージェントハーネス設計書本体](docs/00-agent-harness.md)
2. [Agent・Task・Workspaceのドメインモデル](docs/01-domain-model.md)
3. [Taskライフサイクル](docs/02-task-lifecycle.md)
4. [Agentライフサイクル](docs/03-agent-lifecycle.md)
5. [組み込みAgent](docs/04-built-in-agents.md)
6. [Responses APIランタイム](docs/05-runtime-and-responses-api.md)

## サブ設計書

| 文書 | 内容 |
|---|---|
| [04-built-in-agents.md](docs/04-built-in-agents.md) | L1/L2/L3とHarnessのAgent管理から独立した組み込みAgent群 |
| [06-tools-and-async.md](docs/06-tools-and-async.md) | LLMツール、タイムアウト、`async_id`、Mailbox |
| [07-governance.md](docs/07-governance.md) | Workspace Security Policy、rule-based CASB、Policy更新、事後検知 |
| [08-long-term-memory.md](docs/08-long-term-memory.md) | Task Episodeと独立Wiki Agentによる長期記憶 |
| [09-semantic-wiki-schema.md](docs/09-semantic-wiki-schema.md) | Concept / Schema / Script / Case PatternのMarkdown設計 |
| [10-running-example.md](docs/10-running-example.md) | Root Taskを最後まで処理する具体例 |
| [11-data-model.md](docs/11-data-model.md) | ER図、テーブル、制約、イベント |
| [12-implementation-and-tests.md](docs/12-implementation-and-tests.md) | 実装分割、検証項目、障害試験 |

## 付属物

- `schemas/README.md`: Plane別Schema所有境界と`draft-v0` version規約
- `schemas/draft-v0/`: Control / Execution / Governance / Memory Plane別のSchema catalog
- `schemas/draft-v0/api/work-agent-tools.json`: Responses API向け作業エージェント用Function Tool bundle
- `schemas/draft-v0/api/built-in-agent-outputs.json`: Acceptance Reviewer / Policy・Egress Audit Agent用Structured Output bundle
- `schemas/draft-v0/api/authority-tools.json`: 外部Authority adapter用Egress Grant／Task Escalation回答Tool bundle
- `schemas/domain-types.ts`: 設計確認用の主要TypeScript論理型。runtime validatorの正本ではない
- `examples/semantic/`: 最小frontmatterを使ったSemantic Wikiの実例
- `examples/episodic/`: Task Episode Schemaの説明用実例。runtime保存形式ではない
- `sources/OPENAI_API_NOTES.md`: Responses APIに関する公式仕様確認メモ

## 中核不変条件

1. **Taskには一人のOwnerがいる。**
2. **一つのOwnerは同時に一つの非終端Taskだけを処理する。** waiting中も占有は続く。
3. **並列性はOwnerを増やしてSubtaskを生成することで得る。**
4. **Ownerは完了候補を提出し、独立した軽量Acceptance Reviewerを通過してTaskが完了する。**
5. **親TaskのOwnerは直接の子Taskをキャンセルできるが、子Taskを直接完了させない。**
6. **Sandbox内では原則自由。外向き通信はCASB Egress Control Planeでinline統治する。**
7. **作業階層と統治階層を分離する。親Agentは子TaskのEgress Grantを承認しない。**
8. **外向き通信はWorkspaceに束縛したCASB Ruleでinline制御し、Policy AgentがRuleを更新し、Egress Audit Agentが通過後の通信をレビューする。**
9. **ツールは指定時間だけ同期的に待つ。超過時は処理を継続し、`async_id`を返し、結果をMailboxへ送る。**
10. **長期記憶のエピソード単位はTaskである。Work AgentはWikiを直接探索せず、HarnessがWiki Agentの回答を強制挿入する。**

## バージョン

- 文書版: V4
- 作成日: 2026-07-11
- API仕様確認日: 2026-07-11
