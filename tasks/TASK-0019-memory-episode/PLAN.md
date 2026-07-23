---
task_id: "TASK-0019"
status: draft
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "sol-high"
approved_by: ""
approved_at: ""
planned_implementation_files: 16
planned_implementation_lines: 1200
estimate_points: 8
---

# TASK-0019 PLAN

## 受け入れ条件の具体化

| 条件 | 観測方法 | 根拠 |
|---|---|---|
| terminal snapshot | duplicate/outbox reorderでも一digest snapshot | docs/08 §5, docs/11 §5 |
| lease/retry | crash/lease expiryでpartial破棄・fixed input retry | docs/08 §5 |
| evidence isolation | DML/ATTACH/cross-task/budget超過負例 | docs/08 §5 |
| episode validation | schema/ref/epistemic負例と一件確定試験 | docs/08 §§3,5 |

## 関連Wikiと判断

- docs/04, 05, 07, 08, 11-13。依存: TASK-0006、TASK-0008、TASK-0015、TASK-0018。terminal snapshotはEgressChallenge/Grant/OutboundTransactionの不変evidence refを含み、EpisodeはTask終端後に一度だけ確定する。

## 設計

### 選択案

Memory serviceがinboxをwatermark順に受け、terminal snapshot/evidence tablesとidempotent jobを同一DBへ作る。snapshotはEgressChallenge、Grant、OutboundTransaction refとTask/Workspace bindingを含み、validatorはその不変性を検証してからepisodeを一件保存する。

### 代替案と不採用理由

- 全履歴を一回要約: budget/retry/根拠検証を失う。
- SDK session/traceを永続化: framework依存と秘匿リスクを増やす。
- core/governance DBを直接query: Plane所有・snapshot整合を壊す。

### 責務と境界

- memory Python: evidence DB、inbox、snapshot/job/lease、SDK adapter、validator。
- core/governance: versioned terminal evidenceをoutbox送信。
- SDK Agent: ephemeral read-only調査とstructured output。Wiki maintenance/queryはTASK-0020。

### 不変条件

- task scope固定、terminal snapshot immutable、job最大一episode、partial output非保存、queryはread-only/budget内、terminal Taskをjob失敗で戻さない、evidence refはworktreeから独立する。

### 失敗時・移行・互換性

lease expiryはattempt増分と新sessionで再試行し、上限後`needs_operator`にする。inbox重複/再順序はidempotency key/watermarkで収束。空DB migrationのみ、schema/profile不一致はepisode確定前に失敗させる。

## 変更予定

| ファイル群 | 種別 | 概算行数 | 変更内容 |
|---|---|---:|---|
| memory/src/kakesu_memory/{store,migrations,evidence,snapshot,jobs,service}.py | implementation | 460 | DB/inbox/snapshot/lease |
| memory/src/kakesu_memory/{evidence_tool,openai_runner,validator,contracts}.py | implementation | 300 | read-only SDK/validation |
| core/internal/{control,plane}/memory_outbox.go | implementation | 90 | terminal evidence送信 |
| schemas/draft-v0/memory-plane/{task-episode,evidence-*,episode-job}.schema.json | schema | 170 | episode/query契約 |
| memory/tests/{test_store,test_episode,test_evidence_tool}.py | test | 180 | retry/isolation/validation |

実装対象16ファイル・1,200行。`ceil(16/3)=6`、`ceil(1200/200)=6`のため8pt。

## 実装手順

1. evidence/job schema、migration、inboxとterminal snapshotを実装する。
2. lease worker、task-scoped evidence views、SDK adapterを接続する。
3. episode validatorとidempotent saveを追加する。
4. crash/retry/query abuse/schema/refの負例を追加する。

## 検証計画

- terminal重複/reorder、watermark差分、lease競合/expiry、SDK failure、DML/ATTACH/他Task query、row/byte/token上限、invalid episode/ref/epistemic、needs_operator。
- Python tests、schema/tabletop negative mutation、`make check`。

## 未解決事項

- core/governanceからmemoryへの終端watermark envelopeはTASK-0006のmessage catalogへ追加整合が必要である。

## main Agentレビュー

- [ ] 受け入れ条件が検証可能である。
- [ ] 設計観点と代替案を検討している。
- [ ] QA計画を作成できる。
- [ ] 見積もりが規則どおりである。
- [ ] DEV開始を承認した。
