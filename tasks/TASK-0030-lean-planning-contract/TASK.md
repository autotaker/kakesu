---
task_id: "TASK-0030"
title: "PLAN入力契約とLap計測を軽量化する"
status: plan
created_at: "2026-07-20"
---

# TASK-0030 PLAN入力契約とLap計測を軽量化する

## 目的

PLANを受け入れ条件の再作文ではなく未決の設計判断に集中させ、依存待ちやpreflight不足を計画時間へ混入させない。

## 背景

TASK-0024〜0029ではTASK・PLAN・QA_PLANが計1,665行となり、PLAN作成は最大35分42秒だった。遅延の主因は、同じ条件の三重記述、依存merge前に不安定な詳細を確定しようとしたこと、完了checker・権限・生成物確認の後出し、active planningとdependency waitの混同である。TASK-0024〜0029のLapイベントがゼロだったため、計測開始自体のpreflightも不足していた。

## スコープ

- MainがPlanner/QAへ渡すplanning input packet。
- TASKの条件IDをPLANとQA_PLANが参照する単一対応表。
- 依存非依存の事前計画とdependency-ready reconciliationの分離。
- PLAN遅延原因の分類、完了経路preflight、Lapログ開始確認。
- 関連する製品側の開発文書、Task template、delivery skillの簡素化。

対象外: 製品コード、テスト、runtime/build設定、Schema、製品依存、生成製品入力/成果物、外部観測可能な製品挙動、checker実装。

## 受け入れ条件

- AC-1: Planning input packetが目的・非対象・条件ID・安定した参照・依存状態・許可パス・preflight結果・未決事項を一度だけ定義する。
- AC-2: PLANはTASK本文を複製せず、設計差分、変更範囲、順序、失敗時、見積りだけを記録し、QA_PLANは条件IDごとの観測方法を所有する。
- AC-3: 依存待ちはactive planningから分離され、依存前に固定できない値はdependency-ready reconciliationで差分承認する。
- AC-4: 10分を超えるactive planningは、境界不明・依存不安定・API不明・資料不足・期待不一致・tool/permissionのいずれかを記録し、文章の磨き込みを続けない。
- AC-5: DEV前に完了checker、権限、依存、生成物、作業tree、Lapログの書込・Schema・repository annotationを確認する。
- AC-6: delivery skillは開発文書を再掲せず、実行時に必要な判断と参照先へ縮約される。
- AC-7: 既存Lap Schema/JSONL、製品ゲート、ロール分離、Main所有Git、安全境界は弱めない。

運用リポジトリの`run-lap30`縮約は、このTaskの製品merge treeでは完了を検証できないため対象外とする。外部ガバナンスcommitを完了条件へ接続するchecker変更後に別Taskで行う。

## 完成の定義

- [x] AC-1〜AC-7を満たす。
- [x] 独立したTASK-first QA_PLANと計画レビューがPASSする。
- [x] 安全契約用の対象検査と`make check`がPASSする。
- [x] no-ff merge後に案treeとmerge treeが一致する。

## Blocker

- `make check`が文書変更に伴う用語集・索引の再生成を要求する一方、現行の安全契約checkerは生成索引を許可しない。`requirement_gap`としてTASK-0031で完了経路を先に修正し、その後PLAN/QA_PLANをreconciliationする。
- 解消: TASK-0031（merge `b60e691182127de73fff36660f7d3cfe26bc01e9`）でv2 preflightと候補差分拘束を導入した。最新mainへの載せ直し後、決定的生成で`docs/glossary.yml`だけが実差分となり、索引は内容不変だった。
