---
task_id: "TASK-0056"
change_class: "product"
status: approved
qa_agent: "qa-agent-terra-medium"
qa_role: "independent-qa"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T16:31:48Z"
revision: 2
implementation_reviewed_at: ""
expectation_changed: false
expectation_change_approved_by: ""
---

# TASK-0056 QA PLAN

## 方針

同一candidateについて、package testはconstructor、activation環境、FD所有権、listener種別・address、Linux metadataおよび失敗時cleanupを直接失敗させられることを確認して再実行する。Linux対象はこのmacOS hostでは実行せず、`/usr/bin/true`をtest executorにしたcross-compileだけを行う。実systemd、FD 3の実配送、別UID/GIDのpermission、live socket connectは環境依存の`live-e2e`としてblockedのままとし、他のPASSで代替しない。

## 受け入れ条件との対応

| ケース ID | AC-ID | 観測方法 | 実施モード / 理由 |
|---|---|---|---|
| QA-001 | AC-1 | candidateのpackage testがabsolute/clean directory、正数・platform範囲内のUID/GID、root broker、nil/zero/corrupt Receiverをそれぞれ拒否し、valid inputだけimmutable Receiverとなることを検査する。error文字列にpath、UID/GID、environment、FD、下位errorを含めないnegative assertionを監査する。 | `focused-rerun` / pure constructor入力はhermeticかつdeterministicであり、共通のbounded commandで直接再現できる。 |
| QA-002 | AC-2 | candidate testが`LISTEN_PID=current PID`、`LISTEN_FDS=1`、`LISTEN_FDNAMES=egress`のcanonical完全一致だけを受理し、missing/duplicate/malformed/extra値を拒否することを確認する。認識済み環境のone-shot clear、FD 3一回だけのconversion、conversion後original FD close、close/conversion failureの拒否、fallback/retry/cache/goroutineなしを失敗検出できるassertionで監査する。 | `focused-rerun` / fake sourceとtemporary FDでhermeticに所有権と環境消費を再現できる。 |
| QA-003 | AC-3 | common testがconcrete `*net.UnixListener`以外、wrong network/path、FD type/address mismatchを拒否すること、取得後のdirectory/socket metadata mismatch又はvalidation failureでlistenerをcloseしsocket pathをunlinkしないことを確認する。Linux build-tag testがactual Unix listenerのmetadata readerを検査し、non-LinuxがFD/envからlistenerを返さないfail-closed分岐を含むことを監査する。 | `focused-rerun` / common package testはこのhostで実行し、Linux adapterは下記compile-onlyでbuild対象にする。 |
| QA-004 | AC-4 | candidate diffとhermetic provision/template/parser testsを監査し、socket unitに固定のListenStream、SocketUser、SocketGroup、SocketMode、FileDescriptorName、RemoveOnStopだけがあり一socketであることを確認する。tmpfiles/provision manifest/validator/testがruntime directoryを`broker:agent 0710`で一致させ、他3 directory action、user/service action、deny default、install時non-enable/start、configure/build/install/dist/uninstallのunit inclusionをそれぞれ壊す変更を検出することを確認する。 | `evidence-review` / generated configure、dist/install配線とtemplate固定値はcandidate-bound diff・DEV実行証跡・既存testのfailure-detectionを独立監査する。root/harness checksはここでのDEV証跡監査だけであり、QAの再実行PASSにはしない。 |
| QA-005 | AC-5 | candidateのfocused rerun結果、Linux compile-only結果、test sourceのnegative cases、candidate-bound `make check`、`make distcheck`、root `make lint-docs`、candidate launcher root `make check`、`git diff --check`のDEV証跡を照合する。許可path以外の変更、外部dependency、追加＋削除1,000行超過がないことをdiff統計で確認する。 | `evidence-review` / root/harness checksはDEVが一回実行したcandidate証跡の完全性を監査する対象であり、QAがroot作業を再実行しない。 |
| QA-006 | AC-2, AC-3, AC-4 | 実systemd managerがunitを読んで固定FD 3を配送し、broker/agent別UID/GIDでruntime traversalとsocket connectだけが許可され、停止時にsystemdがsocketをcleanupすることを実環境で確認する。 | `live-e2e` / 現hostはmacOSでありsystemd、FD 3 delivery、別UID/GID permission、live socket connectを安全に検証できないため`blocked`。unit/parser、hermetic test、cross-compileのPASSで置換しない。 |

## 実行するfocused rerun

candidateを固定後、以下を一つのbounded commandとして実行する。前半はplatform-independent package test、後半はLinux build-tag対象を実行せずcompile-onlyにする。`/usr/bin/true`はこのhostで存在確認済みのexecutorである。

```sh
cd tools/dev-agent-harness && GOCACHE=$PWD/.build/go-cache go test -count=1 ./internal/socketactivation && GOOS=linux GOARCH=amd64 GOCACHE=$PWD/.build/go-cache go test -run '^$' -exec /usr/bin/true ./internal/socketactivation
```

期待結果は両方zero exitである。前半の失敗はcandidate implementation/test/fixture failureとして、後半の失敗はLinux compile failureとして分類する。executorによる後半のzero exitはLinux runtime、systemd activation、実FD delivery又はpermission/connectの証拠ではない。

## 境界・異常・回帰

- constructorの拒否診断は固定・非漏洩であり、host path、numeric identity、environment値、FD番号、wrapped OS errorを露出しないこと。
- activation値は緩い数値parseや余分なseparator/spaceを許容せず、recognized failureも含め一回clearすること。unrecognized/malformed入力からlistenerを返さないこと。
- FD 3は複製listenerへの一回だけの移管であり、original FDがcloseされる一方、validationに失敗したlistenerはcloseされ、socket inodeをserviceがunlinkしないこと。
- Unix domain以外、abstract/renamed/wrong path、symlink/non-socket、directoryとsocketのowner/group/mode drift、current EUID mismatch/rootはfail closedであること。
- `broker:agent 0710`はdirectoryのみに適用され、socket `broker:agent 0660`と混同しないこと。既存state/config/audit directoryのownership/modeは変えないこと。
- installは配置だけでありenable/start/restartを追加せず、uninstall・dist・generated configureのいずれにもsocket unitの欠落又は重複がないこと。
- non-Linuxでactivation listenerを受理する実装、process bind/chmod/chown/unlink/stale recovery、multiple FD、TCP/vsock、config schema・依存追加、許可path外の変更は回帰としてFAILに分類する。

## 実装後の再確認

- [ ] candidate commitを固定し、QA-001からQA-005のcandidate-bound evidenceとfocused rerunを独立に確認した。
- [ ] QA-001からQA-005でtestが該当する誤実装を実際に失敗させられるnegative assertionを持つことを確認した。
- [ ] QA-006は現hostで`blocked`のままとし、実施環境・安全なcleanup手順が承認されるまでPASSを記録しない。
- [ ] root/harness checksはDEV証跡としてだけ監査し、QA自身の再実行結果又はlive-e2e PASSと誤記していない。
- [ ] 実装差分とレビュー結果を確認した。
- [ ] 操作手順と期待結果を現行実装に合わせた。
- [ ] 期待結果または範囲を変更した場合、main Agentの承認を得た。

## 改訂履歴

| 改訂 | 日付 | 変更者 | 変更内容 | main承認 |
|---:|---|---|---|---|
| 1 | 2026-08-02 | QA | 独立QA計画を作成 | `pending` |
| 2 | 2026-08-02 | Main | cache条件を固定したbounded commandへ具体化し承認 | `approved` |
