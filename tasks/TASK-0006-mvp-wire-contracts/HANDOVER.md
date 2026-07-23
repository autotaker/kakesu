---
task_id: "TASK-0006"
status: complete
completed_at: "2026-07-18T03:47:52Z"
---

# TASK-0006 HANDOVER

## 成果

- Go Core、Python Memory、Rust Governanceが、canonical JSON Schemaを変更せず、同一のMVP wire envelope、schema catalog、共有fixture、local設定を検査できるようになった。
- review済み実装commit `3e2842052e8229cfb3eef023124d651bca4cbd4d`を、2-parent merge commit `2a80256510901acbb063f12396fde152f5348de7`として製品`main`へ統合した。
- 3言語が共有する5 fixtureについて、`valid.json`だけを受理し、same route、非UTC表記、schema参照不整合、非object payloadの4負例を同じ結果で拒否する契約を確立した。

## 主要な変更

- `configs/local/wire.json`を追加し、`version: 1`、project-root相対の`schemas/draft-v0`、正確に`core-runtime` / `memory-service` / `governance-service`の3 component、および相対・非空・相互非重複の`run/*.sock`を定義した。3言語のreaderは未知field、不正version、component過不足、絶対path、`..`、空endpoint、重複endpointを拒否する。
- `fixtures/wire/envelope/`へ1正常例と4単一違反負例を追加した。正常例は空でないmessage metadata、異なるroute、`correlation_id`、明示した`"causation_message_id": null`、UTC RFC3339の末尾`Z`、object payload、`common/error.schema.json` r1への参照を持つ。
- Go `core/internal/message/envelope.go`、Python `memory/src/kakesu_memory/message.py`、Rust `governance/src/message.rs`へstrict decodeと境界検査を追加した。envelope/payload schemaのfield集合、明示nullと欠落、route、UTC `Z`、object payload、schema参照をfail closedに検査する。
- schema参照はcatalog内の`$id`と`x-schema-revision`で正確に1 artifactへ解決し、そのファイルの未加工バイト列をSHA-256した`sha256:` + lowercase 64 hexと完全一致する場合だけ受理する。正常fixtureのdigestは`sha256:5f5d3aee6fbdcb4325604d71c6ac1b09926070c8f335f481570c15b904e33c32`である。
- Go `core/internal/config/wire.go`、Python `memory/src/kakesu_memory/wire_config.py`、Rust `governance/src/wire_config.rs`へ共有設定readerを追加した。RustにはRFC3339とSHA-256検査用の`chrono` / `sha2`依存を追加した。
- 実装差分は21 path、1,209行追加・9行削除で、`schemas/**`は変更していない。UDS socketの作成・bind・connectや配送処理は追加していない。

## 検証結果

- 独立Reviewerは実装commit `3e2842052e8229cfb3eef023124d651bca4cbd4d`をTask、承認済みPLAN / QA PLANと照合し、P0/P1および検査FAILなしで`PASS`と判定した。Reviewerの`make check`もexit 0だった。
- マージ後QAは製品`main`の`2a80256510901acbb063f12396fde152f5348de7`を対象にQA-001〜QA-005をすべて`PASS`とした。Go 2 package、Python 17 test、Rust 5 testのfocused検査で、共有5 fixtureの計15判定、explicit nullと欠落、設定境界、Schemaの1-byte変異拒否を確認した。
- QAの`make check`はexit 0（18.69秒）。Go build/test/vet、Memory build/20 pytest/ruff、Governance build/test/fmt/clippy、tabletop 4 scenarios、terminology、process 32 tests、docs lint、viewer data、diff checkがPASSした。
- `sha256sum schemas/draft-v0/common/error.schema.json`は`5f5d3aee6fbdcb4325604d71c6ac1b09926070c8f335f481570c15b904e33c32`で、正常fixtureと一致した。一時catalogで末尾spaceを1 byte追加した場合は、3言語すべてがdigest mismatchとして拒否した。
- Reviewerの初回Rust依存取得とQAの初回Go/Python focused実行ではsandbox/cache制約が発生したが、許可済み環境またはリポジトリ既定の`.build` cacheを用いた同一検査がPASSしたため、実装不具合ではなく解消済み`environment_issue`と分類した。

## 判断

- `schemas/draft-v0/common/message-envelope.schema.json` r1をwire構造の正本として維持し、UTC末尾`Z`、route、explicit null、catalog identity/revision/raw-byte digestのcross-record制約は3言語adapterの境界検査とした。本TaskではSchema revision / `$id`を変更しない。
- digestはJSON canonicalization後ではなく、配布されるSchema artifactの正確なバイト列から計算する。空白、改行、整形だけの変更もartifact変更として検出する。
- 言語ごとのfixture/config複製はdriftを隠すため採用せず、リポジトリrootの共有JSONを唯一の相互運用fixture/configとした。
- wire timestampは同じUTC instantでも`+00:00`を許容せず、RFC3339として妥当でUTC offset 0かつwire表現が末尾`Z`であることを要求する。
- UDSは配送手段でありwire契約の正本ではない。本Taskはendpointの解決までとし、listener、ACK、retry、durable Inbox/Outboxを先取りしない。

## 既知の制約と未解決事項

- adapterはenvelope境界とschema artifactのidentity/revision/digestを検査するが、`payload`内容を参照先JSON Schemaで意味検証しない。
- local設定readerはproject-root相対pathを解決するだけで、endpoint親directoryの作成、UDS socketのbind/connect、lifecycle、ACK、retry、durable Inbox/Outboxを実装しない。これらはTask-0008以降の配送Taskの責務である。
- 将来Schemaを変更する後続Taskは、`schemas/README.md`のrevision / `$id`規則に従い、参照する共有fixtureのschema revisionと**raw-byte** SHA-256 digestを同時に更新する必要がある。digestだけ、またはSchemaだけの更新では3言語すべてが拒否する。
- 本Taskで固定したUTC末尾`Z`と共有fixture/configは後続Taskが再定義・言語別複製せず入力契約として利用する。UDS listener / ACKを追加する後続Taskでも、このwire envelopeとlocal endpoint keyを変更しない。

## 運用上の注意

- `causation_message_id`はnullableだがoptionalではない。因果messageがない場合もfieldを省略せず、明示的に`null`を送る。
- `occurred_at`はUTCと等価な時刻ではなく、末尾`Z`のRFC3339 wire表現を送る。`+00:00`や`+09:00`は拒否される。
- Schemaファイルの整形、末尾改行、空白だけを変更した場合もraw-byte digestが変わる。Schema変更Taskでは`sha256sum`相当で実artifactを再計算し、共有fixture参照を同一変更単位で更新する。
- configとfixtureはroot共有pathを直接読む。各言語配下へコピーしたり、環境変数・TOML・YAMLで同じ値を二重管理しない。
- `configs/local/wire.json`にcredential、host/port、秘密値を追加しない。相対UDS endpointの親directory作成とsocket bindは後続のtransport実装が所有する。
- sandbox内でfocused testを実行する場合、Go/Pythonはリポジトリ既定の`.build/go-cache` / `.build/uv-cache`を利用する。cache権限や依存取得失敗はwire契約FAILと混同せず、環境要因を切り分ける。

## Wikiへ引き渡す知識

### 再利用可能な知識

- 複数言語のwire互換性は、言語固有のテストデータではなく、単一の共有正常fixtureと「一件につき一制約だけを破る」共有負例matrixで固定するとdriftを検出しやすい。
- cross-recordなschema参照は、`schema_id`だけでなく`$id` + revisionでcatalog内の解決数を正確に1へ制約し、raw-byte digestまで結合すると、未知・重複・内容driftを同じ境界でfail closedにできる。
- nullable fieldの「明示null」と「field欠落」は、Goでは`json.RawMessage`、Rustではrequired wrapper、Pythonではfield集合の完全一致など、各decoderで存在判定を別に保持する必要がある。
- JSON Schemaの`format: date-time`だけでは`+09:00`等を拒否できない。MVPのwire規約がUTC `Z`を要求する場合は、RFC3339 parseと末尾`Z`の両方をadapterで検査する。
- local endpoint設定は、component集合を正確に固定し、pathをproject root相対・非空・parent traversalなし・相互非重複にすると、秘密値やnetwork設定を持ち込まず後続transportへ安全に引き渡せる。

### 反例・失敗・注意点

- Schema JSONをparse後に再serializeまたはcanonicalizeしてdigestすると、実際に配布するartifactの空白・改行・整形driftを見逃す。digest入力は読み取った未加工バイト列にする。
- UTC offsetが0であることだけを検査すると`+00:00`を受理し、末尾`Z`というwire表現契約を失う。offsetと原wire文字列を両方検査する。
- nullable fieldを通常のoptional型へ直接decodeすると、欠落と明示nullが同じ値になりうる。decode前後のどこかでfield存在を必ず検査する。
- 言語ごとにfixtureやlocal設定を複製すると、単体testがすべてPASSしてもPlane間で異なる契約を受理しうる。3言語のtestはroot共有artifactを読む。
- UDS endpointが設定に存在することをlistener完成とみなしてはならない。本Taskはpath解決だけで、socket作成・ACK・retry・durabilityの保証はない。
- sandboxのcache書込み・依存取得制約によるrunner停止を実装FAILへ直結させない。cacheをリポジトリ`.build`へ向け、同一検査の再現結果で分類する。

### 更新候補ページ

- `wiki/semantic/schemas/mvp-wire-contracts.md`（新規）: envelope不変条件、共有fixture matrix、schema catalogの一意解決、raw-byte digest、explicit null、UTC `Z`を記録する。
- `wiki/semantic/case-patterns/schema-artifact-digest-drift.md`（新規）: Schemaの1-byte変異、canonicalizationを避ける理由、revision / `$id` / fixture digestを同時更新する運用を反例付きで記録する。
- 後続Task-0008の取り込み時に、上記wire契約ページへUDS transportが消費する設定keyと、listener / ACK / retryが別責務であることを追記する。

## ブートストラップ例外

- 該当なし
