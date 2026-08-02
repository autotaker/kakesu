---
task_id: "TASK-0072"
change_class: "product"
status: approved
qa_agent: "qa-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-02T07:48:56Z"
revision: 1
implementation_reviewed_at: ""
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0072 QA PLAN

## 方針

このQA_PLANはTASK.mdの`Planning input packet`だけを期待値の正本として、DEV開始前に独立作成した。候補案のQAでは、同一`candidate_commit`を対象に、下表のfocused rerunを一回だけ独立実行し、そのテストが各失敗を検出できること、テストの弱体化がないこと、候補差分が許可3パス・新規dependencyなしであることを証跡レビューする。QAはcallbackを実WebAuthn検証済みとは扱わず、verified resultをapproval state mutation又はpush authorizationの証拠にはしない。

実WebAuthn authenticator/署名、Tailscale identity/Serve/Grant、HTTP/API/UI/session/cookie/CSRF、実スマートフォン操作は、この候補に含まれず安全な隔離環境も未指定である。これらは`live-e2e`としてblockedのまま記録し、focused-rerun又はevidence-reviewのPASSで代替しない。

## 受け入れ条件との対応

| ケース ID | AC-ID | 観測方法 | 実施モード / 理由 |
|---|---|---|---|
| QA-001 | AC-1 | production constructor/public APIを検査し、公開APIからrandom source/clockを注入できないこと、test-only seamがproduction経路から隔離されることを候補diffとテストで確認する。固定enum、request/digest/operator/RP ID/origin、TTL、capacityの拒否と、32-byte以上・base64url・opaqueなchallengeを確認する。 | `focused-rerun` / hermeticなpackage testで入力検査、unbiased-random seam、opaque encodingを再現でき、外部serviceを要しない。 |
| QA-002 | AC-2 | callbackに渡るbindingがrequest ID、canonical digest、decision、operator、RP ID、exact HTTPS origin、発行/期限に一致すること、assertionをcopyで渡すこと、verified resultがrequest/digest/decision/operator/stable nonsecret credential ID/verified timeだけをcopy ownershipで返すことを確認する。入力・callback側・結果側sliceのmutationがmanager内部状態を変更しないnegative testを確認する。 | `focused-rerun` / exact binding、copy ownership、非漏えい結果はdeterministic unit testで完全に検出可能である。 |
| QA-003 | AC-3 | consume開始のmutex下予約を、成功、verification failure、panic、並行first attempt、unknown/replayの各経路で確認する。各経路の後で再consumeが拒否され、failure/panicが入力・callback詳細を含まない固定error classへ正規化されることを確認する。 | `focused-rerun` / race detector付きのbounded parallel fixtureが予約の原子性、failure/panic/replayとfixed non-leak errorを検出できる。 |
| QA-004 | AC-4 | expiryをconsumeより優先し、期限ちょうど、purge後、capacity回収、clock rollback、Close競合、Close後、new manager（restart相当）で旧challengeがfail closedとなることを確認する。pending challengeがCloseで破棄され、永続化・復元を行わないことを候補diffとテストで確認する。 | `focused-rerun` / injected test clockとbounded fixtureでexpiry priority、clock rollback、capacity/purge、Close/restartを再現できる。 |
| QA-005 | AC-5 | package API、README、候補差分をレビューし、lifecycleの責務とrestart/失敗時の新challenge発行を明示しつつ、実WebAuthn verification、Tailscale identity、verified decision API、approval state mutation、push authorizationへ昇格していないことを確認する。実WebAuthn/Tailscale/HTTP/実スマートフォンのlive確認は未実施理由とともにblockedを維持する。 | `evidence-review` / スコープ非昇格と文書上の信頼境界はcandidate-bound diff/READMEの独立監査が適切であり、未指定の実環境依存検証を代替できない。 |
| QA-006 | AC-6 | `git diff --check`、候補の名前付きdependency/config/build/generated artifact追加なし、許可3パスだけ、additionsがおよそ700〜1,100であることを確認する。DEVのharness check/distcheck、root `make check`、docs lintの実行証跡と終了状態を監査する。 | `evidence-review` / リポジトリ全体検査はDEV実行証跡をcandidateに束縛して独立監査し、QAは下記の高価値かつdeterministicなpackage rerunを一回だけ実施する。 |

## focused-rerun（候補固定後に一回だけ）

```sh
cd tools/dev-agent-harness && go test -race ./internal/approvalchallenge
```

期待結果は終了コード0である。QAは候補の`challenge_test.go`が、少なくとも次を実行して失敗を検出することを確認する。

- production randomと公開APIから隔離されたrandom/clock test seam
- request/digest/decision/operator/RP ID/exact HTTPS origin/TTL/capacityのexact bindingと、assertion/result copy ownership
- mutex下のfirst-attempt reservation、success/failure/panic/concurrent/replay後のone-shot性
- expiry priority、期限ちょうど、clock rollback、capacity/purge、Close競合とrestart相当のnew manager
- callback failure/panic、invalid・unknown・replayed・closedの固定かつ入力値を漏らさないerror class

race報告、panicの逸脱、timeout、又は上記のnegative pathを実行しない/弱める変更はFAILとして分類する。`go test -race`のPASSは実WebAuthn、Tailscale、HTTP、実スマートフォンのlive-e2eを証明しない。

## live-e2eのblocked記録

| ケース ID | 依存する実環境 | 状態 | unblock条件 | 代替PASS |
|---|---|---|---|---|
| LIVE-001 | 実WebAuthn authenticator、RP/origin、credentialと署名検証 | `blocked` | 承認済み隔離環境、登録済みtest credential、明示したcleanup手順、後続の実verifier実装 | なし |
| LIVE-002 | Tailscale identity、Serve/Grant/identity header | `blocked` | 承認済みtailnet test環境とidentity境界を接続する後続Task | なし |
| LIVE-003 | HTTP/API/UI/session/cookie/CSRF | `blocked` | 後続のHTTP接続実装、隔離endpoint、rollback/cleanup手順 | なし |
| LIVE-004 | 実スマートフォンでのpasskey操作 | `blocked` | 承認済み実機手順、test account、recovery/cleanup手順 | なし |

## 境界・異常・回帰

- candidate commit、DEV `make check`の実行結果、focused rerun、各ケースのPASS/FAIL/blocked理由を同一candidateへ結び付ける。
- verifier callbackの成功だけではTASK-0071 storeをapproved/deniedへ遷移させないこと、challengeをdisk/log/env/external DBへ保持しないことを差分で確認する。
- FAILはDEV faultと仮定しない。race/flaky fixture、test expectation、環境・依存、候補実装、証跡不足のいずれかに根拠を結び付けて分類し、live-e2e blockedをPASSへ読み替えない。
- 高リスク信号、candidate-bound証跡不足、diffの許可範囲逸脱、又は検出能力のないtestは`evidence-review` PASSを禁止し、Mainへ差し戻す。

## 実装後の再確認

- [ ] `candidate_commit`がHANDOVERに一意に固定され、QA対象と一致する。
- [ ] 上記focused rerunを一回実行し、結果と失敗検出能力を記録した。
- [ ] QA-001〜QA-006を独立に判定し、LIVE-001〜LIVE-004をblockedのまま記録した。
- [ ] 実装差分、DEV `make check`、harness check/distcheck、docs lint、root `make check`のcandidate-bound証跡を確認した。
- [ ] 期待結果または範囲を変更した場合、main Agentの承認を得た。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | 2026-08-02 | qa-agent-terra-medium | TASK.mdのPlanning input packetだけから独立QA計画を作成 | `main-agent-sol-high` |
