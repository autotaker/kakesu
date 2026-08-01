---
task_id: "TASK-0055"
change_class: "product"
status: approved
qa_agent: "qa-agent-terra-medium"
qa_role: "independent-qa"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T15:55:37Z"
revision: 1
expectation_changed: false
---

# TASK-0055 QA PLAN

## QA scope

期待値正本は TASK.md の `Planning input packet` だけとする。PLAN、実装案、DEV の自己申告から期待値を導かない。candidate の
`tools/dev-agent-harness/internal/peerbinder/` と許可される README 差分を独立確認し、既存 `brokerlistener`、Session、HTTP/TLS、
capability、credential、provider、forwarder、policy、設定、build/generated artifact、外部 dependency を変更していないことを確認する。

この QA は macOS host 上で実行するため、実 Linux socket での `SO_PEERCRED`、実 UID 分離、socket owner/mode、namespace、systemd、
実 dev-agent/broker、real client/VPS は確認できない。これらは live-e2e を blocked とし、Darwin の hermetic reader seam test、Linux
cross-compile、又は source audit の PASS で置換しない。

## Cases

| Case ID | 対象AC | 確認内容と failure detection | Mode | Evidence |
|---|---|---|---|---|
| QA-001 | AC-1 | `New` が正数かつ platform UID range 内の expected UID と、各 1〜128 byte・先頭 ASCII 英数字・残り ASCII 英数字/`._-` の AgentInstanceID/WorkspaceID を持つ一つの Binding だけを受理することを確認する。境界値、invalid identity、nil/typed-nil/corrupt Binder、zero Binder が panic しない fixed non-leak error になる fixture があり、UID/identity/socket/lower error が error/Format に現れる実装、reader 又は subject の alias、default identity、複数 UID fallback を失敗検出する。 | focused-rerun | bundled package tests と candidate source/test audit |
| QA-002 | AC-2 | concrete non-nil `*net.UnixConn` だけを受理し、reader が同期に一回だけ呼ばれることを call-count seam で確認する。exact expected UID のみが固定 `brokerlistener.Subject` の独立 copy を返し、non-Unix/wrapped/nil connection、reader failure、UID mismatch は empty subject と fixed non-leak error になることを確認する。RemoteAddr、socket path、payload、PID/GID、caller declaration から subject を生成又は補完する実装を negative fixture/source audit で失敗検出する。 | focused-rerun | bundled package tests |
| QA-003 | AC-3 | nil/typed-nil/cancelled/deadline-expired context が reader 前に拒否され、reader が cancel した context では reader 後に subject を返さないことを deterministic seam で検出する。reader を goroutine に逃がす timeout、retry、cache、log、又は一 call を超える OS lookup を導入する実装を failure とする。 | focused-rerun | bundled package tests と candidate source audit |
| QA-004 | AC-4 | Linux adapter が `*net.UnixConn.SyscallConn` の `Control` 内だけで standard-library `GetsockoptUcred(fd, SOL_SOCKET, SO_PEERCRED)` を一回使い、SyscallConn/control/getsockopt failures を fixed error に畳むことを source/test evidence で監査する。非Linux adapter が常に fixed fail-closed error で、Darwin を production support と表現しないことを確認する。Linux build-tagged actual Unix socket integration test が peer UID を確認し、cross-compile に含まれることも確認する。 | evidence-review | candidate source/test、bundled Linux cross-compile result、candidate diff |
| QA-005 | AC-5 | package tests が constructor、exact UID、全拒否、subject/reader copy、context 前後 cancel、single call、fixed diagnostics を実際に failure-detect し、test の削除・弱体化又は非決定的/unbounded fixture を検出する。README は listener 単位の static subject + kernel UID 照合のみを保証し、実 UID 分離/socket permission/namespace/VPS が未検証である境界を示すことを確認する。focused package test、Linux cross-compile、harness/root checks、distcheck、README lint、`git diff --check` の DEV candidate evidence を独立監査し、許可 path、追加＋削除700行以下、dependency/config/generated artifact 無しを確認する。 | evidence-review | candidate source/test、README、HANDOVER、DEV command/result、candidate diff |
| QA-006 | 対象外 / live 境界 | approved Linux environment で actual Unix socket pair の `SO_PEERCRED` peer UID、異なる実 UID の reject、socket owner/mode、namespace/systemd、実 dev-agent/broker/client、VPS と安全な cleanup を確認する。 | live-e2e — blocked | 現 host は macOS であり、実 UID/socket permission/VPS の authority と cleanup が未定義。この blocked は他 mode の PASS で代替しない。 |

## Execution rule

QA-001〜003 と package-level failure detection、および Linux cross-compile は candidate に対する一回の bounded focused-rerun に束ねる。QA は
`tools/dev-agent-harness` を cwd として次だけを一回実行する。

```sh
GOCACHE=$PWD/.build/go-cache go test -count=1 ./internal/peerbinder && GOOS=linux GOARCH=amd64 GOCACHE=$PWD/.build/go-cache go test -run '^$' -exec /bin/true ./internal/peerbinder
```

後半は Linux test binary を compile するだけで macOS 上で実行しない。QA-004 と QA-005 の evidence-review は candidate source/test、README、HANDOVER、
DEV command/result、candidate diff を独立監査する。root/harness `make check`、`make distcheck`、README lint、candidate launcher root `make check`、
`git diff --check` は再実行せず、candidate-bound DEV evidence として監査する。fixture が required negative/boundary を失敗検出しない、candidate/tree と
証跡が一致しない、又は command が deterministic/bounded でない場合、その case は FAIL 又は blocked であり evidence-review PASS に置換しない。

## Result criteria

各 case を Planning input packet と candidate-bound evidence に照らして記録する。focused-rerun では command、cwd/cache、exit status、実行 test と
failure-detection evidence を残す。source/evidence audit は concrete UnixConn 限定、single synchronous reader call、exact UID、immutable
binding/independent copy、context 前後 cancel、fixed non-leak diagnostics、Linux syscall placement、non-Linux fail-closed、README の未検証境界を
明示する。

失敗を実装不具合と決めつけず、`implementation_defect`、`qa_plan_defect`、`requirement_gap`、`environment_issue`、`regression`、又は evidence
不足として根拠付きで分類する。QA-006 は安全に実行可能な approved Linux environment が得られるまで blocked のままとする。

## 実装後の再確認

- [ ] candidate source/test、README、HANDOVER、DEV check evidence を独立確認する。
- [ ] 指定 focused-rerun を candidate で一回だけ実行し、package test と Linux cross-compile を確認する。
- [ ] constructor/binding copy、exact UID/UnixConn、context 前後 cancel、single synchronous call、fixed diagnostics、Linux adapter/non-Linux fail-closed の failure detection を確認する。
- [ ] 変更が許可 path、追加＋削除700行以内、dependency/config/generated artifact 無しに収まることを確認する。
- [ ] real Linux UID/socket permission/namespace/systemd/dev-agent/broker/client/VPS live-e2e を PASS に置換せず、期待値又は scope を変更していないことを確認する。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | 2026-08-02 | qa-agent-terra-medium | Planning input packet に基づく独立 QA 計画 | `approved` |
