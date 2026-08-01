---
task_id: "TASK-0040"
title: "Opaque capability registryを実装する"
status: plan
created_at: "2026-08-01"
---

# TASK-0040 Opaque capability registryを実装する

## `Planning input packet`（Main Agent所有）

このsectionをPlannerとQAへ渡す唯一の`planning input packet`とし、各内容をPLAN/QA_PLANへ複製しない。

### 目的

Development Agentへ渡す`cap_...` handleを、実Credentialを含まない短命の参照として発行し、Agent instance、UID、workspace、provider、repository、operation、destination host、期限、使用回数、policy version、revocation epochへ束縛するin-memory registryを実装する。後続proxyがGitHub/OpenAI requestを実Credentialへ置換する前に、漏えいしたhandleのscope逸脱、再利用、失効後利用をfail-closedかつ原子的に拒否できる境界を固定する。

### 対象と対象外

#### 対象

- `internal/capability`のRules、IssueSpec、Request、Grant、Registry API。
- Registry Rulesはcanonical policy version、正の最大TTL、正の最大使用回数、初期revocation epochを持ち、不正値を固定Rules errorで拒否する。
- `Issue`はcanonicalなsubject（agent instance IDとnon-root UID）、workspace ID、provider、repository、TTL、使用回数を検証し、GitHubを`github-rest-read`/`api.github.com`/必須repository、OpenAIを`openai-responses-text`/`api.openai.com`/repositoryなしへ固定する。
- handleは`cap_` prefixと32 byteのcryptographic random値からなるbase64url文字列とし、RegistryにはSHA-256 digestだけを保存する。entropy failureまたはbounded retry内の衝突は固定issue errorでfail-closedにする。
- `Consume`はhandleとRequestのsubject/workspace/provider/repository/operation/destination hostを完全一致で検証し、期限、current revocation epoch、残使用回数をmutex下の一transactionで判定・消費する。scope mismatchは使用回数を消費せず、最後の成功、期限切れ、明示失効、epoch更新後は再利用できない。
- `Revoke`と単調増加する`AdvanceRevocationEpoch`。epoch更新は既存entryを無効化し、未知handle、古い/同じepoch、nil/zero Registryを固定errorで拒否する。
- production constructorは`crypto/rand.Reader`とUTC clockを使い、test用dependencyはpackage内に閉じる。固定error、入力非漏洩、並行消費、raw handle非保持、入力非変更をunit testで確認する。

#### 対象外

- GitHub App private key/installation token、OpenAI API key、Codex Credentialを含む実Credentialの読取、保存、生成、置換。
- HTTP server/proxy、TLS/CA、DNS/address、socket、header/body解析、redirect、upstream通信、Git Smart HTTP、Git認証情報helper接続。
- 永続化、restart後の復元、複数process共有、audit/log、rate/cost/byte budget、Approval/push grant、Tailscale/Passkey。
- config/schema/CLI/systemd/provision manifest、OS user/namespace/firewall、Kakesu本体、外部製品依存の変更。
- 汎用provider/operationの先取り。初期surfaceはTASK-0039が返すGitHub REST readとOpenAI Responses textの二つだけとする。

### 受け入れ条件

- [ ] AC-1: valid RulesからRegistryを生成できる。policy versionは1〜64 byteのsafe ASCII identifier、最大TTLは正かつ24時間以下、最大使用回数は正かつ10,000以下であり、zero/不正/上限超過は固定Rules errorを返す。production Registryはcrypto entropyとUTC clockを使い、nil/zero Registryは発行も利用も許可しない。
- [ ] AC-2: valid IssueSpecは1〜128 byteのcanonical agent instance/workspace ID、non-root UID、provider固有scope、正かつRules内のTTL/使用回数を持つ。Issueは`cap_` + paddingなしbase64urlの32 random bytesを返し、entryにはhandleのSHA-256 digest、固定scope、issued/expires、remaining uses、policy version、revocation epochだけを保持する。不正spec、entropy failure、4回以内で解消しない衝突は入力を含まない固定errorを返し、partial entryを残さない。
- [ ] AC-3: GitHub Requestは同じsubject/workspace、`github` provider、完全一致repository、`github-rest-read` operation、`api.github.com` host、OpenAI Requestは同じsubject/workspace、`openai` provider、repositoryなし、`openai-responses-text` operation、`api.openai.com` hostの全条件一致時だけGrantを返す。unknown/malformed handle、大小文字やprefix/suffixだけの一致、scope/policy不一致は同じ固定denyを返し、scope mismatchでは残使用回数を消費しない。
- [ ] AC-4: `Consume`は期限境界を`now >= expires`で拒否し、成功ごとに使用回数を一回だけ減らし、最後の成功後にentryを削除する。`Revoke`後と`AdvanceRevocationEpoch`後は再利用できず、epochは単調増加だけを受理する。同じ1-use handleへの並行Consumeではexactly oneだけが成功し、race detectorでdata raceがない。
- [ ] AC-5: errorはhandle、agent/workspace/repository、provider input、entropy/clock由来detailを含まない固定値である。Registryはraw handle、実Credential、Request sliceを保持せず、file/environment/process/network/DNS/TLSを使わない。restart時にin-memory entryが失われる性質をfail-safeとしてREADMEへ明記し、永続性や実通信のPASSを主張しない。
- [ ] AC-6: table-driven unit testsがAC-1〜5の代表的なissue/consume/deny/revoke/epoch/expiry/collision/entropy failure/non-leak/不変性/並行性を検出する。`go test -race ./internal/capability`、harness `make check`/`make distcheck`、root `make check`がPASSし、製品差分は`internal/capability/`とharness READMEだけに限定し、合計1,200行を超えない。

### 安定した参照

| 参照ID | 対象 | 固定改訂/ダイジェスト | 用途 |
|---|---|---|---|
| REF-1 | `docs/development/development-agent-harness.md` | main `41e7e33`時点の§7.2、§8、§9、§14〜16 | Opaque handleのscope、期限/予算/revocation、provider境界、段階導入 |
| REF-2 | TASK-0039 egress policy | candidate `709b841` / completion `41e7e33` | 二つの初期provider decision、pure policy/proxy責務分離 |
| REF-3 | Go standard library `crypto/rand`、`crypto/sha256`、`encoding/base64` | repository toolchainのGo version | unpredictable handle、digest-only key、canonical base64url |

### 依存状態

| 依存 | 状態 (`ready` / `pending`) | planning参照 | `ready`後に固定する値 |
|---|---|---|---|
| TASK-0035 | `ready` | fail-closed/fixed safe error pattern | config/schemaは変更せずerror境界だけ参照する |
| TASK-0039 | `ready` | REF-2 | provider operation/host/repository scopeを二つのallow decisionへ固定する |

### 許可パス

- `tools/dev-agent-harness/internal/capability/`
- `tools/dev-agent-harness/README.md`

### 完了経路preflight

| 確認対象 | 結果 | コマンドまたは根拠 |
|---|---|---|
| 完了checker | `ready` | TASK-0037のplanning/candidate/completion gateとpost-merge `task-check`を使う |
| 権限 | `ready` | productはmemory、crypto entropy、clockだけを使い、Credential/network/file/process権限なし |
| 依存状態と参照 | `ready` | TASK-0035/0039完了、REF-1〜3を固定 |
| 生成物の有無と更新方法 | `ready` | Go source/testとREADMEだけ。configure/Makefile/generated配布物は変更しない |
| 割当ワークツリー | `ready` | `worktrees/TASK-0040-dev-agent-capability-registry` |
| Lapログの書込・Schema・`repository annotation` | `ready` | 標準3 commits、legacy bindingや新Schemaを追加しない |

### 未決事項

- なし

### `Dependency-ready reconciliation`

<!-- 依存ready時にMainがready参照、planning参照との差分、AC/設計/scope/QAへの影響、再承認結果を追記する。依存なし又は未readyならN/Aとする。 -->

- TASK-0035とTASK-0039はplanning開始時点でready。実proxy/credential brokerはpendingのまま対象外とし、Grantを実通信または実Credentialの許可とみなさない。

## 背景

TASK-0039でHTTP requestのallowlist判断は固定したが、Agentへ実Credentialを渡さずrequestを同じAgent/workspace/providerへ束縛するstateは未実装である。proxyとCredential storeを先に接続すると、scope検証、token secrecy、network/TLSの欠陥が混在する。digestだけを保持する短命registryを先に完成させ、漏えいhandleの利用可能範囲と原子的消費をhermetic fixtureで検証する。

## 検討すべき設計観点

- handleを実Credentialの暗号化表現にせず、broker-local stateへのrandom referenceにする。
- scopeの全条件をANDで完全一致させ、repository/host/operationのprefix、suffix、normalizationを許可根拠にしない。
- mismatchで使用回数を消費せず、成功、失効、epoch更新を一つのmutexでlinearizeする。
- productionのentropy/clockとdeterministic test dependencyを分離し、test hookを外部APIにしない。
- active entry数やexpiry cleanupのbackground処理を先取りせず、access時削除とepoch全削除だけでbounded lifecycleを保つ。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] 製品変更のplanning/candidate/completionの3 commit経路を満たしている。
- [ ] 同一candidateの独立REVIEW/QA、candidateで一回のroot `make check`、post-merge `task-check`を満たしている。
- [ ] 再利用可能なscope/消費/revocation契約をSemantic Wikiへ記録している。

## 関連コンテキスト

### 意味 Wiki

- Opaque capability registryのscope、digest-only保持、atomic consume、restart fail-safe。

### 判断

- 初期provider surfaceはTASK-0039の二つだけに固定し、generic capability frameworkへ広げない。
- Registryはin-memoryとし、restartで全handleを失効させる。

### 適用しなかった重要な判断

- raw handleの永続化、実Credential同梱、JWT/self-contained token、background cleanup、汎用operation schemaは採用しない。
