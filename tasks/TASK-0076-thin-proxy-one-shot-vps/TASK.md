---
task_id: "TASK-0076"
title: "薄いproxyとrepository one-shot pushをVPS縦断する"
status: draft
created_at: "2026-08-02"
---

# TASK-0076 薄いproxyとrepository one-shot pushをVPS縦断する

## `Planning input packet`（Main Agent所有）

このsectionをPlannerとQAへ渡す唯一の`planning input packet`とし、各内容をPLAN/QA_PLANへ複製しない。

### 目的

Development Agent Harnessの認証転送とpush承認を、provider APIの意味を再実装しない薄いstream proxyとrepository単位のone-shot grantへ置き換える。旧manifest/ref/SHA検査と重複抽象層は削除し、さくらのVPS上でGit pull、限定GitHub REST、OpenAI API、スマートフォン承認後のpushを一つの縦断経路として成立させる。

### 対象と対象外

#### 対象

- `approvalmanifest`を削除し、request/decision/grantをAgent instance/UID、workspace、完全一致repository、短TTL、一回使用、revokeだけへ束縛する。TASK-0071〜0073由来のmanifest digest依存は互換層を残さず置換する。
- 通常のGit read、GitHub REST、OpenAIを、Unix peer/capability/host/repository/credential境界の検証後にmethod/path/query/bodyとstatus/headers/bodyを原則未解釈でbackpressure付きstream転送する。
- pushは完全一致repositoryと`git-receive-pack`だけを最小分類し、本文、pkt-line、ref、old/new SHA、force/deleteを解析しない。該当grantを上流試行開始前に原子的に消費する。
- `egresspolicy`のprovider意味検査、GitHub repository endpoint parser、strict OpenAI JSON検査、Git upload-pack本文/response意味検査、JSON/2xx/Content-Type検査、response全量buffer/固定1 MiB上限、Policy→Transaction→Exchange→Forwarderの重複評価を削除する。
- Tailscale到達主体とPasskey本人確認をrepository単位decisionへ接続するApproval UIを実装する。UIの主文言は「このrepositoryへの次のpush一回」とし、branch/commit/ref/SHA等は参考情報に限定する。
- configure/`make install`でVPSへ配置できるサービス、設定例、rollback/cleanup可能なlive E2E手順を整え、承認済み実環境で縦断確認する。

#### 対象外

- 同一repository内で参考表示と異なる内容を承認済み一回に使用できる残余リスクの除去。
- GitHub/OpenAIのAPI Schema、endpoint、JSON field、model、status又はresponse Content-Typeのproxy内allowlist化。
- 複数repository、複数push、force/delete別承認、ref/SHA完全一致、remote old SHA観測、Git wire本文照合。
- Codex実認証情報の例外方針変更、Kakesu本体runtime/Schema、Tailscale control plane、GitHub/OpenAIサービス自体の変更。
- フェーズ別の互換wrapper、休眠旧実装、将来用parser、追加の形式証跡又は機械check。
<!-- safety_contractの場合: 製品コード、test、runtime/build設定、Schema、製品依存、生成製品入力/成果物、外部観測可能な挙動を変更しない。 -->

### 受け入れ条件

<!-- AC-IDはTask内で一意かつ安定させ、観測可能な結果をここに一度だけ記載する。 -->

- [ ] AC-1: productionから`approvalmanifest`とmanifest digest/ref/SHA/force/delete束縛が削除され、repository単位request/decision/grantだけが残る。grantはAgent instance/UID、workspace、repository、短TTL、未使用、revokeを検査し、並行pushでも上流試行へ進めるのは一回だけである。
- [ ] AC-2: 通常のGit read/GitHub REST/OpenAIはprovider意味を解釈せずstream転送され、非2xx、非JSON、任意response Content-Type、1 MiB超responseも上流値を保持する。Unix peer、Opaque handle、host allowlist、credential置換、TLS CONNECT/local CA、timeout/concurrency/header上限、secret-free auditは維持される。
- [ ] AC-3: pushは完全一致repositoryと`git-receive-pack`だけで分類され、本文を読まず、grantを上流接続又は本文送信前に原子的に消費する。同一repository内の内容差替えは許容し、別repository、別Agent/workspace、再使用、期限後、revoke後、GitHub REST転用を上流到達前に拒否する。
- [ ] AC-4: Approval UIはTailscale到達主体とPasskey user verificationを一回のrepository decisionへ接続し、「このrepositoryへの次のpush一回」を認可文言として表示する。branch/commit/ref/SHA/force/deleteは参考情報と明示され、認可判定へ使用されない。
- [ ] AC-5: 旧provider意味検査、full buffering/1 MiB、2xx/JSON/Content-Type検査、GitHub endpoint parser、Git upload-pack意味検査、Policy→Transaction→Exchange→Forwarder重複層がproductionと対応testから削除され、同じ責務のwrapper又はdead codeが残らない。
- [ ] AC-6: 承認済みさくらVPSで、Agent userから実Credentialを読めない状態のまま、Git pull、限定GitHub REST、OpenAI API、スマートフォン承認後pushが成功する。別repository、grant再使用、期限後、REST転用が拒否され、実token/secretがAgent環境・応答・auditへ出ず、試験branch/commitと配置を手順どおりcleanup又はrollbackできる。
- [ ] AC-7: `./configure && make && make check && make install DESTDIR=...`とfocused race/integration testがPASSし、systemd/設定例/runbookが実装と一致する。失敗時はprovider/環境/認証/実装へ分類し、live E2E未実施をhermetic PASSで代替しない。

### 安定した参照

| 参照ID | 対象 | 固定改訂/ダイジェスト | 用途 |
|---|---|---|---|
| REF-1 | TASK-0074安全契約 | candidate `563f52769cfdc1349271a60262c7c340eb5998ae` / merge `1c7663af64188aeb21d75a02d12036dd8e5f291c` | 認可単位、薄いproxy、削除inventory、残余リスク、単一vertical sliceの正本 |
| REF-2 | Development Agent Harness設計 | `docs/development/development-agent-harness.md` at `1c7663af64188aeb21d75a02d12036dd8e5f291c` | V-01〜V-12と実VPS配置境界 |
| REF-3 | 現行production/test inventory | main `317e260bc60d3580db9936f3345df9d35c136944` | `approvalmanifest/state/challenge/decision`、`egresspolicy/transaction/exchange/forwarder/http/service`の削除・置換起点 |

### 依存状態

| 依存 | 状態 (`ready` / `pending`) | planning参照 | `ready`後に固定する値 |
|---|---|---|---|
| ローカル実装基盤 | `ready` | main `317e260bc60d3580db9936f3345df9d35c136944`、Go/configure test | candidate差分とhermetic受け入れ |
| live VPS/認証/試験repository | `pending` | リポジトリ内にhost alias、tailnet主体、Passkey登録、GitHub App対象repo、OpenAI test credentialの値はない | VPS接続先、operator/Tailscale identity、Passkey enrollment、repository限定GitHub App installation、OpenAI credential、使い捨てtest branchとcleanup権限を秘密非保存で固定する |

### 許可パス

- `tools/dev-agent-harness/`

### 完了経路preflight

| 確認対象 | 結果 | コマンドまたは根拠 |
|---|---|---|
| 完了checker | `ready` | TASK-0075でproduct completion、candidate-bound REVIEW/QA、no-ffを実運用確認済み |
| 権限 | `ready` / liveのみ`pending` | DEVはTask worktreeの`tools/dev-agent-harness/`だけ。MainだけがGit統合。VPS変更は接続先とcleanup承認後だけ行う |
| 依存状態と参照 | `ready` / liveのみ`pending` | REF-1〜3固定。live値はrepositoryへ記録せずdependency-ready reconciliationで識別子だけ固定する |
| 生成物の有無と更新方法 | `ready` | configure生成物は既存`configure.ac`/Makefile経路で更新し、配布物・DESTDIR installを検証する |
| 割当ワークツリー | `ready` | `worktrees/TASK-0076-thin-proxy-one-shot-vps` |
| Lapログの書込・Schema・`repository annotation` | `not-applicable` | Lap/Schema/annotation変更なし。live E2E結果はQA_RESULT/HANDOVERへsecret-freeに記録する |

### 未決事項

- live VPS/認証/試験repositoryの具体値と安全なcleanup対象はoperator入力待ち。ローカル実装・fixtureの期待値は変更しないが、live QA開始前にMainがreconciliationする。

### `Dependency-ready reconciliation`

<!-- 依存ready時にMainがready参照、planning参照との差分、AC/設計/スコープ/QAへの影響、再承認結果を追記する。依存なし又は未readyならN/Aとする。 -->

- live依存がreadyになった時点で、秘密値そのものを記録せず、VPS識別子、tailnet主体、対象repository、試験branch、実施時刻、cleanup方法とAC-6/QAへの影響だけを追記しMainが再承認する。

## 背景

TASK-0039〜0069で認証転送の部品を細分化した結果、provider APIの意味を複数層で重複判定し、responseを全量bufferして1 MiB/2xx/JSON/Content-Typeへ制限する約2,500行のproductionと約3,700行のtestが先行した。一方、スマートフォン承認、repository one-shot grant、実VPS縦断は未成立である。TASK-0074で過剰契約を廃止したため、旧コードを積み増さず削除と縦断を同じTaskで完了させる。

## 検討すべき設計観点

- request/response bodyを認可判断のためにbufferせず、Go `io.Reader`/`io.Copy`とHTTP cancellation/backpressureをどの所有境界へ置くか。
- host/repositoryの一回評価、Opaque capability消費、push grant消費、credential取得、upstream開始の順序を一箇所へ集約すること。
- provider credential交換（GitHub App token取得）はbroker内部処理であり、Agent要求のGitHub REST/OpenAI/Git転送と意味検査を混同しないこと。
- Passkey verification、Tailscale identity headerのtrusted ingress、CSRF/session、pending requestのdurabilityとone-shot grantのatomicityを最小構成で接続すること。
- live E2Eは実push前に対象repository/branchとcleanupを固定し、grant消費後の失敗を自動再許可しないこと。
- 削除SLOCを品質成果として扱う一方、Unix peer/secret境界、resource limit、negative testsを削り過ぎないこと。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] 選択した`change_class`の完了経路と`make check`を満たしている。
- [ ] 製品変更の場合: 実装、テスト、文書、同一案の独立REVIEW/QA、完了後の環境依存ケース確認が完了している。
- [ ] 安全契約変更の場合: Mainの意図・スコープ・受け入れ経路確認、契約検査、許可された統制文書差分の確認が完了している。

## 関連コンテキスト

### 意味 Wiki

- `wiki/semantic/schemas/development-agent-harness-egress-policy.md`
- `wiki/semantic/schemas/development-agent-harness-push-approval-manifest.md`（廃止済み履歴）
- `wiki/semantic/schemas/development-agent-harness-approval-request-store.md`

### 判断

- TASK-0074のrepository one-shot/薄いproxy契約を実装正本とし、旧manifest/provider parserへ互換させない。
- 削除、proxy、approval、live E2Eを別Taskへ分割しない。live外部依存だけを完了前の明示blockerとして扱う。

### 適用しなかった重要な判断

- ref/SHA完全一致認可とprovider API Schema検査はTASK-0074で不採用になったため再検討しない。
