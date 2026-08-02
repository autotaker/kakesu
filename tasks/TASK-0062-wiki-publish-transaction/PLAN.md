---
task_id: "TASK-0062"
change_class: "product"
status: "approved"
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "sol-high"
approved_dev_profile_reason: "共有lock、scope検査、commit/push transactionの自己更新を含み、失敗時のGit状態と他action非回帰を同時に保つ高リスク変更のため。"
approved_dev_profile_risk_signals:
  - "shared publish lock"
  - "generated index integrity"
  - "commit/push transaction reconciliation"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-02T00:04:11Z"
classification_approved_by: "main-agent-sol-high"
classification_approved_at: "2026-08-02T00:04:11Z"
classification_approval_reason: "Wiki publish transaction、index generator、process testsという外部観測可能な開発workflowを変更するため。"
---

# PLAN: Wiki索引生成をMain publish transactionへ統合する

## 根拠と分類

本計画の唯一の要求根拠は`TASK.md`の`Planning input packet`である。evidence publish transaction、index generator、process testsという外部観測可能な開発workflowを変更するため、`change_class`は`product`とする。Kakesu runtime、`tools/dev-agent-harness` runtime、Schema、依存、生成製品成果物、および他evidence actionは変更しない。

MainはDEV開始前に本PLANと独立QA_PLANについて、意図、scope、受け入れ経路の一致を確認する。本PLANの独立レビューは行わない。DEV後の固定candidateに対する独立REVIEW/QAは必須である。

## 変更境界

candidateへ含める変更は次の6パスだけであり、小規模に保つ。

- `scripts/task/unified-lifecycle.mjs`
- `scripts/task/unified-lifecycle.test.mjs`
- `scripts/task/wiki-index.mjs`
- `scripts/task/development-process.test.mjs`
- `AGENTS.md`
- `docs/development/agent-roles.md`

`wiki/AGENTS.md`はcandidateに含めないMain管理差分であり、同一candidateの独立REVIEW/QA後にcompletion transactionへ統合する。stash中のTASK-0024/0026/0030/0037 Wiki本文・receiptもcandidate外であり、修正merge後に別の正規Wiki publish commitとして復元する。Wiki Agent role、標準spawn、編集内容、receipt Schema、Decision/Semantic本文の品質判断、receipt任意性、独立launcherは変更しない。

## 実施設計

1. `ACTION=wiki`だけを既存evidence transactionへ拡張する。
   - `unified-lifecycle.mjs`は既存の共通lockを取得した後、`ACTION=wiki`の場合だけ、dirty Wiki編集を入力として既存`buildWikiIndex`を呼び出し、`wiki/index.json`を決定的に生成する。
   - generatorがclean worktreeを要求する既存standalone経路を、transaction内でlockを無効化する環境偽装、外側lock、一時wrapperで回避しない。transaction自身が同じlockの所有者として生成を実行する。
   - `ACTION`が`wiki`以外の場合はindex生成を呼ばず、既存のplanning/completionおよび他evidence actionの挙動を不変とする。

2. 生成後の最終差分を一つのpublish単位として検証する。
   - transactionは生成前のdirty inputを許可Wiki pathに限定し、生成後の最終変更集合を`wiki/semantic/**`、`wiki/decisions/**`、`wiki/ingestions/**`、`wiki/index.json`だけに再検査する。
   - 許可外path、Schema/リンク/digest不整合、`work-check`、hookのいずれかが失敗した場合はstage/commit/push前にfail-closedで停止し、元のWiki編集と必要なら生成済みindex差分を保持して再試行可能にする。
   - validationを通過してから、既存transactionのstage、commit、pushを一度だけ実行する。hookまたはcommit前のfailureではcommitを作らずdirty差分を保持する。push failureは既存transactionのreconciliation semanticsに従い、子Agent commitまたは追加のindex専用commitを作らない。

3. standalone index buildの役割を限定する。
   - `wiki-index.mjs`は保守用の通常standalone generatorとして残す。
   - 標準Wiki publishはMainがWiki Agent終了後に許可pathを確認して`make evidence-commit TASK=... ACTION=wiki`を一度実行する経路だけとする。標準publishから外部の`make wiki-index`、自動commit、外側lockの偽装を除外する。

4. MainとWiki Agentの所有境界を統制文書へ同期する。
   - `AGENTS.md`と`docs/development/agent-roles.md`に、Wiki Agentは編集だけ、Mainの`ACTION=wiki` transactionだけがlock、index生成、最終scope検査、`work-check`、Gitを所有することを記載する。
   - receiptは明示ingest時だけの任意成果物であり、Wiki依頼のないTaskがAgent/receiptなしで完了できる契約は維持する。

5. focused process testsを追加・更新する。
   - dirtyな許可Wiki差分から、同一lock下でindexを生成し、最終scope/validationを通して一commitへ含める`ACTION=wiki`の成功ケースを検証する。
   - 生成後の許可外path、Schema/リンク/digest不整合、`work-check`/hook/commit前failureでfail-closedになり、commitを作らず編集を保持するケースを検証する。push failureは既存transactionのreconciliationに従い、追加のindex-only commitを作らないことを検証する。
   - `ACTION=wiki`以外でindex生成をしないこと、standalone generatorの通常利用、immutable Decision検査、receipt任意性、既存Wiki Schemaを回帰させないことを検証する。

## 受け入れ条件への対応

| AC | 実施・検証の対応 |
|---|---|
| AC-1 | dirtyな許可Wiki差分を入力に、共有lock内で`buildWikiIndex`を実行し、一commitに含める成功ケース。 |
| AC-2 | 生成前入力と生成後の最終変更集合のscope検査、許可外pathのstage前固定拒否。 |
| AC-3 | transaction内の`work-check`、hook、commit、push、一commit成功、hook/commit前failure時のcommitなし・dirty差分保持、push failure時の既存reconciliationと追加index専用commitなしを検証。 |
| AC-4 | Wiki Agent編集専用とMainのindex/validation/Git所有、receipt任意性を統制文書・process testsで維持。 |
| AC-5 | 非wiki action、standalone index build、Schema/immutable Decision検査のfocused回帰とroot `make check`。 |
| AC-6 | candidateの許可6パス・小規模、stash Wiki差分と対象外runtimeが含まれないことをdiffで監査。 |

## 検証計画

- `scripts/task/unified-lifecycle.test.mjs` のdirty Wiki transaction、scope、hook/commit前failure時の差分保持、push failure時の既存reconciliation cases
- `scripts/task/development-process.test.mjs` のMain/Wiki Agent所有境界
- `scripts/task/wiki-index.mjs` に対する通常standalone buildおよび既存Schema/immutable Decision検査
- docs lint
- root `make check`
- `make task-check TASK=TASK-0062`
- `git diff --check`

QA_PLANは、deterministicで上限付きのtransaction成功・拒否・hook/commit前failure時の差分保持およびpush failure時の既存reconciliationケースを`focused-rerun`、candidate-bound証跡とfailure-detection能力の監査を`evidence-review`に割り当てる。実pushに環境依存のlive-e2eが必要なら、承認済み環境と安全なcleanupがある場合だけ実行し、ない場合はblockedまたはnot-applicableの理由を記録する。hermetic PASSで実環境を代替しない。

## 実施・完了経路

DEV開始前にMainがPLANと独立QA_PLANを承認する。DEV Agentは許可6パスだけで製品差分を一回だけcandidateとして固定する。Reviewer AgentとQA Agentは同じcandidateを相互のPASSを待たず独立に評価する。`wiki/AGENTS.md`はcandidate外のMain管理差分として、独立REVIEW/QA後のcompletion transactionへ統合する。Mainだけがcandidate識別子、`--no-ff --no-commit`検査、completion transaction、main統合、マージ後環境依存確認を所有する。

## リスクと復旧

- dirty差分に対してclean-only standalone generatorを呼ぶリスクは、`ACTION=wiki` transaction内だけで共有lock下の生成を行うことで抑える。
- index生成後のscopeが弱くなるリスクは、最終変更集合で再検査して抑える。
- 生成、hook、commit前failureで編集が失われるリスクは、commitなし・dirty差分保持を確認するfailure testsで抑える。push failureは既存transactionのreconciliation semanticsを維持し、追加のindex-only commitを作らない。
- 他actionまたはstandalone generatorへの回帰は、明示的な非wiki actionと通常standalone buildのfocused regressionで抑える。
- stashの本文/receiptがcandidateへ混入するリスクは、candidate scope diffをMainが監査し、修正merge後まで復元しないことで抑える。

復旧時は許可6パスのcandidate差分だけを戻し、既存のstandalone index generatorとevidence transactionの挙動へ復元する。stash中Wiki差分はcandidateに取り込まない。focused tests、`make check`、`make task-check TASK=TASK-0062`、`git diff --check`を再実行する。

## 引き継ぎ条件

DEVは承認済みPLANと独立QA_PLANの後、許可6 candidate pathsだけで開始する。Wiki Agentは編集専用のままとし、Mainだけが`ACTION=wiki`のlock/index/validation/Gitを所有する。standalone index commitは標準経路に含めず、実装後の独立REVIEW/QAとMain completionを必須とする。
