---
task_id: "TASK-0008"
status: draft
planner_agent: "planner-agent-terra-medium"
approved_by: ""
approved_at: ""
approved_dev_profile: "sol-high"
planned_implementation_files: 20
planned_implementation_lines: 1460
estimate_points: 8
---

# TASK-0008 PLAN

## 受け入れ条件の具体化

| 条件 | 観測方法 | 根拠 |
|---|---|---|
| durable send | sender crash後の未ACK Outboxが再claimされる | `docs/00` §3.1 |
| ACK境界 | receiver crash before/after Inbox commitでACKと重複が正しく収束する | `core/internal/message/store.go` |
| UDS | valid framed JSONのみ受理し、切断/oversize/bad schemaでqueueを失わない | Task-0006 |
| supervision | any必須workerの起動失敗・unexpected exitがReadyを解除し、shutdownが完了を待つ | `core/.../run_group.go`、各Plane README |

## 関連Wikiと判断

- `docs/00-kakesu.md` §3.1、`docs/07-governance.md` §§1.1/8、`docs/13-technology-stack.md` §§5–7。
- Task-0006のwire contract/config、Task-0007の`control.db` transaction/repository。
- `core/internal/message/store.go`、`core/internal/plane/run_group.go`、Memory/Governance README。

## 設計

### 選択案

各Plane DBにOutbox/Inbox/leaseを持たせる。送信側はlocal business transactionでOutboxを追加し、dispatcherがlease付きでclaimしてUDSへframeを送る。receiverはschema/envelope検査後、`message_id` unique Inbox insertをcommitし、その後にACKする。ACK前disconnectはsenderが再送し、duplicate insertはACKするがhandlerを再実行しない。Coreは既存RunGroupをservice supervisorへ拡張し、Python/Rustも同じReady/drain/failure契約を実装する。

### 代替案と不採用理由

- Exactly-once/2PC: DB所有境界と可用性を損なうため不採用。
- socket successをdeliveryとみなす: receiver crashで喪失するため不採用。
- shared DB: Plane単独所有の設計を破るため不採用。

### 責務と境界

- store: local transaction/lease/dedupe。transport: UDS framing/ACK/reconnect。dispatcher: claim/send/retry。handler: Inboxから意味処理を開始する境界。supervisor: workerの生存とshutdown。
- Controlは`control.db`のみ、Memoryは`evidence.db`のみ、Governanceは`governance.db`のみを書き込む。

### 不変条件

- `message_id`はreceiver Inbox内で一意、ACKはcommit後のみ。
- Outbox recordはdelivered ACKまで削除せず、lease expiry後だけ再claim可能。handlerはdedupe成功後に一回だけ走る。
- DB横断transactionを作らない。UDS pathはlocal-only、peer/path allowlist、frame上限を持つ。
- Readyは必要なDB/migrations/listener/dispatcher全ての成功後だけ報告する。

### 失敗時・移行・互換性

- send/connect/ACK失敗はOutboxをpendingへ戻しbackoffする。corrupt/unknown schemaはquarantine/observable failureにして無限handler retryしない。
- receiver DB failure時はACKせずsenderに再送させる。shutdownでは新claimを停止し、in-flightを期限までdrainしてleaseを解放する。
- 初期MVPのqueue schemaは新規で、旧in-memory channelに移行しない。

## 変更予定

見積もり対象は実装コード、Schema、設定ファイルだけとする。

| ファイル | 種別 | 概算変更行数 | 変更内容 |
|---|---|---:|---|
| `core/internal/message/sqlite_store.go` | implementation | 150 | control.db inbox/outbox/lease adapter |
| `core/internal/message/sqlite_store_test.go` | implementation | 120 | ack/dedupe/recovery tests |
| `core/internal/transport/frame.go` | implementation | 85 | bounded framed JSON codec |
| `core/internal/transport/uds_client.go` | implementation | 100 | reconnecting sender |
| `core/internal/transport/uds_server.go` | implementation | 120 | commit-before-ack receiver |
| `core/internal/transport/dispatcher.go` | implementation | 115 | claim/lease/backoff loop |
| `core/internal/transport/transport_test.go` | implementation | 140 | crash/duplicate/disconnect integration |
| `core/internal/plane/supervisor.go` | implementation | 110 | readiness/drain/failure supervision |
| `core/internal/plane/supervisor_test.go` | implementation | 90 | lifecycle tests |
| `core/internal/app/app.go` | implementation | 55 | wire services into supervisor |
| `memory/src/kakesu_memory/store.py` | implementation | 115 | evidence.db queue store |
| `memory/src/kakesu_memory/transport.py` | implementation | 100 | UDS dispatcher |
| `memory/tests/test_transport.py` | implementation | 95 | Python recovery tests |
| `memory/pyproject.toml` | config | 15 | SQLite/transport dep |
| `governance/src/store.rs` | implementation | 115 | governance.db queue store |
| `governance/src/transport.rs` | implementation | 110 | Tokio UDS dispatcher |
| `governance/src/supervisor.rs` | implementation | 85 | Rust worker supervision |
| `governance/tests/transport.rs` | implementation | 110 | ACK/restart tests |
| `governance/Cargo.toml` | config | 20 | SQLite/UDS deps |
| `configs/local/wire.json` | config | 10 | transport retry/frame values |

## 見積もり

```text
file_score = ceil(planned_implementation_files / 3)
line_score = ceil(planned_implementation_lines / 200)
estimate_points = 1, 2, 3, 5, 8, 13のうちmax(1, file_score, line_score)以上の最小値
```

## 実装手順

1. Task-0006 config/frame contractとTask-0007 control storeを確定する。
2. 各Planeのmigration/storeとInbox uniqueness/Outbox leaseを実装する。
3. Core/Python/Rust UDS frame/ACK dispatcherとpeer/path validationを実装する。
4. 各supervisorへworkerを登録し、restart/shutdownを接続する。
5. crash matrix（before commit, after commit before ACK, after ACK）を統合試験する。

## 検証計画

- 各言語のunit tests、cross-process UDS integration、restart/duplicate/fault injection。
- `go test ./...`、`uv run pytest`、`cargo test`、`make check`、schema fixture validation。
- Ready/livenessを確認し、Task-0007のDB以外をCoreが直接開かない静的検査を行う。

## 未解決事項

- Memory/GovernanceのSQLite driver選定とmigration libraryは各言語の既存dependencyを確認してDEVで決めるが、transaction/ACK不変条件は変えない。

## main Agentレビュー

- [ ] 受け入れ条件が検証可能である。
- [ ] 設計観点と代替案を検討している。
- [ ] QA計画を作成できる。
- [ ] 見積もりが規則どおりである。
- [ ] DEV開始を承認した。
