---
task_id: "TASK-0048"
change_class: "product"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "luna-xhigh"
approved_dev_profile_reason: "既存のPolicy/Registry/Transaction/Forwarderを変更せず、新規in-memory合成とfake resolver/RoundTripperだけのhermetic integration testに閉じるため。"
approved_dev_profile_risk_signals: []
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T10:55:32Z"
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

# TASK-0048 PLAN

## AC対応

TASKの条件本文を再掲せず、`planning input packet`のAC-IDに設計を対応させる。

| AC-ID | 設計判断 | 変更パス | 実施順序 | 失敗時の扱い |
|---|---|---|---|---|
| AC-1 | `Rules`は既存の`*egresspolicy.Policy`、`*capability.Registry`、`egresstransaction.CredentialResolver`、注入`http.RoundTripper`、credential/timeout/response-size上限だけを受ける。`New`は範囲とnilを固定Rules errorで検証し、長寿命`Exchange`へ不変依存だけを保存する。 | `tools/dev-agent-harness/internal/brokerexchange/` | 1 | typed nilを含む依存不足、境界外設定、nil/zero receiver又は破損状態はpanic・format詳細なしの固定`exchange-denied`とzero responseへ畳む。 |
| AC-2 | `Do`ごとに非公開のcapture sink、`upstreamforwarder.Forwarder`、`egresstransaction.Transaction`をlocal値として一つずつ生成し、subject/requestを一回だけ同期`Execute`へ渡す。Exchangeは独自のpolicy/capability/credential/forwarding検証を追加しない。 | `tools/dev-agent-harness/internal/brokerexchange/` | 2 | Forwarder/Transaction生成又はExecute失敗ではsink値を公開せず、default transport/client、redirect、retry、request保持を導入しない。 |
| AC-3 | capture sinkは成功responseを一回だけ受理し、Transaction成功かつ正確に一通知の場合だけ`upstreamforwarder.Response`相当のstatus、空又は正規`application/json`、独立本文を戻す。各callのsink/bufferは互いに共有しない。 | `tools/dev-agent-harness/internal/brokerexchange/` | 3 | sink未通知、二重通知、sink/response不整合、alias又はコピー不能な結果はfixed errorとzero responseにし、次回/並行callのstateを読まない。 |
| AC-4 | Policy/Authorization/capability拒否をTransactionの順序へ委譲し、fakeのresolver/transport counterで到達ゼロを確認する。subject/scope不一致のhandleは後続正当callへ残し、Consume後のresolver/Forwarder失敗は既存Registryのまま消費済みとする。 | `tools/dev-agent-harness/internal/brokerexchange/` | 4 | capabilityを復元・再発行せず、同一Exchange内又はwrapperで再Execute/再RoundTripしない。再使用は既存Transactionのdenyへ委譲する。 |
| AC-5 | constructor/Transaction/Forwarder/sinkの全失敗を`exchange-denied`だけへ正規化し、戻り値はzero responseにする。公開値、error、`Format`、長寿命stateへopaque handle、credential、URL、scope、本文、provider又は依存詳細を置かない。 | `tools/dev-agent-harness/internal/brokerexchange/` | 5 | 任意の追加診断、下位error wrap、response partial公開、消費後retryが必要なら停止してMainへ戻す。 |
| AC-6 | real Policy/Registryとpackage-local fake resolver/RoundTripperを使うhermetic integration suiteを追加する。GitHub/OpenAI成功、認可順序、消費境界、単回実行、input/output ownership、non-leak、並行response隔離をrace detectorで観測する。READMEはin-memory合成とlive E2E未実施境界だけを追記する。 | `tools/dev-agent-harness/internal/brokerexchange/`、`tools/dev-agent-harness/README.md` | 2–6 | fake PASSを実provider、実credential、DNS/TLS/system trust、HTTP listener/Agent response writerの成功根拠にせず、外部network/config/dependency/既存package変更が必要なら停止する。 |

## 関連Wikiと判断

- REF-1の`Transaction.Execute`が policy → strict Authorization → `Registry.Consume` → resolver → credential validation → trusted Forwarder の唯一の順序を所有する。Exchangeはこの順序を再実装せず、call-local Forwarderを渡して同期実行する。
- REF-2の`Forwarder`は同一Policyの再評価、注入RoundTripper一回、完全検証後のsink一回を所有する。ExchangeはResponseSinkを外部注入可能にせず、captureだけを内部に閉じる。
- REF-3の意味Wikiに従い、credential-bearing PreparedRequestはTransactionからForwarderの同期呼出中だけに留める。Exchangeの戻り値へ昇格できるのは縮退済み成功responseだけである。
- REF-4のphase 1 surfaceを維持し、GitHub REST readとnon-streaming OpenAI Responses以外、HTTP/TLS/Agent proxy/approvalは追加しない。production wiringとlive E2Eは後続境界のまま残す。
- WikiはDEV許可パスではない。再利用可能な知識が生じた場合だけ、Mainが既存正本への同化とpost-merge処理を所有する。

## 補足設計

### 責務・境界・不変条件

- `Exchange`の長寿命stateは検証済みのPolicy、Registry、resolver、RoundTripperと数値上限だけであり、subject、request、Authorization、capability、credential、sink、responseは保持しない。`Do`のsink、Forwarder、Transaction、response bufferはすべてcall-localにする。
- 成功順序は Rules/receiver 検証 → private sink/Forwarder/Transaction 構築 → `Transaction.Execute`一回 → sinkの正確な一通知確認 → 独立response copy のみである。Forwarder成功前又はTransaction失敗後にcaptureを返さない。
- sinkは`upstreamforwarder.Response`のstatus/content type/bodyを受ける内部実装とし、Delivery時にbodyをcopyする。Exchangeが返す際にもcopyし、callerの後続変更、transport buffer、sink buffer、別call間のaliasを残さない。
- Policy/Registryの実評価と消費、Forwarderの再評価・RoundTrip・response妥当性検査は既存APIに委譲する。ExchangeはURL/scopeを再解析・補正せず、Authorization/body sliceを変えず保持しない。
- fixed errorはExchange境界の`exchange-denied`一つだけとし、zero responseはstatus 0、空content type、nil/empty bodyに正規化する。下位の`invalid-rules`、`transaction-denied`、`upstream-forward-failed`その他のdetailは外へ伝播しない。

### 代替案と不採用理由

- `Transaction`へresponse戻り値を加える案はcredential handoff境界を変更するため採用しない。Forwarderのrequest単位sinkを内部でcaptureする。
- callerがsink又はForwarderを渡す案はresponse state/通知回数をbroker外へ出すため採用しない。`Do`ごとにExchangeが所有する。
- Exchangeでpolicy/capability/Forwarderの検査を複製する案は既存境界との意味分岐を作るため採用しない。既存constructor/固定errorを使い、合成結果だけをfail closedにする。
- response未通知を成功として空responseにする案、又はresolver/transport失敗を再試行する案は、成功と消費済み副作用を偽装するため採用しない。

### 移行・互換性

- 新規`internal/brokerexchange`とREADME追記だけに限定する。`egresspolicy`、`capability`、`egresstransaction`、`upstreamforwarder`、`upstreamtransport`、provider resolver、CLI/config/dependency/生成物は変更しない。
- candidateの製品差分は追加＋削除合計1,000行以下、3 commit経路（planning、製品差分だけのcandidate、completion）を守る。candidate識別子はHANDOVERだけで管理し、PLANへ転記しない。
- README変更があるため既存`pnpm lint:docs`をcandidate前に実行する。新しいgate/check/processは加えない。実provider/Agent proxyはlive E2E blockedのままで、post-merge環境確認の対象としてのみ扱う。

## 変更予定

| パス | 変更内容 |
|---|---|
| `tools/dev-agent-harness/internal/brokerexchange/` | immutable Exchange Rules/New/Do、private one-delivery capture sink、call-local Forwarder/Transaction合成、fixed errorとzero response、およびreal Policy/Registry + fake resolver/RoundTripperによるhermetic race integration testsを追加する。 |
| `tools/dev-agent-harness/README.md` | broker内in-memory exchangeがTransactionとrequest単位Forwarderを合成して縮退済み成功responseだけを返す範囲、実HTTP入口/production wiring/live E2Eが対象外であることを追記する。 |

## 実装手順

1. 新packageの公開Rules/Exchange/Responseと固定errorを定義し、typed nilを含む依存、credential上限、timeout、response上限、nil/zero receiverのfail-closed testを置く。
2. private capture sinkと`Do`のcall-local構成を実装する。既存`upstreamforwarder.New`へPolicy/transport/sink/timeout/response上限、既存`egresstransaction.New`へPolicy/Registry/resolver/Forwarder/credential上限を渡し、subject/requestを`Execute`一回だけに渡す。
3. 一通知だけの成功昇格、未通知/二重通知/errorのzero response化、input/output copy ownershipと固定non-leakを実装し、transaction/forwarderの内部検査を複製していないことを差分で確認する。
4. real Policy/RegistryでGitHub/OpenAI成功、policy/Authorization/subject/scope拒否、Consume前非消費、Consume後のresolver/transport/sink失敗の消費維持、resolver/transport各一回をfake counterで検証する。
5. 並行`Do`でunique body/statusを返すfake transportを用い、response混線・alias・raceがないことを検証してREADMEを責務境界に合わせて追記する。
6. candidate前に`go test -race ./internal/brokerexchange`、harness `make check`、`make distcheck`、READMEを変更した場合の既存`pnpm lint:docs`、`git diff --check`、許可pathと1,000行上限を確認する。最終working bytesをcandidate launcherへ渡し、root `make check`の成功後に製品差分だけのcandidate commitを一回作る。planning/candidate/completion以外のcommit、digest転記、重複candidate記録、追加gate/processは作らない。

## 検証計画

- `go test -race ./internal/brokerexchange`でconstructor/receiver、both providerの成功と実Bearer置換、status/content type/bodyの縮退、caller requestのAuthorization/body不変、output copy、fixed non-leakを検出する。
- real Policy/Registry、fake resolver/RoundTripper/sink countersで、policy・Authorization・capability拒否がresolver/transportに到達しないこと、subject/scope mismatchが非消費で正当subject/scopeが後続に成功することを確認する。
- Consume後のresolver error、transport error、Forwarder/sink異常ではzero response、fixed error、capability消費維持、同じhandleの再試行がresolver/transportへ再到達しないことを確認する。RoundTripと成功sink通知は一request一回に限る。
- parallel requestごとに異なる成功本文とstatusを返すfakeを使い、同時`Do`でresponse stateが混線せずrace detectorがdata raceを報告しないことを確認する。実network、実credential、DNS/TLS、listenerは使用しない。
- candidate前の順序はpackage race test、harness `make check`、`make distcheck`、README変更時の`pnpm lint:docs`、diff検査/行数上限とする。candidate固定後はroot `make check`を一回だけ実行し、実provider/Agent proxyのlive E2EはblockedのままQA_PLANとpost-merge確認へ残す。

## リスクと停止条件

| リスク | 抑制/検出 | 停止条件 |
|---|---|---|
| Exchangeが既存Transaction/Forwarderの順序又は検査を複製して意味が分岐する | local compositionだけに制限し、real Policy/Registryとfake countersで依存到達順序をintegration testする | URL/scope再解析、Authorization抽出、Consume、credential/response検証をExchangeへ追加する必要があれば停止する。 |
| sink未通知・二重通知・並行state共有で成功responseを誤って公開する | private call-local sink、exactly-one delivery、二重copy、parallel unique-response race test | async sink、response state共有、未通知成功又は二重通知の受理が必要なら停止する。 |
| Consume後failureでcapability再利用又は隠れた上流retryが生じる | existing Registryを唯一の消費点に保ち、失敗後replay/counter testを置く | rollback、再発行、同一Do内retry又は複数RoundTripが必要になれば停止する。 |
| fake証跡を実環境の成功に過大化、又は許可外の統合を導入する | READMEにlive E2E境界を明記し、許可2パス/1,000行/既存checkを監査する | external dependency、config/CLI、production resolver/transport、HTTP listener、実network/credentialが必要なら停止する。 |

## 未解決事項

- なし。

## main Agentレビュー

- [x] TASKの全AC-IDへ設計判断、パス、順序、失敗時の扱いを対応させ、条件本文を複製していない。
- [x] REF-1〜4と既存APIの責務を保ち、call-local sink → Forwarder → Transaction合成、zero response、消費境界、並行隔離を具体化している。
- [x] 設計観点と代替案を検討している。
- [x] QA_PLANがTASK-firstで独立作成されている。
- [x] `dependency-ready reconciliation`と完了経路preflightが完了している。
- [x] DEV開始を承認した。

安全契約変更でv2契約を選ぶ場合は、コメントを外して`safety_contract_version: 2`と予定パス・生成パスの配列を記録し、DEV前に`make task-preflight TASK=TASK-0048`を実行する。変更しない種別は空配列とし、通常の予定パスと生成パスを重複させない。独立計画レビューのPASSとMainの分類承認をフロントマターへ記録する。分類変更時はTask、PLAN、QA_PLANを再承認し、承認者と時刻を更新する。
