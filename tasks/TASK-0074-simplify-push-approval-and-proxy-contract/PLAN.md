---
task_id: "TASK-0074"
change_class: "safety_contract"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "sol-high"
approved_dev_profile_reason: "認可単位、credential境界、残余リスクとfail-closed契約を変更するsecurity boundaryであるため。"
approved_dev_profile_risk_signals:
  - "authorization-security-boundary"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-02T09:05:23Z"
classification_approved_by: "main-agent-sol-high"
classification_approved_at: "2026-08-02T09:05:23Z"
classification_approval_reason: "製品成果物を変更せず、push認可、credential proxy、受容する侵害リスクと後続Task順序だけを変更するため。"
safety_contract_version: 2
safety_contract_planned_paths:
  - "docs/development/development-agent-harness.md"
safety_contract_generated_paths: []
---

# TASK-0074 PLAN

## AC対応

TASKの条件本文を再掲せず、`planning input packet`のAC-IDに設計を対応させる。

| AC-ID | 設計判断 | 変更パス | 実施順序 | 失敗時の扱い |
|---|---|---|---|---|
| AC-1 | `push grant`をrepository単位のone-shot authorizationへ置換し、上流試行の開始前に原子的消費する境界を記述する。 | `docs/development/development-agent-harness.md` | 1 | 必須束縛又は消費順序を文書で明確化できなければ、旧契約を残したままMainへ差し戻す。 |
| AC-2 | repository内の内容差替えは残余リスクとして明示し、越境・再利用・期限後・REST転用を拒否する責務をGitHub App権限とgrant境界へ分離する。 | `docs/development/development-agent-harness.md` | 1 | 許容リスクと拒否境界が混同される場合は、契約変更を進めない。 |
| AC-3 | proxyをOpaque capability検証とcredential置換を所有する薄いstream転送境界へ再定義し、必要最小のtransport安全性と削除対象を対にして記述する。 | `docs/development/development-agent-harness.md` | 2 | provider意味検査又は本文解析が残る、又は維持する境界が欠ける場合は差分を不成立とする。 |
| AC-4 | UIの認可文言と参考表示を分離し、旧Taskの履歴性、後続移行、および次の実VPS vertical sliceの優先順を設計書で固定する。 | `docs/development/development-agent-harness.md` | 3 | 旧証跡を書換える必要が生じる、又は後続scopeが分割される場合はMainへ再判断を求める。 |
| AC-5 | 設計書のみを更新する安全契約として固定し、既存証跡、新規機械check/field/wrapper、製品差分を追加しない。 | `docs/development/development-agent-harness.md` | 4 | scope確認で予定外パス又は製品変更が見つかれば実装を停止して再分類する。 |

## 根拠・境界

- 唯一の要求根拠は`TASK.md`の`Planning input packet`とREF-1〜REF-4である。依存はN/Aであり、`dependency-ready reconciliation`の追記は不要である。
- 本件は実装、設定、Schema、依存、生成物、外部挙動を変更しない`safety_contract`である。設計文書が将来の認可根拠を置換するが、旧TASK-0070〜0073の公開証跡は履歴として不変に保つ。
- `backlog.yaml`とTask/QA_PLAN等の証跡はMain管理であり、本PLANのproduct planned pathsには含めない。DEV相当の文書更新対象は`docs/development/development-agent-harness.md`だけである。
- 独立PLAN review、新しいcheck/format/receipt/field/wrapper、または互換層は作らない。QA_PLANはTASK-firstで独立に作成し、Mainが意図・scope・受け入れ経路を確認する。

## 補足設計

### 代替案と不採用理由

- ref/SHA/manifest、Git wire本文、provider JSON/endpointを照合し続ける案は、必要な権限境界を強めずproxyにprovider意味を重複実装するため不採用とする。
- reference情報を認可根拠に残すUI案は、repository一回という承認単位を誤認させるため不採用とする。
- 旧Task証跡を書換える移行案と、将来用wrapperを残す段階移行案は、履歴不変性と削除優先に反するため不採用とする。

### 責務・境界・不変条件

- grantはagent instance/UID、workspace、exact repository、短TTL、revoke、一回消費に束縛する。消費は成功後ではなく`git-receive-pack`の上流試行開始前であり、失敗又は結果不明でも再利用しない。
- proxyが維持するのはUnix peer identity、host allowlist、Opaque handleのsubject/provider/repository/TTL/use/revoke、実credential置換、TLS CONNECT/ローカルCA、resource上限、secret-free auditである。通常経路はHTTP framing、hop-by-hop header、secret境界に必要な最小処理以外を解釈しない。
- pushだけがrepositoryと`git-receive-pack`を最小分類できる。本文は解析せず、Git read、GitHub REST、OpenAI APIもprovider protocolの意味を再実装しない。

### 移行・互換性

- 設計書内のmanifest/digest/ref/SHA/old-SHA照合、strict provider検査、buffer/response制約、重複layerを後続製品Taskで削除する旧契約として明記する。本Taskでは実装、移行、live E2Eを行わない。
- 承認UIは主文言をrepositoryへの次のpush一回とし、branch/commit/ref/SHAは参考表示に限定する。次の製品Taskは薄いproxyと承認後pushを実VPSで縦断確認する単一vertical sliceとして扱う。

## 変更予定

| パス | 変更内容 |
|---|---|
| `docs/development/development-agent-harness.md` | push承認、proxy/provider flow、UI、fail-closed/監査、段階導入、検証表、後続Task分割をrepository one-shot grantと薄いproxyの契約へ整合更新する。 |

## 実装手順

1. push承認の用語、承認対象、state/consume時点、UI文言、残余リスクをrepository one-shot契約へ置換する。
2. provider flowとproxy契約をstream転送へ簡素化し、維持境界と削除する意味検査・buffer・重複layerを明記する。
3. fail-closed、監査/失効、段階導入、検証表、後続Taskの説明を新しい境界と実VPS vertical E2E優先に揃える。
4. 文書内の旧契約残存を検索し、許可パス、生成物なし、既存TASK-0070〜0073不変を確認してMainへ渡す。

## 検証計画

- `rg`でmanifest/ref/SHA/provider parser/response buffer等の旧認可・意味検査が、履歴又は明示された削除対象以外に残らないことを確認する。
- 設計書のAC対応、one-shot消費、残余リスク、proxy境界、UI表示、後続vertical sliceをTASK-first QA_PLANと照合する。
- 安全契約の対象検査、`git diff --check`、許可path確認を実行する。製品`make check`や製品PASS証跡では代替しない。DEV開始前にMainが`make task-preflight TASK=TASK-0074`を実行する。
- 実VPS、Passkey、Tailscale、GitHub App、proxy、grantのlive確認は本TaskでPASSにせず、次の製品Taskの`live-e2e`として残す。

## 未解決事項

- なし

## main Agentレビュー

- [x] TASKの全AC-IDへ設計判断、パス、順序、失敗時の扱いを対応させ、条件本文を複製していない。
- [x] 設計観点と代替案を検討している。
- [x] QA_PLANがTASK-firstで独立作成されている。
- [x] `dependency-ready reconciliation`と完了経路preflightが完了している。
- [x] 安全契約差分の開始を承認した。

安全契約変更でv2契約を選ぶ場合は、コメントを外して`safety_contract_version: 2`と予定パス・生成パスの配列を記録し、DEV前に`make task-preflight TASK=TASK-0074`を実行する。変更しない種別は空配列とし、通常の予定パスと生成パスを重複させない。MainはPLAN/QA_PLANの意図・スコープ・受け入れ経路を確認し、分類承認をフロントマターへ記録する。分類変更時はTask、PLAN、QA_PLANを再承認し、承認者と時刻を更新する。
