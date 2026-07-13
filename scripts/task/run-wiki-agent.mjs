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

const args = parseArgs(process.argv.slice(2));
const action = args.action;
const taskId = args.task;
assertTaskId(taskId);
if (!new Set(["context-task", "context-plan", "ingest"]).has(action)) {
  throw new Error("--action must be context-task, context-plan, or ingest");
}

const root = workRoot(args.work_root);
const backlog = readYaml(path.join(root, "backlog.yaml"));
const task = taskById(backlog, taskId);
const targetByAction = {
  "context-task": "TASK.md",
  "context-plan": "PLAN.md",
  ingest: "HANDOVER.md",
};
const target = path.posix.join(task.task_dir, targetByAction[action]);
resolveInside(root, target, "Wiki Agent target");
const promptByAction = {
  "context-task": `You are already inside the locked Wiki Agent wrapper; do not invoke wiki-context, wiki-ingest, run-work-agent, or another Agent. Review ${target} and the local Wiki. Directly update only the Related Context section of TASK.md with applicable Semantic Wiki pages and Decisions, including reasons for not applying important conflicting Decisions. Follow AGENTS.md and commit directly to main. Do not bypass Git hooks.`,
  "context-plan": `You are already inside the locked Wiki Agent wrapper; do not invoke wiki-context, wiki-ingest, run-work-agent, or another Agent. Review ${target}, TASK.md, and the local Wiki. Directly update only the Related Wiki and Decision section of PLAN.md with applicable knowledge, conflicts, and open points. Follow AGENTS.md and commit directly to main. Do not bypass Git hooks.`,
  ingest: `You are already inside the locked Wiki Agent wrapper; do not invoke wiki-context, wiki-ingest, run-work-agent, or another Agent. Ingest ${target} by directly editing only the Wiki paths allowed by wiki/AGENTS.md. Create an idempotent ingestion receipt from the HANDOVER digest, run wiki-index and work-check without re-entering the Agent launcher, and commit directly to main. Do not modify Task evidence or backlog.yaml. Do not bypass Git hooks.`,
};
const command = [
  ...(args.profile ? ["-p", args.profile] : []),
  ...(args.model ? ["-m", args.model] : []),
  "exec",
  "-C",
  root,
  "--sandbox",
  "workspace-write",
  promptByAction[action],
];

if (args.dry_run === "true") {
  process.stdout.write(`${JSON.stringify({ command: ["codex", ...command], target }, null, 2)}\n`);
} else {
  const release = acquireWorkRepoLock(root);
  try {
    if (git(root, ["config", "--get", "core.hooksPath"]) !== ".githooks") {
      throw new Error("Work repository must configure core.hooksPath=.githooks before running the Wiki Agent");
    }
    const beforeHead = git(root, ["rev-parse", "HEAD"]);
    const result = spawnSync("codex", command, {
      cwd: root,
      stdio: "inherit",
      env: {
        ...process.env,
        WORK_REPO_LOCK_HELD: "1",
        WIKI_ACTION: action,
        WIKI_TARGET: target,
      },
    });
    if (result.error) throw result.error;
    if (result.status !== 0) throw new Error(`Wiki Agent failed with exit code ${result.status ?? 1}`);
    if (git(root, ["branch", "--show-current"]) !== "main") {
      throw new Error("Wiki Agent changed the work repository branch");
    }
    const status = git(root, ["status", "--porcelain"]);
    if (status) throw new Error(`Wiki Agent left uncommitted changes:\n${status}`);
    const afterHead = git(root, ["rev-parse", "HEAD"]);
    if (beforeHead !== afterHead) {
      const changed = git(root, ["diff", "--name-only", `${beforeHead}..${afterHead}`]).split("\n").filter(Boolean);
      const allowed = action === "ingest"
        ? (file) => /^wiki\/(semantic|decisions|ingestions)\//.test(file) || file === "wiki/index.json"
        : (file) => file === target;
      const forbidden = changed.filter((file) => !allowed(file));
      if (forbidden.length) throw new Error(`Wiki Agent committed files outside its scope: ${forbidden.join(", ")}`);
    } else if (action !== "ingest") {
      process.stdout.write("Wiki context was already current; no commit was created.\n");
    }
    const validation = spawnSync(
      process.execPath,
      [path.join(REPO_ROOT, "scripts", "task", "validate-work.mjs"), "--work-root", root],
      { cwd: REPO_ROOT, encoding: "utf8" },
    );
    if (validation.status !== 0) throw new Error(validation.stderr || "Work repository validation failed");
    process.stdout.write(validation.stdout);
    if (action === "ingest" && !fs.existsSync(path.join(root, "wiki", "ingestions", `${taskId}.json`))) {
      throw new Error(`Wiki Agent did not create the ${taskId} ingestion receipt`);
    }
  } finally {
    release();
  }
}
