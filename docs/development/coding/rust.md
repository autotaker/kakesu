# Rustコーディングガイドライン

## 型と所有権

- 不正状態を列挙型とnewtypeで表現不能にする。
- `unsafe`は原則使わない。必要な場合は安全性不変条件、封じ込め境界、試験を記載する。
- 公開APIで所有権移動が不要なら参照を受け取り、不要なcloneを避ける。
- 文字列で状態や種別を分岐せず、型へ閉じ込める。

## エラーと並行処理

- ライブラリ境界では構造化したエラー型を使い、利用者向け表示と原因判定を分離する。
- `unwrap`と`expect`は不変条件が局所的に証明できる場合に限定する。
- ロック、チャネル、Taskの所有者と停止順を明示する。
- panicを通常の制御フローに使わない。

## 検査

```sh
cd governance && cargo fmt --check
cd governance && cargo clippy --locked --all-targets -- -D warnings
cd governance && cargo test --locked
```
