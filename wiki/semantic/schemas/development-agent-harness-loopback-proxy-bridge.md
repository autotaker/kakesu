---
kind: schema
title: Development Agent Harness Loopback Proxy Bridge
---

# Development Agent Harness Loopback Proxy Bridge

## 問い

HTTP proxy endpointだけを扱うAgent clientを、既存の接続元に束縛されたegress Unix socketへ、認可を移さず接続するにはどうするか。

## 固定された接続入口

`internal/proxybridge`は一回だけ`tcp4`の正確な`127.0.0.1:0`をlistenし、返却addressをIPv4 loopbackとnon-zero portまで検証して、canonicalな`http://127.0.0.1:<port>` endpointだけを返す。bind address、port、egress path、TCP upstream又はenvironment overrideをAgent入力として受け取らない。

trusted constructorはabsoluteかつcleanなUnix socket pathと1--64のconcurrency上限だけを保持する。accepted clientごとに、その固定pathへ`unix`でdeadline付きのdialを一回だけ行う。dial失敗時はclientを閉じ、retry、別socket、TCP fallback、payload forwarding又は診断詳細を追加しない。

## streamと終了責務

slotは`Accept`より先に取得するため、上限中は追加connectionのaccept又はUnix dialを開始しない。dial成功後だけ、bridgeはHTTP、CONNECT、TLS又はcredentialを解釈・変更せず、raw bytesを双方向へstreamする。一方向のEOFは反対側のwrite half-closeへ伝え、cancel、copy/close failure又はunexpected accept failureでは両端をcloseしてactive connectionをdrainする。parent context cancellationはlistenerとactive connectionを閉じて正常終了する。

主体binding、CONNECT/control認可、TLS/inner HTTP policyは既存egress Unix serviceに残る。loopback bridgeは認可proxy、credential処理又は診断境界ではない。

## 適用限界

launcherはbridgeを一つのagent sessionへ束縛し、child process/signal/environment lifecycle、CA trust file、Git設定を所有する。詳細は[Development Agent Harness Agent Session Launcher](development-agent-harness-agent-session-launcher.md)を参照する。実OS network namespace/loopback isolation、Unix socket permission/peer UID、実Git/`gh`/OpenAI clientのproxy対応、CA trust、systemd/VPSはlive E2Eで未確認かつblockedであり、hermetic testのPASSで代替しない。

## 関連

- [TASK-0067 HANDOVER](../../../tasks/TASK-0067-agent-loopback-proxy-bridge/HANDOVER.md)
- [Development Agent Harness Egress Transaction](development-agent-harness-egress-transaction.md)
- [Development Agent Harness Proxy CA](development-agent-harness-proxy-ca.md)
- [Development Agent Harness Agent Session Launcher](development-agent-harness-agent-session-launcher.md)
