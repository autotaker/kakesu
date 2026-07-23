---
task_id: "TASK-0001"
title: "開発プロセスとガイドラインを整備する"
status: complete
created_at: "2026-07-14"
---

# TASK-0001 開発プロセスとガイドラインを整備する

## 目的

Task単位のGit開発をPLAN、DEV、QAの3フェーズで統制し、Agentの責務、品質ゲート、証跡、バックログ、開発用Wikiを再利用可能な規約とツールとして整備する。

## 背景

製品リポジトリには専門領域ごとのレビュー規約があるが、Task起票からマージ後QA、差し戻し、知識継承までを一貫して管理する上位プロセスがない。Agentによる開発で状態更新を細かくしすぎず、責務分離と検証可能な証跡を残す必要がある。

## スコープ

### 対象

- Task、Epic、バックログ、6証跡のデータモデルとテンプレート
- PLAN、DEV、QA、レビュー、マージ、revert、バグ化のプロセス
- main、Planner、DEV、Reviewer、QA、Wiki Agentの責務
- Go、Python、Rust、JavaScript・TypeScript、Markdown・YAML、JSON Schemaのガイドライン
- Task生成、フェーズゲート検査、Epicロードマップ、カンバン生成
- 独立した`agent-harness-work` Gitリポジトリ
- Codex CLIによるWiki参照とHANDOVER ingest

### 対象外

- GitHubや外部Task管理サービスとの同期
- ベクトルDBまたは埋め込み検索
- 運用リポジトリのトピックブランチ
- 製品機能の変更

## 受け入れ条件

- [x] 開発プロセス、Agent責務、Task管理、Git、レビュー、QAの正本文書がある。
- [x] 対象言語とSchema・文書のコーディングガイドラインがある。
- [x] `agent-harness-work`が隣接する独立Gitリポジトリとして初期化され、`main`一本で運用できる。
- [x] `TASK.md`、`PLAN.md`、`REVIEW_RESULT.md`、`QA_PLAN.md`、`QA_RESULT.md`、`HANDOVER.md`をTaskごとに生成できる。
- [x] バックログは7状態だけを使い、Epic進捗を完了ポイントとフェーズ分布で表示できる。
- [x] 見積もりは実装コード、Schema、設定の予定ファイル数と行数から規則ベースで算出される。
- [x] フェーズゲート、証跡、依存関係、Wiki索引を機械検査できる。
- [x] QA FAILを実装、QA計画、要件、環境、回帰へ分類し、main Agentが差し戻し先を決める規約がある。
- [x] Wiki Agentが許可範囲だけを更新し、HANDOVER digestによる冪等性を保ち、検査後に直接コミットできる。
- [x] `make check`と運用リポジトリ検査がPASSする。

## 検討すべき設計観点

- 製品リポジトリと運用リポジトリの正本境界
- Agent間の独立性とmain Agentの判断権限
- 状態更新トークンを増やさない最小状態モデル
- 実装前QA計画と実装後認識差の扱い
- main一本の運用リポジトリにおける並行書き込みと排他
- Wiki本文の自律保守とSchema所有の分離
- Decisionの不変性と置換履歴
- 生成HTMLの安全性とローカル閲覧

## 完成の定義

- [x] 受け入れ条件を満たしている。
- [x] 必要な実装、テスト、文書が完成している。
- [x] bootstrap代替レビューと`make check`がPASSしている。
- [x] マージ相当のmain上QAが完了している。
- [x] HANDOVERがWiki Agentにingestされている。

## 関連コンテキスト

### Semantic Wiki

- [Development Task](../../wiki/semantic/concepts/development-task.md)
- [Work Repository Boundary](../../wiki/semantic/schemas/work-repository-boundary.md)
- [Task Delivery](../../wiki/semantic/scripts/task-delivery.md)
- [QA FAIL Attribution](../../wiki/semantic/case-patterns/qa-fail-attribution.md)

### Decision

- [DECISION-0001 PLAN DEV QA Process](../../wiki/decisions/DECISION-0001-plan-dev-qa-process.md)
- [DECISION-0002 Work Repository and Wiki Ownership](../../wiki/decisions/DECISION-0002-work-repository-and-wiki-ownership.md)

### 適用しなかった重要なDecision

- なし
