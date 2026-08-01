---
task_id: "TASK-0054"
status: complete
completed_at: "2026-08-01T15:43:15Z"
candidate_commit: "2cee2f2120b689ba511d2a710a9c6e534ecb49d4"
---

# TASK-0054 HANDOVER

## 成果

- planning transactionがGitを変更する前に、既存DEV selection契約を同じ正本関数で検証するようにした。
- TASK-0053と同じprofile三項目欠落をplanning時点で拒否し、main/Task branch/worktree/index/dirty入力/remoteを不変に保つintegration testを追加した。
- 有効planning fixtureを共通helperで整合し、既存one-commit/no-ff lifecycleを維持した。

## candidate-bound DEV証跡

| コマンド/テスト | 結果 |
|---|---|---|
| candidate launcherのroot `make check`（固定candidateで一回） | `PASS` |
| `node --test scripts/task/unified-lifecycle.test.mjs` | `PASS`（29/29、24.47s） |
| targeted Q-01/candidate diagnostics | `PASS`（5/5） |
| `git diff --check` | `PASS` |

## 主要な変更

- `validatePlanningState`でPLAN/QA role承認を確認した直後、stage/commit/push/Task branch fast-forward前に`validateDevSelection(plan)`を一回呼ぶ。
- profile集合、reason、risk signals、promotionのlogicは複製せず、completion側validationも変更していない。
- lifecycle testの有効PLAN fieldsを`approvedPlan(task)`へ集約し、三DEV fieldを持たないplanningの原子的failure testを一件だけ追加した。

## 検証結果

- candidate `2cee2f2120b689ba511d2a710a9c6e534ecb49d4`、base `2f09c1330cc53912eee463134061c25df4bb0b07`を固定した。製品差分は許可2 files、追加40行・削除4行である。
- exact candidateでcandidate launcherのroot `make check`がPASSした。
- full lifecycle testは初回、Task worktreeに`node_modules`がなくtemp fixtureの`ajv`解決だけで2件失敗した。lockfileから依存を配置し、製品bytes不変の同じworking treeで29/29 PASSしたため`environment_issue`と分類した。
- 新integrationは`DEV_PROFILE_UNKNOWN`に加えてmain/Task branch/worktree HEAD、index、status、PLAN bytes、origin/mainの不変を直接assertする。

## 判断・既知の制約

- 新しいgate、field、profile規則、counter seam又はfield別integration matrixは追加しない。既存validationの実行phaseだけを前倒しした。
- completion側のvalidationは、外部又は旧経路から不正PLANが入った場合の防御として維持する。
- network、実secret、外部作用、live-e2eはなく、temporary Git fixtureだけで受け入れ真実を再現する。
