---
task_id: "TASK-0062"
title: "Wiki索引生成をMain publish transactionへ統合する"
status: draft
created_at: "2026-08-02"
---

# TASK-0062 Wiki索引生成をMain publish transactionへ統合する

## `Planning input packet`（Main Agent所有）

このsectionをPlannerとQAへ渡す唯一の`planning input packet`とし、各内容をPLAN/QA_PLANへ複製しない。

### 目的

標準spawnされたWiki AgentがMain worktreeへ編集を残した後、Mainが`ACTION=wiki`の一つの共通ロック付きpublish transactionで索引生成、スコープ検査、`work-check`、stage、commit、pushまで完了できるようにする。dirty状態で開始できないstandalone `wiki-index`を必須工程にしている矛盾を解消する。

### 対象と対象外

#### 対象

- `evidence-commit`の`wiki` actionだけが、共通lock取得後にdirty Wiki差分から`wiki/index.json`を決定的に生成し、その生成結果を同じscope検査・validation・commitへ含める。
- Wiki publishは一つのcommitだけを作り、索引専用commitや子Agent commitを作らない。
- MainはWiki Agent終了後に許可pathを確認し、`make evidence-commit TASK=... ACTION=wiki`を実行する。transactionが索引生成と`work-check`を所有する。
- 生成後に許可外path、Schema/リンク/digest不整合、hook、commit、pushが失敗した場合はfail-closedに停止し、元のWiki編集を失わず再試行可能にする。
- standalone `make wiki-index`は保守用generatorとして残してもよいが、標準publish経路・自動commit・外側lock偽装には使わない。
- process testsと統制文書を新しいMain所有境界へ同期する。

#### 対象外

- Wiki Agentのrole、標準spawn、編集内容、receipt Schema、Decision/Semantic本文の品質判断は変更しない。
- Wiki receiptをTask完了条件へ戻さず、暗黙ingestや独立launcherを再導入しない。
- 他のevidence action、planning/completion transaction、Kakesu runtime、`tools/dev-agent-harness` runtime、Schema、依存、生成製品成果物は変更しない。
- 現在stash中のTASK-0024/0026/0030/0037 Wiki差分をcandidateへ含めない。修正merge後に復元して別のWiki publish commitにする。

### 受け入れ条件

<!-- AC-IDはTask内で一意かつ安定させ、観測可能な結果をここに一度だけ記載する。 -->

- [ ] AC-1: dirtyな許可Wiki差分がある状態から`ACTION=wiki`が共通lockを取得し、Semantic/Decisionの現行内容から`wiki/index.json`を生成して同じ一commitへ含められる。
- [ ] AC-2: transactionは生成後の全変更pathを`wiki/semantic/**`、`wiki/decisions/**`、`wiki/ingestions/**`、`wiki/index.json`だけに限定し、許可外差分があればstage/commit前に固定拒否する。
- [ ] AC-3: index生成後の`work-check`、hook、commit、pushを既存transaction内で行い、成功時はWiki publish commit一つ、失敗時は子commitや索引専用commitを残さず編集差分を保持する。
- [ ] AC-4: Wiki Agentは引き続き編集のみ、Mainがscope/index/validation/Gitを所有し、receiptは明示ingest時だけの任意成果物である。
- [ ] AC-5: 他actionの挙動、通常のstandalone index build、既存Wiki Schema/immutable Decision検査を回帰させず、focused Node tests、root `make check`、`git diff --check`がPASSする。
- [ ] AC-6: candidateは承認済み6パス以内かつ小規模で、stash中のWiki本文/receipt、Kakesu runtime、dev-agent-harness runtimeを含まない。

### 安定した参照

| 参照ID | 対象 | 固定改訂/ダイジェスト | 用途 |
|---|---|---|---|
| REF-1 | Wiki標準spawn契約 | main `7534307`の`AGENTS.md`、`wiki/AGENTS.md` | Mainがindex/validation/publishを所有する現行契約 |
| REF-2 | 現行Wiki generator/evidence transaction | main `7534307`の`wiki-index.mjs`、`unified-lifecycle.mjs` | clean-only generatorとdirty対応transactionの統合点 |
| REF-3 | 再現した失敗 | 2026-08-02第1 ingest batch | dirty Wiki差分後の`make wiki-index`がclean preconditionで拒否 |

### 依存状態

| 依存 | 状態 (`ready` / `pending`) | planning参照 | `ready`後に固定する値 |
|---|---|---|---|
| なし | `ready` | N/A | N/A |

### 許可パス

- `scripts/task/unified-lifecycle.mjs`
- `scripts/task/unified-lifecycle.test.mjs`
- `scripts/task/wiki-index.mjs`
- `scripts/task/development-process.test.mjs`
- `AGENTS.md`
- `docs/development/agent-roles.md`
- `wiki/AGENTS.md`（Main管理差分。candidateへ含めない）

### 完了経路preflight

| 確認対象 | 結果 | コマンドまたは根拠 |
|---|---|---|
| 完了checker | `ready` | product経路、同一candidateの独立REVIEW/QA、Main管理`wiki/AGENTS.md`のcompletion統合を使う |
| 権限 | `ready` | Wiki子は編集だけ。Mainのtransactionだけがlock/index/validation/Gitを所有する |
| 依存状態と参照 | `ready` | TASK-0059でstandard wiki role統合済み。第1バッチ差分はstashへ安全に退避済み |
| 生成物の有無と更新方法 | `ready` | `wiki/index.json`は同じtransaction内で`buildWikiIndex`から決定的生成する |
| 割当ワークツリー | `ready` | `worktrees/TASK-0062-wiki-publish-transaction` |
| Lapログの書込・Schema・`repository annotation` | `not-applicable` | 新規log/Schema/annotationなし |

### 未決事項

- なし

### `Dependency-ready reconciliation`

<!-- 依存ready時にMainがready参照、planning参照との差分、AC/設計/スコープ/QAへの影響、再承認結果を追記する。依存なし又は未readyならN/Aとする。 -->

- N/A

## 背景

TASK-0059でWiki Agentを標準spawnへ統合し、子からlockとGitを取り上げた。再起動後の実ingestは成功したが、Mainが後処理で`make wiki-index`を呼ぶと、Wiki差分がdirtyであるためgeneratorのclean preconditionに拒否された。旧launcherが外側でlockを保持していた前提が残っており、標準経路では安全に公開できない。

## 検討すべき設計観点

- lockを無効化するenvironment偽装や一時wrapperでは回避しない。
- index生成、validation、commitの間に別writerが入らない一つのtransactionにする。
- actionが`wiki`以外のときindex生成を行わない。
- scope検査は生成後のindexを含む最終path集合へ適用する。
- 既存Wiki差分は修正candidateから分離し、修正後に正規Wiki actionで公開する。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] 選択した`change_class`の完了経路と`make check`を満たしている。
- [ ] 製品変更の場合: 実装、テスト、文書、同一案の独立REVIEW/QA、完了後の環境依存ケース確認が完了している。
- [ ] 安全契約変更の場合: Mainの意図・スコープ・受け入れ経路確認、契約検査、許可された統制文書差分の確認が完了している。

## 関連コンテキスト

### 意味 Wiki

- `wiki/AGENTS.md`、`docs/development/agent-roles.md`

### 判断

- Wiki publishはMainの`ACTION=wiki`一transactionへ集約し、standalone index commitを標準工程から外す。

### 適用しなかった重要な判断

- なし
