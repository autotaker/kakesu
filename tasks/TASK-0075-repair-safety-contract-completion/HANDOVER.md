---
task_id: "TASK-0075"
status: complete
completed_at: "2026-08-02T09:57:03Z"
candidate_commit: "97d29e98161c1319d2410b3bcdce81afda37f92b"
---

# TASK-0075 HANDOVER

## 成果

- 安全契約の完了を製品用REVIEW/QA PASSから分離し、Main承認済みQA_PLANと既存4 safety checksで完了できるようにした。
- HANDOVERのcandidateだけを正本にし、merge/tree/digestの重複転記を削除した。
- `merge-base..candidate`の1コミット差分とno-ff第2親から、main先行時も許可パスを検証できるようにした。

## candidate-bound DEV証跡

| コマンド/テスト | 結果 |
|---|---|---|
| `make check`（candidate transactionで一回） | `PASS`（candidate `97d29e98161c1319d2410b3bcdce81afda37f92b`） |

## 主要な変更

- `completionGate`のproduct条件を維持したまま、安全契約専用の最小証跡分岐を追加した。
- safety done checkerはmerge中の`MERGE_HEAD`又はmain上の一意な2親mergeからcandidateを導出する。
- mainがcandidate固定後に先行したfixture、pending製品結果、改ざん・rename/copy・空差分・非no-ffの回帰検査を追加した。
- 公開済みlegacy安全契約は旧statusと既存merged commitの組合せだけを互換入力とし、新Taskによるlegacy偽装を拒否する。

## 検証結果

- `make check`: `PASS`（candidate transaction）
- `node --test scripts/task/development-process.test.mjs scripts/task/unified-lifecycle.test.mjs`: `PASS`（DEVおよびMain focused rerun）
- `git diff --check`: `PASS`
- candidate差分: 許可された5パスのみ、206追加/62削除

## 判断・既知の制約

- 既存Taskに残る`merged_commit`、tree、digestは遡及変更せず、未使用の互換入力として許容する。
- TASK-0074は本candidateの完了後、新しい安全契約経路の最初の実運用受け入れとして統合する。
