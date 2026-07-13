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
  governance: ["schemas/**", "wiki/SCHEMA.md", "wiki/AGENTS.md", ".githooks/pre-commit", "AGENTS.md", "README.md", "project.yaml"],
};
const allowed = allowedByAction[action];
const matchesAllowed = (file) => allowed.some((rule) => rule.endsWith("/**") ? file.startsWith(rule.slice(0, -2)) : file === rule);
const prompt = `Act as the ${action} writer for ${task.id}. Read AGENTS.md, ${task.task_dir}/TASK.md, relevant evidence, and the local Wiki. Update only: ${allowed.join(", ")}. Complete the action's evidence and commit directly to work repository main. Do not bypass Git hooks. Do not change Wiki content or Schema.`;
const command = [
  ...(args.profile ? ["-p", args.profile] : []),
  ...(args.model ? ["-m", args.model] : []),
  "exec",
  "-C",
  root,
  "--sandbox",
  "workspace-write",
  prompt,
];

if (args.dry_run === "true") {
  process.stdout.write(`${JSON.stringify({ command: ["codex", ...command], allowed }, null, 2)}\n`);
} else {
  const release = acquireWorkRepoLock(root);
  try {
    if (git(root, ["config", "--get", "core.hooksPath"]) !== ".githooks") {
      throw new Error("Work repository must configure core.hooksPath=.githooks");
    }
    const beforeHead = git(root, ["rev-parse", "HEAD"]);
    const result = spawnSync("codex", command, {
      cwd: root,
      stdio: "inherit",
      env: {
        ...process.env,
        WORK_REPO_LOCK_HELD: "1",
        WORK_ACTION: action,
        WORK_ALLOWED_PATHS: JSON.stringify(allowed),
      },
    });
    if (result.error) throw result.error;
    if (result.status !== 0) throw new Error(`Work Agent failed with exit code ${result.status ?? 1}`);
    if (git(root, ["branch", "--show-current"]) !== "main") throw new Error("Work Agent changed the work repository branch");
    const status = git(root, ["status", "--porcelain"]);
    if (status) throw new Error(`Work Agent left uncommitted changes:\n${status}`);
    const afterHead = git(root, ["rev-parse", "HEAD"]);
    if (beforeHead === afterHead) throw new Error(`Work Agent created no commit for ${action}`);
    const changed = git(root, ["diff", "--name-only", `${beforeHead}..${afterHead}`]).split("\n").filter(Boolean);
    const forbidden = changed.filter((file) => !matchesAllowed(file));
    if (forbidden.length) throw new Error(`Work Agent committed files outside ${action}: ${forbidden.join(", ")}`);
    const validation = spawnSync(
      process.execPath,
      [path.join(REPO_ROOT, "scripts", "task", "validate-work.mjs"), "--work-root", root],
      { cwd: REPO_ROOT, encoding: "utf8" },
    );
    if (validation.status !== 0) throw new Error(validation.stderr || "Work repository validation failed");
    process.stdout.write(validation.stdout);
  } finally {
    release();
  }
}
