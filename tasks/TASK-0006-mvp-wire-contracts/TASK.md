---
task_id: "TASK-0006"
title: "MVP通信契約とローカル設定を確立する"
status: complete
created_at: "2026-07-15"
---

# TASK-0006 MVP通信契約とローカル設定を確立する

## 目的

Go Core、Python Memory、Rust Governanceが同一のMVP wire envelope、schema参照、ローカル接続設定、および共有fixtureを用いて、Plane間メッセージを相互に検証できる状態にする。

## 背景

3プロセスの実装では、UDSは配送手段でありメッセージの正本ではない。`docs/00-kakesu.md`と`docs/13-technology-stack.md`は正規JSON Schema、schema ID/revision/digestの記録を要求するが、現行の各言語スタブはenvelope検査を個別に持つだけで、相互互換のfixtureと起動設定がない。

## スコープ

### 対象

- `schemas/draft-v0/common`のEnvelopeとprimitiveを基準に、MVPで使用するcomponent名、UDS endpoint、schema catalog参照、時刻表現、JSONエンコード規約を固定する。
- Go/Python/Rustで共通fixture（正常、route不正、非UTC timestamp、schema参照不正、payload非object）を読み、各実装の境界検査を一致させる。
- ローカル開発用設定と、fixtureの生成・検証を実行するテスト導線を追加する。

### 対象外

- durable Inbox/Outbox、再試行、ACK、Task状態、Workspace隔離、実サービスのUDS listener。
- Schemaの意味的payload追加・既存Schemaのrevision変更（Task-0007以降で扱う）。
- TCP/HTTP API、遠隔配送、コード生成基盤。

## 受け入れ条件

- [ ] 3言語が同じ有効fixtureを受理し、同じ4種の不正fixtureを拒否する自動テストがある。
- [ ] 各fixtureは`message_id`、異なる`from_component`/`to_component`、correlation、UTC timestamp、schema ID/revision/digest、object payloadを明示する。
- [ ] 起動設定から各PlaneのUDS endpointとschema catalog rootを解決でき、デフォルトはローカル専用である。
- [ ] Schema変更時にfixtureのschema参照とdigestを更新しない限り検査が失敗する。

## 検討すべき設計観点

- Canonical Schemaを唯一のwire契約とし、各言語の内部型・エラー型はadapterに限定する。
- JSON object payload、UTC RFC3339時刻、明示的nullの`causation_message_id`を3言語で同値に扱う。
- ローカル設定に秘密情報を置かず、endpointはPlaneごとに衝突しない。

## 完成の定義

- [ ] 受け入れ条件を満たしている。
- [ ] 必要な実装、テスト、文書が完成している。
- [ ] 独立レビューと`make check`がPASSしている。
- [ ] マージ後QAが完了している。
- [ ] HANDOVERがWiki Agentに取り込みされている。

## 関連コンテキスト

### 意味 Wiki

- `docs/00-kakesu.md` §3.1: JSON Schema、永続キュー、UDS、少なくとも一回配送。
- `docs/13-technology-stack.md` §7: wire型と内部ドメイン型の分離、schema参照の永続化。
- `schemas/AGENTS.md`: Schema metadata・strict payload・負例の規約。

### 判断

- Decision: `schemas/draft-v0/common/message-envelope.schema.json`をwireの正本にし、fixtureは言語ごとの複製ではなく共有JSONとして管理する。

### 適用しなかった重要な判断

- gRPC/Protobufへの置換は、canonical JSON Schemaを二重管理にするためMVPでは採用しない。
