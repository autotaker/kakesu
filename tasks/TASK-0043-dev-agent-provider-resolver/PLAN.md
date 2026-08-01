---
task_id: "TASK-0043"
change_class: "product"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "luna-xhigh"
approved_dev_profile_reason: "新規の固定2-path packageに、既存の検証済みBundleとtransaction interfaceを接続する標準libraryのみの実装・hermetic fake transport testを追加するため。実network、transport実装、既存package変更は含めない。"
approved_dev_profile_risk_signals: []
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T07:37:54Z"
planning_reviewed_by: "reviewer-agent-terra-medium"
planning_review_decision: "pass"
planning_reviewed_at: "2026-08-01T07:37:54Z"
classification_approved_by: ""
classification_approved_at: ""
classification_approval_reason: ""
# safety_contract_version: 2
# safety_contract_planned_paths: []
# safety_contract_generated_paths: []
---

# TASK-0043 PLAN

## AC対応

TASKの条件本文を再掲せず、`planning input packet`のAC-IDに設計を対応させる。

| AC-ID | 設計判断 | 変更パス | 実施順序 | 失敗時の扱い |
|---|---|---|---|---|
| AC-1 | `Rules`（検証済み`*brokercredentials.Bundle`、注入`http.RoundTripper`、timeout）と`New`を設ける。1ms–30秒を閉区間で検査し、成功時だけ不変resolverを返す。`Resolve`を持つ型だけを公開し、compile-time interface assertionをtest又はpackage内に置く。 | `tools/dev-agent-harness/internal/providercredentials/` | 1 | nil bundle/transport、範囲外timeoutを固定invalid-rules error、nil/zero receiver、未知provider、scope不一致を値なし固定resolve errorへ畳む。 |
| AC-2 | OpenAIはproviderが`openai`かつ空repositoryだけを受理し、bundle accessorのkeyを直接同期返却する。JWT/transportには分岐前に触れない。 | `tools/dev-agent-harness/internal/providercredentials/` | 2 | repository付きOpenAIと未知providerはnetwork/JWTなしで固定resolve errorにする。 |
| AC-3 | GitHub scopeをTASK-0039と同じ小文字英数字始まり・`._-`許可のcanonical `owner/name`として局所検証し、nameだけを固定一要素JSON bodyへ入れる。installation IDは10進path segment、JWTはAuthorizationだけに束縛する。固定のmethod/HTTPS host/path/headersを組立て、`RoundTrip`を一度だけ直接呼ぶ。 | `tools/dev-agent-harness/internal/providercredentials/` | 3 | request構築/JWT/transport/timeout/3xxを含む非201はretry、redirect、detail転送をせず固定resolve errorにする。default client/transportを作らない。 |
| AC-4 | response bodyを常にdefer closeし、201と`application/json` media typeを確認する。128 KiB上限付き読込み後、単一object・trailing dataなし・top-level field一意性をtoken/expiryを含む全fieldで検査し、未知fieldはskipする。tokenはvisible ASCII byte数で検査する。expiry評価直前にUTC時刻を一度だけ取得し、`expiresAt.After(now) && !expiresAt.After(now.Add(65*time.Minute))`で判定する。 | `tools/dev-agent-harness/internal/providercredentials/` | 4 | nil body、close以外の全response/JSON/size/type/expiry異常を固定resolve errorにし、部分tokenを返さない。close errorも秘密/詳細を返さない。 |
| AC-5 | resolverはbundleと注入transport以外の状態、cache、refresh、singleflightを持たず、GitHub resolveごとに一度だけJWT生成とexchangeを実施する。公開errorと`Format`相当の出力面に秘密・request/response detailを渡さない。READMEはbroker内のtrusted resolver境界と非対象だけを追記する。 | `tools/dev-agent-harness/internal/providercredentials/`、`tools/dev-agent-harness/README.md` | 1、5 | log、environment/file/process、永続化、redirect/retry、transport policyまたはcredentialがAgentへ渡るAPIを必要とする場合は停止して再計画する。 |
| AC-6 | runtime生成RSA credential fixtureとbody-closeを観測するfake `RoundTripper`で、分岐、request束縛、全response境界、timeout、call数、固定error/non-leakをpackage unit testに固定する。同package testから実resolverを`egresstransaction`へ渡し、capability未到達時と有効GitHub/OpenAI時のForwarder境界を検証する。 | `tools/dev-agent-harness/internal/providercredentials/` | 2–6 | fake unit evidenceを実GitHub/TLS/DNS/proxyのPASSにしない。live E2Eはblockedとして残し、必要な既存package変更は停止・Main報告する。 |

## 関連Wikiと判断

- REF-2の順序を保つ。resolverはcapabilityが消費された後にprovider/repositoryだけを受け、成功credentialは`egresstransaction`がtrusted Forwarderへ渡す以外に公開しない。
- REF-3のbundle accessorと`GitHubAppJWT`だけを利用する。secret directory、PEM、installation IDの再parseやsecret sourceの追加はしない。
- REF-4/5に固定されたGitHub endpoint/API versionをrequest定数に閉じ、repository scopeはbodyのname一件で指定する。permissionをbodyへ追加しない。
- 意味Wikiに既存判断と異なる再利用可能な判断が実装中に生じた場合だけ、別途明示依頼により更新する。本Taskの通常完了にWiki更新は含めない。

## 補足設計

### 責務・境界・不変条件

- `providercredentials`は、検証済みbundleからOpenAI key又はGitHub installation tokenを同期的に解決する唯一の責務を持つ。HTTP接続の安全性、forwarding、server、authorization置換、credential file読込み、logging、永続状態は持たない。
- GitHub repository validatorはTASK-0039のcanonical grammarと同値に限定し、`owner/name`以外、空segment、大文字、Unicode、percent encoding、余分なslashを受理しない。ownerはrequest bodyへ複製せず、repository nameのみを一件送る。
- timeoutは`context.WithTimeout`で生成したrequest contextへだけ適用する。注入RoundTripperを`http.Client`で包まないためredirect policyを導入せず、呼出しごとの`RoundTrip`回数は一回である。
- exported `New`はproduction clockとして`time.Now`を閉じ込め、package-private constructorだけがtest用clock関数を受け取る。resolverはexpiry評価直前にそのclockを一回だけ呼びUTC化し、同じ値から下限と65分上限を判定する。clockは不変dependencyであり、production API、token cache、永続状態を増やさない。
- response decoderはmapへのunmarshal（重複keyを見失う）を使わず、top-level object token列を一回走査してfield名の重複を拒否する。未知fieldの値はJSON値として消費して互換性を保ち、必須fieldは一回だけ、期待型だけを受理する。
- token/JWT/OpenAI key、repository、URL、request/response body、parser/transport errorは固定公開error、format、test diagnostic以外の実装出力へ流さない。testの秘密値もruntime fixture限定にする。

### 代替案と不採用理由

- `http.Client.Do`はredirect等の追加動作を持つため、単発交換を表せる注入`RoundTripper`の直接呼出しを採用する。
- default `http.Transport`、TLS/DNS/proxy/IP validationをpackageに持つとtrusted後続transport境界を壊すため採用しない。
- installation-wide token、permission field、static token、cache/refresh/singleflight/retryはrepository最小権限と一request一exchangeの契約を拡張するため採用しない。
- 固定token prefix/40 byte判定はGitHubの新形式を不必要に拒否するため採用しない。`encoding/json`の通常unmarshalだけは重複top-level fieldを検出できないため採用しない。

### 移行・互換性

- 新規internal packageとREADME追記だけで、TASK-0039〜0042、command/configure/Makefile、外部Go module、runtime/configurationを変更しない。
- 既存`egresstransaction` interfaceの利用側としてintegration testを置くが、同packageのAPI/挙動は変更しない。
- 実GitHub受理、実TLS/DNS/proxy、実installation権限・repository install、credential rotationは後続の承認済みlive E2E/Taskへ残し、fake RoundTripperのPASSで代替しない。

## 変更予定

| パス | 変更内容 |
|---|---|
| `tools/dev-agent-harness/internal/providercredentials/` | `CredentialResolver`実装、固定error/rules、canonical repository/request/response validator、上限付きbody処理、runtime credential fixtureとhermetic unit/integration testsを追加する。 |
| `tools/dev-agent-harness/README.md` | broker内resolverのOpenAI no-network・GitHub単発repo限定exchange、Forwarder/transport/live E2Eとの責務分離を簡潔に記録する。 |

## 実装手順

1. packageの最小公開Rules/constructor/resolver API、固定error、nil/zero fail-closed規則を定義し、既存`CredentialResolver`への適合を固定する。
2. OpenAI分岐とcanonical GitHub repository validatorを実装し、OpenAIがJWT/transport未到達であることと拒否分岐をtestで先に固定する。
3. GitHubのJWT取得、固定request組立て、timeout context、直接一回`RoundTrip`を追加し、fake transportでmethod/URL/header/body/call count/context deadlineを検証する。
4. closeを含むresponse境界、strict top-level JSON parser、単一UTC評価時刻と非公開test clock seamを実装し、status/media type/size/duplicate/trailing/type/token/expiryの全異常を拒否するtestを加える。
5. READMEへtrusted利用境界と対象外を追加し、default transport・retry/cache・secret diagnostic面がないことを確認する。
6. 同packageのintegration testで実resolverをTASK-0041 transactionへ注入し、invalid capabilityはtransport未到達、valid OpenAI/GitHubだけがcredentialをtrusted Forwarderへ渡すことを確認する。
7. format、race unit、harness/root check、distcheck、差分SLOC上限とscopeを確認し、実networkを使わない証跡をcandidateへ結ぶ。

## 検証計画

- `go test -race ./internal/providercredentials`でconstructor/nil receiver、OpenAI no-network、provider/repository拒否、GitHub request全束縛、deadline、JWT一回・RoundTrip一回、transport/context/status failureのfixed errorを検出する。
- fake response bodyと固定package-private clockで、成功・nil body・全失敗のclose、201以外、content-type parameter/非JSON、128 KiB境界、single object、trailing data、重複top-level field、missing/wrong-type token/expiry、visible ASCII 1/4096境界、expiryのnow拒否・now直後許可・65分許可・65分超拒否、未知field許容、秘密非漏洩をdeterministicに検出する。
- 実resolverを用いた`egresstransaction`接続testで、無効capabilityにはresolver/transport/Forwarderが到達せず、有効OpenAI/GitHubだけが適切なcredentialをtrusted Forwarderへ一回渡すことを確認する。
- `tools/dev-agent-harness`の`make check`、`make distcheck`、root `make check`、`git diff --check`をcandidateで実行する。base...candidateの許可packageとREADMEの追加・削除合計が1,200行以下であることを記録する。
- 実GitHub network、TLS/DNS/proxy transport、実installation permissionは`live-e2e` blockedであり、このTaskのhermetic PASSに含めない。

## リスクと停止条件

- canonical `owner/name`検査を既存TASK-0039と同値に保てず、egresspolicy APIの変更又は別のrepository normalizationが必要になる場合は停止してMainへ報告する。
- top-level JSON重複検出、128 KiB全体上限、trailing値拒否を同時に保証できず、緩いmap/unmarshal又は未上限readerで代替する必要が生じる場合は停止する。
- default transport、redirect/retry/cache、実network、外部module、config/runtime変更、credential source再解釈、実secret fixtureを必要とする場合は対象外として停止・再計画する。
- error、format、README、test failure出力に実Credential/JWT/token、repository、URL、body、transport/parser detailを出さずに検証不能な場合は停止する。
- fake transportの結果をTLS/DNS/proxy/実GitHub受理の証拠と扱う必要が生じた場合は、live-e2eが未解決のまま停止する。

## 未解決事項

- なし

## main Agentレビュー

- [x] TASKの全AC-IDへ設計判断、パス、順序、失敗時の扱いを対応させ、条件本文を複製していない。
- [x] repo scope、単発RoundTrip、strict response boundary、秘密非漏洩、既存transaction境界、停止条件が対象範囲内で具体化されている。
- [x] 設計観点と代替案を検討している。
- [x] QA_PLANがTASK-firstで独立作成されている。
- [x] `dependency-ready reconciliation`と完了経路preflightが完了している。
- [x] DEV開始を承認した。

## planning review

PASS — 初回にexpiryの現在/65分境界へ単一評価時刻とtest clock seamがなく、境界testがflakyになるMedium findingを検出した。exported `New`はproduction clockを閉じ込め、package-private constructorだけが固定test clockを受け、expiry直前に一度だけUTC nowを取得する設計へ修正した。QA_PLAN revision 2もnow/直後/65分/超過をdeterministicに検出する形へ更新し、再レビューで要件・既存API・許可scopeとの整合を確認した。

安全契約変更でv2契約を選ぶ場合は、コメントを外して`safety_contract_version: 2`と予定パス・生成パスの配列を記録し、DEV前に`make task-preflight TASK=TASK-0043`を実行する。変更しない種別は空配列とし、通常の予定パスと生成パスを重複させない。独立計画レビューのPASSとMainの分類承認をフロントマターへ記録する。分類変更時はTask、PLAN、QA_PLANを再承認し、承認者と時刻を更新する。
