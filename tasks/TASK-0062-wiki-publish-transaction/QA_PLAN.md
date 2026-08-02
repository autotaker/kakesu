---
task_id: "TASK-0062"
change_class: "product"
status: approved
qa_agent: "qa-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-02T00:04:11Z"
revision: 1
implementation_reviewed_at: "2026-08-02T00:27:02Z"
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0062 QA PLAN

## 方針

期待値の正本は `TASK.md` の `Planning input packet` だけとする。PLAN、実装案、DEV自己申告は期待値の根拠にしない。candidateの6 workflow pathとcompletion transactionでのみ統合するMain管理`wiki/AGENTS.md`差分を分離して評価する。対象は`ACTION=wiki`だけのMain-owned publish transactionであり、Wiki Agent、receipt Schema、Wiki本文、他evidence action、Kakesu/runtime、`tools/dev-agent-harness` runtime、Schema、依存、生成物は変更しない。

dirty Wiki差分を含むtemporary repository fixtureで、同じ共通lock内のindex生成→最終scope→`work-check`→stage→一commit→push、失敗時の元編集保持を確認する。live E2Eは割り当てない。

## 受け入れ条件との対応

| ケース ID | AC-ID | 確認内容と failure detection | 実施モード / 理由 |
|---|---|---|---|
| QA-001 | AC-1 | dirtyな許可Wiki差分（Semantic/Decision、必要時ingestion）から`ACTION=wiki`が共通lock取得後に現行内容を使い、決定的な`wiki/index.json`を生成し、全変更を一commitへ含めることを確認する。clean-only standalone generator依存、indexなし、二commit、子Agent commit、lock外生成をnegative fixtureで検出する。 | `focused-rerun` / temporary Git repositoryと決定的index fixtureで、dirty入力から一transactionの受け入れ真実をhermetic・boundedに再現できる。 |
| QA-002 | AC-2 | index生成後の最終path集合だけが`wiki/semantic/**`、`wiki/decisions/**`、`wiki/ingestions/**`、`wiki/index.json`に限定されることを確認する。生成前だけのscope確認、許可外path、index生成後のpath追加、Schema/リンク/digest不整合をstage/commit前の固定拒否としてfailure-detectし、許可外をstageしないことを確認する。 | `focused-rerun` / fixtureにallowed/forbidden pathと壊れたindex入力を注入し、final scopeとfail-closed境界を直接観測できる。 |
| QA-003 | AC-3 | 同一transactionがindex生成後に`work-check`、hook、commit、pushを順に所有し、成功はWiki publish commit一つだけであることを確認する。generator、validation、hook、commit前failureではcommitを作らず既存dirty Wiki編集を保持して再試行可能にすることを確認する。push failureは既存non-planning reconciliationに従うことを回帰監査し、追加のindex-only commitを作らないことを確認する。 | `focused-rerun` / controlled pre-commit failure seam、before/after HEAD/index/worktree snapshot、既存push reconciliation fixtureにより境界をdeterministicに確認できる。 |
| QA-004 | AC-4 | Wiki Agentが編集のみでMainがscope/index/validation/Gitを所有すること、receiptがMain明示ingest時だけの任意成果物であることを`AGENTS.md`、role文書、main-side`wiki/AGENTS.md`とprocess testで確認する。子のlock/index/work-check/stage/commit、暗黙ingest、receipt必須化、独立launcherをnegativeとして検出する。 | `focused-rerun` / source/process fixtureは外部Wikiサービスなしに責務境界と任意receiptを確認できる。 |
| QA-005 | AC-5 | `ACTION=wiki`以外のevidence action、planning/completion transaction、standalone `make wiki-index`の通常clean generator、既存Wiki Schema/immutable Decision検査が回帰しないことを確認する。wiki actionのために外側lockを偽装するenvironment、新wrapper、自動commitをstandalone generatorへ導入することを検出する。 | `focused-rerun` / existing process testsとstandalone generator fixtureはaction分離・既存検査の回帰をhermeticに確認できる。 |
| QA-006 | AC-6 | candidate diffが`wiki/AGENTS.md`を除く許可6 pathに限定され、小規模であり、stash中TASK-0024/0026/0030/0037のWiki本文/receiptを含まないことを確認する。Main-side`wiki/AGENTS.md`はMain所有境界・transaction内index/work-check・receipt任意性だけを同期し、candidateへ混入しないことを監査する。Kakesu/runtime、`tools/dev-agent-harness` runtime、Schema、dependency、生成物の差分を検出する。DEVのroot `make check`/`git diff --check`、Reviewerのcandidate/root check証跡をcommand/cwd/resultまで独立監査し、QAはroot full checkを重複実行しない。 | `evidence-review` / candidate diff、main-side diff、focused test本文とcandidate-bound DEV/Reviewer証跡でscopeと非対象不変を監査できる。 |

## 一つの bounded focused rerun

candidate固定後、QA-001〜005を次の一回だけ実行する。testはdirty Wiki publish、最終scope、failure保持、non-wiki/standalone回帰、receipt任意性の少なくとも一つを壊すと失敗する必要がある。

```sh
node --test scripts/task/unified-lifecycle.test.mjs scripts/task/development-process.test.mjs
```

zero exitだけでは不十分であり、対象testの欠落、skip、弱体化、candidate不一致、またはrequired failure injection/snapshot assertionの欠落は該当ケースをPASSにしない。root `make check`はDEV evidenceとReviewerの独立監査に限り、QAは再実行しない。

## 境界・異常・回帰

- `ACTION=wiki`だけが共通lock内でindex生成を所有する。生成後のfinal scope/validationを通る前にstage/commit/pushせず、成功時は一commitだけである。
- generator、scope、Schema/link/digest検査、work-check、hook、commit前failureはfail closedで、commitを作らず元のdirty Wiki編集を捨てない。push failureは既存non-planning reconciliationを維持し、追加のindex-only commitを作らない。retry/cache/外側lock偽装を追加しない。
- non-wiki action、planning/completion、standalone clean index generatorは変更しない。Wiki Agentは編集のみで、Mainだけがlock/index/validation/stage/commit/pushを所有する。
- receiptは明示ingest時だけの任意成果物であり、Task完了条件へ戻さない。stash中の既存Wiki差分、既存receipt/Decision、Wiki Schemaをcandidateで変更しない。
- candidateへの`wiki/AGENTS.md`混入、またはcompletion transaction外のMain管理差分はFAILである。許可path外、Kakesu/runtime、`tools/dev-agent-harness` runtime、Schema、dependency、生成物、外部観測可能な製品挙動の変更はscope failureとして分類する。

## 実装後の再確認

- [ ] 同一candidateでQA-001〜006を独立に評価し、指定focused Node testsを一回だけ実行した。
- [ ] dirty Wiki差分のlock内index→final scope→work-check→一commit、各failureでの編集保持、non-wiki/standalone回帰、Main/child/receipt境界のnegative failure-detectionを確認した。
- [ ] candidateの6 pathとmerge前main-side`wiki/AGENTS.md`差分を監査し、DEV root `make check`/diff checkとReviewer証跡を確認した。QAはfull checkを重複実行していない。
- [ ] live E2Eを計画せず、stash中Wiki差分、Kakesu/runtime、`tools/dev-agent-harness` runtime、Schema、dependency、生成物が不変であることを確認した。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | 2026-08-02 | qa-agent-terra-medium | TASK-firstの独立QA計画。dirty Wiki publish transaction、final scope、failure保持、non-wiki回帰、Main-owned receipt境界を定義。 | `approved` |
