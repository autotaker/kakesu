# 開発ガイドライン

このディレクトリはKakesuの開発プロセスと品質ゲートの正本である。Task、バックログ、実行証跡、開発用Wikiは製品と同じリポジトリのmain ワークツリーで管理する。

## 文書一覧

| 文書 | 内容 |
|---|---|
| [開発プロセス](development-process.md) | planning、DEV、独立REVIEW/QA、完了の3 コミットフロー |
| [Agent責務](agent-roles.md) | main、Planner、DEV、レビュアー、QA、Wiki Agentの権限境界 |
| [Task管理](task-management.md) | Task契約、証跡、バックログ、Epic進捗 |
| [Gitとワークツリー](git-worktree.md) | ブランチ、ワークツリー、コミット、マージ、後片付け |
| [コードレビュー](code-review.md) | 独立レビューの入力、観点、重大度、PASS条件 |
| [QA](qa.md) | QA計画、実施モード、証跡監査、マージ後確認 |
| [コーディングガイドライン](coding/README.md) | 言語、Schema、文書ごとの実装規約 |

## リポジトリ境界

```text
agent-harness/main                 製品とmain管理証跡の正本
agent-harness/worktrees/TASK-...   main管理証跡を除外した製品変更用ワークツリー
```

`backlog.yaml`、`tasks/`、`wiki/`、`lap30/`、運用viewerはmainだけで更新する。製品変更用ワークツリーはリポジトリ直下の`worktrees/`に置き、このディレクトリとmain管理証跡をGit管理対象の作業領域から除外する。

clone後は`core.hooksPath=.githooks`を設定する。子Agentの標準経路は内部の`agents.spawn_agent`であり、`task_name`（識別子）と`agent_type`（ロール選択）を分離し、異種ロールには`fork_turns="none"`を明示する。識別情報、role、サンドボックス・権限境界の確認と、観測された`model/effort`不一致の警告記録は[Agent責務](agent-roles.md)を参照する。案固定とQAモードの割当は[QA](qa.md)を参照する。

`agent_type`または内部`Spawn Agent`が利用できない場合、または識別情報/role/サンドボックス・権限境界を観測できない場合、親は原因を記録して停止する。`model/effort`の不一致はrequested/observed値とランタイム条件を警告として記録し、境界が明確なら継続する。main管理証跡は親が`planning-gate`/`completion-gate`で公開し、共通ロック、スコープ、hookを一続きで所有する。標準経路はplanning コミット、製品案 コミット、分岐を残すmergeの3 commitsである。
