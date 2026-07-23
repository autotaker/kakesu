---
kind: case-pattern
title: Schema Artifact Digest Drift
---

# Schema Artifact Digest Drift

## 発生条件

複数consumerがSchemaのIDやrevisionだけで参照を結び、実際に配布されるSchema artifactの内容変更を独立に検出できない。

## 対策

catalogでは`$id`とrevisionの組を正確に1 artifactへ解決し、そのファイルの未加工バイト列をSHA-256して参照側のdigestと完全比較する。digestには`sha256:` prefixとlowercase hexを含む固定表現を使う。複数言語のconsumerは同じ共有fixtureと、一件につき一制約を破る負例を読む。

Schemaの変更ではrevision、`$id`、そのSchemaを参照するfixtureのdigestを同じ変更単位で更新する。1-byte mutationを加えたcatalog artifactをfixture更新なしで検査し、すべてのconsumerがdigest mismatchとして拒否することを確認する。

## 反例

Schema JSONをparse後に再serializeまたはcanonicalizeしてhashすると、空白、改行、整形の変更を見逃す。raw-byte digestはそれらもartifact変更として検出するため、canonicalized content digestとは交換できない。

UTC offsetが0であることだけを検査すると`+00:00`を受理し、末尾`Z`というwire表現の契約を失う。同様にnullable fieldをoptional型へ直接decodeすると、field欠落と明示nullを区別できない。表現まで契約に含む場合は、元のfield存在とwire文字列を別に検査する。

## 適用限界

raw-byte digestはartifactの改変検出であり、payloadが参照先Schemaの意味制約を満たすことを証明しない。payload意味検証、配送、ACK、retry、durabilityは別の責務である。

## 関連

- [MVP wire contracts](../schemas/mvp-wire-contracts.md)
- [TASK-0006 handover](../../../tasks/TASK-0006-mvp-wire-contracts/HANDOVER.md)
