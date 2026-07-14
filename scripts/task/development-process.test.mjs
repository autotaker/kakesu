import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import { spawnSync } from "node:child_process";
import { checkTask } from "./check-task.mjs";
import { acquireWorkRepoLock, dateInTimezone, estimatePoints, git, replaceTemplate, resolveInside } from "./lib.mjs";
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
  const release = acquireWorkRepoLock(root, { requireClean: false, requireMain: false });
  assert.throws(() => acquireWorkRepoLock(root, { requireClean: false, requireMain: false }), /Another work repository writer/);
  release();
  const lock = path.join(root, ".locks", "work-repository.lock");
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
test -d .locks/work-repository.lock
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
        assert.equal(fs.existsSync(path.join(root, ".locks", "work-repository.lock")), true);
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
    assert.equal(fs.existsSync(path.join(root, ".locks", "work-repository.lock")), false);
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
      assert.equal(fs.existsSync(path.join(root, ".locks", "work-repository.lock")), false);
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
          assert.equal(fs.existsSync(path.join(root, ".locks", "work-repository.lock")), true);
          if (validations === 2) throw new Error("POST_CHECK_FAILED");
        },
        emit: (entry) => evidence.push(entry),
      }), /POST_CHECK_FAILED/);
      assert.equal(validations, 2);
      assert.equal(git(root, ["rev-parse", "HEAD"]), beforeHead);
      assert.equal(git(root, ["status", "--porcelain"]), "");
      assert.equal(fs.existsSync(path.join(root, ".codex", "config.toml")), false);
      assert.equal(fs.existsSync(path.join(root, ".locks", "work-repository.lock")), false);
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
        assert.equal(fs.existsSync(path.join(root, ".locks", "work-repository.lock")), true);
        evidence.push(entry);
      },
    }), /ROUTING_ADAPTER_DRIFT/);
    assert.equal(git(root, ["rev-parse", "HEAD"]), beforeHead);
    assert.equal(git(root, ["status", "--porcelain"]), "");
    assert.equal(fs.readFileSync(path.join(root, ".codex", "config.toml"), "utf8"), "# committed drift\n");
    assert.equal(fs.existsSync(path.join(root, ".locks", "work-repository.lock")), false);
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
      assert.equal(fs.existsSync(path.join(root, ".locks", "work-repository.lock")), false);
      fs.rmSync(root, { recursive: true, force: true });
    });
  }
});
