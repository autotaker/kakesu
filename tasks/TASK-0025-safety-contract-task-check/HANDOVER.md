---
task_id: "TASK-0025"
status: complete
completed_at: "2026-07-20"
---

# TASK-0025 HANDOVER

## 成果

- `task-check`へ製品変更と安全契約変更の分類別完了gateを実装した。
- product gateを維持し、安全契約ではrename/copy、製品path、証跡欠落・偽装をfail-closedで拒否する。

## candidate-bound DEV証跡

- worktree: `/home/ubuntu/git/agent-harness-work/worktrees/TASK-0025-safety-contract-task-check`
- `candidate_commit`: `bd8186eead0681fc3e963de715ba5d266b4a0379`
- `candidate_tree`: `4d4309217a5d4d1dcc1150fc1cf5009813b91cc6`

| ケース ID | コマンド/テスト | 環境/フィクスチャ | cache条件 | exit | 成果物 ダイジェスト | 未実施理由 |
|---|---|---|---:|---:|---|---|
| QA-001〜QA-005 | `node --test scripts/task/development-process.test.mjs` | Node 22.23.1、一時Git fixture、49 test | local dependency/cache、外部networkなし | 0 | `e1522f15638375fc4df8d07849d676573c90c5e5b56490fa1732ddbeaa861c84` | なし |
| QA-006 | `make check` | TASK-0025 product worktree | warm local build/cache、外部networkなし | 0 | candidate差分digestを参照 | なし |

- QAへ渡すネガティブ検出証拠、テスト弱体化の有無を判定できる差分ダイジェスト: `21d6dd7889eb3cef751f04307e946be3b3186b63e3c7a25aa2a7d04557032857`

## 主要な変更

- `change_class`の互換解釈、分類別Done gate、固定証跡field、tree/digest束縛を追加した。
- process fixtureを39 testへ拡張し、rename/copy、非no-ff、担当不一致、分類mirror、理由/時系列、検査集合、digest/tree不一致を検出する。

## 検証結果

- Focused test: 49/49 PASS。
- `make check`: process 50/50を含む全検査PASS。
- `git diff --check`、用語集同期、文書lint: PASS。

## 判断

- Reviewer初回FAILの4件は通常DEV経路で修正し、candidateを更新した。
- 選択: `not-applicable | qa_carry_forward | focused-rerun | full-rerun`
- Main判断の旧新コミット/tree、全差分とダイジェスト、影響ケース集合、レビュアー/`make check`証拠、理由: TODO
- carry-forward時の`QA_RESULT.md` `CF-1`から`CF-7`: `not-applicable | complete | incomplete`
- 影響QAケース集合が空でない場合の再実行証拠: TODO
- `merge_tree`と案 treeの比較: `4d4309217a5d4d1dcc1150fc1cf5009813b91cc6 == 4d4309217a5d4d1dcc1150fc1cf5009813b91cc6`、PASS。

## 既知の制約と未解決事項

- なし

環境依存ケースがある場合、install/deploy/config生成、実権限、外部作用、実restart/ロールバック/クリーンアップのマージ後確認を省略しない。実環境または安全なクリーンアップが不明なケースはblockedとして残す。

## 運用上の注意

- なし

## Wikiへ引き渡す知識

### 再利用可能な知識

- TODO

### 反例・失敗・注意点

- TODO

### 更新候補ページ

- TODO

## ブートストラップ例外

- 該当なし
