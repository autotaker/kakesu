---
kind: schema
title: Development Agent Harness Passkey Challenge Lifecycle
---

# Development Agent Harness Passkey Challenge Lifecycle

## 移行対象

TASK-0072の`internal/approvalchallenge`は、manifest digestへ束縛したone-shot Passkey/WebAuthn challenge managerとして導入された履歴的実装である。TASK-0074により、manifest/digest、ref、SHAをchallenge又はdecisionの束縛に使用する契約は廃止済みであり、後続実装はrepository単位request/decision/grantへ移行する。

## 現在の境界

Passkeyは、operatorがその場でapprove又はdenyを確認した本人確認だけを担う。TailscaleはApproval UIへの到達と接続元識別を担う。いずれも単独ではpushを認可せず、repository単位one-shot `push grant`の代替にはならない。

後続のchallengeは一回のrepository単位requestとdecisionへ結び付け、成功した本人確認をそのexact decisionへ接続する。grantはAgent instance/UID、workspace、repository、TTL、use、revokeに束縛され、上流`git-receive-pack`試行前に消費される。branch/commit/ref/SHAは参考情報であり、Passkeyの認可束縛には使わない。

## 関連

- [TASK-0072 HANDOVER](../../../tasks/TASK-0072-passkey-challenge-lifecycle/HANDOVER.md)
- [TASK-0074 HANDOVER](../../../tasks/TASK-0074-simplify-push-approval-and-proxy-contract/HANDOVER.md)
- [Approval Request Store](development-agent-harness-approval-request-store.md)
