---
task_id: "TASK-0027"
change_class: "product"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-07-20"
approved_dev_profile: "sol-high"
approved_dev_profile_reason: "SQLite migration、永続transaction、rollback、Schema snapshotを同時に扱う高リスク実装のため"
approved_dev_profile_risk_signals: ["persistence", "concurrency", "schema"]
planning_reviewed_by: ""
planning_review_decision: "pending"
planning_reviewed_at: ""
planned_implementation_files: 2
planned_implementation_lines: 252
estimate_points: 2
---

# TASK-0027 PLAN — Control durable foundation

## Decision and DEV gate

This PLAN implements only TASK-0007 PLAN 0007-A: minimum pure-Go SQLite migration and atomic root-Task creation. It is PLAN evidence only. DEV starts only after Main approval of this PLAN and an independent QA_PLAN. Main alone owns Git, publication, and merge.

## Acceptance mapping

| Acceptance | Observable evidence |
| --- | --- |
| New DB migration | Temporary DB opens with one current schema version. |
| Reopen | Same version close/reopen does not rerun migration and returns the creation read model. |
| Atomic creation | Task, owner Agent, workspace ref, contract v1 JSON snapshot with schema ID/revision/digest, progress v0, and events sequence 1/2 appear together. |
| Forced failure | Injected SQL failure leaves no Task, owner, workspace, contract, progress, or event row. |
| Storage boundary | WAL, foreign keys, busy timeout; open/migration failure is typed and fail-fast. |

## Design

core/internal/control/store.go exposes a Control-owned Store/open port, typed create input/read model, and typed conflict/storage errors. Open configures SQLite and applies an idempotent version-checked embedded migration. Create starts one transaction, inserts current Task and associated owner/workspace, immutable contract v1 JSON bytes plus schema ID/revision/digest, progress v0, then append-only TaskCreated sequence 1 and OwnerAssigned sequence 2. Commit is the sole success point; every error rolls back. Reads reconstruct the creation read model.

Control alone writes control.db. Other Planes do not open it and this Task claims no cross-Plane atomicity. Contract JSON is immutable reference data: schema meaning/validation and schema changes are excluded.

### Invariants

- Successful create observes every required row; failed create observes none.
- Initial events are exactly sequence 1 TaskCreated then sequence 2 OwnerAssigned.
- Existing current schema version is accepted; unsupported/newer/failed migration fails before create.
- No auto retry or silent overwrite handles storage conflict.
- Every Store open configures WAL, foreign keys, and busy timeout.

### Explicit exclusions

TASK-0028 owns owner exclusivity, lifecycle transitions, Task updates, and owner release. TASK-0029 owns versioned contract/progress mutation, historical recovery semantics, and runtime schema validation. This Task excludes CLI/transport, Inbox/Outbox, Agent Run, schema edits, and CGO drivers.

## Planned files and estimate

| File | Type | Lines | Change |
| --- | --- | ---: | --- |
| core/go.mod | config | 18 | Add `modernc.org/sqlite v1.38.2`と間接依存; pure-Go、Go 1.23互換、BSD-3-Clause。 |
| core/go.sum | generated dependency lock | 0 | Go module checksum。見積対象外。 |
| core/internal/control/store.go | implementation | 234 | Open/migration、pragmas、models、atomic create/read、rollback、typed errors、故障注入seam。 |

core/internal/control/store_test.go is required test evidence but excluded from production SLOC. 初期見積190行/1 pointはmigration SQL、typed model/error、故障注入seamを過小評価していた。candidate実測は2 files / 252 physical linesで、file_score=ceil(2/3)=1、line_score=ceil(252/200)=2、therefore estimate is 2 points. 製品Goコード全体はcandidate時点で約693行で、1,500行目標/1,800行hard limit内に十分収まる。可読性やfail-closed検査を削らず実測へ補正する。

## Execution

1. Preflightで`modernc.org/sqlite v1.38.2`の取得、`go 1.23.0`宣言、BSD-3-Clause `LICENSE`、module checksumを確認済み。latest v1.54.0はGo 1.25を要求するため採用しない。DEVはv1.38.2を固定し、CGOへ置換しない。
2. Add driver and Store migration/open boundary.
3. Implement one transaction for create/read and typed failure paths.
4. Add temporary-DB tests: migrate, reopen, create/read, forced rollback with zero partial rows.
5. DEV runs focused tests, formatting, vet, full core tests, make check, and git diff --check. Reviewer and QA independently evaluate the same candidate.

## Verification plan

- cd core && go test ./internal/control
- cd core && go test ./...
- cd core && go vet ./...
- gofmt check for changed Go paths, make check, and git diff --check.
- Tests use temporary SQLite only: no privilege, network, external service, or live-e2e fixture. Focused rerun is valid only while the fixture remains hermetic, deterministic, and bounded.

## Risks and stops

Stop and return to PLAN if the driver cannot be obtained/licensed for Go 1.23, migration requires an unplanned dependency/SQL directory, candidate exceeds the remeasured 252 physical implementation lines without a concrete acceptance need, contract JSON validation becomes necessary, or TASK-0028/0029 behavior is required. Do not compress errors, skip rollback proof, or pull later scope into this slice.

## Main Agent review

- [x] TASK-0007 0007-A is isolated from TASK-0028 and TASK-0029.
- [x] Two implementation paths, remeasured 252 physical lines, and two-point estimate are valid. `go.sum`は生成lockとして見積対象外。初期190行/1 pointはrequirement_gapとしてMainが補正した。
- [x] Acceptance is testable with temporary SQLite.
- [x] Independent QA_PLAN is approved before DEV.
- [x] DEV is approved.
