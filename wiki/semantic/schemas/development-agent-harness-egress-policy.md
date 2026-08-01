---
kind: schema
title: Development Agent Harness Egress Policy
---

# Development Agent Harness Egress Policy

## 問い

Development Agentから外向き通信を実行する前に、GitHubとOpenAIの最小request surfaceをどこでfail-closedに判定するか。

## Pure policyの境界

`internal/egresspolicy`は、RulesとRequestだけを入力にする副作用のないallowlist判断コアである。file、environment、process、network、DNS、TLS、clock、randomを使用せず、Credentialも扱わない。allow decisionは実通信の許可ではなく、後続のproxyが別のsecurity boundaryを満たすための一条件にすぎない。

Rulesは許可repository、許可model、OpenAI body上限、output token上限を生成時に検証してcopyする。生成後にcallerが元sliceを変更してもpolicyの結果は変わらない。nilまたはzero policy、曖昧なURL、未知のfieldはdefault denyとなる。

## 初期allow surface

GitHubは、allowlistへ完全一致するlowercase `owner/repo`に対するcanonical `GET`または`HEAD https://api.github.com[:443]/repos/{owner}/{repo}`と、そのcanonical child pathだけを許可候補にする。query、fragment、userinfo、percent encoding、dot segment、empty segment、別host、別port、write methodは拒否する。

OpenAIは、canonical `POST https://api.openai.com[:443]/v1/responses`だけを許可候補にする。bodyはparameterなし`application/json`のstrict objectとし、許可model、non-empty string input、明示的な`store:false`と`stream:false`、正かつ上限内の`max_output_tokens`を必須にする。追加できるfieldはstring `instructions`だけであり、tool、file、image、background、continuationを含む未知fieldは拒否する。

denyは入力値やparser errorを文字列化せず、固定errorだけを返す。provider別allow decisionは、後続処理がGitHubとOpenAIのCredentialやupstreamを混同しないために分ける。

## 後続Taskの責務

TLS終端、client certificate trust、DNS/address検査、redirect、Credential取得と置換、Opaque capability、audit、rate limit、実upstream通信は本policyの外にある。proxyを実装するときは、本decisionだけで接続せず、それらを別途fail-closedに満たす。

新しいAPI surfaceを許可する場合はgeneric URL例外を加えず、provider、method、path、body、Credential、redirectの境界を別Taskで明示する。

## 関連

- [TASK-0039 HANDOVER](../../../tasks/TASK-0039-dev-agent-egress-policy/HANDOVER.md)
- [Development Agent Harness Config Policy](development-agent-harness-config.md)
