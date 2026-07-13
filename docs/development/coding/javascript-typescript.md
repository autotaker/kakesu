# JavaScript・TypeScriptコーディングガイドライン

## モジュールとデータ

- 新規Node.jsコードは既存パッケージのmodule方式へ合わせる。
- ファイルI/O、プロセス、時刻、外部コマンドの境界を関数へ分離する。
- TypeScriptでは`any`を境界の逃げ道にせず、`unknown`を検証して狭める。
- 外部データは型主張だけで信用せず、Schemaまたは実行時検証を通す。

## エラーとコマンド実行

- Promiseを浮かせず、失敗をawait元へ伝える。
- `spawn`または`execFile`へ引数配列を渡し、利用者入力をシェル文字列へ連結しない。
- 終了コード、signal、stderrを確認する。
- JSONやHTMLへ埋め込む値は文脈に応じてescapeする。

## 検査

```sh
pnpm lint:docs
pnpm test:terminology
node --test scripts/**/*.test.mjs
```

対象テストがglob展開に依存する場合は、実際のパッケージ スクリプトまたは`make check`を正本とする。
