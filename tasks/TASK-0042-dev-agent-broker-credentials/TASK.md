---
task_id: "TASK-0042"
title: "Broker CredentialsとGitHub App JWTを実装する"
status: plan
created_at: "2026-08-01"
---

# TASK-0042 Broker CredentialsとGitHub App JWTを実装する

## `Planning input packet`（Main Agent所有）

このsectionをPlannerとQAへ渡す唯一の`planning input packet`とし、各内容をPLAN/QA_PLANへ複製しない。

### 目的

実CredentialをAgentから隔離したbroker processだけが使う最初の秘密情報境界を実装する。broker専用directoryからGitHub App client ID、installation ID、RSA private key、OpenAI API keyをfail-closedに読み込み、GitHub App認証用の短命RS256 JWTを生成する。秘密の読込み・検証・署名を一つの小さなpackageへ閉じ、後続のinstallation token交換やegress transaction接続が秘密ファイルを再解釈しない状態にする。

### 対象と対象外

#### 対象

- `internal/brokercredentials` packageと、固定basename `github-client-id`、`github-installation-id`、`github-private-key.pem`、`openai-api-key`から成るbroker専用directory layout。
- 実効UIDがnon-rootで、directoryと全fileが同じUID所有、group/other権限なし、directoryはowner read/execute可能、fileはowner read可能かつ実行不可であることの検証。
- Ubuntu実装でdirectory descriptorを一度開き、固定basenameを`openat` + `O_NOFOLLOW|O_CLOEXEC|O_NONBLOCK`で読み、regular file、size上限、読込み前後metadata不変を検証する境界。非Linuxの開発test用実装も同じfile policyと固定basenameを保つ。
- text fileは末尾LFを0又は1個だけ許し、GitHub client IDは1〜128 byteのvisible ASCII、installation IDはleading zeroなしの正の10進`int64`、OpenAI API keyはprefixを仮定せず1〜4,096 byteのvisible ASCIIとして検証する。
- PEMは単一blockだけを受理し、unencrypted PKCS#1又はPKCS#8 RSA private key、2048〜8192 bit、`rsa.PrivateKey.Validate`成功を要求する。
- 読込み済みbundleからOpenAI API key、GitHub installation ID、GitHub App JWTをtrusted broker codeへ提供する最小API。private key/PEMを返すAPI、`String`/JSON marshal、log出力は作らない。
- GitHub App JWTは`alg=RS256`、`typ=JWT`、JSON数値のUnix秒で`iat=now-60秒`、`exp=now+9分`、`iss=GitHub client ID`を持ち、各呼出しでPKCS#1 v1.5/SHA-256署名する。同じUnix秒の同じclaimから同一JWTが得られることを許す。
- file policy、入力parse、RSA形式、JWT署名/claim/時刻、固定errorと秘密非漏洩をhermetic unit testで検証する。

#### 対象外

- GitHub installation access tokenのHTTP交換、cache/refresh/revocation、OpenAI又はGitHubへの実通信、egress transactionのresolver実装。
- HTTP client/server、proxy/TLS/CA/DNS、socket、retry、redirect、response処理、audit/log、metrics。
- Credential作成、rotate、書込み、削除、secret manager/keyring、encrypted PEM、passphrase、HSM/KMS。
- AgentへのCredential公開、環境変数/command lineからの読込み、Codex自身の認証情報、Git credential helper、push/Approval/Tailscale/Passkey。
- config/CLI/systemd/provision/installの変更、Kakesu本体、外部Go module。
- 非Linuxを本番support済みとすること、および実UbuntuのUID/権限隔離をローカルunit testだけでPASSとすること。

### 受け入れ条件

- [ ] AC-1: validな固定4-file directoryをnon-rootのowner UIDで`Load`するとbundleが得られる。空path、root UID、directory/fileのowner不一致、group/other権限、directoryのread/execute不足、fileのread不足/実行bit、symlink、directoryでないpath、missing path、regularでないnode、size超過、読込み中metadata変化は同じ固定load errorで拒否され、FIFOでblockしない。Ubuntuは一つのdirectory descriptorに固定basenameを束縛して読み、caller文字列をrelative filenameへ使わない。
- [ ] AC-2: client ID、installation ID、OpenAI keyは末尾LFを0又は1個だけ除いた後に各境界を満たす場合だけ受理される。空、CR/LF混入、space/control/non-ASCII、過長、installation IDの符号/leading zero/overflow/zeroは固定load errorとなる。OpenAI keyのprefixやprovider固有長は仮定しない。
- [ ] AC-3: private keyは余剰PEM/dataなしの単一unencrypted PKCS#1又はPKCS#8 RSA keyだけを受理し、non-RSA、encrypted、invalid、2048 bit未満、8192 bit超過を固定load errorで拒否する。bundleはraw PEM/private keyを返さず、入力byteを保持せず、秘密を含むformat/marshal APIを持たない。
- [ ] AC-4: `GitHubAppJWT`は各呼出しで署名処理を行い3-part base64url JWTを返す。header/payloadは固定fieldだけ、基準時刻を`now.UTC().Unix()`の整数秒とし、JSON数値の`iat=now-60`、`exp=now+540`と文字列`iss=client ID`を持つ。RS256署名をbundleの公開鍵で検証でき、同じ基準秒なら同一JWTを許す。署名失敗は固定JWT errorとなり、JWT/private key/parser detailをerrorへ含めない。
- [ ] AC-5: bundleのtrusted broker APIはclient ID、installation ID、OpenAI key、短命JWTに限定される。packageは環境変数、command line、network、process、DNS、socket、永続書込み、logを使わず、load/JWT errorはpath、file名、UID、mode、入力、key/token、parser/OS detailを含まない。
- [ ] AC-6: unit testsはvalid load、各file policy、text境界、両RSA形式、拒否形式、JWT claim/signature/time、caller入力不変、固定error/non-leakを検出する。`go test -race ./internal/brokercredentials`、harness `make check`/`make distcheck`、root `make check`がPASSし、base...candidateの対象packageとREADMEの追加＋削除合計は1,200行以下である。

### 安定した参照

| 参照ID | 対象 | 固定改訂/ダイジェスト | 用途 |
|---|---|---|---|
| REF-1 | `docs/development/development-agent-harness.md` | main `1296427`時点の§7、§9、§14〜17 | owner管理secret、broker-only置換、Agent非露出、段階導入 |
| REF-2 | TASK-0041 egress transaction | candidate `c369ec4` / completion `1296427` | trusted resolver/Forwarder境界とCredential上限 |
| REF-3 | [GitHub App JWT公式文書](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/generating-a-json-web-token-jwt-for-a-github-app) | 2026-08-01取得 | RS256、iat skew、10分以内のexp、client ID issuer、Bearer利用 |
| REF-4 | [OpenAI domain secrets公式文書](https://developers.openai.com/api/docs/guides/tools-shell#domain-secrets) | 2026-08-01取得 | placeholderと実secretの分離、許可destinationでだけの認証置換、model-visible stateへの非残存 |

### 依存状態

| 依存 | 状態 (`ready` / `pending`) | planning参照 | `ready`後に固定する値 |
|---|---|---|---|
| TASK-0041 | `ready` | REF-2 | bundle利用者はtrusted broker resolverであり、Agent/transaction callerへ実Credentialを返さない |

### 許可パス

- `tools/dev-agent-harness/internal/brokercredentials/`
- `tools/dev-agent-harness/README.md`

### 完了経路preflight

| 確認対象 | 結果 | コマンドまたは根拠 |
|---|---|---|
| 完了checker | `ready` | planning/candidate/completionの3 commit gateとpost-merge `task-check`を使う |
| 権限 | `ready` | local fixtureだけを作成し、実Credential、network、root/chown、外部作用を使わない。実Ubuntu隔離は後続live E2E |
| 依存状態と参照 | `ready` | TASK-0041完了。GitHub/OpenAI公式契約を2026-08-01に固定 |
| 生成物の有無と更新方法 | `ready` | Go source/testとREADMEだけ。外部module、configure/Makefile/generated配布物は変更しない |
| 割当ワークツリー | `ready` | `worktrees/TASK-0042-dev-agent-broker-credentials` |
| Lapログの書込・Schema・`repository annotation` | `ready` | 標準3 commits。新Schema、digest転記、追加機械checkなし |

### 未決事項

- なし

### `Dependency-ready reconciliation`

- TASK-0041はplanning開始時点でready。今回のbundleはConsume後に呼ばれるtrusted resolverの内部だけで使い、transaction API又はAgent側へsecret sourceを追加しない。installation token交換とresolver接続はpendingのまま対象外とする。

## 背景

TASK-0041により未認可requestがCredential解決へ到達しない順序は固定できたが、resolverが使う実Credential sourceは未実装である。ここで静的installation tokenを直接置くと恒久token運用になり、秘密ファイルの解釈を後続adapterごとに重複させる。まずbroker-ownedな長期secretと短命JWT生成だけを、外部通信なしで単体検証できる境界として実装する。

## 検討すべき設計観点

- secret directoryはtrusted deployment設定が指定する絶対pathとし、内部の名前はcompile-timeの4 basenameだけに固定する。Agent/caller入力をpathへ連結しない。
- Linuxではdirectory FDを認可対象として固定してから`openat`する。fileはopen後のdescriptorだけをstat/read/statし、pathをstatしてから再openしない。
- file policyは実効broker UIDとのownershipとgroup/other不可視性に絞る。親directory tree全体、ACL、mount、SELinux/AppArmorは後続Ubuntu live E2Eで検証し、このTaskへ汎用filesystem scannerを追加しない。
- private keyとAPI keyはbroker process memory内の実Credentialである。package APIをtrusted broker用途へ限定し、Agent RPC/HTTP response/logへ流す実装は対象外にする。
- JWTは外部JWT dependencyを追加せず標準`crypto/rsa`、`crypto/sha256`、`encoding/base64`、`encoding/json`で生成する。期限はGitHub上限より余裕を持たせる。
- test用RSA keyはruntime生成し、repositoryへsecret fixtureを置かない。実owner/chown/root隔離やGitHubでの受理は後続live E2Eへ残す。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] planning/candidate/completionの3 commit経路と`make check`を満たしている。
- [ ] 同一candidateの独立REVIEW/QA、必要なSemantic Wiki更新、post-merge `task-check`を完了している。
- [ ] 実Ubuntu UID/権限隔離と実GitHub token交換をPASSと誤記せず、後続live E2Eとして残している。

## 関連コンテキスト

### 意味 Wiki

- broker secret directoryと短命GitHub App JWTの境界。

### 判断

- 長期GitHub App private keyとOpenAI API keyはbroker専用directoryからだけ読み、GitHub accessには静的installation tokenでなく短命JWTから開始する。

### 適用しなかった重要な判断

- 静的GitHub installation token、環境変数secret、generic secret manager abstraction、JWT外部library、親directory再帰scanner、実HTTP交換の同梱は採用しない。
