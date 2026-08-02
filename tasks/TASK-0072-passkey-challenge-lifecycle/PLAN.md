---
task_id: "TASK-0072"
change_class: "product"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "sol-high"
approved_dev_profile_reason: "認可前段の一回限り capability を、乱数、期限、clock rollback、panic、Close、競合の下で fail-closed にする高リスク境界である。"
approved_dev_profile_risk_signals:
  - "challenge の再利用又は別 request/decision/operator/origin への束縛取り違えが本人確認の前提を崩す"
  - "consume と Close、expiry、clock rollback、verifier panic の競合で once-only を失い得る"
  - "assertion、challenge、credential public key 又は入力値を error/result/長寿命 state に漏らしてはならない"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-02T07:48:56Z"
classification_approved_by: ""
classification_approved_at: ""
classification_approval_reason: ""
# safety_contract_version: 2
# safety_contract_planned_paths: []
# safety_contract_generated_paths: []
---

# PLAN: Passkey challenge one-time lifecycle

## 根拠・分類・実行境界

唯一の要求根拠は `TASK.md` の `Planning input packet`、そこに固定された REF-1〜REF-3、および TASK-0071 の main merge `84ce39263edfb1c642e4c02a2f464d7c2a44e8b7` である。新しい Go package と unit test、README を追加するため `change_class` は `product` とする。本 PLAN は設計証跡だけである。Main が本 PLAN と Task-first の独立 `QA_PLAN.md` を承認するまで DEV を開始しない。Main だけが stage、candidate commit/tree の固定、merge、push、completion の no-ff 検査を行う。

DEV は `dev-sol`/high（`sol-high`）を使用する。DEV は一度だけ製品差分だけの candidate を作り、REVIEW と QA は HANDOVER に固定した同じ candidate から、相互の PASS を待たず独立に開始する。candidate に修正が入れば Main が影響した focused test を再実行する。

candidate は次の3許可パスだけを変更し、追加・削除合計を約700〜1,100行（production code は概ね250〜400行、残りは race/negative/panic/ownership test と README）に収める。

- `tools/dev-agent-harness/internal/approvalchallenge/challenge.go`
- `tools/dev-agent-harness/internal/approvalchallenge/challenge_test.go`
- `tools/dev-agent-harness/README.md`

stdlib と既存 Go module だけを使い、新規 dependency、config/build/deploy/generated artifact、Kakesu runtime/Schema は変更しない。`approvalstate` は読み書きも import もしない。HTTP/API/UI/session/cookie/CSRF、Tailscale Serve/identity header、通知、credential 登録・失効・recovery、WebAuthn の clientData/authenticatorData/signature/RP-ID-hash/origin/UV/counter 検証、disk/log/environment/DB persistence、multi-process/host 共有、push grant/consume/old-SHA/Git/audit は候補外である。

## package contract・lifecycle・信頼境界

`internal/approvalchallenge` は一 process の bounded manager だけを所有する。公開面は、固定 error class、`Rules`、時刻を含まない入力`Request`、immutable `Binding`/`Issued`/`Verified`、`New(rules)`、`Issue(request)`、`Consume(challenge, assertion, verifier)`、`Close()` に限定する。production `New` は `crypto/rand.Reader` と `time.Now` を固定使用し、random reader と clock の注入は package-private constructor/dependencies に閉じ、公開 caller はchallenge、発行/期限時刻、clock、乱数を選べない。

`Rules` は正の秒単位 `TTL` と bounded `MaxPending` だけを持ち、生成後に copy・固定する。V1 は TTL を短い正の上限（package 定数で固定）とし、capacity を有限の上限（package 定数以下）に限定する。invalid rules、nil random source、nil clock は生成前に固定 `invalid` error で拒否する。manager は mutex、pending map、last trusted UTC instant、closed flag だけを保持し、challenge/assertion/verifier output を完了後に保持しない。Close は mutex 下で map を空へ交換し closed を立てる idempotent operation であり、restart 相当の新 manager は旧 map を持たないため旧 challenge を常に unknown として拒否する。

発行する challenge は `crypto/rand` から読む 32 byte を base64url raw encoding した opaque token である。request ID、digest、decision、operator、RP ID、origin その他の意味を token へ埋め込まない。random read が short read/error、token の map 衝突、closed、clock rollback、capacity 超過なら token を返さず固定 error で閉じる。衝突は同一 random value を採用せず bounded retry の後に `internal` error とする。乱数 byte、token、入力 scalar、assertion、下位 error は public error text に含めない。

`Request` はrequest ID、canonical manifest digest、`approve` 又は `deny` のdecision、operator ID、RP ID、exact HTTPS originだけを入力する。`Binding`は検証済みRequestにmanagerが決めたissued/expiry timeを加えたimmutable valueである。Issue は caller の string をそのまま信頼せず、TASK-0071 の request/digest 形式に整合する ASCII bounded scalar と `sha256:` + lowercase 64 hex digest を検査する。decision は固定 enum、operator ID は non-empty bounded scalar、RP ID は lowercase DNS hostname の安全部分集合に限定する。origin は `net/url` で単一の `https` origin として parse/re-serializeでき、userinfo/path/query/fragmentを持たず、hostnameがRP IDと完全一致し、raw spellingもcanonical serializationと完全一致するものだけを受け入れる。port、IP literal、Unicode/percent-encoding、trailing dot、subdomain/suffixの推測はV1で受理しない。この厳格なequalityはWebAuthn標準のorigin/RP validationの代替ではなく、後続verifierへ渡す安全な事前束縛だけである。

Issue は mutex 下で first trusted time を UTC 秒へ正規化し、last trusted time より前なら `clock` で拒否する。clock が同値又は前進したときは、`now >= expiresAt` の pending entry を先に purge して capacity を回収し、次に `now + TTL` の expiry を binding に固定する。expiry の境界は `now >= expiresAt` とし、capacity purge と発行を同じ critical section に置く。input validation 又は capacity error は manager state を変えない。発行済み `Issued` は fresh token と Binding の値コピーだけを返し、token string とすべての accessor は immutable value である。

Consume は opaque challenge と assertion bytes を受けるが、assertion を manager state に保存しない。mutex 下で usable/clock を先に確認し、due pending を purge してから token を lookup する。expiry は token lookup/consume に優先し、due token は削除して `expired`、unknown、purged、replayed、予約済み token は同じ `not_found` class で返す。live token は verifier 起動前に map から削除して reservation を線形化する。従って並行 Consume の最初の一つだけが callback へ進み、success、verifier failure、panic、Close 競合のいずれでも token は復活しない。Close は予約済み token を再挿入せず、新規 Consume/Issue を closed で拒否する。

verifier は一度だけ、fresh copy の complete `Binding`（challenge bytes/token を含む）と fresh copy の assertion を受ける。manager は callback の panic を recover し、panic value/stack を出さない `verification` class へ正規化する。通常の callback error も同じ class とし、assertion、signature、credential public key、WebAuthn 誤り、下位 error を公開しない。callback が成功した場合だけ、返した raw credential identifier を bounded non-empty bytes として一時検査し、domain-separated SHA-256 stable ID を生成する。raw credential ID は直ちに捨て、`Verified` には request ID/digest/decision/operator ID、stable credential ID、verified UTC time のコピーだけを返す。manager は callback 後に mutex を再取得し、closed 又は rollback なら成功 result を返さず fixed error にする。この二度目の gate により Close と正常 callback の競合から Close 後の verified result を作らない。verified result は approvalstate mutation、grant、push authorization を行わず、上位が検証結果をどう利用するかも決定しない。

## AC対応

TASK の条件本文を再掲せず、`planning input packet` の AC-ID に設計を対応させる。

| AC-ID | 設計判断 | 変更パス | 実施順序 | 失敗時の扱い |
|---|---|---|---|---|
| AC-1 | production-only crypto random/system clock、32-byte raw-base64url token、private test seams、strict binding/rules/origin/RP validationを使う。 | `challenge.go`、`challenge_test.go` | 1, 2 | invalid/input/rand/clock/capacity/closed は token/state を渡さない固定 error。 |
| AC-2 | binding と assertion を callback へ独立 copy で一回渡し、raw credential ID は一時 hash の後に破棄して immutable verified view だけを返す。 | `challenge.go`、`challenge_test.go` | 2, 3 | caller/callback が slice を mutation しても pending/result は変わらず、callback error/panic は generic verification failure。 |
| AC-3 | locked lookup→delete reservation を verifier 前に行い、callback は lock 外、post-callback usable gate を lock 内に置く。 | `challenge.go`、`challenge_test.go` | 3, 4 | failure/panic/replay/race/Close にかかわらず再挿入・retry・approvalstate mutation をしない。 |
| AC-4 | due-first purge、`now >= expiry`、monotonic last-time guard、bounded capacity recovery、idempotent Close/new-manager restart rejectionを同一 state machineに置く。 | `challenge.go`、`challenge_test.go` | 1, 3, 4 | expiry/rollback/closed/unknown を fail closed とし、旧challengeを復元又は推測しない。 |
| AC-5 | README は in-memory once-only lifecycle、trusted verifier seam、restart/failure 時の再発行、未実装の暗号 verification/Tailscale/state mutation/grantを明記する。 | `README.md` | 5 | 既存の approved/grant/push 意味を強めず、実環境の本人確認を hermetic PASS と混同しない。 |
| AC-6 | 3 path、stdlib-only、約700〜1,100 additions を candidate 前に確認し、focused race と harness/dist/root/doc/scope checks を同一 candidate evidenceに結ぶ。 | 許可済み3パス | 1--5 | scope/budget/dependency/check failure は candidate を進めず Main へ再計画を戻す。 |

## 状態機械と同時実行の不変条件

```text
new --Issue(valid, capacity)--> pending(token, binding, expiry)
pending --now >= expiry / purge--> removed(expired)
pending --first Consume, now < expiry--> reserved (mapから削除)
reserved --verify success, manager usable--> verified result (保持しない)
reserved --verify error/panic/Close/rollback--> removed (再発行せず)
pending --Close--> removed (全件破棄)
new manager --Consume(old token)--> not_found
```

`reserved` は外部に可視化されない一時状態で、token と assertion を manager の map に残さない。発行、purge、reservation、Close の linearization point は全て同じ mutex 内である。callback は lock を取ったまま呼ばないため deadlock を作らず、callback 自身が Issue/Consume/Close を呼んでも manager の once-only 性を変えない。callback 中の別 Consume は既に deleted token を `not_found` とする。Close は callback を cancel/interrupt しないが、post-callback gate が result を抑止し、予約を再利用可能にしない。manager は goroutine を作らず、timer/background purge、待機 queue、durable journal を持たない。

## 変更予定・実装順

| パス | 変更内容 |
|---|---|
| `tools/dev-agent-harness/internal/approvalchallenge/challenge.go` | fixed public error/types、strict rules/binding parser、production constructor/private dependencies、opaque issuance、locked due-first purge/reservation、panic-normalized verifier call、credential stable-ID derivation、copy ownership と idempotent Close を追加する。 |
| `tools/dev-agent-harness/internal/approvalchallenge/challenge_test.go` | deterministic clock/random/verifier fixtures と semantic、negative、ownership、panic、expiry/rollback/capacity/restart、focused race/Close test matrixを追加する。 |
| `tools/dev-agent-harness/README.md` | Passkey challenge の信頼境界、一回限り reservation、failure/restart の再発行、未実装の real WebAuthn/Tailscale/approval transition/grant を追記する。 |

1. package constants、fixed errors、immutable public values、`Rules`/Binding grammar と private fake clock/random seam を定義する。
2. Issue と due-first capacity purge を一つの locked path として実装し、token size/encoding、binding/origin/RP、expiry boundary、rollback を test で固定する。
3. reservation-before-callback Consume、copied callback input、credential stable-ID result、fixed verification error と panic recovery を実装する。
4. callback reentrancy、parallel consume、Issue/Consume/Close interleaving、expiry/purge/capacity/rollback/new manager、mutation/non-leak の deterministic/race test を追加する。
5. README と path/line/dependency scope を確認し、candidate evidence のコマンドを実行する。

## テスト設計・QAへ渡す証拠

`challenge_test.go` は network、HTTP、real authenticator、Tailscale、filesystem、approvalstate、Git、external DB を使わない。test-only dependencies は deterministic UTC clock、有限の random reader、barrier/channel verifier、panic verifier である。production constructor からこれらを選べないことも compilation/API shape で確認する。`go test -race` は shared fake clock/random を race-free に扱い、timeout を短く bounded にして flaky polling を使わない。

| ケース群 | 検査する観測可能な性質 | 弱体化を検出する負例 |
|---|---|---|
| issue/admission | 32-byte base64url token、distinct random outputs、fixed expiry、request/digest/decision/operator/RP/origin/rules の受理境界 | short/random failure、invalid enum/scalar/digest/RP/origin、http/path/query/port/subdomain、zero TTL/capacity、full capacity が token/state を渡す実装を落とす。 |
| binding/copy/result | callback が exact binding と assertion の独立 copy を一回だけ受け、result は copied scalar/time/stable hash だけを返す | caller assertion と callback binding/assertion/raw credential bytes を callback 後に mutation し、internal/result が追従する実装を落とす。raw challenge/assertion/public key/signature/error text の保持・露出も probeする。 |
| once-only/failure | first Consume だけ callback を実行し、success/error/panic の後に replay 不可 | verifier error/panic を token 再挿入又は detailed error にする mutation、unknown/replay を callback へ通す mutation を落とす。 |
| expiry/capacity/clock | exact expiry が consume より先、purge が capacity を安全に回収、rollback が Issue/Consume を拒否 | `expiry-1ns`/`expiry`/`expiry+1ns`、last-now より前の clock、due token を callback へ渡す、rollbackで purge/issueする実装を落とす。 |
| close/restart/race | Close は pending discard/new issuance reject、new manager は old token reject、並行 consume は callback count=1 | channel barrierで Consume を reservation後に停止し、second Consume、Close、callback release を競合させる。result resurrect、deadlock、callback count>1、data race を検出する。 |
| documentation/scope | README が lifecycle と未実装境界を正しく記載し、candidate は3 path/stdlib-only/budget内 | docs lint、`git diff --check`、name-status と dependency diff を candidate に対して確認する。 |

DEV はcandidate固定前に次を実行し、HANDOVERへコマンドと結果を簡潔に記録する。

```sh
cd tools/dev-agent-harness && GOCACHE=$PWD/.build/go-cache go test -race ./internal/approvalchallenge
cd tools/dev-agent-harness && make check
cd tools/dev-agent-harness && make distcheck
make task-check TASK=TASK-0072
make check
git diff --check
```

QA_PLAN は AC-1〜AC-4 の random/once-only/panic/expiry/rollback/Close/race case を `focused-rerun` に割り当てる。これらは高リスクなので、candidate-bound evidence だけ、又は読み取りだけの `evidence-review` PASS は許容しない。AC-5/AC-6 は README/scope/command evidence の独立 `evidence-review` を使える。actual device authenticator、browser origin behavior、HTTPS/Tailscale identity、approvalstate transaction、grant/push、process restart deployment はこの candidate に live-e2e ケースを作らず、後続 Task で環境と安全な cleanup が承認されるまで未確認として残す。

## 代替案・復旧・停止条件

不採用案は、(1) challenge に binding 情報を encode する方式（opaque random でなく改変・漏洩面を増やす）、(2) callback 成功時まで map entry を残す方式（同時 consume/reentrant callback で二重使用を生む）、(3) verifier failure/panic で token を戻す方式（retry が capability replay になる）、(4) timer goroutine 又は durable store（Close/restart/cleanup と範囲を広げる）、(5) manager が WebAuthn cryptographic semantics を判定する方式、(6) Consume が approvalstate を mutate 又は grant を発行する方式である。

復旧は candidate の3許可パスだけを戻し、新 package を除去して既存 approval request の振舞いを保つ。in-memory package は migration、state cleanup、grant revoke、network rollback を持たない。失敗/expiry/panic/restart 時の運用上の回復は、まだ pending であることを別境界が確認した後に新 challenge を発行することであり、同じ token の再利用ではない。復旧後は focused race suite、harness check、distcheck、task-check、root make check、docs/scope diff check を再実行する。

次が起きたら DEV は candidate を広げず停止して Main へ PLAN 改訂を求める: approvalstate read/mutation、manifest parse/HTTP/Tailscale/WebAuthn library、external dependency、disk/log/config/generated file、multi-process coordination、actual browser/device test、4本目の製品 path、1,100 additions 超過、又は RP/origin compatibility に public-suffix/port/IDNA policy が必要になる場合。受付条件を緩める、errorへ入力を出す、token を再投入する、race test を削ることで収めない。

## 未解決事項

- なし。V1 の scalar 上限、TTL/capacity 上限、origin serialization、fixed error enum、credential stable-ID domain prefix は package 内で一度だけ固定する。実 WebAuthn verifier の選択と暗号検証、Tailscale identity との AND 条件、verified result の approvalstate 適用、grant/push への接続は別 Task の明示的な契約と QA に委ねる。

## main Agentレビュー

- [x] TASK の全 AC-ID を、条件本文を複製せず、設計判断・パス・順序・fail-closed 処理へ対応させた。
- [x] 3許可パス、約700〜1,100 additions、stdlib-only、単一 candidate、`dev-sol`/high と独立 REVIEW/QA を明記した。
- [x] random、binding、reservation、expiry、rollback、Close、panic、race、copy/non-leak と focused QA evidence を具体化した。
- [x] WebAuthn 暗号検証、HTTP/Tailscale、approvalstate mutation、push grant へ scope を広げないことを固定した。
- [x] QA_PLAN が TASK-first で独立作成されている。
- [x] DEV開始を承認した。
