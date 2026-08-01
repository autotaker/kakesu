---
task_id: "TASK-0054"
title: "DEV profile validationをplanning gateへ前倒しする"
status: plan
created_at: "2026-08-02"
---

# TASK-0054 DEV profile validationをplanning gateへ前倒しする

## `Planning input packet`（Main Agent所有）

このsectionをPlannerとQAへ渡す唯一の`planning input packet`とし、各内容をPLAN/QA_PLANへ複製しない。

### 目的

planning gateが承認済みPLANをcommit/pushする前に、completionで既に必須の`approved_dev_profile`、理由、risk signals、promotion整合を同じ正本関数で検証する。TASK-0053でcompletionまで遅延したprofile欠落をplanning時点で原子的に拒否し、後続candidate rebase・再REVIEW/QA・Wiki再同化を発生させない。

### 対象と対象外

#### 対象

- `validatePlanningState`から既存`validateDevSelection`を一回呼び、profile契約不正をstage/commit/push/Task branch fast-forward前に拒否する。
- planning lifecycle fixtureへ正しいDEV profileを設定し、profile欠落時にmain/Task branch/worktreeとplanning差分が不変であるintegration testを追加する。
- 有効なplanning/candidate/completionの既存one-commit/no-ff経路が変わらないことを既存testで確認する。

#### 対象外

- DEV profileの種類、risk signal、promotion規則、role/model/effort契約、PLAN schema又はfrontmatter fieldを追加・変更しない。
- 新しいgate、check command、証跡field、commit、Wiki必須条件、review/QA手順を追加しない。
- candidate/completion、Wiki launcher、backlog状態遷移、製品harness実装を変更しない。

### 受け入れ条件

<!-- AC-IDはTask内で一意かつ安定させ、観測可能な結果をここに一度だけ記載する。 -->

- [x] AC-1: planning gateは既存`validateDevSelection`を正本として、DEV profile/reason/risk signals/promotion不正をplanning commitより前に同じerror codeで拒否する。検証logicを複製せず、completionの意味を変更しない。
- [x] AC-2: profile不正時はmain HEAD、Task branch HEAD、Task worktree HEAD、indexを変更せず、未commitのplanning入力を保持し、commit/push又はworktree fast-forwardを行わない。
- [x] AC-3: 有効な`luna-xhigh`又は`sol-high` planningは従来どおり一つのplanning commitだけを作り、Task branchを同commitへfast-forwardできる。新しいfield、gate、手動stepを要求しない。
- [x] AC-4: integration testは少なくともTASK-0053と同じfrontmatter三項目欠落をplanning gateで失敗検出し、既存`validateDevSelection` unit matrixとthree-commit lifecycle testを弱体化しない。
- [x] AC-5: root `make check`、focused process test、`git diff --check`がPASSし、許可path内で追加＋削除300行以下とする。

### 安定した参照

| 参照ID | 対象 | 固定改訂/ダイジェスト | 用途 |
|---|---|---|---|
| REF-1 | TASK-0053 completion failure | completion `9966f53` / PLAN correction `a1d73f8` | 遅延検出が引き起こしたrebase・再評価の実例 |
| REF-2 | DEV selection validator / planning lifecycle | main `9966f53`時点 | 既存正本関数とplanning transaction境界 |

### 依存状態

| 依存 | 状態 (`ready` / `pending`) | planning参照 | `ready`後に固定する値 |
|---|---|---|---|
| TASK-0053 | `ready` | REF-1 | 欠落field、failure phase、手戻り連鎖を固定する |

### 許可パス

- `scripts/task/unified-lifecycle.mjs`
- `scripts/task/unified-lifecycle.test.mjs`

### 完了経路preflight

| 確認対象 | 結果 | コマンドまたは根拠 |
|---|---|---|
| 完了checker | `ready` | 標準planning/candidate/completionとpost-merge task-checkを使用する |
| 権限 | `ready` | temporary Git fixturesとlocal process testsだけ。network、実secret、外部作用なし |
| 依存状態と参照 | `ready` | TASK-0053完了。既存validatorを移動せず呼出しphaseだけを前倒しする |
| 生成物の有無と更新方法 | `ready` | JavaScript source/testだけ。生成物・依存・Schemaなし |
| 割当ワークツリー | `ready` | `worktrees/TASK-0054-frontload-dev-profile-validation` |
| Lapログの書込・Schema・`repository annotation` | `ready` | 新規log/field/checkなし、標準3 commits |

### 未決事項

- なし

### `Dependency-ready reconciliation`

<!-- 依存ready時にMainがready参照、planning参照との差分、AC/設計/scope/QAへの影響、再承認結果を追記する。依存なし又は未readyならN/Aとする。 -->

- TASK-0053はready。completionで`DEV_PROFILE_UNKNOWN`が初検出され、PLAN訂正後にcandidate直子条件でrebaseが必要となった事実をREF-1へ固定した。期待する製品挙動は変えず、同じvalidationをplanning commit前へ移す。

## 背景

TASK-0053ではPLAN本文に`luna-xhigh`があったがfrontmatter三項目が欠け、planning gateとcandidate launcherを通過した。completion validationで初めて失敗したため、PLAN訂正commit、candidate rebase、exact check、独立REVIEW/QA、Wiki receiptをやり直した。これは品質を上げない遅延であり、既存条件の検出phase不一致が原因である。

## 検討すべき設計観点

- `validateDevSelection`をimportしてplanning state validationから呼ぶだけとし、profile集合やrisk logicを再実装しない。
- 検証はplanning transactionがGitを変更する前に行い、失敗rollback処理を増やさない。
- integration fixtureの既定PLANを有効化し、欠落専用testはHEAD/index/dirty inputs保持を観測する。
- 一回限りのTASK-0053事情を新しい恒久fieldやcheckへ変換しない。

## 完成の定義

- [x] 受け入れ条件を満たしている。
- [x] planning/candidate/completionの3 commitsとcandidate一回のroot `make check`を満たしている。
- [x] 同一candidateの独立REVIEW/QAとpost-merge task-checkを完了している。

## 関連コンテキスト

### 意味 Wiki

- 該当する既存Semanticページなし。Task完了時は再利用可能な知識がなければreceiptだけを作る。

### 判断

- 新規validationを増やさず、既存DEV selection validationの実行phaseだけをplanning commit前へ移す。

### 適用しなかった重要な判断

- completion側の検証を削る案は、既存main履歴や外部commit経路へのdefense-in-depthを失うため採用しない。
- profile fieldをtemplateへ固定値で追加するだけの案は、不正値・promotion整合を早期検出できないため採用しない。
