---
task_id: "TASK-0055"
status: complete
completed_at: "2026-08-01T16:16:31Z"
candidate_commit: "1a24ecf2b1225a83845ae681fc24cb2a7ffbe738"
---

# TASK-0055 HANDOVER

## 成果

- 受理済みUnix connectionを、待受けownerが固定した一つのSubjectとLinux kernelのpeer UIDへ束縛するproduction `brokerlistener.PeerBinder`を追加した。
- concrete `*net.UnixConn`、context前後、exact UIDだけを受理し、非Linux・不正rules・取得失敗・不一致を固定errorへ畳んだ。
- platform-independent failure detection、Linux限定actual Unix socket test、非Linux fail-closed、READMEの適用限界を追加した。

## candidate-bound DEV証跡

| コマンド/テスト | 結果 |
|---|---|---|
| exact candidate launcherのroot `make check` | `PASS`。固定working bytesの検査後、candidate `1a24ecf2b1225a83845ae681fc24cb2a7ffbe738`を作成 |
| `GOCACHE=$PWD/.build/go-cache go test -count=1 ./internal/peerbinder` | `PASS`（macOS、非Linuxadapterと共通core） |
| `GOOS=linux GOARCH=amd64 GOCACHE=$PWD/.build/go-cache go test -run '^$' -exec /usr/bin/true ./internal/peerbinder` | `PASS`（compile only。Linux testを実行したとは扱わない） |
| Windows/FreeBSD/Darwin portability compile | `PASS`（non-Linux fail-closed adapter） |
| `GOCACHE=$PWD/.build/go-cache go vet ./internal/peerbinder` | `PASS` |
| harness `./configure && make check && make distcheck` | `PASS`（live testsは設定どおりskip） |
| terminology validation / `git diff --check` | `PASS` |

## 主要な変更

- `Rules`は正数かつlosslessなexpected UIDと同値UIDを持つ一つの`brokerlistener.Subject`だけを検証・copyし、immutable Binderへ保持する。
- `Bind`はconcrete UnixConnを一回の同期readerへ渡し、context前後とexact UID一致後だけfresh Subject copyを返す。wrapper/TCP/nil/corrupt/reader error/UID mismatchは空Subjectと`peer-bind-denied`になる。
- Linux adapterは`UnixConn.SyscallConn().Control`内で標準libraryの`GetsockoptUcred(SOL_SOCKET, SO_PEERCRED)`を一回使い、UIDだけをcoreへ返す。非Linux adapterはfallbackなしで常に拒否する。
- unit testsはrules/UID/identifier境界、input/return copy、single read、context前後cancel、fixed diagnostics、wrapper/typed nil/corrupt拒否を検出する。Linux build-tag testはactual accepted Unix socketからpeer UIDを読む。
- READMEへ待受け単位の静的Subject/kernel UID照合と未検証の実配置境界を追記した。

## 検証結果

- candidateは`1a24ecf2b1225a83845ae681fc24cb2a7ffbe738`、treeは`6acbca8b50efa2449720e13cc7360d0872a13483`、baseはplanning `b6ddee0771b1a66b7342082a151ddf1c0fa727a7`である。
- base...candidateは許可6 filesだけ、追加432行・削除0行で700行以下。外部dependency、設定、build/generated artifact、既存package変更はない。
- 最初のcandidate前root checkはREADME追加文の`listener`/`peer`頻度lintだけを検出した。glossaryを増やさず新規文を日本語化し、terminology validation PASS後の固定差分でcandidate launcherのroot `make check`がPASSした。
- Task worktreeの初回root check準備では未cached Node package取得がsandbox DNS制限で停止した。lockfile固定dependencyをworktreeへ準備後、正規candidate launcherが同じ製品差分のroot `make check`を完了した。
- actual Linux socket test sourceはLinux test binaryへ含まれcross-compileしたが、現在のmacOS hostでは実行していない。

## 判断・既知の制約

- 一つのUIDから複数主体を引くmapは持たず、一つのBinder/listener lifetimeを一つのSubjectへ対応させる。AgentInstanceID/WorkspaceIDは接続内容、path、PID/GIDから生成しない。
- Binderはconnectionをcloseせず、既存`brokerlistener.Server`がconnection lifecycleと返却Subjectの二重validation/copyを所有する。
- Linux `syscall`はbuild-tag adapterだけに閉じ、`x/sys/unix`、cgo、OS command、retry/cache/goroutine/logを追加していない。
- 実Agent/broker別UID、socket path owner/mode、listener作成/permission、namespace、systemd composition、実client/VPS live E2Eは未実装・未確認であり、hermetic/cross-compile PASSで代替しない。
