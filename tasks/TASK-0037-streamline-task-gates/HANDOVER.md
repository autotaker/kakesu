---
task_id: "TASK-0037"
status: complete
completed_at: "2026-08-01T04:32:12Z"
candidate_commit: "90b023dc75b18362da98d4481b10857eeebb0a97"
---

# TASK-0037 HANDOVER

## 成果

- legacy `qa→done` cleanupからWiki receipt探索とWiki Agent自動実行を削除した。
- 未使用の行数/ファイル数ポイント算術とそのtestを削除した。
- PLAN templateを変更pathと内容だけの表へ縮小した。

## candidate-bound DEV証跡

| コマンド/テスト | 結果 |
|---|---|
| `node --test scripts/task/unified-lifecycle.test.mjs` | `PASS (28/28)` |
| `node --test scripts/task/development-process.test.mjs` | `PASS (66/66)` |
| `make check`（`make candidate-commit`がcandidate固定直前に一回実行） | `PASS` |
| `git diff --check` | `PASS` |

## 主要な変更

- Wiki agentが存在せずreceiptもないfixtureで、legacy cleanupがdoneまで完了しreceiptを生成しないことを固定した。
- backlogの`estimate_points`とviewer用途は維持し、未使用の`estimatePoints` helperだけを除去した。
- 変更は承認済み5 path、追加72行・削除46行に限定した。

## 判断・既知の制約

- `agent-routing`のcanonical route override拒否はcaller入力の安全境界なので変更していない。
- 起動後のobserved model/effort差をwarningとして扱うAGENTS/docs契約、独立REVIEW/QA、candidate固定、原子的completionは現mainのbaselineを維持する。
- 外部サービス、実OS権限、配置、restartを変更しないためlive E2Eはない。
