# Kakesu E2E Tabletop Debugging

4つのplaneをまたぐ代表シナリオについて、コンポーネント間で交換される
Payloadを時系列に並べ、`draft-v0` Schemaへの適合を検査する。

## 成果物

- `executable-scenarios.json`, `executable-scenario-002.json`, `executable-scenario-004-incident.json`: E2E-001〜004のactive component間sequence projection（scenario ID重複は禁止）
- `domain-payloads-canonical.json`: sequenceから独立した既存messageのcanonical domain Payload
- `domain-payloads-001.json`〜`004.json`: 新規中間messageのcanonical domain Payload
- `../../scripts/validate-tabletop-scenarios.mjs`: Schema適合検査
- `independent-review.md`: Schema検査では検出できない因果関係、状態遷移、field不足の独立レビュー
- `sequence-requirements.json`: Grant、Task完了、child、Async resume、Policy改定の必須message chain
- `viewer.html`: Scenario、component route、状態遷移、sequence/canonical Payloadを閲覧する静的viewer

## 実行

```sh
node scripts/validate-tabletop-scenarios.mjs
node scripts/test-tabletop-validator.mjs
```

Viewerはrepository rootでHTTP serverを起動して開くとJSONを自動読込する。

```sh
python3 -m http.server 8000
# http://127.0.0.1:8000/examples/e2e-tabletop/viewer.html
```

`viewer.html`を直接開く場合は、画面上の「JSONを選択」またはdrag and dropで
`examples/e2e-tabletop`のJSON群を読み込める。通常は同梱済みの`viewer-data.js`を
自動読込するため、JSONを選択する必要はない。Payload更新後は次を実行する。

```sh
node scripts/build-tabletop-viewer-data.mjs
```

成功時は、シナリオ数、sequence/canonical Payload数、全planeを通過したことを表示する。検査器は
外部依存を持たせないためDraft 2020-12の利用中keywordだけを実装している。
標準validatorとの差分、step間不変条件、canonical domain bindingの検査範囲は独立レビューに記録する。

`test-tabletop-validator.mjs`は必須message、correlation path、entity state、causation、
idempotency、canonical domain bindingを意図的に壊し、すべて拒否されることを確認する。
