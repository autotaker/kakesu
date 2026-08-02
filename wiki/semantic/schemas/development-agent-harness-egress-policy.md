---
kind: schema
title: Development Agent Harness Egress Policy
---

# Development Agent Harness Egress Policy

## 問い

Agentが実認証情報を得ずに、許可されたproviderへ接続する最小の認証情報差し替え境界をどう保つか。

## 薄いproxy契約

外向きproxyはprovider protocolの意味を再実装するgatewayではない。各要求でUnix socket peer identity、Agent instance/UID、workspace、Opaque capabilityのsubject/provider/repository/TTL/use/revoke、完全一致destination hostを検証し、合格時だけbrokerの実credentialへ置換する。実credentialはAgentへ返さず、redirect先へ再利用しない。

通常のGit read、GitHub REST、OpenAI APIは、HTTP framing、hop-by-hop header、credential秘匿に必要な最小処理を除き、method/path/query/bodyと上流status/headers/bodyを未解釈・非bufferのbackpressure付きstreamとして転送する。timeout、concurrency、request/response header個数・大きさ、接続単位resource budgetは維持する。

pushだけは、完全一致repositoryと`git-receive-pack`を最小分類する。本文、Git wire/pkt-line、参照、SHA、force/deleteを解析又は照合しない。別途必要なrepository単位one-shot `push grant`を、上流接続・本文送信その他の上流試行開始前に原子的に消費する。

## 削除・移行対象

次の旧policy/実装は将来用の互換契約として残さず、次の単一vertical-slice製品Taskで削除する。

- `approvalmanifest`、old/new SHA、ref一覧、force/delete、remote old SHA、Git wire/pkt-line本文照合。
- strict OpenAI JSON field/model/`store`/`stream`検査、GitHub `/repos/{owner}/{repo}` endpoint parser、Git upload-pack本文・response `Content-Type`の意味検査。
- 上流JSON検証、`2xx`限定、`Content-Type`検査、response全量buffer、固定1 MiB response上限。
- `Policy`→`Transaction`→`Exchange`→`Forwarder`の重複評価と不要な抽象層。

GitHub App installationのrepository限定write権限が、実際のGitHub操作範囲の上流安全境界である。provider意味検査を再導入する場合は、実E2Eで繰り返し観測した具体的不具合への最小対策として別Taskで判断する。

## 関連

- [TASK-0074 HANDOVER](../../../tasks/TASK-0074-simplify-push-approval-and-proxy-contract/HANDOVER.md)
- [Development Agent Harness Push Approval Manifest](development-agent-harness-push-approval-manifest.md)
