---
kind: schema
title: Development Agent Harness Push Approval Manifest
---

# Development Agent Harness Push Approval Manifest

## 履歴的実装

`internal/approvalmanifest`は、TASK-0070で導入された、push proposalをcanonical JSON、ref 更新、old/new SHA、force/delete、digestへ束縛する履歴的な実装である。TASK-0074により、この契約は廃止済み（superseded）であり、将来の承認・grant・push認可の根拠として残さない。

そのため、canonicalization、manifest digest、ref/SHA一覧、remote old SHA、Git wire/pkt-line本文との照合は移行先契約に含めない。過去のTASK-0070証跡は実装履歴として保持するが、現在または将来の安全境界を表すものではない。

## 移行先

push承認の安全単位は、完全一致repositoryへの次の`git-receive-pack`一回である。repository単位requestとverified decisionから発行する短命one-shot `push grant`は、Agent instance/UID、workspace、repository、TTL、未使用状態、revokeへ束縛する。上流試行前に原子的に消費し、失敗・結果不明でも再利用しない。

branch、commit、ref、SHA、force/deleteはUIの参考情報にだけ使用でき、認可条件ではない。同一repository内で参考情報と異なる内容が一回pushされうる残余リスクは受容する。一方、別repository、別Agent/UID、別workspace、再使用、期限後、GitHub RESTその他操作への転用はfail-closedに拒否する。

## 関連

- [TASK-0070 HANDOVER](../../../tasks/TASK-0070-push-approval-manifest/HANDOVER.md)
- [TASK-0074 HANDOVER](../../../tasks/TASK-0074-simplify-push-approval-and-proxy-contract/HANDOVER.md)
- [Development Agent Harness Egress Policy](development-agent-harness-egress-policy.md)
