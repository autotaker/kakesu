---
task_id: "TASK-0066"
change_class: "product"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "sol-high"
approved_dev_profile_reason: "Peer-bound CA retrieval changes both sides of a strict Unix control protocol and independently validates certificate material across a private-key boundary."
approved_dev_profile_risk_signals:
  - "public CA versus private-key authority boundary"
  - "strict one-request Unix control protocol and exact EOF framing"
  - "independent X.509/PEM validity checks with clock-dependent acceptance"
  - "copy isolation and diagnostic non-leak requirements"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-02T03:12:56Z"
classification_approved_by: "main-agent-sol-high"
classification_approved_at: "2026-08-02T03:12:56Z"
classification_approval_reason: "peer-bound control protocol、Authority公開interface、Agentが取得できるCA materialという外部観測可能な製品境界を変更するため。"
# safety_contract_version: 2
# safety_contract_planned_paths: []
# safety_contract_generated_paths: []
---

# PLAN: Agent proxy CA control client

## 根拠と分類

唯一の要求根拠は`TASK.md`の`Planning input packet`である。既存peer-bound Unix socketのcontrol protocol、`connectsession.Authority`の公開interface、Agentが取得できる値、およびREADMEの外部観測可能な境界を変更するため、`change_class`は`product`とする。DEVはpacket指定の`sol-high`とする。本PLAN自体には独立PLAN reviewを設けず、MainがTASK-firstで独立作成された`QA_PLAN.md`とともに軽く確認してから承認を記録する。

candidateはpacketで許可された6パスだけを変更し、追加・削除を合わせて約800〜1,100行に収める。CA生成・rotate・reload、disk/trust-store/file/launcher/environment/Git設定、private key又はbroker credential directory、token/handle/repository/subjectの公開、TCP/別socket/override、cache/retry/redirect、chain又は汎用certificate endpoint、dependency/Schema/Kakesu runtime/generated file/live stateは追加・変更しない。CONNECT/TLS leaf、Capability Registry、Issue/Revoke、Git helper、provider forwarding、socket activation/peer bindingの意味も変更しない。実OS socket/別UID、Git/libcurl trust、GitHub/OpenAI/DNS/TLS/systemd/VPSは本candidateのhermetic PASSで代替しない。

## end-to-end設計

1. `connectsession.Authority`を、既存の`Issue(string)`に加えて`PublicCertificatePEM() []byte`を持つ最小の公開certificate accessorへ広げる。productionの`proxyca.Authority`は既存のfresh-copy accessorで満たし、test fakeも公開PEMだけを供給する。control routeはこのinterface以外のauthority state、private key、bundle/pathを受け取らない。
   - `GET /v1/proxy-ca HTTP/1.1`は、固定`Content-Length: 0`だけを持つzero-body control requestとして追加する。Host、Content-Type、Connection、Transfer-Encoding、query/fragment、duplicate/unknown header、noncanonical length、body、header末尾後のearly byte、二つ目のrequestは既存403/closeへ畳む。
   - canonical GETだけがTLS/CONNECT/inner handler/control Issue/Revokeへ進まず、一接続一操作でauthority accessorを一回だけ呼ぶ。nil interface/typed-nil、nil/empty/malformed accessor値、parser/validity failureは同じ固定403/closeであり、値や下位errorをwire又は公開errorへ出さない。
   - control header parserはroute kindを明示して、POST IssueのJSON/body規則とDELETE Revokeのzero-body規則を保ったまま、GETだけのzero-body規則を分岐させる。既存`hasControlExtraByte`とphase deadline/close所有を再利用し、寛容な`net/http` parsingやkeep-aliveを導入しない。

2. serverはPEMをresponseへ書く前に、その値をtransportに渡さない局所validatorで検査する。validatorは1〜4,096 bytes、headerなしの唯一の`CERTIFICATE` PEM block、rest/trailing bytesなしを要求し、DERを`x509.ParseCertificate`する。さらにself-signature、ECDSA P-256 public key、`BasicConstraintsValid`/`IsCA`、`KeyUsageCertSign`、現在時刻が`NotBefore <= now < NotAfter`を全て要求する。
   - `time.Now`への直接依存は、package-private clock function/seamとして一点に閉じる。server testsはt.Cleanupで復元してvalid、not-yet-valid、expiredのcertをdeterministically判定し、productionではUTCの現在時刻だけを用いる。`proxyca`に新しい生成/状態APIやclock injectionを公開しない。
   - 成功時だけ固定wire `HTTP/1.1 200 OK`、`Content-Type: application/x-pem-file`、canonical decimal `Content-Length`、`Connection: close`、空行、validated PEM bytesを一回だけwriteする（header順も固定）。作成したresponse bytesはauthority返却sliceとaliasさせず、write前後のmutationが成功値又はauthority stateへ影響しないようコピーする。write/deadline failureは新しいpartial responseを補償・retryせずsession fixed failureで閉じる。

3. `controlclient`に、absolute/clean Unix socketを受ける公開CA取得関数（例: `ProxyCA(socketPath string) ([]byte, error)`）を加える。入力にrepository/handle/subjectやoverrideを持たせない。`GET /v1/proxy-ca HTTP/1.1\r\nContent-Length: 0\r\n\r\n`を全量writeし、existing private `dialControl` seam経由でtimeout contextを使い一回だけdialする。dial、write、read、closeに5秒の既存phase deadlineを設定し、dial/write/read/closeの失敗はretry/fallbackなしに固定`ErrControl`とnil valueへ畳む。
   - clientは既存server parserをimportせず、自身のbounded response reader/validatorを持つ。最大headerは既存1,024 bytes、bodyは1〜4,096 bytes、全read limitはheader + CA max + one extra byteに固定し、完全なresponse後のEOFを必須にする。chunked、multiple responses/body、early EOF、余分byte、過大header/bodyを受理しない。
   - 成功に必要な唯一のresponseは、固定順序の`HTTP/1.1 200 OK`、`Content-Type: application/x-pem-file`、canonical `Content-Length`、`Connection: close`、その長さのbody、直後EOFである。1xx/204/3xx/403/5xx、header casing/whitespace/order/duplicate/extra、wrong length/type、body framing逸脱は固定拒否する。clientにも別実装のPEM/X.509 validatorとpackage-private clock seamを置き、serverと同じcertificate propertyを独立検証する。
   - 成功値はread bufferからfresh copyを返し、defer/closeやsubsequent caller mutationとaliasさせない。`ErrControl`とそのformat、stdout/stderrを持たないlibrary API、test diagnostic assertionsにはPEM、subject、socket path、request path、private block、下位errorを残さない。

4. regression-facing testsを既存6 path内へ閉じる。`connectsession` testsはreal in-memory `proxyca.Authority`でcanonical responseをparseし、certificate-only/CA properties、exact response bytes、authority accessor call count、closeを確認する。fake authorityはvalid public PEM、nil、malformed/certificate chain/trailing/private PEMを返してserver refusalとcontrol/handler未到達を確認する。
   - session negative matrixはGET method/path/query/HTTP version、missing/leading-zero/duplicate/extra Content-Length、Content-Type/Host/Connection/chunked/body/early/second requestと期限切れ・not-yet-valid/non-CA/no-CertSign/non-P256/non-self-signed valuesを403に固定する。同じ tests で既存 CONNECT、Issue、Revoke のresponse、controller call count、TLS handlerへの非到達を回帰確認する。
   - `controlclient` testsは`net.Pipe`とprivate dial seamでone dial、exact GET bytes、partial write、write/read deadline、close、body and caller mutation isolationを測る。response matrixはstatus、header order/case/whitespace/duplicate/extra、chunked、length/body mismatch、early EOF、extra bytes/second response、PEM size/block/header/rest/private key、parse/signature/CA/key/validityを壊し、常にnil/`ErrControl`かつnon-leakをassertする。clock seamsをparallelにしないか同期/cleanup復元してraceを避ける。

5. `egressservice` composition testのfake authorityを新interfaceに適合させ、factory graphが同じauthorityをSessionへ渡す現状を維持するだけにする。production composition、broker credential loading、socket configurationを広げない。READMEにはこのGETがpeer-bound socketからcertificate-only public CAを一回取得し、後続launcherがtrust fileを作る前段に限ることを記載する。private key/credentialsへの非到達、disk/environment/Git config/実client信頼が対象外であること、hermetic結果がlive deploymentを保証しないことを明記し、certificate、socket、subject、pathを例として載せない。

## AC対応

| AC-ID | 設計判断 | 変更パス | 実施順序 | 失敗時の扱い |
|---|---|---|---|---|
| AC-1 | Authorityをpublic PEM accessorだけに拡張し、route kindを明示したexact GET/zero-body parserで一操作を分離する。 | `internal/connectsession/session.go`、`session_test.go`、`internal/egressservice/service_test.go` | 1 | malformed request、extra byte、nil/invalid authority outputはcontroller/handler/CONNECTへ進めず既存固定403/close。 |
| AC-2 | server-local PEM/X.509 validatorとprivate clock seamを通過したfresh PEM copyだけをfixed 200 wireへ一回writeする。 | `internal/connectsession/session.go`、`session_test.go` | 1–2 | size/block/private/trailing/CA/key/signature/validityまたはwrite failureは成功bodyなしのfixed denial/session close。 |
| AC-3 | clientはsingle Unix dial、phase deadlines、strict bounded response/EOFとserver非共有のvalidatorを持つ。 | `internal/controlclient/client.go`、`client_test.go` | 3 | transport/framing/status/PEM/X.509不一致はretry/fallbackなしのnil + fixed `ErrControl`。 |
| AC-4 | authority、server response、client resultの各copyを独立させ、公開errorとtest assertionを非secret固定値にする。Issue/Revoke/CONNECT suitesも維持する。 | `internal/connectsession/session.go`、`session_test.go`、`internal/controlclient/client.go`、`client_test.go`、`internal/egressservice/service_test.go` | 1–4 | alias、PEM/path/subject/lower-error露出、既存route回帰はcandidate failureとして取り込まない。 |
| AC-5 | packet許可の6 pathと800〜1,100行に限定し、focused race/harness/root/scope検査をcandidateへ結び付ける。 | 許可済み6パス | 1–5 | 行数・scope・対象外逸脱又は検査FAILはMainへ戻し、live環境のPASSに読み替えない。 |

## 変更予定

| パス | 変更内容 |
|---|---|
| `tools/dev-agent-harness/internal/connectsession/session.go` | `Authority` public PEM accessor、exact GET parser/route、server-local bounded CA validator/clock seam、fixed certificate responseを加える。 |
| `tools/dev-agent-harness/internal/connectsession/session_test.go` | canonical/negative GET wire、authority failures、copy/non-leak、validity seam、CONNECT/Issue/Revoke regressionを追加する。 |
| `tools/dev-agent-harness/internal/egressservice/service_test.go` | fake authorityを拡張interfaceへ適合させ、既存graph wiring不変を確認する。 |
| `tools/dev-agent-harness/internal/controlclient/client.go` | CA fetch API、single dial/deadline/close、bounded strict PEM response parser、independent CA validator/copyを実装する。 |
| `tools/dev-agent-harness/internal/controlclient/client_test.go` | `net.Pipe` wire/partial I/O/deadline/close、response and certificate rejection matrix、copy/non-leak/clock seamを追加する。 |
| `tools/dev-agent-harness/README.md` | public-CA retrieval boundary、後続launcherとの分離、private/disk/config/live対象外を記載する。 |

## 検証計画

DEVは`net.Pipe`、in-memory ECDSA P-256 fixture、fake Authority、package-private clock/dial seamsだけを使う。serverではcanonical GET→one accessor→exact 200/close、request framing全拒否、authority output/certificate property全拒否、copy isolation、CONNECT/Issue/Revoke non-regressionを検証する。clientではexact request→one dial→deadline→strict 200 body/EOF→fresh copyを確認し、partial I/O、deadline/close failure、全status/header/framing/EOF/PEM/X.509 invalid matrixをnil/`ErrControl`へ落とす。failure strings/stdout/stderrにはPEM、subject、socket、path、lower errorがないことを直接assertする。

candidate固定前にDEVは少なくとも次を実行する。

```sh
cd tools/dev-agent-harness && GOCACHE=$PWD/.build/go-cache go test -race ./internal/connectsession ./internal/controlclient ./internal/egressservice
cd tools/dev-agent-harness && make check
make task-check TASK=TASK-0066
git diff --check
```

root `make check`はcandidate gateが固定済みworking bytesへ一回だけ実行する。DEV、Reviewer、QAは重複実行せず、そのcandidate-bound結果をHANDOVER/REVIEW/QAで監査する。

QA_PLANはcaseごとに`focused-rerun`、`evidence-review`、必要な`live-e2e`を理由付きで確定する。高リスクのCA/private-key boundary、strict protocol、clock validity、non-leak、copy isolationはcandidate-bound negative testsとrace evidenceの独立監査だけで弱めず、証跡不足なら`evidence-review` PASSにしない。実OS socket/permission、実Git/libcurl trust と外部 TLS/DNS/provider は承認済み環境かつcleanupがなければ`live-e2e`をblockedのままにし、hermetic PASSで置換しない。

## リスクと復旧

- public accessorがprivate authority stateへの拡張入口になるリスクは、`[]byte`のcertificate-only fresh copyだけをinterfaceに置き、bundle/key/path APIを追加しないことで抑える。
- 期限外又はCAでない証明書をserver/clientの片側が受理するリスクは、共有parserではなく同じ全propertyを別validatorとclock seamで検証することで抑える。
- permissive HTTP parsing、extra bytes、partial responseが成功扱いになるリスクは、固定request/response bytes、body/header上限、Content-Length一致、直後EOF、net.Pipe negative matrixで抑える。
- PEM/subject/socket/path/lower errorの露出は、fixed errors、copy-only API、response前検証、failure output assertionsで抑える。
- private test seamがproduction overrideになるリスクは、unexported package-local seamをtest中に復元するだけとし、CLI/environment/config/socket overrideを追加しないことで抑える。

復旧時はこの6許可パスのcandidate製品差分だけを戻す。新しいCA/key/trust file、credential directory、cache、runtime configuration、socket、外部操作を作らないため追加cleanupはない。復旧後は上記focused race suite、harness/root `make check`、`make task-check TASK=TASK-0066`、`git diff --check`を再実行する。

## 引き継ぎ条件

DEVはMain承認済みPLANと独立QA_PLANの後、6許可パスだけで同一candidateを一回固定する。ReviewerとQAは同じcandidateを相互のPASSを待たず独立評価する。Mainだけがstage/commit/merge/push、completion gate、`--no-ff --no-commit`検査、main統合と必要な環境依存確認を所有する。candidateにprivate key、broker credential directory、secret/path diagnostics、launcher/config mutation、dependency/Schema/generated file/live stateを含めない。

## 未解決事項

- なし。packetが固定するsocket boundary、Authority public PEM accessor、response security propertyを前提にする。

## main Agentレビュー

- [x] TASKの全AC-IDへ設計判断、パス、順序、失敗時の扱いを対応させ、条件本文を複製していない。
- [x] public certificate-only accessor、GET zero-body framing、server/client独立validation、clock seam、copy/non-leak、既存route regressionを固定している。
- [x] QA_PLANがTASK-firstで独立作成されている。
- [x] `dependency-ready reconciliation`と完了経路preflightが完了している。
- [x] DEV開始を承認した。
