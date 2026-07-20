# Task管理

## Task契約

`TASK.md`は次の正本である。

- `planning input packet`（目的、対象外、AC-ID付き受け入れ条件、安定した参照、依存状態、許可パス、`preflight`結果、未決事項）
- 背景と対象範囲
- 検討すべき設計観点
- 完成の定義
- 関連するWikiテーマと判断

`planning input packet`はMain Agentが所有し、PlannerとQAへ同一内容を渡す唯一の計画入力である。受け入れ条件は一意で安定したAC-IDを持つ観測可能な結果として一度だけ書き、PLANやQA_PLANへ本文を複製しない。依存が未`ready`なら状態と安定参照、`ready`後に固定する値を明示し、`dependency-ready reconciliation`で`ready`参照と差分、再承認要否を追記する。

## 証跡ファイル

Taskごとに次の6ファイルを持つ。

| ファイル | 所有者 | 役割 |
|---|---|---|
| `TASK.md` | main Agentまたは起票者 | Task契約 |
| `PLAN.md` | Planner Agent | AC-IDに対応する設計判断、変更パス、順序、失敗時の扱い、見積り |
| `REVIEW_RESULT.md` | レビュアー Agent | 同一案の独立レビュー結果（対象コミット/tree付き） |
| `QA_PLAN.md` | QA Agent | AC-IDに対応する独立した観測計画と改訂履歴 |
| `QA_RESULT.md` | QA Agent（ケース結果）、main Agent（carry-forward/merge判断） | 案単位のケース別モード実施結果、未実施/blocked理由、FAIL分類、Main判断 |
| `HANDOVER.md` | DEV、QA、main Agent | candidate-bound DEV証跡、成果、carry-forward/merge確認、運用上の注意、Wiki引き渡し |

### candidate-bound証跡

DEVは`candidate_commit`（評価対象コミット）と`candidate_tree`（そのtree）を固定し、各ケースのケース ID、コマンド/テスト、環境またはフィクスチャ、cache条件、exit、成果物 ダイジェスト、未実施理由を`HANDOVER.md`へ記録する。QAはこの記録を独立監査し、テストのネガティブ検出能力と弱体化の有無、コミット/tree 割り当て、ダイジェスト整合を確認する。既存Taskの証跡へ新規項目を遡及して要求しない。

案単位のQA結果と、REVIEW修正後にMainが結果を引き継ぐ場合の[QAガイドライン](qa.md)の閉じた`CF-1`から`CF-7`は、`QA_RESULT.md`へ記録する。影響QAケース集合が空でなければ該当ケースを再実行し、影響を限定できなければ全面再実行とする。禁止条件が一つでも真または不明なら引き継がない。

Mainは統合後に`merge_tree`を承認案 treeと比較し、環境依存ケースだけを限定確認する。この後続判断も同じTaskの証跡へ記録する。

## バックログ

`backlog.yaml`は索引と状態の正本であり、Task本文を重複させない。Taskは次の最小情報を持つ。

`change_class`は`product | safety_contract`のどちらかを指定する。フィールドがない既存Taskは`product`として扱い、未知値はDone経路を選ばずfail-closedする。`safety_contract`への分類変更はTask、PLAN、QA_PLANの再承認とMainの承認者・時刻を必要とする。

```yaml
- id: TASK-0001
  title: 開発プロセスを整備する
  type: feature
  change_class: product
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

安全契約変更では、`TASK.md`に製品コード、テスト、ランタイム・ビルド設定、Schema、製品依存、生成製品入力/成果物、外部観測可能な挙動を変更しないと明記する。PLANとQA PLANへ`change_class: safety_contract`を記録し、担当レビュアーの計画レビュー、空でない分類承認理由、承認時刻の整合を要求する。

新しい完了契約を使うPLANは`safety_contract_version: 2`を明記し、`safety_contract_planned_paths`と`safety_contract_generated_paths`を配列で宣言する。通常予定パスは既存の安全契約許可パス一覧、生成パスは`docs/99-glossary-index.md`だけを許可し、空パス、絶対パス、`..`、ディレクトリ、glob、配列内・配列間の重複を拒否する。`make task-preflight TASK=TASK-NNNN`はGit履歴を参照せず、この契約をDEV前に検査する。バージョンなしでv2フィールドを使う場合と未知バージョンは拒否し、バージョンもv2フィールドもない既存PLANはlegacy検査を維持する。

完了にはHANDOVERの`process_tests`、`contract_scope`、`docs_lint`、`make_check`のPASS、案/merge treeと正規化したSHA-256、実際のno-ff merge差分の許可パス一覧との照合を必要とする。v2では候補差分を二つの宣言配列の和集合に限定し、宣言した全生成パスが候補差分に存在することを要求する。通常予定パスは差分になくてもよい。名前変更またはコピーは起点と宛先が許可パスでも拒否する。製品用のREVIEW/QA PASS、製品用の完了HANDOVER、Wiki取込記録は要求しない。

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
