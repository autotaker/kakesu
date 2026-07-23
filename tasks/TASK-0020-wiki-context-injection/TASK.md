---
task_id: "TASK-0020"
title: "意味Wiki照会とWork Agent強制注入を実装する"
status: ready
created_at: "2026-07-15"
---

# TASK-0020 意味Wiki照会とWork Agent強制注入を実装する

## 目的

Harnessが`task_start`、`subtask_start`、`resume`、`context_gap`、`contract_change`、`escalation`、`egress_grant`でWiki AgentにMemoryContextを要求し、現行contract/versionに適合する組織記憶をWork Agentの強制区画へ注入する。

## 背景

TASK-0019が確定episodeを供給しても、Work Agentの自由検索では記憶の欠落・古い反例の無秩序な利用・予算逸脱を防げない。Task/contract stateを優先するcontext builderが必要である。依存: TASK-0008、0010、0011、0014、0019。

## スコープ

### 対象

- MemoryContext request/response/job、7 trigger別の強制query、Wiki Agent query mode、source ref/memory version/contract version検証とstale response拒否。
- context builderによる`[AGENT CONTRACT]`、`[CURRENT TASK STATE]`、`[ORGANIZATIONAL MEMORY]`、`[MEMORY USAGE RULE]`区画注入。
- `report_context_gap`の非同期再問合せとmailbox結果、token budget/timeout/fallback。

### 対象外

- Wiki maintenanceの高度な自動patch・自動publish、semantic schema変更、Work AgentのWiki/episode直接read。
- task_start以外の各responseごとの検索、Memoryの新しいepisode編纂。

## 受け入れ条件

- [ ] task/subtask開始、resume、context gap、contract change、escalation、egress grantが一意なphase requestを作り、同じtriggerの再配送は同じjob/resultへ収束する。
- [ ] Wiki queryはTask contract/versionを入力にsemantic/episode/unresolved/source refsとmemory versionを返し、Harnessがschema/version/ref/token budgetと現行contract/version一致を検証し、stale responseを拒否する。
- [ ] Work AgentはWiki filesystem/storeを直接読めず、開始/resume contextには契約・現Task state・組織記憶・usage ruleの4区画が常に存在する。
- [ ] context gapは非同期mailboxで返り、Wiki失敗/timeoutはTask stateを壊さず空の明示degraded contextまたは再試行へ収束する。

## 検討すべき設計観点

- 記憶はinstructionでなく区画化された参考情報で、現行contractが常に優先する。queryとmaintenanceを別job/権限に分ける。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] 必要な実装、テスト、文書が完成している。
- [ ] 独立レビューと`make check`がPASSしている。
- [ ] マージ後QAが完了している。
- [ ] HANDOVERがWiki Agentに取り込みされている。

## 関連コンテキスト

### 意味 Wiki

- docs/04 §7、docs/05 §§2,6,9、docs/06 §§2,12、docs/08 §§7-13,19、docs/11 §2、docs/12 §6。依存: TASK-0008、0010、0011、0014、0019。
### 判断

- Harnessがphaseを強制し、Memoryを明示区画に注入してcontract優先を固定する。
### 適用しなかった重要な判断

- 高度な自動patch/publish、自由検索、毎response query、Work AgentによるWiki直接アクセスを採用しない。
