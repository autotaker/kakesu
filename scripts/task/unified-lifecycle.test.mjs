import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { spawnSync } from "node:child_process";
import test from "node:test";
import { fileURLToPath } from "node:url";
import { workRepoLockDir } from "./lib.mjs";
import { assertComposite, createSparseWorktree, evidenceCommit, managedDigest, resolveOperationsValidation, scopeCheck, sparsePatterns, syncMain, taskStart } from "./unified-lifecycle.mjs";

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
  git(source, "config", "core.hooksPath", ".original-hooks");
  command(process.execPath, [path.join(ROOT, "scripts/task/migrate-operations.mjs"), "--mode", "freeze", "--source", source, "--source-ref", ref, "--target", ROOT, "--fixture", "true"], ROOT);
  assert.match(git(source, "config", "--get", "core.hooksPath"), /agent-harness-frozen-hooks$/);
  fs.appendFileSync(path.join(source, "backlog.yaml"), "# frozen\n");
  git(source, "add", "backlog.yaml");
  command("git", ["commit", "-m", "must fail while frozen"], source, 1);
  command(process.execPath, [path.join(ROOT, "scripts/task/migrate-operations.mjs"), "--mode", "unfreeze", "--source", source, "--source-ref", ref, "--target", ROOT, "--fixture", "true"], ROOT);
  assert.equal(git(source, "config", "--get", "core.hooksPath"), ".original-hooks");
  git(source, "commit", "-m", "allowed after unfreeze");
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

test("task-start publishes evidence and creates one sparse branch/worktree", () => {
  const root = initTaskStartRepository();
  const result = taskStart({ id: "TASK-9001", slug: "start", title: "start fixture", epic: "EPIC-001", push: "true" }, root);
  assert.equal(result.branch, "task/TASK-9001-start");
  assert.equal(git(root, "rev-parse", "main"), git(root, "rev-parse", "origin/main"));
  assert.match(git(root, "show-ref", "--verify", "refs/heads/task/TASK-9001-start"), /TASK-9001-start/);
  for (const relative of ["backlog.yaml", "project.yaml", "tasks", "wiki"]) assert.equal(fs.existsSync(path.join(result.worktree, relative)), false);
});

test("task-start allocation failure removes invocation-created Task evidence and assignment", () => {
  const root = initTaskStartRepository();
  assert.throws(() => taskStart({ id: "TASK-9002", slug: "rollback", title: "rollback fixture", epic: "EPIC-001", push: "true" }, root, () => { throw new Error("injected allocation failure"); }), /rolled back/);
  assert.doesNotMatch(fs.readFileSync(path.join(root, "backlog.yaml"), "utf8"), /TASK-9002/);
  assert.equal(fs.existsSync(path.join(root, "tasks/TASK-9002-rollback")), false);
  assert.notEqual(spawnSync("git", ["show-ref", "--verify", "refs/heads/task/TASK-9002-rollback"], { cwd: root }).status, 0);
  assert.equal(git(root, "status", "--porcelain"), "");
});

test("task-start publish failure removes invocation-created state and leaves remote unchanged", () => {
  const root = initTaskStartRepository();
  const remote = git(root, "remote", "get-url", "origin");
  fs.mkdirSync(path.join(remote, "hooks"), { recursive: true });
  fs.writeFileSync(path.join(remote, "hooks/pre-receive"), "#!/bin/sh\nexit 1\n", { mode: 0o755 });
  const remoteBefore = git(root, "rev-parse", "origin/main");
  assert.throws(() => taskStart({ id: "TASK-9003", slug: "publish", title: "publish fixture", epic: "EPIC-001", push: "true" }, root), /corrective publish requires reconciliation/);
  assert.doesNotMatch(fs.readFileSync(path.join(root, "backlog.yaml"), "utf8"), /TASK-9003/);
  assert.equal(fs.existsSync(path.join(root, "tasks/TASK-9003-publish")), false);
  assert.notEqual(spawnSync("git", ["show-ref", "--verify", "refs/heads/task/TASK-9003-publish"], { cwd: root }).status, 0);
  assert.equal(git(root, "status", "--porcelain"), "");
  assert.equal(git(root, "ls-remote", "origin", "refs/heads/main").split("\t")[0], remoteBefore);
});

test("pre-merge evidence uses the bound candidate validator while post-merge uses main", () => {
  const root = initTaskStartRepository();
  const started = taskStart({ id: "TASK-9004", slug: "candidate-schema", title: "candidate schema fixture", epic: "EPIC-001", push: "true" }, root);
  const candidate = started.worktree;

  fs.rmSync(path.join(root, "schemas"), { recursive: true });
  fs.rmSync(path.join(root, "scripts"), { recursive: true });
  git(root, "add", "-A"); git(root, "commit", "-m", "simulate pre-merge main without schemas");
  const candidateCommit = git(candidate, "rev-parse", "HEAD");
  const candidateTree = git(candidate, "rev-parse", "HEAD^{tree}");
  const base = git(root, "merge-base", "main", candidateCommit);
  const digest = managedDigest(root, base, candidateCommit);
  const handover = path.join(root, "tasks/TASK-9004-candidate-schema/HANDOVER.md");
  const bound = fs.readFileSync(handover, "utf8")
    .replace(/^candidate_commit: ""$/m, `candidate_commit: "${candidateCommit}"`)
    .replace(/^candidate_tree: ""$/m, `candidate_tree: "${candidateTree}"`)
    .replace(/^managed_path_digest: ""$/m, `managed_path_digest: "${digest}"`)
    .replace(/^bootstrap_evidence_commit: ""$/m, `bootstrap_evidence_commit: "${"a".repeat(40)}"`)
    .replace(/^bootstrap_evidence_digest: ""$/m, `bootstrap_evidence_digest: "${"b".repeat(64)}"`);
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
  assert.throws(() => assertComposite(root, "TASK-9000"), /review is not bound/);
  const base = git(root, "rev-parse", "HEAD");
  fs.appendFileSync(path.join(root, "README.md"), "candidate\n");
  git(root, "add", "README.md"); git(root, "commit", "-m", "candidate code");
  const productHead = git(root, "rev-parse", "HEAD");
  assert.doesNotThrow(() => scopeCheck({ event: "pr", base, head: productHead }, root));
  fs.appendFileSync(path.join(root, "backlog.yaml"), "# forbidden evidence\n");
  git(root, "add", "backlog.yaml"); git(root, "commit", "-m", "forbidden evidence");
  assert.throws(() => scopeCheck({ event: "pr", base, head: git(root, "rev-parse", "HEAD") }, root), /main-managed paths/);
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

test("workflow responsibilities are disjoint and required check names are stable", () => {
  const main = fs.readFileSync(path.join(ROOT, ".github/workflows/main-evidence.yml"), "utf8");
  const pr = fs.readFileSync(path.join(ROOT, ".github/workflows/pr-ci.yml"), "utf8");
  const post = fs.readFileSync(path.join(ROOT, ".github/workflows/post-merge.yml"), "utf8");
  assert.match(main, /permissions:\n  contents: read/);
  assert.doesNotMatch(main, /git push|evidence-commit/);
  for (const name of ["Full check", "Task check", "Scope check"]) assert.match(pr, new RegExp(`name: ${name}`));
  assert.match(post, /types: \[closed\]/);
  assert.match(post, /merged == true/);
  assert.match(post, /group: post-merge-/);
  assert.doesNotMatch(`${main}\n${pr}\n${post}`, /workflow_run|auth\.json|CODEX_HOME/);
});
