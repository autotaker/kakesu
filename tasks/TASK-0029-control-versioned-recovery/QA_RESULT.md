---
task_id: "TASK-0029"
status: passed
qa_agent: "qa-agent-terra-medium"
tested_commit: "a908567d08f5f1db0c3c1258c9560af45953d9cc"
candidate_commit: "864f455b563a6fffb043ed297d5cb10e3849b988"
candidate_tree: "5ff201700eb58aedd8635a5456e5a741ae4f5b22"
merge_tree: "5ff201700eb58aedd8635a5456e5a741ae4f5b22"
decision: pass
tested_at: "2026-07-20"
---

# TASK-0029 QA RESULT

## 対象

- corrected candidate/tree: `864f455b563a6fffb043ed297d5cb10e3849b988` / `5ff201700eb58aedd8635a5456e5a741ae4f5b22`
- file-backed temporary SQLite、isolated Go cache、clean worktreeで実行した。
- Reviewerの結果を開始条件にせず独立に開始した。

## 結果

| ケース | モード | 結果 | 証跡 |
|---|---|---|---|
| Q29-01/02/03/05/06/07 | focused-rerun | PASS | targeted suite `-count=20`, exit 0, 6.08s。CAS、watermark negative、rollback、reopen/corruption。 |
| Q29-04 | focused-rerun | PASS | conflict race suite `-race -count=20`, exit 0, 7.85s。 |
| Q29-08 | evidence-review | PASS | scope-limited diff、dependency/Schema/API追加なし。 |
| Q29-09 | focused-rerun | PASS | `make check` exit 0 (19.46s)、`make task-check TASK=TASK-0029` exit 0、`git diff --check` exit 0。 |

- correction diff SHA-256: `1948925dea200b08fd8f705d70a54a5f67c4830d96633e206e1e5c8aea35e220`
- full candidate diff SHA-256: `39f2ccfe174e6b0ebfcccc930b6f90ef1d13655731702aba1aa1fb4e01da6d74`
- テスト削除/弱体化なし。最終worktree clean、HEAD/tree一致。

## FAIL履歴とMain判断

- 初回candidate `af0d8d5` / `4f7b761` はQ29-02/03/05/08/09 FAIL。分類: `implementation_defect`。watermark後退と未来Task sequenceを受理した。
- corrected candidateは影響caseとcandidate-wide checksを再実行した。`qa_carry_forward` は使用していない。
- Main結論: **PASS**。差し戻しなし。

## 残存リスク

- `SQLITE_BUSY` だけの専用deterministic testはない。busy/lockedのtyped conflict分類をcode reviewし、反復2-writer raceで一方のみ成功・partial writeなしを確認した。

## 結論

**PASS**
