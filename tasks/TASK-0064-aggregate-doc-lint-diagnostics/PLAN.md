---
task_id: "TASK-0064"
change_class: "product"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "luna-xhigh"
approved_dev_profile_reason: "既存docs検査の実行制御だけを変更する3パスの局所的なNode/Makefile差分で、Credential、network、OS権限、runtime、Schema境界を変更しないため。"
approved_dev_profile_risk_signals: []
approved_by: "main-agent-sol-high"
approved_at: "2026-08-02T02:05:12Z"
classification_approved_by: "main-agent-sol-high"
classification_approved_at: "2026-08-02T02:05:12Z"
classification_approval_reason: "root Makefileの品質検査実行挙動とprocess testを変更する製品変更であるため。"
# safety_contract_version: 2
# safety_contract_planned_paths: []
# safety_contract_generated_paths: []
---

# TASK-0064 PLAN

## AC対応

TASKの条件本文を再掲せず、`planning input packet`のAC-IDに設計を対応させる。

| AC-ID | 設計判断 | 変更パス | 実施順序 | 失敗時の扱い |
|---|---|---|---|---|
| AC-1 | `lint-docs`は既存3検査を固定配列で定義するNode runnerだけを起動する。runnerはその配列を順に一回ずつ起動し、結果にかかわらず次の検査へ進む。 | `Makefile`, `scripts/run-doc-lints.mjs` | terminology validator → Markdown textlint → `git diff --check` | 個別FAILを記録しても後続を起動し、全完了後に集約non-zeroで終了する。 |
| AC-2 | 子processの標準入出力は継承して各検査の既存stdout/stderrを即時表示する。終了status又は起動例外を失敗として集約し、失敗なしだけzeroとする。 | `scripts/run-doc-lints.mjs` | 各起動の終了/例外を受けて集約状態を更新し、最後に一度だけ終了する。 | 起動例外も失敗として扱い、診断を表示したうえで残る検査を続行する。 |
| AC-3 | commandとargsはrunner内の固定配列とし、shell文字列・repository入力からの組立て・shell実行を使わない。既存検査の内容・順序以外の品質gateを変更しない。 | `Makefile`, `scripts/run-doc-lints.mjs` | Makefileをrunner呼出へ置換し、runnerが既存3commandを所有する。 | 想定外の実行エラーも集約失敗にし、retry/cache/parallel/autofix/追加ログへ代替しない。 |
| AC-4 | fake runnerを注入できる小さな実行境界をrunnerに用意し、既存`test:process`列挙済みtestが起動順、起動回数、集約exit、例外継続を検査する。 | `scripts/run-doc-lints.mjs`, `scripts/task/development-process.test.mjs` | first-fail-continues、multiple-fail、all-pass、spawn-errorを独立ケースで実行する。 | 各ケースは期待する後続実行またはexitを失えば失敗する。candidate gateがroot `make check`を一回実行する。 |
| AC-5 | 差分をTask指定の3パスに限定し、runnerとtestを小規模に保つ。製品本体、Schema、依存、生成物を変更しない。 | 許可3パスのみ | 実装前後に許可パスと概算行数を確認する。 | 範囲又は約250行の上限を外れる必要が出た場合はDEVを停止してMainへ再計画を依頼する。 |

## 関連Wikiと判断

- 意味Wiki・判断は未調査であり、本Taskのplanning input packetにも追加参照はない。実装判断はTASKのREF-1/REF-2とACに限定する。

## 補足設計

### 代替案と不採用理由

- Make recipeの各行へ失敗無視を加える案は、exit集約と起動例外の扱いを明確にテストしにくく、shell展開を避ける要件にも適さないため採用しない。
- 検査の並列起動は出力順と固定順を崩し、対象外であるため採用しない。
- 出力をcaptureして最後にまとめて表示する案は、即時診断表示の設計観点に反するため採用しない。

### 責務・境界・不変条件

- Makefileは既存の`lint-docs`入口をNode runnerへ配線するだけとする。`make check`の入口とdocs以外のtargetは変更しない。
- runnerは検査結果を解釈せず、固定されたcommand/argsを順番に起動して成否を集約する。子processのstdout/stderrを変換・秘匿・追加入力ログ化しない。
- testはfake runnerに限定し、実際のlintルール、用語集、Markdown対象、依存を変更しない。

### 移行・互換性

- 呼出し側は引き続き`make lint-docs`および`make check`を使用する。全PASS時のzeroと既存3検査の内容・順序を維持し、失敗時だけ後続診断も同一実行で得られるようにする。

## 変更予定

| パス | 変更内容 |
|---|---|
| `Makefile` | `lint-docs`の3つの直列recipeを、新規Node runnerを一回呼び出す配線へ置換する。 |
| `scripts/run-doc-lints.mjs` | shellを使わず既存3commandを固定順に逐次実行し、stdioを継承してすべての失敗・起動例外を最終exitへ集約するrunnerを追加する。 |
| `scripts/task/development-process.test.mjs` | 既存`test:process`から常時実行されるtestへ、fake runnerによる4つの受け入れケースを追加する。 |

## 実装手順

1. `Makefile`の`lint-docs`入口を保ち、検査定義・制御をrunnerへ一元化する最小配線を作る。
2. runnerに固定command配列と逐次非shell実行を実装する。各子processはstdio継承とし、normal non-zeroとspawn errorを同じ失敗集約へ反映しても、残る固定commandを必ず実行する。
3. runnerの実行境界をfake runnerへ差し替え可能にし、production起動はその境界に実際の子process起動を渡す。
4. 既存`test:process`列挙済みunit testで、最初のFAIL後に3件すべてが実行されること、複数FAILでnon-zeroとなること、全PASSでzeroとなること、spawn error後も後続を実行してnon-zeroとなることを確認する。
5. 許可パス、依存変更なし、概算250行以内を確認してcandidateを固定する。範囲超過時は、変更を増やさずMainへ戻してPLANを改訂する。

## 検証計画

- DEVはunit testを直接実行し、4ケースのfailure-detection能力（各期待を壊した際にtestがFAILすること）を確認する。
- DEVは`make lint-docs`を実行し、既存3検査が固定順に一回ずつ出力されることと、正常時zeroを確認する。
- candidate gateはroot `make check`を一回実行し、DEVはfocused unit test、`make lint-docs`、`git diff --check`を実行する。失敗はQAガイドラインに従って実装、環境、既存不良等へ分類し、成功の代替証拠にしない。
- QA_PLANではAC-1〜AC-3のrunner挙動とAC-4のtest強度をcandidate-bound evidence-reviewまたは、hermeticかつboundedならfocused-rerunへケース単位で割り当てる。AC-5は許可パス・行数・依存差分のevidence-reviewとする。live-e2eは実OS権限、外部作用、実配置を含まないため不要と明記する。

## 復旧

- candidate前に失敗した場合は、変更を許可3パス内に留めたまま原因を切り分け、fixed command、固定順、stdio継承、集約exitの不変条件を再確認して修正する。
- candidate固定後にREVIEWまたはQAが失敗を示した場合、Mainがcandidateを再固定し、影響するunit testと`make check`を再実行する。検査の削除・緩和、失敗無視、retry、skipでの回避はしない。
- runner導入が既存`make lint-docs`の成功/失敗exitまたは検査内容を維持できない場合、Makefile配線と新規runner/testをcandidateから取り除き、既存の3行recipeへ戻して本Taskを再計画する。復旧に許可外パス、依存、ルール変更が必要なら実装を止めてMain承認を求める。

## 未解決事項

- なし

## main Agentレビュー

- [x] TASKの全AC-IDへ設計判断、パス、順序、失敗時の扱いを対応させ、条件本文を複製していない。
- [x] 設計観点と代替案を検討している。
- [x] QA_PLANがTASK-firstで独立作成され、testが既存`test:process`から実行されるようスコープを補正した。
- [x] `dependency-ready reconciliation`と完了経路preflightが完了している。
- [x] DEV開始を承認した。

安全契約変更でv2契約を選ぶ場合は、コメントを外して`safety_contract_version: 2`と予定パス・生成パスの配列を記録し、DEV前に`make task-preflight TASK=TASK-0064`を実行する。変更しない種別は空配列とし、通常の予定パスと生成パスを重複させない。MainはPLAN/QA_PLANの意図・スコープ・受け入れ経路を確認し、分類承認をフロントマターへ記録する。分類変更時はTask、PLAN、QA_PLANを再承認し、承認者と時刻を更新する。
