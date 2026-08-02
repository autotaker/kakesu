---
task_id: "TASK-0068"
change_class: "product"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "sol-high"
approved_dev_profile_reason: "Opaque API capabilityのprotocol clientとscope別request budgetを同時に変更する安全境界のため。"
approved_dev_profile_risk_signals:
  - "Agentへ渡すOpaque authentication handle"
  - "peer-bound control protocolのexact request/response"
  - "GitHub REST/OpenAIの16-use budgetとGit-read single-useの分離"
  - "subject/workspace/provider/repository/operation/destinationの非消費境界"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-02T04:24:47Z"
classification_approved_by: "main-agent-sol-high"
classification_approved_at: "2026-08-02T04:24:47Z"
classification_approval_reason: "Agent向けOpaque API capabilityのclient surfaceとscope別request budgetを変更する製品変更。"
# safety_contract_version: 2
# safety_contract_planned_paths: []
# safety_contract_generated_paths: []
---

# PLAN: API capability control client

## 根拠と分類

唯一の要求根拠は`TASK.md`の`Planning input packet`である。固定Unix control socketへ送るAgent側request、公開するclient API、controllerが発行するcapabilityの使用回数、およびREADMEの外部観測可能な境界を変更するため、`change_class`は`product`とする。opaque authentication、strict protocol framing、scope-bound request budgetを同時に変更するため、DEV profileは`sol-high`を要求する。

candidateはpacketの6許可パスだけを変更し、追加・削除を合わせて約750〜1,050行に収める。launcher、environment、child process、CA trust、Git config、loopback bridge、runtime path override、依存、Schema、Makefile、Kakesu runtime、生成物、永続化又はlive stateは変更しない。実credential、実`gh`/SDK、GitHub/OpenAIへの通信、DNS/TLS、systemd socket、別UID/VPSもこのcandidate又はhermetic PASSの対象にしない。

MainはDEV開始前に本PLANとTASK-firstで独立に作る`QA_PLAN.md`の意図・scope・受け入れ経路を確認する。本PLANの独立レビューは設けない。DEV後は同一candidateから独立REVIEWとQAを実施する。

## end-to-end設計

1. `controlclient`のpublic surfaceは任意provider、operation、model、request bodyを受けない三つの明示操作に閉じる。既存のGit Smart HTTP `Issue(socketPath, repository)`はそのまま`github-git-read`専用に残し、GitHub REST read用とOpenAI Responses text用の別関数を追加する。
   - GitHub REST関数はabsolute clean fixed Unix socketとcanonical `owner/repo`だけを受け、固定JSON順序の`{"provider":"github","repository":"owner/repo"}`を送る。OpenAI関数はsocketだけを受け、固定`{"provider":"openai"}`を送る。repository、provider、operation、model、pathをAgent入力から混成する汎用Issue APIは追加しない。
   - 各呼出しは一dial、一request、一response、一closeだけであり、既存の短いdial/write/read phase deadlineを維持する。retry、fallback、複数socket、TCP、socket overrideは導入しない。

2. Git-read、GitHub REST、OpenAIのissue requestは一つのprivate strict exchangeへ収束させる。
   - exchangeはcallerが選べない固定request bytesを受け、request全量を一度の論理操作としてwriteし、唯一のbounded `HTTP/1.1 200 OK`、固定順序の`Content-Type: application/json`、canonical `Content-Length`、`Connection: close`、canonical `{"handle":"cap_..."}`、直後EOFだけを成功とする。既存のproxy-CA exchangeはこの変更で意味を変えない。
   - status、header order/casing/whitespace、重複・余分header、body size/length、JSON、noncanonical handle、early EOF、extra byte、dial/deadline/read/write/closeのいずれの失敗も空handle/nilと`ErrControl`だけに畳む。error値とdiagnosticにはsocket、repository、handle、wire、provider、下位errorを保持・出力しない。

3. `capabilitycontrol.Controller`はtrusted scope選択から使用回数を固定する。
   - GitHub REST（selector省略）とOpenAI Responses text（selector省略又は既存の正規selector）はTTL 5分・16 usesを発行する。GitHubの明示`github-git-read` selectorだけはTTL 5分・1 useに保つ。caller、Agent、設定、repository/model入力からusesを受け取らない。
   - 既存のRegistryを唯一のatomic consume ownerのまま使い、正規scope consumeだけが1減るようにする。API handleは16回目に`RemainingUses == 0`となり、17回目を拒否する。subject、workspace、provider、repository、operation、destinationのmismatchはRegistryの既存照合で拒否され、残数を消費しない。
   - allowlist repository、OpenAI model gate、peer subject、revocation、expiry、epoch、provider/host mappingは変更しない。API handleがGit read、push/write、GraphQL/upload、別provider/repository/hostへ交差できないことを回帰として固定する。

4. READMEはlauncherより前のclient/control境界だけを記録する。後続launcherがopaque handleを`GH_TOKEN`/`OPENAI_API_KEY`へ渡し得る前提、GitHub RESTはallowlisted repository、OpenAIはmodelをclient入力に持たないこと、API request budgetは16、Git Smart HTTP readはsingle-useであることを説明する。実token、環境設定、launcher、bridge、実ネットワーク、write/push/approvalは対象外と明示し、handle、socket、repositoryを秘密又は操作可能な例として載せない。

## AC対応

| AC-ID | 設計判断 | 変更パス | 実施順序 | 失敗時の扱い |
|---|---|---|---|---|
| AC-1 | provider/bodyを受けないGitHub REST/OpenAI明示関数を設け、既存Git-read `Issue`を保持する。全issue操作は同一private exchange経由のfixed wire、一dial、deadline、closeにする。 | `internal/controlclient/client.go`、`client_test.go` | 1–2 | invalid socket/repository又はtransport/protocol失敗は一回で終え、empty handleと固定errorだけを返す。 |
| AC-2 | issue種別ごとのrequest生成は共有strict exchangeへ渡し、唯一の200 JSON framing、canonical handle、EOFを受理する。 | `internal/controlclient/client.go`、`client_test.go` | 2 | header/body/JSON/handle/framing/extra byte又はphase failureを固定拒否し、retryもcontext漏洩も行わない。 |
| AC-3 | trusted provider/operation分岐でAPI=16、explicit Git-read=1を決め、Registryのatomic consumeと既存bindingを維持する。 | `internal/capabilitycontrol/control.go`、`control_test.go` | 3 | 17回目、scope/subject/workspace mismatch、expiry/revoke/epochは拒否し、mismatchは残数を減らさない。 |
| AC-4 | Git-read client wire、selector、credential helperのsingle-use lifecycleとCONNECT/control wireを回帰で守り、shared Registryのegress-service compositionでAPI scopeの16回成功・17回目拒否とrevokeを確認する。 | `internal/controlclient/client.go`、`client_test.go`、`internal/capabilitycontrol/control.go`、`control_test.go`、`internal/egressservice/service_test.go` | 2–3 | Git-read挙動、CONNECT/control wire、revoke又はAPI handleのGit/write/push/別scope消費が変わる差分はcandidateから除外する。 |
| AC-5 | 許可6パス・行数予算・対象外をレビュー可能にし、hermetic focused suiteとroot/harness checksをcandidateに結び付ける。 | 許可済み6パス | 1–5 | scope、行数、secret/live state、検査逸脱はMainへ差し戻し、統合しない。 |

## 変更予定

| パス | 変更内容 |
|---|---|
| `tools/dev-agent-harness/internal/controlclient/client.go` | GitHub REST read/OpenAI Responses textの明示issue API、固定request bytes、既存Git-readと共用するprivate strict issue exchangeを追加し、fixed error/deadline/one dial/closeを維持する。 |
| `tools/dev-agent-harness/internal/controlclient/client_test.go` | 三issue操作のexact wire、single dial/deadline/close、invalid inputのdial前拒否、strict response/framing/transport/close failure、fixed-error non-leakを`net.Pipe`/fake dialerで検証する。 |
| `tools/dev-agent-harness/internal/capabilitycontrol/control.go` | trusted provider/operationからAPI 16-useとGit-read single-useを選ぶprivate policyを実装し、既存TTL、subject/allowlist/model gate、Registry委譲を保持する。 |
| `tools/dev-agent-harness/internal/capabilitycontrol/control_test.go` | GitHub REST/OpenAIの16回atomic consume・16回目remaining 0・17回目拒否、各binding mismatch非消費、Git-read 1回、scope crossing/revoke/expiry回帰を追加する。 |
| `tools/dev-agent-harness/internal/egressservice/service_test.go` | production compositionがcontrollerとtransactionへ共有Registryを渡すintegrationで、API handleの16回成功・17回目拒否とrevoke後拒否を回帰する。 |
| `tools/dev-agent-harness/README.md` | launcher前のopaque client/control境界、API/Git-read別budgetと非許可操作・非live範囲を記録する。 |

## 実装手順

1. `controlclient`にpublic APIを増やす前に、existing `Issue`のGit-read requestを意味変更せずにprivate request constructor/issue exchangeへ整理する。各constructorは正規入力とliteral provider/operation/bodyしか作れない形にし、proxy-CA/revoke pathを変更しない。
2. `net.Pipe`のserver側で三つのrequest bytesを比較し、partial write、一dial、read/write deadline、close、success handleを確認する。invalid socket/repository、全種類のmalformed response、lower error sentinel、extra response/byteを加え、return値・error文字列に入力又はwireが含まれないことを固定する。
3. controllerのuses値をoperation scopeで決めるprivate helper又は等価の分岐へ置換する。REST/OpenAIを各16回consumeし、各正規consumeのremainingを確認して17回目を拒否する。Git-readの既存single-useと、各scope mismatch後に正規consumeが可能なことを同じRegistryで検証する。
4. READMEを最小限に更新し、opaque handleを渡す後続launcherとの責務分離、固定scope/budget、禁止範囲とlive-e2e境界を記す。
5. shared Registryを使う既存egress-service integrationにAPI handleの16回成功・17回目拒否・revoke後拒否を追加し、controller unit testだけでproduction compositionのbudget回帰を代替しない。diffを6パス・約750〜1,050行へ収め、focused suite、race、harness/root checks、task check、whitespace checkをcandidateに記録する。Mainが一回だけcandidateを固定し、同一candidateを独立REVIEW/QAへ渡す。

## 検証計画

DEVは次のhermetic suiteを実行する。`net.Pipe`/fake dialerとin-memory Registryだけを使い、実socket、credential、GitHub/OpenAI、DNS/TLSを使用しない。

```sh
(cd tools/dev-agent-harness && GOCACHE=$PWD/.build/go-cache go test -count=1 -race ./internal/controlclient ./internal/capabilitycontrol ./internal/egressservice ./internal/gitcredential && make check && make distcheck)
make task-check TASK=TASK-0068
git diff --check
```

focused testsは、GitHub RESTとOpenAIのexact POST body、Git-readの既存operation body、canonical request header order、single dial/one request/deadline/close、strict 200 JSON/EOF、response/transport/close failureとnon-leakを確認する。`internal/gitcredential`を同じrace実行へ含め、既存Git credential helperのget/erase回帰を直接確認する。controller testsはREST/OpenAIそれぞれの16回の正規consume、16回目remaining 0、17回目拒否、subject/workspace/provider/repository/operation/destination mismatch非消費、Git-read一回、API handleのGit/read-write/別provider/repository/host拒否、revoke/expiry/epoch回帰を確認する。`internal/egressservice`はshared Registryのproduction compositionでAPI 16回成功・17回目拒否・revoke後拒否を直接確認する。CONNECT/server control wireは許可パスに差分を持たないため、harness `make check`/`make distcheck`の回帰検査証跡を監査する。

rootの`make check`はDEVの検証には含めず、candidate固定後にMainが一度だけ所有するcandidate gateとして実行する。

candidate固定後、QA_PLANはwire/framing、scope別budget、Registry atomicity、shared Registry integrationのhermetic決定的ケースを`focused-rerun`へ割当て、candidate-bound tests、negative assertions、6-path/line-budget/対象外、READMEの非live表現を`evidence-review`へ割当てる。credential injection、launcher/env、CA trust、実socket/別UID、実GitHub/OpenAI、DNS/TLS、外部HTTPの確認は`live-e2e`であり、承認済み環境と安全なcleanupがなければblockedのままとし、hermetic PASSで代替しない。

## リスクと復旧

- generic client APIがAgent入力からscopeを広げるリスクは、provider/body/model/operationを受けない明示関数とliteral wireで抑える。
- client parserがserver failure又は余分なbytesを成功扱いするリスクは、共有exchangeの唯一 framing/EOF条件とnegative wire matrixで抑える。
- APIの16 usesがGit-read又はwrite権限にも広がるリスクは、controller内のtrusted scope別uses分岐とcross-scope consume否定で抑える。
- mismatchがbudgetを減らすリスクは、同じhandleへのmismatch→正規consumeを各binding項目で検証して抑える。
- handle、repository、socket、wire、lower errorがdiagnosticへ漏れるリスクは、固定public errorとsentinel non-leak assertionsで抑える。

復旧時はこの6許可パスのcandidate製品差分だけを戻し、現行のGit-read-only clientと全scope single-use controllerへ復元する。新規credential、config、process、socket、cache、永続状態、外部操作を作らないため追加cleanupはない。復旧後は上記focused race suite、harness/root `make check`、`make task-check TASK=TASK-0068`、`git diff --check`を再実行する。

## 引き継ぎ条件

DEVは承認済みPLANと独立QA_PLANの後、6許可パスだけで製品差分candidateを一回だけ固定する。ReviewerとQAは同じcandidateを相互のPASSを待たずに独立評価する。Mainだけがcandidate識別子、`--no-ff --no-commit`確認、mainへの統合、必要なlive-e2e判断を所有する。

## 未解決事項

- なし。packetでreadyとされたTASK-0061/0065のcontrol wire、Registry、fixed socket contractを前提にする。

## main Agentレビュー

- [x] TASKの全AC-IDへ設計判断、パス、順序、失敗時の扱いを対応させ、条件本文を複製していない。
- [x] explicit client surface、shared strict exchange、API 16/Git-read 1 use、mismatch非消費、non-leak、対象外を固定している。
- [x] QA_PLANがTASK-firstで独立作成されている。
- [x] `dependency-ready reconciliation`と完了経路preflightが完了している。
- [x] DEV開始を承認した。
