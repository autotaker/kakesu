---
task_id: "TASK-0001"
status: approved
planner_agent: "main-agent-bootstrap"
approved_by: "main-agent"
approved_at: "2026-07-14"
planned_implementation_files: 16
planned_implementation_lines: 1500
estimate_points: 8
revision: 2
---

# TASK-0001 PLAN

## 受け入れ条件の具体化

| 条件 | 観測方法 | 根拠 |
|---|---|---|
| プロセスとガイドライン | Markdown正本とリンクを確認 | `docs/development/`、ルート`AGENTS.md` |
| 運用リポジトリ | Git branch、構造、検査を確認 | `project.yaml`、`backlog.yaml`、`wiki/` |
| 6証跡生成 | bootstrap TaskをCLIで生成 | `scripts/task/create-task.mjs` |
| 最小状態とフェーズゲート | 正例と不正データの検査 | `check-task.mjs`、`validate-work.mjs` |
| Epic表示 | 単一HTMLを生成して内容確認 | `viewer/index.html` |
| Wiki自律保守 | dry-run、権限規約、ingest receiptを確認 | `run-wiki-agent.mjs`、`wiki/AGENTS.md` |

## 関連WikiとDecision

- `Development Task`と`Task Delivery`をプロセス用語の基準にする。
- `DECISION-0001`に従い、PLAN、DEV、QAと独立レビューを採用する。
- `DECISION-0002`に従い、work側は独立Git、main一本、Wiki Agent直接コミットとする。

## 設計

### 選択案

製品リポジトリへ規約、テンプレート、再利用可能なNode.js CLIを置く。隣接する運用リポジトリへ実データ、証跡、Wiki Schema、Semantic Wiki、Decisionを置く。YAMLは既存の`yaml` packageで読み、追加サービスなしで検査と単一HTML生成を行う。

### 代替案と不採用理由

- 製品リポジトリ内のignoredディレクトリ: worktree間で正本位置が曖昧になる。
- 運用リポジトリを非Git管理: DecisionとWikiの変更履歴、Agent commitの監査性を失う。
- Taskごとの細分化状態: 状態更新コストが増えるため、7状態と証跡ゲートに分ける。
- Task単位ロードマップ: 視認性が落ちるため、Epic単位の期間と完了ポイントを使う。
- Wiki本文のmainレビュー: Wiki Agentの責任を曖昧にするため、Schemaと権限だけをmainが所有する。

### 責務と境界

- main AgentはTask状態、PLAN承認、FAIL判断、製品mainへのマージを所有する。
- フェーズAgentは所有証跡だけを運用リポジトリへ直接コミットする。
- Wiki Agentは`wiki/semantic`、`decisions`、`ingestions`、`index.json`だけを変更する。
- product worktreeとviewer生成物は運用リポジトリのGit対象外とする。

### 不変条件

- DEV AgentとReviewer Agent、DEV AgentとQA Agentは別である。
- `backlog.yaml`のTask本文は`TASK.md`と重複しない。
- Decisionは確定後に意味を変更せず、新Decisionが`supersedes`する。
- 同じHANDOVER digestを重複ingestしない。
- QA FAILの分類と差し戻し先を分離する。

### 失敗時・移行・互換性

- workリポジトリがdirty、main以外、ロック取得不能ならCodex実行を開始しない。
- 13ポイントを超えるTaskは分割またはmain Agent例外を要求する。
- 外部運用リポジトリがないCIでも通常の`make check`は実行可能にする。
- bootstrap Taskだけは未整備だった独立Agentゲートを代替レビューとして記録する。

## 変更予定

見積もり対象は実装コード、Schema、設定ファイルだけとする。

| ファイル群 | 種別 | 概算変更行数 | 変更内容 |
|---|---|---:|---|
| `scripts/task/*.mjs` | implementation | 1,250 | 生成、検査、viewer、Agent呼び出し、共通ロック、hook |
| `Makefile`, `package.json` | configuration | 70 | 開発プロセス用コマンドとテスト |
| work側`schemas/*.json` | schema | 180 | backlog、Decision、ingest receipt |

## 見積もり

`file_score = ceil(16 / 3) = 6`、`line_score = ceil(1500 / 200) = 8`であるため、`estimate_points = 8`とする。テストとMarkdown文書、ロックファイルは対象外である。共通ロック、pre-commit、worktree自動化、追加Schemaをレビュー対応で追加したため、main Agentがrevision 2として再承認した。

## 実装手順

1. 上位プロセスとAgent責務を文書化する。
2. 言語別ガイドラインと証跡テンプレートを追加する。
3. Task生成、検査、viewer、Wiki Agent CLIを実装する。
4. 運用リポジトリ、Schema、初期Wiki、Decisionを構築する。
5. bootstrap証跡を完成させ、全検査を実施する。
6. 製品リポジトリと運用リポジトリを個別にコミットする。

## 検証計画

- Node.js単体テストで見積もり規則とtemplate置換を確認する。
- bootstrap Taskに対して`task-check`と`work-check`を実行する。
- backlog viewerを生成し、Epic進捗と7列カンバンを確認する。
- Wiki Agentコマンドをdry-runし、`codex exec`、cwd、対象ファイルを確認する。
- `make check`と`git diff --check`を実行する。

## 未解決事項

- remote設定は将来検討とし、初期状態では設定しない。
- Semantic検索は初期版でローカルWikiをCodex Agentが探索し、埋め込みDBは導入しない。

## main Agentレビュー

- [x] 受け入れ条件が検証可能である。
- [x] 設計観点と代替案を検討している。
- [x] QA計画を作成できる。
- [x] 見積もりが規則どおりである。
- [x] DEV開始を承認した。
