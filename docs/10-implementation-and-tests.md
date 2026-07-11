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
  artifact-store

governance-plane/
  effect-gateway
  policy-resolver
  policy-judge-runner
  authority-service

memory-plane/
  episode-compiler
  wiki-agent-runner
  wiki-repository
  memory-context-service
```

初期版は一つのApplicationと一つのPostgreSQLでもよい。境界はmoduleとして守る。

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

- Task Episode Compiler
- Episodic store
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
- duplicate idempotency keyで同じ子Taskを再生成しない
- Task graph循環を拒否

### Completion review

- Owner以外のcandidateを拒否
- stale contract versionを拒否
- required child active時に拒否
- Reviewer acceptでcompleted
- rejectでrunning
- insufficient evidenceでwaiting_evidence
- Reviewerが新要件を追加した場合をEvaluatorで検出

### Parent cancellation

- direct childはcancel可
- sibling / grandchildは直接cancel不可
- cascadeで子孫が停止
- grace period後にforced cancel
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
- completed / failed / cancelledすべて保存
- EpisodeからEvidenceへ辿れる
- observed / owner_asserted / compiler_inferredを区別

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
