---
task_id: "TASK-0043"
title: "GitHub installation token交換とprovider resolverを実装する"
status: done
created_at: "2026-08-01"
---

# TASK-0043 GitHub installation token交換とprovider resolverを実装する

## `Planning input packet`（Main Agent所有）

このsectionをPlannerとQAへ渡す唯一の`planning input packet`とし、各内容をPLAN/QA_PLANへ複製しない。

### 目的

TASK-0041のtrusted `CredentialResolver`をTASK-0042のbroker credential bundleへ接続する。OpenAIではbroker memory上のAPI keyを返し、GitHubでは呼出しごとに短命App JWTをGitHub installation access tokenへ一回だけ交換する。GitHub tokenは要求された一つのrepositoryへ限定し、Agent、ログ、永続状態へ実Credentialを残さない。実HTTP接続は注入された単発`http.RoundTripper`境界までとし、TLS/DNS/proxy transportや上流request forwardingを後続Taskから独立して単体検証できる状態にする。

### 対象と対象外

#### 対象

- `internal/providercredentials` packageとして、検証済み`brokercredentials.Bundle`と注入された`http.RoundTripper`を使う`egresstransaction.CredentialResolver`実装。
- providerが`openai`かつrepositoryが空の場合だけ、bundleのOpenAI API keyを返し、networkへ触れない経路。
- providerが`github`かつrepositoryがTASK-0039と同じcanonicalな`owner/name`形式の場合だけ、bundleから呼出しごとにApp JWTを生成し、`POST https://api.github.com/app/installations/{installation_id}/access_tokens`を一回だけ`RoundTrip`する経路。
- GitHub requestを単一repository名へ限定する`{"repositories":["name"]}`本文、`Authorization: Bearer {JWT}`、`Accept: application/vnd.github+json`、`Content-Type: application/json`、`X-GitHub-Api-Version: 2026-03-10`。permissionはrequestで拡張も列挙もせず、GitHub App installation自体を必要なread permissionだけでprovisionする契約を維持する。
- `RoundTrip`をredirect/retryなしで一回だけ呼び、1ms以上30秒以下の必須timeoutをrequest contextへ設定する。transportはtrustedな後続境界とし、このpackageはdefault transportを作らない。
- `201`、JSON content type、128 KiB以下の単一top-level JSON object、重複top-level fieldなし、必須`token`/`expires_at`を検証するresponse境界。未知の公式response fieldは無視し、tokenはprefix/固定長を仮定せず1〜4,096 byteのvisible ASCII、expiryはRFC3339で現在より後かつ65分以内だけを受理する。
- response bodyを全経路でcloseし、成功時にtoken文字列だけを同期的callerへ返す。token/JWT/本文/URL/repository/transport・JSON errorを含めない固定error、nil/zero receiverのfail-closed、format/log/永続保持を作らない。
- hermeticなfake `RoundTripper`とruntime生成credential fixtureによるprovider分岐、request束縛、response境界、timeout、call count、body close、固定error/秘密非漏洩、TASK-0041 transaction接続のunit test。

#### 対象外

- installation tokenのcache、refresh、singleflight、revoke、事前取得、retry、backoff。初期実装は許可済みGitHub requestごとに一回交換し、必要性が実測されるまで状態を増やさない。
- `http.Transport`、connection pool、TLS、CA、DNS、proxy、redirect policy、IP検証、socket、実GitHub/OpenAI通信、実Credentialを使うlive E2E。
- GitHub App/installation作成、repository install、permission変更、秘密file作成/rotate、API version自動追従、GitHub Enterprise Server/GHE.com endpoint。
- GitHub REST/OpenAI request forwarding、HTTP server、Agent入口、Authorization置換、audit/log/metrics、rate/cost budget、Git Smart HTTP、push/approval。
- TASK-0039〜0042のAPI/挙動変更、config/CLI/systemd/provision/install、外部Go module、Kakesu本体。
- token prefix又は長さ40文字への依存。GitHubの新しいstateless installation token形式もvisible ASCIIの上限内なら同じ値として扱う。

### 受け入れ条件

- [x] AC-1: validなbundle、non-nil `RoundTripper`、1ms〜30秒のtimeoutからresolverを構築できる。nil bundle/transport、範囲外timeoutは固定invalid-rules errorとなり、nil/zero resolver、未知provider、providerとrepository形の不一致は固定resolve errorとなる。resolverは`egresstransaction.CredentialResolver`を実装する。
- [x] AC-2: `Resolve("openai", "")`はbundleのOpenAI API keyをそのまま返し、transportとJWT生成へ到達しない。OpenAIでrepositoryがある場合、またはGitHub以外のproviderは値なしの固定resolve errorとなる。
- [x] AC-3: canonical `owner/name`のGitHub解決は、bundleのinstallation IDを10進pathへ、repositoryの`name`だけを1要素の`repositories` JSON配列へ束縛し、固定HTTPS host/path/method/headerを持つrequestをtimeout context付きで`RoundTrip`へ一回だけ渡す。JWTはBearerだけに使い、request本文へ入れない。redirect status、transport error、context timeoutをretryせず固定resolve errorにする。
- [x] AC-4: GitHub responseはnon-nil body、status `201`、media type `application/json`、128 KiB以下の重複fieldなし単一JSON objectで、1〜4,096 byte visible ASCIIの`token`と、現在より後かつ65分以内のRFC3339 `expires_at`を各一つ持つ場合だけtokenを返す。未知fieldと新token形式は許容し、missing/duplicate/wrong type、trailing JSON、過大body、期限外、non-visible tokenは固定resolve errorとなる。bodyは成功/失敗の全経路でcloseされる。
- [x] AC-5: resolverはinstallation tokenをcache/保持せず、各GitHub resolveで新しいJWTと一回の交換だけを行う。packageはdefault transport、redirect、retry、log、environment、file/process、永続書込みを使わず、公開error又はformat結果にOpenAI key、JWT、installation token、response/request本文、repository、URL、parser/transport detailを含めない。
- [x] AC-6: unit testsはOpenAI no-network、GitHub requestの完全な束縛、provider/repository拒否、timeout/call count、response/status/content-type/size/JSON/token/expiry境界、body close、固定error/non-leakを検出する。同じresolverをTASK-0041 transactionへ渡すintegration testで、無効capabilityではtransport未到達、valid GitHub/OpenAI requestだけがprovider credentialをtrusted Forwarderへ渡すことを確認する。`go test -race ./internal/providercredentials`、harness `make check`/`make distcheck`、root `make check`がPASSし、base...candidateの対象packageとREADMEの追加＋削除合計は1,200行以下である。

### 安定した参照

| 参照ID | 対象 | 固定改訂/ダイジェスト | 用途 |
|---|---|---|---|
| REF-1 | `docs/development/development-agent-harness.md` | main `387fe9f`時点の§7、§9、§14〜17 | broker-only secret、repo限定GitHub token、Opaque置換、Agent非露出 |
| REF-2 | TASK-0041 egress transaction | candidate `c369ec4` / completion `1296427` | capability消費後だけ呼ぶresolver interface、固定拒否、trusted Forwarder境界 |
| REF-3 | TASK-0042 broker credentials | candidate `5fea9b8` / completion `387fe9f` | 検証済みbundle、installation ID、短命JWT、OpenAI key |
| REF-4 | [GitHub installation token endpoint公式文書](https://docs.github.com/en/enterprise-cloud@latest/rest/apps/apps#create-an-installation-access-token-for-an-app) | 2026-08-01取得、API version `2026-03-10` | POST path、Bearer JWT、repository scope、201 response、1時間expiry |
| REF-5 | [GitHub installation token生成公式文書](https://docs.github.com/en/enterprise-cloud@latest/apps/creating-github-apps/authenticating-with-a-github-app/generating-an-installation-access-token-for-a-github-app) | 2026-08-01取得 | repository/permission上限、token response、Bearer必須、新stateless token形式 |

### 依存状態

| 依存 | 状態 (`ready` / `pending`) | planning参照 | `ready`後に固定する値 |
|---|---|---|---|
| TASK-0041 | `ready` | REF-2 | resolverはcapability消費後にprovider/repositoryだけを受け、実Credentialはtrusted Forwarder以外へ返さない |
| TASK-0042 | `ready` | REF-3 | JWT/installation ID/OpenAI keyは検証済みbundleからだけ取得し、secret fileを再解釈しない |

### 許可パス

- `tools/dev-agent-harness/internal/providercredentials/`
- `tools/dev-agent-harness/README.md`

### 完了経路preflight

| 確認対象 | 結果 | コマンドまたは根拠 |
|---|---|---|
| 完了checker | `ready` | planning/candidate/completionの3 commit gateとpost-merge `task-check`を使う |
| 権限 | `ready` | runtime生成credential fixtureとfake RoundTripperだけを使い、実secret/network/root/external actionを行わない。実GitHubはblocked live E2Eとして残す |
| 依存状態と参照 | `ready` | TASK-0041/0042完了。GitHub公式endpoint/API version/token形式を2026-08-01に固定 |
| 生成物の有無と更新方法 | `ready` | Go source/testとREADMEだけ。外部module、configure/Makefile/generated配布物は変更しない |
| 割当ワークツリー | `ready` | `worktrees/TASK-0043-dev-agent-provider-resolver` |
| Lapログの書込・Schema・`repository annotation` | `ready` | 標準3 commits。新Schema、digest転記、追加機械checkなし |

### 未決事項

- なし

### `Dependency-ready reconciliation`

- TASK-0041/0042はいずれもplanning開始時点でready。resolver interfaceとbundle APIをそのまま利用し、既存packageを変更しない。後続のTLS/DNS/proxy transportとrequest ForwarderをこのTaskのPASSへ含めない。

## 背景

TASK-0042で長期secretとJWT生成はbroker memoryへ閉じたが、TASK-0041のresolverへ接続されておらず、実際のGitHub requestに使うinstallation tokenはまだ得られない。ここでproxy/server/TLS/cacheまで同時に実装すると、Credential交換のsecret境界を独立に検証できない。まず単発RoundTripとprovider分岐だけを実装し、transport hardeningとforwardingを次の単体QA可能な境界へ残す。

## 検討すべき設計観点

- `http.Client.Do`はredirect処理を含むため使わず、単一exchangeだけを意味する`http.RoundTripper.RoundTrip`を注入する。3xxを成功扱いせず、package自身はdefault transportを選ばない。
- GitHub token requestはinstallation全repositoryを暗黙に許可せず、canonical scopeのrepository名を必ずbodyへ一件だけ入れる。ownerはinstallationによって固定され、上流requestのowner/repo束縛はTASK-0041のScope/Forwarder側でも維持される。
- permission fieldは一部だけ列挙すると許可済みread endpointを意図せず壊す一方、未指定でもinstallation permissionを越えない。初期App installation自体をread-only最小permissionでprovisionし、このexchangeではpermissionを追加しない。
- response全体の公式field集合を固定せず、秘密に関係するtop-level JSONの一意性、token、expiry、sizeだけをfail-closedで検証する。prefixや40文字長をtokenの妥当性に使わない。
- cache/retryは性能要件の実測前に状態・競合・再利用窓を増やすため追加しない。1 capability requestに対して1 exchangeという単純な失敗境界を保つ。
- context timeoutはpackageが保証するが、transportのTLS/DNS/proxy/IP policyは注入側の後続責務である。fake transportによるunit PASSを実Internet安全性のPASSと扱わない。

## 完成の定義

- [x] 受け入れ条件を満たしている。
- [x] planning/candidate/completionの3 commit経路と`make check`を満たしている。
- [x] 同一candidateの独立REVIEW/QAと必要なSemantic Wiki更新を完了している。post-merge `task-check`はcompletion後に実行する。
- [x] 実TLS/DNS/proxy transportと実GitHub/OpenAI受理をPASSと誤記せず、後続Task/live E2Eとして残している。

## 関連コンテキスト

### 意味 Wiki

- provider resolverの単発交換、repo scope、response境界。再利用可能な新判断が実装中に生じた場合だけ更新する。

### 判断

- OpenAI keyはnetworkなしでbundleから解決し、GitHub tokenだけを一repositoryへ絞って呼出しごとに交換する。
- redirect/retry/cache/default transportを持たない小さなresolverにし、TLS/DNS/proxy/Forwarderは後続へ分ける。

### 適用しなかった重要な判断

- `http.Client`、default transport、installation-wide token、static token、permissionの部分列挙、token prefix/40文字検査、cache、retryは採用しない。
