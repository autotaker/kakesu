---
task_id: "TASK-0037"
title: "Taskゲートの重複transactionと検査を削減する"
status: done
created_at: "2026-08-01"
---

# TASK-0037 Taskゲートの重複transactionと検査を削減する

## `Planning input packet`（Main Agent所有）

このsectionをPlannerとQAへ渡す唯一の`planning input packet`とし、各内容をPLAN/QA_PLANへ複製しない。

### 目的

Task品質ゲートの独立性とfail-closed性を維持したまま、同じフェーズ内の証跡を1ファイルずつcommit・全体validationする重複と、同じcandidateに同じ目的で`make check`を繰り返す規則を削除する。direct-main運用でもmerge scopeを正しく検証できる入口を修復し、20分Taskへ近づける。

### 対象と対象外

#### 対象

- `planning-gate`: 承認済みPLAN、QA_PLAN、TASK/backlogの最終状態を検査し、実際に変更された許可内pathだけを1 transactionでcommitするaction。
- `completion-gate`: 固定candidateの製品差分とHANDOVER、post-implementation QA_PLAN、REVIEW_RESULT、QA_RESULT、TASK/backlogを1回のno-ff merge commitへまとめるaction。再利用可能な知識がある場合だけWiki差分も同じtransactionへ含める。
- Task開始時はscaffold、branch、worktreeだけを準備してmain commitを作らず、最初の公開を`planning-gate`にする。
- 両actionの原子性、許可外path拒否、validation一回を固定するprocess tests。
- parentが直前に検証した完全な変更集合のdigestをpre-commit hookが照合し、一致時だけhook側の同一`validate-work`再実行を省略する。digest不一致・通常commitは従来どおりhookが検証する。
- `make task-scope-check ... ALLOW_MERGE=1`が`--allow-merge true`をCLIへ渡し、no-ff candidate mergeをdirect pushと誤判定しない修正。
- candidate commitはHANDOVERだけを正本とし、treeと差分digestはGitから導出する。REVIEW/QAへの同値転記を削除する。
- Reviewerはcandidate-bound check証跡と差分を独立監査し、DEVと同じ`make check`を目的なく再実行しない。QAの独立focused rerunは維持する。
- 推測値の算術一致、全Task Wiki receipt、CF-1〜CF-7、observed model/effort差による停止、全ケース一律の詳細証跡を必須ゲートから削除する。
- TASK-0036の実測と削除した工程を記録し、手戻りなしの標準経路をplanning、candidate、completion mergeの3 commitsにする。
- 開発プロセスは最小の既定から始め、実際に反復した不具合・指摘、または一度でも重大な安全/回復不能事故を防ぐ場合だけルールや機械gateを追加する。単発の軽微なミスへ恒久ルールを足さない。
- retrospective内で既存ルールを定期的に棚卸しし、意味のある不具合を検出していない、手動転記しか検査しない、または回避コストが検出価値を上回る項目は削除・警告化する。棚卸し専用の証跡やTaskは要求しない。

#### 対象外

- PLAN、QA_PLAN、REVIEW、QAそのものの省略、ロール統合、candidate commitの固定の緩和。
- GitHub PR/CI workflow、Wiki Agent自体、製品testの削除。
- `dev-agent-harness`製品コード、Kakesu runtime、Schema、Credential、network、OS境界。
- subagentへの指示文調整を改善成果として数えること。

### 受け入れ条件

<!-- AC-IDはTask内で一意かつ安定させ、観測可能な結果をここに一度だけ記載する。 -->

- [ ] AC-1: `planning-gate`はPLAN、QA_PLAN、TASK、backlogのうち実際に変更された1件以上だけをcommitし、新規Taskの初回だけtaskStartが生成した未追跡の空REVIEW/QA/HANDOVER scaffoldも同じplanning commitへ含められる。最終状態の承認/role/status整合と許可外path不在を検査し、既存追跡済みファイルを形式上dirtyにすることは要求しない。既存個別actionは復旧互換だけに残す。
- [ ] AC-2: `completion-gate`はmain側で未commitの品質証跡として用意したHANDOVERからcandidate commitを読み、そのcommitを第2親とするno-ff mergeを作る。candidate branchはplanning後にmainへfast-forwardした基点から製品差分だけを載せた単一commitでなければ拒否する。製品差分、post-implementation QA_PLAN、REVIEW_RESULT、QA_RESULT、TASK/backlogを1 completion commitで完了させ、REVIEW/QA非PASS、candidate branch不一致、許可外path、未検証内容差し替えを拒否する。失敗時はmerge commitやpartial stagingを残さず、開始前のmain品質証跡bytes/stage状態を復元する。
- [ ] AC-3: planning/completion actionは一回の呼出しにつきlock、scope検査、全体validation、commitを各一回だけ行う。parentがvalidation前に固定した全変更内容とhookのstaged内容が一致する場合だけ重複validationを省略し、不一致・通常commitはhookで検証する。
- [ ] AC-4: candidateの正本はcompletion前にmain側でdirtyなHANDOVERの`candidate_commit`一箇所だけとし、candidate commit自身やcandidate branchへHANDOVERを書かない。candidate tree、許可製品差分、managed digestはGitから導出し、REVIEW_RESULT/QA_RESULT/HANDOVER間の同値転記と形式照合を削除する。
- [ ] AC-5: Reviewerは独立Agent identity、PASS、candidate差分/DEV check証跡を確認した事実を記録する。値が一つしかないversion/mode fieldや同一`make check`再実行は要求しない。QAの独立focused rerun、P0/P1拒否、Main merge所有は維持する。
- [ ] AC-6: `estimate_points`と予定行数/ファイル数の算術完全一致を完了ゲートから削除する。見積値は計画判断と傾向計測の参考情報として残し、不一致だけでDEVを停止しない。
- [ ] AC-7: Wiki receiptの全Task必須を削除し、再利用可能な知識がある場合だけWikiを更新する。知識がないTaskはreceiptなしで完了でき、Wiki/HANDOVER hashの順序依存を持たない。
- [ ] AC-8: CF-1〜CF-7を削除する。candidate変更後は影響するfocused testを再実行し、影響を限定できない場合だけfull rerunする。証拠チェックリストによるQA結果の転用はしない。
- [ ] AC-9: QAケースの標準証跡をケースID、HANDOVER candidate、command、結果に縮小する。環境差、cache、exit詳細、外部artifact、未実施理由は該当する場合だけ記録し、artifact hashの形式だけを検査するルールは持たない。
- [ ] AC-10: observed model/effortが宣言と異なる場合は警告して実際の値を記録するが、それだけで全面停止しない。role欠落、DEV/Reviewer/QA同一人物、sandbox/権限境界不明は従来どおり停止する。
- [ ] AC-11: Task startはscaffold、branch、worktreeだけを原子的に準備し、main commitを作らない。planning commit後にTask branch/worktreeをそのcommitへfast-forwardし、MainはDEV完了時に製品差分を一度だけcommitする。completionはmerge-baseからcandidateまでがその単一commitであることを検査する。`task-scope-check` Make targetは`ALLOW_MERGE=1`をmerge許可へ変換し、未指定時はdirect product pathを拒否する。標準工程はplanning commit、candidate commit、completion no-ff merge commitの3件であり、手戻り・障害復旧・live E2Eがある場合だけ追加transactionを許す。completion後にしか分からない`merged_commit`/`tested_commit`等は証跡へ後追い転記せずGit履歴から導出する。
- [ ] AC-12: process tests、root `make check`、`make task-check TASK=TASK-0037`がPASSする。実装・test行数は品質ゲートにせず、冗長な検査や形式項目を増やさない。
- [ ] AC-13: 開発文書は「最初は維持すべき最小チェックだけ」を既定とする。新しい必須rule/machine gateは、複数Taskで反復した具体的なfailure、または単発でも秘密漏洩・権限逸脱・回復不能変更になり得る重大failureと、そのruleが直接検出する方法が示せる場合だけ追加する。10 Task完了ごとの既存retrospective内で、ruleの検出実績、false positive、所要時間、保守費を既存ログから確認し、効果の薄いruleを削除/警告化する。専用checklist、version field、Task、棚卸しcommitは作らない。

### 安定した参照

| 参照ID | 対象 | 固定改訂/ダイジェスト | 用途 |
|---|---|---|---|
| REF-1 | TASK-0036 lifecycle commits | `c3c84f0..c69bafb`、2026-08-01 | phase時間、19 commit、証跡gap、Wiki順序失敗の実測 |
| REF-2 | evidence transaction | main `c69bafb`の`scripts/task/unified-lifecycle.mjs` | action scope、lock、validation、commit |
| REF-3 | task validator/tests | main `c69bafb`の`check-task.mjs`とprocess tests | candidate binding、Done gate、回帰 |
| REF-4 | reviewer/process contracts | main `c69bafb`の`AGENTS.md`と`docs/development/` | 削除対象と維持する品質契約 |

### 依存状態

| 依存 | 状態 (`ready` / `pending`) | planning参照 | `ready`後に固定する値 |
|---|---|---|---|
| なし | `ready` | N/A | N/A |

### 許可パス

- `scripts/task/unified-lifecycle.mjs`
- `scripts/task/unified-lifecycle.test.mjs`
- `scripts/task/lib.mjs`
- `scripts/task/work-pre-commit.mjs`
- `scripts/task/check-task.mjs`
- `scripts/task/development-process.test.mjs`
- `schemas/operations/backlog.schema.json`（完了時の自己参照`merged_commit`必須を外し、checkerへ一本化する互換修正）
- `templates/task/HANDOVER.md`
- `templates/task/REVIEW_RESULT.md`
- `templates/task/QA_PLAN.md`
- `templates/task/QA_RESULT.md`
- `templates/task/PLAN.md`
- `Makefile`
- `AGENTS.md`
- `docs/development/README.md`
- `docs/development/development-process.md`
- `docs/development/code-review.md`
- `docs/development/task-management.md`
- `docs/development/qa.md`
- `docs/development/agent-roles.md`
- `docs/development/git-worktree.md`

### 完了経路preflight

| 確認対象 | 結果 | コマンドまたは根拠 |
|---|---|---|
| 完了checker | ready | 現mainの`make task-check`と`make work-check`をTASK-0036完了時にPASS |
| 権限 | ready | MainだけがGit write。Agentは許可path編集だけ |
| 依存状態と参照 | ready | REF-1〜4をmain `c69bafb`へ固定 |
| 生成物の有無と更新方法 | ready | glossary/generated artifact変更なし。Node testsは既存fixtureだけ |
| 割当ワークツリー | ready | `worktrees/TASK-0037-streamline-task-gates` |
| Lapログの書込・Schema・`repository annotation` | not-applicable | 20分改善はcommit timestampとcommand計測を使用し、新Lapを開始しない |

### 未決事項

- なし

### `Dependency-ready reconciliation`

<!-- 依存ready時にMainがready参照、planning参照との差分、AC/設計/scope/QAへの影響、再承認結果を追記する。依存なし又は未readyならN/Aとする。 -->

- N/A

## 背景

TASK-0036は10:33:56のTask startから11:18:52のdoneまで44分56秒を要した。計画ゲート9分11秒、DEV〜最終candidate 19分33秒、candidate証跡3分58秒、REVIEW/QA 7分31秒、merge後完了4分43秒で、main commitは19件だった。実装不具合はMain早期監査で2件、用語lintで1件検出した一方、同一check再実行は追加不具合を検出せず、個別evidence commitと省略digest修正、Wiki receipt順序修正が納期を悪化させた。

## 検討すべき設計観点

- 独立性はAgentの入力・実行開始・固定candidateで保証し、証跡値の重複やcommit数では保証しない。
- batchは許可ファイル集合を固定するが、未変更ファイルをdirtyにすることは要求しない。
- ReviewerとQAの検出責務を分け、同じcommandの重複は別の失敗仮説がある場合だけ許す。
- direct-mainとPRのscope検査は同じ`scopeCheck`正本を使い、Make wrapperで引数を失わない。
- 形式値ではなく、Gitから導出できる事実と実行結果だけを機械検査する。
- 新しいruleを追加すること自体を品質改善と数えず、再発したfailureを直接防止できる最小の仕組みだけを採用する。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] 選択した`change_class`の完了経路と`make check`を満たしている。
- [ ] 製品変更の場合: 実装、テスト、文書、同一案の独立REVIEW/QA、`merge_tree`確認、環境依存ケース、Wiki取り込みが完了している。
- [ ] 安全契約変更の場合: 独立計画レビュー、契約検査、`no-ff merge`、案/merge tree一致が完了し、製品REVIEW/QA PASSやWiki receiptを代用証跡として作成していない。

## 関連コンテキスト

### 意味 Wiki

- Task lifecycle、candidate evidence、review/QA、Wiki receipt。

### 判断

- 検出力を維持しないtransaction/checkは標準経路から削除し、既存個別actionは回復用互換入口として残す。

### 適用しなかった重要な判断

- Task分割: 実装範囲は局所的なlifecycle/checker/test/docsで約1,000行に収まるため採用しない。
- PR必須化: userがdirect-mainを選んでおり、今回の改善対象と逆なので採用しない。
