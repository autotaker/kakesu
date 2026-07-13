import fs from "node:fs";
import path from "node:path";
import {
  REPO_ROOT,
  REQUIRED_TASK_FILES,
  acquireWorkRepoLock,
  assertSlug,
  assertTaskId,
  dateInTimezone,
  git,
  parseArgs,
  readYaml,
  replaceTemplate,
  workRoot,
  writeYaml,
} from "./lib.mjs";

const args = parseArgs(process.argv.slice(2));
assertTaskId(args.id);
assertSlug(args.slug);
if (!args.title || !args.epic) {
  throw new Error("--title and --epic are required");
}

const root = workRoot(args.work_root);
const release = acquireWorkRepoLock(root);
let taskDir;
try {
const backlogFile = path.join(root, "backlog.yaml");
const backlog = readYaml(backlogFile);
const project = readYaml(path.join(root, "project.yaml"));
if (!backlog.epics?.some((epic) => epic.id === args.epic)) {
  throw new Error(`Unknown epic: ${args.epic}`);
}
if (backlog.tasks?.some((task) => task.id === args.id)) {
  throw new Error(`Task already exists: ${args.id}`);
}

const relativeTaskDir = `tasks/${args.id}-${args.slug}`;
taskDir = path.join(root, relativeTaskDir);
if (fs.existsSync(taskDir)) {
  throw new Error(`Task directory already exists: ${taskDir}`);
}
fs.mkdirSync(path.dirname(taskDir), { recursive: true });
fs.mkdirSync(taskDir, { recursive: false });

const values = {
  TASK_ID: args.id,
  TITLE: args.title,
  TITLE_YAML: JSON.stringify(args.title),
  DATE: dateInTimezone(project.timezone || "UTC"),
};
for (const filename of REQUIRED_TASK_FILES) {
  const template = fs.readFileSync(path.join(REPO_ROOT, "templates", "task", filename), "utf8");
  fs.writeFileSync(path.join(taskDir, filename), replaceTemplate(template, values), "utf8");
}

backlog.tasks ??= [];
backlog.tasks.push({
  id: args.id,
  title: args.title,
  type: args.type || "feature",
  epic: args.epic,
  status: "backlog",
  priority: args.priority || "P2",
  estimate_points: 1,
  task_dir: relativeTaskDir,
  depends_on: [],
  assignees: {},
});
writeYaml(backlogFile, backlog);
git(root, ["add", "backlog.yaml", relativeTaskDir]);
git(root, ["commit", "-m", `task: create ${args.id}`]);
process.stdout.write(`${taskDir}\n`);
} catch (error) {
  if (taskDir && fs.existsSync(taskDir)) fs.rmSync(taskDir, { recursive: true, force: true });
  try {
    if (git(root, ["status", "--porcelain", "backlog.yaml"])) {
      git(root, ["restore", "--staged", "--worktree", "backlog.yaml"]);
    }
  } catch {
    // Preserve the original error; recovery is best-effort before the initial commit.
  }
  throw error;
} finally {
  release();
}
