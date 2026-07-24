# Agent責務

## 固定ロール

`.codex/agents/*.toml`をモデル、推論 effort、サンドボックス意図の正本とする。

| 責務 | `agent_type` | モデル / effort | 主な境界 |
|---|---|---|---|
| main | `main` | Sol / high | 承認、FAIL分類、証跡トランザクション、PR/merge判断 |
| PLAN | `planner` | Terra / medium | TASK packetからPLANを作成し、実装しない |
| DEV（低リスク） | `dev-luna` | Luna / xhigh | 承認済み`luna-xhigh` PLANだけを実装 |
| DEV（高リスク/不明） | `dev-sol` | Sol / high | 承認済み`sol-high` PLANだけを実装 |
| REVIEW | `reviewer` | Terra / medium | 同一composite 案の独立レビュー |
| QA | `qa` | Terra / medium | TASK-first計画と同一composite 案の独立QA |
| 限定調査 | `explorer` | Luna / medium | 一問、読み取り専用、再委譲禁止 |

異種ロールを起動するときは`fork_turns="none"`を指定する。起動後に期待するモデル/effortとobserved値を照合し、不一致、`agent_type`欠落、内部生成 Agent利用不能では停止して証跡化する。別launcherでロール契約を迂回しない。限定調査だけは一問専用の`make explorer-agent QUESTION=...`を使用できる。

## main Agent

Mainは次を所有し、子へ移譲しない。

- planning 入力 packet、preflight、依存`ready` 照合
- PLAN/QA_PLAN承認、ブランチとワークツリー割当
- FAIL分類、差し戻し、revert判断
- main管理証跡のaction、ロック、stage、コミット、push
- composite 案確認、`ready` PR、merge コミット auto-mergeの有効化
- merge tree比較、qa_carry_forwardまたはrerun、post-merge確認

## Planner / DEV / レビュアー / QA

PlannerはTASK packetからAC-IDに対応する設計、変更パス、順序、failure handling、見積りを作る。QAはPLANを入力にせず同じpacketから観測計画を作る。

DEVは割当済みsparse Task ワークツリーだけで実装・検証する。main管理証跡では担当Taskの`HANDOVER.md`だけを編集できる。stage、コミット、merge、push、`.git`書込みはしない。

レビュアーとQAはDEVと兼任せず、同じコード コミット/tree/managed-path ダイジェストとブートストラップ 証跡 コミット/ダイジェストの組から独立・並行に開始する。相互のPASSを開始条件にしない。軽微修正を自ら行う例外はAGENTS.mdの規則に従う。

## Explorer

Explorerは一度に一件の限定質問だけを読み取り専用で調査する。ファイル編集、Git書込み、スコープ拡大、再委譲、方針決定を行わず、短い根拠要約とファイル参照を返す。`max_depth=0`、`max_threads=1`を維持する。

## main管理証跡

`backlog.yaml`、`tasks/**`、`wiki/**`、`lap30/**`、運用インデックスはmain ワークツリーを正本とし、コード用Task ワークツリーからsparse-checkoutで除外する。子Agentは明示されたmain ルートで許可された証跡だけを編集する。

Mainは編集完了後に次を実行する。

```sh
make evidence-commit TASK=TASK-0001 ACTION=plan
make evidence-commit TASK=TASK-0001 ACTION=review
make evidence-commit TASK=TASK-0001 ACTION=qa-result
```

トランザクションは明示main ルート、共通ロック、action allowlist、軽量検査、完全なstaging set、コミット、pushを所有する。non-fast-forwardではfetch/rebase/revalidate/pushを最大2回再試行し、スコープ外変更、競合、ロック失敗、再試行枯渇でfail-closedに停止する。shared パスは専用actionだけが変更できる。

Wiki AgentはHANDOVERから再利用可能な知識を抽出してWiki パスだけを編集し、公開トランザクションをMainへ返す。認証済みローカル Codexだけを使い、ActionsへAI認証情報を置かない。

## 兼任と完了

- DEVとレビュアー、DEVとQAの兼任を禁止する。
- Mainは例外判断と統合を担い、通常実装を兼任しない。
- 両PASS後だけMainが`make task-pr TASK=...`を実行する。
- レビュアー/QA修正後のcarry-forwardはQAガイドラインのCF-1〜CF-7を全て満たす場合だけMainが選ぶ。
- merge後は承認案とのtree同一性を確認し、環境依存ケースをケース単位で再確認する。
