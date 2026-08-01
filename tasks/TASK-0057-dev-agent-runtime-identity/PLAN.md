---
task_id: "TASK-0057"
change_class: "product_change"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "luna-xhigh"
approved_dev_profile_reason: "新規 internal/runtimeidentity の限定した immutable resolver、Linux/非Linux adapter、hermetic fixture、既存config/example/READMEの最小同期に閉じる。service composition、capability、config version、dependency、repository check、実OS操作を変更しないため。"
approved_dev_profile_risk_signals: []
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T17:28:43Z"
planning_reviewed_by: ""
planning_review_decision: "pending"
planning_reviewed_at: ""
classification_approved_by: ""
classification_approved_at: ""
classification_approval_reason: ""
---

# TASK-0057 PLAN

## 分類・前提

これは製品変更である。config V1 に workspace を一つ追加し、後続の socket activation と peer binder が必要とする numeric identity と共通 `brokerlistener.Subject` を、一 service lifetime の immutable な解決結果から供給できる境界を追加する。DEV は承認済み profile `luna-xhigh` を使用し、planning / candidate / completion の通常 3 commit 経路に従う。初回 candidate は許可パスだけを変更し、追加・削除合計 1,000 行以下に収める。

TASK-0055/0056 の公開契約は変更せず、両者への配線は後続の service composition が所有する。外部 dependency、config version、service binary、capability 発行/配布、repository check は追加・変更しない。生成済み `harness.json.example` は candidate に含めず、入力 template だけを更新する。

## AC対応

TASKの条件本文を再掲せず、`planning input packet`のAC-IDに設計を対応させる。

| AC-ID | 設計判断 | 変更パス | 実施順序 | 失敗時の扱い |
|---|---|---|---|---|
| AC-1 | `Config` に `Identity` を加え、`workspace_id` を既存の strict JSON token pass と typed decode に通す。byte 長と ASCII の identifier 規則を config 内で検証し、保持値を copy する。example、command JSON fixtures、provision の direct Config validation/test を同期する。既存 user/path/network の validate、manifest/action、V1 version は不変にする。 | `tools/dev-agent-harness/internal/config/config.go`、`tools/dev-agent-harness/internal/config/config_test.go`、`tools/dev-agent-harness/internal/command/command_test.go`、`tools/dev-agent-harness/internal/provision/provision.go`、`tools/dev-agent-harness/internal/provision/provision_test.go`、`tools/dev-agent-harness/config/harness.json.example.in` | 1 | unknown/duplicate/missing/不正 ID は既存 fixed config error class に正規化し、入力文字列・path・decoder detail を露出しない。direct Config の workspace 不正は既存 provision invalid-record class で拒否し、manifest/action を生成しない。 |
| AC-2 | `runtimeidentity.Resolver` は固定 agent/broker username と workspace ID を constructor で validate/copy し、private lookup/EUID seam を保持する。Linux `Resolve` は agent user、broker user、agent group を各一回だけ同期 lookup し、canonical decimal から Go `int` と Linux `uint32` の双方へ lossless な正数を得て、EUID/non-root、distinct UID、primary-GID/group-GID 一致を確認する。 | `tools/dev-agent-harness/internal/runtimeidentity/` | 2–3 | invalid rules、typed-nil/nil/corrupt receiver、lookup・parse・EUID・一致検査失敗は結果なしの一 fixed non-leaking error に畳む。username、workspace、UID/GID、NSS/error detail は Error/Format に出さない。 |
| AC-3 | 一 service start が一回呼ぶ解決成功 path だけで `crypto/rand` から一回 16 byte を読み、`agent-` と lowercase hex を連結する。返却 identity は broker UID、agent UID/GID と既存 `brokerlistener.Subject` を一つの immutable snapshot として保持し、各 accessor で Subject と文字列を fresh copy して返す。 | `tools/dev-agent-harness/internal/runtimeidentity/` | 3–4 | entropy short read/failure、lookup 失敗、identity mismatch、receiver corruption、non-Linux は partial result を返さず fixed denial とする。retry/cache/goroutine/log/persistent state を加えない。 |
| AC-4 | core test は package-private lookup/EUID/entropy seams で constructor/copy/corruption、lookup の exact one-call と順序非依存、numeric parse/boundary、EUID/user/group 条件、entropy の exact length/call/failure、fresh instance/Subject copy、fixed diagnostics を検出する。Linux source は build tag で実 production lookup に限定し、non-Linux adapter は常時 denial にする。 | `tools/dev-agent-harness/internal/runtimeidentity/`、`tools/dev-agent-harness/README.md` | 4–5 | fake seam・Linux cross-compile の PASS を real NSS、別 UID/GID、restart、VPS の実行証拠にしない。platform-specific live-e2e は blocked のままとし、別 mode の PASS で代替しない。 |
| AC-5 | focused config/runtimeidentity tests、harness `make check`/`make distcheck`、root `make lint-docs`、candidate root `make check`、`git diff --check` と許可 path/line count を candidate 前に確認する。 | 許可パス全体 | 5–6 | focused failure は candidate を固定せず原因を分類する。scope/line limit/dependency/config version/service composition/check の逸脱は reject し、同一 candidate だけを独立 REVIEW/QA へ渡す。 |

## 責務・境界・不変条件

- config は root-owned V1 input から workspace ID を strict に読むだけであり、instance ID、numeric UID/GID、NSS/EUID lookup、service lifecycle を持たない。`identity.workspace_id` は config lifetime に固定され、Agent instance ID は config に保存しない。
- `runtimeidentity` は config から渡された固定 username/workspace を validate/copy し、Linux identity snapshot を起動ごとに一回だけ解決する。agent、broker、同名 agent group の lookup は Resolve 呼出し当たり各一回で、lookup order を観測可能な API 契約にしない。production NSS access は Linux adapter のみが所有する。
- result は broker UID、agent UID、agent primary GID と同じ snapshot の `brokerlistener.Subject` だけを trusted composition に供給する。getter は mutable alias を渡さず、AgentInstanceID/WorkspaceID は保持時・返却時とも copy する。accessor は user/group record、password/home/shell、raw entropy、lookup failure を公開しない。
- `socketactivation` は返された broker UID/agent GID で systemd-created listener metadata を照合し、`peerbinder` は同じ result の agent UID/Subject を照合する予定である。ただし本 Task はそれらを import、wiring、起動しない。
- Linux 固有 API は build-tag adapter に隔離し、非Linux は user lookup が利用できそうでも fail closed とする。resolver は root/sudo、独自 `/etc/passwd` parser、LDAP/NSS 設定、shell/getent、cache/retry/timeout goroutine、log を所有しない。

## 補足設計

### 代替案と不採用理由

- service main が socket activation と peer binder 用にそれぞれ username lookup と instance-ID generation を行う案は、同一起動で異なる OS snapshot/Subject を用い得るため採用しない。
- workspace ID を起動時 entropy へ混ぜる、または AgentInstanceID を config/persistent state に置く案は、config-fixed workspace と restart ごとに新規の capability subject という lifetime 境界を曖昧にするため採用しない。
- `os/user` の非Linux fallback、NSS command、独自 passwd/group parser、cgo または新 dependency を使う案は、production source と portability/security surface を広げるため採用しない。
- lookup record を public result に返す、diagnostic に identity 値や下位 error を埋め込む案は、trusted composition に不要な data と情報漏洩面を増やすため採用しない。
- 実 user/group 作成、sudo、systemd restart、real NSS/LDAP、VPS を candidate test にする案は、安全な環境と cleanup がこの Task にないため採用しない。

### 移行・互換性

- V1 config document は required `identity.workspace_id` を持つようになる。更新済み example、既存 command JSON fixture、provision の direct Config fixture はこの required input を供給する。version、既存 field の意味、provision action/順序、user mapping は変えない。
- 新 package は future composition 用の resolver を提供するだけで、現行 service binary の通常起動、socket activation、peer binder、broker listener/session/exchange、capability の振舞いを置換しない。username/GID fallback と多 workspace mapping は追加しない。
- non-Linux でも core は build/test 可能だが、production identity 解決には成功しない。Linux cross-compile は adapter の source selection 確認であり、real NSS/UID/GID/live restart の確認ではない。

## 変更予定

| パス | 変更内容 |
|---|---|
| `tools/dev-agent-harness/internal/runtimeidentity/` | immutable Resolver、fixed non-leaking error/Format、private lookup/EUID/entropy seams、canonical numeric conversion、one-shot Linux user/group resolver、fresh result accessors、Linux adapter、non-Linux denial、hermetic core tests と Linux adapter source testを追加する。 |
| `tools/dev-agent-harness/internal/config/config.go` | V1 `identity.workspace_id` struct/decode/validation/copyを追加し、既存 strict parse/error class と他 field の validation を保つ。 |
| `tools/dev-agent-harness/internal/config/config_test.go` | canonical fixture を更新し、workspace の valid/copy と unknown/duplicate/missing/empty/length/ASCII invalid case が既存 class に落ちることを検出する。 |
| `tools/dev-agent-harness/internal/command/command_test.go` | `check-config`、`plan-provision`、`verify-provision` の既存 V1 JSON fixture に正規 workspace ID を加え、command 出力/診断/manifest assertion の意味を変えない。 |
| `tools/dev-agent-harness/internal/provision/provision.go`、`tools/dev-agent-harness/internal/provision/provision_test.go` | direct に渡される `config.Config` でも workspace ID の required identifier 規則を再検証し、valid fixture と missing/invalid direct-config rejection を同期する。manifest の header、10 action、順序、値、side-effect-free boundary は変えない。 |
| `tools/dev-agent-harness/config/harness.json.example.in` | V1 example に正規 workspace ID を一つ加える。生成済み output は更新しない。 |
| `tools/dev-agent-harness/README.md` | config-fixed workspace、service-lifetime instance、Linux-only resolver、socket/peer composition との責務分離、および hermetic/cross-compile と real NSS/UID/GID/restart/VPS live-e2e の境界を追記する。 |

許可外の service binary、`socketactivation`、`peerbinder`、`brokerlistener`、capability/config version/dependency/check、Task/QA_PLAN/Wiki は変更しない。

## 実装手順

1. config V1 の `Identity` field と workspace ID validator/copy を既存 strict parse flow へ追加し、canonical test JSON、command fixture、provision direct-Config fixture、example template を同期する。unknown/duplicate/missing/semantic invalid が既存 fixed class を保つことを test で固定する。provision の direct validation は workspace 規則を追加するだけで、manifest/action の意味を変更しない。
2. `runtimeidentity` core に rules、immutable Resolver、fixed error/Format、private seams、identifier と canonical/lossless numeric validation を定義する。username/workspace は constructor で copy/validate し、zero/nil/typed-nil/corrupt state を public result なしの denial に正規化する。
3. Linux adapter に標準 library の user/group lookup と current EUID read を隔離し、agent/broker/group の各一回 lookup、UID/GID parsing、non-root broker EUID、distinct agent UID、agent primary GID/group GID match を実装する。non-Linux adapter は fixed denial のみを返す。
4. Resolve の成功 path に一回だけの 16-byte crypto entropy と `agent-` lowercase-hex formatting を加え、numeric values と fresh `brokerlistener.Subject` copy を一つの immutable result に閉じ込める。accessor/Resolve return で alias が残らないことを確認する。composition は加えない。
5. hermetic core tests と Linux adapter build-tag test source を追加し、lookup count/order independence、numeric boundary、EUID/user/group rejection、entropy exactness、copy/corruption、fixed diagnostics を検出する。README の責務と live 境界を同期する。
6. focused config/runtimeidentity tests、Linux cross-compile、harness `make check`/`make distcheck`、root `make lint-docs`、candidate launcher の一回の root `make check`、`git diff --check`、許可 path と 1,000 行上限を確認してから製品差分だけの candidate を固定する。同一 candidate を独立 REVIEW/QA と completion へ渡す。

## 検証計画

| 検証 | 目的・主なケース | 実施責任 / 時点 |
|---|---|---|
| config strictness | valid workspace、保持/返却 copy、unknown/duplicate/missing/empty、1/128 byte boundary、先頭・後続 ASCII 規則、不正 UTF-8/空白を確認し、既存 class/version/other fields、command fixtures、provision direct Config validation が同期することを確認する。manifest の 10 action と順序・意味が不変であることも確認する。 | DEV / candidate |
| resolver hermetic core | constructor の username/workspace/rules/corruption、agent/broker/group各一回の lookup、order 非依存、canonical/lossless numeric boundary、current EUID/root/distinct UID/primary GID mismatch、partial-result absence と fixed Error/Format を確認する。 | DEV、独立 QA / candidate |
| entropy and copy | 一解決呼出しの entropy reader が 16-byte exact one callであること、short/failure、新しい `agent-` lowercase hex ID、broker/agent IDs と Subject の整合、getter/Subject string copy を確認する。 | DEV、独立 QA / candidate |
| platform source boundary | `GOOS=linux GOARCH=amd64 go test -run '^$' ./internal/runtimeidentity` で Linux adapter source を cross-compile し、non-Linux adapter が fixed denial になる unit case を確認する。 | DEV / candidate |
| repository checks | focused config/runtimeidentity tests、harness `make check`、`make distcheck`、root `make lint-docs`、candidate root `make check`、`git diff --check`、allowed path/1,000-line count を実行・監査する。 | DEV / candidate、REVIEW/QA / evidence-review |
| post-merge | main で所定の `make task-check TASK=TASK-0057` を実行する。 | Main / completion |

real Linux の別 broker/agent user/group、NSS/LDAP 設定、service start/restart、systemd composition、VPS live E2E は本 Task で安全に再現できない。これらは `live-e2e` を blocked のままとし、hermetic test 又は cross-compile の PASS で代替しない。

## 未解決事項

- なし

## main Agentレビュー

- [x] TASKの全AC-IDへ設計判断、パス、順序、失敗時の扱いを対応させ、条件本文を複製していない。
- [x] config-fixed workspace、one-shot instance ID、Linux numeric identity snapshot、accessor copy、nonLinux denial、composition/live境界を具体化している。
- [x] QA_PLANがTASK-firstで独立作成されている。
- [x] `dependency-ready reconciliation`と完了経路preflightが完了している。
- [x] DEV開始を承認した。

安全契約変更でv2契約を選ぶ場合は、コメントを外して`safety_contract_version: 2`と予定パス・生成パスの配列を記録し、DEV前に`make task-preflight TASK=TASK-0057`を実行する。変更しない種別は空配列とし、通常の予定パスと生成パスを重複させない。独立計画レビューのPASSとMainの分類承認をフロントマターへ記録する。分類変更時はTask、PLAN、QA_PLANを再承認し、承認者と時刻を更新する。
