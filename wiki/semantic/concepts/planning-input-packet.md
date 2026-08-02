---
kind: concept
title: Planning Input Packet
---

# Planning Input Packet

## 問い

PlannerとQAが、重複したTask本文や相互の成果物ではなく、同じ安定した入力から独立に計画するにはどうするか。

## 契約

MainがTask内のplanning input packetを所有し、目的、対象と非対象、安定した参照、依存状態、許可パス、完了経路preflight、未決事項、安定したAC-IDを一度だけ固定する。PlannerとQAはこのpacketだけを入力とし、PLANとQA_PLANへ条件本文を複製しない。

PLANはAC-IDごとの設計判断、変更範囲、順序、失敗時、見積りを所有する。QA_PLANは同じAC-IDごとの観測方法と実施modeを所有する。これにより、設計と受け入れ観測の正本を分けたまま対応関係を追跡できる。

## 依存とpreflight

依存が未readyの間は、依存なしに固定できる計画だけを進め、待機時間をactive planningとして扱わない。依存ready後に参照、設計、scope、期待結果、QA観測へ差分が生じた場合、Mainはdependency-ready reconciliationを記録し、必要なPLAN/QA_PLANの再承認を得る。

DEV前に完了checker、権限、依存、生成物、作業tree、Lapログの書込・Schema・repository annotationをpreflightする。未解決のpreflightは開始済み作業として隠さず、blocked又はnot-startedとして扱う。active planningが10分を超える場合は、境界不明、依存不安定、API不明、資料不足、期待不一致、tool/permissionの原因を記録する。

## 適用限界

packetは新しい独立証跡や受け入れ条件の第二正本ではない。実装、依存、環境、又は要求の変化を自動的に承認するものでもない。既存Lap Schema/JSONLと公開済みTask証跡を遡及変更しない。

## 関連

- [Task delivery](../scripts/task-delivery.md)
- [Development task](development-task.md)
- [Lean planning contract](../../decisions/DECISION-0006-lean-planning-contract.md)
- [TASK-0030](../../../tasks/TASK-0030-lean-planning-contract/TASK.md)
