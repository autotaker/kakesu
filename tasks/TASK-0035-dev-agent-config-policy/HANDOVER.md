---
task_id: "TASK-0035"
status: draft
completed_at: ""
safety_checks:
  process_tests: pending
  contract_scope: pending
  docs_lint: pending
  make_check: pending
safety_checked_at: ""
safety_check_digest: ""
safety_candidate_tree: ""
safety_merge_tree: ""
candidate_commit: "5e5d29e8250d8b2999d2cf6e51e748b7f866b016"
candidate_tree: "cc3ea4ba4b6f99da68e73e370faddc3c1bad2aa1"
managed_path_digest: "a62367397b23bb54347ff9f04951b7a11e2eb2a97e1385c51b9a2e2aef395c40"
bootstrap_evidence_commit: "a063f6d461bbc6ce752d93306f83e4939e299d1e"
bootstrap_evidence_digest: "279dc69dba63337208ac4d0dd065db8055e7bb0b00fb8df5e0f9024d9f283329"
---

# TASK-0035 HANDOVER

## 成果

- Development Agent Harnessのversion 1設定をstrict JSON、意味検証、FD基準file policyでread-only検証する基盤を追加した。
- `dev-agent-harness-setup check-config --config PATH`だけを新たに許可し、設定値を出力せず、他の通常起動はfail-closedのまま維持した。

## candidate-bound DEV証跡

- `candidate_commit`: `5e5d29e8250d8b2999d2cf6e51e748b7f866b016`
- `candidate_tree`: `cc3ea4ba4b6f99da68e73e370faddc3c1bad2aa1`
- `managed_path_digest`: `a62367397b23bb54347ff9f04951b7a11e2eb2a97e1385c51b9a2e2aef395c40`
- `bootstrap_evidence_commit` / `bootstrap_evidence_digest`: `a063f6d461bbc6ce752d93306f83e4939e299d1e` / `279dc69dba63337208ac4d0dd065db8055e7bb0b00fb8df5e0f9024d9f283329`。現repositoryの不変bootstrap manifestを継承する。

| ケース ID | コマンド/テスト | 環境/フィクスチャ | cache条件 | exit | 成果物 ダイジェスト | 未実施理由 |
|---|---|---|---:|---:|---|---|
| QA-035-01 | install済み`dev-agent-harness-setup check-config --config .../harness.json.example` | 明示absolute configure args、temporary `DESTDIR=/tmp/task-0035-install.QY1FiP`、login shell無効 | build cache使用 | 0 | stdout `config version=1 network.default=deny validated` | なし |
| QA-035-02 | `go test -count=1 ./internal/config ./internal/command` | strict/duplicate/unknown/version/trailing fixture | absolute `GOCACHE`、`-count=1` | 0 | stdout SHA-256 `8c8ad914cb859154592c890e97de4168b561fef04f4a4f047bc1483e14b55a6a` | なし |
| QA-035-03 | 同上 | path/user/network/allowlist unknown fixture | 同上 | 0 | 同上 | なし |
| QA-035-04 | 同上 | regular/mode/size/symlink/directory/FIFO fixture、`O_NOFOLLOW|O_NONBLOCK` | 同上 | 0 | 同上 | なし。read中attribute raceの独立QAを要する |
| QA-035-05 | 同上およびcandidate diff監査 | positive/negative tableとcandidate diff | 同上 | 0 | candidate diff SHA-256 `8bc852a6894cbe2077e083f2541726308d5d65b0d5fd527555e03685d0c4f9ed` | 独立mutationはQAが実施する |
| QA-035-06 | `make check` | 全6 binary build、help/version/fail-closed unit/loop | warm absolute build cache | 0 | output SHA-256 `c09c98dc056d1788fb622f303557cfe2642ac0d8d1762a7a44667a7503234ef5` | なし |
| QA-035-07 | explicit `./configure`、`make check`、`make distcheck`、temporary `DESTDIR` install後example検証 | macOS host、Go 1.24、明示absolute directory args | warm absolute build cache | 0 | distcheck output SHA-256 `319226754bcd31e72083ca53681b351d6820f0b4df78d13b8f1e88136b46967a` | なし |
| QA-035-08 | candidate scope/line/dependency監査 | base `5d8ecf1`、candidate commit/tree | N/A | 0 | managed digest `a62367397b23bb54347ff9f04951b7a11e2eb2a97e1385c51b9a2e2aef395c40` | なし |

- QAへ渡すネガティブ検出証拠、テスト弱体化の有無を判定できる差分ダイジェスト: `8bc852a6894cbe2077e083f2541726308d5d65b0d5fd527555e03685d0c4f9ed`。既存test削除はなく、unknown/duplicate/version/trailing/path/user/network/allowlist/mode/size/symlink/directory/FIFOの拒否testを追加した。

## 主要な変更

- `internal/config/config.go`: 64 KiB上限、no-follow/nonblocking open、read前後FD検査、duplicate token scan、strict typed decode、semantic validation、安全なerror class。
- `internal/config/config_test.go`: valid/invalid設定と危険fileのnegative test。
- `internal/command/`: setup `check-config`の固定出力と非漏洩診断。既存command surfaceは維持。
- `README.md`: read-only事前検証とfile policyを追記。

## 検証結果

- candidate固定後の`go test -count=1 ./internal/config ./internal/command`、`make check`、`make distcheck`、`git diff --check`はいずれもexit 0。
- 明示absolute configure argsでinstallした設定例を、install済みsetup binaryで検証してexit 0。
- 実装・test差分は436行、READMEを含む全差分は442行。700〜1,200行は目安であり、ACを削らず予定910行より小さく実装できた。外部module、network、Credential、IPC、永続状態、OS変更はない。

安全契約変更では`safety_checks`を`process_tests`、`contract_scope`、`docs_lint`、`make_check`の4項目だけとし、すべて`pass`を記録する。`safety_check_digest`は案 tree、merge tree、上記順の検査名と結果を`key=value`の改行区切りで正規化し、末尾改行を含めたSHA-256とする。第2親の案 treeとmerge treeもフロントマターへ記録する。製品用のREVIEW/QA PASS、製品用の完了HANDOVER、Wiki取込記録を代用証跡として作成しない。

## 判断

- candidate `5e5d29e`を独立REVIEW/QAへ渡す。修正が必要な場合、設定/parser/file policy/fail-closedに関わるためcarry-forwardせず、影響ケースを再実行する。
- 選択: `not-applicable | qa_carry_forward | focused-rerun | full-rerun`
- Main判断の旧新コミット/tree、全差分とダイジェスト、影響ケース集合、レビュアー/`make check`証拠、理由: TODO
- carry-forward時の`QA_RESULT.md` `CF-1`から`CF-7`: `not-applicable | complete | incomplete`
- 影響QAケース集合が空でない場合の再実行証拠: TODO
- `merge_tree`と案 treeの比較: `pending`

## 既知の制約と未解決事項

- なし

環境依存ケースがある場合、install/deploy/config生成、実権限、外部作用、実restart/ロールバック/クリーンアップのマージ後確認を省略しない。実環境または安全なクリーンアップが不明なケースはblockedとして残す。

## 運用上の注意

- なし

## Wikiへ引き渡す知識

### 再利用可能な知識

- Development Agent HarnessはKakesu本体外の開発基盤であり、version 1設定はstrict decode、deny既定、危険file拒否を共有不変条件にする。

### 反例・失敗・注意点

- `encoding/json`単独ではduplicate keyを拒否できないため、token scanとtyped strict decodeを分離する。
- FIFOはread-only openだけではblockし得るため、file type検査前のopenにも`O_NONBLOCK`が必要である。
- Autoconf既定の`${prefix}`は設定例の絶対pathにならないため、検証fixtureはREADMEどおりabsolute directory argsを明示する。

### 更新候補ページ

- `wiki/semantic/schemas/development-agent-harness-config.md`（新規候補）

## ブートストラップ例外

- 該当なし
