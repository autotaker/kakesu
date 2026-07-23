---
task_id: "TASK-0016"
title: "Governance永続ポリシーストアとfail-closed判断を実装する"
status: ready
created_at: "2026-07-15"
---

# TASK-0016 Governance永続ポリシーストアとfail-closed判断を実装する

## 目的

Rust Governance Planeが単独所有する`governance.db`へWorkspace単位のポリシーと割り当てを永続化し、正規化済み外向き通信に再現可能なルール判断・challenge・監査を記録して、未知・競合・障害を必ず拒否する。

## 背景

TASK-0006のPlaneメッセージ基盤とTASK-0008のWorkspace/Task境界の上に、Agent自己申告に依存しないCASB判断の正本が必要である。Go coreが統治DBを直接開くと所有境界と監査が崩れるため、Unix socket上のバージョン付きメッセージだけで接続する。

## スコープ

### 対象

- `governance.db` migration、PolicyRevision、WorkspaceSecurityPolicyBinding、egress attempt/rule decision/challenge/outboxのRust所有store。
- Workspaceネットワーク識別子からbindingを解決する決定論的ルール評価、deny-overrides、version/digest付きdecision、未許可時のimmutable challenge。
- SQLite永続化失敗、store unavailable/stale、未分類・競合時のfail-closedと監査/送信キュー。

### 対象外

- Linuxネットワーク名前空間・proxy・firewall強制（TASK-0017）、grant申請/有効化（TASK-0018）、LLM Policy Agent、恒久ルール改訂。
- HTTPS本文検査、credential注入、Memory/Wiki機能。

## 受け入れ条件

- [ ] Rust serviceだけが`governance.db`を書き、Go coreはschema version/digest付きUnix socket envelopeで要求・結果を交換する。
- [ ] Workspace binding、固定policy version、正規requestを入力に、同じrule decision（matched rule/reason code/allow-block）を再現する。
- [ ] allowはReady storeの明示allowだけであり、bindingなし・競合・unknown/stale/unavailable・永続化失敗は外部転送前にblockする。
- [ ] blockはattempt、immutable challenge、`EgressBlocked` outboxを同一統治transactionで確定し、allowはattempt/decision/capture manifest/intent記録を先行永続化する。

## 検討すべき設計観点

- Policy主体はWorkspaceでありTask/Agentは監査来歴だけにする。challenge本体は変更せず再試行はObservationへ集約する。
- 二重書込みを避け、Controlとの収束はmessage ID/idempotency key/ACKで行う。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] 必要な実装、テスト、文書が完成している。
- [ ] 独立レビューと`make check`がPASSしている。
- [ ] マージ後QAが完了している。
- [ ] HANDOVERがWiki Agentに取り込みされている。

## 関連コンテキスト

### 意味 Wiki

- docs/07 §§1,3,6,7,8、docs/11 §§1,2,5、docs/12 §5、docs/13 §§4-6。依存: TASK-0006、TASK-0008。
### 判断

- Governance PlaneがDBを単独所有し、inline decisionにLLMを置かず、障害時はfail-closedにする。
### 適用しなかった重要な判断

- Go coreのDB直接アクセス、Agent/Task主体のpolicy、allow-by-default、mutable challengeを採用しない。
