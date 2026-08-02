---
task_id: "TASK-0067"
status: complete
completed_at: "2026-08-02T04:15:00Z"
candidate_commit: "31213784558427e2e38433a03e0b3a09a0051dd8"
---

# TASK-0067 HANDOVER

## 成果

- 固定IPv4 loopbackのephemeral endpointを一回だけ作り、accepted connectionごとにtrusted Unix egress socketへ一回だけ接続する`proxybridge`を追加した。
- slot-before-Accept、dial timeout、opaqueな双方向streamとhalf-close、cancel/accept-failure時のclose/drainを固定した。
- bridgeはHTTP/TLS/認証を解釈せず、既存egress serviceを認可と接続元bindingの正本に保つ。

## candidate-bound DEV証跡

| コマンド/テスト | 結果 |
|---|---|---|
| candidate gateのroot `make check` | `PASS` |
| `GOCACHE=$PWD/.build/go-cache go test -count=1 -race ./internal/proxybridge` | `PASS` |
| harness `make check` | `PASS`（live testは既定`SKIP`） |
| harness `make distcheck` | `PASS` |
| `git diff --check` | `PASS` |

## 主要な変更

- 承認済み3パス、999 additions / 0 deletions、計999 changed lines。
- production listenerはexact `tcp4` / `127.0.0.1:0`だけを使い、返されたaddressをIPv4 loopbackと非zero portまで再検証してcanonical endpointを返す。
- connection-localなdial/copy/close failureは他connection又は公開診断へ拡大せず、unexpected accept failureだけがactive connectionをcancel/drainして固定server errorとなる。

## 検証結果

- candidate gate root `make check`: `PASS`
- focused race、harness check/distcheck、diff check: `PASS`

## 判断・既知の制約

- candidate gate初回はREADMEのterminology 5語/textlint 26件、二回目は日本語化で閾値へ達したterminology 1語だけで停止した。集約docs lintで全件PASSを確認したREADMEだけを修正し、製品コード/testを変えず三回目のgateでPASSした。
- 実OS network namespace/loopback isolation、Unix socket permission/peer UID、実Git/`gh`/OpenAI proxyとCA trust、launcher/child lifecycle、systemd/VPSは未実装又は未確認であり、live E2Eで別途扱う。
