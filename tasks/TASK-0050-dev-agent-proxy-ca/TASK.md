---
task_id: "TASK-0050"
title: "broker内TLS CA読込とleaf証明書発行を実装する"
status: plan
created_at: "2026-08-01"
---

# TASK-0050 broker内TLS CA読込とleaf証明書発行を実装する

## `Planning input packet`（Main Agent所有）

このsectionをPlannerとQAへ渡す唯一の`planning input packet`とし、各内容をPLAN/QA_PLANへ複製しない。

### 目的

TLS interception listenerへ渡す前段として、broker memory内だけで既存CA certificate/private keyを厳格に読込み、初期対応2 host向けの短命leaf certificateを発行する境界を作る。Agentへ返せるのはcopyされた公開CA certificateだけとし、CA private key、入力PEM、leaf private key detailを公開error、format、公開CA出力へ出さない。listener、CONNECT、ファイル配置を混ぜず、証明書の意味と所有権をhermeticに固定する。

### 対象と対象外

#### 対象

- `internal/proxyca`のproduction Authority。Rulesとして単一CA certificate PEM、対応するECDSA P-256 private key PEM、注入Clockだけを受け、入力byte列を保持しない。
- CA PEMの単一block/trailing不在、certificate parse、自己署名、BasicConstraints/IsCA、KeyUsageCertSign、ECDSA P-256 public/private一致、現在有効かつleaf lifetime以上の残存期間を検査する。暗号化PEM、複数block、RSA/Ed25519/弱い又は不一致keyを固定errorで拒否する。
- exact `api.github.com`又は`api.openai.com`だけを受け、呼出しごとに新しいECDSA P-256 leaf key、nonzero random 128-bit serial、単一DNS SAN、ServerAuth、DigitalSignature、短命validityを発行する。CA有効期間を越えず、CN/IP/email/URI/client auth/CA能力を与えない。
- `PublicCertificatePEM`はCA certificateだけの独立copyを返し、AuthorityはCA signer/certificate/public PEMとClockだけを保持する。nil/zero/破損Authority、clock/RNG/signing failureは固定non-leak errorにする。
- generated leafを使う`net.Pipe` TLS handshake、両host/wrong-host、concurrent issuance、unique serial/key、copy/format/non-leakを含むhermetic race test。

#### 対象外

- CA/leaf private keyのファイル読込・生成・rotate・永続化・削除、filesystem permission/owner、秘密ストア、CLI/config/provision/systemd wiring。
- Agent trust storeへの公開CA install、OS/browser/gh/OpenAI SDKのcertificate trust、certificate pinning対応。
- TCP/Unix listener、CONNECT、TLS handshake受付、SNI routing、certificate cache、HTTP handler/server、peer identity、broker composition。
- CRL/OCSP、intermediate CA、path length運用、multiple CA、RSA/Ed25519、wildcard、任意host、IP SAN、client certificate/mTLS、audit。
- 既存brokerhttp/Exchange/Policy/credential/transport package、dependency/build/config/generated artifactの変更。
- 実CA秘密、実Agent、実TLS client、実VPS trust storeを使うlive E2E。hermetic TLS handshakeで実配布を保証しない。

### 受け入れ条件

- [x] AC-1: `New`は単一CA certificate PEM、対応する単一ECDSA P-256 private key PEM、非nil Clockだけを受理する。CAは自己署名、現在有効、BasicConstraintsValid/IsCA、KeyUsageCertSign、leaf lifetime以上の残存期間を満たし、public/private keyが一致する。不正入力、typed nil、nil/zero Authorityはfixed errorになり、入力PEM又はparser detailを公開しない。
- [x] AC-2: Authorityはparse済みCA certificate/signer、copyした公開certificate PEM、Clockだけを保持し、callerのPEM byte sliceを保持又は変更しない。`PublicCertificatePEM`はcertificate blockだけの新しいcopyを毎回返し、CA/leaf private key、入力private PEM、追加blockを返さない。error/Formatは固定で秘密、subject/serial/host/parser detailを含まない。
- [x] AC-3: `Issue`はexact `api.github.com`と`api.openai.com`だけを受理し、空、port付き、case差、末尾dot、wildcard、IP、未知host、control/non-ASCIIを署名前に固定errorで拒否する。拒否はserial/key/certificateを返さず、retry/cache/別host補正を行わない。
- [x] AC-4: 許可hostごとに新しいECDSA P-256 keyとnonzero 128-bit random serialを生成し、単一exact DNS SAN、空CN、ServerAuthだけ、DigitalSignature、IsCA false、BasicConstraintsValidを持つleafを発行する。NotBeforeは現在から5分以内のbackdate、NotAfterは15分以内かつCA期限以前で、IP/email/URI SAN、ClientAuth、CertSignを持たない。
- [x] AC-5: 返す`tls.Certificate`はleaf→CAのchain、parse済みLeaf、対応leaf private keyを持ち、call間でcertificate/private key/serial/bufferを共有しない。公開CAで両hostのTLS 1.2/HTTP1.1相当handshakeとhostname verifyが成功し、wrong host/expired CA/未許可hostはfail closedになる。並行Issueでrace、duplicate serial/key、cross-host SAN混線がない。
- [x] AC-6: in-memory fixtureによるhermetic race testがPEM/block/key/CA validity拒否、input/public output copy、fixed non-leak、host exact拒否、leaf extension/validity/chain、TLS handshake/hostname verify、concurrent uniquenessを検出する。`go test -count=1 -race ./internal/proxyca`、harness `make check`/`make distcheck`、README変更時のTask worktree `make lint-docs`、candidate launcherのroot `make check`がPASSし、base...candidate差分は追加＋削除1,000行以下である。

### 安定した参照

| 参照ID | 対象 | 固定改訂/ダイジェスト | 用途 |
|---|---|---|---|
| REF-1 | Development Agent harness設計 | main `5f5b0eb`時点 | 公開CAだけをAgentへ配り、CA private keyをbroker境界から出さないTLS要件 |
| REF-2 | TASK-0049 broker HTTP Handler | candidate `d5b94f2` / completion `5f5b0eb` | TLS終端後request入口。CA Authorityはその前段でありHandlerを変更しない |
| REF-3 | Go `crypto/x509` / `crypto/tls`標準library | repository toolchain固定版 | PEM/x509 validation、leaf生成、hostname/TLS handshake fixture |
| REF-4 | egress transaction意味Wiki | main `5f5b0eb`時点 | HTTP入口より外側のlistener/TLS/production identityが未実装である適用限界 |

### 依存状態

| 依存 | 状態 (`ready` / `pending`) | planning参照 | `ready`後に固定する値 |
|---|---|---|---|
| TASK-0049 | `ready` | REF-2 | Handler APIを変更せず、後続listenerがAuthorityとHandlerを合成する |

### 許可パス

- `tools/dev-agent-harness/internal/proxyca/`
- `tools/dev-agent-harness/README.md`

### 完了経路preflight

| 確認対象 | 結果 | コマンドまたは根拠 |
|---|---|---|
| 完了checker | `ready` | planning/candidate/completionの3 commit gateとpost-merge `task-check`を使う |
| 権限 | `ready` | DEV/QAはtest内生成CA、in-memory PEM、net.Pipeだけを使い、filesystem秘密、listener port、外部network、実trust storeへ到達しない |
| 依存状態と参照 | `ready` | TASK-0049完了。標準libraryだけで独立Authorityを追加できる |
| 生成物の有無と更新方法 | `ready` | Go source/testとREADMEだけ。dependency、config、配布生成物なし |
| 割当ワークツリー | `ready` | `worktrees/TASK-0050-dev-agent-proxy-ca` |
| Lapログの書込・Schema・`repository annotation` | `ready` | 標準3 commits。新Schema、digest転記、追加機械checkなし |

### 未決事項

- なし

### `Dependency-ready reconciliation`

- TASK-0049はplanning開始時点でready。HandlerはTLS終端後だけを所有するため変更せず、本Taskは前段のin-memory CA読込/leaf発行だけを新packageへ追加する。listener、CONNECT、実秘密ファイル、trust installは後続/live blockedのままとする。

## 背景

HTTP Handlerまでのbroker経路は完成したが、explicit proxyがTLSを終端するための証明書発行境界がない。listenerと同時に秘密file、CA validation、leaf extension、CONNECT state machineを実装すると、暗号材料の欠陥とnetwork lifecycleの欠陥が混ざる。まず入力PEMを保持しないAuthorityと短命leafだけを固定する。

## 検討すべき設計観点

- CA入力はtrustedな上位秘密ストアからmemoryで渡される前提にし、本Taskでpath/environmentを受けない。file policyは別Taskで独立QAする。
- 対応hostを2つへ固定し、wildcardやnormalizeを行わない。provider policyとは別に、証明書を発行してよいSNI候補を狭めるdefense-in-depth境界とする。
- public CA exportとleaf certificate returnを分離し、CA signerを返すAPIを作らない。leaf private keyはTLS serverに必要だがAuthorityのformat/public PEMへ出さない。
- fixed 15分leaf、5分以内backdate、ECDSA P-256、ServerAuthだけに狭め、利用観測なしでconfig surfaceを増やさない。
- `crypto/x509`が生成した結果をparseしてtestし、TLS handshake/hostname verifyでextensionとchainの実効性を確認する。実OS trust installは別境界である。

## 完成の定義

- [x] 受け入れ条件を満たしている。
- [x] planning/candidate/completionの3 commit経路とcandidate一回のroot `make check`を満たしている。
- [x] 同一candidateの独立REVIEW/QAを完了し、実CA file/trust/client/listenerのlive E2E未実施境界をPASSと誤記していない。
- [x] 再利用可能な知識が生じた場合だけ意味Wikiを既存ページへ同化し、post-merge `task-check`をPASSしている。

## 関連コンテキスト

### 意味 Wiki

- `wiki/semantic/schemas/development-agent-harness-egress-transaction.md`

### 判断

- listener/CONNECTより先に、broker memory内のCA validationと2 host限定短命leaf発行を固定する。

### 適用しなかった重要な判断

- CA生成/file rotation/trust installを同時実装する案はmemory cryptographic boundaryとOS lifecycleを混在させるため採用しない。
- 任意host/wildcard、複数algorithm、configurable lifetimeを初期APIへ入れる案は未観測surfaceを増やすため採用しない。
- CA signer又はprivate PEMをexportする案はAgent側露出経路を増やすため採用しない。
