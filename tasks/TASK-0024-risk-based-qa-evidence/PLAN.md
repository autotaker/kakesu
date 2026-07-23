---
task_id: "TASK-0024"
change_class: safety_contract
status: approved
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "sol-high"
approved_dev_profile_reason: "必須開発統制を横断変更する安全契約であり、複数の規範文書・template・skillの一貫性を保つ必要があるため"
approved_dev_profile_risk_signals: ["contract", "cross_cutting"]
approved_by: "main-agent-sol-high"
approved_at: "2026-07-20"
planning_reviewed_by: "reviewer-agent-terra-medium"
planning_review_decision: pass
planning_reviewed_at: "2026-07-20"
classification_approved_by: "main-agent-sol-high"
classification_approved_at: "2026-07-20"
classification_approval_reason: "製品成果物を変更せず、QA実施方法という必須開発統制だけを変更したため"
planned_implementation_files: 0
planned_implementation_lines: 0
estimate_points: 1
---

# TASK-0024 PLAN

## 分類と実行境界

これは製品成果物、製品test、runtime/build設定、製品Schema、依存を変更しない**安全契約変更**である。必須統制（QAの実施方法、並行性、結果引継ぎ、マージ後確認）を変更するため純粋な証跡保守には分類しない。TASK-firstの独立`QA_PLAN.md`、独立した計画レビュー、Main承認を必須とし、製品DEV、製品`REVIEW_RESULT.md`、製品`QA_RESULT.md`のPASSは作成しない。

## 受け入れ条件の具体化

| 条件 | 観測方法 | 根拠 |
|---|---|---|
| modeの事前割当とfail-closed | `QA_PLAN` templateとQA文書が全caseに`evidence-review`、`focused-rerun`、`live-e2e`の一つと理由を要求する。高リスクでもhermetic/deterministicかつbounded fixtureで受け入れ真実を再現できるcaseだけを`focused-rerun`とし、実OS権限/auth stack、実配備/外部作用/環境固有統合へ依存するcaseを`live-e2e`とすることを相互検索する | 高リスクを一律分類せず、再現可能性と受け入れ真実の所在でfail-closedに判定する |
| candidate-bound DEV証跡 | HANDOVER templateとプロセスがcase ID、commit/tree、command/test、環境/fixture、cache、exit、digest、未実施理由を同じ名称で要求し、QAの監査観点を明記する | DEV自己承認ではなく、QAが完全性・弱体化・負例を独立評価する |
| REVIEW/QAの並行評価 | プロセス、役割、両skillが同一candidateを明記し、互いのPASSを開始条件にしないこと、各結果の対象commit/tree記録を要求する | 独立性と順序を混同しない |
| 修正後のcarry-forward | `qa_carry_forward`の必須記録、許可条件、禁止条件、Main専有判断、focused/full rerunへの分岐が全規約とtemplateで一致する | QA FAIL、高リスク、差分不明を省略しない |
| マージ後の扱い | candidate treeとmerge treeの同一性を確認し、環境依存caseが無ければ重複全面QAを省略可能、該当caseがあれば限定確認を維持すると規定する | merge後の外部状態・実権限リスクを残す |
| 後方互換性 | 新規任意欄/sectionは既存Task証跡を無効化しない。Lap30 event Schemaと既存JSONLは変更しない | 公開済み証跡の意味とSchema v1を保存する |

## 関連Wikiと判断

- 正本: `docs/development/development-process.md`、`qa.md`、`agent-roles.md`、`AGENTS.md`、運用リポジトリの`AGENTS.md`。
- TASKの判断を採用する: QA独立性は「独立した期待値、test妥当性、証跡判断」であり、無差別な全面再実行回数ではない。carry-forwardはQAの削除でなく、旧判定を新candidateへ移すMainの記録付き判断である。
- `run-efficient-task-delivery`と`run-lap30`は上位の`AGENTS.md`/開発契約を迂回せず、同じ分類、役割分離、FAIL分類、Main所有Gitを参照するよう整合させる。

## 設計

### 選択案

1. 用語を一度だけ規範的に定義し、全ての文書・template・skillで同じ表記を生成する。`qa_execution_mode` は `evidence-review | focused-rerun | live-e2e`、`candidate_commit` は評価対象コミット、`candidate_tree` はそのtree、`merge_tree` は実際のマージ結果、`qa_carry_forward` はMainが旧新commit/tree、diff、影響case、再実行証拠、理由を残す引継ぎ記録とする。`docs/glossary.yml`を機械生成する用語inventoryの正本として更新し、`validate-terminology.py --write`の出力が変わる場合だけ`docs/99-glossary-index.md`を更新する。既存の`tested_commit`等は互換的に候補識別子として説明し、既存証跡の書換えを要求しない。
2. QA AgentはDEV前にTASKだけから全caseの期待結果・mode・理由・必要証跡を作る。DEV後はcandidate-bound証跡を監査し、`evidence-review`では証跡品質とtest妥当性を確認する。高リスクな挙動でも、受け入れ真実をhermeticかつdeterministicに、bounded fixtureで完全再現できる場合は`focused-rerun`を許可する。対象例はIPC/Schema parser、race/並行性、lifecycle state machine、persistence/error injection、fail-closed negative pathであり、QAは指定境界を独立再実行する。受け入れ真実が実OS権限/auth stack（sudo/PAMを含む）、install/deploy/generated configの実配置、外部サービス/side effect、実restart/rollback/cleanup、環境固有integrationに依存する場合は`live-e2e`を必須とし、実環境境界を独立確認する。再現性、環境、またはcleanupの安全性が不明なら`focused-rerun`へ降格せず、`live-e2e`のblockedとしてPASSさせない。
3. candidate確定後、ReviewerとQAは別Agentで同一candidate/treeを並行評価する。Reviewerは実装/test品質、QAは受け入れ対応と証跡を評価し、双方が相手のPASSを前提にしない。Mainのみが両結果、tree同一性、必要な修正/再評価、Gitを判断する。
4. REVIEW修正でcandidateが変わった場合、Mainは`qa_carry_forward`、focused rerun、full rerunを選ぶ。carry-forwardは非挙動で、全許可条件が明示・証明され、影響caseと再実行証拠を記録できる場合だけに限る。QA FAIL対象、受け入れ条件/QA_PLAN変更、認証認可、秘密、sudo/PAM、IPC/Schema/設定/依存、並行性/lifecycle/persistence/error/fail-closed、test削除/弱体化、差分不明は必ず再実行へfail-closedする。
5. マージ後は`merge_tree == approved candidate_tree`をMainが確認する。同一かつ環境依存caseなしなら全面QAを重複実施しない。install/deploy/config生成、実権限、外部作用、rollback等の環境依存caseはマージ後もcase単位で確認する。tree不一致はcarry-forwardせず、影響を再評価する。

### 代替案と不採用理由

- 従来どおりQAを常にマージ後の全面再実行に固定する: 高リスクには安全だが、candidate-bound証跡の独立監査とcase別リスク判断を活用できず、TASKの目的を満たさない。
- REVIEW PASS後にQAを順次開始する: 同一candidateで独立並行に評価する受け入れ条件に反し、待ち時間だけを理由に統制を緩和する余地も残す。
- REVIEW修正を低リスクという自己申告だけでQA省略する: 差分と禁止条件が検証不能で、認可・秘密・fail-closed等の安全境界を取り逃す。
- Lap30 event Schemaへ新fieldを加える: 対象外であり、固定Schema v1と既存JSONLの意味を変えるため採用しない。

### 責務と境界

| 主体 | 責務 | 禁止/境界 |
|---|---|---|
| QA Agent | TASK-first QA_PLAN、mode/rationale、candidate証跡監査、必要な独立rerun、FAIL候補 | DEV証跡の自己承認、期待値/範囲の無承認変更、Git判断 |
| DEV Agent | candidateへ結び付く実行証跡をHANDOVERへ残す | mode選択の自己承認、QA結果の作成、Git書込み |
| Reviewer | 同一candidateの実装/test品質を独立評価し対象commit/treeを記録 | QA PASS待ち/代替、carry-forward承認 |
| Main Agent | candidate/merge tree照合、carry-forwardまたはrerun選択、承認、FAIL分類、Git | 子Agentへの承認・merge委譲、禁止条件の例外化 |

### 不変条件

- QA_PLANはDEV前、TASKだけから独立作成し、DEV/Reviewer/QAの分離とMain所有Gitを保持する。
- `evidence-review`はtestを実行したという主張だけでPASSにせず、commit/tree binding、test弱体化、negative case、証跡完全性をQAが確認する。
- 高リスクcaseの`focused-rerun`はhermetic、deterministic、bounded fixtureの三条件が揃い、受け入れ真実を完全再現できる場合だけ許可する。IPC/Schema parser、race/並行性、lifecycle state machine、persistence/error injection、fail-closed negative pathは条件を満たすと証明できる場合の例であり、名称だけで自動許可しない。
- 実OS権限/auth stack（sudo/PAM）、install/deploy/generated configの実配置、外部サービス/side effect、実restart/rollback/cleanup、環境固有integrationへ受け入れ真実が依存するcaseは常に`live-e2e`とする。環境または安全なcleanupを用意できない、もしくは安全性が不明なcaseはPASSにせず、`live-e2e`のblocked状態を維持する。
- 不確実性、QA FAIL、受け入れ/QA_PLAN変更、またはtree不一致は省略側へ倒さない。高リスクというラベルだけではmodeを決めず、上記discriminatorを適用する。
- FAIL分類は`implementation_defect | qa_plan_defect | requirement_gap | environment_issue | regression`のまま維持し、最終分類と差し戻し先はMainが所有する。
- `make check`、影響する文書/template/skill検査、`make work-check`は統制文書の検証であり、製品QA PASSの証拠にはしない。

### 失敗時・移行・互換性

- 新規Taskは更新templateを用いる。既存Taskは新しいfieldを遡及必須にせず、以前の証跡を有効とする。途中TaskはMainがQA_PLAN改訂としてmode/rationaleとcandidate bindingを追加承認してから新手順へ移る。既存の公開証跡、Lap30 JSONL、Schemaは書換えない。
- 証跡不足、digest不一致、cache条件不明、candidate/tree不一致、未実施理由の欠落は`evidence-review`のPASSにしない。QAはfocused/full rerunまたはFAIL候補を提示し、Mainが分類・復帰先を記録する。
- `live-e2e`に必要な実環境または安全なcleanupが用意できない場合は、別modeの結果で代替せず、caseを`live-e2e`のblockedとして残す。環境不備はFAIL分類候補を提示できるが、未確認の受け入れ条件をPASSへ変更しない。
- REVIEW修正が禁止条件へ触れる、または影響caseを限定できない場合は`qa_carry_forward`を作らず、更新candidateに対する必要なrerunを要求する。
- 計画レビューで用語・許可条件・template間に矛盾があれば、安全契約変更を承認せずPLANへ差し戻す。

## 変更予定（厳密な許可範囲）

このTaskで通常変更を許可するのは次の21ファイルだけである。加えて、用語generator出力に差分が出る場合だけ`docs/99-glossary-index.md`を許可する。製品コード、test、runtime/build設定、製品Schema、依存、Lap30 event Schema/JSONL、既存Task証跡（本Taskの運用証跡を除く）は変更しない。`agents/openai.yaml`は既存skill metadataが正確であるため変更しない。

| ファイル | 種別 | 概算変更行数 | 変更内容 |
|---|---|---:|---|
| `AGENTS.md` | safety contract | 35 | product経路の並行candidate評価、risk mode、carry-forward、post-merge確認を規定 |
| `docs/development/README.md` | safety contract | 10 | 正本一覧と新QA経路への案内を整合 |
| `docs/development/development-process.md` | safety contract | 80 | candidate-bound workflow、Main判断、tree比較を正本化 |
| `docs/development/qa.md` | safety contract | 65 | mode定義、証跡監査、risk rerun、FAIL時を正本化 |
| `docs/development/agent-roles.md` | safety contract | 25 | QA/Reviewer/Mainの並行時責務を明確化 |
| `docs/development/code-review.md` | safety contract | 20 | 同一candidateでの独立REVIEWと対象commit/tree記録を整合 |
| `docs/development/task-management.md` | safety contract | 20 | Task証跡の役割をcandidate-bound結果とcarry-forwardへ整合 |
| `docs/development/git-worktree.md` | safety contract | 20 | candidate、merge tree、環境依存のpost-merge確認に合わせて保持/削除条件を整合 |
| `docs/glossary.yml` | mechanically generated terminology source | 30 | 新しい規範用語と抽出inventoryをgeneratorで同期 |
| `templates/task/TASK.md` | template | 15 | 新規Taskの完成定義をrisk-based QAとpost-merge条件へ整合 |
| `templates/task/QA_PLAN.md` | template | 25 | case別mode/rationale/required evidence欄 |
| `templates/task/HANDOVER.md` | template | 30 | DEV candidate-bound execution evidence欄 |
| `templates/task/REVIEW_RESULT.md` | template | 15 | candidate commit/treeと並行評価記録欄 |
| `templates/task/QA_RESULT.md` | template | 35 | mode別結果、candidate/tree、carry-forward/Main判断欄 |
| `.agents/skills/run-efficient-task-delivery/SKILL.md` | skill | 35 | full product pathのrisk QAと並行評価の手順整合 |
| `../agent-harness-work/AGENTS.md` | safety contract | 30 | 運用証跡の所有・書込み・安全経路整合 |
| `../agent-harness-work/.agents/skills/run-lap30/SKILL.md` | skill | 40 | Lap30の候補並行評価とpost-merge確認整合。Schema v1は不変更 |
| `../agent-harness-work/tasks/TASK-0024-risk-based-qa-evidence/TASK.md` | operational evidence | 10 | 安全契約の完成定義と証跡境界を本PLANと整合 |
| `../agent-harness-work/tasks/TASK-0024-risk-based-qa-evidence/QA_PLAN.md` | operational evidence | 80 | TASKだけから作る独立計画。製品QA結果ではない |
| `../agent-harness-work/tasks/TASK-0024-risk-based-qa-evidence/PLAN.md` | operational evidence | 本変更 | 本安全契約計画 |
| `../agent-harness-work/backlog.yaml` | operational index | 5 | Mainがphase/approvalを索引化。Task本文を重複しない |

条件付き生成物: `docs/99-glossary-index.md`は`docs/glossary.yml`からのgenerator出力が変わる場合だけ更新し、直接編集しない。`.agents/skills/run-efficient-task-delivery/agents/openai.yaml`と`../agent-harness-work/.agents/skills/run-lap30/agents/openai.yaml`は表示名・説明・既定promptのmetadataが現行契約と一致するため、許可範囲に含めず変更しない。

見積もり対象は実装コード、Schema、設定ファイルだけであり、上記はすべて規約・template・skill・証跡である。

## 見積もり

```text
planned_implementation_files = 0
planned_implementation_lines = 0
file_score = ceil(0 / 3) = 0
line_score = ceil(0 / 200) = 0
estimate_points = 1
```

## 実装手順

1. Main承認後、許可範囲だけを更新し、用語表とfail-closed条件を各正本・template・skillへ同じ意味で反映する。用語は`docs/glossary.yml`をgeneratorで同期し、差分があるときだけ`docs/99-glossary-index.md`を生成する。
2. QA_PLAN、HANDOVER、REVIEW_RESULT、QA_RESULT templateに、必要最小限の新規欄を後方互換的に追加する。
3. candidate並行評価、Main専有`qa_carry_forward`、tree同一性、環境依存のマージ後確認を両`AGENTS.md`、開発文書、両skillで相互参照可能にする。
4. 独立QA AgentがTASKだけからQA_PLANを作成し、独立計画レビューが許可/禁止条件、用語、template、移行、影響チェックを照合する。安全契約のみのため製品DEV/REVIEW/QA結果を生成しない。

## 検証計画

- `rg`で現行の矛盾候補「`PLAN → DEV → review → QA`の順次」「QA only merged main」「`QA_RESULT` merge-after」「TASK template merge-after QA」「merge-before-QA」と、それらの日本語同義表現を横断検索し、更新後の例外がcandidate並行評価、Main判断、fail-closed条件と矛盾しないことを確認する。mode名称、`qa_carry_forward`、candidate/tree、FAIL分類も検索する。
- 各templateを新規Taskの最小入力として読取り、全mode、candidate binding、DEV証跡、Main判断、未実施理由を表現できることを確認する。既存Taskへの遡及field必須化がないことを確認する。
- 計画レビューで、mode discriminatorを表形式またはcase例で検査する。hermetic/deterministic/bounded fixtureで完全再現できるIPC/Schema parser、race/並行性、lifecycle state machine、persistence/error injection、fail-closed negative pathが`focused-rerun`を選択可能であることを確認する。一方、sudo/PAMを含む実OS権限/auth stack、install/deploy/generated config配置、外部サービス/side effect、実restart/rollback/cleanup、環境固有integrationは`live-e2e`必須であり、環境またはcleanupが不明/unsafeならPASSせずblockedとなることを確認する。
- 計画レビューで、carry-forward禁止条件が必要な再実行へ導くこと、QA/Reviewerが同一candidateで独立に始められること、merge tree不一致/環境依存がpost-merge確認を残すことを確認する。
- `python3 scripts/validate-terminology.py --write`を実行し、続けて同validatorと`npm run test:terminology`で`docs/glossary.yml`、条件付き`docs/99-glossary-index.md`、用語lintを検証する。`git diff --check`、影響する文書/template/skill検査、`make check`、`make work-check`を実行する。Lap30 Schema/JSONLを変更しないため、Schema変更検証や製品テストを製品PASSとして主張しない。

## 未解決事項

- なし。具体的なcase modeの選択は将来の各TASK-first QA_PLANで、そのTaskのリスクと前提から独立に決定する。

## main Agentレビュー

- [x] 受け入れ条件が検証可能である。
- [x] 安全契約変更であり、製品変更・純粋な証跡保守ではない。
- [x] 正確な許可範囲、用語、移行、carry-forwardのfail-closed条件がある。
- [x] TASK-first QA_PLANと独立計画レビューを作成できる。
- [x] 製品DEV、製品REVIEW、製品QAのPASSを作らない。
- [x] 見積もりが規則どおりである。
- [x] 安全契約文書の変更開始を承認した（製品DEVは開始しない）。
