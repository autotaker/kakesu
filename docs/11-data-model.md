# 永続化・データモデル設計

## 1. 集約境界

```text
Task Aggregate
  task
  task_contract_versions
  task_progress
  task_progress_events
  task_events
  task_outcome
  ask_requests
  ask_advices
  escalation_requests
  escalation_decisions
  completion_candidates
  completion_review_jobs
  completion_reviews
  authority_requests
  authority_decisions
  control_inbox
  control_outbox

Execution Aggregate
  agent_runs
  agent_run_steps
  agent_run_items
  agent_resources
  resume_cursors
  continuations
  async_operations
  mailbox_entries

Workspace Aggregate
  workspaces
  workspace_snapshots
  artifacts

Governance Aggregate
  workspace_security_policy_bindings
  global_security_policy_bindings
  egress_attempts
  egress_capture_manifests
  egress_rule_decisions
  egress_challenges
  challenge_observations
  grant_requests
  grant_evaluation_jobs
  grant_decisions
  policy_grants
  outbound_transactions
  dns_resolutions
  policy_documents
  egress_review_jobs
  egress_findings
  policy_revision_jobs
  policy_revision_job_findings
  policy_revision_proposals
  policy_revision_authority_requests
  policy_revision_authority_decisions
  policy_revisions
  governance_inbox
  governance_outbox

Memory Aggregate
  episode_compilation_jobs
  task_episodes
  evidence_records
  evidence_blobs
  evidence_links
  memory_context_requests
  wiki_commits
  memory_inbox
  memory_outbox
```

Task状態とTaskイベントは同一トランザクションで更新する。他集約とは送信キュー/イベントで連携する。

### 1.1 データベース所有境界

初期実装は3つのSQLiteへ分ける。

| データベース | 書き込み オーナー | 集約 |
|---|---|---|
| `control.db` | Go コアランタイム | Task、実行、Workspace、責任者、コア 受信キュー/送信キュー |
| `evidence.db` | Python 記憶サービス | 証跡、エピソード、記憶コンテキスト、Wiki ジョブ、記憶 受信キュー/送信キュー |
| `governance.db` | Rust 統治サービス | Workspaceポリシー割り当て、外向き通信、許可、検出事項、改訂、統治 受信キュー/送信キュー |

各サービスは所有外DBを直接開かない。ER図のPlane横断関係は論理結合であり、SQLiteのcross-database FKではない。送信元は参照ID、バージョン、ダイジェストを送信キュー ペイロードへ固定し、受信側は受信キュー コミット後にローカル 記録または固定スナップショットとして検証する。

異なるDBをまたぐ原子トランザクションは作らない。各Plane内ではドメイン 状態と送信キューを同一トランザクションで確定し、Plane間は少なくとも1回配送、永続 `ACK`、冪等 apply、照合で収束させる。詳細は[13-technology-stack.md](13-technology-stack.md)を正本とする。

本書の`uuid`、`timestamptz`、`boolean`は論理型である。SQLite実装ではUUIDとタイムスタンプを正規 テキスト、booleanをCHECK付きintegerとして保存し、アプリケーション 型とSchema 検証器で形式を強制する。

## 2. 主要テーブル

### `agents`

| 列 | 型 | 説明 |
|---|---|---|
| `agent_id` | UUID PK | 論理Agent |
| `profile_id` | テキスト | L1/L2/L3等のプロファイル |
| `status` | テキスト | idle / assigned / retired |
| `current_task_id` | UUID NULL許容 | 現在Task |
| `created_at` | `timestamptz` | 生成時刻 |

### `tasks`

| 列 | 型 | 説明 |
|---|---|---|
| `task_id` | UUID PK | Task ID |
| `parent_task_id` | UUID FK NULL許容 | 直接親 |
| `owner_agent_id` | UUID FK | オーナー |
| `workspace_id` | UUID FK 一意 | 論理Workspace |
| `objective` | テキスト | 現行目的 |
| `acceptance` | テキスト | 現行受け入れ条件 |
| `instructions` | テキスト NULL許容 | 補助指示 |
| `contract_version` | int | 楽観ロック対象 |
| `status` | テキスト | ライフサイクル 状態 |
| `dependency` | テキスト | 必須 / optional |
| `version` | bigint | 状態更新用 |
| `created_at` | `timestamptz` | |
| `started_at` | `timestamptz` NULL許容 | |
| `ended_at` | `timestamptz` NULL許容 | |

### `task_contract_versions`

契約変更の履歴を保存する。過去の完了案がどの契約を基準にしたか追跡できる。

### `task_progress` / `task_progress_events`

現在のTODO形式進捗とappend-onlyな更新履歴を保存する。`task_progress`は`task_id`、`version`、`current_focus_id`、TaskイベントとAgent実行 イベントのウォーターマーク、`updated_at`を持ち、項目は子テーブルまたはJSONとして保持できる。更新はオーナーAgentの認識であり、受け入れ条件達成の正本ではない。

### `task_events`

append-only。`event_id`、`task_id`、`sequence_no`、`event_type`、`payload_ref`、`actor_ref`を持つ。

### `agent_runs`

一Taskの実行セッション。`previous_response_id`は補助列で、復元の必須条件にしない。`normal_step_count`と`last_progress_refresh_step`を持ち、メンテナンス レスポンスを除いたステップ周期で進捗 更新を起動する。

### `agent_run_steps` / `agent_run_items`

Responses API呼び出し単位のステップ メタデータと、完成した出力 項目の正規化記録を保存する。同じオーナー 割り当て内で単調増加する`assignment_event_sequence`を持ち、実行をまたぐ再開コンテキストの選択に使う。リクエスト本文やストリーミング 差分を無条件には保存せず、コンテキスト バージョン／参照／ダイジェストと完成項目を基本とする。保存対象、保持、推論、圧縮 項目、秘匿化の正本は[05-runtime-and-responses-api.md](05-runtime-and-responses-api.md)の「Agent実行記録 ポリシー」とする。

### `agent_resources`

| 列 | 説明 |
|---|---|
| `resource_id` | Agentリソース ID |
| `agent_id` | リソースを所有する論理Agent |
| `assignment_id` | 割り当て スコープのオーナー割当ID、NULL許容 |
| `run_id` | 実行 スコープの場合のAgent実行、NULL許容 |
| `kind` | プロセス / サーバー / ワークツリー / temporary_directory |
| `resource_ref` | プロセス マネージャーやWorkspace マネージャー上の参照 |
| `lifetime` | 実行 / 割り当て / Agent |
| `cleanup_policy` | stop / delete / retain |
| `status` | `active` / cleanup_pending / cleaning / released / needs_operator |
| `retry_count` | クリーンアップ試行回数 |
| `last_error_ref` | 最終クリーンアップエラー、NULL許容 |

Agentまたはツール実行基盤がリソースを登録し、ハーネスがクリーンアップ開始、再試行、運用者移管を管理する。Task終端後もクリーンアップ状態は独立して進み、Task状態を変更しない。

### `resume_cursors`

実行切替境界ごとの最小再開位置を保存する。`cursor_id`、`task_id`、`agent_id`、`source_run_id`、`contract_version`、`task_version`、`progress_version`、Workspace参照、Taskイベント・Agent実行 イベント・メールボックスのウォーターマーク、`created_at`を持つ。

カーソルは意味的要約を持たない。Task、契約、進捗、メールボックス、非同期操作、成果物、Workspaceの正本を置き換えない。新実行が参照したカーソル IDを`agent_runs`へ記録し、再開元を監査可能にする。

### `continuations`

待機理由、awaited イベント、契約バージョン、Workspace スナップショット、コンテキスト スナップショットを保持する。

### `async_operations`

| 列 | 説明 |
|---|---|
| `async_id` | ハーネス 操作 ID |
| `owner_task_id` | 結果を受け取るTask |
| `tool_call_id` | 元Responses 関数 呼び出し |
| `tool_name` | `delegate`等 |
| `status` | `running` / `completed` / `failed` / `cancelled` |
| `sync_deadline` | 直接待機の期限 |
| `result_ref` | 最終結果 |
| `operation_key` | `task_id + call_id + tool_name`からハーネスが生成する重複防止キー |

### `mailbox_entries`

Taskごとのイベントキュー。少なくとも1回 配送を前提とし、`event_id` 一意、`consumed_at` NULL許容、`sequence_no`を持つ。

### `ask_requests` / `ask_advices`

Agent間助言の正本を保存する。リクエストには子/親 Task、双方のオーナーAgent、契約バージョン、質問、状態、`async_id`を持たせ、adviceには回答者と解決時刻を持たせる。Task契約を変更するフィールドは持たない。

### `escalation_requests` / `escalation_decisions`

Taskから責任者への判断移転の正本を保存する。リクエストには要求元Task、親責任者 Taskまたはルート責任者 参照の排他的な一方、契約バージョン、質問、options、状態、`async_id`を持たせる。判断には責任者、判断、任意の契約 パッチ、terminate、解決時刻を持たせる。質問 集約と同一テーブルへ潰さない。

### `completion_candidates` / `completion_review_jobs` / `completion_reviews`

`completion_candidates`はオーナーが提出した結果、成果物、証跡、契約バージョンの不変スナップショットとダイジェストを保存する。`completion_review_jobs`は候補/入力スナップショット参照とダイジェスト、状態、試行、リース、呼び出し 期限、lastエラー、レビュアー プロファイルバージョン、出力スキーマバージョンを持つ。プロセス再起動後も同じ入力を冪等な再試行に使える。期限切れ`reviewing` ジョブは部分レスポンスを破棄して新しい一時セッションで再確保する。`completion_reviews`は確定した構造化 出力と入力ダイジェスト、使用した証跡参照を保存する。

レビュアーの一時API セッション、レスポンス ID、ツール 呼び出し履歴は保存しない。独立性はオーナーと分離した入力スナップショット、プロファイル バージョン、ツール権限、入力ダイジェストで監査する。

レビュー確定時は完了レビュー 挿入、レビュー ジョブ `completed`、Task遷移、`CompletionReviewed` イベントをTask 集約の同一トランザクションでコミットする。

### `workspaces`

Taskと1:1。起点 Workspace、モード、storage 参照、状態を持つ。

### `workspace_security_policy_bindings`

Workspaceと1:1で、セキュリティ プロファイル 参照、`active` 基本ポリシーバージョン、`pending` 改訂/バージョン予約、状態、バージョンを持つ。CASB ルールエンジン、認証情報 ブローカー、ファイアウォールはAgent/Taskではなくこの割り当てを適用する。オーナー交代やAgent実行再開では更新しない。Workspace 分岐時はプラットフォーム ポリシーが許すプロファイルだけを新割り当てへコピーし、一時許可は継承しない。

全体 ベースラインを改定する場合は`global_security_policy_bindings`を対象 行とし、`global:<profile_ref>` キー、`active` バージョン、`pending` 改訂/バージョン予約を持たせる。Workspace 割り当てと同じCAS/`ACK` ライフサイクルを適用する。

### `artifacts`

不変 内容 ダイジェストと論理 参照を持つ。成果物本文はファイルシステムへ保存せず、証跡DBの`evidence_blobs`を参照する。Task 結果、許可申請、エピソードから参照される。

### `egress_attempts` / `outbound_transactions`

外向き通信 強制 点が実通信を観測した時点で`egress_attempts`へWorkspace ネットワーク 識別情報、Task/Agent 来歴、プロトコル、宛先、無害化済み リクエスト メタデータ、本文 ダイジェスト/サイズ/分類、ルール結果を保存する。同じトランザクションで試行を一意参照するキャプチャ マニフェストも必ず作る。マニフェストはリクエスト/レスポンス別の合計 バイト、captured/秘匿済み rangesとチャンク ダイジェスト、切り詰め済み flag/理由、分類、暗号化済み BLOB/キー 参照、完了 状態、保持/ピン留めを持つ。キャプチャ不能・部分キャプチャでもマニフェストを必須とし、欠落範囲と理由を表す。キャプチャは原則全通信を短期保持し、高リスク/検出事項関連を長期保持へ昇格する。許可して転送した通信は`outbound_transactions`へリクエスト/レスポンス ダイジェスト、適用許可、開始・完了状態を保存する。

### `egress_rule_decisions`

試行ごとに許可/拒否、ポリシーバージョン、一致 ルール 参照、理由 コード、評価時刻を保存する。同じ正規 割り当てとポリシーバージョンからルールエンジンの判断を再現でき、後続の外向き通信 レビューが「どのルールで通過したか」を確認する監査正本とする。

### `egress_challenges` / `challenge_observations`

拒否した通信に対して不変な許可確認 コアを作る。Workspace、要求元Task/origin/委譲/契約、正規リクエスト割り当て、無害化済み 参照、理由、許可 適格性を持つ。プラットフォーム ポリシーが決めた`auto_grant_eligible`と`required_authority_ref`、期限も保存する。個々の`egress_attempt`は`challenge_observations` 結合で許可確認へ関連付け、再試行 回数/最終観測時刻はObservation集計から求める。同一割り当て フィンガープリントの短時間再試行だけを同じ許可確認へcoalesceする。

### `grant_requests` / `grant_evaluation_jobs` / `grant_decisions`

`request_grant`からWorkspace、要求元Task/origin/委譲、許可確認、非同期操作、justification、証跡、ハーネス生成操作 キー、状態を保存する。`(workspace_id, challenge_id)`には非終端リクエストのpartial 一意制約を置く。オーナー交代や別`call_id`で再申請されても既存リクエスト/非同期操作を返す。評価 ジョブは入力スナップショット/ダイジェスト、プロファイル/Schemaバージョン、試行、リース、期限、エラーを持ち、技術障害時は同じ入力で再試行する。ポリシーAgentの確定構造化 出力は判断 ID、ジョブ、入力ダイジェスト、プロファイル/Schemaバージョン、決定時刻を持つ不変な`grant_decisions`へ保存するが、一時API セッション、レスポンス ID、ツール 呼び出し履歴は保存しない。

### `policy_grants`

Workspace、起点Task/origin/委譲/契約バージョン、起点 許可確認を`governance.db`へ保存する。起点 許可申請/判断、責任者経由なら起点 責任者の判断 参照、割り当てダイジェストも保存する。完全一致 IP/ポート/プロトコル、認証情報スコープ、`max_uses=1`、接続/バイト 上限、期限、ポリシーバージョン、`pending_activation | active | revoked` 状態、使用回数、revocationも保存する。ポリシー作成トランザクションでは認可経路を検証して`pending` 許可と`pending` 準備完了 送信キューを保存する。強制 点 `ACK`後の有効化 トランザクションで許可を`active`にして制御向け結果送信キューを確定する。制御は結果受信キュー適用時に非同期操作を完了して準備完了 イベントをTaskメールボックスへ追加する。

### `authority_requests` / `authority_decisions`

Control Planeの責任者ゲートウェイが、全Planeから受けた人間・外部責任者通信の正本を保存する。他Planeはこれらのテーブルや外部チャネルへ直接書き込まない。要求元Planeは判断対象と要否を所有し、ゲートウェイは認証・配送・期限・重複排除を所有する。

`require_authority`時にControl Planeは許可申請、許可確認、許可判断 ID、不変 割り当て ダイジェスト、プラットフォーム ポリシーが選んだ責任者、状態、期限を固定する。責任者は承認/拒否だけを返す。`authority_decisions`は認証済み回答者 主体、判断、根拠、決定時刻をリクエストごとに一件保存する。回答トランザクションはリクエストをロックして未解決・未期限切れを検証し、判断と統治向け送信キューを確定する。統治は受信キュー適用時にTask スナップショット、許可確認、ポリシー/DNS 鮮度を再検査して許可または拒否を確定し、制御は結果メッセージで非同期操作とメールボックスを終端する。遅延 レスポンスは既存終端結果へ収束する。

### `dns_resolutions`

WorkspaceごとのFQDN、解決済み IPv4/IPv6、DNS TTL、観測時刻と要求元Task 来歴を保存する。L4 許可は同一Workspaceのスナップショットに含まれるIPだけへ適用し、分岐先へスナップショットを継承せず、サンドボックスからの外部DNS、DoH、DoTによる迂回を許さない。

### `egress_review_jobs` / `egress_findings`

レビュー ジョブは固定ウォーターマーク、選定理由（high リスク / 異常 / ランダム sample / インシデント 再実行）、入力スナップショット/ダイジェスト、プロファイル/Schema バージョン、状態、試行、リース、レビュー 期限、キャプチャ ピン留め、エラーを保存する。一ジョブは最大一検出事項とし、検出事項は対象試行群、`benign | policy_bypass | suspicious | insufficient_evidence`、severity、根拠、証跡を保存する。ジョブ完了と検出事項 挿入を同一トランザクションで確定し、`review_job_id` 一意で再試行重複を防ぐ。外向き通信監査AgentのAgent実行、レスポンス ID、ツール 呼び出し履歴は保存しない。

### `policy_revision_jobs` / `policy_revision_proposals` / `policy_revisions`

検出事項からポリシーAgentを起動するジョブは対象 ポリシー キー、対象Workspace（全体 ベースラインならNULL）、固定入力スナップショット/ダイジェスト、NULL許容 候補 ルール 参照/ダイジェストを保存する。基底 ポリシーバージョン、候補 固定 タイムスタンプ、プロファイル/Schemaバージョン、試行、リース、期限、エラーも保存する。検出事項群は`policy_revision_job_findings` 結合でFK固定する。候補は最終判断前にジョブへ原子的に固定し、提案はジョブの固定値だけをコピーして回帰証跡、`update | no_change | require_authority` 判断、アプリケーション 状態を保存する。

`require_authority`では提案 ID/ダイジェスト、対象 ポリシー キー、スコープ、基底 バージョン、責任者、状態、期限を改訂 責任者への依頼へ固定し、判断へ認証済み回答者、承認/拒否、根拠、時刻を一件保存する。確定改訂は起点 提案、必要な責任者の判断、対象 ポリシー キー/参照/ダイジェスト、基底/previous/新規 バージョン、`pending_activation | active | superseded | cancelled`、`ACK`を保存する。ポリシーAgentの提案からポリシー マネージャーによるバージョン CAS、ルールエンジン `ACK`、`active`切替までを追跡可能にする。ポリシーAgentのAgent実行、レスポンス ID、ツール 呼び出し履歴は保存しない。

### `task_episodes`

Task終端後に一件。`TaskEpisode`は構造化 出力本文型であり、`task_episodes`永続行は`episode_id`、`task_id`、本文を保持する証跡DBへの`evidence_ref`、ダイジェストを持つ。ランタイムではエピソード Markdownファイルを生成しない。

### `episode_compilation_jobs`

終端Taskごとのエピソード Agent調査ジョブを保存する。状態、ステップ/入力/出力 トークン使用量の集計、上限スナップショット、プロファイル/出力 スキーマ バージョン、証跡参照、試行、リース/heartbeat、エラーを持つが、Agent ID、Agent実行、レスポンス ID、ツール 呼び出し履歴は持たない。`task_id`で冪等化し、期限切れリースは新しい一時セッションで最初から再調査する。ジョブ失敗や`needs_operator`はTask状態へ影響させない。

### `evidence_records` / `evidence_blobs` / `evidence_links`

証跡 レイヤーはSQLiteを初期実装の正本とする。

| テーブル | 役割 |
|---|---|
| `evidence_records` | 種別、Task、内容 型、ダイジェスト、サイズ、保持、秘匿化 メタデータ |
| `evidence_blobs` | compressed/暗号化済み 内容のチャンク BLOB |
| `evidence_links` | エピソード Statement、成果物、実行 項目等の根拠関係 |
| `evidence_text` | 検索対象テキストのFTS インデックス。再構築可能 |

証跡 IDと内容 ダイジェストを一意にし、メタデータ 挿入とBLOB保存を同一トランザクションで確定する。証跡 内容を個別ファイル、sidecar JSON、Markdownへ二重保存しない。

成果物など別集約または別DBから取り込む場合は、先に証跡DBでBLOBとメタデータをコミットして不変 `evidence_ref`を得てから、集約側の参照を送信キュー付きトランザクションで確定する。逆順は禁止する。参照されなかった証跡は孤立 照合/GCで回収し、参照確定時と定期監査でダイジェストを照合する。

エピソード 編纂 ジョブ開始時に、対象`task_id`へ固定した`episode_*` 読み取り専用 ビューを接続上へ公開する。エピソード Agentは基底 テーブルへアクセスせず、単一`query_evidence` ツールからこれらのビューだけをSQL クエリする。

コアまたは統治の状態を証跡DBへ取り込む場合は、ジョブ開始前に対象Taskの終端スナップショットをウォーターマーク/ダイジェスト付きメッセージとして受信し、証跡DBへmaterializeしてからジョブスコープのビューを構築する。Agent用接続には別DBを`ATTACH`せず、スナップショット後の変化を混入させない。

## 3. 詳細ER図

```mermaid
erDiagram
    AGENTS ||--o{ TASKS : owns
    TASKS ||--o{ TASKS : parent_of
    TASKS ||--o{ TASK_CONTRACT_VERSIONS : versions
    TASKS ||--o{ TASK_EVENTS : emits
    TASKS ||--o{ AGENT_RUNS : runs
    AGENT_RUNS ||--o{ AGENT_RUN_STEPS : contains
    AGENT_RUN_STEPS ||--o{ AGENT_RUN_ITEMS : emits
    AGENTS ||--o{ AGENT_RESOURCES : owns
    AGENT_RUNS ||--o{ AGENT_RESOURCES : may_create
    TASKS ||--|| TASK_PROGRESS : tracks
    TASK_PROGRESS ||--o{ TASK_PROGRESS_EVENTS : changes
    AGENT_RUNS ||--o{ RESUME_CURSORS : resumes_from
    TASKS ||--o{ CONTINUATIONS : suspends
    TASKS ||--o{ ASYNC_OPERATIONS : starts
    TASKS ||--o{ MAILBOX_ENTRIES : receives
    TASKS ||--o{ COMPLETION_REVIEWS : reviewed
    TASKS ||--o| TASK_OUTCOMES : ends_with
    TASKS ||--o| EPISODE_COMPILATION_JOBS : compiles
    EPISODE_COMPILATION_JOBS ||--o| TASK_EPISODES : produces
    TASKS ||--|| WORKSPACES : uses
    WORKSPACES ||--|| WORKSPACE_SECURITY_POLICY_BINDINGS : governed_by
    WORKSPACES ||--o{ WORKSPACE_SNAPSHOTS : snapshots
    WORKSPACES ||--o{ DNS_RESOLUTIONS : resolves
    TASKS ||--o{ ARTIFACTS : creates
    TASKS ||--o{ EGRESS_ATTEMPTS : originates
    EGRESS_ATTEMPTS ||--|| EGRESS_RULE_DECISIONS : governed_by
    EGRESS_ATTEMPTS ||--|| EGRESS_CAPTURE_MANIFESTS : captured_by
    EGRESS_CHALLENGES ||--o{ CHALLENGE_OBSERVATIONS : observed_as
    EGRESS_ATTEMPTS ||--o| CHALLENGE_OBSERVATIONS : links
    EGRESS_CHALLENGES ||--o{ GRANT_REQUESTS : requested_for
    GRANT_REQUESTS ||--o{ GRANT_EVALUATION_JOBS : evaluates
    GRANT_REQUESTS ||--o| GRANT_DECISIONS : decided_by
    GRANT_REQUESTS ||--o| AUTHORITY_REQUESTS : may_require
    AUTHORITY_REQUESTS ||--o| AUTHORITY_DECISIONS : decided_by
    EGRESS_CHALLENGES ||--o{ POLICY_GRANTS : grants
    GRANT_REQUESTS ||--o{ POLICY_GRANTS : authorizes
    GRANT_DECISIONS ||--o{ POLICY_GRANTS : authorizes
    AUTHORITY_DECISIONS ||--o{ POLICY_GRANTS : may_authorize
    EGRESS_ATTEMPTS ||--o| OUTBOUND_TRANSACTIONS : forwards
    EGRESS_REVIEW_JOBS ||--o| EGRESS_FINDINGS : produces
    EGRESS_FINDINGS ||--o{ POLICY_REVISION_JOB_FINDINGS : drives
    POLICY_REVISION_JOBS ||--o{ POLICY_REVISION_JOB_FINDINGS : includes
    POLICY_REVISION_JOBS ||--o| POLICY_REVISION_PROPOSALS : produces
    POLICY_REVISION_PROPOSALS ||--o| POLICY_REVISION_AUTHORITY_REQUESTS : may_require
    POLICY_REVISION_AUTHORITY_REQUESTS ||--o| POLICY_REVISION_AUTHORITY_DECISIONS : decided_by
    POLICY_REVISION_PROPOSALS ||--o| POLICY_REVISIONS : applied_as
    TASKS ||--o| TASK_EPISODES : compiles

    AGENTS {
      uuid agent_id PK
      text profile_id
      text status
      uuid current_task_id
    }

    TASKS {
      uuid task_id PK
      uuid parent_task_id FK
      uuid owner_agent_id FK
      uuid workspace_id FK
      text status
      int contract_version
      bigint version
    }

    WORKSPACE_SECURITY_POLICY_BINDINGS {
      uuid workspace_id PK
      text profile_ref
      int baseline_policy_version
      uuid pending_revision_id FK
      int pending_policy_version
      text status
      int version
    }

    GLOBAL_SECURITY_POLICY_BINDINGS {
      text target_policy_key PK
      text profile_ref
      int active_policy_version
      uuid pending_revision_id FK
      int pending_policy_version
      int version
    }

    ASYNC_OPERATIONS {
      uuid async_id PK
      uuid owner_task_id FK
      text tool_name
      text status
      text operation_key UK
      text result_ref
    }

    EGRESS_ATTEMPTS {
      uuid attempt_id PK
      uuid workspace_id FK
      uuid task_id FK
      uuid origin_task_id FK
      text delegation_chain_digest
      text binding_digest
      text decision
    }

    EGRESS_RULE_DECISIONS {
      uuid rule_decision_id PK
      uuid attempt_id FK
      text decision
      int policy_version
      text matched_rule_refs
      text reason_codes
      datetime evaluated_at
    }

    EGRESS_CAPTURE_MANIFESTS {
      uuid capture_manifest_id PK
      uuid attempt_id FK
      text request_coverage_json
      text response_coverage_json
      text completion_status
      datetime retention_expires_at
      text pinned_until_ref
    }

    DNS_RESOLUTIONS {
      uuid resolution_id PK
      uuid workspace_id FK
      uuid task_id FK
      text fqdn
      text resolved_ips
      int ttl_seconds
      datetime observed_at
    }

    EGRESS_CHALLENGES {
      uuid challenge_id PK
      uuid workspace_id FK
      uuid task_id FK
      uuid origin_task_id FK
      text binding_json
      text binding_digest
      text reason_codes
      boolean grant_eligible
      boolean auto_grant_eligible
      text required_authority_ref
      datetime expires_at
    }

    CHALLENGE_OBSERVATIONS {
      uuid challenge_id FK
      uuid attempt_id FK
      datetime observed_at
    }

    GRANT_REQUESTS {
      uuid request_id PK
      uuid challenge_id FK
      uuid workspace_id FK
      uuid task_id FK
      uuid origin_task_id FK
      text delegation_chain_digest
      uuid async_id FK
      text justification
      text operation_key UK
      text status
    }

    GRANT_EVALUATION_JOBS {
      uuid job_id PK
      uuid request_id FK
      text input_digest
      text status
      int attempt
      datetime lease_expires_at
    }

    GRANT_DECISIONS {
      uuid decision_id PK
      uuid request_id FK
      text decision
      text input_digest
      text profile_version
      text output_schema_version
      text decision_json
      datetime decided_at
    }

    POLICY_GRANTS {
      uuid grant_id PK
      uuid challenge_id FK
      uuid source_request_id FK
      uuid source_decision_id FK
      uuid source_authority_decision_id FK
      uuid workspace_id FK
      uuid source_task_id FK
      uuid origin_task_id FK
      text delegation_chain_digest
      int contract_version
      text scope_json
      int policy_version
      int max_uses
      int use_count
      int connection_limit
      text status
      datetime expires_at
      datetime revoked_at
    }

    AUTHORITY_REQUESTS {
      uuid authority_request_id PK
      uuid request_id FK
      uuid decision_id FK
      uuid challenge_id FK
      text binding_digest
      text authority_ref
      text status
      datetime expires_at
    }

    AUTHORITY_DECISIONS {
      uuid authority_decision_id PK
      uuid authority_request_id FK
      text responder_principal
      text decision
      text rationale
      datetime decided_at
    }

    EGRESS_REVIEW_JOBS {
      uuid review_job_id PK
      text watermark
      text selection_reason
      text input_digest
      text profile_version
      text status
      int attempt
      text lease_owner
      datetime lease_expires_at
      datetime review_deadline_at
      text capture_pin_ref
      text last_error_ref
    }

    EGRESS_FINDINGS {
      uuid finding_id PK
      uuid review_job_id FK
      text verdict
      text severity
      text rationale
      text evidence_refs
    }

    POLICY_REVISIONS {
      uuid revision_id PK
      uuid workspace_id FK
      text target_policy_key
      uuid source_proposal_id FK
      uuid source_authority_decision_id FK
      int base_policy_version
      int previous_policy_version
      int new_policy_version
      text target_policy_ref
      text target_policy_digest
      text regression_evidence_refs
      text approved_by
      text status
      text activation_ack_ref
      datetime activated_at
    }

    POLICY_REVISION_JOBS {
      uuid job_id PK
      uuid workspace_id FK
      text target_policy_key
      text input_digest
      text candidate_policy_ref
      text candidate_policy_digest
      int base_policy_version
      datetime candidate_fixed_at
      text profile_version
      text output_schema_version
      text status
      int attempt
      datetime lease_expires_at
      datetime invocation_deadline_at
    }

    POLICY_REVISION_PROPOSALS {
      uuid proposal_id PK
      uuid job_id FK
      uuid workspace_id FK
      text target_policy_key
      text candidate_policy_ref
      text candidate_policy_digest
      int base_policy_version
      text decision_json
      text application_status
      text regression_evidence_refs
    }

    POLICY_REVISION_JOB_FINDINGS {
      uuid job_id FK
      uuid finding_id FK
    }

    POLICY_REVISION_AUTHORITY_REQUESTS {
      uuid authority_request_id PK
      uuid proposal_id FK
      text candidate_policy_digest
      uuid workspace_id FK
      text target_policy_key
      int base_policy_version
      text authority_ref
      text status
      datetime expires_at
    }

    POLICY_REVISION_AUTHORITY_DECISIONS {
      uuid authority_decision_id PK
      uuid authority_request_id FK
      text responder_principal
      text decision
      text rationale
      datetime decided_at
    }

    OUTBOUND_TRANSACTIONS {
      uuid transaction_id PK
      uuid attempt_id FK
      uuid workspace_id FK
      uuid task_id FK
      uuid grant_id FK
      text request_binding_digest
      text outer_request_digest
      text response_digest
      text status
    }

    TASK_EPISODES {
      uuid episode_id PK
      uuid task_id FK
      text evidence_ref
      text digest
    }
```

## 4. SQL制約例

```sql
CREATE UNIQUE INDEX one_active_task_per_owner
ON tasks(owner_agent_id)
WHERE status NOT IN ('completed', 'cancelled');

CREATE UNIQUE INDEX one_workspace_per_task
ON workspaces(task_id);

CREATE UNIQUE INDEX one_security_policy_binding_per_workspace
ON workspace_security_policy_bindings(workspace_id);

CREATE UNIQUE INDEX one_outcome_per_task
ON task_outcomes(task_id);

CREATE UNIQUE INDEX one_episode_per_task
ON task_episodes(task_id);

CREATE UNIQUE INDEX async_operation_key
ON async_operations(operation_key);

CREATE UNIQUE INDEX mailbox_event_dedup
ON mailbox_entries(event_id);

CREATE UNIQUE INDEX one_active_grant_request_per_challenge
ON grant_requests(workspace_id, challenge_id)
WHERE status IN ('pending', 'evaluating', 'waiting_authority');

CREATE UNIQUE INDEX one_challenge_observation_per_attempt
ON challenge_observations(attempt_id);

CREATE UNIQUE INDEX one_rule_decision_per_attempt
ON egress_rule_decisions(attempt_id);

CREATE UNIQUE INDEX one_capture_manifest_per_attempt
ON egress_capture_manifests(attempt_id);

CREATE UNIQUE INDEX one_authority_decision_per_request
ON authority_decisions(authority_request_id);

CREATE UNIQUE INDEX one_finding_per_review_job
ON egress_findings(review_job_id);

CREATE UNIQUE INDEX one_proposal_per_policy_revision_job
ON policy_revision_proposals(job_id);

CREATE UNIQUE INDEX one_revision_per_policy_proposal
ON policy_revisions(source_proposal_id);

CREATE UNIQUE INDEX one_pending_revision_per_policy_target
ON policy_revisions(target_policy_key)
WHERE status = 'pending_activation';

CREATE UNIQUE INDEX one_policy_revision_job_finding
ON policy_revision_job_findings(job_id, finding_id);

CREATE UNIQUE INDEX one_revision_authority_request_per_proposal
ON policy_revision_authority_requests(proposal_id);

CREATE UNIQUE INDEX one_revision_authority_decision_per_request
ON policy_revision_authority_decisions(authority_request_id);
```

`egress_rule_decisions.attempt_id`と`egress_capture_manifests.attempt_id`だけをFK方向とし、試行との循環必須FKを作らない。試行、キャプチャ マニフェスト、ルール 判断、外向き 意図はdeferred constraintもしくは1つのトランザクションで確定し、コミット時に一対一を満たす。キャプチャ マニフェストはリクエスト/レスポンス 網羅率のいずれも無言のNULLにせず、`complete | partial | unavailable | incomplete`と欠落理由をCHECK/検証器で要求する。一時許可は起点 リクエスト/判断を必須とし、責任者経由では承認済み起点 責任者の判断を要求するトリガーまたはポリシー マネージャー検証を置く。

ポリシー 改訂 ジョブはfinal 判断受理前に候補 参照/ダイジェスト/基底 バージョン/固定 時刻を原子的に固定する。`update | require_authority`では全候補 フィールドを必須、`no_change`では候補 参照/ダイジェストをNULLにする。提案はジョブ固定値との一致をCHECK/検証器で要求する。対象 行の`pending` reservationとpartial 一意 インデックスを併用し、同じ対象へ複数改訂を配布しない。

循環TaskツリーはDB トリガーまたはTask マネージャーで検査する。

## 5. トランザクション境界

### Task状態 transition

```text
BEGIN
  UPDATE task SET version = version
    WHERE task_id = ? AND version = ?
  validate transition and version
  UPDATE task
  INSERT task_event
  INSERT outbox_event
COMMIT
```

### メールボックス 消費

```text
BEGIN
  claim unconsumed mailbox entries by conditional UPDATE + lease token
  update task / continuation if condition resolves
  mark entries consumed
  insert task events
COMMIT
```

### CASB 拒否と許可反映

以下の統治 集約更新は`governance.db`内、Taskメールボックス、非同期操作、責任者更新は`control.db`内で原子的に行う。両者の間は送信キュー/受信キュー メッセージと`ACK`で接続し、説明中の「同時確定」は同一所有DB内に限る。

通常許可でも外側接続前に外向き通信 試行、ルール 判断、リクエスト キャプチャ マニフェスト、`intent_committed` 外向きトランザクションをコミットする。DNS 上流 クエリとL4 接続も同じ順序を使う。レスポンス キャプチャと完了状態はサンドボックスへ返す前にコミットする。外部到達の可能性がある途中クラッシュはマニフェストを`incomplete`、トランザクションを`outcome_unknown`として照合する。必ず高リスク レビューへ送る。`failed`は外部未到達または失敗が確定した場合だけに使う。監査を永続化できない場合は転送しない。

許可確認と統治の`EgressBlocked`送信キューを`governance.db`の同一トランザクションでコミットしてから、CLIへ拒否レスポンスを返す。制御は受信キューとTaskメールボックスを`control.db`で確定する。許可作成時は統治がWorkspaceポリシー割り当て、許可申請、許可確認、判断、固定済み起点Taskスナップショットを再検査し、`pending_activation`の`PolicyGrant`、ポリシーバージョン、`pending`の準備完了送信キューを確定する。この時点では制御の非同期操作を完了しない。正本となる強制点のバージョン`ACK`を受けた統治の有効化トランザクションで許可を`active`にし、結果送信キューを作る。制御は結果受信キューを適用して非同期操作を`completed`にし、`AsyncCompleted`と`PolicyGrantReady`をTaskメールボックスへ追加する。各Dispatcherは最新のWorkspace/Taskゲート、許可の`active`状態、ポリシーバージョンを再検査する。

Task キャンセルとWorkspace 凍結/archive/destroyでは、制御が操作ゲートを閉じて失効 コマンドを送る。統治は未解決リクエスト/評価 ジョブをキャンセルし、`pending`または`active` 許可を失効し、`pending` 準備完了 送信キューを無効化して`ACK`する。制御は`ACK`適用トランザクションで非同期とメールボックスを終端する。責任者期限切れも制御 判断、統治 拒否、制御 非同期完了の順に冪等 メッセージで収束させる。

強制 点は転送前に`active`/unexpired/unrevoked/割り当て一致/使用回数を条件付き更新し、使用回数 incrementまたはL4 接続 slot 予約と外向きトランザクション 意図を同一原子操作で確定する。予約失敗時は外部へ接続しない。

### 恒久ルール 改訂

ポリシー マネージャーは対象Workspace 割り当てまたは全体 ポリシー 行をロックする。提案の候補ダイジェスト、対象 ポリシー キー、スコープ、回帰結果、必要な承認済み改訂 責任者の判断、`current_version == base_policy_version`、`pending_revision_id IS NULL`を検証する。ロック下の単調シーケンスで新規バージョンを一意発番し、改訂を`pending_activation`で保存すると同時に対象 行へ`pending` 改訂/バージョンを予約する。`active`バージョンは切り替えず、この一件だけをルールエンジンへ配布する。

ルールエンジンのバージョン `ACK`後に同じ対象を再ロックし、`pending_revision_id == revision_id`とバージョンを再検査して改訂を`active`、旧改訂を`superseded`、割り当て 現在 バージョンを新規 バージョンへ切り替え、`pending`予約をclearする。競合提案は配布前に`stale`とし、配布失敗・キャンセル・タイムアウトでは旧バージョンを`active`のまま保って予約を原子的に解放する。

改訂 責任者の拒否/期限切れ、重複・遅延 レスポンスはリクエストをロックして一度だけ終端し、提案 状態と通知送信キューを同一トランザクションで確定する。

## 6. 状態の正本

| 対象 | 正本 |
|---|---|
| Taskライフサイクル | `tasks` + `task_events` |
| Workspace 内容 | Workspace storage + スナップショット |
| Workspace セキュリティ ポリシー | `workspace_security_policy_bindings` + バージョン付きルール documents |
| LLM short 継続情報 | レスポンス ID補助、独自継続情報が正本 |
| 非同期 結果 | `async_operations` + `mailbox_entries` |
| 外向き通信 ポリシー / 監査 | Attempts + ルール Decisions + Challenges + Grants + Transactions + Findings + Revisions |
| エピソード型 記憶 | 不変 Taskエピソード |
| 意味 記憶 | Git管理Markdown |

## 7. 保持

- Taskイベント / Outcomes / 外向き通信 判断・許可確認・許可・検出事項・改訂 監査: 長期保持
- 無害化済み/暗号化済み 外向き通信 キャプチャ: 全通信を短期保持し、高リスク/検出事項関連は長期保持へ昇格
- Agent レスポンス logs: 保持 ポリシーに従う
- 終端 logs: 成果物化された重要部分以外は短期化可能
- Workspace: Task終端後に証跡DBへスナップショットを取り込み、作業実体はポリシーで削除
- Taskエピソード:長期保持
- 意味 Wiki: Git履歴付きで長期保持

PIIや秘密情報を含む証跡は分類し、エピソードや意味本文へ直接複製しない。

## 8. Versioning

- `task.contract_version`: 目的 / 受け入れ条件 / 指示変更
- `task.version`: すべての状態更新
- `memory_version`: クエリ時のWiki コミット
- `candidate_version`: 完了案の連番
- `policy_bundle_digest`: Judgeが読んだポリシー集合
- `request_digest`: CASBが評価・転送した外向き通信 リクエスト

これらをイベントへ記録して再現性を確保する。
