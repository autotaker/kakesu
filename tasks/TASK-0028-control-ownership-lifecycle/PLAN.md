---
task_id: "TASK-0028"
change_class: "product"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-07-20"
approved_dev_profile: "sol-high"
approved_dev_profile_reason: "SQLite排他制約、並行transaction、lifecycle原子性、終端owner解放を扱う高リスク実装のため"
approved_dev_profile_risk_signals: ["persistence", "concurrency", "lifecycle", "fail-closed"]
planning_reviewed_by: ""
planning_review_decision: "pending"
planning_reviewed_at: ""
planned_implementation_files: 2
planned_implementation_lines: 225
estimate_points: 2
---

# TASK-0028 PLAN — Control ownership and lifecycle

## Decision and DEV gate

TASK-0007 PLAN 0007-Bを、TASK-0027 Store APIに依存する独立した1〜2 Lap product sliceとして実装する。本PLANは設計証跡だけであり、Mainが本PLANと独立TASK-first QA_PLANを承認するまでDEVを開始しない。DEVはsol-high、Planner/Reviewer/QAは別のTerra/medium roleとし、DEVが固定した同じcandidate_commit/candidate_treeからReviewerとQAを独立かつ並行に開始する。Mainだけがmainへのmerge/pushとmerge_tree照合を所有する。

## 受け入れ条件の具体化

| 条件 | 観測方法 | fail-closed結果 |
|---|---|---|
| Agentの非終端owner一件 | 2接続・barrier同期で同一Agentへ別Taskを同時割当 | 一方だけcommit。敗者はtyped conflict、partial row/eventなし |
| 許可遷移のみ | 全7状態の直積をtable-driven temporary SQLite testで実行 | 明示辺だけ成功。拒否時Task/progress/owner/event sequence不変 |
| 非終端owner保持 | waiting/suspended/reviewing_completion中の再割当を試行 | 再割当拒否、元owner不変 |
| 終端原子性 | completed/cancelled遷移で故意SQL failureを注入 | state/event/release全rollback。成功時だけ同時確定 |
| 解放後再利用 | terminal commit後に同一Agentを別Taskへ割当 | 新割当成功、旧Task/eventは不変 |

## Fixed transition table

許可辺は次だけとする。

- ready → running
- running → waiting | suspended | reviewing_completion
- waiting → running
- suspended → running
- reviewing_completion → running | completed
- ready | running | waiting | suspended | reviewing_completion → cancelled

completed/cancelledは終端でoutgoing edgeを持たない。waiting/suspended/reviewing_completionは非終端なのでownerを保持する。期待current stateとDB current stateが一致しない要求は競合として拒否する。拒否操作はprogress rowを含む既存read modelを一切変更しない。

## Design

### Selected design

TASK-0027のStoreにmigration version 2を追加し、active owner assignmentをDBで一意にする。SQLiteが部分unique indexを十分表現できる場合はactive assignmentだけを対象にし、terminal時はassignment終了値を同じtransactionで設定する。制約に依存しつつ、assign APIもBEGIN transaction内で現在Task/Agent状態を検査してCAS insert/updateを行う。競合はtyped conflictとして返し、自動retryしない。

新しいlifecycle portはtask ID、expected current state、target state、対応event payloadを受け、明示transition tableを検査してから一transactionでcurrent Task、単調event sequence、必要なterminal owner releaseを更新する。非終端遷移はowner rowを変更しない。terminal resource cleanupやEpisode生成はtransaction外の後続consumer責務であり、owner解放のcommit条件にしない。

### Invariants

- 1非終端Taskは1 owner、1 Agentは同時に1非終端Taskだけを所有する。
- state current rowと対応eventは同一transaction。terminalではowner releaseも同じtransaction。
- waiting/suspendedは障害/待機であり責任解放ではない。
- 拒否/競合/SQL failureではstate、progress、owner、event sequenceがbyte-for-byte不変。
- event sequenceは既存最大値から一つだけ増え、失敗時にgapを作らない。

### Alternatives rejected

- process-local mutexのみ: 複数connection/process競合を防げない。
- application checkだけ: check後insertの競合窓を閉じない。
- eventを後続transactionで追加: current stateとhistoryが分離する。
- terminal後cleanup完了待ち: owner解放を物理resource failureへ結び付ける。
- generic versioned update: TASK-0029のexpected-version/recovery範囲を先取りする。

## Planned implementation boundary

| File | Type | Estimated lines | Change |
|---|---|---:|---|
| core/internal/control/store.go | implementation | 70 | migration v2、active-owner unique constraint、assignment CAS/release helper、typed conflict |
| core/internal/control/lifecycle.go | implementation | 155 | explicit transition table、atomic transition/event/terminal release port |

Production estimate is 2 files / 225 lines. Tests in core/internal/control/store_test.go and core/internal/control/lifecycle_test.go are required but excluded from production SLOC. file_score=ceil(2/3)=1、line_score=ceil(225/200)=2、therefore estimate is 2 points. The approved working range is 210–240 readable lines; above 240 stops before more implementation and returns to PLAN. No semicolon packing, merged error paths, cryptic names, or deleted validation may be used to fit the range.

## 1–2 Lap execution

### Lap 1 — candidate

1. Preflight requires merged TASK-0027, its candidate_tree/merge_tree equality, Store API and migration version, writable bounded temporary SQLite fixture, and approved PLAN/QA_PLAN.
2. DEV adds migration/constraint/CAS, explicit transition table, atomic state/event/release methods, then table-driven all-edge and two-connection concurrency tests.
3. DEV records candidate_commit/tree and focused evidence including fixture path policy, cache condition, command exit, artifact digest, negative-detection mutations, production SLOC, retries, and failure classification.
4. Lap 1 must end with a complete reviewable candidate. Missing concurrency/rollback/all-edge proof is a stop, not Lap 2 implementation inventory.

### Lap 2 — bounded finding correction only

Lap 2 is allowed only for one or two concrete REVIEW/QA findings against the complete Lap-1 candidate, no redesign/schema-policy change/new fixture/0029 scope, and a bounded correction estimated within its first 20 minutes. Otherwise stop and split/replan. No Lap 3.

## Verification plan

DEV runs focused checks first; Reviewer and QA independently rerun the cases assigned by the approved QA_PLAN against the same candidate:

- cd core && go test -count=1 ./internal/control
- cd core && go test -count=20 ./internal/control
- cd core && go test -race ./internal/control
- cd core && go test ./...
- cd core && go vet ./...
- make check
- make task-check TASK=TASK-0028
- git diff --check

Temporary SQLite with two bounded connections and deterministic barriers is hermetic and is expected to support focused-rerun for concurrency/rollback/all-edge truth. QA_PLAN must assign each case exactly one qa_execution_mode with reason and fail-closed condition. No real privilege, deployment, external service, or live-e2e environment is required; if the fixture cannot deterministically force the conflict/rollback, the case is blocked rather than replaced by evidence-review.

## Stop conditions and exclusions

Stop before or during DEV if TASK-0027 Store API is not merged/stable, migration cannot express enforceable owner uniqueness, transition semantics conflict with docs/02 or docs/03, deterministic two-connection conflict/rollback cannot be forced, readable production exceeds 240 lines, or required behavior enters TASK-0029. Exclude expected-version contract/progress update, immutable contract/progress history and recovery, runtime schema validation, owner transfer, Task-tree cascade, resource cleanup, Agent Run concurrency, CLI/transport, and new dependencies.

Any candidate modification after QA is persistence/concurrency/lifecycle/fail-closed sensitive and therefore cannot use qa_carry_forward. Main selects focused or full rerun from the affected case set. Reviewer/QA minor fixes remain subject to the repository role rules, but Main alone integrates to main.

## Main Agent review

- [x] TASK-0007 0007-B is fully represented and TASK-0029 remains excluded.
- [x] Transition table and owner retention/release invariants match docs/02 and docs/03.
- [x] 2 implementation files / 225 lines / 2 points and 240-line stop are approved.
- [x] sol-high DEV profile is approved for persistence/concurrency/lifecycle risk.
- [x] Independent TASK-first QA_PLAN is approved before DEV.
- [x] DEV start is approved.
