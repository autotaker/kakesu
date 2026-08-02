---
task_id: "TASK-0066"
title: "Agent用proxy CA取得境界を実装する"
status: draft
created_at: "2026-08-02"
---

# TASK-0066 Agent用proxy CA取得境界を実装する

## `Planning input packet`（Main Agent所有）

このsectionをPlannerとQAへ渡す唯一の`planning input packet`とし、各内容をPLAN/QA_PLANへ複製しない。

### 目的

Git/gh/OpenAI clientが外向き通信proxyの短命leafを検証するために必要な公開CA certificateだけを、既存のpeer-bound egress Unix socketから取得できるようにする。CA private keyやbroker credential directoryをAgentへ公開せず、TASK-0065と同じstrict control clientから後続launcherが一時trust fileを作れる直前までを完成させる。

### 対象と対象外

#### 対象

- `connectsession`のcontrol面へexact `GET /v1/proxy-ca`一操作を追加し、既存`proxyca.Authority`が保持するcertificate-only public PEMのcopyだけを返す。
- `Authority`境界をpublic certificate accessorまで明示し、nil/malformed/noncanonical/non-CA/private materialをresponse前に拒否する。
- `controlclient`へ一接続一操作のCA取得を追加し、唯一のbounded 200 PEM response、Content-Length、close/EOF、x509 CA constraintsをclient側でも独立検証する。
- net.Pipe/fake Authorityによるhermetic testsでexact request/response wire、既存Issue/Revoke/CONNECT回帰、deadline/one dial/close、上限、非漏洩、copy isolationを確認する。
- READMEへ公開CA取得境界と後続launcher/live境界を記録する。

#### 対象外

- CA certificate/private keyの生成、rotate、reload、disk書込み、trust store登録、一時file、launcher、environment/Git configを実装しない。
- CA private key、broker credential file/path、実token、Opaque handle、repository、subjectをresponse又は診断へ追加しない。
- HTTP CONNECT/TLS leaf発行、Capability Registry、Issue/Revoke意味、Git helper、provider forwarding、systemd socket、peer binderを変更しない。
- TCP、別socket fallback、runtime socket override、cache、retry、redirect、複数certificate chain又は一般purpose certificate endpointを追加しない。
- 実OS socket/permissions/別UID、実Git/libcurl trust、GitHub/OpenAI/DNS/TLS、systemd/VPSをhermetic PASSで代替しない。
<!-- safety_contractの場合: 製品コード、test、runtime/build設定、Schema、製品依存、生成製品入力/成果物、外部観測可能な挙動を変更しない。 -->

### 受け入れ条件

<!-- AC-IDはTask内で一意かつ安定させ、観測可能な結果をここに一度だけ記載する。 -->

- [ ] AC-1: exact `GET /v1/proxy-ca HTTP/1.1`、canonical zero Content-Length、追加header/body/early byteなしの一操作だけがcontrol routeへ入り、valid Authorityからcertificate-only public CA PEMのfresh copyだけを200で返す。malformed method/path/query/header/body、nil/invalid Authority outputは既存固定403へ畳み、CONNECT/Issue/Revokeの意味を変えない。
- [ ] AC-2: serverは1〜4,096 byteの単一canonical `CERTIFICATE` PEMをx509 parseし、self-signed ECDSA P-256 CA、BasicConstraints、CertSignを確認してから、exact Content-Type、canonical Content-Length、Connection close、bodyを一回だけwriteする。private key、複数block、trailing byte、expired/not-yet-validを成功responseへ出さない。
- [ ] AC-3: `controlclient`はabsolute Unix socketへtimeout/deadline付きで一回だけdialし、exact CA requestを送り、唯一のbounded 200 response、fixed header order/framing、body length、直後EOF、同じCA constraintsだけを受理する。chunked/extra/duplicate header/body/bytes、1xx/204/3xx/403/5xx、early EOF、private/multiple/noncanonical PEMは固定errorとなる。
- [ ] AC-4: 成功clientはcaller-owned public PEM copyだけを返し、失敗error/format/stdout/stderrにPEM、subject、socket、path、下位errorを保持しない。既存Issue/Revoke、helper get/store/erase、CONNECT/TLS testsは回帰せず、cache/retry/fallback/file/environmentを追加しない。
- [ ] AC-5: candidateは承認済み6パス・約800〜1,100行以内で、実秘密、launcher/config mutation、dependency、Schema、Kakesu runtime、生成file又はlive stateを含まず、focused race、harness `make check`/distcheck、root `make check`、`git diff --check`がPASSする。

### 安定した参照

| 参照ID | 対象 | 固定改訂/ダイジェスト | 用途 |
|---|---|---|---|
| REF-1 | Harness設計 §8.2、§14 Phase 2 | main `c829dcf` | Agentへ公開CAだけを渡し、private keyをbroker境界へ残す契約 |
| REF-2 | TASK-0050 Proxy CA / TASK-0051 connect session | main `c829dcf` | Authority public copyとstrict control framing/closeの既存境界 |
| REF-3 | TASK-0065 control client | main `c829dcf` | fixed Unix dial、deadline、strict response、non-leak client pattern |

### 依存状態

| 依存 | 状態 (`ready` / `pending`) | planning参照 | `ready`後に固定する値 |
|---|---|---|---|
| TASK-0050 / TASK-0051 / TASK-0065 | `ready` | main `c829dcf` | `proxyca.Authority.PublicCertificatePEM`、control route、client strict transport |

### 許可パス

- `tools/dev-agent-harness/internal/connectsession/session.go`
- `tools/dev-agent-harness/internal/connectsession/session_test.go`
- `tools/dev-agent-harness/internal/egressservice/service_test.go`
- `tools/dev-agent-harness/internal/controlclient/client.go`
- `tools/dev-agent-harness/internal/controlclient/client_test.go`
- `tools/dev-agent-harness/README.md`

### 完了経路preflight

| 確認対象 | 結果 | コマンドまたは根拠 |
|---|---|---|
| 完了checker | `ready` | 通常product 3トランザクション、同一candidate独立REVIEW/QA、no-ff completionを使用する |
| 権限 | `ready` | CA trust/private-key境界とcontrol protocol変更のため`dev-sol`、Mainだけがstage/commit/merge/pushする |
| 依存状態と参照 | `ready` | TASK-0050/0051/0065がmainへ反映済み |
| 生成物の有無と更新方法 | `ready` | Go source/testとREADMEのみ。dependency/configure生成物なし |
| 割当ワークツリー | `ready` | `worktrees/TASK-0066-proxy-ca-control-client` |
| Lapログの書込・Schema・`repository annotation` | `not-applicable` | 新規log/Schema/annotationなし |

### 未決事項

- なし

### `Dependency-ready reconciliation`

<!-- 依存ready時にMainがready参照、planning参照との差分、AC/設計/スコープ/QAへの影響、再承認結果を追記する。依存なし又は未readyならN/Aとする。 -->

- N/A

## 背景

TASK-0065でGit credential helperはOpaque capabilityを取得できるようになったが、Git/libcurlがproxyのMITM leafを検証するための公開CAをAgent側へ安全に渡す経路がない。broker専用credential directoryをAgent-readableにするとprivate key隣接面とOS permission設計が広がるため、既にpeer-boundでAgentから到達可能なegress control socketから公開certificateだけを取得する小さな境界を先に完成させる。file lifetime、proxy bridge、Git config、実clientは後続Taskへ分離する。

## 検討すべき設計観点

- control routeは既存CONNECT/Issue/Revokeと同じ一接続一操作/strict framingを維持し、generic file download又は任意Authority data endpointにしない。
- serverは`Authority`の実装を盲信せず公開PEMを再検証し、private block、chain、trailing data、invalid/expired CAをwireへ出す前に拒否する。clientもserver parserを共有せず同じsecurity propertyを独立検証する。
- CA certificateはsecretではないが、PEMや内部pathをfailure diagnosticへ含めない。callerへ返すsliceはtransport buffer/stateとaliasさせない。
- peer identityは`brokerlistener`/`peerbinder`がSession到達前に成立させる既存境界を維持し、このGETだけ自己申告subject又はqueryを追加しない。
- 後続launcherはこのpublic PEMを一時trust fileへ書くが、本Taskではdisk/environment/process lifecycleを混ぜない。

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
