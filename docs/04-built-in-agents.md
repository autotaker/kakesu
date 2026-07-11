# 組み込みAgent設計

## 1. 対象と位置づけ

本書は、L1・L2・L3のWork Agentとは独立した内部機能として実装する組み込みAgent群を定義する。

```text
Work Agent                  Harness Built-in Agent
-------------------------   ------------------------------
L1 / L2 / L3                Acceptance Reviewer Agent
Taskの単一Owner             Policy Judge Agent
Objective達成に責任を持つ   Episode Agent
delegate / ask / escalate   Wiki Agent
```

組み込みAgentは作業TaskのOwnerにならず、L1・L2・L3の階層にも参加しない。Work Agent間の`delegate`、`ask`、`escalate`で生成・呼出しせず、対応する内部機能が定義済みTriggerからLLM処理を呼び出す。

組み込みAgentはHarnessのAgent管理対象ではない。Agent Registryへ登録せず、Agent ID、AgentStatus、Owner Assignment、Agent Run、Continuation、Agent Resourceを生成・永続化しない。「Agent」は固定instructions、Tool、出力Schemaを備えたLLM処理単位を表す便宜的名称である。

Work Agentのライフサイクルは[03-agent-lifecycle.md](03-agent-lifecycle.md)、Policy Judgeの判定規則は[07-governance.md](07-governance.md)、Episode AgentとWiki Agentの記憶処理は[08-long-term-memory.md](08-long-term-memory.md)を併せて正本とする。

## 2. 共通実行モデル

組み込みAgentは用途ごとの内部コンポーネントがResponses APIのtool loopとして実行する。複数Response Stepを使う場合も、それはコンポーネント内部の一時的なAPI sessionであり、Harnessの`AgentRun`ではない。

```typescript
type BuiltInAgentKind =
  | "acceptance_reviewer"
  | "policy_judge"
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
4. 組み込みAgentはTaskやEffectを直接更新せず、呼出し元コンポーネントが出力を検証して適用する。
5. API session、Response ID、tool call履歴、step、組み込みAgent用Run Recordは永続化しない。
6. 組み込みAgentの障害と、判定対象Taskの失敗を同一視しない。
7. 永続化するのは機能固有の入力digest、確定結果、必要なJob状態だけとする。

## 3. Acceptance Reviewer Agent

Acceptance Reviewer Agentは、Ownerが提出したCompletion Candidateを現行Task ContractのAcceptanceと照合する独立した短命Agentである。一件のCompletion Reviewに対応するReview Jobとして生成し、Taskの実装や修正、一般的なコードレビューは行わない。

### 3.1 Triggerと起動

OwnerがCompletion Candidateを提出すると、Harnessが次を行う。

1. 提出者が現在のOwnerであることを確認する
2. `contract_version`が現行版であることを確認する
3. required childが完了していることを確認する
4. Outcome、Artifact、Evidence参照を解決してCandidateのdigestを固定する
5. Reviewer専用Context、Tool、Structured Output schemaでLLM処理を開始する

OwnerがReviewerのinstructionsや出力を操作できないよう、Ownerとは別の一時的なResponse chainを使う。モデルを分離するかはDeployment Policyで選択できるが、入力Snapshotと実行権限の分離は必須とする。このResponse chainとResponse IDはReview確定または失敗後に破棄する。

### 3.2 入力Context

Reviewerには次だけを渡す。

- Reviewerの責務、禁止事項、判定基準を定めたDeveloper instructions
- Task IDと現行Task Contract
- 固定済みCompletion Candidate
- required childのOutcome要約
- ArtifactおよびEvidenceの参照、digest、取得方法
- Review対象のContract versionとCandidate version

Reviewerは元のAcceptanceを満たすかだけを判定し、新しい要件を追加しない。

### 3.3 Evidence Tool

Reviewerには対象Taskへ固定されたread-only Evidence Viewを問い合わせるToolだけを許可する。Evidence LayerがSQLiteの場合はparameterized `SELECT`を実行する単一Query Toolとし、base table、他Task、write queryへアクセスさせない。

Effect、Workspace変更、Task更新、Delegate、Ask、Escalate、Completion Candidate提出のToolは渡さない。Queryには行数、応答byte数、実行時間、VM step、BLOB chunkの上限を適用する。

### 3.4 出力と適用

最終Responseは`AcceptanceReviewDecision`のStructured Outputとする。

- `accept`: Acceptanceを満たすOutcomeと十分なEvidenceがある
- `reject`: EvidenceからAcceptance未達を確認でき、`unmetAcceptance`を特定できる
- `insufficient_evidence`: 達成・未達を判断できず、`requiredEvidence`を特定できる

Completion Reviewコンポーネントが出力を検証して`CompletionReview`を永続化し、HarnessのTask遷移APIへ確定結果を渡す。ReviewerのResponseやtool call履歴は永続化しない。

```text
accept                → reviewing_completion → completed
reject                → reviewing_completion → running
insufficient_evidence → reviewing_completion → running
```

追加Evidenceの到着に非同期処理が必要な場合だけ、Ownerが`WaitCondition`を登録してTaskを`waiting`へ遷移させる。

### 3.5 障害と終了

API障害、Tool障害、timeout、schema不適合ではReviewを確定せず、固定Candidateに対して冪等に再試行する。再試行回数と最終errorはReview Request側に記録し、Agent Runとしては記録しない。上限を超えた場合はTaskを`suspended`へ遷移し、Owner責任とCandidateを保持する。これはAcceptanceの`reject`ではない。

有効な判定を永続化したら一時的なAPI sessionを破棄する。組み込みAgentにはHarness管理のResource Cleanupを適用しない。

## 4. Policy Judge Agent

Policy Judge AgentはExternal Effect要求をPolicy Cascadeと照合する統治Planeの組み込みAgentである。Work Agentや親Ownerから独立し、`allow | deny | require_approval`と根拠をStructured Outputで返す。Credentialを取得せず、Effectを実行しない。Harnessが検証済み判定をEffect Gatewayへ渡す。詳細は[07-governance.md](07-governance.md)を正本とする。

## 5. Episode Agent

Episode Agentは終端TaskのTask EpisodeをEvidenceから編成するMemory Planeの組み込みAgentである。Task終端Eventから冪等なCompilation Jobを生成するが、永続化するのはJob状態と確定Episodeであり、内部のAgent RunやResponse chainではない。複数Response Stepでread-only Evidence DBを調査した後、Task Episode schemaに準拠するStructured Outputを返す。Job障害で終端Taskの状態を戻さない。詳細は[08-long-term-memory.md](08-long-term-memory.md)の「Episode Agent」を正本とする。

## 6. Wiki Agent

Wiki AgentはTask Episode群からSemantic Wikiを保守し、Work Agentへ挿入するMemory Contextを検索・構成する組み込みAgentである。Work AgentはWikiを直接検索せず、HarnessがWiki Agentの回答を検証してContextへ挿入する。Wiki更新と問合せは別Jobとして扱い、詳細は[08-long-term-memory.md](08-long-term-memory.md)を正本とする。

## 7. 組み込みAgentではない処理

次はLLMを利用しても独立Agentとは扱わない。

- Progress Maintenance Response: 現在のOwner Agentに`update_progress`だけを強制する補助Response
- Effect Gateway: Credentialを保持して検証済みEffectを実行する決定論的サービス
- Resource Cleanup Manager: process、server、worktree等を停止・削除するHarnessサービス
- Episode AgentのEvidence Query Tool: SQLiteへ制限付きQueryを実行する決定論的Tool

特にProgress Maintenance ResponseはOwner Agentの同一Task認識を更新する処理であり、独立した判断主体や組み込みAgent Jobを生成しない。

## 8. 実装上の分離

組み込みAgent種別ごとに、少なくとも次を別設定として管理する。

- Developer instructionsとmodel profile
- Triggerとidempotency key
- 入力Snapshot builder
- 許可ToolとDatabase View
- Structured Output schema
- step、token、timeout、retry上限
- 出力validatorとHarness適用処理
- Evidence retentionと監査Event

一つの汎用「system agent」へ動的promptだけを渡して全責務を兼用させない。Profileと権限境界を種別ごとに固定する。追跡性はAgent Run Recordではなく、機能固有の入力digest、確定結果、その結果が参照するEvidenceで確保する。
