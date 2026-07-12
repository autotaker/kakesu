# Documentation change instructions

このディレクトリ以下の仕様・設計を更新するときは、文章だけで完了扱いにしない。変更した責務、状態、message、永続化境界がSchemaとE2E traceへ影響するかを確認する。

## 必須確認

1. `rg`で同じ概念、状態名、責務、message type、旧仕様の残存箇所を検索する。
2. Plane責務を変更した場合は、全体設計、該当PlaneのSchema catalog、データモデル、代表E2Eの記述を整合させる。
3. 状態遷移またはPlane間messageを変更した場合は、`examples/e2e-tabletop`のactive trace、canonical payload、sequence requirementを更新する。
4. Human Authorityとの通信はControl PlaneのAuthority Gatewayを経由させる。ほかのPlaneから人間への直接経路を書かない。
5. 旧仕様を置換した場合は、上書き順や互換用ファイルとして残さず、参照を移行して旧artifactを削除する。active `scenario_id`の重複は禁止する。

## テスト

変更後にrepository rootで必ず実行する。

```sh
node scripts/build-tabletop-viewer-data.mjs
node scripts/validate-tabletop-scenarios.mjs
node scripts/test-tabletop-validator.mjs
git diff --check
```

Viewerに関係する変更では、再生成後の`viewer-data.js`をコミット対象に含める。件数だけでなく、変更したscenarioのmessage chainとcanonical payloadが意図どおり表示されることを確認する。

## 独立レビュー

責務境界、Task lifecycle、Authority、Incident、SchemaまたはE2E sequenceを変更した場合は、コミット前に独立レビューを行う。可能なら次の観点を別レビュアーへ分ける。

- Sequence review: 端から端まで実行可能か、causation・順序・停止/再開・Plane責務・迂回経路を確認する。
- Schema review: canonical coverage、required field、ID join、状態整合、未使用payload、validatorの偽PASSを確認する。

P0が一件でも残る場合はコミットしない。修正後は同じ観点で再レビューし、両方がPASSしてからコミットする。P1を先送りする場合は、非blockingである根拠と将来の検査方法をレビュー記録へ残す。

机上デバッグの作成・更新には`.agents/skills/tabletop-debug-scenarios/SKILL.md`を使用する。
