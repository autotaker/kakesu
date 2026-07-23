---
task_id: "TASK-0025"
status: approved
qa_agent: "qa-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-07-20"
revision: 3
implementation_reviewed_at: "2026-07-20"
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0025 QA PLAN

## TASK-first baseline

この計画は、PLAN、実装差分、既存の結果証跡を参照せず、TASK.mdだけからDEV開始前に作成した独立ベースラインである。TASK-0025自身は`task-check`とprocess testを変更する製品変更であり、完全な製品変更経路で評価する。実装対象となる安全契約TaskのDone判定について、製品変更Taskの既存統制を削除・緩和せず、正直な計画・レビュー・Main承認・対象検査・merge証跡だけで閉じられることを確認する。

## mode選択と共通fail-closed規則

| mode | このTaskでの使用条件 | fail-closed条件 |
|---|---|---|
| `evidence-review` | 候補に束縛されたprocess test、fixture、差分、DEV証跡から低リスクの静的契約を独立監査できるcase | case ID、candidate commit/tree、command、fixture、cache、exit、artifact digest、未実施理由の欠落、不一致、test削除/弱体化、又は影響不明ならPASS不可 |
| `focused-rerun` | 分類/Done gateを壊す高リスク境界を、隔離した一時`WORK_ROOT` fixtureでhermetic・deterministic・boundedに完全再現できるcase | fixtureが実際のchecker経路を通らない、決定性/上限を示せない、又は実OS/外部環境依存が判明した場合はPASS不可（必要なら`live-e2e`へ昇格） |
| `live-e2e` | 実Git merge/tree、実worktree登録、実配置又は外部/OS権限が正しさに不可欠なcase | 承認済み隔離環境、前提、cleanup/rollbackがない場合はblocked。別modeのPASSで代替しない |

分類field、互換規則、Done gateは偽装で製品経路を回避できる高リスクの安全境界である。従って、正常判定の自己申告や文言検索だけでは合格にせず、正常・unknown・矛盾・製品偽装を含むprocess fixtureを独立に再実行する。FAILはログと再現性を確認して`implementation_defect`、`qa_plan_defect`、`requirement_gap`、`environment_issue`、`regression`のいずれかに分類し、DEV起因と決めつけない。

## 前提・必要証跡

- QAは固定された同一`candidate_commit`/`candidate_tree`に対して、REVIEWのPASSを待たず独立に開始する。
- DEV evidenceはcase ID、candidate commit/tree、実行command/test、環境/fixture、cache条件、exit、artifact digest、未実施理由をHANDOVERへ記録する。QAはそれらと候補差分を突合し、negative検出能力とtest弱体化の有無を確認する。
- 分類の正本は`backlog.yaml.change_class`、計画review/Main承認は`PLAN.md`の`planning_reviewed_by`、`planning_review_decision`、`planning_reviewed_at`、`classification_approved_by`、`classification_approved_at`、閉鎖検査は`HANDOVER.md`の`safety_checks`、`safety_checked_at`、`safety_check_digest`、`safety_candidate_tree`、`safety_merge_tree`とする。
- `backlog.yaml`、PLAN、QA_PLANの分類値一致、reviewer担当への計画review束縛、`classification_approval_reason`と承認時系列、`process_tests`/`contract_scope`/`docs_lint`/`make_check`の完全な検査集合、candidate/merge treeと検査集合から再計算したdigestを個別にnegative検査する。rename/copyによる許可pathへの移動は安全契約として拒否する。
- `focused-rerun` fixtureは、実運用のTask証跡を変更せず、`checkTask`又はCLIの実経路を使用する一時ディレクトリで作る。各fixtureの入力、期待errors又はexit、削除/cleanup結果を残す。
- 安全契約Taskの対象検査は少なくともprocess test、`make task-check TASK=TASK-0025`、変更契約のscope検査、`git diff --check`である。`make check`は候補の広域退行確認として実行するが、製品QA PASSを捏造する根拠にはしない。

## 受け入れ条件との対応

| ケースID | 受け入れ条件・操作 | mode / 理由 | 期待結果 | 必要証跡・fail-closed |
|---|---|---|---|---|
| QA-001 | 正本field `change_class` を持つ新規Task、fieldなしの移行前Task、未知値、TASK本文/field/実差分が矛盾するTaskをfixture化し、classificationの所有者、承認時点、途中変更時の再承認を検査する。 | `focused-rerun` / parserと互換規則はhermetic fixtureで完全再現でき、unknown/矛盾は経路回避に直結する。 | 許可値は`product | safety_contract`だけ。fieldなしの既存Taskは決定的に`product`となり、未知・矛盾・分類変更未再承認はfail-closedでDone不可。拡張子/SLOCのみでは安全契約と判定しない。 | fixture内容、candidate commit/tree、checker command、exit/errors、digest。未知/未指定を暗黙に`safety_contract`扱いする、本文又は差分照合なし、再承認なしの変更を許す場合FAIL。 |
| QA-002 | 製品変更fixtureを、正しいproduct分類でDoneにする正常系と、safety-contractへ偽装して製品REVIEW/QA/HANDOVER/Wikiを省略するnegative系で実行する。 | `focused-rerun` / 既存製品gateの弱体化は高リスクだが、閉じたfixtureで再現可能。 | product DoneはREVIEW PASS（make check/reviewed commit）、QA PASS又はaccepted-with-bugs、HANDOVER、Wiki receipt、commit/tree検査を全て保持する。製品差分/製品分類の偽装は安全経路で閉じられない。 | 正常/negative fixture、期待errors、exit、candidate binding、testが旧gateをassertする差分証跡。必須gateの一つでも省略可能、又はproductをsafetyとして通すならFAIL。 |
| QA-003 | 安全契約fixtureを、承認済みPLAN/QA_PLAN、TASK-first性を示す証跡、独立計画レビュー、Main承認、対象検査、merged commit/treeを満たしてDoneにし、製品REVIEW/QA/HANDOVER/Wiki不在のnegativeでないことを確認する。各必須項目を一つずつ欠落させる。 | `focused-rerun` / safety Done truthはdeterministicなtask evidence fixtureで完全再現できる。 | 上記の安全契約gateだけを全て満たす場合にDone。製品PASS/HANDOVER/Wikiを要求して正直な安全契約Taskをblockしない。一方、PLAN/QA_PLAN/計画review/Main承認/対象検査/merge commit/treeの欠落は個別に拒否する。 | fixture matrix、各caseのerrors、exit、candidate/tree と merge tree の値、対象検査の記録。製品用の証跡を安全Task PASS条件にする、又は安全必須証跡を欠いて通るならFAIL。 |
| QA-004 | TASK-0024の公開済み証跡を遡及改変しないread-only fixtureを検査し、本Taskの統合後にMainが`change_class: safety_contract`と不足する安全契約証拠をappend-onlyで補う閉鎖手順を確認する。 | `focused-rerun` / 実在の代表証跡に対するchecker回帰であり、ローカルfixtureと既存Task読取りでbounded。 | field追加前のTASK-0024は互換規則により`product`として正確にblocked/errorとなる。append-only補足後は、架空の製品REVIEW/QA/HANDOVER/Wiki PASSなしで、実在する安全契約の必要証跡とmerge treeが揃う場合だけDone可能。 | candidate QAではread-only fixtureのexit/log、使用した証跡一覧とdigest、移行分岐を残す。統合後はMainによるappend-only差分と`make task-check TASK=TASK-0024`のexit/logを残す。公開済み証跡の書換え、又はfieldなしを暗黙に安全契約扱いするならFAIL。 |
| QA-005 | classificationを`product`/`safety-contract`間で途中変更したcandidate、PLAN又はQA_PLAN期待値変更、candidate/tree不一致、REVIEW修正後候補をfixture化し、再承認・rerun/carry-forward・merge_tree記録を検査する。 | `focused-rerun` / candidate bindingとfail-closedは安全上重要だが、Git fixtureで決定的に再現できる。 | 分類又は安全契約の意味変更はPLAN/QA_PLAN/計画review/Main承認を再取得し、影響caseを再実行する。`qa_carry_forward`はCF-1〜CF-7が全て成立し影響case集合が空の場合だけ許可。`merge_tree`不一致又は不明は閉鎖不可/再評価となる。 | old/new commit/tree、全diff/digest、影響case集合、CF判断、review/check結果、merge tree比較。QA FAIL、acceptance/QA_PLAN変更、auth/secret/sudo/PAM、IPC/Schema/config/dependency、lifecycle/error/fail-closed、test弱体化、影響不明でcarry-forwardを許すならFAIL。 |
| QA-006 | candidate差分とprocess testを独立監査し、分類別Done gate、既存Task互換、TASK-0024 regression、negative fixture、FAIL分類、スコープを横断確認する。 | `evidence-review` / QA-001〜005の再実行証跡をcandidate-boundで監査できる統合case。 | `scripts/task/check-task.mjs`とprocess testは同一語彙・同一分類規則を実装し、既存証跡を遡及破壊せず、純粋証跡保守を架空Task化しない。対象外のQA mode、role separation、Main Git ownershipは変更されない。 | `git diff --check`、candidate diff、`make check`、`make task-check TASK=TASK-0025`、QA-001〜005のresult/digest、negative assertionの確認。証跡不足、candidate/tree/digest不一致、test削除/弱体化、scope外変更、影響不明ならevidence-review PASS不可。 |

## forward / negative scenarios

| ID | 種別 | 条件 | 期待結果 |
|---|---|---|---|
| FT-001 | 正常 | 新規product Taskの完全な製品証跡 | product Doneのみ許可。 |
| FT-002 | negative | product Taskの`change_class`を`safety_contract`に偽装し、REVIEW/QA/HANDOVER/Wikiを欠落 | Done拒否、偽装根拠をエラーとして示す。 |
| FT-003 | 正常 | 承認済み計画・独立計画review・Main承認・対象検査・merge commit/treeを持つsafety Task | 製品PASS文書を要求せずDone許可。 |
| FT-004 | negative | safety TaskでPLAN、TASK-first QA_PLAN、計画review、Main承認、対象検査、merged commit/treeのいずれか一つを欠落 | 個別にDone拒否。 |
| FT-005 | 境界 | `change_class`未指定の移行前Task | `product`として決定的に解釈し、`safety_contract`への暗黙フォールバックをしない。 |
| FT-006 | negative | 未知classification、TASK本文/差分との矛盾、分類だけを途中変更 | fail-closed、再承認/再評価が完了するまでDone不可。 |
| FT-007 | 回帰 | TASK-0024の既存証跡と、Mainが行うappend-only移行fixture | 移行前は`product`としてblocked/error、移行後は安全契約必須証跡だけで閉鎖可能。遡及書換え又は架空製品PASSを要求しない。 |
| FT-008 | negative | candidate/tree又はmerge_tree不一致、carry-forward禁止条件、削除/弱体化したprocess test | PASS不可。再実行、再評価又は適切なFAIL分類を要求する。 |

## exact checks

実装後、同一candidateで以下を実行し、全commandの環境、cache、exit、要約、artifact digestを記録する。fixture作成コマンドは実装されたtest harnessに合わせてQA時に具体化するが、checker本体を迂回するmockだけでは不可とする。

```sh
git diff --check <approved-base>...<candidate>
node --test scripts/task/development-process.test.mjs
make check
make task-check TASK=TASK-0025
```

さらにcandidate差分について、`change_class`の許可値、fieldなし=`product`の移行規則、unknown/矛盾fail-closed、product/safetyの別gate、計画review/Main承認/対象検査/merge tree、TASK-0024 fixture、product偽装negative assertionを人手で対応付ける。process testが一部だけを実行する場合は、QA-001〜QA-005ごとに実行したtest名とfixtureを記録する。`make task-check TASK=TASK-0024`は、本Task統合後にMainがappend-only移行証跡を追加した時点で実行し、candidate QA中の未移行TASK-0024にPASSを要求しない。

## 実施不能・merge後

- 実OS権限、実配備、外部作用、実restart/rollback/cleanup、環境固有integrationが新たに必要と判明したcaseは`live-e2e`へ昇格する。承認済み隔離環境と安全なcleanupがなければblockedであり、`focused-rerun`又は`evidence-review`で代替PASSしない。
- 未実施command、原因、影響case、再現条件、残余リスクを記録し、`environment_issue`等の候補分類をMainへ渡す。実装・要件・QA計画の矛盾は、都合よく期待値を書換えず`implementation_defect`、`requirement_gap`、又は`qa_plan_defect`として戻す。
- Mainはapproved candidate treeと`merge_tree`を比較する。同一で環境依存caseがなければ重複の全面確認を省略できる。不一致は影響caseを再評価し、環境依存caseは同一でもcase単位のマージ後確認を要する。

## 実装後の再確認

- [ ] 同一candidate commit/treeにDEV証跡、REVIEW、QAを束縛して確認した。
- [ ] QA-001〜QA-005の正常・negative fixtureが実checker経路で実行され、弱体化がない。
- [ ] PLANとの突合を行い、TASK-first期待値との差分を記録した。
- [ ] 期待値又は範囲を変える必要があれば、Main承認前にPASSにしない。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | 2026-07-20 | QA Agent | TASK-first初版。分類・product保全・safety Done・TASK-0024・candidate/merge treeの独立caseとnegative fixtureを固定。 | `pending` |
| 2 | 2026-07-20 | QA Agent / Main Agent | PLAN突合blockerを解消し、既存6証跡内のfield配置を固定。期待結果と試験範囲は不変更。 | `approved` |
| 3 | 2026-07-20 | Main Agent | Review FAILを受け、rename/copy、検査集合/digest、reviewer束縛、分類mirror/理由/時系列のnegative境界を具体化。TASKの期待結果は不変更。 | `approved` |
