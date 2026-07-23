---
task_id: "TASK-0018"
status: draft
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "sol-high"
approved_by: ""
approved_at: ""
planned_implementation_files: 16
planned_implementation_lines: 1200
estimate_points: 8
---

# TASK-0018 PLAN

## 受け入れ条件の具体化

| 条件 | 観測方法 | 根拠 |
|---|---|---|
| request binding/idempotency | invalid/expired/cross-workspace/call replay負例 | docs/06 §8, docs/07 §9 |
| minimal Agent evaluation | fixed sanitized input、output/authority normalization試験 | docs/04 §4, docs/07 §10 |
| pending ACK | ACK前allow不可、restart/replay後の収束試験 | docs/07 §11, docs/11 §5 |
| one-use/manual retry | exact digest一回だけallow、初回自動replayなし | docs/06 §8, docs/10 §9 |

## 関連Wikiと判断

- docs/04-07, 10-12。依存: TASK-0011、0013、0016、0017。scope/authorityはAgentでなくimmutable challengeとplatform policyで確定する。

## 設計

### 選択案

Controlがtool callからidempotent GrantRequest/async operationを作り、Governanceが固定challengeを再検証してevaluation jobをlease実行する。managerは構造化出力をnormalizeし、challengeから決定論的なpending one-use grantとpolicy versionを保存する。enforcer ACKでのみactive化し、outbox/inbox経由でcompletion/mailboxを確定する。

### 代替案と不採用理由

- synchronous Agent結果で即allow: enforcement反映前の抜け道になる。
- Agentがrule/TTL/authorityを返す: 権限ロンダリングになる。
- gateway replay: 意図しない外部作用を再発する。

### 責務と境界

- control: tool validation、async/mailbox、inbox適用。
- governance: request/evaluation/decision/grant、policy ACK検証。
- Policy Agent: ephemeral structured assessment/read-only evidenceのみ。
- enforcer: version ACKとactive grant消費。Authority/恒久改訂は別Task。

### 不変条件

- challenge/Task/Workspace/contract一致、非終端requestは1件、grant scopeはimmutable binding由来、ACK前inactive、active grantは一回・期限内・exact digestのみ、初回commandを再送しない。

### 失敗時・移行・互換性

timeout/crashは固定job入力でlease retryする。ACK喪失はidempotent再配送しactiveを重複生成しない。expired/cancelled challenge/grantはdeny、旧grantなしのDBからmigrationする。

## 変更予定

| ファイル群 | 種別 | 概算行数 | 変更内容 |
|---|---|---:|---|
| core/internal/control/{grant_tool,async,mailbox,governance_inbox}.go | implementation | 300 | tool/async/inbox |
| governance/src/{grant,policy_agent,activation,outbox,store}.rs | implementation | 440 | job/decision/pending→active |
| core/internal/{workagent,execution}/**.go | implementation | 150 | ToolResult/ACK/enforcer client |
| schemas/draft-v0/{governance-plane,control-plane,api}/*grant*.json | schema | 160 | request/decision/grant契約 |
| core/**/**/*_test.go; governance/tests/grant_activation.rs | test | 150 | replay/ACK/one-use |

実装対象16ファイル・1,200行。`ceil(16/3)=6`、`ceil(1200/200)=6`のため8pt。

## 実装手順

1. request/decision/grant schemaとmigration、control tool validationを追加する。
2. fixed-input Policy Agent jobとmanager normalizationを実装する。
3. pending activation/ACK/outbox-inbox/mailboxを接続する。
4. replay、authority normalization、crash、one-use/manual retryを試験する。

## 検証計画

- stale/cross-workspace challenge、duplicate call、Agent malformed output、auto-grant false、ACK前/後、ACK重複、expiry/cancel、digest不一致、二回再実行。
- Go/Rust unit+integration、schema/tabletop sequence/negative mutation、`make check`。

## 未解決事項

- `require_authority`は記録して非終端とし、人間承認の継続実装は後続Taskとする。

## main Agentレビュー

- [ ] 受け入れ条件が検証可能である。
- [ ] 設計観点と代替案を検討している。
- [ ] QA計画を作成できる。
- [ ] 見積もりが規則どおりである。
- [ ] DEV開始を承認した。
