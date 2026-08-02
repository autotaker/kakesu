---
task_id: "TASK-0073"
status: complete
completed_at: "2026-08-02T08:54:54Z"
candidate_commit: "78022718d8897ee71449d1a50564de6da74a4df4"
---

# TASK-0073 HANDOVER

## 成果

- durable pending requestとone-shot challenge verificationを安全な順序で接続するcoordinatorを追加した。
- Beginはstoreからrequest ID/digestを導出し、Completeは固定verifierのConsume後にexact Approve/Denyを一度だけ実行する。
- verified結果単独では成功を返さず、durable transition成功後だけrecordとcredential stable IDを返す。

## candidate-bound DEV証跡

| コマンド/テスト | 結果 |
|---|---|---|
| candidate gateのroot `make check` | `PASS` |
| `GOCACHE=/Users/autotaker/git/agent-harness/.build/go-cache go test -count=1 -race ./internal/approvaldecision` | `PASS` |
| harness `make check` / `make distcheck` | `PASS` |
| docs lint / task-check / `git diff --check` | `PASS` |

## 主要な変更

- 許可3パス、1,072 additions / 0 deletions。新規dependency/config/generated artifactなし。
- production constructorはconcrete store、challenge manager、trusted verifierを固定し、Begin/Complete callerへdigest、verifier、clock、state seamを公開しない。
- Beginはdurable Getがexact pendingの場合だけ、そのrecordのrequest ID/digestとcallerのdecision/operator/RP ID/originをIssueへ渡す。Issue出力の全binding、challenge、時刻も再確認する。
- Completeはverifier前にchallengeをone-shot consumeし、verified decisionからApprove又はDenyだけを選ぶ。credential stable IDは`sha256:`+lowercase64hexを厳密検査する。
- verification/state/persistence/poison failure、terminal/expiry/digest mismatch、opposed challenge競合ではresultを空にし、fallback、challenge復活、自動再発行、成功推測をしない。durable storeのpending transitionがfirst-winsの唯一の正本である。

## 検証結果

- candidate gate root `make check`: `PASS`
- real approvalstate/approvalchallenge approve/deny、durable Get、replay、response-loss reconciliation fixture: `PASS`
- ordering/failure/panic/terminal/digest/persistence/poison/first-wins/concurrency/copy/non-leak race fixture: `PASS`
- harness check/distcheck、docs lint、task-check、scope/dependency/diff監査: `PASS`

## 判断・既知の制約

- DEVは本体＋semantic test、real integration、negative/race＋READMEの3 checkpointに分け、filesystem差分で進捗を観測した。追加gateやTask分割は行っていない。
- README 21行の初回docs lintは既存用語表記48件を検出した。意味を変えず許可README内だけを修正し、再実行とroot checkをPASSした。
- trusted verifierは後続production wiringがactual WebAuthn verifierとして構成する前提であり、このTaskはWebAuthn暗号検証、credential lifecycle、Tailscale identity、HTTP/UI/CSRF、auditを実装しない。
- durable `approved/denied`はgrant、push authorization、実push成功を意味しない。実WebAuthn/Tailscale/スマートフォンと外部作用は後続live E2Eまで未確認である。
