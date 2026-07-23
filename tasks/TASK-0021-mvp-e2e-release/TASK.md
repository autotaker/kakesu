---
task_id: "TASK-0021"
title: "Linux-only MVP E2Eと再起動復旧を完成する"
status: ready
created_at: "2026-07-15"
---

# TASK-0021 Linux-only MVP E2Eと再起動復旧を完成する

## 目的

既存MVP sliceをLinux-only CLIで結合し、Task実行、egress block/grant/manual retry、episode/context、プロセス停止後のoutbox/inbox/lease復旧、install/diagnosticsをリリース可能なE2E証跡として完成する。

## 背景

TASK-0012〜0015、0018、0020の個別契約が揃っても、実プロセス境界・永続状態・Linux P0前提で一貫して復旧できなければMVPを提供できない。これは新domain機能を増やさず横断統合を検証するrelease taskである。

## スコープ

### 対象

- Linux-only CLI install/start/stop/diagnose、既存service wiring/configuration、E2E fixtureとoperational diagnostics。
- root/subtask、async/mailbox、egress block→grant→ACK→manual retry、terminal episode→context injectionの代表経路。
- core/governance/memoryのcrash/restart時のSQLite/outbox/inbox/lease/idempotency復旧と明確なfailure classification。

### 対象外

- 新しいdomain機能、Policy/Wiki/Memoriesの新規意味論、TLS inspection/credential broker/Authority、macOS/Windows対応、性能最適化。
- 個別sliceの設計変更（欠陥は根拠付きで依存Taskへ差し戻す）。

## 受け入れ条件

- [ ] supported Linux hostでCLI install/preflightがP0必須要件を検査し、start/diagnoseがcomponent/version/socket/DB/namespace状態と安全な修復指針を表示する。
- [ ] fixtureがTask/subtask、blocked egress、grantのACK後one-use手動retry、terminal episode、next start/resume context injectionをend-to-endで観測する。
- [ ] control/governance/memoryの各停止点後に再起動して、重複外部作用・grant二重消費・episode二重確定なしにoutbox/inbox/leaseを収束できる。
- [ ] E2EはLinux専用として明示され、失敗はimplementation/requirement/QA-plan/environment/regressionに分類して診断証跡を残す。

## 検討すべき設計観点

- release harnessはdomain policyを再実装せず既存契約を駆動する。実外部ネットワーク/秘密情報へ依存しないlocal fixtureを使用する。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] 必要な実装、テスト、文書が完成している。
- [ ] 独立レビューと`make check`がPASSしている。
- [ ] マージ後QAが完了している。
- [ ] HANDOVERがWiki Agentに取り込みされている。

## 関連コンテキスト

### 意味 Wiki

- docs/07 §2、docs/10 §§9,12-14、docs/11 §5、docs/12 §§5-8、docs/13 §§5,8。依存: TASK-0012、0013、0014、0015、0018、0020。
### 判断

- Linux P0だけをMVP profileとしてE2E化し、release taskは新domain能力を追加しない。
### 適用しなかった重要な判断

- 新しいdomain機能、非Linux profile、実外部サービス依存、slice境界を越える設計変更を採用しない。
