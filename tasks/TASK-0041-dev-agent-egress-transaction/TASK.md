---
task_id: "TASK-0041"
title: "Capability連携済みegress transactionを実装する"
status: plan
created_at: "2026-08-01"
---

# TASK-0041 Capability連携済みegress transactionを実装する

## `Planning input packet`（Main Agent所有）

このsectionをPlannerとQAへ渡す唯一の`planning input packet`とし、各内容をPLAN/QA_PLANへ複製しない。

### 目的

TASK-0039のHTTP allowlist判断とTASK-0040のOpaque capability消費を一つのfail-closedなtransactionへ接続する。Agentが送ったcapability形式のAuthorizationを、requestのcanonical provider scopeと同じsubjectへ束縛して一回消費し、その後にだけtrustedなresolverから実Credentialを取得して上流用Bearer headerへ置換する。Credential-bearing requestはtransaction内部から同期的にtrusted Forwarderへ一度だけ渡し、呼出元へ返さない。後続HTTP proxyが認可順序やscope導出を再実装せず、実CredentialをAgentへ返さない最小境界を固定する。

### 対象と対象外

#### 対象

- `internal/egresspolicy`がallow判断と同時にcanonical provider、repository、operation、destination hostを返すscope API。既存`Authorize`のdecision/error互換性は維持する。
- `internal/egresstransaction`のRules、Subject、Request、PreparedRequest、CredentialResolver、Forwarder、Transaction API。RequestはHTTP adapterが取得したAuthorization値をsliceで受ける。
- 入力Authorization値はslice長が一つで、OpenAIは`Bearer cap_...`、GitHubは`Bearer cap_...`又は`token cap_...`だけを受理し、大小文字、余分な空白、複数値、改行、別schemeを拒否する。
- transactionはHTTP allowlist、Authorization形式、capability scope/期限/uses/epochの順に検証し、成功したcapabilityを一回消費した後だけresolverとForwarderを各一回呼ぶ。
- resolverはcanonical provider/repositoryだけを受け、返した1〜4,096 byteのvisible ASCII Credentialだけを上流用`Authorization: Bearer ...`へ変換する。Credential-bearing PreparedRequestはTransactionが保持するtrusted Forwarderへ同期呼出中だけ渡し、Executeは値を返さず、PreparedRequestにcapability handleを残さない。
- policy/auth/capability/resolverの失敗、nil/zero依存、同一1-use handleの並行実行、入力不変、固定error non-leakをunit testで確認する。

#### 対象外

- 実Credentialのファイル読取、保存、生成、更新、GitHub App JWT/installation token交換、OpenAI key管理。
- Forwarderの実装、HTTP server/proxy listener、CONNECT/TLS終端、CA、DNS/address検証、socket、upstream通信、response、redirect、retry、streaming、ログ/監査。
- Git Smart HTTP、credential helper、push、Approval、Tailscale、Passkey、永続化、config/CLI/systemd/provision変更。
- capabilityのrollbackや返却。resolver/Forwarder失敗は一回の認可試行として消費済みのままfail-closedとし、同じhandleを再利用しない。
- Kakesu本体、外部製品依存、配布/install構成の変更。

### 受け入れ条件

- [ ] AC-1: `egresspolicy`の新scope APIは、既存allowlistと同じ一回の評価からGitHub REST readなら`github`/canonical `owner/repo`/`github-rest-read`/`api.github.com`、OpenAI Responses textなら`openai`/repositoryなし/`openai-responses-text`/`api.openai.com`を返す。denyはzero scopeと既存の固定`request-denied`だけを返し、既存`Authorize`の全decision/errorを変えない。
- [ ] AC-2: valid RulesからTransactionを生成でき、policy、capability Registry、CredentialResolver、Forwarderのnil又はCredential最大長が1〜4,096以外なら固定Rules errorで拒否する。non-nilのzero policy/RegistryはExecute時に固定denyとなる。Subjectはcanonical agent instance、non-root UID、workspaceを保持し、Requestとcaller-owned Authorization/body sliceを保存・変更しない。
- [ ] AC-3: `Execute`はpolicy allow、provider別の厳密なcapability Authorization抽出、scope APIから作る完全一致`capability.Request`のConsumeを順に行う。policy/auth denyではcapability、resolver、Forwarderを使わず、capability denyではresolverとForwarderを使わない。scope/repository/host/operationをcallerの別入力から補完又は正規化しない。
- [ ] AC-4: capability Consume成功後だけresolverをcanonical provider/repositoryで一回呼ぶ。resolver error又は空、上限超過、space/tab/改行/control/non-ASCIIを含むCredentialは固定execute errorとなり、resolver/Forwarder再試行とcapability復元をしない。valid CredentialだけをForwarderへ一回渡す。同じ1-use handleへの並行Executeはexactly oneだけがresolverとForwarderへ到達し、race detectorでdata raceがない。
- [ ] AC-5: Forwarderへ同期的に渡すPreparedRequestはmethod、raw URL、content type、bodyの独立copy、canonical provider scope、`Bearer ` + 実Credentialだけを持ち、Opaque handle又は入力Authorizationを保持しない。ExecuteはCredential-bearing値を返さず、Transactionも保持しない。errorはURL、body、subject、repository、handle、Credential、resolver/Forwarder detailを含まない固定値であり、transaction自身はfile/environment/process/network/DNS/TLSを使わない。
- [ ] AC-6: table-driven unit testsがscope導出、既存Authorize互換、両providerの成功、Authorization境界、policy/scope/capability/resolver/Forwarder deny、Credential検証、消費順序、入力不変、non-leak、並行1-useを検出する。`go test -race ./internal/egresspolicy ./internal/egresstransaction`、harness `make check`/`make distcheck`、root `make check`がPASSする。candidateのbaseとの差分は対象2 packageとREADMEだけとし、`git diff --numstat <base>...<candidate>`の追加行＋削除行の合計を1,200以下とする。

### 安定した参照

| 参照ID | 対象 | 固定改訂/ダイジェスト | 用途 |
|---|---|---|---|
| REF-1 | `docs/development/development-agent-harness.md` | main `2cfab90`時点の§7.2、§8〜9、§12、§14〜16 | capability置換順序、provider scope、fail-closed、段階導入 |
| REF-2 | TASK-0039 egress policy | candidate `709b841` / completion `41e7e33` | canonical HTTP allowlistと固定decision/error |
| REF-3 | TASK-0040 capability registry | candidate `072502c` / completion `2cfab90` | scope完全一致、一回消費、失効、固定error |
| REF-4 | OpenAI Responses OpenAPI | OpenAPI `2.3.0`を2026-08-01取得 | `/v1/responses`の`Authorization: Bearer`入力 |
| REF-5 | GitHub REST authentication / GitHub CLI environment manual | REST API version `2026-03-10`、2026-08-01取得 | `GH_TOKEN`利用とBearer/token scheme互換 |

### 依存状態

| 依存 | 状態 (`ready` / `pending`) | planning参照 | `ready`後に固定する値 |
|---|---|---|---|
| TASK-0039 | `ready` | REF-2 | scope APIは既存policyのallow結果からだけ導出する |
| TASK-0040 | `ready` | REF-3 | Consume APIとprovider/operation/host定数を正本にする |

### 許可パス

- `tools/dev-agent-harness/internal/egresspolicy/`
- `tools/dev-agent-harness/internal/egresstransaction/`
- `tools/dev-agent-harness/README.md`

### 完了経路preflight

| 確認対象 | 結果 | コマンドまたは根拠 |
|---|---|---|
| 完了checker | `ready` | planning/candidate/completionの3 commit gateとpost-merge `task-check`を使う |
| 権限 | `ready` | pure memory transactionとinjected resolverだけ。実Credential/file/network権限なし |
| 依存状態と参照 | `ready` | TASK-0039/0040完了、REF-1〜5を固定 |
| 生成物の有無と更新方法 | `ready` | Go source/testとREADMEだけ。configure/Makefile/generated配布物は変更しない |
| 割当ワークツリー | `ready` | `worktrees/TASK-0041-dev-agent-egress-transaction` |
| Lapログの書込・Schema・`repository annotation` | `ready` | 標準3 commits。新Schema、digest転記、追加機械checkなし |

### 未決事項

- なし

### `Dependency-ready reconciliation`

<!-- 依存ready時にMainがready参照、planning参照との差分、AC/設計/scope/QAへの影響、再承認結果を追記する。依存なし又は未readyならN/Aとする。 -->

- TASK-0039/0040はplanning開始時点でready。新scope APIはTASK-0039のallow面を広げず、transactionはTASK-0040のConsume成功を実Credential解決より前に要求する。実Credential sourceとHTTP forwardingはpendingのまま対象外とする。

## 背景

TASK-0039はHTTP requestだけを評価し、TASK-0040はcallerが渡したscopeだけを評価するため、現状は両方を同じrequestへ適用する責任が未実装である。後続proxyがURLからrepositoryを再解析したり、Credential取得をcapability検証より先に行うと、二つの安全境界の間にscope不一致や秘密処理の不要な起動が生じる。ネットワークや秘密ストアを接続する前に、一意なscope導出、消費順序、置換結果をhermeticに固定する。

## 検討すべき設計観点

- scopeは`egresspolicy`のallow評価と同じparser結果から返し、transaction側でURLを再解析しない。
- Agent入力のAuthorizationと上流用Authorizationを別の値として扱い、PreparedRequestからOpaque handleを除く。
- capabilityを消費してからCredentialを解決し、未認可requestが秘密処理へ到達しない順序を固定する。
- Credential-bearing requestはconstructorで注入されたtrusted Forwarderへの同期呼出にだけ渡し、Executeの戻り値やTransaction stateへ残さない。
- resolver/Forwarder失敗後にhandleを戻すrollbackを作らず、再試行による二重作用を避ける。
- transactionに実HTTP transport、listener、secret fileを先取りせず、後続trusted adapterが実装するForwarder interfaceまでにする。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] planning/candidate/completionの3 commit経路を満たしている。
- [ ] 同一candidateの独立REVIEW/QA、candidateで一回のroot `make check`、post-merge `task-check`を満たしている。
- [ ] 再利用可能なscope導出・消費・Credential置換順序をSemantic Wikiへ記録している。

## 関連コンテキスト

### 意味 Wiki

- egress policyとcapability registryを接続するtransaction順序。

### 判断

- provider scopeはHTTP allowlistの同じ評価から導出し、capability検証より前に実Credential resolverを呼ばない。実Credentialは同期Forwarder呼出だけへ渡す。

### 適用しなかった重要な判断

- URLの再解析、Credential先読み、Credential-bearing戻り値、capability rollback、HTTP transport同梱、generic provider frameworkは採用しない。
