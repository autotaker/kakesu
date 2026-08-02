---
kind: schema
title: Development Agent Harness Passkey Challenge Lifecycle
---

# Development Agent Harness Passkey Challenge Lifecycle

## 問い

pending approval requestに対するPasskey/WebAuthn verifierの入力を、別request・判断・主体へ差し替えられず、失敗や再起動後にも再利用できない一回限りのchallengeとしてどう束縛するか。

## 発行と束縛

`internal/approvalchallenge`はboundedなin-memory managerである。production constructorは`crypto/rand`とsystem clockを所有し、callerにchallenge、乱数又は発行・期限時刻を注入させない。opaqueなbase64url tokenは少なくとも32 bytesのunbiased random値であり、request ID、canonical manifest digest、`approve`/`deny`、operator ID、RP ID、exact HTTPS origin、issued/expiry instantをmanager内のimmutable bindingへ対応付ける。tokenへこれらの値を埋め込まない。

発行前にdecision、request/digest、operator、RP ID、origin/RPの整合、正のTTL、capacityを検査する。pending集合内のtoken衝突はbounded retryし、入力値やsecretを含まない固定error classで拒否する。managerはpending challengeだけを保持し、消費済みtokenの無制限なtombstoneを持たない。

## Consumeの一回性

consumeはmutex下で最初の試行をverifier起動前に原子的にreservationする。unknown、replay、closed、expiry到達、clock rollbackはfail closedであり、expiryはconsumeより先に判定する。reservation後はverifier成功、error、panic、同時試行のどれでもchallengeをpendingへ戻さない。失敗時の再試行には、なお有効な上位requestに対する新challenge発行が必要である。

verifierにはimmutable bindingとcopyしたassertion bytesだけを渡す。成功結果はrequest ID、digest、decision、operator ID、verified time、raw credential IDからdomain-separated digestへ変換したstable IDだけをcopy ownershipで返す。managerはchallenge、raw assertion、signature、credential public keyを結果又は永続状態として保持・公開しない。

## 失効と認可境界

`Close`はpending challengeを破棄する。challengeはdisk、log、環境変数、外部DBへ保存しないため、restart相当の新managerは旧tokenを復元せず拒否する。このin-memory fail-closed特性はmulti-process/host共有、durable recovery、backupを提供しない。

verified resultはWebAuthnのclientData/authenticatorData/signature、RP ID hash、origin、UV flag、counterを暗号学的に検証済みとは意味しない。また、[Approval Request Store](development-agent-harness-approval-request-store.md)の`approved`/`denied` mutation、Tailscale identity、grant発行・消費、push authorization又は実pushを意味しない。[Verified Decision Coordinator](development-agent-harness-verified-decision-coordinator.md)がtrusted verifierの結果を一回の`Consume`後にexact durable transitionへ接続するが、これらの上位境界は明示的に別途接続・検証する。

## 適用限界

実WebAuthn authenticator、credential登録・失効・recovery、Tailscale Serve/identity header、HTTP/UI/session/CSRF、実スマートフォン、push grant、実Git/GitHub通信はこのmanagerのhermetic検査だけでは確認できない。実OS/auth/networkや外部作用を含む接続は、環境に応じたlive E2Eで別途検証する。

## 関連

- [TASK-0072 HANDOVER](../../../tasks/TASK-0072-passkey-challenge-lifecycle/HANDOVER.md)
- [Approval Request Store](development-agent-harness-approval-request-store.md)
- [Verified Decision Coordinator](development-agent-harness-verified-decision-coordinator.md)
- [Push Approval Manifest](development-agent-harness-push-approval-manifest.md)
