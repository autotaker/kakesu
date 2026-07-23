---
kind: schema
title: MVP Wire Contracts
---

# MVP Wire Contracts

## 正本と適用範囲

`schemas/draft-v0/common/message-envelope.schema.json` r1をwire構造の正本とする。Go、Python、Rustのadapterは同じroot共有fixtureとlocal設定を読み、Schemaで表し切れないcross-record制約を境界でfail closedに検査する。

この契約はenvelopeの妥当性、schema artifactへの参照、local endpointの解決までを扱う。UDS listener、socket bind/connect、ACK、retry、durable Inbox/Outbox、およびpayloadを参照先JSON Schemaで意味検証する責務は含まない。

## Envelope不変条件

- decoderはfield集合を厳密に扱い、unknown fieldとrequired fieldの欠落を拒否する。
- `from_component`と`to_component`は異なる。
- `causation_message_id`はnullableだがoptionalではない。因果messageがない場合も明示的な`null`を必要とし、field欠落と同一視しない。
- `occurred_at`はRFC3339として妥当で、UTC offsetが0であるだけでなく、wire表現が末尾`Z`でなければならない。
- `payload`はJSON objectに限る。

## Schema catalog参照

payload schemaはcatalog内でSchemaの`$id`と`x-schema-revision`の組により正確に1 artifactへ解決する。0件または複数件の解決は拒否する。解決したファイルの読み取りバイト列をそのままSHA-256し、fixtureの`sha256:`に続くlowercase 64 hexと完全一致するときだけ参照を受理する。

これはJSONのparse後の値ではなく、配布artifactのidentityに結合する検査である。したがって空白、改行、整形だけの変更もdigest mismatchとして検出される。

## 共有fixtureとlocal設定

相互運用性は言語ごとの複製ではなく、root共有のfixture matrixで固定する。正常例を1件とし、same route、非UTC表記、schema参照不整合、非object payloadをそれぞれ一件だけ破る負例として全adapterが同じaccept/reject結果を持つ。

local設定はproject root相対のschema catalog rootと、`core-runtime`、`memory-service`、`governance-service`の正確なcomponent集合を持つ。各UDS endpointは相対・非空・parent traversalなし・相互非重複とし、credential、host/port、秘密値、listener lifecycleを持ち込まない。

## 運用上の帰結

Schema artifactを将来変更するTaskはrevisionと`$id`の規則に従い、参照fixtureのrevisionおよびraw-byte digestを同じ変更単位で更新する。Schemaだけ、またはdigestだけの更新はadapterに拒否される。

## 関連

- [Schema artifact digest drift](../case-patterns/schema-artifact-digest-drift.md)
- [TASK-0006 handover](../../../tasks/TASK-0006-mvp-wire-contracts/HANDOVER.md)
