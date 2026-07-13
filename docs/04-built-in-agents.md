# 組み込みAgent設計

## 1. 対象と位置づけ

本書は、L1・L2・L3のWork Agentとは独立した内部機能として実装する組み込みAgent群を定義する。

```text
Work Agent                  Built-in LLM Component
-------------------------   ------------------------------
L1 / L2 / L3                Acceptance Reviewer Agent
Taskの単一Owner             Policy Agent
Objective達成に責任を持つ   Egress Audit Agent
delegate / ask / escalate   Episode Agent / Wiki Agent
```

組み込みAgentは作業Taskのオーナーにならず、L1・L2・L3の階層にも参加しない。Work Agent間の`delegate`、`ask`、`escalate`で生成・呼出しせず、対応する内部機能が定義済みトリガーからLLM処理を呼び出す。

組み込みAgentはハーネスのAgent管理対象ではない。Agent レジストリへ登録せず、Agent ID、AgentStatus、オーナー 割り当て、Agent実行、継続情報、Agentリソースを生成・永続化しない。「Agent」は固定指示、ツール、出力Schemaを備えたLLM処理単位を表す便宜的名称である。

Work Agentのライフサイクルは[03-agent-lifecycle.md](03-agent-lifecycle.md)、ポリシーAgentの判定規則は[07-governance.md](07-governance.md)、エピソード AgentとWiki Agentの記憶処理は[08-long-term-memory.md](08-long-term-memory.md)を併せて正本とする。

## 2. 共通実行モデル

組み込みAgentは用途ごとの内部コンポーネントがResponses APIのツール ループとして実行する。複数レスポンス ステップを使う場合も、それはコンポーネント内部の一時的なAPI セッションであり、ハーネスの`AgentRun`ではない。エピソード AgentとWiki AgentはPython 記憶サービス内でOpenAI Agent SDKのephemeral ランナーを利用する。受け入れ条件レビュアーはGo コア、ポリシーAgentと外向き通信監査AgentはRust 統治サービスがそれぞれ所有し、フレームワーク固有型やセッションをPlane間契約へ出さない。

```typescript
type BuiltInAgentKind =
  | "acceptance_reviewer"
  | "policy_agent"
  | "egress_audit_agent"
  | "episode_agent"
  | "wiki_agent";

type BuiltInInvocation = {
  kind: BuiltInAgentKind;
  subject_ref: string;
  input_snapshot: unknown;
}; // process memory内だけで扱い、永続化しない
```

共通不変条件は次のとおり。

1. Agent レジストリ、Agent実行、オーナー 割り当て、オーナー排他を使用しない。
2. トリガー、入力スナップショット、ツール set、出力Schemaを機能種別ごとに固定する。
3. Work Agentの未永続化scratch コンテキストを入力にしない。
4. 組み込みAgentはTaskやポリシー ストアを直接更新せず、呼出し元コンポーネントが出力を検証して適用する。
5. API セッション、レスポンス ID、ツール 呼び出し履歴、ステップ、組み込みAgent用実行記録は永続化しない。
6. 組み込みAgentの障害と、判定対象Taskの失敗を同一視しない。
7. 永続化するのは機能固有の入力ダイジェスト、確定結果、必要なジョブ状態だけとする。

## 3. 受け入れ条件レビュアー Agent

受け入れ条件レビュアー Agentは、オーナーが提出した完了案を現行Task契約の受け入れ条件と照合する独立した短命Agentである。一件の完了レビューに対応するレビュー ジョブとして生成し、Taskの実装や修正、一般的なコードレビューは行わない。

### 3.1 トリガーと起動

オーナーが完了案を提出すると、ハーネスが次を行う。

1. 提出者が現在のオーナーであることを確認する
2. `contract_version`が現行版であることを確認する
3. 必須 子が完了していることを確認する
4. 結果、成果物、証跡、必須 子 結果を含む案 スナップショット 参照/ダイジェストと、レビュアーへ渡す完全な入力 スナップショット 参照/ダイジェストを固定する
5. レビュアー専用コンテキスト、ツール、構造化 出力 スキーマでLLM処理を開始する

オーナーがレビュアーの指示や出力を操作できないよう、オーナーとは別の一時的なレスポンス 連鎖を使う。モデルを分離するかはDeployment ポリシーで選択できるが、入力スナップショットと実行権限の分離は必須とする。このレスポンス 連鎖とレスポンス IDはレビュー確定または失敗後に破棄する。

### 3.2 入力コンテキスト

レビュアーには次だけを渡す。

- レビュアーの責務、禁止事項、判定基準を定めたDeveloper 指示
- Task IDと現行Task契約
- 固定済み完了案
- 必須 子の結果要約
- 成果物および証跡の参照、ダイジェスト、取得方法
- レビュー対象の契約バージョンと案 バージョン

レビュアーは元の受け入れ条件を満たすかだけを判定し、新しい要件を追加しない。

### 3.3 証跡ツール

レビュアーには対象Taskと案 スナップショットに固定された必須 descendant 証跡参照の閉包だけを公開する読み取り専用 証跡 ビューを問い合わせるツールを許可する。証跡 レイヤーがSQLiteの場合はparameterized `SELECT`を実行する単一クエリ ツールとし、基底 テーブル、閉包外のTask、書き込み クエリへアクセスさせない。

外部ネットワーク、Workspace変更、Task更新、`Delegate`、質問、`Escalate`、完了案提出のツールは渡さない。クエリには行数、応答バイト数、実行時間、VM ステップ、BLOB チャンクの上限を適用する。

### 3.4 出力と適用

最終レスポンスは`AcceptanceReviewDecision`の構造化 出力とする。

Responses APIには[built-in-agent-outputs.JSON](../schemas/draft-v0/api/built-in-agent-outputs.json)の`acceptance_review`を`text.format`として指定する。判定を確定する関数 ツールは渡さない。

- `accept`: 受け入れ条件を満たす結果と十分な`evidence_refs`がある
- `reject`: 証跡から受け入れ条件未達を確認でき、`unmet_acceptance`と`evidence_refs`を特定できる
- `insufficient_evidence`: 達成・未達を判断できず、`required_evidence`と確認済み`evidence_refs`を特定できる

完了レビューコンポーネントが出力を検証した後、`CompletionReview` 挿入、`CompletionReviewJob.completed`、Task状態遷移、`CompletionReviewed` イベントをTask 集約の同一トランザクションで確定する。レビュアーのレスポンスやツール 呼び出し履歴は永続化しない。トランザクション前のクラッシュ/再実行ではジョブを再実行し、部分適用を残さない。

```text
accept                → reviewing_completion → completed
reject                → reviewing_completion → running
insufficient_evidence → reviewing_completion → running
```

追加証跡の到着に非同期処理が必要な場合だけ、オーナーが`WaitCondition`を登録してTaskを`waiting`へ遷移させる。

### 3.5 障害と終了

API障害、ツール障害、タイムアウト、スキーマ不適合ではレビューを確定せず、固定案に対して冪等に再試行する。案/入力 スナップショット 参照/ダイジェスト、契約/案 バージョン、再試行回数、プロファイル/Schema バージョン、リース、呼び出し 期限、最終エラーは`CompletionReviewJob`へ記録し、Agent実行としては記録しない。期限切れ`reviewing` ジョブは試行を増やし、部分レスポンスを破棄して新しい一時セッションで再実行する。上限を超えた場合はTaskを`suspended`へ遷移し、Suspension 起点を`built_in_job_failure`としてオーナー責任と案を保持する。これは受け入れ条件の`reject`ではない。

有効な判定を永続化したら一時的なAPI セッションを破棄する。組み込みAgentにはハーネス管理のリソース クリーンアップを適用しない。

## 4. ポリシーAgent

ポリシーAgentはCASB ルールエンジンのhot パスには入らず、ルール更新を判断する組み込みAgentである。主なトリガーは、拒否された不変 外向き通信の許可確認への`request_grant`と、外向き通信監査Agentが確定した検出事項である。

許可申請では`grant | deny | require_authority`を返す。CASB ポリシー マネージャーは許可確認から完全一致 一時 ルールを生成し、バージョン付きポリシーへ反映する。検出事項処理ではポリシーAgentが証跡と現行ルールを調査し、候補 ルール 文書と評価をポリシー Workspaceへ作成して`update | no_change | require_authority`を返す。ポリシー マネージャーはSchema、スコープ、衝突、回帰結果、責任者要否を検証してから新ポリシーバージョンを適用する。

ポリシーAgentの判断は確率的であり、不適切なルール更新がガードレールを弱める可能性を残余リスクとして扱う。Agentへ認証情報や本番ポリシー ストアへの直接書き込み権限は渡さない。更新の意味判断はポリシーAgentが担うが、永続化と配布はポリシー マネージャーが行う。

許可 評価 ジョブは固定入力 ダイジェスト、プロファイル/Schema バージョン、試行、リース、期限、エラーを保存する機能固有ジョブであり、Agent実行ではない。技術障害では同じ入力を再試行し、責任者へ迂回しない。

Responses APIには[built-in-agent-outputs.JSON](../schemas/draft-v0/api/built-in-agent-outputs.json)の`grant_decision`を`text.format`として指定する。CASB ポリシー マネージャーが出力をプラットフォーム上限と許可確認へ照合して一時許可を適用し、組み込みAgentへポリシー更新ツールは渡さない。

検出事項起点では`policy_revision_decision` Schemaを使用する。候補 ルールは読み取り専用 証跡ツールと隔離ポリシー Workspaceで作り、ハーネスがNULL許容 候補 参照/ダイジェスト、基底 ポリシーバージョン、固定 タイムスタンプを改訂 ジョブへ原子的に固定してから最終判断を確定する。構造化 出力に候補 参照を自己申告させない。クラッシュ/再試行ではジョブの固定値だけを使う。

`update`と`require_authority`では固定候補を必須、`no_change`では候補なしを強制する。責任者 主体はポリシーAgentに選ばせず、対象Workspaceのセキュリティ プロファイルとプラットフォーム ポリシーから解決する。

## 5. 外向き通信監査Agent

外向き通信監査Agentは、許可・拒否を含む外向き通信 試行と実際に通過した外向きトランザクションを事後レビューし、CASB ルールをすり抜けた通信を検出する組み込みAgentである。高リスク通信と異常は全件、低リスク通信はバージョン付きサンプリング ポリシーに基づく標本を対象にし、`benign | policy_bypass | suspicious | insufficient_evidence`を構造化 出力で返す。

入力は固定ウォーターマークまでのルール 判断、正規 リクエスト メタデータ、Workspaceポリシー割り当て、認証情報 スコープ、データ 分類、保存範囲と欠落範囲を示すキャプチャ マニフェストとする。保存を許可された暗号化済み キャプチャ/証跡、関連Task契約も入力に含める。未加工 認証情報は渡さない。未捕捉範囲が判断に重要なら`insufficient_evidence`を返す。一レビュー ジョブは1つの`EgressFinding`だけを原子的に確定する。ジョブごとの試行、リース、期限、キャプチャ ピン留め、エラーを保存して再試行時の重複検出事項を一意制約で防ぐ。Agent実行やレスポンス 連鎖は保存しない。critical 検出事項はTask/許可の停止候補と運用者通知を生成するが、LLM自身がポリシー ストアを直接変更しない。

検出後は検出事項をインシデント レビューへ送り、ポリシーAgentが再現証跡を追加してCASB ルール、サンプリング ポリシー、評価 データセット、回帰テストをバージョン付き`PolicyRevision`として更新する。同じ過去通信量へ新ポリシーを再実行し、改善と過剰拒否を確認してから適用する。詳細は[07-governance.md](07-governance.md)を正本とする。

Responses APIには[built-in-agent-outputs.JSON](../schemas/draft-v0/api/built-in-agent-outputs.json)の`egress_review`を`text.format`として指定する。

## 6. エピソード Agent

エピソード Agentは終端TaskのTaskエピソードを証跡から編成するMemory Planeの組み込みAgentである。Task終端イベントから冪等な編纂 ジョブを生成するが、永続化するのはジョブ状態と確定エピソードであり、内部のAgent実行やレスポンス 連鎖ではない。複数レスポンス ステップで読み取り専用 証跡DBを調査した後、Taskエピソード スキーマに準拠する構造化 出力を返す。ジョブ障害で終端Taskの状態を戻さない。詳細は[08-long-term-memory.md](08-long-term-memory.md)の「エピソード Agent」を正本とする。

## 7. Wiki Agent

Wiki AgentはTaskエピソード群から意味 Wikiを保守し、Work Agentへ挿入する記憶コンテキストを検索・構成する組み込みAgentである。Work AgentはWikiを直接検索せず、ハーネスがWiki Agentの回答を検証してコンテキストへ挿入する。Wiki更新と問合せは別ジョブとして扱い、詳細は[08-long-term-memory.md](08-long-term-memory.md)を正本とする。

## 8. 組み込みAgentではない処理

次はLLMを利用しても独立Agentとは扱わない。

- 進捗 メンテナンス レスポンス: 現在のオーナーAgentに`update_progress`だけを強制する補助レスポンス
- 外向き通信Control Plane: HTTPS/DNS プロキシ、ファイアウォール、認証情報 ブローカー、CASB ポリシー マネージャーからなる捕捉・適用境界。ポリシーの意味判断自体は確率的である
- リソース クリーンアップ マネージャー: プロセス、サーバー、ワークツリー等を停止・削除するハーネスサービス
- エピソード Agentの証跡 クエリ ツール: SQLiteへ制限付きクエリを実行する決定論的ツール
- 責任者 主体 / アダプター: Governance Planeの外部判断主体であり、ハーネス管理Agentと組み込みAgentのどちらにも含めない

特に進捗 メンテナンス レスポンスはオーナーAgentの同一Task認識を更新する処理であり、独立した判断主体や組み込みAgent ジョブを生成しない。

## 9. 実装上の分離

組み込みAgent種別ごとに、少なくとも次を別設定として管理する。

- Developer 指示とモデル プロファイル
- トリガーとハーネス生成操作 キー
- 入力スナップショット builder
- 許可ツールとデータベース ビュー
- 構造化 出力 スキーマ
- ステップ、トークン、タイムアウト、再試行上限
- 出力検証器とハーネス適用処理
- 証跡 保持と監査イベント

1つの汎用「システム Agent」へ動的promptだけを渡して全責務を兼用させない。プロファイルと権限境界を種別ごとに固定する。追跡性はAgent実行記録ではなく、機能固有の入力ダイジェスト、確定結果、その結果が参照する証跡で確保する。
