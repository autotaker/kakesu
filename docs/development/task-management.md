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
| `REVIEW_RESULT.md` | レビュアー Agent | 同一案の独立レビュー結果（対象コミット/tree付き） |
| `QA_PLAN.md` | QA Agent | 実装前の受け入れ試験計画と改訂履歴 |
| `QA_RESULT.md` | QA Agent（ケース結果）、main Agent（carry-forward/merge判断） | 案単位のケース別モード実施結果、未実施/blocked理由、FAIL分類、Main判断 |
| `HANDOVER.md` | DEV、QA、main Agent | candidate-bound DEV証跡、成果、carry-forward/merge確認、運用上の注意、Wiki引き渡し |

### candidate-bound証跡

DEVは`candidate_commit`（評価対象コミット）と`candidate_tree`（そのtree）を固定し、各ケースのケース ID、コマンド/テスト、環境またはフィクスチャ、cache条件、exit、成果物 ダイジェスト、未実施理由を`HANDOVER.md`へ記録する。QAはこの記録を独立監査し、テストのネガティブ検出能力と弱体化の有無、コミット/tree 割り当て、ダイジェスト整合を確認する。既存Taskの証跡へ新規項目を遡及して要求しない。

REVIEW修正後にMainが結果を引き継ぐ場合は、非挙動かつ明示した低リスク条件を全て証明し、`qa_carry_forward`として旧新コミット/tree、diff、影響ケース、再実行証拠、理由を記録する。QA FAIL、受け入れ条件/QA_PLAN変更、認証認可・秘密・sudo/PAM・IPC/Schema/設定/依存・並行性/ライフサイクル/persistence/エラー/fail-closed、テスト削除/弱体化、影響不明、案/tree不一致では引き継がずrerunする。マージ後は`merge_tree`を承認案 treeと比較し、環境依存ケースだけを限定確認する。

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
