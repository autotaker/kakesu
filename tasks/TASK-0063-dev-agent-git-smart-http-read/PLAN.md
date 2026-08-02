---
task_id: "TASK-0063"
change_class: "product"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "sol-high"
approved_dev_profile_reason: "credential-bearing protocol/transport risk: opaque Basic capabilityを実tokenへ置換し、Git Smart HTTPのhost・operation・binary response境界を既存egress graph全層で同時に変更するため。"
approved_dev_profile_risk_signals:
  - "credential-bearing protocol/transport"
  - "opaque capability to HTTP Basic replacement"
  - "Git Smart HTTP method/path/query/media-type boundary"
  - "pinned upstream transport and binary response handling"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-02T01:20:43Z"
classification_approved_by: "main-agent-sol-high"
classification_approved_at: "2026-08-02T01:20:43Z"
classification_approval_reason: "Git Smart HTTP readの外部観測可能なrequest/response、authorization置換、許可host、pinned transportを変更する製品変更であるため。"
---

# PLAN: Git Smart HTTP readを限定許可する

## 根拠と分類

本計画の唯一の要求根拠は`TASK.md`の`Planning input packet`である。既存peer認証済みegress socketにおける許可host、認証方式、Git Smart HTTP request/response、upstream接続の外部観測可能な振る舞いを追加するため、`change_class`は`product`とする。提案するDEV profileは`sol-high`であり、MainがPLANと独立したTASK-first QA_PLANの意図、scope、受け入れ経路を確認してから承認を記録する。本PLANの独立PLANレビューは設けない。

candidateは下記の20許可パスだけとし、おおむね1,000〜1,400行に収める。新listener/socket/unit、TCP入口、Agent側credential保存・helper・launcher・Git config/environment注入、Registry永続化、cache、retry、依存、Kakesu本体runtime/Schema/build、実DNS/TLS/GitHub App token、live配置を追加又は変更しない。`git-receive-pack`、write capability、Approval、redirect、LFS、submodule、archive、GitHub REST/Web/GraphQL writeは明示的に対象外である。

## end-to-end設計

1. policyをGitHub REST readと区別するGit Smart HTTP read operationへ拡張する。
   - `egresspolicy`は既存repository allowlistと同じ正規`owner/repo`照合を使い、`github.com`に対して二つのcanonical requestだけを評価する。discoveryはGET `/owner/repo.git/info/refs?service=git-upload-pack`、serviceはPOST `/owner/repo.git/git-upload-pack`である。
   - raw URLをallow根拠として保持し、HTTPS、host（`:443`を含む既存のcanonical表現だけ）、ASCII、userinfoなし、fragment/percent-encodingなし、正規path segment、空/`.`/`..`なしを要求する。discovery以外のquery、重複・空・欠落・順序違いのquery、別service、余分path、repository suffixの逸脱は拒否する。
   - request種別ごとに厳密なContent-Type、body有無及びpolicyの既存`MaxBodyBytes`上限を検査する。discoveryはbody/Content-Typeなし、upload-packはGit service request media typeと非空・上限内のbinary bodyだけとし、JSON解釈は適用しない。成功時だけ新しい`github-git-read` operation、repository、`github.com`を一つのcanonical scopeとして返す。REST/OpenAIの判断値とその既存入力規則は変更しない。

2. capability発行を明示Git read selectorへ分離する。
   - `capability`はGitHub REST readとGit readを別operation/hostとして表現し、scopeの発行・consume完全一致を維持する。Git readは`github.com`、REST readは`api.github.com`のままとし、Git providerだけからoperationを暗黙選択しない。
   - `capabilitycontrol`はpeer-derived subject、同じrepository allowlist、固定TTL 5分、uses 1を保持し、controlの最小入力にGit read selectorを追加する。入力からsubject、TTL、uses、hostを採用せず、repositoryがallowedかつcanonicalでなければRegistryへ到達しない。既存GitHub REST selectorとOpenAI selectorの発行意味を保つ。
   - `connectsession`の既存control JSON parserはGit read selectorを一つの厳密な許可値として通し、unknown/duplicate/欠落field、subject/scope/TTL/uses/host自己申告を従来どおり拒否する。control wireの一接続一操作、size上限、固定失敗応答、closeを変更しない。

3. TLS終端後のrequestをraw queryを失わずtransactionへ渡す。
   - `brokerhttp`はorigin-formのcanonical pathに加え、policyが必要とするdiscoveryの唯一のraw queryだけをinner request URIから正確に取り出し、`https://` URLへ再構成する。他のquery、`?`だけ、percent encoding、fragment、absolute form、host/path不一致はhandlerで拒否する。
   - handlerはGit requestのbinary bodyを既存上限内で一度だけ独立copyし、responseについてGit operationに許される縮退済みContent-Typeをそのまま返す。REST/OpenAIのJSON-only response contractと固定no-store/nosniff、empty 403非漏洩を維持する。

4. capabilityを正確なHTTP Basic credentialへ一回だけ置換する。
   - `egresstransaction`はpolicy評価後にscopeでauthorization grammarを選択する。Git readだけはAuthorization一値のstrict HTTP Basicをbase64 canonical formでdecodeし、usernameが正確に`x-access-token`、passwordが正規`cap_...`である場合だけhandleとして扱う。Bearer又は`token`、空/余分colon、無効base64、control/whitespace、別username、複数headerをconsume前に拒否する。
   - 成功consume後に同じRegistryのgrantとsubject/scope完全一致を確認してresolverを一回呼び、実credentialを`Basic base64("x-access-token:" + real-token)`へ組み立てる。Agent提供Basic値・handleをprepared requestへ転送せず、resolver失敗、credential形式不正、forwarder失敗時にrollback/retry/reissueしない。REST/OpenAIは既存Bearer/token extraction及びBearer upstream置換のままとする。

5. CONNECT、CA、forwarder、pinned transportでGit readだけを閉じた経路にする。
   - `connectsession`は`github.com:443`をCONNECT可能hostに加えるが、TLS SNI、ALPN HTTP/1.1、一request、一connection、timeout、early-byte及び通常CONNECTのstrict header契約を既存hostと同じく要求する。
   - `proxyca`は完全一致`github.com`だけに短命leafを発行し、wildcard・port・他hostを発行しない。
   - `upstreamforwarder`はre-evaluateしたGit scopeにだけBasic authorization、operation対応のAccept/Content-Type、binary bodyを渡す。redirect clientを導入せず、RoundTrip一回、success 2xx、response size上限、request methodとresponse状態の整合を必須にする。discoveryとupload-packの各々に対応する厳密なGit response media typeを検査し、JSON/UTF-8検査はGit responseへ適用しない。unexpected media type/status/body、sink失敗はdelivery前に固定失敗とする。
   - `upstreamtransport`は`github.com`を第三の固定hostとして一回DNS解決、global-unicast address検査、IP literal `:443` dial、元hostのSNI/certificate検証、TLS 1.2以上・HTTP/1.1、proxyなし、connection poolなし、接続後retryなしを既存二hostと同一に保持する。host/port/userinfo/opaque URL/redirect先への変更はfail closedとする。

6. READMEを実装済みの境界へ同期する。
   - Smart HTTP readが二つのcanonical operation、`x-access-token:cap_...`のAgent-facing Basic、broker内のみのreal-token Basic置換、single-use/5分/peer-bound issuance、push・helper/client・live環境の対象外を明確にする。
   - hermetic testが実GitHub、実credential、DNS/TLS、NSS/別UID、systemd/VPS、実`git clone/fetch/pull`の受理を保証しないことを維持する。

## AC対応

| AC-ID | 設計判断 | 変更パス | 実施順序 | 失敗時の扱い |
|---|---|---|---|---|
| AC-1 | canonical Git discovery/upload-packだけをpolicy scopeへ評価し、Git requestのmethod/query/media/bodyを先に閉じる。 | `egresspolicy.go`、`egresspolicy_test.go`、`scope_test.go` | 1 | policy denyのまま、credential/transportへ進めない。 |
| AC-2 | peer subjectとallowlistからGit read selectorをsingle-use 5分grantへ固定し、既存issue selectorを保存する。 | `capability.go`、`capability_test.go`、`capabilitycontrol/control.go`、`capabilitycontrol/control_test.go`、`connectsession/session.go`、`connectsession/session_test.go` | 2 | parser/control/Registryのいずれの不整合も固定拒否し、entryを発行しない。 |
| AC-3 | scopeごとのstrict Basic handle抽出、consume後のresolver一回、real-token Basicへの非可逆置換をtransactionに閉じる。 | `egresstransaction.go`、`egresstransaction_test.go`、`brokerhttp/handler.go`、`brokerhttp/handler_test.go` | 3–4 | consume前入力不正はresolver未到達、consume後失敗はrollback/retryなし、公開errorは固定値。 |
| AC-4 | Git hostをCONNECT/CA/handler/forwarder/pinned transportに一致して追加し、binary responseをoperation別に検査する。 | `connectsession/session.go`、`connectsession/session_test.go`、`proxyca/proxyca.go`、`proxyca/proxyca_test.go`、`brokerhttp/handler.go`、`brokerhttp/handler_test.go`、`upstreamforwarder/upstreamforwarder.go`、`upstreamforwarder/upstreamforwarder_test.go`、`upstreamtransport/upstreamtransport.go`、`upstreamtransport/upstreamtransport_test.go` | 3–5 | host/TLS/request/responseの任一逸脱はsink前にclose/fixed error、redirect/retryしない。 |
| AC-5 | negative matricesを各層で持ち、REST/OpenAIのauth、JSON、transport既存testsを残して回帰を検出する。 | 許可済み全テストパス | 1–5 | scope mismatch、malformed wire、unexpected responseはcredential/network/sink到達前又はdelivery前に拒否する。 |
| AC-6 | 20 path・行数・対象外差分をcandidate diffで監査し、focused raceとroot検査を実行する。 | 許可済み20パス、`README.md` | 6 | scope/line上限逸脱はcandidateを再固定せずMainへ戻し、対象外は取り込まない。 |

## 変更予定

| パス | 変更内容 |
|---|---|
| `tools/dev-agent-harness/internal/egresspolicy/egresspolicy.go` | Git Smart HTTP read decision/scope、canonical discovery query、service別request validationを追加する。 |
| `tools/dev-agent-harness/internal/egresspolicy/egresspolicy_test.go` | allowlist、canonical URL、method/query/media/body、receive-pack及びURL逸脱のdeny matrixを追加する。 |
| `tools/dev-agent-harness/internal/egresspolicy/scope_test.go` | Git readのcanonical scopeとREST/OpenAI compatibilityを確認する。 |
| `tools/dev-agent-harness/internal/capability/capability.go` | Git read operation/hostを明示scopeとして発行・consume可能にする。 |
| `tools/dev-agent-harness/internal/capability/capability_test.go` | Git readとREST readのoperation/host/repository/subject mismatch及びsingle useを検証する。 |
| `tools/dev-agent-harness/internal/capabilitycontrol/control.go` | Git read selectorをpeer-bound、allowlisted、5分、一回issueへ接続する。 |
| `tools/dev-agent-harness/internal/capabilitycontrol/control_test.go` | selector、allowlist、peer subject、固定TTL/uses、既存selector回帰を検証する。 |
| `tools/dev-agent-harness/internal/connectsession/session.go` | `github.com:443` CONNECTと厳密Git read control selectorを追加する。 |
| `tools/dev-agent-harness/internal/connectsession/session_test.go` | CONNECT/TLS/one-request/control parsingのGit host・selectorと既存host回帰を検証する。 |
| `tools/dev-agent-harness/internal/proxyca/proxyca.go` | exact `github.com` leaf発行を許可する。 |
| `tools/dev-agent-harness/internal/proxyca/proxyca_test.go` | Git host leafとnear-match/wildcard拒否、既存host回帰を検証する。 |
| `tools/dev-agent-harness/internal/brokerhttp/handler.go` | canonical Git discovery queryのinner-to-transaction mappingとGit binary response mappingを追加する。 |
| `tools/dev-agent-harness/internal/brokerhttp/handler_test.go` | query、binary body/response、framing、403 non-leak、REST/OpenAI mapping回帰を検証する。 |
| `tools/dev-agent-harness/internal/egresstransaction/egresstransaction.go` | Git-only Basic handle parseとconsume後real-token Basic replacementを追加する。 |
| `tools/dev-agent-harness/internal/egresstransaction/egresstransaction_test.go` | ordering、resolver call count、Basic canonicality、single-use、secret non-forwardingを検証する。 |
| `tools/dev-agent-harness/internal/upstreamforwarder/upstreamforwarder.go` | Git operationごとのrequest headers、binary response/status/media validation、sink deliveryを追加する。 |
| `tools/dev-agent-harness/internal/upstreamforwarder/upstreamforwarder_test.go` | one RoundTrip、Basic replacement、binary response、bad status/media/size及びREST/OpenAI回帰を検証する。 |
| `tools/dev-agent-harness/internal/upstreamtransport/upstreamtransport.go` | `github.com`を固定のpinned HTTPS hostとして追加する。 |
| `tools/dev-agent-harness/internal/upstreamtransport/upstreamtransport_test.go` | Git host resolution/dial/SNI、host拒否、HTTP/1.1、retryなしを確認する。 |
| `tools/dev-agent-harness/README.md` | Git Smart HTTP readの実装済み境界、Basic handle置換、対象外/live境界を同期する。 |

## 検証計画

DEVは、正規readのdiscoveryとupload-pack、peer/allowlist/scope mismatch、Basic handleから実Basicへの置換、resolver及びRoundTrip各一回、push/URL/HTTP/content/body/response逸脱の到達前拒否、REST/OpenAI非回帰を、上記20パス内のhermetic testでfailure-detectする。binary fixtureは小さく固定し、実GitHub、実credential、外部DNS/TLSを使わない。

candidate固定後のQA bounded rerun候補は、次のaffected packageだけを一回実行する。QA_PLANが各ACを`focused-rerun`、`evidence-review`、必要なら未実施の`live-e2e`へ個別に理由付きで確定する。

```sh
cd tools/dev-agent-harness && GOCACHE=$PWD/.build/go-cache go test -race ./internal/egresspolicy ./internal/capability ./internal/capabilitycontrol ./internal/connectsession ./internal/proxyca ./internal/brokerhttp ./internal/egresstransaction ./internal/upstreamforwarder ./internal/upstreamtransport
```

DEVはさらにroot `make check`、`git diff --check`、及び`make task-check TASK=TASK-0063`を実行する。QAはcandidate-bound test本文・negative case・DEV/Reviewerのroot check証跡を独立監査し、high-risk signal又は証跡不足があれば`evidence-review`だけをPASSにしない。実GitHub credential受理、実DNS/TLS、実systemd/VPS、実Git client flowは安全な承認済み環境とcleanupがなければ`live-e2e`をblockedのままにし、hermetic PASSで代替しない。

## リスクと復旧

- RESTとGit readをproviderだけで同一視するリスクは、operation・host・authorization grammar・response media typeをscopeから一貫して選択し、cross-operation mismatch testsで抑える。
- canonical query又はinner mappingが広がるリスクは、discoveryの一つのraw query以外をhandler/policy双方で拒否することで抑える。
- Agent handle又はreal tokenの混同・漏洩リスクは、consume後一回だけのBasic置換、prepared requestへのhandle非保持、固定error/empty 403、non-leak assertionsで抑える。
- binary responseをJSON扱いしてreadを壊す、又はmedia typeを緩めるリスクは、operation別のbounded binary fixturesとunexpected responseのsink前拒否で抑える。
- CONNECT/CA/transportのhost追加がopen proxy又はretry/redirectを生むリスクは、exact host/`:443`、SNI、one RoundTrip、pinned transportのnegative testsで抑える。

復旧時は許可20パスのcandidate製品差分だけを戻し、既存のGitHub REST/OpenAI egress graphへ復元する。新しい外部credential、persistent Registry state、listener/socket、Git client state、live GitHub side effectを作らないため、追加の環境cleanupは発生しない。復旧後は上記focused race suite、root `make check`、`make task-check TASK=TASK-0063`、`git diff --check`を再実行する。

## 引き継ぎ条件

DEVはMain承認済みのPLANと独立QA_PLANの後に、許可20パスだけで同一candidateを一回固定する。ReviewerとQAは同一candidateを相互のPASSを待たず独立に評価し、Mainだけがcompletion、`--no-ff --no-commit`検査、main統合と必要な環境依存確認を所有する。candidateにpush/write、helper/client、依存、Schema、runtime、生成物又はlive configurationを含めない。

## main Agentレビュー

- [x] TASKの全AC-IDへ設計判断、パス、順序、失敗時の扱いを対応させ、条件本文を複製していない。
- [x] Git Smart HTTPのcanonical request、peer-bound capability、Basic credential replacement、CONNECT/CA/transport、binary response、REST/OpenAI不変、復旧を具体化している。
- [x] QA_PLANがTASK-firstで独立作成されている。
- [x] `dependency-ready reconciliation`と完了経路preflightが完了している。
- [x] DEV開始を承認した。
