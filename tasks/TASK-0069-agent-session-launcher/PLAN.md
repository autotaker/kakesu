---
task_id: "TASK-0069"
change_class: "product"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "sol-high"
approved_dev_profile_reason: "既存のstrict control client、loopback bridge、Git credential helperを、child process、限定environment、signal、revokeとtemporary-file cleanupへ一つのfail-closed lifecycleとして束縛するcredential-adjacent統合のため。"
approved_dev_profile_risk_signals:
  - "Opaque GitHub/OpenAI capabilityをchild environmentへ一時注入する境界"
  - "fixed Unix socket/helperのlink-time bindingとAgent入力からの分離"
  - "child exit、signal cancel、bridge drain、CA file removal、handle revocationの順序とexactly-once cleanup"
  - "parent environment/diagnostic/non-leakおよびGit command-scope configuration"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-02T05:08:18Z"
classification_approved_by: "main-agent-sol-high"
classification_approved_at: "2026-08-02T05:08:18Z"
classification_approval_reason: "Opaque credential environment、child process、bridge、CA file、build wiringを新しい外部観測可能なlauncher sessionへ統合する製品変更。"
# safety_contract_version: 2
# safety_contract_planned_paths: []
# safety_contract_generated_paths: []
---

# PLAN: Agent session launcher

## 根拠と分類

唯一の要求根拠は`TASK.md`の`Planning input packet`である。Agentが実行するchild process、CLI、environment、CA trust file、Git command-scope configuration、bridge lifetime、capability revoke、build/install時に埋め込まれる接続先を新たに外部観測可能な一つのsessionとして合成するため、`change_class`は`product`とする。Opaque credential environmentとprocess cleanupの高リスク信号に従いDEV profileは`sol-high`とし、本PLANに独立PLANレビューは置かない。MainはDEV開始前に本PLANとTASK-firstで独立作成する`QA_PLAN.md`の意図、scope、受け入れ経路を確認して承認を記録する。

candidateは7許可パスだけを変更し、追加・削除合計を約850〜1,200行に保つ。3コミット標準はMain所有のplanning commit、製品差分だけのcandidate commit、REVIEW/QA証跡を含むno-ff completion commitとする。Planner/DEV/Reviewer/QAはstage、commit、merge、pushを行わない。

既存のcontrol client、proxy bridge、git credential helper、control protocol、capability budget/policy、egress service、provider credential replacementの意味は変えない。config/dependency/Schema、Kakesu runtime、systemd/sysusers/tmpfiles/provision manifest、生成済み`configure`、live state、実credential、実Git/gh/Codex/OpenAI SDK、外部通信、DNS/TLSはcandidateに含めない。launcherはnetwork isolationを強制せず、実OS default-deny firewall/network namespace、loopback reachability、Unix socket ownership/peer UID、実配置は必要なlive-e2eとして別に扱う。

## end-to-end設計

1. `dev-agent-launcher`だけをsession用binaryにする。`cmd/dev-agent-launcher/main.go`は`SIGINT`と`SIGTERM`を`signal.NotifyContext`で一つのcontext cancellationへ変換し、`command.RunContext`へ渡す。他binaryのsignalやscaffold contractは変えない。
   - `command`はlauncher名のときだけ、exact `run --repository owner/repo -- COMMAND [ARG...]`、single `--help`/`-h`、single `--version`を受ける小さなparserへ分岐する。`--`はcommand開始の唯一のdelimiterであり、repositoryは既存controlclientと同じcanonical lowercase grammarで検証する。空command、NUL、unknown/missing/duplicate/reordered option、delimiterなし、launcher追加optionはsession package/control dial/child startの前にexit 2と固定usage diagnosticへ畳む。
   - command argumentは再parse、join、shell、`sh -c`を通さず`exec.CommandContext`相当へそのまま渡し、stdin/stdout/stderrはchildへ直結する。`--help`と`--version`の成功出力はlauncher固有のusage/versionだけとし、operational failureは常に`dev-agent-launcher: session failed`だけにする。repository、argv、handle、socket/helper/temp path、environment値、下位errorはどの診断にも残さない。

2. `internal/launchsession`をtrusted composition ownerとして新設する。public requestはvalidated canonical repositoryとchild argvだけを持ち、socket、helper、provider、operation、model、proxy endpoint、CA path、environmentを入力に持たない。production dependenciesはlink-time embedded absolute clean egress socketとinstalled absolute helper pathから一度だけ作る。testはpackage-private seamsでのみcontrol/bridge/process/filesystem/clockを差し替え、exportしたoverride、CLI/config/environment input、実socket listenerを導入しない。
   - sessionは順に、(a) fixed pathsをdial/start前に検証、(b) `controlclient.ProxyCA`一回、(c) GitHub REST issue一回、(d) OpenAI issue一回、(e) literal `/tmp`直下でfresh 0700 directoryを安全に作成して公開CAを0600 regular fileへ一回書込み、(f) fixed Unix socketを渡す`proxybridge.New`一回、(g) bridge `Serve`、(h) child start、(i) child wait、を所有する。CA取得、各issue、temp file、bridge、childの順序はテストから観測できるようにするが、実値は出力しない。
   - 任意の前段が失敗したら未開始の後段は開始しない。childはbridgeのready endpointとCA fileが完全に作られ、三つのcontrol取得が成功した後だけ開始する。Git-read credential handleは事前発行しない。固定helperは各Git readで既存single-use control clientを使い、push/write拒否は既存のままとする。

3. CA fileとbridgeの所有権をsessionに閉じる。temporary directoryは親`TMPDIR`、relative path、existing pathを使わず、literal `/tmp`配下で安全なfresh-directory primitiveを一度だけ用い、作成後にdirectory mode `0700`、CA fileがsymlinkでないregular fileかつ`0600`であることを検証する。CA bytesは公開certificateだけを一回writeし、close/permission failureをsetup failureとする。CA file、directory、handleをchild diagnosticsやargvへ移さない。
   - bridgeは既存APIのfixed `tcp4` IPv4 loopback endpointだけを使い、endpointをchild environment用に局所保持する。`Serve`はchildと並行して一回だけ実行し、child終了と予期しないServe終了のどちらかを先に観測する。Serveが先に異常終了した場合はchildを停止してwaitする。bridge serve failure、child start failure、child exit、parent cancel、setup failureのどれでも、新規acceptを止めるためbridge contextをcancelし、Serve完了をdrainしてからlocal CA directoryを再試行なしで削除する。bridge/file cleanup errorは固定launcher failureへ畳み、path又は下位errorを表示しない。
   - cleanupはpartial initializationにも適用する一つのdefer/ownerで逆順に行う。issued GitHub/OpenAI handlesは各々boolean ownershipを持ち、cleanupで一回だけ`Revoke`を試行する。CAだけがdisk resource、handleだけがcontrol resourceであり、revoke失敗をlocal cleanup成功又はchild exitと混同しない。unknown/expired handleのrevoke failureは残余TTLによるfail-safeに任せ、secret diagnosticを増やさない。

4. child environmentはparent mapをmutateせず、fresh sliceへ決定的に組み直す。継承候補は`HOME`、`PATH`、`TERM`、`LANG`、canonicalな`LC_*`、任意の`CODEX_HOME`のみで、duplicate、NUL、malformed entryはfail-closedに扱う。`HOME`/`CODEX_HOME`はCodexが自ら認証情報を読む場所を保つためだけにコピーし、launcherは内容をstat/read/copy/log/expandしない。
   - allowlist外の全値、特に`GH_*`、`OPENAI_*`、HTTP/HTTPS/ALL/NO proxy、CA trust、Git config/credential、SSH、dynamic-loader/runtime injectionをchildへ渡さない。sessionが決める値だけを一度ずつ付加する: opaque `GH_TOKEN`/`OPENAI_API_KEY`、`HTTP_PROXY`/`HTTPS_PROXY`とlowercase pairの同じcanonical loopback endpoint、`SSL_CERT_FILE`/`CURL_CA_BUNDLE`/`REQUESTS_CA_BUNDLE`/`NODE_EXTRA_CA_CERTS`/`GIT_SSL_CAINFO`の同じCA file、`GIT_TERMINAL_PROMPT=0`、Git command-scope configである。環境の重複キーを残さず、childがcontrol socket/helper/temp directoryを選択又は発見するenvironmentを作らない。
   - Gitにはglobal/system/local configurationへ書かないcommand-scope configだけを渡す。existing helper listをempty valueでresetし、absolute embedded helperだけを追加する。GitHub `credential.useHttpPath=true`、HTTPS proxy、CA file、credential prompt/askpassを無効化する設定を固定し、repository、proxy、helper、CAをchild inputまたはparent inherited configから採用しない。Git config count/key/value sequenceとchild argv境界をpackage-private process seamでbyte/unit単位に検査する。

5. process resultはsession packageが正規化する。childはcontextに結び、cancel時は停止して必ずwaitする。normal child exitは0、ordinary nonzero exitはそのexit codeを保つ。signal termination、start/wait error、setup/bridge/local-file cleanup failureは固定nonzero launcher failureへ畳み、child signal numberやlower errorを表示しない。revokeのunknown/expiry等の失敗は残余TTLに任せ、既に得たchild exit結果を上書きしない。parent cancellationではchild wait、bridge drain、file removal、revoke試行が終わるまで`Run`は戻らない。
   - childが終了してもbridgeを先に放置しない。child process lifetimeはbridge lifetimeの内側であり、child wait後にbridge cancel/drain、CA directory removal、issued handle revokeを一回ずつ行う。cleanupの失敗がordinary child nonzeroを秘密付き情報へ変えず、packet指定の固定exit taxonomyを越える外部error surfaceを増やさない。

6. Makefileはlauncher target専用のlinker variablesを追加する。configure由来の`$(runstatedir)/dev-agent-harness/egress.sock`と、install先`$(bindir)/git-credential-dev-agent`をlaunchsessionの非公開link-time stringsへ埋め込む。既存helperのsocket embedding、全binary version value、install layout、`dist`入力の意味は維持する。launcherはruntime path lookup、`PATH`探索、environment/config/flagのhelper/socket overrideをしない。
   - READMEはexact CLI、opaque handlesをchildだけに渡すこと、fixed bridge/CA/Git setup、cleanup、Codex credential location例外、external enforcementが別責務であることを記載する。実token、handle、repository、socket/helper/temp path、proxy endpoint、実環境のcopy-paste configurationは例示しない。hermetic evidenceが実credential、client proxy support、OS isolation/permission、TLS/DNS/provider受理を証明しないことを明確にする。

## AC対応

| AC-ID | 設計判断 | 変更パス | 実施順序 | 失敗時の扱い |
|---|---|---|---|---|
| AC-1 | launcherだけにexact parserを置き、canonical repositoryとnonempty argvをchild/control前に検証する。argv/stdioはshellなしに保持する。 | `internal/command/command.go`、`command_test.go`、`cmd/dev-agent-launcher/main.go` | 1, 5 | invalid/reordered/NUL/missing inputはexit 2 usageのみ。session/control/processは未到達。 |
| AC-2 | launchsessionがembedded fixed socket/helperをpreflightし、CA→GitHub REST→OpenAIを各一回取得してからfile/bridge/childへ進む。 | `internal/launchsession/session.go`、`session_test.go`、`Makefile.in` | 2, 6 | 任意の前段失敗は後段未開始。取得済みhandleだけを一回revokeし、fixed failureへ畳む。 |
| AC-3 | literal `/tmp` fresh 0700 dirと0600 regular CA file、one fixed loopback bridge、cancel/drain→remove→revokeのownerを一つにする。 | `internal/launchsession/session.go`、`session_test.go` | 2--3 | setup/start/normal/nonzero/cancelの全経路でaccept停止、drain、one removal、one revoke。path/errorは露出しない。 |
| AC-4 | fresh allowlist environmentとfixed session entries、Git helper reset/absolute helper、path-aware GitHub credential、proxy/CA/prompt disableをcommand scopeへ固定する。 | `internal/launchsession/session.go`、`session_test.go` | 3--4 | inherited credential/proxy/Git/SSH/loader value、duplicate又はuntrusted overrideはchildへ渡さない。invalid entryはstart前拒否。 |
| AC-5 | context-bound child start/waitとcleanup completionを一run ownerに置き、ordinary child exitだけを保持しその他はfixed nonzero/diagnosticにする。 | `internal/launchsession/session.go`、`session_test.go`、`internal/command/command.go`、`cmd/dev-agent-launcher/main.go` | 4--5 | cancel時はwait+drain後にreturn。revoke failureはsecret diagnostic又はchild resultの不正な上書きをしない。 |
| AC-6 | permitted 7 path/850--1,200 line budget内にsource/test/build/docを閉じ、race、harness、dist/install、root/scope evidenceをcandidateへ束縛する。 | 許可済み7パス | 1--6 | scope/budget/test failureはcandidateへ含めずMainへ戻す。live条件はhermetic PASSへ置換しない。 |

## 変更予定

| パス | 変更内容 |
|---|---|
| `tools/dev-agent-harness/internal/launchsession/session.go` | fixed link-time paths、preflight、one-shot control acquisition/revocation、safe CA directory/file、proxybridge/process lifecycle、fresh environment/Git command config、exit normalization、private seamsを実装する。 |
| `tools/dev-agent-harness/internal/launchsession/session_test.go` | fake control/bridge/process/filesystemとbounded synchronizationでordering/count、all cleanup paths、environment/Git argv、non-leak、cancel/wait/raceを検証する。 |
| `tools/dev-agent-harness/internal/command/command.go` | launcherのexact parser、usage/version/help dispatch、launchsession invocation、fixed diagnosticsを追加し既存commandを回帰させない。 |
| `tools/dev-agent-harness/internal/command/command_test.go` | valid CLI/argv/stdio handoff、unknown/missing/duplicate/reordered/NUL rejection、no session start、fixed output/exit分類を追加する。 |
| `tools/dev-agent-harness/cmd/dev-agent-launcher/main.go` | SIGINT/SIGTERM context bridgeを加え、launcher専用`RunContext`を呼ぶ。 |
| `tools/dev-agent-harness/Makefile.in` | launcher専用のsocket/helper absolute link-time values、implemented launcherを許容するbinary check、install/distcheck wiringを加える。 |
| `tools/dev-agent-harness/README.md` | session CLIと責務境界、environment/cleanup/Codex exception、hermetic/live保証限界を記載する。 |

## 実装手順

1. `launchsession`の最小API、fixed public error、embedded socket/helper variables、canonical repository/argv/path preflight、package-private seamsを定義する。本番入力にprovider、operation、model、socket/helper/proxy/CA overrideを増やさない。
2. acquisition transactionを実装する。strict existing clientsをCA、GitHub REST、OpenAIの固定順で各一回呼び、partial owner flagsとreverse revokeを先にtestする。Git-read issueをここで呼ばない。
3. safe CA directory/fileとproxy bridgeをtransactionへ結合する。literal `/tmp`、mode/regular-file verification、write/close failure、bridge start/serve/cancel/drain、partial setup cleanupを実装し、child start前のready条件を固定する。
4. fresh environment builderとprocess specを実装する。allowlist copy、Codex location exception、deny-by-omission、session-only proxy/CA/opaque handles、Git command-scope helper reset/path-aware/noninteractive configurationを決定的な一意順序で作る。
5. child runnerとresult normalizationを接続する。stdio passthrough、context cancellation、start/wait、ordinary exit preservation、fixed failure、cleanup completion後returnを実装する。command parserとlauncher signal entrypointをそのboundaryへつなぐ。
6. Makefile install-time paths/binary checkとREADMEを実装済みcontractへ更新する。test fixtureでCLI、ordering、environment、Git argv、failure cleanup、non-leak、cancellationを網羅し、最後に許可パスとline budgetを確認する。

## 検証計画

DEVのfocused hermetic suiteは実credential、実socket、external network、real Git/gh/Codex/OpenAI clientを使わない。fake control clientはCA/GitHub/OpenAI issue/revoke call countと順序を記録し、fake bridgeはendpoint/start/serve/cancel/drainを、fake processはargv/environment/stdio/start/wait/cancelを、filesystem seamはliteral `/tmp`要求、mode、regular/symlink/error、remove countを記録する。global seamsはparallelにせずcleanupで復元する。

- CLI: `--help`/`-h`/`--version`と唯一のvalid runを確認し、zero command、NUL、invalid/case違い repository、unknown/missing/duplicate/reordered option、extra launcher input、`--`欠落ではusage、control/process未到達、secret/context-free stderrを確認する。argv境界とstdio objectが変形されないことも確認する。
- acquisition/ordering: invalid embedded path、CA failure、GitHub issue failure、OpenAI issue failure、CA write/close/mode failure、bridge start/serve failure、child start failureごとに、後段未到達、既発行handleだけone revoke、no double revoke、no Git-read issue、fixed diagnosticを確認する。
- lifecycle: normal 0、ordinary child nonzero、signal/start/wait/setup/cleanup failure、child実行中のunexpected bridge failure、already-cancelled/cancel-during-childでchild停止/wait、bridge no-new-accept/drain、CA remove、revokeが各一回かつcompletion順であることを確認する。revoke unknown/expiry failureがhandle/path/lower errorを漏らさずordinary child resultを不当に秘密付きに変えないことを確認する。
- environment/Git: hostile parent `GH_*`/`OPENAI_*`/proxy/CA/Git/credential/SSH/loader variablesが消え、allowlistとoptional `CODEX_HOME`だけが保たれ、session valuesがunique/canonicalであることを確認する。`HOME`/`CODEX_HOME`の内容をfilesystem seamが読まないこと、helper reset後のabsolute helper、`useHttpPath`、proxy/CA、noninteractive credential configの正確なargv/config sequenceを確認する。
- race/non-leak/scope: concurrent cancel/child exit/bridge failureをboundedに同期してdouble close/revoke/remove、goroutine残留、data raceを検出し、error/format/stdout/stderr/environment dumpがhandle/repository/socket/helper/temporary path/command/lower errorを含まないことを確認する。許可外差分、line budget、READMEのnon-live表現をレビューする。

candidate固定前にDEVは少なくとも次を実行する。

```sh
cd tools/dev-agent-harness && GOCACHE=$PWD/.build/go-cache go test -race ./internal/launchsession ./internal/command
cd tools/dev-agent-harness && make check
cd tools/dev-agent-harness && make distcheck
make task-check TASK=TASK-0069
git diff --check
```

さらにinstall stagingで、built launcherに埋め込まれたcontrol socket/helperがconfigure/install layout由来のabsolute pathsであること、installed launcher/helperが想定位置に置かれることを確認する。root `make check`はDEV検証で重複実行せず、Mainがcandidate固定後の同じbytesへcandidate gateとして一回だけ実行する。QA_PLANはACごとに`focused-rerun`、`evidence-review`、必要な`live-e2e`を理由付きで決める。opaque auth、environment non-leak、cleanup/race、build/link-time bindingはcandidate-bound negative/race evidenceが不足すると`evidence-review`だけでPASSにしない。

実OS default-deny/network namespace、loopback reachability、Unix socket permission/peer UID、real clientのproxy/CA/Git helper/Codex authentication挙動、real provider/DNS/TLS、deployment/restart/rollbackは`live-e2e`である。承認済み環境または安全なcleanupがなければblockedとして残し、hermetic testを代替PASSにしない。

## リスクと復旧

- parent credential又はruntime injectionがchildへ漏れるリスクは、denylistではなくfresh allowlistとunique session entries、hostile parent-environment matrixで抑える。Codex exceptionは場所だけをcopyし、launcher自身はsecret bytesを扱わない。child Codexがagent-user側credentialを直接読むことはpacketどおり例外として許可する。
- Agentがsocket/helper/proxy/provider/operation/modelを差替えるリスクは、link-time absolute paths、minimal request、unexported seams、fixed control calls/bridge constructor、CLI/config/environment override不在で抑える。
- partial setup又はraceがhandle、CA dir、bridge goroutineを残すリスクは、owner flags、reverse exactly-once cleanup、cancel→drain→remove→revokeのfailure matrixと`-race`で抑える。
- Gitが別helper/prompt/parent configを探索するリスクは、command-scope helper reset、absolute helper、path-aware credential、prompt disable、Git argv exact assertionsで抑える。
- diagnosticがcredential-adjacent dataをoracle化するリスクは、usage又は固定`session failed`のみ、context-free errors、stdout/stderr/format non-leak assertionで抑える。
- launcherをnetwork enforcementと誤認するリスクは、READMEとQAでrouting/secret replacement boundaryに限定し、OS isolation等をlive-e2eへ分離する。

復旧時は7許可パスのcandidate製品差分だけを戻し、scaffold launcherと既存の独立control/bridge/helperを復元する。sessionが残した可能性がある一時CA directory又はbridge/child/handleについては、candidateのcleanup evidenceで対象を確認してから、承認済み実環境の手順に従い安全にcleanupする。復旧後はfocused race suite、harness/root `make check`、`make distcheck`、`make task-check TASK=TASK-0069`、`git diff --check`を再実行する。

## 引き継ぎ条件

DEVはMain承認済みPLANと独立QA_PLANの後、7許可パスだけで一回のcandidateを固定する。ReviewerとQAは同一candidateを相互のPASSを待たず独立に評価する。Mainだけがcandidate identifier、3コミット標準、stage/commit/merge/push、`--no-ff --no-commit` completion check、main統合、必要なpost-merge live-e2eを所有する。

candidateには実credential、実外部通信、Git push/write、GitHub GraphQL/write、OpenAI upload/admin/files、approval/grant/renew/retry/cache/persistence/audit、config/dependency/Schema/generated/live state、systemd/provision変更を含めない。packetのdependency-ready reconciliationはN/Aであり、TASK-0065--0068のfixed contractsを再解釈又は変更しない。

## 未解決事項

- なし。packetでreadyとされたTASK-0065--0068のcontrol socket、public CA、bridge、API capability contractを前提にする。

## main Agentレビュー

- [x] TASKの全AC-IDへ設計判断、パス、順序、失敗時の扱いを対応させ、条件本文を複製していない。
- [x] exact CLI、link-time socket/helper、fixed acquire order、CA/bridge/process cleanup、fresh environment/Git command scope、exit/non-leakを具体化している。
- [x] `sol-high`、7許可パス、約850--1,200行、3コミット標準、独立PLAN reviewなしを記録している。
- [x] QA_PLANがTASK-firstで独立作成されている。
- [x] `dependency-ready reconciliation`と完了経路preflightが完了している。
- [x] DEV開始を承認した。
