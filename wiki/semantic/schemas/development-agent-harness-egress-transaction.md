---
kind: schema
title: Development Agent Harness Egress Transaction
---

# Development Agent Harness Egress Transaction

## 問い

HTTPの許可判断、Opaque ケイパビリティの一回消費、実認証情報の置換を、解釈のずれや秘密情報の逆流なしにどう接続するか。

## スコープを一度だけ導出する

`egresspolicy.Policy.Evaluate`は、HTTP許可判断と同じ評価からプロバイダー、リポジトリ、操作、宛先ホストを返す。外向き通信トランザクションはURLを再解析せず、この正規スコープをそのままケイパビリティの完全一致検証へ渡す。

GitHub REST readでは`github`、`owner/repo`、`github-rest-read`、`api.github.com`、OpenAI Responses textでは`openai`、リポジトリなし、`openai-responses-text`、`api.openai.com`となる。拒否時は空スコープと固定errorだけを返す。従来の`Authorize`は同じ評価へ委譲し、既存のdecision/error契約を維持する。

## 秘密処理へ到達する順序

`egresstransaction.Transaction.Execute`は、次の順序を固定する。

1. HTTP allowlistを評価する。
2. プロバイダーごとに許したAuthorization値を一つだけ抽出する。
3. 正規スコープとAgent主体を完全一致させてケイパビリティを一回消費する。
4. trusted resolverから実認証情報を一回だけ取得し、長さとvisible ASCIIを検証する。
5. 上流用Bearerへ置換したリクエストをtrusted Forwarderへ同期的に一回だけ渡す。

policy又はAuthorizationの拒否ではケイパビリティ、resolver、Forwarderへ到達しない。ケイパビリティ拒否ではresolverとForwarderへ到達しない。resolver又はForwarderが失敗しても消費済みケイパビリティを戻さず、同じ試行を再実行しない。

## 実認証情報の受渡し境界

Credential-bearingな`PreparedRequest`は、Transactionへ注入されたbroker内のtrusted Forwarderへの同期呼出中だけ渡す。`Execute`は値を返さず、TransactionもPreparedRequest、元のAuthorization、Opaque handle、実認証情報を保持しない。本文は独立コピーを渡す。

## 上流 HTTPS transportとの接続

ForwarderがGitHub又はOpenAIへ送る最初の接続境界には、固定allowlistの`http.RoundTripper`を注入する。transportはoriginと`Request.Host`をDNS前に照合し、resolverの全answerを検査して、一件でもunsafeなaddressを含む集合を拒否する。安全なanswerだけの場合も、検査済みIP literalの443番だけへdialし、元hostnameをTLSのSNIと証明書検証に使う。

このtransportは一request一接続で、環境proxy、keep-alive、自動compression、HTTP/2、redirect、retryを持たない。TCP接続失敗時だけ未使用の検査済みIPへ進めるが、TLS handshake又はHTTP送信開始後は再dialしない。TLS 1.2未満又はHTTP/1.1以外、失敗response、下位のDNS/TLS/socket detailは公開せず、失敗時のbodyはtransportがcloseする。

## Forwarderで再検証して縮退する応答

`internal/upstreamforwarder`は`PreparedRequest`を受けた後、transportへ渡す前に同じpolicyでrequestを再評価し、正規scopeの完全一致と上流Bearerを再検証する。GitHub GET/HEADは空Content-Typeかつ空本文だけに狭め、検証済みrequestを注入RoundTripperへ一回だけ送る。上流headerは実Authorization、固定AcceptとUser-Agent、およびOpenAIのContent-Typeだけに限定し、Agent由来headerやopaque capability handleを転記しない。本文はcaller所有sliceから独立copyする。

成功responseはsize上限内で完全にread、検証、closeしてから、status、正規JSON media type、独立した本文だけをrequest単位sinkへ一回渡す。HEAD/204 responseは空本文だけを受理し、provider error本文と上流headerはAgent側へ返さない。2xx以外、想定外media type、上限超過、UTF-8/JSON不正、read/close/timeout、sinkの各失敗はfail closedにする。

## 呼出し単位のExchange合成

`brokerexchange.Exchange`は、検証済みの依存と上限だけをimmutableに保持し、各`Do`呼出しごとにprivate capture sink、`upstreamforwarder.Forwarder`、`egresstransaction.Transaction`を新規に合成する。これにより既存のpolicy、Authorization、capability、credential、response検査の意味を再実装せずに委譲する。

成功時だけ、captureした縮退responseの独立copyを返す。いずれかの段階が失敗した場合はzero responseと固定`exchange-denied`だけを返し、上流失敗の詳細を公開しない。呼出し間でsinkやresponse本文を共有しないため、並行実行したresponseが相互に混入しない。

policy又はAuthorizationの拒否はcapabilityを消費しない。capability消費後にresolver、transport、Forwarder、captureのいずれかが失敗しても消費をrollbackせず、同一`Do`による上流試行は一回だけとする。

この性質だけでは秘密情報ストア、実network上のprovider受理、Agent側proxy、response writer又はauditを証明しない。実GitHub/OpenAI、実認証情報、Internet DNS/TLS/system trust、Agent proxy/response writerはlive E2Eで別途確認する。

## TLS終端後のHTTP入口

`brokerhttp.Handler`はTLS終端済みのHTTP/1.1 origin-form requestだけを受け、trustedな`SubjectResolver`へはrequest contextだけを渡して、既存の`Exchange`を一回だけ呼び出す。Host又はmethodをresolverの主体解決へ渡さず、provider host/methodの意味、policy、capability、credentialの検証を入口で再実装しない。

Exchangeへ到達する前に、origin-form、構造的に妥当なmethodとauthority、query/raw pathなし、既知のContent-Length、transfer/trailer/upgradeなしを検査する。CONNECTとこのHTTP構造条件に反するrequestは入口で拒否する。resolver、Exchange、又はresponseの検証に失敗した場合も、固定headerだけの空403へ縮退し、retry又は診断本文を生成しない。

成功時はExchangeが返した2xx responseだけを、縮退済みContent-Typeとbody、固定`Cache-Control: no-store`と`X-Content-Type-Options: nosniff`、正確な`Content-Length`で返す。response write失敗後に上流処理を再試行しない。Handlerはlistener/TLS終端、production identity resolver、実provider client、実credential、DNS/system trust、Agent network namespaceを所有せず、それらはlive E2Eの別境界である。

## CONNECTからHTTP handoffまでの一接続境界

`connectsession.Session`は受理済みの一connectionだけを所有し、strict CONNECT、host-bound TLS終端、`brokerhttp.Handler`への単一HTTP/1.1 request handoff、単一response、closeを有限state machineとして順に行う。CONNECT authorityとSNIは`api.github.com`又は`api.openai.com`への完全一致に限り、CAが発行するleafと同じallow surfaceを保つ。

CONNECTでは限定headerだけを受理し、HTTP phaseへ入る前の余分なbyte、framing又は未許可headerを拒否する。TLSはTLS 1.2以上、SNI、HTTP/1.1 ALPNを要求する。TLS終端後は最大一requestをhandlerへ渡し、keep-aliveを許可しない。CONNECT、TLS、HTTPの各phaseには5秒とcallerのより早いdeadline/cancelのうち早い方を適用し、失敗は固定された非漏えい応答へ縮退してconnectionをcloseする。

Sessionはcaller contextをそのままhandlerへ渡すが、CONNECT header、RemoteAddr、SNI又はinner HTTP headerから主体を生成しない。handlerはdeadline/cancelに協調してreturnするtrusted dependencyである。任意の非協調callbackを別goroutineでtimeoutさせるとcallbackが戻らない時にgoroutineを確実にleakするため、Sessionはその強制停止を保証しない。

## listenerでの主体束縛と接続ライフサイクル

`brokerlistener.Server`は注入済みの`net.Listener`を所有し、trustedな`PeerBinder`が得たSubjectを検証し、独立copyをprivate context keyへ束縛してから、一接続だけを扱う既存`Session`へ渡す。`Resolver`はこのprivate contextからのみSubjectの独立copyを一回返す。公開setter、`RemoteAddr`、CONNECT/TLS/HTTP headerによる主体の補完はしない。

productionの`PeerBinder`はlistener lifetimeごとに、ownerが検証して固定した一つのSubjectと期待UIDだけをimmutableに保持する。`Bind`はconcreteな受理済み`*net.UnixConn`からLinux kernelの`SO_PEERCRED` UIDを一回だけ同期取得し、contextの前後確認を通り期待UIDと完全一致した場合だけfresh Subject copyを返す。wrapper、TCP、nil/corrupt connection、不正rules、credential read失敗、UID不一致、non-Linuxは、空Subjectと固定拒否に縮退する。接続内容、socket path、PID、GIDからAgentInstanceID又はWorkspaceIDを導出せず、複数SubjectをUIDから引くmapも持たない。Binder自体はconnectionをcloseせず、Serverがlifecycleを所有する。

SubjectはUID正数、identifierは1〜128 byte、先頭は英数字、残りは英数字又は`._-`に制限する。binderが拒否したSubject、無効なSubject、binder又はSessionのerror/panicは当該connectionをcloseして次のacceptを継続する。

Serverは1〜64の同時接続上限をslot-before-Acceptで適用し、slot取得後にもcancelを再確認してからAcceptする。caller cancel/deadline又はunexpected Accept failureではlistenerを一回だけcloseし、全active connection contextをcancelして、協調するBinder/Sessionのreturnをdrainする。任意の非協調callbackを強制停止したり、timeout用goroutineでleakを隠したりはしない。

## systemd socket activationの所有境界

固定Unix socketのbind、owner/group/mode、停止時cleanupはsystemd socket unitだけが所有する。プロセス内の`socketactivation.Receiver`は、systemdから継承したFD 3を一回だけ受け取り、`brokerlistener.Server`へ渡すlistenerへ変換する境界であって、bind、chmod、chown、unlink、fallback、retry、cache、goroutine又はlogを持たない。

Receiverはcanonicalな`LISTEN_PID`、`LISTEN_FDS=1`、`LISTEN_FDNAMES=egress`だけを受理する。環境を消去してからFDを変換し、original FDは変換後にcloseする。concreteな`*net.UnixListener`かつ固定pathでなければ拒否し、変換後又は検証中に失敗したlistenerもcloseするため、activationは一回限りで再利用されない。

Linuxでは、current non-root EUID、`broker:agent 0710`のruntime directory、そのdirectory FDから`openat(O_PATH|O_NOFOLLOW)`したsocket node、固定listener pathを照合する。nodeは`broker:agent 0660`のUnix socketでなければならない。上限付き`/proc/self/environ` snapshotでraw `LISTEN_*`出現数も数え、canonical値に見えても同名keyの重複を拒否する。non-Linuxでは代替経路なしにfail closedとする。

## 適用限界

このトランザクションはin-memoryの認可接続コアであり、実認証情報の読取・生成、GitHub App token交換、OpenAI key管理、実UID分離、DNS、上流通信、監査、永続化を実装しない。CONNECT/TLS/HTTPの一接続処理、Agent向けTLS interceptionのCA検証とhost限定leaf発行、受理済みUnix connectionのLinux peer UID照合、systemd継承FDの受領は別境界である。実systemd managerによるFD 3配送、実broker/agent別UID/GID、socket permission/connect、停止時cleanup、network namespace、実client/VPS、実GitHub/OpenAI、実Internet DNS/system trust、CA file lifecycle/rotate/trust install、実配置のrestart/rollback/cleanupはなおlive E2Eで確認する。前記transport、CA、Session、listener、peer binder又はsocket activationのhermetic testとcross-compileは、これらのlive E2E又は認証情報非露出の証明にならない。

## 関連

- [TASK-0041 HANDOVER](../../../tasks/TASK-0041-dev-agent-egress-transaction/HANDOVER.md)
- [TASK-0045 HANDOVER](../../../tasks/TASK-0045-dev-agent-upstream-transport/HANDOVER.md)
- [TASK-0047 HANDOVER](../../../tasks/TASK-0047-dev-agent-upstream-forwarder/HANDOVER.md)
- [TASK-0048 HANDOVER](../../../tasks/TASK-0048-dev-agent-broker-exchange/HANDOVER.md)
- [TASK-0049 HANDOVER](../../../tasks/TASK-0049-dev-agent-broker-http-handler/HANDOVER.md)
- [TASK-0051 HANDOVER](../../../tasks/TASK-0051-dev-agent-connect-session/HANDOVER.md)
- [TASK-0052 HANDOVER](../../../tasks/TASK-0052-dev-agent-listener/HANDOVER.md)
- [TASK-0055 HANDOVER](../../../tasks/TASK-0055-dev-agent-linux-peer-binder/HANDOVER.md)
- [TASK-0056 HANDOVER](../../../tasks/TASK-0056-dev-agent-systemd-socket-activation/HANDOVER.md)
- [Development Agent Harness Egress Policy](development-agent-harness-egress-policy.md)
- [Development Agent Harness Capability Registry](development-agent-harness-capability-registry.md)
- [Development Agent Harness Proxy CA](development-agent-harness-proxy-ca.md)
