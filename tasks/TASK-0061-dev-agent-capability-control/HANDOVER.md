---
task_id: "TASK-0061"
status: complete
completed_at: "2026-08-02T01:01:43Z"
candidate_commit: "6ca73fa8ef362d465a723358af90251e985ad337"
---

# TASK-0061 HANDOVER

## 成果

- 既存egress Unix socketに、通常CONNECTと明確に分離した発行・失効control protocolを追加した。
- kernel peer binderがcontextへ固定したAgent instance/UID/workspaceだけを主体とし、GitHub allowlist/OpenAI設定gateの短命・一回限りCapabilityを発行できるようにした。
- controlと既存egress transactionが同一Registryを共有し、subject-bound失効、消費、再利用拒否を一つのライフサイクルで行う。

## candidate-bound DEV証跡

| コマンド/テスト | 結果 |
|---|---|---|
| `make check`（candidate固定直前の最終実行） | `PASS` |
| `git diff --check` | `PASS` |

## 主要な変更

- candidateは`6ca73fa8ef362d465a723358af90251e985ad337`、treeは`e829000833956a50f9e993533def97dbbaca73e5`。
- 承認済み8パス、1,013 additions / 49 deletions。新socket/listener/unit/client/helper、依存、永続化、Kakesu runtime、Schemaの差分はない。

## 検証結果

- `make check`: `PASS`
- `cd tools/dev-agent-harness && GOCACHE=$PWD/.build/go-cache go test -race ./internal/capabilitycontrol ./internal/connectsession ./internal/egressservice ./internal/capability`: `PASS`
- `cd tools/dev-agent-harness && GOCACHE=$PWD/.build/go-cache go test ./...`: `PASS`
- terminology検査、docs lint、`git diff --check`: `PASS`

## 判断・既知の制約

- negative test追加後の初回FAILは、成功失効fixtureが非canonical base64url handleを使ったtest fixture不備と分類し、canonical fixtureに修正後にraceを含め再実行PASSした。
- 初回candidate `make check`はREADME内の英語用語頻度で停止した。glossaryは増やさず日本語表記へ整理し、terminology/docs lintを個別PASS後にcandidate固定を再実行した。初回停止時にcandidate commitは作られていない。
- 実GitHub/OpenAI、DNS/TLS、NSS/別UID、systemd socket、VPS配置はこのTaskの対象外であり、hermetic PASSでlive事実を主張しない。
