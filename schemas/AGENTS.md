# Schema change instructions

このディレクトリ以下のJSON Schemaは、設計時に固定すべきPlane間契約と永続化payloadの検査正本である。実装都合のdraft version変更は許容するが、検査を弱める変更を互換性の名目で黙って行わない。

## Schema更新規則

1. 所有Planeを確認し、Schemaを`control-plane`、`execution-plane`、`governance-plane`、`memory-plane`または`common`へ置く。
2. `$schema`、`$id`、`x-schema-version`、`x-schema-revision`、`x-stability`を持たせる。
3. 原則`additionalProperties: false`とし、ID、timestamp、digest、refは`common/primitives.schema.json`を再利用する。
4. 状態変更、Command、Event、Authority Request/Decisionはgeneric sequence projectionだけで済ませず、canonical domain payloadを作る。
5. RequestとDecision、CommandとEvent、TaskとRunなどの関係は、単体Schemaだけでなくsequence requirementまたはvalidatorでID・version・digestをjoinする。
6. 条件付き必須、状態別field、risk floorなどJSON Schemaで表現できる不変条件はSchemaへ置く。集合一致、順序、graph closureなどcross-record条件はvalidatorへ置く。
7. 新しい具体message typeがcanonical必須判定を名前変更で迂回しないよう、category単位の検査または明示一覧を更新する。
8. Schema catalogのREADMEと関連設計書を同時に更新する。

## Canonical payloadと負例

- active trace内の独立domain state変更にはcanonical payloadを一件対応させる。
- canonical総集合とactive trace使用集合を一致させ、未使用recordを残さない。
- 新しい制約には、その制約を一つだけ破るnegative mutationを可能な限り追加する。
- renameや分割時は旧Schema、旧payload、旧bindingの参照を`rg`で確認して除去する。

## 必須テストとレビュー

```sh
node scripts/validate-tabletop-scenarios.mjs
node scripts/test-tabletop-validator.mjs
git diff --check
```

Viewer対象payloadを変更した場合は、先に`node scripts/build-tabletop-viewer-data.mjs`を実行する。

コミット前にSchema独立レビューを行い、少なくともcanonical coverage、required/conditional field、cross-record join、状態遷移、未使用payload、mutation coverageを確認する。Plane間sequenceも変わる場合は別途Sequence reviewを行う。P0が残る状態でコミットしない。

代表E2Eを新設・変更する場合は`.agents/skills/tabletop-debug-scenarios/SKILL.md`に従う。
