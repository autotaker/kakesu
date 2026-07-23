---
task_id: "TASK-0001"
status: approved
qa_agent: "main-agent-bootstrap"
approved_by: "main-agent"
approved_at: "2026-07-14"
revision: 1
implementation_reviewed_at: "2026-07-14T08:22:10+10:00"
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0001 QA PLAN

## 方針

bootstrap Taskのため独立QA Agentの代わりに、利用者との合意を期待値の正本とし、実装後にCLI、生成物、Git境界、文書間整合をmain Agentが再確認する。

## 前提と環境

- 製品リポジトリ: `../agent-harness`
- 運用リポジトリ: `../agent-harness-work`
- Node.js 20以上、pnpm、Go、uv、Cargoを使用できる。
- 運用リポジトリは`main`でremote未設定である。

## 受け入れ条件との対応

| ケースID | 受け入れ条件 | 操作 | 期待結果 | 証跡 |
|---|---|---|---|---|
| QA-001 | 正本文書 | 文書一覧とリンクを検査 | 必須領域を欠かさず、既存専門規約へ接続する | `docs/development/` |
| QA-002 | 証跡生成 | bootstrap Taskを生成 | 6ファイルとbacklog entryが作られる | Task directory |
| QA-003 | フェーズゲート | `make task-check` | 承認情報と状態に応じたゲートを検査する | command output |
| QA-004 | work全体 | `make work-check` | Epic、Task、依存、Wiki索引、receiptがPASSする | command output |
| QA-005 | viewer | `make backlog-view` | 単一HTMLにEpic進捗と7状態が表示される | `viewer/index.html` |
| QA-006 | Wiki context | context commandをdry-run | `codex exec`がwork repoをcwdにして対象証跡を扱う | JSON output |
| QA-007 | Wiki ingest | ingest commandをdry-run | 許可範囲、digest、検査、直接commitを指示する | JSON output |
| QA-008 | 回帰 | `make check` | 既存build、test、lintを壊さない | command output |
| QA-009 | Git境界 | 両Git statusとignored filesを確認 | product worktreeとviewerをwork repoへcommitしない | `git status` |

## 境界・異常・回帰

- 不正Task ID、重複ID、未知Epic、循環依存を拒否する。
- blocked以外の`resume_status`とblocked時の欠落を拒否する。
- 13ポイント超過を自動承認しない。
- dirtyなwork repoまたはmain以外でWiki Agentを開始しない。
- 同じHANDOVER digestのingestを再適用しない規約とreceiptを確認する。

## 実施不能時の扱い

- sandbox外の運用リポジトリ操作は明示承認で実施する。
- Codex Agentの実呼び出しはbootstrap HANDOVER完成後のingestで確認する。事前確認はdry-runとする。

## 実装後の再確認

- [x] 実装差分とレビュー結果を確認した。
- [x] 操作手順を現行実装に合わせた。
- [x] 期待結果または試験範囲の変更有無を確認した。
- [x] 期待結果または範囲は変更していない。

## 改訂履歴

| revision | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | 2026-07-14 | main-agent-bootstrap | 対話合意に基づく初版 | approved |
