# Task管理

## Task契約

`TASK.md`は次の正本である。

- 目的
- 背景と範囲
- 受け入れ条件
- 検討すべき設計観点
- 完成の定義
- 関連するWikiテーマと判断

受け入れ条件は観測可能な結果として書く。実装手段を不必要に固定せず、境界条件、失敗時、互換性、運用確認が必要なら明記する。

## 証跡ファイル

Taskごとに次の6ファイルを持つ。

| ファイル | 所有者 | 役割 |
|---|---|---|
| `TASK.md` | main Agentまたは起票者 | Task契約 |
| `PLAN.md` | Planner Agent | 設計と実装計画 |
| `REVIEW_RESULT.md` | レビュアー Agent | 独立レビュー結果 |
| `QA_PLAN.md` | QA Agent | 実装前の受け入れ試験計画と改訂履歴 |
| `QA_RESULT.md` | QA Agent | マージ後の実施結果とFAIL分類 |
| `HANDOVER.md` | DEV、QA、main Agent | 成果、運用上の注意、Wiki引き渡し |

## バックログ

`backlog.yaml`は索引と状態の正本であり、Task本文を重複させない。Taskは次の最小情報を持つ。

```yaml
- id: TASK-0001
  title: 開発プロセスを整備する
  type: feature
  epic: EPIC-001
  status: plan
  priority: P1
  estimate_points: 3
  task_dir: tasks/TASK-0001-development-process-foundation
  depends_on: []
  assignees:
    main: main-agent
    planner: planner-agent
    dev: dev-agent
    reviewer: reviewer-agent
    qa: qa-agent
  branch: task/TASK-0001-development-process-foundation
  worktree: worktrees/TASK-0001-development-process-foundation
```

バグは`type: bug`と`origin_task`を持ち、通常Taskと同じフェーズを通る。

## 見積もり

PLANに列挙した変更予定から機械的に算出し、PLAN承認時に固定する。

```text
file_score = ceil(変更予定ファイル数 / 3)
line_score = ceil(変更予定行数 / 200)
raw_points = max(1, file_score, line_score)
estimate_points = 1, 2, 3, 5, 8, 13のうちraw_points以上の最小値
```

対象は実装コード、Schema、設定ファイルである。テスト、フィクスチャ、スナップショット、文書、生成物、ロックファイル、vendorは除外する。実績は別に記録し、見積もりを上書きしない。スコープ変更による再見積もりにはmain Agentの承認を要する。

## Epicとロードマップ

ロードマップはTask単位ではなくEpic単位で表示する。Epic進捗は完了Taskの見積もりポイント合計を全Taskのポイント合計で割る。進行中Taskへ部分点を与えず、フェーズごとのTask件数を併記する。
