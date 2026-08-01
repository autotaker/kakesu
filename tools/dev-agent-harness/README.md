# Development Agentハーネス

Kakesuを開発するための外部開発基盤である。Kakesu本体のランタイム、Goモジュール、配布物には含めない。

現在はbuild/install境界だけを固定したスキャフォールドである。各バイナリは`--version`だけ成功し、通常起動は
未実装としてfail-closedする。

## ブローカーの認証情報

`internal/brokercredentials` は、trusted ブローカーだけが読む秘密情報の境界である。配置ディレクトリには
`github-client-id`、`github-installation-id`、`github-private-key.pem`、`openai-api-key` の4 basename
だけを使い、実効UID所有、グループ/other権限なし、ディレクトリのオーナー read/execute、ファイルのオーナー read
（実行不可）を満たす必要がある。Linuxでは一度開いたディレクトリ descriptorから固定basenameを`openat`で
読み、symlink、通常ファイルでないnode、上限超過、読込み前後のメタデータ変化を拒否する。非Linux readerは開発テスト用
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
