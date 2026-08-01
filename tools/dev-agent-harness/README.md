# Development Agentハーネス

Kakesuを開発するための外部開発基盤である。Kakesu本体のランタイム、Goモジュール、配布物には含めない。

現在はbuild/install境界だけを固定したスキャフォールドである。各バイナリは`--version`だけ成功し、通常起動は
未実装としてfail-closedする。

`dev-agent-harness-setup check-config --config PATH` は、サービス起動前に設定例をローカル検証する read-only
コマンドである。version 1 の strict JSON、絶対かつcleanな配置path、相異なるLinux user名、
`network.default=deny`、group/world writableではないregular設定file（64 KiB以下）だけを受理する。成功時は設定値を表示せず
固定summaryを出力し、失敗時も入力値やpathを診断へ含めない。設定検証はユーザー作成、秘密の読込、network、
IPC、サービス起動を行わない。

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
変更しない。実装後もprovisionとenable/startは明示的な管理操作として分離する。
