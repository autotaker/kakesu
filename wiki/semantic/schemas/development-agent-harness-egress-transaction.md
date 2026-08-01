---
kind: schema
title: Development Agent Harness Egress Transaction
---

# Development Agent Harness Egress Transaction

## 問い

HTTPの許可判断、Opaque ケイパビリティの一回消費、実認証情報の置換を、解釈のずれや秘密情報の逆流なしにどう接続するか。

## スコープを一度だけ導出する

`egresspolicy.Policy.Evaluate`は、HTTP許可判断と同じ評価からプロバイダー、リポジトリ、操作、宛先ホストを返す。外向き通信トランザクションはURLを再解析せず、この正規スコープをそのままケイパビリティの完全一致検証へ渡す。

GitHub REST readでは`github`、`owner/repo`、`github-rest-read`、`api.github.com`、OpenAI Responses textでは`openai`、リポジトリなし、`openai-responses-text`、`api.openai.com`となる。拒否時は空スコープと固定errorだけを返す。従来の`Authorize`は同じ評価へ委譲し、既存のdecision/error契約を維持する。

## 秘密処理へ到達する順序

`egresstransaction.Transaction.Execute`は、次の順序を固定する。

1. HTTP allowlistを評価する。
2. プロバイダーごとに許したAuthorization値を一つだけ抽出する。
3. 正規スコープとAgent主体を完全一致させてケイパビリティを一回消費する。
4. trusted resolverから実認証情報を一回だけ取得し、長さとvisible ASCIIを検証する。
5. 上流用Bearerへ置換したリクエストをtrusted Forwarderへ同期的に一回だけ渡す。

policy又はAuthorizationの拒否ではケイパビリティ、resolver、Forwarderへ到達しない。ケイパビリティ拒否ではresolverとForwarderへ到達しない。resolver又はForwarderが失敗しても消費済みケイパビリティを戻さず、同じ試行を再実行しない。

## 実認証情報の受渡し境界

Credential-bearingな`PreparedRequest`は、Transactionへ注入されたbroker内のtrusted Forwarderへの同期呼出中だけ渡す。`Execute`は値を返さず、TransactionもPreparedRequest、元のAuthorization、Opaque handle、実認証情報を保持しない。本文は独立コピーを渡す。

この性質だけではForwarder実装、秘密情報ストア、HTTP転送の安全性を証明しない。Forwarderが実認証情報を保持、記録、Agent側へ返さないこと、TLS/DNS/socket/redirect/responseをfail-closedに扱うことは後続境界で別途検証する。

## 適用限界

このトランザクションはin-memoryの認可接続コアであり、実認証情報の読取・生成、GitHub App token交換、OpenAI key管理、HTTP listener、CONNECT/TLS終端、CA、DNS、上流通信、監査、永続化を実装しない。成功は外部通信の成功や認証情報非露出のlive証明を意味しない。

## 関連

- [TASK-0041 HANDOVER](../../../tasks/TASK-0041-dev-agent-egress-transaction/HANDOVER.md)
- [Development Agent Harness Egress Policy](development-agent-harness-egress-policy.md)
- [Development Agent Harness Capability Registry](development-agent-harness-capability-registry.md)
