---
task_id: "TASK-0051"
status: complete
completed_at: "2026-08-01T13:51:40Z"
candidate_commit: "9fd8c1ec1a70033c4157485a74e01feda8d6325b"
---

# TASK-0051 HANDOVER

## 成果

- 受理済み一connectionをstrict CONNECT、host-bound TLS、単一HTTP/1.1 requestの順に処理してcloseする`connectsession.Session`を追加した。
- exact 2 host、限定CONNECT header、Issue-before-200、TLS 1.2+/SNI/ALPN、caller context継承、固定non-leak失敗を一つの有限state machineへ固定した。
- real `brokerhttp.Handler`とin-memory CAを用い、拒否、TLS failure、cancel/panic、並行context/host隔離をhermetic race testで検出した。
- 協調Handlerが受け取るHTTP phase deadlineについて、固定5秒上限とcaller側の早いdeadlineの双方を回帰testで観測する。

## candidate-bound DEV証跡

| コマンド/テスト | 結果 |
|---|---|---|
| candidate launcherのroot `make check`（固定前に一回） | `PASS` |
| `GOCACHE=$PWD/.build/go-cache go test -count=1 -race ./internal/connectsession` | `PASS` |
| `make -C tools/dev-agent-harness check` | `PASS` |
| `make -C tools/dev-agent-harness distcheck` | `PASS` |
| `make lint-docs` | `PASS` |
| `git diff --check` | `PASS` |

## 主要な変更

- `internal/connectsession`にAuthority interface、immutable Session、16 KiB strict CONNECT parser、固定403/200、5秒phase deadline、TLS終端、1 MiB body・16 KiB aggregate header上限とheader-name検証を持つsingle HTTP handoff/response writerを追加した。
- optional outer headerを単一visible-ASCII User-Agentと単一`Proxy-Connection: keep-alive`だけに限定し、その他framing/header/early byteを依存前に拒否した。
- READMEにone-connection責務と、listener/identity/CA trust/real client/VPSを実装・確認していない境界を追記した。

## 検証結果

- candidate `9fd8c1ec1a70033c4157485a74e01feda8d6325b`を固定した。base `8d4f8d27bd85d12d8450dbd62170fa226bcd2855`からの製品差分は許可3 files、追加1,000行・削除0行である。
- focused race、harness check/distcheck、Task worktree lint-docs、diff check、candidate launcher root checkがすべてPASSした。

## 判断・既知の制約

- Sessionはconnを完全所有し、CONNECT/TLS/HTTP各phaseで固定5秒又はcaller deadline/cancelの早い方を適用する。一HTTP response後はkeep-aliveせずcloseする。
- 注入するHandlerは`brokerhttp.Handler`相当の信頼済み依存であり、渡されたrequest contextのdeadline/cancelへ協調してreturnする契約である。Sessionはconnection I/O deadline、context伝播、自身のcancel watcherとconnection cleanupを保証するが、任意の非協調callbackを強制停止しない。callbackを別goroutineで包んでtimeoutさせる方式は、callbackが戻らない場合にgoroutineを確実にleakさせるため採用しない。
- caller contextを透過するだけで、CONNECT header、RemoteAddr、SNI、inner headerから主体を生成しない。production peer identityは後続listener境界である。
- 実socket bind/accept、OS peer identity、CA file/rotate/trust install、実gh/OpenAI proxy、network namespace/VPSは未実装・未確認であり、hermetic PASSで代替しない。
