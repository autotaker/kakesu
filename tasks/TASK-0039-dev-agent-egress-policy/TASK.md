---
task_id: "TASK-0039"
title: "GitHub/OpenAIのegress allowlist policyを実装する"
status: done
created_at: "2026-08-01"
---

# TASK-0039 GitHub/OpenAIのegress allowlist policyを実装する

## `Planning input packet`（Main Agent所有）

このsectionをPlannerとQAへ渡す唯一の`planning input packet`とし、各内容をPLAN/QA_PLANへ複製しない。

### 目的

Development Agent HarnessのTLS終端後に使う、GitHub repository-scoped REST readとOpenAI Responses text requestの最小allowlist判断をpure Go packageとして実装する。実network、Credential、Opaque token置換を有効にする前に、未知のhost、method、path、query、bodyをdefault denyする再利用可能なpolicy coreを固定する。

### 対象と対象外

#### 対象

- `internal/egresspolicy`の`New(Rules)`と`Authorize(Request)`相当のpure API。
- Rulesは許可GitHub repository、許可OpenAI model、OpenAI body上限、output token上限を持ち、空集合、重複、非canonical識別子、無効上限を生成時に拒否する。
- GitHubは`GET`/`HEAD https://api.github.com[:443]/repos/{owner}/{repo}`またはその子pathだけをrepository allowlistと照合する。query、fragment、userinfo、percent encoding、dot/empty segmentは拒否する。
- OpenAIは`POST https://api.openai.com[:443]/v1/responses`、parameterなし`application/json`、上限内のstrict JSON objectだけをmodel allowlistと照合する。
- OpenAI bodyは`model`、non-empty string `input`、explicit `store=false`、explicit `stream=false`、positiveかつ上限内の`max_output_tokens`を必須とし、optional string `instructions`だけを追加で許す。duplicate/unknown field、tool/file/image/background/continuation等は拒否する。
- provider別の固定allow decision、単一の固定request-deny error、Rules用固定error、入力非漏洩・非保持・非変更のunit testとREADME境界説明。

#### 対象外

- HTTP server/proxy、CONNECT、TLS終端、CA、DNS/address検査、socket/listener、upstream通信、redirect処理。
- Authorization header解析、`cap_...`検証、実GitHub/OpenAI Credential取得・置換、logging/audit、rate/cost accounting。
- GitHub GraphQL、write method、organization/user endpoint、Git Smart HTTP、download redirect、Git push、`gh`全コマンド互換。
- OpenAI streaming、response continuation、background、tools、file/image/audio、upload、Realtime/Chat Completions、実API呼出し。
- config/schema/CLI/systemd、Phase 0 executor、Kakesu本体の変更、製品依存追加。

### 受け入れ条件

- [x] AC-1: valid Rulesからpolicyを生成できる。GitHub repositoryはlowercase `owner/repo`、OpenAI modelは安全なASCII identifierとして一意かつnon-emptyであり、各provider allowlist、body上限、output token上限の空/重複/不正を固定Rules errorで拒否する。入力sliceは内部へcopyし、生成後にcallerが変更してもpolicy結果は変わらない。
- [x] AC-2: GitHub requestはcanonical `https` URL、host `api.github.com`、port省略または443、userinfo/fragment/queryなし、method `GET`/`HEAD`、path `/repos/{owner}/{repo}`またはnon-empty child segmentだけを候補とする。owner/repoがRulesへ完全一致した場合だけ`allow/github-rest-read`を返し、HTTP、別host/port、別method、root/user/org/GraphQL、allowlist外、percent encoding、dot/empty segment、4 KiB超URLは固定denyを返す。
- [x] AC-3: OpenAI requestはcanonical `POST https://api.openai.com[:443]/v1/responses`、userinfo/fragment/queryなし、parameterなし`application/json`、non-emptyかつRules上限内bodyだけを候補とする。strict objectが許可fieldを正しい型で一度ずつ持ち、allowed model、non-empty string input、explicit falseのstore/stream、positiveかつRules上限内max_output_tokensを満たす場合だけ`allow/openai-responses-text`を返す。
- [x] AC-4: malformed/trailing JSON、duplicate/unknown field、非許可model、欠落/型違い/空input、欠落またはtrueのstore/stream、欠落/0/小数/過大max_output_tokens、別request surface、空/過大bodyを同じ固定denyで拒否する。optional instructionsはstringだけを許し、tool/file/image/background/continuation等を別schemaで先取りせずunknownとして拒否する。
- [x] AC-5: deny decision/errorはURL、repository、model、body、Credentialらしい入力値やparser/OS errorを文字列化しない。nil/zero policyとinvalid requestはallowにならない。policyはfile/environment/process/network/DNS/TLS/clock/randomを使わず、Request bodyを保持・変更しない。
- [x] AC-6: table-driven unit testsがAC-1〜5の代表的なallow/deny境界、non-leak、Rules/Request/body不変性を検出する。harness `make check`、`make distcheck`、root `make check`がPASSし、変更は`internal/egresspolicy/`とREADMEに限定する。実network、Credential、TLSのPASSを主張しない。

### 安定した参照

| 参照ID | 対象 | 固定改訂/取得日 | 用途 |
|---|---|---|---|
| REF-1 | `docs/development/development-agent-harness.md` | main `3d27045`時点の§8、§9、§14、§16 | TLS終端後の検査、GitHub/OpenAI最小flow、段階導入 |
| REF-2 | OpenAI Responses Create API reference/OpenAPI | `https://developers.openai.com/api/reference/resources/responses/methods/create`、2026-08-01取得 | `POST /v1/responses`、store/stream/input/model/max_output_tokens surface |
| REF-3 | GitHub REST repository endpoints | `https://docs.github.com/en/rest/repos/repos?apiVersion=latest`、2026-08-01取得 | `api.github.com/repos/OWNER/REPO`型のrepository-scoped REST path |
| REF-4 | TASK-0035 config policy | main `873993e`時点 | default deny、fixed safe error、pure boundary pattern |

### 依存状態

| 依存 | 状態 (`ready` / `pending`) | planning参照 | `ready`後に固定する値 |
|---|---|---|---|
| TASK-0035 | `ready` | REF-4 | fail-closed/non-leak patternだけを参照し、config schemaは変更しない |
| TLS proxy/Credential broker | `pending` | 対象外 | 後続Taskで本policyを接続するときに別審査する |

### 許可パス

- `tools/dev-agent-harness/internal/egresspolicy/`
- `tools/dev-agent-harness/README.md`

### 完了経路preflight

| 確認対象 | 結果 | コマンドまたは根拠 |
|---|---|---|
| 完了checker | `ready` | TASK-0037のplanning/candidate/completion gateを使う |
| 権限 | `ready` | pure packageでnetwork、file、Credential、process権限なし |
| 依存状態と参照 | `ready` | TASK-0035完了、REF-1〜4を固定 |
| 生成物の有無と更新方法 | `ready` | configureその他生成物を変更しない |
| 割当ワークツリー | `ready` | `worktrees/TASK-0039-dev-agent-egress-policy` |
| Lapログの書込・Schema・`repository annotation` | `ready` | 標準3 commits、legacy binding新規作成なし |

### 未決事項

- なし

### `Dependency-ready reconciliation`

- TASK-0035はplanning開始時点でready。TLS proxy/brokerはpendingのまま対象外とし、本Taskのallow decisionを実network許可とみなさない。

## 背景

Phase 1でproxy、Opaque capability、Credential置換を同時に作ると、request解釈の欠陥とnetwork/TLS/Credentialの欠陥が混在する。最初のsurfaceをrepository-scoped GitHub REST readとstateless non-streaming text-only Responses requestへ限定し、未知操作をpure fixtureで高速に拒否できる正本を先に作る。

## 検討すべき設計観点

- host/method/path/query/bodyの全条件をANDにし、prefix/suffix host照合を使わない。
- URL parserのnormalizationで曖昧なraw入力を許さず、userinfo、fragment、percent encoding、dot/empty segmentを拒否する。
- OpenAI bodyはpolicy判断に必要なfieldだけをstrict parseし、full API schemaを複製しない。未知fieldは将来互換で受理せずdenyする。
- allow decisionは後続のCredential置換や外部接続を含意せず、proxyが別途全境界を満たすまでinertである。

## 完成の定義

- [x] 受け入れ条件を満たしている。
- [x] 製品変更の3 commit経路、同一candidateの独立REVIEW/QA、root check一回、post-merge Task checkを満たしている。
- [x] provider別allow surfaceとpure policy/proxy責務分離を再利用可能なSemantic Wikiへ記録している。

## 関連コンテキスト

### 意味 Wiki

- providerごとの最小allow surface、strict body、pure policyとTLS proxy/Credential brokerの責務分離。

### 判断

- 初期GitHubはqueryなしrepository-scoped GET/HEAD、初期OpenAIはnon-streaming text-only Responses POSTだけとする。
- policy coreはCredentialもnetworkも扱わず、後続proxyが明示的に呼ぶまでinertである。

### 適用しなかった重要な判断

- generic URL allowlist、GitHub GraphQL、OpenAI unknown-field許容、汎用HTTP proxy、実token置換は採用しない。
