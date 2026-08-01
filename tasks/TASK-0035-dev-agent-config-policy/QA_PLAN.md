---
task_id: "TASK-0035"
change_class: "product"
status: approved
qa_agent: "qa-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T10:06:00+10:00"
revision: 3
implementation_reviewed_at: "2026-08-01T10:26:47+10:00"
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0035 QA PLAN — Development Agent Harness configuration policy

## 方針と入力境界

本計画はTASK packetとREF-1〜REF-3だけから作成し、`PLAN.md`を読まず、入力にも期待値の根拠にも用いない。設定、ファイル検査、エラー/fail-closed、install生成物は高リスク信号だが、本Taskは標準library、candidate内の一時directory/file、ローカルsubprocessだけで完全に再現できる。そのため実環境権限・network・service操作を必要とする`live-e2e`は割り当てず、各受け入れ真実を独立に`focused-rerun`する。

- QA開始時、DEVがHANDOVERへ固定した同一`candidate_commit`と`candidate_tree`、managed-path diff digest、実装/テスト行数算定を照合する。未固定、不一致、対象外変更、証跡の欠落/不一致、テスト削除・期待値緩和、又は影響不明なら、該当ケースをPASSにしない。
- fixtureはcandidate worktree配下のテスト専用一時directoryで生成し、絶対pathはそのdirectory内だけを使う。所有者/permission検査は現在のQA userで行い、root、sudo、実`/etc`、実ユーザー、実service、外部networkには触れない。symlink、FIFO/directory等の特殊fileは作成可能なローカルfixtureだけを対象にする。
- `GOCACHE`、`GOMODCACHE`、`GOPATH`、`TMPDIR`、OS/Go version、umask、実行worktreeを各rerunで記録する。networkを必要としないこと、依存downloadが発生しないことを確認し、cache warm/coldの差で判定を変えない。各コマンドのexit、stdout/stderr digest（内容は秘密らしき入力を記録しない）、対象binary/config/test結果、fixture cleanup、未実施理由を`QA_RESULT.md`へ候補に束縛して記録する。
- `candidate_tree`から作る設定/テスト/配布物のSHA-256、`git diff --check`、`git diff --numstat`（手書き実装+testのみを分類して集計）、および`go test ./...`、`make check`、`make distcheck`の完全ログdigestを成果物digestとする。candidateが変われば全focused-rerunを再実行する。設定・install・fail-closedに触れる変更は`qa_carry_forward`対象外である。

## 受け入れ条件との対応

| ケースID | AC-ID | 観測・fixture | `qa_execution_mode` / 理由 | 期待結果と必要証跡 | fail-closed と初期分類 |
|---|---|---|---|---|---|
| QA-035-01 | AC-1 | configure済みの例をcopyしてowner-onlyのregular fixtureにし、setup binaryへ`check-config --config`を与える。stdout/stderrを別captureし、fixture内にpath、3 user名、credential様sentinelを埋める。 | `focused-rerun` / CLI、設定例、captureはcandidate内でdeterministicかつbounded。 | exit 0、stdoutはversion・deny default・validatedを表す安定summaryだけ、stderr空。全sentinel、path、設定全文/JSON断片が両streamに不在。candidate-bound CLI test、入力digest、stream digest、exitを保存する。 | success以外、stdout/stderrへの入力/秘密らしき値の漏洩、summaryの不安定化は`implementation_defect`候補。fixture/capture不能は`environment_issue`。 |
| QA-035-02 | AC-2 | valid fixtureから、未知top/nested field、同一object内のduplicate key、version変更、2 JSON values、値後の非空白trailing dataを一件ずつ生成する。各入力に固有sentinelを置く。 | `focused-rerun` / strict token/decodeとCLI診断をhermeticに再現できる。 | 各caseはnonzero、stdout空、stderrは入力値をechoせず原因分類を持つ。同一object duplicateを検出すること、未知field/version/trailingが相互に受理されないことをcase単位で記録する。 | zero、誤分類、stream漏洩、又はcase欠落は`implementation_defect`候補。要求外のJSON仕様追加が必要なら`requirement_gap`。 |
| QA-035-03 | AC-3 | path fieldsごとに空、相対、cleanでない絶対（`//`、`/x/../y`等）、同一pathの衝突を、usersに重複、空、不正開始文字、不正文字を、network defaultにdeny以外を生成する。V1に`allowlist`は存在しないため、field省略は許可を導出しないこと、`allowlist`（空配列・空objectを含む）の入力はunknown fieldとして拒否することを別fixture化する。 | `focused-rerun` / schema後の意味検証は一時JSONだけで完全再現できる。 | 各拒否入力はnonzeroかつ分類済み非漏洩stderr。field省略が暗黙の許可状態へ変換されず、`allowlist`は値にかかわらずunknown fieldとして拒否され、networkはdenyだけを受理することをstruct/CLI testで確認する。 | 受理、許可への暗黙変換、path/user/network検査の欠落は`implementation_defect`候補。 |
| QA-035-04 | AC-4 | local fixtureでsymlink、directory、FIFO等non-regular、regular 0640/0660/0606、64 KiBちょうど/64 KiB超を作る。open後にpathをsymlink/別fileへ置換するrace fixtureも、実装が提供するtest seamまたは繰返しbounded testで実行する。 | `focused-rerun` / file descriptor基準のfile-policyは権限昇格なしの一時fileで再現でき、外部作用を伴わない。 | symlink/non-regular/group-or-world writable/over-sizeは全てnonzeroかつ非漏洩stderr。64 KiB境界と安全modeのregular fileは期待どおり。race testはpath再解決ではなくopen済みFDのtype/size/modeを判断して危険側を拒否することを観測し、外部作用なしを確認する。 | FD基準raceのテスト/証跡なし、危険file受理、size/mode境界誤りは`implementation_defect`候補。OSがFIFO/symlink作成を禁じる場合は`environment_issue`としてPASSに代替しない。 |
| QA-035-05 | AC-5 | candidateのvalid/invalid unit・CLI testを独立実行し、unknown、duplicate、version、trailing-data、path、user、network、file-type、permission、sizeのcase名、入力、期待を対照する。各拒否判定を受理側へ反転する最小mutationを一時copyで作り、該当testが失敗することを確認する。 | `focused-rerun` / mutationは隔離copyでbounded、元candidateを変更せず検出能力を直接測定できる。 | 全カテゴリにpositive/negativeがあり、各mutationで対応testがnonzero。testのskip、削除、broad matcher、exitだけの弱いassertion、secret echoを許すassertionがないことをdiff/test bodyで監査する。 | 未網羅、mutationを検出しない、test弱体化は`implementation_defect`候補。mutation手順がcandidateを変更し得る/cleanup不能は`qa_plan_defect`又は`environment_issue`。 |
| QA-035-06 | AC-6 | setupの`--help`、`-h`、`--version`をREF-2 scaffold contractと比較する。`check-config`成功/失敗に加え、setupの未知/通常操作、broker、egress、approval、launcher、git-credentialの各通常起動を実行し、stdout/stderr/exitをcaptureする。 | `focused-rerun` / 全6 binaryはcandidate build outputでローカル実行できる。 | setupの既存help/version契約を保ち、導入された`check-config`だけが定義どおり動く。他5 binaryおよびsetupの非`check-config` operational invocationはzeroにならずfail-closedし、成功出力や外部作用を生まない。 | help/version後方互換破壊は`regression`候補。通常起動の成功、拒否緩和、外部作用痕跡は`implementation_defect`候補。 |
| QA-035-07 | AC-7 | clean candidateで`go test ./...`、`./configure --prefix=/usr/local --sysconfdir=/etc --localstatedir=/var --runstatedir=/run`、`make check`、`make distcheck`を実行する。別のtemporary `DESTDIR`に`make install`後、同じconfigure実行で展開されたinstall exampleをcopyせずそのinstall pathからsetup binaryで検証する。 | `focused-rerun` / configure/build/dist/installはローカルtemporary rootに閉じ、Taskが明示的に実行要求する。 | 全command exit 0。distcheck tarball内でも同じ検査が通る。install先のexampleは配置され、check-configがexit 0、非漏洩summaryとなる。実`/etc`等に書込みがないこと、生成済み`configure`が意図せず差分化されないことも記録する。 | command nonzero、tarball欠落、install例の不検証、install先外への書込み、生成物/設定例の不整合は`implementation_defect`候補。toolchain/autoconf欠落は`environment_issue`でありPASSに代替しない。 |
| QA-035-08 | AC-8 | candidate diffを許可path・生成物・fixture・文書・Task/Wikiへ分類し、手書き実装+testだけのadded/modified行を算定する。新module/import、socket/network/process/credential/state/service操作を静的検索し、REF-1の後続境界へ侵入していないかを確認する。 | `evidence-review` / 行数・scope・新境界はcandidate-bound diffとsource監査が真実で、再実行で増える確証ではない。 | 手書き実装+testが目安範囲内、又は超過見込み/新security boundaryがMainへ分割判断として明示済み。外部moduleなし、許可外pathなし、Credential/network/IPC/persistence/OS操作なしをdiff digestと検索結果で記録する。 | 1,200行超過又は新boundaryをMain判断なしで含めるのは`requirement_gap`/`implementation_defect`候補（Mainが最終分類）。行数分類不能、scope不明、candidate不一致はPASS不可。 |

## 横断検査と失敗分類

- QA-035-01〜07の実行前後で`git status --short`を取得し、candidate管理外の変更、fixture残留、実設定/外部service/networkへの副作用がないことを確認する。検査自体が外部作用を要求した場合は本Taskのscope逸脱として`requirement_gap`候補にする。
- 診断の具体文字列は未決であるため、安定性は同じ入力・candidate・環境で同じ分類/stream/exitになること、非漏洩はsentinel不在で判定する。未決の文字列を勝手に固定しない。
- 初期分類は、Task/REFと異なる実装またはmutation未検出を`implementation_defect`、既存help/version/fail-closed破壊を`regression`、Task/REFの曖昧さ・矛盾を`requirement_gap`、fixture/toolchain/権限制約を`environment_issue`、本計画の誤ったfixture/期待を`qa_plan_defect`とする。DEV起因とは仮定せず、最終分類はMainが証拠で決める。
- `evidence-review`のQA-035-08はcandidate commit/tree、diff digest、行数根拠、negative test有無、test弱体化の有無を全て突合できない限りPASSにしない。QA-035-01〜07は高リスクのためevidence-reviewだけではPASSにしない。

## 未実施・candidate・マージ後

- DEVはHANDOVERにケースID、`candidate_commit`、`candidate_tree`、実行command、fixture、cache条件、exit、成果物digest、未実施理由を固定する。QAはREVIEWの結果を開始条件にせず、同じcandidateから独立に開始する。
- どのfocused-rerunもcandidate変更、candidate/tree/digest不一致、必要toolchain/安全なtemporary fixture不能、又は未実施ならblocked又はFAIL候補であり、別ケースのPASSで代替しない。実環境依存の検証はTask対象外であり、live-e2eを偽装しない。
- 設定、parser、file policy、CLI、test、configure/installの候補修正は影響ケースが空にならないため`qa_carry_forward`を禁止する。Mainは影響ケースを再実行し、限定不能なら全ケースを再実行する。
- `merge_tree == candidate_tree`をMainが確認した後、本Taskは環境依存ケースを持たないため、上記candidate rerunを重複して全面実施する必要はない。tree不一致ならこの結論を使わず影響を再評価する。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | 2026-08-01 | qa-agent-terra-medium | TASK-firstの独立QA計画。AC-1〜AC-8へ8ケース（focused-rerun 7、evidence-review 1）を対応し、実環境依存は割り当てない。 | pending |
| 2 | 2026-08-01 | qa-agent-terra-medium | 計画レビューP1に従い、V1の`allowlist`は不存在であり、空を含む入力をunknown fieldとして拒否する期待へ一意化した。 | main-agent-sol-high / 2026-08-01T10:00:25+10:00 |
| 3 | 2026-08-01 | qa-agent-terra-medium | `qa_plan_defect`訂正: QA-035-07のconfigure fixtureを固定の明示prefix/sysconfdir/localstatedir/runstatedir引数へ変更し、未展開`${prefix}`を避けて同じconfigure済みinstall exampleを検証する。期待結果は不変。 | main-agent-sol-high / 2026-08-01T10:06:00+10:00 |
