---
task_id: "TASK-0065"
change_class: "product"
status: approved
qa_agent: "qa-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-02T02:32:01Z"
revision: 1
implementation_reviewed_at: "2026-08-02T03:01:56Z"
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0065 QA PLAN

## 方針

期待値の正本は `TASK.md` の `Planning input packet` だけとする。PLAN、実装案、DEV自己申告は期待値の根拠にしない。同一candidateを、credential input のstrictかつboundedな正規化、issue/revoke control wire、single-dial/deadline/close、Gitのfail-closed output、link-time socket固定、非漏洩、既存harnessの不変という観点で独立に評価する。

net.Pipe/fake dialerとin-memory credential streamsに閉じた試験だけをhermetic PASSの根拠にする。実OS Unix socket/UID、実Git helper invocation、GitHub token/DNS/TLS、GitHub、VPS、install後のsystemd環境は確認しない。承認済み実環境と安全なcleanupがないため、それらは後続の `live-e2e` を `blocked`/`not-run` として分離し、focused rerun又は証跡監査のPASSで代替・主張しない。

## 受け入れ条件との対応

| ケース ID | AC-ID | 確認内容と failure detection | 実施モード / 理由 |
|---|---|---|---|
| QA-001 | AC-1 | `get`が一個のoperation argとbounded credential inputだけを受け、blank line又はEOFで終わる正規の `protocol=https`、exact `github.com` 又は `github.com:443`、canonical `owner/repo.git`からだけrepositoryを一意に導出することを確認する。zero/複数arg、missing/duplicate/conflicting protocol/host/path、URL属性、non-HTTPS/別host/port、dot/empty/encoded/noncanonical path、NUL/CR、overlong line/input、blank終端後のextra bytesをbroker dial前に拒否し、credentialを返さないことをfailure-detectする。 | `focused-rerun` / in-memory input とcounting fake dialerで正規受理、全拒否、broker非到達を決定的・boundedに再現できる。 |
| QA-002 | AC-2 | issue clientがconfigure済みabsolute Unix socketへ一回だけdialし、dial/read/write deadlineを設定して一接続一操作でcloseすることを確認する。exact issue requestが `provider=github`、canonical repository、`operation=github-git-read`だけを送ること、唯一のbounded 200 JSON handle responseだけを受理することをwire bytesと順序で確認する。broker非到達、timeout、early EOF、二回dial、close漏れ、chunked、extra/duplicate header、Content-Length不正、overlong/extra body/bytes、204/403/5xx、malformed JSON、unknown field又は非canonical/non-`cap_` handleを固定拒否することをfailure-detectする。 | `focused-rerun` / net.Pipe とfake dialer/clock又はdeadline recorderで、networkなしにexact bytes、one dial、deadline、close、strict responseを観測できる。 |
| QA-003 | AC-3 | 成功した `get` だけがcredential formatで `username=x-access-token` とcanonical opaque `password=cap_...`を返すことを確認する。unmatched context、input/control/response failureではhandleを含むcredentialを返さず、Gitの探索停止を表す固定fail-closed結果だけを出すことを確認する。stdout/stderrが実token、handle、repository、socket path、credential入力、下位errorを含まないこと、failure時に他helper/prompt、retry、別socket/TCPへ進まないことをfailure-detectする。 | `focused-rerun` / controlled streams、counting dialer、sentinel token/errorを使い、正常outputと全failureの非漏洩・到達禁止を外部secretなしに検出できる。 |
| QA-004 | AC-4 | `store`とunknown一個operationが上限内stdinを読み捨てるだけで、入力を解釈・保存・転送せずstdout/stderrなしのsilent successとなることを確認する。`erase`がcanonical opaque handle一件だけから同じ固定socketへexact DELETE revoke wireを送り、唯一の204だけを成功として受理することをwire bytes、順序、one dial/deadline/closeで確認する。zero/複数argは固定usage failureとする。malformed/noncanonical/multiple/missing handle、duplicate/overlong/CR/NUL/extra bytes、broker非到達、strict response逸脱/EOF/非204は保存又は別操作へ進まず、診断非漏洩のまま拒否することをfailure-detectする。 | `focused-rerun` / in-memory credential stream、net.Pipe、fake dialerでbounded read、store非到達、eraseのexact revoke wireと全negative boundaryをhermeticに再現できる。 |
| QA-005 | AC-5 | `Makefile.in` がconfigure済み `runstatedir` からdefault socketをlink-timeに固定し、生成binaryでその値が有効なことを確認する。credential input、CLI flag、environmentがsocket接続先を変更できないことをfake dialer/linked valueで検出する。`--version`/`--help`、既存program、build/install/distcheckの意味を回帰させず、runtime environmentによるproduction socket overrideを追加していないことを確認する。 | `focused-rerun` / configure/build fixtureとprivate dial seamはdeterministicにlink-time値とoverride拒否を検出できる。build/install/distcheckのcandidate-bound結果はQA-006で独立監査する。 |
| QA-006 | AC-6 | candidate diffが承認済み7 pathだけで約900〜1,200行に収まり、real token、push、launcher/config mutation、dependency、Schema、Kakesu runtime、live stateを含まないことを監査する。focused race suiteがQA-001〜005のstrict input/wire/response、broker非到達、deadline/one dial/close、non-leak、link-time socketを実際に壊すと失敗し、skip/削除/弱体化されていないことを確認する。DEVのharness `make check`、root `make check`、`git diff --check`およびReviewerによる同一candidateの監査がcommand/cwd/resultを持つことを確認し、QAはfull checkを重複実行しない。 | `evidence-review` / candidate diff、README/Makefile/test本文、DEV/Reviewerのcandidate-bound証跡でscope、検査実施、testのfailure-detectionと既存harness不変を独立監査できる。 |
| QA-007 | 対象外（AC-2/AC-5/AC-6の環境依存境界） | 実OS Unix socketとpermissions/別UID、configure/install後の実runstatedir、実Git credential helperのprompt/fallback挙動、実GitHub token/DNS/TLS、GitHub read、systemd、VPSを確認する。 | `live-e2e` / `blocked`/`not-run`。承認済み実環境と安全なcleanup手順がTaskにない。QA-001〜006のPASSはこのケースをPASSにしない。 |

## 一つの bounded focused rerun

candidate固定後、QA-001〜005を次の一回だけ実行する。

```sh
cd tools/dev-agent-harness && GOCACHE=$PWD/.build/go-cache go test -race ./cmd/git-credential-dev-agent ./internal/gitcredential ./internal/controlclient
```

対象testは少なくとも、正規 `get`、duplicate/overlong/CR/NUL/extra-byte のdial前拒否、exact issue/revoke wire、strict 200 JSON handle/204 response、broker非到達、deadline/one dial/close、`store`/unknown operation、fail-closed outputと非漏洩、link-time socket固定とenvironment override拒否のいずれかを壊すと失敗しなければならない。zero exitだけでは不十分である。candidate不一致、test欠落/skip/弱体化、実OS/GitHub/VPS/real tokenへの依存、unbounded/non-deterministic fixture、必要なnegative assertionの欠落は該当ケースをPASSにしない。

QAはこのfocused Go race suiteだけを再実行する。harness `make check`、root `make check`、build/install/distcheck、`git diff --check`はDEV実行とReviewerのcandidate-bound証跡監査で確認し、QAのfocused PASSに置き換えない。

## 境界・異常・回帰

- credential contextはprotocol、host、pathの完全一致だけで導出し、URL属性、duplicate値、暗黙default、入力由来socket、prompt/askpassで補完しない。rejectはissue/revoke broker到達前であり、success以外ではcredential探索を広げない。
- control clientは固定Unix socketだけを一回dialし、deadlineを設定して一接続一操作でcloseする。retry、redirect、fallback socket、TCP、keep-alive、environment/CLI overrideを追加しない。issue/revokeのrequest/responseはserver parserと共有せずclient側でstrictに検証する。
- stdout/stderr/diagnosticにreal token、opaque handle、repository、socket、入力、lower errorを出さず、credential、handle、inputをdisk/cache/stateへ保存しない。`store`は完全なno-opである。
- Makefile変更はconfigure済み `runstatedir` のlink-time socket固定だけに留める。既存binaryのhelp/version、configure、install、distcheckの契約を変更しない。
- live-e2eは `blocked`/`not-run` として残す。focused failureはcandidate、test fixture、実行環境、requirement、証跡に分類し、DEV不具合とは決めつけない。

## 実装後の再確認

- [ ] HANDOVERの`candidate_commit`を基に、同一candidateをQA-001〜006で独立に評価した。
- [ ] QA-001〜005のfocused Go race suiteを一回だけ実行し、input拒否、issue/revoke wire、strict response、broker failure、deadline/one dial/close、fail-closed non-leak、socket固定のfailure detectionを確認した。
- [ ] QA-006のcandidate diff、許可7 path、行数、対象外差分、DEVのharness/root `make check`・build/install/distcheck・`git diff --check`、Reviewer証跡を独立監査した。
- [ ] QA-007をhermetic PASSと分離して `blocked`/`not-run` のまま記録した。
- [ ] 期待結果または範囲を変更した場合、main Agentの承認を得た。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | 2026-08-02 | qa-agent-terra-medium | TASK-firstの独立QA計画。credential入力、issue/revoke strict wire、response、fail-closed non-leak、socket固定、race/Makefile検証とlive境界を定義。 | `approved` |
