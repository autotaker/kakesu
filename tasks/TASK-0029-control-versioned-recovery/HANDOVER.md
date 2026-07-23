---
task_id: "TASK-0029"
status: complete
completed_at: "2026-07-20"
safety_checks:
  process_tests: pass
  contract_scope: pass
  docs_lint: pass
  make_check: pass
safety_checked_at: "2026-07-20"
safety_check_digest: "not-applicable-product-task"
safety_candidate_tree: "5ff201700eb58aedd8635a5456e5a741ae4f5b22"
safety_merge_tree: "5ff201700eb58aedd8635a5456e5a741ae4f5b22"
---

# TASK-0029 HANDOVER

## 成果

- SQLite migration v3、expected-version contract/progress CAS、immutable history、schema参照付きevent、close/reopen recoveryを実装した。
- 同expected versionの並行更新は一方だけ成功し、故意SQL失敗とmigration失敗は部分状態やversion gapを残さない。

## candidate-bound DEV証跡

- `candidate_commit`: `864f455b563a6fffb043ed297d5cb10e3849b988`
- `candidate_tree`: `5ff201700eb58aedd8635a5456e5a741ae4f5b22`

| ケース ID | コマンド/テスト | 環境/フィクスチャ | cache条件 | exit | 成果物 ダイジェスト | 未実施理由 |
|---|---|---|---:|---:|---|---|
| Q29-01/02/05 | `go test -count=1 ./internal/control` | temporary file-backed SQLite、contract/progress history/event/schema refs | isolated `/tmp` Go cache | 0 | changed-file SHA-256 below | なし |
| Q29-03/04/06 | `go test -count=20 ./internal/control` / `go test -race -count=1 ./internal/control` | deterministic conflict barrier、fault injection、post-failure readback | isolated `/tmp` Go cache | 0 | candidate tree `5ff2017...` | なし |
| Q29-07/08 | recovery/migration integration tests in `./internal/control` | close/reopen same DB path、corruption、v2→v3 rollback/cascade | temporary dirs | 0 | candidate tree `5ff2017...` | なし |
| Q29-09 | `make check`; `git diff --check` | repository candidate worktree | existing dependency caches | 0 | diff SHA-256 `39f2ccfe174e6b0ebfcccc930b6f90ef1d13655731702aba1aa1fb4e01da6d74` | なし |

- QAへ渡すネガティブ検出証拠はversion/schema/payload拒否、Task/Agent watermark後退と未来Task sequence拒否、全transaction stageのSQL failure、並行conflict、current/history/event corruption、migration rollbackを含む。テスト削除なし。差分SHA-256: `39f2ccfe174e6b0ebfcccc930b6f90ef1d13655731702aba1aa1fb4e01da6d74`

## 主要な変更

- `core/internal/control/store.go`: migration v3とv2 data保存/table rebuild。
- `core/internal/control/versioned.go`: contract/progress CAS、atomic history/current/event。
- `core/internal/control/recovery.go`: durable full read modelとtyped corruption fail-fast。
- production 418追加行、test 455追加行。承認済み425行stop内。

## 検証結果

- focused PASS、`-count=20` PASS、race PASS、`make check` PASS、`git diff --check` PASS。

安全契約変更では`safety_checks`を`process_tests`、`contract_scope`、`docs_lint`、`make_check`の4項目だけとし、すべて`pass`を記録する。`safety_check_digest`は案 tree、merge tree、上記順の検査名と結果を`key=value`の改行区切りで正規化し、末尾改行を含めたSHA-256とする。第2親の案 treeとmerge treeもフロントマターへ記録する。製品用のREVIEW/QA PASS、製品用の完了HANDOVER、Wiki取込記録を代用証跡として作成しない。

## 判断

- ReviewerとQAが同一candidate commit/treeを独立評価する。
- 選択: `not-applicable | qa_carry_forward | focused-rerun | full-rerun`
- Main判断の旧新コミット/tree、全差分とダイジェスト、影響ケース集合、レビュアー/`make check`証拠、理由: TODO
- carry-forward時の`QA_RESULT.md` `CF-1`から`CF-7`: `not-applicable | complete | incomplete`
- 影響QAケース集合が空でない場合の再実行証拠: TODO
- `merge_commit`: `a908567d08f5f1db0c3c1258c9560af45953d9cc`
- `merge_tree`とcandidate treeの比較: `5ff201700eb58aedd8635a5456e5a741ae4f5b22` で一致。環境依存caseなしのため重複full post-merge QAは省略。

## 既知の制約と未解決事項

- `SQLITE_BUSY`専用fixtureはない。typed分類のcode reviewと反復2-writer raceで代替した。

環境依存ケースがある場合、install/deploy/config生成、実権限、外部作用、実restart/ロールバック/クリーンアップのマージ後確認を省略しない。実環境または安全なクリーンアップが不明なケースはblockedとして残す。

## 運用上の注意

- なし

## Wikiへ引き渡す知識

### 再利用可能な知識

- versioned current/history/eventの不整合は部分read modelを返さずtyped corruption errorにする。

### 反例・失敗・注意点

- `wiki/semantic/schemas/control-durable-store.md`へschema v3、versioned CAS、watermark、recovery integrity契約を追記する。

### 更新候補ページ

- TODO

## ブートストラップ例外

- 該当なし
