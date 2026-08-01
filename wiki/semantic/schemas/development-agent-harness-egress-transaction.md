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

## 上流 HTTPS transportとの接続

ForwarderがGitHub又はOpenAIへ送る最初の接続境界には、固定allowlistの`http.RoundTripper`を注入する。transportはoriginと`Request.Host`をDNS前に照合し、resolverの全answerを検査して、一件でもunsafeなaddressを含む集合を拒否する。安全なanswerだけの場合も、検査済みIP literalの443番だけへdialし、元hostnameをTLSのSNIと証明書検証に使う。

このtransportは一request一接続で、環境proxy、keep-alive、自動compression、HTTP/2、redirect、retryを持たない。TCP接続失敗時だけ未使用の検査済みIPへ進めるが、TLS handshake又はHTTP送信開始後は再dialしない。TLS 1.2未満又はHTTP/1.1以外、失敗response、下位のDNS/TLS/socket detailは公開せず、失敗時のbodyはtransportがcloseする。

## Forwarderで再検証して縮退する応答

`internal/upstreamforwarder`は`PreparedRequest`を受けた後、transportへ渡す前に同じpolicyでrequestを再評価し、正規scopeの完全一致と上流Bearerを再検証する。GitHub GET/HEADは空Content-Typeかつ空本文だけに狭め、検証済みrequestを注入RoundTripperへ一回だけ送る。上流headerは実Authorization、固定AcceptとUser-Agent、およびOpenAIのContent-Typeだけに限定し、Agent由来headerやopaque capability handleを転記しない。本文はcaller所有sliceから独立copyする。

成功responseはsize上限内で完全にread、検証、closeしてから、status、正規JSON media type、独立した本文だけをrequest単位sinkへ一回渡す。HEAD/204 responseは空本文だけを受理し、provider error本文と上流headerはAgent側へ返さない。2xx以外、想定外media type、上限超過、UTF-8/JSON不正、read/close/timeout、sinkの各失敗はfail closedにする。

## 呼出し単位のExchange合成

`brokerexchange.Exchange`は、検証済みの依存と上限だけをimmutableに保持し、各`Do`呼出しごとにprivate capture sink、`upstreamforwarder.Forwarder`、`egresstransaction.Transaction`を新規に合成する。これにより既存のpolicy、Authorization、capability、credential、response検査の意味を再実装せずに委譲する。

成功時だけ、captureした縮退responseの独立copyを返す。いずれかの段階が失敗した場合はzero responseと固定`exchange-denied`だけを返し、上流失敗の詳細を公開しない。呼出し間でsinkやresponse本文を共有しないため、並行実行したresponseが相互に混入しない。

policy又はAuthorizationの拒否はcapabilityを消費しない。capability消費後にresolver、transport、Forwarder、captureのいずれかが失敗しても消費をrollbackせず、同一`Do`による上流試行は一回だけとする。

この性質だけでは秘密情報ストア、実network上のprovider受理、Agent側proxy、response writer又はauditを証明しない。実GitHub/OpenAI、実認証情報、Internet DNS/TLS/system trust、Agent proxy/response writerはlive E2Eで別途確認する。

## 適用限界

このトランザクションはin-memoryの認可接続コアであり、実認証情報の読取・生成、GitHub App token交換、OpenAI key管理、HTTP listener、CONNECT/TLS終端、CA、DNS、上流通信、監査、永続化を実装しない。前記transportのhermetic testも、実GitHub/OpenAI、実Internet DNS、実system trust store、実proxy/firewallでの成功や認証情報非露出を証明しない。

## 関連

- [TASK-0041 HANDOVER](../../../tasks/TASK-0041-dev-agent-egress-transaction/HANDOVER.md)
- [TASK-0045 HANDOVER](../../../tasks/TASK-0045-dev-agent-upstream-transport/HANDOVER.md)
- [TASK-0047 HANDOVER](../../../tasks/TASK-0047-dev-agent-upstream-forwarder/HANDOVER.md)
- [TASK-0048 HANDOVER](../../../tasks/TASK-0048-dev-agent-broker-exchange/HANDOVER.md)
- [Development Agent Harness Egress Policy](development-agent-harness-egress-policy.md)
- [Development Agent Harness Capability Registry](development-agent-harness-capability-registry.md)
