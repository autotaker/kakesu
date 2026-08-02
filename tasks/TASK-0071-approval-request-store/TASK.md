---
task_id: "TASK-0071"
title: "Approval request永続state storeを実装する"
status: done
created_at: "2026-08-02"
---

# TASK-0071 Approval request永続state storeを実装する

## `Planning input packet`（Main Agent所有）

このsectionをPlannerとQAへ渡す唯一の`planning input packet`とし、各内容をPLAN/QA_PLANへ複製しない。

### 目的

TASK-0070のcanonical manifestをbroker-owner領域へ永続化し、承認待ちから承認・拒否・取消・期限切れ・staleまでを、一プロセスの排他所有とcrash-safeな原子的snapshot更新で管理する。スマートフォン承認やpush grantが後から接続できるようにしつつ、このTaskだけでは本人確認、認可発行、実pushを成立させない。

### 対象と対象外

#### 対象

- `internal/approvalstate`へ、owner-only state directoryを排他openするdurable request storeを追加する。
- store rulesへpolicy version、revocation epoch、最大request TTL、最大record数を固定し、production clockとtest-only clock/persistence seamを分離する。
- canonical manifest encodingを`approvalmanifest.Parse`で再検証し、trusted clock、policy/epoch、TTL、request ID一意性を確認して`pending`として原子的に作成する。
- exact request IDとconstant-time digest一致を前提に、`pending → approved/denied/cancelled/expired`、`approved → stale/expired`だけを原子的に許す。approval/denialは後続層が検証済みとして渡すactor IDを束縛するが、store自身はPasskeyを検証しない。
- 一つの固定version snapshotを決定的なrecord順で保持し、temporary file write/fsync/rename/directory fsyncで更新する。process lock、restart recovery、corruption/permission/symlink/duplicate/noncanonical拒否、rename後不確実時のpoisoned fail-closedを扱う。
- record/snapshot/getterはcopy ownershipと固定非漏えいerrorを持ち、mutex下の並行Create/Get/decision/expiry/Closeをrace testで検証する。
- READMEへ状態、durability、single-writer、認可との境界、後続Taskを明記する。

#### 対象外

- HTTP/API/UI、Tailscale Serve/Grant、通知、session/cookie/CSRF、WebAuthn/Passkey登録・challenge・assertion検証を追加しない。
- `push grant`の署名・発行・`consuming/consumed/indeterminate`遷移、Git receive-pack解析、remote old SHA照合、実push、reconciliationを追加しない。
- actor IDを認証済みとstore単独で証明しない。後続Approval serviceだけがverified decision APIを呼べるOS/process境界は別Taskで接続・live確認する。
- append-only監査log、外部DB、複数host/process writer、network filesystem、backup/rotation、key encryptionを追加しない。
- state directoryを作成・chownしない。systemd/tmpfiles/provision/config、実`/var/lib`配置、実UID/permission、restart/rollbackは後続live E2Eとする。
- TASK-0070 manifest形式、既存broker/proxy/launcher、依存、Kakesu本体runtime/Schema、生成/live stateを変更しない。
<!-- safety_contractの場合: 製品コード、test、runtime/build設定、Schema、製品依存、生成製品入力/成果物、外部観測可能な挙動を変更しない。 -->

### 受け入れ条件

<!-- AC-IDはTask内で一意かつ安定させ、観測可能な結果をここに一度だけ記載する。 -->

- [ ] AC-1: `Open`はabsolute cleanな既存root directory、owner-only `0700`、symlink不使用、固定state/lock filenameだけを受理し、Linux/Darwinでnon-blocking exclusive process lockを保持する。同じrootの二重open、relative/traversal、wrong mode/type/symlink、unsupported OSは固定errorでfail-closedとなり、`Close`は一回だけlockを解放する。
- [ ] AC-2: `Create`はTASK-0070のcanonical encodingを再parseし、manifestのpolicy/epochがstore rulesと一致し、`created_at <= trusted now < expires_at`、positive TTLが上限内、record上限内、request ID未使用の場合だけ`pending`を永続化する。同じrequest IDは同一bytesでも自動成功にせずconflictとし、任意digest、state、時刻、pathをcaller入力にしない。
- [ ] AC-3: verified decision/cancel/stale/expiry APIはrequest IDと実manifest digestを照合し、許可された遷移だけを一回適用する。期限到達はdecisionより優先して`expired`へ永続化し、clock rollback、policy/epoch mismatch、digest mismatch、terminal state再遷移、approved以外のstaleを拒否する。`Get`/`ExpireDue`は期限切れactive recordを承認可能として返さない。
- [ ] AC-4: snapshotはversion/generationとrequest ID順のbounded recordsをcanonical JSONとして保存し、各recordのmanifest/digest/state/time/actor整合をOpen時に再検証する。mutationはtemp 0600 regular fileへの全write+fsync、atomic rename、directory fsync後だけmemoryへcommitする。rename前FAILは旧memory/diskを維持し、rename後の結果不確実はstoreをpoisonして再open/reconciliationまで全操作を拒否する。partial/trailing/unknown/duplicate/noncanonical/oversize snapshot又は残存tempを成功として推測しない。
- [ ] AC-5: public record/encoding/list getterはcaller mutationから内部状態を分離し、error/diagnosticは固定classだけでroot、request/actor ID、repository/ref/SHA、digest、manifest/snapshot bytes、lower errorを出さない。nil/closed/poisoned store、同時Create/decision/Get/Closeでpanic/data race/deadlockせず、失敗したmutationがgeneration又はrecordを部分更新しない。
- [ ] AC-6: candidateは承認済み5パス・約900〜1,200 changed linesを目安とし、stdlibと既存`approvalmanifest`だけを使い、HTTP、credential、Git、external DB、config/deploy/generated/live stateを含まない。focused race、restart/corruption/failure-injection fixture、harness `make check`/`make distcheck`、candidate gate root `make check`、`git diff --check`がPASSする。

### 安定した参照

| 参照ID | 対象 | 固定改訂/ダイジェスト | 用途 |
|---|---|---|---|
| REF-1 | Harness設計 §10.1〜10.2、§12 | main `be98c1f` | request state、期限、fail-closed永続化、後続grant境界 |
| REF-2 | TASK-0070 push approval manifest | main `be98c1f` | canonical encoding/digest、ordered ref、validとauthorizationの分離 |

### 依存状態

| 依存 | 状態 (`ready` / `pending`) | planning参照 | `ready`後に固定する値 |
|---|---|---|---|
| TASK-0070 | `ready` | main `be98c1f` | `approvalmanifest.Build/Parse`とimmutable encoding/digest contract |

### 許可パス

- `tools/dev-agent-harness/internal/approvalstate/store.go`
- `tools/dev-agent-harness/internal/approvalstate/store_test.go`
- `tools/dev-agent-harness/internal/approvalstate/lock_unix.go`
- `tools/dev-agent-harness/internal/approvalstate/lock_unsupported.go`
- `tools/dev-agent-harness/README.md`

### 完了経路preflight

| 確認対象 | 結果 | コマンドまたは根拠 |
|---|---|---|
| 完了checker | `ready` | product 3トランザクション、同一candidate独立REVIEW/QA、no-ff completionを使用する |
| 権限 | `ready` | 承認状態とdurable file/lockを扱う高リスク境界なので`dev-sol`/highを使う。Mainだけがstage/commit/merge/pushする |
| 依存状態と参照 | `ready` | TASK-0070がmainとWikiへ反映済み。新規外部dependencyなし |
| 生成物の有無と更新方法 | `ready` | Go source/test、READMEのみ。配布生成物は変更せず`make distcheck`で再生成可能性を確認する |
| 割当ワークツリー | `ready` | `worktrees/TASK-0071-approval-request-store` |
| Lapログの書込・Schema・`repository annotation` | `not-applicable` | 新規log/Schema/annotationなし |

### 未決事項

- なし

### `Dependency-ready reconciliation`

<!-- 依存ready時にMainがready参照、planning参照との差分、AC/設計/スコープ/QAへの影響、再承認結果を追記する。依存なし又は未readyならN/Aとする。 -->

- N/A

## 背景

TASK-0070で承認内容の同一性は固定できたが、現在はrequestを後からスマートフォンで承認するためのdurable stateがない。UIやPasskeyから先に作ると、再起動でpending/approvedが失われる、二重writerでdecisionを上書きする、期限切れをapprovedとして扱う、partial write後に状態を推測する危険がある。このTaskで認可を発行しないsingle-writer storeを先に完成させ、後続層は検証済みdecisionを入力するだけにする。

## 検討すべき設計観点

- manifest bytesをstore独自に正規化せず、TASK-0070 parserを唯一のcontent validatorとして再利用する。
- durable successはrenameだけでなくdirectory fsyncまでとし、それ以後の不確実性は成功/失敗を推測せずpoisonする。
- process lockは複数writerをfail-fastで拒否する。single process内の並行操作はmutexで直列化する。
- active stateの期限切れはauthorizationより優先する。wall clock rollbackを安全側へ拒否し、再起動後もrecord timestampとの単調性を検査する。
- `approved`はverified decisionが保存された状態であり、push permissionではない。grant atomicsは次Taskへ残す。
- actor IDは監査対応用の非secret識別子として保存するが、errorへ出さず、Passkey assertionやcredential bytesを保存しない。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] 選択した`change_class`の完了経路と`make check`を満たしている。
- [ ] 製品変更の場合: 実装、テスト、文書、同一案の独立REVIEW/QA、完了後の環境依存ケース確認が完了している。
- [ ] 安全契約変更の場合: Mainの意図・スコープ・受け入れ経路確認、契約検査、許可された統制文書差分の確認が完了している。

## 関連コンテキスト

### 意味 Wiki

- 未調査

### 判断

- 未調査

### 適用しなかった重要な判断

- なし
