---
task_id: "TASK-0014"
title: "子Task委譲と直接キャンセルを実装する"
status: ready
created_at: "2026-07-15"
---

# TASK-0014 子Task委譲と直接キャンセルを実装する

## 目的

親Work Agentが`delegate`で独立owner/workspaceの子Taskを作り、完了結果を親Mailboxへ届け、直接子だけを安全にcancelできるようにする。

## 背景

Taskは単一ownerの責任単位であり、子への責任移転は通常toolや会話では表せない。TASK-0013のMailbox/再開を配送基盤とする。

## スコープ

### 対象

- `delegate`提案検証、child Task/owner/`empty` workspaceの原子的作成、sync timeoutからasyncへの昇格。
- ChildCompleted/Cancelledの親Mailbox配送、直接childへの`cancel_child_task`と既定cascade cleanup。
### 対象外

- `ask_parent`、`reply_to_child`、`escalate`、authority判断、child間通信、detach/transfer children政策。
- `fork`と`shared_readonly` workspaceの物理adapter。MVPでは未対応capabilityとして起動前に拒否する。

## 受け入れ条件

- [ ] delegateは空目的/acceptance、循環、重複、depth/予算超過、owner/workspace確保不能を拒否し、部分childを残さない。
- [ ] childは親と別ownerおよび新規`empty` Workspaceを持ち、`fork`/`shared_readonly`指定は部分Taskを残さず`unsupported_capability`で拒否する。
- [ ] child終端は一度だけ親Mailboxに結果を配送し、親はTASK-0013の条件で再開できる。
- [ ] 親ownerは直接childだけをcancelでき、grandchild/sibling要求は拒否し、cascade後にTask状態を戻さない。

## 検討すべき設計観点

- delegateはTask直接書込みでなくHarness command。Task作成transactionとresource cleanupを分離する。
- Workspace policyは子に明示束縛し、一時許可・親の責任者判断を継承しない。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] 必要な実装、テスト、文書が完成している。
- [ ] 独立レビューと`make check`がPASSしている。
- [ ] マージ後QAが完了している。
- [ ] HANDOVERがWiki Agentに取り込みされている。

## 関連コンテキスト

### 意味 Wiki

- docs/01 §2-3、docs/02 §3/§9、docs/06 §4/§11、docs/10 §4/§13、docs/11-data-model.md
### 判断

- 親cancelは責任撤回であり中間状態なく`cancelled`を確定し、cleanupは後続非同期処理とする。
### 適用しなかった重要な判断

- ask/escalateをdelegateの結果配送へ混ぜない。親がgrandchildを直接cancelしない。
