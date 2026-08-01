---
task_id: "TASK-0041"
status: complete
completed_at: "2026-08-01T06:25:47Z"
candidate_commit: "c369ec46246be0047103ceea9d7a5e7ea71188b6"
---

# TASK-0041 HANDOVER

## 成果

- TASK-0039のHTTP allowlist scopeとTASK-0040のOpaque capability消費を、一つのfail-closedなtransactionへ接続した。
- Credential-bearing requestを戻り値にせず、capability消費後にだけtrusted Forwarderへ同期的に一回渡す境界を実装した。

## candidate-bound DEV証跡

| コマンド/テスト | 結果 |
|---|---|---|
| `GOCACHE=/tmp/task-0041-dev-gocache go test -race ./internal/egresspolicy ./internal/egresstransaction` | PASS |
| `./configure && make check`（harness） | PASS |
| `make distcheck`（harness最終機能tree） | PASS |
| `make check`（candidate launcher、Mainで一回） | PASS |
| `node scripts/lint-docs.mjs` / `git diff --check` | PASS |
| `git diff --numstat 09570d4...c369ec4` | PASS — 追加673、削除19、合計692行 |

## 主要な変更

- `Policy.Evaluate`を追加し、既存allow判断と同じ評価からcanonical provider/repository/operation/destination hostを返す。既存`Authorize`はこのAPIへ委譲してdecision/error互換を維持する。
- `egresstransaction.New/Execute`、Rules/Subject/Request/PreparedRequest、CredentialResolver、Forwarderを追加した。
- Execute順序をpolicy allow → Authorization一値抽出 → Registry Consume → resolver一回 → Credential検証 → Forwarder同期一回へ固定した。
- OpenAIは`Bearer cap_...`、GitHubは`Bearer cap_...`と`token cap_...`だけを受理する。Credentialは設定上限内のvisible ASCIIだけを上流Bearerへ変換する。
- resolver/Forwarder失敗時もcapabilityを戻さず、retryしない。固定errorへ集約し、handle/Credential/detailを返さない。

## 検証結果

- `make check`: PASS（candidate `c369ec46246be0047103ceea9d7a5e7ea71188b6`）
- focused race test、harness check/distcheck、文書lint、差分check: PASS
- candidate差分: 許可5 files、追加673・削除19、合計692行（上限1,200以下）

## 判断・既知の制約

- Transactionは実Credential sourceやHTTP Forwarderを実装しない。constructorへ注入するresolver/Forwarderはbroker内のtrusted adapterであり、Agent側へ公開しない後続境界である。
- PreparedRequestはForwarder同期呼出にだけ渡す。Transaction stateとExecute戻り値へCredential-bearing値を保持しないが、注入されたForwarder自身の秘密処理・保持は後続実装の責務である。
- 実Credential、file/network/TLS/DNS/HTTP listener/upstream responseのPASSは主張しない。
- planning reviewの初回はCredential受渡し主体の曖昧さを検出し、値を返す案から同期Forwarder handoffへ修正した。Reviewerが指示外で実行したroot checkのPyPI DNS failureは実装証跡に含めず、修正後planning reviewは読み取りだけでPASSした。
