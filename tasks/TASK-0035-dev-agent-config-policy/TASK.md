---
task_id: "TASK-0035"
title: "Development Agent Harnessの設定・deny-by-defaultポリシー基盤を実装する"
status: draft
created_at: "2026-08-01"
---

# TASK-0035 Development Agent Harnessの設定・deny-by-defaultポリシー基盤を実装する

## `Planning input packet`（Main Agent所有）

このsectionをPlannerとQAへ渡す唯一の`planning input packet`とし、各内容をPLAN/QA_PLANへ複製しない。

### 目的

`tools/dev-agent-harness`に、後続のBroker、Egress Proxy、Approval Service、Launcherが共有するversion 1の設定契約と、許可を一つも暗黙付与しないdeny-by-defaultポリシー基盤を実装する。管理者がインストール後の設定をサービス起動前にローカル検証できる`dev-agent-harness-setup check-config`を提供し、不正・曖昧・危険な設定を外部作用なしで拒否する。

### 対象と対象外

#### 対象

- JSON設定version 1の型、上限付き読込、strict decode、意味検証。
- 未知field、重複key、未知version、末尾data、非regular file、symbolic link、過大file、group/world writable fileの拒否。
- 絶対かつcleanな設定・状態・runtime path、相互に異なるLinux user名、`network.default == "deny"`の検証。
- `dev-agent-harness-setup check-config --config PATH`の成功・失敗契約と、秘密や設定全文を出力しない診断。
- configure後の設定例、既存`make check`、`make distcheck`との整合。
- valid/invalid fixtureを使う単体・CLIテスト。

#### 対象外

- 実Credential、Opaque capability、provider token、secret fileの読込・保存・置換。
- GitHub、OpenAI、Tailscaleその他の外部通信、TLS/HTTP proxy、Git credential helper。
- Unix socket/IPC、永続状態、監査log、approval/push state machine。
- OS user作成、systemd enable/start、namespace、firewall、sudo/PAM、VPSへのinstall。
- `dev-agent-broker`、`dev-agent-egress`、`dev-agent-approval`、`dev-agent-launcher`の通常起動を成功させること。
- Kakesu本体のruntime、Go workspace/module、build、配布物、Schema、Plane契約への組込み。

### 受け入れ条件

- [ ] AC-1: version 1設定例を指定した`dev-agent-harness-setup check-config --config PATH`はexit 0になり、設定version、deny既定、検証済みであることだけを含む安定したsummaryをstdoutへ出し、設定全文・path・user名・credentialらしき値を出力しない。
- [ ] AC-2: 未知field、同一object内の重複key、未知version、複数JSON値または末尾の非空白dataを含む入力はexit non-zeroとなり、入力内容をechoせず原因分類をstderrへ出す。
- [ ] AC-3: 空または相対/非cleanなpath、重複するuser名、不正なLinux user名、`network.default`が`deny`以外の設定はexit non-zeroとなる。空のallowlistやfield省略を暗黙の許可へ変換しない。
- [ ] AC-4: 設定pathがsymbolic link、regular file以外、64 KiB超過、またはgroup/world writableの場合は読込前後のraceを考慮したfile descriptor基準の検査で拒否され、外部作用を起こさない。
- [ ] AC-5: valid、unknown、duplicate、version、trailing-data、path、user、network、file-type、permission、sizeのpositive/negative testがあり、negative testを意図的に受理側へ反転すると該当testが失敗する。
- [ ] AC-6: `dev-agent-harness-setup`の`--help`と`--version`は既存契約を維持し、`check-config`以外の通常操作および他の5バイナリの通常起動は従来どおりfail-closedする。
- [ ] AC-7: `tools/dev-agent-harness`内で`go test ./...`、`make check`、`make distcheck`が成功し、`make install DESTDIR=...`へ配置されるconfigure済み設定例を`check-config`で検証できる。
- [ ] AC-8: 手書き実装差分は実装コード・テストを合わせて700〜1,200行を目安とし、生成済み`configure`、fixture、文書、Task/Wiki証跡は除外する。1,200行を超える見込み、または新たなsecurity boundaryが必要になった場合はMainが別Taskへ分割する。

### 安定した参照

| 参照ID | 対象 | 固定改訂/ダイジェスト | 用途 |
|---|---|---|---|
| REF-1 | `docs/development/development-agent-harness.md` | commit `14af73e`時点 | 外部開発基盤、fail-closed、段階導入、配置境界 |
| REF-2 | `tools/dev-agent-harness` scaffold | commit `14af73e` | 既存CLI、設定例、configure/make install契約 |
| REF-3 | `tools/dev-agent-harness/go.mod` | commit `14af73e`の`go 1.24`、外部module依存なし | 実装言語と標準libraryのみの依存境界 |

### 依存状態

| 依存 | 状態 (`ready` / `pending`) | planning参照 | `ready`後に固定する値 |
|---|---|---|---|
| 設計とscaffold | `ready` | REF-1、REF-2 | N/A |
| 実VPS/Tailscale/GitHub/OpenAI | `pending` | 本Taskの対象外 | 後続live-e2e Taskで固定し、本TaskのPASS条件にしない |

### 許可パス

- `tools/dev-agent-harness/cmd/dev-agent-harness-setup/`
- `tools/dev-agent-harness/internal/command/`
- `tools/dev-agent-harness/internal/config/`
- `tools/dev-agent-harness/config/harness.json.example.in`
- `tools/dev-agent-harness/README.md`

### 完了経路preflight

| 確認対象 | 結果 | コマンドまたは根拠 |
|---|---|---|
| 完了checker | `ready` | `make task-preflight TASK=TASK-0035`と`make task-check TASK=TASK-0035`を使用する |
| 権限 | `ready` | repository内のGo source、test、設定例、READMEだけを変更し、host/VPSや外部serviceを変更しない |
| 依存状態と参照 | `ready` | REF-1〜REF-3を固定し、外部環境依存は対象外 |
| 生成物の有無と更新方法 | `ready` | `configure`を変更・再生成しない。test用一時fileと`distcheck`出力はGit管理外 |
| 割当ワークツリー | `ready` | `worktrees/TASK-0035-dev-agent-config-policy`、branch `task/TASK-0035-dev-agent-config-policy` |
| Lapログの書込・Schema・`repository annotation` | `ready` | Main管理rootだけがTask/Wiki証跡を更新し、製品worktreeからmain管理pathへ書かない |

### 未決事項

- エラーの内部型とsummaryの具体的な文字列は、ACの非漏洩・安定性を維持する範囲でPlannerが決める。
- user名の検証はLinux useradd互換の安全な部分集合を採用し、distribution固有の最大集合を再現しない。

### `Dependency-ready reconciliation`

<!-- 依存ready時にMainがready参照、planning参照との差分、AC/設計/scope/QAへの影響、再承認結果を追記する。依存なし又は未readyならN/Aとする。 -->

- 実環境依存は本Taskの対象外である。後続Taskでreadyになっても本Taskへ追加せず、個別TaskのACとQAへ固定する。

## 背景

scaffoldの設定例は配置path、OS user、deny既定を表現しているが、現在のバイナリは設定を読まず通常起動を一律拒否する。Brokerやproxyを先に実装すると、各processが独自に曖昧な設定を解釈し、未知fieldや危険なfileを受理する恐れがある。外部通信を有効化する前に、共有するstrictな設定境界と管理者向け事前検証を単独Taskで固定する。

## 検討すべき設計観点

- JSON token列を一度検査してobject内の重複keyを拒否した後、`DisallowUnknownFields`相当のstrict decodeを行う順序。
- pathを文字列だけでなく、open済みfile descriptorのtype、size、modeで検査し、symbolic linkをfail-closedに扱う方法。
- parse、schema、semantic、file-policy、I/Oを機械可読な分類へ分けつつ、入力値をerrorへ埋め込まない診断。
- CLIのexit code/stdout/stderrをtest可能にし、service用commandの既存fail-closed挙動を回帰させない構造。
- provider固有fieldや将来のallowlistを先取りせず、version追加時に明示migrationを要求できる型境界。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] 選択した`change_class`の完了経路と`make check`を満たしている。
- [ ] 製品変更の場合: 実装、テスト、文書、同一案の独立REVIEW/QA、`merge_tree`確認、環境依存ケース、Wiki取り込みが完了している。
- [ ] 安全契約変更の場合: 独立計画レビュー、契約検査、`no-ff merge`、案/merge tree一致が完了し、製品REVIEW/QA PASSやWiki receiptを代用証跡として作成していない。

## 関連コンテキスト

### 意味 Wiki

- Task完了後、Development Agent HarnessがKakesu本体外であること、設定version 1、strict decode、deny既定、危険な設定fileの拒否条件を新規Wiki topicへ取り込む。

### 判断

- 最初の実装Taskは実Credentialやnetworkを扱わず、純粋な設定・ポリシー境界に限定する。
- 設定形式はscaffoldとの互換を維持してJSONとし、agent向けruntime protocolとしてJSONLへ変更しない。
- 外部Go dependencyを追加せず、標準libraryで実装する。

### 適用しなかった重要な判断

- JSON Schema generatorやYAML libraryの導入は、依存と生成物を増やすため本Taskでは採用しない。
- 設定fileにprovider credential、Opaque capability、host allowlistを追加する案は、該当security boundaryの後続Taskまで採用しない。
