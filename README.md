# 階層型エージェントハーネス設計書一式 V4

このパッケージは、Responses APIを推論基盤として使う階層型エージェントハーネスの統合設計書である。作業エージェント、Task、Workspace、非同期ツール、External Effect統治、完了レビュー、長期記憶Wikiを一つの整合したモデルにまとめている。

## 最初に読む文書

1. [エージェントハーネス設計書本体](docs/00-agent-harness.md)
2. [Agent・Task・Workspaceのドメインモデル](docs/01-domain-model.md)
3. [Taskライフサイクル](docs/02-task-lifecycle.md)
4. [Responses APIランタイム](docs/03-runtime-and-responses-api.md)

## サブ設計書

| 文書 | 内容 |
|---|---|
| [04-tools-and-async.md](docs/04-tools-and-async.md) | LLMツール、タイムアウト、`async_id`、Mailbox |
| [05-governance.md](docs/05-governance.md) | Sandbox境界、Policy Cascade、独立Policy Judge |
| [06-long-term-memory.md](docs/06-long-term-memory.md) | Task Episodeと独立Wiki Agentによる長期記憶 |
| [07-semantic-wiki-schema.md](docs/07-semantic-wiki-schema.md) | Concept / Schema / Script / Case PatternのMarkdown設計 |
| [08-running-example.md](docs/08-running-example.md) | Root Taskを最後まで処理する具体例 |
| [09-data-model.md](docs/09-data-model.md) | ER図、テーブル、制約、イベント |
| [10-implementation-and-tests.md](docs/10-implementation-and-tests.md) | 実装分割、検証項目、障害試験 |

## 付属物

- `schemas/work-agent-tools.json`: Responses API向け作業エージェント用Function Tool定義
- `schemas/judge-tools.json`: Policy Judge / Acceptance Reviewer用Function Tool定義
- `schemas/domain-types.ts`: 主要なTypeScript型
- `examples/semantic/`: 最小frontmatterを使ったSemantic Wikiの実例
- `examples/episodic/`: Task Episodeの実例
- `sources/OPENAI_API_NOTES.md`: Responses APIに関する公式仕様確認メモ

## 中核不変条件

1. **Taskには一人のOwnerがいる。**
2. **一つのOwnerは同時に一つの非終端Taskだけを処理する。** waiting中も占有は続く。
3. **並列性はOwnerを増やしてSubtaskを生成することで得る。**
4. **Ownerは完了候補を提出し、独立した軽量Acceptance Reviewerを通過してTaskが完了する。**
5. **親TaskのOwnerは直接の子Taskをキャンセルできるが、子Taskを直接完了させない。**
6. **Sandbox内では原則自由。Sandbox外への作用だけをEffect Gatewayで統治する。**
7. **作業階層と統治階層を分離する。親Agentは子のExternal Effectを承認しない。**
8. **Policy Judgeは作業Agentから独立し、Policy Cascadeを評価する。実行はGatewayだけが行う。**
9. **ツールは指定時間だけ同期的に待つ。超過時は処理を継続し、`async_id`を返し、結果をMailboxへ送る。**
10. **長期記憶のエピソード単位はTaskである。Work AgentはWikiを直接探索せず、HarnessがWiki Agentの回答を強制挿入する。**

## バージョン

- 文書版: V4
- 作成日: 2026-07-11
- API仕様確認日: 2026-07-11
