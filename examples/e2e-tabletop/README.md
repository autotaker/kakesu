# Kakesu E2E Tabletop Debugging

4つのPlaneをまたぐ代表シナリオについて、コンポーネント間で交換される
ペイロードを時系列に並べ、`draft-v0` Schemaへの適合を検査する。

## 成果物

- `executable-scenarios.json`, `executable-scenario-002.json`, `executable-scenario-004-incident.json`: E2E-001〜004の`active` コンポーネント間シーケンス 投影（scenario ID重複は禁止）
- `domain-payloads-canonical.json`: シーケンスから独立した既存メッセージの正規 ドメイン ペイロード
- `domain-payloads-001.json`〜`004.json`: 新規中間メッセージの正規 ドメイン ペイロード
- `../../scripts/validate-tabletop-scenarios.mjs`: Schema適合検査
- `independent-review.md`: Schema検査では検出できない因果関係、状態遷移、フィールド不足の独立レビュー
- `sequence-requirements.json`: 許可、Task完了、子、非同期 再開、ポリシー改定の必須メッセージ 連鎖
- `viewer.html`: シナリオ、コンポーネント 経路、状態遷移、シーケンス/正規 ペイロードを閲覧する静的viewer

## 実行

```sh
node scripts/validate-tabletop-scenarios.mjs
node scripts/test-tabletop-validator.mjs
```

ビューアーはリポジトリ ルートでHTTP サーバーを起動して開くとJSONを自動読込する。

```sh
python3 -m http.server 8000
# http://127.0.0.1:8000/examples/e2e-tabletop/viewer.html
```

`viewer.html`を直接開く場合は、画面上の「JSONを選択」またはdrag and dropで
`examples/e2e-tabletop`のJSON群を読み込める。通常は同梱済みの`viewer-data.js`を
自動読込するため、JSONを選択する必要はない。ペイロード更新後は次を実行する。

```sh
node scripts/build-tabletop-viewer-data.mjs
```

成功時は、シナリオ数、シーケンス/正規 ペイロード数、全Planeを通過したことを表示する。検査器は
外部依存を持たせないため下書き 2020-12の利用中keywordだけを実装している。
標準検証器との差分、ステップ間不変条件、正規 ドメイン 割り当ての検査範囲は独立レビューに記録する。

`test-tabletop-validator.mjs`は必須メッセージ、相関 パス、entity 状態、因果関係、
冪等性、正規 ドメイン 割り当てを意図的に壊し、すべて拒否されることを確認する。
