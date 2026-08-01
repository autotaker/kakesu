---
task_id: "TASK-0045"
title: "provider上流HTTPS transportを実装する"
status: done
created_at: "2026-08-01"
---

# TASK-0045 provider上流HTTPS transportを実装する

## `Planning input packet`（Main Agent所有）

このsectionをPlannerとQAへ渡す唯一の`planning input packet`とし、各内容をPLAN/QA_PLANへ複製しない。

### 目的

TASK-0043のprovider resolverと後続request Forwarderへ注入できる、broker側の既定HTTPS `http.RoundTripper`を実装する。接続先を`api.github.com`と`api.openai.com`に限定し、環境proxyを使わず、DNS解決結果のprivate/loopback/link-local等を接続前に拒否し、検査済みIPへ直接dialしながら元hostnameのTLS証明書を検証する。最初は再試行、redirect、keep-alive、HTTP/2を持たない一request一接続の小さい境界にし、hermeticなTLS/DNS testで単体QAできる状態にする。

### 対象と対象外

#### 対象

- `internal/upstreamtransport`のproduction constructorと`http.RoundTripper`実装。利用可能なoriginは暗黙443の`https://api.github.com`と`https://api.openai.com`だけとし、userinfo、明示port、opaque URL、未知scheme/host、およびURL authorityと異なる`Request.Host`をnetwork前に拒否する。
- system DNSをbroker側で一回解決し、空answer、zone付きaddress、unspecified、loopback、private、link-local、multicast、非global-unicast、又はpublic/private混在answerを全体拒否する。検査済みIP literalへだけdialし、DNS名をsocket dialへ再度渡さない。
- TLS 1.2以上、元hostnameのSNI/証明書検証、system root CA、HTTP/1.1限定、環境proxy/自動compression/keep-alive無効、固定connect/TLS handshake/response-header timeout、redirect/retryなし。
- response/error全経路のbody/connection所有権、context cancellation、固定non-leak error、idle connection closeを検証する。request Authorization/body、DNS/TLS/socket error、hostname/IPをerror又はformatへ転送しない。
- package-private dependency seam、runtime生成CA/certificateと`net.Pipe`又はloopback listenerを使うhermetic test。許可origin成功、authority/`Request.Host`境界、dial先IP、SNI/certificate、HTTP request/response、拒否address群、mixed answer、TLS/version/cert/timeout、proxy env非使用、単回DNS/dial、秘密非漏洩を一つのfocused race testで検出する。

#### 対象外

- Agent向けHTTP/HTTPS proxy listener、CONNECT、TLS interception、専用CAの生成・配布・rotation、Agent側trust store、DNS proxy、firewall/nftables、Tailscale又はsystemd接続。
- egress policy/capability/transaction/resolverの変更、request Forwarder、Authorization置換、response body上限/capture/audit、grant、Git Smart HTTP、push/pull、OpenAI Responses body処理。
- DNS TTL cache、CNAME来歴、DoH/DoT遮断、publicなControl Plane IP allow/deny設定、IPv6拡張header、複数接続競争、connection pool、HTTP/2/3、retry/backoff、redirect。
- 実GitHub/OpenAI、実Internet DNS、実system trust store、実proxy/firewallを使うlive E2E。hermetic TLS testのPASSでそれらを保証しない。
- 新しいdependency、config/CLI、service、install artifact、生成物、恒久的な外部network test。

### 受け入れ条件

- [x] AC-1: `New`は固定安全値を持つ`http.RoundTripper`を返し、nil/zero receiverを含め、暗黙443の正規`https://api.github.com`又は`https://api.openai.com`以外、および非emptyでURL authorityと完全一致しない`Request.Host`をDNS/dial前に固定errorで拒否する。環境の`HTTP_PROXY`/`HTTPS_PROXY`/`NO_PROXY`を参照せず、redirectを実装しない。
- [x] AC-2: 各requestはbroker側resolverを一回だけ呼び、全answerを正規化・重複排除して検査する。空又は一件でもzone付き、unspecified、loopback、private、link-local、multicast、非global-unicastを含むanswerはdialなしで拒否し、全件安全な場合だけ返却順の検査済みIP literalとport 443へ上限付きでdialする。hostnameをdialerへ渡さない。
- [x] AC-3: dial後は元のallowlisted hostnameをSNIと証明書検証名に使い、TLS 1.2以上かつHTTP/1.1だけを受理する。system root CA以外のproduction trust又は`InsecureSkipVerify`を持たず、connect/TLS handshake/response header timeoutとrequest context cancellationを強制する。
- [x] AC-4: transportはkeep-alive、自動compression、HTTP/2、proxy、redirect、request retryを行わず、一requestにつきDNS一回・dial最大answer数・TLS成功一回とする。dial失敗時だけ未使用の検査済みIPへ進めるが、TLS handshake又はHTTP送信開始後は別IPへ再試行しない。
- [x] AC-5: 公開error/formatは固定値だけで、Authorization、body、DNS answer、hostname/IP、underlying DNS/dial/TLS/HTTP detailを含まない。`RoundTrip`がresponseとerrorを同時に受けた場合を含む失敗response bodyをcloseし、callerへ成功response bodyの所有権だけを渡す。`CloseIdleConnections`は安全に呼べる。
- [x] AC-6: runtime生成TLS fixtureと注入resolver/dialerによるhermetic testが両originの成功、exact dial IP:443、SNI/hostname verification、TLS/HTTP version、全address拒否、mixed answer、fallback境界、timeout/cancel、proxy env無視、call count、body close、固定error/non-leakを検出する。`go test -race ./internal/upstreamtransport`、harness `make check`/`make distcheck`、candidate launcherのroot `make check`がPASSし、base...candidateの対象packageとREADME差分は追加＋削除1,000行以下である。

### 安定した参照

| 参照ID | 対象 | 固定改訂/ダイジェスト | 用途 |
|---|---|---|---|
| REF-1 | TASK-0043 provider resolver | candidate `1914d7e` / completion `70e14d5` | 注入`http.RoundTripper`、単発GitHub交換、default transport非所有境界 |
| REF-2 | TASK-0041 egress transaction | candidate `c369ec4` / completion `1296427` | capability消費後だけcredential-bearing requestを渡すtrusted Forwarder境界 |
| REF-3 | provider credentials意味Wiki | main `65dd24e`時点 | 未解決のTLS/CA/DNS/proxy/IP責務と実network非保証 |
| REF-4 | Go標準library | repositoryのGo 1.24契約 | `net/http.Transport`、`net.Resolver`、`net.Dialer`、`crypto/tls`だけで実装し外部dependencyを増やさない |

### 依存状態

| 依存 | 状態 (`ready` / `pending`) | planning参照 | `ready`後に固定する値 |
|---|---|---|---|
| TASK-0043 | `ready` | REF-1 | resolverは注入RoundTripper以外を変更せず、production transportを後続packageとして合成可能 |

### 許可パス

- `tools/dev-agent-harness/internal/upstreamtransport/`
- `tools/dev-agent-harness/README.md`

### 完了経路preflight

| 確認対象 | 結果 | コマンドまたは根拠 |
|---|---|---|
| 完了checker | `ready` | planning/candidate/completionの3 commit gateとpost-merge `task-check`を使う |
| 権限 | `ready` | production constructorは実DNS/TLSを利用可能だが、DEV/QAは注入dependencyとlocal TLS fixtureだけを使い外部networkへ到達しない |
| 依存状態と参照 | `ready` | TASK-0043完了。注入RoundTripper契約と非対象transport境界をREF-1で固定 |
| 生成物の有無と更新方法 | `ready` | Go source/testとREADMEだけ。生成物、dependency、config変更なし |
| 割当ワークツリー | `ready` | `worktrees/TASK-0045-dev-agent-upstream-transport` |
| Lapログの書込・Schema・`repository annotation` | `ready` | 標準3 commits。新Schema、digest転記、追加機械checkなし |

### 未決事項

- なし

### `Dependency-ready reconciliation`

- TASK-0043はplanning開始時点でready。resolverの注入契約を維持し、既存packageを変更しない。実GitHub受理はこのTaskでもlive E2E blockedのままとする。

## 背景

provider resolverはGitHub installation token交換を注入RoundTripperへ渡せるが、既定transportを持たない。安易に`http.DefaultTransport`を注入すると環境proxy、DNS再解決、private/metadata address、暗黙のconnection reuseやretryをbrokerの安全境界へ持ち込む。request ForwarderやAgent向けproxyと同時に作る前に、brokerからproviderへ出る一接続のDNS/TLS/socket規則を独立packageとして固定する。

## 検討すべき設計観点

- origin allowlist検査、DNS answer全体検査、検査済みIP dial、hostname TLS検証の順序を崩さず、underlying errorを公開しない。
- production APIは固定安全値のconstructorだけにし、resolver/dialer/root CA/clock等のtest seamはpackage-privateに閉じる。
- `http.Transport`の自動proxy、compression、HTTP/2、stale connection retryを無効にするため、最初はkeep-aliveなしHTTP/1.1一request一接続とする。性能最適化は実測後の別Taskにする。
- mixed DNS answerは安全なaddressだけを選ばず全体拒否し、検査と利用の対応を明瞭にする。dial fallbackは接続未成立時だけに限定する。
- permanent external-network test、新deny設定、CNAME/TTL cache、public IP blocklistを先回りして追加しない。

## 完成の定義

- [x] 受け入れ条件を満たしている。
- [x] planning/candidate/completionの3 commit経路とcandidate一回のroot `make check`を満たしている。
- [x] 同一candidateの独立REVIEW/QAを完了し、live E2E未実施境界をPASSと誤記していない。
- [x] 新しい再利用可能なtransport安全境界を意味Wikiへ一ページだけ更新し、post-merge `task-check`をPASSしている。

## 関連コンテキスト

### 意味 Wiki

- `wiki/semantic/schemas/development-agent-harness-provider-credentials.md`
- `wiki/semantic/schemas/development-agent-harness-egress-transaction.md`

### 判断

- provider別request処理やAgent向けproxyより先に、brokerの上流DNS/TLS接続を一request一接続の小さい`RoundTripper`として固定する。

### 適用しなかった重要な判断

- system/browser向け汎用HTTP client、環境proxy、keep-alive、HTTP/2、retry、redirect、DNS cacheを採用しない。
- private answerを除外して残りへ接続する挙動は、mixed answerを部分的に受理するため採用しない。
