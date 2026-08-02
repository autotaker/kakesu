---
task_id: "TASK-0071"
change_class: "product"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "sol-high"
approved_dev_profile_reason: "owner-only durable state、OS lock、crash recovery、clock safety、並行 mutation を一候補で整合させる高リスク境界である。"
approved_dev_profile_risk_signals:
  - "approval state の永続化失敗又は複数 writer が後続の認可判断を取り違え得る"
  - "rename/fsync の確定境界、restart、clock rollback、poisoned fail-closed を同時に扱う"
  - "canonical manifest/digest と actor を非漏えいの immutable record へ束縛する"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-02T06:51:59Z"
classification_approved_by: "main-agent-sol-high"
classification_approved_at: "2026-08-02T06:51:59Z"
classification_approval_reason: "approval request の durable Go package とテスト、README contract を追加する製品変更。"
# safety_contract_version: 2
# safety_contract_planned_paths: []
# safety_contract_generated_paths: []
---

# PLAN: Approval request durable state store

## 根拠・分類・実行境界

唯一の要求根拠は `TASK.md` の `Planning input packet` と固定参照 REF-1/REF-2 である。正規 manifest を永続的な request state へ変換する公開 Go contract を追加するため `change_class` は `product` とする。REF-1 の pending/approved/stale/expired 境界を採用するが、Tailscale、Passkey、grant、実 push は接続しない。`approved` は後続層が検証済みとして渡す actor を記録した状態にすぎず、認可・消費・再実行を意味しない。

DEV は `dev-sol`/high（`sol-high`）とする。Main は DEV 前に本 PLAN と TASK-first の独立 `QA_PLAN.md` を確認・承認する。独立 PLAN review は置かない。DEV は一回だけ製品差分の candidate を固定し、REVIEW と QA は同じ candidate から独立に実施する。stage/commit/merge/push、candidate identifier、completion の no-ff check は Main 専有である。

candidate は次の 5 許可パスだけを変更し、追加・削除合計を約900〜1,200行に収める。

- `tools/dev-agent-harness/internal/approvalstate/store.go`
- `tools/dev-agent-harness/internal/approvalstate/store_test.go`
- `tools/dev-agent-harness/internal/approvalstate/lock_unix.go`
- `tools/dev-agent-harness/internal/approvalstate/lock_unsupported.go`
- `tools/dev-agent-harness/README.md`

stdlib と既存 `approvalmanifest` 以外は使わない。HTTP/API/UI、Tailscale/Grant、Passkey、通知、credential、Git、external DB、監査 log、config/deploy/generated/live state、Kakesu runtime/Schema、既存 launcher/broker/proxy は候補外である。state directory の作成/chown、実 `/var/lib`、multi-host/network filesystem、backup、鍵暗号化、実 UID/permission/restart/rollback も実装又は hermetic PASS の対象にしない。

## package contract・状態・永続化設計

`internal/approvalstate` は broker-owner が既に用意した一 directory を一 process だけで開く store owner とする。公開面を `Rules`、`Open(root, rules)`、`Store`、immutable `Record`/`Snapshot`、`Create(canonicalManifest []byte)`、request ID と manifest digest を受ける verified approval/denial、cancel、stale、`Get`、`ExpireDue`、`Close` に限定する。decision/cancel/stale の API は caller に state・時刻・path を選ばせず、approval/denial だけは上位で検証済みの非空 actor ID を scalar として保存する。公開 error は固定 class のみで、root、ID、actor、manifest/snapshot bytes、digest、repository/ref/SHA、下位 OS error を包まない。

`Rules` は package 内で一回だけ検査・copy し、policy version、revocation epoch、正の最大 TTL、最大 record 数を固定する。production `Open` は `time.Now` と実 file operations を使い、clock と phase-aware persistence operations は非公開 `newWith…` にだけ注入して deterministic failure/restart test を可能にする。rules、clock、persistence seam は public API や production caller から選べない。

Create は `approvalmanifest.Parse` が返す canonical immutable manifest 以外を受理しない。store は Parse の `Encoding()` を再取得して入力との byte equality、policy/epoch equality、trusted UTC now に対する `created_at <= now < expires_at`、positive TTL 上限、未使用 request ID、record 上限を確認する。同一 ID は bytes/digest が同じ場合も conflict とする。manifest 自身の request ID/digest/timestamps を record へ copy するため、caller bytes や getter の mutation は内部 state を変えない。

状態は `pending`、`approved`、`denied`、`cancelled`、`expired`、`stale` の固定 enum とし、terminal state からの遷移を作らない。exact request ID と `crypto/subtle.ConstantTimeCompare` による digest 一致が全 mutation の前提である。`pending` から approved/denied/cancelled/expired のみ、`approved` から stale/expired のみを許可する。各 action の前に期限を判定し、到達済みなら先に `expired` を永続化して approval/denial 等を拒否する。current rules との policy/epoch 不一致、rollback clock、digest 不一致、approved 以外の stale は拒否する。`Get` と `ExpireDue` は mutex 内で due active records を同じ snapshot mutation として expire するので、期限切れを approval 可能な record として返さない。

snapshot は V1 format version、generation、最後に観測した UTC whole-second trusted time、request-ID lexical order の record array を持つ compact canonical JSON 一文書とする。record は request ID、copied manifest encoding、derived digest、state、manifest created/expires、必要な decision time/actor を持つ。固定 wire struct と strict scan/decode/re-encode byte equality で、unknown/duplicate/missing/trailing/noncanonical/oversize を拒否する。Open は全 record で `approvalmanifest.Parse`、raw byte equality、request/digest/timestamp/policy/epoch、state/actor transition shape、strict sort/unique、bound/record count、snapshot generation/time の整合を再検証する。空 snapshot は Open 時だけ初期化・永続化せず memory の generation 0 とし、state file 又は残存 temp が存在する場合は既存正常状態を推測せず拒否する。

Open は absolute clean root のみを受理し、root と固定 `state.json`/`lock`/temp names を basename で扱う。root は current EUID 所有の既存 `0700` directory、state/lock/temp は symlink でない regular node として descriptor ベースに検査する。root を作らず、wrong mode/type/owner、relative/traversal、state symlink、残った temp、unsupported OS は固定 failure で閉じる。`lock_unix.go` は Linux/Darwin の non-blocking advisory exclusive lock を一度取得し、`lock_unsupported.go` は常に fail closed とする。Store の mutex は process 内 Create/Get/decision/expiry/Close を直列化し、Close は idempotent に一度だけ descriptor/lock を解放する。

mutation は current immutable map を clone して期限処理と一遷移を適用し、sorted canonical candidate snapshot を作る。固定 temp basename を `0600` の新規 regular fileとして exclusive create し、全 write、file fsync、close、atomic same-directory rename、directory fsync の順に実施する。directory fsync 成功後だけ generation と memory snapshot を交換する。write/file-sync/close の rename 前 failure は temp を best-effort で除去し、old memory と canonical state file を維持する。rename が失敗した場合、又は rename 成功後の directory fsync が失敗した場合は namespace 結果を断定せず immediate poison とする。poisoned store は Close 以外を固定 error で拒否し、利用者は Close→Open と上位 reconciliation を明示的に行う。rename 後に成功を推測、再試行、partial record の memory commit はしない。

## AC対応

| AC-ID | 設計判断 | 変更パス | 実施順序 | 失敗時の扱い |
|---|---|---|---|---|
| AC-1 | absolute existing `0700` owner directory と fixed names を descriptor/lstat で検査し、Unix non-blocking exclusive lock と idempotent Close を持つ。 | `store.go`、`lock_unix.go`、`lock_unsupported.go`、`store_test.go` | 1, 2 | path/node/mode/owner/OS/lock conflict を固定 error で拒否し、lock を取れなければ state を読書きしない。 |
| AC-2 | Parse/re-encoding equality、rules/clock/TTL/unique/capacity 検査後だけ pending clone を persist する。 | `store.go`、`store_test.go` | 2, 3 | invalid/conflict/expired/capacity は write 前に拒否し、bytes/state/generation を変えない。 |
| AC-3 | ID+constant-time digest guard、due-first expiry、transition table、persisted last-now rollback guard を一つの locked mutation path に置く。 | `store.go`、`store_test.go` | 3, 4 | mismatch/clock rollback/terminal/invalid stale は no-op failure。期限優先の expiry persist が失敗すれば decision も適用しない。 |
| AC-4 | strict bounded V1 snapshot、sorted records、Open full validation、temp→file fsync→rename→dir fsync、phase-aware poison を使う。 | `store.go`、`store_test.go`、`lock_unix.go` | 1--4 | pre-rename は old state 維持、rename uncertainty は poison、corrupt/noncanonical/temp は Open failure。 |
| AC-5 | private mutable storage、copy record/snapshot/list/encoding getters、safe fixed errors、mutex と Close gate を用いる。 | `store.go`、`store_test.go` | 2--4 | nil/closed/poisoned/concurrent callers は panic/race/deadlock なしに fail closed、失敗 mutation は partial update なし。 |
| AC-6 | 5 paths/stdlib-only を維持し、focused race/restart/failure fixture と harness/dist/root/scope check を同一 candidate evidence に結ぶ。 | 許可済み5パス | 1--5 | scope/line budget/test/distribution failure は candidate を進めず Main へ戻す。 |

## 代替案・変更予定・実装順

不採用案は、(1) in-memory mutex のみ（restart/二 process writerを満たさない）、(2) rename を durable success と扱う（directory metadata の crash window を残す）、(3) failure 後に state file を読み直して成功を推測する（rename ambiguity と rollback を安全に解消できない）、(4) store が Passkey/remote old SHA を検証する（信頼境界と後続 task を混同する）、(5) sqlite/external dependency（許可範囲と stdlib-only 条件外）である。

| パス | 変更内容 |
|---|---|
| `internal/approvalstate/store.go` | public value/error/rules/store API、strict snapshot codec/validator、manifest binding、state machine、clock guard、copy ownership、Open/Close、atomic persistence と private test seamsを実装する。 |
| `internal/approvalstate/store_test.go` | deterministic temp-dir/clock/persistence fixtures、semantic/transition/restart/corruption/ownership/non-leak/failure/race matrixを実装する。 |
| `internal/approvalstate/lock_unix.go` | Linux/Darwin build tag の safe descriptor lock/unlock を実装する。 |
| `internal/approvalstate/lock_unsupported.go` | complementary build tag の fail-closed lock implementation を置く。 |
| `README.md` | state、single-writer/durability/poison/reopen、verified actor の境界、approval と grant/push の非同義、live configuration/restart が後続であることを追記する。 |

1. constants、safe errors、immutable types/rules、platform lock と root/node validationを作る。
2. canonical snapshot codec と Open/restart validation、private clock/filesystem seam を実装する。
3. Create と common copy-on-write persistence transactionを実装する。
4. due-first decision/cancel/stale/expiry、rollback/poison/Close concurrency を共通 locked pathへ追加する。
5. hostile-input、failure-injection、race/restart、READMEを仕上げる。

## 検証・復旧・引継ぎ

focused suite は temporary directory、fixed clock、in-process lock と injected persistence seam だけを使い、network、Passkey、Tailscale、Git、external DB、実 deployment を使わない。少なくとも root/lock/mode/type/symlink/temp/unsupported platform、canonical manifest mutation/policy/epoch/TTL/capacity/duplicate、全許可・拒否遷移、expiry precedence、rollback、restart generation/sort/strict snapshot mutations、pre-rename vs rename/dir-sync uncertainty、getter/input mutation/non-leak、parallel Create/Get/decision/ExpireDue/Close を table/race tests で覆う。failure fixture は each phase の generation、disk bytes、poison status を観測し、post-rename failure を normal failure と誤判定しない。

DEV は candidate 固定前に少なくとも次を実行する。

```sh
cd tools/dev-agent-harness && GOCACHE=$PWD/.build/go-cache go test -race ./internal/approvalstate
cd tools/dev-agent-harness && make check
cd tools/dev-agent-harness && make distcheck
make task-check TASK=TASK-0071
git diff --check
```

Main は同一 candidate に root `make check` を一回実行する。QA_PLAN は AC ごとに `focused-rerun` 又は `evidence-review` を理由付きで割り当てる。atomic persistence、lock、restart、clock rollback、poison、race は高リスクのため negative/failure/race evidence のない evidence-review 単独 PASS を禁止する。実 UID/permission、filesystem durability across actual power loss、systemd deployment/restart/rollback、Tailscale/Passkey/grant/remote は live-e2e の後続範囲であり、本 task の hermetic PASS で置き換えない。

復旧は candidate の5許可パスだけを戻し、新 package を除去して既存 read-only/push拒否挙動を保つ。candidate は live state を作成しないので migration、external cleanup、grant revoke はない。実装中の poison は state を書換えず Close→Open/reconciliation を要求する。復旧後は focused race suite、harness/root check、distcheck、task-check、diff check を再実行する。

## 未解決事項

- なし。V1 の state/lock/temp basename、record actor grammar/length、snapshot byte/record bounds、fixed error enum、last-observed-time wire fieldはこの PLAN に従い package 内で一度だけ固定する。複数 writer/host、監査、passkey verification、grant consuming/indeterminate、実配置の durability 確認は別 Task で明示追加する。

## main Agentレビュー

- [x] TASKの全AC-IDへ設計判断、パス、順序、失敗時の扱いを対応させ、条件本文を複製していない。
- [x] owner-only durable store、canonical manifest再検証、状態遷移、strict snapshot、atomic/poison semantics、copy/non-leak を具体化している。
- [x] `dev-sol`/high、5許可パス、約900〜1,200行、単一candidate、独立PLAN reviewなしを記録している。
- [x] QA_PLANがTASK-firstで独立作成されている。
- [x] `dependency-ready reconciliation`と完了経路preflightが完了している。
- [x] DEV開始を承認した。
