---
task_id: "TASK-0040"
status: complete
completed_at: "2026-08-01T05:52:08Z"
candidate_commit: "072502c7363d0f0ebcf964fe820dd0fae2f35eff"
---

# TASK-0040 HANDOVER

## 成果

- 実Credentialを含まない`cap_...` handleを短命scopeへ束縛するin-memory Registryを追加した。
- digest-only保持、provider固定scope、期限/使用回数/失効世代の原子的な検証と消費を実装した。

## candidate-bound DEV証跡

| コマンド/テスト | 結果 |
|---|---|---|
| `GOCACHE=/tmp/task-0040-dev-gocache go test -race ./internal/capability` | PASS |
| `./configure && make check`（harness） | PASS |
| `make distcheck`（最終tree、展開先の`make check`を含む） | PASS |
| `make check`（candidate launcher） | PASS |
| `git diff --check` | PASS |

## 主要な変更

- `Registry.New/Issue/Consume/Revoke/AdvanceRevocationEpoch`と固定errorを追加した。
- 32 byteのcrypto entropyをpaddingなしbase64url handleへ変換し、Registry mapにはSHA-256 digestだけを保持する。
- GitHub REST readとOpenAI Responses textの二scopeだけを発行し、subject/workspace/repository/operation/hostを完全一致で照合する。
- mutex下でexpiry、policy version、失効世代、remaining usesを判定し、成功だけを一回消費する。1-useの並行Consumeはexactly oneだけが成功する。
- production clockはUTC wall timeとmonotonic elapsedを分離し、wall rollbackでTTLを延長せず、wall forwardでは早期にfail-closedとする。呼出元へ返す時刻だけUTCへ変換する。
- READMEへrestart時の全handle失効とproxy/Credential/network非対象を追記した。

## 検証結果

- `make check`: PASS（candidate `072502c7363d0f0ebcf964fe820dd0fae2f35eff`）
- race test、harness check、最終tree distcheck: PASS
- candidate差分: 3 files、759行追加

## 判断・既知の制約

- Registryはin-memoryであり、process restart後の復元や複数process共有を行わない。entry消失は全handleが無効になるfail-safe動作である。
- Grantは実Credential取得や外向き通信を許可せず、後続broker/proxyが別のsecurity boundaryとして接続する。
- candidate作成前のcheap lintでREADME表記を是正した。最初のcandidate launcherは`epoch`の用語出現閾値だけで停止し、glossaryを増やさず日本語表記へ直して同じ全体検査をPASSした。
- 旧candidate `cb26908d`はReviewerがwall-clock rollbackによるTTL延長を検出してFAILした。結果を持ち越さずmonotonic elapsedとwall-forward fixtureを追加し、本candidateで全検査を再実行した。
