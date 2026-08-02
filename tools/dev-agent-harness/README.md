# Development Agentハーネス

Kakesuを開発するための外部開発基盤である。Kakesu本体のランタイム、Goモジュール、配布物には含めない。

## Push承認manifestの値境界

`internal/approvalmanifest`は、後続の承認ストア、Passkey許可確認、一回限りのpush許可が同じpush内容を
参照するためのpure Go値境界である。呼出し側が用意したリクエストID、Agent/workspace識別情報、正規
`owner/repo`と完全一致するGitHub HTTPSリモート、ポリシーバージョン/失効世代、UTC秒単位の作成/失効時刻、
および順序付き参照更新を`Build`へ渡すと、不変なmanifestを返す。値、時刻、ID、ポリシー、認可を生成又は
信頼判定せず、ファイルシステム、ネットワーク、Git子プロセス、永続状態を使用しない。

V1は1〜32件の`refs/heads/`安全部分集合だけを受理する。各更新は40桁小文字SHA-1 object IDとzeroセンチネル、
明示的な`force`/`delete`によって作成、通常更新、強制更新、削除のいずれかを表す。参照の入力順も承認内容へ
含まれ、重複、無変更、flag/センチネル不整合、tagその他の参照は拒否される。

manifestは固定フィールド順のcompact JSONである。`request_digest`以外の全フィールドを含むペイロードへ
`dev-agent-harness/push-approval-manifest/v1`のNUL終端ドメイン分離prefixを付け、実際にSHA-256を計算する。
公開ダイジェストは`sha256:`と64桁小文字hexであり、呼出し側から指定できない。`Parse`は32,768バイトを上限に、
重複/未知/欠落キー、型・意味不整合、ダイジェスト改変、フィールド順、空白、エスケープ、数値・時刻表記、
後続データを拒否し、パッケージ自身の正規バイト列だけを受理する。`Encoding`と`RefUpdates`は毎回コピーを返す。

妥当なmanifestは承認済み、許可済み、消費済み又はpush可能を意味しない。このパッケージは未加工Smart HTTP/pkt-line、
リモート旧SHA観測、force推定、Passkey、承認/許可状態、署名・発行・原子的消費、実認証情報、実push、
GitHub通信、監査を実装しない。これらは正規値へ明示的に束縛する後続境界と実環境確認の責務である。

`dev-agent-egress serve --config PATH` が唯一の外向き通信サービス起動面である。起動は設定、実行時識別情報、固定
`config_dir/credentials` 認証情報束、既存constructorの依存構成、ソケット受領、Serveの順に一回ずつ進み、どの失敗も固定診断へ
畳む。SIGINT/SIGTERMは協調的なキャンセル処理へ変換される。

`dev-agent-launcher`は次の一形式だけで一つのcoding-agentセッションを起動する。

```text
dev-agent-launcher run --repository owner/repo -- COMMAND [ARG...]
```

リポジトリは小文字の正規`owner/repo`、`COMMAND`は空でない直接実行`argv`でなければならない。`launcher`はシェル、ランタイム指定の
ソケット/helper/プロバイダー/モデル/プロキシ、追加optionを受理しない。`--help`、`-h`、`--version`以外の不完全または並べ替えた形式は、
制御ソケット又は子へ到達する前に`usage` エラーとなる。`COMMAND`の`argv`境界とstdin/stdout/stderrは直接子へ渡される。

セッションはbuild/install layoutから埋め込まれた外向き通信用Unix ソケットとGit 認証情報 helperだけを使う。公開プロキシ CA、対象
リポジトリのGitHub REST用Opaque handle、OpenAI Responses用Opaque handleをこの順で各一回取得し、literal `/tmp`配下のセッション専用
`0700` ディレクトリへ公開CAだけを`0600` `regular` ファイルとして置いて、IPv4 loopback `bridge`を一つ起動する。handleはディスク、`argv`、診断へ書かず、
子の`GH_TOKEN`と`OPENAI_API_KEY`だけへ渡す。Git Smart HTTP readは固定helperが操作ごとにsingle-use handleを取得する。

子 environmentは親environmentから`HOME`、`PATH`、`TERM`、`LANG`/正規`LC_*`、任意の`CODEX_HOME`だけを選び直す。
`HOME`/`CODEX_HOME`は子自身がCodex認証を探索できる場所として保持するだけで、`launcher`はその内容を読取り、複製、検証又は出力しない。
親のAPI 認証情報、プロキシ/CA、Git 認証情報/`config`、SSH、loader/ランタイム injection値は継承しない。セッション値としてupper/lower HTTP(S)
プロキシ、CA `trust`変数、非対話Git設定を一意に設定し、Git コマンド スコープでは既存認証情報 helper列を空値でresetしてから固定absolute helperだけを
追加し、GitHubのpath-aware 認証情報、プロキシ、CA、prompt無効化を固定する。システム/全体 Git `config`は読み込まず、ローカル設定は変更しない。

子終了、通常のnonzero exit、起動失敗、親キャンセル、予期しない`bridge`終了の全経路で、新規受理を止め、`active` `bridge`を`drain`してからCA
ディレクトリを一回削除し、発行済みAPI handleを各一回失効要求する。キャンセル又は`bridge` failureでは子を停止して待機してから戻る。正常終了は0、
通常の子 nonzeroはその状態を保ち、`signal`、setup/start/待機/ローカル クリーンアップ failureは固定failureへ畳む。失効要求がunknown又は期限切れで失敗しても
子の既存状態を秘密付き診断へ置き換えず、短いTTLを残余fail-safeとする。

この`launcher`とプロキシ environmentはネットワーク隔離の強制境界ではない。クライアントがプロキシを無視して直接通信することを防ぐdefault-deny ファイアウォール/
ネットワーク 名前空間、loopback隔離、Unix ソケットの所有権と`peer` UID、実Codex/Git/`gh`/OpenAI SDKのプロキシ/CA/helper受理、実認証情報、DNS/TLS、
プロバイダー受理、systemd/VPS配置・`restart`・ロールバック・クリーンアップは承認済み環境のlive E2E対象である。hermetic テスト又はinstall stagingの成功はこれらを
証明しない。

その他の補助実行ファイルは従来どおり`--version`と検査系の明示面だけを提供し、未実装の通常起動はfail-closedで拒否する。

## プロバイダー上流HTTPS 転送方式

`internal/upstreamtransport` は、ブローカーから `api.github.com`、`github.com`、`api.openai.com` へ出るための注入可能な
`http.RoundTripper` 境界である。各リクエストでDNSを一度だけ解決し、全answerを検査してから安全なIP literalの
`*:443`へ直接接続する。TLSは元のhostnameで証明書とSNIを検証し、TLS 1.2以上・HTTP/1.1だけを使用する。
プロキシ、リダイレクト、keep-alive、自動compression、接続後の再試行はこの境界にない。実プロバイダー、実Internet DNS、
システム trust ストア、プロキシ/ファイアウォールの受理はlive E2Eの対象であり、hermetic パッケージ テストでは保証しない。

## ブローカーの認証情報

`internal/brokercredentials` は、trusted ブローカーだけが読む秘密情報の境界である。配置ディレクトリには
`github-client-id`、`github-installation-id`、`github-private-key.pem`、`openai-api-key`、
`proxy-ca-cert.pem`、`proxy-ca-key.pem` の6 basenameだけを固定順で使い、実効UID所有、グループ/other権限なし、
ディレクトリのオーナー read/execute、ファイルのオーナー read（実行不可）を満たす必要がある。Linuxでは一度開いた
ディレクトリ descriptorから固定basenameを`openat`で読み、symlink、hardlink、通常ファイルでないnode、上限超過、
読込み前後のメタデータ変化を拒否する。6ファイルはstartup時に同じ秘密ディレクトリから一度だけスナップショットされ、全ての
検証に成功した場合だけtrusted compositionへ検証済みの`proxyca.Authority`と公開CA証明書のコピーを渡す。非Linux readerは開発テスト用
であり、本番サポートを意味しない。

正常に読み込まれたバンドルは、検証済みのクライアント ID、installation ID、OpenAI API キーと、GitHub App用の
短命RS256 JWT生成だけをtrusted ブローカーへ提供する。JWTは整数Unix秒の`iat=now-60`、`exp=now+540`、
`iss=クライアント ID`を使う。秘密ファイルの作成・rotate・書込み、環境変数やコマンド lineからの読込み、HTTP/
ネットワーク、Agentへの実認証情報公開、installation トークン交換はこの境界の対象外である。実配置のUID隔離と
実GitHub受理は後続live E2Eで確認する。

`dev-agent-harness-setup check-config --config PATH` は、サービス起動前に設定例をローカル検証する読み取り専用
コマンドである。バージョン1のstrict JSON、絶対かつcleanな配置パス、相異なるLinux user名、
`network.default=deny`、グループまたはその他ユーザーから書き込み可能ではないregular設定ファイル（64 KiB以下）だけを受理する。成功時は設定値を表示せず
固定要約を出力し、失敗時も入力値やパスを診断へ含めない。設定検証はユーザー作成、秘密の読込、ネットワーク、
IPC、サービス起動を行わない。

V1設定の`identity.workspace_id`は、1〜128バイトで先頭がASCII英数字、残りがASCII英数字または`._-`の
設定固定識別子である。`internal/runtimeidentity`はこのworkspaceと固定ユーザー名をconstructorで検証し、
起動単位ごとにLinuxのユーザー/グループ検索（各一回）と現在のnon-rootブローカーEUIDを突き合わせる。
成功時はインスタンスID（`agent-` + 16バイトの暗号乱数をlowercase hex化）、ブローカーUID、エージェントUID/GID、
`brokerlistener.Subject`を同じ解決結果から返す。非Linux、NSS/UID/GID不一致、検索または乱数失敗は
常に固定エラーへ畳み、username・workspace・numeric IDや下位エラーを診断へ出さない。ソケット有効化とPeerBinderの
組み合わせ、実NSS・別UID/GID・サービス再起動・VPS確認はこの境界の対象外であり、隔離テストとLinux cross-compileは
それらのlive E2Eを意味しない。

`dev-agent-harness-setup plan-provision --config PATH --target-root PATH` は、対象OSへ渡す
配置計画の望ましい状態を確認するための読み取り専用dry-runである。成功時はヘッダー1行と、固定順序
（ユーザー3件、ディレクトリ5件、サービス3件）の計11 actionを正規JSONLとしてstdoutへ出力する。設定ディレクトリの
固定`credentials`子ディレクトリはブローカー所有`0700`としてサービス記録の直前に置かれる。
ユーザーは`/nonexistent`・`/usr/sbin/nologin`・locked・ホーム非作成、ディレクトリは`0750`と所有者/グループ、
サービスはブローカーuserかつ`enabled=false`・`started=false`として表現される。論理パスはcleanな
`target-root`配下の表示用対象パスへ写像される。

このコマンドはtarget-rootをstatせず、user/グループ/ディレクトリを作成・変更せず、systemd、プロセス、ネットワーク、
IPCも起動しない。executor、sysusers、tmpfiles、unitの実行は後続の別境界であり、このmanifest出力からは
暗黙に許可されない。全記録をメモリ上で検証・serializeしてからstdoutへ一度だけ書き込むため、
writerがpartialバイトを返してエラーになった場合の巻戻しはできず、再試行や再出力は行わない。

`dev-agent-harness-setup verify-provision` は、config、manifest、指定ルートを受け取り、
OSへ適用する前のmanifestを読み取り専用で検証する。manifestは一度だけ開いたファイルディスクリプタから読み、
末尾symlinkでないregularファイル、グループまたはその他ユーザーから書き込み可能でないモード、128 KiB以下、
読取前後で変化しないサイズ・モードおよびバイト数を満たす必要がある。入力を解釈する独立parserは持たず、同じ設定と
指定ルートから`provision.Build`で生成した正規バイト列との完全一致だけを受理する。成功時の出力は
`provision manifest version=1 actions=11 verified`という固定要約だけであり、失敗時も入力パス、設定値、
manifest本文、ユーザー名、OSエラー本文を診断へ含めない。設定、manifest、指定ルートその他ホスト状態は
変更せず、OS適用処理、管理者権限、systemd、プロセス、ネットワーク、IPCはこの検証境界の対象外である。

## プロバイダー 認証情報の解決

`internal/providercredentials` は、検証済み ブローカー バンドル と注入された trusted
`http.RoundTripper` の間だけで認証情報を解決する。OpenAI は バンドル 内の API キー を
ネットワーク なしで返し、GitHub は要求された 正規 `owner/name` の リポジトリ 名一件だけを
短命 installation トークン へ呼出しごとに単発交換する。リクエスト は固定 HTTPS endpoint、リポジトリ
スコープ、タイムアウト、API バージョン に束縛され、レスポンス は上限付き JSON 境界で検証される。

この パッケージ は default 転送方式、TLS/DNS/プロキシ、リダイレクト/再試行、cache、永続化、ログ、実 GitHub
通信を実装しない。解決した成功値は trusted `egresstransaction` から Forwarder へ渡すためだけに
使い、Agent や診断へ 認証情報・JWT・リクエスト/レスポンスの詳細を公開しない。実 転送方式 と
GitHub installation の受理は後続の live E2E 境界で確認する。

## プロバイダー上流レスポンス転送

`internal/upstreamforwarder` は、既存の `egresstransaction.PreparedRequest` を同じ
`egresspolicy.Policy` で再評価してから、注入済み `http.RoundTripper` を同期的に一度だけ
呼び出す信頼済みブローカー境界である。REST/OpenAIは実`Bearer`認証とJSONヘッダーを維持する。
Git Smart HTTP readだけはブローカー内で組み立てた`x-access-token:<real token>`のBasic認証と、
discovery/upload-packに対応する厳密なGitメディア型を送る。リダイレクト、再試行、既定転送方式、環境プロキシは選択しない。

2xxレスポンスは上限付きで全量を読み、本文を閉じてから、REST/OpenAIはUTF-8の有効なJSONとJSONメディア型を、
Git readは200と操作対応メディア型を検査する。Git本文はバイト列のまま扱う。成功時だけHTTP状態コード、メディア型、独立コピーした本文をリクエスト単位の
sinkへ一度渡す。上流ヘッダー、プロバイダーエラー本文、URL、スコープ、認証情報、下位エラーは
sinkと公開エラーに出さない。実プロバイダー、実認証情報、DNS/TLS、Agent側レスポンスwriterへの接続は
このhermetic境界の対象外であり、後続のlive E2Eで確認する。

## ブローカー内メモリ上の外向き通信交換

`internal/brokerexchange` は、呼出しごとに非共有のレスポンス受け口、既存の
`upstreamforwarder`、`egresstransaction` を同期的に一度だけ合成する。成功時だけ、
上流から縮退されたHTTP状態、正規化されたJSON内容型、独立コピーの本文を返し、
拒否・依存失敗・受け口通知不整合は `exchange-denied` と空のレスポンスへ畳む。ケイパビリティの
消費、認証情報のBearer置換、ポリシー/レジストリの評価は既存境界へ委譲し、ロールバックや再試行は
行わない。

このパッケージはAgent向けHTTP入口、待受け、peerからの呼出元識別、本番認証情報の解決・転送方式の
組立て、既定転送方式、実GitHub/OpenAI、DNS/TLS、外部ネットワークを実装しない。実プロバイダー
受理と実配置は後続のlive E2E境界で確認する。

## TLS終端後のHTTP入口

`internal/brokerhttp` は、TLS終端済みの `net/http` リクエストから HTTP/1.1 の
origin-form と既知の本文長形式だけを受け取り、コンテキストだけから解決した呼出元と独立コピーした
メソッド、Host/パス、discoveryに限る正規`service=git-upload-pack`クエリ、Content-Type、Authorization、上限内本文を一度だけ既存の
`brokerexchange` へ渡す。成功時は2xxの縮退済みレスポンスと固定のno-store/nosniff ヘッダーだけを
返し、入力不整合、呼出元解決失敗、交換拒否は空の403へ畳む。

この入口はTCP待受け、TLS、HTTP/2、absolute-form、chunked、実配置の呼出元解決、実プロバイダー、
外部ネットワーク、再試行、診断本文を実装しない。hermeticなテスト成功は実環境の受理を意味せず、
実配置確認は後続のlive E2E境界で行う。

## ブローカー内プロキシCA

`internal/proxyca` は、メモリで受け取った単一のECDSA P-256 CAと対応鍵を検証し、公開CA証明書の
コピーだけを返す。`api.github.com`、`github.com`、`api.openai.com` の完全一致名に限り、呼出しごとに新しい
P-256 leaf鍵と短命のサーバー証明書を発行する。CA秘密鍵、入力PEM、leaf秘密鍵は公開値や診断へ
出さず、証明書・鍵・シリアル番号を呼出し間で共有しない。

この境界は証明書ファイルの読書き、鍵の配置・更新、接続受付、CONNECT、SNI振分け、OS証明書ストア、
実クライアントや外部ネットワークを実装しない。メモリ内TLS検証の成功は実配置の受理を意味せず、
実環境確認は後続のlive E2E境界で行う。

## Agent側CONNECTセッション

`internal/connectsession` は、受理済みの一つの接続だけを所有し、厳格な三つのホスト向けCONNECTまたは
同じソケット上の一操作制御を明確に分岐する。CONNECTでは発行済み証明書でTLS 1.2以上・SNI一致・
ALPN `http/1.1`を終端し、呼び出し元コンテキストを引き継いだHTTP/1.1リクエスト一件だけを注入済み
`brokerhttp.Handler`へ同期的に渡す。制御では`POST /v1/capabilities`によるGitHub/OpenAIケイパビリティ発行と、
`DELETE /v1/capabilities/cap_...`による失効、および完全一致する`GET /v1/proxy-ca`による公開CA取得だけを扱い、
TLS・リーフ発行・内側のHTTP処理へ進まない。

制御リクエストは16,384バイト以下のリクエスト行とヘッダー、正規`Content-Length`、発行時だけ1〜512バイトの厳格な
JSONと`Content-Type: application/json`を要求する。chunked、upgrade、keep-alive、追加ヘッダー、earlyバイト、
複数リクエストを拒否し、成功・失敗とも一操作後にcloseする。公開CA取得は正規な本文なしGETだけを受理し、
ECDSA P-256の自己署名CA、基本制約、CertSign用途、現在の有効期間を再検査した単一の証明書だけのPEMについて、
新しいコピーだけを返す。発行成功時は不透明handleだけ、失効時は空本文を返し、
拒否は入力値やhandle、URL、allowlist、主体、認証情報、下位エラーを含まない空本文403へ畳む。

CONNECTヘッダーの上限と各フェーズは5秒（呼び出し元の期限またはキャンセルが早ければそちら）に固定し、
TLS後の失敗は追加HTTP応答なしでcloseする。HTTP/2、汎用CONNECT、再試行、接続受付、呼び出し元識別情報生成、
CAファイルの信頼設定、認証情報helperはこのサーバー境界に含めない。

`net.Pipe`とメモリ内CA、fake依存でのテスト成功は、実接続受付、OS識別情報、実クライアントの証明書信頼、外部ネットワーク、
GitHub/OpenAIのlive E2Eを保証しない。これらの配置・認証・環境依存の確認は後続のlive E2E境界で行う。

## Agent接続の受理と主体コンテキスト

`internal/brokerlistener` は、外側で用意された `net.Listener` を所有し、設定した上限のスロットを
`Accept` より先に取得してから、注入された trusted `PeerBinder` と一接続 `Session` を同期的に合成する。
Binder が返す `egresstransaction.Subject` は UID とケイパビリティ互換の識別子を検証・コピーした後、公開されていない
コンテキストのキーへ束縛されるため、下流の `Resolver` はコンテキストからそのコピーだけを解決する。拒否、依存エラー、panic は
接続単位で閉じ、予期しない `Accept` エラーまたは呼出元のキャンセルは待受けを閉じ、協調的な処理を drain して終了する。

この境界は待受けの生成・待受け開始、Linux peer 認証情報（`SO_PEERCRED`）、実 UID/名前空間、systemd、TLS/HTTP、
実クライアント、外部ネットワーク、live VPS を実装または検証しない。インメモリの待受け、`net.Pipe`、協調的な注入依存での
hermetic テストは、実環境の到達制御や OS 識別情報を保証しない。

`internal/peerbinder` は、待受けごとに静的に固定した一つの Subject と、Linuxカーネル認証の接続元UIDを照合する。
接続内容、アドレス、PID/GID、自己申告値から Subject を生成・補完せず、非Linuxではフェイルクローズとする。
この境界のテストは実 UID 分離、ソケットの所有者/モード、名前空間、待受けの生成・合成、
systemd、または VPS の live 配置を検証しない。

## Agent向けケイパビリティ制御

`internal/capabilitycontrol` は、`brokerlistener.Resolver`が接続コンテキストから解決したAgentインスタンス、UID 0以外、
workspaceだけを主体として、本番トランザクションと同じメモリー内`capability.Registry`へ短命なhandleを
発行する。リクエストから主体、TTL、使用回数、ホストを受け取らない。GitHubは設定allowlist内の正規
`owner/repo`一件に固定し、既存のselector省略はREST read、明示`operation=github-git-read`だけは`github.com`のGit readを発行する。
GitHub REST readとOpenAI ResponsesテキストはTTL 5分・16回使用、Git Smart HTTP readだけはTTL 5分・一回使用である。
OpenAIは設定のモデルallowlistが非空の場合だけプロバイダースコープを発行する。個別モデルは制御やケイパビリティスコープへ
複製せず、既存の外向き通信ポリシーが実リクエスト本文から検査する。

16回の予算は同じ主体、workspace、プロバイダー、リポジトリ、操作、宛先ホストに束縛されたAPIリクエストだけを許可する。
一致した消費だけが`capability.Registry`内で原子的に残数を一つ減らし、16回目の成功後は拒否する。不一致は残数を減らさず、
API handleをGit read、push/書き込み、upload、別プロバイダー、別リポジトリ、別ホストへ転用できない。失効、期限切れ、
失効世代の既存の拒否境界も変えない。

失効は同じ接続元由来の主体のインスタンス/UID/workspaceが完全一致する正規handle一件だけを削除する。
unknown、malformed、期限切れ、別主体は固定拒否となり、handle以外の秘密や許可内部値をAgentへ返さない。
別レジストリ、永続化、cache、ロールバック、再試行、認証情報コピーは持たない。認証情報helper、起動機構、環境変数注入は
別コンポーネント又は後続Taskの境界である。

hermeticテストはこのメモリー内ライフサイクルと既存CONNECTの回帰を確認するが、実DNS/TLS、実GitHub/OpenAI、実NSS/別UID、
systemdソケット、VPS配置の受理を保証しない。

## Agent向けAPIケイパビリティクライアント

`internal/controlclient`は既存の固定Unix制御ソケットへ一接続一操作で発行要求を送る。GitHub REST read用の明示関数は
絶対かつ正規化済みのソケットパスと正規なallowlistリポジトリだけを受け、プロバイダーとリクエストJSONを固定する。
OpenAI Responsesテキスト用の明示関数はソケットパスだけを受け、プロバイダーを固定する。どちらも操作、モデル、HTTPパス又は
任意の本文をAgent入力として受けない。Git Smart HTTP用の既存`Issue`は明示`github-git-read`を含む従来の通信形式を保ち、
一回使用のhandleを操作ごとに取得する。

三つの発行操作は共有する非公開交換処理を使い、一回だけ接続し、書き込みと読み込みへ短い期限を設定して、一つのリクエストと
一つのレスポンスの後に接続を閉じる。固定順序の`200 application/json`ヘッダー、正規なContent-Length、唯一の正規JSON handle、
本文直後のEOFだけを成功とする。HTTP状態、ヘッダー、長さ、JSON、handle、EOF、接続、期限、読み込み、書き込み又は切断の不一致は、
再試行や代替をせず空値と固定エラーへ畳む。エラーはソケット、リポジトリ、handle、通信内容又は下位エラーを保持しない。

後続の起動機構は取得した不透明handleを`GH_TOKEN`又は`OPENAI_API_KEY`へ渡せるが、このクライアントは環境変数、子プロセス、
CA信頼ファイル、Git設定、loopback bridge、認証情報置換、リポジトリ/モデルallowlist又はHTTPポリシーを構成しない。
GitHub書き込み/GraphQL、OpenAI admin/files/upload、Git push、承認、cache、更新、永続化、別ソケット、TCPも追加しない。
hermeticな偽接続で確認できるのは通信形式、応答構造、接続寿命と固定エラーの境界までであり、実認証情報、実GitHub/OpenAI、
DNS/TLS、実ソケット権限、別UID、systemd又はVPS配置は、承認済み環境で別途live E2E確認する。

## Agent向け公開プロキシCAクライアント

`internal/controlclient.ProxyCA`は、絶対かつ正規化済みの既存の外向き通信用Unixソケットへ一回だけ接続し、上記の固定GETを
一回送る。固定順序の200応答ヘッダー、1〜4,096バイトの正規な単一`CERTIFICATE` PEM、宣言長との一致、
本文直後のEOFを上限付きで読み、サーバーとは別のX.509検査でP-256自己署名CA、基本制約、CertSign用途、
現在の有効期間を確認する。成功値は呼出元所有の公開証明書のコピーであり、CA秘密鍵、認証情報ディレクトリ、
主体、リポジトリ、不透明参照を取得または返さない。失敗はソケット、パス、PEM、主体、下位エラーを保持しない固定エラーとなる。

このクライアントは再試行、キャッシュ、TCPや別ソケットへの代替接続、証明書チェーン、CA更新・再読込み、信頼ストア登録、
一時ファイル、環境変数、Git設定、起動機構、プロセスのライフサイクルを実装しない。後続の起動機構が必要な期間だけ信頼ファイルを
作る直前の取得境界に限られ、本コンポーネントはディスクへ書かない。`net.Pipe`による成功は実OS権限、別UID、実Git/libcurlの
信頼、GitHub/OpenAI/DNS/TLS、systemd/VPS配置を保証せず、それらは承認済み環境で別途live E2E確認する。

## Agent用loopbackプロキシ中継

`internal/proxybridge`は、HTTPプロキシの接続先だけを受け取れるAgentクライアントと、既存の接続元に束縛された外向き通信用
Unixソケットの間を接続する前段である。待受けは固定のIPv4 loopbackで`tcp4`を使い、OS割り当ての一時ポートだけを
使用する。呼出元へは正規なloopback HTTP endpointだけを返す。各クライアント接続は構築時に固定した絶対かつ
正規化済みのUnixソケットへ、期限付きで一回だけ接続する。接続上限の枠は受理前に取得し、再試行、別ソケット、
TCP上流、代替経路を選ばない。

Unixソケットへの接続成功後はHTTP、CONNECT、TLS、認証情報を解釈・変更せず、バイトを双方向へそのまま
ストリーム転送する。一方向のEOFは反対側の書き込み側half-closeへ伝える。キャンセル、接続失敗、待受け失敗では
両端を閉じ、全接続処理の完了を待つ。主体の割り当て、CONNECT/制御の認可、TLS/inner HTTPポリシーは引き続き
Unixソケット側の外向き通信サービスだけが所有し、loopback中継は認可プロキシや診断境界にならない。

このパッケージはAgentネットワーク名前空間の隔離、Unixソケットの権限と接続元UID、CA信頼用ファイル、Git設定、
子プロセスやシグナルのライフサイクル、実Git/`gh`/OpenAIクライアントのプロキシ対応、systemd/VPS配置を構成または
証明しない。偽の待受けと接続器、および`net.Pipe`によるhermeticテストも、これらのlive環境条件のPASSを意味しない。

## Git read用認証情報helper

`git-credential-dev-agent`は、allowlistにある正規な`github.com`リポジトリをHTTPSで読むための、
手動設定用Git認証情報helperである。一致する`get`では、既存の外向き通信制御サービスへ一回限りの
不透明ケイパビリティを要求し、固定ユーザー名`x-access-token`とhandleだけを返す。
プロバイダーの実トークンは受け取らず、Gitへも返さない。入力又は制御の失敗時は`quit=true`だけを返し、
別helper又は対話promptへの認証情報探索を停止する。

helperは完全一致するHTTPSホストと正規な`owner/repo.git`パスだけを受理する。`store`は上限付き入力を
読み捨てて保存せず、`erase`は正規な不透明handle一件だけを失効する。未知の操作はhelperプロトコルに従い、
出力なしで無視する。

制御用Unixソケットは、configure済み`runstatedir`からリンク時に固定する。コマンド引数、認証情報フィールド、
環境変数、設定又は作業ディレクトリでは置換できない。このコンポーネントはGit設定を変更せず、
clone/fetch/pull、push、認証情報のcache又は永続化、起動機構の設定を行わない。配置済みソケット、GitHub、
DNS、TLS又はsystemdの実環境受け入れ条件もこの境界では確認しない。

## Build

リリースtarballには生成済み`configure`を含める。

```sh
./configure \
  --prefix=/usr/local \
  --sysconfdir=/etc \
  --localstatedir=/var \
  --runstatedir=/run \
  --with-systemd-unit-dir=/etc/systemd/system
make
make check
```

Git checkoutで`configure.ac`を変更した場合だけ、`Autoconf`で`configure`を再生成する。

```sh
autoconf
```

## Install

```sh
sudo make install
```

パッケージのステージングには`DESTDIR`を使える。

```sh
make install DESTDIR="$PWD/package-root"
```

`make install`はファイルを配置するだけである。OSユーザー、秘密、実設定、tailnet、外部サービス、サービス状態は
変更しない。実装後も配置計画とenable/startは明示的な管理操作として分離する。

## systemdによる外向き通信用ソケット

`dev-agent-egress.socket`が`/run/dev-agent-harness/egress.sock`を作成し、`dev-agent-broker:dev-agent`の
`0660`ソケット、停止時の後始末、FD名`egress`を所有する。tmpfilesとprovisionの配置表は実行用ディレクトリだけを
`dev-agent-broker:dev-agent 0710`へ整え、他の状態・設定・監査用ディレクトリの所有権とアクセス権を変更しない。インストールは
unitの配置だけで、enable/start/restartは行わない。

`internal/socketactivation`はLinuxブローカーの現在EUID、固定位置、Unix待受け、ディレクトリとソケットの所有者・所属グループ・
アクセス権を検証し、systemdから渡されたFD 3を一回だけ受け取る。プロセス自身の待受け作成、chmod/chown、古いソケットの削除、
代替待受けはない。非Linuxはfail-closedであり、隔離試験またはLinuxのコンパイル確認の成功は実systemd、別UID/GIDによる
権限制御、実ソケット接続を意味しない。それらのlive E2Eは承認済みLinux環境で別途確認する。

## 外向き通信 ポリシー コア

`internal/egresspolicy` は、TLS終端後の後続コンポーネントが利用する副作用のないGo製
allowlist判断コアである。`New(Rules)` が許可するGitHubリポジトリとOpenAIモデル、
JSON本文上限、出力トークン上限を検証・コピーし、`Policy.Authorize(Request)` は入力値だけを
調べて固定された判断を返す。許可面は次の三つだけで、未知の操作面は拒否となる。

- `GET`/`HEAD https://api.github.com[:443]/repos/{owner}/{repo}` とその正規な子パス
- `GET https://github.com[:443]/{owner}/{repo}.git/info/refs?service=git-upload-pack` と
  `POST https://github.com[:443]/{owner}/{repo}.git/git-upload-pack`（厳密なリクエストメディア型・非空で上限付きのバイト列本文）
- `POST https://api.openai.com[:443]/v1/responses` の追加パラメーターを含まない`application/json`、
  `store=false`・`stream=false`の上限付きテキスト専用本文

URLのパーセント符号化、ユーザー情報、クエリ、フラグメント、曖昧なパス要素、非正規なホスト/
ポートは許可根拠にならない。OpenAI本文も厳密なJSONオブジェクトとして検査され、重複/未知フィールド、
ストリーミング・ツール・ファイル/画像等の操作面は拒否される。`Authorize`の許可は実際の通信を
行わず、許可判断はネットワーク接続を意味しない。

このパッケージはHTTP サーバー/プロキシ、リダイレクト、TLS、DNS、認証情報取得・置換、Authorization
ヘッダー、ログ記録/監査、利用量/費用計測を実装しない。接続、認証情報境界、監査、失敗時の
クリーンアップは後続プロキシ/ブローカーの責務であり、利用側で別途明示的に適用する。

## Opaque ケイパビリティ レジストリ

`internal/capability` は、Agentへ渡す `cap_...` を実認証情報ではなく、in-memory レジストリへの
短命な参照として発行する。`New(Rules)` でポリシーバージョン、TTL、使用回数、失効世代の
上限を固定し、`Issue(IssueSpec)` は次の三つのスコープだけを生成する。

- GitHub: リポジトリ必須、`github-rest-read`、`api.github.com`
- GitHub Git: リポジトリと明示操作必須、`github-git-read`、`github.com`
- OpenAI: リポジトリなし、`openai-responses-text`、`api.openai.com`

handleは32バイトのcrypto/rand値をpaddingなしbase64urlで符号化したものだが、レジストリが保持する
map キーはSHA-256 ダイジェストだけである。`Consume(Request)` はAgent インスタンス、non-root UID、workspace、
プロバイダー、リポジトリ、操作、宛先 ホストを完全一致させ、期限、失効世代、残使用回数の判定と
一回の消費をmutex下で原子的に行う。スコープ不一致は使用回数を消費せず、最後の使用、期限切れ、
`Revoke`、接続元由来の主体完全一致を要求する`RevokeForSubject`、失効世代の更新後は再利用できない。

productionではmonotonic elapsedを内部TTLに使い、呼出元へ返すIssued/Expires時刻だけUTCへ変換する。

このレジストリは永続化せず、プロセス再起動時に全entryが失われるfail-safe設計である。実認証情報の
取得・保存・置換、プロキシ、HTTP/TLS/DNS、監査、呼出頻度/費用の制限、config/CLI、複数プロセス共有は
対象外で、成功結果は実通信や認証情報を意味しない。

## 外向き通信 トランザクション

`internal/egresstransaction` は、上記二つの境界を接続する in-memory
トランザクション である。`Policy.Evaluate` は allowlist の同じ評価から
正規 プロバイダー スコープ を返し、`Transaction.Execute` はその スコープ に
厳密な `Authorization` ケイパビリティ 値を束縛して `Registry.Consume` を一度
だけ実行する。消費成功後だけ注入済み resolver を呼び、visible ASCII の
認証情報をREST/OpenAIでは上流用`Bearer`へ、Git readではAgentが提示した厳密な
`Basic base64(x-access-token:cap_...)`を消費後に`Basic base64(x-access-token:<real token>)`へ置換してtrustedなForwarderへ同期的に
一度だけ渡す。

Git readはupload-pack discoveryとPOSTだけを許可し、receive-pack/push、認証情報ヘルパー、Git config/launcher、
実`clone/fetch/pull`は含まない。hermeticテストは実GitHub認証情報、実DNS/TLS、別UID/NSS、systemd/VPS配置を保証しない。

トランザクション は prepared リクエスト や 認証情報 を返さず、ケイパビリティ handle
や caller 所有の リクエスト slice も保持しない。resolver/Forwarder の失敗は
固定で情報を漏らさない 拒否 とし、ロールバック/再試行 は行わない。認証情報
ストア、ファイル、environment、プロセス、HTTP 転送方式、ネットワーク、DNS、TLS は
この パッケージ の対象外である。

## 承認リクエスト状態

`internal/approvalstate`は、ブローカーオーナーが用意した`0700`の既存ディレクトリで
正規承認manifestを永続管理する単一書込み主体のストアである。状態は`pending`、`approved`、`denied`、
`cancelled`、`expired`、`stale`の6種だけで、期限到達は判断より優先される。manifestは保存前と
再起動時に`approvalmanifest.Parse`で再検証され、ポリシーバージョン、失効世代、TTL、
リクエストID、ダイジェストに束縛される。

Linux/Darwinではプロセス間のnon-blocking exclusiveロックを保持する。スナップショットは
リクエストID順の正規JSONとし、`0600`一時ファイルへの全書込み、ファイル同期、同一名への原子的置換、
ディレクトリ同期が成功した後だけメモリー状態を更新する。置換結果が不確実な失敗、または置換後の
ディレクトリ同期failureはストアをpoisonし、`Close`、再`Open`、上位照合まで成功を推測しない。
getterはmanifestと記録一覧をコピーし、固定エラーclassはパス、ID、ダイジェスト、manifest、
下位OSエラーを公開しない。

`Approve`と`Deny`に渡す判断者IDは上位層で本人確認済みであることが前提であり、このパッケージは
Passkey/WebAuthnを検証しない。`approved`は永続判断にすぎず、許可、push権限、実push、
消費完了のいずれも意味しない。実配置の作成/chown、OS/プロセス境界、systemd restart/ロールバック、
通知、スマートフォン承認、許可発行とpush照合は後続Taskで接続し、承認済み環境で確認する。

## Passkey許可確認のライフサイクル

`internal/approvalchallenge`は、承認リクエストのIDと正規manifestダイジェスト、`approve`/`deny`判断、
operator ID、RP ID、完全一致するHTTPS origin、期限を、プロセス内の不透明なランダム`challenge`へ束縛する。
最初の`Consume`試行はtrusted verifierを呼ぶ前に`challenge`を予約し、成功、検証失敗、`panic`、
同時試行、`Close`のいずれでも再利用しない。期限到達は消費より優先され、時刻の巻き戻りは
フェイルクローズとなる。呼出し関数へ渡す割り当てと主張はコピーし、成功結果は未加工の認証情報IDを
domain-separated hashへ変換して`challenge`、主張、署名、認証情報publicキーを残さない。

マネージャーは永続化、timer、background purgeを持たず、`Close`又はプロセス再起動で`pending`の
`challenge`を全て失う。失敗、期限切れ、再起動後の回復は、承認リクエストがまだ`pending`であることを
別境界で確認して新しい`challenge`を発行することであり、旧`token`の再試行ではない。この呼出し関数の
接続点自体は実WebAuthnの署名、origin/RP ID hash、UV、counter又は認証情報状態を検証せず、
Tailscale識別情報とのAND条件、検証済み判断API、`approvalstate`の変異、許可又はpush認可も実装しない。
実認証器、HTTPS/Tailscale/スマートフォンを使う環境依存確認は、後続Taskの承認済み環境とクリーンアップ手順を要する。
