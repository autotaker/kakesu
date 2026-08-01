---
task_id: "TASK-0052"
change_class: "product"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "luna-xhigh"
approved_dev_profile_reason: "新規の有限な listener composition と hermetic fixture に閉じ、既存 Session と隣接 package を変更しないため。"
approved_dev_profile_risk_signals: []
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T14:02:58Z"
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

# TASK-0052 PLAN

## AC対応

TASKの条件本文を再掲せず、`planning input packet`のAC-IDに設計を対応させる。

| AC-ID | 設計判断 | 変更パス | 実施順序 | 失敗時の扱い |
|---|---|---|---|---|
| AC-1 | `brokerlistener.Rules`はtrustedかつcontext協調の`PeerBinder`、TASK-0051互換の`Session`、`1..64`の`MaxConcurrent`だけを受ける。`New`はtyped-nilも検出し、検証済み依存と上限だけを保持するimmutable `Server`を返す。`Format`は固定型名だけを出す。 | `tools/dev-agent-harness/internal/brokerlistener/` | 1 | Rules、nil/zero/破損Server、nil context/listenerを固定公開errorへ畳み、依存・subject・address・下位errorをerror/Formatへ含めない。 |
| AC-2 | `Serve`は渡されたlistenerを所有し、rootから派生したrun contextとcancelを作る。bounded semaphoreのslotを先に取得してからだけ`Accept`し、accepted connごとに`WaitGroup`管理の同期処理を開始する。binderは一回だけSession前に呼び、検証・copy済みSubjectを非公開context keyへ束縛してSessionを一回だけ呼ぶ。 | `tools/dev-agent-harness/internal/brokerlistener/` | 2–4 | binder拒否又はinvalid subjectはSessionへ到達させず当該connをclose、slotを返却してacceptを継続する。queue、retry/backoff、worker pool、default dependencyを追加しない。 |
| AC-3 | `Resolver`はServerだけが書けるprivate keyを読むcontext-only値とし、subjectのUIDと両identifierを既存意味で検証して、返却前にも独立copyする。公開setterやcaller入力からの補完を持たない。 | `tools/dev-agent-harness/internal/brokerlistener/` | 3 | missing/wrong-type/invalid/cancelled contextは固定resolver errorだけを返す。RemoteAddr、listener address、HTTP/CONNECT/TLS/headerは読まず、connection間のvalue/sliceを共有しない。 |
| AC-4 | accepted connection処理はdeferでconn close、slot release、WaitGroup Doneを必ず実行し、binder/Sessionのpanicをrecoverしてconnection-local failureにする。Session errorも同じくaccept継続の理由にしない。unexpected Accept errorだけはserver failureとしてrun cancel、listener close、全処理drain後に固定server errorを返す。 | `tools/dev-agent-harness/internal/brokerlistener/` | 4 | connection-local error/panicのdetailを返却・Format・logへ出さない。accept errorでretry、別listener、部分成功を返さず、drain後のみ固定errorを返す。 |
| AC-5 | Serve開始時のcancel watcherはcaller contextだけを待ち、cancel/deadline時にlistener closeでAcceptを解除する。Acceptが返った後はcaller contextの状態を優先して正常cancelとunexpected errorを区別し、run cancel→全connection協調cancel→WaitGroup drainの順でnilを返す。 | `tools/dev-agent-harness/internal/brokerlistener/` | 4–5 | 非協調callbackをkillするgoroutine、timeout、background retryを作らない。協調callbackのreturnまでdrainし、watcher、accepted conn、listenerを残さない。 |
| AC-6 | package-local bounded fake listener、`net.Pipe`、counting/blocking/panic fake binder、context観測Sessionを用いるrace suiteを作る。公開READMEはこのin-memory listener boundaryとlive OS/network未実施範囲のみ説明する。 | `tools/dev-agent-harness/internal/brokerlistener/`、`tools/dev-agent-harness/README.md` | 5–6 | fake PASSをreal bind、OS peer identity、systemd、network namespace、VPS/live E2Eの根拠にしない。差分が800–900行目標または追加＋削除1,000行を超えるならcandidate前に設計を縮小・再確認する。 |

## 関連Wikiと判断

- REF-1の`connectsession.Session`は一受理済みconnectionのCONNECT/TLS/HTTPだけを所有する。listenerはその前にprivate subject contextを作って一回委譲するだけで、Sessionを変更しない。
- REF-2の`brokerhttp.SubjectResolver`消費者には、新しい`Resolver`をcontext-only実装として渡せる。HTTP request、remote addressその他の外部値からidentityを作らない。
- REF-3の既存Subject検証意味（UID正数、AgentInstanceID/WorkspaceIDが各1〜128 byteのcapability互換identifier）をlistener側の唯一のaccept条件として使い、値は束縛前とResolve返却前にcopyする。
- REF-4の自己申告でないtrusted identity境界を保つ。production PeerBinderが将来OS固有inputを生産する余地だけを残し、本Taskは実socket/OS/live環境を触らない。
- WikiはDEV許可パスではない。再利用可能な知識が生じた場合だけ、Mainが既存正本への同化とpost-merge処理を所有する。

## 補足設計

### 責務・境界・不変条件

- `Server`の長寿命stateは検証済みbinder、Session、MaxConcurrentのみとする。listener、root/run context、cancel、semaphore、WaitGroup、accepted conn、subjectはすべてServe又はconnection-localである。
- `Serve`の順序は receiver/input検証 → caller由来run context/cancelとcancel watcher開始 → slot取得 → Accept → connection処理をWaitGroupへ登録 → bind一回 → Subject validate/copy/private binding → Session一回 → conn close/slot release である。slotを取得できない間はAcceptを呼ばず、満杯接続はlistener backlogに留める。
- semaphore取得はcontext selectでcancelを同時に観測する。Accept失敗の分類は、caller/run contextがcancelledなら正常停止、そうでなければunexpected failureとする。後者は新規acceptを止め、listener closeとrun cancelを一度だけ行い、全connection処理が戻るまでwaitする。
- connection処理の最外deferでpanic isolation、conn close、slot release、Doneを固定する。binderとSessionは同期的にそのgoroutineで呼び、context協調というtrusted契約に依存するため、タイムアウトや別goroutineへの逃避で停止を偽装しない。
- private context keyはunexported type/valueとし、Subject構造体およびidentifierのbyte/stringデータをcopyして束縛する。`Resolver.Resolve`はcontext cancellationを先に拒否し、型・値を再検証してもう一度copyするので、binder/caller/Session/並行connectionが同じsubject可変データを共有しない。

### 代替案と不採用理由

- Accept後にslotを得て即closeする案は満杯時の接続挙動を変え、要求されたbacklogによる自然なbackpressureを破るため採用しない。
- public context key/setter、RemoteAddr、HTTP/TLS/headerからSubjectを補完する案はtrusted binderのみがidentity生産者という境界を壊すため採用しない。
- worker pool、独自queue、accept retry/backoffを置く案は長寿命stateと停止順序を増やすため採用しない。
- binder/Sessionをtimeout goroutineに隔離して強制的にServeを返す案はnon-cooperative dependencyのleakを隠すため採用しない。trusted context協調契約の下でdrainする。
- Linux credential、実Unix/TCP listener、systemdと同時に実装する案はhermetic accept lifecycleとlive OS identityを混在させるため採用しない。

### 移行・互換性

- 新規`internal/brokerlistener`とREADME追記だけに限定し、connectsession、brokerhttp、Transaction/Exchange/Policy/capability、config/CLI/dependency/build/generated artifactは変更しない。
- candidateの製品差分は初回800–900行を目標に、追加＋削除合計1,000行以下とする。planning/candidate/completionの標準3 commitsを保ち、新gate/check、digest転記、追加のcommit経路は作らない。
- fake listenerによる検証はproduction binding・OS identity・actual bind/systemd/VPSを保証しない。これらlive環境依存ケースはQA_PLANとpost-merge確認でblockedのまま明示する。

## 変更予定

| パス | 変更内容 |
|---|---|
| `tools/dev-agent-harness/internal/brokerlistener/` | immutable Rules/Server、trusted PeerBinder/Session interfaces、private Subject contextとResolver、bounded slot-before-Accept Serve lifecycle、fixed errors、そしてfake listener/net.Pipe/fake callbackのhermetic race testsを追加する。 |
| `tools/dev-agent-harness/README.md` | Serverが注入listenerを所有し、trusted binderからのprivate context identityをSessionへ渡す範囲と、actual bind/OS identity/systemd/live E2Eが対象外であることを追記する。 |

## 実装手順

1. `brokerlistener`の公開Rules、trusted `PeerBinder`/Session interface、fixed error、`New`、`Server.Format`を定義する。reflect等でtyped-nilを安全に拒否し、MaxConcurrentの1..64境界、nil/zero/破損receiverをtable testでfail closedにする。
2. unexported context key、Subject validate/copy helper、`Resolver.Resolve`を実装する。binderが返すsubjectのUID/identifierを検証してcopy後だけprivate bindingし、Resolveのmissing/wrong-type/invalid/cancelledと再copyをunit testする。
3. Serve-local semaphore、run context/cancel、WaitGroup、listener-close watcherを実装する。slot取得をAcceptより前に置き、cancelとslot待機をselectで区別する。accepted connは直ちにWaitGroup管理へ渡し、connection-local root-derived contextを生成する。
4. connection処理へbinder-before-sessionの同期フロー、panic isolation、deferによるconn close/slot release/Doneを追加する。binder拒否・invalid Subject・binder/Session error/panicが次のAcceptを妨げずSession到達ゼロ又は一回を保つことをcounter fixtureで固定する。
5. Accept error/cancel shutdownを完成する。unexpected Accept errorはlistener close→run cancel→drain→fixed server error、caller cancel/deadline由来closeは同じdrain後nilに分岐し、二重close・watcher残留を避ける。callbackを別goroutineに逃がさず、blocking cooperative fakeがcontext Done後にreturnすることでdrainを検出する。
6. bounded fake listenerと`net.Pipe`で、slot-before-Accept、上限、bind順序、一回呼出、rejected/invalid継続、parallel identity隔離、Resolver private性/copy、binder/Session panic・error、unexpected Accept failure、cancel/deadline/drain/conn-listener closeを`go test -race`で検証する。READMEを責務境界に合わせて追記し、行数を800–900目標・1,000上限へ収める。
7. candidate前にfocused package race、harness `make check`、`make distcheck`、README変更時の既存`pnpm lint:docs`、`git diff --check`、許可path/行数を確認する。最終working bytesをcandidate launcherへ渡し、root `make check`成功後に製品差分だけのcandidate commitを一回作る。planning/candidate/completion以外のcommit、新たな検査gate、既存Task/Lap Schema変更を作らない。

## 検証計画

- `go test -race ./internal/brokerlistener`でRules/New、typed nil、receiver、nil context/listener、private Resolverのmissing/wrong-type/invalid/cancelled、subject copyを検出する。
- bounded fake listenerのAccept counterとgate channelで、slotを取るまでAcceptされないこと、active SessionがMaxConcurrentを超えないこと、release後だけ次のAcceptへ進むことを確認する。`net.Pipe`の各accepted connがcloseされることも観測する。
- fake binder/Session countersとcontext観測でbinder一回がSession前であること、reject/invalidでSession未到達、Session/binder error又はpanic後も次connをacceptすること、parallel connのunique subjectがResolver経由で混線・aliasしないことを確認する。
- scripted listenerのunexpected errorでは固定server error、listener close、all context cancel、cooperative callback return後drainを確認する。caller cancel/deadlineではAccept解除、同一drain、nil returnを確認し、callback timeout/escape goroutineを用いない。
- candidate前はpackage race test、harness `make check`、`make distcheck`、README変更時の`pnpm lint:docs`、diff検査/行数上限を実行する。candidate後はroot `make check`を一回だけ実行し、actual listener/OS identity/systemd/VPSはlive-e2e blockedとして別モードPASSで代替しない。

## リスクと停止条件

| リスク | 抑制/検出 | 停止条件 |
|---|---|---|
| full時にAccept後rejectとなり上限とbackpressureの意味が変わる | semaphore acquisitionをAccept直前に固定し、gated fake listenerのAccept counter testを置く | queue、worker pool、Accept後のadmission/rejectが必要なら停止する。 |
| public/caller supplied値やmutable subjectがidentityを偽装・混線する | unexported key、validate/copy twice、parallel unique identity race fixtureを使う | public setter、RemoteAddr/header補完、Subjectをcopy不能な形で保存する必要があれば停止する。 |
| panic/error/cancelでconn、slot、watcher又は処理が残る | connection最外defer、single shutdown path、cooperative blocking fixtureによるdrain検証 | callback強制停止、timeout leak goroutine、drainを待たないreturnが必要なら停止する。 |
| accept errorをcancelと誤分類してserver failureを隠す | context状態をAccept後に確認し、scripted unexpected error/cancel closeを別caseで固定する | error detailの返却、retry/別listener、cancel以外を正常終了にする必要があれば停止する。 |
| hermetic証跡をlive OS/networkの成功に過大化する | READMEとQAで未実施境界を固定し、許可2パス/1,000行/既存checkを監査する | actual bind、Linux credential、systemd、外部network/config/秘密が必要になれば停止する。 |

## 未解決事項

- なし。

## main Agentレビュー

- [x] TASKの全AC-IDへ設計判断、パス、順序、失敗時の扱いを対応させ、条件本文を複製していない。
- [x] standard `net.Listener`/context/semaphore/WaitGroupだけのbounded accept、private subject binding、panic isolationと協調drainを具体化している。
- [x] 設計観点と代替案を検討している。
- [x] QA_PLANがTASK-firstで独立作成されている。
- [x] `dependency-ready reconciliation`と完了経路preflightが完了している。
- [x] DEV開始を承認した。

安全契約変更でv2契約を選ぶ場合は、コメントを外して`safety_contract_version: 2`と予定パス・生成パスの配列を記録し、DEV前に`make task-preflight TASK=TASK-0052`を実行する。変更しない種別は空配列とし、通常の予定パスと生成パスを重複させない。独立計画レビューのPASSとMainの分類承認をフロントマターへ記録する。分類変更時はTask、PLAN、QA_PLANを再承認し、承認者と時刻を更新する。
