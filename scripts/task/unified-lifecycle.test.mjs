import assert from "node:assert/strict";
import crypto from "node:crypto";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { spawnSync } from "node:child_process";
import test from "node:test";
import { fileURLToPath } from "node:url";
import { parse as parseYaml } from "yaml";
import { changedContentDigest, workRepoLockDir } from "./lib.mjs";
import { assertComposite, candidateCommit, completionGate, createSparseWorktree, evidenceCommit, managedDigest, resolveOperationsValidation, scopeCheck, sparsePatterns, syncMain, taskStart } from "./unified-lifecycle.mjs";

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");

function command(program, argv, cwd, expected = 0) {
  const result = spawnSync(program, argv, { cwd, encoding: "utf8" });
  assert.equal(result.status, expected, `${program} ${argv.join(" ")}\n${result.stderr}\n${result.stdout}`);
  return result;
}

function git(root, ...argv) { return command("git", argv, root).stdout.trim(); }

function writeProject(file) {
  fs.writeFileSync(file, "version: 2\nproject_id: agent-harness\nrepository_path: .\nevidence_root: .\ndefault_branch: main\ntimezone: Pacific/Guam\nworktree_root: worktrees\n");
}

function commitBootstrapManifest(root) {
  const body = { version: 1, task_id: "TASK-0033", entries: [] };
  const digest = crypto.createHash("sha256").update(`${JSON.stringify(body)}\n`).digest("hex");
  const file = path.join(root, "tasks/TASK-0033-unify-work-repository/BOOTSTRAP_MANIFEST.json");
  fs.mkdirSync(path.dirname(file), { recursive: true });
  fs.writeFileSync(file, `${JSON.stringify({ ...body, manifest_sha256: digest }, null, 2)}\n`);
  git(root, "add", file); git(root, "commit", "-m", "bootstrap fixture");
  return { commit: git(root, "rev-parse", "HEAD"), digest };
}

function initRepository() {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "unified-lifecycle-"));
  git(root, "init", "-b", "main");
  git(root, "config", "user.name", "Fixture");
  git(root, "config", "user.email", "fixture@example.invalid");
  fs.writeFileSync(path.join(root, ".gitignore"), ".locks/\nworktrees/\nnode_modules/\n");
  fs.writeFileSync(path.join(root, "project.yaml"), "version: 2\nrepository_path: .\nevidence_root: .\ndefault_branch: main\n");
  fs.mkdirSync(path.join(root, "tasks/TASK-9000-fixture"), { recursive: true });
  for (const file of ["TASK.md", "PLAN.md", "QA_PLAN.md", "REVIEW_RESULT.md", "QA_RESULT.md", "HANDOVER.md"]) {
    fs.writeFileSync(path.join(root, "tasks/TASK-9000-fixture", file), `---\ntask_id: TASK-9000\n---\n# ${file}\n`);
  }
  fs.writeFileSync(path.join(root, "backlog.yaml"), [
    "version: 1", "project: agent-harness", "epics: []", "tasks:", "  - id: TASK-9000", "    title: fixture",
    "    type: chore", "    epic: EPIC-900", "    status: dev", "    priority: P2", "    estimate_points: 1",
    "    task_dir: tasks/TASK-9000-fixture", "    depends_on: []", "    branch: task/TASK-9000-fixture", "    worktree: worktrees/TASK-9000-fixture", "",
  ].join("\n"));
  fs.writeFileSync(path.join(root, "README.md"), "fixture\n");
  git(root, "add", "."); git(root, "commit", "-m", "fixture baseline");
  return root;
}

function backlog(tasks) {
  return ["version: 1", "project: agent-harness", "epics:", "  - id: EPIC-001", "    title: fixture", "    target_start: 2026-01-01", "    target_end: 2026-12-31", "tasks:",
    ...tasks.flatMap((task) => [
      `  - id: ${task.id}`, `    title: ${task.id}`, "    type: chore", "    epic: EPIC-001", `    status: ${task.status}`,
      "    priority: P2", "    estimate_points: 1", `    task_dir: ${task.dir}`, "    depends_on: []",
    ]), ""].join("\n");
}

function initMigrationSource() {
  const source = fs.mkdtempSync(path.join(os.tmpdir(), "migration-source-"));
  git(source, "init", "-b", "main"); git(source, "config", "user.name", "Fixture"); git(source, "config", "user.email", "fixture@example.invalid");
  const tasks = [];
  for (let index = 1; index <= 32; index += 1) {
    const id = `TASK-${String(index).padStart(4, "0")}`;
    const dir = `tasks/${id}-fixture`;
    tasks.push({ id, dir, status: "done" });
    fs.mkdirSync(path.join(source, dir), { recursive: true });
    fs.writeFileSync(path.join(source, dir, "TASK.md"), `---\ntask_id: ${id}\n---\n`);
  }
  fs.mkdirSync(path.join(source, "wiki")); fs.writeFileSync(path.join(source, "wiki/index.json"), "{}\n");
  fs.mkdirSync(path.join(source, "lap30")); fs.writeFileSync(path.join(source, "lap30/events.jsonl"), "\n");
  fs.mkdirSync(path.join(source, "viewer")); fs.writeFileSync(path.join(source, "viewer/index.html"), "fixture\n");
  fs.writeFileSync(path.join(source, "backlog.yaml"), backlog(tasks));
  git(source, "add", "."); git(source, "commit", "-m", "REF-2 fixture");
  const ref = git(source, "rev-parse", "HEAD");
  const current = { id: "TASK-0033", dir: "tasks/TASK-0033-unify-work-repository", status: "dev" };
  fs.mkdirSync(path.join(source, current.dir));
  for (const file of ["TASK.md", "PLAN.md", "QA_PLAN.md", "REVIEW_RESULT.md", "QA_RESULT.md", "HANDOVER.md"]) fs.writeFileSync(path.join(source, current.dir, file), `---\ntask_id: TASK-0033\n---\n`);
  fs.writeFileSync(path.join(source, "backlog.yaml"), backlog([...tasks, current]));
  return { source, ref };
}

function initTaskStartRepository() {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "task-start-"));
  git(root, "init", "-b", "main"); git(root, "config", "user.name", "Fixture"); git(root, "config", "user.email", "fixture@example.invalid");
  fs.writeFileSync(path.join(root, ".gitignore"), ".locks/\nworktrees/\nnode_modules/\n");
  writeProject(path.join(root, "project.yaml"));
  fs.cpSync(path.join(ROOT, "schemas/operations"), path.join(root, "schemas/operations"), { recursive: true });
  fs.mkdirSync(path.join(root, "scripts"), { recursive: true });
  fs.cpSync(path.join(ROOT, "scripts/task"), path.join(root, "scripts/task"), { recursive: true });
  fs.symlinkSync(path.join(ROOT, "node_modules"), path.join(root, "node_modules"), "dir");
  fs.writeFileSync(path.join(root, "backlog.yaml"), ["version: 1", "project: agent-harness", "epics:", "  - id: EPIC-001", "    title: fixture", "    target_start: '2026-01-01'", "    target_end: '2026-12-31'", "tasks: []", ""].join("\n"));
  fs.mkdirSync(path.join(root, "wiki"));
  fs.writeFileSync(path.join(root, "wiki/index.json"), `${JSON.stringify({ version: 1, pages: [] }, null, 2)}\n`);
  git(root, "add", "."); git(root, "commit", "-m", "unified baseline");
  const remote = fs.mkdtempSync(path.join(os.tmpdir(), "task-start-remote-"));
  command("git", ["init", "--bare"], remote); git(root, "remote", "add", "origin", remote); git(root, "push", "-u", "origin", "main");
  return root;
}

function setFrontmatter(file, values) {
  let content = fs.readFileSync(file, "utf8");
  for (const [key, value] of Object.entries(values)) {
    const line = `${key}: ${JSON.stringify(value)}`;
    const pattern = new RegExp(`^${key}:.*$`, "m");
    content = pattern.test(content) ? content.replace(pattern, line) : content.replace(/^---\n/, `---\n${line}\n`);
  }
  fs.writeFileSync(file, content);
}

function approvedPlan(task) {
  return { status: "approved", planner_agent: task.assignees.planner, approved_by: task.assignees.main, approved_at: "2026-08-01T00:00:00Z", approved_dev_profile: "luna-xhigh", approved_dev_profile_reason: "fixture", approved_dev_profile_risk_signals: [] };
}

function prepareThreeCommitFixture() {
  const root = initTaskStartRepository();
  const started = taskStart({ id: "TASK-9010", slug: "three", title: "three commit fixture", epic: "EPIC-001", push: "false" }, root);
  const taskDir = path.join(root, "tasks/TASK-9010-three");
  const backlogValue = parseYaml(fs.readFileSync(path.join(root, "backlog.yaml"), "utf8"));
  const task = backlogValue.tasks.find((entry) => entry.id === "TASK-9010");
  setFrontmatter(path.join(taskDir, "PLAN.md"), approvedPlan(task));
  setFrontmatter(path.join(taskDir, "QA_PLAN.md"), { status: "approved", qa_agent: task.assignees.qa, approved_by: task.assignees.main, approved_at: "2026-08-01T00:00:00Z" });
  const planning = evidenceCommit({ root, action: "planning-gate", taskId: "TASK-9010", message: "planning TASK-9010", push: false, validate: false });
  fs.writeFileSync(path.join(started.worktree, "Makefile"), "check:\n\t@true\n");
  fs.mkdirSync(path.join(started.worktree, "src"), { recursive: true });
  fs.writeFileSync(path.join(started.worktree, "src/product.txt"), "candidate\n");
  fs.rmSync(path.join(started.worktree, "scripts/task/run-explorer-agent.mjs"));
  const candidate = candidateCommit({ root, taskId: "TASK-9010", candidateRoot: started.worktree });
  setFrontmatter(path.join(taskDir, "HANDOVER.md"), { candidate_commit: candidate.candidate_commit });
  setFrontmatter(path.join(taskDir, "REVIEW_RESULT.md"), { reviewer_agent: task.assignees.reviewer, decision: "pass", reviewed_at: "2026-08-01T00:00:00Z" });
  setFrontmatter(path.join(taskDir, "QA_RESULT.md"), { qa_agent: task.assignees.qa, decision: "pass", tested_at: "2026-08-01T00:00:00Z" });
  return { root, taskDir, task, started, planning, candidate };
}

function prepareLegacyQaCleanupFixture() {
  const fixture = prepareThreeCommitFixture();
  const merge = completionGate({ root: fixture.root, taskId: "TASK-9010", validate: false, push: false });
  const backlogFile = path.join(fixture.root, "backlog.yaml");
  const backlogValue = parseYaml(fs.readFileSync(backlogFile, "utf8"));
  const task = backlogValue.tasks.find((entry) => entry.id === "TASK-9010");
  task.status = "qa";
  task.merged_commit = merge.commit;
  setFrontmatter(path.join(fixture.taskDir, "PLAN.md"), {
    approved_dev_profile: "sol-high",
    approved_dev_profile_reason: "fixture",
    approved_dev_profile_risk_signals: ["fixture"],
  });
  setFrontmatter(path.join(fixture.taskDir, "REVIEW_RESULT.md"), {
    reviewed_commit: fixture.candidate.candidate_commit,
    make_check: "pass",
  });
  setFrontmatter(path.join(fixture.taskDir, "QA_PLAN.md"), {
    implementation_reviewed_at: "2026-08-01T00:00:00Z",
  });
  setFrontmatter(path.join(fixture.taskDir, "QA_RESULT.md"), {
    tested_commit: merge.commit,
  });
  setFrontmatter(path.join(fixture.taskDir, "HANDOVER.md"), {
    status: "complete",
    completed_at: "2026-08-01T00:00:00Z",
  });
  fs.writeFileSync(backlogFile, `${JSON.stringify(backlogValue, null, 2)}\n`);
  git(fixture.root, "add", "backlog.yaml", "tasks/TASK-9010-three");
  git(fixture.root, "commit", "-m", "fixture legacy qa state");
  git(fixture.root, "push", "origin", "main");

  // The standard Wiki role has no launcher; syncMain still completes legacy
  // cleanup because Wiki receipts are optional.
  return fixture;
}

test("migration binds REF-2, 32 historical tasks, TASK-0033 overlay, and target digests", () => {
  const { source, ref } = initMigrationSource();
  const target = fs.mkdtempSync(path.join(os.tmpdir(), "bootstrap-target-"));
  writeProject(path.join(target, "project.yaml"));
  const apply = command(process.execPath, [path.join(ROOT, "scripts/task/migrate-operations.mjs"), "--mode", "apply", "--source", source, "--source-ref", ref, "--target", target, "--fixture", "true"], ROOT);
  const manifest = JSON.parse(apply.stdout);
  assert.equal(manifest.category_counts.historical_tasks, 32);
  assert.equal(manifest.category_counts.current_tasks, 1);
  assert.ok(manifest.entries.some((entry) => entry.file === "viewer/index.html"));
  command(process.execPath, [path.join(ROOT, "scripts/task/migrate-operations.mjs"), "--mode", "verify", "--target", target], ROOT);
  fs.appendFileSync(path.join(target, "lap30/events.jsonl"), "tamper\n");
  command(process.execPath, [path.join(ROOT, "scripts/task/migrate-operations.mjs"), "--mode", "verify", "--target", target], ROOT, 1);
});

test("migration rejects a source revision other than the fixed full commit", () => {
  const { source } = initMigrationSource();
  const target = fs.mkdtempSync(path.join(os.tmpdir(), "bootstrap-ref-"));
  writeProject(path.join(target, "project.yaml"));
  command(process.execPath, [path.join(ROOT, "scripts/task/migrate-operations.mjs"), "--mode", "plan", "--source", source, "--source-ref", "HEAD", "--target", target], ROOT, 1);
});

test("migration freeze blocks every source commit and unfreeze restores hooks", () => {
  const { source, ref } = initMigrationSource();
  git(source, "add", "."); git(source, "commit", "-m", "current overlay");
  const expectedHead = git(source, "rev-parse", "HEAD");
  const authority = fs.mkdtempSync(path.join(os.tmpdir(), "freeze-authority-"));
  git(authority, "init", "-b", "main");
  git(source, "config", "core.hooksPath", ".original-hooks");
  const common = git(source, "rev-parse", "--git-common-dir");
  const frozenHooks = path.resolve(source, common, "agent-harness-frozen-hooks");
  fs.mkdirSync(frozenHooks); fs.writeFileSync(path.join(frozenHooks, "pre-commit"), "#!/bin/sh\nexit 1\n", { mode: 0o755 });
  fs.writeFileSync(path.resolve(source, common, "agent-harness-evidence-frozen"), `${JSON.stringify({ authority, source_ref: ref, prior_hooks_path: ".original-hooks", frozen_hooks_path: frozenHooks })}\n`);
  git(source, "config", "core.hooksPath", frozenHooks);
  command(process.execPath, [path.join(ROOT, "scripts/task/migrate-operations.mjs"), "--mode", "freeze", "--source", source, "--source-ref", ref, "--expected-head", expectedHead, "--target", authority, "--fixture", "true"], ROOT);
  command("git", ["status", "--porcelain"], source, 128);
  command("git", ["commit", "--no-verify", "-m", "must fail"], source, 128);
  command("git", ["commit-tree", `${expectedHead}^{tree}`, "-m", "must fail"], source, 128);
  command(process.execPath, [path.join(ROOT, "scripts/task/migrate-operations.mjs"), "--mode", "unfreeze", "--source", source, "--source-ref", ref, "--expected-head", expectedHead, "--target", authority, "--fixture", "true"], ROOT);
  assert.equal(git(source, "config", "--get", "core.hooksPath"), ".original-hooks");
  fs.appendFileSync(path.join(source, "backlog.yaml"), "# unfrozen\n");
  git(source, "add", "backlog.yaml");
  git(source, "commit", "-m", "allowed after unfreeze");
});

test("bootstrap verify follows immutable HANDOVER binding after current evidence changes", () => {
  const { source, ref } = initMigrationSource();
  const target = fs.mkdtempSync(path.join(os.tmpdir(), "bootstrap-bound-"));
  git(target, "init", "-b", "main"); git(target, "config", "user.name", "Fixture"); git(target, "config", "user.email", "fixture@example.invalid");
  writeProject(path.join(target, "project.yaml"));
  const applied = command(process.execPath, [path.join(ROOT, "scripts/task/migrate-operations.mjs"), "--mode", "apply", "--source", source, "--source-ref", ref, "--target", target, "--fixture", "true"], ROOT);
  const manifest = JSON.parse(applied.stdout);
  git(target, "add", "."); git(target, "commit", "-m", "bootstrap");
  const bootstrapCommit = git(target, "rev-parse", "HEAD");
  const handover = path.join(target, "tasks/TASK-0033-unify-work-repository/HANDOVER.md");
  fs.writeFileSync(handover, `---\ntask_id: TASK-0033\nbootstrap_evidence_commit: ${bootstrapCommit}\nbootstrap_evidence_digest: ${manifest.manifest_sha256}\n---\nupdated evidence\n`);
  git(target, "add", handover); git(target, "commit", "-m", "bind bootstrap");
  fs.appendFileSync(path.join(target, "lap30/events.jsonl"), "current evidence changed\n");
  command(process.execPath, [path.join(ROOT, "scripts/task/migrate-operations.mjs"), "--mode", "verify", "--target", target], ROOT);
  fs.writeFileSync(handover, fs.readFileSync(handover, "utf8").replace(bootstrapCommit, git(target, "rev-parse", "HEAD")));
  command(process.execPath, [path.join(ROOT, "scripts/task/migrate-operations.mjs"), "--mode", "verify", "--target", target], ROOT, 1);
});

test("evidence transaction commits only the action allowlist and fails closed on lock/scope", () => {
  const root = initRepository();
  const handover = path.join(root, "tasks/TASK-9000-fixture/HANDOVER.md");
  fs.appendFileSync(handover, "allowed\n");
  const committed = evidenceCommit({ root, action: "handover", taskId: "TASK-9000", message: "fixture evidence", push: false, validate: false });
  assert.match(committed.commit, /^[0-9a-f]{40}$/);
  fs.appendFileSync(handover, "second\n");
  fs.appendFileSync(path.join(root, "README.md"), "forbidden\n");
  assert.throws(() => evidenceCommit({ root, action: "handover", taskId: "TASK-9000", message: "scope", push: false, validate: false }), /scope violation/);
  command("git", ["restore", "."], root);
  fs.appendFileSync(handover, "locked\n");
  const lock = workRepoLockDir(root);
  fs.mkdirSync(lock, { recursive: true });
  fs.writeFileSync(path.join(lock, "owner.json"), `${JSON.stringify({ pid: process.pid })}\n`);
  assert.throws(() => evidenceCommit({ root, action: "handover", taskId: "TASK-9000", message: "lock", push: false, validate: false }), /holds/);
});

test("bootstrap lock stays outside an old main working tree without .locks ignore", () => {
  const root = initRepository();
  fs.writeFileSync(path.join(root, ".gitignore"), "worktrees/\n");
  git(root, "add", ".gitignore"); git(root, "commit", "-m", "simulate old main ignore rules");
  const manifest = path.join(root, "tasks/TASK-0033-unify-work-repository/BOOTSTRAP_MANIFEST.json");
  fs.mkdirSync(path.dirname(manifest), { recursive: true });
  fs.writeFileSync(manifest, "{}\n");
  const result = evidenceCommit({ root, action: "bootstrap", taskId: "TASK-0033", message: "bootstrap fixture", push: false, validate: false });
  assert.deepEqual(result.changed, ["tasks/TASK-0033-unify-work-repository/BOOTSTRAP_MANIFEST.json"]);
  assert.equal(git(root, "status", "--porcelain"), "");
  assert.equal(fs.existsSync(path.join(root, ".locks")), false);
  assert.equal(fs.existsSync(workRepoLockDir(root)), false);
});

test("task-start allocates evidence without a main commit", () => {
  const root = initTaskStartRepository();
  const before = git(root, "rev-parse", "main");
  const result = taskStart({ id: "TASK-9001", slug: "start", title: "start fixture", epic: "EPIC-001", push: "true" }, root);
  assert.equal(result.commit, null);
  assert.equal(git(root, "rev-parse", "main"), before);
  assert.equal(result.branch, "task/TASK-9001-start");
  assert.equal(git(root, "rev-parse", "main"), git(root, "rev-parse", "origin/main"));
  assert.match(git(root, "show-ref", "--verify", "refs/heads/task/TASK-9001-start"), /TASK-9001-start/);
  assert.equal(git(result.worktree, "rev-parse", "HEAD"), git(root, "rev-parse", "main"));
  for (const relative of ["backlog.yaml", "project.yaml", "tasks", "wiki"]) assert.equal(fs.existsSync(path.join(result.worktree, relative)), false);
});

test("task-start allocation failure removes invocation-created Task evidence and assignment", () => {
  const root = initTaskStartRepository();
  assert.throws(() => taskStart({ id: "TASK-9002", slug: "rollback", title: "rollback fixture", epic: "EPIC-001", push: "true" }, root, () => { throw new Error("injected allocation failure"); }), /resources were removed/);
  assert.doesNotMatch(fs.readFileSync(path.join(root, "backlog.yaml"), "utf8"), /TASK-9002/);
  assert.equal(fs.existsSync(path.join(root, "tasks/TASK-9002-rollback")), false);
  assert.notEqual(spawnSync("git", ["show-ref", "--verify", "refs/heads/task/TASK-9002-rollback"], { cwd: root }).status, 0);
  assert.equal(git(root, "status", "--porcelain"), "");
});

test("task-start does not invoke a main commit hook", () => {
  const root = initTaskStartRepository();
  const hook = path.join(root, ".git/hooks/pre-commit");
  fs.writeFileSync(hook, "#!/bin/sh\nexit 1\n", { mode: 0o755 });
  const result = taskStart({ id: "TASK-9003", slug: "commit", title: "commit fixture", epic: "EPIC-001", push: "true" }, root);
  assert.equal(result.commit, null);
  assert.match(fs.readFileSync(path.join(root, "backlog.yaml"), "utf8"), /TASK-9003/);
  assert.equal(fs.existsSync(path.join(root, "tasks/TASK-9003-commit")), true);
});

test("task-start does not publish the planning scaffold", () => {
  const root = initTaskStartRepository();
  const remote = git(root, "remote", "get-url", "origin");
  const attempts = path.join(remote, "attempts");
  fs.mkdirSync(path.join(remote, "hooks"), { recursive: true });
  fs.writeFileSync(path.join(remote, "hooks/pre-receive"), `#!/bin/sh\nprintf x >> '${attempts}'\nexit 1\n`, { mode: 0o755 });
  const remoteBefore = git(root, "rev-parse", "origin/main");
  const result = taskStart({ id: "TASK-9003", slug: "publish", title: "publish fixture", epic: "EPIC-001", push: "true" }, root);
  assert.equal(result.commit, null);
  assert.match(fs.readFileSync(path.join(root, "backlog.yaml"), "utf8"), /TASK-9003/);
  assert.equal(fs.existsSync(path.join(root, "tasks/TASK-9003-publish")), true);
  assert.equal(git(root, "rev-parse", "HEAD"), remoteBefore);
  assert.equal(git(root, "ls-remote", "origin", "refs/heads/main").split("\t")[0], remoteBefore);
  assert.equal(fs.existsSync(attempts) ? fs.readFileSync(attempts, "utf8") : "", "");
});

test("Q-01/Q-08 planning and candidate path is atomic and exactly one commit", () => {
  const fixture = prepareThreeCommitFixture();
  try {
    assert.match(fixture.planning.commit, /^[0-9a-f]{40}$/);
    assert.equal(git(fixture.root, "rev-parse", fixture.task.branch), fixture.candidate.candidate_commit);
    const candidateParents = git(fixture.root, "rev-list", "--parents", "-n", "1", fixture.candidate.candidate_commit).split(" ");
    assert.equal(candidateParents[1], fixture.planning.commit);
    assert.equal(Number(git(fixture.root, "rev-list", "--count", `${fixture.planning.commit}..${fixture.candidate.candidate_commit}`)), 1);
    assert.equal(fixture.candidate.changed.some((file) => file.startsWith("tasks/")), false);
    assert.equal(git(fixture.root, "rev-parse", "HEAD"), fixture.planning.commit);
  } finally {
    fs.rmSync(fixture.root, { recursive: true, force: true });
  }
});

test("candidate make check failure reports stderr and stdout diagnostics", () => {
  const root = initTaskStartRepository();
  try {
    const started = taskStart({ id: "TASK-9014", slug: "check-diagnostics", title: "check diagnostics fixture", epic: "EPIC-001", push: "false" }, root);
    const taskDir = path.join(root, "tasks/TASK-9014-check-diagnostics");
    const task = parseYaml(fs.readFileSync(path.join(root, "backlog.yaml"), "utf8")).tasks.find((entry) => entry.id === "TASK-9014");
    setFrontmatter(path.join(taskDir, "PLAN.md"), approvedPlan(task));
    setFrontmatter(path.join(taskDir, "QA_PLAN.md"), { status: "approved", qa_agent: task.assignees.qa, approved_by: task.assignees.main, approved_at: "2026-08-01T00:00:00Z" });
    evidenceCommit({ root, action: "planning-gate", taskId: "TASK-9014", message: "planning TASK-9014", push: false, validate: false });
    fs.writeFileSync(path.join(started.worktree, "Makefile"), "check:\n\t@printf 'stdout diagnostic\\n'; printf 'stderr diagnostic\\n' >&2; exit 1\n");
    fs.mkdirSync(path.join(started.worktree, "src"), { recursive: true });
    fs.writeFileSync(path.join(started.worktree, "src/product.txt"), "diagnostic candidate\n");
    assert.throws(() => candidateCommit({ root, taskId: "TASK-9014", candidateRoot: started.worktree }), (error) => {
      assert.match(error.message, /stdout diagnostic/);
      assert.match(error.message, /stderr diagnostic/);
      return true;
    });
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test("Q-01 planning preflight failure leaves main and allocation unchanged", () => {
  const root = initTaskStartRepository();
  try {
    const started = taskStart({ id: "TASK-9012", slug: "planning-fail", title: "planning fail", epic: "EPIC-001", push: "false" }, root);
    const before = git(root, "rev-parse", "main");
    const taskDir = path.join(root, "tasks/TASK-9012-planning-fail");
    const taskValue = parseYaml(fs.readFileSync(path.join(root, "backlog.yaml"), "utf8")).tasks[0];
    setFrontmatter(path.join(taskDir, "PLAN.md"), approvedPlan(taskValue));
    setFrontmatter(path.join(taskDir, "QA_PLAN.md"), { status: "approved", qa_agent: taskValue.assignees.qa, approved_by: taskValue.assignees.main, approved_at: "2026-08-01T00:00:00Z" });
    const backlogFile = path.join(root, "backlog.yaml");
    const value = parseYaml(fs.readFileSync(backlogFile, "utf8"));
    value.tasks[0].worktree = "worktrees/missing-planning-worktree";
    fs.writeFileSync(backlogFile, JSON.stringify(value));
    assert.throws(() => evidenceCommit({ root, action: "planning-gate", taskId: "TASK-9012", message: "bad planning", push: false, validate: false }), /requires the recorded Task worktree/);
    assert.equal(git(root, "rev-parse", "main"), before);
    assert.equal(fs.existsSync(started.worktree), true);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test("Q-01 planning rejects a missing DEV profile before any Git mutation", () => {
  const root = initTaskStartRepository();
  try {
    const started = taskStart({ id: "TASK-9015", slug: "profile-fail", title: "profile fail", epic: "EPIC-001", push: "false" }, root);
    const taskDir = path.join(root, "tasks/TASK-9015-profile-fail");
    const task = parseYaml(fs.readFileSync(path.join(root, "backlog.yaml"), "utf8")).tasks[0];
    const planFile = path.join(taskDir, "PLAN.md");
    setFrontmatter(planFile, { status: "approved", planner_agent: task.assignees.planner, approved_by: task.assignees.main, approved_at: "2026-08-01T00:00:00Z" });
    setFrontmatter(path.join(taskDir, "QA_PLAN.md"), { status: "approved", qa_agent: task.assignees.qa, approved_by: task.assignees.main, approved_at: "2026-08-01T00:00:00Z" });
    fs.appendFileSync(planFile, "dirty planning input\n");
    const mainBefore = git(root, "rev-parse", "main");
    const branchBefore = git(root, "rev-parse", task.branch);
    const worktreeBefore = git(started.worktree, "rev-parse", "HEAD");
    const indexBefore = git(root, "diff", "--cached", "--binary");
    const statusBefore = git(root, "status", "--porcelain");
    const planBefore = fs.readFileSync(planFile);
    const remoteBefore = git(root, "rev-parse", "origin/main");
    assert.throws(() => evidenceCommit({ root, action: "planning-gate", taskId: "TASK-9015", message: "bad DEV profile", push: true, validate: false }), /DEV_PROFILE_UNKNOWN/);
    assert.equal(git(root, "rev-parse", "main"), mainBefore);
    assert.equal(git(root, "rev-parse", task.branch), branchBefore);
    assert.equal(git(started.worktree, "rev-parse", "HEAD"), worktreeBefore);
    assert.equal(git(root, "diff", "--cached", "--binary"), indexBefore);
    assert.equal(git(root, "status", "--porcelain"), statusBefore);
    assert.deepEqual(fs.readFileSync(planFile), planBefore);
    assert.equal(git(root, "rev-parse", "origin/main"), remoteBefore);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test("Q-01 planning push failure restores main, Task branch, and worktree", () => {
  const root = initTaskStartRepository();
  try {
    const started = taskStart({ id: "TASK-9013", slug: "planning-push", title: "planning push", epic: "EPIC-001", push: "false" }, root);
    const taskDir = path.join(root, "tasks/TASK-9013-planning-push");
    const value = parseYaml(fs.readFileSync(path.join(root, "backlog.yaml"), "utf8"));
    const task = value.tasks[0];
    setFrontmatter(path.join(taskDir, "PLAN.md"), approvedPlan(task));
    setFrontmatter(path.join(taskDir, "QA_PLAN.md"), { status: "approved", qa_agent: task.assignees.qa, approved_by: task.assignees.main, approved_at: "2026-08-01T00:00:00Z" });
    const mainBefore = git(root, "rev-parse", "main");
    const branchBefore = git(root, "rev-parse", task.branch);
    const remote = git(root, "remote", "get-url", "origin");
    fs.writeFileSync(path.join(remote, "hooks/pre-receive"), "#!/bin/sh\nexit 1\n", { mode: 0o755 });
    assert.throws(() => evidenceCommit({ root, action: "planning-gate", taskId: "TASK-9013", message: "planning push fail", push: true, validate: false }), /retry limit/);
    assert.equal(git(root, "rev-parse", "main"), mainBefore);
    assert.equal(git(root, "rev-parse", task.branch), branchBefore);
    assert.equal(git(started.worktree, "rev-parse", "HEAD"), branchBefore);
    assert.equal(git(root, "rev-parse", "origin/main"), mainBefore);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test("Q-02/Q-04 completion creates one no-ff merge from main-side HANDOVER", () => {
  const fixture = prepareThreeCommitFixture();
  try {
    const result = completionGate({ root: fixture.root, taskId: "TASK-9010", validate: false, push: false });
    const parents = git(fixture.root, "rev-list", "--parents", "-n", "1", result.commit).split(" ");
    assert.equal(parents.length, 3);
    assert.equal(parents[2], fixture.candidate.candidate_commit);
    assert.equal(git(fixture.root, "branch", "--show-current"), "main");
  } finally {
    fs.rmSync(fixture.root, { recursive: true, force: true });
  }
});

test("Q-02 completion abort restores dirty quality evidence after a merge conflict", () => {
  const fixture = prepareThreeCommitFixture();
  try {
    const handover = path.join(fixture.taskDir, "HANDOVER.md");
    const review = path.join(fixture.taskDir, "REVIEW_RESULT.md");
    const before = [fs.readFileSync(handover), fs.readFileSync(review)];
    fs.mkdirSync(path.join(fixture.root, "src"), { recursive: true });
    fs.writeFileSync(path.join(fixture.root, "src/product.txt"), "main conflict\n");
    git(fixture.root, "add", "src/product.txt"); git(fixture.root, "commit", "-m", "main conflict");
    assert.throws(() => completionGate({ root: fixture.root, taskId: "TASK-9010", validate: false, push: false }), /transaction aborted/);
    assert.deepEqual(fs.readFileSync(handover), before[0]);
    assert.deepEqual(fs.readFileSync(review), before[1]);
    assert.match(git(fixture.root, "status", "--porcelain"), /HANDOVER/);
    assert.equal(spawnSync("git", ["rev-parse", "MERGE_HEAD"], { cwd: fixture.root }).status, 128);
  } finally {
    fs.rmSync(fixture.root, { recursive: true, force: true });
  }
});

test("Q-02 completion push failure rolls back the local merge and restores evidence", () => {
  const fixture = prepareThreeCommitFixture();
  try {
    const beforeHead = git(fixture.root, "rev-parse", "HEAD");
    const handover = path.join(fixture.taskDir, "HANDOVER.md");
    const review = path.join(fixture.taskDir, "REVIEW_RESULT.md");
    const qa = path.join(fixture.taskDir, "QA_RESULT.md");
    const beforeEvidence = [fs.readFileSync(handover), fs.readFileSync(review), fs.readFileSync(qa)];
    const remote = git(fixture.root, "remote", "get-url", "origin");
    git(fixture.root, "push", "origin", "HEAD:main");
    fs.writeFileSync(path.join(remote, "hooks/pre-receive"), "#!/bin/sh\nexit 1\n", { mode: 0o755 });
    assert.throws(() => completionGate({ root: fixture.root, taskId: "TASK-9010", validate: false, push: true }), /remote main unchanged.*local merge commit rolled back/);
    assert.equal(git(fixture.root, "rev-parse", "HEAD"), beforeHead);
    assert.deepEqual(fs.readFileSync(handover), beforeEvidence[0]);
    assert.deepEqual(fs.readFileSync(review), beforeEvidence[1]);
    assert.deepEqual(fs.readFileSync(qa), beforeEvidence[2]);
    assert.equal(fs.existsSync(path.join(fixture.root, "src/product.txt")), false);
    assert.equal(spawnSync("git", ["rev-parse", "MERGE_HEAD"], { cwd: fixture.root }).status, 128);
  } finally {
    fs.rmSync(fixture.root, { recursive: true, force: true });
  }
});

test("Q-03 hook rejects tampered trusted bytes and ordinary Task commits", () => {
  const root = initTaskStartRepository();
  try {
    git(root, "checkout", "-b", "task/TASK-9011-hook");
    fs.writeFileSync(path.join(root, "README.md"), "tampered\n");
    git(root, "add", "README.md");
    const hook = path.join(ROOT, "scripts/task/work-pre-commit.mjs");
    const baseEnv = { ...process.env, WORK_REPO_LOCK_HELD: "1", WORK_PARENT_COMMIT: "1", WORK_ACTION: "candidate-commit", WORK_ALLOWED_PATHS: "[\"README.md\"]", WORK_VALIDATED_DIGEST: "0".repeat(64) };
    const tampered = spawnSync(process.execPath, [hook, "--work-root", root], { cwd: root, env: baseEnv, encoding: "utf8" });
    assert.notEqual(tampered.status, 0);
    const ordinary = spawnSync(process.execPath, [hook, "--work-root", root], { cwd: root, env: { ...process.env }, encoding: "utf8" });
    assert.notEqual(ordinary.status, 0);
    assert.match(changedContentDigest(root, { cached: true }), /^[0-9a-f]{64}$/);
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test("pre-merge evidence uses the bound candidate validator while post-merge uses main", () => {
  const root = initTaskStartRepository();
  const started = taskStart({ id: "TASK-9004", slug: "candidate-schema", title: "candidate schema fixture", epic: "EPIC-001", push: "true" }, root);
  const candidate = started.worktree;

  fs.rmSync(path.join(root, "schemas"), { recursive: true });
  fs.rmSync(path.join(root, "scripts"), { recursive: true });
  git(root, "add", "-A"); git(root, "commit", "-m", "simulate pre-merge main without schemas");
  const candidateCommit = git(candidate, "rev-parse", "HEAD");
  const handover = path.join(root, "tasks/TASK-9004-candidate-schema/HANDOVER.md");
  const bound = fs.readFileSync(handover, "utf8")
    .replace(/^candidate_commit: ""$/m, `candidate_commit: "${candidateCommit}"`);
  fs.writeFileSync(handover, bound);
  const selected = resolveOperationsValidation(root, "handover", "TASK-9004", candidate);
  assert.equal(selected.mode, "pre-merge-candidate");
  assert.equal(selected.validatorRoot, candidate);
  const committed = evidenceCommit({ root, action: "handover", taskId: "TASK-9004", message: "candidate-bound evidence", push: false, candidateRoot: candidate });
  assert.deepEqual(committed.changed, ["tasks/TASK-9004-candidate-schema/HANDOVER.md"]);

  fs.cpSync(path.join(candidate, "schemas"), path.join(root, "schemas"), { recursive: true });
  fs.mkdirSync(path.join(root, "scripts"), { recursive: true });
  fs.cpSync(path.join(candidate, "scripts/task"), path.join(root, "scripts/task"), { recursive: true });
  git(root, "add", "schemas", "scripts/task"); git(root, "commit", "-m", "simulate merged validator");
  fs.rmSync(path.join(candidate, "schemas/operations/backlog.schema.json"));
  const mergedSelection = resolveOperationsValidation(root, "handover", "TASK-9004", candidate);
  assert.deepEqual(mergedSelection, { mode: "main", validatorRoot: root, schemaRoot: root });
});

test("evidence push attempts stop after two non-fast-forward retry cycles", () => {
  const root = initRepository();
  const remote = fs.mkdtempSync(path.join(os.tmpdir(), "evidence-remote-"));
  command("git", ["init", "--bare"], remote);
  git(root, "remote", "add", "origin", remote);
  git(root, "push", "origin", "main");
  fs.writeFileSync(path.join(remote, "hooks/pre-receive"), "#!/bin/sh\nexit 1\n", { mode: 0o755 });
  fs.appendFileSync(path.join(root, "tasks/TASK-9000-fixture/HANDOVER.md"), "retry\n");
  assert.throws(() => evidenceCommit({ root, action: "handover", taskId: "TASK-9000", message: "retry", push: true, validate: false }), /retry limit/);
});

test("sparse code worktree excludes every main-managed evidence path", () => {
  const root = initRepository();
  for (const relative of ["wiki/index.json", "lap30/events.jsonl", "viewer/index.html"]) {
    fs.mkdirSync(path.join(root, path.dirname(relative)), { recursive: true });
    fs.writeFileSync(path.join(root, relative), "evidence\n");
  }
  git(root, "add", "."); git(root, "commit", "-m", "evidence paths");
  const worktree = path.join(root, "worktrees/TASK-9001-sparse");
  createSparseWorktree(root, "task/TASK-9001-sparse", worktree);
  assert.deepEqual(sparsePatterns(), ["/*", "!/backlog.yaml", "!/project.yaml", "!/tasks/", "!/wiki/", "!/lap30/", "!/viewer/index.html"]);
  assert.equal(fs.existsSync(path.join(worktree, "README.md")), true);
  for (const relative of ["backlog.yaml", "project.yaml", "tasks", "wiki", "lap30", "viewer/index.html"]) assert.equal(fs.existsSync(path.join(worktree, relative)), false, relative);
});

test("managed digest excludes evidence-only changes", () => {
  const root = initRepository();
  const base = git(root, "rev-parse", "HEAD");
  fs.appendFileSync(path.join(root, "README.md"), "code\n");
  fs.appendFileSync(path.join(root, "backlog.yaml"), "# evidence\n");
  git(root, "add", "."); git(root, "commit", "-m", "mixed");
  const head = git(root, "rev-parse", "HEAD");
  const digest = managedDigest(root, base, head);
  assert.match(digest, /^[0-9a-f]{64}$/);
  assert.notEqual(digest, managedDigest(root, head, head));
});

test("composite binding and PR scope reject stale or main-managed evidence", () => {
  const root = initRepository();
  git(root, "branch", "task/TASK-9000-fixture");
  assert.throws(() => assertComposite(root, "TASK-9000"), /HANDOVER candidate/);
  const base = git(root, "rev-parse", "HEAD");
  fs.appendFileSync(path.join(root, "README.md"), "candidate\n");
  git(root, "add", "README.md"); git(root, "commit", "-m", "candidate code");
  const productHead = git(root, "rev-parse", "HEAD");
  assert.doesNotThrow(() => scopeCheck({ event: "pr", base, head: productHead }, root));
  fs.appendFileSync(path.join(root, "backlog.yaml"), "# forbidden evidence\n");
  git(root, "add", "backlog.yaml"); git(root, "commit", "-m", "forbidden evidence");
  assert.throws(() => scopeCheck({ event: "pr", base, head: git(root, "rev-parse", "HEAD") }, root), /main-managed paths/);
});

test("PR scope ignores diverged main evidence but rejects candidate evidence", () => {
  const root = initRepository();
  git(root, "checkout", "-b", "task/TASK-9001-scope");
  fs.appendFileSync(path.join(root, "README.md"), "candidate\n");
  git(root, "add", "README.md"); git(root, "commit", "-m", "candidate code");
  const productHead = git(root, "rev-parse", "HEAD");

  git(root, "checkout", "main");
  fs.appendFileSync(path.join(root, "tasks/TASK-9000-fixture/HANDOVER.md"), "main evidence\n");
  git(root, "add", "tasks/TASK-9000-fixture/HANDOVER.md"); git(root, "commit", "-m", "advance main evidence");
  const advancedBase = git(root, "rev-parse", "HEAD");
  assert.doesNotThrow(() => scopeCheck({ event: "pr", base: advancedBase, head: productHead }, root));

  git(root, "checkout", "task/TASK-9001-scope");
  fs.appendFileSync(path.join(root, "tasks/TASK-9000-fixture/QA_RESULT.md"), "candidate evidence\n");
  git(root, "add", "tasks/TASK-9000-fixture/QA_RESULT.md"); git(root, "commit", "-m", "candidate evidence");
  assert.throws(() => scopeCheck({ event: "pr", base: advancedBase, head: git(root, "rev-parse", "HEAD") }, root), /main-managed paths/);
});

test("main scope accepts only the HANDOVER-bound candidate merge", () => {
  const root = initRepository();
  git(root, "checkout", "-b", "task/TASK-9000-fixture");
  fs.appendFileSync(path.join(root, "README.md"), "candidate\n");
  git(root, "add", "README.md"); git(root, "commit", "-m", "candidate");
  const candidate = git(root, "rev-parse", "HEAD");
  git(root, "checkout", "main");
  fs.writeFileSync(path.join(root, "tasks/TASK-9000-fixture/HANDOVER.md"), `---\ntask_id: TASK-9000\ncandidate_commit: ${candidate}\n---\n`);
  git(root, "add", "tasks/TASK-9000-fixture/HANDOVER.md"); git(root, "commit", "-m", "bind candidate");
  const firstParent = git(root, "rev-parse", "HEAD");
  git(root, "merge", "--no-ff", "task/TASK-9000-fixture", "-m", "merge candidate");
  const merge = git(root, "rev-parse", "HEAD");
  assert.equal(scopeCheck({ event: "main", base: firstParent, head: merge, allow_merge: "true" }, root).candidate_commit, candidate);
  fs.appendFileSync(path.join(root, "backlog.yaml"), "# injected merge evidence\n");
  git(root, "add", "backlog.yaml"); git(root, "commit", "--amend", "--no-edit");
  assert.doesNotThrow(() => scopeCheck({ event: "main", base: firstParent, head: git(root, "rev-parse", "HEAD"), allow_merge: "true" }, root));

  const arbitrary = initRepository();
  git(arbitrary, "checkout", "-b", "arbitrary");
  fs.appendFileSync(path.join(arbitrary, "README.md"), "unbound\n");
  git(arbitrary, "add", "README.md"); git(arbitrary, "commit", "-m", "unbound product");
  git(arbitrary, "checkout", "main");
  const arbitraryFirst = git(arbitrary, "rev-parse", "HEAD");
  git(arbitrary, "merge", "--no-ff", "arbitrary", "-m", "arbitrary merge");
  assert.throws(() => scopeCheck({ event: "main", base: arbitraryFirst, head: git(arbitrary, "rev-parse", "HEAD"), allow_merge: "true" }, arbitrary), /not bound/);
});

test("sync FAST only updates main and normal empty sync is idempotent", () => {
  const root = initTaskStartRepository();
  const bin = fs.mkdtempSync(path.join(os.tmpdir(), "sync-bin-"));
  fs.writeFileSync(path.join(bin, "gh"), "#!/bin/sh\nprintf 'success\\n'\n", { mode: 0o755 });
  const priorPath = process.env.PATH;
  process.env.PATH = `${bin}:${priorPath}`;
  try {
    assert.deepEqual(syncMain({ fast: "1", repo: "fixture", push: "false" }, root), { fast: true });
    assert.deepEqual(syncMain({ fast: "0", repo: "fixture", push: "false" }, root), { fast: false, no_op: true });
    assert.deepEqual(syncMain({ fast: "0", repo: "fixture", push: "false" }, root), { fast: false, no_op: true });
  } finally {
    process.env.PATH = priorPath;
  }
});

test("sync legacy qa cleanup does not require or create a Wiki receipt", () => {
  const fixture = prepareLegacyQaCleanupFixture();
  const bin = fs.mkdtempSync(path.join(os.tmpdir(), "sync-bin-"));
  fs.writeFileSync(path.join(bin, "gh"), "#!/bin/sh\nprintf 'success\\n'\n", { mode: 0o755 });
  const priorPath = process.env.PATH;
  process.env.PATH = `${bin}:${priorPath}`;
  try {
    const receipt = path.join(fixture.root, "wiki/ingestions/TASK-9010.json");
    assert.equal(fs.existsSync(receipt), false);
    const result = syncMain({ fast: "0", repo: "fixture", push: "false" }, fixture.root);
    assert.match(result.commit, /^[0-9a-f]{40}$/);
    assert.equal(fs.existsSync(receipt), false);
    const synced = parseYaml(fs.readFileSync(path.join(fixture.root, "backlog.yaml"), "utf8"));
    const task = synced.tasks.find((entry) => entry.id === "TASK-9010");
    assert.equal(task.status, "done");
    assert.equal(task.branch, undefined);
    assert.equal(task.worktree, undefined);
  } finally {
    process.env.PATH = priorPath;
    fs.rmSync(fixture.root, { recursive: true, force: true });
  }
});

test("workflow responsibilities are disjoint and required check names are stable", () => {
  const main = fs.readFileSync(path.join(ROOT, ".github/workflows/main-evidence.yml"), "utf8");
  const pr = fs.readFileSync(path.join(ROOT, ".github/workflows/pr-ci.yml"), "utf8");
  const post = fs.readFileSync(path.join(ROOT, ".github/workflows/post-merge.yml"), "utf8");
  const prWorkflow = parseYaml(pr);
  const postWorkflow = parseYaml(post);
  const fullSteps = prWorkflow.jobs.full.steps;
  const scopeSteps = prWorkflow.jobs.scope.steps;
  const postSteps = postWorkflow.jobs.record.steps;
  assert.match(main, /permissions:\n  contents: read/);
  assert.doesNotMatch(main, /git push|evidence-commit/);
  for (const name of ["Full check", "Task check", "Scope check"]) assert.match(pr, new RegExp(`name: ${name}`));
  assert.ok(fullSteps.some((step) => step.uses === "astral-sh/setup-uv@v9.0.0"), "Full check must set up uv with the supported action tag");
  assert.ok(scopeSteps.some((step) => /^pnpm\/action-setup@/.test(step.uses ?? "")), "Scope check must set up pnpm");
  assert.ok(scopeSteps.some((step) => /\bpnpm install --frozen-lockfile\b/.test(step.run ?? "")), "Scope check must install locked Node dependencies");
  assert.match(post, /types: \[closed\]/);
  assert.match(post, /merged == true/);
  assert.match(post, /group: post-merge-/);
  assert.ok(postSteps.some((step) => step.uses === "pnpm/action-setup@v4" && step.with?.version === "9.15.2"), "Post merge evidence must set up the declared pnpm version");
  assert.ok(postSteps.some((step) => /\bpnpm install --frozen-lockfile\b/.test(step.run ?? "")), "Post merge evidence must install locked Node dependencies");
  const authorStep = postSteps.find((step) => step.name === "Configure repository author");
  assert.deepEqual(authorStep?.run.trim().split(/\r?\n/), [
    'git config --local user.name "github-actions[bot]"',
    'git config --local user.email "41898282+github-actions[bot]@users.noreply.github.com"',
  ]);
  assert.doesNotMatch(`${main}\n${pr}\n${post}`, /workflow_run|auth\.json|CODEX_HOME/);
});
