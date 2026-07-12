# Governance Plane Schema catalog — draft-v0

Workspace Security Policy、CASB Rule、Credential スコープ、Egress audit、temporary Grant、恒久Policy Revision、Authorityが判断する対象と要否を所有する。人間との送受信は所有せず、すべてControl PlaneのAuthority Gatewayを経由する。

## P0

以下は`draft-v0:r1`として追加済みである。

| Schema | 固定する内容 |
|---|---|
| `workspace-security-profile.schema.json` | Workspace Policy Binding、Credential スコープ、ネットワーク capability、pending Revision |
| `casb-policy.schema.json` | Policy ID/バージョン、target key、Rule priority、match、allow/deny、deny-overrides |
| `casb-rule.schema.json` | destination/プロトコル/port、method/path/body、classification、Credential、TTL/use limit |
| `egress-request-binding.schema.json` | canonical リクエスト ダイジェストを構成する全フィールド |
| `egress-attempt.schema.json` | Workspace/Task/Agent provenance、binding、classification、Rule結果 |
| `egress-challenge.schema.json` | Workspace、binding、reason、grant/authority eligibility、expiry |
| `egress-rule-decision.schema.json` | matched Rule、Policy バージョン、allow/block、reason |
| `egress-capture-manifest.schema.json` | captured/redacted range、truncation、inspection limit、retention/pin |
| `outbound-transaction.schema.json` | intent、forwarded、`completed`、`failed`、outcome unknown |
| `policy-grant.schema.json` | Workspace-scoped temporary Rule、source Decision、limits、activation |
| `policy-revision.schema.json` | Job、fixed candidate、Proposal、Authority、CAS、pending/active、`ACK` |
| `policy-revision-job.schema.json` | Finding join、入力 スナップショット、Decision前candidate固定、lease/再試行 |
| `governance-authority-request.schema.json` | Grant / RevisionをAuthorityへ提示するimmutable ペイロード |
| `governance-authority-decision.schema.json` | responder、approve/deny、rationale、decided time |
| `policy-agent-input.schema.json` | Grant/Revision評価の固定入力 スナップショット |
| `egress-audit-input.schema.json` | Attempt、Rule Decision、Capture Manifest、Policy Bindingの固定スナップショット |

## P1

| Schema | 固定する内容 |
|---|---|
| `dns-resolution.schema.json` | Workspace-scoped DNS provenance、TTL、resolved IP |
| `credential-binding.schema.json` | broker sentinel、Workspace、provider principal、resource スコープ |
| `policy-regression-result.schema.json` | replay dataset、見逃し、過剰block、coverage |
| `policy-candidate.schema.json` | Revision Jobに固定するcandidate ref/ダイジェスト、base バージョン、fixed timestamp |
| `security-incident.schema.json` | Finding、risk、containment、Revisionを結ぶIncident lifecycle |
| `incident-risk-assessment.schema.json` | Rule floor、Reviewer推奨、effective risk、Human gate |
| `incident-containment.schema.json` | Grant revoke、Egress制限、Workspace freeze、固定graph上の祖先・発生元・子孫Task suspend集合 |
| `incident-authority-request.schema.json` | High/Critical Incident dispositionとTask resumeの人判断要求 |
| `incident-authority-decision.schema.json` | keep `suspended` / resume / cancelの認証済み判断 |
| `policy-finding.schema.json` | benign / bypass / suspicious / insufficient evidence |

## 現在のAPI adapter

Grant Decision、Egress Review、Policy Revision Decision、Authority Decision Toolは`../api/`の合成bundleに含まれる。candidate Rule本文はStructured Outputへ含めず、`casb-policy.schema.json`で別に検証する。
