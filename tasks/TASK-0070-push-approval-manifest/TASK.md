---
task_id: "TASK-0070"
title: "Push承認manifestのcanonical境界を実装する"
status: draft
created_at: "2026-08-02"
---

# TASK-0070 Push承認manifestのcanonical境界を実装する

## `Planning input packet`（Main Agent所有）

このsectionをPlannerとQAへ渡す唯一の`planning input packet`とし、各内容をPLAN/QA_PLANへ複製しない。

### 目的

後続のApproval state store、Passkey challenge、one-shot `push grant`が同じpush内容へ結び付くための、曖昧さのない承認manifest値を実装する。構造化された一件のpush proposalを厳格に検査し、全参照更新を含むcanonical bytesと実際に内容から計算したdomain-separated SHA-256 digestを一意に生成・再検証できるようにする。このTaskだけではpushを許可せず、承認状態、UI、永続化、Git wire解析を持たない。

### 対象と対象外

#### 対象

- `internal/approvalmanifest`へ、構造化push proposalからimmutableなmanifestを構築する小さなGo packageを追加する。
- `request_id`、agent/workspace identity、repository、固定GitHub HTTPS remote、全ref update、policy version/revocation epoch、created/expiresを上限付きで厳格に検査する。
- ref updateの入力順序を保持し、重複refを拒否する。v1は`refs/heads/*`の安全な部分集合、lowercase 40桁object ID、create/update/force/deleteの整合だけを受理する。
- digest対象からderivedな`request_digest`自身だけを除外し、固定field順・固定表現のcanonical payloadへdomain separationを付けてSHA-256を計算する。公開manifest encodingには検証済みdigestを含める。
- 自身が出力するcanonical encodingだけをstrict parseでき、unknown/duplicate/missing field、non-canonical表現、digest不一致、過大入力を拒否する。再encodeと再parseが同じbytes/digestになることを保証する。
- caller所有slice/bytesを保持せず、getterもcopyを返す。errorは固定分類とfield/indexだけを返し、repository、ref、SHA、identity、digest又はraw inputを含めない。
- focused unit/fuzz-style bounded testsとREADMEで保証・制限・後続境界を明記する。

#### 対象外

- raw `git-receive-pack`/pkt-line/side-band/capability negotiationの解析、Gitコマンド起動、remote old SHA取得、force推定、GitHub通信を追加しない。
- Approval状態機械、永続store、通知、HTTP/UI、Tailscale Serve/Grant、WebAuthn/Passkey、session/cookie/CSRFを追加しない。
- `push grant`の署名・発行・原子的消費・reconciliation、実credential、実push、auditを追加しない。
- manifest constructorが時刻、request ID、policy又はidentityを生成・信頼判定しない。trusted clock、ID一意性、TTL/policy判断は後続state storeが所有する。
- branch ref以外、SHA-256 object format、GitHub以外のremote、複数repository、submodule/LFS、既存control/proxy/launcher挙動を追加又は変更しない。
- config Schema、依存、Kakesu本体runtime/Schema、deploy/generated/live stateを変更しない。
<!-- safety_contractの場合: 製品コード、test、runtime/build設定、Schema、製品依存、生成製品入力/成果物、外部観測可能な挙動を変更しない。 -->

### 受け入れ条件

<!-- AC-IDはTask内で一意かつ安定させ、観測可能な結果をここに一度だけ記載する。 -->

- [ ] AC-1: constructorは必須identity/request/policy field、canonical lowercase `owner/repo`、exact `https://github.com/<owner>/<repo>.git`、UTC whole-secondかつ`created_at < expires_at`の時刻を検査し、missing、NUL/non-ASCII、境界超過、remote/repository不一致、非canonical時刻を固定errorで拒否する。値の生成、現在時刻/TTL/policy妥当性の判断は行わない。
- [ ] AC-2: 1〜32件のref updateを入力順のまま束縛し、重複ref、過大入力、v1安全subset外のref、uppercase/non-40-hex object IDを拒否する。zero object IDをcreate/delete sentinelとして扱い、create、通常update、明示force、deleteのflag/object整合を検査し、zero→zero、同一old/newその他のno-opを拒否する。
- [ ] AC-3: canonical payloadはfield order、JSON spelling、array order、timestamp、integer、booleanを一意にし、derived `request_digest`だけを除いたmanifest全体から、固定v1 domain prefix付きSHA-256を計算する。公開encodingの`request_digest`は実計算値`sha256:<64 lowercase hex>`であり、任意digest入力を受け付けない。同一proposalは同一bytes/digest、束縛field又はref順序の一箇所でも変わればdigestが変わる。
- [ ] AC-4: strict parserはcanonical encoderの出力だけを受理し、unknown/duplicate/missing key、field順/空白/escape/number/time/digestの非canonical表現、trailing data、digest不一致を拒否する。parse→encodeはbyte-identicalであり、parserとconstructorは同じvalidationを通る。
- [ ] AC-5: packageは入力slice/raw bytesを保持せず、ref updates/encoding getterの変更が内部値又は後続結果を変えない。公開error/diagnosticは固定categoryとfield又はupdate indexだけで、入力値、identity、repository、remote、ref、object ID、digest、canonical/raw bytesを出さない。panicせずboundedに失敗する負例を持つ。
- [ ] AC-6: candidateは3許可パス・約800〜1,100 changed linesを目安とし、外部依存、I/O、network、clock、randomness、Git subprocess、状態永続化を含まない。focused race test、harness/root `make check`、`make distcheck`、`git diff --check`がPASSする。

### 安定した参照

| 参照ID | 対象 | 固定改訂/ダイジェスト | 用途 |
|---|---|---|---|
| REF-1 | Harness設計 §9.4、§10.1〜10.2 | main `3478e63` | push read拒否、manifest全体の束縛、Passkey/push grant共通digest、後続状態機械との境界 |
| REF-2 | TASK-0069 agent session launcher | main `3478e63` | 現sessionはpush/Approval対象外であり、read capabilityからwriteへ昇格させない境界 |

### 依存状態

| 依存 | 状態 (`ready` / `pending`) | planning参照 | `ready`後に固定する値 |
|---|---|---|---|
| Harness設計 / TASK-0069 | `ready` | main `3478e63` | manifest最小fieldと現sessionのpush拒否境界 |

### 許可パス

- `tools/dev-agent-harness/internal/approvalmanifest/manifest.go`
- `tools/dev-agent-harness/internal/approvalmanifest/manifest_test.go`
- `tools/dev-agent-harness/README.md`

### 完了経路preflight

| 確認対象 | 結果 | コマンドまたは根拠 |
|---|---|---|
| 完了checker | `ready` | product 3トランザクション、同一candidate独立REVIEW/QA、no-ff completionを使用する |
| 権限 | `ready` | 承認digestの安全境界なので`dev-sol`/highを使う。Mainだけがstage/commit/merge/pushする |
| 依存状態と参照 | `ready` | 設計とTASK-0069がmain `3478e63`へ反映済み。新規runtime dependencyなし |
| 生成物の有無と更新方法 | `ready` | Go source/test、READMEのみ。生成物は変更せず`make distcheck`で配布再生成可能性を確認する |
| 割当ワークツリー | `ready` | `worktrees/TASK-0070-push-approval-manifest` |
| Lapログの書込・Schema・`repository annotation` | `not-applicable` | 新規log/Schema/annotationなし |

### 未決事項

- なし

### `Dependency-ready reconciliation`

<!-- 依存ready時にMainがready参照、planning参照との差分、AC/設計/スコープ/QAへの影響、再承認結果を追記する。依存なし又は未readyならN/Aとする。 -->

- N/A

## 背景

現状はread-only sessionまで完成しているが、push承認を構成する状態store、Passkey、grantが共有できるcontent identityがまだない。先にUIやstoreを作ると、各層が異なるJSON整形、ref集合又はdigest対象を持ち、`new_sha`だけの承認や再encode差によるgrant取り違えを招く。このTaskで小さなpure value boundaryを先に固定し、後続は生成済みcanonical bytes/digestを利用する。ただしGit wire bytesとの一致は未実装であり、この成果だけでpush許可を成立させない。

## 検討すべき設計観点

- canonicalizationは「JSONとして読める」ではなく一つのbyte列だけを許し、parserもround-trip equalityで非canonical入力を拒否する。
- `request_digest`は形式検査ではなく、payloadから再計算してconstant-timeに照合する実security propertyとする。digest自身をdigest対象へ含める循環は避ける。
- ref配列はsetへ変換せず入力順序も承認対象へ含める。重複refは意味が曖昧なため拒否する。
- v1の受理範囲を通常のGitHub branch pushに狭める。未知又は表現不能なref/object formatは寛容に通さず後続version/taskへ送る。
- manifest packageはtrusted clock、policy、remote観測又はauthorizationを所有しない。validな値はapproved/grantedを意味しない。
- immutable boundaryはGoのslice aliasにも適用し、constructor、parser、getterのmutation regressionを負例で検出する。

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
