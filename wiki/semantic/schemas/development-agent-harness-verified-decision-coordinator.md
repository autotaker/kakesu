---
kind: schema
title: Development Agent Harness Verified Decision Coordinator
---

# Development Agent Harness Verified Decision Coordinator

## 移行対象

TASK-0073の`internal/approvaldecision`は、manifest digestで同一性を確認したchallenge結果をdurable approval stateへ接続する履歴的実装である。TASK-0074により、manifest/digest、ref、SHAをdecision又はpush認可に束縛する契約は廃止済みであり、既存実装を将来の認可根拠として維持しない。

## 移行先の順序

後続の単一vertical sliceでは、verified Passkey結果をrepository単位requestのapprove/deny decisionへ一回だけ接続する。Passkeyは本人確認、TailscaleはUI到達と接続元識別であり、どちらもpush authorizationではない。

approve decisionから発行するone-shot `push grant`だけが、完全一致repositoryへの次の`git-receive-pack`一回を認可する。grantはAgent instance/UID、workspace、repository、短TTL、未使用、revokeへ束縛し、上流試行前に消費する。同一repository内の別内容への一回利用は受容するが、別repository、別Agent/workspace、再使用、期限後、GitHub RESTその他操作への転用は拒否する。branch/commit/ref/SHAはUI参考情報であり、decision/grantの束縛には使わない。

## 関連

- [TASK-0073 HANDOVER](../../../tasks/TASK-0073-verified-decision-coordinator/HANDOVER.md)
- [TASK-0074 HANDOVER](../../../tasks/TASK-0074-simplify-push-approval-and-proxy-contract/HANDOVER.md)
- [Approval Request Store](development-agent-harness-approval-request-store.md)
