# Schema構成と下書き バージョン規約

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

Plane ディレクトリの正規 Schemaが正本である。`api/`は複数PlaneのSchemaからResponses API形式へ合成するアダプターであり、ドメイン モデルの所有境界にはしない。

## 下書き バージョン

初期実装中のSchema familyは`draft-v0`とする。`draft-v0`の間は後方互換性を保証せず、実装知見に基づくbreaking 変更を許可する。

正規 Schemaは次のメタデータを持つ。

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "urn:kakesu:<plane>:<schema-name>:draft-v0:r1",
  "x-schema-version": "draft-v0",
  "x-schema-revision": 1,
  "x-stability": "draft"
}
```

- breaking / non-breakingを問わず、検証結果へ影響する変更では`x-schema-revision`と`$id`末尾を増やす。
- 永続化する入力 スナップショット、出力、イベント、ポリシー 文書には`schema_id`、`schema_revision`、`schema_digest`を記録する。
- 同じ`$id`の内容を上書きしない。永続インスタンスが存在する改訂は検証器とともに保持する。
- `draft-v0`内では移行を必須にしないが、再実行時はインスタンスが記録した改訂で検証する。
- 実装契約を安定化した時点で`v1` familyを作り、それ以降のcompatibility ポリシーを別途定める。

`draft-v0`はJSON Schema仕様の下書き番号ではない。JSON Schema dialectは2020-12に固定し、製品Schemaの安定度を`draft-v0`で表す。

## Product 名前空間

Schema URNの製品名前空間は`urn:kakesu:`とする。Kakesuへの改名前の`urn:agent-harness:` 名前空間は実装・永続化開始前の下書き 成果物であり、`active` aliasとして残さない。外部永続インスタンスが生じた後に製品名前空間を変更する場合は、旧検証器保持と明示的移行を必須とする。

## 正規 SchemaとAPI アダプター

OpenAI 関数 ツールや構造化出力が受け付けるJSON Schemaはdialectのsubsetである。そのため次を分離する。

```text
canonical JSON Schema
  -> semantic validator / persistence validator
  -> Responses API subsetへcompile
  -> draft-v0/api/*.json
```

条件付き制約、参照パターン、サイズ 上限などがAPI subsetで表現できない場合も、正規 Schemaまたは適用前意味 検証器から削除しない。

## Schema化する境界

次のいずれかに該当するデータはJSON Schemaを正本にする。

1. LLMへ渡す固定入力 スナップショットまたはLLMの構造化 出力
2. 関数 呼び出し 入力 / 出力
3. メールボックス、送信キュー、イベント Busを通るペイロード
4. 責任者 アダプターとのリクエスト / 判断
5. ルールエンジンが実行するポリシー 文書
6. クラッシュ 復旧や再試行で再実行する不変 スナップショット
7. 証跡、エピソード、記憶コンテキストとして長期保持する文書

DB内部だけで完結する正規化行は、外部化・スナップショット化しない限りSQL制約と論理型だけでもよい。

## 実装順

各PlaneのREADMEで`P0`を先に実装する。`draft-v0:r1`ではCommon、制御、実行、統治、記憶のP0 正規 Schemaを追加済みである。次の変更は実装検証の結果に応じて改訂を増やす。
