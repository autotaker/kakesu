---
task_id: "TASK-0070"
change_class: "product"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "sol-high"
approved_dev_profile_reason: "approval、Passkey、one-shot push grantが共有するcontent identityを決めるpure Go security boundaryであり、strict parser、canonical encoding、digest、immutable値の相互整合を一候補内で監査可能にするため。"
approved_dev_profile_risk_signals:
  - "approved/grantedを意味してはならないcontent digestが後続の認可境界へ渡る"
  - "canonical parserの緩み又はdigest対象漏れがref updateの取り違えを生む"
  - "slice/raw-byte alias又は診断が入力値を漏らす境界"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-02T06:04:03Z"
classification_approved_by: "main-agent-sol-high"
classification_approved_at: "2026-08-02T06:04:03Z"
classification_approval_reason: "承認対象のcanonical encodingとdigestを外部観測可能なGo package contractとして追加する製品変更。"
# safety_contract_version: 2
# safety_contract_planned_paths: []
# safety_contract_generated_paths: []
---

# PLAN: Push approval manifest canonical boundary

## 根拠と分類

唯一の要求根拠は`TASK.md`の`Planning input packet`である。新しいGo packageとunit test、利用者向けREADMEによって承認対象の値表現を外部観測可能な契約として追加するため、`change_class`は`product`とする。固定参照REF-1の「receive-packはread capabilityで通さず、manifest全体をPasskey/push grantへ束縛する」境界と、REF-2の「現sessionはwriteへ昇格しない」境界を採用する。ここで生成する値は有効なproposalを表すだけで、approved、granted、consumed、又は実push可能を表さない。

DEV profileは`dev-sol`/high（表記`sol-high`）とする。MainはDEV開始前に本PLANとTASK-firstで別作成する`QA_PLAN.md`の意図、scope、受け入れ経路を確認して承認を記録する。独立PLANレビューは置かない。標準3トランザクション、同一candidateからの独立REVIEW/QA、candidateのno-ff completion checkはMainが所有し、Planner/DEV/Reviewer/QAはstage、commit、merge、pushをしない。

candidateは次の3許可パスだけを変更し、追加・削除合計を約800〜1,100行に保つ。

- `tools/dev-agent-harness/internal/approvalmanifest/manifest.go`
- `tools/dev-agent-harness/internal/approvalmanifest/manifest_test.go`
- `tools/dev-agent-harness/README.md`

Go標準ライブラリ以外の依存、I/O、network、clock、randomness、Git subprocess、状態永続化、config/Schema、Kakesu runtime、既存control/proxy/launcher、generated `configure`、deploy/live stateは変更しない。raw Smart HTTP/pkt-line/capability negotiation、remote old SHAの観測、forceの推定、GitHub通信、approval state machine、Passkey、grantの署名・発行・消費・auditもこのcandidateに含めない。

## package contract とcanonical形式

`internal/approvalmanifest`を、構造化proposalを一回検査してimmutable manifestへ変換するpure-value ownerとして新設する。公開APIは最小限に固定する。

- `Proposal`はrequest、agent/workspace、repository、remote、ordered `[]RefUpdate`、policy、revocation epoch、created/expiresを入力として持つ。`RefUpdate`はref、expected old SHA、new SHA、force、deleteだけを持つ。proposalはcaller所有であり、packageは値を生成又は意味判定しない。
- `Build(Proposal) (*Manifest, error)`はproposalを検査してcanonical public bytesとdigestを生成する。caller supplied digest、clock、ID、policy、authorization依存は受け取らない。
- `Parse([]byte) (*Manifest, error)`はpackage自身が生成したpublic encodingだけを受理する。成功後の`Encoding()`、`Digest()`、scalar getter、`RefUpdates()`はimmutable stateのcopyまたはscalarを返す。`Manifest`の内部fieldは非公開にし、`Encoding()`と`RefUpdates()`は常に新しいcopyを返す。
- `ErrorClass`、`Field`、`Error`、`ClassOf`（必要なら`LocationOf`）で、固定error分類と固定field名、ref update indexだけを判別可能にする。`Error()`もinput値を組み込まず、repository、remote、identity、ref、SHA、digest、raw/canonical bytes、下位parser errorを返さない。unexpected errorも固定internal/encode分類へ畳む。

V1のconstantsをsourceに一箇所で固定する。format version、max proposal string lengths、max 32 refs、ref/object-ID/encoding byte caps、zero object ID、digest label、domain prefix、固定timestamp layoutを散在させない。上限は最大で32の256-byte refと40-byte IDs、およびbounded identity/policy fieldsを十分収め、parse前にwhole inputを上限で拒否できる値（32 KiB）にする。これにより不正inputがallocation、CPU、diagnosticを不定に増やさない。

proposal検査はconstructorとparserが共有するprivate validation関数にまとめる。全text fieldはASCII、NULなし、上限内、固定token grammarとし、repositoryだけはlowercase canonical `owner/repo`に限定する。remoteは検査済みrepositoryから組み立てたexact `https://github.com/<owner>/<repo>.git`とのbyte一致を要求する。時刻はUTC location、nanosecond 0、strict RFC3339 `Z` whole-second formに限定し、`created_at < expires_at`だけを判定する。現在時刻、最大TTL、policyの有効性、request IDの一意性は後続storeの責務として評価しない。

refは入力順を維持するsliceとして検査・copyし、mapへ並べ替えない。`refs/heads/`直下のV1安全subset（ASCII path segment、空segment、`..`、`.lock`、control、空白、`~`、`^`、`:`、`?`、`*`、`[`、`\\`を拒否し、長さを固定上限内）だけを受理する。ref文字列をkeyとするprivate seen setは重複検出だけに使い、output順へ影響させない。old/newはexact 40 lowercase hexかzero sentinelであり、次の4表現だけを正規とする。

| 種別 | expected old | new | force | delete |
|---|---|---|---|---|
| create | zero | non-zero | false | false |
| normal update | non-zero | 異なるnon-zero | false | false |
| force update | non-zero | 異なるnon-zero | true | false |
| delete | non-zero | zero | false | true |

zero-to-zero、同一non-zero old/new、sentinelとflagの組合せ違いを拒否する。forceはwireから推定せず、proposalの明示値をそのまま束縛する。

canonical payloadはfixed-order structを`encoding/json`へ渡して一意に作る。payload fieldはformat version、request/identity/repository/remote、ordered ref array、policy/revocation、timestampsをこの順で持ち、`request_digest`は含めない。public encodingは同じ順序の全fieldにderived `request_digest`を最後に追加したcompact JSON一文書とする。stringsをstrict safe ASCIIへ限定し、timestampsをV1 layoutへ整形し、integerは`uint64`、booleanはJSON literalにするため、map、custom number、nullable/optional field、HTML/raw escapeの裁量を持ち込まない。

digestはpayload bytesだけに固定V1 domain prefixをbyte連結して`sha256.Sum256`する。public fieldは`sha256:`と64桁lowercase hexの実計算値であり、任意digest入力はない。parse時はpayloadを再encodeして同じdigestを再計算し、supplied digestとの比較に`crypto/subtle.ConstantTimeCompare`を使う。最後に再生成したpublic bytesとraw bytesの完全一致を要求する。この順序でunknown、duplicate、missing、field order、space、escape、numeric/time/digest spelling、trailing dataを個別に受理せず、`Parse`→`Encoding`のbyte identityを構造上保証する。

JSON入力は先にbounded token scannerで一文書だけを走査し、objectごとのduplicate key、構文、trailing dataを固定分類にする。その後`DisallowUnknownFields`付きtyped decodeを行い、unknown field、型、欠損値を固定field/parse分類へ畳む。Go decoderが通常受け入れるduplicate又は代替表現をtyped decodeの成功と見なさず、scanner、shared semantic validation、digest照合、byte equalityの全段を通過条件とする。parserはpackage内encoderを再利用し、別canonicalizerやaccept-then-normalize経路を持たない。

## AC対応

| AC-ID | 設計判断 | 変更パス | 実施順序 | 失敗時の扱い |
|---|---|---|---|---|
| AC-1 | shared validatorでidentity/request/policy/repository/remote/timeを検査し、remoteをcanonical repositoryから導出する。clock/TTL/policy/ID生成を持たない。 | `manifest.go`、`manifest_test.go`、`README.md` | 1, 5 | field付き固定invalid分類。値、raw input、lower errorを返さずmanifestを作らない。 |
| AC-2 | ordered copied ref slice、seen setだけのduplicate検出、heads-only grammar、40-lowercase hex/zero sentinelと4-state表でV1更新を限定する。 | `manifest.go`、`manifest_test.go` | 1--2 | index/fieldだけを返し、invalid/no-op/unsupported updateは全体を拒否する。 |
| AC-3 | fixed-order payload/public wire struct、domain-separated SHA-256、derived-only digest、ref orderをhash inputにする。 | `manifest.go`、`manifest_test.go`、`README.md` | 2--3 | digestが入力又は循環対象になる経路を持たず、任意のbound-field/order差は別bytes/digestになる。 |
| AC-4 | bounded duplicate-aware scan、strict typed decode、shared validation、constant-time digest check、exact reencode comparisonでencoder outputだけをparseする。 | `manifest.go`、`manifest_test.go` | 3--4 | syntax/unknown/duplicate/missing/noncanonical/trailing/digest mismatchを固定parse分類で拒否する。 |
| AC-5 | private storage、constructor/parserのinput copy、slice/encoding getter copy、non-leaking typed errorを同一packageに閉じる。 | `manifest.go`、`manifest_test.go` | 1--4 | nil/oversize/malformed/mutated caller dataでもpanicせずbounded errorを返す。 |
| AC-6 | 3許可パス、約800〜1,100行、stdlib-only pure packageを保ち、focused race/harness/dist/root/scope checksをcandidate evidenceへ結ぶ。 | 許可済み3パス | 1--5 | scope、line budget、test、distribution確認のFAILはcandidateを進めずMainへ戻す。 |

## 変更予定

| パス | 変更内容 |
|---|---|
| `tools/dev-agent-harness/internal/approvalmanifest/manifest.go` | V1 value types、fixed safe error surface、shared semantic validator、bounded duplicate-aware JSON scanner、fixed-order payload/public encoders、domain-separated SHA-256、strict parser、immutable Manifest gettersを実装する。 |
| `tools/dev-agent-harness/internal/approvalmanifest/manifest_test.go` | deterministic valid fixtures、validation/error-location matrix、canonical/digest golden assertions、parser mutation corpus、ownership/non-leak、bounded fuzz-style/race testsを実装する。 |
| `tools/dev-agent-harness/README.md` | manifest packageのpurpose、proposal→manifest usage、strict bytes/digest/immutability contract、V1 ref limits、後続approval/grantとGit wire/live environmentの非責務を追記する。 |

## 実装手順

1. `manifest.go`にV1 constants、input/output types、error classesとnon-sensitive field/index location、non-exported immutable storageを定義する。constructorとparserで再利用するASCII/token/repository/remote/time/ref/object-ID/update validationを実装し、proposal sliceをvalidator通過後にcopyする。
2. fixed-order wire structsを定義し、payload encode、domain-prefix digest、public encodeを小さいprivate関数に分ける。`Build`はshared validation→payload bytes→digest→public bytes→internal copiesの一本道だけにし、payload/public encoderが任意mapやcaller digestを扱えないようにする。
3. bounded raw input scannerを追加する。duplicate keysとtrailing dataをdecoderの曖昧さに依存せず検出し、typed decodeのunknown/missing/type failureを固定errorへ変換する。`Parse`はdecode→shared validation→canonical payload/digest→constant-time supplied digest check→public-byte equality→immutable manifest生成の順に固定する。
4. focused testsを追加する。valid create/normal/force/deleteのordered multi-ref fixtureから固定JSONとdigest、Build/Parse byte identity、各bound field/ref orderの一箇所変更がdigestを変えることを検証する。error assertionはclass/field/indexだけを確認し、input内容をmessageに期待しない。
5. hostile input、ownership、documentationを仕上げる。all required scalar/text/time boundaries、non-ASCII/NUL、case/remote mismatch、duplicate/out-of-set refs、object/flag/no-op combinations、oversize、unknown/duplicate/missing/order/whitespace/escape/number/time/digest/trailing mutationsをtable testsにする。caller `[]RefUpdate`とraw `[]byte`、returned updates/encodingを成功後に書換えてもmanifest bytes/digestが不変であることを確認する。READMEは外部I/O/authorizationを行わないpure boundaryであることを明記する。

## 検証計画

focused suiteは外部通信、filesystem、clock、randomness、Git、approval storeを使わない。fixture timestamp、request/identity/policy、ref valuesを明示し、golden outputは独立に固定したV1 bytes/digestと比較する。fuzz-style testは`Parse`入力を32 KiBへtruncateして、任意bytesでpanicせず、成功時だけ`Parse(data).Encoding()==data`、再parse後もsame digest/bytesとなる性質を確認する。parallel mutation/copy getter testは`go test -race`で実行する。

- semantic matrix: required fields、ASCII/NUL/length、canonical lowercase repository、exact remote、UTC second time、strict ordering、ref duplicates/safe grammar、lowercase 40-hex、zero/create/update/force/delete/no-opを、classとfield/indexだけで検査する。
- canonical/digest matrix: fixed output field/array order、timestamp/integer/boolean spelling、domain prefixを含むknown SHA-256、digest field自身をpayloadから除外すること、identity/repository/remote/policy/epoch/time/ref field又はref順序の全変更でdigestが変わることを検査する。
- parser matrix: unknown/duplicate/missing key、wrong nested shape、duplicate ref data、space/newline、key order、escaped safe string、alternate number、offset/fractional time、uppercase/bad digest、extra document/trailing bytes、digest mismatchを拒否し、canonical encoder出力だけがstrict parseを通ることを検査する。
- ownership/non-leak matrix: input proposal slice、parser raw bytes、returned update slice、returned encodingをmutateしても後続getter/digest/encodingが変わらないこと、nil/oversize/untrusted valuesがpanicしないこと、all public error formsにfixture identity/repository/remote/ref/SHA/digest/raw bytesが含まれないことを検査する。

candidate固定前にDEVは少なくとも次を実行する。

```sh
cd tools/dev-agent-harness && GOCACHE=$PWD/.build/go-cache go test -race ./internal/approvalmanifest
cd tools/dev-agent-harness && make check
cd tools/dev-agent-harness && make distcheck
make task-check TASK=TASK-0070
git diff --check
```

Mainは同一candidateにroot `make check`を一回実行する。QA_PLANは各ACに`focused-rerun`又は`evidence-review`を根拠付きで割当てる。canonical parser、digest target、immutability、non-leakはnegative/race evidenceが不足すると`evidence-review`単独でPASSにしない。実GitHub、Git Smart HTTP解析、実remote old SHA、Passkey、grant atomics、OS/network/deploymentは本Taskの対象外であり、hermetic PASSの代替にしない。

## リスクと復旧

- JSON decoderの寛容さが複数表現を同じ値へcollapseするリスクは、duplicate-aware scan、typed strict decode、shared validation、reencode byte equalityの多層判定で抑える。parserは正規化した別の表現を返さない。
- digest対象からrepository、remote、identity、policy/epoch、force/delete、old SHA、ref order等が落ちるリスクは、payload wire structを唯一のhash sourceにし、field-by-field digest-difference testsとknown digest goldenで検出する。
- parse/build validationの乖離はshared validatorとParse後のBuild相当payload recomputationで抑える。time/TTL/policyの将来判断を先取りして既存proposalを無効化しない。
- Go slice/raw-byte aliasが後続grant bindingを変えるリスクは、constructor/parser/getterのcopy ownershipとmutation regression/race testsで抑える。scalar strings以外のmutable storageを公開しない。
- errorが安全境界をdiagnostic oracle化するリスクは、fixed class/field/indexだけのtyped errorsとnegative message scansで抑える。

復旧時は3許可パスのcandidate製品差分だけを戻し、新規`approvalmanifest` packageを除去して既存read-only sessionのpush拒否を保つ。state store、grant、runtime、実remoteにはこのcandidateが変更を加えないため、外部stateのmigrationやcleanupはない。復旧後はfocused race suite、harness/root `make check`、`make distcheck`、`make task-check TASK=TASK-0070`、`git diff --check`を再実行する。

## 引き継ぎ条件

DEVはMain承認済みPLANと独立QA_PLANの後、3許可パスだけで一回のcandidateを固定する。REVIEWとQAは同一candidateを互いのPASSを待たず独立に評価する。Mainだけがcandidate identifier、stage/commit/merge/push、completion no-ff check、main統合を所有する。

candidateにはapproval state、HTTP/UI、Passkey、grant、actual push、Git wire parser、GitHub通信、clock/TTL判断、ID/policy生成、storage/audit、runtime/config/dependency/generated/deploy/live-stateの変更を含めない。packetのdependency-ready reconciliationはN/Aであり、REF-1/REF-2のread-to-write separationを再解釈又は緩和しない。

## 未解決事項

- なし。V1の具体的なsafe ref grammar、field lengths、32 KiB parse cap、domain prefix、fixed error class/field enumは本PLANに従いpackage内contractとして一回だけ固定する。将来のtag ref、SHA-256 object format、Git wire capabilityはV2又は別Taskで明示追加する。

## main Agentレビュー

- [x] TASKの全AC-IDへ設計判断、パス、順序、失敗時の扱いを対応させ、条件本文を複製していない。
- [x] pure Go boundary、API、shared validation、canonical encoding、domain-separated digest、strict parse、ownership/error surfaceを具体化している。
- [x] `dev-sol`/high、3許可パス、約800〜1,100行、3トランザクション、独立PLAN reviewなしを記録している。
- [x] QA_PLANがTASK-firstで独立作成されている。
- [x] `dependency-ready reconciliation`と完了経路preflightが完了している。
- [x] DEV開始を承認した。
