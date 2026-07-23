---
task_id: "TASK-0029"
change_class: "product"
status: approved
planner_agent: "planner-agent-terra-medium"
approved_by: "main-agent-sol-high"
approved_at: "2026-07-20"
approved_dev_profile: "sol-high"
approved_dev_profile_reason: "SQLite migration、expected-version CAS、immutable history、並行更新、rollback、close/reopen recoveryを扱う高リスク実装のため"
approved_dev_profile_risk_signals: ["persistence", "concurrency", "schema-reference", "recovery", "fail-closed"]
planning_reviewed_by: ""
planning_review_decision: "pending"
planning_reviewed_at: ""
planned_implementation_files: 4
planned_implementation_lines: 425
estimate_points: 3
---

# TASK-0029 PLAN — Versioned state and recovery

## Decision and DEV gate

TASK-0007 PLAN 0007-Cだけを、TASK-0027/0028後の最終1〜2 Lap product sliceとして実装する。本PLANは設計証跡でありDEV承認ではない。DEVはsol-high、Planner/Reviewer/QAは別Terra/medium roleとする。Mainが本PLANと独立TASK-first QA_PLANを承認し、依存preflightを完了するまでDEVを開始しない。

TASK-0028は本PLAN作成時点で未mergeである。DEV開始条件は、TASK-0027とTASK-0028が独立REVIEW/QA PASS後にMainへmerge済み、各candidate_treeとmerge_treeが一致し、実際にmergeされたStore/lifecycle API、migration version、typed conflict、event sequence契約を再確認できること。想定と差があればコードで吸収せず、本PLANとQA_PLANを改訂・再承認する。

DEVがcandidate_commit/candidate_treeを固定した後、ReviewerとQAはその同一candidateから独立かつ並行に開始する。Mainだけがmainへのmerge/push、修正後rerun判断、merge_tree照合を所有する。

## Acceptance mapping

| Condition | Observable temporary-SQLite evidence | Failure invariant |
|---|---|---|
| Contract CAS | expected current vからv+1へ更新 | stale/future/skipped/same-version replacementはcurrent/history/event不変 |
| Progress CAS | expected current vからv+1へ更新 | conflictはcurrent/history/event不変 |
| Immutable history | v1/v2/v3 snapshot/historyとschema refsを列挙 | 旧rowのUPDATE/DELETEなし |
| Atomicity | history append後またはcurrent/event更新前に故意SQL失敗 | transaction全rollback、version gapなし |
| Concurrent update | 2 connection/barrierで同じexpected versionを更新 | 一方だけcommit、敗者typed conflict |
| Recovery | close/reopen前後のfull read modelを比較 | current Task/owner、contract/progress、history、event sequenceが一致 |

## Versioned update contract

### Contract update

Inputはtask ID、expected contract version、new version、immutable contract JSON bytes、schema ID/revision/digestを含む。new versionはexpected+1のみ受理する。一transactionでcurrent contract pointer/versionをCAS更新し、新snapshotをappendし、対応するContractChanged eventを次sequenceでappendする。旧snapshotは更新・削除しない。JSON Schemaの意味検証は行わず、bytesとschema referenceの整合に必要な非空/形式境界だけを保存portで検査する。

### Progress update

Inputはtask ID、expected progress version、new version、progress payload、through-task-event sequence等の既存read-model metadata、schema ID/revision/digestを含む。new versionはexpected+1のみ受理する。一transactionでappend-only progress historyを追加し、current progressをCAS置換し、対応するProgressRefreshed eventを次sequenceでappendする。旧historyは更新・削除しない。

### Recovery read model

DB close/reopen後、persisted current tablesを起点にTask current state、active owner、workspace参照、current contract/progressを取得し、immutable contract snapshots、progress history、Task eventsをversion/sequence順で付加する。event replayだけでcurrentを再生成しない。currentとhistoryのversion/digest/schema referenceが矛盾、欠落、重複、非単調ならtyped storage/corruption errorでfail-fastし、部分read modelを返さない。

## Invariants

- Contract/progress更新はそれぞれexpected current version一致かつ単調+1だけを受理する。
- history append、current update、event appendは同じtransactionで全件commit/rollbackする。
- 競合、busy、stale versionを自動retryまたはsilent overwriteしない。
- event sequenceは成功transactionごとに一つ増え、失敗時にgapを作らない。
- schema ID/revision/digestはsnapshot/history/eventへ不変に結び付ける。
- Recoveryはowner/lifecycleのTASK-0028不変条件を変更せず、current/history矛盾を隠さない。

## Alternatives rejected

- Mutable contract/progress one-row only: 旧versionと監査履歴を失う。
- Event replay only: current read model復旧を全履歴再生へ依存させる。
- Last-write-wins/autoretry: stale writerを成功に見せlost updateを起こす。
- One generic patch API: contract/progress固有のversionとhistory境界を曖昧にする。
- Runtime Schema validator addition: library/resolution設計が未承認でsliceを超える。

## Planned implementation boundary

| File | Type | Estimated lines | Change |
|---|---|---:|---|
| core/internal/control/store.go | implementation | 50 | migration v3、contract/progress current+history tables/constraints、schema refs |
| core/internal/control/versioned.go | implementation | 190 | typed inputs/models、contract/progress expected-version CAS、atomic history/current/event updates |
| core/internal/control/recovery.go | implementation | 100 | close/reopen full read model load、ordering/integrity checks、typed corruption error |
| core/internal/control/lifecycle.go | implementation | 2 | v3 TaskEvent schema referenceを既存lifecycle read modelでも復元する互換読取 |

Initial estimate was 3 files / 340 lines. Recoveryの明示integrity error経路が100→127行、merged v2 lifecycle read modelのschema-ref互換が2行必要と判明したため、Mainがdraft実測4 files / 372 readable linesへ補正した。Candidate直前に正本`docs/02-task-lifecycle.md`の`ProgressRefreshed`が`progress_version`とTask/Agent Run両watermarkを要求することを検出し、受入条件の曖昧化やコード圧縮を避けるためMain裁量で400行へ一度だけ補正した。最初のcandidateでREVIEW/QAがwatermark後退と未来Task sequence受理を1件の具体的findingとして検出し、Lap 2をその修正だけに限定して425行へ補正した。Required tests live in core/internal/control/versioned_test.go and core/internal/control/recovery_test.go and are excluded from production SLOC. file_score=ceil(4/3)=2、line_score=ceil(425/200)=3、therefore final measurement is 3 points. Readable working range is 320–425 lines; above 425 stops for revision or split. Corrected candidate後の全体product Go physical SLOCは1,297行で1,500目標/1,800 hard limit内。Do not fit by semicolon packing, generic untyped maps, combined contract/progress errors, deleted integrity checks, or cryptic names.

## 1–2 Lap execution

### Lap 1 — complete candidate

1. Preflight confirms TASK-0027/0028 merge and tree equality, then maps this PLAN to the actual Store API, migration version, transaction helper, error taxonomy, and event sequence port. Any mismatch stops before DEV.
2. DEV adds migration v3 and separately readable contract/progress CAS ports, then recovery loader/integrity checks.
3. DEV adds table-driven normal/stale/future/skipped/forced-failure cases, deterministic two-connection concurrency, multi-version old-history assertions, and close/reopen byte-equivalent read-model comparison.
4. DEV records candidate commit/tree, commands, temporary fixture/cache conditions, exit, artifact digests, negative-detection evidence, SLOC, retries, and classification. Lap 1 ends only with a complete reviewable candidate.

### Lap 2 — bounded finding correction only

Lap 2 is exceptional and allowed only for one or two concrete REVIEW/QA findings against the complete Lap-1 candidate, no API redesign/new dependency/schema-validator/new fixture/scope change, and a bounded correction demonstrably complete within its first 20 minutes. Otherwise split/replan. No Lap 3.

## Verification plan

DEV runs focused checks first. Reviewer and QA independently execute approved QA modes against the same candidate:

- cd core && go test -count=1 ./internal/control
- cd core && go test -count=20 ./internal/control
- cd core && go test -race ./internal/control
- cd core && go test ./...
- cd core && go vet ./...
- make check
- make task-check TASK=TASK-0029
- git diff --check

Temporary SQLite with deterministic barriers, close/reopen, and injected transaction failures is hermetic, deterministic, and bounded, so focused-rerun may establish acceptance truth when the QA_PLAN explicitly assigns it. If actual close/reopen, concurrent conflict, or forced rollback cannot be deterministically reproduced, that case is blocked; evidence-review cannot substitute. No privilege, external service, deployment, or live-e2e environment is expected.

## Stop conditions and exclusions

Stop if TASK-0028 is unmerged, its API/schema/event sequence differs from this assumption, migration v3 cannot preserve existing 0027/0028 data, concurrent CAS/rollback cannot be deterministic, current/history integrity requires runtime Schema validation/new dependency, readable production exceeds the remeasured 425-line stop, or recovery requires backup/replication/event replay framework. Exclude owner/lifecycle semantic changes, generic Task updates, schema changes/validator, transport/CLI, Inbox/Outbox, Plane-crossing atomicity, backup/PITR/replication, and all behavior beyond contract/progress versioned state.

Changes after QA touch persistence/concurrency/schema-reference/recovery/fail-closed behavior and therefore are ineligible for qa_carry_forward. Main selects affected focused reruns or full rerun. Reviewer/QA minor fixes follow repository rules, but only Main integrates to main.

## Main Agent review

- [x] TASK-0007 0007-C is fully represented without TASK-0027/0028 scope regression.
- [x] TASK-0028 merge/API/tree precondition is explicit and verified before DEV.（merge `63ffb0e`、tree `c7050bd`。schema v2、`Store`/`ConflictError`、payload付きevent sequence APIを再照合済み）
- [x] Contract/progress CAS, immutable history/current, recovery, and corruption fail-fast are testable.
- [x] Remeasured 4 implementation files / 418 lines / 3 points and 425-line stop are approved. The bounded Lap 2 adjustment validates canonical progress watermarks; no scope expansion.
- [x] sol-high DEV profile is approved for persistence/concurrency/recovery risk.
- [x] Independent TASK-first QA_PLAN is approved before DEV.
- [x] DEV start is approved.（TASK-0028依存merge/tree一致と実API再照合後）
