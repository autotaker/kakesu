# 技術スタックと実装境界

## 1. 選定結果

初期製品はローカルで動作するCLI アプリケーションとし、責務の性質に合わせて3つの実装単位へ分ける。

| 実装単位 | 対象Plane / コンポーネント | 技術 |
|---|---|---|
| コアランタイム | CLI、Control Plane、Work Agent Plane、Execution Plane | Go |
| 記憶サービス | Memory Plane | Python + OpenAI Agent SDK |
| 統治サービス | Governance Plane | Rust |

利用者からは`kakesu`という1つのCLIに見せる。コアランタイムが記憶サービスと統治サービスのプロセス ライフサイクルを管理するが、各サービスの状態正本と判断責任は奪わない。

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

## 2. コアランタイム: Go

### 選定理由

- CLIを単一binaryとして配布しやすい。
- `context.Context`をCLI終了、Task キャンセル、ツール タイムアウト、プロセス停止へ一貫して伝播できる。
- goroutineとチャネルでdispatcherやワーカーを簡潔に構成できる。
- プロセス、signal、Unixドメインソケット、ファイルシステム、Git、Linux ランタイム アダプターを同じ言語で扱える。
- OpenAIのGo API helperを利用でき、Responses API アダプターの追従負担を抑えられる。
- Rust/Tokioより所有権と`Send`境界の実装負担が小さく、Control Planeの状態機械へ集中できる。

### 適用範囲

- CLI コマンドとconfiguration
- Task マネージャー、実行コーディネーター、メールボックス、非同期操作、継続情報
- Work AgentのResponses API ツール ループ
- Workspace、終端、プロセス、スナップショット、クリーンアップ
- Plane間送信キュー/受信キュー dispatcher
- 人間の責任者ゲートウェイ

### concurrency規約

goroutine、チャネル、プロセスメモリをTaskやメッセージの正本にしない。

- goroutineを無管理で起動せず、コアランタイムの実行 グループへ登録する。
- `context.Context`を全I/O境界へ渡す。
- チャネルは同一プロセス内のwake-up通知と上限付き 作業 handoffにだけ使う。
- 永続 メッセージ、リース、試行、結果は`control.db`へ保存する。
- shutdownでは新規確保を止め、実行中処理をdrainし、未完了ジョブを再確保可能な状態にする。

## 3. 記憶サービス: Python + OpenAI Agent SDK

### 選定理由

Memory PlaneはOS 強制ではなく、証跡をツールで調査して構造化 出力を生成するAgent コンポーネントである。OpenAI Agent SDKが提供するAgent、ランナー、関数 ツール、構造化 出力を再利用し、エピソード AgentとWiki Agent固有の処理へ集中する。

### フレームワーク利用境界

SDKは一時的なツール ループとして使用する。次をフレームワークの正本にしない。

- Taskライフサイクル
- エピソード 編纂 ジョブのリース / 再試行
- 永続セッションまたはconversation history
- レスポンス ID、ツール 呼び出し履歴、途中チェックポイント
- 人間の責任者 ルーティング

エピソード ジョブの入力スナップショット、ダイジェスト、予算、試行はMemory Planeが`evidence.db`へ保存する。ワーカー障害時は部分セッションを破棄し、固定入力から新しいSDK 実行を開始する。

SDK標準tracingは既定で無効化し、証跡本文やPrompt本文を外部トレースへ複製しない。ジョブ ID、入力/出力 ダイジェスト、Schema/プロファイル バージョン、usage、エラー コードだけを無害化済み telemetryへ記録する。

### 他フレームワークを初期採用しない理由

- Microsoft Agent フレームワークはグラフ ワークフローが必要になった場合の候補だが、初期エピソード Agentは単一Agentの反復ツール 呼び出しで足りる。
- LangGraphのチェックポイント/threadは、固定入力から再調査する現在のジョブ 復旧と正本が重複する。
- フレームワーク変更を可能にするため、記憶サービスのIPC契約へSDK固有型を出さない。

## 4. 統治サービス: Rust

### 選定理由

- TLS、HTTP、DNS、ソケット、上限付き buffer、プロトコル parserを安全に実装できる。
- Linux ファイアウォール、ネットワーク名前空間、netlink、認証情報を保持するメモリを低レベルで制御できる。
- 外側の接続を作成する前の先行書き込み 監査とfail-closed処理を明示的な型で表現できる。
- 秘密情報を扱うコンポーネントをGC ランタイムやWork Agent プロセスから分離できる。

### 適用範囲

- HTTPS 傍受 プロキシ
- DNS プロキシ
- Workspaceスコープのファイアウォール アダプター
- compiled CASB ルールエンジン
- 外向き通信 試行 / 許可確認 / 許可 / トランザクション
- 認証情報 ブローカー
- 外向き通信 retrospective レビューとポリシー 改訂 orchestration
- `governance.db`と統治 送信キュー/受信キュー

Rust内のTokio Taskはネットワーク接続と有限ジョブの実行資源であり、ポリシー、許可、インシデント、メッセージの正本ではない。long-lived TaskはSupervisor レジストリへ登録し、detached 生成を禁止する。

## 5. Plane間通信

Plane間の意味的通信はバージョン付きJSON Schemaメッセージで行う。初期転送方式はUnixドメインソケット、encodingはJSONとする。

通信共通形式は[`message-envelope.schema.json`](../schemas/draft-v0/common/message-envelope.schema.json)をそのまま使用し、`from_component`、`to_component`、`correlation_id`、`causation_message_id`、`payload_schema`を保持する。Planeはコンポーネント レジストリから解決し、言語固有の追加フィールドを共通形式へ持ち込まない。

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

- ソケット、Go チャネル、BEAM メールボックス、Tokio チャネルを配送の正本にしない。
- 配送は少なくとも1回、適用は`message_id`と冪等キーでeffectively-onceにする。
- `ACK`は受信処理完了ではなく、受信側受信キューへの永続 コミットを表す。
- ドメイン処理結果は新しい送信キュー メッセージとして返し、リクエスト ソケットを長時間占有しない。
- クエリは同期RPCを許可するが、結果にバージョン、ウォーターマーク、ダイジェストを付ける。
- 人間の責任者への依頼/判断は必ずControl Planeの責任者ゲートウェイを通す。

### Plane横断の原子性

異なるSQLite データベースをまたぐトランザクションは作らない。原子性は各Planeの集約内に限定し、Plane間ワークフローは冪等 メッセージ、`ACK`、照合で収束させる。

安全性に関わる処理は、必要な`ACK`を受けるまで安全側状態を維持する。

- 統治への許可 有効化 `ACK`前は`PolicyGrantReady`を発行しない。
- Task キャンセル時はコアが操作ゲートを閉じて新規作用を止め、統治の許可 失効 `ACK`後にクリーンアップを完了する。
- 統治 監査 コミットに失敗した通信は転送しない。
- 記憶 取り込み失敗はTask終端を戻さず、ジョブを再試行する。

Transport `ACK`や照合 メッセージはinfrastructure メッセージであり、Taskやポリシーの意味を変えない限り代表E2Eのドメイン シーケンス投影には表示しなくてよい。

## 6. 永続化

| ストア | オーナー | 主な内容 |
|---|---|---|
| `control.db` | Go コアランタイム | Task、Agent実行、メールボックス、非同期、継続情報、責任者、コア 送信キュー/受信キュー |
| `evidence.db` | Python 記憶サービス | 証跡、BLOB、FTS、エピソード、記憶 ジョブ、記憶 送信キュー/受信キュー |
| `governance.db` | Rust 統治サービス | ポリシー、割り当て、試行、許可確認、許可、トランザクション、検出事項、改訂、統治 送信キュー/受信キュー |
| 意味 Wiki Git リポジトリ | 記憶サービス | 概念、Schema、スクリプト、ケース パターン |

全SQLiteはWAL、foreign キー、busy タイムアウトを有効にする。各データベースは所有サービスだけが書き込みし、ほかのPlaneはファイルを直接開かない。Agent用証跡 クエリ 接続では`query_only`、Authorizer、クエリ 予算を強制する。

## 7. Schemaとコード generation

JSON SchemaをPlane間Wire 契約の正本とする。

- Go、Python、Rustの内部ドメイン 型はWire 型から分離する。
- 受信時と永続スナップショット作成時に正規 Schemaで検証する。
- Schema ID、改訂、ダイジェストをメッセージとジョブ 入力へ保存する。
- 正規 ペイロードを三言語の検証器/decoderで読む契約 テストを追加する。
- API アダプターやフレームワーク固有Schemaから正規 Schemaを逆生成しない。

## 8. Build、テスト、リント

リポジトリ ルートの`Makefile`を統一入口とする。

```sh
make build
make test
make lint
make check
```

`check`は三言語のbuild/テスト/リントに加えて、tabletop viewer生成、Schema/E2E 検証器、ネガティブ 変異、`git diff --check`を実行する。

## 9. 再評価条件

- コアランタイムを複数nodeへ分散する場合はSQLite 送信キュー 転送方式とGo プロセス モデルを再評価する。
- 記憶 ワークフローが固定グラフ、HITL、途中チェックポイントを必要とした場合はMicrosoft Agent フレームワークまたはLangGraphを再評価する。
- 統治サービスが複数ホストを統治する場合はポリシー配布と監査 ストアを分離する。
- SQLiteのsingle-writer待ちがSLOを満たさなくなった場合だけサーバー データベースを検討する。

## 10. 選定時の一次資料

- [Go コンテキスト](https://go.dev/blog/context)
- [OpenAI SDKs and CLI](https://developers.openai.com/api/docs/libraries)
- [OpenAI Agent SDK: Agent](https://openai.github.io/openai-agents-python/agents/)
- [OpenAI Agent SDK: Tools](https://openai.github.io/openai-agents-python/tools/)
- [OpenAI Agent SDK: Tracing](https://openai.github.io/openai-agents-python/tracing/)
- [Microsoft Agent フレームワーク overview](https://learn.microsoft.com/en-us/agent-framework/overview/)
- [LangGraph persistence](https://docs.langchain.com/oss/python/langgraph/persistence)
