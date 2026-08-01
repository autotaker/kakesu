---
task_id: "TASK-0036"
status: passed
qa_agent: "qa-agent-terra-medium"
tested_commit: "0efd3a1b7fcf2ecc452bfda97cecdeee907c4b4d"
candidate_commit: "8f07079e03e8b408e4450bb94b79915646f923a7"
candidate_tree: "287f382df73a489707d9920086ae164b894b8c7d"
managed_path_digest: "9667121cb08ea80927dcba823f843d05cd92b08934121194fd026c45cc38fde3"
bootstrap_evidence_commit: "a063f6d461bbc6ce752d93306f83e4939e299d1e"
bootstrap_evidence_digest: "279dc69dba63337208ac4d0dd065db8055e7bb0b00fb8df5e0f9024d9f283329"
merge_tree: "c5fa0c3e8c28ca644551e8df6d90c9218ec9e7d6"
decision: pass
tested_at: "2026-08-01T11:16:28+10:00"
---

# TASK-0036 QA RESULT

## 対象

- 案 コミット/tree: `8f07079e03e8b408e4450bb94b79915646f923a7` / `287f382df73a489707d9920086ae164b894b8c7d`
- `main` / merge tree: merge `0efd3a1b7fcf2ecc452bfda97cecdeee907c4b4d` / `c5fa0c3e8c28ca644551e8df6d90c9218ec9e7d6`
- `merge_tree`はマージ後にMainが記録し、案 QAでは未設定とする: 第2親がtested candidate `8f07079`と一致し、scope-check `--allow-merge true`がPASS。環境依存caseは0件なのでcandidate focused rerunを重複実行しない。
- QA PLAN 改訂: revision 4。post-implementation reviewで期待変更なし。
- 環境: Darwin arm64、candidate worktree、`GOCACHE=/private/tmp/task0036-qa-gocache`、`-count=1`。network、sudo、実OS user/serviceは使用しない。

## 結果

| ケースID | モード | 対象案 コミット/tree | 結果 | 証跡（コマンド/テスト、環境/フィクスチャ、cache、exit、成果物 ダイジェスト、ネガティブ検出能力、テスト弱体化の有無） | 未実施/blocked理由 |
|---|---|---|---|---|---|
| QA-001〜008 | `focused-rerun` | `8f07079` / `287f382` | `pass` | `go test -count=1 ./internal/provision ./internal/command` exit 0、output SHA-256 `066648d4f83cc8419846e3166242abc4babb5d288a8626540def4c08b589c787`。exact 11-line JSONL、3/4/3順序、mapping、invalid/non-leak、single-write/no-retry、snapshot、process/network/IPC import不在を確認。 | なし |
| QA-009 | `evidence-review` | 同上 | `pass` | candidate harness `make check` exit 0。6 binary build、`go vet ./...`、setup以外5 binaryのfail-closedを確認。 | なし |
| QA-010 | `evidence-review` | 同上 | `pass` | HANDOVERのcandidate-bound `make distcheck`、root check、docs lintのlog/full digestを監査。重複root checkは未実施。 | なし |
| QA-011 | `evidence-review` | 同上 | `pass` | candidate外隔離copyでorder/disabled-stopped/containment/writer guardを各1箇所弱体化。対応testは全てexit 1、full log digestはHANDOVERと一致。candidate source不変。 | なし |
| QA-012 | `evidence-review` | 同上 | `pass` | 差分は許可5 files、implementation+test 770追加行。executor/OS/template/configure/外部依存の新境界なし。 | なし |

## 発見事項

軽微指摘をQA Agentが直接修正した場合は、修正コミットとTask ブランチへの取り込みを記録する。取り込み後は解消済みとしてPASSにでき、再QAまたは`qa_carry_forward`を要求しない。

| ID | FAIL分類 | 影響 | 差し戻し候補 | 内容 |
|---|---|---|---|---|
| QA-EVIDENCE-001 | `environment_issue` | 製品影響なし | HANDOVER | 初回QA-011は省略digest/pathで監査不能。full digestと保存path追記後、対象ケースだけ再監査して解消。candidate変更なし。 |

## main Agent判断

- 結論: `pass`
- 差し戻し先: なし
- revert / バグ化: なし
- 判断理由: QA-001〜012は全てPASS。初回の証跡保持gapは製品不具合ではなく、full digest/pathを回復して限定再監査した。

## 未実施項目

- なし

## Main-owned `qa_carry_forward` / 再実行判断

- 選択: `not-applicable`
- `CF-1` 旧QA PASSと旧`candidate_commit`/`candidate_tree`の束縛: `not-applicable` / 証拠: candidate変更なし
- `CF-2` 旧新案の全差分と差分ダイジェスト: `not-applicable` / 証拠: candidate変更なし
- `CF-3` 変更は実行されない誤字、空白、コメント、リンク、証跡メタデータだけで、製品挙動、ランタイム、テスト、Schema、設定、依存、生成物、外部公開契約または安全契約、受け入れ条件、QA_PLANの意味変更がない: `not-applicable` / 証拠: candidate変更なし
- `CF-4` 影響QAケース集合: `[]`。空でなければcarry-forwardせず該当ケースを再実行する。
- `CF-5` 独立レビュアーによる挙動、テスト、安全性、契約への影響なしの確認と、新案の`make check` PASS: `not-applicable`
- `CF-6` QA FAIL、受け入れ条件/QA_PLAN変更、認証認可、秘密、sudo/PAM、IPC/Schema/設定/依存、並行性/ライフサイクル/persistence/エラー/fail-closed、テスト削除/弱体化、影響不明、証跡と評価対象の案/tree不一致が全て偽: `not-applicable`
- `CF-7` Main記録（旧新コミット/tree、全差分とダイジェスト、空の影響ケース集合、レビュアー/`make check`証拠、理由）: `not-applicable`

## 結論

`pass`
