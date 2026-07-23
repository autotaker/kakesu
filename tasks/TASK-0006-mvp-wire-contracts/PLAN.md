---
task_id: "TASK-0006"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-07-18T12:19:27+09:00"
approved_dev_profile: "sol-high"
approved_dev_profile_reason: "共通JSON Schema catalog、共有fixture、3言語の境界adapter、3プロセスが解決するlocal設定を同時に変更する。digest照合と時刻・route検査の不一致はPlane間でsilent acceptanceを生む契約上のリスクであり、局所的なLuna作業ではないため。"
approved_dev_profile_risk_signals:
  - cross_cutting
  - schema_contract
  - configuration_contract
  - multi_language_interoperability
planned_implementation_files: 8
planned_implementation_lines: 645
estimate_points: 5
---

# TASK-0006 PLAN

## DEV profile選定証跡

```yaml
profile: sol-high
model: gpt-5.6-sol
reasoning_effort: high
decision_rule: "Schema、設定、複数言語のwire境界を横断する、または相互運用性の不確実性があるTaskはsol-high"
reason: "同じJSONをGo/Python/Rustでdecode・catalog照合し、UTC、route、object payload、実ファイルのSHA-256を一貫してfail closedにする必要がある"
risk_signals: [cross_cutting, schema_contract, configuration_contract, multi_language_interoperability]
```

DEV中にfixture、catalog探索、JSON decoderのいずれかでここにない言語差・Schema解釈差を発見した場合も、`sol-high` を継続する。profileをLunaへ降格しない。承認後に要求が増えた場合はmain Agentが再見積もりとPLAN改訂を承認する。

## 受け入れ条件の具体化

| 条件 | 観測方法 | 期待結果 | 根拠 |
|---|---|---|---|
| 共有wire fixture | Go `go test ./...`、Python `uv run --project memory pytest`、Rust `cargo test --locked`がリポジトリrootの同一5 JSONを読む | `valid.json`だけを受理し、4つの負例を全言語で拒否する | TASK.md、`docs/13-technology-stack.md` §5 |
| metadataとUTC | valid fixtureに空でない`message_id`、異なる`from_component`/`to_component`、`correlation_id`、explicit `null`の`causation_message_id`、末尾`Z`のRFC3339 timestamp、schema参照、object payloadがある | offset付きだが非UTCのRFC3339 timestampは拒否する。既存r1 Schemaの`format: date-time`をrevisionなしで変えず、MVP adapterの追加wire規約として検査する | TASK.md、`message-envelope.schema.json`、`schemas/AGENTS.md` |
| catalogとdigest | fixtureの`payload_schema`を`schema_catalog_root`配下で`$id`と`x-schema-revision`により一意に解決する | `schema_digest`は解決済み対象Schemaファイルの**正確なバイト列**をSHA-256し、`sha256:` + lowercase 64 hexで完全一致する場合だけ受理する。JSON正規化・整形・改行変換は行わない | `schemas/README.md`、TASK明確化 |
| 4種の負例 | validと同じ構造を基に、違反を一つだけ導入したfixtureを読む | `invalid-same-route.json`、`invalid-non-utc-timestamp.json`、`invalid-schema-reference-digest.json`、`invalid-non-object-payload.json`をいずれも拒否する | TASK明確化、`schemas/AGENTS.md`の単一違反負例規約 |
| local設定 | 3言語の設定reader testが同一`configs/local/wire.json`をプロジェクトroot基準で解決する | `version: 1`、相対`schema_catalog_root: schemas/draft-v0`、`components`に`core-runtime`、`memory-service`、`governance-service`だけを持ち、相対かつ相互に重複しないUDS endpointを得る。秘密値・listener接続はない | TASK.md、`docs/00-kakesu.md` §3.1 |
| Schema更新の検出 | catalog対象Schemaのコピーを一バイトだけ変更した一時test catalogを使い、fixture digestを変更しない | 3言語すべてがdigest mismatchとして拒否する。fixtureのschema参照/digestを更新しないSchema変更は通らない | TASK.md |

## 関連Wikiと判断

- `docs/00-kakesu.md` §3.1は、JSON Schemaを意味的メッセージの正本、UDSを配送手段とし、各Planeのqueueを正本とする。したがって本Taskはlistener、ACK、queue、再送を実装しない。
- `docs/13-technology-stack.md` §5は`message-envelope.schema.json`をそのまま共通形式とし、`from_component`、`to_component`、`correlation_id`、`causation_message_id`、`payload_schema`を保持すると定める。
- `schemas/README.md`は検証結果に影響する既存Schemaの変更時にrevision/$idを増やす。TASKの対象外条件に従い、本TaskはSchemaを変更しない。UTCとcatalog byte-digestはr1 Schemaに対するadapter境界の追加検査であり、既存Schemaのrevisionを暗黙に変更しない。
- `schemas/AGENTS.md`はcross-recordの結合を検証器へ置き、単一制約を破る負例を求める。catalog上の`$id`、revision、ファイルdigestの結合はこのTaskの3言語adapterで検査する。
- Task-0008は本Taskの`configs/local/wire.json`、envelope validation、catalog照合を前提にUDS frame/ACKを追加する。endpoint形式・設定キーをTask-0008で変更しない。

## 設計

### 決定した構成

canonical wire構造は既存の`schemas/draft-v0/common/message-envelope.schema.json` r1のままとする。共有fixtureは`fixtures/wire/envelope/`に置き、全言語が各自のsource rootからリポジトリrootを発見して同じファイルを読む。fixture decoderはunknown/missing fieldを許容せず、`causation_message_id`はexplicit `null`を必須として、欠落をnullへ補完しない。

各adapterは次を同じ順序でfail closedに検査する。

1. JSON envelopeをdecodeし、required metadata、distinct route、末尾`Z`のRFC3339 timestamp、object payloadを検査する。
2. 設定をプロジェクトroot基準で読み、`schema_catalog_root`を解決する。絶対path、`..`を含むendpoint、未知component、欠落/余分なcomponent、重複endpoint、空endpointを拒否する。
3. catalog配下のSchemaを読み、fixtureの`schema_id`とSchemaの`$id`、fixtureのrevisionと`x-schema-revision`を照合する。該当が0件または複数件なら拒否する。
4. 解決したSchemaファイルの読み取りバイト列をそのままSHA-256し、`sha256:<lowercase-hex>`をfixtureの`schema_digest`と完全比較する。

`valid.json`は`common/error.schema.json`をpayload Schemaとして参照し、そのファイルの実バイト列digestを記録する。payloadは同Schemaのrequired fieldsを満たすobjectにする。`invalid-schema-reference-digest.json`だけは同じpayloadとroute/timestampを保ったまま、catalogに存在しないschema ID/revisionとdigestを組にした参照へ変える。各負例は指定の一条件だけを破る。

`configs/local/wire.json`は次の論理形を固定する。相対値はプロジェクトrootから解決し、endpointの親ディレクトリ作成・socket bindはTask-0008の責務である。

```json
{
  "version": 1,
  "schema_catalog_root": "schemas/draft-v0",
  "components": {
    "core-runtime": { "uds_endpoint": "run/core-runtime.sock" },
    "memory-service": { "uds_endpoint": "run/memory-service.sock" },
    "governance-service": { "uds_endpoint": "run/governance-service.sock" }
  }
}
```

### 代替案と不採用理由

- Schema JSONをcanonicalizeしてdigestする案は、コメント、空白、改行を含む実際に配布・検証するartifactの改変を検出できないため不採用。対象ファイルのバイト列SHA-256を用いる。
- 言語ごとのfixture複製は互換性driftを検出できないため不採用。root共有JSONを使う。
- UTCを各言語のtimezone-aware判定だけにする案は`+09:00`を通すため不採用。RFC3339 parse成功に加えてUTC offset 0かつwire表現末尾`Z`を要求する。
- UDSをこのTaskでbindする案は、durable Inbox/Outbox、ACK、再送というTask-0008の原子性境界を先取りするため不採用。
- TOML/YAMLまたは環境変数へ設定を複製する案は3言語間driftと秘密値混入を増やすため不採用。1つの非秘密JSON設定を読む。

### 責務と不変条件

- Schemaは既存r1をwire構造の正本として保持する。adapterはJSON Schemaの意味payload validatorを導入せず、catalog identity/revision/digestとenvelope境界だけを検査する。
- fixtureは言語非依存の互換証跡であり、個別言語のコピーや生成物を正本にしない。
- configはcomponent名、schema catalog root、相対UDS endpointだけを所有する。credential、network address、listener lifecycleを含めない。
- component集合は正確に3つで、endpointは相対、非空、互いに異なり、`core-runtime` / `memory-service` / `governance-service`以外を許容しない。
- schema referenceはcatalog内の一意の`$id`、同一revision、正確な`sha256:hex`に結合する。digest prefixなしやuppercaseは受理しない。
- `payload`はJSON objectだけ、`from_component != to_component`、timestampはUTC RFC3339の`Z`表記だけを受理する。明示nullと欠落は同値でない。

### 失敗時・移行・互換性

fixture/config parse不良、未知または曖昧なschema、revision/digest不一致、不正route/timestamp/payloadはテスト・起動前設定読込で観測可能なerrorとしてfail fastにする。socketへ接続しない。`draft-v0`に旧fixture/versionの互換shimは作らない。既存の単体constructor testsは維持し、新しい共有fixture testに移すことで既存の`sha256:fixture`等のplaceholderをcatalog-valid扱いにしない。

Schemaの内容を将来変更する場合は、`schemas/README.md`のrevision/$id規則を別Taskで適用し、参照fixtureのrevisionと対象Schemaのbyte digestを更新する。本TaskはSchema artifactを変更しないため、revisionを増やさない。

## 変更予定

見積もり対象は実装コードと設定のみである。test、fixture、文書は表へ記載しても見積もりから除外する。

| ファイル | 種別 | 概算変更行数 | 変更内容 |
|---|---|---:|---|
| `configs/local/wire.json` | configuration | 20 | version、catalog root、正確に3 componentと相対・非重複UDS endpointを定義する非秘密設定。 |
| `core/internal/message/envelope.go` | implementation | 105 | strict JSON decode、UTC `Z` validation、object payload、catalog identity/revision/byte-digest検査を追加。 |
| `core/internal/config/wire.go` | implementation | 100 | shared local JSON設定のroot-relative解決とcomponent/endpoint validation。 |
| `memory/src/kakesu_memory/message.py` | implementation | 95 | strict fixture decode、UTC `Z`、object、catalog SHA-256検査。 |
| `memory/src/kakesu_memory/wire_config.py` | implementation | 90 | local設定のdecode/validationとroot-relative解決。 |
| `governance/src/message.rs` | implementation | 125 | serde decodeのrequired/null区別、RFC3339 UTC、object、catalog SHA-256検査。 |
| `governance/src/wire_config.rs` | implementation | 100 | local設定のserde decode/validationとroot-relative解決。 |
| `governance/src/lib.rs` | implementation | 10 | wire config moduleを公開する。 |
| `fixtures/wire/envelope/valid.json` | fixture（除外） | - | `common/error.schema.json`の実バイト列digestを持つ正常envelope。 |
| `fixtures/wire/envelope/invalid-same-route.json` | fixture（除外） | - | from/toが同一の単一違反。 |
| `fixtures/wire/envelope/invalid-non-utc-timestamp.json` | fixture（除外） | - | `+09:00`等、RFC3339だが末尾`Z`でない単一違反。 |
| `fixtures/wire/envelope/invalid-schema-reference-digest.json` | fixture（除外） | - | catalogとschema ID/revision/digestが結合しない単一違反。 |
| `fixtures/wire/envelope/invalid-non-object-payload.json` | fixture（除外） | - | array/string/nullではなくobjectである条件だけを破る負例。 |
| `core/internal/message/fixture_test.go`、`core/internal/config/wire_test.go` | test（除外） | - | fixture matrix、catalog one-byte mutation、config正負例。 |
| `memory/tests/test_wire_fixtures.py`、`memory/tests/test_wire_config.py` | test（除外） | - | 同じfixture matrix、catalog mutation、config正負例。 |
| `governance/tests/wire_fixtures.rs`、`governance/tests/wire_config.rs` | test（除外） | - | 同じfixture matrix、catalog mutation、config正負例。 |

見積もり対象は8ファイル、645行。`file_score = ceil(8 / 3) = 3`、`line_score = ceil(645 / 200) = 4`、`raw_points = 4`。許容scale `1, 2, 3, 5, 8, 13` のうち4以上の最小値は5のため、`estimate_points: 5` とする。

## 実装手順

1. `configs/local/wire.json`と3言語のsetting readerを追加し、component集合、root-relative catalog、endpointのlocal-only制約をテストする。
2. 正常fixtureを作り、`common/error.schema.json`の現時点の正確なバイト列から`sha256:hex`を生成して記録する。そこから単一違反の4 fixtureを作る。
3. Go、Python、RustのEnvelope adapterへstrict decode、nullの存在確認、UTC `Z`、route、object payload、catalog ID/revision/byte-digest照合を実装する。
4. 各言語で正常1件・負例4件・config負例・catalog one-byte mutationを自動試験し、3実装が同じaccept/reject matrixになることを確認する。
5. 既存言語単体試験と全体`make check`を実行し、独立ReviewerがSchema/fixture/configの結合とTask-0008へ持ち越す配送責務を確認する。

## 検証計画

- Go: `cd core && go test ./...`。shared fixture 5件、strict decode、config component/endpoint負例、temporary catalogの一バイト変更を確認する。
- Python: `uv run --project memory pytest`。Goと同じfixture名・期待結果・catalog mutationを確認する。
- Rust: `cd governance && cargo test --locked`。Go/Pythonと同じmatrix、explicit nullの存在、catalog mismatchを確認する。
- 正常fixtureは各readerで同一`schema_id`、revision、digestを解決し、対象`common/error.schema.json`のraw bytesをhashして一致する。schemaファイルだけを変えたtemporary catalogでは全readerが失敗する。
- 設定の正例は3 endpointをroot-relativeに解決する。version不正、component不足/余剰、absolute/`..` endpoint、重複endpoint、absolute catalog rootを各language readerが拒否する。
- `make check`、`git diff --check`、Schema independent reviewを実行する。JSON Schema artifactは変更しないためtabletop scenario更新・viewer再生成は不要である。

## DEV開始前の未解決事項

なし。`common/error.schema.json`をfixtureのpayload Schemaにすること、raw-byte SHA-256、project-root-relative setting、5 fixtureの名前と違反内容、3 component/endpointの値をこのPLANで固定する。実装中にSchema revision変更、payload意味validator、UDS bind/ACKが必要と判明した場合はTask範囲外としてmain Agentへ戻す。

## main Agentレビュー

- [x] 受け入れ条件が検証可能である。
- [x] 設計観点と代替案を検討している。
- [x] QA計画を作成できる。
- [x] 見積もりが規則どおりである。
- [x] DEV profile `sol-high`、理由、risk signalsを承認した。
- [x] DEV開始を承認した。
