---
task_id: "TASK-0028"
change_class: "product"
status: approved
qa_agent: "qa-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-07-20"
revision: 1
implementation_reviewed_at: "2026-07-20"
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0028 QA PLAN — Control ownership and lifecycle

## 独立したQA方針

本計画はDEV開始前にTASK本文だけから固定し、その後PLANと突合した。対象はControl-owned SQLite Storeのowner排他、Task lifecycle、event永続化、および終端時owner解放である。並行性、persistence、lifecycle、rollback/fail-closedは高リスク信号だが、二つの一時SQLite connection、決定的barrier、故意のSQL失敗注入を用いれば外部サービス・実権限・配置・安全な実環境cleanupなしに受入れ真実を完全再現できる。従って全ケースを`focused-rerun`とする。`evidence-review`だけでPASSしてはならない。

TASK-0029のexpected-version contract/progress更新、immutable snapshot/history、recovery read model、クラッシュ後recoveryを観測対象にも実装要求にも含めない。これらが差分、migration、fixture、API、検査に現れた場合はscope逸脱としてFAIL候補とする。

## 共通前提・固定観測

- QAはDEVがHANDOVERに固定した同一`candidate_commit`と`candidate_tree`をcheckout/照合して、ReviewerのPASSを待たずに開始する。不一致、未固定、又はdigest不一致なら全ケースをPASSにしない。
- 独立したtemporary SQLite databaseをケースごとに作成する。共有永続DB、実OS権限/auth、deploy、外部サービス、live restart/cleanupは使用しない。SQLite connectionは少なくとも二つを同時に開け、busy/conflictが自動retryされないことを観測可能にする。
- 各拒否/故意失敗の前後で、対象Task current state、progress/read model、active owner assignment、TaskEvent内容と最大sequenceをsnapshot比較する。失敗後の再読込でも同一でなければpartial writeとしてFAILにする。
- 各成功遷移でTask current stateと新TaskEventを同一DB transactionのcommit後に再読込し、sequenceが直前最大値から正確に1増えることを確認する。terminal成功時だけowner解放も同じcommit後に確認する。
- QAは次を候補に合わせて実行し、コマンド、fixture、cache条件、exit、出力digest、未実施理由を`QA_RESULT.md`へ残す: `cd core && go test -count=1 ./internal/control`、該当testを含む反復実行、`cd core && go test -race ./internal/control`、`cd core && go test ./...`、`cd core && go vet ./...`、`make check`、`make task-check TASK=TASK-0028`、`git diff --check`。実装に応じて対象test名を記録するが、全edge/rollback/二接続競合を除外しない。

## 受け入れ条件とケース

| ケースID | 受け入れ条件・境界 | `qa_execution_mode` / 理由 / fail-closed | 独立操作 | 期待結果・必要証跡 |
|---|---|---|---|---|
| QA-001 | 同一Agentへの二つの非終端Taskの並行割当 | `focused-rerun`。二接続・barrier付き一時SQLiteで排他真実を再現できる。barrier、二connection、又は同時競合を強制できなければblocked。 | ready Task A/Bと同一Agentを用意し、二connectionからbarrier同期で割当を開始する。成功者/敗者を入れ替えて反復する。 | 正確に一方だけ成功し、敗者はtyped conflict（自動retryなし）。敗者Taskのowner、Task state、event、progressにpartial writeなし。成功者だけに期待owner/event。connection/barrierログ、前後snapshot、error type、event sequence、DB digestを残す。 |
| QA-002 | 全許可edge | `focused-rerun`。table-driven一時SQLiteが全許可遷移とeventを決定的に検証できる。全13 edgeを個別に初期化・観測できなければblocked。 | owner付きTaskを各source stateへ用意し、ready→running、running→waiting/suspended/reviewing_completion、waiting→running、suspended→running、reviewing_completion→running/completed、ready/running/waiting/suspended/reviewing_completion→cancelledをそれぞれ要求する。 | 13 edgeだけ成功。各成功でcurrent stateと対応TaskEventが同一commit、sequenceは+1。非終端edgeはowner不変、completed/cancelledはowner解放。case別source/target、pre/post snapshot、event payload/sequence、transaction/readback digestを残す。 |
| QA-003 | 全拒否state pair、terminalからの再遷移、expected current state不一致 | `focused-rerun`。7状態の直積とCAS不一致をbounded tableで再現できる。49 pair全件（許可13以外の36件）又はそれと同値の列挙が欠ければblocked。 | completed/cancelledを含む全7×7 source/targetをtable-drivenで実行する。許可13 edgeを除く全36 edgeを拒否確認する。別途DB current stateと異なるexpected current stateで、許可edgeと拒否edgeの双方を試す。 | 全拒否がtyped validation/conflictで失敗し、Task state、progress、owner、event列と最大sequenceが前後完全不変。terminalにはoutgoing edgeなし。edge matrix、expected-vs-actual、pre/post snapshot digest、error typeを残す。 |
| QA-004 | waiting/suspended/reviewing_completionでのowner保持 | `focused-rerun`。非終端各stateと別Task再割当を一時SQLiteで決定的に再現できる。三state又は再割当拒否のいずれかを観測できなければblocked。 | owner付きTaskをrunningから各stateへ遷移し、同一Agentを別ready Taskへ割当する。 | 各stateで元assignmentは保持され、別Task割当はtyped conflict。別Taskのstate/owner/event/progressにpartial writeなし。各stateのsnapshotと再割当失敗証跡を残す。 |
| QA-005 | completed/cancelledのstate/event/owner解放原子性とforced rollback | `focused-rerun`。故意SQL失敗注入を一時SQLite transaction境界で再現できる。terminalの両種、state/event/releaseの各段階失敗、又はrollback readbackが欠ければblocked。 | reviewing_completion→completed、および各非終端→cancelledについて、state更新・event追加・owner releaseの途中になるよう故意失敗を注入する。失敗後に再読込し、次に注入なしで成功させる。 | 故意失敗はstate/event/owner/progress/sequence全てrollbackしgapなし。成功時のみterminal current state、terminal event、owner解放が一transactionで確定する。注入点、transaction error、pre/post/reopen snapshot、sequence、成功時digestを残す。 |
| QA-006 | owner解放後の同一Agent再利用 | `focused-rerun`。terminal commit直後の別Task割当を一時SQLiteで再現できる。terminal後のreadbackと再割当ができなければblocked。 | owner付きTaskをcompleted及びcancelledへ確定し、同一Agentを別ready Taskへ割当する。 | 新Task割当が成功し、旧terminal Taskとそのeventは不変。新旧assignment/event sequenceのsnapshot、terminal/reassignment順序、DB digestを残す。 |
| QA-007 | DB制約＋CAS、非partial write、retry禁止の回帰防止 | `focused-rerun`。二connection競合とDB constraint violationを決定的に観測できる。process-local同期だけで成功する、またはtyped conflict/constraint検証が不可視ならblocked。 | API前の競合窓を広げるbarrier/fixtureで二connection割当を反復し、制約違反を呼出し境界でtyped conflictへ分類することを確認する。failure後に自動retry回数、row/eventを読む。 | process-local mutex単独に依存せずDBで一件制限。呼出しは自動retryしない。失敗側にTask/owner/event/progressのwriteなし。反復回数、connection識別子、error classification、retry count、snapshot digestを残す。 |
| QA-008 | TASK-0029を含まない回帰・統合範囲 | `focused-rerun`。候補差分と既存control testを独立に検査できる。versioned contract/progress/recovery又はSchema/driver/transport等の対象外変更があればFAIL候補。 | candidate diff、migration/API/testを読み、`go test ./...`、`go vet ./...`、`make check`、`make task-check TASK=TASK-0028`、`git diff --check`を実行する。TASK-0029語彙・version column・contract/progress mutation/recovery変更と、driver/Schema/transport/owner transfer等を照合する。 | 既存control回帰なし、scope外実装なし、差分検査PASS。範囲外が必要なら`requirement_gap`としてPLANへ、実装済み逸脱なら`implementation_defect`としてDEVへ返す。diff digest、コマンドexit、対象外照合結果を残す。 |

## 失敗判定と実施不能時

- candidate/tree、case evidence、fixtureの決定性、二connection/barrier、SQL故意失敗注入、又はpre/post/reopen snapshotのどれかが欠ける場合、該当`focused-rerun`はblockedであり、DEV自己証跡や`evidence-review`でPASSに置換しない。
- owner競合で二方成功、敗者partial write、waiting/suspended/reviewing_completionでowner解放、許可外edge/terminal outgoingの受理、拒否時のstate/progress/owner/event/sequence変化、terminal成功のeventまたはrelease分離、forced rollback後のgap/残存更新、又はterminal後の再利用不能はFAIL候補である。
- 初期分類は、合意した期待値との差分は`implementation_defect`→DEV、TASK/PLANの矛盾・未定義は`requirement_gap`→PLAN、test手順/fixture期待の誤りは`qa_plan_defect`→QA、SQLite/runner/lock等で決定的fixtureを作れない場合は`environment_issue`→QAまたは基盤Task、既存control挙動の破壊は`regression`→DEVとする。最終分類・差戻し先はMainが根拠とともに決定する。
- 本Taskの候補修正はpersistence/concurrency/lifecycle/fail-closedに影響するため`qa_carry_forward`は禁止。Mainは影響caseのfocused rerun、又は影響を限定できないとき全再実行を選ぶ。

## TASK-first期待値とPLAN突合

TASK本文から固定した13許可edge、36拒否edge、非終端owner保持、terminalのstate/event/release原子性、二connection競合、forced rollback、owner再利用、TASK-0029除外はPLANのtransition table、invariants、verification/stop conditionsと一致する。PLANのtemporary SQLite、二connection/barrier、intentional SQL failure、全edge test、race/repetition/check commandsはこの計画の方法を狭めず具体化している。相違又は期待値変更はない（`expectation_changed: false`）。

## 案・マージ後確認

- DEVはHANDOVERへ各ケースの`candidate_commit`、`candidate_tree`、コマンド/test、fixture、cache条件、exit、artifact digest、未実施理由を固定する。QAは実際のcandidate/treeと照合する。
- REVIEWとQAは同一candidateから互いのPASSを開始条件にせず独立に実施する。
- Mainはcandidate修正後、carry-forwardを選ばず、本計画の影響ケースを再実行するか全面再実行する。
- 全ケースはhermetic temporary SQLiteであり環境依存`live-e2e`はない。Mainは`merge_tree == approved candidate_tree`を確認し、一致しない場合は変更影響を再評価する。

## 実装後の再確認

- [ ] candidate_commit/treeと全DEV case evidenceを照合した。
- [ ] QA-001〜QA-008を同一候補から独立に実施した。
- [ ] 差分にTASK-0029又は対象外の先取りがないことを確認した。
- [ ] 失敗・blockedを根拠付きで分類し、Mainが最終判断した。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | 2026-07-20 | qa-agent-terra-medium | TASK-firstの独立QA計画。PLAN突合で期待値変更なし。 | `approved by main-agent-sol-high` |
