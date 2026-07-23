---
task_id: "TASK-0025"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "sol-high"
approved_dev_profile_reason: "Done gateを横断し、安全契約を製品ゲート回避に使わせないfail-closed判定、Git履歴照合、互換性を同時に変更するため"
approved_dev_profile_risk_signals: ["safety_contract", "authorization_boundary", "cross_cutting", "migration"]
approved_by: "main-agent-sol-high"
approved_at: "2026-07-20"
planned_implementation_files: 1
planned_implementation_lines: 190
estimate_points: 1
---

# TASK-0025 PLAN

## 分類と実行境界

これは製品コード（`scripts/task/check-task.mjs`）とそのprocess testを変更する製品変更である。完全な `PLAN → QA_PLAN → DEV → 同一candidateから独立かつ並行のREVIEW/QA → merge後確認` を適用する。安全契約変更そのものを閉じる機能を実装するが、このTask自身は安全契約経路に分類しない。

DEVは `sol-high` を用いる。分類の誤りは製品変更を軽量経路で不正にDoneへ進める安全上の問題であり、field、Task本文、Git差分、既存のmerge/evidence条件を横断してfail-closedに保つ必要がある。

## 受け入れ条件の具体化

| 条件 | 観測方法 | PASS基準 |
|---|---|---|
| 分類の機械判定とfail-closed移行 | `change_class` を持つ最小fixtureで、未指定、未知値、Task本文との矛盾、許容外の差分を順に `checkTask` へ与える | `product` と `safety_contract` 以外、欠落、矛盾、検査不能な履歴はエラーとなる。既存fieldなしTaskは互換規則により `product` と解釈され、安全契約のDone gateを選べない。 |
| 製品変更のgate不変 | 現行製品fixtureを `product` としてDone相当まで満たし、REVIEW PASS、QA PASS/accepted-with-bugs、complete HANDOVER、Wiki receipt、review/merge/tested commit検査の各必須値を一つずつ欠落させる | 現在のエラー条件が全て維持され、`product` が安全契約用の証跡だけでDoneにならない。 |
| 安全契約の正直なDone gate | safety-contract fixtureに承認済みPLAN/QA_PLAN、独立計画レビュー、Main承認、対象検査、merged commit/tree証拠を用意してDoneを検査する | 製品 `REVIEW_RESULT.md` / `QA_RESULT.md` のPASS、製品用complete HANDOVER、Wiki receiptを要求せずDoneを許す。一方で各安全契約の必須証拠の欠落、pending、担当不一致、commit/tree不一致は拒否する。 |
| 分類偽装のnegative test | `change_class: safety_contract` としながら、Task契約が製品変更を宣言するケース、およびmerge対象に製品実装・test・runtime/build設定・Schema・宣言済み依存・生成製品入力/成果物を含めるケースを作る | どちらも安全契約gateへ進めず、どのTaskもfieldだけで製品gateを回避できない。許容する安全契約差分は規範/統制文書・template・skill・運用証跡に限り、曖昧なパス又は履歴は拒否する。 |
| TASK-0024回帰と閉鎖準備 | 実在するTASK-0024の既存merge commit/tree、承認済みPLAN、TASK-first QA_PLAN、独立計画レビュー/Main承認、統制文書検査の証跡をfixture又はread-only regression inputとして照合する | 既存の製品REVIEW/QA/HANDOVER/Wiki PASSを捏造・遡及編集せず、安全契約として検査可能である。閉鎖時にはMainが `change_class` と必要な安全契約証拠を記録して再検査する手順をHANDOVERへ残す。 |
| 既存証跡の互換性 | 既存Taskのfrontmatterにfieldを加えず `product` として検査するfixtureと、既存の6証跡ファイルのtask_id照合を実行する | 過去Taskを一括変換せず、既存製品Doneの解釈・必須gate・`bootstrap_exception`の例外を変更しない。 |

## 関連Wikiと判断

- 意味WikiはTask本文で未調査であるが、このTaskの判断根拠は下記の開発正本とTASK-0024の実証跡で閉じており、DEV開始条件にはしない。
- 製品リポジトリの`docs/development/development-process.md`と`docs/development/task-management.md`は、製品TaskのPLAN/QA_PLAN、同一candidateの独立REVIEW/QA、no-ff merge、HANDOVER/Wiki receiptを正本化している。
- TASK-0024は製品成果物を変更せず安全契約の計画・独立計画レビュー・Main承認・統制文書検査を完了したが、現行検査器は製品Done gateを一律適用するため、架空PASSを作らず `blocked` に置かれている。

## 設計

### 選択案

backlog Task recordに `change_class: product | safety_contract` を正本fieldとして追加する。fieldがない既存Taskは決定的に `product` と解釈する。新規の安全契約Taskは作成時から明示し、分類を変更する場合はTask契約、PLAN、QA_PLANの再承認とMainの理由・時刻を要求する（途中で静かに経路を切り替えない）。

`check-task` は共通のTask ID、状態、6証跡ファイル、承認済みPLAN/QA_PLAN、見積、担当者、phase・Git基礎検査を先に行う。分類後、Done判定を次のように分岐する。

| 分類 | 必須のDone証拠 | 明示的に要求しない証拠 |
|---|---|---|
| `product` | 現行のREVIEW PASS + `make_check: pass` + reviewed commit、QA pass/accepted-with-bugs + tested commit、complete HANDOVER、Wiki ingestion receipt、no-ff mergeとcommit到達性 | なし（既存の全製品gateを維持） |
| `safety_contract` | 承認済みPLAN、TASK-first承認済みQA_PLAN、独立した計画レビューのPASS、Main承認、対象統制文書検査のPASS、merged commitと`merge_tree`/candidate tree一致又は差異のfail-closed記録、分類と実差分の照合 | 製品REVIEW_RESULT PASS、製品QA_RESULT PASS、製品用complete HANDOVER、Wiki ingestion receipt |

安全契約の証跡は既存製品用frontmatterを意味だけ変えて流用しない。分類の正本は`backlog.yaml`の`change_class`とし、`PLAN.md`と`QA_PLAN.md`にも同じ値を記録して矛盾を拒否する。独立計画レビューとMainの分類承認は`PLAN.md` frontmatterの`planning_reviewed_by`、`planning_review_decision`、`planning_reviewed_at`、`classification_approved_by`、`classification_approved_at`、`classification_approval_reason`へ記録し、reviewer担当と承認時系列へ束縛する。閉鎖検査は`HANDOVER.md` frontmatterの`safety_checks`、`safety_checked_at`、`safety_check_digest`、`safety_candidate_tree`、`safety_merge_tree`へ記録する。`safety_checks`は`process_tests`、`contract_scope`、`docs_lint`、`make_check`の4件を全て`pass`とし、digestはcandidate tree、merge tree、正規化した検査集合から再計算する。製品用HANDOVERの`status: complete`は要求しない。検査器は空値・`pending`・担当不一致を拒否し、文字列の存在でなくfield値、日時、commit/tree、Git objectを検証する。Git差分のrename/copyは移動元の製品pathを隠せるため安全契約では拒否する。

差分照合は拡張子・SLOCでは判定しない。安全契約TaskのTask契約に「製品コード、test、runtime/build設定、Schema、製品依存、生成製品入力/成果物、外部観測可能な挙動を変更しない」旨の明示的な対象外宣言を要求し、merge commitのsecond parentとの差分をGitで得る。差分パスは規範/統制文書、Task template、skill、運用証跡の限定allowlistに一致することを要求する。製品入力に当たるpath、test/implementation、allowlist外、non-no-ff merge、parent/historyを確定できない場合は安全契約としてFAILする。allowlistは現行TASK-0024の実差分を基準に最小化し、将来の新しい安全契約pathはPLAN/QA_PLANの再承認を伴う明示的な検査器変更なしに通さない。

### 代替案と不採用理由

- frontmatterのfieldだけで経路を選ぶ: Task作者が製品変更を安全契約と偽装できるため不採用。
- ファイル拡張子又はSLOCだけで分類する: 文書/template/生成物にも製品入力があり、意味と実差分を判定できないため不採用。
- 安全契約Taskにも空の製品REVIEW/QA/HANDOVER/Wiki receiptを要求する: 架空のPASSを正規化し、TASK-0024の失敗を再現するため不採用。
- 全既存Taskへfieldと新証跡を遡及追加する: 公開済み証跡を破壊し、移行コスト・誤変更が大きいため不採用。
- 安全契約のDone判定を手作業だけにする: 再現性とnegative検査を失い、`task-check`の安全境界を残せないため不採用。

### 責務と境界

- main Agent: 分類、途中変更の再承認、Task状態、no-ff merge、TASK-0024の再検査・閉鎖を所有する。
- Planner / QA Agent: TASK-firstで安全契約の期待証拠、negative case、`qa_execution_mode`とfail-closed条件を独立に定義する。QAは製品QA PASSを代用しない。
- DEV: `check-task` とprocess testのみを実装し、candidate commit/treeと検査証拠をHANDOVERへ固定する。
- Reviewer / QA: 同一candidateから並行に、分類の偽装、gate緩和、テスト弱体化、TASK-0024回帰を独立評価する。
- `check-task`: Taskメタデータと運用証跡、product repositoryのGit履歴を読む検査器であり、Task本文の意図を推測して経路を救済しない。

### 不変条件

- `product` の既存DEV/QA/done gate、担当分離、DEV profile、branch/worktree、`bootstrap_exception`、no-ff merge検査を削除・弱化しない。
- 未指定、未知値、矛盾、空の安全契約証跡、Git履歴不明、allowlist外差分は常にfail-closedする。
- 安全契約では製品PASSを作らない。反対に、製品Taskは安全契約証跡だけでDoneにしない。
- 既存Taskのfield欠落は `product` であり、既存証跡を遡及変換しない。
- QA mode、carry-forward、FAIL分類、役割分離、Main所有Git、Lap30 event Schemaは変更しない。

### 失敗時・移行・互換性

- field不備・Task契約と差分の矛盾・不正merge形状・Git object不在・allowlist外差分は `requirement_gap` 又は `implementation_defect` 候補として、MainがPLAN/DEVへ戻す。環境でGit履歴を読めない場合は `environment_issue` 候補でありPASSに置換しない。
- 安全契約のplan review又はMain承認が不足する場合は `qa_plan_defect` / `requirement_gap` 候補としてDoneを拒否する。
- 既存fieldなしTaskは `product` 扱いなので、移行中に既存製品Doneが安全契約証跡の追加を要求されない。既存安全契約Taskを閉じるときだけ、Mainが明示fieldと必要証跡をappend-onlyで追加し、再検査する。
- TASK-0024は本Taskの製品gateが完了・mainへ統合された後に、Mainが安全契約のfield/証跡を補い、`make task-check TASK=TASK-0024`、必要な統制文書検査、`merge_tree`照合を行う。必要情報が欠ける場合はblockedのままにし、製品QA/HANDOVER/Wikiを捏造しない。

## 変更予定

見積もり対象は実装コード、Schema、設定ファイルだけとする。test、fixture、Task証跡、文書は見積もりから除外する。

| ファイル | 種別 | 概算変更行数 | 変更内容 |
|---|---|---:|---|
| `scripts/task/check-task.mjs` | implementation | 190 | `change_class` の正規化、既存fieldなし=`product` の互換規則、安全契約の証跡/merge tree/差分allowlist検査、分類別Done gateを追加し、現行製品gateを保持する。 |
| `scripts/task/development-process.test.mjs` | test | 250 | product regression、field不備、分類偽装、safety-contract PASS/各必須証跡欠落、TASK-0024互換fixtureを追加する。 |
| `docs/development/{development-process.md,task-management.md}` | documentation | 50 | `change_class`の所有者、移行、分類変更時再承認、安全契約Done証跡を正本に整合する。 |
| `templates/task/{TASK.md,PLAN.md,QA_PLAN.md,REVIEW_RESULT.md}` | template | 60 | 新規Taskの分類と、安全契約の計画レビュー/Main承認/差分検査を記録できる欄を追加する。 |

## 見積もり

```text
planned_implementation_files = 1
planned_implementation_lines = 190
file_score = ceil(1 / 3) = 1
line_score = ceil(190 / 200) = 1
raw_points = max(1, 1, 1) = 1
estimate_points = 1
```

テスト、fixture、文書、templateは規則により除外する。実装中にallowlistを支える共有ライブラリ、Schema、設定を追加する必要が判明した場合は、Main承認の上でPLANとbacklog見積を再計算する。

## 実装手順

1. DEVは承認済みPLAN/QA_PLANを読み、Task/PLAN/QA_PLANの分類fieldと安全契約証跡の保存場所を一つに固定する。現行製品gateのテストを先にcharacterizeする。
2. `check-task`に分類正規化（field欠落=`product`、許容値以外はエラー）と、Task契約・Git merge差分の安全契約照合を実装する。エラーはTask ID、欠落又は矛盾する証拠、拒否理由を含める。
3. common/DEV gateを変更せず、Done gateだけを分類別関数又は明確な分岐へ分離する。safety-contract側では計画レビュー、Main承認、統制文書検査、candidate/merge tree、差分範囲を全て検査する。
4. process testに最小Git repository fixtureを追加し、製品gateの回帰、safety-contract正常系、各失敗系、fieldなし互換、TASK-0024のknown merge/証跡形状を固定する。偽装ケースは製品実装/test pathを含めて必ず拒否する。
5. 規範文書とTask templateを実装契約へ整合し、`rg`で旧来の「全Taskに製品REVIEW/QA/HANDOVER/Wiki必須」の規範が安全契約へ残っていないことを確認する。
6. `make check`、対象Task check、fixture testを実行してcandidate commit/treeを固定し、HANDOVERへケースごとのcommand、fixture、cache条件、exit、digest、未実施理由を記録する。

## 検証計画

QA AgentはTASKだけから次のケースを事前に定義し、各ケースに `focused-rerun` を基本として割り当てる。これはNodeのtemp Git fixtureでhermetic・deterministic・boundedに分類とGit差分を完全再現でき、分類境界は高リスクで `evidence-review` 単独を許せないためである。実OS権限、外部サービス、実配置、restart/rollbackは対象に含めないため `live-e2e` は不要である。ただしfixtureがGit差分・tree到達性を再現できない場合はPASSにせず、計画を更新する。

| ケース | mode | 操作 | 期待結果・negative検出 |
|---|---|---|---|
| QA-001 product regression | `focused-rerun` | 現行製品Done fixtureで各REVIEW/QA/HANDOVER/Wiki/commit条件を一つずつ欠落 | 現行と同じ拒否を維持し、安全契約証跡では代替不可。 |
| QA-002 field/migration | `focused-rerun` | fieldなし、未知値、`product`、`safety_contract`、途中変更で再承認証跡なしを検査 | fieldなし=`product`、その他不正/不足はfail-closed。 |
| QA-003 safety-contract正常・欠落 | `focused-rerun` | 承認PLAN/QA_PLAN、独立計画レビュー、Main承認、統制文書検査、candidate/merge treeを備えたfixtureと、各値を欠落したfixtureを比較 | 正常系だけがDone。製品PASS/HANDOVER/Wiki receiptを要求しない。 |
| QA-004 classification spoof | `focused-rerun` | Task本文の対象外宣言なし、製品実装/test/config/Schema/dependency/生成入力、allowlist外、履歴不明を注入 | 全て安全契約経路を拒否し、分類fieldの偽装を検出する。 |
| QA-005 TASK-0024 regression | `focused-rerun` | merge commit/treeと安全契約の証跡をread-onlyで照合し、製品PASSなしの経路を検査 | 事実に合う安全契約証拠だけで閉鎖可能。欠落時はblockedで、捏造PASSは不要。 |

実装後、DEV、Reviewer、QAはそれぞれ同一candidateに対して少なくとも次を実行・記録する。

```sh
node --test scripts/task/development-process.test.mjs
make check
make task-check TASK=TASK-0025 WORK_ROOT=/home/ubuntu/git/agent-harness-work
git diff --check <approved-base>...<candidate>
```

さらにReviewer/QAは `git diff --name-status <candidate>^1 <candidate>^2` と安全契約fixtureの差分を読み、allowlistがTASK-0024を通しながら製品pathを許していないこと、`change_class`なしの既存Taskが `product` に留まること、テストがnegative caseを実際にfailにできることを独立確認する。各実行にはcandidate commit/tree、環境（Node version/temp directory）、cache条件、exit、出力digest、未実施理由を結び付ける。

## 未解決事項

- なし。TASK-0024の実際のno-ff mergeはsecond parentをcandidateとして扱い、履歴形状を確定できない場合にfirst-parentの広い差分へ黙ってfallbackしない。

## main Agentレビュー

- [x] 受け入れ条件が観測可能な結果、negative case、互換規則を含む。
- [x] `product` の既存gateを残し、安全契約がfieldだけでは通れない設計である。
- [x] TASK-0024の閉鎖が架空の製品PASSを要求しない。
- [x] QA計画がcase別mode、理由、fail-closed条件を作成できる。
- [x] 見積もりが規則どおりである。
- [x] DEV開始を承認した。
