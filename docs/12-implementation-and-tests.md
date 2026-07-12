# 実装分割・検証計画

## 1. 推奨サービス境界

```text
core/ (Go)
  cli
  control-plane/
    task-manager
    run-coordinator
    mailbox-service
    async-operation-manager
    authority-gateway
  work-agent-plane/
    responses-adapter
    context-builder
    action-dispatcher
  execution-plane/
    sandbox-manager
    terminal-executor
    workspace-store
  control.db

memory/ (Python + OpenAI Agents SDK)
  episode-agent-runner
  wiki-agent-runner
  wiki-repository
  memory-context-service
  evidence.db

governance/ (Rust)
  https-interception-proxy
  dns-proxy
  firewall-manager
  baseline-policy-engine
  grant-service
  policy-agent-runner
  credential-broker
  governance.db
```

利用者からは1つのローカルCLI applicationとして提供するが、MemoryとGovernanceは独立プロセスにする。Core、Memory、Governanceはそれぞれ所有SQLiteへだけwriteし、Outbox/InboxとUnix domain socketで非同期メッセージを配送する。Evidenceを個別ファイルへfallbackしない。技術選定と再評価条件は[13-technology-stack.md](13-technology-stack.md)を正本とする。

## 2. 実装順序

### Phase 1: Task Runtime

- Go Coreのサービス run groupとgraceful shutdown
- `control.db` migration、Core Inbox/Outbox
- UDS メッセージ transportとSchema validation
- Agent / Task / Workspace
- Owner排他
- terminal
- `delegate`
- Task Mailbox
- Completion Candidate + Acceptance Review
- direct child cancellation

### Phase 2: Async統一

- `timeout_ms`
- `async_id`
- async operation table
- Mailbox delivery
- Harness生成Operation Key / cancellation
- プロセス restart recovery

### Phase 3: Governance

- Rust Governance Serviceと`governance.db`
- Governance Inbox/Outbox、Controlとの`ACK`/reconciliation
- TaskごとのNetwork egress強制routing
- HTTPS Interception Proxy / Harness DNS Proxy
- WorkspaceスコープのFirewall
- バージョン付きCASB Rule Engineによるinline allow/block
- Egress Challenge / Mailbox notification
- `request_grant` / Policy Agent
- CASB Policy Manager / Credential Broker
- Authority routing / Audit ledger
- Egress Audit Agent / Finding / Policy feedback loop
- Policy Revision Job / candidate Rule Proposal / バージョン付きdeployment

### Phase 4: Memory

- Python Memory Serviceと`evidence.db`
- OpenAI Agents SDKのephemeral Runner adapter
- Memory Inbox/Outbox、Coreとのterminal スナップショット transfer
- 複数StepのTask Episode Agent
- SQLite `query_evidence` Tool
- Task Episode Structured Output
- Episode Compilation Job / 再試行 / validation
- Episodic store
- SQLite Evidence Store / FTS / backup
- Semantic Markdown リポジトリ
- Wiki Agent maintenance/クエリ
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

1. Agent AがT1を`running`
2. T2をAへassign
3. rejectされること
4. T1が`waiting`でもrejectされること
5. T1終端後はT2をassignできること

### Task creation

- AgentがDBへ直接Taskを作れない
- `delegate` ProposalからHarnessがID/Owner/Workspaceを確定
- Ownerまたは論理Workspace準備失敗時に中間Taskレコードを残さない
- 作成成功時の初期状態は`ready`
- 同じ`call_id`の再配送で同じ子Task Operationへ収束する
- Task graph循環を拒否

### Completion review

- Owner以外のcandidateを拒否
- stale contract バージョンを拒否
- required child active時に拒否
- Reviewer acceptで`completed`
- rejectで`running`
- insufficient evidenceで`running`へ戻り、OwnerがEvidenceを追加できる
- Reviewerが新要件を追加した場合をEvaluatorで検出
- Reviewer invocationでAgent Registry / Agent Run / Response ID / tool call履歴を永続化しない
- Completion Review JobはCandidate/入力 スナップショット、ダイジェスト、attempt/エラー/leaseを保存し、プロセス再起動後に同じ入力を再試行する一方、Response/tool履歴は保存しない
- Reviewer出力の全`evidence_refs`をTask スコープとダイジェストで検証する

### Built-in Agentコンポーネント

- 全組み込みAgent invocationでAgent/Runレコードを生成しない
- 一時Response chainとtool call履歴を永続化しない
- 機能固有Jobのlease期限切れ後は新しいsessionで最初から再試行する
- 機能固有Job metadata、確定結果、Evidence、Profile/Schema バージョンで監査可能であり、Agent Run Recordへ依存しない
- Episode Jobのlease期限切れを回収し、部分Responseを使わず再調査する
- Evidence BLOB コミット後・Aggregate参照確定前の障害でorphanを回収する
- Aggregateが未コミットのEvidence参照を公開しない
- Control DBとEvidence SQLiteを分離した構成でterminal スナップショットからEpisode viewを構築する
- Memory SDK Session、trace、checkpointをJob recoveryの正本にしない
- Memory Inboxへの重複配送を`message_id`で1回だけ適用する
- Reviewerは固定済みrequired descendant Evidenceを読めるが、無関係Task Evidenceを拒否する
- Completion Review確定Transactionの各境界でcrash/replayして部分適用を残さない

### Parent cancellation

- direct childはcancel可
- sibling / grandchildは直接cancel不可
- Authority決定で中間状態なく`cancelled`になる
- cascadeで子孫Taskも`cancelled`になる
- Agent Resource Cleanup失敗でTask状態が戻らない
- 外部へ到達済みOutbound Transactionをrollbackしない

### Ask / Escalation

- Ask replyで`contract_patch`または`terminate=true`を拒否する
- Escalationへ`response_kind: advice`を返す操作を拒否する
- リクエスト Aggregate種別と`response_kind`の不一致を拒否する
- Root EscalationをRoot Authority refへ配送し、親Taskを要求しない

## 4. Async tests

| Case | 期待結果 |
|---|---|
| Toolが期限内完了 | `completed`をFunction Callへ返す |
| 期限超過 | `accepted + async_id`、処理継続 |
| 後日完了 | Mailboxへ`AsyncCompleted` |
| イベント重複 | イベント_idで一回だけ適用 |
| Harness再起動 | async tableから監視再開 |
| cancel二重呼び出し | 冪等 |
| `await_async`期限超過 | 元Operationを複製せずWait Groupの`async_id`を返す |
| Task終端後の結果 | orphan policyに従い監査 |
| stale contract result | Context バージョン差をAgentへ明示 |

### Context Compaction

- Compaction前後でTask status、Owner、Contract バージョンが変わらない
- 旧`previous_response_id`を新Runへ引き継がない
- Current Contract / State / Mailbox / Async Operationを正本から再読込する
- Task Progressを設定Step周期でforced `update_progress`により更新する
- Maintenance Responseを通常Step数へ含めない
- Progress Refresh失敗時に直前バージョンを保持する
- pending async、child、Artifact、Workspaceを各正本から再読込する
- Progress未観測のAgent Run Eventsを再開Contextへ含める
- token budget超過時も未解決EventとDecision／エラーを優先する
- Reasoning、opaque compaction item、Secretを再開Contextへ入れない
- Compaction中に届いたMailbox Eventを新Runが受け取る
- stale Resume CursorからRunを開始しない
- 未処理Function Callの結果を永続化するまでCompactionしない
- Resume Cursor生成失敗時に旧Runを失わない

## 5. Governance tests

### Sandbox escape

- direct ネットワーク接続を拒否
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

1. Parentがネットワーク操作目的のChildをSpawn
2. Child通信がblockされChallengeを生成
3. Grant判断がParent Ownerへ行かないこと
4. Task identityとdelegation chainが保持されること
5. GrantがChild Taskだけへ束縛されること

### CASB enforcement

- HTTPS Proxy / Firewall / DNS Proxyを迂回できない
- Security PolicyをAgent/TaskではなくWorkspace ネットワーク identityから解決する
- Owner交代やAgent Run再開でWorkspace Policy Bindingが変わらない
- 同じAgentでもWorkspaceが異なれば別Policy Bindingを適用する
- Workspace forkで一時GrantとAuthority approvalを継承しない
- Credential BrokerがWorkspace Bindingからスコープを解決し、Owner交代で変化しない
- Workspace Aのsentinel/approvalをWorkspace Bやfork先で使用できない
- unknown destination、raw IP、DoH/DoT、QUICをP0で拒否する
- CASB unavailable時にfail closedする
- blockした通信を外部へforwardしない
- Governance Outbox コミット後にCLIへblock レスポンスを返し、Control Inbox再送後もTask Mailboxを重複させない
- 同じblock 再試行を同一Challengeへcoalesceする
- Mailbox Event重複をイベント IDで排除する
- canonical リクエスト ダイジェストへクエリ、Policy対象header、body、Credential スコープ、Policy/DNS バージョンを含める
- CONNECT、Upgrade、WebSocket、ambiguous framing、検査前streaming送信を拒否する
- redirectを新しいEgress Attemptとして評価する
- private/link-local/loopback/metadata/Control Plane IPとDNS rebindingを拒否する
- DNS TTL満了後の新IPを既存L4 Grantへ追加しない
- 未知FQDNをupstream DNSへ送る前にblockしてFQDN-only Challengeを作る
- P0 DNS ProxyがA/AAAA以外、過長label、高rate、高entropy クエリを拒否する
- DNS クエリ名を使ったcovert exfiltrationを拒否し、TaskとPolicy バージョンを監査する
- ICMP、raw socket、未対応IP プロトコル、IPv6 extension pathを拒否する
- inline allow/blockでLLMを呼ばず、同じbindingとPolicy バージョンからRule判断を再現する
- AttemptへWorkspace、Policy バージョン、matched Rule refs、reason codesを結び付ける
- 未分類とRule conflictをdefault blockする
- Rule EngineまたはPolicy Store障害時にblockする
- compiled Rule cacheをWorkspace、canonical binding、Policy バージョン、TTLのスコープ外へ再利用しない
- allow通信でもAttempt、Rule Decision、リクエスト capture、Outbound intentをコミットするまで外側接続を作らない
- 監査ストア障害時に観測不能なforwardを行わない
- DNS upstream クエリとL4 接続にもwrite-ahead監査を適用する
- レスポンス captureをコミットするまでSandboxへレスポンスを返さない
- forward途中crashを`incomplete` ManifestとしてreconcileしReview対象にする
- 外部到達が不明なcrashを`outcome_unknown` Transactionとして、確定`failed`と区別してhigh-risk Reviewへ送る

### Grant ワークフロー

- Challengeなしの`request_grant`を拒否する
- 要求元TaskがChallengeのWorkspaceを使用していなければ拒否する
- temporary RuleをWorkspaceへ束縛し、別Workspaceから利用できない
- Agent指定のdestination、TTL、Policy patch、Credential、idempotency keyを受け付けない
- HTTPS GrantをTask、destination、method/path、body ダイジェストへ束縛する
- L4 Grantをexact IP、port、TTL、接続 limitへ束縛する
- Grant反映前に`PolicyGrantReady`を配送しない
- block済み通信を自動再生しない
- Task cancellationでGrantをrevokeする
- 並行再試行でone-shot Grantを1回だけatomic reserveできる
- reserveとOutbound Transaction intentの間にcrashしても二重forwardしない
- 同じChallengeへの非終端Grant Requestを一件に制約する
- Grant/Authority待ち中のTask cancellationでCore Action gateを先に閉じ、Governance revoke `ACK`後にAsyncCancelledへ一度だけ収束する
- TTL満了・revoke後に新規接続と新規送信を拒否し、既存接続をgrace period後に終了する
- Credential sentinelを外へ転送せず、スコープ不一致やredirect先へ実Credentialを注入しない
- `auto_grant_eligible=false`のChallengeでAgentの`grant`判断を自動適用しない
- production、data export、raw L4の必須AuthorityをLLM判断で迂回できない
- Platform PolicyがAuthorityを選び、Policy Agent出力からAuthorityを受け付けない
- Policy作成後も`ACK`まではGrant/Request/Asyncを未完了に保つ
- Governance activation TransactionでGrant activeと結果Outboxを確定し、Control Inbox適用TransactionでRequest/Async `completed`とReady Eventを同時確定する
- cancellationがpending Grantと未配送Ready Eventを無効化する
- Ready DispatcherがTask active、Grant active、Policy バージョンを再検査する
- Policy Grantからsource Grant Request/Decisionを一意に復元できる
- Authority経由Grantがapprove済みAuthority Decisionなしでは作成できない

### Authority Grant

- Authority RequestへGrant Request、Challenge、binding ダイジェスト、Authority、期限を固定する
- Authorityはexact スコープのapprove/denyだけを返し、条件付きスコープ拡張を受け付けない
- Authority回答時にTask、Challenge、Policy/DNS freshnessを再検査する
- Policy Agent技術障害をAuthorityへ迂回しない
- Authority Decisionへ認証済みresponder principalとrationaleを一件だけ保存する
- Authority期限切れでControl Decision、Governance deny、Control Async/Mailbox終端を冪等メッセージ chainとして収束させる
- 期限後・重複回答が既存終端結果を変更しない
- `cancel_async`がactivation前だけワークフロー全体をcancelし、activation後は`not_cancellable`を返す
- `cancel_async`がpending Grantをrevokeし未配送Ready Eventを無効化する
- Workspace freeze/archive/destroyが先にCore Action gateを閉じ、Governance `ACK`までcleanup完了にせず、GrantとReady Eventを無効化する
- freezeとGrant activation/Authority回答の競合でBinding activeをlock下再検査する

### Policy Agent isolation

- terminalなし
- ネットワークなし
- Credentialなし
- Policy Store writeなし
- sanitized ChallengeとTask Contextだけを入力する
- `grant`/`deny`でquestionがnull、`require_authority`で非nullであることをvalidatorで強制する
- Policy AgentのGrant出力からスコープを受け付けず、Policy ManagerがChallengeからexact スコープを導出する

### Egress retrospective review

- high-riskとanomalyを全件、低riskをバージョン付きrandom samplingで選定する
- 固定watermarkと入力ダイジェストからEgress Reviewを再現できる
- ダイジェストだけでなくPolicyに従うsanitized/encrypted captureをEvidenceとして参照できる
- Capture Manifestがtotal bytes、captured/redacted ranges、chunk ダイジェスト、truncation、欠落理由、completion statusを持つ
- paddingで攻撃ペイロードをcapture上限外へ追い出しても`partial`と欠落範囲を認識し、必要なら`insufficient_evidence`にする
- raw CredentialがcaptureとReview Agent入力へ入らない
- allow済み通信から`policy_bypass` Findingを作成できる
- critical FindingがOperator通知とTask/Grant停止候補を生成する
- FindingからCASB Rule/Policy Agent Profile/Eval/回帰テストのRevisionを追跡できる
- Policy AgentがFindingからcandidate RuleとEvalを作り、Policy Managerだけがバージョン付きRuleへ反映する
- Rule改定を対象Workspaceまたはglobal baseline スコープへ明示的に束縛する
- Harness固定candidate ref/ダイジェストとbase バージョンだけをProposalへ確定し、Structured Outputにrefを自己申告させない
- final Decision前にcandidate ref/ダイジェスト/base バージョン/fixed timeをRevision Jobへ原子的に保存し、crash/再試行で取り違えない
- `update`/`require_authority`で固定candidateを必須、`no_change`でcandidateなしにする
- Workspace スコープのProposalを別Workspaceやglobalへ拡張できない
- Revision Authority RequestへProposal/ダイジェスト/スコープ/base バージョン/Authority/期限を固定する
- Revision Authority Decisionへ認証済みresponderを一件だけ保存し、deny/expiry/late レスポンスを冪等終端する
- Policy適用時にcurrent バージョンとbase バージョンをCASし、競合Proposalをstaleにする
- target Policyごとにpending Revisionを一件だけ許可し、Bindingへpending revision/バージョンを予約する
- new Policy バージョンをtarget lock下の単調sequenceで一意発番する
- Revisionをpending activationで保存し、Rule Engine `ACK`前は旧バージョンをactiveのまま保つ
- `ACK`後にpending revision ID一致を検査してRevision activeとWorkspace Binding current バージョンを原子的に切り替える
- 配布失敗・cancel・タイムアウトで旧active バージョンを維持しpending予約を解放する
- Rule配布失敗時に正本とEnforcement Pointのバージョンを分裂させない
- FindingとPolicy Revision Jobをjoin/FKで固定し、別WorkspaceのFindingを混入させない
- 新旧Policyを同じ過去trafficへreplayして見逃しと過剰blockを比較する
- Audit Agent停止時にbacklog、coverage低下、review latencyを検知する
- high-risk/anomalyのingest時にReview Jobとcapture pinを作り、終端までGCしない
- low-risk samplingをcapture expiry前に確定し、選定captureをpinする
- review deadline超過時にretention延長、容量制御、coverage gap記録とOperator通知をする
- 一Review JobからFindingを一件だけatomic finalizeし、再試行で重複しない
- random sample、anomaly、red-team replayを組み合わせ、同一LLMだけに依存しない

## 6. Memory tests

### Episode boundary

- terminal commandごとにEpisodeを作らない
- Task終端時に一Episode
- `completed` / `cancelled`を保存し、`suspended`ではEpisodeを確定しない
- EpisodeからEvidenceへ辿れる
- observed / owner_asserted / compiler_inferredを区別
- Episode AgentがSQLとkeyset paginationでEvidenceを複数Step調査できる
- 調査中はRead-only Function Call、完了時はSchema準拠メッセージを返す
- Toolsと`text.format`を同時指定し、HarnessがPhase切替しない
- 未処理Function CallがあるメッセージをEpisodeとして確定しない
- 存在しないsource refや不正なepistemic statusを拒否
- Compilation Job失敗でTask終端状態が戻らない
- Episode本文とsource EvidenceがSQLiteだけに保存され、個別ファイルを生成しない
- Workspace スナップショット取込後にworktreeを削除してもEvidenceを読める
- Evidence metadataとBLOB保存が同一Transactionでrollbackされる
- SQLite backupからEvidence refとダイジェストを復元できる
- `query_evidence`がSELECT以外、DDL、PRAGMA、ATTACH、extensionを拒否する
- Compilation JobのTask_id外のrowへアクセスできない
- クエリ タイムアウト、VM step、row、返却bytes、BLOB chunk上限が機能する

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

- Responses API タイムアウト
- Background Response lost
- Run Coordinator crash after tool start / before 出力 return
- Mailbox delivery重複
- Workspace スナップショット失敗
- Reviewer unavailable
- Policy Agent unavailable
- Policy反映後・GrantReady配送前にPolicy Manager crash
- HTTPS Proxy block後・Mailbox配送前にOutbox worker crash
- Compaction中のMailbox Event到着
- Resume Cursor作成後・新Run開始前のContract / Progress バージョン変更
- 未完了Function Callがある状態でのCompaction要求
- Resume Cursor参照切れまたは生成失敗

### Agent Run Record

- Streaming中断時に未完成deltaを正本化しない
- 同じレスポンス ID / 出力 item ID / call IDの再配送を重複適用しない
- Tool dispatch intent コミット後のcrashから再開できる
- SecretをFunction argumentsとエラー detailからredactする
- Progress Maintenance Stepを通常Step countへ加算しない
- 短期Run log削除後もEpisodeの主要主張を長期Evidenceから検証できる
- Wiki コミット conflict

各障害でTask、Grant Request、Mailbox Eventが二重適用されず、Continuationから再開できることを確認する。

## 8. Observability

### Metrics

- active Tasks by status
- owner utilization
- Task latency / wait latency
- async タイムアウト promotion rate
- reviewer reject / insufficient evidence rate
- egress allow / block rate
- grant / deny / authority rate
- duplicate mailbox delivery rate
- episode compile lag
- memory context hit / gap rate

### Traces

Work Agent traceはRoot Taskから子Task、Response、async operation、Egress Challenge、Grantへ伝播する。組み込みAgentではprovider Response IDやcall IDを永続traceへ含めず、機能固有の`review_id`、`challenge_id`、`grant_id`と入力ダイジェストを使う。

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

LLM自然言語だけでなく、Contract バージョン、canonical リクエスト ダイジェスト、Challenge binding ダイジェスト、Policy バージョン、Artifact ダイジェストを保存する。

## 9. Security invariants as code

次はPromptだけでなくRuntimeで強制する。

- owner active Task unique
- child cancellation direct relation
- no direct ネットワーク from Sandbox
- no credentials in Work Agent
- Built-in Agent tool allowlist
- 再試行 リクエスト binding == immutable Challenge binding == Policy Grant binding
- Grant use reservationとOutbound Transaction intentのatomic コミット
- `completed` requires accepted review
- Episode only after terminal state

## 10. 初期構成の妥協点

MVPでは次を単純化できる。

- L3/L2/L1を同じモデルでProfileだけ変更
- Task Mailboxを`control.db`のSQLite tableで実装
- Workspaceをcontainer volume + Git ブランチで実装
- Semantic WikiをGit リポジトリで実装
- Policy Agentを単一の一時API sessionで実装
- ReviewerをOwner Agent Runから分離した単一の一時API sessionで実装
- Plane間transportをUnix domain socket + JSONに限定し、Brokerを導入しない

ただし作業階層と統治階層、Owner排他、Completion Review、Egress Control Planeの強制境界は省略しない。
