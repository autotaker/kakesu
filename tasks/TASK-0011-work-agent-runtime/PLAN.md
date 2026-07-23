---
task_id: "TASK-0011"
status: draft
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "sol-high"
approved_by: ""
approved_at: ""
planned_implementation_files: 16
planned_implementation_lines: 1350
estimate_points: 8
---

# TASK-0011 PLAN

## 受け入れ条件の具体化

| 条件 | 観測方法 | 根拠 |
|---|---|---|
| Task排他と冪等dispatch | 並行/replay unit testでAPI呼出しと作用が各一回 | docs/05 §6, §7 |
| durable step記録 | SQLite再起動試験でstep/item/intent/result参照を照合 | docs/05 §7, docs/11 |
| 正本再開 | 旧Run停止後に新Runの入力・cursor watermarkを検査 | docs/05 §9 |
| 失敗原子性 | API/DB障害注入で部分Task遷移がないことを確認 | docs/05 §7 |

## 関連Wikiと判断

- docs/01, 03, 05, 11, 12, 13。依存: TASK-0007/0008/0010。

## 設計

### 選択案

control.db repositoryを正本にし、Run coordinatorがTask lock下でcontext読込→API step→completed item/dispatch intent commit→tool結果/continuation commitを進める。Responses clientはportに分離し、resumeは新Runとcursor snapshotから行う。

### 代替案と不採用理由

- API会話IDだけで復旧: Task状態・Mailbox・契約版を復元できない。
- goroutine/channelをRun正本にする: 再起動・重複配送を扱えない。
- 外部APIをDB transaction内で呼ぶ: lock保持と不確定外部作用を招く。

### 責務と境界

- `control`: Task lock、Run/step/item/continuation永続化。
- `workagent`: Context builder、Responses client adapter、出力検証とcoordinator。
- `message`: dispatch intentのoutbox接続。後続Taskがtool意味を実装する。

### 不変条件

- 一Task一step、ID種別分離、completed itemとintentの先行commit、再開時の旧response chain非継承。
- 推論本文・秘密情報をTask正本へ保存しない。cursorは契約版・workspace snapshot・watermarkと一致する。

### 失敗時・移行・互換性

API/DB失敗はstepをfailed/retryableにし、未完intentを再確保する。初期schema migrationは空DBから作成し、既存scaffoldに互換APIを要求しない。

## 変更予定

| ファイル群 | 種別 | 概算行数 | 変更内容 |
|---|---|---:|---|
| core/internal/control/{store,task_lock,run_repository,continuation}.go | implementation | 390 | SQLite正本、排他、cursor |
| core/internal/workagent/{service,coordinator,context,responses_client,output}.go | implementation | 520 | step loopとAPI adapter |
| core/internal/message/{store,dispatcher}.go | implementation | 110 | durable intent境界 |
| core/internal/app/app.go, core/go.mod | implementation | 70 | DIとSQLite依存 |
| schemas/draft-v0/execution-plane/{agent-run-event,continuation,resume-context}.schema.json | schema | 260 | 実行/再開契約の確定 |
| core/internal/{control,workagent}/**/*_test.go | test（見積外） | — | replay、再起動、障害注入 |

実装対象16ファイル・1,350行。`ceil(16/3)=6`、`ceil(1350/200)=7`のため8pt。

## 実装手順

1. control.db migration、repository、Task lockとRun/step/itemモデルを追加する。
2. Context builderとResponses client portを実装し、通常stepの二段階commitを接続する。
3. cursor生成・新Run再構築、進捗maintenance step、replay recoveryを追加する。
4. schema/contract testとfocused failure test後に`make check`を実行する。

## 検証計画

- Go unit/integration: parallel step、response/call replay、DB/API失敗、process restart、stale cursor、secret redaction。
- `go test ./...`（core）、schema/tabletop検証、`make check`。

## 未解決事項

- 実API SDKの固定版と認証注入方式はDEV開始前に既存依存方針へ整合させる。契約境界はclient portで固定する。

## main Agentレビュー

- [ ] 受け入れ条件が検証可能である。
- [ ] 設計観点と代替案を検討している。
- [ ] QA計画を作成できる。
- [ ] 見積もりが規則どおりである。
- [ ] DEV開始を承認した。
