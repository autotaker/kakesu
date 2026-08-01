# Development Agentハーネス

Kakesuを開発するための外部開発基盤である。Kakesu本体のランタイム、Goモジュール、配布物には含めない。

現在はbuild/install境界だけを固定したスキャフォールドである。各バイナリは`--version`だけ成功し、通常起動は
未実装としてfail-closedする。

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
