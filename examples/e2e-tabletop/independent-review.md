# E2E Tabletop Debugging 独立レビュー

## 最終判定

**PASS**。Schema reviewer、sequence reviewerの双方が、対象4シナリオを無条件PASSと判定した。

| 検査対象 | 結果 |
|---|---:|
| E2E Scenario | 4 PASS |
| Sequence Payload | 100 PASS |
| Canonical domain Payload | 98 PASS |
| Sequence Requirement | 4 PASS |
| Negative mutation | 9 reject |
| Idempotent redelivery | PASS |
| Nested correlation | PASS |

## 初回レビューで発見した不足と解消

| 発見 | 解消内容 |
|---|---|
| GrantがChallengeからactiveへ直行 | Request、Decision、pending activation、Rule Engine ACK、active、Ready、CLI retryを追加 |
| ToolCallとEgressの因果が不明 | Tool call、process、attempt、challenge、grant、transactionをID結合 |
| Review後のTask terminal確定がない | ReviewingCompletion、Review Input/Output、TaskCompleted、Episode Inputを追加 |
| Child Task生成・親統合が不完全 | Child Workspace/Task、Child Episode、Mailbox consume、Parent Integration/Review/Episodeを追加 |
| Asyncがrunningのまま完了通知 | Async completed永続化、Mailbox consume、ResumeContext、新Run、TaskResumedを追加 |
| Policy RevisionがACK前にactive | Candidate、Regression、Authority、pending revision、ACK、activeを分離 |
| Requirementが宣言だけでvacuous pass | 必須type、順序、field join、direct causationを各Scenarioへ適用 |
| Generic Schemaでdomain field不足を検出不能 | 98 messageをcanonical domain Schemaへ接続し、projectionと値を比較 |
| Task状態が任意文字列 | canonical Task Event、状態連続性、許可遷移表を追加 |
| Workspace fork/Grant継承が未定義 | Workspace Created Schemaで親、mode、Policy binding、Grant非継承を固定 |
| Mailbox consumeが未定義 | consumer、sequence、watermark、冪等性、status、時刻をSchema化 |
| 再配送を一律拒否 | 同一operation fingerprintとredelivery metadataを持つ再配送だけ許可 |

## 独立sequence review

4シナリオとも、開始messageからTask Episode確定まで到達可能である。

- E2E-001: Test、GitHub Egress Grant、retry、PR transaction、Task完了
- E2E-002: Child生成、Child限定Grant、Child Episode、Mailbox consume、Parent統合・完了
- E2E-003: Async wait、completion、Mailbox consume、Compaction後の新Run再開、Task完了
- E2E-004: bypass finding、Policy candidate/regression、Authority、ACK後activation、remediation完了

## 独立Schema review

100 sequence Payloadのうち、独立domain stateを表す98 messageはcanonical Schemaで検証する。
残る2件は独立domain stateを更新しないprojectionである。

- `ExecutionAuditRecord`: 実行後の監査projection
- `ParentIntegration`: Mailbox消費後の親Run内部処理

Sequenceとcanonical Payloadはmessage IDだけでなく、Task、Workspace、同名ref/digest、
Task Eventのfrom/to stateで照合する。

## 機械検査

```sh
node scripts/validate-tabletop-scenarios.mjs
node scripts/test-tabletop-validator.mjs
```

Negative testは、必須message欠落、join path欠落、状態gap、不正cause、誤ったprior cause、
idempotency衝突、canonical binding欠落、projection/canonical不一致、不正状態遷移を拒否する。

## 非blockingな将来メモ

- 製品実装時はAjv等の完全なDraft 2020-12 validatorでも検証する。
- DB unique constraint、並行delivery race、Rule Engine実装はintegration testで確認する。
- `shared_readonly` WorkspaceのE2Eを追加するとき、Workspace Created Schemaのmode拡張要否を判断する。
