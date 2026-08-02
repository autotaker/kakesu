---
task_id: "TASK-0075"
change_class: "product"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "sol-high"
approved_dev_profile_reason: "completion-gate、Git証跡検証、process test、統制文書を横断し、安全契約の偽PASSとsecurity evidence境界を変更するため。"
approved_dev_profile_risk_signals:
  - "cross-cutting-completion-control"
  - "security-evidence-boundary"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-02T09:28:29Z"
classification_approved_by: "main-agent-sol-high"
classification_approved_at: "2026-08-02T09:28:29Z"
classification_approval_reason: "completion checker/lifecycle/process testという外部観測可能な開発workflow toolingを変更するため。"
# safety_contract_version: 2
# safety_contract_planned_paths: []
# safety_contract_generated_paths: []
---

# TASK-0075 PLAN

## AC対応

TASKの条件本文を再掲せず、`planning input packet`のAC-IDに設計を対応させる。

| AC-ID | 設計判断 | 変更パス | 実施順序 | 失敗時の扱い |
|---|---|---|---|---|
| AC-1 | `completionGate`とdone checkerに`change_class: safety_contract`専用分岐を置く。安全契約はapproved QA_PLAN、canonical HANDOVER candidate、Main安全検査だけを要求し、REVIEW/QA resultは読取もPASS生成も要求しない。product分岐は現状の独立identity、candidate/DEV check監査、QA判定を維持する。 | `scripts/task/unified-lifecycle.mjs`, `scripts/task/check-task.mjs`, 両test | 1 | safety分岐がproduct gateへ波及する、又はpending resultをPASS扱いする場合は変更を不成立とする。 |
| AC-2 | candidateの正本をHANDOVERの`candidate_commit`だけに限定する。merge進行中は`MERGE_HEAD`、commit後はmain上のexact two-parent mergeの第2親から同一candidate束縛を導出し、`merged_commit`、candidate/merge tree、digestを安全契約の要求・生成・照合対象から外す。 | `scripts/task/check-task.mjs`, `scripts/task/unified-lifecycle.mjs`, 両test, `docs/development/development-process.md` | 2 | pre-commitとpost-commitで異なるcandidateを許す、又は重複fieldを再導入する実装は拒否する。 |
| AC-3 | v2宣言はcandidateとそのmerge-baseの`--name-status --find-renames --find-copies-harder`差分で検証する。rename/copy、空差分、未宣言・main-managed・許可外path、生成path欠落を拒否し、差分のtree同一性は要求しない。 | `scripts/task/check-task.mjs`, `scripts/task/development-process.test.mjs`, `scripts/task/unified-lifecycle.test.mjs` | 3 | diff導出が候補以外のevidenceを含む、又は既存negative caseが通る場合は停止して差分基準を再検討する。 |
| AC-4 | HANDOVERには既存4 `safety_checks`全件passと`safety_checked_at`だけを残し、Main分類、PLAN/QA_PLAN承認、製品成果物除外、v2 path契約を既存検査で維持する。新field、version、receipt、CLI又はcompletion transactionは追加しない。 | `scripts/task/check-task.mjs`, `scripts/task/development-process.test.mjs`, `docs/development/development-process.md` | 4 | 4項目・時刻・Main承認のいずれかが弱まる、又は予定外の永続証跡が必要になる場合は不成立とする。 |
| AC-5 | focused fixtureを安全契約のpending review/QA、merge進行中とmain merge後、candidate/path spoof、重複field不在に更新し、product fixtureの独立REVIEW/QA、candidate/DEV check、no-ff検出を回帰固定する。root検査と文書は同じ経路を説明する。 | `scripts/task/development-process.test.mjs`, `scripts/task/unified-lifecycle.test.mjs`, `docs/development/development-process.md` | 5 | product gateの欠落を許す、又はfocused negative test/root checkが失敗する場合はcandidateを渡さない。 |

## 根拠・境界

- 要求根拠は`TASK.md`の`planning input packet`、REF-1〜REF-3だけである。TASK-0074 fixed candidateはcompletion受入れの実例であり、本Taskでその証跡を書換えない。
- 本Taskはchecker/lifecycle/test/documentationという外部観測可能な開発workflow toolingを変更するため`product`である。一方、実装する特例は`change_class: safety_contract`の完了経路だけであり、Kakesu runtime、Schema、依存、生成物、live stateは変更しない。
- candidate差分はcandidateのmerge-baseから導出する。main側のHANDOVER、TASK/QA_PLAN、backlog等のcompletion evidenceはcandidate差分に混ぜず、completion scopeの既存制約に委ねる。
- 既存のlegacy安全契約・product証跡は互換入力として維持する。新しい安全契約で削除するfieldを要求、生成、backfillしない。

## 補足設計

### 代替案と不採用理由

- safety契約にも空のREVIEW_RESULT/QA_RESULTを作りPASSにする案は、AGENTSの「製品実装なしではPASSを作らない」契約と矛盾するため不採用とする。
- merge SHA、candidate/merge tree、hash digestをHANDOVER/backlogへ記録する案は、Gitが導出できる値の自己参照・重複であり、pre-commit mergeでは記録不能なため不採用とする。
- safety mergeでcandidate treeとmerge treeの一致を必須にする案は、main-managed completion evidenceを持つ標準no-ff mergeを不当に拒否するため不採用とする。
- productとsafetyの双方でREVIEW/QA要求を緩める共通化、新しいreceipt/transaction/CLI/versionの追加は、ACの境界と最小変更方針に反するため不採用とする。

### 責務・境界・不変条件

- Mainだけがmain側HANDOVER、`--no-ff --no-commit`、completion transaction、main統合を所有する。安全契約のMain検査は、承認済みPLAN/QA_PLAN、分類、candidate、4 safety checks、scopeを確認する。
- pre-commit checkerは存在する`MERGE_HEAD`をcandidateと照合する。post-commit checkerはdefault branch到達済みのexact two-parent mergeを探索し、その第2親をcandidateと照合する。いずれもcandidate以外の親数・ref・SHA転記を信頼しない。
- safety v2 scopeはcandidate diffだけを宣言集合と比較する。main-managed path、rename/copy、未宣言path、生成pathの欠落、製品artifact exclusion逸脱はfail-closedである。
- product completionのREVIEW/QA独立性、review本文のcandidate/DEV check監査、QA判定、candidate一回性、no-ff第2親、legacy互換は不変である。

### 移行・互換性

- v2の新規安全契約はHANDOVERの`candidate_commit`と既存4 safety checks/時刻を使う。`merged_commit`、`safety_candidate_tree`、`safety_merge_tree`、`safety_check_digest`は新規done経路から除去する。
- 既存Task、既存legacy completion binding、Lap30 event schema/JSONLは遡及変更しない。移行用の追加commit、receipt又はbackfillは作らない。

## 変更予定

| パス | 変更内容 |
|---|---|
| `scripts/task/check-task.mjs` | safety done検査をcanonical candidateとGit導出のno-ff束縛・candidate diff scopeへ置換し、重複field/digest要求を除去する。 |
| `scripts/task/unified-lifecycle.mjs` | completion gateに安全契約専用のevidence requirement分岐を追加し、既存のmerge前`MERGE_HEAD`検証とproduct gateを保持する。 |
| `scripts/task/development-process.test.mjs` | safety done fixtureとpositive/negative検査を新証跡モデルへ同期し、productとlegacy互換の回帰を保持する。 |
| `scripts/task/unified-lifecycle.test.mjs` | completion transactionの安全契約分岐、merge中/merge後第2親束縛、product gate不変をfocusedに検証する。 |
| `docs/development/development-process.md` | safety契約の3コミットcompletion、Main検査、Git導出、製品PASS非代用を最小限明文化する。 |

## 実装手順

1. checkerにsafety専用のcandidate/merge解決とmerge-base差分検証を実装し、重複evidence helper・要求を削除する。
2. lifecycleのcompletion requirementをclassificationで分岐し、安全経路はapproved QA_PLANとMain safety evidence、製品経路は既存REVIEW/QA evidenceを使う。
3. process/lifecycle fixtureを新しい安全経路のpositiveとfail-closed mutationに更新し、productの既存要求を明示回帰する。
4. 開発プロセス文書を実際の3コミット経路へ同期し、許可5パス外の変更、生成物、新規永続field/transactionがないことを確認する。

## 検証計画

- `node --test scripts/task/development-process.test.mjs`で、安全契約のpending REVIEW/QA許容、4 safety checks・時刻・承認・scope failure、MERGE_HEAD/main history、第2親、rename/copy/未宣言/生成pathのnegative casesを確認する。
- `node --test scripts/task/unified-lifecycle.test.mjs`で、安全契約completion transactionとproduct completionの独立REVIEW/QA・candidate/DEV check/no-ff回帰を確認する。
- `make check`、`make task-check TASK=TASK-0075`、対象文書lint、`git diff --check`、許可5パスのscope確認を実行する。QA_PLANはhermetic fixtureを`focused-rerun`、candidate-bound証跡を`evidence-review`へ理由付きで割り当てる。
- TASK-0074 fixed candidateは、Mainが本Taskの安全契約completionを実運用で受け入れる際のpost-merge確認対象とする。本PLANではREVIEW_RESULT/QA_RESULT PASSで代替しない。

## 未解決事項

- なし

## main Agentレビュー

- [x] TASKの全AC-IDへ設計判断、パス、順序、失敗時の扱いを対応させ、条件本文を複製していない。
- [x] product分類と安全契約専用分岐の境界、許可5パス、sol-high選定理由を確認した。
- [x] QA_PLANがTASK-firstで独立作成されている。
- [x] `dependency-ready reconciliation`と完了経路preflightが完了している。
- [x] DEV開始を承認した。

MainはPLAN/QA_PLANの意図・スコープ・受け入れ経路を確認し、分類承認をフロントマターへ記録する。分類変更時はTask、PLAN、QA_PLANを再承認し、承認者と時刻を更新する。
