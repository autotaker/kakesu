---
task_id: "TASK-0045"
change_class: "product"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "luna-xhigh"
approved_dev_profile_reason: "新規internal packageへ固定origin・DNS/IP・TLS接続境界とhermetic testを閉じる局所実装であり、既存policy/resolver、OS権限、実network又は外部dependencyを変更しないため。"
approved_dev_profile_risk_signals: []
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T08:37:57Z"
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

# TASK-0045 PLAN

## AC対応

TASKの条件本文を再掲せず、`planning input packet`のAC-IDに設計を対応させる。

| AC-ID | 設計判断 | 変更パス | 実施順序 | 失敗時の扱い |
|---|---|---|---|---|
| AC-1 | 公開APIは固定production値を閉じ込め、errorを返さない`New() *Transport`、固定のcomparable error、及び`http.RoundTripper`を実装するexported concrete `Transport`に限定する。`RoundTrip`の最初にnil/zero receiver、request/URL、厳密な`https`・allowlisted hostname・implicit 443を構文検証する。非empty `Request.Host`は正規URL authorityと完全一致する場合だけ受理し、userinfo、port、opaque、未知origin又はauthority/Host不一致をresolver/dial前に拒否する。`http.Client`を作らないためredirectは実装しない。 | `tools/dev-agent-harness/internal/upstreamtransport/` | 1 | nil/zero receiverを含む全ての入力・構造エラーを同じ固定公開errorへ畳み、network dependency、request body、Authorization、URL detailを触れずに返す。Host拒否時もDNS/dial 0回とする。 |
| AC-2 | package-private resolver seamはhostnameを一回だけ受け、answerを`netip`で正規化・重複排除する。空、zone付き、unspecified、loopback、private、link-local、multicast、non-global-unicastを一件でも含むanswerは集合全体を拒否する。安全な返却順のIP literalだけを候補にし、package-private dial seamへ`IP:443`を渡す。 | `tools/dev-agent-harness/internal/upstreamtransport/` | 2 | resolver error、invalid/mixed answer、dial候補枯渇を固定errorにし、hostnameをsocketへ再送せず、拒否時はdial 0回とする。 |
| AC-3 | `http.Transport`を一request用の内部実装として構成し、`DialContext`だけで検査済みIPへTCP接続する。`DialTLSContext`は使わない。Go契約上これを設定するとHTTPSで`TLSClientConfig`と`TLSHandshakeTimeout`が無視されるためである。Transport自身にTLS 1.2下限、元hostnameの`ServerName`、system rootのみ、handshake/header timeoutを設定し、HTTP/1-onlyを明示する。 | `tools/dev-agent-harness/internal/upstreamtransport/` | 3 | TCP接続成功後のTLS handshake、certificate/SNI、HTTP version、header timeout又はrequest contextの失敗は固定errorで終了し、次IPを試さない。production trust差替え又は`InsecureSkipVerify`が必要になれば停止してMainへ報告する。 |
| AC-4 | `Proxy:nil`、`DisableCompression:true`、`DisableKeepAlives:true`、HTTP/2無効を固定する。keep-alive無効により使用済みstale connectionをTransportがretryする前提をなくし、一request一接続とする。`DialContext`内でのみ未使用候補へ順次fallbackし、TCP dial失敗時だけ最大answer数まで進める。 | `tools/dev-agent-harness/internal/upstreamtransport/` | 3 | TLS開始後又はHTTP送信開始後の失敗はretry/backoff/redirectなしで固定errorとする。proxy environmentを読まないため環境値による経路変更もない。 |
| AC-5 | exported `Transport`の`RoundTrip`は内部の`http.Transport`の`response,error`を常に検査し、成功responseだけをcallerへ返す。error併存response、nil/invalid response、HTTP/TLS/dial失敗ではbodyをcloseして固定errorへ正規化する。公開`CloseIdleConnections`は内部transportへ安全に委譲し、nil/zero receiverではno-opにする。format/error面にinputやunderlying detailを流さない。 | `tools/dev-agent-harness/internal/upstreamtransport/` | 4 | close失敗を含め失敗responseの所有権をcallerへ渡さず、成功bodyだけをcaller所有にする。nil/zero `CloseIdleConnections`もpanicさせず、固定errorの非漏洩を崩す必要が出れば停止する。 |
| AC-6 | runtime生成CA/leaf certificate、`net.Pipe`又はloopback listener、package-private resolver/dialer/root-pool seamで外部networkなしのtestを作る。両origin、exact IP:443、SNI/hostname validation、TLS 1.2/HTTP/1.1、拒否集合/mixed answer、dial-only fallback、timeout/cancel、proxy env非使用、回数、close、fixed error/non-leakを一つのfocused race suiteで検出する。READMEはtransport責務とlive境界だけを簡潔に追記する。 | `tools/dev-agent-harness/internal/upstreamtransport/`、`tools/dev-agent-harness/README.md` | 2–6 | fixtureは実GitHub/OpenAI、実DNS、system roots、proxy/firewallの証拠にしない。許可外package、外部dependency、config/CLI、外部network testが必要なら停止する。 |

## 関連Wikiと判断

- REF-1の注入`RoundTripper`境界を維持する。`providercredentials`、resolver、egress transaction、Forwarderは変更せず、後続利用者が`New()`の戻り値を注入できるだけにする。
- REF-2/REF-3の通り、credential-bearing requestのforwarding、secret処理、実GitHub/OpenAI受理は本packageの責務に加えない。TLS/DNS/proxy/IPの小さいbroker outbound境界だけをここで固定する。
- WikiはDEV許可パスではない。candidateの独立REVIEW/QA後、Mainがmain正本の既存運用として意味更新とpost-merge `task-check`を所有する。よってDEV計画はread-only参照とREADME追記に限定し、完成定義と許可パスは責務分離で整合する。

## 補足設計

### 責務・境界・不変条件

- exported `Transport`は`http.RoundTripper`を実装し、`New() *Transport`だけが固定timeout、system `net.Resolver`/`net.Dialer`、system root CAを持つ有効値を作る。test依存と内部`http.Transport`は公開しない。test-only resolver/dialer/root-pool constructorはpackage-privateにする。
- origin/`Request.Host` validation → DNS一回 → answer集合の全検査 → IP-literal TCP dial → 元hostnameによるTLS SNI/certificate verification → HTTP/1.1 response、の順序を崩さない。IPの文字列表現、DNS名、URL hostをdialerの`addr`へ混在させない。
- `DisableKeepAlives:true`が接続再利用を除くため、Transportの「既に成功利用されたconnectionでのnetwork error」retry条件を満たすstale connectionを作らない。new TCP connectionへのdial-only fallbackと、TLS/HTTP後retry禁止を別々にtestする。
- production TLS設定は`MinVersion: tls.VersionTLS12`、`ServerName`に元allowlisted hostname、`RootCAs:nil`（system roots）、`InsecureSkipVerify:false`を維持する。HTTP/2は`ForceAttemptHTTP2:false`に加えHTTP/1-only protocol configurationで明示し、ALPNもHTTP/1.1だけを提示する。
- public error、`Error()`、format相当の出力は固定文字列だけにする。request Authorization/body、DNS answer、hostname/IP、underlying errorをログ・wrap・format・test外diagnosticへ出さない。

### 代替案と不採用理由

- `http.DefaultTransport`はproxy environment、DNS再解決、connection reuse及びretryの既定を持つため使わない。
- `DialTLSContext`で自前TLSを完結させる案は、GoのTransportが`TLSClientConfig`と`TLSHandshakeTimeout`を無視する契約となり、SNI/certificate/timeoutの固定がTransport設定から検証不能になるため採用しない。
- 安全なDNS answerだけを選ぶ、DNS cache/CNAME追跡/DoH、HTTP/2/3、pool/retry/backoff、proxy listener/CONNECTは、集合検査又は一接続境界を拡張するため採用しない。

### 移行・互換性

- 新規internal packageとREADME追記だけで、既存公開CLI/config/module dependency、TASK-0041/0043 package、生成物を変更しない。差分は許可2パスの追加・削除合計1,000行以下に保つ。
- live E2Eは実provider、Internet DNS、system trust、proxy/firewallと安全なcleanupが未提供のためblockedのままとし、hermetic TLS/DNS testのPASSで代替しない。

## 変更予定

| パス | 変更内容 |
|---|---|
| `tools/dev-agent-harness/internal/upstreamtransport/` | 固定安全constructor、origin/DNS/IP validator、HTTP/1.1 TLS Transport wrapper、package-private test seams、runtime TLS fixtureを用いるhermetic unit/race testsを追加する。 |
| `tools/dev-agent-harness/README.md` | brokerのallowlisted upstream transport、DNS/IP/TLS/one-request boundary、後続Forwarderとlive E2Eの対象外を簡潔に記録する。 |

## 実装手順

1. errorなしの`New() *Transport`、`http.RoundTripper`適合、nil/zero `RoundTrip` fail-closed・`CloseIdleConnections` no-op、strict originとnonempty `Request.Host`/URL authority完全一致のDNS前検証、package-private dependency seamを定義する。
2. DNS answerの正規化・全件検査とIP literal candidate生成を実装し、reject-before-dial、単回resolver、exact dial targetのtestsを置く。
3. `DialContext`を使うHTTP/1.1-only Transportを構成する。proxy/compression/keep-alive/HTTP2を無効化し、TLS config/timeoutsをTransportに適用してdial-only fallback境界を固定する。
4. response/error ownership、body close、context/timeout、fixed non-leak errorと`CloseIdleConnections`を実装し、TLS又はHTTP以後のretryがないことをtestする。
5. runtime TLS CA/certificate と fake resolver/dialerでfocused race suiteを完成させ、READMEへ責務・非対象を追記する。
6. candidate launcherを一回だけ実行してroot `make check`をcandidate-bound証跡にし、package race test、harness `make check`/`make distcheck`、scope/diff line limitと`git diff --check`を確認する。

## 検証計画

- `go test -race ./internal/upstreamtransport`で、`New() *Transport`の固定値とinterface適合、nil/zero `RoundTrip`のreject・`CloseIdleConnections` no-op、origin及び非empty `Request.Host`/URL authority不一致のreject-before-network（DNS/dial 0回）、DNS全answer reject・dedupe・一回呼出、IP:443 dial、両originのSNI/certificate、TLS 1.2/HTTP/1.1、proxy env不使用、TCP-only fallback、TLS/HTTP後no-retry、timeouts/cancellation、body ownership/close、fixed non-leak errorを検出する。
- fixtureはloopback/`net.Pipe`とruntime certificateだけを使い、実provider、実DNS、実system trust、実proxy/firewallを使わない。
- candidateでharness `make check`、`make distcheck`、candidate launcherのroot `make check`（一回）、`git diff --check`を実行する。base...candidateの変更パスが許可2パスだけで、追加・削除合計が1,000行以下を確認する。
- 実provider接続、実Internet DNS、実system CA/proxy/firewallは`live-e2e` blockedとしてQA_PLANへ明記し、focused結果を代替PASSにしない。

## リスクと停止条件

- `net/http.Transport`の公開APIだけでHTTP/1-only、SNI/certificate、TLS timeout、dial-only fallbackを同時に保持できない場合、`DialTLSContext`又はTLS設定無効化で迂回せず停止する。
- public API化したtest seam、production root CA差替え、`InsecureSkipVerify`、DNS hostname dial、mixed answer部分許可、connection reuse/retryが必要になれば停止してMainへ報告する。
- 許可外の既存package/Wiki、external module、config/CLI、実network testを必要とする場合、又はerror/diagnostic non-leakを保てない場合は停止する。

## 未解決事項

- なし。

## main Agentレビュー

- [x] TASKの全AC-IDへ設計判断、パス、順序、失敗時の扱いを対応させ、条件本文を複製していない。
- [x] `DialTLSContext`のGo契約、DNS全answer検査、IP literal dial、hostname TLS verification、HTTP/1-only、keep-alive/retry境界を具体化した。
- [x] 設計観点と代替案を検討している。
- [x] QA_PLANがTASK-firstで独立作成されている。
- [x] `dependency-ready reconciliation`と完了経路preflightが完了している。
- [x] DEV開始を承認した。

安全契約変更でv2契約を選ぶ場合は、コメントを外して`safety_contract_version: 2`と予定パス・生成パスの配列を記録し、DEV前に`make task-preflight TASK=TASK-0045`を実行する。変更しない種別は空配列とし、通常の予定パスと生成パスを重複させない。独立計画レビューのPASSとMainの分類承認をフロントマターへ記録する。分類変更時はTask、PLAN、QA_PLANを再承認し、承認者と時刻を更新する。
