# 開発ガイドライン

このディレクトリはKakesuの開発プロセスと品質ゲートの正本である。Task、バックログ、実行証跡、開発用Wikiは製品コードと分離した`agent-harness-work`リポジトリで管理する。

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
agent-harness/       製品コード、開発規約、再利用可能なツール
agent-harness-work/  バックログ、Task証跡、Wiki、Decision
```

運用リポジトリは独立したGitリポジトリとし、`main`一本で運用する。製品リポジトリのトピックブランチとワークツリーは運用リポジトリの`worktrees/`に置くが、運用リポジトリ自身のGit管理対象にはしない。

cloneまたは初期作成後に`make work-init`を一度実行し、共有`.githooks/pre-commit`を有効にする。子Agentの標準経路は内部の`agents.spawn_agent`であり、`task_name`（識別子）と`agent_type`（ロール選択）を分離し、異種ロールには`fork_turns="none"`を明示する。ロール対応と`model/effort`の照合、異常時の停止・証跡化、`Explorer`の一問・`read-only`・再委譲禁止の契約は[Agent責務](agent-roles.md)を参照する。案/treeの固定とQA モードの割当は[QA](qa.md)を参照する。

`agent_type`または内部`Spawn Agent`が利用できない場合、または起動後の`model/effort`が契約と不一致の場合だけ、親は原因を記録し、（不一致なら子の成果を採用せず停止・証跡化した後に）`fallback`可否を判断する。選択した場合に限り、親が`make work-agent`（`Explorer`は一問専用の`make explorer-agent`）を`fallback`として使う。運用リポジトリへ証跡を書く場合、ネイティブ/`fallback`を問わず親が共通ロックを保持し、スコープ検査、`hook`、`stage`、`commit`、事後検査を所有する。`fallback`ランチャーはこのロックを編集開始からコミット後検査まで保持し、フックがSchema、フェーズゲート、action別変更範囲を検証する。生成アダプターの`make work-config-sync`はAgentを起動しない専用の親所有経路であり、同じロックとフックを使って生成、コミット、事後検査を一続きで行う。PLAN→DEVの後、レビュアーとQAは同一案から独立かつ並行に評価する。DEVとレビュアー/QAの分離、子のGit禁止、Main所有の承認・統合は起動方式によらず不変である。マージ後は`merge_tree`と案 treeを比較し、環境依存ケースだけを確認する。
