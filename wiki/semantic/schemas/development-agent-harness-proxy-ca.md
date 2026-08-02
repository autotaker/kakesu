---
kind: schema
title: Development Agent Harness Proxy CA
---

# Development Agent Harness Proxy CA

## 問い

Agent向けのTLS interceptionで、ブローカーのCA秘密鍵を保持・検証し、許可された上流だけの短命leaf証明書をどのように発行するか。

## in-memory CAの境界

`proxyca.Authority`は、broker memoryで受け取った自己署名ECDSA P-256 CA certificate/keyを厳格に検証してから、parse済みcertificateとsignerだけを保持する。入力PEMは保持せず、公開certificateのcopyだけをexportできる。CA private materialをexportするAPIは置かない。

CA入力は単一の余剰dataのないPEMとして扱い、自己署名、CA用途、鍵との一致を満たさない入力を拒否する。失敗は入力値、PEM、鍵又は検証詳細を含まない固定errorへ縮退する。

## host限定のleaf発行

Authorityは完全一致する`api.github.com`、`github.com`、`api.openai.com`だけを対象にする。`github.com`のleafはGit Smart HTTP readのCONNECT/SNI面をpolicyと同じく狭めるものであり、GitHub RESTやpushを許可するものではない。各呼出しごとに独立したP-256 private key、serial、SANを持つ短命leaf certificateを発行し、呼出し間でkey又はcertificate stateを共有しない。その他のhostnameは発行前に固定errorで拒否する。

公開CA copy、TLS hostname検証、並行発行時のserial/key/SAN隔離はhermetic testで確認する。この証明書境界は、egress policyが導出したprovider hostを広げず、provider別の許可面をTLS interceptionにも保つ。

## 適用限界

この境界はCA file lifecycle、OS trust store、listener、CONNECT、SNI routing、実client、実VPS、実GitHub/OpenAIへの接続を実装又は確認しない。hermetic TLSのPASSは、それらのlive E2E、実配置、restart、rollback又はcleanupの証拠にならない。

## 関連

- [TASK-0050 HANDOVER](../../../tasks/TASK-0050-dev-agent-proxy-ca/HANDOVER.md)
- [TASK-0063 HANDOVER](../../../tasks/TASK-0063-dev-agent-git-smart-http-read/HANDOVER.md)
- [Development Agent Harness Egress Transaction](development-agent-harness-egress-transaction.md)
- [Development Agent Harness Egress Policy](development-agent-harness-egress-policy.md)
