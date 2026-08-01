---
task_id: "TASK-0048"
status: complete
completed_at: "2026-08-01T11:14:40Z"
candidate_commit: "4536a196dbabf65d98ac7f3884fa08e1acce4308"
---

# TASK-0048 HANDOVER

## 成果

- 既存のegress transactionとrequest単位Forwarderを、呼出しごとのprivate sinkで同期合成する`brokerexchange.Exchange`を追加した。
- 成功時だけ縮退済みresponseの独立copyを返し、全失敗をzero responseと固定`exchange-denied`へ畳んだ。
- capabilityの認可前非消費とConsume後失敗時の消費維持、単回上流試行、並行response分離をhermetic testで固定した。

## candidate-bound DEV証跡

| コマンド/テスト | 結果 |
|---|---|---|
| `GOCACHE=$PWD/.build/go-cache go test -race ./internal/brokerexchange` | PASS（両provider、認可/消費境界、固定error、copy、並行隔離） |
| `make check`（`tools/dev-agent-harness`） | PASS |
| `make distcheck`（`tools/dev-agent-harness`） | PASS |
| `make lint-docs`（repository root） | PASS |
| `git diff --check` | PASS |
| candidate launcherのroot `make check` | PASS |

## 主要な変更

- `internal/brokerexchange`に検証済み依存と上限だけを保持するimmutable `Exchange`を追加した。
- 各`Do`でcapture sink、Forwarder、Transactionを生成し、既存policy/Authorization/capability/credential/response検査を再実装せず委譲した。
- real Policy/Registryとfake resolver/RoundTripperで、GitHub/OpenAI成功、非消費/消費境界、no retry、non-leak、raceを検証した。
- READMEへin-memory exchangeの責務と、HTTP入口・production wiring・live E2Eが対象外であることを追記した。

## 検証結果

- candidate launcher root `make check`: PASS
- focused race test、harness `make check`、harness `make distcheck`、root `make lint-docs`、`git diff --check`: PASS
- base `419bac724da5fd0aa3d6e6615001a16e684f9e70`からcandidateまでの製品差分は許可された3ファイル、追加558行・削除0行で1,000行以下。

## 判断・既知の制約

- Exchangeは既存Transaction/Forwarderの意味を複製せず、call-local合成と成功response captureだけを所有する。
- resolver又はtransport到達後の失敗ではcapabilityをrollbackせず、同じhandleの再試行も下位依存へ再到達しない。
- 実GitHub/OpenAI、実credential、DNS/TLS/system trust、Agent proxy、production resolver/transport wiringは未実施であり、QA-009のlive E2E blockedを他のPASSで代替しない。
