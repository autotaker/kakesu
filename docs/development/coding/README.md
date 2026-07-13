# コーディングガイドライン

共通原則を適用したうえで、変更ファイルに対応する言語別文書を使用する。

| 対象 | ガイドライン |
|---|---|
| Go | [go.md](go.md) |
| Python | [python.md](python.md) |
| Rust | [rust.md](rust.md) |
| JavaScript / TypeScript | [javascript-typescript.md](javascript-typescript.md) |
| Markdown / YAML | [markdown-yaml.md](markdown-yaml.md) |
| JSON Schema | [json-schema.md](json-schema.md) |

## 共通原則

- 責務境界とデータ所有者を先に決める。
- 公開契約と内部実装を分離する。
- 不正状態を型、Schema、検証器の最も近い層で拒否する。
- エラーを握りつぶさず、呼び出し元が判断できる文脈を付ける。
- 時刻、乱数、外部I/Oをテスト可能な境界へ分離する。
- ログへ秘密、認証情報、未無害化の利用者入力を出さない。
- 変更した失敗モードを検出するテストを追加する。
- 既存の単純な構造で表現できる場合、新しい抽象化や状態を増やさない。
