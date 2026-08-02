---
kind: script
title: Task Delivery
---

# Task Delivery

## Trigger

バックログから実施するTaskを選び、main AgentがPlanner Agentをアサインする。

## 標準進行

1. Mainが目的、非対象、安定参照、依存状態、許可パス、preflight、未決事項を含むplanning input packetを固定する。
2. Planner AgentとQA Agentは同じpacketだけを入力に、AC-IDで対応するPLANと実装前QA計画を独立に作る。PLANは設計、QA_PLANは観測を所有する。
3. main AgentがPLANとQA_PLANを承認する。依存がreadyになる前後で設計、scope、期待結果が変わる場合はreconciliationと再承認を行う。
4. main AgentがDEV、Reviewer、QA、ブランチ、worktreeを割り当てる。
5. QA_PLANはDEV開始前に各caseへ`evidence-review`、`focused-rerun`、`live-e2e`のいずれかと理由を割り当てる。
6. DEV Agentが実装、テスト、文書、candidate-boundの`make check`を一度完了し、HANDOVERへcandidateと実行したcommand/結果を残す。
7. Reviewer AgentとQA Agentは同じcandidate commit/treeを独立かつ並行に評価する。互いのPASSを開始条件にしない。
8. main Agentは両評価、candidate treeとmerge treeの同一性、必要な再評価を確認してから`--no-ff`で`main`へマージする。
9. 環境依存caseだけをマージ後に確認する。HANDOVERをWiki Agentがingestし、Taskを閉じる。

## QAの選択

`evidence-review`では、QAがHANDOVERのcandidate、case ID、command/結果を突合し、candidate差分、negative case、testの弱体化を独立に監査する。DEVの実行主張だけではPASSにしない。環境、cache、exit詳細、artifact digest、未実施理由は該当する場合だけ記録する。

高リスクcaseは、受け入れ真実をhermetic、deterministic、boundedなfixtureで完全に再現できる場合だけ`focused-rerun`を選べる。実OSの権限/auth、実配置、外部作用、restart/rollback/cleanup、環境固有integrationに依存するcaseは`live-e2e`とする。実環境または安全なcleanupがない場合はblockedのままとし、別modeのPASSで代替しない。

## 計画の境界

依存待ちはactive planningと分ける。active planningが10分を超える場合は、境界不明、依存不安定、API不明、資料不足、期待不一致、tool/permissionのいずれかを記録し、文章の磨き込みを続けない。DEV前のpreflightでは完了checker、権限、依存、生成物、worktree、Lapログの書込・Schema・repository annotationを確認する。

## 分岐

- QA計画の誤りはQAへ戻す。
- 実装不具合はDEVへ戻す。
- 要件または設計の不足はPLANへ戻す。
- candidateが変わった場合、影響するfocused testを再実行する。影響を限定できない場合だけfull rerunを行う。根拠チェックリストによる旧QA結果の転用はしない。
- 重大なマージ後不具合はrevertし、限定的不具合はバグTask化できる。

## 効率化

効率化はPLAN承認、独立Reviewer、独立QAを省略してはならない。各gateはlock、scope検査、全体validation、commitを一回のtransactionとして扱い、必要十分な出力を一度で明示する。case単位で証跡監査、限定再実行、実環境E2Eを選び、同じ目的のcheck再実行や形式だけの証跡を増やさない。

予測可能なnetworkまたはpermissionの失敗は、環境要件と再実行条件を先に切り分ける。同一の失敗を条件を変えずに反復せず、実行不能の扱いはQA FAIL attributionに従う。

独立した文書lintは、既存の検査内容と品質基準を変えず、固定されたargv配列をshellなしのrunnerから順に一回ずつ実行する。一件が失敗又は起動不能でも残りを省略せず、各検査の標準出力・標準エラーを即時に表示してから失敗を集約する。`make lint-docs`はこの診断集約の入口とし、利用者が一回の起動で全指摘を修正できるようにする。並列化、retry、cache、自動修正、入力からのcommand組立ては導入しない。

## 終了条件

受け入れ条件、レビュー、QA、HANDOVER、Wiki ingestが完了し、発見事項が解消または追跡されている。

## 関連

- [QA FAIL attribution](../case-patterns/qa-fail-attribution.md)
- [Planning input packet](../concepts/planning-input-packet.md)
- [Streamlined task gates](../../decisions/DECISION-0007-streamlined-task-gates.md)
- [Lean planning contract](../../decisions/DECISION-0006-lean-planning-contract.md)
- [TASK-0005 HANDOVER](../../../tasks/TASK-0005-main-agent-efficient-delivery/HANDOVER.md)
