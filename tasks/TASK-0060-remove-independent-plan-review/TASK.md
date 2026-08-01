---
task_id: "TASK-0060"
title: "独立PLANレビューを廃止してMain軽量確認へ統合する"
status: draft
created_at: "2026-08-02"
---

# TASK-0060 独立PLANレビューを廃止してMain軽量確認へ統合する

## `Planning input packet`（Main Agent所有）

このsectionをPlannerとQAへ渡す唯一の`planning input packet`とし、各内容をPLAN/QA_PLANへ複製しない。

### 目的

PLANのための独立Reviewer起動、専用証跡field、機械的PASS要求を廃止する。MainがPLANとQA_PLANについてユーザ意図、変更範囲、受け入れ経路の一致だけを軽く確認してDEV開始を承認し、品質評価は固定candidateに対する実装REVIEW/QAへ集中させる。

### 対象と対象外

#### 対象

- 製品変更と安全契約変更のいずれでも、独立した計画レビューを必須工程から外す。
- MainはPLAN/QA_PLANの意図、scope、受け入れ経路だけを確認し、既存の`approved_by`/`approved_at`で承認を記録する。新しいversion、receipt、checklist fieldは追加しない。
- 新TaskのPLAN templateから`planning_reviewed_by`、`planning_review_decision`、`planning_reviewed_at`を削除する。
- safety contract checkerからReviewer計画PASSとその時刻順序チェックを削除し、MainのPLAN/QA_PLAN承認、分類承認、Task-first QA_PLAN、契約固有検査は維持する。
- 統制文書、Agent責務、process testsを同じ契約へ同期する。

#### 対象外

- DEV後の固定candidateに対する独立実装REVIEWとQA、DEV/Reviewer/QAの担当分離は変更しない。
- QA_PLANをTASK-firstでQA担当が作ること、Mainが承認することは変更しない。
- PLAN作成をPlannerが担当すること、DEV profile選択、classification approval、許可path、秘密境界、no-ff completion、`make check`は変更しない。
- 既存Taskの履歴証跡を遡及削除・書換えしない。古い`planning_review_*` fieldが残っていても互換性のため受理するが、新Taskでは生成・要求しない。
- Kakesu product runtime、`tools/dev-agent-harness` runtime、Schema、製品依存、生成製品入力/成果物を変更しない。開発workflow checker/template/testは変更対象とする。

### 受け入れ条件

<!-- AC-IDはTask内で一意かつ安定させ、観測可能な結果をここに一度だけ記載する。 -->

- [ ] AC-1: 新TaskのPLAN templateに`planning_review_*` fieldや独立計画レビュー指示がなく、Mainの既存PLAN/QA_PLAN承認だけでDEV開始経路を表現できる。
- [ ] AC-2: safety contractの検査は独立Reviewer計画PASSを要求せず、MainによるPLAN/QA_PLAN・分類承認と契約固有のscope/安全検査は従来どおりFAILを検出する。
- [ ] AC-3: AGENTSと開発文書はMainの軽量確認を「意図・範囲・受け入れ経路」に限定し、PLANの技術的な独立レビューを工程として要求しない。
- [ ] AC-4: 固定candidateに対する独立実装REVIEW/QA、担当分離、candidate-bound check、no-ff completionは変更されず、focused process testsとroot `make check`がPASSする。
- [ ] AC-5: 既存Taskの`planning_review_*`履歴fieldは書換えず互換入力として許容し、新規checker/templateから情報量のない必須fieldと専用FAIL fixtureだけを削除する。

### 安定した参照

| 参照ID | 対象 | 固定改訂/ダイジェスト | 用途 |
|---|---|---|---|
| REF-1 | ユーザ方針 | 2026-08-02本Task要求 | PLAN/QA_PLANはMainが意図一致だけを軽く確認する |
| REF-2 | 現行plan review checker/template | main `8cffcd2` | 削除対象のReviewer PASS fieldと機械検査 |
| REF-3 | 実装後REVIEW/QA契約 | main `8cffcd2` | 維持する独立品質評価 |

### 依存状態

| 依存 | 状態 (`ready` / `pending`) | planning参照 | `ready`後に固定する値 |
|---|---|---|---|
| なし | `ready` | N/A | N/A |

### 許可パス

- `AGENTS.md`
- `docs/development/agent-roles.md`
- `docs/development/development-process.md`
- `docs/development/task-management.md`
- `templates/task/PLAN.md`
- `templates/task/TASK.md`
- `scripts/task/check-task.mjs`
- `scripts/task/development-process.test.mjs`

### 完了経路preflight

| 確認対象 | 結果 | コマンドまたは根拠 |
|---|---|---|
| 完了checker | `ready` | 製品変更の通常3コミット経路、同一candidateの独立REVIEW/QA、no-ff completionを使用する |
| 権限 | `ready` | Planner/QAは計画を編集し、Mainだけが承認・Git統合する。DEV後のReviewer/QA分離は維持する |
| 依存状態と参照 | `ready` | 依存なし。現行main `8cffcd2`を基準にする |
| 生成物の有無と更新方法 | `ready` | generated artifactなし。template/checker/docs/testだけを直接更新する |
| 割当ワークツリー | `ready` | `worktrees/TASK-0060-remove-independent-plan-review` |
| Lapログの書込・Schema・`repository annotation` | `not-applicable` | 新規log/Schema/annotationなし |

### 未決事項

- なし

### `Dependency-ready reconciliation`

<!-- 依存ready時にMainがready参照、planning参照との差分、AC/設計/scope/QAへの影響、再承認結果を追記する。依存なし又は未readyならN/Aとする。 -->

- N/A

## 背景

独立PLANレビューは、実装前の推測を別Agentが形式確認する往復を増やし、実装candidateの品質を直接上げていない。TASK-0059ではMainがPLAN/QA_PLANの意図・範囲・受け入れ経路だけを確認して十分に進行できた。品質評価は実物があるDEV後の独立REVIEW/QAへ集中させる。

## 検討すべき設計観点

- 既存のMain承認fieldを再利用し、新しい確認証跡を増やさない。
- 独立QA_PLANは実装前のfailure detectionを設計する役割として残すが、そのPLAN自体へのReviewer PASSは作らない。
- 古いTaskの履歴互換性は保ちつつ、新しいTaskから不要fieldを生成しない。
- process testは「plan reviewer欠落でFAIL」を削除し、Main承認欠落や分類/scope不整合でFAILする有効な検査を維持する。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] 選択した`change_class`の完了経路と`make check`を満たしている。
- [ ] 製品変更の場合: 実装、テスト、文書、同一案の独立REVIEW/QA、完了後の環境依存ケース確認が完了している。
- [ ] 安全契約変更の場合: Mainの軽量確認、契約検査、許可された統制文書差分の確認が完了している。

## 関連コンテキスト

### 意味 Wiki

- `docs/development/development-process.md`、`docs/development/task-management.md`

### 判断

- 計画品質はMainの意図・scope・受け入れ経路確認で担保し、独立評価は実装candidateへ集中する。

### 適用しなかった重要な判断

- なし
