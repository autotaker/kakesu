---
task_id: "TASK-0029"
status: passed
reviewer_agent: "reviewer-agent-terra-medium"
reviewed_commit: "864f455b563a6fffb043ed297d5cb10e3849b988"
candidate_commit: "864f455b563a6fffb043ed297d5cb10e3849b988"
candidate_tree: "5ff201700eb58aedd8635a5456e5a741ae4f5b22"
decision: pass
make_check: pass
reviewed_at: "2026-07-20"
---

# TASK-0029 REVIEW RESULT

## 対象

- ブランチ: `task/TASK-0029-control-versioned-recovery`
- 修正済みcandidate: `864f455b563a6fffb043ed297d5cb10e3849b988`
- tree: `5ff201700eb58aedd8635a5456e5a741ae4f5b22`
- TASK / PLAN / QA_PLANを読み、QAのPASSを開始条件にせず独立評価した。

## 実行した検査

| コマンド | 結果 | 備考 |
|---|---|---|
| `git diff --check af0d8d5..864f455` | PASS | exit 0 |
| `cd core && go test -count=20 -run TestProgressUpdateRejectsRegressingAndFutureWatermarksWithoutMutation ./internal/control` | PASS | exit 0 |
| `make check` | PASS | exit 0、corrected candidateで一度実行 |

## 受け入れ条件

| 条件 | 結果 | 根拠 |
|---|---|---|
| contract/progress CASとimmutable history | PASS | expected+1、history/current/event同一transaction |
| rollbackと並行競合 | PASS | 全stage fault test、2-writer repeated test |
| recovery integrity | PASS | close/reopen、current/history/event corruption fail-fast |
| schema refsとprogress watermarks | PASS | 両watermark後退と未来Task sequenceをwrite前にtyped conflict |
| migration/scope/SLOC | PASS | v2→v3保存、DDL rollback、418/425 production行 |

## 指摘履歴

| ID | 重大度 | 状態 | 内容 |
|---|---|---|---|
| R29-01 | P1 | resolved by `864f455` | 初回candidate `af0d8d5` はwatermark後退/未来Task sequenceを受理した。修正と状態不変testを取り込み解消。 |

## 残存リスクと結論

- Agent Runの実在上限照合はAgent Run自体が本Task対象外のため対象外。非後退は検証済み。
- 重大findingなし。結論: **PASS**
