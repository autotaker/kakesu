# E2E Tabletop Debugging 独立レビュー

## 最終判定

**PASS**。Schema reviewer、sequence reviewerの双方が、対象4シナリオを無条件PASSと判定した。

| 検査対象 | 結果 |
|---|---:|
| E2E Scenario | 4 PASS |
| Sequence Payload | 124 PASS |
| Canonical domain Payload | 119 PASS |
| Sequence Requirement | 4 PASS |
| Negative mutation | 11 reject |
| Idempotent redelivery | PASS |
| Nested correlation | PASS |

## 初回レビューで発見した不足と解消

| 発見 | 解消内容 |
|---|---|
| GrantがChallengeからactiveへ直行 | Request、Decision、pending activation、Rule Engine `ACK`、active、Ready、CLI 再試行を追加 |
| ToolCallとEgressの因果が不明 | Tool call、プロセス、attempt、challenge、grant、transactionをID結合 |
| Review後のTask terminal確定がない | ReviewingCompletion、Review Input/Output、`TaskCompleted`、Episode Inputを追加 |
| Child Task生成・親統合が不完全 | Child Workspace/Task、Child Episode、Mailbox consume、Parent Integration/Review/Episodeを追加 |
| Asyncが`running`のまま完了通知 | Async `completed`永続化、Mailbox consume、ResumeContext、新Run、TaskResumedを追加 |
| Policy Revisionが`ACK`前にactive | Candidate、Regression、Authority、pending revision、`ACK`、activeを分離 |
| Requirementが宣言だけでvacuous pass | 必須type、順序、フィールド join、direct causationを各Scenarioへ適用 |
| Generic Schemaでdomain フィールド不足を検出不能 | 98 メッセージをcanonical domain Schemaへ接続し、projectionと値を比較 |
| Task状態が任意文字列 | canonical Task Event、状態連続性、許可遷移表を追加 |
| Workspace fork/Grant継承が未定義 | Workspace Created Schemaで親、mode、Policy binding、Grant非継承を固定 |
| Mailbox consumeが未定義 | consumer、sequence、watermark、冪等性、status、時刻をSchema化 |
| 再配送を一律拒否 | 同一operation fingerprintとredelivery metadataを持つ再配送だけ許可 |

## 独立sequence review

4シナリオとも、開始メッセージからTask Episode確定まで到達可能である。

- E2E-001: Test、GitHub Egress Grant、再試行、PR transaction、Task完了
- E2E-002: Child生成、Child限定Grant、Child Episode、Mailbox consume、Parent統合・完了
- E2E-003: Async wait、completion、Mailbox consume、Compaction後の新Run再開、Task完了
- E2E-004: bypass finding、Policy candidate/regression、Authority、`ACK`後activation、remediation完了

## 独立Schema review

124 sequence Payloadのうち、独立domain stateを表す119 メッセージはcanonical Schemaで検証する。
残りは独立domain stateを更新しないprojectionであり、代表例は次である。

- `ExecutionAuditRecord`: 実行後の監査projection
- `ParentIntegration`: Mailbox消費後の親Run内部処理

Sequenceとcanonical Payloadはメッセージ IDだけでなく、Task、Workspace、同名ref/ダイジェスト、
Task Eventのfrom/to stateで照合する。

## 機械検査

```sh
node scripts/validate-tabletop-scenarios.mjs
node scripts/test-tabletop-validator.mjs
```

Negative testは、必須メッセージ欠落、join path欠落、状態gap、不正cause、誤ったprior cause、
idempotency衝突、canonical binding欠落、projection/canonical不一致、不正状態遷移を拒否する。

## 非blockingな将来メモ

- 製品実装時はAjv等の完全なDraft 2020-12 validatorでも検証する。
- DB unique constraint、並行delivery race、Rule Engine実装はintegration testで確認する。
- `shared_readonly` WorkspaceのE2Eを追加するとき、Workspace Created Schemaのmode拡張要否を判断する。

## 004 Incident責務改定

独立レビュー後、E2E-004はGovernance主体のIncident ワークフローへ改定した。Policy修正を
Control PlaneのRemediation Taskとして扱わず、High risk判定、一時containment、Task suspend、
Human Incident Authority、Policy Revision Authority、remediation後のHuman resume判断を表現する。

改定後はcontainment解除、停止済みRunとは別のAgent Run開始、Task再開までを追加した。
E2E-004のHuman Incident、Revision、Task resumeの各Authority通信は、Governance Planeから
Control PlaneのAuthority GatewayへRequestを渡し、GatewayからDecisionを返す経路に統一する。
Governance Planeから人間への直接通信は許可しない。
Task tree fixtureによる祖先・発生元・子孫のcascadeと兄弟除外を追加した。最終機械検査は、
`4 scenarios / 124 sequence payloads / 119 canonical domain payloads`でPASSし、
sequence reviewerとschema reviewerの独立再レビューもともにPASSした。未使用canonical ペイロードは
検査エラーとし、停止・開始双方のAgent Run command/イベントをcanonical Schemaで検証する。

非blockingな実装上の課題として、`AgentRunStarted`から`TaskResumed`までExecutionがTool Callを
dispatchしないgate、Incident固有negative mutation、Authority リクエスト kindとdecisionの条件制約、
containmentの適用・解除時刻制約を追加する。

## 2026-07-13 技術スタック・プロセス境界レビュー

Go Core、Python Memory、Rust Governanceへの実装分割と、`control.db`、`evidence.db`、`governance.db`のwrite ownershipをレビューした。

- Domain メッセージのsource/target Plane、Authority Gateway経路、Task/Grant/Episodeの状態遷移は変更していない。
- Outbox転送、Inbox durable `ACK`、再送、reconciliationはinfrastructure メッセージであり、既存domain sequence projectionへ追加しない。
- `EgressBlocked`はGovernance AggregateとGovernance OutboxをコミットしてからCLIへblockを返し、Control Inbox適用後にTask Mailboxへ一度だけ追加する。
- Plane横断Transactionを廃止しても、Grant activation/revokeではGovernance `ACK`までAction gateを閉じるため安全性を弱めない。
- Memory FrameworkのSession/checkpointは正本にせず、既存Episode Jobの固定入力、lease、再試行 semanticsを維持する。

Schema reviewではcanonical ペイロードのフィールド、メッセージ type、状態enumに変更がないことを確認したため、Schema revisionとnegative mutationの追加は不要と判定した。baseline、11 negative mutations、idempotent redelivery、nested correlationは再実行してPASSした。

## Kakesu namespace改名レビュー

製品名をKakesuへ変更し、CLI、package、crate、Python module、Schema URN、canonical ペイロード、Viewerを同時移行した。`draft-v0`の実装・永続化開始前であるため、旧`urn:agent-harness:`をactive aliasとして残さず`urn:kakesu:`へ置換した。

Schemaのフィールド、required条件、状態enum、メッセージ type、sequence order、causation、correlationは変更していない。したがって新しいnegative mutationは追加せず、旧namespaceがactive artifactへ残っていないことと、全canonical ペイロードが新namespaceで検証されることをrename固有の検査とした。
