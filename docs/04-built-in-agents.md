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

組み込みAgentは作業TaskのOwnerにならず、L1・L2・L3の階層にも参加しない。Work Agent間の`delegate`、`ask`、`escalate`で生成・呼出しせず、対応する内部機能が定義済みTriggerからLLM処理を呼び出す。

組み込みAgentはHarnessのAgent管理対象ではない。Agent Registryへ登録せず、Agent ID、AgentStatus、Owner Assignment、Agent Run、Continuation、Agent Resourceを生成・永続化しない。「Agent」は固定instructions、Tool、出力Schemaを備えたLLM処理単位を表す便宜的名称である。

Work Agentのライフサイクルは[03-agent-lifecycle.md](03-agent-lifecycle.md)、Policy Agentの判定規則は[07-governance.md](07-governance.md)、Episode AgentとWiki Agentの記憶処理は[08-long-term-memory.md](08-long-term-memory.md)を併せて正本とする。

## 2. 共通実行モデル

組み込みAgentは用途ごとの内部コンポーネントがResponses APIのtool loopとして実行する。複数Response Stepを使う場合も、それはコンポーネント内部の一時的なAPI sessionであり、Harnessの`AgentRun`ではない。Episode AgentとWiki AgentはPython Memory Service内でOpenAI Agents SDKのephemeral Runnerを利用する。Acceptance ReviewerはGo Core、Policy AgentとEgress Audit AgentはRust Governance Serviceがそれぞれ所有し、Framework固有型やSessionをPlane間契約へ出さない。

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

1. Agent Registry、Agent Run、Owner Assignment、Owner排他を使用しない。
2. Trigger、入力Snapshot、Tool set、出力Schemaを機能種別ごとに固定する。
3. Work Agentの未永続化scratch contextを入力にしない。
4. 組み込みAgentはTaskやPolicy Storeを直接更新せず、呼出し元コンポーネントが出力を検証して適用する。
5. API session、Response ID、tool call履歴、step、組み込みAgent用Run Recordは永続化しない。
6. 組み込みAgentの障害と、判定対象Taskの失敗を同一視しない。
7. 永続化するのは機能固有の入力ダイジェスト、確定結果、必要なJob状態だけとする。

## 3. Acceptance Reviewer Agent

Acceptance Reviewer Agentは、Ownerが提出したCompletion Candidateを現行Task ContractのAcceptanceと照合する独立した短命Agentである。一件のCompletion Reviewに対応するReview Jobとして生成し、Taskの実装や修正、一般的なコードレビューは行わない。

### 3.1 Triggerと起動

OwnerがCompletion Candidateを提出すると、Harnessが次を行う。

1. 提出者が現在のOwnerであることを確認する
2. `contract_version`が現行版であることを確認する
3. required childが完了していることを確認する
4. Outcome、Artifact、Evidence、required child Outcomeを含むCandidate スナップショット ref/ダイジェストと、Reviewerへ渡す完全な入力 スナップショット ref/ダイジェストを固定する
5. Reviewer専用Context、Tool、Structured Output schemaでLLM処理を開始する

OwnerがReviewerのinstructionsや出力を操作できないよう、Ownerとは別の一時的なResponse chainを使う。モデルを分離するかはDeployment Policyで選択できるが、入力Snapshotと実行権限の分離は必須とする。このResponse chainとResponse IDはReview確定または失敗後に破棄する。

### 3.2 入力Context

Reviewerには次だけを渡す。

- Reviewerの責務、禁止事項、判定基準を定めたDeveloper instructions
- Task IDと現行Task Contract
- 固定済みCompletion Candidate
- required childのOutcome要約
- ArtifactおよびEvidenceの参照、ダイジェスト、取得方法
- Review対象のContract バージョンとCandidate バージョン

Reviewerは元のAcceptanceを満たすかだけを判定し、新しい要件を追加しない。

### 3.3 Evidence Tool

Reviewerには対象TaskとCandidate スナップショットに固定されたrequired descendant Evidence参照のclosureだけを公開するread-only Evidence Viewを問い合わせるToolを許可する。Evidence LayerがSQLiteの場合はparameterized `SELECT`を実行する単一Query Toolとし、base table、closure外のTask、write クエリへアクセスさせない。

外部ネットワーク、Workspace変更、Task更新、Delegate、Ask、Escalate、Completion Candidate提出のToolは渡さない。Queryには行数、応答byte数、実行時間、VM step、BLOB chunkの上限を適用する。

### 3.4 出力と適用

最終Responseは`AcceptanceReviewDecision`のStructured Outputとする。

Responses APIには[built-in-agent-outputs.json](../schemas/draft-v0/api/built-in-agent-outputs.json)の`acceptance_review`を`text.format`として指定する。判定を確定するFunction Toolは渡さない。

- `accept`: Acceptanceを満たすOutcomeと十分な`evidence_refs`がある
- `reject`: EvidenceからAcceptance未達を確認でき、`unmet_acceptance`と`evidence_refs`を特定できる
- `insufficient_evidence`: 達成・未達を判断できず、`required_evidence`と確認済み`evidence_refs`を特定できる

Completion Reviewコンポーネントが出力を検証した後、`CompletionReview` insert、`CompletionReviewJob.completed`、Task状態遷移、`CompletionReviewed` EventをTask Aggregateの同一Transactionで確定する。ReviewerのResponseやtool call履歴は永続化しない。Transaction前のcrash/replayではjobを再実行し、部分適用を残さない。

```text
accept                → reviewing_completion → completed
reject                → reviewing_completion → running
insufficient_evidence → reviewing_completion → running
```

追加Evidenceの到着に非同期処理が必要な場合だけ、Ownerが`WaitCondition`を登録してTaskを`waiting`へ遷移させる。

### 3.5 障害と終了

API障害、Tool障害、タイムアウト、schema不適合ではReviewを確定せず、固定Candidateに対して冪等に再試行する。Candidate/入力 スナップショット ref/ダイジェスト、Contract/Candidate バージョン、再試行回数、Profile/Schema バージョン、lease、invocation deadline、最終エラーは`CompletionReviewJob`へ記録し、Agent Runとしては記録しない。期限切れ`reviewing` Jobはattemptを増やし、部分Responseを破棄して新しい一時sessionで再実行する。上限を超えた場合はTaskを`suspended`へ遷移し、Suspension sourceを`built_in_job_failure`としてOwner責任とCandidateを保持する。これはAcceptanceの`reject`ではない。

有効な判定を永続化したら一時的なAPI sessionを破棄する。組み込みAgentにはHarness管理のResource Cleanupを適用しない。

## 4. Policy Agent

Policy AgentはCASB Rule Engineのhot pathには入らず、Rule更新を判断する組み込みAgentである。主なTriggerは、blockされたimmutable Egress Challengeへの`request_grant`と、Egress Audit Agentが確定したFindingである。

Grant申請では`grant | deny | require_authority`を返す。CASB Policy ManagerはChallengeからexact temporary Ruleを生成し、バージョン付きPolicyへ反映する。Finding処理ではPolicy AgentがEvidenceと現行Ruleを調査し、candidate Rule documentとEvalをPolicy Workspaceへ作成して`update | no_change | require_authority`を返す。Policy ManagerはSchema、スコープ、衝突、回帰結果、Authority要否を検証してから新Policyバージョンを適用する。

Policy Agentの判断は確率的であり、不適切なRule更新がガードレールを弱める可能性を残余riskとして扱う。AgentへCredentialや本番Policy Storeへの直接write権限は渡さない。更新の意味判断はPolicy Agentが担うが、永続化と配布はPolicy Managerが行う。

Grant Evaluation Jobは固定入力 ダイジェスト、Profile/Schema バージョン、attempt、lease、deadline、エラーを保存する機能固有Jobであり、Agent Runではない。技術障害では同じ入力を再試行し、Authorityへ迂回しない。

Responses APIには[built-in-agent-outputs.json](../schemas/draft-v0/api/built-in-agent-outputs.json)の`grant_decision`を`text.format`として指定する。CASB Policy Managerが出力をPlatform上限とChallengeへ照合してPolicy Grantを適用し、組み込みAgentへPolicy更新Toolは渡さない。

Finding起点では`policy_revision_decision` Schemaを使用する。candidate Ruleはread-only Evidence Toolと隔離Policy Workspaceで作り、Harnessがnullable candidate ref/ダイジェスト、base Policy バージョン、fixed timestampをRevision Jobへ原子的に固定してから最終Decisionを確定する。Structured Outputにcandidate refを自己申告させない。crash/再試行ではJobの固定値だけを使う。

`update`と`require_authority`では固定candidateを必須、`no_change`ではcandidateなしを強制する。Authority principalはPolicy Agentに選ばせず、対象WorkspaceのSecurity ProfileとPlatform Policyから解決する。

## 5. Egress Audit Agent

Egress Audit Agentは、許可・拒否を含むEgress Attemptと実際に通過したOutbound Transactionを事後レビューし、CASB Ruleをすり抜けた通信を検出する組み込みAgentである。high-risk通信とanomalyは全件、低risk通信はバージョン付きsampling policyに基づく標本を対象にし、`benign | policy_bypass | suspicious | insufficient_evidence`をStructured Outputで返す。

入力は固定watermarkまでのRule Decision、canonical リクエスト metadata、Workspace Policy Binding、Credential スコープ、data classification、保存範囲と欠落範囲を示すCapture Manifestとする。保存を許可されたencrypted capture/Evidence、関連Task Contractも入力に含める。raw Credentialは渡さない。未捕捉範囲が判断に重要なら`insufficient_evidence`を返す。一Review Jobは1つの`EgressFinding`だけを原子的に確定する。Jobごとのattempt、lease、deadline、capture pin、エラーを保存して再試行時の重複Findingをunique制約で防ぐ。Agent RunやResponse chainは保存しない。critical findingはTask/Grantの停止候補とOperator通知を生成するが、LLM自身がPolicy Storeを直接変更しない。

検出後はFindingをIncident reviewへ送り、Policy Agentが再現Evidenceを追加してCASB Rule、sampling policy、Eval dataset、回帰テストをバージョン付き`PolicyRevision`として更新する。同じ過去trafficへ新Policyをreplayし、改善と過剰blockを確認してから適用する。詳細は[07-governance.md](07-governance.md)を正本とする。

Responses APIには[built-in-agent-outputs.json](../schemas/draft-v0/api/built-in-agent-outputs.json)の`egress_review`を`text.format`として指定する。

## 6. Episode Agent

Episode Agentは終端TaskのTask EpisodeをEvidenceから編成するMemory Planeの組み込みAgentである。Task終端Eventから冪等なCompilation Jobを生成するが、永続化するのはJob状態と確定Episodeであり、内部のAgent RunやResponse chainではない。複数Response Stepでread-only Evidence DBを調査した後、Task Episode schemaに準拠するStructured Outputを返す。Job障害で終端Taskの状態を戻さない。詳細は[08-long-term-memory.md](08-long-term-memory.md)の「Episode Agent」を正本とする。

## 7. Wiki Agent

Wiki AgentはTask Episode群からSemantic Wikiを保守し、Work Agentへ挿入するMemory Contextを検索・構成する組み込みAgentである。Work AgentはWikiを直接検索せず、HarnessがWiki Agentの回答を検証してContextへ挿入する。Wiki更新と問合せは別Jobとして扱い、詳細は[08-long-term-memory.md](08-long-term-memory.md)を正本とする。

## 8. 組み込みAgentではない処理

次はLLMを利用しても独立Agentとは扱わない。

- Progress Maintenance Response: 現在のOwner Agentに`update_progress`だけを強制する補助Response
- Egress Control Plane: HTTPS/DNS Proxy、Firewall、Credential Broker、CASB Policy Managerからなる捕捉・適用境界。Policyの意味判断自体は確率的である
- Resource Cleanup Manager: プロセス、サーバー、worktree等を停止・削除するHarnessサービス
- Episode AgentのEvidence Query Tool: SQLiteへ制限付きQueryを実行する決定論的Tool
- Authority principal / adapter: Governance Planeの外部判断主体であり、Harness管理Agentと組み込みAgentのどちらにも含めない

特にProgress Maintenance ResponseはOwner Agentの同一Task認識を更新する処理であり、独立した判断主体や組み込みAgent Jobを生成しない。

## 9. 実装上の分離

組み込みAgent種別ごとに、少なくとも次を別設定として管理する。

- Developer instructionsとmodel profile
- TriggerとHarness生成Operation Key
- 入力Snapshot builder
- 許可ToolとDatabase View
- Structured Output schema
- step、token、タイムアウト、再試行上限
- 出力validatorとHarness適用処理
- Evidence retentionと監査Event

1つの汎用「system Agent」へ動的promptだけを渡して全責務を兼用させない。Profileと権限境界を種別ごとに固定する。追跡性はAgent Run Recordではなく、機能固有の入力ダイジェスト、確定結果、その結果が参照するEvidenceで確保する。
