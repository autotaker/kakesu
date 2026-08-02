---
kind: schema
title: Development Agent Harness Capability Registry
---

# Development Agent Harness Capability Registry

## 問い

Development Agentへ実認証情報を渡さず、漏えいし得る短命handleをAgentとrequestの最小権限へどう束縛するか。

## Opaque handleと保存state

`internal/capability`は、32 byteのcrypto entropyから`cap_...` handleを発行するin-memory レジストリである。handleは実認証情報の暗号化表現やself-contained tokenではなく、broker-local stateへのrandom referenceである。

callerへ返したraw handleはレジストリへ保存しない。mapのキーは完全handleのSHA-256 ダイジェストであり、entryは発行済みスコープ、発行/期限時刻、残使用回数、ポリシーバージョン、失効世代だけを保持する。entropy failureと上限内に解消しない衝突はpartial entryを残さず固定errorで失敗する。

## 初期スコープ

発行時にAgent インスタンスID、non-root UID、workspace ID、プロバイダー、リポジトリ、TTL、使用回数を検証する。初期プロバイダー面は次の三つだけである。

- GitHub: 完全一致するlowercase `owner/repo`、`github-rest-read`、`api.github.com`
- Git Smart HTTP read: 完全一致するlowercase `owner/repo`、`github-git-read`、`github.com`
- OpenAI: リポジトリなし、`openai-responses-text`、`api.openai.com`

callerは宛先ホストを任意指定して発行できない。既存GitHub RESTとOpenAIはoperation省略時の固定defaultを維持する一方、Git Smart HTTP readは明示`github-git-read` selectorでだけ発行する。発行はpeer-derived subject、同一repository、固定5分TTL、1回使用へ束縛する。request利用時はhandleに加えてsubject、workspace、プロバイダー、リポジトリ、操作、宛先ホストを正規化せず完全一致させる。

## 原子的な利用と失効

`Consume`は一つのmutex transactionでentry lookup、ポリシーバージョン、失効世代、期限、残使用回数、全スコープを判定する。成功時だけ使用回数を一回減らし、最後の成功前にentryを削除する。スコープ不一致は正当なhandleの使用回数を消費しない。

内部TTLはprocess内のmonotonic elapsedで判定し、wall clock後退による有効期間の延長を許さない。monotonic readingを除いたwall clockも独立に比較し、時刻が前進した場合は早い方で失効する。呼出元へ返す発行/期限時刻だけをUTCへ変換する。

`Revoke`は一つのhandleを削除し、`AdvanceRevocationEpoch`は失効世代の単調増加と全entry失効を同じtransactionで行う。unknown/malformed/expired/revoked handleと古い失効世代は同じ固定denyとなる。1-use handleへ並行requestがあっても成功はexactly oneである。

レジストリは永続化しない。process restartで全entryが失われ、既存handleが全て無効になることをfail-safe動作とする。復元、複数process共有、実認証情報、proxy、TLS/DNS、外向き通信、監査、費用制限は後続Taskの別境界である。

## 関連

- [TASK-0040 HANDOVER](../../../tasks/TASK-0040-dev-agent-capability-registry/HANDOVER.md)
- [TASK-0063 HANDOVER](../../../tasks/TASK-0063-dev-agent-git-smart-http-read/HANDOVER.md)
- [Development Agent Harness Egress Policy](development-agent-harness-egress-policy.md)
