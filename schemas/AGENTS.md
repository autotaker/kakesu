# Schema変更手順

このディレクトリ以下のJSON Schemaは、設計時に固定すべきPlane間契約と永続化ペイロードの検査正本である。実装都合の下書き バージョン変更は許容するが、検査を弱める変更を互換性の名目で黙って行わない。

## Schema更新規則

1. 所有Planeを確認し、Schemaを`control-plane`、`execution-plane`、`governance-plane`、`memory-plane`または`common`へ置く。
2. `$schema`、`$id`、`x-schema-version`、`x-schema-revision`、`x-stability`を持たせる。
3. 原則`additionalProperties: false`とし、ID、タイムスタンプ、ダイジェスト、参照は`common/primitives.schema.json`を再利用する。
4. 状態変更、コマンド、イベント、責任者への依頼/判断はgeneric シーケンス 投影だけで済ませず、正規 ドメイン ペイロードを作る。
5. リクエストと判断、コマンドとイベント、Taskと実行などの関係は、単体Schemaだけでなくシーケンス requirementまたは検証器でID・バージョン・ダイジェストを結合する。
6. 条件付き必須、状態別フィールド、リスク 下限などJSON Schemaで表現できる不変条件はSchemaへ置く。集合一致、順序、グラフ 閉包などcross-record条件は検証器へ置く。
7. 新しい具体メッセージ 型が正規必須判定を名前変更で迂回しないよう、category単位の検査または明示一覧を更新する。
8. Schema カタログのREADMEと関連設計書を同時に更新する。

## 正規 ペイロードと負例

- `active` トレース内の独立ドメイン 状態変更には正規 ペイロードを一件対応させる。
- 正規総集合と`active` トレース使用集合を一致させ、未使用記録を残さない。
- 新しい制約には、その制約を1つだけ破るネガティブ 変異を可能な限り追加する。
- renameや分割時は旧Schema、旧ペイロード、旧割り当ての参照を`rg`で確認して除去する。

## 必須テストとレビュー

```sh
node scripts/validate-tabletop-scenarios.mjs
node scripts/test-tabletop-validator.mjs
git diff --check
```

ビューアー対象ペイロードを変更した場合は、先に`node scripts/build-tabletop-viewer-data.mjs`を実行する。

コミット前にSchema独立レビューを行い、少なくとも正規 網羅率、必須/conditional フィールド、cross-record 結合、状態遷移、未使用ペイロード、変異 網羅率を確認する。Plane間シーケンスも変わる場合は別途シーケンス レビューを行う。P0が残る状態でコミットしない。

代表E2Eを新設・変更する場合は`.agents/skills/tabletop-debug-scenarios/SKILL.md`に従う。
