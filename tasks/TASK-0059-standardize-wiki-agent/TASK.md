---
task_id: "TASK-0059"
title: "Wiki Agentを標準spawn経路へ統合する"
status: draft
created_at: "2026-08-02"
---

# TASK-0059 Wiki Agentを標準spawn経路へ統合する

## `Planning input packet`（Main Agent所有）

このsectionをPlannerとQAへ渡す唯一の`planning input packet`とし、各内容をPLAN/QA_PLANへ複製しない。

### 目的

Wiki Agentだけが独立`codex exec` launcherを使う例外を廃止し、他ロールと同じ内部`agents.spawn_agent`へ統合する。Wiki Agentは編集だけを担当し、親Mainが単一writer調整、許可path確認、検証、commitを所有する。Wiki receiptは明示的なWiki ingestの任意成果物として残し、Task完了条件にはしない。

### 対象と対象外

#### 対象

- `scripts/task/run-wiki-agent.mjs`、`make wiki-context`、`make wiki-ingest`、`WIKI_PROFILE/MODEL/EFFORT`と`legacy-wiki`経路を削除する。
- `.codex`へ標準`wiki` roleを追加し、Mainが`agents.spawn_agent(task_name=..., agent_type="wiki", fork_turns="none", ...)`で起動する契約を定義する。
- Wiki AgentはMainが指定したWiki pathだけを編集し、stage/commit/merge、`.git`書込、別Agent起動を行わない。Mainはwriterを直列化し、既存の共通lock付きpublish transaction、許可path検査、`work-check`、commitを所有する。
- launcher固有testを削除し、標準role registry、launcher不在、Main所有境界、receipt任意性を既存process testで検出する。
- `AGENTS.md`、Wiki規約、Agent責務を新しい起動・所有境界へ同期する。

#### 対象外

- Wiki本文の意味変更、既存receipt/Decisionの削除・書換え、Wiki Schema変更。
- `agents.spawn_agent`ランタイム自体の実装、製品runtime、Memory PlaneのWiki Agent設計変更。
- Planner/DEV/Reviewer/QA/Explorerの起動経路やモデル契約の変更。
- completion gateへのWiki receipt必須化、暗黙のWiki起動、専用の外部送信承認フロー。
- Kakesuの製品コード/test/runtime/build設定、Schema、製品依存、生成製品入力/成果物、および`tools/dev-agent-harness` runtimeは変更しない。開発workflow toolingのMake入口、実行script、process test、`.codex` role registryは変更対象とする。

### 受け入れ条件

<!-- AC-IDはTask内で一意かつ安定させ、観測可能な結果をここに一度だけ記載する。 -->

- [ ] AC-1: 実行・設定pathに`run-wiki-agent.mjs`、`legacy-wiki`、Wiki専用`codex exec`、Wiki専用model/profile/effort変数、`wiki-context`/`wiki-ingest` Make入口が残らない。Task証跡内の削除理由の記録は検索対象から除く。
- [ ] AC-2: `.codex`の正規role registryに`wiki`があり、Terra/medium、workspace-write、編集専用、子Agent禁止、Git書込禁止を宣言する。規約はMainによる標準`agents.spawn_agent`の`agent_type="wiki"`と`fork_turns="none"`を唯一の起動経路とする。
- [ ] AC-3: Mainは同時writerを作らず、起動前に対象と許可pathを固定し、子の終了後に差分scope、`make wiki-index`（索引変更時）、`make work-check`を確認して、既存の共通lock付きMain transactionだけでcommitする。Wiki Agentはlock、validation、stage、commitを所有しない。
- [ ] AC-4: Wiki依頼がないTaskはWiki Agentもreceiptもなしで完了できる現行契約を維持する。receipt Schemaと既存receipt検証は残すが、launcher削除に伴ってreceipt作成を完了条件や暗黙処理へ戻さない。
- [ ] AC-5: launcher固有fixtureを削除し、standard wiki role、launcher/Make入口不在、親所有境界、receipt任意性をfocused process testsが検出する。影響するNode tests、docs lint、work-check、root make check、git diff checkがPASSする。
- [ ] AC-6: 変更は宣言した開発workflow tooling pathだけに限定し、Kakesu product runtimeとtools/dev-agent-harness runtimeを変更しない。`wiki/AGENTS.md`はMain管理差分としてcompletion transactionで統合し、candidateへ含めない。既存TASK-0058 Wiki差分は検証済み完了物として保持し、遡及変更しない。

### 安定した参照

| 参照ID | 対象 | 固定改訂/ダイジェスト | 用途 |
|---|---|---|---|
| REF-1 | Wiki launcher/Make入口 | main `09a73422` | 削除対象の独立`codex exec`経路 |
| REF-2 | 標準Agent role registry | main `09a73422`の`.codex/`と`agent-routing.mjs` | `wiki` role統合先 |
| REF-3 | Main transaction / Wiki任意契約 | main `09a73422`の`unified-lifecycle.mjs`、TASK-0037 | lock・scope・validation・commitとreceipt非必須の維持 |

### 依存状態

| 依存 | 状態 (`ready` / `pending`) | planning参照 | `ready`後に固定する値 |
|---|---|---|---|
| なし | `ready` | N/A | N/A |

### 許可パス

- `Makefile`
- `scripts/task/run-wiki-agent.mjs`（削除）
- `scripts/task/agent-routing.mjs`
- `scripts/task/development-process.test.mjs`
- `scripts/task/unified-lifecycle.test.mjs`
- `.codex/config.toml`
- `.codex/agents/wiki.toml`（新規）
- `AGENTS.md`
- `wiki/AGENTS.md`
- `docs/development/agent-roles.md`

### 完了経路preflight

| 確認対象 | 結果 | コマンドまたは根拠 |
|---|---|---|
| 完了checker | `ready` | `wiki/AGENTS.md`をcandidate外のMain管理差分として、同一candidateの独立REVIEW/QA後にcompletion transactionで統合する。planning review、製品変更のゲート、no-ff tree同一性を使用する |
| 権限 | `ready` | Wiki子Agentは編集のみ、MainだけがGit/lock/scope/validationを所有する |
| 依存状態と参照 | `ready` | TASK-0037でreceipt任意化済み、TASK-0058はmain反映済み |
| 生成物の有無と更新方法 | `ready` | generated Wiki/glossaryなし。`.codex/agents/wiki.toml`だけ新規正規設定 |
| 割当ワークツリー | `ready` | `worktrees/TASK-0059-standardize-wiki-agent` |
| Lapログの書込・Schema・`repository annotation` | `not-applicable` | 新規Schema/log/annotationなし |

### 未決事項

- なし

### `Dependency-ready reconciliation`

<!-- 依存ready時にMainがready参照、planning参照との差分、AC/設計/scope/QAへの影響、再承認結果を追記する。依存なし又は未readyならN/Aとする。 -->

- N/A

## 背景

TASK-0058完了時、Wikiだけが独立`codex exec`を起動したため、専用の外部送信承認とlauncher内部lock/rollbackが標準subagent経路から分岐した。またTaskを`done`へ先に進めるとlauncherの全体検証がmerge未完了を理由に失敗し、状態を`qa`へ戻して再実行する余分な往復が発生した。receipt自体は既に任意なのに、専用launcherが必須工程のように見えることも運用を複雑化した。

## 検討すべき設計観点

- Agent起動はroleごとに別process launcherを作らず、標準spawnと正規`.codex` roleへ一本化する。
- 子Agentの編集権限と、Mainの公開権限を分離する。子にlockやGit commitを渡さない。
- receiptの既存整合検証は有用なので残し、「作る場合は正しい」と「必ず作る」を混同しない。
- 削除で達成できる箇所に新しいwrapper、token、version、checklistを導入しない。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] product変更経路と`make check`を満たしている。
- [ ] 実装、テスト、文書、同一candidateの独立REVIEW/QA、completion後に必要な環境依存ケース確認が完了している。

## 関連コンテキスト

### 意味 Wiki

- `wiki/AGENTS.md`、`docs/development/agent-roles.md`

### 判断

- Wiki役割は維持するが、起動とpublish所有権は標準子Agent契約へ統合する。

### 適用しなかった重要な判断

- 独立launcherを内部spawn APIへ置換する新wrapper: 起動例外を別名で残すだけなので不採用。
- receipt/schemaの削除: 既存の任意ingest成果物の整合検証まで失うため不採用。
