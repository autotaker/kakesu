---
task_id: "TASK-0036"
status: draft
completed_at: ""
safety_checks:
  process_tests: pass
  contract_scope: pass
  docs_lint: pass
  make_check: pass
safety_checked_at: "2026-08-01T11:04:28+10:00"
safety_check_digest: "not-applicable-product-task"
safety_candidate_tree: "287f382df73a489707d9920086ae164b894b8c7d"
safety_merge_tree: ""
candidate_commit: "8f07079e03e8b408e4450bb94b79915646f923a7"
candidate_tree: "287f382df73a489707d9920086ae164b894b8c7d"
managed_path_digest: "9667121cb08ea80927dcba823f843d05cd92b08934121194fd026c45cc38fde3"
bootstrap_evidence_commit: "a063f6d461bbc6ce752d93306f83e4939e299d1e"
bootstrap_evidence_digest: "279dc69dba63337208ac4d0dd065db8055e7bb0b00fb8df5e0f9024d9f283329"
---

# TASK-0036 HANDOVER

## 成果

- `dev-agent-harness-setup plan-provision --config PATH --target-root PATH`を追加し、対象OSのuser、ディレクトリ、serviceの望ましい状態を、ヘッダー1行＋action 10行の決定的JSONLとして出力する。
- manifestは全recordを事前構築・検証・serializeし、writerを1回だけ呼ぶ。target rootやhostは変更せず、process、network、IPCを開始しない。

## candidate-bound DEV証跡

- `candidate_commit`: `8f07079e03e8b408e4450bb94b79915646f923a7`
- `candidate_tree`: `287f382df73a489707d9920086ae164b894b8c7d`
- `managed_path_digest`: `9667121cb08ea80927dcba823f843d05cd92b08934121194fd026c45cc38fde3`
- `bootstrap_evidence_commit` / `bootstrap_evidence_digest`: `a063f6d461bbc6ce752d93306f83e4939e299d1e` / `279dc69dba63337208ac4d0dd065db8055e7bb0b00fb8df5e0f9024d9f283329`

| ケース ID | コマンド/テスト | 環境/フィクスチャ | cache条件 | exit | 成果物 ダイジェスト | 未実施理由 |
|---|---|---|---:|---:|---|---|
| QA-001 | `go test -count=1 ./...` | final candidate、temporary target root、exact 11-line JSONL fixture | 専用`GOCACHE`、`-count=1` | 0 | log SHA-256 `daa212321eb271b6c0a7c2f40bc18f30a8d295e841391823b8550944b05f955f` | なし |
| QA-002 | 同上 | 3 userのrole/name対応、locked/non-login/home非作成、direct Config negative | 同上 | 0 | 同上 | なし |
| QA-003 | 同上 | 4 directoryのmode/owner/group、audit関係、target mapping | 同上 | 0 | 同上 | なし |
| QA-004 | 同上 | empty/relative/non-clean/NUL/root logical、same-string coordinate positive | 同上 | 0 | 同上 | なし |
| QA-005 | 同上 | 3 serviceの固定名、broker user、disabled/stopped | 同上 | 0 | 同上 | なし |
| QA-006 | 同上 | CLIの不足/余剩/順序/root/config拒否とstderr非漏洩 | 同上 | 0 | 同上 | なし |
| QA-007 | 同上 | prefix+error/short writer、call count=1、retryなし | 同上 | 0 | 同上 | なし |
| QA-008 | 同上とsource/diff監査 | sentinel tree snapshot、production import/seamの外部作用不在 | 同上 | 0 | candidate diff SHA-256 `a6be558f466a9b72c7241fa9ac14a00808402754e6045e25a96fa8d207adf5e3` | なし |
| QA-009 | harness `make check` | 全6 binaryのbuild/version/通常起動fail-closed | warm harness cache | 0 | log SHA-256 `ccc50ca362e72d8ceb95f2af2ca116cc55e065f79d82816806f2744cdc9a67a2` | なし |
| QA-010 | harness `make distcheck`、root `PYTEST_ADDOPTS=--ignore=worktrees make check` | final candidate、外部作用なし、root依存は事前取得済み | warm cache | 0 / 0 | distcheck `07cd8275989b76c7b9e35edebdad11bc5b8d24f2da070f686fe5d300c88b4eb2`、root check `94959f5af9b2b63e6ed88ae26e8a64afa710c4a1e70eaddce0dc5bb6eb279bdb` | live-e2eはTask対象外 |
| QA-011 | 隔離copyでorder/disabled-stopped/containment/writer guardを各1箇所弱体化し対応test実行 | `/tmp/task0036-mutations.dZfx0k`、candidate sourceは不変 | 専用`GOCACHE` | 全4件 1（期待FAIL） | log SHA-256 `fe9bd20...`、`fbf190b8...`、`48d10b37...`、`294aa8df...` | なし |
| QA-012 | `git diff --numstat` / scope・import監査 | base `c3c84f0`→candidate `8f07079` | N/A | 0 | 5 files、implementation+test 770 added lines、managed digestは上記 | なし |

- QAへ渡すネガティブ検出証拠はQA-011の4件で、すべて対応testが期待どおりFAILした。既存testの削除・skip・assertion緩和はない。candidate diff SHA-256は`a6be558f466a9b72c7241fa9ac14a00808402754e6045e25a96fa8d207adf5e3`。

## 主要な変更

- `internal/provision`: manifest型、strict target mapping、direct Config validation、全record validator、canonical JSONL、single-write adapter。
- `internal/command`: setupの厳密な`plan-provision`引数と安全なexit/stderr変換。
- unit/CLI tests: exact bytes、path/user/directory/service不変条件、writer失敗、target snapshot、非漏洩、mutation guard。
- README: 読み取り専用の配置計画とexecutor非対象の境界。

## 検証結果

- final candidate上で`go test -count=1 ./...`、harness `make check`、`make distcheck`、targeted textlint、root `make check`、`git diff --check`がPASSした。
- 手書きimplementation+testは770追加行で、700〜1,200行の目安内。外部module、OS executor、network、IPC、Credential、systemd/template/configure変更はない。

安全契約変更では`safety_checks`を`process_tests`、`contract_scope`、`docs_lint`、`make_check`の4項目だけとし、すべて`pass`を記録する。`safety_check_digest`は案 tree、merge tree、上記順の検査名と結果を`key=value`の改行区切りで正規化し、末尾改行を含めたSHA-256とする。第2親の案 treeとmerge treeもフロントマターへ記録する。製品用のREVIEW/QA PASS、製品用の完了HANDOVER、Wiki取込記録を代用証跡として作成しない。

## 判断

- 初回implementation commit `0649ee6`後、Main監査でlogical pathとtarget-rootのcoordinate同一文字列を正常系へ訂正した。root docs lintの用語頻度指摘はREADMEのみで解消し、最終candidate `8f07079`を固定した。
- 選択: `not-applicable`
- Main判断の旧新コミット/tree、全差分とダイジェスト、影響ケース集合、レビュアー/`make check`証拠、理由: 現時点は`not-applicable`。独立REVIEW/QA後にcandidate変更がある場合だけ更新する。
- carry-forward時の`QA_RESULT.md` `CF-1`から`CF-7`: `not-applicable`
- 影響QAケース集合が空でない場合の再実行証拠: 現時点は該当なし。
- `merge_tree`と案 treeの比較: `pending`

## 既知の制約と未解決事項

- なし

環境依存ケースがある場合、install/deploy/config生成、実権限、外部作用、実restart/ロールバック/クリーンアップのマージ後確認を省略しない。実環境または安全なクリーンアップが不明なケースはblockedとして残す。

## 運用上の注意

- なし

## Wikiへ引き渡す知識

### 再利用可能な知識

- manifest V1のfieldとrecord順序はexact bytes testで固定しており、互換拡張はversionを曖昧に維持せず別Taskで扱う。
- target mappingはlogical pathとtarget-rootを異なるcoordinateとして扱い、logical `/`だけを空の相対成分として拒否する。

### 反例・失敗・注意点

- OS-level writerがprefixを書いてからerrorを返した場合、既に書かれたbyteは巻き戻せない。application契約は事前全件検証と1回write、retry/re-emitなしまでである。

### 更新候補ページ

- `wiki/semantic/schemas/development-agent-harness-provision-manifest.md`（新規候補）

## ブートストラップ例外

- 該当なし
