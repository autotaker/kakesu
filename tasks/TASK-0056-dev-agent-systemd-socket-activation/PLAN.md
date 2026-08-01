---
task_id: "TASK-0056"
change_class: "product_change"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "luna-xhigh"
approved_dev_profile_reason: "新規 internal/socketactivation の限定したFD受領・Linux metadata adapter・hermetic testと、既存deploy/provisionの固定desired-state差分、configure再生成だけに閉じる。既存service binaryのcomposition、config field、依存、check、実systemd操作を変更しないため。"
approved_dev_profile_risk_signals: []
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T16:31:48Z"
planning_reviewed_by: ""
planning_review_decision: "pending"
planning_reviewed_at: ""
classification_approved_by: ""
classification_approved_at: ""
classification_approval_reason: ""
# safety_contract_version: 2
# safety_contract_planned_paths: []
# safety_contract_generated_paths: []
---

# TASK-0056 PLAN

## 分類・前提

これは製品変更である。TASK-0055 が固定した accepted `*net.UnixConn` の peer identity 境界を変更せず、その前段で
systemd 所有の listener を一回だけ受領して検証する新規 package を加える。socket path の作成、所有者・mode、停止時
cleanup は socket unit、runtime directory の desired state は tmpfiles と provision manifest、継承FDの取得・検証・
Go listener への所有権移管は `socketactivation` がそれぞれ所有する。

DEV は承認済み profile `luna-xhigh` を使用し、planning / candidate / completion の通常3 commit経路に従う。初回
candidate は許可パスだけを変更し、追加・削除合計1,000行以下に収める。新dependency、新config field、新check、
service binaryのcomposition、process-created bind/unlink は追加しない。`configure.ac` を更新した後、repo所定の
`autoconf` により生成済み `configure` を再生成して同一candidateへ含める。

## AC対応

TASKの条件本文を再掲せず、`planning input packet`のAC-IDに設計を対応させる。

| AC-ID | 設計判断 | 変更パス | 実施順序 | 失敗時の扱い |
|---|---|---|---|---|
| AC-1 | `Rules`/`Receiver` は runtime directory、broker UID、agent GID を初期化時に検査・copyし、basename `egress.sock` から唯一の期待pathを導く。numeric IDは正数かつOS境界へlosslessに収まるもの、brokerはnon-root、directoryはabsolute/cleanだけを許す。packageの固定errorと固定Formatを用い、test seam は非公開に留める。 | `tools/dev-agent-harness/internal/socketactivation/` | 1 | nil/typed-nil/zero/corrupt Receiver、root・範囲外ID、不正pathをlistenerなしの固定非漏洩errorに畳む。path、UID/GID、env、FD、下位errorは診断に出さない。 |
| AC-2 | `Take` は current PID と完全一致するcanonical `LISTEN_PID`、`LISTEN_FDS=1`、`LISTEN_FDNAMES=egress`だけを受理し、FD 3を一回だけ `net.FileListener` へ渡す。認識済みactivation envは取得試行時に消去し、複製が得られたらoriginal fileを必ずcloseしてlistener所有権だけを返す。 | `tools/dev-agent-harness/internal/socketactivation/` | 2 | missing/duplicate/malformed env、追加FD、FD conversion/type mismatch、original close failureは固定errorで拒否する。fallback bind、cache/retry、別goroutine、logを作らず、取得済みlistenerはcloseする。 |
| AC-3 | Linux adapter は一descriptor/metadata境界で current EUID、runtime directory、socket nodeを読み、`UnixListener` のnetworkとaddressを唯一の期待pathへ照合する。directory は `broker:agent 0710` の実directory、socketはsymlinkでないUnix socketかつ `broker:agent 0660`だけを許す。non-Linux adapterは一律denialにする。 | `tools/dev-agent-harness/internal/socketactivation/` | 3 | EUID/root、stat/type/owner/group/mode/address不一致、metadata取得失敗はlistenerをcloseし固定errorにする。失敗時もpathをunlinkせず、TCP・abstract・別pathを返さない。 |
| AC-4 | `dev-agent-egress.socket.in` は fixed `ListenStream`、SocketUser/Group/Mode、`FileDescriptorName=egress`、`RemoveOnStop=yes`の一socketのみを宣言する。tmpfiles と manifest の runtime directory だけを `broker:agent 0710` へ合わせ、他3 directory、user/service action、deny default、install時non-enable/startを保持する。configureの出力列とMakefileのunit列を過不足なく同期する。 | `tools/dev-agent-harness/deploy/systemd/dev-agent-egress.socket.in`、`deploy/tmpfiles/dev-agent-harness.conf.in`、`internal/provision/provision.go`、`internal/provision/provision_test.go`、`Makefile.in`、`configure.ac`、`configure` | 4–5 | template/manifestの不整合、unitのinstall/dist/uninstall漏れ、runtime以外のownership/mode変化はtestまたはdistribution検査でfailとする。unit配置はservice enable/startを行わない。 |
| AC-5 | core testはnon-public env/FD/stat seamでconstructor、exact env、one-shot clear、FD 3 conversion、original/listener close ownership、type/address、metadata、cleanup、fixed diagnosticを検出する。Linux build-tag testは実temp Unix listenerに対するmetadata readerを検査し、READMEには責務分離とlive境界だけを記す。 | `tools/dev-agent-harness/internal/socketactivation/`、`internal/provision/provision_test.go`、`README.md` | 6 | hermetic/cross-compile PASSを実systemd、実別UID/GID、実permission/connect、VPSの証拠にしない。live-e2eはblockedのままとし、別モードのPASSで代替しない。 |

## 責務・境界・不変条件

- systemd socket unit は `/run/dev-agent-harness/egress.sock` のbind、socket node `broker:agent 0660`、`RemoveOnStop` cleanupを唯一所有する。tmpfiles と provision は directory `/run/dev-agent-harness` の `broker:agent 0710` desired stateだけを所有し、Agentにはtraverse/connectに必要なgroup権限だけを与える。
- `socketactivation.Receiver` はprocessが継承したFDを一回だけ受領し、Linux上でcurrent EUID、directory/socket metadata、listenerのconcrete type/network/pathを照合するだけである。socketを作成・chmod/chown・unlinkせず、listenerをServer/PeerBinderへ接続せず、config又はusername lookupを持たない。
- activation environmentは緩い数値parseや部分一致を許さず、recognized inputをclearして同一processでの再消費を防ぐ。`net.FileListener`が複製したFDは新listenerが所有し、original FDは成功・失敗を問わずpackageの取得処理でcloseする。
- Linux固有のOS/metadata読取はbuild-tag adapterに隔離し、non-Linuxはenv/FDが正しく見えてもlistenerを返さない。metadata検証失敗はlistenerをcloseするが、filesystem nodeは削除しない。
- existing `dev-agent-egress` binary、service unit、`brokerlistener`/`peerbinder`/Server/Session composition、config schema、sysusersの意味、provisionのaction数/順序とservice non-enable/startは変更しない。

## 代替案と不採用理由

- processが `ListenUnix`、chmod/chown、stale inode検査/unlinkを行う案は、directory書込権限又はpath差替え・cleanup競合をserviceに持ち込むため採用しない。
- activation envをsd_listen_fds相当の許容parse又は複数FDへ広げる案は、唯一のFD 3 / `egress` 契約とone-shot消費を曖昧にするため採用しない。
- socket path、mode、UID/GIDをconfig fieldにする案は、固定配備契約に可変設定とnumeric name resolutionを加えるため採用しない。
- service unitやbinaryへReceiverを配線する案は、後続Taskのcomposition範囲であり、今回の受領・検証境界を超えるため採用しない。
- real systemd manager、sudo、実ユーザー分離、connect permission、VPSをtestする案はlive-e2eの前提とcleanup権限を満たさないため採用しない。新dependencyや新repository checkも導入しない。

## 変更予定

| パス | 変更内容 |
|---|---|
| `tools/dev-agent-harness/internal/socketactivation/` | immutable Receiverとfixed non-leaking diagnostics、private env/FD/metadata seams、one-shot FD 3 transfer、Linux descriptor/metadata validator、non-Linux denial、core hermetic testsとLinux-only metadata testを追加する。 |
| `tools/dev-agent-harness/internal/provision/provision.go` | canonical runtime directory recordだけを broker owner / agent group / `0710` に変更し、既存action count・順序・他directory/serviceの契約を維持する。 |
| `tools/dev-agent-harness/internal/provision/provision_test.go` | canonical JSONLとdirectory assertionsを更新し、runtime-only ownership/mode変更とvalidator拒否を検出する。 |
| `tools/dev-agent-harness/deploy/systemd/dev-agent-egress.socket.in` | systemdがsocket path、owner、group、mode、FD name、停止時cleanupを所有する一socket unit templateを追加する。 |
| `tools/dev-agent-harness/deploy/tmpfiles/dev-agent-harness.conf.in` | runtime directory一行だけを broker:agent `0710` に更新する。 |
| `tools/dev-agent-harness/Makefile.in` | socket unitを既存`SYSTEMD_UNITS`へ加え、install/uninstall/distcleanとdistribution inputsが同じunit列から追跡するようにする。 |
| `tools/dev-agent-harness/configure.ac`、`tools/dev-agent-harness/configure` | socket unitを`AC_CONFIG_FILES`へ登録し、`autoconf`で生成済みconfigureを正規再生成する。 |
| `tools/dev-agent-harness/README.md` | systemdとtmpfiles/provision/Receiverの責務、Linux-only fail-closed検証、実systemd・実UID/GID・permission/connectのlive-e2e未実施境界を追記する。 |

許可外の `cmd/dev-agent-egress`、existing service unit、config、dependency manifest、Task/QA_PLAN/Wiki、repository checkは変更しない。

## 実装手順

1. `socketactivation` coreにRules、immutable Receiver、fixed error/Format、path/ID検証と非公開test seamを定義する。期待socket pathはclean runtime directoryと固定basenameから一度だけ導く。
2. `Take` にenv exactness、recognized env clear、FD 3の一回conversion、original descriptor close、concrete `*net.UnixListener` gateとfailure cleanupを実装する。成功listenerはmetadata検証へだけ渡し、compositionは加えない。
3. Linux build-tag adapterでEUID、directory/socket nodeのdescriptor/metadataとlistener addressを検証し、non-Linux adapterはfixed denialを返す。失敗時closeは行うがunlinkは加えない。
4. core/Linux-only testsを追加し、invalid/corrupt constructor、env permutations、one-shot consumption、FD/listener ownership、Unix type/address、owner/group/mode/symlink、failure cleanup、fixed non-leaking diagnosticsを検出する。provision testのcanonical fixtureはruntime recordだけを更新する。
5. socket unitとtmpfilesを追加/更新し、`Makefile.in`と`configure.ac`へunitを接続する。`autoconf`で`configure`を再生成し、configure/build/install/dist/uninstallがunitを過不足なく扱うことを確認する。
6. READMEの境界を更新し、focused socketactivation/provision tests、Linux cross-compile、harness `make check`/`make distcheck`、root `make lint-docs`、candidate launcherのroot `make check`、`git diff --check`、許可pathと1,000行上限をcandidate前に確認する。同一candidateを独立REVIEW/QAへ渡す。

## 検証計画

| 検証 | 目的・主なケース | 実施責任 / 時点 |
|---|---|---|
| socketactivation hermetic core | Newのpath/UID/GID/typed-nil、canonical activation env、env clear一回、FD 3だけのconversion、UnixListener type/address、original FDとlistenerのclose ownership、metadata failure cleanup、固定error/Formatを確認する。 | DEV / candidate |
| Linux adapter test | temp Unix listenerのactual directory/socket metadata reader、current EUID/non-root gate、owner/group/mode/type/symlink/address拒否を確認する。 | Linux上のDEV/CI。現在の非Linux QAはsource監査とcross-compileだけで、実行PASSを主張しない。 |
| provision/template assertions | manifest JSONLでruntime recordだけが `broker:agent 0710`、他3 directoryと10 action順序、serviceのdisabled/not-startedが不変であること、tmpfiles/socket templateの固定値を確認する。 | DEV / candidate |
| build/distribution | focused package tests、`GOOS=linux GOARCH=amd64 go test -run '^$' ./internal/socketactivation`、harness `make check`、`make distcheck`、root `make lint-docs`、`git diff --check`、allowed path/line countを確認する。candidate launcherのroot `make check`は一回だけ実行する。 | DEV / candidate、REVIEW/QA / evidence-review |
| post-merge | mainで所定の `make task-check TASK=TASK-0056` を実行する。 | Main / completion |

実systemd manager、unit enable/start/restart、実broker/agent UID/GID、socketへの実connect/permission、namespace/VPSのlive E2Eは本Taskで安全に再現できない。これらは`live-e2e`をblockedのままとし、unit/parser/hermetic test又はcross-compileのPASSで代替しない。

## 移行・互換性

- 新規Receiverは呼出元へ検証済み `*net.UnixListener` を返せる状態にするだけで、現行binaryの通常起動・service wiring・外部endpointは変更しない。後続compositionが導入されるまでlistenerをprocessが作る互換fallbackはない。
- Linux以外ではbuildできてもactivation listenerを取得しない。TCP、vsock、abstract Unix namespace、複数socketはsupportしない。
- `configure` はtarballに含める生成物であるため、`configure.ac`と同一candidateで正規再生成する。新しいmigration、config input、dependency、persistent stateはない。

## 未解決事項

- なし

## main Agentレビュー

- [x] TASKの全AC-IDへ設計判断、パス、順序、失敗時の扱いを対応させ、条件本文を複製していない。
- [x] socket unit、tmpfiles/provision、Receiverのpath/metadata/FD ownershipの責務分離とfail-closed境界を具体化している。
- [x] service composition、config/dependency/check、process bind/unlink、real systemd/live E2Eを範囲外に固定している。
- [x] QA_PLANがTASK-firstで独立作成されている。
- [x] `dependency-ready reconciliation`と完了経路preflightが完了している。
- [x] DEV開始を承認した（approved_dev_profile: `luna-xhigh`）。

安全契約変更でv2契約を選ぶ場合は、コメントを外して`safety_contract_version: 2`と予定パス・生成パスの配列を記録し、DEV前に`make task-preflight TASK=TASK-0056`を実行する。変更しない種別は空配列とし、通常の予定パスと生成パスを重複させない。独立計画レビューのPASSとMainの分類承認をフロントマターへ記録する。分類変更時はTask、PLAN、QA_PLANを再承認し、承認者と時刻を更新する。
