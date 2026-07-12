# Governance Plane Schema catalog — draft-v0

Workspace Security Policy、CASB Rule、Credential scope、Egress audit、temporary Grant、恒久Policy Revision、対応Authorityを所有する。

## P0

以下は`draft-v0:r1`として追加済みである。

| Schema | 固定する内容 |
|---|---|
| `workspace-security-profile.schema.json` | Workspace Policy Binding、Credential scope、network capability、pending Revision |
| `casb-policy.schema.json` | Policy ID/version、target key、Rule priority、match、allow/deny、deny-overrides |
| `casb-rule.schema.json` | destination/protocol/port、method/path/body、classification、Credential、TTL/use limit |
| `egress-request-binding.schema.json` | canonical request digestを構成する全field |
| `egress-attempt.schema.json` | Workspace/Task/Agent provenance、binding、classification、Rule結果 |
| `egress-challenge.schema.json` | Workspace、binding、reason、grant/authority eligibility、expiry |
| `egress-rule-decision.schema.json` | matched Rule、Policy version、allow/block、reason |
| `egress-capture-manifest.schema.json` | captured/redacted range、truncation、inspection limit、retention/pin |
| `outbound-transaction.schema.json` | intent、forwarded、completed、failed、outcome unknown |
| `policy-grant.schema.json` | Workspace-scoped temporary Rule、source Decision、limits、activation |
| `policy-revision.schema.json` | Job、fixed candidate、Proposal、Authority、CAS、pending/active、ACK |
| `policy-revision-job.schema.json` | Finding join、input snapshot、Decision前candidate固定、lease/retry |
| `governance-authority-request.schema.json` | Grant / RevisionをAuthorityへ提示するimmutable payload |
| `governance-authority-decision.schema.json` | responder、approve/deny、rationale、decided time |
| `policy-agent-input.schema.json` | Grant/Revision評価の固定input snapshot |
| `egress-audit-input.schema.json` | Attempt、Rule Decision、Capture Manifest、Policy Bindingの固定snapshot |

## P1

| Schema | 固定する内容 |
|---|---|
| `dns-resolution.schema.json` | Workspace-scoped DNS provenance、TTL、resolved IP |
| `credential-binding.schema.json` | broker sentinel、Workspace、provider principal、resource scope |
| `policy-regression-result.schema.json` | replay dataset、見逃し、過剰block、coverage |
| `policy-candidate.schema.json` | Revision Jobに固定するcandidate ref/digest、base version、fixed timestamp |
| `policy-finding.schema.json` | benign / bypass / suspicious / insufficient evidence |

## 現在のAPI adapter

Grant Decision、Egress Review、Policy Revision Decision、Authority Decision Toolは`../api/`の合成bundleに含まれる。candidate Rule本文はStructured Outputへ含めず、`casb-policy.schema.json`で別に検証する。
