---
task_id: "TASK-0011"
title: "Work Agent Responses APIランタイムを実装する"
status: ready
created_at: "2026-07-15"
---

# TASK-0011 Work Agent Responses APIランタイムを実装する

## 目的

Work AgentのResponses APIステップを、Task単位の排他制御、永続化、再開可能なコンテキスト再構築付きで実行する。

## 背景

現行Go coreはPlaneの起動雛形だけであり、Responses API連鎖をTaskの正本にせず再起動後に復旧するランタイムがない。TASK-0007、0008、0010で確立するTask/Workspace/Agent基盤の上に置く。

## スコープ

### 対象

- Responses APIの通常・進捗メンテナンスステップ、出力項目、tool dispatch intent/result、使用量をcontrol.dbへ記録する。
- `previous_response_id`を同一Runの最適化として扱い、continuation/resume cursorから新Runを安全に再構築する。
- Task lock、call/response/output item/operation keyの冪等化、正本からのContext View再構築を実装する。

### 対象外

- terminal、非同期操作、Mailbox、delegate、完了レビューのドメイン実装（TASK-0012〜0015）。
- OpenAI background response、コンテキスト圧縮の独自要約、Memory/Governanceサービス実装、CLI UX。

## 受け入れ条件

- [ ] 同一Taskへの並行stepは一つだけで、競合要求は二重のResponses呼出しやTask作用を残さない。
- [ ] completed outputとdispatch intentを先にコミットし、再実行で同じ`call_id`/操作キーを重複適用しない。
- [ ] Runを停止後も、Task契約・状態・進捗・Workspace・未解決参照から新Runを開始でき、旧`previous_response_id`を再開境界を越えて使わない。
- [ ] API失敗、永続化失敗、未確定tool結果を観測可能な失敗/再試行可能状態にし、Task正本を部分更新しない。

## 検討すべき設計観点

- Responses APIはpolicy stepであり、Task/Run/continuation IDと分離する。
- 機密・推論本文を正本化せず、参照、ダイジェスト、秘匿済み証跡を保存する。
- SQLite transactionと外部API呼出しを分離し、再開カーソルのウォーターマークを検証する。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] 必要な実装、テスト、文書が完成している。
- [ ] 独立レビューと`make check`がPASSしている。
- [ ] マージ後QAが完了している。
- [ ] HANDOVERがWiki Agentに取り込みされている。

## 関連コンテキスト

### 意味 Wiki

- docs/01-domain-model.md（AgentRun）、docs/03-agent-lifecycle.md、docs/05-runtime-and-responses-api.md、docs/11-data-model.md

### 判断

- Responses APIは一推論stepだけを担い、HarnessがTask・継続・永続状態を所有する。
- 同一Run内だけ`previous_response_id`を使い、論理再開は正本再読込で行う。

### 適用しなかった重要な判断

- API conversation/historyまたはin-memory goroutineを再開の正本にはしない。
