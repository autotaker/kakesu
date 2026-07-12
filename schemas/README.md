# Schema構成とDraft version規約

JSON Schemaは、状態と判断の所有境界に合わせて次の4 Planeへ分ける。

```text
schemas/
  README.md
  domain-types.ts                 # 設計確認用の論理型。runtime validatorの正本ではない
  draft-v0/
    common/                       # Plane間で共有するprimitive / envelope
    control-plane/                # Task、Contract、Mailbox、人間との唯一のAuthority routing境界
    execution-plane/              # Agent Run、Tool result、Async、Continuation
    governance-plane/             # Workspace Security Policy、CASB、Grant、Audit
    memory-plane/                 # Evidence、Task Episode、Memory Context、Wiki
    api/                          # Responses APIへ渡す合成済みadapter bundle
```

Plane directoryのcanonical Schemaが正本である。`api/`は複数PlaneのSchemaからResponses API形式へ合成するadapterであり、domain modelの所有境界にはしない。

## Draft version

初期実装中のSchema familyは`draft-v0`とする。`draft-v0`の間は後方互換性を保証せず、実装知見に基づくbreaking changeを許可する。

canonical Schemaは次のmetadataを持つ。

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "urn:agent-harness:<plane>:<schema-name>:draft-v0:r1",
  "x-schema-version": "draft-v0",
  "x-schema-revision": 1,
  "x-stability": "draft"
}
```

- breaking / non-breakingを問わず、検証結果へ影響する変更では`x-schema-revision`と`$id`末尾を増やす。
- 永続化するinput snapshot、output、Event、Policy documentには`schema_id`、`schema_revision`、`schema_digest`を記録する。
- 同じ`$id`の内容を上書きしない。永続instanceが存在するrevisionはvalidatorとともに保持する。
- `draft-v0`内ではmigrationを必須にしないが、replay時はinstanceが記録したrevisionで検証する。
- 実装契約を安定化した時点で`v1` familyを作り、それ以降のcompatibility policyを別途定める。

`draft-v0`はJSON Schema仕様のDraft番号ではない。JSON Schema dialectは2020-12に固定し、製品Schemaの安定度を`draft-v0`で表す。

## Canonical SchemaとAPI adapter

OpenAI Function ToolやStructured Outputsが受け付けるJSON Schemaはdialectのsubsetである。そのため次を分離する。

```text
canonical JSON Schema
  -> semantic validator / persistence validator
  -> Responses API subsetへcompile
  -> draft-v0/api/*.json
```

conditional constraint、reference pattern、size limitなどがAPI subsetで表現できない場合も、canonical Schemaまたは適用前semantic validatorから削除しない。

## Schema化する境界

次のいずれかに該当するデータはJSON Schemaを正本にする。

1. LLMへ渡す固定input snapshotまたはLLMのStructured Output
2. Function Call input / output
3. Mailbox、Outbox、Event Busを通るpayload
4. Authority adapterとのrequest / decision
5. Rule Engineが実行するPolicy document
6. crash recoveryや再試行でreplayするimmutable snapshot
7. Evidence、Episode、Memory Contextとして長期保持する文書

DB内部だけで完結する正規化rowは、外部化・snapshot化しない限りSQL制約と論理型だけでもよい。

## 実装順

各PlaneのREADMEで`P0`を先に実装する。`draft-v0:r1`ではCommon、Control、Execution、Governance、MemoryのP0 canonical Schemaを追加済みである。次の変更は実装検証の結果に応じてrevisionを増やす。
