# 技術スタックと実装境界

## 1. 選定結果

初期製品はローカルで動作するCLI applicationとし、責務の性質に合わせて3つの実装単位へ分ける。

| 実装単位 | 対象Plane / コンポーネント | 技術 |
|---|---|---|
| Core Runtime | CLI、Control Plane、Work Agent Plane、Execution Plane | Go |
| Memory Service | Memory Plane | Python + OpenAI Agents SDK |
| Governance Service | Governance Plane | Rust |

利用者からは`kakesu`という1つのCLIに見せる。Core RuntimeがMemory ServiceとGovernance Serviceのプロセス lifecycleを管理するが、各Serviceの状態正本と判断責任は奪わない。

```text
kakesu (Go)
  ├─ CLI
  ├─ Control Plane
  ├─ Work Agent Plane
  ├─ Execution Plane
  ├─ control.db
  ├─ memory-service (Python)
  │    ├─ OpenAI Agents SDK
  │    ├─ evidence.db
  │    └─ semantic Wiki repository
  └─ governance-service (Rust)
       ├─ HTTPS / DNS enforcement
       ├─ CASB Rule Engine
       ├─ Credential Broker
       └─ governance.db
```

## 2. Core Runtime: Go

### 選定理由

- CLIを単一binaryとして配布しやすい。
- `context.Context`をCLI終了、Task cancellation、Tool タイムアウト、プロセス停止へ一貫して伝播できる。
- goroutineとチャネルでdispatcherやworkerを簡潔に構成できる。
- プロセス、signal、Unix domain socket、filesystem、Git、Linux ランタイム adapterを同じ言語で扱える。
- OpenAIのGo API helperを利用でき、Responses API adapterの追従負担を抑えられる。
- Rust/Tokioより所有権と`Send`境界の実装負担が小さく、Control Planeの状態機械へ集中できる。

### 適用範囲

- CLI commandとconfiguration
- Task Manager、Run Coordinator、Mailbox、Async Operation、Continuation
- Work AgentのResponses API tool loop
- Workspace、terminal、プロセス、スナップショット、cleanup
- Plane間Outbox/Inbox dispatcher
- Human Authority Gateway

### concurrency規約

goroutine、チャネル、プロセス memoryをTaskやメッセージの正本にしない。

- goroutineを無管理で起動せず、Core Runtimeのrun groupへ登録する。
- `context.Context`を全I/O境界へ渡す。
- チャネルは同一プロセス内のwake-up通知とbounded work handoffにだけ使う。
- durable メッセージ、lease、attempt、結果は`control.db`へ保存する。
- shutdownでは新規claimを止め、実行中処理をdrainし、未完了Jobを再claim可能な状態にする。

## 3. Memory Service: Python + OpenAI Agents SDK

### 選定理由

Memory PlaneはOS enforcementではなく、EvidenceをToolで調査してStructured Outputを生成するAgent コンポーネントである。OpenAI Agents SDKが提供するAgent、Runner、Function Tool、Structured Outputを再利用し、Episode AgentとWiki Agent固有の処理へ集中する。

### Framework利用境界

SDKは一時的なtool loopとして使用する。次をFrameworkの正本にしない。

- Task lifecycle
- Episode Compilation Jobのlease / 再試行
- 永続Sessionまたはconversation history
- Response ID、tool call履歴、途中checkpoint
- Human Authority routing

Episode Jobの入力スナップショット、ダイジェスト、budget、attemptはMemory Planeが`evidence.db`へ保存する。worker障害時は部分sessionを破棄し、固定入力から新しいSDK Runを開始する。

SDK標準tracingは既定で無効化し、Evidence本文やPrompt本文を外部traceへ複製しない。Job ID、入力/出力 ダイジェスト、Schema/Profile バージョン、usage、エラー codeだけをsanitized telemetryへ記録する。

### 他Frameworkを初期採用しない理由

- Microsoft Agent Frameworkはgraph ワークフローが必要になった場合の候補だが、初期Episode Agentは単一Agentの反復Tool Callで足りる。
- LangGraphのcheckpoint/threadは、固定入力から再調査する現在のJob recoveryと正本が重複する。
- Framework変更を可能にするため、Memory ServiceのIPC契約へSDK固有型を出さない。

## 4. Governance Service: Rust

### 選定理由

- TLS、HTTP、DNS、socket、bounded buffer、プロトコル parserを安全に実装できる。
- Linux firewall、ネットワーク namespace、netlink、Credential memoryを低levelで制御できる。
- 外側の接続を作成する前のwrite-ahead auditとfail-closed処理を明示的な型で表現できる。
- Secretを扱うコンポーネントをGC ランタイムやWork Agent プロセスから分離できる。

### 適用範囲

- HTTPS interception proxy
- DNS proxy
- Workspaceスコープのfirewall adapter
- compiled CASB Rule Engine
- Egress Attempt / Challenge / Grant / Transaction
- Credential Broker
- Egress retrospective reviewとPolicy revision orchestration
- `governance.db`とGovernance Outbox/Inbox

Rust内のTokio taskはネットワーク接続と有限Jobの実行資源であり、Policy、Grant、Incident、メッセージの正本ではない。long-lived taskはSupervisor registryへ登録し、detached spawnを禁止する。

## 5. Plane間通信

Plane間の意味的通信はバージョン付きJSON Schemaメッセージで行う。初期transportはUnix domain socket、encodingはJSONとする。

Wire Envelopeは[`message-envelope.schema.json`](../schemas/draft-v0/common/message-envelope.schema.json)をそのまま使用し、`from_component`、`to_component`、`correlation_id`、`causation_message_id`、`payload_schema`を保持する。Planeはコンポーネント registryから解決し、言語固有の追加フィールドをEnvelopeへ持ち込まない。

```text
sender local transaction
  → domain state更新
  → Outboxへmessageを追加
  → commit
  → UDSで送信
  → receiver Inboxへmessage_id uniqueでcommit
  → ACK
  → sender Outboxをdeliveredへ更新
```

### 原則

- socket、Go チャネル、BEAM mailbox、Tokio チャネルを配送の正本にしない。
- 配送はat-least-once、適用は`message_id`とidempotency keyでeffectively-onceにする。
- `ACK`は受信処理完了ではなく、受信側Inboxへのdurable コミットを表す。
- Domain処理結果は新しいOutbox メッセージとして返し、リクエスト socketを長時間占有しない。
- Queryは同期RPCを許可するが、結果にバージョン、watermark、ダイジェストを付ける。
- Human Authority Request/Decisionは必ずControl Plane Authority Gatewayを通す。

### Plane横断の原子性

異なるSQLite データベースをまたぐTransactionは作らない。原子性は各PlaneのAggregate内に限定し、Plane間ワークフローはidempotent メッセージ、`ACK`、reconciliationで収束させる。

安全性に関わる処理は、必要な`ACK`を受けるまで安全側状態を維持する。

- GovernanceへのGrant activation `ACK`前は`PolicyGrantReady`を発行しない。
- Task cancellation時はCoreがAction gateを閉じて新規作用を止め、GovernanceのGrant revoke `ACK`後にcleanupを完了する。
- Governance audit コミットに失敗した通信はforwardしない。
- Memory ingest失敗はTask終端を戻さず、Jobを再試行する。

Transport `ACK`やreconciliation メッセージはinfrastructure メッセージであり、TaskやPolicyの意味を変えない限り代表E2Eのdomain sequence projectionには表示しなくてよい。

## 6. 永続化

| Store | Owner | 主な内容 |
|---|---|---|
| `control.db` | Go Core Runtime | Task、Agent Run、Mailbox、Async、Continuation、Authority、Core Outbox/Inbox |
| `evidence.db` | Python Memory Service | Evidence、BLOB、FTS、Episode、Memory Job、Memory Outbox/Inbox |
| `governance.db` | Rust Governance Service | Policy、Binding、Attempt、Challenge、Grant、Transaction、Finding、Revision、Governance Outbox/Inbox |
| Semantic Wiki Git リポジトリ | Memory Service | Concept、Schema、Script、Case Pattern |

全SQLiteはWAL、foreign key、busy タイムアウトを有効にする。各データベースは所有Serviceだけがwriteし、ほかのPlaneはfileを直接開かない。Agent用Evidence クエリ 接続では`query_only`、Authorizer、クエリ budgetを強制する。

## 7. Schemaとcode generation

JSON SchemaをPlane間Wire Contractの正本とする。

- Go、Python、Rustの内部domain typeはWire typeから分離する。
- 受信時と永続スナップショット作成時にcanonical Schemaで検証する。
- Schema ID、revision、ダイジェストをメッセージとJob 入力へ保存する。
- canonical ペイロードを三言語のvalidator/decoderで読むcontract testを追加する。
- API adapterやFramework固有Schemaからcanonical Schemaを逆生成しない。

## 8. Build、test、lint

リポジトリ rootの`Makefile`を統一入口とする。

```sh
make build
make test
make lint
make check
```

`check`は三言語のbuild/test/lintに加えて、tabletop viewer生成、Schema/E2E validator、negative mutation、`git diff --check`を実行する。

## 9. 再評価条件

- Core Runtimeを複数nodeへ分散する場合はSQLite Outbox transportとGo プロセス modelを再評価する。
- Memory ワークフローが固定graph、HITL、途中checkpointを必要とした場合はMicrosoft Agent FrameworkまたはLangGraphを再評価する。
- Governance Serviceが複数hostを統治する場合はPolicy配布とaudit storeを分離する。
- SQLiteのsingle-writer待ちがSLOを満たさなくなった場合だけサーバー データベースを検討する。

## 10. 選定時の一次資料

- [Go Context](https://go.dev/blog/context)
- [OpenAI SDKs and CLI](https://developers.openai.com/api/docs/libraries)
- [OpenAI Agents SDK: Agents](https://openai.github.io/openai-agents-python/agents/)
- [OpenAI Agents SDK: Tools](https://openai.github.io/openai-agents-python/tools/)
- [OpenAI Agents SDK: Tracing](https://openai.github.io/openai-agents-python/tracing/)
- [Microsoft Agent Framework overview](https://learn.microsoft.com/en-us/agent-framework/overview/)
- [LangGraph persistence](https://docs.langchain.com/oss/python/langgraph/persistence)
