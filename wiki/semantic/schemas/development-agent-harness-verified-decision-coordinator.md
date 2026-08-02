---
kind: schema
title: Development Agent Harness Verified Decision Coordinator
---

# Development Agent Harness Verified Decision Coordinator

## 問い

durable approval requestとone-shot verifier challengeを、呼出し側が認可対象又は信頼境界を差し替えられない順序で接続し、検証済み入力をdurable decisionへいつ昇格させるか。

## 所有と順序

`internal/approvaldecision`のproduction constructorはconcrete approval state store、challenge manager、trusted verifierを一度だけ固定する。`Begin`はcaller supplied digest、clock、state又はchallengeを受けず、store `Get`で確認したexact pending recordのrequest IDとderived digestだけを、callerのdecision、operator、RP ID、originとともに`Issue`へ渡す。

`Complete`は固定verifierでchallengeを一回だけ`Consume`してから、verified bindingのexact request ID、digest、operator、decisionだけを使う。decisionが`approve`なら`Approve`、`deny`なら`Deny`を一回だけ選び、durable transitionが成功した場合だけdurable recordとstable credential IDを返す。verified resultだけ、又はcallerが失った応答は成功を意味しない。

## 失敗・競合・再照合

verification failure又はpanic、replay、expiry、terminal state、digest mismatch、transition/persistence/poison failureでは、resultを空にして固定の非漏えいerrorを返す。`Consume`済みchallengeを復活、再消費、自動再発行せず、別decisionへのfallback又は成功応答の再構成もしない。response loss後はcallerがstore `Get`でdurable stateを照合する。

同一requestのapprove/deny challengeが競合しても、first-winsの唯一の正本はstoreのatomic pending transitionである。coordinatorは別のlock、rollback又は成功推測を持たない。新challengeは`Begin`がrequestがなおpendingであることを確認した時だけ発行できる。poisonからの復旧はstoreの`Close`、`Open`、上位reconciliationが所有する。

## 認可境界と適用限界

trusted verifierはこのcoordinatorがactual WebAuthn cryptographic verification、credential lifecycle、又はTailscale identityを実装済みであることを意味しない。durable `approved`/`denied`もaudit、grant、push authorization、実push成功を意味しない。HTTP/UI/session/CSRF、実WebAuthn authenticatorとcredential、Tailscale、外部作用、deployment/restart/rollbackはlive E2Eで別途確認する。

## 関連

- [TASK-0073 HANDOVER](../../../tasks/TASK-0073-verified-decision-coordinator/HANDOVER.md)
- [Approval Request Store](development-agent-harness-approval-request-store.md)
- [Passkey Challenge Lifecycle](development-agent-harness-passkey-challenge-lifecycle.md)
