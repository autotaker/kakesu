---
task_id: "TASK-0061"
title: "Agent向けCapability発行・失効control sessionを実装する"
status: draft
created_at: "2026-08-02"
---

# TASK-0061 Agent向けCapability発行・失効control sessionを実装する

## `Planning input packet`（Main Agent所有）

このsectionをPlannerとQAへ渡す唯一の`planning input packet`とし、各内容をPLAN/QA_PLANへ複製しない。

### 目的

現行`dev-agent-egress`が生成する空のCapability Registryへ、kernel peerで識別済みのAgentが限定されたOpaque capabilityを発行・失効できるcontrol sessionを追加する。発行した同じregistry entryを既存のGitHub/OpenAI外向き通信transactionが消費できるようにし、認証情報をAgentへ渡さずPhase 1の実通信経路を成立させる。

### 対象と対象外

#### 対象

- 既存egress Unix socket上で、通常のHTTPS `CONNECT`と明確に区別できるboundedな一接続一操作control protocolを定義する。
- GitHubは設定allowlist内の正規`owner/repo`一件、OpenAIは設定に許可modelがある場合だけ、接続contextへ束縛済みのAgent instance/UID/workspaceに対して短命・回数制限付き`cap_...`を発行する。
- 発行と外向き通信で同じin-memory `capability.Registry`を共有する。別registry、永続化、実認証情報コピーを作らない。
- Agentが提示した正規handleを失効できる。unknown/malformed/期限切れ、主体・provider・repository・操作不一致は固定拒否とする。
- control requestはstrict、上限付き、Content-Length必須、chunked/upgrade/keep-aliveなし、一操作後closeとし、入力値・handle・下位エラーを診断へ出さない。
- 既存のCONNECT/TLS/HTTP外向き通信経路を回帰させないhermetic testと、発行→既存transaction消費→再利用拒否のcomposition testを追加する。

#### 対象外

- `git-credential-dev-agent`、launcher、環境変数注入、Unix socket clientは次Taskとする。
- Git Smart HTTP、`github.com`、push、Approval、Tailscale、Passkey、Codex auth例外は実装しない。
- 新しいsocket/unit/listener、TCP/localhost入口、registry永続化、cache、再試行、監査永続化を追加しない。
- 実GitHub/OpenAI、実DNS/TLS、実NSS/別UID、systemd socket、VPS配置をhermetic PASSで代替しない。
- Kakesu本体のruntime、Go workspace、Schema、依存、生成物は変更しない。

### 受け入れ条件

<!-- AC-IDはTask内で一意かつ安定させ、観測可能な結果をここに一度だけ記載する。 -->

- [ ] AC-1: peer binderがcontextへ束縛したAgent instance/UID/workspace以外からsubjectを受け取らず、設定allowlist内GitHub repositoryまたはOpenAIだけへ固定TTL・回数上限のOpaque capabilityを発行する。
- [ ] AC-2: control protocolは既存egress socket上の通常CONNECTと衝突せず、strictかつboundedな一接続一操作だけを受理し、malformed/unknown/過剰入力/early bytes/chunked/keep-aliveを認証情報取得前に固定拒否する。
- [ ] AC-3: 発行controlと既存`egresstransaction`は同じRegistry instanceを共有し、発行handleを対応scopeで消費できる。不一致や再利用は拒否され、失敗時にrollback/retryしない。
- [ ] AC-4: revokeは正規handle一件だけを失効し、以後の使用を拒否する。unknown/malformed handleや不正主体で実認証情報、handle、URL、設定値、下位エラーを出力しない。
- [ ] AC-5: 既存CONNECT/TLS/HTTP、GitHub REST/OpenAI policy、provider credential置換、socket/peer identityの意味を変更せず、focused Go tests、race、root `make check`、`git diff --check`がPASSする。
- [ ] AC-6: candidateは承認済み8パス以内、約1,000行規模に収まり、Kakesu runtime・Schema・依存・配布境界とlive VPS状態を変更しない。

### 安定した参照

| 参照ID | 対象 | 固定改訂/ダイジェスト | 用途 |
|---|---|---|---|
| REF-1 | Harness設計 §7.2、§9.1〜9.2、Phase 1 | main `7534307` | Opaque capabilityのsubject/scopeとGH/OpenAI最小flow |
| REF-2 | 既存Capability Registry/transaction | main `7534307` | 発行・消費・失効を共有する正本 |
| REF-3 | 既存CONNECT/session/service composition | main `7534307` | 同じAgent socket、peer context、通常proxy回帰境界 |

### 依存状態

| 依存 | 状態 (`ready` / `pending`) | planning参照 | `ready`後に固定する値 |
|---|---|---|---|
| なし | `ready` | N/A | N/A |

### 許可パス

- `tools/dev-agent-harness/internal/capabilitycontrol/control.go`（新規）
- `tools/dev-agent-harness/internal/capabilitycontrol/control_test.go`（新規）
- `tools/dev-agent-harness/internal/connectsession/session.go`
- `tools/dev-agent-harness/internal/connectsession/session_test.go`
- `tools/dev-agent-harness/internal/egressservice/service.go`
- `tools/dev-agent-harness/internal/egressservice/service_test.go`
- `tools/dev-agent-harness/internal/capability/capability.go`
- `tools/dev-agent-harness/README.md`

### 完了経路preflight

| 確認対象 | 結果 | コマンドまたは根拠 |
|---|---|---|
| 完了checker | `ready` | 通常product 3コミット経路、同一candidateの独立REVIEW/QA、no-ff completionを使用する |
| 権限 | `ready` | dev-lunaがTask worktreeの8パスだけを実装し、MainだけがGit統合する |
| 依存状態と参照 | `ready` | TASK-0039〜0058でpolicy、registry、transaction、peer context、session、service compositionがmainへ反映済み |
| 生成物の有無と更新方法 | `ready` | 新Go packageだけ。code generation、dependency追加、configure再生成なし |
| 割当ワークツリー | `ready` | `worktrees/TASK-0061-dev-agent-capability-control` |
| Lapログの書込・Schema・`repository annotation` | `not-applicable` | 新規log/Schema/annotationなし |

### 未決事項

- なし

### `Dependency-ready reconciliation`

<!-- 依存ready時にMainがready参照、planning参照との差分、AC/設計/スコープ/QAへの影響、再承認結果を追記する。依存なし又は未readyならN/Aとする。 -->

- N/A

## 背景

TASK-0058で外向き通信サービスのproduction graphは完成したが、Registryはstartup時に空で、発行経路がない。したがってAgentが正しいGitHub/OpenAIリクエストを作ってもOpaque capabilityを得られず、現状は常に拒否される。Git helperやlauncherより先に、同じRegistryへ安全に到達する最小control境界が必要である。

## 検討すべき設計観点

- controlは既存peer認証済みsocketを再利用し、新しい到達経路を増やさない。
- repository/provider allowlistは既存configから固定し、Agent入力だけでscopeを拡大しない。
- issuanceの成功応答にOpaque handleだけを含め、grant内部情報や実認証情報を返さない。
- protocol parserとissuer policyを分け、通常CONNECTのparser/timeout/close契約を弱めない。
- 次Taskのcredential helperが利用できる小さなclient contractを固定するが、client実装は混ぜない。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] 選択した`change_class`の完了経路と`make check`を満たしている。
- [ ] 製品変更の場合: 実装、テスト、文書、同一案の独立REVIEW/QA、完了後の環境依存ケース確認が完了している。
- [ ] 安全契約変更の場合: Mainの意図・スコープ・受け入れ経路確認、契約検査、許可された統制文書差分の確認が完了している。

## 関連コンテキスト

### 意味 Wiki

- `docs/development/development-agent-harness.md` §7.2、§9、§14〜16

### 判断

- Git helperより先にCapability供給元を実装し、供給元のないclient scaffoldを作らない。

### 適用しなかった重要な判断

- なし
