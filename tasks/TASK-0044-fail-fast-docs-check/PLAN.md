---
task_id: "TASK-0044"
change_class: "product"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "luna-xhigh"
approved_dev_profile_reason: "root Makefileの既存check prerequisiteを一行だけ並べ替える局所変更であり、新規rule、script、dependency又は安全境界の変更を含まないため。"
approved_dev_profile_risk_signals: []
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T08:17:42Z"
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

# TASK-0044 PLAN

## 方針と境界

これは root `Makefile` の標準 `check` orchestration を変更する製品変更である。既存の `lint-docs` recipe を複製せず、`check` の最初の prerequisite として一回だけ実行する。後段では既存 `lint` aggregate を `lint-core`、`lint-memory`、`lint-governance` に展開し、`lint` 自体と各公開 subtarget の定義・recipe は変更しない。これにより non-parallel `make check` の command 集合を保ったまま、文書lintが成功してから既存の build、test、残る言語lint、viewer data生成、最終 diff check を実行する。

予定製品差分は root `Makefile` のみであり、追加と削除の合計は10行以下に抑える。新しい rule/script/test/glossary/checklist、依存更新、生成物、CI/hook/Task lifecycle、並列実行の保証は導入しない。TASK-0043 は ready であり、同Taskの反復遅延failureを根拠にするだけで、本計画の要件、scope、検証期待値を変更しない。

## AC対応

TASKの条件本文を再掲せず、`planning input packet`のAC-IDに設計を対応させる。

| AC-ID | 設計判断 | 変更パス | 実施順序 | 失敗時の扱い |
|---|---|---|---|---|
| AC-1 | `check` の prerequisite を `lint-docs`、既存 `build`、既存 `test`、既存3 lint subtarget の順に並べる。既存 recipe の viewer data生成と最終 `git diff --check` は `check` recipe の最後に残す。dry-runでは terminology validator、`pnpm lint:docs`、その直後の文書用 diff check が core/memory/governance build/test/lint command より先であり、viewer data生成と最後の diff check が末尾であることを比較する。 | `Makefile` | 1–3 | command出現順、または末尾recipeの位置が異なればcandidateを作らず、Makefileの prerequisite 順だけを見直す。 |
| AC-2 | `lint-docs` は一回だけ直接参照し、後段は `lint` を変更せず3 subtargetを直接参照する。build/test aggregate、各subtarget、`lint` 公開target、全recipeの command/意味は維持する。 | `Makefile` | 1–2 | dry-runの正規化したcommand集合に増減又は重複があれば、追加target/rule/scriptを作らずcheck prerequisiteだけを修正する。 |
| AC-3 | dependency取得を伴わない既存状態で `UV=false make -o node_modules/.modules.yaml check` を実行し、最初の terminology validator command を即時失敗させる。標準non-parallel `check` の停止codeと、product build/test commandが開始されない出力だけを観測する。 | `Makefile`（検証時の非永続 command overrideのみ） | 4 | `node_modules/.modules.yaml` が存在しない、override前にproduct commandが出る、又は終了がzeroなら、network/依存更新を行わず検証を停止してprecondition又は並びを是正対象として戻す。 |
| AC-4 | candidate差分を root `Makefile` 一ファイル、追加＋削除10行以下に限定する。通常のroot `make check`は最終candidate byteに対して一回実行し、空白は `git diff --check` で確認する。 | `Makefile` | 5 | scope/行数超過、通常check失敗、又はdiff check失敗ならcandidateを固定せず、失敗を実装・環境・検証前提の該当分類でMainへ返す。 |

## 関連Wikiと判断

- Wiki更新なし。既存検査の実行順だけを変更し、新しい再利用可能な意味・用語・ruleを導入しない。
- `lint` aggregateを変更せず、`check`だけで既存3 lint subtargetを後段に列挙する。standalone `make lint` 利用者の順序と挙動を維持する。

## 補足設計

### 代替案と不採用理由

- `lint-docs`を `lint` にも残したまま `check` の先頭へ追加する案は、同一文書lintを二回実行してAC-2に反するため不採用。
- `lint` の内部順序を変えて `check: build test lint` を維持する案は、build/testより先に完了させるAC-1を満たさないため不採用。
- 新しいfail-fast wrapper、順序assertion test、または専用scriptを追加する案は、単純なMakefile orchestrationの範囲とAC-2の対象外に反するため不採用。
- `make -j` の順序を制御するdependency edgeを増やす案は、parallel実行を保証しない対象外を広げるため不採用。

### 責務・境界・不変条件

- `lint-docs` は既存の `node-deps` prerequisite と三つの既存commandを所有したままにする。`check` はそれを一回だけ先に選ぶorchestratorであり、recipe内容を移動・変更しない。
- `build`、`test`、`lint-core`、`lint-memory`、`lint-governance` は後段で各一回だけ実行する。`lint` aggregate はpublic targetとして定義も意味も維持するが、`check` のdependencyには残さない。
- viewer data生成と最後の `git diff --check` は `check` recipeのままとし、文書lint内の既存 `git diff --check` と区別して両方を一回ずつ維持する。
- fault injectionはmake変数と `-o` による一回限りの実行だけで、repository fileを編集せず、`node_modules/.modules.yaml` の再生成、network、dependency更新、実product commandを起こさない。

### 移行・互換性

- 入力・生成物・依存・versionの移行はない。`make lint` と個別targetの呼出しは従来どおりで、保証対象は標準non-parallel `make check` の順序だけである。

## 変更予定

| パス | 変更内容 |
|---|---|
| `Makefile` | `check` の prerequisite を `lint-docs`、`build`、`test`、`lint-core`、`lint-memory`、`lint-governance` の順へ最小変更する。recipe、`lint` aggregate、他targetは変更しない。 |

上記以外のパスは変更しない。

## 実装手順

1. 現行 `make -n check` をbaselineとして記録し、各既存commandを一回ずつ含むこと、文書lintが `lint` の末尾にあることを確認する。
2. root `Makefile` の `check` prerequisiteだけを最小差分で並べ替える。`lint-docs`を先頭にし、後段には `build`、`test`、既存3 lint subtargetを置く。`check` recipe、`lint` aggregate、全subtarget recipeには触れない。
3. candidateの `make -n check` をbaselineと比較する。全commandの集合と回数が同一で、文書lint三commandの位置、後段command、viewer data生成、二つのdiff checkの相対順がAC-1/AC-2どおりであることを確認する。
4. `node_modules/.modules.yaml` が既に存在することをread-onlyで確認後、`UV=false make -o node_modules/.modules.yaml check` を一回実行する。nonzero終了、terminology validatorでの停止、build/test未到達を出力から確認する。ファイル改変、dependency install、networkアクセスが必要になれば実行せず停止する。
5. 最終candidate byteで通常のroot `make check` を一回実行し、`git diff --check`、変更path、追加＋削除行数を確認する。Mainは同一candidateの独立REVIEW/QAおよびmerge後 `make task-check TASK=TASK-0044` を続行する。

## 検証計画

- dry-run比較: baseとcandidateの `make -n check` 出力を比較し、terminology validator、`pnpm lint:docs`、文書用 `git diff --check` が最初のcore/memory/governance build/test/lint commandより先、viewer data生成と最終 `git diff --check` が最後であることを確認する。各既存commandの出現回数は一回で不変とする。
- fault injection: `node_modules/.modules.yaml` がある依存取得不要な状態だけで `UV=false make -o node_modules/.modules.yaml check` を実行する。terminology validatorでnonzero終了し、core/memory/governance build/testおよび後段lint、viewer data生成が出力にないことを確認する。これは永続testやscriptにしない。
- candidate validation: 最終candidate byteへのroot `make check` を一回、`git diff --check` を実行する。差分がroot `Makefile`だけであり、追加＋削除10行以下であることを監査する。
- Mainの独立QAはQA_PLANのケース単位モードに従いcandidate-bound dry-run/fault/通常check証跡を監査し、環境依存確認を別モードPASSで代替しない。本Taskにはlive-e2eを要求する環境境界はない。

## 未解決事項

- なし

## main Agentレビュー

- [x] TASKの全AC-IDへ設計判断、パス、順序、失敗時の扱いを対応させ、条件本文を複製していない。
- [x] 設計観点と代替案を検討している。
- [x] QA_PLANがTASK-firstで独立作成されている。
- [x] `dependency-ready reconciliation`と完了経路preflightが完了している。
- [x] DEV開始を承認した。

安全契約変更でv2契約を選ぶ場合は、コメントを外して`safety_contract_version: 2`と予定パス・生成パスの配列を記録し、DEV前に`make task-preflight TASK=TASK-0044`を実行する。変更しない種別は空配列とし、通常の予定パスと生成パスを重複させない。独立計画レビューのPASSとMainの分類承認をフロントマターへ記録する。分類変更時はTask、PLAN、QA_PLANを再承認し、承認者と時刻を更新する。
