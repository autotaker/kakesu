---
task_id: "TASK-0023"
title: "P0 HTTPS傍受・正規化プロキシを実装する"
status: ready
created_at: "2026-07-15"
---

# TASK-0023 P0 HTTPS傍受・正規化プロキシを実装する

## 目的

P0 HTTPSをTask-local CAでTLS終端し、HTTP/1.1/2をcanonicalizeしてrequest digestと監査バリアを確定してからだけ外側接続を作る。

## 背景

`docs/07` §4はproxy迂回を許さず、ambiguous framingやstreaming-before-inspectionを拒否し、外部到達前の永続監査を要求する。Task-local CA trustはWorkspace identityとファイルシステムallowlistに結び、workload起動前に限定配置する。依存: TASK-0006、TASK-0008、TASK-0009、TASK-0016、TASK-0022。

## スコープ

### 対象

- Workspace identityとLandlock/LSM allowlistへ結び付けたTask-local CA/trust、TLS termination、HTTP/1.1/2 canonicalization、request/body digest。
- redirect/rebinding/ambiguous framing/generic CONNECT/Upgrade/WebSocket/streaming-before-inspection拒否。
- Governance判定と、外側接続前のEgressAttempt/decision/capture/`intent_committed` OutboundTransaction audit barrier。

### 対象外

- credential broker、非HTTP TLS/SNI L4、本文の高度分類・LLM分類。
- network namespace/firewall route（TASK-0017）、workspace identity生成（TASK-0009）。

## 受け入れ条件

- [ ] Task-local CA trustはTASK-0009のWorkspace identityとTASK-0022のread-only runtime allowlistへ結び付けてworkload起動前に配置され、hostや別Workspaceへ漏れない。proxyはTLSを終端してHTTP/1.1/2を一つのcanonical requestへ正規化する。
- [ ] canonical request digest、policy header/body digest、workspace/task/contract/policy bindingをGovernanceへ渡す。
- [ ] redirect、DNS rebinding、ambiguous framing、generic CONNECT/Upgrade/WebSocket、本文検査前streamingは外側接続なしで拒否される。
- [ ] allow時も外側TLS接続より先に不変EgressAttempt・rule decision・capture manifest・`intent_committed` OutboundTransactionを永続化し、永続化不能なら転送しない。

## 検討すべき設計観点

- HTTP version差をpolicy inputへ漏らさず、外部作用の可能性がある全経路をaudit barrierの後に置く。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] 必要な実装、テスト、文書が完成している。
- [ ] 独立レビューと`make check`がPASSしている。
- [ ] マージ後QAが完了している。
- [ ] HANDOVERがWiki Agentに取り込みされている。

## 関連コンテキスト

### 意味 Wiki

- docs/07 §§3-4,6-7、docs/08 §4、docs/11 §§2,5、docs/12 §5、docs/13 §§2,4。依存: TASK-0006、TASK-0008、TASK-0009、TASK-0016、TASK-0022。

### 判断

- P0はinterception可能なHTTPSだけをbuffer・canonicalize・auditして転送し、未対応TLS/曖昧HTTPは拒否する。

### 適用しなかった重要な判断

- credential injection、non-HTTP TLS、本文高度分類、pass-through CONNECTは不採用。
