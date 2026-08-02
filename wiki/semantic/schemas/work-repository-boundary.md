---
kind: schema
title: Work Repository Boundary
---

# Work Repository Boundary

## 登場主体

- 製品リポジトリ: コード、テスト、製品文書、開発ガイドラインを所有する。
- 運用リポジトリ: バックログ、Task証跡、開発用Wiki、Decisionを所有する。
- 製品worktree: Taskブランチの作業場所であり、運用リポジトリのGit管理対象ではない。

## 関係

運用リポジトリの`project.yaml`が製品リポジトリへの相対パス、既定ブランチ、worktree配置、マージ方式を定義する。製品リポジトリの再利用可能なスクリプトが運用リポジトリを検査する。

## 構造上の制約

- 製品変更はTaskブランチとworktreeで行う。
- 運用リポジトリは`main`一本で運用する。
- 運用リポジトリへの公開はMainが共通lockで直列化する。
- Wiki AgentはMain指定のWiki本文だけを編集する。標準起動は`agents.spawn_agent(agent_type="wiki")`であり、独立`codex exec` launcherは使わない。
- Mainが許可path、差分scope、索引生成が必要な場合の実行、検証、stage、commit、pushを所有する。Wiki Agentはlock、検証、Git操作又は`.git`書込みを行わない。
- Wiki receiptは明示的ingest時だけの任意成果物であり、Wiki依頼がないTaskの完了条件ではない。

## 関連

- [Repository and Wiki ownership](../../decisions/DECISION-0002-work-repository-and-wiki-ownership.md)
