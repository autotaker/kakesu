---
task_id: "TASK-0005"
title: "main AgentのTask実行効率化スキルを整備する"
status: complete
created_at: "2026-07-15"
---

# TASK-0005 main AgentのTask実行効率化スキルを整備する

## 目的

main AgentがTaskのゲートと責務分離を維持したまま、不要なAgent往復、長大なtool出力、重複検証、既知の権限失敗を減らすためのリポジトリ内スキルを整備する。TASK-0003実行セッションの定量的な振り返りを根拠資料として残し、次回以降のmain AgentがTask開始時に再利用できる状態にする。

## 背景

TASK-0003のMultiAgentV2検証と内部`agents.spawn_agent`への標準経路移行は、PLAN / DEV / REVIEW / QAの必須ゲートを通して完了した。一方、対象セッションは43分30秒を要し、約2,152万token相当の累積入力、31回の`wait_agent`、約25万文字のtool出力、複数回の完全`make check`が発生した。その97.6%はキャッシュ済み入力であり、ゲート自体よりも、各Agentの完了条件不足、全文出力、60秒固定ポーリング、後発の文書lint発見が往復を増やしていた。

OpenAI Codex issue #32705は、MultiAgentV2で`task_name`とrole選択が別物であること、異種role起動に`agent_type`と`fork_turns="none"`が必要であること、実効モデル・effort・permissionの観測が必要であることを背景とする。スキルはこの起動契約を維持し、CLIランチャーへの安易な切替を効率化手段にしない。

## スコープ

### 対象

- `.agents/skills/run-efficient-task-delivery/`に配置するmain Agent向けのリポジトリ内スキル
- Task開始時の事前読取り、Agentごとの完了契約、出力上限、検証順序、権限見積もり、振り返りの手順
- TASK-0003セッションの計測値、主要な時間・token要因、同じゲートを維持した改善目標を記録する参照資料
- `skill-creator`の必須構造、命名、metadata、validation契約に適合する`SKILL.md`と`agents/openai.yaml`
- スキルのトリガーと実行手順を独立Agentが再現できることのforward test

### 対象外

- PLAN / DEV / REVIEW / QAゲート、role分離、親所有commit、mainだけのmergeの緩和
- `.codex`のrole定義、model / effort、sandbox、permission profileの変更
- `agents.spawn_agent`、`make work-agent`、Task Schema、ランチャー実装の変更
- 全セッション履歴を自動集計する新規ツールや外部telemetryの導入
- 回顧値を受け入れゲートやSLOとして強制する変更
- グローバルCodexスキルやユーザー設定の変更

## 受け入れ条件

- [x] `.agents/skills/run-efficient-task-delivery/SKILL.md`がTask開始・継続・遅延振り返りをトリガーとし、main Agentの実行手順を命令形で簡潔に定義する。
- [x] スキルは、適用される`AGENTS.md`と検証順序の事前読取り、スコープ固定、一体ずつの内部Spawn Agent、roleごとの完了条件、出力抑制、局所検証から一回の完全検証への順序、既知の権限・依存関係の扱いを含む。
- [x] スキルは、`agents.spawn_agent`の`task_name`と`agent_type`を区別し、roleを越える起動で`fork_turns="none"`を明示し、model / effort不一致で停止する現行契約を効率化のために緩和しない。
- [x] 参照資料が43分30秒、約2,152万token累積入力、キャッシュ率97.6%、`wait_agent` 31回 / 22分16秒、tool出力約25万文字という観測値、主要因、43分から22–28分への改善仮説を出典と限界付きで記録する。
- [x] `agents/openai.yaml`は必要最小限のinterface metadataだけを持ち、`default_prompt`が`$run-efficient-task-delivery`を明示する。
- [x] `skill-creator`の`quick_validate.py`、`git diff --check`、リポジトリの`make check`がPASSする。
- [x] 独立Agentによるforward testで、スキル単体からゲートを維持した効率化手順を引き出せ、処理省略やCLI fallbackの通常化を推奨しない。

## 検討すべき設計観点

- `SKILL.md`を次回の実行手順に限定し、過去セッション固有の計測値と根拠を`references/`へ分離する段階的開示
- ゲート省略ではなく、各Agentへの一回の明確な完了契約、局所検証、出力制限で往復を減らす境界
- tokenをモデル課金と同一視せず、キャッシュ済み入力を含むハーネス上の累積計数として限界を明記すること
- 将来のTask振り返りで同じ指標を追記・比較する手順と、目標値を強制的な合否基準にしない扱い
- MultiAgentV2の現行契約とissue #32705の制約を参照しつつ、スキルを一時的なランタイム回避策に固定しないこと
- main Agentの最終判断、親所有commit、ロック、スコープ検査をAgentに誤って委譲しないプロンプト設計

## 完成の定義

- [x] 受け入れ条件を満たしている。
- [x] 必要な実装、テスト、文書が完成している。
- [x] 独立レビューと`make check`がPASSしている。
- [x] マージ後QAが完了している。
- [x] HANDOVERがWiki Agentに取り込みされている。

## 関連コンテキスト

### 意味 Wiki

- [Task Delivery](../../wiki/semantic/scripts/task-delivery.md): PLANからQAまでのゲート、親所有commit、mainだけのmergeを維持する。
- [Codex Agent Model Routing](../../wiki/semantic/schemas/codex-agent-model-routing.md): 内部Spawn Agentのrole、model / effort、Explorer境界を現行契約として扱う。
- [Work Repository Boundary](../../wiki/semantic/schemas/work-repository-boundary.md): Task証跡と製品スキルの所有境界、共通ロック、親所有commitを維持する。

### 判断

- [DECISION-0001 PLAN DEV QA Process](../../wiki/decisions/DECISION-0001-plan-dev-qa-process.md): 効率化でゲートと独立roleを省略しない。
- [DECISION-0002 Work Repository and Wiki Ownership](../../wiki/decisions/DECISION-0002-work-repository-and-wiki-ownership.md): スキルは製品リポジトリ、Task証跡は運用リポジトリに置く。
- [DECISION-0003 Codex Agent Model Routing](../../wiki/decisions/DECISION-0003-codex-agent-model-routing.md): ローカルrole契約と内部Spawn Agentを標準経路とする現行判断を維持する。

### 適用しなかった重要な判断

- なし
