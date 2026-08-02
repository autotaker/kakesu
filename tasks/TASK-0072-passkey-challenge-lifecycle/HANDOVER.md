---
task_id: "TASK-0072"
status: complete
completed_at: "2026-08-02T08:24:07Z"
candidate_commit: "3446398cb3271b20f9032709bd29322efa76d74a"
---

# TASK-0072 HANDOVER

## 成果

- request digest、`approve/deny`、operator、RP ID、origin、期限へopaque random challengeを束縛するbounded in-memory managerを追加した。
- verifier起動前の原子的予約により、成功、失敗、panic、同時試行、Closeの全経路でchallengeを一回限りにした。
- TASK-0071で使用済みのGo 1.25 APIとmodule契約の不一致を、`go` directiveの1行訂正で解消した。

## candidate-bound DEV証跡

| コマンド/テスト | 結果 |
|---|---|---|
| candidate gateのroot `make check` | `PASS` |
| `GOCACHE=/Users/autotaker/git/agent-harness/.build/go-cache go test -count=1 -race ./internal/approvalchallenge` | `PASS` |
| harness `make check` / `make distcheck` | `PASS` |
| root docs lint / `git diff --check` | `PASS` |

## 主要な変更

- 許可4パス、1,096 additions / 1 deletion。challenge package/test/READMEは1,095 additions、`go.mod`は`go 1.24`から`go 1.25`への1行だけを変更した。
- production constructorは`crypto/rand`とsystem clockを固定し、callerはchallenge、乱数、発行/期限時刻を注入できない。test seamはpackage内に閉じた。
- strictなrequest/digest/decision/operator/RP ID/exact HTTPS origin、TTL、capacityを検査し、expiryをconsumeより優先する。同一秒内を含むclock rollbackもraw UTC instantで拒否する。
- verifierへimmutable bindingとassertion copyを渡し、panic/errorを固定classへ正規化する。raw credential IDはmanager-owned copyからdomain-separated digestへ変換し、assertion、signature、public keyとともに保持しない。
- Closeはpendingを破棄し、restart相当の新managerは旧challengeを復元しない。verified結果はapprovalstate mutation、WebAuthn verification、grant又はpush authorizationを意味しない。

## 検証結果

- candidate gate root `make check`: `PASS`
- focused race、semantic/validation/random/collision/copy/failure/panic/expiry/rollback/reentrant/concurrent/Close/restart fixture: `PASS`
- harness check/distcheck、docs lint、task-check、scope/line/dependency/diff監査: `PASS`

## 判断・既知の制約

- 初回DEV個体が差分0で停滞したため同じPLAN/Task粒度の別個体へ切り替えた。再起動個体では本体、semantic test、negative/race testのcheckpointを分け、filesystemで進捗を観測した。
- DEV focused test後のharness checkが、TASK-0071の`os.Root.Rename`はGo 1.25 APIなのにmoduleが`go 1.24`のままという実回帰を検出した。安全なdescriptor-relative renameを弱めず、planning scopeを`go.mod`1行へ最小拡張して再検証した。
- productionの256-bit random tokenはpending集合との衝突をbounded retryする。消費済みtoken全履歴のtombstoneはmemoryを無制限化するため保持せず、crypto/randの再衝突確率を受容する。
- 実WebAuthn署名/RP ID hash/origin/UV/counter、credential登録/失効、Tailscale identity/Serve/Grant、HTTP/UI/CSRF、approvalstate mutation、実スマートフォン、push grantは未実装・未確認であり、hermetic PASSで代替しない。
