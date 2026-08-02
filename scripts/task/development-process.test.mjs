import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import { spawnSync } from "node:child_process";
import { checkTask } from "./check-task.mjs";
import { acquireWorkRepoLock, dateInTimezone, git, replaceTemplate, resolveInside, workRepoLockDir } from "./lib.mjs";
import {
  rollbackWorkRepository,
  validateDevSelection,
} from "./agent-routing.mjs";
import { runWorkConfigSync } from "./run-work-config-sync.mjs";
import { runDocLints } from "../run-doc-lints.mjs";

const REPO_ROOT = path.resolve(path.dirname(new URL(import.meta.url).pathname), "../..");

function fakeDocLintSpawn(outcomes, calls) {
  return (command, args, options) => {
    calls.push({ command, args, options });
    const outcome = outcomes[calls.length - 1];
    if (outcome instanceof Error) throw outcome;
    return outcome ?? { status: 0 };
  };
}

function assertDocLintCalls(calls) {
  assert.deepEqual(calls.map(({ command, args }) => [command, ...args]), [
    ["fake-uv", "run", "--project", "memory", "python", "scripts/validate-terminology.py"],
    ["fake-pnpm", "lint:docs"],
    ["git", "diff", "--check"],
  ]);
  assert.equal(calls.length, 3);
  assert.ok(calls.every(({ options }) => options.shell === false && options.stdio === "inherit"));
  assert.equal(calls[0].options.env.UV_CACHE_DIR, "/tmp/fake-uv-cache");
}

test("doc lint runner aggregates failures without skipping fixed checks", async (t) => {
  await t.test("first failure continues to later checks", () => {
    const calls = [];
    const status = runDocLints({
      uv: "fake-uv",
      pnpm: "fake-pnpm",
      uvCacheDir: "/tmp/fake-uv-cache",
      spawn: fakeDocLintSpawn([{ status: 4 }, { status: 0 }, { status: 0 }], calls),
    });
    assert.equal(status, 1);
    assertDocLintCalls(calls);
  });

  await t.test("multiple failures remain one aggregated failure", () => {
    const calls = [];
    const status = runDocLints({
      uv: "fake-uv",
      pnpm: "fake-pnpm",
      uvCacheDir: "/tmp/fake-uv-cache",
      spawn: fakeDocLintSpawn([{ status: 1 }, { status: 2 }, { status: 0 }], calls),
    });
    assert.equal(status, 1);
    assert.equal(calls.length, 3);
  });

  await t.test("all checks passing returns zero", () => {
    const calls = [];
    const status = runDocLints({
      uv: "fake-uv",
      pnpm: "fake-pnpm",
      uvCacheDir: "/tmp/fake-uv-cache",
      spawn: fakeDocLintSpawn([{ status: 0 }, { status: 0 }, { status: 0 }], calls),
    });
    assert.equal(status, 0);
    assertDocLintCalls(calls);
  });

  await t.test("spawn error continues and returns non-zero", () => {
    for (const failure of [{ error: new Error("missing fake uv") }, new Error("fake spawn throw")]) {
      const calls = [];
      const reports = [];
      const status = runDocLints({
        uv: "fake-uv",
        pnpm: "fake-pnpm",
        uvCacheDir: "/tmp/fake-uv-cache",
        report: () => reports.push("doc lint command failed to start"),
        spawn: fakeDocLintSpawn([failure, { status: 0 }, { status: 0 }], calls),
      });
      assert.equal(status, 1);
      assertDocLintCalls(calls);
      assert.deepEqual(reports, ["doc lint command failed to start"]);
      assert.doesNotMatch(reports[0], /fake-uv|missing fake uv|fake spawn throw/);
    }
  });
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

function createDoneTaskFixture({
  taskId = "TASK-0090",
  changeClass,
  productPath = false,
  changedPaths,
  renameSpoof = false,
  copySpoof = false,
  nonNoFf = false,
  extraParent = false,
  emptyDiff = false,
  mainAdvance = false,
  legacyTask0024 = false,
  legacyProduct = false,
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
  } else if (!emptyDiff) {
    const fixturePaths = changedPaths ?? [productPath ? "scripts/product.mjs" : "docs/development/contract.md"];
    for (const changedPath of fixturePaths) {
      fs.mkdirSync(path.dirname(path.join(repository, changedPath)), { recursive: true });
      fs.writeFileSync(path.join(repository, changedPath), "changed\n");
      git(repository, ["add", changedPath]);
    }
  }
  git(repository, emptyDiff ? ["commit", "--allow-empty", "-m", "candidate"] : ["commit", "-m", "candidate"]);
  const candidateCommit = git(repository, ["rev-parse", "HEAD"]);
  git(repository, ["checkout", "main"]);
  if (mainAdvance) {
    fs.writeFileSync(path.join(repository, "main-evidence.md"), "main evidence\n");
    git(repository, ["add", "main-evidence.md"]);
    git(repository, ["commit", "-m", "main evidence"]);
  }
  if (extraParent) {
    git(repository, ["checkout", "-b", "other"]);
    fs.writeFileSync(path.join(repository, "other.md"), "other\n");
    git(repository, ["add", "other.md"]);
    git(repository, ["commit", "-m", "other"]);
    git(repository, ["checkout", "main"]);
    git(repository, ["merge", "--no-ff", "-m", "merge", "task", "other"]);
  } else {
    git(repository, nonNoFf ? ["merge", "--ff-only", "task"] : ["merge", "--no-ff", "-m", "merge", "task"]);
  }
  const mergedCommit = git(repository, ["rev-parse", "HEAD"]);
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
    decision: "pass",
  };
  if (legacyProduct) {
    reviewMetadata.reviewed_commit = candidateCommit;
    reviewMetadata.make_check = "pass";
    reviewMetadata.reviewed_at = "2026-07-20T00:00:00Z";
  }
  writeTaskEvidence(taskDir, "REVIEW_RESULT.md", reviewMetadata, "Audited candidate diff and DEV make check evidence.\n");
  const qaResultMetadata = {
    task_id: taskId,
    qa_agent: "qa",
    tested_at: "2026-07-20T00:00:00Z",
    decision: "pass",
  };
  if (legacyProduct) qaResultMetadata.tested_commit = mergedCommit;
  writeTaskEvidence(taskDir, "QA_RESULT.md", qaResultMetadata);
  const safetyChecks = Object.fromEntries(SAFETY_CHECK_KEYS.map((key) => [key, "pass"]));
  const handoverMetadata = {
    task_id: taskId,
    status: "complete",
    completed_at: "2026-07-20T00:00:00Z",
    candidate_commit: candidateCommit,
    safety_checks: safetyChecks,
    safety_checked_at: "2026-07-20T00:00:00Z",
  };
  writeTaskEvidence(taskDir, "HANDOVER.md", handoverMetadata);
  fs.mkdirSync(path.join(root, "wiki", "ingestions"), { recursive: true });
  fs.writeFileSync(path.join(root, "wiki", "ingestions", `${taskId}.json`), "{}\n");
  const task = {
    id: taskId,
    status: "done",
    estimate_points: 1,
    task_dir: path.relative(root, taskDir),
    assignees: { main: "main", planner: "planner", dev: "dev-sol-high", reviewer: "reviewer", qa: "qa" },
  };
  if (legacyProduct) task.merged_commit = mergedCommit;
  if (changeClass !== undefined) task.change_class = changeClass;
  if (legacyProduct) {
    fs.writeFileSync(path.join(root, "backlog.yaml"), JSON.stringify({ version: 1, project: "agent-harness", epics: [], tasks: [task] }));
    git(root, ["init", "-b", "main"]);
    git(root, ["config", "user.name", "fixture"]);
    git(root, ["config", "user.email", "fixture@example.invalid"]);
    git(root, ["add", "backlog.yaml", "project.yaml", "tasks"]);
    git(root, ["commit", "-m", "legacy evidence baseline"]);
  }
  return { root, repository, taskDir, backlog: { tasks: [task] }, taskId, candidateCommit, mergedCommit, planMetadata, qaPlanMetadata, reviewMetadata, qaResultMetadata, handoverMetadata };
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
    assert.deepEqual(checkTask(fixture.root, fixture.backlog, fixture.taskId), []);
    fixture.backlog.tasks[0].estimate_points = 13;
    assert.deepEqual(checkTask(fixture.root, fixture.backlog, fixture.taskId), []);
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
    "HANDOVER status": (fixture) => { fixture.handoverMetadata.status = "draft"; },
    "HANDOVER completed_at": (fixture) => { delete fixture.handoverMetadata.completed_at; },
    "candidate missing": (fixture) => { delete fixture.handoverMetadata.candidate_commit; },
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

test("legacy product Done accepts old bindings without the new candidate contract", () => {
  const fixture = createDoneTaskFixture({ legacyProduct: true });
  try {
    assert.match(git(fixture.root, ["show", "HEAD:backlog.yaml"]), /merged_commit/);
    assert.match(git(fixture.root, ["show", "HEAD:tasks/TASK-0090-fixture/REVIEW_RESULT.md"]), /reviewed_commit/);
    delete fixture.handoverMetadata.candidate_commit;
    writeTaskEvidence(fixture.taskDir, "HANDOVER.md", fixture.handoverMetadata);
    writeTaskEvidence(fixture.taskDir, "REVIEW_RESULT.md", fixture.reviewMetadata, "Legacy review evidence.\n");
    assert.deepEqual(checkTask(fixture.root, fixture.backlog, fixture.taskId), []);
    fixture.backlog.tasks[0].merged_commit = "0".repeat(40);
    fs.writeFileSync(path.join(fixture.root, "backlog.yaml"), JSON.stringify({ version: 1, project: "agent-harness", epics: [], tasks: fixture.backlog.tasks }));
    git(fixture.root, ["add", "backlog.yaml"]);
    git(fixture.root, ["commit", "-m", "invalid committed legacy binding"]);
    assert.ok(checkTask(fixture.root, fixture.backlog, fixture.taskId).some((error) => error.includes("legacy done Git evidence is invalid")));
  } finally {
    fs.rmSync(fixture.root, { recursive: true, force: true });
  }
});

test("uncommitted legacy field injection remains on the new contract", () => {
  const fixture = createDoneTaskFixture();
  try {
    fs.writeFileSync(path.join(fixture.root, "backlog.yaml"), JSON.stringify({ version: 1, project: "agent-harness", epics: [], tasks: fixture.backlog.tasks }));
    git(fixture.root, ["init", "-b", "main"]);
    git(fixture.root, ["config", "user.name", "fixture"]);
    git(fixture.root, ["config", "user.email", "fixture@example.invalid"]);
    git(fixture.root, ["add", "backlog.yaml", "project.yaml", "tasks"]);
    git(fixture.root, ["commit", "-m", "new contract baseline"]);
    fixture.backlog.tasks[0].merged_commit = fixture.mergedCommit;
    delete fixture.handoverMetadata.candidate_commit;
    writeTaskEvidence(fixture.taskDir, "HANDOVER.md", fixture.handoverMetadata);
    for (const status of ["plan", "dev"]) {
      fixture.backlog.tasks[0].status = status;
      assert.ok(checkTask(fixture.root, fixture.backlog, fixture.taskId).some((error) => error.includes("legacy fields cannot be introduced")));
    }
    fixture.backlog.tasks[0].status = "done";
    assert.ok(checkTask(fixture.root, fixture.backlog, fixture.taskId).some((error) => error.includes("done requires HANDOVER candidate_commit")));
  } finally {
    fs.rmSync(fixture.root, { recursive: true, force: true });
  }
});

test("new product Done accepts a completion merge in progress and schema leaves binding cross-file", () => {
  const fixture = createDoneTaskFixture();
  try {
    git(fixture.repository, ["reset", "--hard", `${fixture.candidateCommit}^`]);
    git(fixture.repository, ["merge", "--no-ff", "--no-commit", fixture.candidateCommit]);
    assert.deepEqual(checkTask(fixture.root, fixture.backlog, fixture.taskId), []);
    const schema = JSON.parse(fs.readFileSync(path.join(REPO_ROOT, "schemas/operations/backlog.schema.json"), "utf8"));
    assert.equal(schema.$defs.task.allOf.some((rule) => rule.then?.required?.includes("merged_commit")), false);
  } finally {
    fs.rmSync(fixture.root, { recursive: true, force: true });
  }
});

test("safety_contract Done accepts the HANDOVER candidate while its no-ff merge is in progress", () => {
  const fixture = createDoneTaskFixture({ changeClass: "safety_contract", mainAdvance: true });
  try {
    git(fixture.repository, ["reset", "--hard", `${fixture.mergedCommit}^1`]);
    git(fixture.repository, ["merge", "--no-ff", "--no-commit", fixture.candidateCommit]);
    assert.deepEqual(checkTask(fixture.root, fixture.backlog, fixture.taskId), []);
  } finally {
    fs.rmSync(fixture.root, { recursive: true, force: true });
  }
});

test("legacy safety_contract planning review fields remain compatible without v2 opt-in", () => {
  const fixture = createDoneTaskFixture({ taskId: "TASK-0024", changeClass: "safety_contract", legacyTask0024: true });
  try {
    fs.rmSync(path.join(fixture.root, "wiki"), { recursive: true, force: true });
    writeTaskEvidence(fixture.taskDir, "REVIEW_RESULT.md", { task_id: fixture.taskId, decision: "pending" });
    writeTaskEvidence(fixture.taskDir, "QA_RESULT.md", { task_id: fixture.taskId, decision: "pending" });
    fixture.handoverMetadata.status = "safety_contract_complete";
    delete fixture.handoverMetadata.candidate_commit;
    fixture.backlog.tasks[0].merged_commit = fixture.mergedCommit;
    writeTaskEvidence(fixture.taskDir, "HANDOVER.md", fixture.handoverMetadata);
    assert.deepEqual(checkTask(fixture.root, fixture.backlog, fixture.taskId), []);
    fixture.planMetadata.planning_reviewed_by = "other";
    fixture.planMetadata.planning_review_decision = "pending";
    fixture.planMetadata.planning_reviewed_at = "not-a-timestamp";
    writeTaskEvidence(fixture.taskDir, "PLAN.md", fixture.planMetadata);
    assert.deepEqual(checkTask(fixture.root, fixture.backlog, fixture.taskId), []);
  } finally {
    fs.rmSync(fixture.root, { recursive: true, force: true });
  }
});

test("published legacy safety_contract v2 remains compatible without candidate_commit", () => {
  const fixture = createDoneTaskFixture({ taskId: "TASK-0030", changeClass: "safety_contract", safetyContractV2: true });
  try {
    fixture.handoverMetadata.status = "safety_contract_complete";
    delete fixture.handoverMetadata.candidate_commit;
    fixture.backlog.tasks[0].merged_commit = fixture.mergedCommit;
    Object.assign(fixture.handoverMetadata, {
      safety_check_digest: "0".repeat(64),
      safety_candidate_tree: "0".repeat(40),
      safety_merge_tree: "0".repeat(40),
    });
    writeTaskEvidence(fixture.taskDir, "HANDOVER.md", fixture.handoverMetadata);
    assert.deepEqual(checkTask(fixture.root, fixture.backlog, fixture.taskId), []);
  } finally {
    fs.rmSync(fixture.root, { recursive: true, force: true });
  }
});

test("new safety_contract cannot spoof legacy status to omit candidate_commit", () => {
  const fixture = createDoneTaskFixture({ changeClass: "safety_contract", safetyContractV2: true });
  try {
    fixture.handoverMetadata.status = "safety_contract_complete";
    delete fixture.handoverMetadata.candidate_commit;
    writeTaskEvidence(fixture.taskDir, "HANDOVER.md", fixture.handoverMetadata);
    assert.ok(checkTask(fixture.root, fixture.backlog, fixture.taskId)
      .some((error) => error.includes("requires HANDOVER candidate_commit")));
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
      writeTaskEvidence(fixture.taskDir, "REVIEW_RESULT.md", { task_id: fixture.taskId, decision: "pending" });
      writeTaskEvidence(fixture.taskDir, "QA_RESULT.md", { task_id: fixture.taskId, decision: "pending" });
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

test("safety_contract rejects main-managed paths and a spoofed HANDOVER candidate", async (t) => {
  await t.test("main-managed path", () => {
    const fixture = createDoneTaskFixture({ changeClass: "safety_contract", changedPaths: ["tasks/TASK-0090-fixture/TASK.md"] });
    try {
      assert.ok(checkTask(fixture.root, fixture.backlog, fixture.taskId)
        .some((error) => error.includes("product or unapproved path")));
    } finally {
      fs.rmSync(fixture.root, { recursive: true, force: true });
    }
  });
  await t.test("candidate mismatch", () => {
    const fixture = createDoneTaskFixture({ changeClass: "safety_contract" });
    try {
      fixture.handoverMetadata.candidate_commit = git(fixture.repository, ["rev-parse", `${fixture.candidateCommit}^`]);
      writeTaskEvidence(fixture.taskDir, "HANDOVER.md", fixture.handoverMetadata);
      assert.ok(checkTask(fixture.root, fixture.backlog, fixture.taskId)
        .some((error) => error.includes("no unique main exact two-parent no-ff merge")));
    } finally {
      fs.rmSync(fixture.root, { recursive: true, force: true });
    }
  });
});

test("safety_contract rejects rename, copy, and non-no-ff spoofing", async (t) => {
  for (const [name, options, expected] of [
    ["rename", { renameSpoof: true }, "product or unapproved path"],
    ["copy", { copySpoof: true }, "product or unapproved path"],
    ["empty candidate diff", { emptyDiff: true }, "product or unapproved path"],
    ["non-no-ff", { nonNoFf: true }, "two-parent no-ff"],
    ["extra merge parent", { extraParent: true }, "two-parent no-ff"],
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

test("safety_contract rejects incomplete checks and invalid check time", async (t) => {
  const mutations = {
    "missing exact check": (fixture) => { delete fixture.handoverMetadata.safety_checks.docs_lint; },
    "unexpected check": (fixture) => { fixture.handoverMetadata.safety_checks.extra = "pass"; },
    "failed check": (fixture) => { fixture.handoverMetadata.safety_checks.make_check = "pending"; },
    "invalid check time": (fixture) => { fixture.handoverMetadata.safety_checked_at = "not-a-timestamp"; },
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

test("safety_contract tolerates legacy merge, tree, and digest fields as unused input", () => {
  const fixture = createDoneTaskFixture({ changeClass: "safety_contract" });
  try {
    fixture.backlog.tasks[0].merged_commit = "0".repeat(40);
    Object.assign(fixture.handoverMetadata, {
      safety_check_digest: "0".repeat(64),
      safety_candidate_tree: "0".repeat(40),
      safety_merge_tree: "0".repeat(40),
    });
    writeTaskEvidence(fixture.taskDir, "HANDOVER.md", fixture.handoverMetadata);
    assert.deepEqual(checkTask(fixture.root, fixture.backlog, fixture.taskId), []);
  } finally {
    fs.rmSync(fixture.root, { recursive: true, force: true });
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

test("PLAN template uses Main approval without an independent planning review", () => {
  const template = fs.readFileSync(path.join(REPO_ROOT, "templates/task/PLAN.md"), "utf8");
  assert.doesNotMatch(template, /planning_reviewed_(?:by|at)|planning_review_decision/);
  assert.match(template, /MainはPLAN\/QA_PLANの意図・スコープ・受け入れ経路を確認し/);
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
  const explorerLauncher = fs.readFileSync(path.resolve(import.meta.dirname, "run-explorer-agent.mjs"), "utf8");
  const configSyncLauncher = fs.readFileSync(path.resolve(import.meta.dirname, "run-work-config-sync.mjs"), "utf8");
  const hook = fs.readFileSync(path.resolve(import.meta.dirname, "work-pre-commit.mjs"), "utf8");
  assert.match(workLauncher, /stdio:\s*\["ignore",\s*"pipe",\s*"pipe"\]/);
  assert.match(workLauncher, /WORK_PARENT_COMMIT:\s*"1"/);
  assert.match(workLauncher, /WORK_CHILD_(?:COMMIT_FORBIDDEN|STAGE_FORBIDDEN)|validateChildOutcome/);
  assert.match(workLauncher, /rollbackWorkRepository\(root, beforeHead\)/);
  assert.match(workLauncher, /commit:\s*null/);
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

test("Wiki uses the standard edit-only role without a legacy launcher", () => {
  const root = path.resolve(REPO_ROOT);
  const makefile = fs.readFileSync(path.join(root, "Makefile"), "utf8");
  const config = fs.readFileSync(path.join(root, ".codex/config.toml"), "utf8");
  const role = fs.readFileSync(path.join(root, ".codex/agents/wiki.toml"), "utf8");
  const agents = fs.readFileSync(path.join(root, "AGENTS.md"), "utf8");
  const roleGuide = fs.readFileSync(path.join(root, "docs/development/agent-roles.md"), "utf8");
  assert.equal(fs.existsSync(path.join(root, "scripts/task/run-wiki-agent.mjs")), false);
  for (const removed of ["wiki-context", "wiki-ingest", "WIKI_CONTEXT_TARGET", "WIKI_PROFILE", "WIKI_MODEL", "WIKI_EFFORT", "legacy-wiki"]) {
    assert.doesNotMatch(makefile, new RegExp(removed));
  }
  assert.match(config, /\[agents\.wiki\][\s\S]*config_file = "agents\/wiki\.toml"/);
  assert.match(role, /^name = "wiki"$/m);
  assert.match(role, /^model = "gpt-5\.6-terra"$/m);
  assert.match(role, /^model_reasoning_effort = "medium"$/m);
  assert.match(role, /^sandbox_mode = "workspace-write"$/m);
  assert.match(role, /^max_threads = 1$/m);
  assert.match(role, /^max_depth = 0$/m);
  assert.match(role, /Do not spawn another Agent/);
  assert.match(role, /stage, commit, merge/);
  assert.match(role, /\.git/);
  for (const contract of [agents, roleGuide]) {
    assert.match(contract, /make evidence-commit TASK=\.\.\. ACTION=wiki/);
    assert.match(contract, /dirty Wiki差分[\s\S]*索引生成[\s\S]*生成後の最終スコープ検査[\s\S]*work-check[\s\S]*ステージング[\s\S]*単一コミット[\s\S]*push/);
    assert.match(contract, /スタンドアロンの`make wiki-index`は保守用generator/);
    assert.match(contract, /明示的な取り込み時だけ[\s\S]*任意成果物/);
    assert.doesNotMatch(contract, /索引変更時だけ`make wiki-index`/);
  }
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
