---
kind: schema
title: Development Agent Harness Approval Request Store
---

# Development Agent Harness Approval Request Store

## 移行対象

TASK-0071の`internal/approvalstate`はcanonical manifestとdigestへ束縛したdurable request storeとして導入された履歴的実装である。TASK-0074により、manifest/digestをrequestの同一性又は認可根拠に使う契約は廃止済みである。既存実装とTASK-0071証跡は履歴として保持するが、後続実装はそれを拡張・互換維持しない。

## 移行先の責務

後続の単一vertical sliceは、repository単位のrequestとapprove/deny decisionを保持し、そこからrepository単位one-shot `push grant`を発行する。grantはAgent instance/UID、workspace、完全一致repository、短いTTL、一回使用、revokeへ束縛する。grantは上流`git-receive-pack`試行前に原子的に消費し、上流失敗又は結果不明でも再使用しない。

branch、commit、ref、SHA、force/deleteはrequest/decision/grantの認可条件ではなくUI参考情報だけにする。PasskeyとTailscaleはoperatorの本人確認又はUI到達を担い、push認可そのものを発行しない。同一repositoryの別内容への一回利用は受容するが、repository/Agent/workspace越境、再使用、期限後、GitHub RESTその他操作への転用は拒否する。

## 関連

- [TASK-0071 HANDOVER](../../../tasks/TASK-0071-approval-request-store/HANDOVER.md)
- [TASK-0074 HANDOVER](../../../tasks/TASK-0074-simplify-push-approval-and-proxy-contract/HANDOVER.md)
- [Development Agent Harness Egress Policy](development-agent-harness-egress-policy.md)
