---
task_id: "TASK-0047"
change_class: "product"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "luna-xhigh"
approved_dev_profile_reason: "既存のPolicy、PreparedRequest契約、注入RoundTripperを変更せず、新規internal package内の同期変換とhermetic fake transport/body/sink試験に閉じるため。"
approved_dev_profile_risk_signals: []
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T10:17:51Z"
planning_reviewed_by: ""
planning_review_decision: "pending"
planning_reviewed_at: ""
classification_approved_by: ""
classification_approved_at: ""
classification_approval_reason: ""
# safety_contract_version: 2
# safety_contract_planned_paths: []
# safety_contract_generated_paths: []
---

# TASK-0047 PLAN

## AC対応

TASKの条件本文を再掲せず、`planning input packet`のAC-IDに設計を対応させる。

| AC-ID | 設計判断 | 変更パス | 実施順序 | 失敗時の扱い |
|---|---|---|---|---|
| AC-1 | `Rules`で同一`*egresspolicy.Policy`、注入`http.RoundTripper`、request単位`ResponseSink`、timeout、response byte上限を受ける`New`と、固定comparable errorを定義する。concrete Forwarder は`egresstransaction.Forwarder`適合をコンパイル時に確認し、依存は生成後不変にする。 | `tools/dev-agent-harness/internal/upstreamforwarder/` | 1 | nil依存、timeout/size境界外、nil/zero receiver、破損内部状態は panic せず同じ固定errorへ畳む。Rules入力・内部詳細をerror又はformat面へ出さない。 |
| AC-2 | `Forward`の最初に`PreparedRequest`を`Policy.Evaluate`へ再投入し、allow decisionと得られた全scope fieldの完全一致を要求する。authorizationは固定Bearer prefixを除く visible ASCII credentialだけを長さ検査し、bodyはcaller所有領域から独立copyする。 | `tools/dev-agent-harness/internal/upstreamforwarder/` | 2 | policy/scope/credential/inputの拒否ではtransportとsinkを呼ばず、入力sliceを変更・保持せず固定errorだけを返す。再評価と保持scopeが一致しない値を補正・正規化しない。 |
| AC-3 | requestごとのtimeout contextと独立bodyから一つだけ上流`http.Request`を作る。headerはprovider種別で決まる最小allowlistを新しいrequestへ明示設定し、注入RoundTripperの`RoundTrip`を一回だけ同期呼出する。 | `tools/dev-agent-harness/internal/upstreamforwarder/` | 3 | request生成、context期限/cancel、RoundTrip error、nil/併存responseでは成功を公開せず、取得済みbodyをcloseして固定errorにする。client/default transport/proxy、redirect、retryを導入しない。 |
| AC-4 | transport responseはForwarderが所有して必ずcloseする。2xx以外を先に拒否し、HEAD/204は本文が空の場合だけ、その他の空bodyも空成功として縮退する。本文ありは`max+1`読取で上限を検出し、UTF-8、JSON構文、JSON media typeを全て検証してから受理する。 | `tools/dev-agent-harness/internal/upstreamforwarder/` | 4 | status、HEAD/204の非empty body、content type、read/close、context又はresponse+errorの異常ではsinkを呼ばない。partial本文、provider error本文、header又は下位errorを返却・記録しない。close errorも成功へ昇格させない。 |
| AC-5 | sinkへ渡す値型はstatus、空又は正規`application/json` content type、独立copy本文だけに限定し、検証・close完了後の成功で一回だけ同期通知する。Forwarderはresponse、credential又はrequest由来の可変値を状態に保持しない。 | `tools/dev-agent-harness/internal/upstreamforwarder/` | 5 | sink errorは固定errorへ写像し、再通知・再送しない。transport/caller/sink間のalias、upstream header、URL、scope、Authorization、response/error detailが必要になる設計は停止してMainへ戻す。 |
| AC-6 | package-private fake RoundTripper、計数/失敗可能body、sinkを使うrace対応unit suiteで、両providerの成功、拒否順序、header最小化、timeout/cancel、応答検証、close/ownership/non-leakを検出する。READMEはbrokerの合成責務とlive E2Eの限界のみ追記する。 | `tools/dev-agent-harness/internal/upstreamforwarder/`、`tools/dev-agent-harness/README.md` | 2–6 | fakeのPASSを実provider、実DNS/TLS/system trust、認証情報又はAgent向けwriterの根拠にしない。許可外path、dependency、config、外部network、listener又は追加processが必要なら停止する。 |

## 関連Wikiと判断

- REF-1の`Transaction.Execute`は、policy allow、capability消費、credential解決の後にcredential-bearing `PreparedRequest`を同期Forwarderへ一度だけ渡す。Forwarderはその既存interfaceを実装するだけで、Transactionの戻り値や消費順序を変更しない。
- REF-2の`upstreamtransport.Transport`は注入する`http.RoundTripper`のまま用いる。origin/DNS/IP/TLS/HTTP1 と一接続境界を再実装せず、Forwarderはcaller制御のtransportや`http.DefaultTransport`を選ばない。
- REF-3の責務分離に従い、Forwarderは実credentialを同期呼出外へ保持・公開せず、上流の成功応答を最小のsink値へ縮退する。上流response、proxy、Agent response writer、auditは別境界である。
- REF-4のphase 1 surfaceを維持する。GitHub REST readと非streaming OpenAI response以外のprovider機能、Git Smart HTTP、streaming、redirect/retry、任意headerの許可は追加しない。
- WikiはDEV許可パスではない。新たな再利用可能知識が実際に生じた場合だけ、Mainが既存main正本への同化とpost-merge処理を所有する。

## 補足設計

### 責務・境界・不変条件

- `upstreamforwarder`は、`Policy`が出す正規scopeを唯一の送信許可根拠にする。`PreparedRequest.Scope`を自力で復元せず、同じ評価のscopeとの完全一致を確認することで、直接呼出でもcredentialの送信先を限定する。
- 成功経路は、Rules/receiver検証 → PreparedRequest再評価とBearer検証 → 独立上流request作成 → timeout内の単回RoundTrip → response検証・copy・close → 単回sink の順序に固定する。各requestのcontext、request、response bufferはローカルであり、長寿命Forwarderに共有mutable stateを置かない。
- 上流requestには実`Authorization`、固定`Accept`、固定`User-Agent`、OpenAIだけのJSON `Content-Type`以外を移さない。特にAgent由来header、Host、cookie、provider error/headerを転記しない。
- response本文はsinkの前に全量をbounded bufferへ独立copyして検証し、成功後にも別copyをsinkへ渡す。空成功も同じ最小response表現に正規化し、sinkはrequest単位で後続gatewayが合成する。
- exported errorと必要なformat出力は固定labelだけとし、URL、scope、Authorization、本文、header、下位errorをwrap、ログ、保存又は診断へ流さない。

### 代替案と不採用理由

- Transactionにresponse戻り値を追加する案は、既存のcredential handoff interfaceと責務を広げるため採用しない。request単位sinkがgateway側のresponse writer合成を待てる。
- `http.Client`又はdefault transportを作る案は、redirect、environment proxy又はtransport選択をForwarder境界へ持ち込むため採用しない。注入RoundTripperの直接単回呼出を使う。
- response headerや非2xx本文を透過する案は、未観測の情報面をAgentへ追加するため採用しない。必要な最小allowlistは実統合観測後の別Taskで判断する。
- streaming、compression展開、incremental sink、retry、cacheを加える案は、1 MiB bounded copyと「完全検証後に一回通知」の失敗原子性を崩すため採用しない。

### 移行・互換性

- 新規internal packageとREADME追記だけで、`egresspolicy`、`egresstransaction`、`upstreamtransport`、provider resolver、CLI/config/dependency/生成物を変更しない。base...candidateの追加・削除合計は1,000行以下に保つ。
- live E2Eは実provider、Internet DNS/TLS、system trust、認証情報、Agent listener/response writerと安全なcleanupが対象外のためblockedのままとし、fake transport testのPASSで代替しない。

## 変更予定

| パス | 変更内容 |
|---|---|
| `tools/dev-agent-harness/internal/upstreamforwarder/` | production constructor、Forwarder/response sink値型、policy/scope/credential再検証、最小上流request、bounded response縮退、fixed error、fake transport/body/sinkによるhermetic race testsを追加する。 |
| `tools/dev-agent-harness/README.md` | brokerのForwarderが既存transactionと注入upstream transportを接続する範囲、縮退する成功response、live E2Eへ残る境界を簡潔に追記する。 |

## 実装手順

1. 新packageの固定error、Rules、response sink、constructor、interface適合を定義し、nil/zeroとRules境界を先にtestする。
2. 同一PolicyでのPreparedRequest再評価、scope完全一致、Bearer credential検査、caller bodyの所有権分離を実装し、拒否時のtransport/sinkゼロ到達をtestする。
3. provider別header allowlist、timeout context、単回注入RoundTripとresponse/error所有権を実装する。default transport、client、retry又はredirectを導入しない。
4. status・HEAD/204/空body、content type、bounded read、UTF-8/JSON、closeを順にfail-closed化し、正規responseの独立copyだけをsinkへ一回渡す。
5. fake transport/body/sinkで成功・異常・race/alias/non-leak試験を完成させ、READMEへ責務と非対象を追記する。
6. candidateの製品差分だけを一回固定し、candidate launcherでroot `make check`を一回だけ実行する。package race、harness `make check`/`make distcheck`、`git diff --check`、許可pathと1,000行上限はcandidate-bound証跡で確認する。planning/candidate/completionの3 commit以外のcommit、digest転記、重複candidate記録、追加processは作らない。

## 検証計画

- `go test -race ./internal/upstreamforwarder`で、Rules/receiver、両providerの再評価・scope不一致・credential境界、input不変、許可header、timeout/cancel、RoundTrip/sink各一回、response+error、2xx/HEAD/204/空本文、JSON media type、size/UTF-8/JSON/status/read/close/sink失敗、全body close、copy ownership、fixed non-leak errorを検出する。
- test fixtureはfake RoundTripper、reader/closer、sinkだけを使い、外部network、実DNS/TLS、実認証情報、listener、環境proxyを使わない。並行`Forward`ではraceがなく、各試行の通知/transport回数が混線しないことを確認する。
- candidateでharness `make check`、`make distcheck`、candidate launcherのroot `make check`（一回）、`git diff --check`、base...candidateのpathと追加＋削除合計を検査する。実provider受理とAgent response writerは`live-e2e` blockedとしてQA_PLANへ残し、focused-rerunのPASSで代替しない。

## リスクと停止条件

| リスク | 抑制/検出 | 停止条件 |
|---|---|---|
| policy再評価とPreparedRequest scopeの解釈差で別originへcredentialを送る | 同一`Policy.Evaluate`のscope全field完全一致と、transport前拒否のfake test | scopeの部分一致、独自URL解析、scope補正が必要になれば停止してMainへ戻す。 |
| responseのpartial/alias/header/error detailがAgent側へ漏れる | bounded read、完全検証・close後の最小sink値、copy/close/non-leak test | 任意header、非2xx本文、URL、credential又は下位errorをsink/errorへ出す必要が生じれば停止する。 |
| timeout/cancel/response+errorで通知又はbody所有権が二重化する | request-local context、単回RoundTrip/sink、failure body close、fake counter/race test | retry、redirect、非同期通知、close不能なfailure responseの黙認が必要なら停止する。 |
| scope外の統合変更又はfake証跡の過大主張 | 許可2パスとcandidate diff上限を監査し、live E2Eを明示的にblockedに保つ | external dependency、config/CLI、network/OS試験、proxy listener、既存package/Wiki変更が必要なら停止する。 |

## 未解決事項

- なし。

## main Agentレビュー

- [x] TASKの全AC-IDへ設計判断、パス、順序、失敗時の扱いを対応させ、条件本文を複製していない。
- [x] REF-1/REF-2の既存interface・注入transport境界を広げず、policy再評価、scope完全一致、response縮退の順序を具体化している。
- [x] 設計観点と代替案を検討している。
- [x] QA_PLANがTASK-firstで独立作成されている。
- [x] `dependency-ready reconciliation`と完了経路preflightが完了している。
- [x] DEV開始を承認した。

安全契約変更でv2契約を選ぶ場合は、コメントを外して`safety_contract_version: 2`と予定パス・生成パスの配列を記録し、DEV前に`make task-preflight TASK=TASK-0047`を実行する。変更しない種別は空配列とし、通常の予定パスと生成パスを重複させない。独立計画レビューのPASSとMainの分類承認をフロントマターへ記録する。分類変更時はTask、PLAN、QA_PLANを再承認し、承認者と時刻を更新する。
