---
task_id: "TASK-0073"
title: "検証済みPasskey判断coordinatorを実装する"
status: draft
created_at: "2026-08-02"
---

# TASK-0073 検証済みPasskey判断coordinatorを実装する

## `Planning input packet`（Main Agent所有）

このsectionをPlannerとQAへ渡す唯一の`planning input packet`とし、各内容をPLAN/QA_PLANへ複製しない。

### 目的

TASK-0071のdurable request storeとTASK-0072のone-shot challenge managerを、信頼されたPasskey verifier seamの前後で安全に接続する。challenge発行時はstoreを正本としてdigest/stateを束縛し、検証成功後は同じrequest/digest/decision/operatorだけを`Approve/Deny`へ適用する。複数challenge、期限切れ、競合、永続化失敗でもverified結果から成功を推測せず、実WebAuthn/HTTP/Tailscaleを後続Taskとして独立に接続できるcoordinatorを作る。

### 対象と対象外

#### 対象

- `internal/approvaldecision`へside-effect順序を所有するcoordinatorを追加する。production constructorは`approvalstate.Store`、`approvalchallenge.Manager`、一つのtrusted verifierを固定し、Begin/Complete callerがverifierを差し替えられない。
- `Begin`はrequest ID、`approve/deny`、operator ID、RP ID、originを受け、store `Get`のpending recordとderived digestだけからchallenge requestを構成する。callerにdigest/state/time/challengeを選ばせない。
- `Complete`はchallenge managerで最初の試行をconsumeし、verified bindingのdecisionに応じてexact request ID/digest/operatorをstore `Approve/Deny`へ一度だけ渡す。成功結果はdurable recordのstate、request/digest/actorとcredential stable IDだけを返す。
- challenge verification成功後にrequestがexpired/cancelled/terminal、digest/policyが不一致、又はstore persistenceが失敗した場合は結果を返さず固定errorにし、消費済みchallengeを復活・再試行しない。
- 同一requestに複数のapprove/deny challengeが存在しても、storeの最初のpending transitionだけを成功させる。並行Begin/Complete、verifier panic/failure、state mutation failure、response再試行をbounded race/negative testで検出する。
- productionの実packageを使うintegration fixtureと、failure/orderingを注入するpackage-private interfaces/fakesを分け、公開APIからstore/challenge/verifier結果を偽装するseamを追加しない。
- errorは固定classだけを返し、request/operator/challenge/digest/assertion/credential/lower errorを含めない。入力/結果sliceの保持をしない。
- READMEへ順序、不確実性、再試行、信頼境界、後続WebAuthn/Tailscale/HTTP/audit/grantを明記する。

#### 対象外

- WebAuthn assertionの暗号学的検証、credential登録/lookup/失効/counter、実Passkey、実スマートフォンを実装しない。
- HTTP/API/UI/session/cookie/CSRF、Tailscale Serve/Grant/identity header、通知、process/OS到達制御を追加しない。
- approvalstate/approvalchallenge/approvalmanifestの既存format/state machineを変更せず、challenge又はcredential materialを永続化しない。
- audit log、credential stable IDの永続記録、push grant、`consuming/consumed/indeterminate`、Git wire/remote照合、実pushを追加しない。
- 新規dependency、go.mod、config/build/deploy/generated artifact、Kakesu本体runtime/Schemaを変更しない。
<!-- safety_contractの場合: 製品コード、test、runtime/build設定、Schema、製品依存、生成製品入力/成果物、外部観測可能な挙動を変更しない。 -->

### 受け入れ条件

<!-- AC-IDはTask内で一意かつ安定させ、観測可能な結果をここに一度だけ記載する。 -->

- [ ] AC-1: production constructorはnon-nil store/challenge manager/trusted verifierを一度だけ固定し、Beginはpending store recordからrequest IDとdigestを導出してexact decision/operator/RP ID/originへchallengeを束縛する。callerはdigest/verifier/timeを注入できない。
- [ ] AC-2: Completeはchallengeをverifier前にone-shot consumeし、そのverified bindingだけからstore `Approve/Deny`を選ぶ。durable transition成功後だけrecordとcredential stable IDを返し、verified result単独をapproved扱いしない。
- [ ] AC-3: expiry、terminal/digest mismatch、concurrent approve/deny、verifier failure/panic、store persistence/poison failureでは固定errorを返し、challenge再利用、自動再発行、別decision fallback、成功推測を行わない。
- [ ] AC-4: real approvalstate/approvalchallenge integrationとpackage-private failure fakesにより、順序（Get→Issue、Consume→Approve/Deny）、exact binding、一回性、first durable decision wins、copy/non-leak、bounded raceを検出する。
- [ ] AC-5: package/READMEはtrusted verifier seamを実WebAuthn/Tailscale identityと扱わず、audit、grant、push authorizationへ昇格しない。実環境依存ケースをblockedのまま明示する。
- [ ] AC-6: 変更は許可3パス、約700〜1,100 additions、新規dependency/configなしに収まり、focused `go test -race`、harness check/distcheck、root `make check`、docs lint、diff checkがPASSする。

### 安定した参照

| 参照ID | 対象 | 固定改訂/ダイジェスト | 用途 |
|---|---|---|---|
| REF-1 | `tasks/TASK-0071-approval-request-store/HANDOVER.md` | main merge `84ce39263edfb1c642e4c02a2f464d7c2a44e8b7` | durable pending/approved/denied/expired transition |
| REF-2 | `tasks/TASK-0072-passkey-challenge-lifecycle/HANDOVER.md` | main merge `1b77cbf3b2298d61e8834e77f268b11224dbfb99` | one-shot challenge/verified binding lifecycle |
| REF-3 | `wiki/semantic/schemas/development-agent-harness-passkey-challenge-lifecycle.md` | main `6095936277d21854dd06108c4e2377923e28023a` | verifier結果を認可へ昇格しない境界 |

### 依存状態

| 依存 | 状態 (`ready` / `pending`) | planning参照 | `ready`後に固定する値 |
|---|---|---|---|
| TASK-0071 / TASK-0072 | `ready` | `84ce39263edfb1c642e4c02a2f464d7c2a44e8b7` / `1b77cbf3b2298d61e8834e77f268b11224dbfb99` | N/A |

### 許可パス

- `tools/dev-agent-harness/internal/approvaldecision/coordinator.go`
- `tools/dev-agent-harness/internal/approvaldecision/coordinator_test.go`
- `tools/dev-agent-harness/README.md`

### 完了経路preflight

| 確認対象 | 結果 | コマンドまたは根拠 |
|---|---|---|
| 完了checker | `ready` | product 3トランザクション、同一candidate独立REVIEW/QA、no-ff completionを使用する |
| 権限 | `ready` | verified decisionからdurable stateを変更する高リスク境界なので`dev-sol`/high。Mainだけがstage/commit/merge/pushする |
| 依存状態と参照 | `ready` | TASK-0071/0072とWikiがmainへ反映済み。Go 1.25、stdlib-only、新規dependencyなし |
| 生成物の有無と更新方法 | `ready` | Go source/test、READMEのみ。配布生成物は変更せず`make distcheck`を確認する |
| 割当ワークツリー | `ready` | `worktrees/TASK-0073-verified-decision-coordinator` |
| Lapログの書込・Schema・`repository annotation` | `not-applicable` | 新規log/Schema/annotationなし |

### 未決事項

- なし

### `Dependency-ready reconciliation`

<!-- 依存ready時にMainがready参照、planning参照との差分、AC/設計/スコープ/QAへの影響、再承認結果を追記する。依存なし又は未readyならN/Aとする。 -->

- N/A

## 背景

TASK-0071はverified actorを受け取るだけで本人確認を行わず、TASK-0072はverified resultを返すだけでdurable stateを変更しない。この間の順序を各HTTP handlerへ直接書くと、callerによるdigest/verifier差し替え、検証前のstate mutation、検証成功だけでapproved扱い、競合時の別decision fallback、永続化失敗後のchallenge再利用が起き得る。このTaskで副作用順序を一箇所に固定し、後続HTTP/Tailscale/WebAuthn層は入力identityとactual verifierを構成するだけにする。

## 検討すべき設計観点

- challenge発行時のdigestはcaller入力ではなくdurable recordから導出する。
- trusted verifierはconstructorで固定し、Completeごとのcallback引数にしない。production wiring以外のfake seamはpackage-private test用に閉じる。
- challenge consumeを先、durable decisionを後に行う。逆順は未検証decisionを永続化するため禁止する。
- store transition失敗時はchallengeを失った安全側状態とし、新challenge発行にはrequestがなおpendingかBeginで再確認する。
- 複数challengeのfirst-winsはcoordinator内の別lockではなくdurable storeの原子的pending transitionを正本にする。
- durable transition成功後の応答喪失はstateのGetで照合し、同challenge replayで成功応答を再構成しない。
- credential stable IDは結果へ返すが、このTaskではauditへ永続化しない。永続audit不能時のside effect順序は後続Taskで設計する。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] 選択した`change_class`の完了経路と`make check`を満たしている。
- [ ] 製品変更の場合: 実装、テスト、文書、同一案の独立REVIEW/QA、完了後の環境依存ケース確認が完了している。
- [ ] 安全契約変更の場合: Mainの意図・スコープ・受け入れ経路確認、契約検査、許可された統制文書差分の確認が完了している。

## 関連コンテキスト

### 意味 Wiki

- `wiki/semantic/schemas/development-agent-harness-approval-request-store.md`
- `wiki/semantic/schemas/development-agent-harness-passkey-challenge-lifecycle.md`

### 判断

- verifierはconstructor固定、digestはstore由来、durable transition成功だけをdecision成功とする。
- challenge consume後のstate failureは同tokenを再利用せず、pendingなら新challengeからやり直す。

### 適用しなかった重要な判断

- なし
