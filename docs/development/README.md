# 開発ガイドライン

このディレクトリはKakesuの開発プロセスと品質ゲートの正本である。Task、バックログ、実行証跡、開発用Wikiは製品コードと分離した`agent-harness-work`リポジトリで管理する。

## 文書一覧

| 文書 | 内容 |
|---|---|
| [開発プロセス](development-process.md) | PLAN、DEV、QAと差し戻しの全体フロー |
| [Agent責務](agent-roles.md) | main、Planner、DEV、レビュアー、QA、Wiki Agentの権限境界 |
| [Task管理](task-management.md) | Task契約、証跡、バックログ、Epic進捗 |
| [Gitとワークツリー](git-worktree.md) | ブランチ、ワークツリー、コミット、マージ、後片付け |
| [コードレビュー](code-review.md) | 独立レビューの入力、観点、重大度、PASS条件 |
| [QA](qa.md) | QA計画、実施、FAIL分類、revertとバグ化 |
| [コーディングガイドライン](coding/README.md) | 言語、Schema、文書ごとの実装規約 |

## リポジトリ境界

```text
agent-harness/       製品コード、開発規約、再利用可能なツール
agent-harness-work/  バックログ、Task証跡、Wiki、Decision
```

運用リポジトリは独立したGitリポジトリとし、`main`一本で運用する。製品リポジトリのトピックブランチとワークツリーは運用リポジトリの`worktrees/`に置くが、運用リポジトリ自身のGit管理対象にはしない。

cloneまたは初期作成後に`make work-init`を一度実行し、共有`.githooks/pre-commit`を有効にする。運用リポジトリへ書くAgentは`make work-agent`または役割別の専用コマンドから起動する。ランチャーが共通ロックを実行全体で保持し、フックがSchema、フェーズゲート、action別変更範囲をコミット前に検証する。
