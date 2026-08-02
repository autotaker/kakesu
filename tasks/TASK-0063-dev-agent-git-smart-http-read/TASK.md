---
task_id: "TASK-0063"
title: "Git Smart HTTP readを限定許可する"
status: draft
created_at: "2026-08-02"
---

# TASK-0063 Git Smart HTTP readを限定許可する

## `Planning input packet`（Main Agent所有）

このsectionをPlannerとQAへ渡す唯一の`planning input packet`とし、各内容をPLAN/QA_PLANへ複製しない。

### 目的

既存のpeer認証済みegress socketとOpaque capability経路を使い、設定allowlist内の一つのGitHub repositoryに対するGit Smart HTTPのread操作だけを成立させる。Agentへ実GitHub credentialを渡さず、`git-upload-pack`だけを許可し、`git-receive-pack`を含むpush経路は後続の承認Taskまで拒否したままにする。

### 対象と対象外

#### 対象

- `github.com/<owner>/<repo>.git/info/refs?service=git-upload-pack`のdiscovery GETと、同一repositoryの`git-upload-pack` POSTだけをcanonical URL・method・query・media type・bounded bodyで許可する。
- GitHub REST readとは別のGit read operation/host scopeをCapability Registryとcontrol issueへ追加し、peer-derived subject、repository allowlist、TTL 5分、1回使用、同一Registryの既存契約を維持する。
- AgentがGit credential protocol相当のHTTP Basicで提示する`x-access-token:cap_...`をGit read scopeでだけ受理し、capability消費後に既存resolverが得た実tokenをupstreamのHTTP Basicへ置換する。REST/OpenAIのBearer/token契約は維持する。
- CONNECT、MITM CA、inner HTTP mapping、policy、transaction、forwarder、pinned transportの各層で`github.com:443`をGit readのみに通し、redirect、retry、credential forwarding先の変更を許さない。
- binary Git responseを上限付きで応答sinkへ返し、Git operationごとの厳密なresponse media typeと成功statusを検査する。
- 実GitHubへ接続しないhermetic testで、正規read flow、scope/subject一致、credential置換、pushおよびURL/HTTP逸脱の拒否、既存REST/OpenAI回帰を確認する。

#### 対象外

- `git-credential-dev-agent`のclient実装、Git config/launcher/environment注入、実`git clone/fetch/pull`は後続Taskとする。
- `git-receive-pack`、push capability、Approval、スマホ/Tailscale UI、grant queue、write token発行は実装しない。
- redirectは同一host/repositoryであっても追従しない。LFS、submodule、archive、GitHub Web/GraphQL/REST writeを追加しない。
- 新しいlistener/socket/unit、TCP入口、credential/tokenのAgent側保存、registry永続化、cache、retry、依存追加、Kakesu本体runtime/Schema/build境界を変更しない。
- 実DNS/TLS/GitHub App token、別UID/NSS、systemd/VPS配置をhermetic PASSで代替しない。
<!-- safety_contractの場合: 製品コード、test、runtime/build設定、Schema、製品依存、生成製品入力/成果物、外部観測可能な挙動を変更しない。 -->

### 受け入れ条件

<!-- AC-IDはTask内で一意かつ安定させ、観測可能な結果をここに一度だけ記載する。 -->

- [ ] AC-1: allowlist内の正規`owner/repo`について、GET `/<owner>/<repo>.git/info/refs?service=git-upload-pack`とPOST `/<owner>/<repo>.git/git-upload-pack`だけがGit read scopeへ評価され、method/path/query/media type/body上限の逸脱と`git-receive-pack`は認証情報取得・network到達前に固定拒否される。
- [ ] AC-2: control issueは明示的なGit read selectorから`provider=github`、同一repository、Git read operation、`github.com` hostの5分・1回Capabilityをpeer-derived subjectへ発行し、既存GitHub REST issueとOpenAI issueの意味を変えない。
- [ ] AC-3: Git read requestだけが厳密なHTTP Basic `x-access-token:cap_...`からOpaque handleを抽出し、同じRegistryで消費した後にresolverを一度呼び、upstreamへ`x-access-token:<real token>`のHTTP Basicを一度だけ送る。handle・token・URL・下位errorを応答または診断へ出さない。
- [ ] AC-4: CONNECT、CA、inner HTTP mapping、forwarder、pinned transportは`github.com:443`を正規Git read flowにだけ通し、redirect/retryを行わず、binary responseをoperation対応media type・size/status検査後にだけsinkへ渡す。
- [ ] AC-5: push/receive-pack、repository/host/operation/subject mismatch、malformed Basic、URL encoding/dot/empty segment、余分/欠落query、誤Content-Type、過剰/不正body、unexpected responseを拒否し、既存GitHub REST/OpenAIのBearer/token・JSON・transport契約を回帰させない。
- [ ] AC-6: candidateは承認済み20パス以内、おおむね1,000〜1,400行規模に収まり、helper/launcher/Approval/live state、Kakesu runtime、Schema、依存、生成物を変更せず、focused race、root `make check`、`git diff --check`がPASSする。

### 安定した参照

| 参照ID | 対象 | 固定改訂/ダイジェスト | 用途 |
|---|---|---|---|
| REF-1 | Harness設計 §7.2、§9.3〜9.4、Phase 2 | main `07feb1a` | Git Smart HTTP read、opaque helper credential、push拒否の設計境界 |
| REF-2 | TASK-0061 Capability control/Registry/transaction | main `07feb1a` | Git read scopeを追加する既存のpeer-bound発行・一回消費経路 |
| REF-3 | 既存CONNECT/CA/handler/forwarder/transport | main `07feb1a` | `github.com`を追加する層と既存REST/OpenAI回帰境界 |

### 依存状態

| 依存 | 状態 (`ready` / `pending`) | planning参照 | `ready`後に固定する値 |
|---|---|---|---|
| なし | `ready` | N/A | N/A |

### 許可パス

- `tools/dev-agent-harness/internal/egresspolicy/egresspolicy.go`
- `tools/dev-agent-harness/internal/egresspolicy/egresspolicy_test.go`
- `tools/dev-agent-harness/internal/egresspolicy/scope_test.go`
- `tools/dev-agent-harness/internal/capability/capability.go`
- `tools/dev-agent-harness/internal/capability/capability_test.go`
- `tools/dev-agent-harness/internal/capabilitycontrol/control.go`
- `tools/dev-agent-harness/internal/capabilitycontrol/control_test.go`
- `tools/dev-agent-harness/internal/connectsession/session.go`
- `tools/dev-agent-harness/internal/connectsession/session_test.go`
- `tools/dev-agent-harness/internal/proxyca/proxyca.go`
- `tools/dev-agent-harness/internal/proxyca/proxyca_test.go`
- `tools/dev-agent-harness/internal/brokerhttp/handler.go`
- `tools/dev-agent-harness/internal/brokerhttp/handler_test.go`
- `tools/dev-agent-harness/internal/egresstransaction/egresstransaction.go`
- `tools/dev-agent-harness/internal/egresstransaction/egresstransaction_test.go`
- `tools/dev-agent-harness/internal/upstreamforwarder/upstreamforwarder.go`
- `tools/dev-agent-harness/internal/upstreamforwarder/upstreamforwarder_test.go`
- `tools/dev-agent-harness/internal/upstreamtransport/upstreamtransport.go`
- `tools/dev-agent-harness/internal/upstreamtransport/upstreamtransport_test.go`
- `tools/dev-agent-harness/README.md`

### 完了経路preflight

| 確認対象 | 結果 | コマンドまたは根拠 |
|---|---|---|
| 完了checker | `ready` | 通常product 3トランザクション、同一candidate独立REVIEW/QA、no-ff completionを使用する |
| 権限 | `ready` | credential-bearing protocol/transport変更のためDEVは`dev-sol`、Mainだけがstage/commit/merge/pushする |
| 依存状態と参照 | `ready` | TASK-0061までのpolicy、Registry、control、transaction、CONNECT/service graphがmain `07feb1a`へ反映済み |
| 生成物の有無と更新方法 | `ready` | Go source/testとREADMEのみ。code generation、dependency、configure再生成なし |
| 割当ワークツリー | `ready` | `worktrees/TASK-0063-dev-agent-git-smart-http-read` |
| Lapログの書込・Schema・`repository annotation` | `not-applicable` | 新規log/Schema/annotationなし |

### 未決事項

- なし

### `Dependency-ready reconciliation`

<!-- 依存ready時にMainがready参照、planning参照との差分、AC/設計/スコープ/QAへの影響、再承認結果を追記する。依存なし又は未readyならN/Aとする。 -->

- N/A

## 背景

TASK-0061でAgentは限定Capabilityを発行・失効できるようになったが、GitHub側は`api.github.com`のREST readだけであり、Gitのpull/fetchが使う`github.com` Smart HTTPは全層で拒否される。credential helperを先に実装しても利用可能なread経路がないため、Phase 2を「Smart HTTP read」と「helper/client」に分け、まず既存egress graphのend-to-end read境界を完成させる。pushはログインユーザの非同期スマホ承認と短命write grantを必要とするため、このTaskへ混ぜない。

## 検討すべき設計観点

- GitHub RESTとGit Smart HTTPをprovider名だけで区別せず、operation/hostの完全一致をCapability、policy、authorization scheme、forwarder response検査まで一貫させる。
- discoveryだけにqueryを許すため、inner HTTP mapperは任意queryを解禁せず、canonicalな単一queryを保持できる最小構造にする。
- Git protocol payloadはbinaryであるためJSON検査を適用せず、media type・size・status・operationでfail closedにする。packet内容の意味解析はwrite防止に必要ないため追加しない。
- capability handleと実tokenは同じBasic wire位置を使うが、抽出、消費、resolver、置換の順序を固定し、Agent入力をupstreamへ転送しない。
- 一Taskのend-to-end品質を保つため層を分割した不動作中間Taskを作らず、変更は20パス以内・約1,400行上限として実装とnegative testsを同じcandidateに収める。

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
