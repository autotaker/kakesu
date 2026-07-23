---
task_id: "TASK-0010"
title: "ルートTask CLIとOwner・Workspace割当を実装する"
status: ready
created_at: "2026-07-15"
---

# TASK-0010 ルートTask CLIとOwner・Workspace割当を実装する

## 目的

ユーザーがroot TaskをCLIから提出でき、Control Planeがowner AgentとLinux P0 workspaceを原子的に割り当て、実行可能な初期状態を返すようにする。

## 背景

root Taskの責任者は人間であり、HarnessがTask/owner/workspaceを確定する。CLIがTask recordを直接書く、またはAgentを先にbusyにする方式は、失敗時に孤立ownerやworkspaceを残す。依存するTask-0007のControl transaction、TASK-0009のruntime基盤、TASK-0022のfail-closed FS強制を組み合わせる必要がある。

## スコープ

### 対象

- `kakesu task create`（または同等のroot Taskサブコマンド）の入力・出力・exit code。
- root Taskのobjective/acceptance/instructions検証、owner profile選択、idle Agent reservation、empty workspace準備、Control transactionによるcommit/compensation。
- Task ID/workspace ID/owner ID/statusを機械可読JSONと人間可読表示で返す。
- 成功、validation failure、owner不足、workspace失敗、競合割当を対象とするCLI統合試験。

### 対象外

- 子Task delegate、Task実行/Responses API、completion、cancel、workspace fork/shared_readonly。
- owner Agentの自動provision、非Linux P0 profile、外部API/daemon管理UI。
- TASK-0022のLandlock/LSM・mount boundary実装自体（本Taskは安全なworkspace開始をそのadmissionに依存する）。

## 受け入れ条件

- [ ] CLIは空でないobjectiveとacceptance、任意instructions、owner profileを受け、無効入力ではDBやworkspaceを変更せず非0終了する。
- [ ] 成功時は1 transaction境界でTask contract、TaskEvent、単一owner割当、workspace参照を確定し、JSON出力に全IDと初期statusを含める。
- [ ] owner reservationまたはworkspace準備が失敗すると、Taskは可視化されず、Agentはidleへ戻り、作成済み物理resourceは冪等に回収される。
- [ ] 同じidle Agentへ競合するcreate要求は一つだけ成功し、成功Taskは`ready`から実行開始可能である。
- [ ] root Taskは`parent_task_id = null`で、Workspaceは`empty`、一時grantを継承しない。

## 検討すべき設計観点

- CLI adapterとControl application serviceの責務分離、prepare/commit/compensateの順序、operatorに返す失敗分類。
- ownerの排他予約とworkspace作成の相関ID、再試行のidempotency。
- root Taskが人間の責任下にあることとAgent ownerの実行責任を混同しない。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] 必要な実装、テスト、文書が完成している。
- [ ] 独立レビューと`make check`がPASSしている。
- [ ] マージ後QAが完了している。
- [ ] HANDOVERがWiki Agentに取り込みされている。

## 関連コンテキスト

### 意味 Wiki

- `docs/00-kakesu.md` §§2.4, 6, 8: Control Planeの人間境界とHarnessによる確定。
- `docs/01-domain-model.md` §§1–3: single ownerとWorkspace。
- `docs/06-tools-and-async.md`: Task command入力の目的/受入条件/owner profile/workspace mode。
- `docs/02-task-lifecycle.md`, `docs/03-agent-lifecycle.md`: 初期状態、owner lifecycle。
- 依存: TASK-0007、TASK-0009、TASK-0022。root Task作成はTASK-0022が安全なFS profileをadmitできる場合だけworkspaceを開始する。

### 判断

- Decision: CLIはapplication serviceを呼ぶthin adapterとし、Task/owner/workspaceを別々に永続化しない。物理workspaceはprepare後、Control commit失敗時に補償cleanupする。

### 適用しなかった重要な判断

- CLIからSQLiteを直接更新する案は、owner/workspace整合性と将来のAPI境界を損なうため不採用。
