import path from "node:path";
import { spawnSync } from "node:child_process";
import {
  REPO_ROOT,
  acquireWorkRepoLock,
  assertTaskId,
  git,
  parseArgs,
  readYaml,
  taskById,
  workRoot,
} from "./lib.mjs";
import { buildLaunchEvidence, codexCommand, resolveFixedRoute, rollbackWorkRepository, validateChildOutcome } from "./agent-routing.mjs";

const args = parseArgs(process.argv.slice(2));
assertTaskId(args.task);
const action = args.action;
const supported = new Set(["task", "plan", "qa-plan", "review", "qa-result", "handover", "main-transition", "governance"]);
if (!supported.has(action)) throw new Error(`--action must be one of ${[...supported].join(", ")}`);
const root = workRoot(args.work_root);
const backlog = readYaml(path.join(root, "backlog.yaml"));
const task = taskById(backlog, args.task);
const inTask = (...files) => files.map((file) => path.posix.join(task.task_dir, file));
const allowedByAction = {
  task: inTask("TASK.md"),
  plan: inTask("PLAN.md"),
  "qa-plan": inTask("QA_PLAN.md"),
  review: inTask("REVIEW_RESULT.md"),
  "qa-result": inTask("QA_RESULT.md"),
  handover: inTask("HANDOVER.md"),
  "main-transition": ["backlog.yaml", ...inTask("TASK.md", "PLAN.md", "QA_PLAN.md", "QA_RESULT.md", "HANDOVER.md")],
  governance: ["schemas/**", "wiki/SCHEMA.md", "wiki/AGENTS.md", ".githooks/pre-commit", ".codex/**", "AGENTS.md", "README.md", "project.yaml"],
};
const allowed = allowedByAction[action];
const route = resolveFixedRoute({
  action,
  planFile: path.join(root, task.task_dir, "PLAN.md"),
  args,
});
const explorerLauncher = path.join(REPO_ROOT, "scripts", "task", "run-explorer-agent.mjs");
const prompt = `Act as the ${route.role} Agent for ${task.id}. Read AGENTS.md, ${task.task_dir}/TASK.md, approved PLAN/QA evidence, and relevant local Wiki. Update only: ${allowed.join(", ")}. Do not stage or commit, invoke another role Agent, change Wiki content or Schema outside the allowlist, or write to .git. The launcher parent owns scope validation and commit. For at most one bounded read-only repository question, invoke node ${JSON.stringify(explorerLauncher)} --root ${JSON.stringify(root)} --question "<question>". Do not use natural-language or custom-agent delegation for Explorer.`;
const command = codexCommand(route, root, prompt);

function changedFiles(repository) {
  const tracked = git(repository, ["diff", "--name-only"]).split("\n").filter(Boolean);
  const staged = git(repository, ["diff", "--cached", "--name-only"]).split("\n").filter(Boolean);
  const untracked = git(repository, ["ls-files", "--others", "--exclude-standard"]).split("\n").filter(Boolean);
  return [...new Set([...tracked, ...staged, ...untracked])];
}

function validateWork() {
  const result = spawnSync(process.execPath, [path.join(REPO_ROOT, "scripts", "task", "validate-work.mjs"), "--work-root", root], {
    cwd: REPO_ROOT,
    encoding: "utf8",
    stdio: ["ignore", "pipe", "pipe"],
  });
  if (result.status !== 0) throw new Error("WORK_VALIDATION_FAILED");
}

function emitEvidence(fields) {
  process.stdout.write(`${JSON.stringify(buildLaunchEvidence({ route, cwd: root, allowedPaths: allowed, ...fields }))}\n`);
}

if (args.dry_run === "true") {
  emitEvidence({ childResult: null, commit: null });
} else {
  const release = acquireWorkRepoLock(root);
  let childResult = null;
  let commit = null;
  let beforeHead = null;
  try {
    if (git(root, ["config", "--get", "core.hooksPath"]) !== ".githooks") {
      throw new Error("WORK_HOOKS_PATH_INVALID");
    }
    beforeHead = git(root, ["rev-parse", "HEAD"]);
    const result = spawnSync("codex", command, {
      cwd: root,
      encoding: "utf8",
      stdio: ["ignore", "pipe", "pipe"],
      env: { ...process.env, WORK_REPO_LOCK_HELD: "1" },
    });
    if (result.error) throw result.error;
    childResult = { exit_code: result.status ?? 1 };
    if (git(root, ["branch", "--show-current"]) !== "main") throw new Error("WORK_BRANCH_CHANGED");
    const stagedByChild = git(root, ["diff", "--cached", "--name-only"]).split("\n").filter(Boolean);
    const changed = validateChildOutcome({
      childExit: result.status ?? 1,
      beforeHead,
      afterHead: git(root, ["rev-parse", "HEAD"]),
      stagedFiles: stagedByChild,
      changedFiles: changedFiles(root),
      allowedPaths: allowed,
    });
    git(root, ["add", "--", ...changed]);
    validateWork();
    const commitEnv = {
      ...process.env,
      WORK_REPO_LOCK_HELD: "1",
      WORK_PARENT_COMMIT: "1",
      WORK_ACTION: action,
      WORK_ALLOWED_PATHS: JSON.stringify(allowed),
    };
    git(root, ["commit", "-m", `${action}: update ${task.id}`], { env: commitEnv });
    commit = git(root, ["rev-parse", "HEAD"]);
    validateWork();
    if (changedFiles(root).length) throw new Error("WORK_PARENT_LEFT_DIRTY");
    emitEvidence({ childResult, commit });
  } catch (error) {
    let failure = error;
    if (beforeHead) {
      try {
        rollbackWorkRepository(root, beforeHead);
      } catch (rollbackError) {
        failure = new Error(`${error.message};WORK_ROLLBACK_FAILED:${rollbackError.message}`);
      }
    }
    emitEvidence({ childResult, commit: null, error: failure.message });
    throw failure;
  } finally {
    release();
  }
}
