---
task_id: "TASK-0065"
change_class: "product"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "sol-high"
approved_dev_profile_reason: "Opaque capabilityをGit credential protocolへ返すauth/protocol境界であり、credential input parsing、固定Unix control wire、fail-closed Git探索停止、link-time socket固定を同時に扱うため。"
approved_dev_profile_risk_signals:
  - "opaque authentication capability boundary"
  - "Git credential helper protocol and fail-closed prompt suppression"
  - "strict Unix control protocol client and response framing"
  - "link-time fixed socket path with private test seam"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-02T02:32:01Z"
classification_approved_by: "main-agent-sol-high"
classification_approved_at: "2026-08-02T02:32:01Z"
classification_approval_reason: "Git credential helperの入出力、Opaque capability制御client、link-time runtime設定という外部観測可能な製品挙動を変更するため。"
# safety_contract_version: 2
# safety_contract_planned_paths: []
# safety_contract_generated_paths: []
---

# PLAN: Git read credential helper

## 根拠と分類

唯一の要求根拠は`TASK.md`の`Planning input packet`である。Gitのcredential helperとしての標準入出力、Gitが観測する認証探索の停止、既存egress control socketへのclient protocol、リンクされるbinaryを追加・変更するため、`change_class`は`product`とする。DEVは`sol-high`とする。Mainは本PLANとTASK-firstで独立した`QA_PLAN.md`の意図、scope、受け入れ経路を確認してから承認を記録する。本PLAN自体の独立レビューは設けない。

candidateはpacketの7許可パスだけを変更し、追加・削除を合わせて約900〜1,200行に収める。実token、Git config又は環境の変更、credential cache/disk/state、launcher、retry、redirect、別socket/TCP、server/Registry/Smart HTTP policy、Schema、依存、Kakesu runtime、live stateを追加又は変更しない。`git-receive-pack`、push、Approval、GitHub token取得、実`git clone/fetch/pull`、systemd/別UID/実GitHub/DNS/TLSの確認も対象外である。

## end-to-end設計

1. helperのcommand surfaceを`internal/gitcredential`へ閉じ、`cmd/git-credential-dev-agent`はそのRun境界だけを呼ぶ。
   - `--version`と`--help`/`-h`は既存binaryと同じ成功意味を維持する。operationはちょうど一つだけ受け入れる。zero又は複数argは固定usage failure（exit 2）とし、credential、repository、socket、下位errorを含めない固定診断だけをstderrへ出す。
   - `get`の成功はstdoutに正確に`username=x-access-token\npassword=<canonical cap_ handle>\n\n`だけを返す。stdoutへ別field、token、repository、URL、socket又は診断を足さない。
   - `get`の全拒否（入力不一致、dial/write/read/deadline/close失敗、非canonical response、issue拒否を含む）は、stdoutに正確に`quit=true\n\n`だけを返し、stderrには何も出さない。exitはGit helperの正常な応答として0に固定する。これにより次helper又はinteractive prompt/askpassへの探索を広げない。
   - `store`はGit側のpipe書込みを途中終了させないよう4 KiB上限内でstdinを読み捨てるが、属性を解釈・保存せず、socket接続・stdout/stderr出力なしでexit 0にする。未知の一個operationも同様にboundedに読み捨て、出力なしでexit 0にする。`erase`は下記の厳格なcanonical handle入力だけを読み、成功時は出力なしでexit 0にする。`erase`失敗はstdout/stderrを空にしたexit 1の固定failureにし、raw handleを出さない。

2. `get`/`erase`で使うcredential input parserを一つにし、security contextを曖昧に補完しない。
   - reader全体を4 KiBまでで一度だけ取得し、4 KiB超過、NUL、CR、blank終端後のbyte、終端なしの読取失敗を拒否する。blank line又は最後のlineのEOFを終端として同等に受理し、EOF直前の非空lineも一行として処理する。これによりScannerのtoken制限やblank後の隠しpayloadを導入しない。
   - `get`は`protocol=https`、`host=github.com`又は`host=github.com:443`、`path=owner/repo.git`の三属性を各一回だけ受理する。field順序は問わないが、欠落、duplicate、空値、未知field、`url`属性、userinfo、query/fragment、leading/trailing slash、port違い、host大小文字・IPv6・encoding、`.`/`..`/empty segment、非canonical owner/repo又は`.git`以外のsuffixをbroker到達前に拒否する。導出して送るrepositoryはsuffixを除いたcanonical `owner/repo`だけとする。
   - `erase`は`password=<canonical cap_...>`一属性だけを各一回だけ受理し、protocol/host/path/url/username又は他fieldを混在させない。正規handleは既存capability形式（`cap_` + 32 byte raw-URL-base64、再encode一致）だけである。これにより失効対象を一件に固定する。

3. 新規`internal/controlclient`はproduction socketへ一接続一操作だけを行い、server parserをimport又は共有しない。
   - Issueは固定順序のJSON `{"provider":"github","repository":"<canonical owner/repo>","operation":"github-git-read"}`を一回だけ作り、`POST /v1/capabilities HTTP/1.1`、単一の`Content-Type: application/json`、canonical `Content-Length`だけの固定wireを送る。既存server parserが許可しない`Connection`を含む追加headerは送らない。repositoryはJSON escapeが不要なcanonical grammarなので、client入力をwireへそのまま混入させない。
   - Revokeは`DELETE /v1/capabilities/<canonical handle> HTTP/1.1`と`Content-Length: 0`だけの固定wireを一回だけ送る。body、JSON、追加header、複数requestはない。
   - clientはabsolute Unix socketだけへdialし、dial、write、response header、body、EOF/closeの各phaseに固定した短いdeadlineを設定する。requestを全量writeし、成功responseを読み終えた後にEOF以外のextra byte又はclose不成立を拒否し、retry/fallbackをしない。
   - Issue成功は、唯一の`HTTP/1.1 200 OK`、固定順序で重複・余分なしの`Content-Type: application/json`、canonicalかつ上限内の`Content-Length`、`Connection: close`、その長さと一致する唯一の`{"handle":"cap_..."}` body、直後EOFだけを受理する。chunked、1xx/204/3xx/403/5xx、header casing/whitespace/framingの逸脱、複数JSON値、malformed/noncanonical handle、early EOF、body/extra bytesは固定拒否する。
   - Revoke成功は、唯一の`HTTP/1.1 204 No Content`、固定順序で重複・余分なしの`Content-Length: 0`と`Connection: close`、bodyなし、直後EOFだけを受理する。それ以外は固定拒否する。公開Errorは型・固定値だけとし、wire、handle、repository、socket、下位errorをformat/error文字列へ保持しない。

4. socket pathはconfigure値からhelperのlink時にだけ固定し、テスト用のproduction overrideを作らない。
   - `Makefile.in`にhelper専用のtarget-specific linker settingを加え、`$(runstatedir)/dev-agent-harness/egress.sock`を`internal/gitcredential`の非公開stringへ`-X`で埋め込む。ほかのbinaryの`command.Version` link flag、build/install/distcheckの意味は保持する。
   - commandとcontrol clientはCLI flag、credential attribute、environment、cwd、configからsocket pathを受け取らない。空又はnon-absoluteのlink値はdial前に拒否する。
   - testだけは`gitcredential`又は`controlclient` package内の非公開dial function seamを一時差替え、`net.Pipe`を返す。exportしたsetter、environment toggle、実socket listenerは導入しない。並列testを避けるか、seamの復元をmutexで直列化してraceを防ぐ。

5. READMEは手動でGit credential helperとして利用する境界を説明する。
   - helperがHTTPS `github.com`のallowlisted canonical read repositoryだけにsingle-use opaque passwordを返し、成功以外は`quit=true`で探索を止めること、`store`が保存しないこと、`erase`だけがhandleを失効することを記載する。
   - configure済みrunstatedirのsocketへ接続するが、environment/Agent inputで差し替えられないこと、Git configの自動変更・実Git操作・push・credential/token保存・live受理は対象外であることを明記する。実token、socket path、repository、handleを例・診断として掲載しない。

## AC対応

| AC-ID | 設計判断 | 変更パス | 実施順序 | 失敗時の扱い |
|---|---|---|---|---|
| AC-1 | 4 KiB bounded parserがblank/EOF終端を一様に読み、三つの`get` context又は一つの`erase` handleだけをcanonical化する。 | `internal/gitcredential/helper.go`、`internal/gitcredential/helper_test.go` | 1 | 解析失敗はbroker未到達。`get`は正確な`quit=true\n\n`、`erase`は非secret固定failure。 |
| AC-2 | private dial seam越しにabsolute fixed Unix socketへ一度だけ接続し、Issue/Revokeの唯一のrequest/response wireをstrictに検証する。 | `internal/controlclient/client.go`、`internal/controlclient/client_test.go` | 2 | timeout、close、header/body/handle/statusの任一不整合はretryなしの固定拒否。 |
| AC-3 | helperは成功時の二fieldだけを返し、全`get`失敗をGit探索停止responseへ固定する。 | `internal/gitcredential/helper.go`、`internal/gitcredential/helper_test.go`、`cmd/git-credential-dev-agent/main.go` | 1–3 | stdout/stderrにsecret又はcontextを漏らさず、prompt/他helperへfallbackさせない。 |
| AC-4 | `store`とunknown一argは入力をboundedに読み捨てるだけでsave/parse/dialなし、`erase`だけが一handleのDELETE/204 lifecycleを持つ。 | `internal/gitcredential/helper.go`、`internal/gitcredential/helper_test.go`、`internal/controlclient/client.go`、`internal/controlclient/client_test.go` | 2–3 | invalid erase又は204以外は保存・再試行・credential出力なしで失敗する。 |
| AC-5 | target-specific `-X`でconfigureのrunstatedirをhelper link時に埋め、private seamだけでhermetic testを行う。 | `Makefile.in`、`cmd/git-credential-dev-agent/main.go`、`internal/gitcredential/helper.go`、`internal/gitcredential/helper_test.go` | 3–4 | non-absolute/unset pathはdial前に拒否し、environment等から接続先を採用しない。 |
| AC-6 | 7 path/900〜1,200行、対象外、READMEのnon-live表現、race/focused/root/scope検査をcandidateに結び付ける。 | 許可済み7パス | 4–5 | 予算・許可パス・対象外逸脱はcandidateをMainへ戻し、取り込まない。 |

## 変更予定

| パス | 変更内容 |
|---|---|
| `tools/dev-agent-harness/cmd/git-credential-dev-agent/main.go` | scaffold command呼出しをhelper専用Run境界へ置換し、既存version/helpを維持する。 |
| `tools/dev-agent-harness/internal/controlclient/client.go` | 固定Unix dial、deadline/close、厳格Issue/Revoke wire、bounded canonical response parser、固定errorを実装する。 |
| `tools/dev-agent-harness/internal/controlclient/client_test.go` | `net.Pipe`/fake dialerでwire bytes、順序、deadline、partial I/O、close、各malformed responseと非漏洩を検証する。 |
| `tools/dev-agent-harness/internal/gitcredential/helper.go` | operation dispatch、bounded credential parser、canonical repository/handle、`quit=true` fail-close output、store/erase/unknown semantics、private seamを実装する。 |
| `tools/dev-agent-harness/internal/gitcredential/helper_test.go` | blank/EOF、duplicate/URL/host/path/NUL/CR/overlong/extra-byte拒否、success/failure出力、broker未到達、store/erase/unknown/arg数、non-leakを検証する。 |
| `tools/dev-agent-harness/Makefile.in` | helperだけへconfigure済みrunstatedir socketをlinkし、既存programのversion/build/install/distcheckを維持する。 |
| `tools/dev-agent-harness/README.md` | helperのopaque read-only境界、fail-close、固定socket、手動運用と明示的対象外を記載する。 |

## 検証計画

DEVは`net.Pipe`とin-memory credential streamsを用い、正規`get`のexact stdoutとIssue wire、正規`erase`のDELETE/204、socket deadline/closeを確認する。同時に、blank/EOF、duplicate/missing/unknown/URL属性、host/protocol/path/handle逸脱、NUL/CR/overlong/blank後extra、partial dial/write/read、chunked/extra/malformed/early-EOF/status response、`store`、unknown operation、zero/multiple argを壊すと失敗するnegative testを作る。各failure caseはdial/control未到達又は固定出力・secret非露出をassertし、実socket/実GitHub/実tokenを使わない。

candidate固定後のQA focused-rerun候補は次だけを一回実行する。QA_PLANがACごとに`focused-rerun`、`evidence-review`、必要なら`live-e2e`を理由付きで確定する。

```sh
cd tools/dev-agent-harness && GOCACHE=$PWD/.build/go-cache go test -race ./internal/controlclient ./internal/gitcredential
```

DEVはさらに`make check`、rootの`make check`、`make task-check TASK=TASK-0065`、`git diff --check`を実行する。QAはcandidate-bound source/test/wire/negative assertions、DEV/Reviewerの検査証跡、7 path/行数/対象外を独立監査する。opaque auth/protocolの高リスク又は証跡不足があれば`evidence-review`だけをPASSにしない。実Git clientのprompt suppression、systemd socket、別UID、実GitHub/DNS/TLSは承認済み環境と安全なcleanupがなければ`live-e2e`をblockedのままにし、hermetic PASSで代替しない。

## リスクと復旧

- Gitがfailure後に別credential sourceへ進むリスクは、`get`の全failureをbyte固定の`quit=true\n\n`かつexit 0にして抑える。固定出力の改変、空出力、下位errorの表示を許さない。
- Git属性のURL展開又はduplicateが別repositoryを指すリスクは、raw URLを解析せず、三fieldの完全一致とcanonical pathからのみrepositoryを導出して抑える。
- clientが寛容なHTTP parserでserver failure/extra dataを成功扱いするリスクは、request・responseのstatus/header order/framing/body/EOFを唯一値に固定し、`net.Pipe`のnegative matrixで抑える。
- test seamが本番socket overrideになるリスクは、同一package内の非公開dial seamに限定し、socket値のexport、CLI/environment/configを追加しないことで抑える。
- handle/repository/socket/実tokenがdiagnosticに出るリスクは、公開errorを固定値にし、stdout/stderrとfmt出力のnon-leak assertionを持つことで抑える。

復旧時は7許可パスのcandidate製品差分だけを戻し、既存egress control server及びscaffold helperへ復元する。新しいcredential保存、Git config、runtime state、実socket、外部GitHub操作を作らないため、追加cleanupはない。復旧後は上記focused race suite、root `make check`、`make task-check TASK=TASK-0065`、`git diff --check`を再実行する。

## 引き継ぎ条件

DEVはMain承認済みPLANと独立QA_PLANの後、7許可パスだけで同一candidateを一回固定する。ReviewerとQAは同じcandidateを相互のPASSを待たずに独立評価し、Mainだけがcompletion、`--no-ff --no-commit`検査、main統合と必要な環境依存確認を所有する。candidateは実token、push、Git設定/launcher、依存、Schema、runtime、生成物又はlive configurationを含めない。

## 未解決事項

- なし。packetで固定されたcontrol serverのwireとsocket位置を前提に実装する。

## main Agentレビュー

- [x] TASKの全AC-IDへ設計判断、パス、順序、失敗時の扱いを対応させ、条件本文を複製していない。
- [x] `get` fail-close bytes、blank/EOF・canonical input、strict Issue/Revoke response、link-time socketとprivate seam、store/erase/unknownの境界を固定している。
- [x] QA_PLANがTASK-firstで独立作成されている。
- [x] `dependency-ready reconciliation`と完了経路preflightが完了している。
- [x] DEV開始を承認した。
