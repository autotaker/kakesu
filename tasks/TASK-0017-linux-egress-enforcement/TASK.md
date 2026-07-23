---
task_id: "TASK-0017"
title: "Linux外向き通信強制ブリッジを実装する"
status: ready
created_at: "2026-07-15"
---

# TASK-0017 Linux外向き通信強制ブリッジを実装する

## 目的

Linux P0 runtimeが与えるWorkspace network identityへDNS/proxy/firewall routeを配線し、managed endpoint以外の外向き通信を機械的に拒否する。HTTPS内容の解釈はTASK-0023、統治判断はTASK-0016の正本とする。

## 背景

CASB storeだけではCLIの直接接続を止められない。LinuxをMVP正本として、TASK-0009が作るnamespace/cgroup identityをrouteの強制点へ結合する必要がある。依存: TASK-0009、TASK-0016、TASK-0023。

## スコープ

### 対象

- TASK-0009のWorkspace network identityをDNS listener、HTTPS proxy listener、egress firewall routeへ結合し、managed endpointへの唯一の経路を配線・回収する。
- DNSのA/AAAA route、DNS/DoH/DoT/QUIC/raw/ICMP/生IP/private/link-local/metadata拒否、TASK-0023 HTTPS proxyへのtransport handoff。
- proxy/firewall/統治障害時のdefault denyと監査可能なblock。

### 対象外

- network namespace/cgroup/process identityの生成・probe（TASK-0009）。
- HTTPS TLS termination、HTTP/1.1/2 canonicalization、request digest、redirect/rebinding/framing/streaming拒否（TASK-0023）。
- TLS本文検査、credential broker、非HTTP TLS/SNI L4 proxy。
- grant workflow（TASK-0018）、macOS/Windows profile、P1/P2隔離。

## 受け入れ条件

- [ ] TASK-0009のworkspace network identityを持つsandbox processは、managed DNSとTASK-0023 HTTPS proxy listenerだけへrouteされ、外部IP/port・DNS bypass・UDP/QUIC・host/control socketへ直接到達できない。
- [ ] DNSは許可queryだけを統治へ正規化し、unknown FQDNを上流へ漏らさずchallenge経路へ渡す。
- [ ] HTTPS transportはWorkspace identityをTASK-0023へ渡す。HTTPS metadataのcanonicalization、TASK-0016へのpolicy handoff、allow前のouter connection禁止はTASK-0023の受け入れ条件で検証する。
- [ ] setup/teardown/adapter障害はprofile提供または通信を拒否し、隔離・否定試験で再現できる。

## 検討すべき設計観点

- firewall routeが最終強制点であり、identityはTASK-0009が生成する。IPv4/IPv6とDNS rebindingを同じdeny modelで扱い、HTTPS解釈を重複実装しない。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] 必要な実装、テスト、文書が完成している。
- [ ] 独立レビューと`make check`がPASSしている。
- [ ] マージ後QAが完了している。
- [ ] HANDOVERがWiki Agentに取り込みされている。

## 関連コンテキスト

### 意味 Wiki

- docs/07 §§2-5、docs/11 §2、docs/12 §5、docs/13 §§2,4。依存: TASK-0009、TASK-0016、TASK-0023。
### 判断

- Linux P0だけを提供し、managed DNS/HTTPS proxy以外をdefault denyにする。
### 適用しなかった重要な判断

- TLS本文検査、credential broker、非HTTP TLS、OS横断profileを本Taskに含めない。
