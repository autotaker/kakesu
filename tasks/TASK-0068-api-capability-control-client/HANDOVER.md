---
task_id: "TASK-0068"
status: complete
completed_at: "2026-08-02T04:53:53Z"
candidate_commit: "dc17338db42713ebdae76497e7521fd42ce29020"
---

# TASK-0068 HANDOVER

## 成果

- `controlclient`へGitHub REST read/OpenAI Responses text用の明示issue操作を追加し、既存Git-readとstrict exchangeを共有した。
- peer-bound controllerはAPI scopeへ固定16 uses、Git-readへ従来どおり1 useを発行する。scope mismatchはbudgetを消費しない。
- production compositionの共有Registry integrationもAPI 16回成功、17回目拒否、revoke後拒否へ更新した。

## candidate-bound DEV証跡

| コマンド/テスト | 結果 |
|---|---|---|
| candidate gateのroot `make check` | `PASS` |
| `GOCACHE=$PWD/.build/go-cache go test -count=1 -race ./internal/controlclient ./internal/capabilitycontrol ./internal/gitcredential ./internal/egressservice` | `PASS` |
| harness `make check` | `PASS`（live testは既定`SKIP`） |
| harness `make distcheck` | `PASS` |
| 集約 `make lint-docs` | `PASS` |
| `git diff --check` | `PASS` |

## 主要な変更

- 承認済み6パス、646 additions / 121 deletions、計767 changed lines。
- client public APIはprovider/operation/model/bodyを受けず、GitHub RESTとOpenAIのliteral requestだけを構築する。transport/framing/handle/EOF failureは固定errorへ畳み、retry/fallbackしない。
- controllerのuses選択はexplicit fail-closed switchで、unknown operationは0 usesとなりRegistry発行に失敗する。

## 検証結果

- candidate gate root `make check`: `PASS`
- focused race、harness check/distcheck、docs lint、diff check: `PASS`

## 判断・既知の制約

- 最初のharness checkでegressservice integrationの旧REST single-use期待だけがFAILした。API 16-useの直接回帰と分類し、Task/PLAN/QA_PLANを6許可パスへ補正後、同integrationを16回成功・17回目拒否・revoke後拒否へ更新して全検査PASSとした。
- 実credential、実GitHub/OpenAI、実`gh`/SDK、DNS/TLS、Unix socket permission/別UID、systemd/VPSは未確認であり、live E2Eとして別途扱う。
