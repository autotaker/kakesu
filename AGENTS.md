# 開発作業の共通規約

このリポジトリの変更は、原則として外部の運用リポジトリ`../agent-harness-work`に登録したTaskを起点に進める。正本は[開発プロセス](docs/development/README.md)とする。

## 必須事項

1. Taskごとに`task/TASK-NNNN-short-slug`ブランチと専用ワークツリーを使う。
2. `PLAN / DEV / QA`の各ゲートを飛ばさない。
3. DEV開始前に承認済み`PLAN.md`と`QA_PLAN.md`を用意する。
4. DEV Agentとレビュアー Agent、DEV AgentとQA Agentを分離する。
5. レビュアー Agentは独立レビューと`make check`を完了し、外部運用リポジトリの`REVIEW_RESULT.md`へ証跡を残す。
6. main Agentだけが`main`へ`--no-ff`でマージする。
7. マージ後QAのFAILは実装不具合と決めつけず、[QAガイドライン](docs/development/qa.md)に従って原因を分類する。
8. 配下に別の`AGENTS.md`がある場合は、その追加手順も守る。
9. 運用リポジトリへ証跡を書くAgentは`make work-agent TASK=TASK-NNNN ACTION=<action>`から起動し、共通ロックを実行全体で保持する。直接並行編集しない。

## 共通検査

```sh
make check
make task-check TASK=TASK-NNNN
```

外部運用リポジトリの場所は`WORK_ROOT`で上書きできる。既定値は`../agent-harness-work`である。
