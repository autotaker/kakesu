---
task_id: "TASK-0042"
change_class: "product"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "luna-xhigh"
approved_dev_profile_reason: "固定4-fileの新規internal package、標準libraryのRSA/JWT、hermetic testに限定し、実network・token交換・runtime接続を含めないため"
approved_dev_profile_risk_signals: []
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T06:43:37Z"
planning_reviewed_by: "reviewer-agent-terra-medium"
planning_review_decision: "pass"
planning_reviewed_at: "2026-08-01T06:43:20Z"
classification_approved_by: ""
classification_approved_at: ""
classification_approval_reason: ""
# safety_contract_version: 2
# safety_contract_planned_paths: []
# safety_contract_generated_paths: []
---

# TASK-0042 PLAN

## AC対応

TASKの条件本文を再掲せず、`planning input packet`のAC-IDに設計を対応させる。

| AC-ID | 設計判断 | 変更パス | 実施順序 | 失敗時の扱い |
|---|---|---|---|---|
| AC-1 | `Load`はtrusted deploymentから渡されるdirectory pathだけを入口にし、4 basenameはpackage内定数に固定する。Linuxはdirectory FDを一度だけ開いてdirectory policyを確認し、同FD相対の`openat`で各FDを取得する。`O_NOFOLLOW`、`O_CLOEXEC`、`O_NONBLOCK`、open後のregular-file確認、上限付き読込み、read前後のFD metadata照合を同じload境界に閉じる。非Linuxは同じbasenameとfile policyを保持する開発test実装へ分岐する。 | `tools/dev-agent-harness/internal/brokercredentials/` | 1–2 | path・ownership・mode・node種別・size・metadata変化・読込み失敗を、詳細を含まない固定load errorに正規化する。部分bundleは返さない。 |
| AC-2 | 共通のtext正規化とfield別validatorを用意し、LF許容後のbyte境界、visible ASCII、client ID、正の10進`int64`、provider非依存のOpenAI keyを検査する。 | `tools/dev-agent-harness/internal/brokercredentials/` | 3 | 形式、範囲、overflowを固定load errorに畳み込み、入力値をerrorまたは保持状態へ渡さない。 |
| AC-3 | PEM入力は単一block・余剰dataなしに限定してparseし、unencrypted PKCS#1/PKCS#8 RSAのみを受理する。鍵長と`rsa.PrivateKey.Validate`を検査後、bundle内部だけにkeyを保持し、PEM/raw keyを返すexport・文字列化・marshal面を作らない。 | `tools/dev-agent-harness/internal/brokercredentials/` | 4 | parse、形式、暗号化、鍵種、鍵長、妥当性の失敗を固定load errorにし、parser detailやPEMを露出しない。 |
| AC-4 | bundleは内部の現在時刻取得を使い、`now.UTC().Unix()`の整数秒から固定3-field payload `iat=now-60`、`exp=now+540`、`iss=client ID`をJSON数値/文字列で構成する。標準libraryで固定header/payloadをbase64url化し、各呼出しでRS256署名する最小`GitHubAppJWT` APIを提供する。同じ基準秒の決定的な署名は同一JWTを許す。 | `tools/dev-agent-harness/internal/brokercredentials/` | 5 | 署名の失敗は、JWT・鍵・内部detailを含まない固定JWT errorに正規化する。 |
| AC-5 | exported APIはtrusted brokerが必要とするclient ID、installation ID、OpenAI API key、短命JWTに限定する。読込み、parse、署名の責務を当packageへ封じ、環境・command line・network・process・永続書込み・logへの依存を導入しない。READMEはbroker-onlyの配置、利用境界、非対象を説明する。 | `tools/dev-agent-harness/internal/brokercredentials/`、`tools/dev-agent-harness/README.md` | 1、6 | 誤用を助長するsecret source/APIを追加せず、すべての公開エラーを非漏洩の固定値に保つ。 |
| AC-6 | hermetic unit testで正系、file policy、text、RSA、JWT、caller入力不変、固定errorと秘密非漏洩を、それぞれpackage境界で観測する。RSA keyはtest実行時に生成し、実secret fixtureを置かない。 | `tools/dev-agent-harness/internal/brokercredentials/` | 2–5、7 | Linux固有のFD保証をLinux testで検出し、非Linuxでは同一policyの開発test実装を検証する。実UID/chown/root隔離および実GitHub受理はPASSと主張せず後続live E2Eへ残す。 |

## 関連Wikiと判断

- REF-1のbroker-only置換とAgent非露出を守り、bundleはTASK-0041のtrusted resolver内部でのみ消費可能な値を返す。transaction APIやAgent側へcredential sourceを広げない。
- REF-3に合わせ、外部JWT moduleを追加せず標準`crypto/rsa`、`crypto/sha256`、`encoding/base64`、`encoding/json`でRS256と短命claimを構成する。
- REF-4に合わせ、実secretをmodel-visible state、ログ、error、fixtureへ残さず、OpenAI keyのprefixまたはprovider固有長を仕様化しない。
- TASK-0041はreadyであり、今回の境界をinstallation token交換またはegress transaction接続まで拡張しない。

## 補足設計

### Linux openat境界

- Linux専用の読み込み経路は、callerのdirectory文字列を認可済みdirectory FDへ解決する一回だけに使う。各secretは固定basenameだけをFD相対で開き、basename以外の相対path、path再stat、再openをしない。
- directoryと各fileの所有者・permissionを実効UIDと照合する。directoryはowner read/execute、fileはowner readかつ実行不可を要し、group/other permissionを拒否する。
- 開いたfile FDをregular fileと確認してから、定めた上限内で読込む。open後・read後のmetadataが一致しなければ競合として拒否し、FIFO等は`O_NONBLOCK`とnode種別検査によりblockせず拒否する。
- non-Linux実装は本番supportを標榜しない。開発testで同じ固定basename、owner/mode、regular-file、上限、読込み前後の安定性というpolicyを表現する範囲に留める。

### 責務・境界・不変条件

- `Load`は完全に検証済みのbundleだけを返し、任意の失敗でbundleを返さない。bundleは入力byteを保持せず、private keyはJWT署名の内部用途だけにする。
- text validatorは正規化後の値だけを受理判定に使うが、secretをformat、marshal、log、エラー、永続出力へ渡さない。公開errorはload用とJWT用の固定値に限定する。
- JWT生成は呼出しごとに署名処理を行い、header/payloadのfieldを拡張可能なmapにせず固定構造にする。NumericDateは整数Unix秒のJSON数値とする。同一秒・同一claimのPKCS#1 v1.5署名が同一JWTになることを許し、無用なnonceやrandom claimを追加しない。署名鍵、parser detail、トークン自体は返却エラーに含めない。
- packageはfilesystem読み込みとin-memory検証・署名のみを担い、HTTP、network、環境変数、CLI、process、credential rotation/書込みを扱わない。

### 代替案と不採用理由

- pathをfileごとに検査してから再openする方式はTOCTOUとsymlinkすり替えを避けられないため採用しない。
- generic secret manager/filesystem scannerは対象を拡げ、固定4-file境界を曖昧にするため採用しない。
- JWT libraryまたはPEM fixtureの追加は、標準library限定とsecretをrepositoryへ置かない方針に反するため採用しない。
- static installation token、環境変数、実HTTP token交換は、broker secret sourceとJWT生成に限定した範囲外のため採用しない。

### 移行・互換性

- 新規internal packageであり、既存command、config、provision、deployment設定、外部Go moduleは変更しない。
- READMEにはbroker専用directoryの利用境界だけを追加し、実credential値、運用作成・rotate手順、非Linux本番supportは記載しない。
- 実UbuntuのUID/permission隔離、親tree/ACL/mount/MAC、GitHubとのtoken交換は後続の承認済みlive E2Eで確認する。ローカルunit testで代替PASSにしない。

## 変更予定

| パス | 変更内容 |
|---|---|
| `tools/dev-agent-harness/internal/brokercredentials/` | 固定layoutを読む`Load`、platform別の安全なfile読込み、field/PEM validator、非漏洩bundle API、標準library RS256 JWT、hermetic unit testsを追加する。 |
| `tools/dev-agent-harness/README.md` | broker-only credential directory、trusted利用境界、JWTの責務と対象外を簡潔に記録する。 |

## 実装手順

1. packageの最小公開API、固定error、固定basename、bundleの非露出境界を定義する。
2. platform別readerを実装する。Linuxはdirectory FDと`openat`、non-Linuxは開発test用の同一file policyとする。
3. text fieldの末尾LF処理と各形式・長さvalidatorを追加する。
4. 単一PEM RSA private keyのparse、鍵長、`Validate`検査を追加し、bundleへ検証済みの内部鍵だけを組み込む。
5. 時刻取得と標準library署名を局所化し、固定claimの`GitHubAppJWT`を実装する。
6. READMEへ安全な消費境界だけを反映し、既存runtime/configurationに接続しないことを確認する。
7. package unit testを追加・実行し、Linuxとnon-Linuxの責務差、秘密非漏洩、caller入力不変、JWT検証を確認する。

## 検証計画

- packageのrace付きunit testで、正常bundle、file policy拒否、固定basename拘束、nonblocking拒否、metadata変化、text/RSA境界、JWT署名、整数Unix秒claim、同一秒の決定性、固定error、秘密非漏洩、caller入力不変を検出する。
- harnessの`make check`と`make distcheck`、rootの`make check`を実施し、追加・削除合計の対象packageとREADMEが入力パケットの上限内であることを確認する。
- 実UbuntuのUID・permission隔離と実GitHub token交換は環境依存の後続live E2Eとして残し、この検証のPASS根拠に含めない。

## リスクと停止条件

- Linuxでdirectory FD相対の安全なopenまたはFD metadata安定性を実現できず、path再解決や一般化で代替する必要が生じた場合は停止してMainへ報告する。
- non-Linuxのtest実装がLinux policyと固定basenameを保てない、または本番supportを暗黙に約束する形になる場合は停止する。
- 公開API、error、test output、READMEにPEM、private key、OpenAI key、JWT、入力値、OS/parser detailが現れる設計になる場合は停止する。
- 外部module、runtime/configuration変更、HTTP token交換、実credential・root/chown・networkを必要とする検証が必要になった場合は、対象外として停止し再計画する。

## 未解決事項

- なし

## main Agentレビュー

- [x] TASKの全AC-IDへ設計判断、パス、順序、失敗時の扱いを対応させ、条件本文を複製していない。
- [x] Linux openat境界、non-Linux開発test実装、秘密非漏洩、標準library JWT、停止条件が対象範囲内で具体化されている。
- [x] 設計観点と代替案を検討している。
- [x] QA_PLANがTASK-firstで独立作成されている。
- [x] `dependency-ready reconciliation`と完了経路preflightが完了している。
- [x] DEV開始を承認した。

## planning review

PASS — 初回に、同一秒のPKCS#1 v1.5署名は決定的でJWTも同一になる点と、NumericDateのJSON型・秒精度が未指定な点を検出した。TASK/PLAN/QAを、各呼出しでの署名処理、同一秒の同一JWT許容、`now.UTC().Unix()`基準の整数JSON `iat`/`exp`へ修正し、再レビューで整合を確認した。追加のnonce、PSS、random claimは導入しない。

安全契約変更でv2契約を選ぶ場合は、コメントを外して`safety_contract_version: 2`と予定パス・生成パスの配列を記録し、DEV前に`make task-preflight TASK=TASK-0042`を実行する。変更しない種別は空配列とし、通常の予定パスと生成パスを重複させない。独立計画レビューのPASSとMainの分類承認をフロントマターへ記録する。分類変更時はTask、PLAN、QA_PLANを再承認し、承認者と時刻を更新する。
