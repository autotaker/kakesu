---
task_id: "TASK-0072"
title: "Passkey challengeの一回限りlifecycleを実装する"
status: draft
created_at: "2026-08-02"
---

# TASK-0072 Passkey challengeの一回限りlifecycleを実装する

## `Planning input packet`（Main Agent所有）

このsectionをPlannerとQAへ渡す唯一の`planning input packet`とし、各内容をPLAN/QA_PLANへ複製しない。

### 目的

TASK-0071のpending承認requestに対し、後続WebAuthn verifierが使うrandom challengeをrequest digest、判断、operator、RP ID、origin、期限へ一意に束縛し、同時実行・失敗・期限切れ・process restartで再利用できない一回限りのlifecycle境界を作る。実WebAuthn署名検証やHTTPをこのTaskへ混ぜず、スマートフォン承認の次の安全な接続点を用意する。

### 対象と対象外

#### 対象

- `internal/approvalchallenge`へbounded in-memory challenge managerを追加する。productionは`crypto/rand`とsystem clockだけを使い、test seamを公開APIから分離する。
- challengeをrequest ID、canonical manifest digest、`approve/deny`判断、operator ID、RP ID、exact HTTPS origin、発行/期限へ束縛し、32-byte以上のunbiased random値をbase64urlで返す。
- challenge発行前に固定enum、長さ、文字集合、origin/RP整合、正のTTL、capacityを検査し、入力値を含まない固定error classを返す。
- verifier callbackへchallenge bindingとassertion bytesのcopyを渡し、最初のconsume試行をmutex下で原子的に予約する。成功・検証失敗・panic・同時試行のいずれでも同じchallengeを再利用させない。
- expiryをconsumeより優先し、期限到達、clock rollback、unknown/replayed challenge、closed managerをfail closedにする。Closeはpending challengeを破棄し、restartで復元しない。
- verified結果はrequest ID、digest、decision、operator ID、credentialの非可逆stable ID、verified timeだけをcopy ownershipで返し、challenge/assertion/signature/credential public keyを保持・公開しない。
- READMEへlifecycle、信頼境界、restart/失敗時の再発行、実WebAuthn verifierとapproval state mutationが後続であることを記載する。
- TASK-0071で使用した`os.Root.Rename`がGo 1.25 APIである一方、moduleの`go` directiveが1.24のまま残った回帰を解消し、harnessの最小Go言語/API契約を1.25へ一致させる。dependencyは追加しない。

#### 対象外

- WebAuthn clientData/authenticatorData/signature/credential public key、counter、RP ID hash、origin、UV flagの暗号学的検証、credential登録/失効/recoveryを実装しない。
- verifier callbackの結果だけでTASK-0071 storeを`approved/denied`へ変更せず、HTTP/API/UI/session/cookie/CSRF、Tailscale Serve/Grant/identity header、通知を追加しない。
- challengeをdisk、log、環境変数、外部DBへ保存せず、複数process/host共有、backup/recoveryを追加しない。
- push grant、`consuming/consumed/indeterminate`、Git wire解析、remote old SHA照合、credential、実push、auditを追加しない。
- `go.mod`の`go` directive以外のdependency、config/build/deploy/generated artifact、Kakesu本体runtime/Schemaを変更しない。
<!-- safety_contractの場合: 製品コード、test、runtime/build設定、Schema、製品依存、生成製品入力/成果物、外部観測可能な挙動を変更しない。 -->

### 受け入れ条件

<!-- AC-IDはTask内で一意かつ安定させ、観測可能な結果をここに一度だけ記載する。 -->

- [ ] AC-1: production managerは安全なrandom challengeを生成し、固定されたrequest/digest/decision/operator/RP ID/origin/TTL/capacity以外を受理せず、callerがchallenge又はclockを注入できない。
- [ ] AC-2: verifierはexact bindingとassertion copyだけを受け取り、成功時のverified resultは同じrequest/digest/decision/operatorと非secret credential IDへ束縛され、入力/出力sliceのmutationで内部状態が変わらない。
- [ ] AC-3: 最初のconsume試行だけが予約され、成功、verification failure、panic、同時試行、replayの後に同じchallengeを再使用できない。verifier failure/panicを内部情報なしの固定errorへ正規化する。
- [ ] AC-4: expiryはconsumeより優先され、期限ちょうど、purge、capacity回収、clock rollback、Close競合、restart相当の新managerで旧challenge拒否をbounded race testが検出する。
- [ ] AC-5: packageとREADMEはchallenge lifecycleをWebAuthn verification、Tailscale identity、verified decision API、approval state mutation、push authorizationへ昇格させず、実環境依存条件を未確認のまま明示する。
- [ ] AC-6: 変更は許可4パス、約700〜1,100 additionsと`go 1.25`への最小toolchain契約訂正、新規dependencyなしに収まり、focused `go test -race`、harness check/distcheck、root `make check`、docs lint、diff checkがPASSする。

### 安定した参照

| 参照ID | 対象 | 固定改訂/ダイジェスト | 用途 |
|---|---|---|---|
| REF-1 | `tasks/TASK-0071-approval-request-store/HANDOVER.md` | main merge `84ce39263edfb1c642e4c02a2f464d7c2a44e8b7` | digest/decision/expiryを永続requestへ束縛する既存境界 |
| REF-2 | `docs/development/development-agent-harness.md` §11.2 | main `a4bde718a164302bca26b770766d4e998d591627` | Tailscale identityとPasskey UVのAND条件、challengeの一回性 |
| REF-3 | `wiki/semantic/schemas/development-agent-harness-approval-request-store.md` | main `a4bde718a164302bca26b770766d4e998d591627` | approvedとauthorizationを分離する再利用知識 |

### 依存状態

| 依存 | 状態 (`ready` / `pending`) | planning参照 | `ready`後に固定する値 |
|---|---|---|---|
| TASK-0071 | `ready` | `84ce39263edfb1c642e4c02a2f464d7c2a44e8b7` | N/A |

### 許可パス

- `tools/dev-agent-harness/internal/approvalchallenge/challenge.go`
- `tools/dev-agent-harness/internal/approvalchallenge/challenge_test.go`
- `tools/dev-agent-harness/README.md`
- `tools/dev-agent-harness/go.mod`

### 完了経路preflight

| 確認対象 | 結果 | コマンドまたは根拠 |
|---|---|---|
| 完了checker | `ready` | product 3トランザクション、同一candidate独立REVIEW/QA、no-ff completionを使用する |
| 権限 | `ready` | challenge/verification lifecycleは認可前段の高リスク境界なので`dev-sol`/highを使う。Mainだけがstage/commit/merge/pushする |
| 依存状態と参照 | `ready` | TASK-0071とWikiがmainへ反映済み。新規外部dependencyなし |
| 生成物の有無と更新方法 | `ready` | Go source/test、READMEのみ。配布生成物は変更せず`make distcheck`で再生成可能性を確認する |
| 割当ワークツリー | `ready` | `worktrees/TASK-0072-passkey-challenge-lifecycle` |
| Lapログの書込・Schema・`repository annotation` | `not-applicable` | 新規log/Schema/annotationなし |

### 未決事項

- なし

### `Dependency-ready reconciliation`

<!-- 依存ready時にMainがready参照、planning参照との差分、AC/設計/スコープ/QAへの影響、再承認結果を追記する。依存なし又は未readyならN/Aとする。 -->

- N/A

## 背景

TASK-0071で承認requestを安全に永続化できたが、現在はスマートフォンで行う本人操作をrequest digestと判断へ一回限りに束縛する境界がない。先にHTTPやWebAuthn libraryを接続すると、challenge再使用、別request/判断への差し替え、同時検証、期限切れ、restart後の復元推測をservice各所で個別実装する危険がある。このTaskで暗号検証器の前後を明確にし、実WebAuthn verifierとTailscale/HTTP接続を後続Taskとして独立QAできるようにする。

DEV focused race後のharness checkで、TASK-0071のdescriptor-relative atomic renameがGo 1.25で追加された`os.Root.Rename`を使う一方、moduleの`go` directiveが1.24のままのため`go vet`が停止する回帰を検出した。安全なroot束縛を弱める置換ではなく、実際に必要な最小Go API契約へdirectiveを合わせる。

## 検討すべき設計観点

- challenge bytesはopaqueであり、request情報を埋め込まずrandomnessとmanager内bindingで対応付ける。
- verifier callbackは後続の暗号検証実装を差し込むseamであり、callback自体を本人確認済みと偽装しない。実serviceは信頼されたverifierだけを構成する。
- consume開始時にchallengeを予約し、検証失敗でも再利用させない。再試行はactive requestに対する新challenge発行とする。
- verification中にrequestが失効しても、後続store mutationが期限を優先して拒否する。verified result単独ではapprovedへ遷移しない。
- in-memoryであることはrestart時に全challengeを失効させるfail-safe設計であり、durable recoveryを推測しない。
- callback panic、raw assertion、challenge、credential public key、入力値をerror又はmanager stateへ残さない。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] 選択した`change_class`の完了経路と`make check`を満たしている。
- [ ] 製品変更の場合: 実装、テスト、文書、同一案の独立REVIEW/QA、完了後の環境依存ケース確認が完了している。
- [ ] 安全契約変更の場合: Mainの意図・スコープ・受け入れ経路確認、契約検査、許可された統制文書差分の確認が完了している。

## 関連コンテキスト

### 意味 Wiki

- `wiki/semantic/schemas/development-agent-harness-approval-request-store.md`
- `wiki/semantic/schemas/development-agent-harness-push-approval-manifest.md`

### 判断

- challenge lifecycleはWebAuthn暗号検証と分離し、実依存選定をこのTaskで先取りしない。
- 検証失敗したchallengeは再試行せず、新challengeを発行する。

### 適用しなかった重要な判断

- なし
