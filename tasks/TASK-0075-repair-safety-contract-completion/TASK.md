---
task_id: "TASK-0075"
title: "安全契約completionの偽PASSと重複digestを廃止する"
status: draft
created_at: "2026-08-02"
---

# TASK-0075 安全契約completionの偽PASSと重複digestを廃止する

## `Planning input packet`（Main Agent所有）

このsectionをPlannerとQAへ渡す唯一の`planning input packet`とし、各内容をPLAN/QA_PLANへ複製しない。

### 目的

安全契約変更を、製品用REVIEW_RESULT/QA_RESULTの偽PASSや自己参照するmerge SHA、candidate/merge tree・digestの重複転記なしで、planning・candidate・no-ff completionの3コミット経路へ統合できるようにする。固定candidate、承認済みQA_PLAN、Mainの契約監査、許可path、既存check、no-ff second parentは維持する。

### 対象と対象外

#### 対象

- `completion-gate`は`change_class: safety_contract`の場合、製品用REVIEW_RESULT/QA_RESULTのPASS、reviewer/qa実行者、review本文を要求しない。承認済みTASK-first QA_PLAN、HANDOVERのcanonical `candidate_commit`、Mainの安全検査は要求する。
- safety contractのdone検査から`merged_commit`の同一merge内記録、`safety_candidate_tree`、`safety_merge_tree`、`safety_check_digest`を削除する。候補とmergeはGitから導出し、HANDOVERへ重複転記しない。
- completion前の`MERGE_HEAD`又はcompletion後のmain履歴から、HANDOVER candidateがexact two-parent no-ff mergeのsecond parentであることを検査する。
- candidateの差分pathをmerge-baseから導出し、v2 `safety_contract_planned_paths/generated_paths`と一致し、main-managed pathや未宣言pathを含まないことを検査する。
- `safety_checks`の既存4項目、`safety_checked_at`、Main承認、分類、QA_PLAN承認、製品成果物除外は維持する。新field、version、receipt、digestを追加しない。
- product completionの独立REVIEW/QA、candidate-bound DEV check、no-ff merge、既存legacy互換を変更しない。
- focused process testsと開発プロセス文書を最小限同期する。

#### 対象外

- Development Agent Harness製品、TASK-0074の設計候補、Kakesu runtime、Schema、依存、生成物、live stateを変更しない。
- 安全契約変更へ新たなReviewer/QA Agent、結果ファイルPASS、tree/digest receipt、追加commitを要求しない。
- planning/candidate/completionの3コミット以外の新しいtransaction又はCLIを追加しない。
- product Taskの品質ゲートを弱めない。
<!-- safety_contractの場合: 製品コード、test、runtime/build設定、Schema、製品依存、生成製品入力/成果物、外部観測可能な挙動を変更しない。 -->

### 受け入れ条件

<!-- AC-IDはTask内で一意かつ安定させ、観測可能な結果をここに一度だけ記載する。 -->

- [ ] AC-1: safety contractのcompletionはREVIEW_RESULT/QA_RESULTがpendingでも、承認済みQA_PLAN、canonical candidate、Main安全検査が揃えば進行し、両結果のPASSを生成又は要求しない。
- [ ] AC-2: safety contract doneはHANDOVERのcandidate_commit一箇所を正本とし、merged_commit、candidate/merge tree、digest転記なしで、Gitからexact no-ff second-parent mergeを検証する。
- [ ] AC-3: v2 planned/generated pathはcandidate diffから検証され、未宣言path、main-managed path、rename/copy、製品除外逸脱を従来どおり拒否する。
- [ ] AC-4: 既存4 safety checks、時刻、Main分類/PLAN/QA_PLAN承認を維持し、新field/version/receipt/追加transactionを導入しない。
- [ ] AC-5: product completionは固定candidateへの独立REVIEW/QA、candidate diff/DEV check監査、no-ff second parentを従来どおり要求し、focused negative testsとroot `make check`がPASSする。

### 安定した参照

| 参照ID | 対象 | 固定改訂/ダイジェスト | 用途 |
|---|---|---|---|
| REF-1 | AGENTS.md 安全契約変更 | main `c8896afca417d2b83e9e3b3d0e09e3246a99cd1d` | 製品実装なしではREVIEW_RESULT/QA_RESULT PASSを作らない正本契約 |
| REF-2 | `checkSafetyContractDone` / `completionGate` | main `c8896afca417d2b83e9e3b3d0e09e3246a99cd1d` | 偽PASS・重複tree/digest・自己参照merged_commitの停止原因 |
| REF-3 | TASK-0074 fixed candidate | `563f52769cfdc1349271a60262c7c340eb5998ae` | 修正後completionで最初に受け入れる実例 |

### 依存状態

| 依存 | 状態 (`ready` / `pending`) | planning参照 | `ready`後に固定する値 |
|---|---|---|---|
| TASK-0074 | `ready` | candidate `563f52769cfdc1349271a60262c7c340eb5998ae` | safety completionの実運用受け入れ |

### 許可パス

- `scripts/task/check-task.mjs`
- `scripts/task/unified-lifecycle.mjs`
- `scripts/task/development-process.test.mjs`
- `scripts/task/unified-lifecycle.test.mjs`
- `docs/development/development-process.md`

### 完了経路preflight

| 確認対象 | 結果 | コマンドまたは根拠 |
|---|---|---|
| 完了checker | `ready` | 現行product completionを使い、同一candidate REVIEW/QAとno-ffを維持する |
| 権限 | `ready` | DEVは許可5パスのみ。Reviewer/QAは同じcandidate、MainだけがGit統合 |
| 依存状態と参照 | `ready` | TASK-0074 fixed candidateで現行停止を再現済み |
| 生成物の有無と更新方法 | `ready` | 生成物なし。script/test/docだけを直接更新する |
| 割当ワークツリー | `ready` | `worktrees/TASK-0075-repair-safety-contract-completion` |
| Lapログの書込・Schema・`repository annotation` | `not-applicable` | 新規log/Schema/annotationなし |

### 未決事項

- なし

### `Dependency-ready reconciliation`

<!-- 依存ready時にMainがready参照、planning参照との差分、AC/設計/スコープ/QAへの影響、再承認結果を追記する。依存なし又は未readyならN/Aとする。 -->

- N/A

## 背景

TASK-0074の安全契約candidate固定後、現行completionは製品用REVIEW/QA PASSを強制するためAGENTSと矛盾した。また、done検査はmerge commit SHAをそのmerge自身のbacklogへ記録する自己参照と、Gitから導出できるcandidate/merge tree・digestの手転記を要求する。偽証跡又はhook迂回で進めず、情報量のない要件を削除して既存3コミット経路を再利用する。

## 検討すべき設計観点

- pre-commitのno-ff merge中は`MERGE_HEAD`、commit後はmain上のmerge parentから同じ事実を検証できること。
- safety契約のscopeはcandidate diff、Main証跡はcompletion evidenceとして分離し、tree equalityを要求しないこと。
- product経路の条件分岐を変えず、safety contractだけを明示分岐すること。
- 古い安全契約証跡は遡及変更せず、新契約では削除したfieldを要求も生成もしないこと。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] 選択した`change_class`の完了経路と`make check`を満たしている。
- [ ] 製品変更の場合: 実装、テスト、文書、同一案の独立REVIEW/QA、完了後の環境依存ケース確認が完了している。
- [ ] 安全契約変更の場合: Mainの意図・スコープ・受け入れ経路確認、契約検査、許可された統制文書差分の確認が完了している。

## 関連コンテキスト

### 意味 Wiki

- 未調査

### 判断

- Gitから導出できるmerge/tree情報はHANDOVERへ転記せず、candidate commitだけを正本にする。
- 安全契約の品質判断はMainの承認済みQA_PLANと契約検査で行い、製品結果PASSを偽装しない。

### 適用しなかった重要な判断

- なし
