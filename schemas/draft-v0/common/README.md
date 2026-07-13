# Common Schema カタログ — draft-v0

Plane間で共有するprimitiveと共通形式を所有する。業務上の判断やライフサイクルは各Planeへ置く。

## P0

以下は`draft-v0:r1`として追加済みである。

| Schema | 固定する内容 |
|---|---|
| `primitives.schema.json` | UUID/ID、論理 参照、SHA-256 ダイジェスト、タイムスタンプ、duration、シーケンス、上限付き テキスト |
| `schema-reference.schema.json` | `schema_id`、`schema_revision`、`schema_digest` |
| `error.schema.json` | エラー コード、メッセージ、retryable、details 参照、safe diagnostics |
| `evidence-reference.schema.json` | 証跡 / 成果物 / スナップショット 参照とダイジェスト |
| `event-envelope.schema.json` | イベント ID、subject、シーケンス、タイムスタンプ、actor、ペイロード スキーマ 参照 |
| `message-envelope.schema.json` | コンポーネント、相関、因果関係を持つcross-plane メッセージ 共通形式 |
| `sequence-invariant.schema.json` | predecessor/successorと結合 フィールドによるステップ間制約 |
| `sequence-message.schema.json` | 机上実行用メッセージのコンポーネント間因果、冪等性、entity状態遷移 |

## P1

| Schema | 固定する内容 |
|---|---|
| `pagination.schema.json` | カーソル、上限、切り詰め済み |
| `lease.schema.json` | オーナー、期限切れ、試行 |
| `budget.schema.json` | ステップ、トークン、バイト、期限上限 |

すべての配列には必要に応じて`maxItems`と`uniqueItems`、文字列には`minLength` / `maxLength`、タイムスタンプには`format: date-time`を指定する。
