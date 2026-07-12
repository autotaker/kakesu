# Common Schema catalog — draft-v0

Plane間で共有するprimitiveとEnvelopeを所有する。業務上のDecisionやLifecycleは各Planeへ置く。

## P0

以下は`draft-v0:r1`として追加済みである。

| Schema | 固定する内容 |
|---|---|
| `primitives.schema.json` | UUID/ID、logical ref、SHA-256 digest、timestamp、duration、sequence、bounded text |
| `schema-reference.schema.json` | `schema_id`、`schema_revision`、`schema_digest` |
| `error.schema.json` | error code、message、retryable、details ref、safe diagnostics |
| `evidence-reference.schema.json` | Evidence / Artifact / Snapshot refとdigest |
| `event-envelope.schema.json` | event ID、subject、sequence、timestamp、actor、payload schema ref |
| `message-envelope.schema.json` | component、correlation、causationを持つcross-plane message envelope |
| `sequence-invariant.schema.json` | predecessor/successorとjoin fieldによるstep間制約 |
| `sequence-message.schema.json` | 机上実行用messageのcomponent間因果、冪等性、entity状態遷移 |

## P1

| Schema | 固定する内容 |
|---|---|
| `pagination.schema.json` | cursor、limit、truncated |
| `lease.schema.json` | owner、expiry、attempt |
| `budget.schema.json` | step、token、byte、deadline上限 |

すべてのarrayには必要に応じて`maxItems`と`uniqueItems`、文字列には`minLength` / `maxLength`、timestampには`format: date-time`を指定する。
