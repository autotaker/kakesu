---
task_id: "TASK-0014"
status: draft
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "sol-high"
approved_by: ""
approved_at: ""
planned_implementation_files: 18
planned_implementation_lines: 1350
estimate_points: 8
---

# TASK-0014 PLAN

## 受け入れ条件の具体化

| 条件 | 観測方法 | 根拠 |
|---|---|---|
| atomic delegate | 妥当性/資源失敗でchildなし、成功で全関連行あり | docs/02 §3 |
| 独立owner/workspace | child親のID/policy binding分離を照合 | docs/01 §3 |
| 結果配送 | child終端/replayで親Mailbox一entry | docs/06 §4, docs/10 |
| 直接cancel/cascade | sibling/grandchild拒否と子孫終端試験 | docs/02 §9 |

## 関連Wikiと判断

- docs/01, 02, 05, 06, 10, 11, 12, 13。依存: TASK-0007、0010、0011、0013。

## 設計

### 選択案

delegate command handlerが同一control transactionでproposal dedupe、Task/owner/`empty` workspace/progress/async operationを作成し、child schedulerを起動する。`fork`/`shared_readonly`はWorkspace作成前に未対応capabilityとして拒否する。終端handlerはoutboxで親Mailboxイベントを発行する。cancelは直接関係を検証してcascadeを確定後、resource cleanup jobを起動する。
### 代替案と不採用理由

- parentがchild recordを直接作る: owner/workspace検証を迂回する。
- shared owner/worktree: 責任とsecurity bindingを混同する。
- cleanup完了を待ってcancel遷移: resource障害で責任撤回を曖昧にする。
### 責務と境界

- control: proposal、Task tree、owner allocation、cancel/cascade、result outbox。
- execution: Workspace作成/cleanup resource。
- workagent: tool schema/同期deadlineのみ。Mailbox適用はTASK-0013。
### 不変条件

- childは一owner/一workspace、tree無循環、operation key一意、結果一回、親cancelは直接child限定、Task終端後に新作用なし。
### 失敗時・移行・互換性

作成の任意構成要素失敗はtransaction rollback。child resource cleanup失敗はTask状態を戻さず再試行対象にする。初期schema migrationのみ。

## 変更予定

| ファイル群 | 種別 | 概算行数 | 変更内容 |
|---|---|---:|---|
| core/internal/control/{delegate,task_tree,owner_allocator,cancel_child}.go | implementation | 480 | proposal/create/cascade |
| core/internal/control/{task_repository,async_repository,termination}.go | implementation | 200 | atomic state/outbox |
| core/internal/execution/{workspace_factory,cleanup}.go | implementation | 140 | child workspace/resource |
| core/internal/workagent/{tools,delegate_dispatch}.go | implementation | 100 | tool boundary |
| schemas/draft-v0/{control-plane/task-contract,control-plane/child-completed,control-plane/task-command,api/work-agent-tools}.json | schema | 430 | delegate/cancel契約 |
| core/internal/**/**/*_test.go | test（見積外） | — | failure/tree/cancel |

実装対象18ファイル・1,350行。`ceil(18/3)=6`、`ceil(1350/200)=7`のため8pt。

## 実装手順

1. proposal/tree/cancel schemaとrepository制約を追加する。
2. owner/workspace原子的作成とscheduler/async昇格を実装する。
3. child終端outbox配送、direct cancel/cascade/cleanupを接続する。
4. tree/replay/resource failure tests、`make check`を実行する。

## 検証計画

- invalid/duplicate proposal、`empty` workspace成功、`fork`/`shared_readonly`の起動前拒否、owner不足、sync/async結果、result replay、direct/sibling/grandchild cancel、cascade cleanup failure。
- core Go tests、schema/tabletop negative mutation、`make check`。

## 未解決事項

- なし。MVPのchild Workspaceは`empty`に限定し、`fork`/`shared_readonly`は後続EPICのcapabilityとする。

## main Agentレビュー

- [ ] 受け入れ条件が検証可能である。
- [ ] 設計観点と代替案を検討している。
- [ ] QA計画を作成できる。
- [ ] 見積もりが規則どおりである。
- [ ] DEV開始を承認した。
