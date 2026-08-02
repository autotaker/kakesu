---
task_id: "TASK-0074"
status: complete
completed_at: "2026-08-02T10:03:55Z"
candidate_commit: "563f52769cfdc1349271a60262c7c340eb5998ae"
safety_checks:
  process_tests: pass
  contract_scope: pass
  docs_lint: pass
  make_check: pass
safety_checked_at: "2026-08-02T10:03:55Z"
---

# TASK-0074 HANDOVER

## 成果

- push承認を、完全一致repositoryへ束縛した「次の`git-receive-pack`一回」へ簡素化した。
- ref/SHA/manifest/本文一致を認可根拠から外し、同一repository内の内容差替えリスクを明示的に受容した。
- 通常のGit read、GitHub REST、OpenAIを薄い認証差し替えstream転送とし、旧意味検査・全量buffer・重複層を削除対象に固定した。

## candidate-bound DEV証跡

| コマンド/テスト | 結果 |
|---|---|---|
| `make check`（candidate transactionで一回） | `PASS`（candidate `563f52769cfdc1349271a60262c7c340eb5998ae`） |

## 主要な変更

- grantはAgent/UID、workspace、repository、短TTL、一回消費、revokeへ束縛し、上流push試行前に原子的に消費する。
- 別repository、別主体/workspace、再使用、期限後、REST転用は拒否し、repository限定GitHub App write権限を上流境界にする。
- 承認UIは「このrepositoryへの次のpush一回」を主文言とし、branch/commit/ref/SHA等は参考表示だけにする。
- TASK-0070を廃止済みとし、TASK-0071〜0073をrepository単位モデルへの移行対象にした。次の単一製品Taskを実VPS縦断E2Eへ優先する。

## 検証結果

- `make check`: `PASS`（candidate transaction）
- `make lint-docs`: `PASS`（candidate transaction内）
- process tests: `PASS`（candidate transaction内）
- contract scope: `PASS`（MainがTASK ACとcandidateの設計書1パス差分を対照）
- `git diff --check`: `PASS`

## 判断・既知の制約

- 同一repository内で承認時の参考表示と異なる内容を一回pushできる残余リスクは受容する。
- 製品コードと実VPSは本Taskで変更せず、旧実装の削除、薄いproxy、repository one-shot grant、縦断E2Eを次の単一製品Taskで扱う。
- 製品実装がないためREVIEW_RESULT/QA_RESULTのPASSは生成せずpendingのままにする。
