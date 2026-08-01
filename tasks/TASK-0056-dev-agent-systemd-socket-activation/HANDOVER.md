---
task_id: "TASK-0056"
status: complete
completed_at: "2026-08-01T17:16:30Z"
candidate_commit: "40ccc20cb2e72fa1f7d51e1f6c6b0e714f179562"
---

# TASK-0056 HANDOVER

## 成果

- systemd所有の固定Unix socketを、継承FD 3とactivation環境から一回だけ受領する`socketactivation.Receiver`を追加した。
- Linuxではprocess identity、directory FDから開いたsocket node metadata、listener pathを照合し、非Linuxではfail closedにした。
- runtime directoryを`broker:agent 0710`へ変更し、socket unitとconfigure/install/dist配線を追加した。

## candidate-bound DEV証跡

| コマンド/テスト | 結果 |
|---|---|---|
| initial candidate launcherのroot `make check` | `PASS`。initial candidate `85ab4f0739345a43b9984d08835b2560002f259a`を作成 |
| review修正後のroot `make check` | `PASS`。検査済みtreeをcandidate `40ccc20cb2e72fa1f7d51e1f6c6b0e714f179562`へsquash固定 |
| `GOCACHE=$PWD/.build/go-cache go test -count=1 ./internal/socketactivation` | `PASS`（macOS common/non-Linux tests） |
| `GOOS=linux GOARCH=amd64 GOCACHE=$PWD/.build/go-cache go test -run '^$' -exec /usr/bin/true ./internal/socketactivation` | `PASS`（compile only） |
| harness `make check` / `make distcheck` | `PASS` |
| root `make lint-docs` / `git diff --check` | `PASS` |

## 主要な変更

- `Receiver.Take`はcanonicalな`LISTEN_PID`、`LISTEN_FDS=1`、`LISTEN_FDNAMES=egress`だけを認識し、環境を消去してFD 3を一回だけ変換する。original FDは変換後にcloseし、失敗listenerもcloseする。
- concrete `*net.UnixListener`と固定pathだけを受理する。Linux adapterはcurrent non-root EUID、runtime directory `broker:agent 0710`を検査し、そのdirectory FDから`openat(O_PATH|O_NOFOLLOW)`したsocket nodeが`broker:agent 0660`のUnix socketであることを照合する。
- Linuxでは上限付き`/proc/self/environ` snapshotからraw `LISTEN_*`出現数を数え、canonical値でも同名keyが重複すれば拒否する。
- `dev-agent-egress.socket`が固定path、user/group/mode、FD名、停止時cleanupを宣言する。tmpfiles/provisionはruntime directoryだけを変更し、他directory、action数・順序、service非enable/startを維持する。
- core testsはrules、固定診断、activationのone-shot、canonical near-miss・missing・duplicate、FD/listener close ownershipとsingle conversion、type/address失敗を検出する。Linux test sourceはactual Unix socket metadataとmode/owner/group/type/symlink drift拒否、node保持を検査する。
- READMEへsystemd／provision／Receiverの責務とlive確認境界を記録した。

## 検証結果

- candidateは`40ccc20cb2e72fa1f7d51e1f6c6b0e714f179562`、baseはplanning `68dc14ac38335eb116586c61ffca21c9e07e6507`である。
- base...candidateは許可13 filesだけ、追加782行・削除13行、合計795行で1,000行以下。外部dependency、新config field、新check、service compositionはない。
- `configure.ac`変更後に`autoconf`で`configure`を再生成した。configure/build/install/dist/uninstallは新socket unitを既存unit列から扱う。
- initial candidateの独立REVIEW/QAは、pathname socketとlistener FDのinode比較によるLinux常時拒否と、duplicate env拒否・negative test不足を検出した。再評価でFD factory/conversion call-countと個別metadata driftの検出不足を解消し、root `make check`済みの同一treeをcompletion gate互換の単一candidateへsquash固定した。

## 判断・既知の制約

- systemdがsocketのbind、owner/group/mode、停止時cleanupを所有し、Receiverはbind/chmod/chown/unlink、fallback、retry、cache、goroutine、logを行わない。
- runtime directoryはAgent groupにtraverseだけを与え、write/list権限を与えない。socket nodeはAgent groupへconnectに必要な権限だけを与える。
- 実systemd managerによるFD 3配送、実broker/agent別UID/GID、実socket permission/connect、停止時cleanup、VPS live E2Eは現在のmacOS hostでは未実施であり、hermetic testやLinux compile-onlyで代替しない。
