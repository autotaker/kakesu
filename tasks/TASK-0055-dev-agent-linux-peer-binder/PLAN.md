---
task_id: "TASK-0055"
change_class: "product_change"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "luna-xhigh"
approved_dev_profile_reason: "新規 internal/peerbinder の有限な constructor/Bind と build-tag adapter、hermetic fixture、README追記だけに閉じ、既存 PeerBinder interface、listener/session 合成、設定、service、外部作用を変更しないため。"
approved_dev_profile_risk_signals: []
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T15:55:37Z"
planning_reviewed_by: ""
planning_review_decision: "pending"
planning_reviewed_at: ""
classification_approved_by: ""
classification_approved_at: ""
classification_approval_reason: ""
---

# TASK-0055 PLAN

## 分類・前提

これは製品変更である。TASK-0052 の `brokerlistener.PeerBinder` と `Subject` は変更せず、それを実装する
新規 package にだけ OS identity の境界を置く。DEV は承認済み profile `luna-xhigh` を使用し、planning / candidate /
completion の通常 3 commit 経路に従う。初回 candidate は製品差分だけを追加・削除合計 700 行以下に収め、新しい
dependency、設定、build/generated artifact、check を追加しない。

## AC対応

TASK の条件本文を再掲せず、`planning input packet` の AC-ID に設計を対応させる。

| AC-ID | 設計判断 | 変更パス | 実施順序 | 失敗時の扱い |
|---|---|---|---|---|
| AC-1 | `Rules` は expected UID と一つの `brokerlistener.Subject` だけを受け、既存 Subject と同じ identifier 規則、および Linux `uint32` とGo `int`の双方へlosslessに収まるUID範囲を検査する。検証済み値と reader seam を文字列 copy して immutable `Binder` に閉じ込める。 | `tools/dev-agent-harness/internal/peerbinder/` | 1 | invalid Rules、nil/typed-nil/corrupt receiver、範囲外 UID は空 Subject と単一固定 error に畳み、identity・UID・socket・下位 error を Format/error に出さない。 |
| AC-2 | `Bind` は concrete かつ non-nil の `*net.UnixConn` だけを type assert で受理する。context を reader 前後に確認し、一回だけ reader を同期呼出し、exact UID 時だけ固定 Subject の新しい copy を返す。 | `tools/dev-agent-harness/internal/peerbinder/` | 2–3 | wrapper/TCP/nil、reader failure、UID 不一致、reader 後の cancel は空 Subject と固定 error にする。RemoteAddr、socket path、payload、PID/GID、caller supplied identity は読まず、retry/cache/goroutine/log も追加しない。 |
| AC-3 | platform-independent core は reader を非公開の小さな seam として保持し、constructor/receiver/context/type/copy/error の policy を unit test 可能にする。reader は OS adapter だけが提供し、connection ごとに一回しか読めない。 | `tools/dev-agent-harness/internal/peerbinder/` | 1–3 | nil seam、typed-nil context/connection、前後 cancellation、corrupt Binder を全て同一 fail-closed result に正規化する。 |
| AC-4 | Linux adapter は `SyscallConn` と `Control` 内の標準 library `GetsockoptUcred(fd, SOL_SOCKET, SO_PEERCRED)` だけで UID を一回取得し、変換前に int 表現可能性を確認する。non-Linux build-tag adapter は reader を常に失敗させる。 | `tools/dev-agent-harness/internal/peerbinder/` | 3–4 | `SyscallConn`、Control、getsockopt、nil/範囲外 credential の詳細を公開せず固定 error に畳む。non-Linux に UID/metadata fallback や production support 表現を追加しない。 |
| AC-5 | hermetic core test は counting/failing/blocking reader と UnixConn fixture で constructor、exact/mismatch UID、copy、前後 cancel、一回呼出、固定診断を検出する。Linux build-tag integration test は実 Unix socket の accepted connection で kernel peer UID を確認する。README は static subject + kernel UID の保証と live 境界だけを記す。 | `tools/dev-agent-harness/internal/peerbinder/`、`tools/dev-agent-harness/README.md` | 4–6 | fake seam の PASS を socket permission、namespace、実 UID 分離、listener creation/configuration、systemd/VPS の証拠にせず、live-e2e を blocked のまま残す。 |

## 責務・境界・不変条件

- `Binder` は listener owner が信頼して固定する一つの Subject と expected UID の対応だけを表す。kernel UID が一致しても、connection、address、protocol、PID/GID から Subject を生成・補完しない。一 listener を複数 Subject/UID へ多重化しない。
- core は `brokerlistener` に依存して Subject 型を使用するだけで、同 package の validation/copy private helper を再利用しない。identifier validation と copy は `peerbinder` 内で明示し、保持時と返却時に caller alias を残さない。
- `Bind` は caller context の `Err()` を OS 読取の直前と直後に確認する。OS read を context timeout 用 goroutine に移さず、接続ごとに一回の同期照合とする。
- Linux 固有 syscall は build-tag file のみに隔離する。core の reader result は UID のみとし、PID/GID、fd、path、`Ucred`、下位 error を結果・error・Format に運ばない。non-Linux は同じ public core を build できても fixed denial だけである。
- `peerbinder` は受理済み connection を close せず、socket の作成、listen/bind/unlink、permission、UID name resolution、config/CLI/composition、Session/HTTP/TLS/policy を所有しない。

## 代替案と不採用理由

- UID から global Subject map を引く案は、multi-tenant listener、lifetime/revocation state、config を持ち込み listener ごとの固定 binding を壊すため採用しない。
- `net.Conn` / `syscall.Conn` 一般を受けて adapter を選ぶ案は wrapper/TCP を identity 根拠にできるため採用しない。具体的 `*net.UnixConn` だけを許可する。
- RemoteAddr、socket path、header/payload、PID/GID の補完は kernel authenticated peer identity ではないため採用しない。
- `x/sys/unix`、cgo、OS command、非 Linux credential API は dependency/portability surface を広げるため採用しない。Go standard-library syscall adapter と non-Linux denial に限定する。
- read timeout/retry/cache/log、実 listener や socket permission/namespace/VPS test は bounded OS authentication boundary を越えるため採用しない。

## 変更予定

| パス | 変更内容 |
|---|---|
| `tools/dev-agent-harness/internal/peerbinder/` | immutable Binder、fixed non-leaking error/Format、Subject validation/copy、context and concrete UnixConn gate、reader seam、Linux SO_PEERCRED adapter、non-Linux fail-closed adapter、hermetic unit tests と Linux-only socket integration test を追加する。 |
| `tools/dev-agent-harness/README.md` | listener 単位の静的 Subject と Linux kernel UID 照合の保証、および UID separation/socket permission/namespace/listener composition/live VPS を未検証として追記する。 |

`brokerlistener`、全既存 package、設定/CLI/service、生成物、Task/QA_PLAN/backlog/Wiki は変更しない。

## 実装手順

1. `peerbinder` core の `Rules`、immutable `Binder`、fixed error/Format、identifier/UID validation、Subject copy と typed-nil-safe guards を定義する。expected UID は Linux credentialの`uint32`とSubjectの`int`へlosslessに収まる正数範囲だけを許す。
2. non-public reader seam を core へ接続し、`Bind` の context-before → concrete UnixConn gate → one reader call → context-after → exact UID → fresh Subject copy の順序を実装する。全 reject result を empty Subject と fixed error に統一する。
3. build-tag Linux adapter に `SyscallConn`/`Control` と一回の `GetsockoptUcred` を実装し、Control callback と outer call の失敗を返却しない固定結果へ正規化する。build-tag non-Linux adapter は常時 denial reader にする。
4. hermetic core tests を追加し、invalid/typed-nil/corrupt constructor/receiver/context/conn、UID boundary/exact mismatch、reader failure、一回呼出、before/after cancel、copy と error/Format non-leak を固定する。fake reader は `*net.UnixConn` を要求する core policy と独立に count 可能にする。
5. Linux-only test で temp Unix listener と client/accepted `*net.UnixConn` を生成し、accepted peer UID が current effective UID と一致して static Subject が返ることを確認する。test は listener path、credential detail を assertion failure 以外の公開 output に利用しない。
6. README を境界に合わせ、focused package test、Linux cross-compile、harness `make check`/`make distcheck`、README lint、`git diff --check`、許可 path と 700 行上限を candidate 前に確認する。candidate launcher が root `make check` を一回実行して製品差分だけを固定し、その同一 candidate を独立 REVIEW/QA と completion へ渡す。

## 検証計画

| 検証 | 目的・主なケース | 実施責任 / 時点 |
|---|---|---|
| hermetic package test | Rules の UID/identifier/typed-nil、zero/corrupt Binder、concrete UnixConn だけの受理、reader failure、exact/mismatch UID、Subject copy、fixed error/Format を確認する。 | DEV / candidate |
| call and context test | reader counter で一回だけの同期呼出を確認し、nil/typed-nil/already-cancelled context は reader 前に、reader 中 cancellation は reader 後に denial となることを確認する。 | DEV、独立 QA / candidate |
| Linux socket integration | Linux だけでactual Unix accept peerのSO_PEERCRED UIDをadapter経由で検出する。実行UIDが正数ならBinder成功も確認し、root環境ではreaderがUID 0を取得してconstructorがroot subjectを拒否する境界を確認する。PID/GID/path/payloadはidentityに使わない。 | Linux上のDEV/CI。現在のmacOS QAはsource監査とcross-compileだけで、実行PASSを主張しない。 |
| portability compilation | non-Linux fail-closed adapter が選択可能であり、`GOOS=linux GOARCH=amd64 go test -run '^$' ./internal/peerbinder` が Linux cross-compile することを確認する。 | DEV / candidate |
| repository checks | harness package test、`make check`、`make distcheck`、README lint、`git diff --check`、allowed path/line count を実行し、candidate launcher の root `make check` は一回だけ行う。 | DEV / candidate、REVIEW/QA / evidence-review |
| post-merge | main で所定の `make task-check TASK=TASK-0055` を実行する。 | Main / completion |

実 Agent/broker UID provisioning、socket path owner/mode、systemd socket activation、network namespace、real client、VPS live E2E は本 Task の受け入れ真実を再現しないため実施しない。live-e2e は blocked のままとし、別モードの PASS で代替しない。

## 移行・互換性

- 新規 package は既存 `brokerlistener.PeerBinder` を実装するが、Server の dependency injection と Subject validation/copy を変更しない。production wiring は対象外のため、既存 caller の振舞いを置換しない。
- Linux 以外は build 可能でも Binder が必ず拒否する。TCP/vsock、Windows/macOS を support と見せる互換 fallback は追加しない。
- Task の `Dependency-ready reconciliation` と completion preflight は ready。新しい設定値、persistent mapping、migration、generated file はない。

## 未解決事項

- なし。

## main Agentレビュー

- [x] TASK の全 AC-ID へ設計判断、パス、順序、失敗時の扱いを対応させ、条件本文を複製していない。
- [x] listener-static Subject、concrete UnixConn、one-shot kernel UID read、前後 context cancellation、copy/fail-closed を具体化している。
- [x] Linux adapter と non-Linux denial を build tag で分離し、socket/config/composition/live 境界を拡張していない。
- [x] QA_PLAN が TASK-first で独立作成されている。
- [x] `dependency-ready reconciliation` と完了経路 preflight が完了している。
- [x] DEV 開始を承認した（approved_dev_profile: `luna-xhigh`）。
