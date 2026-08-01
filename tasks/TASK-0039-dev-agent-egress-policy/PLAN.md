---
task_id: "TASK-0039"
change_class: "product"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "luna-xhigh"
approved_dev_profile_reason: "標準libraryだけのpure Go policy packageとunit tests、READMEの局所変更で、network・Credential・TLS・OS権限・外部作用を含まないため"
approved_dev_profile_risk_signals: []
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T04:48:37Z"
planning_reviewed_by: "reviewer-agent-terra-medium"
planning_review_decision: "pass"
planning_reviewed_at: "2026-08-01T04:48:37Z"
classification_approved_by: ""
classification_approved_at: ""
classification_approval_reason: ""
---

# TASK-0039 PLAN

## AC対応

TASKの条件本文を再掲せず、`planning input packet`のAC-IDに設計を対応させる。

| AC-ID | 設計判断 | 変更パス | 実施順序 | 失敗時の扱い |
|---|---|---|---|---|
| AC-1 | `Rules` を GitHub repository群、OpenAI model群、OpenAI body/output上限の値型とし、`New(Rules)` が canonical な値だけを内部の集合へコピーして policy を作る。空集合、重複、識別子不正、非正の上限は同一の Rules 用安全エラーに集約する。 | `internal/egresspolicy/egresspolicy.go`、`internal/egresspolicy/egresspolicy_test.go` | 1 | policyを返さず、caller入力・不正値・parser詳細を含まない固定エラーで拒否する。 |
| AC-2 | `Request` は method、raw URL、content type、bodyだけを受け取る純粋な値とし、GitHub用の raw URL 検査と path segment 検査を全条件成立時だけ allow decision へ到達させる。 | `internal/egresspolicy/egresspolicy.go`、`internal/egresspolicy/egresspolicy_test.go` | 2 | canonical URL、repository、methodのいずれかが不一致なら、共通の request-deny error と allow でない決定を返す。 |
| AC-3 | OpenAIは raw URL・POST・完全一致のcontent type・bounded bodyを先に確認し、duplicate検出を含む strict object decode と field/上限検証を通った場合だけ text-only の allow decision を返す。 | `internal/egresspolicy/egresspolicy.go`、`internal/egresspolicy/egresspolicy_test.go` | 3 | JSON、model、input、store、stream、output token、未知surfaceの差異を区別して外部へ出さず、共通denyへ集約する。 |
| AC-4 | decoderの入力全消費、object重複key検出、unknown field拒否、各許可fieldの型・必須性を別々に確認する。optional `instructions` は string に限り、他の機能を表すfieldは拡張解釈しない。 | `internal/egresspolicy/egresspolicy.go`、`internal/egresspolicy/egresspolicy_test.go` | 3 | malformed、trailing、duplicate、unknown、型違い、空・上限超過のbodyはすべて共通denyとする。 |
| AC-5 | policyは Rule の内部集合だけを保持し、Request/bodyを保持・変更しない。nil/zero policy、nil相当の不完全Request、予期しない値を fail-closed とし、file、environment、process、network、DNS、TLS、clock、randomを導入しない。 | `internal/egresspolicy/egresspolicy.go`、`internal/egresspolicy/egresspolicy_test.go` | 全工程 | allow以外の全経路は安全な固定errorだけを返す。入力由来または標準library由来のエラーを返さない。 |
| AC-6 | positive/negative の table-driven testsでRules、GitHub、OpenAI、non-leak、入力・copy不変性を観測し、READMEには純粋policyの利用範囲と proxy/Credential境界を記す。 | `internal/egresspolicy/egresspolicy_test.go`、`README.md` | 4 | 代表境界を観測できない、許可外path・依存・副作用が必要になる、または検査が失敗する場合は候補を完了扱いにしない。 |

## 補足設計

### APIと判断の固定

- packageは `Rules`、`Request`、opaqueなpolicy値、`New(Rules)`、policyの `Authorize(Request)` を公開する。`Request` はHTTP clientやURL objectを受け取らず、raw URL、method、content type、bodyという検査対象そのものを値で受け取る。
- `Rules` のrepositoryは lowercase ASCII の `owner/repo` 一形式、modelは先頭を英数字とし英数字・`.`・`_`・`-`だけからなる安全なASCII identifierに限定する。repository/modelの集合はいずれも空を許容しない。body/output上限はいずれも正の整数とする。
- `New` は各sliceを検証後に新規mapへ格納する。公開するRulesやRequestのsliceをpolicyが再利用しないため、New後のRules変更とAuthorize後のbody変更のどちらも既存policyの判断に影響しない。
- allow時のdecisionはGitHub REST read用とOpenAI Responses text用の二つの固定値だけとする。invalid Rulesには一つの固定Rules error、Authorizeの全拒否には一つの固定request-deny errorを使い、decisionはallow値にしない。成功・失敗のいずれもURL、repository、model、body、Credentialらしい値、URL/JSON parser errorを文字列化しない。

### GitHub canonical surface

- URLのraw byte長を4 KiB以下に制限し、raw percent encodingを先に拒否する。標準libraryのURL parserは構造確認だけに使い、scheme、raw host、明示port、userinfo、query、fragment、opaque形式を個別に exact match で確認する。hostは `api.github.com`、portは省略または443だけを許可する。
- methodはexact `GET` または `HEAD`、pathは `/repos/{owner}/{repo}` を先頭に、続く場合は空、dot、dot-dotを含まないnon-empty segmentだけを許す。root、user/org/GraphQLなどrepository-scopedでないsurfaceはpath prefixでは許可しない。
- owner/repoはpathからそのまま得たcanonical文字列でRules集合と完全一致させる。hostの大小文字化、portの正規化、path clean、unescape、redirectの追跡を許可根拠にしない。

### OpenAI strict text-only surface

- URLは `https://api.openai.com`（port省略または443）と `/v1/responses` の完全一致だけを受ける。userinfo、query、fragment、opaque形式、別path/host/port、別method、空body、Rules上限を超えるbody、`application/json`以外またはparameter付きcontent typeを、decode前に拒否する。
- bodyはUTF-8のJSON object一つだけとし、token走査でduplicate keyを検出してから、unknown fieldを拒否するtyped decodeを行う。二つ目のJSON値または非空白末尾も拒否する。
- objectは allowed model、non-empty string `input`、`store:false`、`stream:false`、positiveかつRules上限内のinteger `max_output_tokens`を必須とする。`instructions` はstringなら追加で許可する。object/array input、tool/file/image/background/continuationや将来fieldはunknown fieldとして拒否し、全API schemaを複製しない。

### 代替案と不採用理由

- generic HTTP request又は`net/http` requestの受入れ: header、redirect、transportなど未承認の責務をpolicy APIへ持ち込むため不採用。最小の値型を用いる。
- host/pathのprefix照合又はparser正規化後の照合: suffix host、encoded path、dot/empty segmentなどの曖昧な入力を許すため不採用。raw制約とsegment単位の完全一致を組み合わせる。
- `encoding/json`の通常decodeだけ: duplicate keyを見逃すため不採用。先行token scan、strict typed decode、全入力消費を組み合わせる。
- OpenAIのunknown fieldをforward compatibilityとして許容: text-onlyという最小surfaceを広げるため不採用。将来のsurfaceは別Taskで明示的に追加する。
- proxy、TLS、Credential置換、Authorization header解釈を同時に実装: policy判断と外部作用の根拠を混在させるため不採用。Authorizeのallowは実network許可を意味しない。

### 責務・境界・不変条件

- `internal/egresspolicy` だけがRules validation、raw request解釈、provider別allow decision、安全errorを所有する。標準libraryのみを用い、OS状態や外部サービスを読む・書く操作を持たない。
- Authorizeは一回の呼出しに閉じた計算であり、input bytes、Rules slice、Request値を変更しない。返却されたpolicyは内部の集合を通じてだけ参照し、変更可能なcaller memoryの別名を保持しない。
- READMEはこの判断がTLS終端後に後続proxyから呼ばれる想定であること、actual connection/redirect/Credential/token/accounting/auditは範囲外であること、unknown surfaceがdefault denyであることを説明する。

## 変更予定

| パス | 種別 | 変更内容 |
|---|---|---|
| `tools/dev-agent-harness/internal/egresspolicy/egresspolicy.go` | implementation | pure Rules/policy/Request API、immutable allowlist生成、canonical GitHub判定、strict OpenAI body判定、固定decision/error。 |
| `tools/dev-agent-harness/internal/egresspolicy/egresspolicy_test.go` | test | Rules、GitHub URL/path、OpenAI JSON/schema、固定deny non-leak、copy/body不変性、副作用なしのunit coverage。 |
| `tools/dev-agent-harness/README.md` | documentation | policy APIの目的、二つのallow surface、default-denyと非対象の実network/Credential/proxy責務。 |

## 実装手順

1. `Rules`、`Request`、policy内部集合、固定decision/errorを定義し、canonical Rules validationとcopyを実装する。
2. raw URLの共通fail-closed確認を置き、GitHubのmethod/host/port/path/repository照合を実装する。
3. OpenAIのrequest surface確認、bounded strict JSON object parser、fieldと上限の検証を実装する。
4. table-driven unit testsとREADMEを追加し、許可面と拒否面、non-leak、immutable/copy境界、依存しない外部作用を確認する。

## 検証計画

- harness内のunit testは、valid/invalid Rules、Rules変更後のpolicy不変性、GitHubのcanonical allowとURL・method・repository拒否、OpenAIのstrict body allowとfield/JSON/body拒否、nil/zero policy、deny textの非漏洩、Request/body不変性を確認する。
- focused rerunでは `go test ./...`、`make check`、`make distcheck`、`git diff --check` を実施対象とする。network、Credential、TLS、redirect、live provider call、proxy接続は対象外としてPASSに代用しない。
- candidate固定時にMainがroot `make check`を一回だけ実行する。QAは同一candidateからfocused rerunで独立に確認し、root checkの再実行を要求しない。

## リスクと停止条件

| リスク | 抑制/検出 | 停止条件 |
|---|---|---|
| URL parserの正規化が曖昧なraw inputを許可する | raw encoding・authority・path segmentの拒否とnegative unit cases | raw文字列からcanonical性を確認できないsurfaceが必要になればMainへ戻す。 |
| JSON decoderがschema外又は重複keyを取りこぼす | token scan、strict decode、全入力消費、negative unit cases | strict objectを保てない拡張要求は本Task外として停止する。 |
| error又はpolicy状態が入力・Credentialらしい値を保持/露出する | 固定errorのexact assertion、Rules copyとRequest/body不変性のtests | 入力由来文字列又はmutable caller memoryを必要とする設計は採用しない。 |
| pure policyがnetwork/Credential/proxy責務へ拡大する | import/scope auditとREADME boundary | I/O、transport、token処理、redirect又は設定変更が必要ならMainへ再分類を依頼する。 |

## 未解決事項

- なし。公開するGo identifierの正確な綴りは、このPLANの値型・固定error/decision・純粋性の契約を満たす範囲でDEVが決める。

## main Agentレビュー

- [x] TASKの全AC-IDへ設計判断、パス、順序、失敗時の扱いを対応させ、条件本文を複製していない。
- [x] pure API、canonical URL、strict body、copy/non-leakの境界が設計で一意である。
- [x] QA_PLANがTASK-firstで独立作成されている。
- [x] `dependency-ready reconciliation`と完了経路preflightが完了している。
- [x] DEV開始を承認した。
