---
task_id: "TASK-0057"
change_class: "product"
status: approved
qa_agent: "qa-agent-terra-medium"
qa_role: "independent-qa"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T17:28:43Z"
revision: 2
implementation_reviewed_at: "2026-08-01T17:53:59Z"
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0057 QA PLAN

## QA scope

期待値正本は `TASK.md` の `Planning input packet` だけとする。PLAN、実装案、DEV自己申告から期待値を導かない。candidate の許可pathだけを対象に、config V1 の strictness、command fixture、provisionのdirect Config validation、immutable runtime identity、Linux adapter と非Linux fail-closed、README 境界を確認する。provision manifest/actionの意味、service binary、socket activation、PeerBinder、brokerlistener/Session/Exchange composition、capability、systemd、credential、provider、HTTP/TLS、外部dependency は対象外である。

実LinuxのNSS、別broker/agent UID/GID、sysusers、service restart、VPS はこのhostで安全に再現できない `live-e2e` 境界であり、blocked のままとする。fake lookup/entropy、macOS common test、Linux cross-compile はその実環境事実のPASSに置換しない。

## 受け入れ条件との対応

| ケース ID | AC-ID | 確認内容と failure detection | 実施モード / 理由 | 証跡 |
|---|---|---|---|---|
| QA-001 | AC-1 | config V1 が必須 `identity.workspace_id` を独立copyし、1〜128 byte、先頭ASCII英数字、以降ASCII英数字/`.`/`_`/`-`だけを受理することを確認する。128/129 byte、empty、先頭不正、途中不正、欠落、未知field、duplicate key を個別fixtureで拒否し、既存version と users/path/network を変えないことを確認する。example、command fixture、provision direct validationとconfig testが同じ必須値を持ち、provision manifest/actionを変えないことを監査する。 | `evidence-review` / config/parser/consumer testとcandidate-bound DEV test証跡を差分・test本文から独立監査できる。QA focused commandはruntimeidentity packageに限定する。 | candidate diff、config/example/consumer test source、DEV command/result |
| QA-002 | AC-2 | constructor が agent/broker username と workspace ID をcopy・検証し、nil/zero/corrupt receiver を panic なしの固定非漏洩errorで拒否することを確認する。Linux `Resolve` では agent user、broker user、同名agent group の lookup が各一回だけで、順序には依存しない。call counter が lookup の欠落/重複を失敗させ、canonical decimal 以外、Go int 又はLinux uint32へのlossy値、zero/negative ID、root/current EUID broker、不一致broker EUID、同一UID、primary GID不一致を各々拒否することを直接確認する。 | `focused-rerun` / package-private fake lookup/EUID seam により、OS実lookupを使わず完全にhermetic・deterministic・boundedに受入れ真実とexact call countを再現できる。 | 指定command、実行test名、call-count fixture、正常/拒否case、exit |
| QA-003 | AC-3 | lookup 成功時に entropy reader が一回だけ16 byteを要求し、毎回新しい `agent-` + 32 lowercase hex の instance ID を生成することを確認する。短読取、entropy error、余分なentropy call、lookup/identity mismatch では partial identity/Subject を返さず固定非漏洩errorとなることを failure-detect する。success result の broker UID、agent UID/GID、Subject が整合し、各accessor が新しいcopyを返すため、返却値のmutation がResolver又は次の結果を汚染しないことを確認する。cache/retry/goroutine/log、username/workspace/UID/GID/lower error の診断露出を source/test audit で検出する。 | `focused-rerun` / deterministic fake entropy と lookup によりfresh-instance、exact length/call、copy とfailureを同じbounded package testで直接観測できる。 | 指定command、entropy call/length fixture、freshness/copy fixture、fixed-diagnostic assertions、exit |
| QA-004 | AC-4 | bundled hermetic test が QA-002〜003 の constructor/copy/corrupt receiver、lookup call count、numeric/EUID/user/group 境界、entropy call/length/failure、fresh ID/Subject copy、fixed diagnostics を実際に誤実装へ失敗させることを監査する。Linux adapter source がcandidateにありcross-compile対象であり、nonLinux が常にfail closedであることを確認する。Linux build-tag test をこのmacOS hostで実行したと主張せず、下記後半のcompile-only成功を実NSS/別UID実行の証拠にしない。 | `focused-rerun` / common tests はmacOSで実行し、Linux sourceは同一commandのcompile-onlyで検査できる。ただし実Linux integration はlive-e2e blocked。 | 指定command、test bodyのnegative assertion、Linux compile exit、candidate source |
| QA-005 | AC-5 | QA-001〜004のcandidate-bound証跡、testのnegative failure-detection、candidate diff を独立監査する。DEV実行の harness `make check`/`make distcheck`、root `make lint-docs`、candidate root `make check`、`git diff --check` はQAが重複実行せず、同一candidateのcommand/cwd/exitを照合する。許可path、追加＋削除1,000行以下、外部dependencyなし、config version変更なし、service compositionなしをdiffで検証する。 | `evidence-review` / root/harness checks はDEV一回のcandidate-bound証跡をQAが独立監査する対象であり、別candidate又はQA再実行のPASSではない。 | HANDOVER、DEV logs、candidate diff/numstat、README、test source |

## 一つの bounded focused rerun

candidate を固定後、QA-002〜004を次の一つのcommandで一回だけ実行する。前半はmacOS上のplatform-independent common testをcacheなしで実行し、後半はLinux build-tag sourceを実行せずcompile-onlyにする。`/usr/bin/true` はLinux binaryをhostで実行しないexecutorである。

```sh
cd tools/dev-agent-harness && GOCACHE=$PWD/.build/go-cache go test -count=1 ./internal/runtimeidentity && GOOS=linux GOARCH=amd64 GOCACHE=$PWD/.build/go-cache go test -run '^$' -exec /usr/bin/true ./internal/runtimeidentity
```

両段階ともzero exitが期待値である。test selection が零件、cache条件の欠落、candidate/tree不一致、required negative assertion の欠落、unbounded/non-deterministic fixture、又はLinux compile failureは該当caseをPASSにしない。後半のzero exitはLinux runtime、NSS、別UID/GID、sysusers又はVPSの証拠ではない。

## 境界・異常・回帰

- resolver は固定username/workspaceをconstructorでcopyし、任意identity API、numeric fallback、独自passwd parser、LDAP/remote NSS、getent/shell、cache/retry/timeout goroutineを追加しない。
- `Resolve` は一回ごとにlookupを三回（agent user、broker user、agent group各一回）とentropyを一回だけ行う。lookup順序を仕様化せず、count以外の順序依存testはPASS根拠にしない。
- numeric text はcanonical decimalのみで、leading zero、符号、空白、overflow、zero、negative、lossy conversion を受理しない。brokerはcurrent EUIDと完全一致かつnon-root、agentはdistinct positive UID、agent primary GIDは同名group GIDと完全一致でなければならない。
- error/Format はusername、workspace、UID/GID、lookup/entropy/platform errorを含まず、失敗時のresult/Subjectはpartialにならない。nonLinuxはidentityを返さずfail closedである。
- instance ID はservice-lifetimeごとにfreshで、`agent-` とlowercase hex以外を含まない。accessor返却copyのalias/corruptionは次のread又はResolveへ伝播しない。
- real Linux NSS、別UID/GID、sysusers、service restart、VPS は `live-e2e` blocked である。承認済みLinux環境、対象identity、実施手順と安全なcleanupが明確になるまで、他modeのPASSで置換しない。
- 許可path外、外部dependency、config version/既存path/users/network意味、service composition、生成された `harness.json.example`、実root/sudo又はhost user変更は regression/scope failure として扱う。

## Result criteria

candidate commitはHANDOVERだけを正本とする。QA_RESULTにはfocused-rerunのcommandと結果、QA-001〜005の判定、QA-006のblocked境界、findingがあれば再現根拠を記録する。testのskip・弱体化・広すぎるassertionはPASSにしない。

## 実装後の再確認

- [x] 同一candidateに対しQA-001〜005を独立に評価し、指定のfocused-rerunを一回だけ実行した。
- [x] lookup/entropyのexact call count、numeric/EUID/user/group拒否、fresh instance、accessor copy、fixed non-leak diagnosticsのfailure detectionを確認した。
- [x] config strictness とexample同期、許可path、追加＋削除1,000行以下、dependency/config version/service compositionなしを確認した。
- [x] root/harness checksはDEV candidate evidenceとしてのみ監査し、QAによる重複実行又はlive-e2e PASSと記録していない。
- [x] 実Linux NSS/別UID/GID/sysusers/VPSをblockedのままとし、期待値又はscopeを変更していない。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | 2026-08-02 | qa-agent-terra-medium | TASK-firstの独立QA計画。AC-1〜AC-5、single bounded rerun、live-e2e blocked境界を定義。 | `pending` |
| 2 | 2026-08-02 | main-agent-sol-high | command/provision同期をscopeへ反映し、重複candidate/tree記録を削除して承認。 | `approved` |
