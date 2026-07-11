# 実装分割・検証計画

## 1. 推奨サービス境界

```text
control-plane/
  task-manager
  run-coordinator
  mailbox-service
  async-operation-manager

execution-plane/
  sandbox-manager
  terminal-executor
  workspace-store
  evidence-store (SQLite)

governance-plane/
  effect-gateway
  policy-resolver
  policy-judge-runner
  authority-service

memory-plane/
  episode-agent-runner
  wiki-agent-runner
  wiki-repository
  memory-context-service
```

初期版は一つのApplicationでよい。Evidence LayerはSQLite、Semantic WikiはGit管理Markdownとし、境界はmoduleとして守る。Control Planeの状態DBを同じSQLiteへ置くか別DBへ置くかはdeployment規模で選べるが、Evidenceを個別ファイルへfallbackしない。

## 2. 実装順序

### Phase 1: Task Runtime

- Agent / Task / Workspace
- Owner排他
- terminal
- delegate
- Task Mailbox
- Completion Candidate + Acceptance Review
- direct child cancellation

### Phase 2: Async統一

- `timeout_ms`
- `async_id`
- async operation table
- Mailbox delivery
- idempotency / cancellation
- process restart recovery

### Phase 3: Governance

- Network egress遮断
- Effect Gateway Adapter
- payload normalization
- Policy Cascade
- independent Judge
- Authority routing
- Audit ledger

### Phase 4: Memory

- 複数StepのTask Episode Agent
- SQLite `query_evidence` Tool
- Task Episode Structured Output
- Episode Compilation Job / retry / validation
- Episodic store
- SQLite Evidence Store / FTS / backup
- Semantic Markdown repository
- Wiki Agent maintenance/query
- Harness forced injection
- context gap feedback

### Phase 5: Optimization

- multiple owner profiles
- child concurrency limit
- Response chain compaction
- background Responses
- memory evaluation questions
- cost / latency routing

Response chain compactionは、定期Progress Refresh、最小Resume Cursor、正本の再読込、旧Run停止、新Run開始を分離して実装する。後付けの完全な会話要約だけで再開しない。

## 3. Core acceptance tests

### Owner排他

1. Agent AがT1をrunning
2. T2をAへassign
3. rejectされること
4. T1がwaitingでもrejectされること
5. T1終端後はT2をassignできること

### Task creation

- AgentがDBへ直接Taskを作れない
- delegate ProposalからHarnessがID/Owner/Workspaceを確定
- Ownerまたは論理Workspace準備失敗時に中間Taskレコードを残さない
- 作成成功時の初期状態は`ready`
- duplicate idempotency keyで同じ子Taskを再生成しない
- Task graph循環を拒否

### Completion review

- Owner以外のcandidateを拒否
- stale contract versionを拒否
- required child active時に拒否
- Reviewer acceptでcompleted
- rejectでrunning
- insufficient evidenceで`running`へ戻り、OwnerがEvidenceを追加できる
- Reviewerが新要件を追加した場合をEvaluatorで検出

### Parent cancellation

- direct childはcancel可
- sibling / grandchildは直接cancel不可
- Authority決定で中間状態なくcancelledになる
- cascadeで子孫Taskもcancelledになる
- Agent Resource Cleanup失敗でTask状態が戻らない
- succeeded External Effectをrollbackしない

## 4. Async tests

| Case | 期待結果 |
|---|---|
| Toolが期限内完了 | `completed`をFunction Callへ返す |
| 期限超過 | `accepted + async_id`、処理継続 |
| 後日完了 | Mailboxへ`AsyncCompleted` |
| event重複 | event_idで一回だけ適用 |
| Harness再起動 | async tableから監視再開 |
| cancel二重呼び出し | 冪等 |
| Task終端後の結果 | orphan policyに従い監査 |
| stale contract result | Context version差をAgentへ明示 |

### Context Compaction

- Compaction前後でTask status、Owner、Contract versionが変わらない
- 旧`previous_response_id`を新Runへ引き継がない
- Current Contract / State / Mailbox / Async Operationを正本から再読込する
- Task Progressを設定Step周期でforced `update_progress`により更新する
- Maintenance Responseを通常Step数へ含めない
- Progress Refresh失敗時に直前versionを保持する
- pending async、child、Artifact、Workspaceを各正本から再読込する
- Progress未観測のAgent Run Eventsを再開Contextへ含める
- token budget超過時も未解決EventとDecision／errorを優先する
- Reasoning、opaque compaction item、Secretを再開Contextへ入れない
- Compaction中に届いたMailbox Eventを新Runが受け取る
- stale Resume CursorからRunを開始しない
- 未処理Function Callの結果を永続化するまでCompactionしない
- Resume Cursor生成失敗時に旧Runを失わない

## 5. Governance tests

### Sandbox escape

- direct network接続を拒否
- host filesystem mountを拒否
- Secret environmentが存在しない
- Effect GatewayだけがCredentialを取得

### Privilege laundering

1. ParentがEffect目的のChildをSpawn
2. ChildがEffectを要求
3. approval routeがParentへ行かないこと
4. originがParent Taskとして保持されること
5. AuthorityがPolicyから解決されること

### Payload binding

- Judge評価後にpayload変更すると実行拒否
- target identity変更で再評価
- retryが同じidempotency keyなら二重実行しない

### Judge isolation

- terminalなし
- networkなし
- request_effectなし
- Policy編集不可
- requester explanationをAuthoritative factとして混ぜない

## 6. Memory tests

### Episode boundary

- terminal commandごとにEpisodeを作らない
- Task終端時に一Episode
- completed / cancelledを保存し、suspendedではEpisodeを確定しない
- EpisodeからEvidenceへ辿れる
- observed / owner_asserted / compiler_inferredを区別
- Episode AgentがSQLとkeyset paginationでEvidenceを複数Step調査できる
- 調査中はRead-only Function Call、完了時はSchema準拠messageを返す
- Toolsと`text.format`を同時指定し、HarnessがPhase切替しない
- 未処理Function CallがあるmessageをEpisodeとして確定しない
- 存在しないsource refや不正なepistemic statusを拒否
- Compilation Job失敗でTask終端状態が戻らない
- Episode本文とsource EvidenceがSQLiteだけに保存され、個別ファイルを生成しない
- Workspace snapshot取込後にworktreeを削除してもEvidenceを読める
- Evidence metadataとBLOB保存が同一Transactionでrollbackされる
- SQLite backupからEvidence refとdigestを復元できる
- `query_evidence`がSELECT以外、DDL、PRAGMA、ATTACH、extensionを拒否する
- Compilation Jobのtask_id外のrowへアクセスできない
- query timeout、VM step、row、返却bytes、BLOB chunk上限が機能する

### Forced injection

- Task startでWiki Agentが必ず呼ばれる
- Work AgentにWiki filesystem accessがない
- ContractとMemoryが別区画
- past Objectiveを命令として混ぜない
- context gapで追加MemoryがMailboxへ届く

### Semantic Wiki

- frontmatterはkind/titleだけ
- 段落単位にEpisode link
- single Episodeから普遍則を作らない
- 反例を削除しない
- Concept / Schema / Script / Case Patternが混在しない

## 7. 障害注入

- Responses API timeout
- Background Response lost
- Run Coordinator crash after tool start / before output return
- Mailbox delivery重複
- Workspace snapshot失敗
- Reviewer unavailable
- Policy Judge unavailable
- Effect成功後、結果記録前にGateway crash
- Compaction中のMailbox Event到着
- Resume Cursor作成後・新Run開始前のContract / Progress version変更
- 未完了Function Callがある状態でのCompaction要求
- Resume Cursor参照切れまたは生成失敗

### Agent Run Record

- Streaming中断時に未完成deltaを正本化しない
- 同じresponse ID / output item ID / call IDの再配送を重複適用しない
- Tool dispatch intent commit後のcrashから再開できる
- SecretをFunction argumentsとerror detailからredactする
- Progress Maintenance Stepを通常Step countへ加算しない
- 短期Run log削除後もEpisodeの主要主張を長期Evidenceから検証できる
- Wiki commit conflict

各障害でTaskとEffectが二重実行されず、Continuationから再開できることを確認する。

## 8. Observability

### Metrics

- active Tasks by status
- owner utilization
- Task latency / wait latency
- async timeout promotion rate
- reviewer reject / insufficient evidence rate
- effect allow / deny / authority rate
- duplicate mailbox delivery rate
- episode compile lag
- memory context hit / gap rate

### Traces

Trace IDはRoot Taskから子Task、Response、async operation、Effect、Review、Episodeへ伝播する。

```text
root_task_id
  task_id
  run_id
  response_id
  call_id
  async_id
  effect_id / review_id
```

### Audit

LLM自然言語だけでなく、Contract version、payload digest、Policy bundle digest、Artifact digestを保存する。

## 9. Security invariants as code

次はPromptだけでなくRuntimeで強制する。

- owner active Task unique
- child cancellation direct relation
- no direct network from Sandbox
- no credentials in Work Agent
- Judge tool allowlist
- evaluated payload digest == executed payload digest
- completed requires accepted review
- Episode only after terminal state

## 10. 初期構成の妥協点

MVPでは次を単純化できる。

- L3/L2/L1を同じモデルでProfileだけ変更
- Task MailboxをPostgreSQL tableで実装
- Workspaceをcontainer volume + Git branchで実装
- Semantic WikiをGit repositoryで実装
- Policy Judgeを単一Runで実装
- Reviewerを単一の軽量Runで実装

ただし作業階層と統治階層、Owner排他、Completion Review、Gateway-only Effectは省略しない。
