---
task_id: "TASK-0054"
change_class: "product"
status: approved
qa_agent: "qa-agent-terra-medium"
qa_role: "independent-qa"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T15:29:18Z"
revision: 1
implementation_reviewed_at: ""
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0054 QA PLAN

## QA scope

期待値の正本は TASK.md の `Planning input packet` だけとする。PLAN、実装案、DEV の自己申告から期待値を導かない。candidate の変更は `scripts/task/unified-lifecycle.mjs` と `scripts/task/unified-lifecycle.test.mjs` の二 path に限定し、追加＋削除は 300 行以下であることを監査する。

QA は planning Git 変更より前に既存 `validateDevSelection` が一回だけ呼ばれることを確認する。profile の種類、risk signal、promotion 規則、role/model/effort 契約、schema/frontmatter、新 field/gate/check/manual step、candidate/completion/Wiki/backlog の意味を追加・変更しない。

## Cases

| Case ID | 対象AC | 確認内容と failure detection | Mode | Evidence |
|---|---|---|---|---|
| QA-001 | AC-1 | candidate source/diff により、`validatePlanningState` が既存 `validateDevSelection` を一回だけ呼び、profile、reason、risk signals、promotion 整合の不正を planning commit 前に completion と同じ error code で拒否することを監査する。validator logic の複製、completion validation の削除・意味変更がないことを確認する。 | evidence-review | candidate diff/source/test、HANDOVER、DEV command/result |
| QA-002 | AC-2 | TASK-0053 と同じ frontmatter 三項目をまとめて欠落させる一件の planning integration fixture が、Git 書込み前に失敗を検出することを確認する。失敗後に main HEAD、Task branch HEAD、Task worktree HEAD、index が不変で、未commit の planning 入力（dirty diff）が保持され、commit/push/worktree fast-forward が起きないことを failure-detecting assertion で確認する。 | focused-rerun | `scripts/task/unified-lifecycle.test.mjs` の candidate 実行、candidate source/test |
| QA-003 | AC-3 | valid 標準 fixture の planning が新 field/gate/manual step なしで従来どおり一つの planning commit を作り、Task branch を同一 commit へ fast-forward できることを lifecycle assertions で確認する。既存 `validateDevSelection` unit matrix により `luna-xhigh` と `sol-high` の両 profile 契約が保たれ、valid planning から candidate、completion までの既存 three-commit/no-ff 回帰を弱体化・削除していないことを監査する。 | focused-rerun | `scripts/task/unified-lifecycle.test.mjs` の candidate 実行、candidate source/test |
| QA-004 | AC-4 | `validateDevSelection` の既存 unit matrix と three-commit lifecycle test が残り、不正 profile/reason/risk signals/promotion と三項目欠落の負ケースを実際に failure として観測できることを監査する。pass-only fixture、期待 error の緩和、assertion 削除で回帰検出能力を失わせていないことを確認する。 | evidence-review | candidate diff/source/test、HANDOVER、DEV command/result |
| QA-005 | AC-5 | candidate diff が許可二 path 内、追加＋削除 300 行以下であり、`git diff --check`、root `make check`、agent-routing test は candidate-bound DEV 証跡の command/result としてのみ監査する。QA はこれらを再実行しない。focused rerun が失敗・非決定的・非 hermetic・非上限付き、又は証跡と candidate が不一致なら該当 case は FAIL/blocked とし、evidence-review PASS で置換しない。 | evidence-review | candidate diff/stat、HANDOVER、DEV command/result |

## Execution rule

QA-002 と QA-003 は同じ一回の focused-rerun に束ねる。QA は repository root を cwd とし、candidate に対して次だけを一回実行する。

```sh
node --test scripts/task/unified-lifecycle.test.mjs
```

QA は root `make check`、agent-routing test、`git diff --check`、追加 test、追加 rerun を実行しない。これらは DEV 証跡を独立監査するだけとする。live-e2e は不要であり、実施ケースを置かない。

## Result criteria

各 case は candidate-bound evidence と planning input packet に照らして記録する。focused-rerun には command、repository-root cwd、exit status、実行 test と、三項目欠落時の Git/dirty-input 原子性および valid planning/three-commit 回帰を残す。

失敗を実装不具合と決めつけず、`implementation_defect`、`qa_plan_defect`、`requirement_gap`、`environment_issue`、`regression`、又は evidence 不足として根拠付きで分類する。

## 実装後の再確認

- [ ] candidate source/test、HANDOVER、DEV check evidence を独立確認した。
- [ ] 指定 focused-rerun を repository root で一回だけ実行した。
- [ ] planning 前の既存 validator 一回呼出し、三項目欠落時の Git/dirty-input 原子性、valid planning と three-commit 回帰の failure detection を確認した。
- [ ] 変更が許可二 path と追加＋削除 300 行以内に収まり、新 field/gate/check/manual step がないことを確認した。
- [ ] root `make check`、agent-routing test、`git diff --check` を再実行せず DEV 証跡だけを監査し、期待値または scope を変更していないことを確認した。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | 2026-08-02 | QA (Terra/medium) | Planning input packet に基づく独立 QA 計画 | `approved` |
