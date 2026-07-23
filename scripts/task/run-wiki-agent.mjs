import fs from "node:fs";
import path from "node:path";
import { spawnSync } from "node:child_process";
import {
  REPO_ROOT,
  acquireWorkRepoLock,
  assertTaskId,
  git,
  parseArgs,
  readYaml,
  resolveInside,
  taskById,
  workRoot,
} from "./lib.mjs";
import { buildLaunchEvidence, rollbackWorkRepository } from "./agent-routing.mjs";

const args = parseArgs(process.argv.slice(2));
const action = args.action;
const taskId = args.task;
assertTaskId(taskId);
if (!new Set(["context-task", "context-plan", "ingest"]).has(action)) throw new Error("--action must be context-task, context-plan, or ingest");
const root = workRoot(args.work_root);
const task = taskById(readYaml(path.join(root, "backlog.yaml")), taskId);
const targetByAction = { "context-task": "TASK.md", "context-plan": "PLAN.md", ingest: "HANDOVER.md" };
const target = path.posix.join(task.task_dir, targetByAction[action]);
resolveInside(root, target, "Wiki Agent target");
const allowed = action === "ingest" ? ["wiki/semantic/**", "wiki/decisions/**", "wiki/ingestions/**", "wiki/index.json"] : [target];
const matchesAllowed = (file) => allowed.some((rule) => rule.endsWith("/**") ? file.startsWith(rule.slice(0, -2)) : file === rule);
const route = {
  role: "wiki",
  profile: args.profile ?? "legacy-wiki",
  model: args.model ?? "gpt-5.6-terra",
  effort: args.effort ?? "medium",
  sandbox: "workspace-write",
};
const promptByAction = {
  "context-task": `Review ${target} and the local Wiki. Update only the Related Context section of TASK.md. Do not stage, commit, invoke another Agent, or write to .git.`,
  "context-plan": `Review ${target}, TASK.md, and the local Wiki. Update only the Related Wiki and Decision section of PLAN.md. Do not stage, commit, invoke another Agent, or write to .git.`,
  ingest: `Ingest ${target} by editing only Wiki paths allowed by wiki/AGENTS.md. Create an idempotent receipt and update the index. Do not stage, commit, modify Task evidence, invoke another Agent, or write to .git.`,
};
const command = [
  "exec", "-C", root, "--sandbox", "workspace-write", "-m", route.model,
  "-c", `model_reasoning_effort=${JSON.stringify(route.effort)}`, promptByAction[action],
];
const changedFiles = () => [...new Set([
  ...git(root, ["diff", "--name-only"]).split("\n").filter(Boolean),
  ...git(root, ["diff", "--cached", "--name-only"]).split("\n").filter(Boolean),
  ...git(root, ["ls-files", "--others", "--exclude-standard"]).split("\n").filter(Boolean),
])];
const validateWork = () => {
  const result = spawnSync(process.execPath, [path.join(REPO_ROOT, "scripts", "task", "validate-work.mjs"), "--work-root", root], {
    cwd: REPO_ROOT, encoding: "utf8", stdio: ["ignore", "pipe", "pipe"],
  });
  if (result.status !== 0) throw new Error("WORK_VALIDATION_FAILED");
};
const emit = (fields) => process.stdout.write(`${JSON.stringify(buildLaunchEvidence({ route, cwd: root, allowedPaths: allowed, legacy: true, ...fields }))}\n`);

if (args.dry_run === "true") {
  emit({ childResult: null, commit: null });
} else {
  const editOnly = args.commit === "false";
  const release = acquireWorkRepoLock(root, { requireClean: !editOnly });
  let childResult = null;
  let beforeHead = null;
  try {
    if (git(root, ["config", "--get", "core.hooksPath"]) !== ".githooks") throw new Error("WORK_HOOKS_PATH_INVALID");
    beforeHead = git(root, ["rev-parse", "HEAD"]);
    const result = spawnSync("codex", command, { cwd: root, encoding: "utf8", stdio: ["ignore", "pipe", "pipe"], env: { ...process.env, WORK_REPO_LOCK_HELD: "1" } });
    if (result.error) throw result.error;
    childResult = { exit_code: result.status ?? 1 };
    if (result.status !== 0) throw new Error("WIKI_CHILD_FAILED");
    if (git(root, ["rev-parse", "HEAD"]) !== beforeHead) throw new Error("WORK_CHILD_COMMIT_FORBIDDEN");
    if (git(root, ["diff", "--cached", "--name-only"])) throw new Error("WORK_CHILD_STAGE_FORBIDDEN");
    const changed = changedFiles();
    if (!changed.length && action !== "ingest") {
      emit({ childResult, commit: null });
    } else {
      if (!changed.length) throw new Error("WIKI_INGEST_NO_CHANGES");
      const forbidden = changed.filter((file) => !matchesAllowed(file));
      if (forbidden.length) throw new Error(`WORK_SCOPE_VIOLATION:${forbidden.join(",")}`);
      validateWork();
      if (action === "ingest" && !fs.existsSync(path.join(root, "wiki", "ingestions", `${taskId}.json`))) throw new Error("WIKI_RECEIPT_MISSING");
      if (editOnly) {
        emit({ childResult, commit: null });
      } else {
        git(root, ["add", "--", ...changed]);
        git(root, ["commit", "-m", `wiki: ${action} ${taskId}`], { env: {
          ...process.env, WORK_REPO_LOCK_HELD: "1", WORK_PARENT_COMMIT: "1", WIKI_ACTION: action, WIKI_TARGET: target,
        } });
        const commit = git(root, ["rev-parse", "HEAD"]);
        validateWork();
        if (changedFiles().length) throw new Error("WORK_PARENT_LEFT_DIRTY");
        emit({ childResult, commit });
      }
    }
  } catch (error) {
    let failure = error;
    if (beforeHead) {
      try {
        rollbackWorkRepository(root, beforeHead);
      } catch (rollbackError) {
        failure = new Error(`${error.message};WORK_ROLLBACK_FAILED:${rollbackError.message}`);
      }
    }
    emit({ childResult, commit: null, error: failure.message });
    throw failure;
  } finally {
    release();
  }
}
