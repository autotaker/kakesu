---
task_id: "TASK-0016"
status: draft
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "sol-high"
approved_by: ""
approved_at: ""
planned_implementation_files: 15
planned_implementation_lines: 1100
estimate_points: 8
---

# TASK-0016 PLAN

## 受け入れ条件の具体化

| 条件 | 観測方法 | 根拠 |
|---|---|---|
| ownership/socket | Go直接SQLiteなし、envelope schema/digest負例 | docs/07 §1.1, docs/13 §5 |
| binding/rule decision | workspace別binding、priority/deny-overrides/version再現試験 | docs/07 §7 |
| fail-closed | missing/stale/unavailable/競合/transaction failureで転送なし | docs/07 §§3,7 |
| audit/challenge | block transactionのattempt/challenge/outbox、allow先行記録試験 | docs/07 §§6-8, docs/11 §5 |

## 関連Wikiと判断

- docs/07, 11, 12, 13。依存: TASK-0006、0008。統治DB単独所有、Workspace主体、inline LLMなし、fail-closedを固定する。

## 設計

### 選択案

Rust store/repositoryがmigrationと統治集約を所有し、socket handlerが正規requestとWorkspace network identityを受ける。binding→versioned policy→deterministic evaluatorの順で`EgressRuleDecision`を作り、blockではimmutable challenge/outbox、allowでは監査先行書込みを同一SQLite transactionで確定する。

### 代替案と不採用理由

- coreから共有SQLiteを開く: 所有・transaction・監査境界を壊す。
- Agent/Taskをpolicy keyにする: ネットワーク強制主体と一致しない。
- unavailable時allowまたはLLM hot path: 安全性/遅延/再現性を満たさない。

### 責務と境界

- governance Rust: DB、migration、binding、rule evaluator、attempt/decision/challenge/outbox。
- core: versioned envelopeを送受信してTask mailboxへ適用するのみ。
- execution adapter: network identityとcanonical requestを供給するが強制はTASK-0017。

### 不変条件

- 明示allowかつReady storeだけallow、policy versionとrequest digestはdecisionへ固定、challengeはimmutable、監査を保存できなければ転送しない、統治DB外の原子性はACK/idempotencyで収束する。

### 失敗時・移行・互換性

空DBからmigrationする。migration/store/socket障害はblockし、outbox未配送は統治DB内で再送する。旧in-memory雛形は本番判断経路から除去し、schema revision非互換は拒否する。

## 変更予定

| ファイル群 | 種別 | 概算行数 | 変更内容 |
|---|---|---:|---|
| governance/src/{store,migrations,policy,decision,challenge,outbox}.rs | implementation | 620 | SQLite集約、評価、transaction/outbox |
| governance/src/{lib,lifecycle,message,main}.rs | implementation | 180 | socket service/wiring/contract |
| governance/tests/{policy_store,fail_closed,message_contract}.rs | test | 180 | binding・障害・原子性 |
| schemas/draft-v0/governance-plane/{casb-policy,casb-rule,egress-*.schema}.json | schema | 120 | 永続/decision/challenge契約 |

実装対象15ファイル・1,100行。`ceil(15/3)=5`、`ceil(1100/200)=6`のため8pt。

## 実装手順

1. schema/migrationとstore所有境界を追加する。
2. binding解決、priority/deny-overrides evaluator、decision/challengeを実装する。
3. socket envelope/outboxとfail-closed transactionを接続する。
4. 再現性、競合、障害、先行監査の負例を追加する。

## 検証計画

- binding欠落、policy revision違い、rule競合、unknown/stale/unavailable、SQLite書込み失敗、challenge再観測、outbox再送。
- Rust unit/integration、schema validator/tabletop negative mutation、`make check`。

## 未解決事項

- Unix socketの配置・起動所有はTASK-0006のruntime contractへ合わせる。

## main Agentレビュー

- [ ] 受け入れ条件が検証可能である。
- [ ] 設計観点と代替案を検討している。
- [ ] QA計画を作成できる。
- [ ] 見積もりが規則どおりである。
- [ ] DEV開始を承認した。
