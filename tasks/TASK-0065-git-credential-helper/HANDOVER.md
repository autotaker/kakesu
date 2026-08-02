---
task_id: "TASK-0065"
status: complete
completed_at: "2026-08-02T03:01:56Z"
candidate_commit: "487fbbb8cac3b17b16acadbd6930342465a784fa"
---

# TASK-0065 HANDOVER

## 成果

- `git-credential-dev-agent`をGit credential helperとして実装し、exact HTTPS GitHub repositoryに限って既存control socketからsingle-use Git-read Opaque capabilityを取得する。
- `get`成功時は固定usernameとOpaque handleだけを返し、全拒否時は`quit=true`で他helper/promptへの探索を停止する。`store`/unknownはbounded no-op、`erase`は一handleだけを失効する。
- control clientはconfigure固定Unix socketへ一回だけ接続し、既存serverと一致するIssue/Revoke wire、bounded strict response、deadline/close、固定非漏洩errorを実装する。

## candidate-bound DEV証跡

| コマンド/テスト | 結果 |
|---|---|---|
| `make check`（candidate固定直前） | `PASS` |
| `GOCACHE=$PWD/.build/go-cache go test -race ./cmd/git-credential-dev-agent ./internal/gitcredential ./internal/controlclient` | `PASS` |
| 同focused race suite `-count=10` | `PASS` |
| harness `./configure && make check && make distcheck`（`/tmp` exact candidate copy） | `PASS` |
| `make task-check TASK=TASK-0065` | `PASS` |
| `git diff --check` | `PASS` |

## 主要な変更

- 承認済み7パス、1,012 additions / 2 deletions。新規`controlclient`/`gitcredential` sourceとtestが990行、既存main/Makefile.in/READMEが+22/-2行である。
- credential inputは4 KiB上限、blank又はEOF終端、exact `protocol`/`host`/`path`だけを受理し、URL、duplicate、CR/NUL、overlong、blank後extra、非canonical repositoryをdial前に拒否する。
- helper専用targetへconfigure済み`runstatedir/dev-agent-harness/egress.sock`をlinkし、environment/CLI/input/config/cwdによるoverrideを持たない。

## 検証結果

- root `make check`: `PASS`
- focused raceと10回反復: `PASS`
- harness configure/build/check/distcheck: `PASS`（live testは設計どおり`SKIP`）
- Task check、7-path scope、line budget、`git diff --check`: `PASS`

## 判断・既知の制約

- root check初回のnode依存未配置とsandbox DNSはenvironment failureに分類し、承認済み依存取得後に同じcheckをPASSした。
- 初回focused hangはtest fixtureの固定issue body長誤算とpeer close後のdeadline再設定が原因で、製品wireを変えずheader-derived fixtureと単一bounded response readへ修正し、10回反復で安定性を確認した。
- 実OS Unix socket/permissions/別UID、実Git helper invocation、GitHub token/DNS/TLS、GitHub read、systemd/VPSは未確認であり、QA-007を`blocked`/`not-run`のまま後続live境界へ分離する。
