---
task_id: "TASK-0053"
change_class: "product_change"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-08-01T14:36:24Z"
planning_reviewed_by: ""
planning_review_decision: "pending"
planning_reviewed_at: ""
classification_approved_by: "main-agent-sol-high"
classification_approved_at: "2026-08-01T14:36:24Z"
classification_approval_reason: "broker credential reader、Bundle公開境界、Linux file policy、テストおよびREADMEの製品挙動を変更するため"
---

# TASK-0053 PLAN

## 分類・前提

これは製品変更である。TASKの依存状態はreadyであり、REF-1の単一 directory-fd reader とREF-2の
`proxyca.New` / `Authority` 契約を合成する。許可パス外、生成物、依存追加、設定・CLI・サービス構成は
変更しない。初回candidateは製品差分のみで追加・削除合計を約800行、最大1,000行に収める。

DEVは承認済みprofile `luna-xhigh` を使用し、planning/candidate/completionの標準3 commit経路に従う。
新しいgate又はcheckは設けない。candidate固定後に同一candidateから独立REVIEWとQAを行い、root
`make check` はDEV candidateで一回だけ実施する。

## AC対応

TASKの条件本文を再掲せず、`planning input packet`のAC-IDに設計を対応させる。

| AC-ID | 設計判断 | 変更パス | 実施順序 | 失敗時の扱い |
|---|---|---|---|---|
| AC-1 | basename配列を既存4件から固定順・重複なしの6件へ一元拡張する。readerの返却配列はこの順序だけで意味付け、Loadは件数不一致、空値、パス不正を既存の単一失敗に畳む。 | `internal/brokercredentials/brokercredentials.go`、各reader test | 1, 2 | `nil, ErrLoad`。既存credentials/JWT/Formatの公開契約は変更しない。 |
| AC-2 | Linuxの一度だけ開いたdirectory FDと同じ`readSecretAt`ループを二CA fileにも使う。既存のread前後metadata同一性比較を保ち、regular-file検査へ`nlink == 1`を明示してhardlinkを拒否する。 | `internal/brokercredentials/reader_linux.go`、package tests | 1, 4 | open/stat/read/metadataまたはpolicyのいずれの失敗も再open・fallbackなしで`ErrLoad`。 |
| AC-3 | 4既存credentialの構文検証が成功した後、読込み済みCA bytesを一回だけ`proxyca.New`へ渡す。broker側にPEM/key/certificate parserを追加しない。`nowUTC`を呼出し時に参照する`proxyca.ClockFunc` closureを渡す。 | `internal/brokercredentials/brokercredentials.go`、package tests | 2, 3 | CA不正、鍵不一致、時刻/余命不足を区別せず`nil, ErrLoad`。入力bytesとpartial Bundleを保持・返却しない。 |
| AC-4 | Bundleには検証済みのparsed `*proxyca.Authority`だけを非公開で保持し、`ProxyCAAuthority()`を唯一のCA利用入口にする。nil、zero、構築済みでない/破損状態はnilに正規化し、secret/raw/signer/path/marshal accessorは追加しない。 | `internal/brokercredentials/brokercredentials.go`、package tests | 3, 5 | accessorsと`Format`は固定・非漏洩の振舞いを維持し、無効状態でpanicしない。 |
| AC-5 | 6-file fixtureから既存credential/JWTとAuthorityを同時に取得できることを固定する。Authorityの公開PEM copy、2 host発行、JWT、並行利用を既存境界へ委譲して回帰を検出する。 | `internal/brokercredentials/*_test.go` | 4, 5 | どのCA load失敗もcredentialだけを含むBundleを返さない。callerによるpublic copy変更は内部状態へ到達しない。 |
| AC-6 | hermetic package testsをreader policy、6-file atomicity、CA拒否、境界非漏洩、clock、concurrencyへ拡張し、既存negative testが依然失敗を検出する形にする。READMEはsnapshotとtrusted composition境界だけを記す。 | `internal/brokercredentials/`、`README.md` | 4, 5, 6 | 不実行のgenerate/rotate/watch/trust/live VPSはPASS代替にせず対象外として明記する。 |

## 責務・境界・不変条件

### 読込みとall-or-nothing

- 固定layoutは既存4件の後に`proxy-ca-cert.pem`、`proxy-ca-key.pem`を置く。basename集合は一箇所だけを
  readerの走査順とLoadのindex対応に使い、CA専用のpath join/open/stat/read処理を作らない。
- Linux readerは絶対かつcleanなdirectoryを一つのFDとして開き、6回の`openat`をそのFDへ行う。directoryと
  fileのeffective-UID所有、mode/type、O_NOFOLLOW、size上限、regular file、read前後metadata同一性を
  全fileに同一適用する。file policyはlink countが一つであることも要求する。
- 読込みループ、件数確認、4 credential検証、CA生成の順に成功して初めてBundleを構築する。途中失敗では
  Bundleを生成せず、個別原因・filename・PEM・key・時刻・OS/parser detailを公開しない。
- non-Linux compatibility readerはfixed basename配列を共有して6 inputsへ追随するだけであり、本番保証を
  広げない。unsupported readerのfail-closed契約も維持する。

### CA委譲、保持、clock

- `proxyca.New(proxyca.Rules{...})` はLoadごとに一回だけ呼ぶ。certificate/keyのsingle-PEM、self-signed
  CA、P-256、key match、有効期間・leaf余命の意味検証はproxycaにのみある。
- `Rules.Clock`には、ロード時に時刻を凍結した値でなく、`proxyca.ClockFunc(func() time.Time { return nowUTC().UTC() })`
  相当のclosureを渡す。これによりLoad時のCA検証と、Load後の`Authority.Issue`は同じpackage-private clock seamの
  その時点のUTC値を使用する。
- Bundleはparsed Authorityと既存の解析済みcredentialだけを保持する。CA入力sliceはNew呼出し後にBundleへ
  格納せず、公開証明書はAuthorityの既存copy APIからのみ取得する。
- `ProxyCAAuthority()`は正常に構築されたAuthorityだけを返し、nil receiver、zero Bundle、欠けた又は不整合な
  内部状態ではnilを返す。返したAuthorityで利用可能な公開面は既存の`PublicCertificatePEM`と`Issue`に限定する。
  private key、raw PEM、signer、filename、serial化用の新公開面は作らない。
- `Bundle.Format`は新しいAuthorityを含めても固定labelのままとする。公開エラー、Format、accessorのテストは
  credential/CA秘密値、入力PEM、path、ホスト以外の下位detailを出さないことを確認する。

### 並行性とsnapshot

- Loadはstartup時の一回の6-file snapshotであり、reload/watch/rotationを導入しない。read中のmetadata/content
  変化はreaderのbefore/after検査で拒否する。
- 成功後のBundleは、JWT生成とAuthorityの公開CA copy/leaf発行を並行利用しても個別呼出しの状態を混線させない。
  public PEMは呼出しごとのcopyなので、callerが変更しても次回の取得・発行に影響しない。

## 代替案と不採用理由

- CA専用readerまたはCA用のdirectory再open: FD固定、TOCTOU、file policyを二重実装し、all-or-nothingを弱めるため不採用。
- brokercredentials内のPEM/certificate/key parser: `proxyca.New`の検証意味を再実装・乖離させるため不採用。
- PEM、private signer、file pathを返すBundle accessor又はBundle marshal: trusted brokerの必要最小権限と秘密境界を
  広げるため不採用。
- Load時の`nowUTC()`を値としてAuthorityへ渡す: Load後にIssueする際のclock seam更新を反映できないため不採用。
- CA生成、atomic replace、rotate/reload/watch、trust store/client設定、listener composition、VPS確認: startup read-only
  境界を越えるため不採用。

## 変更予定

| パス | 変更内容 |
|---|---|
| `tools/dev-agent-harness/internal/brokercredentials/brokercredentials.go` | 6固定basename、Loadからの一回のproxyca委譲、parsed Authorityの非公開保持、nil-safe `ProxyCAAuthority()`、clock closureを実装する。 |
| `tools/dev-agent-harness/internal/brokercredentials/reader_linux.go` | 共有basename走査のままCA二fileを保護し、Linux file validationでhardlink (`nlink != 1`) を拒否する。 |
| `tools/dev-agent-harness/internal/brokercredentials/reader_nonlinux.go` | 共有basenameにより6-file layoutへ追随する既存開発用互換readerのテスト期待を整合する。必要なproduction意味は追加しない。 |
| `tools/dev-agent-harness/internal/brokercredentials/brokercredentials_test.go` | 6-file valid fixture、all-or-nothing、CA validation/error folding、accessor/Format、clock/JWT、copy、concurrencyの回帰とnegative検出を追加する。 |
| `tools/dev-agent-harness/internal/brokercredentials/reader_fifo_test.go` | CA basenameを含むFIFO/special-node拒否が共有readerで維持されることを確認する。 |
| `tools/dev-agent-harness/README.md` | 6-file layout、同一秘密directoryからのstartup snapshot、trusted compositionに渡す公開CA copyとhost限定Authorityだけを記し、lifecycle/trust/live範囲外を保つ。 |

`proxyca` package、生成物、依存宣言、設定、service binary、Task/QA_PLAN/backlog/Wikiは変更しない。

## 実装手順

1. `brokercredentials`の固定basenameとfixtureを6-file layoutへ更新し、readerが既存の単一FD・固定順走査で
   6件を返すことを確認する。Linux validationへlink count条件を加える。
2. Loadの既存4 credential検証を保ったまま、完了後にCA bytesを一回だけ`proxyca.New`へ渡し、全失敗を
   `ErrLoad`へ畳む。成功後にのみparsed Authorityを含むBundleを作る。
3. non-leaking `ProxyCAAuthority()`と動的`nowUTC` closureを追加する。既存のJWT seamは変更せず、CA Issueも
   同じ可変clock値を見ることをテスト可能にする。
4. 正常・negative testsを拡張する。欠落/空、symlink/FIFO/device、owner/mode/size、hardlink、metadata race、
   malformed/multiple PEM、mismatch、non-P256、non-CA、expired/insufficient lifetimeを`ErrLoad`とno Bundleへ
   結び付ける。
5. nil/zero/corrupt Bundle、Format/error non-leak、public certificate copy、2 hostのfresh leaf、既存credentialと
   JWT、並行copy/Issue/JWTの隔離を確認する。race hookはCA fileも含む6-file読込みがpartial成功しないことを
   検出する。
6. READMEの境界を更新し、許可パス・差分行数を確認してcandidateを一回だけ固定する。標準candidate検証と
   独立REVIEW/QA、merge後の所定task-checkへ引き渡す。

## 検証計画

| 検証 | 目的・主なケース | 実施責任 / 時点 |
|---|---|---|
| package hermetic tests | 6-file正常load、既存ClientID/InstallationID/OpenAI/JWT、Authority公開copyと2 host Issueを確認する。 | DEV / candidate |
| package negative tests | filename欠落・空、size/mode/owner、symlink/FIFO/device/hardlink、before/after metadata変更、CA各種不正を固定`ErrLoad`とno partial Bundleへ結び付ける。 | DEV、独立QA / candidate |
| clock and boundary tests | Load後に変更した`nowUTC`がIssueに効くこと、nil/zero/corrupt accessors、Format/error non-leak、raw secret API不在、caller変更済みpublic copyの隔離を検出する。 | DEV、独立QA / candidate |
| concurrency tests | 同時JWT、public copy、leaf Issueで状態・keys・valuesが混線しないことと6-file failure atomicityを確認する。 | DEV、独立QA / candidate |
| focused race | Go race detectorで上記の並行利用とreader hookを限定再実行し、共有状態のraceを検出する。 | 独立QA / candidate、`focused-rerun` |
| repository checks | candidateでroot `make check`を一回、candidate-bound diff/check evidenceを独立REVIEW/QAが監査する。README lint、harness check/distcheckは既存make checkに含まれる範囲で記録する。 | DEV / candidate、REVIEW/QA / evidence-review |
| post-merge | 所定の`make task-check TASK=TASK-0053`をmainで実行する。 | Main / completion |

実UID provisioning、実secret、root、network、trust store、client TLS、generate/rotate/reload/watch、VPS live E2Eは
本Taskの受け入れ真実を再現しないため実施しない。これらを別モードのPASSで代替しない。

## 移行・互換性

- 既存4-file deploymentは新しい必須CA二fileがない限りloadできず、これは6-file all-or-nothing契約による
  意図したfail-closed移行である。互換fallbackは追加しない。
- `ClientID`、`InstallationID`、`OpenAIAPIKey`、`GitHubAppJWT`、`ErrLoad`、`ErrJWT`、Bundle Formatの既存公開意味を
  保つ。追加する公開面はAuthorityのread-only入口一つだけである。
- restart以外の読み直しや更新は行わない。provision/rotation/trustとlive verificationは後続Taskの明示的境界とする。

## 未解決事項

- なし。TASKの`Dependency-ready reconciliation`と完了経路preflightはreadyであり、追加の設計判断を要しない。

## main Agentレビュー

- [x] TASKの全AC-IDへ設計判断、パス、順序、失敗時の扱いを対応させ、条件本文を複製していない。
- [x] 単一FD reader、`proxyca.New`委譲、secret非公開、動的clock、Linux nlink、all-or-nothing、競合隔離の判断を具体化した。
- [x] QA_PLANがTASK-firstで独立作成されている。
- [x] `dependency-ready reconciliation`と完了経路preflightが完了している。
- [x] DEV開始を承認した（approved_dev_profile: `luna-xhigh`）。
