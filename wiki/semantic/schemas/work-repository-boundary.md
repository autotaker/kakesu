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
- 運用リポジトリへの書き込みはロックで直列化する。
- Agentは所有範囲だけを直接コミットする。
- Wiki本文の所有者はWiki Agent、Wiki Schemaの所有者はmain Agentである。

## 関連

- [Repository and Wiki ownership](../../decisions/DECISION-0002-work-repository-and-wiki-ownership.md)
