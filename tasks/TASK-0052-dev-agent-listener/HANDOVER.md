---
task_id: "TASK-0052"
status: complete
completed_at: "2026-08-01T14:27:06Z"
candidate_commit: "0d8ccc8d535a1bc8accc3a4cea376e411eac008c"
---

# TASK-0052 HANDOVER

## 成果

- 注入済み`net.Listener`を所有し、slot-before-Acceptで同時処理数を制限する`brokerlistener.Server`を追加した。
- trusted PeerBinderのSubjectを検証・copyしてprivate context keyへ束縛し、既存Sessionへ一回だけ渡すcontext-only `Resolver`を追加した。
- per-connection error/panic隔離、unexpected Accept failure、caller cancel/deadline、協調Binder/Session drainを標準context/semaphore/WaitGroupだけで固定した。

## candidate-bound DEV証跡

| コマンド/テスト | 結果 |
|---|---|---|
| candidate launcherのroot `make check`（固定candidateで一回） | `PASS` |
| `GOCACHE=$PWD/.build/go-cache go test -count=1 -race ./internal/brokerlistener` | `PASS`（最終1.56s） |
| `make -C tools/dev-agent-harness distcheck`（固定candidate） | `PASS` |
| `make lint-docs`（README最終文面） | `PASS` |
| `git diff --check` | `PASS` |

## 主要な変更

- `internal/brokerlistener`にPeerBinder/Session interfaces、immutable Server、private Subject context/Resolver、1〜64のbounded accept lifecycleと固定errorを追加した。
- slot取得後かつcancel再確認後だけAcceptし、binder拒否/invalid Subject/binder・Session error/panicをconnection-local closeへ畳んで次のacceptを継続する。
- caller cancel/deadline又はunexpected Accept failureでlistenerを一度だけcloseし、全connection contextをcancelして協調処理をdrainする。callbackを別goroutineへ逃がすtimeoutを作らない。
- READMEに待受け合成の責務と、実bind/OS identity/systemd/network namespace/VPSを未実装・未確認とする境界を追記した。

## 検証結果

- candidate `0d8ccc8d535a1bc8accc3a4cea376e411eac008c`を固定した。base `653746b07a062e5825261ba7a3f1627cc68f9362`からの製品差分は許可3 files、追加993行・削除0行である。
- focused raceはprivate Resolver、slot-before-Accept、binder-before-session、subject copy/validation、上限、per-connection error/panic、unexpected Accept、active cancel/deadline、協調Binder drain、全path connection closeを検出してPASSした。
- 固定candidateでcandidate launcherのroot checkとdistcheckがPASSし、README最終文面のlintとdiff checkもPASSした。

## 判断・既知の制約

- PeerBinderとSessionはtrustedかつ渡されたcontextのcancel/deadlineへ協調してreturnする依存契約である。Serverは任意の非協調callbackを強制停止せず、timeout goroutineでleakを隠さない。
- SubjectはUID正数、identifier 1〜128 byte・先頭英数字・残り英数字又は`._-`として検証し、private contextへの束縛前とResolver返却時にcopyする。公開setter、RemoteAddr、protocol headerからの補完はない。
- 実socket生成/bind、Linux peer credential、Agent/UID/workspace mapping、network namespace、systemd、実client/VPSは未実装・未確認であり、fake listener PASSで代替しない。
