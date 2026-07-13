# Pythonコーディングガイドライン

## 構造と型

- 公開関数、境界データ、永続化入出力に型注釈を付ける。
- dataclass、TypedDict、検証モデルを責務に応じて使い分け、辞書の暗黙契約を広げない。
- import時に外部I/Oや重い初期化を行わない。
- 同期と非同期の境界を明示し、ライブラリ内部で無断にイベント ループを作らない。

## エラーと資源

- 裸の`except`を使わず、回復可能な例外だけを捕捉する。
- 例外を置換する場合は`raise ... from ...`で原因を保つ。
- ファイル、接続、一時資源はコンテキスト マネージャーで閉じる。
- パス、subprocess引数、外部入力を文字列連結でシェルへ渡さない。

## 検査

```sh
uv run --project memory pytest
uv run --project memory ruff check memory
uv run --project memory ruff format --check memory
```
