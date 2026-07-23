---
task_id: "TASK-0002"
title: "Codex Agentのモデルルーティングを導入する"
status: complete
created_at: "2026-07-14"
---

# TASK-0002 Codex Agentのモデルルーティングを導入する

## 目的

Codex Agentを役割とTaskリスクに応じた明示的なモデル・reasoning effortで起動し、品質を維持しながらトークン消費を抑える。各起動の割当、権限境界、作業場所、成果コミットを検証可能にし、最初の通常Taskとして既存のPLAN / DEV / QAプロセスを実運用できる状態にする。

## 背景

TASK-0001で製品リポジトリと運用リポジトリを分離し、共通ロック、役割分離、PLAN / DEV / QAゲート、独立レビュー、main Agentだけによる`--no-ff`マージを導入した。一方、現行ランチャーは任意のprofileまたはmodelをCodex CLIへ渡せるだけで、役割ごとのmodelとreasoning effort、DEVの選定根拠、子Agentの権限、起動結果を一貫して検査できない。

高能力モデルを全調査へ一律に使うとトークンを浪費するが、低コストモデルへの曖昧な振り分けは契約、Schema、セキュリティ、並行処理、migrationを伴う変更の品質を損なう。役割固定の割当と保守的なDEVルーティングをプロジェクト設定とランチャーで明示し、必要なリポジトリ調査だけを限定的なread-only explorerへ委譲できるようにする必要がある。

## スコープ

### 対象

- main、PLAN、DEV、QA、REVIEW Agentのmodelとreasoning effortの割当
- Taskリスクに基づくDEV profileの決定、PLAN承認時の記録、LunaからSolへの昇格記録
- 各役割から利用できるread-only explorerと、rootから2階層までの委譲制御
- `agent-harness`と`agent-harness-work`の両方を起点に有効なproject-scoped Codex設定
- 役割別ランチャー、既存ランチャーとの互換方針、入力検証、起動証跡
- model、reasoning effort、深さ、権限、起動結果を検証する自動テストと必要な開発文書

### 対象外

- Agentが扱う製品機能、ドメインモデル、API、Schemaの機能変更
- 運用リポジトリと製品リポジトリの正本境界、Gitブランチ戦略、フェーズ状態モデルの再設計
- PLAN / DEV / QAゲート、役割分離、独立レビュー、マージ権限の緩和
- グローバルなCodex設定や、文書化されていないmodel aliasへの依存
- explorerによる編集、commit、実装方針の決定、無制限な調査または再委譲
- model自体の品質評価、学習、価格最適化

## 受け入れ条件

- [x] main Agentは`gpt-5.6-sol`、reasoning effort `high`で起動する。
- [x] PLAN、QA、REVIEW Agentは役割ごとに固定され、いずれも`gpt-5.6-terra`、reasoning effort `medium`で起動する。
- [x] DEV AgentはTaskごとに`gpt-5.6-luna` / `xhigh`または`gpt-5.6-sol` / `high`のどちらか一方を使用し、他の値では起動できない。
- [x] DEV profileはPLAN承認時に明示され、適用した決定基準と選定理由を人が読めて機械検査可能な形で記録する。LunaからSolへ昇格した場合は、変更前後のprofileと昇格理由を履歴として残す。
- [x] Luna / `xhigh`は、作業が明確、局所的、機械的に検証可能であり、かつ高リスクな契約、セキュリティ、並行処理、migrationへの影響がない場合に限って選択される。曖昧、横断的、高リスク、または契約、Schema、セキュリティ、並行処理、migrationへ影響する場合はSol / `high`を選択し、判定に迷う場合もSol / `high`へ倒す。
- [x] main、PLAN、DEV、QA、REVIEWの各Agentは、1回につき一つの限定されたリポジトリ調査質問を`gpt-5.6-luna`、reasoning effort `medium`のexplorerへ委譲できる。
- [x] explorerはread-onlyで、簡潔な根拠要約とファイル参照だけを返し、編集、commit、scope拡大、別Agentのspawnを行えない。
- [x] Agentの最大深さはroot -> role Agent -> explorerを許可し、explorerからのfan-outを禁止する。並行thread数と委譲指示は、必要のないfan-outとraw log転送を抑えながら根拠品質を保つよう制限される。
- [x] project-scoped Codex設定が、`agent-harness`を起点とするsessionと`agent-harness-work`を起点とするsessionの両方で同じ役割定義を解決し、文書化されていないmodel aliasや利用者のグローバル設定に依存しない。
- [x] 自動テストがmain、PLAN、QA、REVIEW、explorerおよびDEVの両profileについて、選択されるmodelとreasoning effortを証明する。
- [x] 自動テストが最大深さ、explorerのread-only設定と再委譲禁止、無効なDEV profileの起動前FAILを証明する。
- [x] 既存の明示的なprofile / model上書きをどう扱うか、および既存ランチャーの後方互換性をPLANで決定し、役割固定の割当を破らず、その決定どおりの正常系と異常系を自動テストで証明する。
- [x] Agent起動ごとに、割り当てたmodel、reasoning effort、working directory、許可されたwrite scope、実行後のcommitを、機密情報や大量のraw logを残さず検査できる。
- [x] 製品リポジトリと運用リポジトリの境界、運用リポジトリの共通ロック、役割分離、PLAN / DEV / QAゲート、独立レビュー、main Agentだけによる`--no-ff`マージが維持され、既存の検査と`make check`がPASSする。

## 検討すべき設計観点

- 役割固定のmodel / effortをproject-scoped設定、役割別Agent定義、ランチャーのどこで正本化し、設定の重複やdriftを検出するか
- `agent-harness`と`agent-harness-work`が兄弟リポジトリであることを踏まえ、どちらのrootからも同一の設定と役割ファイルを安全に解決する方法
- main、PLAN、QA、REVIEWの固定割当と、既存の`PROFILE` / `MODEL`明示上書きおよびランチャー互換性との優先順位。非交渉条件である固定割当を維持した互換方針をPLANで明文化すること
- DEVの選定値、決定基準、理由、承認者、昇格履歴をどの証跡へ記録し、PLAN承認ゲートとランチャーがどう照合するか
- Luna選択は全ての低リスク条件を満たす場合に限定し、Sol選択は一つでも高リスク条件に該当する場合または不明な場合に適用する、保守的で決定的な評価順序
- explorerへ渡す質問を一つのbounded questionに限定し、回答を要約とファイル参照へ絞るprompt契約
- 最大深さをroot = 0、role Agent = 1、explorer = 2として扱い、role Agentからのexplorer起動を許可しつつ、root直下のexplorerを含めexplorer自身の再委譲を禁止する方法
- read-only sandboxだけでなく、explorerが編集、commit、scope拡大、Agent spawnを行わないことを設定、指示、テストで多層的に保証する方法
- token conservationを主要目標として、既定では委譲せず、必要な調査だけを小さいthread数、短い入力、簡潔な出力で行うこと。監査性のために会話全文やraw logを保存しないこと
- dry-runと実行後検査の双方でmodel、effort、cwd、write scope、commitを構造化して観測し、起動前FAILやcommitなしも判別できる証跡設計
- CLI引数、project-scoped設定、Agent role設定の優先順位と、無効値、role不一致、未承認DEV profileをAgent起動前に拒否するfail-closed設計
- launcher、設定解決、権限制約の単体テストに加え、最初の通常Taskのロック取得からcommit後検査までを過度な外部依存なく証明する統合テスト

## 完成の定義

- [x] 受け入れ条件を満たしている。
- [x] 必要な実装、テスト、文書が完成している。
- [x] 独立レビューと`make check`がPASSしている。
- [x] マージ後QAが完了している。
- [x] HANDOVERがWiki Agentに取り込みされている。

## 関連コンテキスト

### 意味 Wiki

- [Development Task](../../wiki/semantic/concepts/development-task.md): 本Taskを、製品のドメインTaskとは別の、目的・受け入れ条件・証跡を持つ運用上の開発単位として扱う。
- [Task Delivery](../../wiki/semantic/scripts/task-delivery.md): PLAN承認、実装前QA計画、独立レビュー、mainへの`--no-ff`マージ、マージ後QAという既存の進行順を維持する。
- [Work Repository Boundary](../../wiki/semantic/schemas/work-repository-boundary.md): 両リポジトリから解決するproject-scoped設定、書き込みロック、Agentの所有範囲を設計・検証する際の境界とする。
- [QA FAIL Attribution](../../wiki/semantic/case-patterns/qa-fail-attribution.md): ルーティングのテストまたはマージ後QAがFAILした場合に、DEVへ自動帰責せず、設定・QA計画・要件・環境を分類してmain Agentが差し戻し先を決める。
- [Wiki Ingestion](../../wiki/semantic/scripts/wiki-ingestion.md): 完了後のHANDOVERをdigest付きでWikiへ取り込み、再利用可能な知識だけをSemantic WikiまたはDecisionへ反映する終了手順として適用する。

### 判断

- [DECISION-0001 PLAN DEV QA Process](../../wiki/decisions/DECISION-0001-plan-dev-qa-process.md): 役割別model routingは既存のPLAN / DEV / QA責務分離、独立Reviewer、main Agentのみのマージ判断を変更せず、その起動方法を具体化する。
- [DECISION-0002 Work Repository and Wiki Ownership](../../wiki/decisions/DECISION-0002-work-repository-and-wiki-ownership.md): `agent-harness-work`を運用証跡の正本として維持し、設定解決・起動証跡・Wiki取り込みでも製品リポジトリとの境界とWiki Agentの所有権を守る。

### 適用しなかった重要な判断

- なし。ローカルWikiにあるaccepted DecisionはDECISION-0001とDECISION-0002のみで、いずれも本Taskと競合しない。前者の役割分離とゲートは固定割当・explorer権限制御で維持し、後者の正本境界とWiki所有権はproject-scoped設定と起動証跡の対象範囲を制約する。したがって、既存Decisionを置換または不適用とせず、model routingと限定的なread-only調査委譲をその内側に追加する。
