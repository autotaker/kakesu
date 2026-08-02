---
task_id: "TASK-0076"
change_class: "product"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "sol-high"
approved_dev_profile_reason: "Unix peer/credential/TLS 境界を保ったまま streaming proxy、永続 one-shot grant、Passkey UI、systemd/VPS 配置を同一縦断経路で置換する security・streaming・persistence・UI 横断変更であるため。"
approved_dev_profile_risk_signals:
  - "authorization-security-boundary"
  - "credential-and-tls-boundary"
  - "streaming-resource-boundary"
  - "durable-one-shot-race"
  - "trusted-ingress-and-passkey-ui"
  - "live-vps-external-integration"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-02T10:26:58Z"
classification_approved_by: "main-agent-sol-high"
classification_approved_at: "2026-08-02T10:26:58Z"
classification_approval_reason: "proxy、認可state/UI、systemd/configure、外部観測可能な通信とpush挙動を変更する製品変更であるため。"
# safety_contract_version: 2
# safety_contract_planned_paths: []
# safety_contract_generated_paths: []
---

# TASK-0076 PLAN

## 根拠・分類・実行境界

唯一の期待値根拠は `TASK.md` の `Planning input packet`（REF-1〜3 を含む）である。TASK-0074 の安全契約は、その packet が固定する設計参照であり、旧 TASK 証跡を更新する理由にはしない。現行の `tools/dev-agent-harness/` production、test、configure/install、deployment を置換するため `change_class` は `product` とする。Kakesu 本体 runtime/Schema、既存 Task 証跡、Lap/JSONL、Codex 実認証情報例外は対象外である。

本 Planner の役割設定は `planner` / Terra medium / `workspace-write`、この実行環境で観測した sandbox は `workspace-write` である。PLAN 証跡だけを編集し、実装、承認、Git 操作、commit/merge、他 Agent 起動は行わない。DEV は Main が本 PLAN と独立した TASK-first `QA_PLAN.md` を承認した後にだけ `sol-high` で開始する。candidate は製品差分を一回だけ固定し、REVIEW と QA は同じ `candidate_commit` を独立に評価する。Main のみが stage、commit、completion gate、main への no-ff 統合を所有する。

候補の許可範囲は `tools/dev-agent-harness/` 全体だけである。1,000 行という目標は削除による小型化の指標にとどめ、AC-1〜7 を満たす単一 vertical slice の完結性を優先する。追加は薄い転送、repository grant、Approval UI、必要な deployment/runbook とその検証だけに限り、旧形式を変換する wrapper、旧 parser を温存する adapter、休眠 package、将来用 parser、新形式専用 check は作らない。

## AC対応

TASK の条件本文を再掲せず、`planning input packet` の AC-ID に設計を対応させる。

| AC-ID | 設計判断 | 変更パス群 | 実施順序 | 失敗時の扱い |
|---|---|---|---|---|
| AC-1 | `approvalmanifest` と digest/ref/SHA 依存を削除し、request/decision/grant の durable record を agent instance/Unix UID、workspace、正規化済み完全一致 repository、TTL、未使用、revoke 世代へ束縛する。push 試行だけが compare-and-consume を所有し、競合した一件以外は上流へ出さない。 | `internal/approvalmanifest/` を削除、`approvalstate/`・`approvalchallenge/`・`approvaldecision/` と approval command/service/test を repository grant model へ置換 | 1 | migration、restart、clock、persistence、revoke 又は CAS failure は grant 発行/消費成功を推測せず fail-closed。消費後の失敗は再使用せず新 request/decision を要求する。 |
| AC-2 | 通常 Git read、GitHub REST、OpenAI は peer/capability/host/repository/credential 判定を一度だけ行った後、method/path/query/body と status/headers/body を未解釈・非全量 buffer・backpressure 付きに relay する。 | `egresspolicy/`、`egresstransaction/`、`brokerexchange/`、`upstreamforwarder/`、`brokerhttp/`、`egressservice/`、関連 test/config/README | 2 | 上流/クライアントの cancellation、header/resource budget、TLS/credential/transport failure は stream を閉じ、直接通信・retry に fallback しない。非2xx・非JSON・任意 Content-Type・1 MiB 超は上流のまま返す。 |
| AC-3 | push は exact repository と `git-receive-pack` だけを最小分類し、pkt-line、本文、ref、SHA、force/delete を読まない。grant の atomic consume を upstream dial/body write より前に同じ request path へ統合し、REST/read/API との capability を分離する。 | 薄い proxy の production/test、`gitcredential/`、`capability/`・`capabilitycontrol/`・`connectsession/` と repository grant integration | 3 | repository/subject/workspace/operation mismatch、expiry、revoke、再使用、REST 転用は consume せず上流到達前に拒否する。consume 後の disconnect、timeout、upstream error、結果不明は consumed のままにする。 |
| AC-4 | Approval backend は localhost trusted ingress だけを受け、Tailscale Serve identity と Grant による到達制約、CSRF/session、Passkey RP ID/origin/challenge/UV を一つの repository decision に AND 接続する。主文言は「このrepositoryへの次のpush一回」、reference は認可外と明示する。 | `cmd/dev-agent-approval/`、approval HTTP/UI/state integration、systemd/config/example、README/runbook、関連 test | 4 | header の直受け、tailnet identity/Grant 不一致、CSRF/session/Passkey failure、expired/replayed challenge、通知 failure は decision/grant/push を作らない。Funnel/public listener と自動 push 再開を導入しない。 |
| AC-5 | 旧 provider semantics と重複 evaluation を、test を含めて削除する。残す薄い一箇所の責務は peer/host/capability/repository/grant/credential 判定と framed stream relay である。 | 旧 package と対応 test/README/config から `approvalmanifest`、endpoint/JSON/model/upload-pack parser、response 2xx/JSON/Content-Type/1 MiB/full-buffer、Policy→Transaction→Exchange→Forwarder chain を削除 | 5 | 旧用語・package import・テスト fixture・wrapper/dead code が残る、又は provider content を再び authorization 根拠にする場合は candidate を不成立とする。 |
| AC-6 | approved live environment で Agent UID が実 credential を読めないまま Git pull、限定 GitHub REST、OpenAI、phone decision 後 push と negative cases を一続きに実施し、試験 repository/branch/deployment を cleanup/rollback する。 | `deploy/`、`configure.ac`、`Makefile.in`、config example、README/runbook と live harness（秘密非保存） | 6 | live dependency 未ready、credential exposure、identity/Passkey/Serve failure、cleanup/rollback 不明、negative case 未実施は `live-e2e` を pending/blocked のままにする。hermetic PASS で AC-6 を置換しない。 |
| AC-7 | source/build/install layout、systemd/config/runbook と新しい runtime graph を同時に整合させ、race/integration と package install を candidate-bound に検証する。failure は provider/environment/auth/implementation に分類する。 | `configure.ac`、`Makefile.in`、`config/`、`deploy/`、README、全関連 production/test | 7 | configure/build/install/test/runbook mismatch は candidate を進めない。live failure を実装不具合と決めつけず、証拠と分類を QA_RESULT/HANDOVER へ secret-free に記録する。 |

## 責務統合・削除境界

新経路は、(1) Unix socket peer から trusted subject を得る、(2) opaque capability と exact host/repository/workspace/TTL/revoke を一度だけ検証、(3) provider credential を broker 側だけで置換、(4) push なら same authority path で grant を upstream 試行前に原子的 consume、(5) hop-by-hop framing と resource limits を除き双方向に stream relay、という順序に統合する。通常経路は provider API、Git wire、body、response meaning を所有しない。GitHub App installation の repository/permission 範囲は上流安全境界として維持し、push grant を GitHub REST/API/OpenAI/read へ転用しない。

維持する不変境界は、Unix peer/agent instance/UID と workspace の照合、opaque capability、exact host と exact repository、broker-only credential と local CA private material、TLS CONNECT/local CA、timeout/concurrency/request-response header の上限、cancellation/backpressure、secret-free audit、deny-all Agent network と trusted ingress である。監査は subject/repository/decision/grant lifecycle/結果区分だけを記録し、capability/token/Authorization、Passkey assertion、本文、branch/ref/SHA は記録しない。

削除する抽象層は、`approvalmanifest` とその canonical digest/state binding、`egresspolicy` の provider-specific semantics、`egresstransaction`、`brokerexchange`、`upstreamforwarder` が繰返す policy/prepare/exchange/response validation である。併せて GitHub repository endpoint parser、strict OpenAI JSON/model/endpoint checks、Git upload-pack request/response checks、push pkt-line/body/ref/SHA/force/delete logic、response 2xx/JSON/Content-Type requirement、response full buffering/fixed 1 MiB cap を production と対応 test から取り除く。設定の provider meaning allowlist（OpenAI model など）は host/repository/capability scope に必要でない限り削除し、repository 限定は capability/grant と GitHub App installation に一元化する。

## 実装順序

1. 現行 package/config/deploy の依存を削除先から切り、repository-only approval state machine を定義する。request は repository と subject/workspace のみを durable に保存し、decision は Tailscale identity + Passkey UV を受け、grant は TTL/revoke/unused を持つ。旧 state の読み取りや変換を持たない。
2. egress runtime graph を単一の authorization-and-stream relay に再構成する。HTTP body/response body は `io.Reader`/`io.Copy` 相当で所有権・close・cancel を扱い、上流 status/header/body を stream する。resource protection は semantic body cap ではなく connection/header/timeout/concurrency に限定する。
3. Git credential と relay の push discrimination を接続する。exact GitHub repository と `git-receive-pack` だけを classification input にし、grant CAS 成功後に限り upstream attempt を開始する。通常 Git read/GitHub/OpenAI と grant authority を混ぜない。
4. localhost-only Approval UI を作り、Tailscale Serve が付与する trusted identity を backend allowlist と照合し、Passkey UV/CSRF/session/challenge を decision に接続する。reference の表示を必要最小にし、認可文言との混同を UI/test で防ぐ。
5. 旧 package、imports、tests、documentation/config fields、build targets を一括削除し、残った single runtime graph と state/UI/test へ整理する。互換 wrapper と parser 保存はしない。
6. configure/install/systemd/config example/runbook を実装後の binary、socket、state、trusted ingress、rollback/cleanup 手順へ一致させる。`make install DESTDIR=...` は配置のみで、秘密、tailnet、外部 service、enable/start を変更しない。
7. hermetic/race/integration を通し、candidate を一度固定して独立 REVIEW/QA に渡す。Main の承認後、dependency-ready reconciliation を経た approved VPS で live-e2e を行い、失敗分類と cleanup/rollback 結果を secret-free に記録する。

## 変更予定

| パス群 | 変更内容 |
|---|---|
| `tools/dev-agent-harness/internal/approvalmanifest/` | package と test を削除する。旧 manifest の compatibility reader/wrapper は作らない。 |
| `tools/dev-agent-harness/internal/approvalstate/`, `approvalchallenge/`, `approvaldecision/` | manifest/digest record を repository request/decision/grant、Passkey-backed decision、atomic consume/revoke/expiry/persistence に置換し、race/restart/failure tests を同じ契約へ更新する。 |
| `tools/dev-agent-harness/internal/{egresspolicy,egresstransaction,brokerexchange,upstreamforwarder,brokerhttp,egressservice}/` | 重複 policy/transaction/exchange/forwarder と provider meaning validation を削除し、one-pass authorization と bounded streaming relay に統合する。 |
| `tools/dev-agent-harness/internal/{capability,capabilitycontrol,connectsession,brokerlistener,peerbinder,upstreamtransport,providercredentials,brokercredentials,proxyca,gitcredential}/` | peer/host/capability/repository/credential/TLS/resource boundary を維持する最小 API に接続し、push grant と non-push capability の分離を実装・検証する。 |
| `tools/dev-agent-harness/cmd/`, `internal/command/` | egress、approval、broker、launcher の production wiring を削除後の single graph/UI に揃え、未実装 scaffold の振舞いを残さない。 |
| `tools/dev-agent-harness/config/`, `deploy/`, `configure.ac`, `Makefile.in` | repository/grant/approval runtime に必要な設定例、localhost approval service、socket/systemd、install/live hook を同期し、provider semantic allowlist と旧 artifact を除去する。 |
| `tools/dev-agent-harness/README.md` | operator/Agent setup、Tailscale Serve/Grant・Passkey approval、one-shot failure/retry、secret-free audit、VPS rollout、rollback/cleanup、local と live の検証境界を実装に一致させる。 |

## 検証計画

- hermetic unit/integration で、normal Git read/GitHub REST/OpenAI の arbitrary method/path/query/body と non2xx/nonJSON/arbitrary content type/1 MiB超 response が未解釈 streaming され、backpressure/cancellation/close/header/connection/timeout limits が維持されることを確認する。
- focused race/restart test で、同一 repository の並行 push が一件だけ consume して upstream attempt へ進むこと、別 repository/subject/workspace、reused/expired/revoked grant、REST misuse、consume後 upstream failure/response loss が upstream 前拒否又は再利用不能となることを確認する。本文、pkt-line、ref、SHA、force/delete を判定していないことも test scope と source inventory で確認する。
- UI/integration で localhost trusted ingress、Serve identity allowlist、Tailscale Grant 前提、CSRF/session、Passkey RP/origin/challenge/UV、replay/expiry、主文言と reference disclaimer を確認する。実 Tailscale/Passkey は live-e2e であり fixture の PASS はその代替ではない。
- `./configure && make && make check && make install DESTDIR=...`、focused race/integration、`git diff --check`、candidate scope を実行する。配布/install で config/systemd/runbook と binary/layout が一致し、install が secret/external state を作らないことを確認する。
- live-e2e は dependency-ready 後だけ、approved Sakura VPS で Agent credential non-readability、Git pull、限定 GitHub REST、OpenAI、phone approval 後の push と negative cases を行う。試験 branch/commit/deployment の cleanup 又は rollback を観測して AC-6 を判定する。未実施は `pending`/`blocked` のままとし、`evidence-review` 又は focused rerun の PASS で代替しない。

## live 依存と dependency-ready reconciliation

ローカル実装基盤は ready であり、実装と hermetic acceptance は開始できる。一方、VPS 接続先、operator/Tailscale identity、Passkey enrollment、repository 限定 GitHub App installation、OpenAI credential、使い捨て test branch、cleanup authority は pending である。PLAN/QA_PLAN/コード/config/repository に秘密値を要求・保存・出力しない。

live を開始する前に Main は秘密値そのものではなく、VPS 識別子、tailnet operator identity、target repository、test branch、実施時刻、cleanup/rollback authority と方法だけを reconciliation 記録に固定し、REF-1〜3 と本 PLAN/QA_PLAN の scope・AC・design・live case との差分を判定して再承認する。これらが未確定、実 credential の Agent 非可視性が未確認、または安全な cleanup が確認できない場合、DEV の local candidate は進められても live QA は blocked とし、完了にしない。

## 不採用案

- 削除、thin proxy、approval、live E2E を別 Task に分ける案は、TASK-0074 と input packet の single vertical slice 契約に反するため不採用とする。
- ref/SHA/branch/force/delete、Git body/pkt-line、provider endpoint/JSON/status/Content-Type を再導入する案は、provider semantics の重複と誤った保証を復活させるため不採用とする。
- 旧 manifest/state parser の migration reader、compatibility wrapper、future parser、追加の形式 check を残す案は、削除優先と no-dead-code 条件に反するため不採用とする。
- live dependency がない間に credential、host、tailnet、Passkey、repository 値を推測して記録する案、または mock/PASS で live-e2e を置換する案は不採用とする。

## 未解決事項

- live dependency は pending。dependency-ready reconciliation と Main の再承認なしに AC-6 の live-e2e を開始又は PASS にしない。

## main Agentレビュー

- [x] TASK の全 AC-ID に設計判断、パス群、順序、failure handling が対応し、条件本文を複製していない。
- [x] product 分類、`sol-high` 選定、single vertical slice、削除優先、許可範囲が妥当である。
- [x] 独立した TASK-first `QA_PLAN.md` があり、case ごとの `evidence-review` / `focused-rerun` / `live-e2e` と pending dependency を分離している。
- [x] dependency-ready reconciliation と完了経路 preflight を確認し、live 未ready を hermetic PASS で代替しない。
- [x] ローカルDEV開始を承認した。live QA開始はdependency-ready reconciliationまで承認しない。

Main は PLAN/QA_PLAN の意図・scope・acceptance route を確認し、分類承認と開始承認をフロントマターへ記録する。分類又は live dependency の前提が変わる場合は、TASK、PLAN、QA_PLAN を再承認する。
