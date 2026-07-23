---
task_id: "TASK-0027"
status: complete
completed_at: "2026-07-20"
safety_checks:
  process_tests: pending
  contract_scope: pending
  docs_lint: pending
  make_check: pending
safety_checked_at: ""
safety_check_digest: ""
safety_candidate_tree: ""
safety_merge_tree: ""
---

# TASK-0027 HANDOVER

## 成果

- Control-owned SQLite Storeのversion 1 migrationと原子的root Task作成を実装した。
- WAL、foreign keys、busy timeoutをopen時に設定し、未知・重複・混在schema versionをfail-closedで拒否する。
- Task、owner、workspace、contract v1 snapshot、progress v0、sequence 1/2 eventを一transactionで作成し、故意SQL failure時は全件rollbackする。

## candidate-bound DEV証跡

- `candidate_commit`: `ca2b505747553b0b719ae251532f74a014f831c9`
- `candidate_tree`: `dd75cbfbfd803e8d40b858f31d8a45f6b0a6c867`

| ケース ID | コマンド/テスト | 環境/フィクスチャ | cache条件 | exit | 成果物 ダイジェスト | 未実施理由 |
|---|---|---|---:|---:|---|---|
| QA-001 | `go test -count=1 ./internal/control`（migration/reopen/pragma/unknown・mixed・duplicate version/init failure） | temporary SQLite DB | `-count=1` | 0 | `c6c50a1fd0a31c6aa4960696d7c5a9f1890539cc777f3d7a36326fb5cd07db66` | なし |
| QA-002 | 同上（atomic create/readとtable/event count） | temporary SQLite DB | `-count=1` | 0 | 同上 | なし |
| QA-003 | 同上（contract metadata/payload、progress v0） | temporary SQLite DB | `-count=1` | 0 | 同上 | なし |
| QA-004 | 同上（transaction途中SQL failure、全6 table rollback、reopen、retry） | temporary SQLite DB | `-count=1` | 0 | 同上 | なし |
| QA-005 | candidate changed-path/dependency/scope監査 | candidate commit/tree | N/A | 0 | `bda419edc6542e2941052d3eafd397cf82ac1550dcefbc2eac2e318775704680` | なし |
| QA-006 | `make check`; `git diff --check` | candidate worktree | build/test cache使用 | 0 | candidate treeを正本 | なし |

- QAへ渡すネガティブ検出証拠、テスト弱体化の有無を判定できる差分ダイジェスト: candidate binary diff SHA-256 `86f04985cc280441e96604976cd05ad6d4913ab550419f64fbd8f29eae1b0579`。新規negative testsを追加し、既存testの削除・弱体化はない。

## 主要な変更

- `core/go.mod` / `go.sum`: Go 1.23互換のpure-Go `modernc.org/sqlite v1.38.2`を固定。
- `core/internal/control/store.go`: migration、pragma、typed error、atomic create/readを追加。
- `core/internal/control/store_test.go`: temporary DBの正常・再open・競合・故障注入・rollback testを追加。

## 検証結果

- DEV focused test PASS、DEV/Mainの`make check` PASS、`git diff --check` PASS。
- production `store.go`は234 physical linesで計画185行を49行超過した。全体製品Goコードは約693行で1,500行目標/1,800行hard limitに余裕があり、fault injection seam、typed error、fail-closed schema確認を削らず可読性を優先してMainが候補化を承認した。

安全契約変更では`safety_checks`を`process_tests`、`contract_scope`、`docs_lint`、`make_check`の4項目だけとし、すべて`pass`を記録する。`safety_check_digest`は案 tree、merge tree、上記順の検査名と結果を`key=value`の改行区切りで正規化し、末尾改行を含めたSHA-256とする。第2親の案 treeとmerge treeもフロントマターへ記録する。製品用のREVIEW/QA PASS、製品用の完了HANDOVER、Wiki取込記録を代用証跡として作成しない。

## 判断

- 選択: pure-Go SQLiteと単一transactionを採用し、CGO、in-memory代替、owner lifecycle/versioned recoveryの先取りはしない。
- 選択: `not-applicable`
- Main判断の旧新コミット/tree、全差分とダイジェスト、影響ケース集合、レビュアー/`make check`証拠、理由: candidateは修正なし。REVIEW/QAは`ca2b505` / `dd75cbf`をPASSし、merge commit `cb31790`のtreeも`dd75cbf`と一致した。
- carry-forward時の`QA_RESULT.md` `CF-1`から`CF-7`: `not-applicable`
- 影響QAケース集合が空でない場合の再実行証拠: 空。全caseはmerge前に同一treeのtemporary SQLiteで完了し、環境依存caseなし。
- `merge_tree`と案 treeの比較: `equal`（`dd75cbfbfd803e8d40b858f31d8a45f6b0a6c867`）

## 既知の制約と未解決事項

- なし

環境依存ケースがある場合、install/deploy/config生成、実権限、外部作用、実restart/ロールバック/クリーンアップのマージ後確認を省略しない。実環境または安全なクリーンアップが不明なケースはblockedとして残す。

## 運用上の注意

- なし

## Wikiへ引き渡す知識

### 再利用可能な知識

- schema version tableは値だけでなく行数も検証し、duplicate/mixed rowをfail-closedにする。
- commit後に同じcancel可能contextでreadを行うと「commit済みだがerror」を返し得るため、返却read modelはtransaction内で構築してからcommitする。

### 反例・失敗・注意点

- SQLite module取得は既定cacheの書込制限で失敗し得る。承認済みの書込可能cacheを明示する。

### 更新候補ページ

- Control Planeの永続Store、migration、原子的Task作成の設計ページ。

## ブートストラップ例外

- 該当なし
