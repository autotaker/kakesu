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
  https-interception-proxy
  dns-proxy
  firewall-manager
  baseline-policy-engine
  grant-service
  policy-grant-agent-runner
  credential-broker
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
- Harness生成Operation Key / cancellation
- process restart recovery

### Phase 3: Governance

- TaskごとのNetwork egress強制routing
- HTTPS Interception Proxy / Harness DNS Proxy
- Workspace-scoped Firewall
- versioned CASB Rule Engineによるinline allow/block
- Egress Challenge / Mailbox notification
- `request_grant` / Policy Agent
- CASB Policy Manager / Credential Broker
- Authority routing / Audit ledger
- Egress Audit Agent / Finding / Policy feedback loop
- Policy Revision Job / candidate Rule Proposal / versioned deployment

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
- 同じ`call_id`の再配送で同じ子Task Operationへ収束する
- Task graph循環を拒否

### Completion review

- Owner以外のcandidateを拒否
- stale contract versionを拒否
- required child active時に拒否
- Reviewer acceptでcompleted
- rejectでrunning
- insufficient evidenceで`running`へ戻り、OwnerがEvidenceを追加できる
- Reviewerが新要件を追加した場合をEvaluatorで検出
- Reviewer invocationでAgent Registry / Agent Run / Response ID / tool call履歴を永続化しない
- Completion Review JobはCandidate/input snapshot、digest、attempt/error/leaseを保存し、process再起動後に同じ入力を再試行する一方、Response/tool履歴は保存しない
- Reviewer出力の全`evidence_refs`をTask scopeとdigestで検証する

### Built-in agent components

- 全組み込みAgent invocationでAgent/Runレコードを生成しない
- 一時Response chainとtool call履歴を永続化しない
- 機能固有Jobのlease期限切れ後は新しいsessionで最初から再試行する
- 機能固有Job metadata、確定結果、Evidence、Profile/Schema versionで監査可能であり、Agent Run Recordへ依存しない
- Episode Jobのlease期限切れを回収し、部分Responseを使わず再調査する
- Evidence BLOB commit後・Aggregate参照確定前の障害でorphanを回収する
- Aggregateが未commitのEvidence参照を公開しない
- Control DBとEvidence SQLiteを分離した構成でterminal snapshotからEpisode viewを構築する
- Reviewerは固定済みrequired descendant Evidenceを読めるが、無関係Task Evidenceを拒否する
- Completion Review確定Transactionの各境界でcrash/replayして部分適用を残さない

### Parent cancellation

- direct childはcancel可
- sibling / grandchildは直接cancel不可
- Authority決定で中間状態なくcancelledになる
- cascadeで子孫Taskもcancelledになる
- Agent Resource Cleanup失敗でTask状態が戻らない
- 外部へ到達済みOutbound Transactionをrollbackしない

### Ask / Escalation

- Ask replyで`contract_patch`または`terminate=true`を拒否する
- Escalationへ`response_kind: advice`を返す操作を拒否する
- request Aggregate種別と`response_kind`の不一致を拒否する
- Root EscalationをRoot Authority refへ配送し、親Taskを要求しない

## 4. Async tests

| Case | 期待結果 |
|---|---|
| Toolが期限内完了 | `completed`をFunction Callへ返す |
| 期限超過 | `accepted + async_id`、処理継続 |
| 後日完了 | Mailboxへ`AsyncCompleted` |
| event重複 | event_idで一回だけ適用 |
| Harness再起動 | async tableから監視再開 |
| cancel二重呼び出し | 冪等 |
| `await_async`期限超過 | 元Operationを複製せずWait Groupの`async_id`を返す |
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
- Credential Brokerだけが実Credentialを取得
- Linux MVP Profileで全P0隔離テストを通す
- P0未実装のOS/Profileを`available`として公開しない
- macOS / Windowsを未対応のままLinux-only MVPをリリースできる
- P1/P2未実装をP0失敗として扱わない
- P1 Capabilityを要求するTaskを未実装Profileへscheduleしない
- `shared_readonly`対応を表明する各adapterでwriteを拒否する

### Privilege laundering

1. Parentがnetwork操作目的のChildをSpawn
2. Child通信がblockされChallengeを生成
3. Grant判断がParent Ownerへ行かないこと
4. Task identityとdelegation chainが保持されること
5. GrantがChild Taskだけへ束縛されること

### CASB enforcement

- HTTPS Proxy / Firewall / DNS Proxyを迂回できない
- Security PolicyをAgent/TaskではなくWorkspace network identityから解決する
- Owner交代やAgent Run再開でWorkspace Policy Bindingが変わらない
- 同じAgentでもWorkspaceが異なれば別Policy Bindingを適用する
- Workspace forkで一時GrantとAuthority approvalを継承しない
- Credential BrokerがWorkspace Bindingからscopeを解決し、Owner交代で変化しない
- Workspace Aのsentinel/approvalをWorkspace Bやfork先で使用できない
- unknown destination、raw IP、DoH/DoT、QUICをP0で拒否する
- CASB unavailable時にfail closedする
- blockした通信を外部へforwardしない
- ChallengeとMailbox Outboxをcommit後にCLIへblock responseを返す
- 同じblock retryを同一Challengeへcoalesceする
- Mailbox Event重複をevent IDで排除する
- canonical request digestへquery、Policy対象header、body、Credential scope、Policy/DNS versionを含める
- CONNECT、Upgrade、WebSocket、ambiguous framing、検査前streaming送信を拒否する
- redirectを新しいEgress Attemptとして評価する
- private/link-local/loopback/metadata/Control Plane IPとDNS rebindingを拒否する
- DNS TTL満了後の新IPを既存L4 Grantへ追加しない
- 未知FQDNをupstream DNSへ送る前にblockしてFQDN-only Challengeを作る
- P0 DNS ProxyがA/AAAA以外、過長label、高rate、高entropy queryを拒否する
- DNS query名を使ったcovert exfiltrationを拒否し、TaskとPolicy versionを監査する
- ICMP、raw socket、未対応IP protocol、IPv6 extension pathを拒否する
- inline allow/blockでLLMを呼ばず、同じbindingとPolicy versionからRule判断を再現する
- AttemptへWorkspace、Policy version、matched Rule refs、reason codesを結び付ける
- 未分類とRule conflictをdefault blockする
- Rule EngineまたはPolicy Store障害時にblockする
- compiled Rule cacheをWorkspace、canonical binding、Policy version、TTLのscope外へ再利用しない
- allow通信でもAttempt、Rule Decision、request capture、Outbound intentをcommitするまで外側connectionを作らない
- 監査ストア障害時に観測不能なforwardを行わない
- DNS upstream queryとL4 connectionにもwrite-ahead監査を適用する
- response captureをcommitするまでSandboxへresponseを返さない
- forward途中crashを`incomplete` ManifestとしてreconcileしReview対象にする
- 外部到達が不明なcrashを`outcome_unknown` Transactionとして、確定`failed`と区別してhigh-risk Reviewへ送る

### Grant workflow

- Challengeなしの`request_grant`を拒否する
- requester TaskがChallengeのWorkspaceを使用していなければ拒否する
- temporary RuleをWorkspaceへ束縛し、別Workspaceから利用できない
- Agent指定のdestination、TTL、Policy patch、Credential、idempotency keyを受け付けない
- HTTPS GrantをTask、destination、method/path、body digestへ束縛する
- L4 Grantをexact IP、port、TTL、connection limitへ束縛する
- Grant反映前に`PolicyGrantReady`を配送しない
- block済み通信を自動再生しない
- Task cancellationでGrantをrevokeする
- 並行retryでone-shot Grantを一回だけatomic reserveできる
- reserveとOutbound Transaction intentの間にcrashしても二重forwardしない
- 同じChallengeへの非終端Grant Requestを一件に制約する
- Grant/Authority待ち中のTask cancellationでRequest cancel、Grant revoke、AsyncCancelledを原子的に確定する
- TTL満了・revoke後に新規connectionと新規送信を拒否し、既存connectionをgrace period後に終了する
- Credential sentinelを外へ転送せず、scope不一致やredirect先へ実Credentialを注入しない
- `auto_grant_eligible=false`のChallengeでAgentの`grant`判断を自動適用しない
- production、data export、raw L4の必須AuthorityをLLM判断で迂回できない
- Platform PolicyがAuthorityを選び、Policy Agent出力からAuthorityを受け付けない
- Policy作成後もACKまではGrant/Request/Asyncを未完了に保つ
- ACK activation TransactionでGrant active、Request/Async completed、Ready Eventを同時確定する
- cancellationがpending Grantと未配送Ready Eventを無効化する
- Ready DispatcherがTask active、Grant active、Policy versionを再検査する
- Policy Grantからsource Grant Request/Decisionを一意に復元できる
- Authority経由Grantがapprove済みAuthority Decisionなしでは作成できない

### Authority Grant

- Authority RequestへGrant Request、Challenge、binding digest、Authority、期限を固定する
- Authorityはexact scopeのapprove/denyだけを返し、条件付きscope拡張を受け付けない
- Authority回答時にTask、Challenge、Policy/DNS freshnessを再検査する
- Policy Agent技術障害をAuthorityへ迂回しない
- Authority Decisionへ認証済みresponder principalとrationaleを一件だけ保存する
- Authority期限切れでRequest、Grant結果、Async、Mailboxを原子的に終端する
- 期限後・重複回答が既存終端結果を変更しない
- `cancel_async`がactivation前だけworkflow全体をcancelし、activation後は`not_cancellable`を返す
- `cancel_async`がpending Grantをrevokeし未配送Ready Eventを無効化する
- Workspace freeze/archive/destroyが未解決Grant/Authority/Job/Asyncをcancelし、GrantとReady Eventを無効化する
- freezeとGrant activation/Authority回答の競合でBinding activeをlock下再検査する

### Policy Agent isolation

- terminalなし
- networkなし
- Credentialなし
- Policy Store writeなし
- sanitized ChallengeとTask Contextだけを入力する
- `grant`/`deny`でquestionがnull、`require_authority`で非nullであることをvalidatorで強制する
- Policy AgentのGrant出力からscopeを受け付けず、Policy ManagerがChallengeからexact scopeを導出する

### Egress retrospective review

- high-riskとanomalyを全件、低riskをversioned random samplingで選定する
- 固定watermarkと入力digestからEgress Reviewを再現できる
- digestだけでなくPolicyに従うsanitized/encrypted captureをEvidenceとして参照できる
- Capture Manifestがtotal bytes、captured/redacted ranges、chunk digest、truncation、欠落理由、completion statusを持つ
- paddingで攻撃payloadをcapture上限外へ追い出しても`partial`と欠落範囲を認識し、必要なら`insufficient_evidence`にする
- raw CredentialがcaptureとReview Agent入力へ入らない
- allow済み通信から`policy_bypass` Findingを作成できる
- critical FindingがOperator通知とTask/Grant停止候補を生成する
- FindingからCASB Rule/Policy Agent Profile/Eval/回帰テストのRevisionを追跡できる
- Policy AgentがFindingからcandidate RuleとEvalを作り、Policy Managerだけがversioned Ruleへ反映する
- Rule改定を対象Workspaceまたはglobal baseline scopeへ明示的に束縛する
- Harness固定candidate ref/digestとbase versionだけをProposalへ確定し、Structured Outputにrefを自己申告させない
- final Decision前にcandidate ref/digest/base version/fixed timeをRevision Jobへ原子的に保存し、crash/retryで取り違えない
- `update`/`require_authority`で固定candidateを必須、`no_change`でcandidateなしにする
- Workspace scopeのProposalを別Workspaceやglobalへ拡張できない
- Revision Authority RequestへProposal/digest/scope/base version/Authority/期限を固定する
- Revision Authority Decisionへ認証済みresponderを一件だけ保存し、deny/expiry/late responseを冪等終端する
- Policy適用時にcurrent versionとbase versionをCASし、競合Proposalをstaleにする
- target Policyごとにpending Revisionを一件だけ許可し、Bindingへpending revision/versionを予約する
- new Policy versionをtarget lock下の単調sequenceで一意発番する
- Revisionをpending activationで保存し、Rule Engine ACK前は旧versionをactiveのまま保つ
- ACK後にpending revision ID一致を検査してRevision activeとWorkspace Binding current versionを原子的に切り替える
- 配布失敗・cancel・timeoutで旧active versionを維持しpending予約を解放する
- Rule配布失敗時に正本とEnforcement Pointのversionを分裂させない
- FindingとPolicy Revision Jobをjoin/FKで固定し、別WorkspaceのFindingを混入させない
- 新旧Policyを同じ過去trafficへreplayして見逃しと過剰blockを比較する
- Audit Agent停止時にbacklog、coverage低下、review latencyを検知する
- high-risk/anomalyのingest時にReview Jobとcapture pinを作り、終端までGCしない
- low-risk samplingをcapture expiry前に確定し、選定captureをpinする
- review deadline超過時にretention延長、容量制御、coverage gap記録とOperator通知を行う
- 一Review JobからFindingを一件だけatomic finalizeし、retryで重複しない
- random sample、anomaly、red-team replayを組み合わせ、同一LLMだけに依存しない

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
- Policy Agent unavailable
- Policy反映後・GrantReady配送前にPolicy Manager crash
- HTTPS Proxy block後・Mailbox配送前にOutbox worker crash
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

各障害でTask、Grant Request、Mailbox Eventが二重適用されず、Continuationから再開できることを確認する。

## 8. Observability

### Metrics

- active Tasks by status
- owner utilization
- Task latency / wait latency
- async timeout promotion rate
- reviewer reject / insufficient evidence rate
- egress allow / block rate
- grant / deny / authority rate
- duplicate mailbox delivery rate
- episode compile lag
- memory context hit / gap rate

### Traces

Work Agent traceはRoot Taskから子Task、Response、async operation、Egress Challenge、Grantへ伝播する。組み込みAgentではprovider Response IDやcall IDを永続traceへ含めず、機能固有の`review_id`、`challenge_id`、`grant_id`と入力digestを使う。

```text
root_task_id
  task_id
  run_id
  response_id
  call_id
  async_id
  challenge_id / grant_id

built_in_operation
  review_id / job_id
  input_digest
```

### Audit

LLM自然言語だけでなく、Contract version、canonical request digest、Challenge binding digest、Policy version、Artifact digestを保存する。

## 9. Security invariants as code

次はPromptだけでなくRuntimeで強制する。

- owner active Task unique
- child cancellation direct relation
- no direct network from Sandbox
- no credentials in Work Agent
- Built-in Agent tool allowlist
- retry request binding == immutable Challenge binding == Policy Grant binding
- Grant use reservationとOutbound Transaction intentのatomic commit
- completed requires accepted review
- Episode only after terminal state

## 10. 初期構成の妥協点

MVPでは次を単純化できる。

- L3/L2/L1を同じモデルでProfileだけ変更
- Task MailboxをPostgreSQL tableで実装
- Workspaceをcontainer volume + Git branchで実装
- Semantic WikiをGit repositoryで実装
- Policy Agentを単一の一時API sessionで実装
- ReviewerをOwner Agent Runから分離した単一の一時API sessionで実装

ただし作業階層と統治階層、Owner排他、Completion Review、Egress Control Planeの強制境界は省略しない。
