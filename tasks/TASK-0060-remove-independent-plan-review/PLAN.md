---
task_id: "TASK-0060"
change_class: "product"
status: "approved"
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "luna-xhigh"
approved_dev_profile_reason: "template、checker、process test、統制文書を限定して同期し、Kakesu product runtime、tools/dev-agent-harness runtime、Schema、依存、生成物を変更しないため。"
approved_dev_profile_risk_signals: []
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T23:26:36Z"
classification_approved_by: "main-agent-sol-high"
classification_approved_at: "2026-08-01T23:26:36Z"
classification_approval_reason: "template、checker、process test、統制文書という外部観測可能な開発workflow toolingを変更するため。"
---

# PLAN: 独立PLANレビューを廃止してMain軽量確認へ統合する

## 根拠と分類

本計画の唯一の要求根拠は`TASK.md`の`Planning input packet`である。template、workflow checker、process testおよび統制文書の外部観測可能な開発workflowを変更するため、`change_class`は`product`とする。Kakesu product runtime、`tools/dev-agent-harness` runtime、Schema、製品依存、生成製品入力・成果物は変更しない。

本PLANの独立レビューは実施しない。MainはDEV開始前にPLANと独立してTASK-firstで作成されたQA_PLANについて、ユーザ意図、変更範囲、受け入れ経路の一致だけを軽量確認し、既存の`approved_by`と`approved_at`で承認を記録する。技術的な独立品質評価は、DEV後の固定candidateに対するREVIEW/QAへ維持する。

## 変更境界

変更候補は次の8パスに限定する。

- `AGENTS.md`
- `docs/development/agent-roles.md`
- `docs/development/development-process.md`
- `docs/development/task-management.md`
- `templates/task/PLAN.md`
- `templates/task/TASK.md`
- `scripts/task/check-task.mjs`
- `scripts/task/development-process.test.mjs`

既存Taskの履歴証跡は変更しない。古い`planning_review_*` fieldは互換入力として受理し続けるが、新Taskのtemplate、checkerの必須条件、専用FAIL fixtureからは除去する。新field、check、version、receipt、または独立PLANレビュープロセスを追加しない。

## 実施設計

1. 新Task用templateから独立PLANレビューを除去する。
   - `templates/task/PLAN.md`から`planning_reviewed_by`、`planning_review_decision`、`planning_reviewed_at`を削除する。
   - `templates/task/TASK.md`の新Task向け案内から、独立ReviewerのPLAN PASS・時刻を必須工程とする表現を除去する。
   - Mainが既存の`approved_by`/`approved_at`でPLANとQA_PLANの軽量確認を記録する経路を表現し、新しい証跡fieldは増やさない。

2. safety contract checkerの必須条件を最小に更新する。
   - `scripts/task/check-task.mjs`からReviewerによる計画PASSおよびその時刻順序を必須とする検査を削除する。
   - MainによるPLAN/QA_PLAN承認、classification approval、TASK-first QA_PLAN、および既存の契約固有scope・安全検査は維持する。
   - 古いTaskに残る`planning_review_*` fieldは、存在しても失敗にせず互換入力として許容する。新Taskにそのfieldがないことは有効な状態とする。

3. 統制文書とAgent責務を同じ工程へ同期する。
   - `AGENTS.md`、`docs/development/agent-roles.md`、`docs/development/development-process.md`、`docs/development/task-management.md`で、Mainの確認を意図・scope・受け入れ経路に限定する。
   - PLANの技術的な独立レビュー、Reviewer起動、機械的PASSを要求する記述を除去する。
   - PlannerのPLAN作成、QA担当によるTASK-first QA_PLAN作成、Main承認、DEV profile選択、classification approval、許可path・秘密境界、no-ff completionは変更しない。

4. focused process testsを新契約へ更新する。
   - `development-process.test.mjs`から、plan reviewer fieldの欠落だけを理由にFAILするfixture・assertionを削除する。
   - reviewer fieldがない新TaskがMain承認で進行できること、古いreviewer fieldを持つ既存Taskが互換入力として受理されることを検出する。
   - Main承認欠落、classification承認欠落、TASK-first QA_PLAN違反、scope/安全検査の不整合が引き続きFAILすることを検出する。
   - DEV後の同一candidateに対する独立REVIEW/QA、役割分離、candidate-bound check、no-ff completionを弱体化するfixture・assertionは変更しない。

## 受け入れ条件への対応

| AC | 実施・検証の対応 |
|---|---|
| AC-1 | PLAN/TASK templateから`planning_review_*`と独立レビュー指示を除去し、既存Main承認fieldだけで開始経路を表現する。 |
| AC-2 | safety checkerからReviewer PASSと時刻順序の要求を除去し、Main承認、classification、TASK-first QA_PLAN、scope/安全検査のFAIL検出を維持する。 |
| AC-3 | 4つの統制文書を、Main軽量確認と独立PLANレビュー非要求の同一契約へ同期する。 |
| AC-4 | DEV後の同一candidate独立REVIEW/QA、役割分離、candidate-bound check、no-ff completionをfocused process testで維持確認する。 |
| AC-5 | 新fieldを生成・要求せず、既存Taskの`planning_review_*`を互換入力として受理するaccepted caseを追加する。 |

## 実施・完了経路

DEV開始前に、Mainが本PLANと独立QA_PLANを軽量確認・承認する。DEV AgentはTaskブランチで製品差分だけのcandidateを一度作成する。candidate固定後、Reviewer AgentとQA Agentが互いのPASSを開始条件とせず、同じcandidateから独立にREVIEWとQAを実施する。Mainだけがcandidate固定、`--no-ff --no-commit`検査、completion transaction、main統合を所有する。

この変更はPLANの独立レビューだけを廃止する。実装後REVIEW/QA、DEV/Reviewer/QAの分離、candidate-bound証跡、必要なマージ後環境依存確認は維持する。

## 検証計画

- `scripts/task/development-process.test.mjs` のfocused process tests
- safety contract checkerに関する対象ケース
- docs lint
- `make check`
- `make task-check TASK=TASK-0060`
- `git diff --check`

QA_PLANは各ケースを`evidence-review`、`focused-rerun`、`live-e2e`のいずれかに理由付きで割り当てる。template/checker/process-testのhermeticな受け入れ真実は`focused-rerun`候補とし、candidate-bound証跡およびDEV `make check`の監査は`evidence-review`で扱う。実OS権限、実配置、外部作用に依存するケースがなければ`live-e2e`はnot-applicableとし、その理由を記録する。

## リスクと復旧

- 独立PLANレビューだけでなくDEV後REVIEW/QAまで弱体化するリスクは、後者の契約とfocused assertionを明示的に維持して抑える。
- 古いTaskをreader側で拒否するリスクは、legacy `planning_review_*`を任意の互換入力としてテストすることで抑える。
- Main確認が技術的な独立レビューへ戻るリスクは、文書を意図・scope・受け入れ経路に限定し、新field/checklistを追加しないことで抑える。
- Kakesu product runtime、tools/dev-agent-harness runtime、Schema、依存、生成物の差分が見つかった場合は実装を停止してMainが再評価する。

復旧時は、template、checker、文書、process testを許可path単位で従前の契約へ戻し、focused tests、`make check`、`make task-check TASK=TASK-0060`を再実行する。既存Taskの履歴fieldは書換えない。

## 引き継ぎ条件

DEVは承認済みPLANとQA_PLANの後に、許可8パスだけで実施する。新規独立PLANレビューを開始しない。DEV後は同一candidateの独立REVIEW/QAを必須とし、Mainだけが完了ゲートとmain統合を所有する。
