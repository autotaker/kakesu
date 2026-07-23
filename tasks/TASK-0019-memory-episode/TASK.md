---
task_id: "TASK-0019"
title: "Taskエピソード取込と編纂を実装する"
status: ready
created_at: "2026-07-15"
---

# TASK-0019 Taskエピソード取込と編纂を実装する

## 目的

terminal Taskの固定証跡スナップショットを`evidence.db`へ取り込み、lease/retry付きEpisode compilation jobがOpenAI Agents SDKの揮発セッションとread-only evidence toolで構造化Task episodeを一度だけ確定できるようにする。

## 背景

TASK-0006/0008/0015/0018で確定するTask・実行・review・Governance証跡を、Agent会話履歴そのものではなくTask Episodeとして再利用可能にする。Memory PlaneはDBを単独所有し、終端Taskをjob障害で戻してはならない。

## スコープ

### 対象

- `evidence.db` migration、終端watermark/digest snapshot、evidence record/blob/link/text、episode compilation job lease/heartbeat/retry。snapshotには不変な`EgressChallenge`、Grant、`OutboundTransaction` evidence refを含める。
- task-scoped read-only SQLite views/query budget、OpenAI Agents SDK structured `task_episode` output、schema/reference/epistemic validationと一回保存。
- core/governance outboxからの冪等取込、crash後の固定入力からの再編纂。

### 対象外

- Wiki更新/semantic patch、Wiki query/context injection（TASK-0020）、Work AgentからのWiki直接探索。
- Agent SDK response/tool traceの永続化、terminal Task状態の変更、実行中のlive cross-plane query。

## 受け入れ条件

- [ ] terminal eventごとに契約・進捗・events/runs/children/reviews/egress/artifactsをwatermark/digest付きで冪等に`evidence.db`へ固定する。
- [ ] workerはjob lease/heartbeatを使い、期限切れ時はattemptを増やして新しい揮発SDK sessionで同じsnapshotから再開し、部分出力を保存しない。
- [ ] `query_evidence`はtask-scoped SELECT/WITH SELECTだけをbudget内で返し、DML/DDL/PRAGMA/ATTACH/他Task/live Plane参照を拒否する。
- [ ] schema・reference・epistemic stateとGovernance egress evidenceのTask/Workspace binding・不変性を検証したepisodeだけをTaskごとに一件保存し、失敗はterminal Taskを戻さず`needs_operator`へ分類する。

## 検討すべき設計観点

- durable stateはjob/snapshot/episodeだけで、SDK session/response ID/tool historyは揮発。DB取込後にworkspaceを消してもevidence refを壊さない。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] 必要な実装、テスト、文書が完成している。
- [ ] 独立レビューと`make check`がPASSしている。
- [ ] マージ後QAが完了している。
- [ ] HANDOVERがWiki Agentに取り込みされている。

## 関連コンテキスト

### 意味 Wiki

- docs/04 §§6-7、docs/05 §§7,9、docs/08 §§3-5,18-19、docs/11 §§1-2,5、docs/12 §6、docs/13 §§3,5。依存: TASK-0006、0008、0015。
### 判断

- Task Episodeを長期記憶単位とし、snapshot固定・read-only query・lease retryで再現性を確保する。
### 適用しなかった重要な判断

- Wiki更新、live DB横断検索、Agent trace永続化、job失敗時のTask rollbackを採用しない。
