---
task_id: "TASK-0066"
status: complete
completed_at: "2026-08-02T03:37:49Z"
candidate_commit: "3b9fe7cbd3adcb844744d28ef7f04bd83c08d3a6"
---

# TASK-0066 HANDOVER

## 成果

- peer-bound egress control socketへexact `GET /v1/proxy-ca`を追加し、valid Authorityのcertificate-only公開CA PEMだけを固定200/closeで返す。
- `controlclient.ProxyCA`が同じsocketへ一回だけ接続し、strict bounded responseと独立X.509検証を通過したcaller-owned PEM copyだけを取得する。
- server/client双方が単一canonical PEM、self-signed ECDSA P-256 CA、Basic Constraints、CertSign、validityを独立確認し、private/multiple/trailing/invalid materialを固定拒否する。

## candidate-bound DEV証跡

| コマンド/テスト | 結果 |
|---|---|---|
| candidate gateのroot `make check` | `PASS` |
| `GOCACHE=$PWD/.build/go-cache go test -race ./internal/connectsession ./internal/controlclient ./internal/egressservice` | `PASS` |
| harness `make check` | `PASS`（live testは既定`SKIP`） |
| harness `make distcheck` | `PASS` |
| `make task-check TASK=TASK-0066` | `PASS` |
| `git diff --check` | `PASS` |

## 主要な変更

- 承認済み6パス、611 additions / 19 deletions、計630 changed lines。計画目安約800〜1,100行を下回ったが、既存strict transportを再利用できた結果なので水増ししていない。
- `connectsession.Authority`は既存leaf `Issue`にcertificate-only fresh-copy accessorだけを追加し、private key/bundle/path APIを広げていない。
- GETはexact byte matchのzero-body routeであり、既存CONNECT、Issue、Revokeのparser、response、controller/handler到達意味を維持する。

## 検証結果

- candidate gate root `make check`: `PASS`
- focused race（connectsession/controlclient/egressservice）: `PASS`
- harness check/distcheck、Task check、diff check: `PASS`

## 判断・既知の制約

- candidate gate初回はREADMEのterminology 7語/textlint 25件でFAILし、TASK-0064の集約runnerが全件を一回で表示した。READMEだけを既存日本語文体へ一括修正し、docs lint PASS後のcandidate gateでroot checkをPASSした。
- source configureが作った未追跡socket artifactはcandidate前に除去し、製品差分は6許可パスだけである。
- 実OS Unix socket/permissions/別UID、実Git/libcurl trust、GitHub/OpenAI/DNS/TLS、systemd/VPS、後続launcherのtrust-file lifecycleは未確認であり、live-e2eで別途扱う。
