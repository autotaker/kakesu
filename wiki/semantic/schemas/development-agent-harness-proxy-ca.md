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

## 公開CAのcontrol取得

peer-bound egress Unix socketは、`GET /v1/proxy-ca HTTP/1.1` とcanonicalなzero `Content-Length`だけを一接続一操作として受け入れる。valid Authorityからはcertificate-only public CA PEMのfresh copyだけを、固定200 responseとcloseで返す。method/path/query/header/body又はearly byteの逸脱、nil又は不正なAuthority出力は固定403へ縮退し、CONNECT、Issue、Revokeの意味を変えない。

serverはresponse前に、clientはserver実装と別にresponse後に、単一canonical `CERTIFICATE` PEM、self-signed ECDSA P-256 CA、Basic Constraints、CertSign、validityを検証する。private material、複数block、trailing byte、期限外又は不正なcertificateは受理・返却せず、失敗はPEM、subject、socket/path、下位errorを露出しない。Authority、server response、client returnはそれぞれaliasしないcopyである。

## 適用限界

launcherはpublic CA copyをsession専用の0600 trust fileへ置き、childへCA trust変数を設定する。詳細は[Development Agent Harness Agent Session Launcher](development-agent-harness-agent-session-launcher.md)を参照する。この境界はOS trust store、listener、CONNECT、SNI routing、実client、実VPS、実GitHub/OpenAIへの接続を実装又は確認しない。hermetic TLS/control testのPASSは、それらのlive E2E、実配置、restart、rollback又はcleanupの証拠にならない。

## 関連

- [TASK-0050 HANDOVER](../../../tasks/TASK-0050-dev-agent-proxy-ca/HANDOVER.md)
- [TASK-0063 HANDOVER](../../../tasks/TASK-0063-dev-agent-git-smart-http-read/HANDOVER.md)
- [TASK-0066 HANDOVER](../../../tasks/TASK-0066-proxy-ca-control-client/HANDOVER.md)
- [Development Agent Harness Egress Transaction](development-agent-harness-egress-transaction.md)
- [Development Agent Harness Egress Policy](development-agent-harness-egress-policy.md)
- [Development Agent Harness Agent Session Launcher](development-agent-harness-agent-session-launcher.md)
