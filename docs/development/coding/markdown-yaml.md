# Markdown・YAMLコーディングガイドライン

## Markdown

- 一文書に一つの中心的な責務を持たせる。
- 正本を明示し、同じ規則を複数文書へコピーしない。
- 見出し階層を飛ばさず、相対リンクが存在することを確認する。
- コマンド例は実行場所、前提、期待結果が分かる形にする。
- 状態、責務、メッセージ、永続化境界を変更する場合は`docs/AGENTS.md`にも従う。

## YAML

- 機械処理するYAMLには`version`を持たせ、対応するSchemaを用意する。
- ID、列挙型、日付形式を自由記述にしない。
- 本文をYAMLへ複製せず、Markdown正本への参照を持たせる。
- anchorとmerge キーは読み手と検証器の挙動が不明瞭になるため、運用データでは使わない。
- 配列順に意味がある場合はSchemaまたは説明で明示する。

## 検査

```sh
pnpm lint:docs
pnpm test:terminology
git diff --check
```
