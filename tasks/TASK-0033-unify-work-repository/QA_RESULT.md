---
task_id: "TASK-0033"
status: passed
qa_agent: "qa-agent-terra-medium"
tested_commit: "20f29abf663409f6d8d9f0d2cd1203e5ba0f6669"
candidate_commit: "20f29abf663409f6d8d9f0d2cd1203e5ba0f6669"
candidate_tree: "a54bab2cf40ec7ffa9009b0b99388730927bbdbe"
managed_path_digest: "dab9deccbc8884fadf604bca1653e3af6ee1cead333fb418fc900404a6053fc4"
bootstrap_evidence_commit: "a063f6d461bbc6ce752d93306f83e4939e299d1e"
bootstrap_evidence_digest: "279dc69dba63337208ac4d0dd065db8055e7bb0b00fb8df5e0f9024d9f283329"
merge_tree: ""
decision: pass
tested_at: "2026-07-23T12:20:00+10:00"
---

# TASK-0033 QA RESULT

## 対象

- 案コミット/tree: `20f29abf663409f6d8d9f0d2cd1203e5ba0f6669` / `a54bab2cf40ec7ffa9009b0b99388730927bbdbe`。独立再計算した tree と一致した。
- composite binding: managed-path digest は `dab9deccbc8884fadf604bca1653e3af6ee1cead333fb418fc900404a6053fc4`。bootstrap evidence は `a063f6d461bbc6ce752d93306f83e4939e299d1e` / `279dc69dba63337208ac4d0dd065db8055e7bb0b00fb8df5e0f9024d9f283329` であり、`make bootstrap-verify` が同じ digest を出力した。
- `main` / merge tree: 未マージのため merge tree は未設定。
- QA PLAN 改訂: Revision 2。TASK.md と QA_PLAN.md のみから独立に開始し、PLAN/HANDOVER/REVIEW_RESULT を開始根拠または判定根拠に用いなかった。
- 環境: macOS sandbox、candidate sparse worktree、製品 main evidence root、隔離 Git/bare-remote fixture。GitHub、Codex 認証、PR、archive への外部書込みは行っていない。

## 結果

| ケースID | モード | 対象案 コミット/tree | 結果 | 証跡（コマンド/テスト、環境/フィクスチャ、cache、exit、成果物 ダイジェスト、ネガティブ検出能力、テスト弱体化の有無） | 未実施/blocked理由 |
|---|---|---|---|---|---|
| QA-001 | `focused-rerun` | `20f29ab` / `a54bab2` | `pass` | candidate 全体を `../agent-harness-work` と `WORK_ROOT` で secret-free 機械探索し残存なし（exit 0）。`make task-check TASK=TASK-0033` exit 0、`make work-check MAIN_ROOT=…` exit 0（2 epic / 33 Task / 14 Wiki）。migration と single-root の positive/negative は lifecycle fixture 18/18 PASS。 | なし |
| QA-002 | `focused-rerun` | `20f29ab` / `a54bab2` | `pass` | `node --test scripts/task/unified-lifecycle.test.mjs` exit 0、18/18 PASS。隔離 bare remote fixture が task-start 成功、allocation/commit/publish failure の rollback、remote 不変、retry 上限を検出する。`make check` の process suite は97/97 PASSで、テスト削除・弱体化は観測されない。 | なし |
| QA-003 | `focused-rerun` | `20f29ab` / `a54bab2` | `pass` | `make bootstrap-verify MAIN_ROOT=…` exit 0、immutable manifest digest `279dc…8329`。fixture は freeze/unfreeze、immutable binding、allowlist/lock/scope、2回で停止する non-fast-forward retry、sparse checkout、rollback の negative を含め18/18 PASS。 | なし |
| QA-004 | `focused-rerun` | `20f29ab` / `a54bab2` | `pass` | workflow の event/permission/concurrency と read-only main CI、scope negative を lifecycle fixture で再実行し PASS。`make check` exit 0、`git diff --check a063f6d461bbc6ce752d93306f83e4939e299d1e 20f29abf663409f6d8d9f0d2cd1203e5ba0f6669` exit 0。前 candidate の EOF whitespace defect はこの range に残存しない。 | なし |
| QA-005 | `live-e2e` | `20f29ab` / `a54bab2` | `blocked (post-gate)` | composite binding と PR-scope negative は fixture で PASS。 | 実 GitHub auth、承認済み test repository、ready PR、merge-commit auto-merge、cleanup を Main が用意して確認する必要がある。fixture PASS では代替しない。 |
| QA-006 | `live-e2e` | `20f29ab` / `a54bab2` | `blocked (post-gate)` | required context 名 `Full check` / `Task check` / `Scope check` と workflow の静的責務分離は fixture で PASS。 | QA-005 の実 PR で required checks、失敗時の merge 保留、全成功時の merge を確認する必要がある。 |
| QA-007 | `live-e2e` | `20f29ab` / `a54bab2` | `blocked (post-gate)` | closed+merged 条件、push 書込み不在、`workflow_run` 不在、Task/PR concurrency の静的 negative は fixture で PASS。 | Main 所有の実 merged PR、event 再送、main 更新、cleanup を確認する必要がある。 |
| QA-008 | `live-e2e` | `20f29ab` / `a54bab2` | `blocked (post-gate)` | fixture は `FAST=1` の同期のみと空 sync の idempotency を PASS。 | merge 後、認証済み local Codex 環境で実 Wiki ingest/done/receipt/cleanup/rollback を確認する必要がある。 |
| QA-009 | `focused-rerun` | `20f29ab` / `a54bab2` | `pass` | migration fixture は固定 REF-2、historical Done 32件、TASK-0033 overlay、tamper negative を再実行して18/18 PASS。`make work-check` と immutable bootstrap verify はともに exit 0。 | なし |
| QA-010 | `live-e2e` | `20f29ab` / `a54bab2` | `blocked (post-gate)` | archive は実行していない。fixture は archive 前の migration binding を検証する。 | QA-009 後、Main が対象/authority/unarchive・rollback を確認した承認済み public repository でのみ実施できる。 |

## 発見事項

| ID | FAIL分類 | 影響 | 差し戻し候補 | 内容 |
|---|---|---|---|---|
| - | - | - | - | pre-gate focused-rerun に FAIL はない。 |

## main Agent判断

- 結論: `pass`（pre-gate candidate QA）。
- 差し戻し先: なし。
- revert / バグ化: 不要。
- 判断理由: candidate/tree/digest と bootstrap binding が一致し、QA-001〜004・QA-009 の指定 focused-rerun が成功した。QA-005〜008・QA-010 は環境依存の post-gate `live-e2e` であり、未実施を PASS として代替していない。

## 未実施項目

- QA-005〜QA-008、QA-010: PR/required checks/merge workflow、local Codex Wiki ingest、public repository archive に関する post-gate live-e2e。Main 所有の承認済み実環境と安全な cleanup が必要。

## Main-owned `qa_carry_forward` / 再実行判断

- 選択: `full-rerun`。
- `CF-1`〜`CF-7`: `not-applicable`。前 candidate の QA FAIL 後の新 candidate であり、workflow、Schema、lock、lifecycle、外部認証を含むため carry-forward は禁止される。影響する focused-rerun を全面再実行した。

## 結論

`pass` — candidate の pre-gate QA は PASS。QA-005〜QA-008 と QA-010 の実 GitHub/Codex live-e2e は、PR/merge 後に Main がケース単位で確認する。
