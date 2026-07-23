---
task_id: "TASK-0020"
status: draft
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "sol-high"
approved_by: ""
approved_at: ""
planned_implementation_files: 15
planned_implementation_lines: 1100
estimate_points: 8
---

# TASK-0020 PLAN

## 受け入れ条件の具体化

| 条件 | 観測方法 | 根拠 |
|---|---|---|
| forced phases | start/subtask/resume/context_gap/contract_change/escalation/egress_grantのrequest/job重複試験 | docs/08 §9 |
| response validation | bad schema/ref/version/budgetとstale contract/version responseを注入前拒否 | docs/08 §§10-11 |
| compartment injection | 4区画、contract優先、direct-read不可試験 | docs/05 §2, docs/08 §12 |
| failure recovery | timeout/degraded/retry/mailbox再配送試験 | docs/06 §§12-17 |

## 関連Wikiと判断

- docs/04-06, 08, 11-12。依存: TASK-0008、0010、0011、0014、0019。query/maintenanceを分離し、Harness強制注入を採る。

## 設計

### 選択案

Control/context builderが7 triggerごとにcontract/versionを固定したidempotent MemoryContextRequestを送る。Harnessはschema/ref/budgetに加え現行contract/version一致を検証し、stale responseを注入せず再要求してから4区画を渡す。

### 代替案と不採用理由

- Work Agent自由検索: 強制性、token統制、反例選択を保証できない。
- system instructionへ混在: contract/記憶の優先順位が不明瞭。
- 毎response検索: latency/costと同じ状態の再取得を増やす。

### 責務と境界

- core: phase trigger、context builder、async/mailbox、direct access拒否。
- memory/Wiki query: contractベース選択とversioned response。
- Work Agent: injected viewを参照しgapを報告するだけ。maintenance/patchは範囲外。

### 不変条件

- Work AgentはWiki/storeを直接読まない、phase以外で検索しない、contract/current stateはmemoryより優先、source refs/version/budgetを検証、同一triggerは冪等、failureでTask stateを改変しない。

### 失敗時・移行・互換性

query lease/timeoutはretryし、期限後はmemory unavailableを明記したdegraded contextを返す。response schema/version不一致は注入せず再取得する。初期空Wikiは空semantic contextとversionを返し起動を妨げない。

## 変更予定

| ファイル群 | 種別 | 概算行数 | 変更内容 |
|---|---|---:|---|
| core/internal/control/{context_request,context_builder,async,mailbox}.go | implementation | 350 | trigger/job/injection |
| core/internal/workagent/{service,tools,context_view}.go | implementation | 150 | direct-read拒否/gap tool |
| memory/src/kakesu_memory/{context_service,wiki_query,context_jobs}.py | implementation | 260 | query mode/job |
| schemas/draft-v0/memory-plane/{memory-context-request,memory-context}.schema.json | schema | 130 | request/response契約 |
| core/**/**/*_test.go; memory/tests/test_context.py | test | 210 | lifecycle/validation/failure |

実装対象15ファイル・1,100行。`ceil(15/3)=5`、`ceil(1100/200)=6`のため8pt。

## 実装手順

1. request/response schemaとmemory context job/migrationを追加する。
2. lifecycle trigger、idempotency、query workerを接続する。
3. 4区画context builderとcontext gap mailboxを実装する。
4. direct-read、version/ref、degraded/retry、budgetの負例を追加する。

## 検証計画

- 4 phase、duplicate/reordered event、contract変更後resume、invalid/oversize source、empty Wiki、query crash/timeout、context gap async、Work Agent direct-path deny。
- Go/Python tests、schema/tabletop negative mutation、`make check`。

## 未解決事項

- semantic repositoryのread APIとmemory versionのGit commit解決はTASK-0019のstore schemaに合わせる。

## main Agentレビュー

- [ ] 受け入れ条件が検証可能である。
- [ ] 設計観点と代替案を検討している。
- [ ] QA計画を作成できる。
- [ ] 見積もりが規則どおりである。
- [ ] DEV開始を承認した。
