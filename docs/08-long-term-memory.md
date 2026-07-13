# 長期記憶の設計書 V2

## 1. 目的

長期記憶は、Work Agentが自由検索する共有データベースではない。独立したWiki AgentがTaskエピソード群を編纂し、ハーネスが現在Taskに必要な文脈だけを強制挿入する基盤機能である。

```text
Task Execution
  → Task Episode
  → Wiki Agent Maintenance
  → Episodic / Semantic Wiki
  → Wiki Agent Query
  → Harness Injection
  → Work Agent
```

### 1.1 実装・フレームワーク境界

Memory PlaneはPythonの独立サービスとして実装し、OpenAI Agent SDKをエピソード AgentとWiki Agentの一時ツール runnerに使用する。フレームワークのセッション、トレース、チェックポイント、レスポンス ID、ツール 呼び出し履歴をジョブやTaskの正本にしない。SDK標準tracingは既定で無効化し、無害化済みなジョブ メタデータだけを独自telemetryへ出す。

記憶サービスは`evidence.db`と意味 Wiki リポジトリを単独で書き込み所有する。Go コアとは記憶 送信キュー/受信キューとUnixドメインソケットでメッセージを交換し、DB ファイルを共有しない。エピソード 編纂 ジョブのリース 期限切れ時はフレームワーク セッションを再開せず、固定入力 スナップショットから新しい実行で最初から再調査する。

初期フレームワークはOpenAI Agent SDKとする。固定グラフ、HITL、途中チェックポイントが必要になった場合だけMicrosoft Agent フレームワークまたはLangGraphを再評価する。詳細は[13-technology-stack.md](13-technology-stack.md)を正本とする。

## 2. なぜメッセージを未加工単位にしないか

LLMメッセージ、終端 コマンド、ツール 呼び出しは細かすぎる。

- 1つの意図が多数のメッセージへ分散する
- 試行錯誤や撤回が多い
- オーナー責任と完了条件が見えない
- 長期的に検索するとノイズが支配する

長期記憶の時系列単位は、単一オーナーが責任を持ったTaskとする。

```text
Message / Command / Tool Call = Evidence / Activity
Task Episode                  = Episodic Memory Unit
```

## 3. Taskエピソード

Taskが`completed`または`cancelled`に入った後、エピソード Agentが1つの不変記録を作る。`suspended`中はTask進捗、再開 カーソル、障害証跡を保存するが、エピソードを確定しない。

エピソード AgentはTask進捗の履歴をCourseとUnresolvedの主要入力にする。ただし進捗はオーナー申告なので、Taskイベント、ツール結果、成果物、完了レビューと照合し、観測事実と混同しない。

```typescript
type TaskEpisode = {
  episode_id: string;
  task_id: string;
  parent_task_id?: string;
  owner_agent_id: string;

  situation: {
    objective: string;
    acceptance: string;
    instructions?: string;
    initial_context_summary: string;
  };

  temporal_context: {
    started_at: string;
    ended_at: string;
  };

  course: {
    summary: string;
    important_transitions: EpisodeTransition[];
    child_episode_refs: string[];
  };

  outcome: {
    status: "completed" | "cancelled";
    owner_judgement: string;
    acceptance_review_ref?: string;
    artifact_refs: string[];
    outbound_transaction_refs: string[];
  };

  surprises: EpisodeStatement[];
  decisions: EpisodeStatement[];
  unresolved: EpisodeStatement[];
  evidence_refs: string[];
};
```

TaskエピソードはTask 結果の単なる要約ではない。初期状況、重要な転換、予想外、最終判断を含める。

## 4. 証跡 レイヤー

エピソードから低位証拠へ辿れるようにする。証跡 レイヤーの正本はSQLite等のtransactional データベースとし、証跡ごとのMarkdown、JSON、ログ、BLOBファイルは作らない。

```text
Task Episode
  ├─ Agent response logs
  ├─ terminal logs
  ├─ Git commits / diffs
  ├─ artifacts
  ├─ child Task outcomes
  ├─ Ask / Advice
  ├─ Escalation / Decision
  ├─ completion reviews
  └─ Egress Challenge / Grant / Outbound Transaction records
```

通常の記憶検索ではエピソードを読む。根拠確認や矛盾解消のときだけ低位証跡へ掘る。

Agent レスポンス ログの具体的な保存範囲と保持は[05-runtime-and-responses-api.md](05-runtime-and-responses-api.md)の「Agent実行記録 ポリシー」を正本とする。エピソードに必要な主張を短期実行 ログだけへ依存させない。

### 証跡 データベース

```typescript
type EvidenceRecord = {
  evidence_id: string;
  kind:
    | "agent_run_item"
    | "tool_log"
    | "artifact"
    | "workspace_snapshot"
    | "decision"
    | "review"
    | "egress"
    | "episode";
  task_id?: string;
  content_type: string;
  content_digest: string;
  byte_length: number;
  retention_class: "long" | "policy" | "short";
  redaction_status: "none" | "redacted" | "encrypted";
  created_at: string;
};
```

本文は`evidence_blobs`へBLOBとして保存し、大きい内容は固定サイズでチャンク化する。メタデータ、本文、起点 関係を同じDB内で管理する。

```text
evidence_records   metadata / digest / retention
evidence_blobs     compressed or encrypted BLOB chunks
evidence_links     source -> target / relation
evidence_text      optional FTS index
```

`artifact://...`、`evidence://...`などの論理 参照はDB 記録を指すURIであり、ファイルシステム パスではない。エピソード Agentは読み取り専用 SQL ツールで証跡DBをpageして読む。

Workspace内の作業ファイルは証跡 レイヤーではない。根拠として固定する時点でハーネスが内容またはスナップショットを証跡DBへ取り込み、ダイジェストを確定する。取り込み後はワークツリーを削除しても証跡参照が壊れない。

## 5. エピソード Agent

エピソード AgentはMemory Planeに属する専用Agentである。Work AgentのTaskオーナーにはならず、終端Taskごとのエピソード 編纂 ジョブを複数レスポンス ステップで調査・編成する。1回のLLM呼び出しですべての履歴を要約する方式には依存しない。

### 編纂 ジョブ

```typescript
type EpisodeCompilationJob = {
  job_id: string;
  task_id: string;
  status:
    | "pending"
    | "investigating"
    | "validating"
    | "completed"
    | "needs_operator";
  step_count: number;
  input_tokens: number;
  output_tokens: number;
  max_steps: number;
  input_token_budget: number;
  output_token_budget: number;
  artifact_read_byte_budget: number;
  deadline_at: string;
  profile_version: string;
  output_schema_version: string;
  lease_owner?: string;
  lease_expires_at?: string;
  evidence_refs: string[];
  attempt: number;
  last_error_ref?: string;
};
```

Memory PlaneはTask終端イベントからジョブを冪等に生成する。ジョブは処理の再試行と重複防止のための機能固有レコードであり、Agent実行ではない。内部レスポンス ID、ツール 呼び出し履歴、ステップごとの入出力は永続化しない。ワーカーはリース付きでジョブを確保し、heartbeatで期限を更新する。期限切れの`investigating` / `validating` ジョブは`attempt`を増やして再確保し、部分レスポンスを破棄して新しい一時セッションで初期索引から再調査する。ジョブ失敗は終端Taskの状態を戻さず、再試行後も解消しなければ`needs_operator`にする。

### 初期コンテキスト

ハーネスは全履歴本文ではなく、調査用索引を初期コンテキストへ渡す。

- Final Task契約と結果
- Task進捗の現在値とバージョン
- Taskイベント、Agent実行、子Task、成果物、判断、レビュー、外向き通信/許可の件数とカーソル
- 主要成果物と証跡の参照
- エピソード Agentの調査上限と出力Schema

エピソード Agentは不足する詳細を単一の読み取り専用 SQL ツールで取得する。

関数 ツール引数は[evidence-query-tool.スキーマ.JSON](../schemas/draft-v0/memory-plane/evidence-query-tool.schema.json)、結果は[evidence-query-result.スキーマ.JSON](../schemas/draft-v0/memory-plane/evidence-query-result.schema.json)を正本とする。

### 読み取り専用 証跡ツール

```typescript
query_evidence({
  sql: string,
  params?: Array<string | number | boolean | null>,
  max_rows?: number
})
```

ツールはSQLiteのparameterized `SELECT`または`WITH ... SELECT`だけを実行し、列名付きJSON rows、切り詰め済み flag、次page用カーソル情報を返す。エピソード Agentには初期コンテキストで利用可能な読み取り専用 ビュー、column、関係、FTS構文を提示する。

```text
episode_task
episode_contract_versions
episode_progress_events
episode_task_events
episode_agent_run_steps
episode_agent_run_items
episode_child_outcomes
episode_decisions
episode_reviews
episode_egress_events
episode_artifacts
episode_evidence_text
```

各ビューは編纂 ジョブの`task_id`へ固定される。SQLite 接続は`query_only`、基底 テーブル非公開、拡張無効とし、Authorizerで`INSERT / UPDATE / DELETE / DDL / PRAGMA / ATTACH / DETACH`を拒否する。ハーネスはクエリ タイムアウト、VM ステップ、行、返却バイト、BLOB チャンクの上限を強制する。

Memory Planeはジョブ開始前に、コアと統治の送信キューから受信した終端Taskの契約、進捗、Task/Agent実行 イベント、子 結果、判断、レビューをウォーターマークとダイジェスト付き終端スナップショットとして証跡DBへ取り込む。外向き通信の許可確認、一時許可、外向きトランザクション、成果物索引も同じスナップショットへ含める。そのスナップショットからジョブスコープのTEMP/読み取り専用 ビューを構築する。構築後にAgent用接続を開き、接続自身による`ATTACH`は禁止する。実行中に他Plane DBの変化を直接参照させない。

エピソード AgentはSQLの`LIMIT`とkeyset paginationを使って必要な証跡だけを複数ステップで調査する。巨大BLOBを一度に返さず、テキスト ビューまたはチャンク インデックスを指定して読む。

次の操作は許可しない。

- Task、契約、進捗、Workspaceの変更
- 外向き通信や一時許可
- Taskの生成・キャンセル
- Agentリソース操作
- Wikiの直接更新

### 関数 呼び出しと構造化 出力

各Responses API呼び出しには、最初から`query_evidence`とTaskエピソード用`text.format: json_schema`を同時に指定し、`tool_choice: "auto"`とする。

`task_episode_schema`の正本は[`task-episode.schema.json`](../schemas/draft-v0/memory-plane/task-episode.schema.json)である。

```typescript
responses.create({
  model,
  input,
  tools: [query_evidence_tool],
  tool_choice: "auto",
  text: {
    format: {
      type: "json_schema",
      name: "task_episode",
      strict: true,
      schema: task_episode_schema
    }
  }
});
```

証跡が不足している間は関数 呼び出しを返し、ハーネスが同じ`call_id`の`function_call_output`を返して連鎖を継続する。十分と判断した時点で、モデルはSchemaに従ったメッセージを返す。ハーネスは調査フェーズと最終生成フェーズを分けず、`finish_investigation` ツール、`tool_choice: "none"`への切替、最終化専用レスポンスを要求しない。

一レスポンスに複数関数 呼び出しがある場合はすべて処理する。未処理呼び出しがあるメッセージは最終エピソードとして確定しない。

### 上限

Memory Planeはジョブ作成時に`max_steps`、入力/出力 トークン 予算、成果物読取バイト、ジョブ 期限、プロファイル/出力 スキーマ バージョンを固定する。上限へ近づいた場合は「確認できない事項を`unresolved`へ残し、利用済み証跡だけで確定する」旨を次入力へ追加する。上限超過や構造化 出力未生成が続く場合はジョブを再試行または`needs_operator`にする。

### 出力と検証

最終メッセージは`TaskEpisode` JSON Schemaへ適合しなければならない。構造化 出力は形を保証するが内容の正しさまでは保証しないため、ハーネスは次を検証する。

- ジョブのTask ID、最終契約、結果との一致
- 全`source_refs`、成果物、外向きトランザクション、子 エピソード参照の存在
- `observed`がハーネス観測証跡を参照していること
- Task進捗由来の解釈が`owner_asserted`であること
- 未完了進捗が`unresolved`へ反映されていること
- 秘密情報や不透明 継続情報 BLOBを本文へ含めていないこと

検証成功後に`task_episodes`へ一件だけ保存する。

### Epistemic 状態

```typescript
type EpisodeStatement = {
  text: string;
  source_refs: string[];
  epistemic_status:
    | "observed"
    | "owner_asserted"
    | "compiler_inferred";
};
```

### Observed

状態遷移、テスト exit コード、外向き通信 拒否/転送結果、成果物 ダイジェストなど、ハーネスが観測したもの。

### オーナー申告

「原因はXだった」「この成果で受け入れ条件を満たした」などオーナーの解釈。

### Compiler inferred

複数イベントからエピソード Agentが再構成した説明。既存フィールド名との互換のため`compiler_inferred`を維持する。

三者を混同しない。

## 6. エピソード型 記憶と意味 記憶

| 層 | 問い | 基本形式 |
|---|---|---|
| エピソード型 | 何が、いつ、どのTask文脈で起きたか | Taskエピソード |
| 意味 | この領域をどう理解し、何を予測すべきか | 概念 / Schema / スクリプト / ケース パターン |

意味 Wikiはエピソードの要約集ではない。複数エピソードから抽象化・統合された認知モデルである。

詳細は[09-semantic-wiki-schema.md](09-semantic-wiki-schema.md)を参照。

## 7. Wiki Agent

Wiki AgentはWork Agentから独立したロールで、2つのモードを持つ。

### メンテナンス モード

- 新エピソードを読む
- 既存意味 Wikiとの適合を評価する
- 概念 / Schema / スクリプト / ケース パターンを更新する
- 矛盾や反例を保持する
- Markdown パッチを評価して公開する

### クエリ モード

- 現在Task契約を読む
- 関連意味 モデルを選ぶ
- 必要なら類似エピソードを選ぶ
- Task固有の記憶コンテキストを生成する

Wiki Agentは作業Taskを実行せず、外部ネットワークや外向き通信 許可権限を持たない。

## 8. Work Agentのアクセス

Work Agentは原則として意味 Wikiファイルやエピソード ストアを直接読まない。

理由を示す。

- 記憶探索という別目的が現在Taskへ混入する
- 必要な記憶の存在をAgent自身が知らない
- 古い・反例・supersededな説明を無秩序に読む
- トークン予算を制御しにくい
- 検索しないAgentが多い

例外はWiki保守・評価を目的とする専用ロールだけである。

## 9. ハーネスによる強制問い合わせ

ハーネスは次の時点でWiki Agentへコンテキストを要求する。

| フェーズ | 目的 |
|---|---|
| `task_start` | 関連概念、既知制約、失敗例、標準スクリプト |
| `subtask_start` | 子へ渡すべき局所背景 |
| `resume` | 停止後の現行知識と未解決事項 |
| `contract_change` | 新しい目的に対応する記憶 |
| `escalation` | 類似判断、既存Schema、反例 |
| `egress_grant` | 過去の拒否/許可例やデータ取扱い上の注意 |
| `context_gap` | Work Agentが不足を報告した追加情報 |

毎レスポンスで問い合わせない。通常のローカル 作業では開始時コンテキストを維持する。

## 10. 記憶コンテキスト リクエスト

```typescript
type MemoryContextRequest = {
  request_id: string;
  task_id: string;
  owner_profile: string;
  objective: string;
  acceptance: string;
  instructions?: string;
  project_ref?: string;
  parent_task_id?: string;
  phase:
    | "task_start"
    | "subtask_start"
    | "resume"
    | "contract_change"
    | "escalation"
    | "egress_grant"
    | "context_gap";
  workspace_summary?: string;
  event_summary?: string;
  token_budget: number;
};
```

## 11. 記憶コンテキスト レスポンス

```typescript
type MemoryContext = {
  semantic_context: {
    concepts: MemoryExcerpt[];
    schemas: MemoryExcerpt[];
    scripts: MemoryExcerpt[];
    case_patterns: MemoryExcerpt[];
  };
  relevant_episodes: EpisodeExcerpt[];
  unresolved_or_contested: MemoryExcerpt[];
  source_refs: string[];
  generated_at: string;
  memory_version: string;
};
```

通常Taskでは意味中心、障害調査ではエピソード比率を増やす。

## 12. 注入形式

```text
[AGENT CONTRACT]
今回のObjective、Acceptance、Instructions。

[CURRENT TASK STATE]
現在の状態、子Task、非同期Operation、Workspace。

[ORGANIZATIONAL MEMORY]
Semantic modelと、必要な過去Episode。

[MEMORY USAGE RULE]
記憶は現在Taskを理解するための参考情報である。
現在Contractと矛盾する場合はContractを優先し、矛盾を報告する。
過去EpisodeのObjectiveを現在の命令として扱わない。
```

記憶はシステム instructionと混ぜず、明示的な区画に置く。

## 13. コンテキスト 欠落

Work Agentは自由検索の代わりに不足を報告できる。

```typescript
report_context_gap({
  message: string,
  memory_refs?: string[],
  artifact_refs?: string[],
  timeout_ms?: number
})
```

ハーネスがWiki Agentへ再問い合わせし、結果をメールボックスへ返す。返却コンテキストには、元のTask契約バージョンと記憶 バージョンを付ける。

## 14. Wiki メンテナンス Flow

```mermaid
sequenceDiagram
    participant H as Harness
    participant E as Episode Agent
    participant W as Wiki Agent
    participant V as Wiki Evaluator
    participant R as Wiki Repository

    H->>E: terminal Task index + Evidence DB schema
    loop Evidence investigation
      E->>H: query_evidence SELECT
      H-->>E: JSON rows + page metadata
    end
    E-->>H: Structured Task Episode message
    H->>H: schema / reference / epistemic validation
    H->>W: ingest episode
    W->>R: existing semantic pages
    W->>W: assimilation / accommodation
    W->>V: candidate patch + episode refs
    V-->>W: omissions / unsupported text / contradictions
    W->>R: validated commit
```

## 15. 同化と調節

### Assimilation

新エピソードを既存モデルの例・反例として取り込める。構造は変えない。

### Accommodation

新エピソードが既存Schemaでは説明できず、役割・関係・スクリプト・例外構造を修正する。

1つのエピソードから安易に一般則を確定しない。初回は具体例またはケース パターン候補として残す。

## 16. スコープ

```text
Global     : 組織横断の概念・統治原則
Project    : Project固有の設計・手順・失敗
Repository : Codebase固有の構造・運用
Task       : 実行中の一時Context。終了後にEpisodeへ移る
```

意味ページは本文で適用範囲を説明する。フロントマターへ大量のスコープ メタデータを持たせない。

## 17. 訂正と矛盾

- エピソードは上書きしない
- 訂正イベントまたはCorrection エピソードを追加する
- 意味ページは新証跡に応じて更新する
- 古い理解はGit履歴と本文リンクから辿れる
- 反例を削除しない
- 未解決の矛盾は本文へ明示する

## 18. Storage

```text
Memory Plane
├── evidence.db
│   ├── task_episodes
│   ├── evidence_records
│   ├── evidence_blobs
│   ├── evidence_links
│   └── evidence_text (FTS)
└── semantic/
│   ├── concepts/
│   ├── schemas/
│   ├── scripts/
│   └── case-patterns/
```

実装ファイル名は`evidence.db`を標準とする。記憶 受信キュー/送信キュー、エピソード 編纂 ジョブ、記憶コンテキスト生成ジョブも同じDBへ置き、記憶サービスだけが書き込みする。

Taskエピソードと低位証跡はDBへ保存し、個別ファイルを生成しない。意味 Wikiだけは人間が保守・レビューできるMarkdownとしてGit管理する。DB バックアップはSQLite Online Backup APIまたは整合したスナップショットを使い、稼働中DBファイルの単純コピーには依存しない。

## 19. 不変条件

```text
時系列記憶の単位はTask Episode
MessageはEvidenceであり長期記憶単位ではない
EpisodeはTask終端後に一度だけ確定する
Work AgentはWikiファイルを直接探索しない
HarnessがWiki Agentを強制的に呼ぶ
Semantic記憶は単一Claim一覧ではない
Semantic本文の事実・説明は文または段落単位でEpisodeへリンクする
Wiki更新はWork Agent Runから分離したWiki maintenance Jobで実行する
```
