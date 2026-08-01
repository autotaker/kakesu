# Development Agentハーネス

Kakesuを開発するための外部開発基盤である。Kakesu本体のランタイム、Goモジュール、配布物には含めない。

現在はbuild/install境界だけを固定したスキャフォールドである。各バイナリは`--version`だけ成功し、通常起動は
未実装としてfail-closedする。

## プロバイダー上流HTTPS 転送方式

`internal/upstreamtransport` は、ブローカーから `api.github.com` と `api.openai.com` へ出るための注入可能な
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

`dev-agent-harness-setup plan-provision --config PATH --target-root PATH` は、対象OSへ渡す
配置計画の望ましい状態を確認するための読み取り専用dry-runである。成功時はヘッダー1行と、固定順序
（ユーザー3件、ディレクトリ4件、サービス3件）の計10 actionを正規JSONLとしてstdoutへ出力する。
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
`provision manifest version=1 actions=10 verified`という固定要約だけであり、失敗時も入力パス、設定値、
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
呼び出す信頼済みブローカー境界である。送信時のヘッダーは実 `Bearer` 認証、固定の
`Accept: application/json` と `User-Agent`、OpenAI の場合だけ `Content-Type: application/json`
に限定する。リダイレクト、再試行、既定転送方式、環境プロキシは選択しない。

2xxレスポンスは上限付きで全量を読み、本文を閉じてからUTF-8の有効なJSONとJSONメディア型を
確認する。成功時だけHTTP状態コード、正規化したメディア型、独立コピーした本文をリクエスト単位の
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
メソッド、Host/パス、Content-Type、Authorization、上限内本文を一度だけ既存の
`brokerexchange` へ渡す。成功時は2xxの縮退済みレスポンスと固定のno-store/nosniff ヘッダーだけを
返し、入力不整合、呼出元解決失敗、交換拒否は空の403へ畳む。

この入口はTCP待受け、TLS、HTTP/2、absolute-form、chunked、実配置の呼出元解決、実プロバイダー、
外部ネットワーク、再試行、診断本文を実装しない。hermeticなテスト成功は実環境の受理を意味せず、
実配置確認は後続のlive E2E境界で行う。

## ブローカー内プロキシCA

`internal/proxyca` は、メモリで受け取った単一のECDSA P-256 CAと対応鍵を検証し、公開CA証明書の
コピーだけを返す。`api.github.com` と `api.openai.com` の完全一致名に限り、呼出しごとに新しい
P-256 leaf鍵と短命のサーバー証明書を発行する。CA秘密鍵、入力PEM、leaf秘密鍵は公開値や診断へ
出さず、証明書・鍵・シリアル番号を呼出し間で共有しない。

この境界は証明書ファイルの読書き、鍵の配置・更新、接続受付、CONNECT、SNI振分け、OS証明書ストア、
実クライアントや外部ネットワークを実装しない。メモリ内TLS検証の成功は実配置の受理を意味せず、
実環境確認は後続のlive E2E境界で行う。

## Agent側CONNECTセッション

`internal/connectsession` は、受理済みの一つの接続だけを所有し、strictな二つのホスト向けCONNECTを一度だけ
検査してから、発行済み証明書でTLS 1.2以上・SNI一致・ALPN `http/1.1`を終端する。その後は呼び出し元コンテキストを
そのまま引き継いだHTTP/1.1リクエスト一件だけを注入済み`brokerhttp.Handler`へ同期的に渡し、レスポンス完了後に
接続を閉じる。CONNECTヘッダーの上限と各フェーズは5秒（呼び出し元の期限またはキャンセルが早ければそちら）に固定し、
拒否は空本文の403、TLS後の失敗は追加HTTP応答なしでcloseする。keep-alive、HTTP/2、upgrade、汎用CONNECT、
再試行、接続受付、呼び出し元識別情報生成、CAファイルの信頼設定はこの境界に含めない。

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

## 外向き通信 ポリシー コア

`internal/egresspolicy` は、TLS終端後の後続コンポーネントが利用する副作用のないGo製
allowlist判断コアである。`New(Rules)` が許可するGitHubリポジトリとOpenAIモデル、
JSON本文上限、出力トークン上限を検証・コピーし、`Policy.Authorize(Request)` は入力値だけを
調べて固定された判断を返す。許可面は次の二つだけで、未知の操作面は拒否となる。

- `GET`/`HEAD https://api.github.com[:443]/repos/{owner}/{repo}` とその正規な子パス
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
上限を固定し、`Issue(IssueSpec)` は次の二つのプロバイダー スコープだけを生成する。

- GitHub: リポジトリ必須、`github-rest-read`、`api.github.com`
- OpenAI: リポジトリなし、`openai-responses-text`、`api.openai.com`

handleは32バイトのcrypto/rand値をpaddingなしbase64urlで符号化したものだが、レジストリが保持する
map キーはSHA-256 ダイジェストだけである。`Consume(Request)` はAgent インスタンス、non-root UID、workspace、
プロバイダー、リポジトリ、操作、宛先 ホストを完全一致させ、期限、失効世代、残使用回数の判定と
一回の消費をmutex下で原子的に行う。スコープ不一致は使用回数を消費せず、最後の使用、期限切れ、
`Revoke`、失効世代の更新後は再利用できない。

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
認証情報 を上流用 `Bearer` 値へ変換して trusted な Forwarder へ同期的に
一度だけ渡す。

トランザクション は prepared リクエスト や 認証情報 を返さず、ケイパビリティ handle
や caller 所有の リクエスト slice も保持しない。resolver/Forwarder の失敗は
固定で情報を漏らさない 拒否 とし、ロールバック/再試行 は行わない。認証情報
ストア、ファイル、environment、プロセス、HTTP 転送方式、ネットワーク、DNS、TLS は
この パッケージ の対象外である。
