---
task_id: "TASK-0051"
change_class: "product"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "luna-xhigh"
approved_dev_profile_reason: "新規の受理済み一connection有限状態機械に限定し、既存Authority/Handlerを変更せず、net.Pipeとin-memory CA/fake依存によるhermetic integration testだけで検証できるため。"
approved_dev_profile_risk_signals: []
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T12:34:06Z"
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

# TASK-0051 PLAN

## AC対応

TASKの条件本文を再掲せず、`planning input packet`のAC-IDに設計を対応させる。

| AC-ID | 設計判断 | 変更パス | 実施順序 | 失敗時の扱い |
|---|---|---|---|---|
| AC-1 | `Authority`を`Issue(string) (tls.Certificate, error)`だけに狭めたinterface、`Rules{Authority, Handler}`、immutable `Session`、`New`、`Serve(context.Context, net.Conn)`を定義する。反射でtyped nilを拒否し、Sessionは二依存以外の長寿命stateを持たない。 | `tools/dev-agent-harness/internal/connectsession/` | 1 | invalid Rules、nil/zero/破損receiver、nil/typed-nil context/connはpanicせずfixed session errorへ畳む。公開Format/errorに依存詳細を含めない。 |
| AC-2 | CONNECT phaseは5秒とcaller deadlineの早い方を接続deadlineにし、16 KiB bounded readerでHTTP/1.1のrequest line/header終端まで一件だけ読む。exact authorityの二host、同一のCONNECT Host/RequestURI、空body、Hostに加えて任意の単一visible-ASCII `User-Agent`（256 byte以下）と任意の単一`Proxy-Connection: keep-alive`（大小文字不問）だけを受け、成功前の余剰byteも拒否する。 | `tools/dev-agent-harness/internal/connectsession/` | 2 | method/version/authority/header/framing、duplicate header、User-Agentのcontrol/過長、Proxy-Connectionの複数値・keep-alive以外、その他header、early-byte不正はAuthority/Handler前に固定wire 403（`Content-Length: 0`、`Connection: close`、空body）のみを一回書き、closeする。 |
| AC-3 | valid CONNECT後にtargetへ`Issue`を一回だけ呼び、成功時に固定200を一回書く。TLS serverはそのleafだけ、TLS 1.2 minimum、exact targetのServerName callback、ALPN `http/1.1`だけ、client certificate未要求で構成する。TLS phaseにも5秒/caller deadlineを適用する。 | `tools/dev-agent-harness/internal/connectsession/` | 3 | Issue、200 write、SNI/ALPN/version/handshake failureはretry、host/certificate fallback、追加HTTP診断をせずcloseする。200後はTLS外の403を絶対に作らない。 |
| AC-4 | TLS成功後のHTTP phaseは5秒/caller deadline内に`http.Server`ではない単発parser/response writerを使い、HTTP/1.1 request一件だけへcaller contextをそのまま設定し、injected Handlerを同期一回呼ぶ。request完了後に接続を閉じ、後続bytesを処理しない。 | `tools/dev-agent-harness/internal/connectsession/` | 4 | outer CONNECT、RemoteAddr、SNI、inner headerをSubject/contextへ変換せず、parser/handler/write failure、HTTP/2、pipelining、keep-alive、fallback/retryは失敗としてcloseする。 |
| AC-5 | `Serve`の先頭で所有権を引受け、deferで全経路のconn closeとdeadline解除を保証する。phaseごとのdeadlineは固定5秒とcaller cancellation/deadlineの早い方で更新し、panic recoveryもfixed error/closeへ収束させる。local parser/buffer/TLS stateのみを用いる。 | `tools/dev-agent-harness/internal/connectsession/` | 1–5 | stall、EOF、cancel、panic、dependency/write errorでgoroutine/connを残さず、target/header/SNI/peer/context/下位errorをwire response、error、Formatへ出さない。 |
| AC-6 | net.Pipe、in-memory CA、real `brokerhttp.Handler`、fake Exchange/context-only resolver、counting Authorityを使うrace integration suiteを追加する。READMEは有限sessionの範囲とlive E2E未確認境界だけを追記する。 | `tools/dev-agent-harness/internal/connectsession/`、`tools/dev-agent-harness/README.md` | 2–6 | fake PASSを実listener、OS identity、CA files/trust、実client/network/providerの証拠にせず、既存package変更、socket bind、外部networkが必要なら停止する。 |

## 関連Wikiと判断

- REF-1の`brokerhttp.Handler`はTLS後のstrict HTTP/1.1 requestをcontext-only resolverとExchangeへ一回渡す唯一の入口である。Sessionはrequest contextを透過するだけで、Host/SNI/peerから主体を作らない。
- REF-2の`proxyca.Authority.Issue`は二hostのleaf発行だけを所有する。Sessionはhostを正規化せず、CONNECT targetとSNIのexact一致を確認して一leafを一回だけ使う。
- REF-3/4に従い、custom CA対応clientだけを将来対象とし、generic CONNECT、upgrade、HTTP/2、複数request、listener/identity/trust lifecycleを本Taskへ入れない。
- WikiはDEV許可パスではない。再利用可能な知識が生じる場合だけMainが既存正本への同化を所有する。

## 補足設計

### 責務・境界・不変条件

- `Serve`は一connectionを完全所有する有限状態機械である。CONNECT read/validate → `Issue(target)`一回 → fixed 200 → host-bound TLS handshake → HTTP/1.1 request一回 → Handler一回 → response完了 → close の順序を崩さない。
- 固定limitはCONNECT header 16 KiB、CONNECT/TLS/HTTP各phase 5秒である。各phaseは`min(caller deadline, now+5s)`をconnへ設定し、cancel時はconn closeでread/write/handshakeを解除する。TLS後HTTP本文上限・構造検査は既存Handlerへ委譲する。
- CONNECT parserは最小manual parserとし、正常化、folding、duplicate許容をしない。request lineは`CONNECT host:443 HTTP/1.1`、Hostは同じhost:portとし、追加で単一のvisible-ASCIIかつ256 byte以下の`User-Agent`と、sole valueが大小文字不問`keep-alive`の単一`Proxy-Connection`をそれぞれ任意に許す。重複、複数値、他値、その他header、body/framing/upgrade/proxy auth又はheader後のpre-TLS byteをfail closedにする。
- TLS configは`MinVersion: TLS1.2`、`NextProtos: ["http/1.1"]`、`ClientAuth: NoClientCert`とし、ServerName callbackでSNIをtarget完全一致以外拒否する。ALPN未選択又は異値も拒否する。authorityが返したleaf、context、buffersは一session以外へ出さない。
- TLS後はrequest contextを`context.WithCancel`等で値を置換せずcaller context由来に保つ。server listener/default handler/`http.Server`のkeep-alive machineryを使わず、connection closeを明示したresponse writerで一requestだけを完結する。

### 代替案と不採用理由

- generic CONNECT tunnel又はSNIだけで上流を選ぶ案はTLS後のbroker HTTP policyを迂回するため採用しない。
- `http.Server`を接続ごとに使う案はkeep-alive、複数request、HTTP/2の既定surfaceを持ち込むため採用しない。
- `Issue`を200後に行う案、SNI不一致で別leafを発行する案、handshake failureへHTTP responseを混ぜる案はhost bindingとwire phaseを曖昧にするため採用しない。
- RemoteAddr/CONNECT/Host/SNIからSubjectを作る案はOS identity boundaryを偽装するため採用しない。

### 移行・互換性

- 新規`internal/connectsession`とREADMEだけを変更する。`proxyca`、`brokerhttp`、Exchange/Transaction/Forwarder/Policy/transport、config/CLI/service/build/generated artifactを変更しない。
- 追加＋削除は1,000行以下、planning/candidate/completionの3 commit経路を守る。candidate識別子はHANDOVERだけで管理し、PLANへdigest又は重複candidate情報を転記しない。
- 実listener/OS identity/CA file/trust/real client/Internet providerはlive E2E blockedのまま残す。新しいgate/check/processは追加しない。

## 変更予定

| パス | 変更内容 |
|---|---|
| `tools/dev-agent-harness/internal/connectsession/` | Authority interface、immutable Session、strict bounded CONNECT parser、fixed deny/established write、host-bound TLS、single-request HTTP/1.1 handoff、deadline/close/panic管理、net.Pipe hermetic race integration testsを追加する。 |
| `tools/dev-agent-harness/README.md` | CONNECT→TLS→single broker HTTP requestのone-connection責務、strictな非対象面、live E2E未実施境界を追記する。 |

## 実装手順

1. interface、Rules、Session、fixed error/Format、typed-nil validationとall-path close/deadline helperを定義し、5秒phase limitと16 KiB CONNECT limitを定数化する。
2. strict CONNECT byte parser、two exact host binding、fixed 403/200 responseを実装し、invalid inputがAuthority/Handlerの前で一回のdeny/closeになるtestを置く。
3. Issue-before-200、one leaf、TLS 1.2/SNI/ALPN/client-cert禁止のhandshakeを実装し、200後TLS failureがwire診断を追加しないことをtestする。
4. TLS connection上でsingle HTTP/1.1 parser/response writerを実装し、caller contextをrequestへそのまま渡してreal Handlerを一回だけ呼ぶ。後続request/keep-alive/HTTP2を受理しない。
5. net.Pipe + in-memory CA、real Handler/fake Exchange/context resolverで成功・拒否・cancel/stall/panic/parallel isolationをrace検証し、READMEを境界に合わせる。
6. candidate前に`go test -count=1 -race ./internal/connectsession`、harness `make check`、`make distcheck`、README変更時のTask worktree `make lint-docs`、`git diff --check`、許可pathと1,000行上限を確認する。製品差分だけをcandidateとして一回固定後、candidate launcherのroot `make check`を一回実行する。

## 検証計画

- `go test -count=1 -race ./internal/connectsession`でRules/receiver/typed nil、両hostのCONNECT→200→TLS→real Handler success、exact fixed 403/200と空body、Issue一回を検出する。
- net.Pipe bytesでmethod/version/RequestURI/Host不一致、Host/User-Agent/Proxy-Connectionの重複、visible ASCII外又は256 byte超のUser-Agent、sole case-insensitive `keep-alive`以外/複数値のProxy-Connection、その他header、body/content length/transfer/trailer/upgrade/proxy authorization、CONNECT後early/pipelined bytesがAuthority/Handler前に拒否されることを確認する。単一の許可User-Agent及び許可Proxy-Connectionは両host成功fixtureで受理を確認する。
- in-memory CA clientでTLS 1.1、SNI mismatch、ALPNなし/不一致、certificate/handshake failureを確認し、200後に追加のHTTP response、retry、別host/leafがないことを確認する。
- real `brokerhttp.Handler` + fake Exchange/context-only resolverで、caller contextのみがresolverへ届き、outer/inner自己申告値でSubjectが変わらず、Handler/Exchangeが一request一回だけであることを確認する。
- cancellation、deadline stall、EOF、handler panic/write failure、parallel sessionsを確認し、all-path close、goroutine残留なし、response/context/certificate/buffer/deadlineの混線なしをrace detectorで検出する。実socket bind、OS trust、外部networkは使用しない。
- candidate前に上記package command、`make check`、`make distcheck`、README変更時の`make lint-docs`、diff/path/line limitを実行し、candidate後にroot `make check`を一回だけ実行する。live E2EはblockedのままQA_PLAN/post-merge確認へ残す。

## リスクと停止条件

| リスク | 抑制/検出 | 停止条件 |
|---|---|---|
| CONNECT authority、SNI、leaf、TLS後Hostの解釈差で別hostへ到達する | exact literal binding、Issue一回、SNI callback、real Handler fixtureで層別negative test | normalization、wildcard/fallback、generic host/port、SNI-only許可が必要なら停止する。 |
| HTTP smuggling/early byte又はkeep-aliveで一connection複数requestになる | 16 KiB bounded strict parser、pre-TLS余剰拒否、single HTTP parser、close-after-response test | chunked/upgrade/pipelining/HTTP2/connection reuseを受理する必要があれば停止する。 |
| cancellation/stall/panicでFD/goroutineが残る、又は200後にwire診断を漏らす | phase min-deadline、cancel-close、defer close/recovery、net.Pipe stall/panic test | listener lifecycle、async handler、追加error response又は安全にclose不能な経路が必要なら停止する。 |
| hermetic証跡を実配置/identity/trust受理へ過大化する | READMEの適用限界、net.Pipe/in-memory dependency限定、live E2E blockedを維持 | CA file/trust、OS peer identity、real client/network、service wiringが必要なら別Taskへ戻す。 |

## 未解決事項

- なし。

## main Agentレビュー

- [x] TASKの全AC-IDへ設計判断、パス、順序、失敗時の扱いを対応させ、条件本文を複製していない。
- [x] strict CONNECT → host-bound TLS → context-preserving single HTTP/1.1 request、fixed deadline/limit、close ownershipを具体化している。
- [x] 設計観点と代替案を検討している。
- [x] QA_PLANがTASK-firstで独立作成されている。
- [x] `dependency-ready reconciliation`と完了経路preflightが完了している。
- [x] DEV開始を承認した。

安全契約変更でv2契約を選ぶ場合は、コメントを外して`safety_contract_version: 2`と予定パス・生成パスの配列を記録し、DEV前に`make task-preflight TASK=TASK-0051`を実行する。変更しない種別は空配列とし、通常の予定パスと生成パスを重複させない。独立計画レビューのPASSとMainの分類承認をフロントマターへ記録する。分類変更時はTask、PLAN、QA_PLANを再承認し、承認者と時刻を更新する。
