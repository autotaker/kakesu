---
task_id: "TASK-0068"
title: "GitHub REST/OpenAI用capability clientを実装する"
status: draft
created_at: "2026-08-02"
---

# TASK-0068 GitHub REST/OpenAI用capability clientを実装する

## `Planning input packet`（Main Agent所有）

このsectionをPlannerとQAへ渡す唯一の`planning input packet`とし、各内容をPLAN/QA_PLANへ複製しない。

### 目的

後続launcherが`GH_TOKEN`と`OPENAI_API_KEY`へ実認証情報ではなくOpaque capabilityを設定できるようにする。既存control socketからGitHub REST read用とOpenAI Responses text用のhandleを取得するstrict clientを追加し、長時間Agent sessionで一回しか通信できない現行API capabilityを、固定かつ小さいrequest budgetへ広げる。Git Smart HTTP readは従来どおり一回限りに保つ。

### 対象と対象外

#### 対象

- `controlclient`へGitHub REST readとOpenAI Responses textの明示issue操作を追加し、既存fixed Unix socketへ一接続一操作のexact requestを送る。
- GitHub RESTはcanonical allowlist repository、OpenAIはrepository/model等のAgent入力なしという既存server contractに合わせる。
- API capabilityの固定使用回数を16へし、Git read capabilityは一回限りを維持する。TTL、subject/workspace/provider/repository/operation/destination bindingは変えない。
- strict response framing、canonical Opaque handle、deadline/one dial/close、固定errorと非漏洩を既存client transportの共有private helperで保つ。
- hermetic testsでexact wire、scope別budget、17回目拒否、mismatch非消費、Git read回帰、failure/non-leakを確認し、READMEへlauncher前段の境界を記録する。

#### 対象外

- launcher、environment、`GH_TOKEN`/`OPENAI_API_KEY`設定、child process、CA trust file、Git config、loopback bridge起動を実装しない。
- GitHub write/GraphQL、OpenAI admin/files/upload、Git push、approval/push grantを許可しない。
- capabilityのTTL、16を超えるbudget、更新、永続化、cache、retry、複数socket、TCP、runtime path overrideを追加しない。
- provider credential取得/置換、HTTP policy、model/repository allowlist、DNS/TLS/upstream、Registryのatomic consume意味を変更しない。
- 実GitHub/OpenAI credential、実`gh`/SDK、systemd socket/別UID/VPSをhermetic PASSで代替しない。
<!-- safety_contractの場合: 製品コード、test、runtime/build設定、Schema、製品依存、生成製品入力/成果物、外部観測可能な挙動を変更しない。 -->

### 受け入れ条件

<!-- AC-IDはTask内で一意かつ安定させ、観測可能な結果をここに一度だけ記載する。 -->

- [ ] AC-1: GitHub REST issueはabsolute clean fixed Unix socketとcanonical `owner/repo`だけを受け、exact `POST /v1/capabilities`へbody `{"provider":"github","repository":"owner/repo"}`を送り、OpenAI issueはsocketだけを受けてexact body `{"provider":"openai"}`を送る。各操作は一回dial、一回request、deadline、closeを使い、Agent入力のprovider/operation/body/model/pathを受けない。
- [ ] AC-2: clientは唯一のbounded `200 application/json`、canonical Content-Length/header order、直後EOF、exact `{"handle":"cap_..."}`だけを成功とする。status/header/body/JSON/handle/framing/extra byte、dial/deadline/read/write/close failureはnil/emptyと固定errorになり、socket/repository/handle/wire/lower-errorを診断へ出さず、retry/fallbackしない。
- [ ] AC-3: peer-bound controllerが発行するGitHub REST/OpenAI capabilityはTTL 5分、fixed 16 usesとし、各正規consumeで原子的に1減少して16回目だけremaining 0、17回目は拒否する。subject/workspace/provider/repository/operation/destination mismatchはbudgetを消費せず、revoke/expiry/epochの既存意味を維持する。
- [ ] AC-4: `github-git-read` selectorのcapabilityは従来どおりsingle useで、既存`controlclient.Issue`、Git credential helper get/erase、CONNECT/control wireの意味を変えない。API handleはGit read/push/write/別provider/repository/hostへ使用できない。
- [ ] AC-5: candidateは承認済み5パス・約750〜1,050 changed linesを目安とし、実token/key、launcher/env/config/Makefile/dependency/Schema/Kakesu runtime/generated file/live stateを含まない。focused race、harness `make check`/distcheck、candidate gate root `make check`、`git diff --check`がPASSする。

### 安定した参照

| 参照ID | 対象 | 固定改訂/ダイジェスト | 用途 |
|---|---|---|---|
| REF-1 | Harness設計 §7.2、§9.1、§9.2 | main `8ec015d` | GH/OpenAIへOpaque handleだけを渡し、provider別scopeとbudgetをbrokerで拘束する契約 |
| REF-2 | TASK-0061 capability control | main `8ec015d` | peer-bound issue/revoke、REST/OpenAI/Git-read selectorと固定wire |
| REF-3 | TASK-0065 controlclient | main `8ec015d` | fixed Unix dial、strict response、Git-read issue/revoke、non-leak transport |

### 依存状態

| 依存 | 状態 (`ready` / `pending`) | planning参照 | `ready`後に固定する値 |
|---|---|---|---|
| TASK-0061 / TASK-0065 | `ready` | main `8ec015d` | controller issue scopes、strict control protocol/client transport |

### 許可パス

- `tools/dev-agent-harness/internal/controlclient/client.go`
- `tools/dev-agent-harness/internal/controlclient/client_test.go`
- `tools/dev-agent-harness/internal/capabilitycontrol/control.go`
- `tools/dev-agent-harness/internal/capabilitycontrol/control_test.go`
- `tools/dev-agent-harness/README.md`

### 完了経路preflight

| 確認対象 | 結果 | コマンドまたは根拠 |
|---|---|---|
| 完了checker | `ready` | 通常product 3トランザクション、同一candidate独立REVIEW/QA、no-ff completionを使用する |
| 権限 | `ready` | Opaque auth、protocol、request budgetの安全境界なので`dev-sol`。Mainだけがstage/commit/merge/pushする |
| 依存状態と参照 | `ready` | TASK-0061/0065がmainへ反映済み |
| 生成物の有無と更新方法 | `ready` | Go source/testとREADMEのみ。configure/dependency/generated fileなし |
| 割当ワークツリー | `ready` | `worktrees/TASK-0068-api-capability-control-client` |
| Lapログの書込・Schema・`repository annotation` | `not-applicable` | 新規log/Schema/annotationなし |

### 未決事項

- なし

### `Dependency-ready reconciliation`

<!-- 依存ready時にMainがready参照、planning参照との差分、AC/設計/スコープ/QAへの影響、再承認結果を追記する。依存なし又は未readyならN/Aとする。 -->

- N/A

## 背景

controllerはGitHub REST/OpenAI capabilityを既に発行できるが、Agent側`controlclient`はGit-read専用操作しか公開していない。また全scopeが一回限りなので、long-livedなcoding-agent sessionへ一個のenvironment handleを渡すと最初のAPI request後に利用不能になる。Registryとserviceは16 usesを上限として既に固定しているため、API readだけをその上限まで許し、Git helperが操作ごとに取得するGit-readはsingle-useに残す。

## 検討すべき設計観点

- client public APIはprovider文字列や任意bodyを受けず、GitHub REST/OpenAIごとの明示関数にして許可scope以外を表現不能にする。
- 使用回数はcaller/Agent/config入力にせず、controllerがscopeから1又は16を決める。mismatchは既存Registry契約どおり消費しない。
- 16 usesは認可scopeを広げず、同じsubject/workspace/provider/repository/operation/host内のrequest budgetだけを広げる。write、push、GraphQL又はuploadを許可する根拠にしない。
- 既存Git-read `Issue`とstrict response parserを複製せずprivate exchangeを共有するが、server-side scope判定とclient-side framing検証は混同しない。

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
