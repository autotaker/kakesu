---
task_id: "TASK-0027"
status: pass
qa_agent: "qa-agent-terra-medium"
tested_commit: "cb31790b7eebdff70038daa9e087a2a8eb80c9e1"
candidate_commit: "ca2b505747553b0b719ae251532f74a014f831c9"
candidate_tree: "dd75cbfbfd803e8d40b858f31d8a45f6b0a6c867"
merge_tree: "dd75cbfbfd803e8d40b858f31d8a45f6b0a6c867"
decision: pass
tested_at: "2026-07-20"
---

# TASK-0027 QA RESULT

## 対象

- 案 コミット/tree: `ca2b505747553b0b719ae251532f74a014f831c9` / `dd75cbfbfd803e8d40b858f31d8a45f6b0a6c867`
- `main` / merge tree: `cb31790b7eebdff70038daa9e087a2a8eb80c9e1` / `dd75cbfbfd803e8d40b858f31d8a45f6b0a6c867`
- `merge_tree`はマージ後にMainが記録し、案 QAでは未設定とする: 記録済み、candidate treeと一致。
- QA PLAN 改訂: revision 2。`go.sum`を承認済み生成checksum lockとしてQA-005へ明記したqa_plan_defect補正。製品期待値不変。
- 環境: Go 1.23.12、file-backed temporary SQLite、cache無効focused test。

## 結果

| ケースID | モード | 対象案 コミット/tree | 結果 | 証跡（コマンド/テスト、環境/フィクスチャ、cache、exit、成果物 ダイジェスト、ネガティブ検出能力、テスト弱体化の有無） | 未実施/blocked理由 |
|---|---|---|---|---|---|
| QA-001 | `focused-rerun` | `ca2b505` / `dd75cbf` | `pass` | migration/reopen/pragmas、unknown/mixed/duplicate version、init failureを`go test -count=1 -v ./internal/control/...`で確認。exit 0、output SHA-256 `41cc608de2159645d60efdff301f62adb0712eb357b220c5061f3603cbfa1e69` | なし |
| QA-002 | `focused-rerun` | 同上 | `pass` | atomic createと6種row/sequence 1/2 eventsをtemporary DBで確認 | なし |
| QA-003 | `focused-rerun` | 同上 | `pass` | contract v1 metadata/payload、progress v0とread modelを確認 | なし |
| QA-004 | `focused-rerun` | 同上 | `pass` | 故意SQL failure後の全rollback、close/reopen、同一入力retryを確認。`-count=20`もPASS | なし |
| QA-005 | `evidence-review` | 同上 | `pass` | 4 approved paths、modernc SQLite v1.38.2、GoVersion 1.23.0、BSD-3-Clause、scope外APIなし、diff SHA-256 `86f04985cc280441e96604976cd05ad6d4913ab550419f64fbd8f29eae1b0579` | なし |
| QA-006 | `evidence-review` | 同上 | `pass` | `make check`、`git diff --check`、補正後`make task-check TASK=TASK-0027` PASS。worktree clean | なし |

## 発見事項

軽微指摘をQA Agentが直接修正した場合は、修正コミットとTask ブランチへの取り込みを記録する。取り込み後は解消済みとしてPASSにでき、再QAまたは`qa_carry_forward`を要求しない。

| ID | FAIL分類 | 影響 | 差し戻し候補 | 内容 |
|---|---|---|---|---|
| QF-001 | requirement_gap | 証跡のみ | Main/PLAN | 初期190行/1ptと実測の不一致。work commit `1afee33`で252行/2ptへ補正、candidate不変、再確認PASS |
| QF-002 | qa_plan_defect | QA scope記述のみ | Main/QA | `go.sum`を未承認pathとした誤記。QA_PLAN rev2/work commit `92d3741`で承認生成lockへ補正、期待値不変、再確認PASS |

## main Agent判断

- 結論: `PASS`
- 差し戻し先: なし
- revert / バグ化: 不要
- 判断理由: focused persistence境界、negative failure、scope、全体checkが同一candidateでPASS。証跡gapは製品treeを変えず補正・再確認済み。

## 未実施項目

- なし

## Main-owned `qa_carry_forward` / 再実行判断

- 選択: `not-applicable | qa_carry_forward | focused-rerun | full-rerun`
- `CF-1` 旧QA PASSと旧`candidate_commit`/`candidate_tree`の束縛: `pending` / 証拠: TODO
- `CF-2` 旧新案の全差分と差分ダイジェスト: `pending` / 証拠: TODO
- `CF-3` 変更は実行されない誤字、空白、コメント、リンク、証跡メタデータだけで、製品挙動、ランタイム、テスト、Schema、設定、依存、生成物、外部公開契約または安全契約、受け入れ条件、QA_PLANの意味変更がない: `pending` / 証拠: TODO
- `CF-4` 影響QAケース集合: `[]`。空でなければcarry-forwardせず該当ケースを再実行する。
- `CF-5` 独立レビュアーによる挙動、テスト、安全性、契約への影響なしの確認と、新案の`make check` PASS: `pending` / レビュアー証拠、コマンド、結果: TODO
- `CF-6` QA FAIL、受け入れ条件/QA_PLAN変更、認証認可、秘密、sudo/PAM、IPC/Schema/設定/依存、並行性/ライフサイクル/persistence/エラー/fail-closed、テスト削除/弱体化、影響不明、証跡と評価対象の案/tree不一致が全て偽: `pending`
- `CF-7` Main記録（旧新コミット/tree、全差分とダイジェスト、空の影響ケース集合、レビュアー/`make check`証拠、理由）: TODO

## 結論

`PASS`
