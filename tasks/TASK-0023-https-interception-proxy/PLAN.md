---
task_id: "TASK-0023"
status: draft
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "sol-high"
approved_by: ""
approved_at: ""
planned_implementation_files: 16
planned_implementation_lines: 1200
estimate_points: 8
---

# TASK-0023 PLAN

## 受け入れ条件の具体化

| 条件 | 観測方法 | 根拠 |
|---|---|---|
| TLS/canonicalization | Task-local CA trustがWorkspace identityとread-only runtime allowlistへ限定配置され、host/別Workspaceへ漏れず、HTTP/1.1/2同値requestが同じcanonical formになる | docs/07 §4 |
| rejection | redirect/rebinding/framing/CONNECT/Upgrade/streaming負例でouter connectなし | docs/07 §§4-5 |
| digest binding | request/body digestとworkspace/task/contract/policy bindingが0016へ届く | docs/07 §§6-7 |
| audit barrier | persist failure時は外側接続なし、allow時はintent_committed先行 | docs/07 §7 |

## 関連Wikiと判断

- docs/07 §§3-4,6-7、docs/08 §4、docs/11 §5、docs/12 §5、docs/13 §§2,4。依存: TASK-0006、TASK-0008、TASK-0009、TASK-0016、TASK-0022。

## 設計

### 選択案

Go HTTPS proxyがWorkspaceごとのTask-local leaf CAを管理し、TASK-0009のWorkspace identityにbindしたtrust bundleをTASK-0022のread-only runtime allowlistだけへworkload起動前にmountしてTLSを終端する。HTTP/1.1/2 parserはmethod/authority/path/query/header/bodyをcanonical requestへ変換してdigestを作る。validatorが危険なframing/redirect/rebinding/streamingを拒否し、0016の決定後もGovernance audit barrierをcommitするまでdialerを解放しない。

### 代替案と不採用理由

- TLS pass-through/CONNECT: policy inputと本文検査前barrierを保証できないため不採用。
- version別policy parser: 意味差とbypassを作るため不採用。
- credential broker/高度分類同居: P0 transport boundaryを超えるため不採用。

### 責務と境界

- TASK-0017はroute、当TaskはTLS/HTTP/digest/audit barrier、TASK-0016はrule decision、TASK-0018はgrant workflowを所有する。

### 不変条件

- outer connectionはcanonical validation、policy decision、永続audit barrierの後だけに存在する。秘密の生値を監査しない。unsupported/ambiguous protocolはdenyする。

### 失敗時・移行・互換性

CA/parse/audit/policy失敗はfail-closed。外部到達可能性のあるcrashはmanifest incomplete/transaction outcome_unknownとして照合対象にする。P0に旧pass-through互換はない。

## 変更予定

| ファイル群 | 種別 | 概算行数 | 変更内容 |
|---|---|---:|---|
| core/internal/egress/{https_proxy,tls_ca,http_canonical,request_digest}.go | implementation | 460 | termination/canonicalization/digest |
| core/internal/egress/{request_validator,redirect,audit_barrier,dialer}.go | implementation | 260 | rejects/commit-before-dial |
| core/internal/governance/egress_client.go | implementation | 90 | binding/decision contract |
| core/internal/egress/*_test.go | test | 270 | HTTP1/2 and negative tests |
| schemas/draft-v0/governance-plane/{egress-attempt,outbound-transaction}.schema.json | schema | 70 | audit binding |
| docs/runtime/https-proxy.md | documentation | 50 | P0 support contract |

実装対象16ファイル・1,200行。`ceil(16/3)=6`、`ceil(1200/200)=6`のため8pt。

## 実装手順

1. Workspace identityとfilesystem profileへbindしたCA/trust配置、TLS、canonical request/digest contractを実装し、host・別Workspaceへtrustが漏れないconsumer testを追加する。
2. HTTP/1.1/2 validationと拒否行列を追加する。
3. 0016 decisionとaudit barrier/dialer gateを接続する。
4. protocol and persistence negative testsを実行する。

## 検証計画

- workload起動前のtrust配置、host・別Workspaceへのtrust非漏洩、HTTP/1.1/2 equivalence、redirect/rebinding、CL/TE・chunk/trailer ambiguity、CONNECT/Upgrade/WebSocket、streaming、audit-store outage、outer dial orderingを検証する。
- Go tests、schema/tabletop negative mutation、`make check`、`git diff --check`。

## 未解決事項

- 対象HTTP/2 libraryのframe正規化境界はDEV開始前に固定し、曖昧な入力は互換性より拒否を優先する。

## main Agentレビュー

- [ ] 受け入れ条件が検証可能である。
- [ ] 設計観点と代替案を検討している。
- [ ] QA計画を作成できる。
- [ ] 見積もりが規則どおりである。
- [ ] DEV開始を承認した。
