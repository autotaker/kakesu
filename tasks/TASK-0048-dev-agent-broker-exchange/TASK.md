---
task_id: "TASK-0048"
title: "broker内egress交換を合成する"
status: plan
created_at: "2026-08-01"
---

# TASK-0048 broker内egress交換を合成する

## `Planning input packet`（Main Agent所有）

このsectionをPlannerとQAへ渡す唯一の`planning input packet`とし、各内容をPLAN/QA_PLANへ複製しない。

### 目的

TASK-0041のpolicy/capability/credential transactionとTASK-0047のrequest単位response sink Forwarderを、brokerから一回呼べるin-memory交換へ合成する。callerはAgent subjectとopaque capability付きrequestを渡し、成功時だけ縮退済みresponseを受け取る。失敗時はresponse、実認証情報、opaque handle、provider detailを返さず、capability消費順序と単回上流試行を既存契約のまま維持する。

### 対象と対象外

#### 対象

- `internal/brokerexchange`のproduction constructorと同期`Do`。Rulesとして既存Policy、Registry、CredentialResolver、注入RoundTripper、credential上限、上流timeout、response本文上限だけを受け取る。
- `Do(subject, egresstransaction.Request)`ごとに非共有response sink、TASK-0047 Forwarder、TASK-0041 Transactionを構成し、Transactionを同期一回だけ実行する。成功時だけ`upstreamforwarder.Response`相当のstatus、正規content type、独立本文を返す。
- 任意のpolicy/capability/resolver/forwarder失敗、sink未通知又は不整合を固定errorとzero responseへ畳む。消費済みcapabilityを復元、再発行又は同じ交換内で再試行しない。
- callerのrequest/Authorization/body sliceを変更・保持せず、実認証情報、opaque handle、URL、scope、provider response/error、dependency detailをerror又はformatへ出さない。
- real Policy/Registryとfake resolver/RoundTripperを使うhermetic integration test。GitHub/OpenAI成功、拒否順序、scope不一致時の非消費、resolver又は上流失敗後の消費維持、単回呼出、response分離、並行request混線なし、固定non-leak errorを検出する。

#### 対象外

- Agent向けHTTP request parser/handler/listener、absolute URL又はHostの再構成、HTTP status/error writer、CONNECT、TLS interception、CA、socket/IPC、peer credential/subject解決。
- command/config/service wiring、broker processの起動、credential bundle読込、production provider resolver/transport生成。既存dependencyをこのTaskで変更せず、後続composition rootが注入する。
- `egresspolicy`、`capability`、`egresstransaction`、`upstreamforwarder`、`upstreamtransport`、`providercredentials`の変更又はwrapper内での再実装。
- provider error本文/header、streaming、redirect/retry、cache、persistence、audit、rate/cost、Git Smart HTTP、push/pull/approval。
- 実GitHub/OpenAI、実認証情報、Internet DNS/TLS/system trust、Agent proxyを使うlive E2E。fake dependencyのPASSで実provider受理を保証しない。
- 新dependency、build/install/config/生成物、恒久外部network test。

### 受け入れ条件

- [ ] AC-1: `New`は非nil Policy/Registry/CredentialResolver/RoundTripper、1〜4,096 byteのcredential上限、1ms〜30秒のtimeout、1 byte〜1 MiBのresponse上限だけを受理し、値を保持したimmutable `Exchange`を返す。不正Rulesは固定error、nil/zero receiverの`Do`はzero responseと固定errorになり、dependencyをformatへ出さない。
- [ ] AC-2: `Do`は呼出しごとにprivate capture sink、`upstreamforwarder.Forwarder`、`egresstransaction.Transaction`を一つずつ構成し、入力subject/requestをTransactionへ同期一回だけ渡す。caller所有のBody/Authorization sliceを変更・保持せず、既定transport、network client、redirect又はretryを自ら選ばない。
- [ ] AC-3: Transactionが成功し、request単位sinkが正確に一回通知された場合だけresponseを返す。status、空又は正規`application/json` content type、caller/dependencyとaliasしない本文以外を公開せず、次の又は並行する`Do`とresponse stateを共有しない。
- [ ] AC-4: policy/Authorization/capability拒否ではresolver/transportへ到達しない。subject/scope不一致のcapabilityは消費せず後続の正しい交換で使える一方、resolver又はForwarderへ到達した失敗は既存Transaction契約どおりcapabilityを消費したままとし、同じhandleで再試行してもresolver/transportへ再到達しない。
- [ ] AC-5: 任意のconstructor、Transaction、Forwarder、sink通知不在又はdependency失敗はzero responseと固定`exchange-denied`だけに畳む。opaque handle、上流credential、request/response本文、URL、scope、provider、下位errorをerror、format又は保持stateへ出さず、capability rollback、再発行、上流再試行をしない。
- [ ] AC-6: real Policy/Registryとfake resolver/RoundTripperによるhermetic race testが両provider成功、実Bearer置換、policy/subject/scope/Authorization拒否順序、非消費/消費境界、resolver/transport/sink単回、zero response on failure、input/output copy、並行response分離、fixed error/non-leakを検出する。`go test -race ./internal/brokerexchange`、harness `make check`/`make distcheck`、candidate launcherのroot `make check`がPASSし、base...candidate差分は追加＋削除1,000行以下である。

### 安定した参照

| 参照ID | 対象 | 固定改訂/ダイジェスト | 用途 |
|---|---|---|---|
| REF-1 | TASK-0041 egress transaction | candidate `c369ec4` / completion `1296427` | policy→capability Consume→resolver→Forwarderの順序、失敗時の消費維持、PreparedRequest非公開契約 |
| REF-2 | TASK-0047 upstream Forwarder | candidate `57e7411` / completion `ceb3ec6` | request単位ResponseSink、縮退success response、単回RoundTrip、固定error境界 |
| REF-3 | egress transaction意味Wiki | main `ceb3ec6`時点 | transaction/transport/forwarder各境界と、production wiring/Agent proxyが未実装である適用限界 |
| REF-4 | Development Agent harness設計 | main `ceb3ec6`時点 | phase 1のGitHub REST read/OpenAIと、HTTP/TLS/Git/approvalの後続境界 |

### 依存状態

| 依存 | 状態 (`ready` / `pending`) | planning参照 | `ready`後に固定する値 |
|---|---|---|---|
| TASK-0041 | `ready` | REF-1 | Transaction APIとcapability消費順序を変更せずcall単位で生成する |
| TASK-0047 | `ready` | REF-2 | request単位sinkへ渡されるsafe responseだけを戻り値へ昇格する |

### 許可パス

- `tools/dev-agent-harness/internal/brokerexchange/`
- `tools/dev-agent-harness/README.md`

### 完了経路preflight

| 確認対象 | 結果 | コマンドまたは根拠 |
|---|---|---|
| 完了checker | `ready` | planning/candidate/completionの3 commit gateとpost-merge `task-check`を使う |
| 権限 | `ready` | DEV/QAはreal in-memory Policy/Registryとfake resolver/transportだけを使い、外部network、実認証情報、OS権限へ到達しない |
| 依存状態と参照 | `ready` | TASK-0041/0047完了。既存API、消費順序、response境界をREF-1/2で固定 |
| 生成物の有無と更新方法 | `ready` | Go source/testとREADMEだけ。dependency、config、生成物なし |
| 割当ワークツリー | `ready` | `worktrees/TASK-0048-dev-agent-broker-exchange` |
| Lapログの書込・Schema・`repository annotation` | `ready` | 標準3 commits。新Schema、digest転記、追加機械checkなし |

### 未決事項

- なし

### `Dependency-ready reconciliation`

- TASK-0041/0047はいずれもplanning開始時点でready。既存packageを変更せず、新packageがcall単位で合成する。HTTP handler/listenerと実provider受理はこのTaskでも後続/live E2E blockedのままとする。

## 背景

Transactionはresponseを返さずtrusted Forwarderを同期呼出し、Forwarderはrequest単位sinkへresponseを渡す。これは秘密とresponse stateを各境界へ閉じるが、brokerの利用側がcall単位sinkを安全に生成し、Transaction成功とresponse通知を一つの戻り値へ束縛するcomposition rootがまだない。HTTP parsingやTLS終端を加える前に、このin-memory交換を独立に固定する。

## 検討すべき設計観点

- `Exchange`はPolicy/Registry/Resolver/Transportだけを長寿命依存として持ち、response、subject、request、capability、credentialを保持しない。sink、Forwarder、Transactionは各`Do`のlocal値にする。
- Transaction/Forwarderの内部安全検査を複製せず、両者のconstructorと固定errorを利用する。Exchangeの役割は順序、call-local response capture、zero-on-failureの合成だけに限定する。
- capability消費境界をintegration testで観測する。認可前のscope/subject mismatchはhandleを残し、resolver/上流失敗はhandleを戻さない。失敗を便利にretryするwrapperは作らない。
- ResponseSinkはExchange内部実装だけとし、外部callerが任意sinkを注入してresponse stateを横取りしない。成功時のcapture一回を確認し、未通知/二重通知はfail-closedにする。
- HTTP request/response変換、subjectをpeer credentialから解決する境界、production dependency wiringは混ぜず、次Taskでこの`Do`だけを呼ぶ。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] planning/candidate/completionの3 commit経路とcandidate一回のroot `make check`を満たしている。
- [ ] 同一candidateの独立REVIEW/QAを完了し、実provider/Agent proxyのlive E2E未実施境界をPASSと誤記していない。
- [ ] 再利用可能な知識が生じた場合だけ意味Wikiを既存ページへ同化し、post-merge `task-check`をPASSしている。

## 関連コンテキスト

### 意味 Wiki

- `wiki/semantic/schemas/development-agent-harness-egress-transaction.md`

### 判断

- HTTP handlerより先に、Transactionとrequest単位Forwarder/sinkを一つのin-memory broker exchangeへ合成する。

### 適用しなかった重要な判断

- Transactionへresponse戻り値を追加する案は既存秘密境界を広げるため採用しない。
- callerが任意ResponseSinkを注入する案はresponse stateと通知回数の責務を外へ漏らすため採用しない。
- HTTP handler、production resolver/transport生成、listener/TLSを同時実装する案は失敗原因と認証境界を一Taskで混在させるため採用しない。
