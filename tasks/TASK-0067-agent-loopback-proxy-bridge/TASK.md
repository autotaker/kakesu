---
task_id: "TASK-0067"
title: "Agent用loopback proxy bridgeを実装する"
status: draft
created_at: "2026-08-02"
---

# TASK-0067 Agent用loopback proxy bridgeを実装する

## `Planning input packet`（Main Agent所有）

このsectionをPlannerとQAへ渡す唯一の`planning input packet`とし、各内容をPLAN/QA_PLANへ複製しない。

### 目的

Git/gh/OpenAIなどTCPのHTTP proxy endpointだけを扱うAgent clientを、既存のpeer-bound egress Unix socketへ接続できるようにする。その前段として、固定Unix socketへだけ転送するAgent側loopback bridgeを、launcherや環境設定から独立した小さなライフサイクル境界として実装する。

### 対象と対象外

#### 対象

- `internal/proxybridge`へ、IPv4 loopbackのephemeral portだけをlistenし、接続ごとにconfigure/link時固定予定のegress Unix socketへ一回だけdialするbridgeを追加する。
- callerが子clientへ渡せるcanonical `http://127.0.0.1:<port>` endpointを返し、任意bind address、port、Unix path、TCP upstream又はenvironment overrideを公開しない。
- bounded concurrency、dial timeout、双方向stream、half-close、context cancellation、listener/connection cleanupを実装する。
- accept/dial/copy/close failureを入力値と下位errorを含まない固定error又は接続単位の拒否へ畳み、別socket、retry、fallbackを行わない。
- fake listener/dialerと`net.Pipe`によるhermetic race test、READMEの実装境界を追加する。

#### 対象外

- `dev-agent-launcher`のCLI/子process起動、signal/environment allowlist、temporary directory、CA trust file、Git config、credential helper設定は実装しない。
- proxy CA取得、Opaque capability発行、HTTP CONNECT/TLS/inner HTTP policy、credential置換、DNS/upstream転送の意味を変更しない。
- HTTPをbridge内でparseせず、認可・主体binding・header/body上限をloopback portへ移さない。
- wildcard、IPv6、固定port、Agent入力のaddress/path、TCP/別Unix socket fallback、retry、cache、監査/log、実秘密を追加しない。
- 実OS network namespace、loopback到達性、Unix socket permission/peer UID、実Git/gh/OpenAI、systemd/VPSをhermetic PASSで代替しない。
<!-- safety_contractの場合: 製品コード、test、runtime/build設定、Schema、製品依存、生成製品入力/成果物、外部観測可能な挙動を変更しない。 -->

### 受け入れ条件

<!-- AC-IDはTask内で一意かつ安定させ、観測可能な結果をここに一度だけ記載する。 -->

- [ ] AC-1: bridgeは`tcp4`のexact `127.0.0.1:0`だけを一回listenし、OS割当portを検証してcanonical `http://127.0.0.1:<port>`だけを返す。wildcard/IPv6/non-loopback/fixed又は不正port、nil/typed-nil listenerを開始前に固定拒否し、address/portをCLI・environment・caller入力から選ばせない。
- [ ] AC-2: accepted client接続ごとに、trusted constructorが保持するabsolute clean egress Unix socket pathへ`unix`で一回だけtimeout付きdialする。dial failure時はそのclientをcloseし、retry、別path/network、payload forwarding、診断へのaddress/path/lower-error露出を行わない。
- [ ] AC-3: dial成功後はbytesを変形・保持せず双方向にstreamし、各方向のEOFを反対側のwrite half-closeへ伝える。context cancellation又はconnection failureは両端をcloseし、Serve終了時に全connection goroutineを回収する。CONNECT/HTTP/TLSを解釈せず既存egress serviceを認可の正本に保つ。
- [ ] AC-4: concurrencyは1〜64の固定上限内で、上限到達中は追加接続のUnix dialを開始しない。listener acceptの予期しないfailureは新規acceptを止め、active connectionをcancel/drainして固定server errorを返す。親context cancellationはlistenerとactive connectionを閉じ、正常終了する。
- [ ] AC-5: candidateは新規package source/testとREADMEの3パス、約650〜950 changed linesを目安とし、launcher/command/Makefile/config/dependency/Schema/Kakesu runtime/generated file/live state/実秘密を含まない。focused race、harness `make check`/distcheck、candidate gateのroot `make check`、`git diff --check`がPASSする。

### 安定した参照

| 参照ID | 対象 | 固定改訂/ダイジェスト | 用途 |
|---|---|---|---|
| REF-1 | Harness設計 §6.2、§8.1、§14 Phase 2 | main `d894220` | Agent namespace内loopbackと固定egress入口、実client read経路の境界 |
| REF-2 | TASK-0051 connect session / broker listener | main `d894220` | Unix socket側がpeer bindingとstrict CONNECT/control認可を所有する既存契約 |
| REF-3 | TASK-0065 / TASK-0066 | main `d894220` | Git helperと公開CA clientが同じ固定egress socketを使う後続launcher入力 |

### 依存状態

| 依存 | 状態 (`ready` / `pending`) | planning参照 | `ready`後に固定する値 |
|---|---|---|---|
| TASK-0051 / TASK-0066 | `ready` | main `d894220` | peer-bound egress Unix socket、strict CONNECT/control、public CA取得境界 |

### 許可パス

- `tools/dev-agent-harness/internal/proxybridge/bridge.go`（新規）
- `tools/dev-agent-harness/internal/proxybridge/bridge_test.go`（新規）
- `tools/dev-agent-harness/README.md`

### 完了経路preflight

| 確認対象 | 結果 | コマンドまたは根拠 |
|---|---|---|
| 完了checker | `ready` | 通常product 3トランザクション、同一candidate独立REVIEW/QA、no-ff completionを使用する |
| 権限 | `ready` | loopback listener、concurrency、Unix bridgeの安全境界を変更するため`dev-sol`。Mainだけがstage/commit/merge/pushする |
| 依存状態と参照 | `ready` | TASK-0051/0066がmainへ反映済み |
| 生成物の有無と更新方法 | `ready` | Go source/testとREADMEのみ。configure/dependency/generated fileなし |
| 割当ワークツリー | `ready` | `worktrees/TASK-0067-agent-loopback-proxy-bridge` |
| Lapログの書込・Schema・`repository annotation` | `not-applicable` | 新規log/Schema/annotationなし |

### 未決事項

- なし

### `Dependency-ready reconciliation`

<!-- 依存ready時にMainがready参照、planning参照との差分、AC/設計/スコープ/QAへの影響、再承認結果を追記する。依存なし又は未readyならN/Aとする。 -->

- N/A

## 背景

TASK-0065でGit credential helper、TASK-0066で公開CA取得clientまで完成したが、Git/libcurl系clientは現在のUnix socketをHTTP proxy URLとして直接使えない。launcherへCA file、environment、process exec、bridgeを一度に実装すると、network lifecycleとfilesystem/process cleanupのfailureを単体で判定できない。先にraw byte bridgeだけを完成させ、次Taskでlauncherがそのendpointと公開CAを子processへ束縛する。

## 検討すべき設計観点

- bridgeは認可proxyではない。loopbackから来たbytesを既存peer-bound Unix serviceへ運ぶだけで、CONNECT/TLS/HTTP policy、subject binding、capability消費はegress serviceに残す。
- bind endpointはAgent network namespace内だけで使う前提だが、本Taskのhermetic testはnamespace isolationを証明しない。host namespaceでの運用可を主張しない。
- body sizeやsession durationをbridge独自に制限するとGit/OpenAI streamの意味を変えるため、concurrencyとdial phaseだけをboundedにし、active streamはcontext/EOFで終了する。
- client EOF、upstream EOF、片方向failure、cancel、accept failureがgoroutine/FD leak又は全connectionの無期限待ちにならないことをrace testで確認する。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] 選択した`change_class`の完了経路と`make check`を満たしている。
- [ ] 製品変更の場合: 実装、テスト、文書、同一案の独立REVIEW/QA、完了後の環境依存ケース確認が完了している。
- [ ] 安全契約変更の場合: Mainの意図・スコープ・受け入れ経路確認、契約検査、許可された統制文書差分の確認が完了している。

## 関連コンテキスト

### 意味 Wiki

- 未調査

### 判断

- 未調査

### 適用しなかった重要な判断

- なし
