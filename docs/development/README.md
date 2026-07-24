# 開発ガイドライン

このディレクトリはKakesuの開発プロセスと品質ゲートの正本である。Task、バックログ、実行証跡、開発用Wikiは製品と同じリポジトリのmain ワークツリーで管理する。

## 文書一覧

| 文書 | 内容 |
|---|---|
| [開発プロセス](development-process.md) | PLAN、DEV、candidate-boundなREVIEW/QA、差し戻しの全体フロー |
| [Agent責務](agent-roles.md) | main、Planner、DEV、レビュアー、QA、Wiki Agentの権限境界 |
| [Task管理](task-management.md) | Task契約、証跡、バックログ、Epic進捗 |
| [Gitとワークツリー](git-worktree.md) | ブランチ、ワークツリー、コミット、マージ、後片付け |
| [コードレビュー](code-review.md) | 独立レビューの入力、観点、重大度、PASS条件 |
| [QA](qa.md) | QA計画、実施モード、証跡監査、FAIL分類、carry-forward、revertとバグ化 |
| [コーディングガイドライン](coding/README.md) | 言語、Schema、文書ごとの実装規約 |

## リポジトリ境界

```text
agent-harness/main                 製品とmain管理証跡の正本
agent-harness/worktrees/TASK-...   main管理証跡を除外した製品変更用ワークツリー
```

`backlog.yaml`、`tasks/`、`wiki/`、`lap30/`、運用viewerはmainだけで更新する。製品変更用ワークツリーはリポジトリ直下の`worktrees/`に置き、このディレクトリとmain管理証跡をGit管理対象の作業領域から除外する。

clone後は`core.hooksPath=.githooks`を設定する。子Agentの標準経路は内部の`agents.spawn_agent`であり、`task_name`（識別子）と`agent_type`（ロール選択）を分離し、異種ロールには`fork_turns="none"`を明示する。ロール対応と`model/effort`の照合、異常時の停止・証跡化、`Explorer`の一問・`read-only`・再委譲禁止の契約は[Agent責務](agent-roles.md)を参照する。案/treeの固定とQA モードの割当は[QA](qa.md)を参照する。

`agent_type`または内部`Spawn Agent`が利用できない場合、または起動後の`model/effort`が契約と不一致の場合、親は原因を記録して停止する。main管理証跡は親が`make evidence-commit TASK=... ACTION=...`で公開し、共通ロック、action別スコープ、hook、最大2回のリモート 再試行を一続きで所有する。PLAN→DEVの後、レビュアーとQAは同一composite 案から独立かつ並行に評価する。両PASS後にMainが`make task-pr`を実行し、必須 check付きmerge コミット auto-mergeを有効にする。
