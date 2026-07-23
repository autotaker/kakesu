---
task_id: "TASK-0018"
title: "一時許可申請とone-use有効化を実装する"
status: ready
created_at: "2026-07-15"
---

# TASK-0018 一時許可申請とone-use有効化を実装する

## 目的

`EgressBlocked`を起点にWork Agentの`request_grant`を最小入力で評価し、Policy Agent判断を検証して、強制点のACK後だけ有効なWorkspace限定・完全一致・one-use grantを通知する。最初のCLI通信は自動再生せずAgentが手動再実行する。

## 背景

TASK-0011のWork Agent tool境界、TASK-0013の非同期/mailbox再開、TASK-0016のchallengeとTASK-0017の強制点を結び、権限をAgent出力や同期応答だけで広げない。

## スコープ

### 対象

- `EgressBlocked → request_grant → GrantRequest/evaluation job → structured grant/deny → pending_activation → policy ACK → active/PolicyGrantReady`の冪等状態機械。
- 不変challengeとTask/Workspace/契約/evidenceの照合、Policy Agentの最小評価入力、one-use完全一致rule、Agentの明示再実行。

### 対象外

- 人間Authorityの実装、責任者UX、恒久PolicyRevisionの改訂、TLS本文検査/credential broker。
- 初回拒否通信のgateway自動再生、grantの任意scope/TTL/credential指定。

## 受け入れ条件

- [ ] 有効な同一Workspace challengeだけが`request_grant`を作れ、同じTask/callまたは非終端workspace/challengeは同一request/async operationへ収束する。
- [ ] Policy Agentは無害化済み固定入力を構造化`grant|deny|require_authority`で返し、managerがauto-grant/authority上限を再検査する。
- [ ] grantはchallenge由来の完全一致・期限付きone-useで`pending_activation`に保存され、強制点のpolicy-version ACK前には使えない。
- [ ] ACK後の`active`だけがAsyncCompleted/PolicyGrantReadyをメールボックスへ配送し、Agentが元CLIを手動再実行した一回だけ消費できる。

## 検討すべき設計観点

- Policy Agentはrule/store更新権限を持たず、scope/authorityはchallengeとplatform policyが決定する。Control/Governance間は二相DB transactionでなくoutbox/inbox/ACKで収束する。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] 必要な実装、テスト、文書が完成している。
- [ ] 独立レビューと`make check`がPASSしている。
- [ ] マージ後QAが完了している。
- [ ] HANDOVERがWiki Agentに取り込みされている。

## 関連コンテキスト

### 意味 Wiki

- docs/04 §4、docs/05 §10、docs/06 §§8,13-17、docs/07 §§9-15、docs/10 §9、docs/11 §5。依存: TASK-0011、0013、0016、0017。
### 判断

- grantはACKまでinactive、完全一致one-use、初回通信は再生しない。
### 適用しなかった重要な判断

- 人間Authority、恒久rule改訂、Agentによるscope生成、自動replayを採用しない。
