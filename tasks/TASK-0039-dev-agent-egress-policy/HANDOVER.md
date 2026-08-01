---
task_id: "TASK-0039"
status: complete
completed_at: "2026-08-01T05:05:59Z"
candidate_commit: "709b84194115e11c9a301c78fe93341ee7d6427d"
---

# TASK-0039 HANDOVER

## 成果

- GitHub repository-scoped REST readとOpenAI Responses text requestをdefault denyで判定するpure Go packageを追加した。
- canonical URL、strict JSON body、Rules copy、固定deny errorを一つの副作用なし境界にまとめた。

## candidate-bound DEV証跡

| コマンド/テスト | 結果 |
|---|---|---|
| `go test ./...`（harness） | PASS |
| `./configure && make check && make distcheck`（harness） | PASS |
| `make check`（candidate launcher） | PASS |
| `git diff --check` | PASS |

## 主要な変更

- `egresspolicy.New`がrepository/model allowlistとbody/token上限を検証してcopyする。
- `Authorize`がGitHubのcanonical `GET`/`HEAD`とOpenAIのstrictな`POST /v1/responses`だけをprovider別decisionで許可する。
- table-driven testで代表allow/deny、入力非漏洩、Rules/Request/body不変性、nil/zero policyを検出する。
- READMEへpolicy coreと後続proxy/Credential brokerの責務境界を追記した。

## 検証結果

- `make check`: PASS（candidate `709b84194115e11c9a301c78fe93341ee7d6427d`）
- harness check/distcheck: PASS
- candidate差分: 3 files、596行追加

## 判断・既知の制約

- 実network、TLS、Credential、proxy、redirectは対象外であり、実API通信のPASSを主張しない。
- allow decisionは後続proxyが明示的に接続するまで外向き通信を許可しない。
- candidate作成前の全体検査でREADMEの用語・文書lintに失敗したため、glossaryや規則を増やさず文面だけを是正し、同じ全体検査をPASSした。
