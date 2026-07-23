---
task_id: "TASK-0033"
change_class: product
status: approved
planner_agent: "planner-agent-terra-medium"
approved_dev_profile: "sol-high"
approved_dev_profile_reason: "二重repositoryのcutover、sparse worktree、main証跡トランザクション、Git競合/rebase retry、GitHub CI/auto-merge、post-merge書込みを同時に変更し、権限・状態遷移・fail-closed境界を横断して検証する必要があるため"
approved_dev_profile_risk_signals:
  - repository_topology
  - git_concurrency
  - authorization_boundary
  - ci_automation
  - state_transition
  - cross_cutting
approved_by: "main-agent-sol-high"
approved_at: "2026-07-23T10:04:33+10:00"
planning_reviewed_by: ""
planning_review_decision: "pending"
planning_reviewed_at: ""
classification_approved_by: ""
classification_approved_at: ""
classification_approval_reason: ""
planned_implementation_files: 13
planned_implementation_lines: 1165
estimate_points: 8
---

# PLAN: TASK-0033

## Classification and implementation contract

This is a product change: it moves tracked runtime automation, schemas, hooks,
CI, and externally observable Task delivery from two repositories into one.
The DEV profile is `sol-high` because the change crosses repository topology,
Git concurrency, CI authorization, and the protected PLAN/DEV/REVIEW/QA
contract. `planned_implementation_files` and
`planned_implementation_lines` count only implementation code, operational
Schema, and executable configuration; tests, fixtures, documentation, Task
evidence, migrated content, and generated output are intentionally excluded.
`estimate_points: 8` is the maximum scale band for this 13-file, 1,165-line
cross-cutting implementation estimate.

## 変更予定

| 実装対象 | 種別 | 見積行 | 具体化 |
|---|---:|---:|---|
| `Makefile` | executable configuration | 180 | unified rootの`task-start`/`task-pr`/`sync`/検査入口とmain evidence transactionの起動を定義する。 |
| `.github/workflows/main-evidence.yml` | CI configuration | 105 | main direct pushをread-only evidence validationへ限定し、製品path混入を拒否する。 |
| `.github/workflows/pr-ci.yml` | CI configuration | 85 | merge candidateの`make check`、Task checker、scope checkerをrequired checksとして実行する。 |
| `.github/workflows/post-merge.yml` | CI configuration | 110 | merged `pull_request.closed`だけを契機に、Task/PR concurrency付きの冪等状態記録を行う。 |
| `.githooks/pre-commit`、`.gitignore` | hook/configuration | 37 | evidence actionのscope逸脱を早期拒否し、単一repoのworktree生成物を除外する。 |
| `.codex/config.toml` | configuration | 24 | main root正本とsparse Task worktreeのroutingを固定し、外部work adapterを除去する。 |
| `project.yaml` | executable configuration | 25 | single-rootの既定branch、worktree配置、merge方式、証跡正本とbootstrap/freeze境界を固定する。 |
| `viewer/**` | implementation code | 95 | 統合先のbacklog/Task/Wiki/Lap30パスを読むviewer入力とindex表示を更新する。 |
| `schemas/operations/**` | Schema | 145 | 旧work運用Schemaを製品Schemaと衝突しないnamespaceへ配置し、Task/Wiki/state/digest検査を定義する。 |
| `scripts/task/**` | implementation code | 295 | sparse checkout、allowlist+lock+rebase retry evidence commit、bootstrap validator/composite candidate binding、PR/CI/postmerge/sync/migration validatorを実装する。 |
| `templates/task/**` | executable template/configuration | 44 | unified evidence fields、candidate digest、postmerge idempotency keyを生成する。 |
| `package.json` | executable configuration | 20 | unified validator/viewer/task scriptをpackage commandへ配線する。 |
| 実装小計 |  | **1,165** | **13 implementation files/units。テスト・fixture・docs・evidence・移行済みcontent・生成物はこの算定外。** |

The implementation uses `agent-harness` at REF-1 as the only repository after
cutover. Its main worktree owns shared evidence. Each Task code worktree is a
branch worktree under `worktrees/TASK-*` with per-worktree sparse-checkout that
excludes all main-managed operational paths. No child agent stages, commits, or
pushes: the main-owned task commands perform the bounded Git transactions.

Before DEV, Main must reconcile the pending GitHub dependency: authenticated
PR actor, repository auto-merge permission/ruleset, and the exact required CI
check names. A changed value, unavailable privilege, or a ruleset that cannot
require the planned checks blocks the CI/auto-merge portion; it must not be
silently replaced by manual merge or a weaker check.

## Ordered design

1. Establish the one-repository layout and its bootstrap contract without
   history import. Put the single-root branch/worktree/merge settings in
   `project.yaml`; place operational schemas under `schemas/operations/`,
   rather than merging their names into existing product schema namespaces.
   Replace two-root documentation/configuration with the main-root/sparse-
   worktree model, regenerate indexes only from fixed inputs, and remove
   runtime `WORK_ROOT` and `../agent-harness-work` use.
2. Build deterministic validators and hooks for the unified layout before
   making write entry points. They validate migrated counts and source/target
   digests, YAML/JSON Schema, Task/Wiki/link/state consistency, generated index
   freshness, sparse exclusions, action scope, and forbidden external-root
   references. The validator produces a bootstrap manifest that binds the
   `backlog.yaml`, `tasks/**`, `wiki/**`, `lap30/**`, and `project.yaml` source
   digests. A mismatch leaves both the bootstrap and cutover uncommitted.
3. After DEV fixes that validator, Main runs it against REF-2 and creates the
   validated bootstrap evidence commit directly on product `main`: the
   manifest and migrated `backlog.yaml`, 32 historical Done Tasks plus current
   TASK-0033 evidence, Wiki/Decisions, Lap30, and `project.yaml` are the
   commit's permitted scope. Main then records its commit/digest, freezes
   `agent-harness-work` against every new evidence write, and treats product
   `main` as the only evidence authority. A validation, scope, commit, or push
   failure leaves the old repository unfrozen and prevents the code PR path.
4. Rebase the Task code branch onto the successful bootstrap commit, update its
   sparse worktree from that main, and remove all main-managed evidence paths
   from its PR diff. The resulting evaluation unit is a composite candidate:
   immutable `(code_commit, code_managed_path_digest)` paired with immutable
   `(bootstrap_evidence_commit, bootstrap_evidence_digest)`. Review and QA
   must independently record and evaluate the same four values; subsequent
   review/QA result commits are main evidence, not candidate content. Any
   missing pairing, rebase conflict, stale main, or evidence path in the PR
   fails closed; there is no TASK-0033 scope exception.
5. Implement main-owned Task lifecycle commands. `task-start` requires a clean,
   current main, creates and validates Task/backlog material, commits and
   pushes that evidence, then creates `task/TASK-*` and its sparse worktree.
   On any creation or publish failure it removes only resources created in this
   invocation and reports a recoverable, non-published state; individual
   publish commands cannot bypass this all-or-stop entry point.
6. Add one `evidence-commit` transaction per declared action. It resolves the
   explicit main root, obtains a common local lock, validates its action
   allowlist and shared-path ownership, performs lightweight checks, stages
   only the permitted paths, commits, and pushes. A non-fast-forward performs
   at most two fetch/rebase/revalidate/retry cycles. Conflict, dirty scope,
   exhausted retry, lock timeout, hook failure, or remote mismatch aborts with
   no broad staging and no continuation by another action. Review/QA evaluate a
   composite candidate from the sparse worktree but record results through this
   main-root transaction.
7. Separate CI and PR phases. Main push receives read-only evidence CI only;
   it rejects product-path contamination and validates evidence integrity but
   has no credentials or write step. `task-pr` independently rechecks REVIEW
   and QA PASS bindings to the same candidate commit and managed-path digest,
   rejects main-managed paths on the branch, then pushes the branch, creates a
   ready PR, and enables merge-commit auto-merge. PR CI runs required `make
   check`, target Task validation, and scope validation on the merge candidate.
8. Install a single post-merge writer. Only `pull_request.closed` with
   `merged=true` may call the post-merge evidence action. It uses a Task/PR
   concurrency key, records `merged_commit` and state only when absent, and
   otherwise exits no-op. Push CI never writes; workflows do not chain through
   `workflow_run`; bot/evidence commits are excluded from recursive work.
9. Implement local `sync` as the recovery and completion path: require clean
   main, fast-forward safely and prune remotes; stop on conflict, dirty tree,
   or red PR CI; ingest each eligible HANDOVER through its digest receipt using
   local Codex credentials only; mark done, remove only clean merged worktrees
   and branches, validate, and publish through the evidence transaction.
   `FAST=1` performs only synchronization and pruning, explicitly retaining
   Wiki ingestion and done state for a later normal run.
10. Cut over only after source/target count and digest reports pass for the 32
    historical Done Tasks plus current TASK-0033 evidence, backlog,
    Wiki/Decision content, Lap30, and viewer. Run the unified checks and
    fixture-based lifecycle/CI tests. Archive `autotaker/kakesu-work` only
    after successful cutover evidence. Before archive, rollback first disables
    the PR/auto-merge path, restores the recorded old-work write authority,
    then reverts the bootstrap commit and discards/recreates the code branch
    so no evidence returns to its PR; it never copies a partial main state back
    to work. After archive, restore the archived repository before the same
    coordinated bootstrap/code rollback. No historical published evidence is
    rewritten.

## Acceptance-condition traceability

| AC | Design decision and change paths | Sequence and failure handling |
|---|---|---|
| AC-1 | Unified root in `Makefile`, `scripts/task/**`, `docs/development/**`, `.codex/**`, `project.yaml`, `AGENTS.md`; bootstrap-migrate `backlog.yaml`, `tasks/**`, `wiki/**`, `lap30/**`, `viewer/**`, `schemas/operations/**`. | Steps 1–3. External-root reference scan or schema/link/index/source-digest failure blocks bootstrap and cutover; no compatibility adapter remains at runtime. |
| AC-2 | Atomic `task-start` in `Makefile` and `scripts/task/**`, with templates and hooks. | Step 5. Validate and publish evidence before sparse worktree creation; any later failure rolls back only invocation-created branch/worktree and reports the required recovery action. |
| AC-3 | Per-worktree sparse patterns, explicit main-root routing, bootstrap freeze, action allowlists, lock/retry transaction in `scripts/task/**`; scope hook in `.githooks/**`. | Steps 3–6. Bootstrap failure leaves work writable; after bootstrap, forbidden checkout/PR evidence path, unowned shared path, dirty state, lock/race/conflict, or exhausted retry fails closed without broad staging/commit. |
| AC-4 | Read-only main workflow plus operational validators under `.github/**`, `scripts/task/**`, `schemas/operations/**`. | Step 7. Product-path diff, invalid evidence, or attempted CI write fails CI; workflow filters and no-write design prevent recursion. |
| AC-5 | Composite candidate binding—code commit/digest plus bootstrap evidence commit/digest—in Task evidence; `task-pr` and PR scope validation in `scripts/task/**`, `.github/**`, `tasks/**`. | Steps 4 and 7. Missing/mismatched paired evidence, a stale/rebase-conflicted code branch, or any main-managed PR path rejects PR creation/auto-merge. |
| AC-6 | Required PR workflow checks in `.github/workflows/pr-ci.yml`; command wiring in `Makefile`. | Step 7. Any failed/pending required check prevents auto-merge; unavailable required-check/ruleset configuration remains dependency-blocked. |
| AC-7 | Idempotent closed-and-merged workflow and post-merge action in `.github/**`, `scripts/task/**`, Task schema/state validators. | Step 8. Non-merged closure is ignored; existing merge record is no-op; lock/concurrency/write failure leaves state pending for a safe retry, never loops. |
| AC-8 | `sync` orchestration, receipt-based local Wiki ingest, cleanup guards in `Makefile`, `scripts/task/**`, `wiki/**`, `tasks/**`. | Step 9. Dirty/conflict/red-CI stops before cleanup or done state; `FAST=1` cannot ingest or mark done; repeat normal sync is receipt/idempotency-safe. |
| AC-9 | Bootstrap snapshot import and migration manifest/checkers in `backlog.yaml`, `tasks/**`, `wiki/**`, `lap30/**`, `viewer/**`, `project.yaml`, `schemas/operations/**`, docs. | Steps 2–4 and 10. Count/digest or semantic validation mismatch blocks bootstrap/archive; rollback restores the frozen work authority before reverting bootstrap and recreating the code branch. |

## Verification and gate plan

DEV will add hermetic temporary-repository tests for sparse checkout exclusion,
task-start rollback, action allowlists, lock contention, retry cap, candidate
digest mismatch, PR scope rejection, loop/no-op behavior, and `sync`/`FAST=1`
idempotency. It will run the unified `make check`, the target Task checker,
the new operational/migration validators, and `git diff --check` on the fixed
candidate. The required GitHub permission/ruleset and actual auto-merge are
environment-dependent: they stay explicit `live-e2e` cases for independent QA
after dependency reconciliation, rather than being inferred from mocked CI.

Independent REVIEW and QA begin from the same composite candidate: code commit
and managed-path digest plus bootstrap evidence commit and evidence digest.
Any failure is classified as implementation defect, QA plan defect,
requirement/design gap, environment issue, or regression; Main chooses the
return gate. Auto-merge and archive are never treated as a remedy for a failed
local or CI check.

## Scope boundaries

No work-repository Git history is imported. Existing Done Task meaning,
published Wiki/Decision conclusions, and Lap30 event meaning remain immutable;
only their location, verified indexes, and future write route change. GitHub
Actions receives no ChatGPT authentication material and performs no AI Wiki
ingestion. The implementation retains independent PLAN/REVIEW/QA, fixed role
contracts, and Main-owned integration decisions.
