import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import { spawnSync } from "node:child_process";
import { createHash } from "node:crypto";
import { checkTask } from "./check-task.mjs";
import { acquireWorkRepoLock, dateInTimezone, estimatePoints, git, replaceTemplate, resolveInside, workRepoLockDir } from "./lib.mjs";
import { rollbackWorkRepository, validateDevSelection } from "./agent-routing.mjs";
import { runWorkConfigSync } from "./run-work-config-sync.mjs";

test("estimatePoints uses implementation file and line scores", () => {
  assert.equal(estimatePoints(2, 80), 1);
  assert.equal(estimatePoints(5, 250), 2);
  assert.equal(estimatePoints(8, 500), 3);
  assert.equal(estimatePoints(4, 900), 5);
  assert.equal(estimatePoints(20, 1200), 8);
});

test("estimatePoints rejects work above the scale", () => {
  assert.throws(() => estimatePoints(40, 3000), /split the task/);
  assert.throws(() => estimatePoints(-1, 10), /non-negative integers/);
  assert.throws(() => estimatePoints(1.5, 10), /non-negative integers/);
  assert.throws(() => estimatePoints(1, "200"), /non-negative integers/);
});

test("resolveInside rejects absolute and traversing paths", () => {
  assert.equal(resolveInside("/tmp/work", "tasks/TASK-0001-a"), "/tmp/work/tasks/TASK-0001-a");
  assert.throws(() => resolveInside("/tmp/work", "../escape"), /escapes/);
  assert.throws(() => resolveInside("/tmp/work", "/tmp/escape"), /relative path/);
});

function writeTaskEvidence(taskDir, filename, metadata, body = "") {
  const yaml = Object.entries(metadata).map(([key, value]) => `${key}: ${JSON.stringify(value)}`).join("\n");
  fs.writeFileSync(path.join(taskDir, filename), `---\n${yaml}\n---\n${body}`);
}

const SAFETY_CHECK_KEYS = ["process_tests", "contract_scope", "docs_lint", "make_check"];

function safetyCheckDigest(candidateTree, mergeTree, checks) {
  const normalized = [
    `candidate_tree=${candidateTree}`,
    `merge_tree=${mergeTree}`,
    ...SAFETY_CHECK_KEYS.map((key) => `${key}=${checks[key]}`),
  ].join("\n");
  return createHash("sha256").update(`${normalized}\n`).digest("hex");
}

function createDoneTaskFixture({
  taskId = "TASK-0090",
  changeClass,
  productPath = false,
  changedPaths,
  renameSpoof = false,
  copySpoof = false,
  nonNoFf = false,
  legacyTask0024 = false,
  safetyContractV2 = false,
  plannedPaths,
  generatedPaths,
} = {}) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "task-done-gate-"));
  const repository = path.join(root, "product");
  const taskDir = path.join(root, "tasks", `${taskId}-fixture`);
  fs.mkdirSync(repository, { recursive: true });
  fs.mkdirSync(taskDir, { recursive: true });
  git(repository, ["init", "-b", "main"]);
  git(repository, ["config", "user.name", "fixture"]);
  git(repository, ["config", "user.email", "fixture@example.invalid"]);
  fs.writeFileSync(path.join(repository, "README.md"), "baseline\n");
  if (renameSpoof || copySpoof) {
    fs.mkdirSync(path.join(repository, "docs", "development"), { recursive: true });
    fs.writeFileSync(path.join(repository, "docs", "development", "old.md"), "rename me\n");
  }
  git(repository, ["add", "."]);
  git(repository, ["commit", "-m", "baseline"]);
  git(repository, ["checkout", "-b", "task"]);
  if (renameSpoof) {
    git(repository, ["mv", "docs/development/old.md", "docs/development/new.md"]);
  } else if (copySpoof) {
    fs.copyFileSync(path.join(repository, "docs", "development", "old.md"), path.join(repository, "docs", "development", "copy.md"));
    git(repository, ["add", "docs/development/copy.md"]);
  } else {
    const fixturePaths = changedPaths ?? [productPath ? "scripts/product.mjs" : "docs/development/contract.md"];
    for (const changedPath of fixturePaths) {
      fs.mkdirSync(path.dirname(path.join(repository, changedPath)), { recursive: true });
      fs.writeFileSync(path.join(repository, changedPath), "changed\n");
      git(repository, ["add", changedPath]);
    }
  }
  git(repository, ["commit", "-m", "candidate"]);
  const candidateCommit = git(repository, ["rev-parse", "HEAD"]);
  const candidateTree = git(repository, ["rev-parse", "HEAD^{tree}"]);
  git(repository, ["checkout", "main"]);
  git(repository, nonNoFf ? ["merge", "--ff-only", "task"] : ["merge", "--no-ff", "-m", "merge", "task"]);
  const mergedCommit = git(repository, ["rev-parse", "HEAD"]);
  const mergeTree = git(repository, ["rev-parse", "HEAD^{tree}"]);
  fs.writeFileSync(path.join(root, "project.yaml"), "repository_path: product\ndefault_branch: main\n");

  const exclusion = legacyTask0024
    ? "### 対象外\n\n- 製品コード、製品test、runtime/build設定、製品Schema、製品依存、製品挙動。\n"
    : "### 対象外\n\n- 製品コード、test、runtime/build設定、Schema、製品依存、生成製品入力/成果物、外部観測可能な挙動を変更しない。\n";
  writeTaskEvidence(taskDir, "TASK.md", { task_id: taskId }, exclusion);
  const planMetadata = {
    task_id: taskId,
    change_class: changeClass ?? "product",
    status: "approved",
    planner_agent: "planner",
    approved_by: "main",
    approved_at: "2026-07-20T00:00:00Z",
    approved_dev_profile: "sol-high",
    approved_dev_profile_reason: "fixture",
    approved_dev_profile_risk_signals: ["cross_cutting"],
    planned_implementation_files: 1,
    planned_implementation_lines: 1,
    estimate_points: 1,
    planning_reviewed_by: "reviewer",
    planning_review_decision: "pass",
    planning_reviewed_at: "2026-07-20T00:00:00Z",
    classification_approved_by: "main",
    classification_approved_at: "2026-07-20T00:00:00Z",
    classification_approval_reason: "fixture classification",
  };
  if (safetyContractV2) {
    planMetadata.safety_contract_version = 2;
    planMetadata.safety_contract_planned_paths = plannedPaths ?? ["docs/development/contract.md"];
    planMetadata.safety_contract_generated_paths = generatedPaths ?? [];
  }
  writeTaskEvidence(taskDir, "PLAN.md", planMetadata);
  const qaPlanMetadata = {
    task_id: taskId,
    change_class: changeClass ?? "product",
    status: "approved",
    qa_agent: "qa",
    approved_by: "main",
    approved_at: "2026-07-20T00:00:00Z",
    implementation_reviewed_at: "2026-07-20T00:00:00Z",
    expectation_changed: false,
  };
  writeTaskEvidence(taskDir, "QA_PLAN.md", qaPlanMetadata);
  const reviewMetadata = {
    task_id: taskId,
    reviewer_agent: "reviewer",
    reviewed_commit: candidateCommit,
    decision: "pass",
    make_check: "pass",
  };
  writeTaskEvidence(taskDir, "REVIEW_RESULT.md", reviewMetadata);
  const qaResultMetadata = {
    task_id: taskId,
    qa_agent: "qa",
    tested_commit: mergedCommit,
    tested_at: "2026-07-20T00:00:00Z",
    decision: "pass",
  };
  writeTaskEvidence(taskDir, "QA_RESULT.md", qaResultMetadata);
  const safetyChecks = Object.fromEntries(SAFETY_CHECK_KEYS.map((key) => [key, "pass"]));
  const handoverMetadata = {
    task_id: taskId,
    status: "complete",
    completed_at: "2026-07-20T00:00:00Z",
    safety_checks: safetyChecks,
    safety_checked_at: "2026-07-20T00:00:00Z",
    safety_check_digest: safetyCheckDigest(candidateTree, mergeTree, safetyChecks),
    safety_candidate_tree: candidateTree,
    safety_merge_tree: mergeTree,
  };
  writeTaskEvidence(taskDir, "HANDOVER.md", handoverMetadata);
  fs.mkdirSync(path.join(root, "wiki", "ingestions"), { recursive: true });
  fs.writeFileSync(path.join(root, "wiki", "ingestions", `${taskId}.json`), "{}\n");
  const task = {
    id: taskId,
    status: "done",
    estimate_points: 1,
    task_dir: path.relative(root, taskDir),
    merged_commit: mergedCommit,
    assignees: { main: "main", planner: "planner", dev: "dev-sol-high", reviewer: "reviewer", qa: "qa" },
  };
  if (changeClass !== undefined) task.change_class = changeClass;
  return { root, taskDir, backlog: { tasks: [task] }, taskId, planMetadata, qaPlanMetadata, reviewMetadata, qaResultMetadata, handoverMetadata };
}

function createSafetyPreflightFixture(planOverrides = {}) {
  const taskId = "TASK-0091";
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "task-preflight-"));
  const taskDir = path.join(root, "tasks", `${taskId}-fixture`);
  fs.mkdirSync(taskDir, { recursive: true });
  for (const filename of ["TASK.md", "REVIEW_RESULT.md", "QA_PLAN.md", "QA_RESULT.md", "HANDOVER.md"]) {
    writeTaskEvidence(taskDir, filename, { task_id: taskId });
  }
  const planMetadata = {
    task_id: taskId,
    change_class: "safety_contract",
    safety_contract_version: 2,
    safety_contract_planned_paths: ["docs/development/contract.md"],
    safety_contract_generated_paths: ["docs/99-glossary-index.md"],
    ...planOverrides,
  };
  for (const [key, value] of Object.entries(planMetadata)) {
    if (value === undefined) delete planMetadata[key];
  }
  writeTaskEvidence(taskDir, "PLAN.md", planMetadata);
  const backlog = { tasks: [{ id: taskId, status: "plan", change_class: "safety_contract", task_dir: path.relative(root, taskDir) }] };
  return { root, backlog, taskId };
}

test("safety_contract v2 preflight accepts unique declared planned and generated paths", async (t) => {
  const valid = createSafetyPreflightFixture();
  try {
    assert.deepEqual(checkTask(valid.root, valid.backlog, valid.taskId, { phase: "preflight" }), []);
  } finally {
    fs.rmSync(valid.root, { recursive: true, force: true });
  }
  for (const [name, overrides, expected] of [
    ["missing declaration", { safety_contract_generated_paths: undefined }, "requires safety_contract_generated_paths"],
    ["unapproved planned path", { safety_contract_planned_paths: ["scripts/product.mjs"] }, "unapproved path"],
    ["duplicate declaration", { safety_contract_planned_paths: ["docs/development/contract.md", "docs/development/contract.md"] }, "duplicate"],
    ["duplicate across declarations", { safety_contract_planned_paths: ["docs/99-glossary-index.md"] }, "duplicate"],
    ["empty path", { safety_contract_planned_paths: [""] }, "invalid repository file path"],
    ["absolute path", { safety_contract_planned_paths: ["/docs/development/contract.md"] }, "invalid repository file path"],
    ["traversing path", { safety_contract_planned_paths: ["docs/development/../contract.md"] }, "invalid repository file path"],
    ["directory path", { safety_contract_planned_paths: ["docs/development/contract"] }, "invalid repository file path"],
    ["glob path", { safety_contract_planned_paths: ["docs/development/*.md"] }, "invalid repository file path"],
    ["version absent with v2 fields", { safety_contract_version: undefined }, "require safety_contract_version"],
    ["unknown version", { safety_contract_version: 3 }, "unsupported safety_contract_version"],
  ]) {
    await t.test(name, () => {
      const fixture = createSafetyPreflightFixture(overrides);
      try {
        assert.ok(checkTask(fixture.root, fixture.backlog, fixture.taskId, { phase: "preflight" })
          .some((error) => error.includes(expected)));
      } finally {
        fs.rmSync(fixture.root, { recursive: true, force: true });
      }
    });
  }
});

test("safety_contract v2 preflight permits docs glossary index only as declared generated path", async (t) => {
  const valid = createSafetyPreflightFixture();
  try {
    assert.deepEqual(checkTask(valid.root, valid.backlog, valid.taskId, { phase: "preflight" }), []);
  } finally {
    fs.rmSync(valid.root, { recursive: true, force: true });
  }
  for (const [name, overrides] of [
    ["glossary index as planned path", { safety_contract_planned_paths: ["docs/99-glossary-index.md"], safety_contract_generated_paths: [] }],
    ["arbitrary generated path", { safety_contract_generated_paths: ["docs/98-generated.md"] }],
  ]) {
    await t.test(name, () => {
      const fixture = createSafetyPreflightFixture(overrides);
      try {
        assert.ok(checkTask(fixture.root, fixture.backlog, fixture.taskId, { phase: "preflight" })
          .some((error) => error.includes("unapproved path")));
      } finally {
        fs.rmSync(fixture.root, { recursive: true, force: true });
      }
    });
  }
});

test("product Done gates remain required and missing change_class stays product", () => {
  const fixture = createDoneTaskFixture();
  try {
    assert.deepEqual(checkTask(fixture.root, fixture.backlog, fixture.taskId), []);
    fs.rmSync(path.join(fixture.root, "wiki", "ingestions", `${fixture.taskId}.json`));
    assert.ok(checkTask(fixture.root, fixture.backlog, fixture.taskId).some((error) => error.includes("Wiki ingestion receipt")));
    fs.mkdirSync(path.join(fixture.root, "wiki", "ingestions"), { recursive: true });
    fs.writeFileSync(path.join(fixture.root, "wiki", "ingestions", `${fixture.taskId}.json`), "{}\n");
    writeTaskEvidence(fixture.taskDir, "REVIEW_RESULT.md", { ...fixture.reviewMetadata, decision: "pending" });
    assert.ok(checkTask(fixture.root, fixture.backlog, fixture.taskId).some((error) => error.includes("review PASS")));
    fixture.backlog.tasks[0].change_class = "unknown";
    assert.ok(checkTask(fixture.root, fixture.backlog, fixture.taskId).some((error) => error.includes("change_class")));
  } finally {
    fs.rmSync(fixture.root, { recursive: true, force: true });
  }
});

test("product Done rejects each representative QA, HANDOVER, and commit omission", async (t) => {
  const mutations = {
    "QA decision": (fixture) => { fixture.qaResultMetadata.decision = "pending"; },
    "QA identity": (fixture) => { fixture.qaResultMetadata.qa_agent = "other"; },
    "QA tested commit missing": (fixture) => { delete fixture.qaResultMetadata.tested_commit; },
    "HANDOVER status": (fixture) => { fixture.handoverMetadata.status = "draft"; },
    "HANDOVER completed_at": (fixture) => { delete fixture.handoverMetadata.completed_at; },
    "merged commit missing": (fixture) => { delete fixture.backlog.tasks[0].merged_commit; },
    "reviewed commit invalid": (fixture) => { fixture.reviewMetadata.reviewed_commit = "0".repeat(40); },
    "tested commit invalid": (fixture) => { fixture.qaResultMetadata.tested_commit = "0".repeat(40); },
  };
  for (const [name, mutate] of Object.entries(mutations)) {
    await t.test(name, () => {
      const fixture = createDoneTaskFixture();
      try {
        mutate(fixture);
        writeTaskEvidence(fixture.taskDir, "REVIEW_RESULT.md", fixture.reviewMetadata);
        writeTaskEvidence(fixture.taskDir, "QA_RESULT.md", fixture.qaResultMetadata);
        writeTaskEvidence(fixture.taskDir, "HANDOVER.md", fixture.handoverMetadata);
        assert.notDeepEqual(checkTask(fixture.root, fixture.backlog, fixture.taskId), []);
      } finally {
        fs.rmSync(fixture.root, { recursive: true, force: true });
      }
    });
  }
});

test("legacy safety_contract plan remains on legacy validation without v2 opt-in", () => {
  const fixture = createDoneTaskFixture({ taskId: "TASK-0024", changeClass: "safety_contract", legacyTask0024: true });
  try {
    fs.rmSync(path.join(fixture.root, "wiki"), { recursive: true, force: true });
    writeTaskEvidence(fixture.taskDir, "REVIEW_RESULT.md", { task_id: fixture.taskId, decision: "pending" });
    writeTaskEvidence(fixture.taskDir, "QA_RESULT.md", { task_id: fixture.taskId, decision: "pending" });
    assert.deepEqual(checkTask(fixture.root, fixture.backlog, fixture.taskId), []);
    const plan = path.join(fixture.taskDir, "PLAN.md");
    fs.writeFileSync(plan, fs.readFileSync(plan, "utf8").replace('planning_review_decision: "pass"', 'planning_review_decision: "pending"'));
    assert.ok(checkTask(fixture.root, fixture.backlog, fixture.taskId).some((error) => error.includes("planning review PASS")));
  } finally {
    fs.rmSync(fixture.root, { recursive: true, force: true });
  }
});

test("safety_contract v2 Done verifies candidate diff is declared and generated paths exist", async (t) => {
  await t.test("declared candidate diff", () => {
    const fixture = createDoneTaskFixture({
      changeClass: "safety_contract",
      safetyContractV2: true,
      changedPaths: ["docs/development/contract.md", "docs/99-glossary-index.md"],
      generatedPaths: ["docs/99-glossary-index.md"],
    });
    try {
      assert.deepEqual(checkTask(fixture.root, fixture.backlog, fixture.taskId), []);
    } finally {
      fs.rmSync(fixture.root, { recursive: true, force: true });
    }
  });
  await t.test("undeclared candidate path", () => {
    const fixture = createDoneTaskFixture({
      changeClass: "safety_contract",
      safetyContractV2: true,
      changedPaths: ["docs/development/contract.md", "docs/development/other.md"],
    });
    try {
      assert.ok(checkTask(fixture.root, fixture.backlog, fixture.taskId)
        .some((error) => error.includes("undeclared path")));
    } finally {
      fs.rmSync(fixture.root, { recursive: true, force: true });
    }
  });
  await t.test("declared generated path missing", () => {
    const fixture = createDoneTaskFixture({
      changeClass: "safety_contract",
      safetyContractV2: true,
      generatedPaths: ["docs/99-glossary-index.md"],
    });
    try {
      assert.ok(checkTask(fixture.root, fixture.backlog, fixture.taskId)
        .some((error) => error.includes("generated path is missing")));
    } finally {
      fs.rmSync(fixture.root, { recursive: true, force: true });
    }
  });
});

test("safety_contract rejects product-path classification spoofing", () => {
  const fixture = createDoneTaskFixture({ changeClass: "safety_contract", productPath: true });
  try {
    const errors = checkTask(fixture.root, fixture.backlog, fixture.taskId);
    assert.ok(errors.some((error) => error.includes("product or unapproved path")));
  } finally {
    fs.rmSync(fixture.root, { recursive: true, force: true });
  }
});

test("safety_contract rejects rename, copy, and non-no-ff spoofing", async (t) => {
  for (const [name, options, expected] of [
    ["rename", { renameSpoof: true }, "product or unapproved path"],
    ["copy", { copySpoof: true }, "product or unapproved path"],
    ["non-no-ff", { nonNoFf: true }, "two-parent no-ff"],
  ]) {
    await t.test(name, () => {
      const fixture = createDoneTaskFixture({ changeClass: "safety_contract", ...options });
      try {
        assert.ok(checkTask(fixture.root, fixture.backlog, fixture.taskId).some((error) => error.includes(expected)));
      } finally {
        fs.rmSync(fixture.root, { recursive: true, force: true });
      }
    });
  }
});

test("safety_contract rejects missing or inconsistent planning evidence", async (t) => {
  const mutations = {
    "reviewer mismatch": (fixture) => { fixture.planMetadata.planning_reviewed_by = "other"; },
    "PLAN class mismatch": (fixture) => { fixture.planMetadata.change_class = "product"; },
    "QA PLAN class mismatch": (fixture) => { fixture.qaPlanMetadata.change_class = "product"; },
    "classification reason missing": (fixture) => { delete fixture.planMetadata.classification_approval_reason; },
    "approval timestamp inconsistent": (fixture) => { fixture.planMetadata.classification_approved_at = "2026-07-19T00:00:00Z"; },
  };
  for (const [name, mutate] of Object.entries(mutations)) {
    await t.test(name, () => {
      const fixture = createDoneTaskFixture({ changeClass: "safety_contract" });
      try {
        mutate(fixture);
        writeTaskEvidence(fixture.taskDir, "PLAN.md", fixture.planMetadata);
        writeTaskEvidence(fixture.taskDir, "QA_PLAN.md", fixture.qaPlanMetadata);
        assert.notDeepEqual(checkTask(fixture.root, fixture.backlog, fixture.taskId), []);
      } finally {
        fs.rmSync(fixture.root, { recursive: true, force: true });
      }
    });
  }
});

test("safety_contract rejects incomplete checks, digest mismatch, and tree mismatch", async (t) => {
  const mutations = {
    "missing exact check": (fixture) => { delete fixture.handoverMetadata.safety_checks.docs_lint; },
    "unexpected check": (fixture) => { fixture.handoverMetadata.safety_checks.extra = "pass"; },
    "failed check": (fixture) => { fixture.handoverMetadata.safety_checks.make_check = "pending"; },
    "digest mismatch": (fixture) => { fixture.handoverMetadata.safety_check_digest = "0".repeat(64); },
    "tree mismatch": (fixture) => { fixture.handoverMetadata.safety_candidate_tree = "0".repeat(40); },
  };
  for (const [name, mutate] of Object.entries(mutations)) {
    await t.test(name, () => {
      const fixture = createDoneTaskFixture({ changeClass: "safety_contract" });
      try {
        mutate(fixture);
        writeTaskEvidence(fixture.taskDir, "HANDOVER.md", fixture.handoverMetadata);
        assert.notDeepEqual(checkTask(fixture.root, fixture.backlog, fixture.taskId), []);
      } finally {
        fs.rmSync(fixture.root, { recursive: true, force: true });
      }
    });
  }
});

test("DEV gate rejects missing role separation and worktree assignment", () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "task-gate-"));
  const taskDir = path.join(root, "tasks", "TASK-0001-gate-test");
  fs.mkdirSync(taskDir, { recursive: true });
  const frontmatters = {
    "TASK.md": { task_id: "TASK-0001" },
    "PLAN.md": { task_id: "TASK-0001", status: "approved", approved_by: "main", approved_at: "2026-07-14", planned_implementation_files: 1, planned_implementation_lines: 1, estimate_points: 1 },
    "REVIEW_RESULT.md": { task_id: "TASK-0001" },
    "QA_PLAN.md": { task_id: "TASK-0001", status: "approved", approved_by: "main", approved_at: "2026-07-14" },
    "QA_RESULT.md": { task_id: "TASK-0001" },
    "HANDOVER.md": { task_id: "TASK-0001" },
  };
  for (const [filename, metadata] of Object.entries(frontmatters)) {
    const yaml = Object.entries(metadata).map(([key, value]) => `${key}: ${JSON.stringify(value)}`).join("\n");
    fs.writeFileSync(path.join(taskDir, filename), `---\n${yaml}\n---\n`);
  }
  const backlog = { tasks: [{ id: "TASK-0001", status: "dev", estimate_points: 1, task_dir: "tasks/TASK-0001-gate-test", assignees: { dev: "same", reviewer: "same", qa: "same" } }] };
  const errors = checkTask(root, backlog, "TASK-0001");
  assert.ok(errors.some((error) => error.includes("assignees.main")));
  assert.ok(errors.some((error) => error.includes("DEV Agent and Reviewer Agent")));
  assert.ok(errors.some((error) => error.includes("task branch")));
  fs.rmSync(root, { recursive: true, force: true });
});

test("replaceTemplate rejects unknown placeholders", () => {
  assert.equal(replaceTemplate("{{TASK_ID}}", { TASK_ID: "TASK-0001" }), "TASK-0001");
  assert.equal(replaceTemplate("title: {{TITLE_YAML}}", { TITLE_YAML: JSON.stringify('quote " title') }), 'title: "quote \\" title"');
  assert.throws(() => replaceTemplate("{{UNKNOWN}}", {}), /Missing template value/);
});

test("work repository lock rejects active owners and recovers stale owners", () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "work-lock-"));
  git(root, ["init", "-b", "main"]);
  const release = acquireWorkRepoLock(root, { requireClean: false, requireMain: false });
  assert.throws(() => acquireWorkRepoLock(root, { requireClean: false, requireMain: false }), /Another work repository writer/);
  release();
  const lock = workRepoLockDir(root);
  fs.mkdirSync(lock, { recursive: true });
  fs.writeFileSync(path.join(lock, "owner.json"), '{"pid":99999999}\n');
  const releaseRecovered = acquireWorkRepoLock(root, { requireClean: false, requireMain: false });
  releaseRecovered();
  assert.equal(fs.existsSync(lock), false);
  fs.rmSync(root, { recursive: true, force: true });
});

test("dateInTimezone respects the project timezone", () => {
  assert.match(dateInTimezone("Pacific/Guam"), /^\d{4}-\d{2}-\d{2}$/);
});

test("DEV profile evidence rejects unknown, missing, and risky Luna selections", () => {
  assert.throws(() => validateDevSelection({ approved_dev_profile: "other", approved_dev_profile_reason: "x", approved_dev_profile_risk_signals: [] }), /DEV_PROFILE_UNKNOWN/);
  assert.throws(() => validateDevSelection({ approved_dev_profile: "sol-high", approved_dev_profile_risk_signals: ["security"] }), /REASON_MISSING/);
  assert.throws(() => validateDevSelection({ approved_dev_profile: "luna-xhigh", approved_dev_profile_reason: "x", approved_dev_profile_risk_signals: ["migration"] }), /LUNA_HAS_RISK/);
});

test("launchers close child stdin and reserve commits for the lock-owning parent", () => {
  const workLauncher = fs.readFileSync(path.resolve(import.meta.dirname, "run-work-agent.mjs"), "utf8");
  const wikiLauncher = fs.readFileSync(path.resolve(import.meta.dirname, "run-wiki-agent.mjs"), "utf8");
  const explorerLauncher = fs.readFileSync(path.resolve(import.meta.dirname, "run-explorer-agent.mjs"), "utf8");
  const configSyncLauncher = fs.readFileSync(path.resolve(import.meta.dirname, "run-work-config-sync.mjs"), "utf8");
  const hook = fs.readFileSync(path.resolve(import.meta.dirname, "work-pre-commit.mjs"), "utf8");
  for (const launcher of [workLauncher, wikiLauncher]) {
    assert.match(launcher, /stdio:\s*\["ignore",\s*"pipe",\s*"pipe"\]/);
    assert.match(launcher, /WORK_PARENT_COMMIT:\s*"1"/);
    assert.match(launcher, /WORK_CHILD_(?:COMMIT_FORBIDDEN|STAGE_FORBIDDEN)|validateChildOutcome/);
    assert.match(launcher, /rollbackWorkRepository\(root, beforeHead\)/);
    assert.match(launcher, /commit:\s*null/);
  }
  assert.match(explorerLauncher, /spawn\("codex", invocation\.command/);
  assert.match(explorerLauncher, /stdio:\s*\["ignore",\s*"pipe",\s*"pipe"\]/);
  assert.match(workLauncher, /run-explorer-agent\.mjs/);
  assert.match(workLauncher, /Do not use natural-language or custom-agent delegation for Explorer/);
  assert.match(configSyncLauncher, /acquireWorkRepoLock\(root\)/);
  assert.match(configSyncLauncher, /WORK_PARENT_COMMIT:\s*"1"/);
  assert.match(configSyncLauncher, /WORK_ACTION:\s*"work-config-sync"/);
  assert.match(configSyncLauncher, /syncWorkAdapter\(\{ productRoot, adapterRoot: root, check: true \}\)/);
  assert.match(configSyncLauncher, /rollbackWorkRepository\(root, beforeHead\)/);
  assert.doesNotMatch(configSyncLauncher, /--no-verify|spawnSync\("codex"/);
  assert.match(hook, /WORK_PARENT_COMMIT/);
  assert.match(hook, /lock-owning launcher parent/);
});

function createConfigSyncFixture({ hookExit = 0, committedDrift = false } = {}) {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "work-config-sync-"));
  git(root, ["init", "-b", "main"]);
  git(root, ["config", "user.name", "fixture"]);
  git(root, ["config", "user.email", "fixture@example.invalid"]);
  fs.mkdirSync(path.join(root, ".githooks"));
  fs.writeFileSync(path.join(root, ".gitignore"), ".locks/\nhook.log\n");
  fs.writeFileSync(path.join(root, "baseline.txt"), "baseline\n");
  fs.writeFileSync(path.join(root, ".githooks", "pre-commit"), `#!/bin/sh
set -eu
test "\${WORK_REPO_LOCK_HELD:-}" = "1"
test "\${WORK_PARENT_COMMIT:-}" = "1"
test "\${WORK_ACTION:-}" = "work-config-sync"
test "\${WORK_ALLOWED_PATHS:-}" = '[".codex/config.toml"]'
test -d "$(git rev-parse --git-common-dir)/agent-harness-locks/work-repository.lock"
test "$(git diff --cached --name-only)" = ".codex/config.toml"
printf 'invoked\\n' >> hook.log
exit ${hookExit}
`, { mode: 0o755 });
  const baselineFiles = [".gitignore", ".githooks/pre-commit", "baseline.txt"];
  if (committedDrift) {
    fs.mkdirSync(path.join(root, ".codex"));
    fs.writeFileSync(path.join(root, ".codex", "config.toml"), "# committed drift\n");
    baselineFiles.push(".codex/config.toml");
  }
  git(root, ["add", ...baselineFiles]);
  git(root, ["commit", "-m", "baseline"]);
  git(root, ["config", "core.hooksPath", ".githooks"]);
  return root;
}

test("work config sync owns lock, hook, commit, post-check, and concise evidence", () => {
  const root = createConfigSyncFixture();
  const beforeHead = git(root, ["rev-parse", "HEAD"]);
  const evidence = [];
  let validations = 0;
  try {
    const result = runWorkConfigSync({
      adapterRoot: root,
      validateWork() {
        validations += 1;
        assert.equal(fs.existsSync(workRepoLockDir(root)), true);
      },
      emit: (entry) => evidence.push(entry),
    });
    assert.equal(result.changed, true);
    assert.equal(result.owner, "lock-owning-parent");
    assert.match(result.digest, /^[a-f0-9]{64}$/);
    assert.match(result.commit, /^[a-f0-9]{40}$/);
    assert.notEqual(result.commit, beforeHead);
    assert.equal(validations, 2);
    assert.equal(git(root, ["show", "-s", "--format=%s", "HEAD"]), "governance: sync work adapter");
    assert.equal(git(root, ["show", "--format=", "--name-only", "HEAD"]), ".codex/config.toml");
    assert.equal(fs.readFileSync(path.join(root, "hook.log"), "utf8"), "invoked\n");
    assert.equal(git(root, ["status", "--porcelain"]), "");
    assert.equal(fs.existsSync(workRepoLockDir(root)), false);
    assert.equal(evidence.length, 1);
    assert.ok(JSON.stringify(evidence[0]).length < 1000);

    const committedHead = git(root, ["rev-parse", "HEAD"]);
    const noChange = runWorkConfigSync({ adapterRoot: root, validateWork() {}, emit: (entry) => evidence.push(entry) });
    assert.equal(noChange.changed, false);
    assert.equal(noChange.commit, null);
    assert.equal(git(root, ["rev-parse", "HEAD"]), committedHead);
    const checked = runWorkConfigSync({ adapterRoot: root, mode: "check", validateWork() {}, emit: (entry) => evidence.push(entry) });
    assert.equal(checked.mode, "check");
    assert.equal(checked.changed, false);
    assert.equal(checked.commit, null);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test("work config sync rolls back hook and post-check failures", async (t) => {
  await t.test("hook failure", () => {
    const root = createConfigSyncFixture({ hookExit: 7 });
    const beforeHead = git(root, ["rev-parse", "HEAD"]);
    const evidence = [];
    try {
      assert.throws(() => runWorkConfigSync({ adapterRoot: root, validateWork() {}, emit: (entry) => evidence.push(entry) }));
      assert.equal(git(root, ["rev-parse", "HEAD"]), beforeHead);
      assert.equal(git(root, ["status", "--porcelain"]), "");
      assert.equal(fs.existsSync(path.join(root, ".codex", "config.toml")), false);
      assert.equal(fs.existsSync(workRepoLockDir(root)), false);
      assert.equal(evidence.length, 1);
      assert.equal(evidence[0].commit, null);
      assert.notEqual(evidence[0].error, null);
    } finally {
      fs.rmSync(root, { recursive: true, force: true });
    }
  });

  await t.test("post-check failure after commit", () => {
    const root = createConfigSyncFixture();
    const beforeHead = git(root, ["rev-parse", "HEAD"]);
    const evidence = [];
    let validations = 0;
    try {
      assert.throws(() => runWorkConfigSync({
        adapterRoot: root,
        validateWork() {
          validations += 1;
          assert.equal(fs.existsSync(workRepoLockDir(root)), true);
          if (validations === 2) throw new Error("POST_CHECK_FAILED");
        },
        emit: (entry) => evidence.push(entry),
      }), /POST_CHECK_FAILED/);
      assert.equal(validations, 2);
      assert.equal(git(root, ["rev-parse", "HEAD"]), beforeHead);
      assert.equal(git(root, ["status", "--porcelain"]), "");
      assert.equal(fs.existsSync(path.join(root, ".codex", "config.toml")), false);
      assert.equal(fs.existsSync(workRepoLockDir(root)), false);
      assert.equal(evidence.length, 1);
      assert.equal(evidence[0].commit, null);
      assert.match(evidence[0].error, /POST_CHECK_FAILED/);
    } finally {
      fs.rmSync(root, { recursive: true, force: true });
    }
  });
});

test("work config check detects committed drift while holding the common lock", () => {
  const root = createConfigSyncFixture({ committedDrift: true });
  const beforeHead = git(root, ["rev-parse", "HEAD"]);
  const evidence = [];
  try {
    assert.throws(() => runWorkConfigSync({
      adapterRoot: root,
      mode: "check",
      validateWork() {
        assert.fail("repository validation must not mask adapter drift");
      },
      emit: (entry) => {
        assert.equal(fs.existsSync(workRepoLockDir(root)), true);
        evidence.push(entry);
      },
    }), /ROUTING_ADAPTER_DRIFT/);
    assert.equal(git(root, ["rev-parse", "HEAD"]), beforeHead);
    assert.equal(git(root, ["status", "--porcelain"]), "");
    assert.equal(fs.readFileSync(path.join(root, ".codex", "config.toml"), "utf8"), "# committed drift\n");
    assert.equal(fs.existsSync(workRepoLockDir(root)), false);
    assert.equal(evidence.length, 1);
    assert.equal(evidence[0].mode, "check");
    assert.equal(evidence[0].commit, null);
    assert.match(evidence[0].error, /ROUTING_ADAPTER_DRIFT/);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

function createRollbackFixture() {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "work-rollback-"));
  git(root, ["init", "-b", "main"]);
  git(root, ["config", "user.name", "fixture"]);
  git(root, ["config", "user.email", "fixture@example.invalid"]);
  fs.writeFileSync(path.join(root, ".gitignore"), ".locks/\nignored-cache\n");
  fs.writeFileSync(path.join(root, "ignored-cache"), "preserve\n");
  fs.writeFileSync(path.join(root, "tracked.txt"), "clean\n");
  git(root, ["add", ".gitignore", "tracked.txt"]);
  git(root, ["commit", "-m", "baseline"]);
  return root;
}

test("failure rollback restores HEAD, index, worktree, untracked files, and lock", async (t) => {
  const scenarios = {
    "child nonzero": (root) => {
      const result = spawnSync(process.execPath, ["-e", "require('fs').writeFileSync('tracked.txt','child failure\\n');require('fs').writeFileSync('child.tmp','x');process.exit(7)"], { cwd: root });
      assert.equal(result.status, 7);
    },
    "scope violation": (root) => {
      spawnSync(process.execPath, ["-e", "require('fs').writeFileSync('tracked.txt','scope\\n');require('fs').writeFileSync('forbidden.txt','x')"], { cwd: root });
    },
    "child stage attempt": (root) => {
      fs.writeFileSync(path.join(root, "tracked.txt"), "staged\n");
      git(root, ["add", "tracked.txt"]);
    },
    "child commit attempt": (root) => {
      fs.writeFileSync(path.join(root, "tracked.txt"), "committed\n");
      git(root, ["add", "tracked.txt"]);
      git(root, ["commit", "-m", "forbidden child commit"]);
    },
    "hook failure": (root) => {
      fs.mkdirSync(path.join(root, ".githooks"));
      fs.writeFileSync(path.join(root, ".githooks", "pre-commit"), "#!/bin/sh\nexit 1\n", { mode: 0o755 });
      fs.writeFileSync(path.join(root, "tracked.txt"), "hook failure\n");
      git(root, ["add", "tracked.txt"]);
      const result = spawnSync("git", ["-c", "core.hooksPath=.githooks", "commit", "-m", "must fail"], { cwd: root });
      assert.notEqual(result.status, 0);
    },
    "validation failure": (root) => {
      fs.writeFileSync(path.join(root, "tracked.txt"), "invalid\n");
      fs.writeFileSync(path.join(root, "validation.tmp"), "invalid\n");
      git(root, ["add", "tracked.txt"]);
    },
  };

  for (const [name, mutate] of Object.entries(scenarios)) {
    await t.test(name, () => {
      const root = createRollbackFixture();
      const beforeHead = git(root, ["rev-parse", "HEAD"]);
      const release = acquireWorkRepoLock(root);
      try {
        mutate(root);
        rollbackWorkRepository(root, beforeHead);
        assert.equal(git(root, ["rev-parse", "HEAD"]), beforeHead);
        assert.equal(git(root, ["diff", "--cached", "--name-only"]), "");
        assert.equal(git(root, ["status", "--porcelain"]), "");
        assert.equal(fs.readFileSync(path.join(root, "ignored-cache"), "utf8"), "preserve\n");
      } finally {
        release();
      }
      const releaseAgain = acquireWorkRepoLock(root);
      releaseAgain();
      assert.equal(fs.existsSync(workRepoLockDir(root)), false);
      fs.rmSync(root, { recursive: true, force: true });
    });
  }
});
