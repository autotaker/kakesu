---
task_id: "TASK-0053"
title: "broker credential bundleへproxy CA読込を統合する"
status: plan
created_at: "2026-08-02"
---

# TASK-0053 broker credential bundleへproxy CA読込を統合する

## `Planning input packet`（Main Agent所有）

このsectionをPlannerとQAへ渡す唯一の`planning input packet`とし、各内容をPLAN/QA_PLANへ複製しない。

### 目的

既存`brokercredentials.Load`が安全に一度だけ開くbroker専用directoryへproxy CA certificate/keyを固定basenameで追加し、他の実認証情報と同じfile policyでall-or-nothingに読み込んで、検証済み`proxyca.Authority`だけをBundleからtrusted brokerへ渡せるようにする。別のfile readerを複製せず、CA private key又は入力PEMを公開API、Format、errorへ出さない。

### 対象と対象外

#### 対象

- broker credential directoryの固定入力を既存4 filesから6 filesへ拡張し、`proxy-ca-cert.pem`と`proxy-ca-key.pem`を追加する。既存Linux `openat`/`O_NOFOLLOW`/owner・mode・regular file・size・read前後metadata検査を同じdirectory fdから両fileへ適用する。
- `brokercredentials.Load`が6 filesを全て読み終えて既存GitHub/OpenAI値を検証した後、既存package-private clock seamを参照するclockで`proxyca.New`を一回呼ぶ。CA不正、鍵不一致、期限不足を他のload failureと同じ固定`ErrLoad`へ畳み、partial Bundleを返さない。
- `Bundle.ProxyCAAuthority()`が検証済みAuthorityへのread-onlyな利用入口を返す。nil/zero/破損Bundleはnilを返し、PEM/key/signer accessor、marshal、file path、raw input accessorを追加しない。Authorityの既存`PublicCertificatePEM`/`Issue`だけを利用する。
- test fixtureを6-file layoutへ更新し、CA成功、欠落、symlink/special node、owner/mode/size、read中metadata変化、malformed/multiple/mismatched/non-P256/non-CA/expired又はleaf発行余命不足をfail closedに検出する。既存4 credential/JWT検証を弱めない。
- READMEへbroker bundleがCA materialを同じ秘密directoryからstartup時にsnapshotし、公開CA copyとhost限定Authorityだけをtrusted compositionへ渡す境界を追記する。

#### 対象外

- CA certificate/private keyの生成、書込み、atomic replace、rotate、renewal、reload、watch、rollback、backup、失効。既存fileをstartup時に読むだけとする。
- Agent側への公開CA配置、OS trust store更新、environment変数、client設定、certificate pinning、実TLS client確認。
- 実UID/user作成、directory/file provision、chmod/chown、systemd credential、kernel keyring、外部secret manager。
- `proxyca`の証明書意味、host allowlist、leaf lifetime/APIの変更。
- listener/session/broker composition、config/CLI/service binary、実GitHub/OpenAI/network、VPS live E2E。
- 非Linux readerのproduction保証。既存の開発用互換readerの意味を広げない。

### 受け入れ条件

- [ ] AC-1: fixed broker directory layoutは既存4 basenameに`proxy-ca-cert.pem`、`proxy-ca-key.pem`を重複なく固定順で加えた6 filesだけとする。`Load`は全6 files成功時だけBundleを返し、欠落、余分な解釈、空入力、path不正、nil/zero状態をpanicせず固定`ErrLoad`へ畳む。既存credential値、JWT、Formatの公開契約を変更しない。
- [ ] AC-2: Linuxでは既存の一directory fdと各`openat`によりCA二fileにも絶対clean directory、effective UID非root、directory owner/mode/type、file owner/mode/regular/no-follow/size、read前後のdev/inode/mode/uid/gid/size/nlink/mtime/ctime同一性を適用する。symlink/FIFO/device、hardlink又はmetadata/content race、group/other accessを拒否し、別path reopenやfallbackを使わない。
- [ ] AC-3: `Load`は既存credential検証後にCA certificate/keyを`proxyca.New`へ一回だけ渡し、single PEM、self-signed CA、ECDSA P-256、key一致、現在有効かつleaf発行に十分な余命という既存契約を再実装せず委譲する。失敗はpath、PEM、key、certificate、時刻、parser/OS detailを含まない`ErrLoad`でpartial Bundleなしとなり、入力byte sliceをBundleへ保持しない。
- [ ] AC-4: 成功Bundleの`ProxyCAAuthority()`だけが非nil Authorityを返す。nil/zero/破損Bundleはnilとなり、private key/PEM/signer/file accessorを公開しない。取得Authorityは公開CA certificateの独立copyとexact 2 hostsのfresh leafだけを発行し、Bundle/AuthorityのFormat又はerrorにcredential/CA detailを出さない。
- [ ] AC-5: valid fixtureで既存ClientID/InstallationID/OpenAI key/JWTと新Authorityが同時に利用でき、CA failure時は既存値だけのBundleを返さない。並行した公開CA copy/leaf/JWT利用でstateやsecretが混線せず、callerが公開CA sliceを変更しても後続結果へ影響しない。既存security negative testsのfailure detectionを維持する。
- [ ] AC-6: hermetic race testが6-file all-or-nothing、Linux file policy/TOCTOU、CA parse/key/validity negative、nil/Format/non-leak accessor、2-host issue/public copy、既存credential/JWT regression、並行隔離を検出する。focused race、harness check/distcheck、README lint、candidate launcher root `make check`がPASSし、許可path内で追加＋削除1,000行以下とする。

### 安定した参照

| 参照ID | 対象 | 固定改訂/ダイジェスト | 用途 |
|---|---|---|---|
| REF-1 | TASK-0042 broker credentials | candidate `5fea9b8` / completion `387fe9f` | trusted directory、一directory fd、fixed basename、Bundle/JWT契約 |
| REF-2 | TASK-0050 proxy CA | candidate `ffc220f` / completion `688e927` | CA/keyの既存検証と公開CA/leaf API |
| REF-3 | Broker credentials / Proxy CA Wiki | main `c1b00a9`時点 | 秘密file境界、未実装lifecycle/live制限 |

### 依存状態

| 依存 | 状態 (`ready` / `pending`) | planning参照 | `ready`後に固定する値 |
|---|---|---|---|
| TASK-0042 | `ready` | REF-1 | 既存secure file readerとBundleを拡張し意味を弱めない |
| TASK-0050 | `ready` | REF-2 | `proxyca.New`とAuthority APIを変更せず利用する |

### 許可パス

- `tools/dev-agent-harness/internal/brokercredentials/`
- `tools/dev-agent-harness/README.md`

### 完了経路preflight

| 確認対象 | 結果 | コマンドまたは根拠 |
|---|---|---|
| 完了checker | `ready` | 標準planning/candidate/completion 3 commitsとpost-merge task-check |
| 権限 | `ready` | temp directory/test UID/in-memory generated CAだけ。実secret、root、network、trust store不使用 |
| 依存状態と参照 | `ready` | TASK-0042/0050完了。既存readerとproxyca APIだけを合成する |
| 生成物の有無と更新方法 | `ready` | Go source/testとREADMEだけ。config/build/dependency/生成物なし |
| 割当ワークツリー | `ready` | `worktrees/TASK-0053-dev-agent-broker-proxy-ca-files` |
| Lapログの書込・Schema・`repository annotation` | `ready` | 新Schema/機械check/digestなし、標準3 commits |

### 未決事項

- なし

### `Dependency-ready reconciliation`

<!-- 依存ready時にMainがready参照、planning参照との差分、AC/設計/scope/QAへの影響、再承認結果を追記する。依存なし又は未readyならN/Aとする。 -->

- TASK-0042/0050はいずれもplanning開始時点でready。CA file bytesはbrokercredentialsが一度だけ読み、意味検証と保持はproxycaへ委譲する。別reader又はCA parserを追加しない。

## 背景

in-memory proxy CAとlistener/sessionは完成したが、broker processがCA materialを安全に取得する経路がない。既存broker credential directory readerは同じ所有者・秘密file policyを既に実装しているため、CAだけ別readerにするとTOCTOU/security logicが重複する。Bundleへ同じstartup snapshotとして統合する。

## 検討すべき設計観点

- fixed basename集合とreader policyは一箇所に保ち、CA用にpath join/open/stat logicを複製しない。
- Bundleはparsed Authorityだけを保持し、入力PEM slicesを保持しない。Authorityは既存のprivate signer非公開契約を保つ。
- `nowUTC` seamはJWTとCA validation/issuanceで同じbroker clock sourceを参照するが、Authorityへ渡すClockFuncは呼出時にも現在値を取得する。
- CA失敗をcredential failureと区別する診断は秘密file状態を推測させるため公開しない。全load failureを既存ErrLoadへ畳む。
- rotate/reloadを暗黙に実装せず、Bundleは一startup snapshotとする。後続Taskでatomic replacement/restart手順を設計する。
- 新しいgate/check/glossary語は追加せず、既存testsを拡張する。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] planning/candidate/completionの3 commit経路とcandidate一回のroot `make check`を満たしている。
- [ ] 同一candidateの独立REVIEW/QAを完了し、実provision/rotate/trust/client/VPS live E2E未実施をPASSと誤記していない。
- [ ] 再利用可能な知識だけを意味Wikiへ同化し、post-merge `task-check`をPASSしている。

## 関連コンテキスト

### 意味 Wiki

- `wiki/semantic/schemas/development-agent-harness-broker-credentials.md`
- `wiki/semantic/schemas/development-agent-harness-proxy-ca.md`

### 判断

- CA materialを別readerにせず、broker credential startup snapshotへ統合する。

### 適用しなかった重要な判断

- CA専用file readerはsecurity/TOCTOU logicを重複させるため採用しない。
- CA private key/PEMをBundle accessorで返す案は秘密境界を広げるため採用しない。
- 本Taskでgenerate/rotate/watch/trust installまで行う案はread-only startup境界とOS作用を混在させるため採用しない。
