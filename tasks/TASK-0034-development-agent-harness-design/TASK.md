---
task_id: "TASK-0034"
title: "Kakesu開発用リモートAgent Harnessを設計する"
status: draft
created_at: "2026-08-01"
---

# TASK-0034 Kakesu開発用リモートAgent Harnessを設計する

## `Planning input packet`（Main Agent所有）

このsectionをPlannerとQAへ渡す唯一の`planning input packet`とし、各内容をPLAN/QA_PLANへ複製しない。

### 目的

さくらのVPS（Ubuntu）でKakesuを開発する際に、CodexのようなコーディングAgentを最小権限で運用する外部基盤の設計案を作る。スマートフォンから非同期に高リスク操作を承認できるようTailscaleを採用し、Kakesu本体のarchitecture、runtime、配布物、依存関係とは明確に分離する。

### 対象と対象外

#### 対象

- ログイン利用者、Agent実行利用者、Credential Broker、Egress Proxy、Approval Serviceの責務と信頼境界。
- GitHub CLI/API、Git Smart HTTPのfetch/pull/push、Agentが呼ぶOpenAI APIの最小対応。
- Codex自身の認証だけに実Credentialsを許す例外と、model生成コマンドへの非継承。
- Tailscale Serve、Grants、スマートフォン、Passkeyを使う非同期承認。
- Opaque capability、GitHub App installation token、pushのexact one-use grant、監査、失効、復旧、段階導入。
- 外部開発基盤であることを示す名称、配置、状態、非採用境界、将来採用時の再審査条件。

#### 対象外

- 製品コード、test、runtime/build設定、Schema、製品依存、生成製品入力/成果物、外部観測可能な製品挙動を変更しない。
- KakesuのPlane、Plane間message、Authority、Task/Agent Run state、tabletop scenarioへの組み込み。
- VPSへのinstall、Tailscale tailnet設定、GitHub App作成、Passkey登録、秘密配置、実サービス起動。
- Cloudflare Tunnel、Tailscale Funnel、SSHログインによる日常承認。
- root/kernel、Tailscale control plane、GitHub、OpenAI自体が侵害された場合の防御。

### 受け入れ条件

- [x] AC-1: 文書冒頭、component名、依存関係、対象外、将来採用条件から、これはKakesu本体ではなくKakesu開発専用の外部`Development Agent Harness`提案であると一意に判断できる。
- [x] AC-2: owner/login利用者とagent利用者を分離し、agentがownerの秘密、GitHub実token、OpenAI API key、Codex実Credentialsをfilesystem、environment、process、socket、network経由で取得できない境界を定義する。
- [x] AC-3: `gh`、Agent codeのOpenAI API、Git Smart HTTP fetch/pullがOpaque capabilityを提示し、Broker/Proxyだけが宛先・repo・operationに限定した短命実Credentialsへ置換するflowを定義する。
- [x] AC-4: pushはrepo、ref、expected old SHA、new SHA、force/delete、policy version、expiryへ束縛したexact one-use grantを必要とし、承認中にprocessを保持せず、承認後もAgentが明示的に再実行する非同期state machineを定義する。
- [x] AC-5: Approval UIはlocalhostだけで待受け、Tailscale Serve経由のtailnet内HTTPSに限定し、Funnelを無効化する。Tailscale identity/Grantsと毎回のPasskey user verificationを両方要求し、通知自体には承認権限を持たせない。
- [x] AC-6: fail-closedなnetwork/credential境界、脅威モデル、秘密の保管と例外、監査時の秘匿、失効、端末紛失、stale/expired/replay/TOCTOU、障害復旧を定義する。
- [x] AC-7: 最小構成、導入段階、未決の実装選択、検証matrix、live VPSでしか確認できない項目を区別し、後続の実装Taskへ分割できる。
- [x] AC-8: 公式参照を明示し、Tailscale Serve/Grants、Git credential helper、GitHub App installation authentication、Codex sandbox/network/credential注意事項と設計判断の対応が追跡できる。

### 安定した参照

| 参照ID | 対象 | 固定改訂/ダイジェスト | 用途 |
|---|---|---|---|
| REF-1 | Tailscale Serve documentation | 2026-01-20 validated版、2026-08-01取得 | tailnet限定HTTPS、identity header、localhost backend、Funnelとの差 |
| REF-2 | Tailscale Grants documentation/syntax | 2026-01-05 validated版、2026-08-01取得 | deny-by-defaultの接続・app capability制御 |
| REF-3 | Git `gitcredentials` documentation | Git 2.55.0掲載、2026-08-01取得 | custom credential helperの入出力契約 |
| REF-4 | GitHub App installation authentication | 2026-08-01取得 | repo/permission限定tokenとGit over HTTPS |
| REF-5 | OpenAI Agent approvals & security | 2026-08-01取得 | OS sandbox、command network policy、credential露出時の注意 |
| REF-6 | OpenAI Codex environment variables | 2026-08-01取得 | Codex auth、CA bundle、job-wide secretを避ける境界 |
| REF-7 | W3C Web Authentication Level 3 | 2025-01-13 Candidate Recommendation Snapshot | Passkey user verificationの標準根拠 |

### 依存状態

| 依存 | 状態 (`ready` / `pending`) | planning参照 | `ready`後に固定する値 |
|---|---|---|---|
| 外部製品の現行仕様 | `ready` | REF-1〜REF-7 | N/A（実装時に再確認） |
| Kakesu本体への採用判断 | `pending` | 本Taskでは対象外 | 別Taskで製品scope、Plane責務、Schema/E2E、依存方針を再審査する |

### 許可パス

- `docs/development/development-agent-harness.md`
- `docs/glossary.yml`
- `tasks/TASK-0034-development-agent-harness-design/`
- `backlog.yaml`

### 完了経路preflight

| 確認対象 | 結果 | コマンドまたは根拠 |
|---|---|---|
| 完了checker | `ready` | `make task-preflight TASK=TASK-0034`、安全契約v2 checker |
| 権限 | `ready` | workspace内の文書とTask証跡のみ。外部設定変更なし |
| 依存状態と参照 | `ready` | REF-1〜REF-7を取得。将来の製品採用は明示的に対象外 |
| 生成物の有無と更新方法 | `ready` | `uv run --project memory python scripts/validate-terminology.py --write`で`docs/glossary.yml`を更新する。`docs/99-glossary-index.md`は再生成後も内容不変であり、生成diffへ宣言しない |
| 割当ワークツリー | `ready` | 安全契約文書のためmain管理worktreeで作業し、製品実装worktreeを作らない |
| Lapログの書込・Schema・`repository annotation` | `ready` | 本TaskでLap Schema/JSONLを変更せず、実行中Task証跡のみを作る |

### 未決事項

- Credential Broker、Egress Proxy、Approval Serviceの実装言語とprocess分割は後続実装Taskで決める。
- Tailscale app capabilities headerをidentityの補助に使うかは、Passkeyとbackend-side allowlistを必須としたうえで実装時に決める。
- Codexの実Credentialsを親processへ注入する具体方式は、利用するCodex surface/版を固定する実装Taskで再確認する。

### `Dependency-ready reconciliation`

- Kakesu本体への採用は未readyかつ本Taskの完了条件ではない。採用判断がreadyになっても本Taskを拡張せず、別の製品変更Taskを起票する。
- 2026-08-01: 設計文書候補に対する`make lint-docs`が用語source/indexのdriftを検出した。ACの意味と製品除外は変更せず、`docs/glossary.yml`を通常予定path、`docs/99-glossary-index.md`を生成pathへ追加してPLAN/QA_PLANを再承認する。
- 2026-08-01: generator初回実行で`docs/glossary.yml`は更新されたが、project term集合は不変で`docs/99-glossary-index.md`にGit差分がないことを確認した。安全契約v2の生成path存在条件に従い、索引を宣言から外して`generated_paths: []`へ戻す。ACの意味と製品除外は不変。

## 背景

Agentへ通常のGitHub/OpenAI credentialsを渡すと、prompt injectionや悪意あるrepository codeから読み取り・外部送信され得る。一方、pushの都度SSHへログインして対話承認する方式は、スマートフォン中心の非同期運用に合わない。本設計は、低権限Agentへ用途限定のOpaque capabilityだけを渡し、実Credentialsと人間承認をAgent外へ隔離する。

## 検討すべき設計観点

- OS identity、filesystem、process、IPC、network namespace、systemd unitの境界。
- TLS interceptionを使う範囲と、HTTPS-only client、certificate pinning、HTTP/2/WebSocketへの対応。
- 認証置換前後でrequest body、host、repo、operationをどう検証するか。
- Codex本体auth例外を、Agent code用OpenAI API capabilityと混同させないこと。
- Tailscale device/user identity、Grants、Serve identity header、Passkeyの多層化。
- 長時間不在を前提にした承認requestの永続化、expiry、再実行、取消、競合。
- 将来Kakesuへ採用する場合に、外部開発基盤の仮定を製品要件へ無断移植しないこと。

## 完成の定義

- [x] AC-1〜AC-8を満たしている。
- [x] 独立したTASK-first QA_PLANと独立計画レビューが完了している。
- [ ] 安全契約用の対象検査、文書検査、`git diff --check`、`make check`がPASSしている。
- [ ] 公開する場合は安全契約の`no-ff merge`と案/merge tree一致を満たす。
- [x] 製品REVIEW/QA PASS、製品DEV証跡、Wiki receiptを代用証跡として作成していない。

## 関連コンテキスト

### 意味 Wiki

- 本Taskでは未使用。Kakesu本体の意味契約へ変更を加えない。

### 判断

- 承認経路はCloudflare Tunnelではなく、private tailnetへ閉じられるTailscale Serveを採用する。
- approvalは接続identityだけで完了させず、毎回Passkey user verificationを要求する。
- Git transportはSSH鍵をAgentへ持たせず、HTTPS Smart HTTPとcustom credential helperを使う。

### 適用しなかった重要な判断

- 公開URLが不要なのでCloudflare Tunnelは採用しない。
- Funnel、恒久PAT、Agent保有SSH key、owner SSH loginによる日常承認は採用しない。
