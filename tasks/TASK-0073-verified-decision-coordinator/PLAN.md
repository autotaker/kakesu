---
task_id: "TASK-0073"
change_class: "product"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "sol-high"
approved_dev_profile_reason: "durable approval state を変える直前に one-shot verification を接続し、consume、永続化失敗、競合、応答喪失の全てで fail-closed にする高リスク境界である。"
approved_dev_profile_risk_signals:
  - "verified result を durable decision 又は authorization と誤認すると、検証前 mutation 又は成功推測を生む"
  - "challenge consume と store transition の間の failure/poison/race は再利用又は相反する判断を誘発し得る"
  - "constructor 又は public API の seam が digest、時刻、verifier を caller に選ばせると信頼境界が崩れる"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-02T08:33:50Z"
classification_approved_by: ""
classification_approved_at: ""
classification_approval_reason: ""
# safety_contract_version: 2
# safety_contract_planned_paths: []
# safety_contract_generated_paths: []
---

# PLAN: Verified passkey decision coordinator

## 根拠・分類・実行境界

唯一の要求根拠は `TASK.md` の `Planning input packet` と、その固定参照 REF-1〜REF-3 である。既存の durable store と one-shot challenge manager を合成する新 Go package、test、README を追加するため、`change_class` は `product` とする。既存の `approvalstate` format/state machine と `approvalchallenge` lifecycle は変更しない。`approved`/`denied` は durable record の状態であって、grant、audit、push authorization 又は実 push 成功を意味しない。

DEV は `dev-sol`/high（`sol-high`）を用いる。Main が本 PLAN と TASK-first の独立 `QA_PLAN.md` を承認するまで DEV を開始しない。DEV は製品差分だけの candidate を一回固定し、REVIEW と QA は HANDOVER のその `candidate_commit` を基準に、相互の PASS を待たず独立に確認する。stage、commit、merge、push、candidate の固定、completion の no-ff 検査は Main 専有である。

candidate の製品差分は次の3許可パスだけとし、追加・削除合計を約700〜1,100行に収める。

- `tools/dev-agent-harness/internal/approvaldecision/coordinator.go`
- `tools/dev-agent-harness/internal/approvaldecision/coordinator_test.go`
- `tools/dev-agent-harness/README.md`

stdlib と既存 module/package だけを使い、新規 dependency、`go.mod`、config/build/deploy/generated artifact を変更しない。WebAuthn assertion の暗号検証、credential lifecycle、HTTP/API/UI/session、Tailscale Serve/identity、audit、grant/consume、Git remote/push、通知、disk persistence、Kakesu runtime/Schema は候補外である。実 WebAuthn、HTTPS、Tailscale、OS/process boundary、実 deployment/restart/rollback は後続 Task の `live-e2e` であり、本 Task の hermetic PASS で代替しない。

## coordinator contract・順序・失敗境界

`internal/approvaldecision` は side effect の順序だけを所有する。公開 production constructor は `New(store *approvalstate.Store, challenges *approvalchallenge.Manager, verifier approvalchallenge.Verifier)` のように concrete 3依存を non-nil で一度だけ受け取り、coordinator 内に固定する。`Complete` は verifier を引数に受けず、`Begin`/`Complete` の caller は store、manager、trusted verifier、clock、digest、state、challenge を差し替えられない。constructor と public operation は fixed coordinator error class だけを返し、request/operator/challenge/digest/assertion/credential/lower error を連結又は公開しない。

`Begin(requestID, decision, operatorID, rpID, origin)` は caller supplied digest を持たない。最初に store `Get(requestID)` を一回呼び、返った record が exact `Pending` の場合だけ、その immutable `RequestID()` と `Digest()`、caller の exact approve/deny、operator、RP ID、origin から `approvalchallenge.Request` を組み立てて `Issue` する。Get が expiry を durable にした、terminal、digest/state 不整合、Issue failure のいずれでも issued value を返さず、challenge の再発行や store mutation を試みない。これにより store-derived binding の前に token を作らず、pending の再確認は Begin のたびに同じ durable store を正本にする。

`Complete(challenge, assertion)` は固定 verifier を使う `Consume` を一回だけ先に呼ぶ。Consume failure、panic-normalized verification failure、expiry、unknown/replay は store に触れず fixed coordinator error に正規化する。verified binding の exact request ID/digest/operator/decision だけから、decision が approve なら `Approve`、deny なら `Deny` を一回だけ選ぶ。verified result は caller supplied request/digest/operator による上書き、別 decision fallback、再 Consume、再 Issue を許さない。durable transition が成功した後だけ immutable result として durable record の request ID/digest/state/actor および stable credential ID を返す。verification 単独、又は response を失った Complete を成功として返さない。

store が expired/cancelled/terminal、digest mismatch、transition conflict、persistence failure、又は poison を返した場合、challenge はすでに消費済みのまま result を空にして fixed error を返す。store の atomic pending transition が同一 request 上の複数 approve/deny challenge の唯一の arbiter なので、coordinator 独自の first-wins lock、decision rollback、成功推測を追加しない。response loss 後は caller が store `Get` で durable state を照合する。旧 challenge の replay で成功応答を再構成せず、request がなお pending であることを Begin が確認した場合だけ新 challenge を作れる。poison では `Close`→`Open` と上位 reconciliation を要求し、この package は recovery を実装しない。

production public contract を弱めず test failure を注入するため、`coordinator.go` 内だけに store operation（`Get`/`Approve`/`Deny`）と challenge operation（`Issue`/fixed-verifier `Consume`）の package-private interfaces/adapters を置く。production `New` は concrete dependencies からのみ adapters を作る。private test constructor は fakes を受けられるが export せず、fake verifier result、store record、challenge/token/time を public API 経由で偽装できる seam を作らない。public result は scalar/value copy のみとし、assertion input は保持せず、issued/result の mutable input/output を保持又は返却しない。

## AC対応

| AC-ID | 設計判断 | 変更パス | 実施順序 | 失敗時の扱い |
|---|---|---|---|---|
| AC-1 | concrete store/manager/verifier を固定する constructor と、`Get` record の pending/request/digest だけから組む Begin request を置く。 | `coordinator.go`、`coordinator_test.go` | 1, 2 | nil/invalid/closed/expired/terminal Get と Issue failure は固定 error、issued result なし、mutation/retry なし。 |
| AC-2 | fixed verifier の `Consume` を先行させ、verified binding の decision に一対一で `Approve`/`Deny` を選び、transition success 後だけ copied result を返す。 | `coordinator.go`、`coordinator_test.go` | 2, 3 | Consume/verification failure は store untouched、store failure は consumed token のまま fixed error、verified-only success は返さない。 |
| AC-3 | store atomic pending transition を first durable decision wins の正本とし、failure/poison/response loss/replay に rollback、fallback、success reconstruction を持たせない。 | `coordinator.go`、`coordinator_test.go`、`README.md` | 3, 4 | expiry/terminal/digest/transition/persistence/poison は empty result と fixed error。pending の再開始は新 Begin の Get 成功時だけ。 |
| AC-4 | production adapter を使う real approvalstate/approvalchallenge fixture と、順序・failureを観測する private fakes を分離し、race/non-leak/copy assertions を置く。 | `coordinator_test.go` | 1--4 | order/call-count/mutation/race leak の検出不能又は fake-only success は test failure とする。 |
| AC-5 | package comment/READMEで verifier seam を trusted input と限定し、WebAuthn/Tailscale/HTTP/audit/grant/push へ authority を昇格しない。環境依存確認は blocked と書く。 | `coordinator.go`、`README.md` | 4 | これらの transport/identity/persistence side effect を追加しない。live requirement は hermetic evidence で PASS に置換しない。 |
| AC-6 | 3 path、stdlib-only、single candidate、700〜1,100 additions を候補前に確認し、focused race/harness/root/docs/diff を candidate-bound command/result として残す。 | 許可済み3パス | 1--5 | scope、size、dependency、format、test、distribution の FAIL は candidate を進めず分類して Main へ戻す。 |

## 代替案・変更予定・実装順

不採用案は、(1) Begin が caller digest を受ける案（durable record との binding を失う）、(2) Complete 引数で verifier を受ける案（trusted seam を呼出しごとに差替え可能にする）、(3) Approve/Deny を Consume 前に呼ぶ案（未検証 mutation を許す）、(4) coordinator mutex に first-wins を持つ案（durable store の atomic transition と二重の正本になる）、(5) persistence failure 後に Consume/replay 又は Get 推測で result を再構成する案（one-shot と poison fail-closed を破る）、(6) fake-only exported constructor（本番信頼境界を公開する）である。

| パス | 変更内容 |
|---|---|
| `internal/approvaldecision/coordinator.go` | fixed errors、immutable result、public production constructor/Begin/Complete、private adapters/interfaces/test constructor、store-derived Begin、Consume→decision transition、safe error mapping/copy boundaryを実装する。 |
| `internal/approvaldecision/coordinator_test.go` | real store+manager integration fixture、private ordering/failure fakes、approve/deny/binding/one-shot/first-wins/expiry/terminal/digest/persistence-poison/response-loss/non-leak/race matrixを実装する。 |
| `README.md` | coordinator の順序、durable-first meaning、consume後 failure と retry、trusted verifier の限定境界、後続 live-e2e 範囲を追記する。 |

1. fixed error/value API と concrete production constructor、private adapter boundaryを定義する。
2. Begin の `Get`→pending check→store-derived Request→Issue path と、input/state injection の拒否を実装する。
3. Complete の fixed-verifier Consume→exact verified binding→Approve/Deny path、result copy、failure mapping を実装する。
4. real integration fixture と private fakesで success、negative、failure、poison、response-loss、parallel approve/deny、copy/non-leak を固定する。
5. README と focused/race/repository evidence を仕上げ、単一candidateへ渡す。

## 検証・復旧・引継ぎ

focused test は temporary owner-only store directory、canonical test manifest、real `approvalstate.Store` と real `approvalchallenge.Manager`、固定 trusted verifier を組む integration fixture を少なくとも一つ使用する。これは `Get→Issue` と `Consume→Approve/Deny`、exact record digest/decision/operator binding、durable result、同一challenge replay拒否、複数challengeの first durable decision wins、response loss後の Get reconciliation を確認する。private fakes は Get/Issue/Consume/Approve/Deny の order と count、verification panic/failure、expiry/terminal/digest mismatch、transition/persistence/poison failure を注入し、failureで後続 action、fallback、token resurrection、success result がないことを検出する。

同じ suite は Begin/Complete の並行実行、opposed approve/deny、replay、Close/failure相当を bounded goroutine/barrier で反復し、`-race` による data race を検出する。assertion slice mutation、returned record/result value の mutation、error text を検査して、challenge/assertion/raw credential/request/operator/digest/lower error を長寿命 state/result/error に漏らさないことを確認する。test は cryptographic WebAuthn、HTTP、Tailscale、audit/grant、実 deployment を stub success で主張しない。

DEV は candidate 固定前に少なくとも次を実行し、command と結果だけを HANDOVER の表に残す。

```sh
cd tools/dev-agent-harness && GOCACHE=$PWD/.build/go-cache go test -count=1 -race ./internal/approvaldecision
cd tools/dev-agent-harness && ./configure && make check && make distcheck
make lint-docs
make task-check TASK=TASK-0073
git diff --check
```

Main は同一candidateに root `make check` を一回実行する。QA は candidate から focused race suite を独立に一回再実行し、harness/root/docs/diff は candidate-bound DEV証跡を `evidence-review` する。failure は実装不具合と即断せず、candidate regression、baseline、environment、test flaw、仕様/QA不整合に分類する。実 WebAuthn/HTTPS/Tailscale/OS deployment/restart/rollback は approved environment と cleanup がないため blocked のままとし、別 mode の PASS で代替しない。

復旧は candidate の3許可パスだけを戻し、既存 state/challenge package の挙動を残す。新 coordinator は新 durable wire format、challenge/credential persistence、migration、external cleanup を作らない。poison からの復旧は store の既存 Close→Open と上位 reconciliation を用い、coordinator が state を書き換えない。復旧後は focused race、harness check/distcheck、root check、docs lint、task check、diff check を再実行する。

HANDOVER は `candidate_commit` を既存の一箇所だけで管理し、candidate SHA/tree/digest の重複証跡、carry-forward項目、追加receiptを作らない。DEV evidence は各 command と結果の lean table のみとする。

## 未解決事項

- なし。公開 API の正確な Go identifier と fixed coordinator error enum は、上記 contract を満たす最小面として実装時に package 内で一度だけ固定する。

## main Agentレビュー

- [x] TASKの全AC-IDへ設計判断、パス、順序、失敗時の扱いを対応させ、条件本文を複製していない。
- [x] production constructor固定、store-derived digest、Consume→Approve/Deny、durable first-wins、poison/response-loss、private fake seam、non-leakを具体化している。
- [x] `dev-sol`/high、3許可パス、約700〜1,100行、stdlib-only、single candidate、lean HANDOVERを記録している。
- [x] QA_PLANがTASK-firstで独立作成されている。
- [x] `dependency-ready reconciliation`と完了経路preflightが完了している。
- [x] DEV開始を承認した。

安全契約変更でv2契約を選ぶ場合は、コメントを外して`safety_contract_version: 2`と予定パス・生成パスの配列を記録し、DEV前に`make task-preflight TASK=TASK-0073`を実行する。変更しない種別は空配列とし、通常の予定パスと生成パスを重複させない。MainはPLAN/QA_PLANの意図・スコープ・受け入れ経路を確認し、分類承認をフロントマターへ記録する。分類変更時はTask、PLAN、QA_PLANを再承認し、承認者と時刻を更新する。
