---
task_id: "TASK-0040"
change_class: "product"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "luna-xhigh"
approved_dev_profile_reason: "in-memoryの純粋なscope registry、標準libraryのentropy/clock、mutexによる局所的な状態遷移に限定し、Credential、proxy、network、永続化、外部依存を含まないため"
approved_dev_profile_risk_signals: []
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T05:24:39Z"
planning_reviewed_by: "reviewer-agent-terra-medium"
planning_review_decision: "pass"
planning_reviewed_at: "2026-08-01T05:24:10Z"
classification_approved_by: ""
classification_approved_at: ""
classification_approval_reason: ""
---

# TASK-0040 PLAN

## AC対応

TASKの条件本文を再掲せず、`planning input packet`のAC-IDに設計を対応させる。

| AC-ID | 設計判断 | 変更パス | 実施順序 | 失敗時の扱い |
|---|---|---|---|---|
| AC-1 | `Rules`、opaqueな`Registry`、固定安全errorを定義し、`New(Rules)` はpolicy version、TTL、uses、initial epochを検証してproduction entropyとUTC clockを注入する。 | `internal/capability/capability.go`、`internal/capability/capability_test.go` | 1 | invalid Rulesとnil/zero Registryはallow又は部分初期化にせず、入力を含まない固定errorを返す。 |
| AC-2 | `IssueSpec`をagent instance、non-root UID、workspace、provider固有scope、TTL、usesの値型とし、`Issue`が検証済みscopeをentryへ格納する。handleはrandom bytesから発行し、entry keyはdigestだけにする。 | `internal/capability/capability.go`、`internal/capability/capability_test.go` | 2 | invalid spec、entropy read失敗、retry上限内に解消しないcollisionは固定issue errorに集約し、entryを残さない。 |
| AC-3 | `Request`をhandleと完全一致させるscope値、`Grant`をCredentialなしの検証済みscope値として定義する。`Consume`は二つのprovider surfaceだけを完全一致で比較する。 | `internal/capability/capability.go`、`internal/capability/capability_test.go` | 3 | malformed/unknown handleとscope不一致は同一denyへ集約し、不一致時のremaining usesを変更しない。 |
| AC-4 | Registry mutexの臨界区間内でepoch、expiry、remaining uses、scopeを判定し、成功時だけ一回減算する。最後の成功・expiry・revoke・epoch advanceはentryを失効又は削除する。 | `internal/capability/capability.go`、`internal/capability/capability_test.go` | 3 | stale/same epoch、unknown revoke、期限境界、再利用、同一handleの並行消費はfail-closedとし、成功は一回だけにする。 |
| AC-5 | registryはdigest keyed entryと固定scope/stateだけを持ち、raw handle、Credential、caller-owned sliceを保持しない。READMEにin-memory restart失効と実proxy/Credential非対象を記す。 | `internal/capability/capability.go`、`internal/capability/capability_test.go`、`README.md` | 4 | error、Grant、保存state、READMEのいずれからも入力・Credential・entropy/clock detailを露出せず、I/Oや永続化を導入しない。 |
| AC-6 | table-driven unit testsをRules、Issue、Consume、revoke、epoch、expiry、collision、entropy failure、non-leak、不変性、並行性へ割り当て、harnessの既存検査でpackage integrationを確認する。 | `internal/capability/capability_test.go`、`README.md` | 4 | race、許可外path、外部依存、副作用、scope逸脱、未検出のnegative caseがあれば候補を完了扱いにしない。 |

## 補足設計

### API、型、state

- packageは `Rules`、`IssueSpec`、`Request`、`Grant`、`Registry` と `New(Rules)`、`Issue(IssueSpec)`、`Consume(Request)`、`Revoke(handle)`、`AdvanceRevocationEpoch(epoch)` を公開する。Registry内部のentry、mutex、entropy reader、clockは公開しない。
- `Rules` はpolicy version、最大TTL、最大uses、initial revocation epochを持つ。policy versionは1〜64 byteで、agent instance/workspace IDは1〜128 byteで、いずれも先頭英数字かつ英数字・`.`・`_`・`-`だけからなるASCII identifierに限定する。TTLは正かつ24時間以下、usesは正かつ10,000以下、UIDは0を受理しない。
- `IssueSpec` のproviderは `github` と `openai` の二つだけである。GitHubはcanonical lowercase `owner/repo`、`github-rest-read`、`api.github.com`に固定しrepositoryを必須とする。OpenAIはrepositoryなし、`openai-responses-text`、`api.openai.com`に固定する。operation又はhostをcallerが任意指定する型にはしない。
- `Request` はhandle、subject、UID、workspace、provider、repository、operation、destination hostを持つ。`Consume`はrequestを正規化せず、entryの発行時scopeと全fieldを完全一致させる。`Grant` は成功したentryの固定scope、issued/expires、remaining uses、policy version、revocation epochを返すが、handle又は実Credentialを含まない。
- Registryはmutexで保護した`SHA-256(handle)` keyed mapとcurrent revocation epochだけを状態にする。entryにはdigest、固定scope、issued/expires、remaining uses、policy version、発行時epochだけを置く。handle文字列、entropy bytes、Requestの参照をstateへ保存しない。

### Issue、digest、entropy

- `Issue` はspecをRulesとprovider固定surfaceへ照合してから、32 random bytesを読み、paddingなしbase64urlで符号化した値へ`cap_`を一度だけ付ける。map keyはその完全handleから得たSHA-256 digestだけである。
- production constructorは`crypto/rand.Reader`と`time.Now().UTC`を使う。package-private constructor又はdependency値だけがdeterministic entropy/clockを受けられ、exported API、README、他packageからテストhookとして到達できない。
- digest collisionはmutex下で判定して初めてentryを挿入する。entropy read後のcollisionだけを最大4回まで再試行し、最後まで解消しなければ固定issue errorで終了する。entropy failure又はcollision failureではentryを挿入せず、partial stateを残さない。

### Consume、revoke、epochの原子性

- `Consume` はhandleの形式とdigestを確認した後、mutex下でentry lookup、current epoch照合、`now >= expires`、remaining uses、scope完全一致を一つのtransactionとして扱う。scope mismatchはentryを残したままdenyにし、usesを消費しない。
- 成功時だけremaining usesを一回減らし、最後のuseなら同じ臨界区間でentryを削除してからGrantを構築する。expiry、revoke、epoch mismatchのentryも同じ臨界区間で削除又は無効化するため、失敗後の再利用を許さない。
- `Revoke` はdigestで既存entryだけを削除する。`AdvanceRevocationEpoch` はcurrent epochより大きい値だけを受理し、更新と既存entryの全失効をmutex下で行う。古い又は同値epoch、unknown/malformed handle、nil/zero Registryは固定安全errorで拒否する。
- mutexはIssue insertion、Consume判定/消費、Revoke、epoch updateのlinearization pointである。同じ1-use handleへの競合Consumeは一方だけがGrantを返し、他方はdenyとなる。

### 代替案と不採用理由

- JWT又はself-contained handle: scopeと失効状態をhandleへ持たせ、漏えい時の即時revokeとdigest-only stateを損なうため不採用。
- raw handleをmap key又は永続記録に使う: memory dump、log、将来のstorageから露出する面を増やすため不採用。digestだけをlookup keyにする。
- provider、operation、hostをgenericな自由文字列として発行: TASK-0039の二つのallow surfaceより広い権限を作るため不採用。provider別固定scopeだけを構築する。
- scope mismatchでもuseを消費する: typo/攻撃の試行で正当なhandleを失効させ、ACのatomic scope契約に反するため不採用。
- background cleanup又はTTL ticker: clock/process lifecycleと状態遷移を増やすため不採用。Consume時のexpiry削除とepoch全失効だけを使う。
- file/database、proxy、Credential brokerを接続する: restart、network、実Credentialの検証を混在させるため不採用。RegistryのGrantは実通信許可を意味しない。

### 責務・境界・不変条件

- `internal/capability` はin-memory reference、scope binding、uses、expiry、revoke、epochだけを所有する。実Credential、header/body、URL解析、TLS/DNS/socket、logging/audit、rate/cost、config、永続化は所有しない。
- 全ての拒否経路は固定errorだけを返す。handle、agent/workspace、repository、provider、entropy error、clock error又はmap内部状態をエラー文字列へ透過しない。
- Requestを変更又は保存せず、Grantはentryからコピーして返す。Rulesやcaller-owned memoryを内部stateの別名として持たない。production codeはfile/environment/process/network/DNS/TLSを使わない。
- process restartでentry mapが失われることは、全既存handleが使えなくなるfail-safe性質である。recovery、共有registry、再発行/復元、実transportは後続の別Taskで扱う。

## 変更予定

| パス | 種別 | 変更内容 |
|---|---|---|
| `tools/dev-agent-harness/internal/capability/capability.go` | implementation | in-memory Registry、Rules/IssueSpec/Request/Grant、digest-only handles、provider固定scope、mutex下のConsume/revoke/epoch、production entropy/UTC clock。 |
| `tools/dev-agent-harness/internal/capability/capability_test.go` | test | Rules/spec、fixed provider scope、handle/digest、entropy/collision、consume/revoke/expiry/epoch、non-leak/immutable、concurrent consume/race coverage。 |
| `tools/dev-agent-harness/README.md` | documentation | opaque handle registryの用途、二つのscope、digest-only/in-memory/restart fail-safe、proxy/Credential/network非対象。 |

## 実装手順

1. Rules validation、公開値型、固定安全error、mutex保護state、production constructorとpackage-private test dependencyを実装する。
2. provider固定のIssueSpec validation、random handle生成、digest lookup、bounded collision retry、entry insertを実装する。
3. Request scope完全一致、atomic Consume、Grant copy、Revoke、monotonic epoch updateとentry失効を実装する。
4. representative unit testsとREADMEを追加し、race検査を含むharnessの既存検査で境界を確認する。

## 検証計画

- unit testsはvalid/invalid Rules、canonical IDとUID、TTL/uses範囲、二つのprovider固定scope、random handle形式とraw handle非保持、entropy failure、bounded collisionとpartial state不在を確認する。
- Consume coverageは全scope fieldのexact match、malformed/unknown handle、mismatch時のuses不変、期限境界、最後のuse、revoke、epoch更新、固定error non-leak、Request/Grantの不変性を確認する。
- 同一1-use handleへの並行Consumeでsuccessが一件だけであり、`go test -race ./internal/capability` がraceを報告しないことを確認する。
- DEV検証ではharnessの `make check` と `make distcheck`、差分の空白検査を実施対象とする。candidate固定時のroot `make check`はMainが一回だけ実行し、QAは同一candidateからfocused rerunを独立に実施する。実Credential、proxy、network、TLS、永続化、restart復元は対象外であり、PASSに代用しない。

## リスクと停止条件

| リスク | 抑制/検出 | 停止条件 |
|---|---|---|
| handle又はCredentialがregistry state/errorに残る | digest-only map key、fixed error、memory/non-leak tests | raw handle又はCredentialを保存・出力する必要が生じたら停止してMainへ戻す。 |
| scope mismatchと成功の間にuses/epoch状態が競合する | single mutex transaction、1-use concurrent consume、race detector | mutex外で判定と消費を分離する必要があれば実装を停止する。 |
| generic providerやhostが固定allow surfaceを広げる | provider別IssueSpec validation、Request exact match tests | 新provider/operation/host、URL/body解釈が必要なら別Taskへ再分類する。 |
| in-memory stateを永続性や実通信の根拠と誤認する | READMEのrestart fail-safeとproxy/Credential非対象明記 | storage、shared process、network接続が必要になれば本Task範囲を超える。 |

## 未解決事項

- なし。公開identifierの正確な綴りとfixed errorの短い文言は、このPLANの値型、非漏洩、provider固定scope、原子性を満たす範囲でDEVが決める。

## main Agentレビュー

- [x] TASKの全AC-IDへ設計判断、パス、順序、失敗時の扱いを対応させ、条件本文を複製していない。
- [x] digest-only handle storage、scope完全一致、atomic consume、expiry/revoke/epochの境界が一意である。
- [x] QA_PLANがTASK-firstで独立作成されている。
- [x] `dependency-ready reconciliation`と完了経路preflightが完了している。
- [x] DEV開始を承認した。
