---
task_id: "TASK-0007"
status: draft
planner_agent: "planner-agent-terra-medium"
approved_by: ""
approved_at: ""
approved_dev_profile: "sol-high"
approved_dev_profile_reason: "SQLite transaction、永続化、競合制約、Schema snapshotをまたぐ高リスクのため"
approved_dev_profile_risk_signals: ["persistence", "concurrency", "schema", "cross_cutting"]
planned_implementation_files: 20
planned_implementation_lines: 1380
estimate_points: 8
---

# TASK-0007 PLAN

## 結論とDEVゲート

TASK-0007全体は、現行Coreが`control`のIdleServiceだけであること、SQLite導入、永続モデル、競合、状態機械、クラッシュ復旧、Schema snapshotを同時に導入することから、1〜2 Lapで完了できる境界を超える。既存draftの20実装ファイル・約1,380行・8ptもこの判断を裏付ける。

受け入れ条件を落とすことはしない。このPLANは、main Agentが下記の3 Taskへ契約を分割するまでTASK-0007のDEV開始を承認しない。分割後の各Taskは元の受け入れ条件を一意に引き継ぎ、TASK-0007は全3 Taskの完了後にのみ完了とする。最初のsliceだけをTASK-0007として実装して完了扱いにしてはならない。front matterの20ファイル・1,380行・8ptは元TASK全体の見積であり、承認対象ではない。

`sol-high`を指定する。SQLite transaction、永続化、競合制約、将来consumerの境界はいずれも高リスク信号であり、Lunaの局所・明確・低リスク条件を満たさない。

## 受け入れ条件の具体化と分割提案

| 元の条件 | 観測可能な結果 | 提案Task | 依存 |
|---|---|---|---|
| Taskの原子的作成 | Task、owner Agent割当、workspace参照、contract v1 snapshot、progress v0、`TaskCreated`/`OwnerAssigned` eventが単一transactionで全件commitまたは全件rollbackする | **0007-A Durable foundation** | TASK-0006 |
| owner単一同時Task | 同じAgentへ競合して2 Taskを割り当てても一方だけ成功し、失敗側は部分永続化しない | **0007-B Ownership and lifecycle** | 0007-A |
| 許可遷移のみ | `docs/02-task-lifecycle.md`の明示辺だけを受理し、拒否時はTask状態、progress、event sequenceが不変 | **0007-B Ownership and lifecycle** | 0007-A |
| versioned contract | 更新は期待versionと単調増加を検査し、旧contract snapshotと`ContractChanged` eventを保持する | **0007-C Versioned state and recovery** | 0007-A, 0007-B |
| progressのcurrent/historyと復旧 | 楽観version付きcurrentとappend-only履歴を別保存し、DB close/reopen後も同じread modelを返す | **0007-C Versioned state and recovery** | 0007-A, 0007-B |
| migration/reopen/競合/違法遷移の自動試験 | 各sliceのhermeticなtemporary SQLite integration testで全4種を覆い、0007-C終了時に元条件を満たす | 0007-A〜C | 上記 |

### 0007-A: DEV可能な第一slice（1 Lap）

**契約:** `control.db`を開くための最小SQLite migrationと、root Task作成を原子的に永続化するControl-owned storeを実装する。transport/CLI、Task更新、owner解放、lifecycleの更新、contract/progress更新、Inbox/Outbox、Agent Runは対象外とする。

**完了条件:** 新規DBへのmigration、既存の同一schema version DBの再open、正常作成、故意のSQL失敗によるrollbackをtemporary SQLiteで自動試験する。作成read modelにはTask、owner Agent、workspace reference、contract v1 JSON snapshot（schema ID/revision/digestを含む）、progress v0、sequence 1/2の作成・owner eventを含める。Schemaの意味検証やSchema変更は行わず、既存draft-v0 schemaを不変な参照として保存する。

**実装境界（見積対象）:**

| ファイル | 種別 | 概算行数 | 内容 |
|---|---|---:|---|
| `core/go.mod` | config | 5 | pure-Go SQLite driverを明示依存として追加する（CGO必須driverは採用しない） |
| `core/internal/control/store.go` | implementation | 185 | migration version確認、DB pragma、create input/read model、単一transactionのinsert/rollback、typed conflict/storage error |

テスト（見積対象外）は`core/internal/control/store_test.go`へ置く。migration SQLはGoの埋め込み定数として第一sliceに閉じ、0007-B以降で複数migrationになった時だけ専用migration directoryへ抽出する。第一sliceの実装は上記2ファイル・190行、`file_score=ceil(2/3)=1`、`line_score=ceil(190/200)=1`、従って**1 point**である。

### 0007-B: Ownership and lifecycle（最大2 Lap、別Task）

0007-Aのtransaction portを拡張し、Agentの非終端owner一件をDB制約とcompare-and-setで守る。`ready → running`、`running → waiting|suspended|reviewing_completion`、`waiting|suspended → running`、`reviewing_completion → running|completed`、各非終端状態から`cancelled`だけを遷移表として実装する。終端時はowner解放とeventを同一transactionで確定する。並行割当と全許可/拒否辺をSQL integration testで検証する。

### 0007-C: Versioned state and recovery（最大2 Lap、別Task）

expected contract/progress versionを持つ更新port、immutable contract snapshots、append-only progress history、current progress read model、contract/progress/event schema referenceの永続化を追加する。close/reopen、競合更新、更新失敗時の原子性、旧snapshot/event保存をintegration testで検証する。Schemaの実行時検証が必要になった場合は、採用ライブラリと既存schema resolution方式をPLANレビューへ戻し、依存追加を無承認で行わない。

## 設計根拠と不変条件

- `control.db`はGo Coreのみが書く。ほかのPlaneはDBを直接開かず、Plane間の原子性を求めない（`docs/13-technology-stack.md` §§5–6）。
- `tasks`はcurrent state、contract/event/progress historyはappend-only、progress currentは復旧用read modelとする（`docs/01-domain-model.md`、`docs/11-data-model.md`）。
- ownerの排他はin-memory mutexで代替せずSQLiteの同一transactionと制約で保証する。`waiting`/`suspended`は非終端なのでownerを解放しない（`docs/03-agent-lifecycle.md` §§4, 9, 12）。
- 状態遷移ではcurrent Task、対応TaskEvent、必要なowner/progress更新を同一transactionで確定する。拒否操作はいかなるpartial writeも残さない（`docs/02-task-lifecycle.md` §§1, 11）。
- 全SQLite接続はWAL、foreign keys、busy timeoutを設定する。DB open/migration不能はfail-fastにし、競合を自動retryまたはsilent overwriteしない（`docs/13-technology-stack.md` §6）。

## 代替案と不採用

- 一括実装: 8pt/約1,380行の未検証変更となり、1〜2 Lap境界および独立QAの原因分離を損なうため不採用。
- in-memory store/mutex: 再openとprocess競合を満たさないため不採用。
- event sourcingのみ: MVPの復旧read modelを再生に依存させ、current/history分離の条件を満たしにくいため不採用。
- mutable contract一行: 過去contractとeventの監査性を失うため不採用。
- CGO依存SQLite driver: CI/実行環境のC toolchainへ可用性を結び付けるため、pure-Go driverが利用可能であることを0007-Aの依存確認で確定する。

## 検証と引継ぎ条件

各分割Taskは独立QA_PLANを先に作り、caseごとに`focused-rerun`（temporary SQLite、hermetic・deterministic・bounded）または候補証跡の`evidence-review`を理由付きで指定する。実OS権限、実配置、外部サービス、外部副作用は対象外なので`live-e2e`で代替PASSを作らない。

DEVは各sliceで`gofmt`、`cd core && go test ./...`、`cd core && go vet ./...`、`make check`、`git diff --check`を実行し、candidate commit/tree、fixture、cache条件、exit、artifact digestをHANDOVERへ結び付ける。ReviewerとQAは同一candidateから独立並行に開始する。後続sliceへ進む前に、前sliceの承認candidate treeとmerge treeの同一性をmain Agentが確認する。

## 未解決・main Agent判断待ち

1. TASK-0007を上記0007-A〜Cへ新規Taskとして分割し、それぞれのTASK.mdと依存をmain Agentが作成・承認すること。これはPlannerの権限外であり、本TASK.mdは変更しない。
2. 0007-Aで使うpure-Go SQLite driverの取得可能性、ライセンス、Go 1.23互換性をDEV開始前に確認すること。利用不可なら環境/依存のFAILとしてPLANへ戻し、CGO driverへ自動置換しない。
3. `task-contract.schema.json`と`task-event.schema.json`の実行時validatorの選択は、0007-Cの設計判断とする。第一sliceはschema ID/revision/digestつきbytesの不変保存までに限定する。

## main Agentレビュー

- [ ] TASK-0007を0007-A〜Cへ分割することを承認し、各Taskの受け入れ条件と依存を作成した。
- [ ] 0007-Aの範囲が元TASKの完了ではなく、最初の独立Taskであることを確認した。
- [ ] `sol-high`と1 point見積を承認した。
- [ ] 独立QA_PLANが承認されるまでDEVを開始しない。
