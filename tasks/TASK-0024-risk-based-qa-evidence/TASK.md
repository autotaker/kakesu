---
task_id: "TASK-0024"
title: "QAを証跡監査とリスク別再実行へ変更する"
status: done
created_at: "2026-07-20"
---

# TASK-0024 QAを証跡監査とリスク別再実行へ変更する

## 目的

TASK-first QA_PLANの独立性を維持したまま、DEV後QAを証跡監査、限定再実行、実環境E2Eへリスク別に分け、REVIEWとQAを同一candidate commitから並行実行できる開発プロセスへ変更する。低リスクなREVIEW修正では、条件付きのQA結果引き継ぎにより重複再実行を省略できるようにする。

## 背景

Lap30実績ではQA/QA_PLANのFAIL 10件中8件が実装前のplanning defect、1件が実装・証跡defect、1件がenvironment issueだった。独立した受け入れ設計は有効だが、DEV後に同じ自動テストを独立して全面再実行する限界効用は低い。実sudo/PAM、秘密情報、IPC、並行性など独立実行が必要な境界を残しつつ、candidate commitへ結び付いたDEV証跡の独立監査を標準にする。

## スコープ

### 対象

- `QA_PLAN.md`の各ケースへ`evidence-review | focused-rerun | live-e2e`をDEV前に割り当てる契約。
- DEVがcandidate commitに結び付いたコマンド、test、環境、cache無効化、結果、digestを運用証跡へ記録する契約。
- REVIEWとQAが同じcandidate commitを独立かつ並行に評価する手順。
- REVIEW修正後の`qa_carry_forward`、限定再実行、完全再実行の判定条件とMain所有判断。
- merge treeと承認candidate treeが同じ場合の結果引き継ぎと、環境依存項目だけを対象にしたマージ後確認。
- `AGENTS.md`、開発プロセス/QA文書、Task証跡template、`run-efficient-task-delivery`、`run-lap30`の整合。

### 対象外

- 製品コード、製品test、runtime/build設定、製品Schema、製品依存、製品挙動。
- QA_PLANの独立作成、DEV/Reviewer/QAの役割分離、Main所有Git、FAIL分類の緩和。
- 高リスク境界またはQA自身がFAILしたケースの無条件な再QA省略。
- Lap30 event Schemaの変更。

## 受け入れ条件

- [x] QA_PLANはTASKだけから独立作成し、各ケースへ3つの実施modeのいずれかを理由付きで指定する。曖昧または高リスクなケースは`live-e2e`または`focused-rerun`へfail-closedする。
- [x] DEV証跡はcase ID、candidate commit/tree、command/test、環境/fixture、cache条件、exit、artifact digest、未実施理由を持ち、QAはtestの弱体化、negative case、commit binding、証跡完全性を独立監査する。
- [x] REVIEWとQAは同じcandidate commitから互いのPASSを前提にせず並行でき、各結果が対象commitを明記する。
- [x] REVIEW修正後のQA省略は非挙動または明示した低リスク条件をすべて満たす場合だけMainが`qa_carry_forward`として旧新commit、差分、影響case、再実行証拠、理由を記録する。QA FAIL対象、受け入れ/QA_PLAN変更、認証認可、秘密、sudo/PAM、IPC/Schema/設定/依存、並行性/lifecycle/persistence/error/fail-closed、test削除/弱体化、影響不明では省略できない。
- [x] merge treeが承認candidate treeと同一で環境依存caseがない場合はマージ後全面QAを省略できる。環境依存、install/deploy/config生成、実権限、外部作用、rollbackはマージ後確認を維持する。
- [x] templateとスキルが同じmode、証跡項目、carry-forward条件、FAIL分類を使用し、既存Taskの証跡形式を破壊しない。

## 検討すべき設計観点

- QAの独立性を「独立再実行回数」ではなく「期待値、test妥当性、証跡判断の独立性」として保持する。
- DEVが作ったtestと証跡を自己承認させず、Reviewerは実装/test品質、QAは受け入れ対応/証跡を別観点で判定する。
- 低リスク分類を便利なラベルにせず、列挙条件と禁止条件、Mainの記録を必要とする。

## 完成の定義

- [x] 受け入れ条件を満たしている。
- [x] PLANとTASK-first QA_PLANを独立作成し、Mainが承認している。
- [x] 独立した計画レビューが全受け入れ条件と回帰条件をPASSしている。
- [x] 影響する文書、template、スキル検証、`make check`、`make work-check`がPASSしている。
- [x] 製品DEV、製品REVIEW、製品QAのPASSを作っていない。

## 関連コンテキスト

## 運用上のブロッカー

- 製品変更と受け入れ検証は完了し、merge commit `9b7e8beb68c91204f5529fa297fb3385d5f05350`のtreeは承認候補treeと一致した。
- 既存`task-check`が安全契約変更にも製品`REVIEW_RESULT.md`/`QA_RESULT.md` PASS、完全HANDOVER、Wiki receiptを一律要求し、新しい軽量経路でDoneにできない。架空のPASSを作らず、検査器の経路分類対応を後続Taskへ分離する。

### 意味 Wiki

- `docs/development/development-process.md`、`docs/development/qa.md`、`docs/development/agent-roles.md`。

### 判断

- TASK-first QA_PLANは維持し、DEV後QAの既定をcandidate-bound evidence reviewへ変更する。独立実行はcase単位のriskで選ぶ。
- QA carry-forwardはQAの削除ではなく、旧判定を新commitへ移す独立したMain判断として記録する。

### 適用しなかった重要な判断

- なし
