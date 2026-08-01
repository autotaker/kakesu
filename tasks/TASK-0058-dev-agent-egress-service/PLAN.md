---
task_id: "TASK-0058"
change_class: "product"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "luna-xhigh"
approved_dev_profile_reason: "ユーザ指定のDEV Lunaを用い、既存のcredential/socket/identity constructorを変更しない有限なproduction composition、strict config/provision/CLI/unit同期とpackage-private hermetic seamに閉じる。live OS操作、実秘密、外部通信、依存追加は行わず、Solへ昇格を要する未制御リスクはない。"
approved_dev_profile_risk_signals: []
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T18:08:34Z"
planning_reviewed_by: "main-agent-sol-high"
planning_review_decision: "pass"
planning_reviewed_at: "2026-08-01T18:08:34Z"
classification_approved_by: "main-agent-sol-high"
classification_approved_at: "2026-08-01T18:08:34Z"
classification_approval_reason: "config、provision、service binary/systemdと外部観測可能な起動挙動を変更する製品変更。"
---

# TASK-0058 PLAN

## 分類・前提

これは製品変更である。V1 config/provision、`dev-agent-egress` の唯一の operational surface、systemd
input template、既存安全境界を一つの起動 graph に接続する。DEV は承認後 `dev-agent-luna-xhigh` を使用し、
candidate は許可 path だけ、追加・削除合計 1,100 行以下に収める。生成済み configure/config は candidate に
含めず、`.in` のみを更新する。

この Task は各 package の policy、credential reader、TLS/HTTP、socket、peer、identity の公開意味を変えない。
実 NSS/systemd FD/秘密/ネットワークは hermetic test または Linux cross-compile で PASS と扱わず、completion
時点の live-e2e は環境と安全な cleanup が未指定のため blocked のままとする。

## AC対応

TASKの条件本文を再掲せず、`planning input packet`のAC-IDに設計を対応させる。

| AC-ID | 設計判断 | 変更パス | 実施順序 | 失敗時の扱い |
|---|---|---|---|---|
| AC-1 | top-level `Config.Egress`（JSON `egress`）に2 allowlist を加え、strict token pass/typed decode の後で各 list を 1–32、重複なし、既存 `egresspolicy.New` と同じ受理規則に検証・copy する。provision は既存 user、4 directory、3 service の順を保ち、config child `credentials` を broker:broker `0700` として directory の末尾・sequence 8 に一件追加し、service を sequence 9–11、action 11 にする。 | config/provision と各 test、example、command fixture | 1–2 | missing/empty/33件/duplicate/unknown/不正値は既存 config fixed class、direct Config は provision fixed class に畳み、manifest は生成しない。credentials 追加に伴う service sequence 9–11 への shift 以外、既存 action の意味・相対順を変えない。 |
| AC-2 | 新設 `egressservice` が config を一回 load、同じ config values で runtime resolver を一回 New/Resolve して broker 主体を確認した後、固定 `config_dir/credentials` を一回 load する。全依存の New 成功後だけ receiver の `Take` を一回呼び、返却 listener を直ちに `Server.Serve` に移譲する。 | `internal/egressservice/`、command/main と test | 3–5 | 各前段の failure は以降の loader/constructor/Take/Serve を呼ばず、service 固定 error と exit に正規化する。Take 成功後は Server が listener を所有・close/drain し、service は二重 close/再 Take しない。 |
| AC-3 | 既存 constructor だけで policy、empty registry、transport、credential resolver、exchange、context-only listener resolver + HTTP handler、bundle の proxy CA authority + CONNECT session、identity の UID/Subject を持つ PeerBinder、listener Server、socket Receiver を一 lifetime graph にする。同一 resolved identity の broker UID/agent GID を Receiver、agent UID/Subject を Binder へ渡す。 | `internal/egressservice/` | 3–4 | constructor/dependency corruption は listener なしの固定 start error に畳む。default client、fallback/retry/cache、別 identity/NSS lookup、追加 goroutine/network は導入しない。空 Registry は issuerなしで保持し、未知 handle の拒否を通常の fail-closed path とする。 |
| AC-4 | command は `dev-agent-egress serve --config PATH` だけを service に渡す。main は SIGINT/SIGTERM から cancelable context を作り、service error を fixed non-leaking diagnostic/exit に変換する。`--version` と no-args の既存 fail-closed 契約を保つ。unit template は固定 config path と socket activation を宣言し、credential/environment/capability の引数・環境渡しを置かない。 | command/main test、service template、README | 5–6 | invalid argument は service を開始せず既存 command の usage exit、config/identity/credential/graph/Take/Serve/cancel failure は path・identity・secret・FD/cause を出さない固定診断となる。Serve の cancel は Server の協調 shutdown に任せる。 |
| AC-5 | package-private factory/dependency seam だけで startup event、各 exact call、identity object/value wiring、failure boundary と returned-listener ownership を hermetic に検出する。focused package/command/config/provision tests、Linux cross-compile、harness/root checks と diff scope/count を candidate に束縛する。 | 許可 path 全体 | 6–7 | focused failure、line/scope 超過、non-hermetic dependency は candidate を固定せず分類する。live systemd/NSS/secret/provider case は evidence-review/focused-rerun の PASS で置換しない。 |

## 責務・境界・不変条件

- `config` は allowlist を strict V1 input として validate/copy するだけで、policy 評価、秘密 path の選択、identity、I/O を追加しない。`egresspolicy` の既存 private recognizer と同一の grammar/bounds を config 境界に明示し、両 package の許容/拒否 table を揃えて drift を検出する（公開 API や対象外 package は変更しない）。
- `provision` は desired-state manifest のみを所有する。credentials は config directory の固定 child で、broker-owned `0700`、directory records の既存順を壊さず service record 前に置く。executor、秘密作成、mode の実適用は対象外である。
- `egressservice` は production composition と startup sequencing のみを所有する。外から注入可能な production factory は公開せず、test hook は package-private に閉じる。config と credential bundle は start 一回につき各一回だけ読み、identity の唯一 snapshot 以外を socket/peer へ渡さない。
- graph の固定値は policy `egress-v1`、Registry TTL 10分/uses 16/epoch 1、policy body 64 KiB/output 4096、resolver timeout 10秒、exchange credential 4096/forward timeout 30秒/response 1 MiB、HTTP body 64 KiB、Server concurrent 16 とする。`upstreamtransport.New()` が唯一の transport で、HTTP default client/環境 proxy は選択しない。
- graph は `egresspolicy.New` → `capability.New` → `upstreamtransport.New` → `providercredentials.New` → `brokerexchange.New` → `brokerhttp.New`（`brokerlistener.Resolver{}`）→ `connectsession.New`（bundle `ProxyCAAuthority()`）→ `peerbinder.New` → `brokerlistener.New` → `socketactivation.New` の順で構成する。receiver の `Take` はすべての `New` 後で一回だけ、`Server.Serve(ctx, listener)` の直前である。
- `brokerlistener.Server` が Take 後の listener の唯一の lifecycle owner である。start failure のうち Take 前には listener が存在せず、Take 自身の失敗時には receiver が cleanup し、Serve は cancellation 時に listener を閉じて accepted sessions を drain する。service が listener を保存、close、再利用しない。
- command/main は入力検証と signal-to-context bridge のみを持つ。config load は command で先読みしない。固定 service error は stdout/stderr、Error formatting、exit から config path、username/UID、credential、CA、socket/environment、下位 cause を漏らさない。

## 変更予定

| パス | 変更内容 |
|---|---|
| `tools/dev-agent-harness/internal/egressservice/**` | startup composition、固定 error と package-private hermetic factory/event seam、および order/ownership/diagnostic testsを追加する。 |
| `tools/dev-agent-harness/internal/config/config.go`、`config_test.go` | V1 egress allowlist field、strict validate/copy、boundary/duplicate/invalid testsを追加する。 |
| `tools/dev-agent-harness/internal/provision/provision.go`、`provision_test.go` | credentials directory/action count 11 の deterministic manifest と exact sequence/owner/mode testsを更新する。 |
| `tools/dev-agent-harness/internal/command/command.go`、`command_test.go` | egress serve の exact argument gate と fixed service error exit を追加し、既存 setup/scaffold/version contract を保つ。 |
| `tools/dev-agent-harness/cmd/dev-agent-egress/main.go` | SIGINT/SIGTERM cancel context を command/service entrypoint に渡す。 |
| `tools/dev-agent-harness/config/harness.json.example.in` | 両 allowlist の valid minimal exampleを加える。 |
| `tools/dev-agent-harness/deploy/systemd/dev-agent-egress.service.in` | fixed config path、socket unit wiring、broker identityを宣言し、secret/capability/env argument を追加しない。 |
| `tools/dev-agent-harness/README.md` | operational command、startup trust/order、empty Registry と hermetic/live boundaryを簡潔に同期する。 |

## 実装手順

1. config の top-level `egress` に `github_repositories` と `openai_models` を追加し、existing fixed error class のまま strict parse、missing/unknown/duplicate、1/32 boundary、copy、policy-compatible value を test する。example と command の valid fixture をこの schema に同期する。
2. provision の logical directory table へ fixed `filepath.Join(config_dir, "credentials")` を sequence 8 で追加し、ActionCount/header、directory count、service sequence 9–11、`0700` broker ownership を exact-byte tests とともに 11 action へ更新する。既存 path/user/service fields の意味と相対順を変えない。
3. `internal/egressservice` に public な小さい `Serve(ctx, configPath)`（名称は既存 package style に合わせる）を置き、非公開 dependency bundle を介して config→identity resolver/Resolve→fixed credential load を順に一回実行する。各 boundary の errors は新 package の一固定 start error へ畳む。
4. resolved config/bundle/identity から上記固定 constructor graph を合成する。`brokerlistener.Resolver{}` は HTTP handler の context-only subject resolver にだけ渡し、bundle CA は CONNECT session にだけ渡す。Receiver と Binder に同じ identity snapshot の respective values を渡すことを event seam で観測可能にする。
5. 全 constructor 成功後に receiver.Take を一度だけ呼び、成功 listener を直ちに Server.Serve へ一回移譲する。config/identity/credential/各 New/Take/Serve の failure table、Take前 non-reachability、Take後 ownership、cancel が Server に流れることを service test で固める。seam は external API に露出しない。
6. command の `serve --config PATH` exact gate と main の `signal.NotifyContext` を接続する。version/no args/help の既存 contract はテストで保護する。unit template、README を更新し、service unit に config path/socket dependency があり、secret/capability/environment input がないことを text test/inspection で確認する。
7. まず changed-package tests と Linux cross-compile を実行し、約20分の候補時間を守るため、full `make check`/`make distcheck` は schema/composition failures を解消してから各一回だけ実行する。candidate 前に `git diff --check`、許可 path、numstat ≤1,100 を確認し、Main の通常 candidate/独立 REVIEW/QA/completion へ渡す。

## 検証計画

| 検証 | 目的・主なケース | 実施責任 / 時点 |
|---|---|---|
| config/provision hermetic | strict lists の valid/copy、missing/empty/over-limit/duplicate/invalid/unknown、example/command fixture、11 action と credentials exact directory record を検出する。 | DEV / candidate |
| service composition hermetic | config→Resolve→credential→all constructors→Take→Serve の exact one-call/order、固定 limits、identity snapshot の Receiver/Binder wiring、empty Registry unknown-handle denialを検出する。 | DEV、独立QA / candidate |
| service failure/cleanup hermetic | 各前段 failure の後段非到達、Take 前 listener 不在、Take failure cleanup、Serve error/cancel の fixed error と Server-only ownership、diagnostic non-leak を検出する。 | DEV、独立QA / candidate |
| CLI/unit/doc | serve argument gates、version/no-args fail-closed、signal context handoff、unit config/socket wiring と no secret/env/capability argument、README boundary を検出する。 | DEV、独立QA / candidate |
| focused build | affected Go package tests と `GOOS=linux GOARCH=amd64 go test`/build を実施する。非Linux adapter failure を実 NSS/systemd 証拠にしない。 | DEV / candidate、QA / focused-rerun where hermetic |
| repository evidence | harness `make check`/`make distcheck`、root `make lint-docs`、candidate root `make check`、`git diff --check`、allow-path/line-count を candidate-bound evidence として監査する。 | DEV / candidate、REVIEW/QA / evidence-review |
| post-merge live boundary | 実 systemd FD、secret ownership、NSS、socket permission、provider/DNS/TLS は承認済み環境と cleanup 手順が未指定のため `live-e2e` blocked。Main は代替 PASS にしない。 | Main / completion |

## 代替案と不採用理由

- command が config を validate してから service に渡す案は config の二重 read と startup order 違反になるため採用しない。
- constructor 毎に identity を resolve、または Receiver/Binder が独自 lookup をする案は snapshot 一貫性を失うため採用しない。
- Take を credential/graph construction 前に行う案は失敗時に inherited listener を早期消費するため採用しない。
- static capability の発行、environment/file での handle 配布は issuer/trusted delivery 未実装かつ対象外のため採用しない。
- exported test factories、default HTTP client、fallback/retry/cache、production e2e fixture は package boundary と scope/時間上限を越えるため採用しない。

## 未解決事項

- なし。実 Linux/systemd/credential/provider の live-e2e は preflight のとおり completion で blocked と記録する。

## main Agentレビュー

- [x] TASKの全AC-IDへ設計判断、パス、順序、失敗時の扱いを対応させ、条件本文を複製していない。
- [x] config 一回読込、identity/credential/constructor/Take/Serve の順、同一 identity wiring、listener ownership を確認した。
- [x] fixed limits、空 Registry、non-leaking diagnostics、live-e2e blocked を確認した。
- [x] QA_PLANがTASK-firstで独立作成されている。
- [x] `dependency-ready reconciliation`と完了経路preflightが完了している。
- [x] DEV開始を承認した。
