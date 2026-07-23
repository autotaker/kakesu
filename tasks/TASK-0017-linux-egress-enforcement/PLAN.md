---
task_id: "TASK-0017"
status: draft
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "sol-high"
approved_by: ""
approved_at: ""
planned_implementation_files: 14
planned_implementation_lines: 1050
estimate_points: 8
---

# TASK-0017 PLAN

## 受け入れ条件の具体化

| 条件 | 観測方法 | 根拠 |
|---|---|---|
| identity/firewall route | TASK-0009のworkspace network identityから許可endpoint以外のconnectが失敗 | docs/07 §§2-3 |
| DNS capture | A/AAAAのみ正規化、DoH/生IP/rebinding負例 | docs/07 §5 |
| HTTPS transport handoff | identityがTASK-0023 listenerへ渡り、HTTPS解釈が重複しない | docs/07 §§3-4 |
| fail closed | adapter/proxy/governance停止中の転送なし | docs/07 §3 |

## 関連Wikiと判断

- docs/07, 11, 12, 13。依存: TASK-0009、TASK-0016、TASK-0023。namespace/cgroup生成はTASK-0009、HTTPS parsing/canonicalizationはTASK-0023の正本とする。

## 設計

### 選択案

egress bridgeはTASK-0009が作成済みのWorkspace network identityへfirewall routeを接続し、DNS listenerとTASK-0023 proxy listenerだけをallowする。DNS routeは統治へqueryを渡す。HTTPSはidentityをTASK-0023へ渡すだけで、canonical metadataやouter connection判定を持たない。

### 代替案と不採用理由

- 環境変数proxyだけ: direct socketを防げない。
- TLS MITM/canonicalizationを同時実装: TASK-0023のP0 HTTPS boundaryと検証が重複するため不採用。

### 責務と境界

- TASK-0009: namespace/cgroup/process identity/lifecycle。
- egress bridge: identity-to-DNS/proxy/firewall routesとroute lifecycle。
- TASK-0023: HTTPS canonicalization、governance request、outer forwarding barrier。TASK-0016: policy判断。

### 不変条件

- sandboxから外部へ直結不可、DNS/TASK-0023 proxy以外はdeny、Workspace identityはTASK-0009由来、route障害はdeny。HTTPSのallow前outer connection禁止はTASK-0023が保証する。

### 失敗時・移行・互換性

TASK-0009 identityが存在しない、またはfirewall route要件が満たせない環境は通信を開始しない。route途中/cleanup失敗は残存ruleを監査し回収する。非Linuxはunsupportedとして明示する。

## 変更予定

| ファイル群 | 種別 | 概算行数 | 変更内容 |
|---|---|---:|---|
| core/internal/egress/{firewall_route,workspace_route,dns_route,lifecycle}.go | implementation | 420 | identity配線・deny route・回収 |
| core/internal/egress/{dns_proxy,https_transport,proxy_client}.go | implementation | 310 | DNS捕捉・TASK-0023 transport handoff |
| core/internal/{app,execution}/**/*_test.go | test | 180 | isolation/bypass/failure |
| scripts/{linux-egress-fixture,verify-linux-profile}.sh | test tooling | 80 | Linux integration fixture |
| docs/runtime/linux-profile.md | documentation | 60 | 前提/非対応機能 |

実装対象14ファイル・1,050行。`ceil(14/3)=5`、`ceil(1050/200)=6`のため8pt。

## 実装手順

1. TASK-0009 identity contractを検証し、firewall route lifecycleを作る。
2. managed DNS routeとTASK-0023 HTTPS transport handoffを接続する。
3. IPv4/IPv6/bypass/rebinding/障害の隔離試験を追加する。
4. Linux専用の起動診断を文書化し`make check`へ統合する。

## 検証計画

- direct TCP/UDP、DoH/DoT/QUIC、raw/ICMP、生IP/private/metadata、IPv6、TASK-0023 listener停止、cleanup/restart。HTTPS canonicalization/outer audit barrierはTASK-0023の独立試験で検証する。
- Linux integration fixture、Go tests、schema/tabletop、`make check`。

## 未解決事項

- CIでTASK-0009 identityとfirewall routeを許可するLinux runner capabilityをQA_PLANで固定する。

## main Agentレビュー

- [ ] 受け入れ条件が検証可能である。
- [ ] 設計観点と代替案を検討している。
- [ ] QA計画を作成できる。
- [ ] 見積もりが規則どおりである。
- [ ] DEV開始を承認した。
