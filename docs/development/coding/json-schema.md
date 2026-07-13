# JSON Schemaコーディングガイドライン

## 基本

- `$schema`、`$id`、版、安定度を既存規約に合わせる。
- 原則`additionalProperties: false`とし、意図しない契約拡張を拒否する。
- ID、日時、ダイジェスト、参照などの共通定義を再利用する。
- 条件付き必須や状態別不変条件はSchemaで表現し、集合、順序、グラフ結合は検証器へ置く。

## 変更

- 制約追加には、その制約だけを破る負例を追加する。
- 名前変更や分割では旧Schema、参照、フィクスチャ、検証器の残存を検索する。
- Schemaを弱める変更を互換性の名目で行わない。
- Plane間契約と正規ペイロードを変更する場合は`schemas/AGENTS.md`と`docs/AGENTS.md`に従う。

## 検査

```sh
node scripts/validate-tabletop-scenarios.mjs
node scripts/test-tabletop-validator.mjs
git diff --check
```
