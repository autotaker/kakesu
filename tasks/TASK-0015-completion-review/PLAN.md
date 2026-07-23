---
task_id: "TASK-0015"
status: draft
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "sol-high"
approved_by: ""
approved_at: ""
planned_implementation_files: 17
planned_implementation_lines: 1250
estimate_points: 8
---

# TASK-0015 PLAN

## 受け入れ条件の具体化

| 条件 | 観測方法 | 根拠 |
|---|---|---|
| candidate前検査 | owner/version/pending child/async負例でjobなし | docs/02 §7 |
| 独立入力 | snapshot digest/profile/tool権限の監査試験 | docs/04 §3, docs/11 |
| decision evidence | schema/存在性/Task帰属の負例試験 | docs/01 §5 |
| 原子的終端 | accept/reject/insufficient/retry障害試験 | docs/02 §7 |

## 関連Wikiと判断

- docs/01, 02, 04, 05, 06, 11, 12, 13。依存: TASK-0007、0011、0013、0014。

## 設計

### 選択案

complete handlerがimmutable candidate/input snapshotとreview jobを作り、review workerがResponses structured outputとread-only evidence portを使う。validatorがdecision/evidenceを照合し、control transactionでreview/job/Task event/statusを適用する。
### 代替案と不採用理由

- ownerがcompletedへ遷移: 独立Acceptance判定を欠く。
- reviewerにTask更新tool: 判定と状態更新の監査境界が崩れる。
- live queryをreview入力にする: 判定再現性とdigest整合を失う。
### 責務と境界

- control: preflight、snapshot/job/review、状態遷移。
- workagent/builtin reviewer: ephemeral Responses invocationとread-only evidence adapter。
- evidence port: ref存在・Task帰属・digest検証。Mailbox/childの正本は既存repository。
### 不変条件

- candidateは提出時contract/version/evidenceを固定、reviewerはownerと分離、全decision evidence必須、acceptのみcompleted、同一jobは最大一review。
### 失敗時・移行・互換性

review timeout/worker crashは部分出力を破棄して同一inputでlease再試行する。evidence不足はrunningへ戻し、既存candidateを改変しない。空DB migrationのみ。

## 変更予定

| ファイル群 | 種別 | 概算行数 | 変更内容 |
|---|---|---:|---|
| core/internal/control/{completion_candidate,completion_review,review_job,preflight}.go | implementation | 470 | snapshot/job/atomic apply |
| core/internal/control/{task_transition,evidence_validator}.go | implementation | 150 | transition/ref検証 |
| core/internal/workagent/{acceptance_reviewer,review_client,evidence_tool}.go | implementation | 220 | 独立ephemeral reviewer |
| core/internal/app/app.go | implementation | 30 | worker wiring |
| schemas/draft-v0/{control-plane/completion-review-input,control-plane/completion-review-output,api/built-in-agent-outputs,api/work-agent-tools}.json | schema | 380 | candidate/review契約 |
| core/internal/**/**/*_test.go | test（見積外） | — | preflight/decision/retry |

実装対象17ファイル・1,250行。`ceil(17/3)=6`、`ceil(1250/200)=7`のため8pt。

## 実装手順

1. candidate/review schema、migration、preflightとimmutable snapshotを追加する。
2. lease/retry workerとread-only evidence/reviewer structured outputを実装する。
3. decision validatorとatomic transition/eventを接続する。
4. negative evidence/retry/transition tests、`make check`を実行する。

## 検証計画

- invalid owner/version/ref、pending required child/async、reviewer tool権限、evidence不存在/他Task参照、3種decision、lease expiry/replay、accept transaction failure。
- core Go tests、schema/tabletop negative mutation、`make check`。

## 未解決事項

- evidence serviceの初期read-only query adapterはTASK-0007のcontrol/evidence contractに合わせる。

## main Agentレビュー

- [ ] 受け入れ条件が検証可能である。
- [ ] 設計観点と代替案を検討している。
- [ ] QA計画を作成できる。
- [ ] 見積もりが規則どおりである。
- [ ] DEV開始を承認した。
