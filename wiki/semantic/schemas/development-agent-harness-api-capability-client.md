---
kind: schema
title: Development Agent Harness API Capability Client
---

# Development Agent Harness API Capability Client

## 問い

GitHub REST readとOpenAI Responses textに必要なOpaque capabilityを、実認証情報や可変なprovider入力をAgentへ渡さず取得するにはどうするか。

## 明示issue操作と固定wire

`internal/controlclient`は固定Unix control socketへ一接続一操作で発行要求を送る。`IssueGitHubREST`はabsoluteかつcleanなsocket pathとcanonicalな`owner/repo`だけを受け、`POST /v1/capabilities`へ`{"provider":"github","repository":"owner/repo"}`を送る。`IssueOpenAI`はsocket pathだけを受け、同じpathへ`{"provider":"openai"}`を送る。

provider、operation、model、HTTP path、任意bodyはcaller入力にしない。既存`Issue`は明示`github-git-read`のGit Smart HTTP read用wireを保つ。各操作は一回だけdialしてrequestを送り、deadline付きで一つのresponseを読み、connectionをcloseする。

## 応答境界と非漏洩

成功は唯一のbounded `200 application/json`、canonicalなheader順序とContent-Length、exact `{"handle":"cap_..."}`、本文直後のEOFだけである。status、header、body、JSON、handle、framing又はextra byteの不一致、dial/deadline/read/write/close failureは空値と固定errorへ畳む。retryとfallbackはしない。

error又はdiagnosticはsocket、repository、handle、wire、lower-level errorを保持・出力しない。

## 使用回数と適用限界

peer-bound controllerはGitHub REST readとOpenAI Responses textのAPI scopeへ固定16 usesを発行し、16回目の正規consume後のremainingは0、17回目は拒否する。Git Smart HTTP readは1 useのままである。subject、workspace、provider、repository、operation、destinationのmismatchは拒否され、budgetを消費しない。

launcher、environment、`GH_TOKEN`/`OPENAI_API_KEY`設定、child process、CA trust file、Git config、loopback bridge、実providerは実装していない。実credential、実GitHub/OpenAI、実`gh`/SDK、DNS/TLS、Unix socket permission/別UID、systemd/VPSは未確認であり、hermetic testのPASSで代替しない。

## 関連

- [TASK-0068 HANDOVER](../../../tasks/TASK-0068-api-capability-control-client/HANDOVER.md)
- [Development Agent Harness Capability Registry](development-agent-harness-capability-registry.md)
- [Development Agent Harness Git Credential Helper](development-agent-harness-git-credential-helper.md)
