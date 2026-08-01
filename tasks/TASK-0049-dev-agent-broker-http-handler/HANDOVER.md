---
task_id: "TASK-0049"
status: complete
completed_at: "2026-08-01T11:48:55Z"
candidate_commit: "d5b94f2123a48bb5884f797af9bf6dac82de3cb7"
---

# TASK-0049 HANDOVER

## 成果

- TLS終端済みHTTP/1.1 origin-form requestを、context-onlyのtrusted呼出元解決と既存Exchangeへ一回だけ接続する`brokerhttp.Handler`を追加した。
- protocol/framing/body不整合、resolver/Exchange失敗、invalid responseを、固定headerだけのempty 403へ畳んだ。
- 成功時は2xx、縮退content type/body、固定no-store/nosniff、正確なContent-Lengthだけを返す。

## candidate-bound DEV証跡

| コマンド/テスト | 結果 |
|---|---|---|
| `GOCACHE=$PWD/.build/go-cache go test -count=1 -race ./internal/brokerhttp` | PASS（HTTP mapping、拒否、両provider、copy、並行隔離） |
| `make check`（`tools/dev-agent-harness`） | PASS |
| `make distcheck`（`tools/dev-agent-harness`） | PASS |
| `make lint-docs`（repository root、Task worktree） | PASS |
| `git diff --check` | PASS |
| candidate launcherのroot `make check` | PASS |

## 主要な変更

- `internal/brokerhttp`へExchange/SubjectResolver interface、immutable Rules/New/Handlerを追加した。
- HTTP/1.1 origin-form、構造的method/authority、no query/raw path、known Content-Length、no transfer/trailer/upgradeをExchange前に検査する。
- resolverへrequest contextだけを渡し、Host/methodのprovider意味とpolicy/capability/credential検査は既存Exchangeへ委譲する。
- fake resolver/Exchangeとreal Exchange + fake上流dependencyで両provider、単回性、fixed deny、header allowlist、raceを検証した。
- READMEへHandler責務とlistener/TLS/production identity/live E2Eの対象外境界を追記した。

## 検証結果

- candidate launcher root `make check`: PASS
- focused uncached race、harness `make check`/`make distcheck`、Task worktree `make lint-docs`、`git diff --check`: PASS
- baseからcandidateまでの製品差分は許可された3ファイル、追加704行・削除0行で1,000行以下。

## 判断・既知の制約

- Handlerはprovider host/method allowlistを持たず、CONNECTとHTTP構造だけを入口で拒否し、provider意味は既存Policyへ委譲する。
- HTTP/2、absolute-form、chunked、listener/TLS、production identity resolver、実gh/OpenAI client/provider、実credential、DNS/system trust、Agent network namespaceは未実施である。
- live E2E blockedはhermetic PASSで代替しない。response write failureはretry又は診断本文を生成しない。
